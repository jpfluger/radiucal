BIN=bin/
PLUGIN=plugins/
MAIN=radiucal.go
SRC=$(MAIN) $(shell find $(PLUGIN) -type f) $(TESTS)
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
	cd $(PLUGIN)$@ && go test -v

radiucal:
	go build -o $(BIN)radiucal -ldflags '-X main.vers=$(VERSION)' $(MAIN)

format:
	@echo $(SRC)
	exit $(shell gofmt -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)


tools:
	cd tools && make -C .
