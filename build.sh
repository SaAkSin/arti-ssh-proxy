#!/bin/bash

# Build for Linux x86_64
echo "Building for Linux amd64..."
GOOS=linux GOARCH=amd64 go build -o bin/arti-ssh-agent-amd64 ./cmd/agent

# Build for Linux ARM64
echo "Building for Linux arm64..."
GOOS=linux GOARCH=arm64 go build -o bin/arti-ssh-agent-arm64 ./cmd/agent

echo "Build complete. Binaries are in ./bin/"
ls -lh ./bin/
