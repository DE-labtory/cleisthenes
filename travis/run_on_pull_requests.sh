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

# total test coverage
cover=$(awk '{print $3}' total.out | tail -n 1 | cut -d . -f 1)

# project criteria
criteria=80

# remove
rm total.out cover.out

if [ "$cover" -gt "$criteria" ]; then
    echo "Success to cover test coverage - got : $cover"
else
    echo "Test coverage criteria fail - got : $cover expected : $criteria"
    exit 1
fi

exit 0
