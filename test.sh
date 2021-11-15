#!/bin/sh
GO111MODULE=on go test -v ./... -coverprofile="$1/cover.out" | go-junit-report > "$1/report.xml"
gocov convert "$1/cover.out" | gocov-xml > "$1/coverage.xml"
gcov2lcov -infile="$1/cover.out" -outfile="$1/coverage.lcov"

GO111MODULE=on go vet ./...
gofmt -l $(find . -type f -name '*.go'| grep -v "/vendor/")
[ "`gofmt -l $(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]

./bin/test-attribution-txt.sh
