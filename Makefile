EXTENSION ?=
DIST_DIR ?= dist/
GOOS ?= $(shell uname -s | tr "[:upper:]" "[:lower:]")
ARCH ?= $(shell uname -m)
BUILDINFOSDET ?=

DOCKER_REPO := synfinatic
VPNEXIT_NAME := vpnexiter
VPNEXIT_VERSION := $(shell git describe --tags 2>/dev/null $(git rev-list --tags --max-count=1))
VERSION_PKG        := $(shell echo $(VPNEXIT_VERSION) | sed 's/^v//g')
ARCH               := x86_64
LICENSE            := MIT
URL                := https://github.com/synfinatic/vpnexiter
DESCRIPTION        := VPN Exiter: A VPN Exit Manager for Routers
BUILDINFOS         := ($(shell date +%FT%T%z)$(BUILDINFOSDET))
LDFLAGS            := '-X main.version=$(VPNEXIT_VERSION) -X main.buildinfos=$(BUILDINFOS)'
OUTPUT_VPNEXIT     := $(DIST_DIR)vpnexiter-$(VPNEXIT_VERSION)-$(GOOS)-$(ARCH)$(EXTENSION)

ALL: vpnexiter

build: server/vpnexiter.go
	go build ./...

test: test-race vet unittest

PHONY: run
run:
	go run server/*.go

clean:
	rm -f main

clean-go:
	go clean -i -r cache -modcache

vpnexiter: $(OUTPUT_VPNEXIT)

$(OUTPUT_VPNEXIT): prepare
	go build -ldflags $(LDFLAGS) -o $(OUTPUT_VPNEXIT) server/vpnexiter.go

PHONY: docker-run
	docker run -it --rm -p 5000:5000/tcp $(DOCKER_REPO)/$(VPNEXIT_NAME):latest

PHONY: docker-build
docker-build:
	docker build -t $(DOCKER_REPO)/$(VPNEXIT_NAME):latest .

PHONY: docker-clean
docker-clean:
	docker image rm $(DOCKER_REPO)/$(VPNEXIT_NAME):latest

PHONY: docker-shell
docker-shell:
	docker run -it --rm $(DOCKER_REPO)/$(VPNEXIT_NAME):latest /bin/bash

.PHONY: unittest
unittest:
	go test ./...

.PHONY: test-race
test-race:
	@echo checking code for races...
	go test -race ./...

.PHONY: vet
vet:
	@echo checking code is vetted...
	go vet $(shell go list ./...)

.PHONY: prepare
prepare:
	mkdir -p $(DIST_DIR)
