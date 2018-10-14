all: client server 
.PHONY: all

protobuf:
	cd rpc && $(MAKE)
.PHONY: protobuf



SHELL=/bin/bash
VERSION=$(shell cat VERSION)
GITBRANCH=$(shell git rev-parse --abbrev-ref HEAD)
GITCOMMIT=$(shell git log -1 --pretty=format:%h)
BUILD_TIME=$(shell date '+%Y-%m-%dT%H:%M:%S%z')

local: client server
.PHONY: local

client:
	go build -o client/client -v -ldflags "-X main.Version=${VERSION} -X main.GitBranch=${GITBRANCH}  -X main.GitCommit=${GITCOMMIT} -X main.BuildTime=${BUILD_TIME}" ./client
.PHONY: client

server:
	go build -o server/server -v -ldflags "-X main.Version=${VERSION} -X main.GitBranch=${GITBRANCH}  -X main.GitCommit=${GITCOMMIT} -X main.BuildTime=${BUILD_TIME}" ./server
.PHONY: server


clean:
	rm -f client/client
	rm -f server/server
.PHONY: clean

distclean: clean
	rm -f client/client.log
	rm -f server/server.log
.PHONY: distclean

PROJECT = github.com/webull/go-replica
BUILD_IMG=golang:1.10.1
build:
	docker run --rm  -v $(shell pwd):/go/src/${PROJECT} -w /go/src/${PROJECT} --rm ${BUILD_IMG}   bash -c "make local"
