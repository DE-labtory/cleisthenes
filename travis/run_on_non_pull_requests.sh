#!/bin/bash

# goimports test
diff <(goimports -d $(find . -type f -name '*.go' -not -path "*/vendor/*")) <(printf "")

if [ $? -ne 0 ]; then
echo "goimports format error" >&2
exit 1
fi

# run go test
go test -race -p 1 -mod=vendor ./...

if [ $? -ne 0 ]; then
echo "go test fail" >&2
exit 1
fi

# generate go test coverage file
go test ./... -race -p 1 -coverprofile cover.out -covermode=atomic

if [ $? -ne 0 ]; then
echo "go test coverage fail" >&2
exit 1
fi

# notify to coveralls
goveralls -coverprofile=coverage.out -service=travis-ci

if [ $? -ne 0 ]; then
echo "go update test coverage fail" >&2
exit 1
fi

exit 0
