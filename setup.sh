#!/bin/sh

export GOROOT=/usr/local/go119
export GOPATH=/tmp/cirrinagopath
export PATH=${PATH}:${GOPATH}/bin

go mod download -x

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

cd cirrina
protoc \
	--go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	cirrina.proto
