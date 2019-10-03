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

package formatter

import (
	"fmt"

	"github.com/Rightech/ric-edge/pkg/utils"
	"github.com/sirupsen/logrus"
)

// original source here
// https://github.com/onrik/logrus/blob/3d051631ae6df8ca3e6ad32039e202f27f1fa3cc/filename/filename.go

type Formatter struct {
	field     string
	finder    utils.FilePosCallFinder
	formatter func(file, function string, line int) string
	original  logrus.Formatter
}

func (f Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	modified := entry.WithField(f.field, f.formatter(f.finder.FindCaller()))
	modified.Level = entry.Level
	modified.Message = entry.Message
	return f.original.Format(modified)
}

func (f Formatter) GetOriginal() logrus.Formatter {
	return f.original
}

type RecordFmt func(file, function string, line int) string

func Build(orgF logrus.Formatter, field string, rfmt RecordFmt) Formatter {
	if rfmt == nil {
		rfmt = func(file, function string, line int) string {
			return fmt.Sprintf("%s:%d", file, line)
		}
	}

	f := Formatter{
		field: field,
		finder: utils.FilePosCallFinder{
			Skip:         5,
			SkipPrefixes: []string{"logrus/", "logrus@"},
		},
		formatter: rfmt,
		original:  orgF,
	}

	return f
}
