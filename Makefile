BIN_DIR ?= bin
BINARY ?= $(BIN_DIR)/limavm-tui
GO ?= go

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS = -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(BUILD_DATE)

.PHONY: all build run install test vet clean docker-build help

all: build

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the limavm-tui static binary
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/limavm-tui
	@echo "Built $(BINARY)"

run: build ## Build and run limavm-tui
	./$(BINARY)

install: ## Install binary to $(GOPATH)/bin or $(GOBIN)
	CGO_ENABLED=0 $(GO) install -trimpath -ldflags "$(LDFLAGS)" ./cmd/limavm-tui
	@echo "Installed limavm-tui to $(shell $(GO) env GOPATH)/bin"

test: ## Run unit tests
	$(GO) test -v -race ./...

vet: ## Run go vet and check code formatting
	$(GO) vet ./...
	@test -z "$$($(GO)fmt -l .)" || (echo "Unformatted files:" && $(GO)fmt -l . && exit 1)

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)

docker-build: ## Build Docker container for cross-compiling
	docker build -t limavm-tui-builder:latest .
