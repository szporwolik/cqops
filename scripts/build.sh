#!/usr/bin/env bash
set -euo pipefail

VERSION=$(cat VERSION)
BUILD_DIR="build"
mkdir -p "$BUILD_DIR"

LDFLAGS="-s -w -X github.com/szporwolik/cqops/internal/version.Version=${VERSION}"

echo "Building cqops ${VERSION} for linux/amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/cqops-linux-amd64" ./cmd/cqops

echo "Building cqops ${VERSION} for linux/arm64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/cqops-linux-arm64" ./cmd/cqops

echo "Building cqops ${VERSION} for current platform..."
CGO_ENABLED=0 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/cqops" ./cmd/cqops

echo "Done. Binaries in $BUILD_DIR/"
