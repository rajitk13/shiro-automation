#!/bin/bash

# Setup script for Shiro development environment

echo "Setting up Shiro development environment..."

# Configure Git hooks path
git config core.hooksPath .githooks
echo "✓ Git hooks configured to use .githooks/"

# Make pre-commit hook executable
chmod +x .githooks/pre-commit
echo "✓ Pre-commit hook made executable"

# Check for golangci-lint
if ! command -v golangci-lint &> /dev/null; then
    echo "⚠ golangci-lint not found"
    echo "  Install it from: https://golangci-lint.run/usage/install/"
    echo "  macOS: brew install golangci-lint"
    echo "  Linux: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$(go env GOPATH)/bin"
else
    echo "✓ golangci-lint found at $(which golangci-lint)"
fi

echo "Setup complete! Git hooks will now run automatically before commits."
