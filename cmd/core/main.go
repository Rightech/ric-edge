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

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Rightech/ric-edge/internal/app/core/config"
	"github.com/Rightech/ric-edge/internal/app/core/entrypoint"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// set at build time via ldflags
var version string // nolint: gochecknoglobals

func main() {
	config.Setup(version)

	log.Info("Version: ", viper.GetString("version"))

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	err := entrypoint.Start(signalCh)
	if err != nil {
		log.Fatal(err)
	}
}
