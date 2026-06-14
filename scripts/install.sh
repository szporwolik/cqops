#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION=$(cat "$SCRIPT_DIR/../VERSION")
BUILD_DIR="$SCRIPT_DIR/../build"
INSTALL_DIR="${HOME}/.local/bin"
DESKTOP_DIR="${HOME}/.local/share/applications"
ICON_BASE="${HOME}/.local/share/icons/hicolor"
PIXMAPS_DIR="${HOME}/.local/share/pixmaps"
ICON_SRC="$SCRIPT_DIR/../assets/cqops-icon.svg"

echo "=== CQOPS v${VERSION} Installer (Linux) ==="

mkdir -p "$INSTALL_DIR" "$DESKTOP_DIR" "$PIXMAPS_DIR"

# Generate icons in multiple sizes (IceWM menus use small icons)
SIZES="16 24 32 48 64 128 256"
if command -v rsvg-convert &>/dev/null; then
    for sz in $SIZES; do
        ICON_DIR="${ICON_BASE}/${sz}x${sz}/apps"
        mkdir -p "$ICON_DIR"
        rsvg-convert -w $sz -h $sz "$ICON_SRC" -o "$ICON_DIR/cqops.png"
    done
    echo "  Icons  : ${ICON_BASE}/{16..256}x{16..256}/apps/cqops.png"
elif command -v magick &>/dev/null; then
    for sz in $SIZES; do
        ICON_DIR="${ICON_BASE}/${sz}x${sz}/apps"
        mkdir -p "$ICON_DIR"
        magick "$ICON_SRC" -resize ${sz}x${sz} "$ICON_DIR/cqops.png"
    done
    echo "  Icons  : ${ICON_BASE}/{16..256}x{16..256}/apps/cqops.png"
else
    echo "  Icons  : skipping (rsvg-convert or imagemagick not found)"
fi

# Also place a copy in pixmaps (IceWM fallback path)
cp "${ICON_BASE}/48x48/apps/cqops.png" "$PIXMAPS_DIR/cqops.png" 2>/dev/null || true

# Generate XPM icon for IceWM (which traditionally prefers XPM)
if command -v magick &>/dev/null; then
    magick "${ICON_BASE}/48x48/apps/cqops.png" "$PIXMAPS_DIR/cqops.xpm" 2>/dev/null || true
    echo "  XPM    : $PIXMAPS_DIR/cqops.xpm (IceWM)"
elif command -v convert &>/dev/null; then
    convert "${ICON_BASE}/48x48/apps/cqops.png" "$PIXMAPS_DIR/cqops.xpm" 2>/dev/null || true
    echo "  XPM    : $PIXMAPS_DIR/cqops.xpm (IceWM)"
fi

# Rebuild icon cache
if command -v gtk-update-icon-cache &>/dev/null; then
    gtk-update-icon-cache -f -t "$ICON_BASE" 2>/dev/null || true
    echo "  Cache  : hicolor icon cache updated"
fi

BIN="$BUILD_DIR/cqops"
if [[ ! -f "$BIN" ]]; then
	# Fall back to platform-specific binary from build-all
	BIN="$BUILD_DIR/cqops-linux-amd64"
fi
if [[ ! -f "$BIN" ]]; then
    echo "Building cqops..."
    "$SCRIPT_DIR/build.sh"
    BIN="$BUILD_DIR/cqops-linux-amd64"
fi
# Remove old binary first (ok on Linux even if running — the inode stays alive
# for the running process, but the directory entry is freed for the new copy).
rm -f "$INSTALL_DIR/cqops"
cp "$BIN" "$INSTALL_DIR/cqops"
chmod +x "$INSTALL_DIR/cqops"
echo "  Binary : $INSTALL_DIR/cqops"

cat > "$DESKTOP_DIR/cqops.desktop" << EOF
[Desktop Entry]
Name=CQOPS
Comment=Amateur Radio Logging
Exec=$INSTALL_DIR/cqops
Icon=cqops
Terminal=true
Type=Application
Categories=HamRadio;Utility;
EOF
echo "  Menu   : Applications � Ham Radio � CQOPS"

# Refresh desktop menu database
if command -v update-desktop-database &>/dev/null; then
    update-desktop-database "$DESKTOP_DIR" 2>/dev/null || true
    echo "  Menu DB: refreshed"
fi

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "${HOME}/.bashrc"
    echo "  PATH   : added to ~/.bashrc (new shells only)"
fi

echo ""
echo "CQOPS v${VERSION} installed. Run 'cqops' or use the app menu."
echo "Uninstall: ./scripts/uninstall.sh"
