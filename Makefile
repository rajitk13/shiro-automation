.PHONY: build build-runtime build-webhook test clean lint help install-deps install

# Build targets
build: build-runtime build-webhook

build-runtime:
	go build -o shiro ./cmd/runtime

build-webhook:
	go build -o webhook-server ./cmd/webhook-server

# Install shiro to PATH
install:
	./scripts/install.sh

# Run tests
test:
	go test -v -cover ./...

# Clean build artifacts
clean:
	rm -f shiro webhook-server
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
	@echo "  make build          Build both runtime and webhook server"
	@echo "  make build-runtime  Build runtime binary"
	@echo "  make build-webhook  Build webhook server binary"
	@echo "  make install        Install shiro to /usr/local/bin (requires sudo)"
	@echo "  make test           Run tests with coverage"
	@echo "  make clean          Remove build artifacts"
	@echo "  make lint           Run linter"
	@echo "  make install-deps   Install development dependencies"
	@echo "  make pre-commit     Run pre-commit hooks"
	@echo "  make help           Show this help message"
