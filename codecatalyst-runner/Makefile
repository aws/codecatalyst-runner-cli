INSTALL_DIR ?= /usr/local/bin
TAGS ?= all
VERSION := $(shell cat VERSION)
COMMIT_ID = $(shell git rev-parse HEAD)

.PHONY: all
all: deps format lint test build

.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: format
format:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run --timeout=5m

.PHONY: test
test:
	go test -json -v -coverprofile=coverage.txt  --tags=$(TAGS) -test.timeout=30m ./... 2>&1 | gotestfmt

.PHONY: test-short
test-short:
	go test -json -v -coverprofile=coverage.txt  --tags=$(TAGS) -short ./... 2>&1 | gotestfmt

.PHONY: build
build:
	go build -o dist/local/ccr main.go

.PHONY: install
install: build
	@cp dist/local/ccr $(INSTALL_DIR)/ccr
	@chmod 755 $(INSTALL_DIR)/ccr
	@ccr --version

publish: build
	@tar -cpf dist/local/ccr-$(COMMIT_ID).tar.gz -C dist/local ccr
	CCR_SHA256=$$(sha256sum dist/local/ccr-$(COMMIT_ID).tar.gz | head -c 64) CCR_COMMIT=$(COMMIT_ID) CCR_VERSION=$(VERSION) envsubst < homebrew/ccr.rb.tmpl > dist/local/ccr.rb
	brew amazon-upload dist/local/ccr-$(COMMIT_ID).tar.gz

.PHONY: upgrade
upgrade:
	@go get -t -u ./...
	go mod tidy

.PHONY: clean
clean:
	rm -rf dist
