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

package jsonrpc

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

type TransportWithConnect interface {
	Transport
	Connect() error
}

func ServeWithReconnect(ctx context.Context, cli TransportWithConnect, caller Caller) {
	srv := New(cli, caller)

	if v, ok := caller.(interface {
		InjectRPC(RPC)
	}); ok {
		v.InjectRPC(srv)
	}

	for ctx.Err() == nil {
		err := srv.Serve(ctx)
		if err != nil {
			t := time.NewTicker(retriesSleep)
			counter := 0
			for counter >= 0 {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					if counter%5 == 0 {
						log.Info("try reconnect to core")
					}
					counter++
					err := cli.Connect()
					if err != nil {
						log.WithError(err).Debug("reconnect error")
						continue
					}

					counter = -1
				}
			}
			t.Stop()
		}
	}
}
