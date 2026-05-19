#!/bin/bash
set -e

echo "Building shiro..."
go build -o shiro ./cmd/runtime

echo "Installing shiro to /usr/local/bin..."
sudo cp shiro /usr/local/bin/
sudo chmod +x /usr/local/bin/shiro

echo "Shiro installed successfully!"
echo "Run 'shiro help' to get started"
