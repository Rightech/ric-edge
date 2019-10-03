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
	*websocket.Conn
	l         *log.Entry
	name, sid string
}

// Service represent web socket server (or http fallback server)
type Service struct {
	upgrader websocket.Upgrader
	srv      *http.Server

	mx    sync.RWMutex
	conns map[string]conn

	rmx  sync.Mutex
	req  map[string]chan<- []byte
	done chan struct{}
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

// New create new WebSocket server
func New(port int) (*Service, error) {
	if !(1 <= port && port <= 65535) {
		return nil, errors.New("ws.new: wrong port")
	}

	ws := &Service{
		conns: make(map[string]conn, 10),
		req:   make(map[string]chan<- []byte, 10),
		done:  make(chan struct{}),
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

func (s *Service) handler(w http.ResponseWriter, r *http.Request) {
	logger := r.Context().Value(loggerKey).(*log.Entry)
	sid := r.Context().Value(sessionKey).(string)

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

	wsc := conn{c, logger, connectorType, sid}
	s.mx.Lock()
	s.conns[connectorType] = wsc
	s.mx.Unlock()

	go s.listen(wsc)
}

func (s *Service) closeConnOnErr(conn conn) {
	conn.Close()

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
			conn.l.WithFields(log.Fields{
				"mt": mt,
				"m":  string(msg),
			}).Error("unknown message type")
			continue
		}

		idVal := jsoniter.ConfigFastest.Get(msg, "id")

		if idVal.LastError() != nil {
			conn.l.WithError(idVal.LastError()).Error("get id from json")
			continue
		}

		id := idVal.ToString()

		s.rmx.Lock()
		ch, ok := s.req[id]
		delete(s.req, id)
		s.rmx.Unlock()
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

	err := conn.WriteMessage(websocket.TextMessage, payload)
	if err != nil {
		s.closeConnOnErr(conn)

		conn.l.WithError(err).Error("ws.conn.write")

		resp <- jsonrpc.BuildErrResp(id, errNotAvailable.AddData("sid", conn.sid))
		return resp
	}

	s.rmx.Lock()
	s.req[id] = resp
	s.rmx.Unlock()

	return resp
}
