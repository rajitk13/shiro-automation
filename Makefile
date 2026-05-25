.PHONY: build build-runtime test clean lint help install-deps install

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-X main.version=$(VERSION)"

# Build targets
build: build-runtime

build-runtime:
	go build $(LDFLAGS) -o shiro ./cmd/runtime

# Install shiro to PATH
install:
	./scripts/install.sh

# Run tests
test:
	go test -v -cover ./...

# Clean build artifacts
clean:
	rm -f shiro
	rm -rf .shiro/

# Run linter
lint:
	golangci-lint run

# Install development dependencies
install-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
	pip install pre-commit
	pre-commit install

# Run pre-commit hooks
pre-commit:
	pre-commit run --all-files

# Show help
help:
	@echo "Shiro - AI-Native CI Workflow Runtime"
	@echo ""
	@echo "Available commands:"
	@echo "  make build          Build runtime binary"
	@echo "  make build-runtime  Build runtime binary"
	@echo "  make install        Install shiro to /usr/local/bin (requires sudo)"
	@echo "  make test           Run tests with coverage"
	@echo "  make clean          Remove build artifacts"
	@echo "  make lint           Run linter"
	@echo "  make install-deps   Install development dependencies"
	@echo "  make pre-commit     Run pre-commit hooks"
	@echo "  make help           Show this help message"
