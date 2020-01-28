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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/Rightech/ric-edge/internal/app/ble/handler"
	"github.com/Rightech/ric-edge/internal/app/common"
	"github.com/Rightech/ric-edge/internal/pkg/ws"
	"github.com/Rightech/ric-edge/pkg/jsonrpc"
)


func Start(done <-chan os.Signal) error {
	usePlugin := viper.GetBool("ble.use_plugin")

	var (
		hand jsonrpc.Caller
		err  error
	)

	if !usePlugin {
		hand, err = handler.New()
		if err != nil {
			return err
		}
	} else {
		hand, err = common.LoadPlugin("ble")
		if err != nil {
			log.WithError(err).Error("plugin open error. fallback to default")
			hand, err = handler.New()
			if err != nil {
				return err
			}
		} else {
			log.Debug("use plugin")
		}
	}

	cli, err := ws.New(viper.GetInt("ws_port"), viper.GetString("version"),
		viper.GetString("ble.ws_path"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	go jsonrpc.ServeWithReconnect(ctx, cli, hand)

	<-done
	cancel()
	cli.Close()
	// hand.Close()

	return nil
}
