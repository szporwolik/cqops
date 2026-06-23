#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

VERSION=$(cat VERSION)
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
export CGO_ENABLED=0

DIST_DIR="dist"
mkdir -p "$DIST_DIR"

LDFLAGS="-s -w -X github.com/szporwolik/cqops/internal/version.Version=$VERSION -X github.com/szporwolik/cqops/internal/version.BuildDate=$BUILD_DATE"

echo "=== CQOps v$VERSION — Linux Package Build ==="

# ---------------------------------------------------------------------------
# 1. Check prerequisites
# ---------------------------------------------------------------------------
if ! command -v nfpm &>/dev/null; then
    echo "ERROR: nfpm not found. Install it: go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest"
    exit 1
fi
echo "[✓] nfpm found: $(which nfpm)"

# ---------------------------------------------------------------------------
# 2. Build binaries for each Linux arch
# ---------------------------------------------------------------------------
ARCHS=("amd64" "arm64")

for arch in "${ARCHS[@]}"; do
    echo ""
    echo "[BUILD] linux/$arch"
    GOOS=linux GOARCH=$arch go build -ldflags "$LDFLAGS" -o "build/cqops-linux-$arch" ./cmd/cqops/
    echo "  Binary : build/cqops-linux-$arch ($(du -h build/cqops-linux-$arch | cut -f1))"
done

# ---------------------------------------------------------------------------
# 3. Package with nfpm
# ---------------------------------------------------------------------------
PACKAGERS=("deb" "rpm" "archlinux")

for pkg in "${PACKAGERS[@]}"; do
    # Map nfpm packager name to file extension.
    case "$pkg" in
        deb)       ext="deb" ;;
        rpm)       ext="rpm" ;;
        archlinux) ext="pkg.tar.zst" ;;
    esac

    for arch in "${ARCHS[@]}"; do
        echo ""
        echo "[PACKAGE] $pkg / $arch"

        # Generate nfpm config with substituted arch and version.
        NFFILE="dist/nfpm-${pkg}-${arch}.yaml"
        sed -e "s/__VERSION__/${VERSION}/g" \
            -e "s/__ARCH__/${arch}/g" \
            nfpm.yaml > "$NFFILE"

        nfpm package \
            -f "$NFFILE" \
            -p "$pkg" \
            -t "$DIST_DIR/" \
            -r "$arch" \
            --packager "${pkg}"

        target="${DIST_DIR}/cqops_${VERSION}_linux_${arch}.${ext}"
        if [ -f "$target" ]; then
            echo "  Package : $target ($(du -h "$target" | cut -f1))"
        else
            echo "  ERROR: $target not created"
            exit 1
        fi
    done
done

echo ""
echo "=== Done ==="
ls -lh "$DIST_DIR"/*.deb "$DIST_DIR"/*.rpm "$DIST_DIR"/*.pkg.tar.zst 2>/dev/null || true
