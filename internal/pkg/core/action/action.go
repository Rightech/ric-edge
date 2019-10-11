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

package action

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

// Service is a action service client
// operates with actions via http
type Service struct {
	cli     *http.Client
	baseURL string
}

const defaultRequestTimeout = 15 * time.Second

func New(port int) (Service, error) {
	if !(1 <= port && port <= 65535) {
		return Service{}, errors.New("action: bad port")
	}

	s := Service{&http.Client{
		Timeout: defaultRequestTimeout,
	}, "http://127.0.0.1:" + strconv.Itoa(port)}

	return s, s.ping()
}

func (s Service) ping() error {
	resp, err := s.cli.Head(s.baseURL)
	if err != nil {
		return err
	}

	resp.Body.Close()

	return nil
}

func (s Service) Add(payload []byte) (body []byte, err error) {
	resp, err := s.cli.Post(
		s.baseURL+"/func", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	body, err = checkResp(resp)
	resp.Body.Close()
	return
}

func (s Service) Delete(name string) (body []byte, err error) {
	req, err := http.NewRequest("DELETE", s.baseURL+"/func/"+name, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.cli.Do(req)
	if err != nil {
		return nil, err
	}

	body, err = checkResp(resp)
	resp.Body.Close()
	return
}

func (s Service) Call(name string, payload []byte) (body []byte, err error) {
	resp, err := s.cli.Post(s.baseURL+"/func/"+name, "application/json",
		bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	body, err = checkResp(resp)
	if err != nil {
		return
	}

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	body = result

	resp.Body.Close()
	return
}

func checkResp(resp *http.Response) ([]byte, error) {
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		result, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return result, errors.New("action: error response")
	}

	return nil, nil
}
