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

package update

import (
	"net/http"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

const githubURL = "https://api.github.com/repos/Rightech/ric-edge/releases/latest"

type release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string
		URL  string `json:"browser_download_url"`
	}
}

func Check(currentVer, name string) string {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(githubURL)
	if err != nil {
		log.WithError(err).Warn("check update request")
		return ""
	}
	defer resp.Body.Close()

	var release release
	err = jsoniter.ConfigFastest.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		log.WithError(err).Warn("decode check update response")
		return ""
	}

	if release.TagName[1:] != currentVer { // [1:] required to trim v prefix
		for _, ass := range release.Assets {
			if strings.HasPrefix(ass.Name, name) {
				return "New version available. Download it: " + ass.URL
			}
		}
	}

	return ""
}
