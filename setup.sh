#!/bin/sh

export GOROOT=/usr/local/go123

if [ ! -d /tmp/cirrinagopath ]; then
  mkdir /tmp/cirrinagopath
fi

export GOPATH=/tmp/cirrinagopath
export PATH=${GOROOT}/bin:${PATH}:${GOPATH}/bin

go mod download -x

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
go install github.com/daixiang0/gci@v0.13.5
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5
go install go.uber.org/mock/mockgen@v0.5.0

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
