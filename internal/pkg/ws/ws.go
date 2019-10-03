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
	"errors"
	"io"
	"io/ioutil"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	ws   *websocket.Conn
	u    url.URL
	done chan struct{}
}

func New(port int, path string) (Service, error) {
	if !(1 <= port && port <= 65535) {
		return Service{}, errors.New("ws.new: wrong port")
	}

	s := Service{
		u:    url.URL{Scheme: "ws", Host: "localhost:" + strconv.Itoa(port), Path: path},
		done: make(chan struct{}),
	}

	return s, s.Connect()
}

func (s *Service) Connect() error {
	c, resp, err := websocket.DefaultDialer.Dial(s.u.String(), nil)
	if err != nil {
		if resp != nil {
			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.WithError(err).Error("read error response body")
				return err
			}
			resp.Body.Close()
			log.Error(string(data))
		}
		return err
	}

	resp.Body.Close()
	s.ws = c

	log.Info("connected to core")

	return nil
}

func (s Service) Close() error {
	close(s.done)
	s.ws.WriteControl( // nolint: errcheck
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "shutdown"),
		time.Now().Add(time.Second),
	)

	return s.ws.Close()
}

func (s Service) NextWriter() (io.WriteCloser, error) {
	return s.ws.NextWriter(websocket.TextMessage)
}

func (s Service) NextReader() (io.Reader, error) {
	mt, r, err := s.ws.NextReader()
	if err != nil {
		select {
		case <-s.done:
			log.Info("disconnected from core (because going to shutdown)")
			return nil, err
		default:
		}

		if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
			log.Info("disconnected from core (because core going to normal shutdown)")
			return nil, err
		}

		log.WithError(err).Info("disconnected from core")
		return nil, err
	}

	if mt != websocket.TextMessage {
		return nil, errors.New("unknown message type: " + strconv.Itoa(mt))
	}

	return r, nil
}
