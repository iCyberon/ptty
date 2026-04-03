.PHONY: build run test lint clean build-all

BINARY=ptty
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/ptty

run: build
	./$(BINARY)

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY)-windows-amd64.exe

build-all: clean
	go build $(LDFLAGS) -o $(BINARY) ./cmd/ptty
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 ./cmd/ptty
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-windows-amd64.exe ./cmd/ptty
