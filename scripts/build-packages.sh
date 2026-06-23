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
ARCHS=("amd64" "arm64" "armv7")

for arch in "${ARCHS[@]}"; do
    echo ""
    # Map Go arch to build params and Debian arch name.
    case "$arch" in
        armv7) goos=linux; goarch=arm; goarm=7; deb_arch="armhf" ;;
        *)     goos=linux; goarch="$arch"; goarm=""; deb_arch="$arch" ;;
    esac
    echo "[BUILD] linux/$deb_arch"
    if [ -n "$goarm" ]; then
      GOOS=$goos GOARCH=$goarch GOARM=$goarm go build -ldflags "$LDFLAGS" -o "build/cqops-linux-$deb_arch" ./cmd/cqops/
    else
      GOOS=$goos GOARCH=$goarch go build -ldflags "$LDFLAGS" -o "build/cqops-linux-$deb_arch" ./cmd/cqops/
    fi
    echo "  Binary : build/cqops-linux-$deb_arch ($(du -h build/cqops-linux-$deb_arch | cut -f1))"
done

# ---------------------------------------------------------------------------
# 3. Package with nfpm
# ---------------------------------------------------------------------------
PACKAGERS=("deb")

for pkg in "${PACKAGERS[@]}"; do
    case "$pkg" in
        deb) ext="deb" ;;
    esac

    for arch in "${ARCHS[@]}"; do
        # Map to Debian arch name.
        case "$arch" in
            armv7) deb_arch="armhf" ;;
            *)     deb_arch="$arch" ;;
        esac

        echo ""
        echo "[PACKAGE] deb / $deb_arch"

        NFFILE="dist/nfpm-deb-${deb_arch}.yaml"
        sed -e "s/__VERSION__/${VERSION}/g" \
            -e "s/__ARCH__/${deb_arch}/g" \
            nfpm.yaml > "$NFFILE"

        nfpm package \
            -f "$NFFILE" \
            -p deb \
            -t "$DIST_DIR/" \
            --packager deb \
            --target "${DIST_DIR}/cqops_${VERSION}_${deb_arch}.deb"

        target="${DIST_DIR}/cqops_${VERSION}_${deb_arch}.deb"
        if [ -f "$target" ]; then
            echo "  Package : $target ($(du -h "$target" | cut -f1))"
        else
            echo "  ERROR: $target not created"
            exit 1
        fi
    done
done
rm -f "$DIST_DIR"/nfpm-*.yaml

echo ""
echo "=== Done ==="
ls -lh "$DIST_DIR"/*.deb 2>/dev/null || true
