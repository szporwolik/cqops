#!/usr/bin/env bash
set -euo pipefail

VERSION=$(cat VERSION)
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_DIR="build"
mkdir -p "$BUILD_DIR"

# Patch winres.json with current version before Windows builds
VER4="${VERSION}.0"
if command -v jq &>/dev/null; then
  jq --arg v4 "$VER4" --arg v3 "$VERSION" \
    '.RT_MANIFEST."#1"."0409".identity.version = $v4 |
     .RT_VERSION."#1"."0000".fixed.file_version = $v4 |
     .RT_VERSION."#1"."0000".fixed.product_version = $v4 |
     .RT_VERSION."#1"."0000".info."0409".FileVersion = $v3 |
     .RT_VERSION."#1"."0000".info."0409".ProductVersion = $v3' \
    winres/winres.json > winres/winres.json.tmp && mv winres/winres.json.tmp winres/winres.json
fi
# Regenerate .syso if go-winres is available
if command -v go-winres &>/dev/null; then
  (cd winres && go-winres make)
fi

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
