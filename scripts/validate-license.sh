#!/bin/bash

# Copyright 2019 Rightech IoT. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# original script stolen from here
# https://github.com/helm/helm/blob/935ee90d9ff3af9ffc22735c0d6c092c5050f3ef/scripts/validate-license.sh

set -euo pipefail
IFS=$'\n\t'

find_files() {
  find . -not \( \
    \( \
      -wholename './vendor' \
      -o -wholename './pkg/proto' \
      -o -wholename '*testdata*' \
      -o -wholename './third_party' \
      -o -wholename '*_generated.go' \
    \) -prune \
  \) \
  \( -name '*.go' -o -name '*.sh' -o -name 'Dockerfile' \)
}

failed_license_header=($(grep -L 'Licensed under the Apache License, Version 2.0 (the "License");' $(find_files))) || :
if (( ${#failed_license_header[@]} > 0 )); then
  echo "Some source files are missing license headers."
  for f in "${failed_license_header[@]}"; do
    echo "  $f"
  done
  exit 1
fi

failed_copyright_header=($(grep -L 'Copyright 2019 Rightech IoT' $(find_files))) || :
if (( ${#failed_copyright_header[@]} > 0 )); then
  echo "Some source files are missing the copyright header."
  for f in "${failed_copyright_header[@]}"; do
    echo "  $f"
  done
  exit 1
fi
