.PHONY: build test lint fmt install clean snapshot release

BIN        := timebombs
PKG        := ./cmd/timebombs
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -s -w -X main.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BIN) $(PKG)

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	# Format only buildable Go packages so testdata/ fixtures stay intact.
	gofmt -s -w $(shell go list -f '{{.Dir}}' ./...)

install:
	go install -ldflags "$(LDFLAGS)" $(PKG)

clean:
	rm -rf bin/ dist/

# Build snapshot artifacts with goreleaser (no tag required).
snapshot:
	goreleaser release --snapshot --clean

# Publish a release. Requires GITHUB_TOKEN and a pushed tag.
release:
	goreleaser release --clean
