#!/bin/sh

export GOROOT=/usr/local/go119
export GOPATH=/tmp/cirrinagopath
export PATH=${PATH}:${GOPATH}/bin

cd cirrina
protoc \
	--go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	cirrina.proto
