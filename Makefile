NAME=locksmith
VERSION=$(shell git describe --tags --always)

default: build

build:
	mkdir -p bin
	go build -ldflags '-X main.locksmithVersion=${VERSION}' -o bin/${NAME} .

build-static:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static" -X main.locksmithVersion=${VERSION}' -o bin/${NAME} .

build-static-all:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static" -X main.locksmithVersion=${VERSION}' -o bin/${NAME} .
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static" -X main.locksmithVersion=${VERSION}' -o bin/${NAME}-bootstrap ./boostrap/
