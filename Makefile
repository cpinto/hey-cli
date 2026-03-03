BIN_DIR := ./bin
BINARY  := $(BIN_DIR)/hey
MODULE  := ./cmd/hey

.PHONY: build test lint clean install

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) $(MODULE)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BIN_DIR)

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  INSTALL_DIR := /usr/local/bin
else
  INSTALL_DIR := /usr/bin
endif

install: build
	sudo install $(BINARY) $(INSTALL_DIR)/hey
