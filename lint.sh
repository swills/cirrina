#!/bin/sh

SCRIPT=$(readlink -f "$0")
SCRIPTDIR=$(dirname "$SCRIPT")
cd ${SCRIPTDIR}

. ./.venv/bin/activate

export GOPATH=/tmp/cirrinagopath
export PATH=${PATH}:${GOPATH}/bin
export GOMAXPROCS=12

gci write --skip-generated --skip-vendor -s standard -s default -s 'prefix(cirrina)' cirrinad cirrinactl
nice pre-commit run --all-files

go test -v ./... -coverprofile=coverage.txt -covermode count -tags test > /dev/null
go tool cover -func coverage.txt | tail -n 1
