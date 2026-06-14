#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION=$(cat "$SCRIPT_DIR/../VERSION")
BUILD_DIR="$SCRIPT_DIR/../build"
INSTALL_DIR="${HOME}/.local/bin"
DESKTOP_DIR="${HOME}/.local/share/applications"
ICON_DIR="${HOME}/.local/share/icons/hicolor/256x256/apps"
ICON_SRC="$SCRIPT_DIR/../assets/cqops-icon.svg"

echo "=== CQOPS v${VERSION} Installer (Linux) ==="

mkdir -p "$INSTALL_DIR" "$DESKTOP_DIR" "$ICON_DIR"

# Generate icon
if command -v rsvg-convert &>/dev/null; then
    rsvg-convert -w 256 -h 256 "$ICON_SRC" -o "$ICON_DIR/cqops.png"
    echo "  Icon   : $ICON_DIR/cqops.png"
elif command -v magick &>/dev/null; then
    magick "$ICON_SRC" -resize 256x256 "$ICON_DIR/cqops.png"
    echo "  Icon   : $ICON_DIR/cqops.png"
else
    echo "  Icon   : skipping (rsvg-convert or imagemagick not found)"
fi

BIN="$BUILD_DIR/cqops-linux-amd64"
if [[ ! -f "$BIN" ]]; then
    echo "Building cqops..."
    "$SCRIPT_DIR/build.sh"
fi
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

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "${HOME}/.bashrc"
    echo "  PATH   : added to ~/.bashrc (new shells only)"
fi

echo ""
echo "CQOPS v${VERSION} installed. Run 'cqops' or use the app menu."
echo "Uninstall: ./scripts/uninstall.sh"
