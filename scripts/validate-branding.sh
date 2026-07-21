#!/usr/bin/env bash
# CQOps branding validation — checks that all packaging channels use
# consistent descriptions, icon references, and product naming.
#
# Usage: bash scripts/validate-branding.sh
# Exit 0 = clean, exit 1 = violations found.

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VIOLATIONS=0

red() { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }

check_not_contains() {
	local file="$1" pattern="$2" label="$3"
	if grep -q "$pattern" "$file" 2>/dev/null; then
		red "  FAIL: $label in $file"
		VIOLATIONS=$((VIOLATIONS + 1))
	fi
}

check_contains() {
	local file="$1" pattern="$2" label="$3"
	if ! grep -q "$pattern" "$file" 2>/dev/null; then
		red "  MISSING: $label in $file"
		VIOLATIONS=$((VIOLATIONS + 1))
	fi
}

echo "=== CQOps Branding Validation ==="
echo ""

# 1. No stale "Fast, minimal Go TUI" branding (except changelogs / copilot-instructions).
echo "[1] Stale 'Fast, minimal Go TUI' branding..."
for f in $(grep -rl "Fast, minimal Go TUI" "$ROOT" --include='*.yaml' --include='*.yml' --include='*.json' --include='*.desktop' --include='*.nsi' --include='*.md' --include='*.go' 2>/dev/null); do
	case "$f" in
		*CHANGELOG.md|*copilot-instructions.md|*branding.sh|*validate-branding.sh)
			;; # allowed — historical/changelog or AI instructions
		*)
			check_not_contains "$f" "Fast, minimal Go TUI" "stale 'Fast, minimal Go TUI'"
			;;
	esac
done
echo ""

# 2. No generic terminal icon in .desktop files.
echo "[2] Generic icon fallbacks..."
for f in $(grep -rl "Icon=utilities-terminal\|Icon=terminal\|Icon=application-x-executable" "$ROOT" --include='*.desktop' --include='*.yml' --include='*.yaml' 2>/dev/null); do
	check_not_contains "$f" "Icon=utilities-terminal\|Icon=terminal\|Icon=application-x-executable" "generic icon"
done
echo ""

# 3. Desktop files must use Icon=cqops and Name=CQOps.
echo "[3] Desktop file consistency..."
for f in $(find "$ROOT" -name '*.desktop' 2>/dev/null); do
	check_contains "$f" "Icon=cqops" "Icon=cqops"
	check_contains "$f" "Name=CQOps" "Name=CQOps"
done
# Also check inline .desktop in release.yml.
check_contains "$ROOT/.github/workflows/release.yml" "Icon=cqops" "Icon=cqops in AUR inline .desktop"
check_contains "$ROOT/.github/workflows/release.yml" "Name=CQOps" "Name=CQOps in AUR inline .desktop"
echo ""

# 4. Package descriptions should use canonical short description or similar.
echo "[4] Package description consistency..."
check_contains "$ROOT/.github/workflows/release.yml" "Fast, offline-first amateur radio logger" "canonical description in AUR pkgdesc"
check_contains "$ROOT/nfpm.yaml" "fast, offline-first amateur radio logger" "canonical description in nfpm"
echo ""

# 5. No hardcoded stale version in winres.json.
echo "[5] Windows version metadata..."
check_not_contains "$ROOT/winres/winres.json" '"version": "0\.[0-9]' "hardcoded version in winres.json"
echo ""

# 6. NSIS DisplayName uses canonical format.
echo "[6] NSIS installer metadata..."
check_contains "$ROOT/installer/cqops.nsi" "CQOps" "product name in NSIS"
check_contains "$ROOT/installer/cqops.nsi" "PRODUCT_PUBLISHER" "publisher field in NSIS"
echo ""

# 7. CLI help text.
echo "[7] CLI help text..."
check_contains "$ROOT/internal/cli/root.go" "CQOps" "product name in CLI"
echo ""

# 8. README tagline.
echo "[8] README branding..."
check_contains "$ROOT/README.md" "Less clicking. More radio." "tagline in README"
check_contains "$ROOT/README.md" "fast, offline-first amateur radio logger" "canonical description in README"
echo ""

echo "=== Result ==="
if [ $VIOLATIONS -eq 0 ]; then
	green "All branding checks passed."
	exit 0
else
	red "$VIOLATIONS branding violation(s) found."
	exit 1
fi
