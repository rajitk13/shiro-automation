#!/bin/bash
set -e

echo "Detecting platform..."

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)     OS_NAME="linux";;
  Darwin*)    OS_NAME="darwin";;
  CYGWIN*)    OS_NAME="windows";;
  MINGW*)     OS_NAME="windows";;
  *)          echo "Unsupported OS: $OS"; exit 1;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)     ARCH_NAME="amd64";;
  i686)       ARCH_NAME="386";;
  aarch64)    ARCH_NAME="arm64";;
  arm64)      ARCH_NAME="arm64";;
  *)          echo "Unsupported architecture: $ARCH"; exit 1;;
esac

echo "Detected: $OS_NAME-$ARCH_NAME"

# Determine binary name
if [ "$OS_NAME" = "windows" ]; then
  BINARY_NAME="shiro-${OS_NAME}-${ARCH_NAME}.exe"
else
  BINARY_NAME="shiro-${OS_NAME}-${ARCH_NAME}"
fi

echo "Downloading $BINARY_NAME..."

# Download from GitLab CI artifacts
DOWNLOAD_URL="https://gitlab.com/rajitk13/shiro-automation/-/jobs/artifacts/main/raw/dist/${BINARY_NAME}?job=build"

# Download
curl -LO "$DOWNLOAD_URL"

# Verify download
if [ ! -f "$BINARY_NAME" ]; then
  echo "Failed to download binary"
  exit 1
fi

# Make executable (not for Windows)
if [ "$OS_NAME" != "windows" ]; then
  chmod +x "$BINARY_NAME"
fi

# Install to PATH
INSTALL_DIR="/usr/local/bin"
if [ -d "$INSTALL_DIR" ] && [ -w "$INSTALL_DIR" ]; then
  sudo mv "$BINARY_NAME" "$INSTALL_DIR/shiro"
  echo "Installed to $INSTALL_DIR/shiro"
else
  echo "Installing to current directory as shiro"
  mv "$BINARY_NAME" shiro
  echo "Add to PATH: export PATH=\"$(pwd):$PATH\""
fi

echo "Shiro installed successfully!"
echo "Run: shiro help"
