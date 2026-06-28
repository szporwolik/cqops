#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${HOME}/.local/bin"
DESKTOP_DIR="${HOME}/.local/share/applications"
ICON_BASE="${HOME}/.local/share/icons/hicolor"
PIXMAPS_DIR="${HOME}/.local/share/pixmaps"

echo "=== CQOps Uninstaller (Linux) ==="

rm -f "$INSTALL_DIR/cqops"
echo "  Removed binary"

rm -f "$DESKTOP_DIR/cqops.desktop"
echo "  Removed desktop entry"

# Remove all icon sizes
for sz in 16 24 32 48 64 128 256; do
    rm -f "${ICON_BASE}/${sz}x${sz}/apps/cqops.png"
done
rm -f "$PIXMAPS_DIR/cqops.png" "$PIXMAPS_DIR/cqops.xpm"
echo "  Removed icons"

# Rebuild icon cache
if command -v gtk-update-icon-cache &>/dev/null; then
    gtk-update-icon-cache -f -t "$ICON_BASE" 2>/dev/null || true
fi

# Refresh desktop menu database
if command -v update-desktop-database &>/dev/null; then
    update-desktop-database "$DESKTOP_DIR" 2>/dev/null || true
fi

# Remove from shell profiles
for rc in "${HOME}/.profile" "${HOME}/.bashrc" "${HOME}/.zshrc"; do
    sed -i '\|export PATH="'"$INSTALL_DIR"':$PATH"|d' "$rc" 2>/dev/null || true
done
echo "  Removed from shell profiles"

echo ""
echo "CQOps uninstalled."
