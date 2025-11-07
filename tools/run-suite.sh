#!/bin/bash

_repo=${1}
_path=${_repo##*/}

git clone ${_repo}
cd ${_path}
go work init
go work use .
go work edit -replace github.com/go-ap/storage-conformance-suite=../storage-conformance-suite
go mod tidy
go test -tags conformance -cover -race -count=1 -json ./... | go run github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt -showteststatus
