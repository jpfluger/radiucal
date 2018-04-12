BIN=bin/
SOURCE=src/
PLUGIN=plugins/
SRC=$(shell find $(SOURCE) -type f) $(shell find $(PLUGIN) -type f)
PLUGINS=$(shell ls $(PLUGIN) | grep -v "common.go")

VERSION=
ifeq ($(VERSION),)
	VERSION=master
endif
export GOPATH := $(PWD)/vendor
.PHONY: tools

all: clean deps modules radiucal tools format

deps:
	git submodule update --init --recursive

modules: $(PLUGINS)

$(PLUGINS):
	@echo $@
	go build --buildmode=plugin -o $(BIN)$@.rd $(PLUGIN)$@/plugin.go

radiucal:
	go build -o $(BIN)radiucal -ldflags '-X main.vers=$(VERSION)' $(SOURCE)main.go

format:
	@echo $(SRC)
	exit $(shell gofmt -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

tools:
	cd tools && make -C .
