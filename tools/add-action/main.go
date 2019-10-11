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
	"github.com/Rightech/ric-edge/internal/app/core/config"
	paho "github.com/eclipse/paho.mqtt.golang"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const code = `const xml = require('fast-xml-parser');
const math = require('mathjs');

function handle(packet) {
    const time = packet.time || Date.now();
    const iso = new Date(+time).toISOString();
    const rand = Math.random();
    const x = xml.parse('<a>10</a>');
    console.log(new Date().toISOString(), packet);
    return { iso, rand, x };
}

module.exports = handle;
`

const pkg = `{
    "name":"ric-v3-5d681eb0b6a1c026ca418faf",
    "private":true,
    "dependencies":{
        "fast-xml-parser":"latest",
        "mathjs":"6.1.0"
    }
}
`

func getPayload() []byte {
	data := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "add-action",
		"params": map[string]string{
			"name":    "ric-v3-5d681eb0b6a1c026ca418faf",
			"package": pkg,
			"code":    code,
		},
	}

	payload, err := jsoniter.ConfigFastest.Marshal(&data)
	if err != nil {
		panic(err)
	}

	return payload
}

func main() {
	config.Setup()

	opts := paho.NewClientOptions().
		AddBroker(viper.GetString("core.mqtt.url"))

	client := paho.NewClient(opts)
	defer client.Disconnect(500)

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Error(token.Error())
		return
	}

	token = client.Publish("ric-edge/action/command", 2, false, getPayload())
	if token.Wait() && token.Error() != nil {
		log.Error(token.Error())
		return
	}
}
