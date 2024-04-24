#!/bin/sh

SCRIPT=$(readlink -f "$0")
SCRIPTDIR=$(dirname "$SCRIPT")
cd ${SCRIPTDIR}

. ./.venv/bin/activate

export GOPATH=/tmp/cirrinagopath
export PATH=${PATH}:${GOPATH}/bin

gci write --skip-generated --skip-vendor -s standard -s default -s 'prefix(cirrina)' cirrinad cirrinactl
pre-commit run --all-files
