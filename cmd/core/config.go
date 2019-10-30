//
// Copyright 2019 Rightech IoT. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate go run -tags=gen gen.go

package main

import (
	"io"
	"os"

	"github.com/Rightech/ric-edge/cmd/core/data"
	_ "github.com/shurcooL/vfsgen" // required to go mod find this
)

func printDefaultConfig() {
	file, err := data.Assets.Open("/default-config.toml")
	if err != nil {
		os.Stderr.WriteString(err.Error())
		return
	}

	_, err = io.Copy(os.Stdout, file)
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
}
