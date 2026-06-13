# Makefile for the NetLoc8 MCP server.
#
# --------------------------------------------------------------------------
# Go Concept: Cross-Compilation
# --------------------------------------------------------------------------
# Go can build binaries for any OS/architecture from any machine by setting
# two environment variables:
#
#   GOOS   = target operating system (linux, darwin, windows)
#   GOARCH = target CPU architecture (amd64, arm64)
#
# This means you can build a Linux binary on your Mac, or a Windows binary
# on Linux. No special toolchain needed — the Go compiler handles it all.
#
# "darwin" is macOS's internal name (named after Charles Darwin).
# "amd64" is 64-bit Intel/AMD. "arm64" is Apple Silicon (M1/M2/M3/M4).
# --------------------------------------------------------------------------

# The name of the output binary.
BINARY_NAME = netloc8-mcp

# Default target — what runs when you just type "make".
.PHONY: build
build:
	go build -o $(BINARY_NAME) .

# Run go vet (static analysis — catches common mistakes).
.PHONY: vet
vet:
	go vet ./...

# Run all tests.
.PHONY: test
test:
	go test -v -count=1 ./...

# Clean build artifacts.
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*

# Build for all platforms (macOS, Linux, Windows × Intel and ARM).
# CGO_ENABLED=0 produces a fully static binary with no C library
# dependencies — it runs on any machine without installing anything.
.PHONY: build-all
build-all:
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64  .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64  .
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64   .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o $(BINARY_NAME)-linux-arm64   .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o $(BINARY_NAME)-windows-arm64.exe .
