#!/bin/sh
set -e

REPO="rajitk13/shiro-automation"
INSTALL_DIR="${SHIRO_INSTALL_DIR:-/usr/local/bin}"
INSECURE="${SHIRO_INSECURE_TLS:-}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  mingw*|msys*|cygwin*) OS="windows" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  armv7l) ARCH="arm" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

BINARY="shiro-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY}"
DEST="${INSTALL_DIR}/shiro"

echo "Installing shiro for ${OS}/${ARCH}..."
echo "Downloading from ${DOWNLOAD_URL}"

CURL_OPTS="-fsSL"
if [ "$INSECURE" = "1" ] || [ "$INSECURE" = "true" ]; then
  CURL_OPTS="${CURL_OPTS}k"
fi

curl ${CURL_OPTS} -o "$DEST" "$DOWNLOAD_URL"
chmod +x "$DEST"

echo "✓ shiro installed to $DEST"
echo ""
echo "Run: shiro --help"
