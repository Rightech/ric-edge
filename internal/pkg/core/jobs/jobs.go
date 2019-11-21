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

package jobs

import (
	"github.com/robfig/cron/v3"
)

type Service struct {
	*cron.Cron
}

func New() Service {
	crn := cron.New(
		cron.WithParser(
			cron.NewParser(
				cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom |
					cron.Month | cron.Dow | cron.Descriptor,
			),
		),
	)

	crn.Start()

	return Service{crn}
}

func (s Service) AddFunc(spec string, fn func()) (int, error) {
	id, err := s.Cron.AddFunc(spec, fn)
	return int(id), err
}

func (s Service) Remove(id int) {
	s.Cron.Remove(cron.EntryID(id))
}
