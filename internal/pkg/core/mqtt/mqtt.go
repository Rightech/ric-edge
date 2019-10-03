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

package mqtt

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/Rightech/ric-edge/pkg/log/logger"
	"github.com/Rightech/ric-edge/pkg/store/mqtt"
	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	cli paho.Client
	rpc rpc
}

const (
	requestTopic  = "ric-edge/+/command" // + - connector type
	responseTopic = "ric-edge/%s/response"
	qos           = 1
)

type rpc interface {
	Call(string, []byte) []byte
}

func New(url, clientID, cert, key string, db mqtt.DB, cli rpc) (Service, error) {
	var certs []tls.Certificate

	if cert != "" && key != "" {
		pair, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return Service{}, err
		}

		certs = []tls.Certificate{pair}
	}

	paho.CRITICAL = logger.New("critical", log.ErrorLevel)
	paho.ERROR = logger.New("error", log.DebugLevel)
	paho.WARN = logger.New("warn", log.DebugLevel)

	opts := paho.NewClientOptions().
		AddBroker(url).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetStore(mqtt.NewStore(db)).
		SetCleanSession(false).
		SetKeepAlive(5 * time.Second).
		SetOrderMatters(false)

	if certs != nil {
		opts = opts.SetTLSConfig(&tls.Config{
			Certificates: certs,
		})

		log.Debug("mqtt tls enabled")
	}

	client := paho.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return Service{}, token.Error()
	}

	s := Service{client, cli}

	token = client.Subscribe(requestTopic, qos, s.rpcCallback)
	if token.Wait() && token.Error() != nil {
		return Service{}, token.Error()
	}

	log.Info("mqtt ready")

	return s, nil
}

func (s Service) rpcCallback(cli paho.Client, msg paho.Message) {
	connectorID := strings.Split(msg.Topic(), "/")[1]

	resp := s.rpc.Call(connectorID, msg.Payload())

	token := s.cli.Publish(fmt.Sprintf(responseTopic, connectorID), qos, false, resp)
	if token.WaitTimeout(time.Minute) && token.Error() != nil {
		log.WithFields(log.Fields{
			"response":  string(resp),
			"connector": connectorID,
			"request":   string(msg.Payload()),
			"error":     token.Error(),
		}).Error("err publish response")
	}
}

func (s Service) Close() error {
	s.cli.Disconnect(uint(time.Second / time.Millisecond))
	return nil
}
