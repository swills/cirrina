#!/bin/sh

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

rm cirrinactl ; go123 build -v -ldflags="-X cirrina/cirrinactl/cmd.mainVersion=${VER} -s -w -extldflags -static" . || exit 1
