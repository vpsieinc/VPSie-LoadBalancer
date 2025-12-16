.PHONY: all build build-agent build-images build-amd64 build-arm64 test clean help

VERSION ?= 1.0.0
GOARCH ?= amd64
GOOS ?= linux

# Binary names
AGENT_BINARY := vpsie-lb-agent

# Directories
BUILD_DIR := build
OUTPUT_DIR := output
CMD_DIR := cmd/agent

# Go build flags
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

all: build

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: build-agent ## Build all binaries

build-agent: ## Build the agent binary
	@echo "Building agent for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BUILD_DIR)/$(AGENT_BINARY)-$(GOARCH) ./$(CMD_DIR)
	@echo "Agent binary created: $(BUILD_DIR)/$(AGENT_BINARY)-$(GOARCH)"

build-agent-all: ## Build agent for all architectures
	@$(MAKE) build-agent GOARCH=amd64
	@$(MAKE) build-agent GOARCH=arm64

build-images: build-amd64 build-arm64 ## Build both amd64 and arm64 images

build-amd64: ## Build amd64 qcow2 image using Packer
	@echo "Building amd64 image..."
	@mkdir -p $(OUTPUT_DIR)/amd64
	cd packer && packer build -var="version=$(VERSION)" debian-amd64.pkr.hcl

build-arm64: ## Build arm64 qcow2 image using Packer
	@echo "Building arm64 image..."
	@mkdir -p $(OUTPUT_DIR)/arm64
	cd packer && packer build -var="version=$(VERSION)" debian-arm64.pkr.hcl

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test -v -race ./pkg/...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -race ./tests/integration/...

fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

lint: ## Run golangci-lint
	@echo "Running linter..."
	golangci-lint run ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(OUTPUT_DIR)
	rm -f coverage.out

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.DEFAULT_GOAL := help
