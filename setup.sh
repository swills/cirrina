#!/bin/sh

export GOROOT=/usr/local/go121

if [ ! -d /tmp/cirrinagopath ]; then
  mkdir /tmp/cirrinagopath
fi

export GOPATH=/tmp/cirrinagopath
export PATH=${GOROOT}/bin:${PATH}:${GOPATH}/bin

go mod download -x

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/daixiang0/gci@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install go.uber.org/mock/mockgen@latest

python3 -m venv .venv
./.venv/bin/pip3 install --upgrade pip
./.venv/bin/pip3 -v --no-input --require-virtualenv install pre-commit --no-binary :all:

cd cirrina

protoc \
	--go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	cirrina.proto

cd ..

which direnv > /dev/null

if [ ${?} -eq 0 ]; then
  direnv allow
fi

pre-commit run --all-files
