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
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func printStat(ctx context.Context, totalSize int64, file *os.File) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Downloaded: 100%")
			return
		case <-ticker.C:
			st, err := file.Stat()
			if err != nil {
				continue
			}

			percent := float64(st.Size()) / float64(totalSize) * 100

			log.Infof("Downloaded: %.0f%%", percent)
		}
	}
}

func Download(url string) {
	idx := strings.LastIndexByte(url, '/')
	if idx < 0 {
		log.Warn("bad url format")
		return
	}

	name := url[idx+1:]

	client := http.Client{
		Timeout: 10 * time.Minute,
	}

	resp, err := client.Get(url)
	if err != nil {
		log.WithError(err).Warn("download fail")
		return
	}
	defer resp.Body.Close()

	file, err := os.Create(name)
	if err != nil {
		log.WithError(err).Warn("create fail")
		return
	}

	defer file.Close()

	ctx, cancel := context.WithCancel(resp.Request.Context())
	defer cancel()
	go printStat(ctx, resp.ContentLength, file)

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.WithError(err).Warn("write fail")
		return
	}

	err = file.Chmod(os.ModePerm)
	if err != nil {
		log.WithError(err).Warn("set permission fail")
		return
	}
}
