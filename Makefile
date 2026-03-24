.PHONY: build build-windows build-linux test test-coverage clean help fmt lint deps run skill

# Binary name
BINARY_NAME=pomelo-db

# Output directory
BIN_DIR=bin

# Binary outputs
BINARY_WINDOWS=$(BIN_DIR)/$(BINARY_NAME).exe
BINARY_LINUX_AMD64=$(BIN_DIR)/$(BINARY_NAME)-linux-amd64
BINARY_LINUX_ARM64=$(BIN_DIR)/$(BINARY_NAME)-linux-arm64

# Skill directory
SKILL_SRC=.claude/skills/pomelo-db
SKILL_DST=$(HOME)/.claude/skills/pomelo-db

GOPROXY ?= https://goproxy.cn,https://goproxy.io,direct
ALPINE_MIRROR ?= mirrors.aliyun.com

# Go build flags
LDFLAGS=-ldflags "-s -w"

# Default target
.DEFAULT_GOAL := help

# Detect platform binary suffix
ifeq ($(OS),Windows_NT)
  BINARY_SUFFIX=.exe
else
  BINARY_SUFFIX=
endif
BINARY_LOCAL=$(BIN_DIR)/$(BINARY_NAME)$(BINARY_SUFFIX)

# Build local binary (current OS/arch)
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BINARY_LOCAL) .
	@echo "Build complete: $(BINARY_LOCAL)"

# Build Windows binary
build-windows:
	@echo "Building Windows binary..."
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_WINDOWS) .
	@echo "Build complete: $(BINARY_WINDOWS)"

# Build Linux binaries (AMD64 + ARM64) using Docker buildx
build-linux:
	@echo "Building Linux binaries (AMD64 + ARM64) using Docker buildx..."
	@echo "GOPROXY: $(GOPROXY)"
	@echo "ALPINE_MIRROR: $(ALPINE_MIRROR)"
	@mkdir -p $(BIN_DIR)
	docker buildx build -f Dockerfile.builder \
		--platform linux/amd64,linux/arm64 \
		--build-arg GOPROXY=$(GOPROXY) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		--output type=local,dest=$(BIN_DIR) \
		.
	@mv $(BIN_DIR)/linux_amd64/$(BINARY_NAME) $(BINARY_LINUX_AMD64) 2>/dev/null || true
	@mv $(BIN_DIR)/linux_arm64/$(BINARY_NAME) $(BINARY_LINUX_ARM64) 2>/dev/null || true
	@rm -rf $(BIN_DIR)/linux_amd64 $(BIN_DIR)/linux_arm64
	@echo ""
	@echo "All Linux binaries built successfully:"
	@ls -lh $(BIN_DIR)/*-linux-* 2>/dev/null || dir $(BIN_DIR)\*-linux-*

# Run the application (for development)
# Usage: make run ARGS="--datasource test --execute 'SELECT 1'"
run:
	@echo "Running $(BINARY_NAME)..."
	@if [ -z "$(ARGS)" ]; then \
		go run . --help; \
	else \
		go run . $(ARGS); \
	fi

# Run tests
# Usage: make test coverage=true
test:
	@if [ "$(coverage)" = "true" ]; then \
		echo "Running tests with coverage..."; \
		go test -v -coverprofile=coverage.out ./...; \
		go tool cover -html=coverage.out -o coverage.html; \
		echo "Coverage report generated: coverage.html"; \
	else \
		echo "Running tests..."; \
		go test -v ./...; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Code formatted"

# Run linters
lint:
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}
	golangci-lint run ./...
	@echo "Linting complete"

# Install/update dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies installed"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR) coverage.out coverage.html
	go clean
	@echo "Clean complete"

# Install skill and binary
skill: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/..."
	cp $(BINARY_LOCAL) /usr/local/bin/$(BINARY_NAME)$(BINARY_SUFFIX)
	@echo "Installing pomelo-db skill to $(SKILL_DST)..."
	@mkdir -p $(SKILL_DST)
	rsync -av --delete $(SKILL_SRC)/ $(SKILL_DST)/
	@echo "Done. Binary: /usr/local/bin/$(BINARY_NAME)$(BINARY_SUFFIX) | Skill: $(SKILL_DST)"

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Development:"
	@echo "  run                - Run the application"
	@echo "  test               - Run tests"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  fmt                - Format code with gofmt"
	@echo "  lint               - Run golangci-lint"
	@echo "  deps               - Install/update dependencies"
	@echo "  clean              - Clean build artifacts"
	@echo "  skill              - Install skill to Claude Code (~/.claude/skills/)"
	@echo ""
	@echo "Build:"
	@echo "  build              - Build binary for current platform"
	@echo "  build-windows      - Build Windows binary (pomelo-db.exe)"
	@echo "  build-linux        - Build Linux binaries (AMD64 + ARM64 via Docker buildx)"
	@echo ""
	@echo "Output:"
	@echo "  Local:    $(BINARY_LOCAL)"
	@echo "  Windows:  bin/$(BINARY_NAME).exe"
	@echo "  Linux:    bin/$(BINARY_NAME)-linux-{amd64,arm64}"
	@echo ""
	@echo "Environment variables:"
	@echo "  GOPROXY           - Go module proxy (default: goproxy.cn)"
	@echo "  ALPINE_MIRROR     - Alpine mirror for Docker builds (default: mirrors.aliyun.com)"
	@echo ""
	@echo "Help:"
	@echo "  help               - Show this help"
