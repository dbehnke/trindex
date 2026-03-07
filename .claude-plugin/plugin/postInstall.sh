#!/usr/bin/env bash
set -euo pipefail

REPO="dbehnke/trindex"
INSTALL_DIR="${CLAUDE_PLUGIN_ROOT}/bin"
BINARY="${INSTALL_DIR}/trindex"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

# Get latest release version
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
TARBALL="trindex_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

echo "Installing trindex ${VERSION} (${OS}/${ARCH})..."
mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" | tar -xz -C "$INSTALL_DIR" trindex
chmod +x "$BINARY"
echo "trindex installed to ${BINARY}"
