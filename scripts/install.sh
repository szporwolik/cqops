#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION=$(cat "$SCRIPT_DIR/../VERSION")
BUILD_DIR="$SCRIPT_DIR/../build"
INSTALL_DIR="${HOME}/.local/bin"
DESKTOP_DIR="${HOME}/.local/share/applications"
ICON_DIR="${HOME}/.local/share/icons/hicolor/256x256/apps"

echo "=== CQOPS v${VERSION} Installer (Linux) ==="

mkdir -p "$INSTALL_DIR" "$DESKTOP_DIR" "$ICON_DIR"

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
Terminal=true
Type=Application
Categories=HamRadio;Utility;
EOF
echo "  Menu   : Applications › Ham Radio › CQOPS"

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "${HOME}/.bashrc"
    echo "  PATH   : added to ~/.bashrc (new shells only)"
fi

echo ""
echo "CQOPS v${VERSION} installed. Run 'cqops' or use the app menu."
echo "Uninstall: ./scripts/uninstall.sh"
