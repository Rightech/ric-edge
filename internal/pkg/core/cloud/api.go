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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	jsoniter "github.com/json-iterator/go"
)

type Service struct {
	baseURL url
	client  client
}

func New(baseURL, token, v string) (Service, error) {
	if baseURL == "" {
		return Service{}, errors.New("cloud: empty url")
	}

	s := Service{client: newClient(token, v), baseURL: newURL(baseURL)}

	return s, s.ping()
}

func (s Service) ping() error {
	resp, err := s.client.Head(s.baseURL.Self())
	if err != nil {
		return err
	}

	resp.Body.Close()

	return nil
}

func errIfBadStatus(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return errors.New(jsoniter.ConfigFastest.Get(data, "message").ToString())
}

func (s Service) LoadModel(id string) (m Model, err error) {
	var resp *http.Response
	resp, err = s.client.Get(s.baseURL.GetModel(id))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = errIfBadStatus(resp)
	if err != nil {
		err = fmt.Errorf("load.model[%s]:%w", id, err)
		return
	}

	err = jsoniter.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return
	}

	err = m.prepare()

	return
}

func (s Service) LoadObject(id string) (o Object, err error) {
	var resp *http.Response
	resp, err = s.client.Get(s.baseURL.GetObject(id))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = errIfBadStatus(resp)
	if err != nil {
		err = fmt.Errorf("load.object[%s]:%w", id, err)
		return
	}

	err = jsoniter.NewDecoder(resp.Body).Decode(&o)

	return
}
