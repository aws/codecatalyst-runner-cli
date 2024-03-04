.PHONY: all
all: deps format lint test

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
	go test -json -v -coverprofile=coverage.txt -test.timeout=10m ./... 2>&1 | gotestfmt

.PHONY: test-short
test-short:
	go test -json -v -coverprofile=coverage.txt -short ./... 2>&1 | gotestfmt

.PHONY: upgrade
upgrade:
	@go get -t -u ./...
	go mod tidy

.PHONY: clean
clean:
	echo "ok"
