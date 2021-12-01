SHELL=/usr/bin/env bash

YELLOW = "\e[33;1m"

all: build-deps build
.PHONY: all build

unexport GOFLAGS

LOTUS_PATH:=./extern/lotus/

build-deps:
	git submodule update --init --recursive
	make -C ${LOTUS_PATH} deps

build: build-deps
	go mod tidy
	rm -rf lotus-redo
	rm -rf lotus-wdpsot
	go build -o lotus-redo ./cmd/lotus-redo/main.go
	go build -o lotus-wdpsot ./cmd/lotus-wdpost/main.go
	echo  -e $(YELLOW) "run 'sudo make install' add binary in your PATH."
install:
	install -C lotus-redo /usr/local/bin/lotus-redo
	install -C lotus-wdpsot /usr/local/bin/lotus-wdpsot

.PHONY: clean
clean:
	-rm -f lotus-redo lotus-wdpsot /usr/local/bin/lotus-redo /usr/local/bin/lotus-wdpsot
