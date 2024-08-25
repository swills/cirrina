#!/bin/sh

CLOCKBIN=/usr/local/libexec/poudriere/clock
CLOCK="${CLOCKBIN} -monotonic"

if [ -f ${CLOCKBIN} ]; then
  START_TIME=$(${CLOCK})
fi

elapsed_time () {
  END_TIME=$(${CLOCK})
  _elapsed=$((${END_TIME} - ${START_TIME}))
  seconds=$((${_elapsed} % 60))
  minutes=$(((${_elapsed} / 60) % 60))
  hours=$((${_elapsed} / 3600))
  _duration=$(printf "%02d:%02d:%02d" ${hours} ${minutes} ${seconds})
  echo "Elapsed time: ${_duration}"
}

SCRIPT=$(readlink -f "$0")
SCRIPTDIR=$(dirname "$SCRIPT")
cd ${SCRIPTDIR}

. ./.venv/bin/activate

export GOROOT=/usr/local/go122
if [ ! -d /tmp/cirrinagopath ]; then
  mkdir /tmp/cirrinagopath
fi
export GOPATH=/tmp/cirrinagopath
export PATH=${GOROOT}/bin:${PATH}:${GOPATH}/bin
NCPU=$(getconf NPROCESSORS_ONLN)
RESERVED_CPUS=4
export GOMAXPROCS=$((${NCPU}-${RESERVED_CPUS}))

idprio -t nice gci write --skip-generated --skip-vendor -s standard -s default -s 'prefix(cirrina)' cirrinad cirrinactl
idprio -t nice pre-commit run --all-files

idprio -t nice go test -v ./... -coverprofile=coverage.txt -covermode count -tags test > /dev/null
idprio -t nice go tool cover -func coverage.txt | tail -n 1

idprio -t nice go test -race ./cirrinad

if [ -f ${CLOCKBIN} ]; then
  elapsed_time
fi
