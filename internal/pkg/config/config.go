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
	"flag"

	"github.com/Rightech/ric-edge/pkg/log/formatter"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func Init(version []string) {
	cfgPath := flag.String("config", "config.toml", "path to configuration file")
	flag.Parse()

	viper.SetConfigFile(*cfgPath)

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	if len(version) > 0 && version[0] != "" {
		viper.Set("version", version[0])
	} else {
		viper.Set("version", "unknown")
	}

	viper.SetDefault("log_level", "info")
	viper.SetDefault("log_format", "text")
	viper.SetDefault("ws_port", 9000)

	setupLogger()
}

func logFormatter() log.Formatter {
	tsFormat := "2006-01-02 15:04:05"

	format := viper.GetString("log_format")

	var logFmt log.Formatter
	switch format {
	case "text":
		logFmt = &log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: tsFormat,
		}
	case "json":
		logFmt = &log.JSONFormatter{
			TimestampFormat: tsFormat,
		}
	default:
		log.Fatal("unknown log format. use text or json")
	}

	return logFmt
}

func setupLogger() {
	filenameFormatter := formatter.Build(logFormatter(), "source", nil)
	log.SetFormatter(filenameFormatter)

	lvl, err := log.ParseLevel(viper.GetString("log_level"))

	if err != nil {
		log.Fatal("config:setupLogger:", err)
	}

	log.SetLevel(lvl)
}
