#!/bin/bash

# goimports test
diff <(goimports -d $(find . -type f -name '*.go' -not -path "*/vendor/*")) <(printf "")

if [ $? -ne 0 ]; then
echo "goimports format error" >&2
exit 1
fi

# run go test
go test -v -mod=vendor ./...

if [ $? -ne 0 ]; then
echo "go test fail" >&2
exit 1
fi

# go test with race condition option and check coverage
go test ./... -race -coverprofile cover.out -covermode=atomic

if [ $? -ne 0 ]; then
echo "go test coverage fail" >&2
exit 1
fi

if [ -f cover.out ]; then
    go tool cover -func cover.out > total.out
fi

# notify to coveralls
goveralls -coverprofile=cover.out -service=travis-ci

if [ $? -ne 0 ]; then
echo "go update test coverage fail" >&2
exit 1
fi

# remove
rm total.out cover.out

exit 0
