#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION=$(cat "$SCRIPT_DIR/../VERSION")
BUILD_DIR="$SCRIPT_DIR/../build"
ASSETS_DIR="$SCRIPT_DIR/../assets"
INSTALL_DIR="${HOME}/.local/bin"
DESKTOP_DIR="${HOME}/.local/share/applications"
ICON_BASE="${HOME}/.local/share/icons/hicolor"
PIXMAPS_DIR="${HOME}/.local/share/pixmaps"
ICON_SRC="$ASSETS_DIR/cqops-icon.svg"

echo "=== CQOps v${VERSION} Installer (Linux) ==="

mkdir -p "$INSTALL_DIR" "$DESKTOP_DIR" "$PIXMAPS_DIR"

# Generate icons in multiple sizes.
# Prefer rsvg-convert (fast, correct), then magick (IM7), then convert (IM6).
SIZES="16 24 32 48 64 128 256"
if command -v rsvg-convert &>/dev/null; then
    for sz in $SIZES; do
        ICON_DIR="${ICON_BASE}/${sz}x${sz}/apps"
        mkdir -p "$ICON_DIR"
        rsvg-convert -w $sz -h $sz "$ICON_SRC" -o "$ICON_DIR/cqops.png"
    done
    echo "  Icons  : ${ICON_BASE}/{16..256}x{16..256}/apps/cqops.png (rsvg)"
elif command -v magick &>/dev/null; then
    for sz in $SIZES; do
        ICON_DIR="${ICON_BASE}/${sz}x${sz}/apps"
        mkdir -p "$ICON_DIR"
        magick "$ICON_SRC" -resize ${sz}x${sz} "$ICON_DIR/cqops.png"
    done
    echo "  Icons  : ${ICON_BASE}/{16..256}x{16..256}/apps/cqops.png (imagemagick)"
elif command -v convert &>/dev/null; then
    for sz in $SIZES; do
        ICON_DIR="${ICON_BASE}/${sz}x${sz}/apps"
        mkdir -p "$ICON_DIR"
        convert -background none "$ICON_SRC" -resize ${sz}x${sz} "$ICON_DIR/cqops.png"
    done
    echo "  Icons  : ${ICON_BASE}/{16..256}x{16..256}/apps/cqops.png (imagemagick6)"
else
    echo "  Icons  : no SVG converter found — installing 48x48 fallback only"
    # Bare-minimum fallback: copy the pre-built 48x48 PNG into hicolor.
    if [[ -f "$ASSETS_DIR/cqops.png" ]]; then
        mkdir -p "${ICON_BASE}/48x48/apps"
        cp "$ASSETS_DIR/cqops.png" "${ICON_BASE}/48x48/apps/cqops.png"
    fi
fi

# Copy pre-built 48x48 PNG and XPM to pixmaps (ships with the repo).
if [[ -f "$ASSETS_DIR/cqops.png" ]]; then
    cp "$ASSETS_DIR/cqops.png" "$PIXMAPS_DIR/cqops.png"
fi
if [[ -f "$ASSETS_DIR/cqops.xpm" ]]; then
    cp "$ASSETS_DIR/cqops.xpm" "$PIXMAPS_DIR/cqops.xpm"
    echo "  XPM    : $PIXMAPS_DIR/cqops.xpm (IceWM)"
else
    # Fallback: generate XPM via ImageMagick if available.
    if command -v magick &>/dev/null; then
        magick "${ICON_BASE}/48x48/apps/cqops.png" "$PIXMAPS_DIR/cqops.xpm" 2>/dev/null || true
        echo "  XPM    : $PIXMAPS_DIR/cqops.xpm (IceWM, generated)"
    elif command -v convert &>/dev/null; then
        convert "${ICON_BASE}/48x48/apps/cqops.png" "$PIXMAPS_DIR/cqops.xpm" 2>/dev/null || true
        echo "  XPM    : $PIXMAPS_DIR/cqops.xpm (IceWM, generated)"
    fi
fi

# Rebuild icon cache
if command -v gtk-update-icon-cache &>/dev/null; then
    gtk-update-icon-cache -f -t "$ICON_BASE" 2>/dev/null || true
    echo "  Cache  : hicolor icon cache updated"
fi

# Pick the right binary for this architecture.
BIN="$BUILD_DIR/cqops"
if [[ ! -f "$BIN" ]]; then
	ARCH=$(uname -m)
	case "$ARCH" in
		aarch64|arm64)  BIN="$BUILD_DIR/cqops-linux-arm64" ;;
		x86_64|amd64)   BIN="$BUILD_DIR/cqops-linux-amd64" ;;
		*)              BIN="$BUILD_DIR/cqops-linux-amd64" ;; # best guess
	esac
fi
if [[ ! -f "$BIN" ]]; then
    echo "Building cqops..."
    "$SCRIPT_DIR/build.sh"
    ARCH=$(uname -m)
    case "$ARCH" in
        aarch64|arm64)  BIN="$BUILD_DIR/cqops-linux-arm64" ;;
        *)              BIN="$BUILD_DIR/cqops-linux-amd64" ;;
    esac
fi
# Remove old binary first (ok on Linux even if running — the inode stays alive
# for the running process, but the directory entry is freed for the new copy).
rm -f "$INSTALL_DIR/cqops"
cp "$BIN" "$INSTALL_DIR/cqops"
chmod +x "$INSTALL_DIR/cqops"
echo "  Binary : $INSTALL_DIR/cqops"

cat > "$DESKTOP_DIR/cqops.desktop" << EOF
[Desktop Entry]
Name=CQOps
Comment=Amateur Radio Logging
Exec=$INSTALL_DIR/cqops
Icon=cqops
Terminal=true
Type=Application
Categories=Network;HamRadio;
StartupWMClass=cqops
EOF
echo "  Menu   : Applications → Ham Radio → CQOps"

# Refresh desktop menu database
if command -v update-desktop-database &>/dev/null; then
    update-desktop-database "$DESKTOP_DIR" 2>/dev/null || true
    echo "  Menu DB: refreshed"
fi

# Add to PATH in shell profiles (bash, zsh, posix).
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    for rc in "${HOME}/.profile" "${HOME}/.bashrc" "${HOME}/.zshrc"; do
        if [[ -f "$rc" ]] && grep -q "$INSTALL_DIR" "$rc" 2>/dev/null; then
            continue  # already present in this rc file
        fi
        echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$rc"
    done
    # If none of the rc files existed, at least write .profile.
    if [[ ! -f "${HOME}/.profile" ]] && [[ ! -f "${HOME}/.bashrc" ]] && [[ ! -f "${HOME}/.zshrc" ]]; then
        echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "${HOME}/.profile"
    fi
    echo "  PATH   : added to shell profiles (new shells only)"
fi

echo ""
echo "CQOps v${VERSION} installed. Run 'cqops' or use the app menu."
echo "Uninstall: ./scripts/uninstall.sh"
