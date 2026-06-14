#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${HOME}/.local/bin"
DESKTOP_DIR="${HOME}/.local/share/applications"

echo "=== CQOPS Uninstaller (Linux) ==="

rm -f "$INSTALL_DIR/cqops"
echo "  Removed binary"

rm -f "$DESKTOP_DIR/cqops.desktop"
echo "  Removed desktop entry"

sed -i '/cqops/d' "${HOME}/.bashrc" 2>/dev/null || true
echo "  Removed from ~/.bashrc"

echo ""
echo "CQOPS uninstalled."
