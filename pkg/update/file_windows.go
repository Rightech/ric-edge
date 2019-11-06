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
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func setPerm(file *os.File) error {
	return nil
}

func getName(url string) string {
	idx := strings.LastIndexByte(url, '/')
	if idx < 0 {
		log.Warn("wrong url format")
		return ""
	}

	name := url[idx+1:]

	sidx := strings.LastIndexByte(url[:idx], '/')
	if sidx < 0 {
		log.Warn("wrong url format")
		return ""
	}

	ver := url[sidx+1 : idx]

	tmp := strings.Split(name, ".")
	tmp[0] += "-" + ver
	return strings.Join(tmp, ".")
}
