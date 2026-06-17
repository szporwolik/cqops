#!/usr/bin/env bash
set -euo pipefail

VERSION=$(cat VERSION)
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_DIR="build"
mkdir -p "$BUILD_DIR"

LDFLAGS="-s -w -X github.com/szporwolik/cqops/internal/version.Version=${VERSION} -X github.com/szporwolik/cqops/internal/version.BuildDate=${BUILD_DATE}"

targets=(
  "windows amd64 .exe"
  "windows arm64 .exe"
  "linux   amd64"
  "linux   arm64"
  "darwin  amd64"
  "darwin  arm64"
)

for target in "${targets[@]}"; do
  read -r os arch ext <<< "$target"
  name="cqops-${os}-${arch}${ext}"
  echo "Building cqops ${VERSION} for ${os}/${arch}..."
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/$name" ./cmd/cqops
done

echo "Done. Binaries in $BUILD_DIR/"

# Install to GOPATH/bin (if --install flag passed)
if [[ "${1:-}" == "--install" ]]; then
  echo "Installing cqops ${VERSION}..."
  CGO_ENABLED=0 go install -ldflags="$LDFLAGS" ./cmd/cqops
  echo "Installed to $(go env GOPATH)/bin/cqops"
fi
