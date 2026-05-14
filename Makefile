PKG = github.com/k1LoW/dirtop
COMMIT = $(shell git rev-parse --short HEAD)
BUILD_LDFLAGS = "-s -w -X $(PKG)/version.Revision=$(COMMIT)"

default: build

ci: depsdev test

build:
	go build -ldflags=$(BUILD_LDFLAGS) -trimpath -o dirtop .

test:
	go test ./... -coverprofile=coverage.out -covermode=count -count=1

lint:
	golangci-lint run ./...
	go vet -vettool=`which gostyle` -gostyle.config=$(PWD)/.gostyle.yml ./...

depsdev:
	go install github.com/Songmu/gocredits/cmd/gocredits@latest
	go install github.com/k1LoW/gostyle@latest

credits: depsdev
	go mod download
	gocredits -skip-missing -w .
	cat _EXTRA_CREDITS >> CREDITS

prerelease_for_tagpr:
	gocredits -skip-missing -w .
	cat _EXTRA_CREDITS >> CREDITS
	git add CHANGELOG.md CREDITS go.mod go.sum

.PHONY: default ci build test lint depsdev credits prerelease_for_tagpr
