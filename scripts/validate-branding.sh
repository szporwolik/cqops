#!/usr/bin/env bash
# CQOps branding validation — must be run from repo root or via absolute path.
# Sources branding.sh for canonical values, then checks all packaging surfaces.
#
# Usage: ./scripts/validate-branding.sh
#        bash scripts/validate-branding.sh
# Exit 0 = clean, exit 1 = violations found.

set -euo pipefail
SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VIOLATIONS=0

# Source canonical branding values.
# shellcheck source=/dev/null
. "$ROOT/scripts/branding.sh"

red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }

fail() {
	red "  FAIL [$1]: $2"
	VIOLATIONS=$((VIOLATIONS + 1))
}

check_contains() {
	local file="$1" pattern="$2" label="$3"
	if ! grep -qF "$pattern" "$file" 2>/dev/null; then
		fail "$label" "expected '$pattern' not found in $file"
	fi
}

echo "=== CQOps Branding Validation ==="
echo "ROOT: $ROOT"
echo ""

# ── 1. Stale branding strings ──────────────────────────────────────────
echo "[1] Stale 'Fast, minimal Go TUI' branding..."
while IFS= read -r f; do
	case "$f" in
		*CHANGELOG.md|*copilot-instructions.md|scripts/*.sh) ;;
		*) fail "stale-string" "$f contains 'Fast, minimal Go TUI'" ;;
	esac
done < <(grep -rlF "Fast, minimal Go TUI" "$ROOT" \
	--include='*.yaml' --include='*.yml' --include='*.json' \
	--include='*.desktop' --include='*.nsi' --include='*.go' 2>/dev/null || true)
echo ""

# ── 2. No generic icon fallbacks ───────────────────────────────────────
echo "[2] Generic icon fallbacks..."
while IFS= read -r f; do
	fail "generic-icon" "$f uses utilities-terminal or generic icon"
done < <(grep -rlF 'Icon=utilities-terminal' "$ROOT" \
	--include='*.desktop' --include='*.yml' --include='*.yaml' 2>/dev/null || true)
echo ""

# ── 3. Desktop files use canonical values ──────────────────────────────
echo "[3] Desktop file consistency..."
check_contains "$ROOT/installer/cqops.desktop" "Icon=$CQOPS_ICON_NAME" "desktop-Icon"
check_contains "$ROOT/installer/cqops.desktop" "Name=$CQOPS_PRODUCT_NAME" "desktop-Name"
check_contains "$ROOT/installer/cqops.desktop" "GenericName=$CQOPS_DESKTOP_GENERIC_NAME" "desktop-GenericName"
check_contains "$ROOT/installer/cqops.desktop" "Comment=$CQOPS_DESKTOP_COMMENT" "desktop-Comment"
check_contains "$ROOT/installer/cqops.desktop" "Terminal=true" "desktop-Terminal"
check_contains "$ROOT/installer/cqops.desktop" "StartupNotify=false" "desktop-StartupNotify"
echo ""

# ── 4. Package descriptions ────────────────────────────────────────────
echo "[4] Package descriptions..."
check_contains "$ROOT/.github/workflows/release.yml" "$CQOPS_AUR_PKGDESC" "AUR-pkgdesc"
check_contains "$ROOT/nfpm.yaml" "fast, offline-first amateur radio logger" "nfpm-description"
echo ""

# ── 5. README ──────────────────────────────────────────────────────────
echo "[5] README branding..."
check_contains "$ROOT/README.md" "$CQOPS_TAGLINE" "README-tagline"
echo ""

# ── 6. CLI ─────────────────────────────────────────────────────────────
echo "[6] CLI help text..."
check_contains "$ROOT/internal/cli/root.go" "$CQOPS_PRODUCT_NAME" "CLI-product-name"
echo ""

# ── 7. NSIS ────────────────────────────────────────────────────────────
echo "[7] NSIS installer..."
check_contains "$ROOT/installer/cqops.nsi" "$CQOPS_WIN_INSTALLED_NAME" "NSIS-product-name"
echo ""

# ── 8. Product name casing ─────────────────────────────────────────────
echo "[8] No incorrect product-name casing..."
while IFS= read -r f; do
	case "$f" in
		*CHANGELOG.md|scripts/*.sh|*README.md) ;;
		*go.sum|*go.mod|*.syso|*.ico|*.png|*.jpg|*.svg) ;;
		*)
			fail "bad-casing" "$f contains 'CQOPS' or 'CqOps' — use 'CQOps' or 'cqops'"
			;;
	esac
done < <(grep -rlE '\bCQOPS\b|\bCqOps\b' "$ROOT" \
	--include='*.go' --include='*.yaml' --include='*.yml' --include='*.json' \
	--include='*.desktop' --include='*.nsi' --include='*.md' 2>/dev/null || true)
echo ""

echo "=== Result ==="
if [ "$VIOLATIONS" -eq 0 ]; then
	green "All branding checks passed ($VIOLATIONS violations)."
	exit 0
else
	red "$VIOLATIONS branding violation(s) found."
	exit 1
fi
