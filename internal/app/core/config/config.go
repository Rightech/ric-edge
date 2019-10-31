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

package config

import (
	"github.com/Rightech/ric-edge/internal/pkg/config"
	"github.com/spf13/viper"
)

// Setup viper and load configuration from config.toml and env variables
// also it setup logger
func Setup(version ...string) {
	config.Init(version)

	viper.SetDefault("core.rpc_timeout", "1m")
	viper.SetDefault("core.db.path", "storage.db")
	viper.SetDefault("core.db.clean_state", false)

	viper.SetDefault("core.mqtt.url", "tls://sandbox.rightech.io:8883")
	viper.SetDefault("core.mqtt.cert_file", "")
	viper.SetDefault("core.mqtt.key_path", "")

	viper.SetDefault("core.cloud.url", "https://sandbox.rightech.io/api/v1")
}
