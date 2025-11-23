#!/bin/bash

_repo=${1}
_path=${_repo##*/}

rm -rf ${_path}
git clone ${_repo}
cd ${_path}
if [ ! -f go.work ]; then
    go work init
    go work use .
    go work edit -replace github.com/go-ap/storage-conformance-suite=../storage-conformance-suite
fi
go mod tidy
go test -tags conformance -cover -race -count=1 -json ./... | go run github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt -showteststatus
