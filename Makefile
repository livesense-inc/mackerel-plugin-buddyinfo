VERSION = v0.0.1
REPO = mackerel-plugin-buddyinfo
OWNER = livesense-inc
BIN = $(REPO)
CURRENT_REVISION ?= $(shell git rev-parse --short HEAD)
LDFLAGS = -w -s -X 'main.version=$(VERSION)' -X 'main.gitcommit=$(CURRENT_REVISION)'

all: clean test build

test:
	go test ./internal

build:
	go build -ldflags="$(LDFLAGS)" -trimpath -o bin/$(BIN) ./

cross: clean
	goxc -build-gcflags="-trimpath=$(shell pwd)" -build-ldflags="$(LDFLAGS)" -d=./dist

deploy: cross
	ghr -u $(OWNER) -r $(REPO) $(VERSION) ./dist/snapshot

builddep:
	GO111MODULE=off go get -v -u github.com/laher/goxc github.com/tcnksm/ghr

clean:
	rm -rf bin dist

.PHONY: test build cross deploy dep depup clean
