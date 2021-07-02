MAKEFLAGS = -j1
SHELL := /bin/bash

# default target

all: build



###################
## BUILD SECTION ##
###################
export GO111MODULE := on
export CGO_ENABLED := 0
export GOPROXY     := direct
export GOSUMDB     := off

build:
	@echo "Building for local consumption"
	@go build pkg/goflip/*.go

build_rpi:
	@echo "Building for Raspberry PI"
	export GOOS=linux
	export GOARCH=arm
	export GOARM=6
	@go build pkg/goflip/*.go


godeps:
	@echo "Getting go modules"
	@go mod download

clean: ## clean build output
	@rm -rf bin/