#!/bin/sh
basedir=`pwd`/gopath/src/github.com/ecsteam/buildpack-usage
outdir=`pwd`/gopath-tested

mkdir -p ${outdir} > /dev/null 2>&1

set -e
set -x

export GOPATH=`pwd`/gopath

apk update && apk upgrade && apk add git

# Install Glide
go get -u github.com/Masterminds/glide/...

# Vendor dependencies
cd ${basedir}
glide install

# Run tests
go test ./...

cp -Rvf `pwd`/gopath ${outdir}/
