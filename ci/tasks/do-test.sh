#!/bin/sh
basedir=`pwd`/gopath/src/github.com/ecsteam/buildpack-usage

set -e
set -x

export GOPATH=`pwd`/gopath

# Install Glide
go get -u github.com/Masterminds/glide/...

# Vendor dependencies
cd ${basedir}
glide install

# Run tests
go test ./...
