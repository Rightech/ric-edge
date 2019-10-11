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

import "errors"

// Noop no operation action service
// used when action disabled
type Noop struct{}

func (Noop) Add([]byte) ([]byte, error) {
	return nil, errors.New("action: disabled")
}

func (Noop) Delete(string) ([]byte, error) {
	return nil, errors.New("action: disabled")
}

func (Noop) Call(string, []byte) ([]byte, error) {
	return nil, errors.New("action: disabled")
}
