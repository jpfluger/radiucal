BIN=bin/
SOURCE=src/
SRC=$(shell find $(SOURCE) -type f)

VERSION=
ifeq ($(VERSION),)
	VERSION=master
endif
export GOPATH := $(PWD)/vendor

all: clean deps radiacal format

deps:
	git submodule update --init --recursive

radiacal:
	go build -o $(BIN)radiacal -ldflags '-X main.vers=$(VERSION)' $(SOURCE)main.go

format:
	@echo $(SRC)
	exit $(shell gofmt -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
