BINARY := dirstral

.PHONY: all build run fmt vet lint test check ci install clean

all: check

build:
	go build -o bin/$(BINARY) ./cmd/dirstral

run:
	go run ./cmd/dirstral

fmt:
	gofmt -w ./cmd ./internal ./tests

vet:
	go vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint is required. Install: https://golangci-lint.run/welcome/install/" && exit 1)
	golangci-lint run

test:
	go test ./...

check: fmt vet lint test

ci: vet test

install:
	go install ./cmd/dirstral

clean:
	rm -rf bin
