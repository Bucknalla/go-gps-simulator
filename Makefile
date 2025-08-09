# Go GPS Simulator Build Configuration

# Binary name
BINARY_NAME=gps-simulator

# Get version information from git
GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Determine version: use git tag if available, otherwise use "dev"
ifeq ($(GIT_TAG),)
    VERSION := dev
else
    VERSION := $(GIT_TAG)
endif

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)"

# Default target
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) version $(VERSION) ($(GIT_COMMIT))"
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build for release (with optimizations)
.PHONY: build-release
build-release:
	@echo "Building $(BINARY_NAME) version $(VERSION) ($(GIT_COMMIT)) - Release"
	go build $(LDFLAGS) -ldflags "-s -w -X main.Version=$(VERSION) -X main.Commit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" -o $(BINARY_NAME) .

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*.exe
	rm -f coverage.*
	rm -f *.gpx

# Show version information that would be built
.PHONY: version-info
version-info:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Install binary to GOPATH/bin
.PHONY: install
install:
	go install $(LDFLAGS) .

# Run tests
.PHONY: test
test:
	go test ./...

# Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary with version information"
	@echo "  build-release - Build optimized release binary"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  clean         - Remove build artifacts"
	@echo "  install       - Install to GOPATH/bin"
	@echo "  test          - Run tests"
	@echo "  version-info  - Show version information"
	@echo "  help          - Show this help message"
