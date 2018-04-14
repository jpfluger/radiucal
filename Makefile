BIN=bin/
PLUGIN=plugins/
MAIN=radiucal.go context.go
SRC=$(MAIN) $(shell find $(PLUGIN) -type f | grep "\.go$$")
PLUGINS=$(shell ls $(PLUGIN) | grep -v "common.go")

VERSION=
ifeq ($(VERSION),)
	VERSION=master
endif
export GOPATH := $(PWD)/vendor
.PHONY: tools plugins

all: clean deps plugins radiucal tools format

deps:
	git submodule update --init --recursive

plugins: $(PLUGINS)

$(PLUGINS):
	@echo $@
	go build --buildmode=plugin -o $(BIN)$@.rd $(PLUGIN)$@/plugin.go
	cd $(PLUGIN)$@ && go test -v

radiucal:
	go test -v
	go build -o $(BIN)radiucal -ldflags '-X main.vers=$(VERSION)' $(MAIN)

format:
	@echo $(SRC)
	exit $(shell echo $(SRC) | grep "\.go$$" | gofmt -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

tools:
	cd tools && make -C .
