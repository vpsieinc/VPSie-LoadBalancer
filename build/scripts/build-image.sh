#!/bin/bash
set -e

# Build script for creating VPSie Load Balancer images
# Usage: ./build-image.sh [amd64|arm64|all]

VERSION="${VERSION:-1.0.0}"
ARCH="${1:-all}"

echo "=== Building VPSie Load Balancer Images ==="
echo "Version: $VERSION"
echo "Architecture: $ARCH"

# Build agent binary first
echo "Building agent binary..."
cd "$(dirname "$0")/../.."

if [ "$ARCH" = "amd64" ] || [ "$ARCH" = "all" ]; then
    echo "Building amd64 agent..."
    GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o build/vpsie-lb-agent-amd64 ./cmd/agent
fi

if [ "$ARCH" = "arm64" ] || [ "$ARCH" = "all" ]; then
    echo "Building arm64 agent..."
    GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o build/vpsie-lb-agent-arm64 ./cmd/agent
fi

# Build images with Packer
if [ "$ARCH" = "amd64" ] || [ "$ARCH" = "all" ]; then
    echo "Building amd64 image with Packer..."
    cd packer
    packer build -var="version=$VERSION" debian-amd64.pkr.hcl
    cd ..
fi

if [ "$ARCH" = "arm64" ] || [ "$ARCH" = "all" ]; then
    echo "Building arm64 image with Packer..."
    cd packer
    packer build -var="version=$VERSION" debian-arm64.pkr.hcl
    cd ..
fi

echo "=== Build complete ==="
echo "Images available in output/ directory"
