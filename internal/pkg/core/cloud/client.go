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

package cloud

import (
	"net/http"
	"time"
)

const defaultRequestTimeout = 15 * time.Second

type client struct {
	http.Client
	token, userAgent string
}

func newClient(token, version string) client {
	cli := http.Client{
		Timeout: defaultRequestTimeout,
	}

	return client{cli, token, "ric-edge/" + version}
}

func (c *client) head(url string) (*http.Response, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

func (c *client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", c.userAgent)

	return c.Client.Do(req)
}
