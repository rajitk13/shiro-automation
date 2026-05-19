#!/bin/bash
set -e

echo "Setting up development environment..."

# Install pre-commit
if command -v pre-commit &> /dev/null; then
    echo "pre-commit already installed"
else
    echo "Installing pre-commit..."
    pip install pre-commit
fi

# Install pre-commit hooks
echo "Installing pre-commit hooks..."
pre-commit install

# Install golangci-lint
if command -v golangci-lint &> /dev/null; then
    echo "golangci-lint already installed"
else
    echo "Installing golangci-lint..."
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
fi

echo "Development environment setup complete!"
echo ""
echo "Run 'pre-commit run --all-files' to lint all files"
echo "Run 'golangci-lint run' to run Go linter"
