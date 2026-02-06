#!/bin/bash

# Configuration
REPO="SaAkSin/arti-ssh-proxy"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="arti-ssh-agent"
SERVICE_NAME="arti-ssh"

# Detect OS and Arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$OS" != "linux" ] && [ "$OS" != "darwin" ]; then
    echo "Error: Only Linux and macOS are supported."
    exit 1
fi

case "$OS" in
    linux)
        OS_KEY="linux"
        ;;
    darwin)
        OS_KEY="darwin"
        ;;
esac

case "$ARCH" in
    x86_64)
        ARCH_KEY="amd64"
        ;;
    aarch64|arm64)
        ARCH_KEY="arm64"
        ;;
    *)
        echo "Error: Unsupported architecture $ARCH"
        exit 1
        ;;
esac

ASSET_NAME="arti-ssh-agent-${OS_KEY}-${ARCH_KEY}"

# Function to get latest version
get_latest_version() {
    curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Version selection
VERSION=$1
if [ -z "$VERSION" ]; then
    echo "Fetching latest version..."
    VERSION=$(get_latest_version)
fi

if [ -z "$VERSION" ]; then
    echo "Error: Could not determine version."
    exit 1
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$ASSET_NAME"

echo "Downloading $ASSET_NAME ($VERSION)..."
curl -L -o "$BINARY_NAME" "$DOWNLOAD_URL"

if [ $? -ne 0 ]; then
    echo "Error: Download failed."
    exit 1
fi

echo "Installing to $INSTALL_DIR..."
chmod +x "$BINARY_NAME"
sudo mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"

echo "Installation complete!"
echo "Run '$BINARY_NAME -url wss://...' to start."
