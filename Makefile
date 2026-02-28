BINARY := dirstral

.PHONY: build run test install clean

build:
	go build -o bin/$(BINARY) ./cmd/dirstral

run:
	go run ./cmd/dirstral

test:
	go test ./...

install:
	go install ./cmd/dirstral

clean:
	rm -rf bin
