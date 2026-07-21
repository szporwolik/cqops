# CQOps Branding — Canonical Source of Truth
#
# Sourced by validate-branding.sh and optionally by CI release steps.
# All packaging channels must use values derived from this file.
#
# shellcheck shell=sh
# shellcheck disable=SC2034  # variables are consumed by sourcing scripts

# ── Product identity ───────────────────────────────────────────────────────
CQOPS_PRODUCT_NAME="CQOps"
CQOPS_PRODUCT_EXEC="cqops"
CQOPS_TAGLINE="Less clicking. More radio."
CQOPS_ICON_NAME="cqops"
CQOPS_PUBLISHER="Szymon Porwolik"
CQOPS_URL="https://github.com/szporwolik/cqops"
CQOPS_LICENSE="Apache-2.0"

# ── Short description (package managers, desktop Comment, tooltips) ─────────
CQOPS_SHORT_DESC="Fast, offline-first amateur radio logging for the terminal"

# ── Medium description (constrained fields, release notes) ─────────────────
CQOPS_MEDIUM_DESC="A fast, offline-first amateur radio logger for the terminal, built for portable operations, SOTA/POTA, contests, and club stations. Supports WSJT-X, Wavelog, GPS, APRS, multiple operators and logbooks, and a built-in browser dashboard."

# ── Full description (package long descriptions, README, website) ──────────
CQOPS_FULL_DESC="CQOps is a fast, offline-first amateur radio logger for the terminal, built for portable and field operations, SOTA/POTA activations, contests, and club stations with rotating operators. Log contacts manually or automatically through WSJT-X, switch operators and logbooks instantly, synchronize with Wavelog, and use GPS-aware station positioning with APRS support and a built-in browser dashboard. CQOps runs on Raspberry Pi, older laptops, low-power computers, local terminals, and remote systems over SSH."

# ── Desktop entry ─────────────────────────────────────────────────────────
CQOPS_DESKTOP_COMMENT="Fast, offline-first amateur radio logging for the terminal"
CQOPS_DESKTOP_GENERIC_NAME="Amateur Radio Logger"
CQOPS_DESKTOP_CATEGORIES="HamRadio;Utility;Network;"
CQOPS_DESKTOP_KEYWORDS="ham radio;amateur radio;logger;logging;SOTA;POTA;contest;APRS;WSJT-X;Wavelog;"

# ── Windows metadata ──────────────────────────────────────────────────────
CQOPS_WIN_FILE_DESC="CQOps — Amateur Radio Logger"
CQOPS_WIN_INSTALLED_NAME="CQOps"

# ── AUR metadata ──────────────────────────────────────────────────────────
CQOPS_AUR_PKGDESC="Fast, offline-first amateur radio logger for the terminal"
