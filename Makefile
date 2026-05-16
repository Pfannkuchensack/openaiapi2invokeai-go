BINARY := invoke-openai-proxy
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/version.Version=$(VERSION) \
	-X github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/version.Commit=$(COMMIT) \
	-X github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/version.Date=$(DATE)

.PHONY: build run clean cross

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/proxy

run: build
	./bin/$(BINARY)

clean:
	rm -rf bin/

cross:
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-linux-amd64 ./cmd/proxy
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-linux-arm64 ./cmd/proxy
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-darwin-arm64 ./cmd/proxy
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-windows-amd64.exe ./cmd/proxy
