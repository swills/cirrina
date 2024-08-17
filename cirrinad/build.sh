#!/bin/sh


# sqlite3 cirrina.sqlite .dump > my_vms_dump.sql

#export CGO_ENABLED=0

GITINSTALLED=$(command git >/dev/null 2>&1; echo $?)

if [ ${GITINSTALLED} -eq 1 ]; then
  INGIT=$(git rev-parse --is-inside-work-tree)
  if [ "${INGIT}" == "true" ]; then
    VER=$(git rev-parse --short HEAD)
    R=$(git diff-index --quiet HEAD -- ; echo $?)
    if [ ${R} -ne 0 ]; then
      VER=${VER}-dirty
    fi
  else
    VER="unknown"
  fi
else
  VER="unknown"
fi

rm cirrinad ; go122 build -ldflags="-X main.mainVersion=${VER} -s -w -extldflags -static" .
