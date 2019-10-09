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
	"context"
	"errors"
	"os"

	"github.com/Rightech/ric-edge/internal/app/modbus/handler"
	"github.com/Rightech/ric-edge/internal/pkg/ws"
	"github.com/Rightech/ric-edge/pkg/jsonrpc"
	"github.com/goburrow/modbus"
	"github.com/spf13/viper"
)

func Start(done <-chan os.Signal) error {
	var mcli modbus.Client

	if !(viper.GetBool("modbus.tcp") || viper.GetBool("modbus.rtu")) {
		return errors.New("modbus.tcp or modbus.rtu should be enabled")
	}

	if viper.GetBool("modbus.tcp") {
		mcli = modbus.TCPClient(viper.GetString("modbus.addr"))
	} else {
		mcli = modbus.RTUClient(viper.GetString("modbus.addr"))
	}

	cli, err := ws.New(viper.GetInt("ws_port"), viper.GetString("modbus.ws_path"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	go jsonrpc.ServeWithReconnect(ctx, cli, handler.New(mcli))

	<-done
	cancel()
	cli.Close()

	return nil
}
