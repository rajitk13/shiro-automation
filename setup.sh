#!/bin/bash

# Setup script for Shiro development environment

echo "Setting up Shiro development environment..."

# Configure Git hooks path
git config core.hooksPath .githooks
echo "✓ Git hooks configured to use .githooks/"

# Make pre-commit hook executable
chmod +x .githooks/pre-commit
echo "✓ Pre-commit hook made executable"

echo "Setup complete! Git hooks will now run automatically before commits."
