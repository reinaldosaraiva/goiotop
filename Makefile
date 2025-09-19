.PHONY: all build clean test test-coverage lint fmt install uninstall run help
.PHONY: build-all build-linux build-darwin build-windows
.PHONY: docker-build docker-run proto docs

# Variables
BINARY_NAME := goiotop
MAIN_PATH := ./cmd/goiotop
BUILD_DIR := ./bin
DIST_DIR := ./dist
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOVET := $(GOCMD) vet
GOLINT := golangci-lint

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"
BUILD_FLAGS := -v -trimpath

# OS/Arch combinations for cross-compilation
PLATFORMS := linux/amd64 linux/386 linux/arm linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/386

# Default target
all: test build

# Help target
help:
	@echo "GoIOTop - System Monitoring Tool"
	@echo ""
	@echo "Available targets:"
	@echo "  build         Build the binary for current platform"
	@echo "  build-all     Build for all supported platforms"
	@echo "  build-linux   Build for Linux (amd64)"
	@echo "  build-darwin  Build for macOS (amd64/arm64)"
	@echo "  build-windows Build for Windows (amd64)"
	@echo "  test          Run all tests"
	@echo "  test-coverage Run tests with coverage"
	@echo "  test-cli      Run CLI-specific tests"
	@echo "  test-tui      Run TUI-specific tests"
	@echo "  bench         Run benchmarks"
	@echo "  lint          Run linter"
	@echo "  fmt           Format code"
	@echo "  vet           Run go vet"
	@echo "  install       Install binary to GOPATH/bin"
	@echo "  uninstall     Remove installed binary"
	@echo "  clean         Remove build artifacts"
	@echo "  deps          Download dependencies"
	@echo "  tidy          Tidy go modules"
	@echo "  run           Run the application"
	@echo "  run-tui       Run in TUI mode (interactive)"
	@echo "  run-display   Run in display mode (like iotop)"
	@echo "  run-collect   Run in collect mode"
	@echo "  run-help      Show CLI help"
	@echo "  docker-build  Build Docker image"
	@echo "  docker-run    Run in Docker container"
	@echo "  docs          Generate documentation"
	@echo "  proto         Generate protobuf files"
	@echo "  release       Create release artifacts"
	@echo "  version       Display version information"

# Build the project
build: deps
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms
build-all: deps
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1) \
		GOARCH=$$(echo $$platform | cut -d'/' -f2) \
		output="$(DIST_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH" ; \
		if [ "$$GOOS" = "windows" ]; then output="$$output.exe"; fi ; \
		echo "Building $$output..." ; \
		GOOS=$$GOOS GOARCH=$$GOARCH $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $$output $(MAIN_PATH) ; \
	done
	@echo "Cross-platform build complete"

# Platform-specific builds
build-linux: deps
	@echo "Building for Linux..."
	@mkdir -p $(DIST_DIR)
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@echo "Linux build complete"

build-darwin: deps
	@echo "Building for macOS..."
	@mkdir -p $(DIST_DIR)
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "macOS build complete"

build-windows: deps
	@echo "Building for Windows..."
	@mkdir -p $(DIST_DIR)
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "Windows build complete"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

# CLI-specific targets
run-tui: build
	@echo "Running TUI mode..."
	@$(BUILD_DIR)/$(BINARY_NAME) tui

run-display: build
	@echo "Running display mode..."
	@$(BUILD_DIR)/$(BINARY_NAME) display

run-collect: build
	@echo "Running collect mode..."
	@$(BUILD_DIR)/$(BINARY_NAME) collect

run-help: build
	@$(BUILD_DIR)/$(BINARY_NAME) --help

# Run tests
test:
	@echo "Running tests..."
	@$(GOTEST) -v -race -short ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE)

# Run CLI-specific tests
test-cli:
	@echo "Running CLI tests..."
	@$(GOTEST) -v -race ./cmd/goiotop/...
	@$(GOTEST) -v -race ./internal/presentation/...

# Run TUI-specific tests
test-tui:
	@echo "Running TUI tests..."
	@$(GOTEST) -v -race ./internal/presentation/tui/...
	@$(GOTEST) -v -race -run TestTUI ./cmd/goiotop/...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem ./...

# Run linter
lint:
	@echo "Running linter..."
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run ./... ; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/" ; \
		exit 1 ; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	@$(GOFMT) -s -w .
	@$(GOCMD) fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@$(GOVET) ./...

# Install binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete: /usr/local/bin/$(BINARY_NAME)"
	@echo "Run '$(BINARY_NAME) --help' to get started"

# Uninstall binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstall complete"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@rm -f *.test
	@rm -f *.out
	@echo "Clean complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOGET) -v ./...

# Tidy go modules
tidy:
	@echo "Tidying go modules..."
	@$(GOMOD) tidy

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	@$(GOGET) -u ./...
	@$(GOMOD) tidy

# Security scan
security:
	@echo "Running security scan..."
	@if command -v gosec > /dev/null; then \
		gosec ./... ; \
	else \
		echo "gosec not installed. Install it with: go install github.com/securego/gosec/v2/cmd/gosec@latest" ; \
	fi

# Check for vulnerabilities
vuln-check:
	@echo "Checking for vulnerabilities..."
	@$(GOCMD) list -json -m all | nancy sleuth

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):$(VERSION) -t $(BINARY_NAME):latest .

# Docker run
docker-run: docker-build
	@echo "Running in Docker container..."
	@docker run --rm -it --pid host --network host $(BINARY_NAME):latest

# Generate documentation
docs:
	@echo "Generating documentation..."
	@if command -v godoc > /dev/null; then \
		echo "Starting godoc server on http://localhost:6060" ; \
		godoc -http=:6060 ; \
	else \
		echo "godoc not installed. Install it with: go install golang.org/x/tools/cmd/godoc@latest" ; \
	fi

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	@if [ -d "proto" ]; then \
		protoc --go_out=. --go_opt=paths=source_relative \
			--go-grpc_out=. --go-grpc_opt=paths=source_relative \
			proto/*.proto ; \
	else \
		echo "No proto files found" ; \
	fi

# Create release
release: test build-all
	@echo "Creating release $(VERSION)..."
	@mkdir -p release/$(VERSION)
	@cp -r $(DIST_DIR)/* release/$(VERSION)/
	@cp README.md LICENSE release/$(VERSION)/
	@cd release && tar -czf $(BINARY_NAME)-$(VERSION).tar.gz $(VERSION)
	@echo "Release created: release/$(BINARY_NAME)-$(VERSION).tar.gz"

# Display version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/godoc@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install github.com/sonatype-nexus-community/nancy@latest
	@echo "Development environment ready"

# Continuous Integration
ci: deps vet lint test-coverage build

# Quick check before commit
pre-commit: fmt vet lint test

.DEFAULT_GOAL := help