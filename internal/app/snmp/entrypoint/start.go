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
	"os"

	"github.com/spf13/viper"

	"github.com/Rightech/ric-edge/internal/app/snmp/handler"
	"github.com/Rightech/ric-edge/internal/pkg/ws"
	"github.com/Rightech/ric-edge/pkg/jsonrpc"
)

func Start(done <-chan os.Signal) error {
	hand, err := handler.New(
		viper.GetString("snmp.host_port"),
		viper.GetString("snmp.community"),
		viper.GetString("snmp.version"),
		viper.GetString("snmp.mode"),
		viper.GetString("snmp.auth_protocol"),
		viper.GetString("snmp.auth_key"),
		viper.GetString("snmp.priv_protocol"),
		viper.GetString("snmp.priv_key"),
		viper.GetString("snmp.security_name"))
	if err != nil {
		return err
	}

	cli, err := ws.New(viper.GetInt("ws_port"), viper.GetString("version"),
		viper.GetString("snmp.ws_path"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	go jsonrpc.ServeWithReconnect(ctx, cli, hand, jsonrpc.CatchPanic(viper.GetBool("catch_panic")))

	<-done
	cancel()
	cli.Close()
	hand.Close()

	return nil
}
