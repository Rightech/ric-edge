/**
 * Copyright 2019 Rightech IoT. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ws

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/Rightech/ric-edge/pkg/jsonrpc"
	"github.com/Rightech/ric-edge/pkg/nanoid"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

const (
	httpTimeout        = 1000 * time.Second
	httpMaxHeaderBytes = 1 << 20
)

type ctxKey string

const (
	loggerKey  ctxKey = "loggerKey"
	sessionKey ctxKey = "sessionKey"
)

// websocket connection with logger and session id
type conn struct {
	cmx *sync.Mutex
	*websocket.Conn
	l         *log.Entry
	name, sid string
	rmx       *sync.Mutex
	req       map[string]chan<- []byte
}

// Service represent web socket server (or http fallback server)
type Service struct {
	upgrader  websocket.Upgrader
	srv       *http.Server
	verConstr *semver.Constraints
	mx        sync.RWMutex
	conns     map[string]conn

	done       chan struct{}
	requestsCh chan<- []byte
}

// this wrapper add logger with request id to request context
func requestsWrapper(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid := nanoid.New()

		logger := log.WithField("sid", sid).WithField("addr", r.RemoteAddr)

		ctx := context.WithValue(r.Context(), loggerKey, logger)
		ctx = context.WithValue(ctx, sessionKey, sid)

		r = r.WithContext(ctx)

		f.ServeHTTP(w, r)
	}
}

func versionToConstr(v string) (*semver.Constraints, error) {
	if v[0] == 'v' {
		v = v[1:]
	}

	tmp := strings.Split(v, ".")

	if len(tmp) < 2 {
		return nil, errors.New("wrong version format")
	}

	if len(tmp) == 2 {
		tmp = append(tmp, "")
	}

	tmp[2] = "0"

	v = strings.Join(tmp, ".")

	return semver.NewConstraint("~" + v)
}

// New create new WebSocket server
func New(port int, version string, requestsCh chan<- []byte) (*Service, error) {
	if !(1 <= port && port <= 65535) {
		return nil, errors.New("ws.new: wrong port")
	}

	vc, err := versionToConstr(version)
	if err != nil {
		return nil, err
	}

	ws := &Service{
		verConstr:  vc,
		conns:      make(map[string]conn, 10),
		done:       make(chan struct{}),
		requestsCh: requestsCh,
	}

	ws.upgrader.CheckOrigin = func(r *http.Request) bool {
		return true // allow all origins
	}

	ws.upgrader.Error = func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		err := writeError(w, reason, status)
		if err != nil {
			logger := r.Context().Value(loggerKey).(*log.Entry)
			logger.WithError(err).Error("write error (ws handler)")
		}
	}

	srv := &http.Server{
		Addr:           "localhost:" + strconv.Itoa(port),
		Handler:        requestsWrapper(ws.handler),
		ReadTimeout:    httpTimeout,
		WriteTimeout:   httpTimeout,
		MaxHeaderBytes: httpMaxHeaderBytes,
	}

	ws.srv = srv

	return ws, nil
}

// Start WebSocket server
func (s *Service) Start(ctx context.Context) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		log.Info("ws ready")
		err := s.srv.ListenAndServe() // blocks current goroutine
		if err != http.ErrServerClosed {
			errCh <- fmt.Errorf("ws:Start:%w", err)
			return
		}

		close(errCh)
	}()

	return errCh
}

func (s *Service) Close() error {
	close(s.done)

	s.mx.RLock()
	for _, v := range s.conns {
		v.WriteControl( // nolint: errcheck
			websocket.CloseMessage,
			websocket.FormatCloseMessage(
				websocket.CloseNormalClosure, "shutdown"),
			time.Now().Add(time.Second),
		)
		v.Close()
	}
	s.mx.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.srv.Shutdown(ctx)
}

func (s *Service) checkVersion(w http.ResponseWriter, r *http.Request, logger *log.Entry) bool {
	connectorVer, err := semver.NewVersion(r.Header.Get("x-connector-version"))
	if err != nil {
		err := fmt.Errorf("broken connector version: %w", err)
		if err := writeError(w, err, http.StatusBadRequest); err != nil {
			logger.WithError(err).Error("ws:handler:write error")
		}

		return false
	}

	if !s.verConstr.Check(connectorVer) {
		err := errors.New("incompatible connector/core version")
		if err := writeError(w, err, http.StatusBadRequest); err != nil {
			logger.WithError(err).Error("ws:handler:write error")
		}

		return false
	}

	return true
}

func (s *Service) handler(w http.ResponseWriter, r *http.Request) {
	logger := r.Context().Value(loggerKey).(*log.Entry)
	sid := r.Context().Value(sessionKey).(string)

	if !s.checkVersion(w, r, logger) {
		return
	}

	url := strings.Split(r.URL.Path, "/")
	if len(url) != 2 {
		err := errors.New("path should be /<connector_type>")
		if err := writeError(w, err, http.StatusBadRequest); err != nil {
			logger.WithError(err).Error("ws:handler:write error")
		}

		return
	}

	connectorType := url[1]

	s.mx.RLock()
	_, ok := s.conns[connectorType]
	s.mx.RUnlock()

	if ok {
		err := errors.New("connector already exists")
		if err := writeError(w, err, http.StatusBadRequest); err != nil {
			logger.WithError(err).Error("ws:handler:write error")
		}

		return
	}

	logger = logger.WithField("n", connectorType)

	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.WithError(err).Error("ws:handler:upgrade error")
		return
	}

	logger.Info("new connection")

	wsc := conn{
		Conn: c, l: logger, name: connectorType, sid: sid,
		cmx: new(sync.Mutex),
		rmx: new(sync.Mutex),
		req: make(map[string]chan<- []byte, 10),
	}

	s.mx.Lock()
	s.conns[connectorType] = wsc
	s.mx.Unlock()

	go s.listen(wsc)
}

func (s *Service) closeConnOnErr(conn conn) {
	conn.rmx.Lock()
	conn.req = nil
	conn.rmx.Unlock()
	conn.cmx.Lock()
	conn.Close()
	conn.cmx.Unlock()

	s.mx.Lock()
	delete(s.conns, conn.name)
	s.mx.Unlock()
}

func (s *Service) listen(conn conn) {
	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-s.done:
				close(s.requestsCh)
				conn.l.Info("disconnect client because server dies (normal)")

				return
			default:
			}

			s.closeConnOnErr(conn)

			ll := conn.l
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				ll = ll.WithError(err)
			}

			ll.Info("client disconnect")

			return
		}

		if mt != websocket.TextMessage {
			conn.l.WithFields(log.Fields{"mt": mt, "m": string(msg)}).
				Error("unknown message type")

			continue
		}

		// check this message request or response
		methodVal := jsoniter.ConfigFastest.Get(msg, "method")
		if methodVal.ValueType() != jsoniter.InvalidValue {
			// this msg is request
			s.requestsCh <- msg

			continue
		}

		idVal := jsoniter.ConfigFastest.Get(msg, "id")
		if idVal.LastError() != nil {
			conn.l.WithError(idVal.LastError()).Error("get id from json")

			continue
		}

		id := idVal.ToString()

		conn.rmx.Lock()
		ch, ok := conn.req[id]
		delete(conn.req, id)
		conn.rmx.Unlock()

		if !ok {
			panic("resp chan not found")
		}

		ch <- msg
	}
}

func (s *Service) Call(name, id string, payload []byte) <-chan []byte {
	resp := make(chan []byte, 1)

	s.mx.RLock()
	conn, ok := s.conns[name]
	s.mx.RUnlock()

	if !ok {
		resp <- jsonrpc.BuildErrResp(id, errNotFound)
		return resp
	}

	conn.cmx.Lock()
	err := conn.WriteMessage(websocket.TextMessage, payload)
	conn.cmx.Unlock()

	if err != nil {
		s.closeConnOnErr(conn)

		conn.l.WithError(err).Error("ws.conn.write")

		resp <- jsonrpc.BuildErrResp(id, errNotAvailable.AddData("sid", conn.sid))

		return resp
	}

	conn.rmx.Lock()
	conn.req[id] = resp
	conn.rmx.Unlock()

	return resp
}
