NAME=vault-fernet-locksmith
VERSION=$(shell git describe --tags --always --dirty)
VERSIONSTRING="github.com/aevox/${NAME}/cmd.version=${VERSION}"

default: build

build:
	mkdir -p bin
	go build -ldflags '-X ${VERSIONSTRING}' -o bin/${NAME} .

build-static:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static" -X ${VERSIONSTRING}' -o bin/${NAME} .
