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

if [ -z "$2" ]; then
  echo "usage: $0 <package-path> <version>"
  exit 1
fi

package=$1
version=$2

# default platforms
platforms_string="linux/amd64,linux/386,linux/arm64,linux/arm"

# add additional platforms
if [ -n "$3" ]; then
    platforms_string=$platforms_string","$3
fi

# read platforms to array
IFS=',' read -r -a platforms <<< "$platforms_string"

package_split=(${package//\// })
package_name=${package_split[-1]}

bin_dir='./build'

echo 'Name: '$package_name
echo 'Release version: '$version

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    goos=${platform_split[0]}
    goarch=${platform_split[1]}
    output_name=$bin_dir'/'$package_name'-'$goos'-'$goarch
    if [ $goos = "windows" ]; then
        output_name+='.exe'
    fi

    printf 'Build: '$platform'.... '

    if [ -f $output_name ]; then
        echo 'already exists'
        continue
    fi

    CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
        -a \
        -mod vendor \
        -installsuffix cgo \
        -ldflags "-w -s -X main.version=$version" \
        -o $output_name \
        $package

    if [ $? -ne 0 ]; then
        printf 'An error has occurred! Aborting the script execution...'
        exit 1
    fi

    echo 'OK'
done

upx $bin_dir/$package_name*
