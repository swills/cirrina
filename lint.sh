#!/bin/sh

SCRIPT=$(readlink -f "$0")
SCRIPTDIR=$(dirname "$SCRIPT")
cd ${SCRIPTDIR}

. ./.venv/bin/activate

export GOROOT=/usr/local/go121
if [ ! -d /tmp/cirrinagopath ]; then
  mkdir /tmp/cirrinagopath
fi
export GOPATH=/tmp/cirrinagopath
export PATH=${GOROOT}/bin:${PATH}:${GOPATH}/bin
export GOMAXPROCS=$(sysctl -n hw.ncpu)

idprio -t nice gci write --skip-generated --skip-vendor -s standard -s default -s 'prefix(cirrina)' cirrinad cirrinactl
idprio -t nice pre-commit run --all-files

idprio -t nice go test -v ./... -coverprofile=coverage.txt -covermode count -tags test > /dev/null
idprio -t nice go tool cover -func coverage.txt | tail -n 1

idprio -t nice go test -race ./cirrinad
