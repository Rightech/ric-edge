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

package logger

import (
	"fmt"

	"github.com/Rightech/ric-edge/pkg/utils"
	log "github.com/sirupsen/logrus"
)

type Logger struct {
	*log.Entry
	level  log.Level
	finder utils.FilePosCallFinder
}

func New(level string, lvl log.Level) Logger {
	stdL := log.StandardLogger()

	newL := log.New()
	newL.SetLevel(stdL.Level)

	originalFmt := stdL.Formatter.(interface {
		GetOriginal() log.Formatter
	}).GetOriginal()

	newL.SetFormatter(originalFmt)

	l := newL.WithField("wrapped_level", level)

	return Logger{l, lvl, utils.FilePosCallFinder{
		Skip: 3, SkipPrefixes: []string{"logrus/", "logrus@"}}}
}

func (l Logger) Print(v ...interface{}) {
	file, _, line := l.finder.FindCaller()
	src := fmt.Sprintf("%s:%d", file, line)

	l.WithField("source", src).Log(l.level, v...)
}

func (l Logger) Println(v ...interface{}) {
	file, _, line := l.finder.FindCaller()
	src := fmt.Sprintf("%s:%d", file, line)

	l.WithField("source", src).Logln(l.level, v...)
}

func (l Logger) Printf(format string, v ...interface{}) {
	file, _, line := l.finder.FindCaller()
	src := fmt.Sprintf("%s:%d", file, line)

	l.WithField("source", src).Logf(l.level, format, v...)
}
