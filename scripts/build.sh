#!/bin/bash

if [ -z "$2" ]; then
  echo "usage: $0 <package-path> <version>"
  exit 1
fi

package=$1
version=$2

package_split=(${package//\// })
package_name=${package_split[-1]}

bin_dir='./build'

platforms=("linux/amd64" "linux/386" "windows/amd64" "windows/386")

echo 'Name: '$package_name
echo 'Release version: '$version

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    goos=${platform_split[0]}
    goarch=${platform_split[1]}
    output_name=$bin_dir'/'$package_name'-'$goos'-'$goarch'-'$version
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

    upx $output_name

    if [ $? -ne 0 ]; then
        printf 'An error has occurred! Aborting the script execution...'
        exit 1
    fi
    echo 'OK'
done
