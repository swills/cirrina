#!/bin/sh

export TZ=UTC

git tag `date +v0.%y.%m.%d.%H.%M.%S`
git push --tags
