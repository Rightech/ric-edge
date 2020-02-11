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

package entrypoint

import (
	"os"
	"time"

	"github.com/etcd-io/bbolt"
	"github.com/spf13/viper"

	"github.com/Rightech/ric-edge/internal/app/core/rpc"
	"github.com/Rightech/ric-edge/internal/pkg/core/cloud"
	"github.com/Rightech/ric-edge/internal/pkg/core/jobs"
	"github.com/Rightech/ric-edge/internal/pkg/core/mqtt"
	"github.com/Rightech/ric-edge/internal/pkg/core/ws"
	"github.com/Rightech/ric-edge/pkg/lua"
)

func Start(done <-chan os.Signal) error { // nolint: funlen
	db, err := bbolt.Open(viper.GetString("core.db.path"), 0600, &bbolt.Options{
		Timeout:      time.Second,
		FreelistType: bbolt.FreelistArrayType,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	api, err := cloud.New(
		viper.GetString("core.cloud.url"),
		viper.GetString("core.cloud.token"),
		viper.GetString("version"),
	)
	if err != nil {
		return err
	}

	// this channel needs to communicate between jsonrpc transport and rpcCli service
	// in this channel transport send jsonrpc requests
	requestsCh := make(chan []byte)

	sock, err := ws.New(viper.GetInt("ws_port"),
		viper.GetString("version"), requestsCh)
	if err != nil {
		return err
	}

	errCh := sock.Start()

	stateCh := make(chan []byte)

	luaMachine := lua.New()

	// wait while connectors reconnects
	// before continue
	time.Sleep(2 * time.Second)

	rpcCli, err := rpc.New(
		viper.GetString("core.id"),
		viper.GetDuration("core.rpc_timeout"),
		luaMachine, db, viper.GetBool("core.db.clean_state"),
		sock, api, jobs.New(), stateCh, requestsCh)
	if err != nil {
		return err
	}

	mqttCli, err := mqtt.New(
		viper.GetString("core.mqtt.url"),
		rpcCli.GetEdgeID(),
		viper.GetString("core.mqtt.cert_file"),
		viper.GetString("core.mqtt.key_path"),
		db, rpcCli, stateCh,
	)
	if err != nil {
		return err
	}

	defer func() {
		mqttCli.Close()
		sock.Close()
	}()

	select {
	case err := <-errCh:
		return err
	case <-done:
		return nil
	}
}
