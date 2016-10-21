#!/bin/sh
basedir=`pwd`/gopath/src/github.com/ecsteam/buildpack-usage
build_dir=`pwd`/build-output/build
version_file=`pwd`/version/number

mkdir ${build_dir} > /dev/null

set -e
set -x

export GOPATH=`pwd`/gopath

# Run tests
cd ${basedir}
for os in "linux windows darwin"; do
    suffix=${os}
    if [ "windows" = "${os}" ]; then
        suffix="windows.exe"
    elif [ "darwin" = "${os}" ]; then
        suffix="macosx"
    fi

    GOOS=${os} GOARCH=amd64 go build -ldflags="-X github.com/ecsteam/buildpack-usage/command.version=`cat ${version_file}`" -o ${build_dir}/buildpack-usage-${suffix}
done
