# CQOps Branding — Canonical Source of Truth
#
# This file is the single authoritative reference for all CQOps branding,
# package metadata, desktop entries, installer metadata, and public-facing
# product descriptions. All packaging channels and release workflows must
# derive their descriptions from this file.
#
# Format: KEY=value (shell-compatible). Lines starting with # are comments.

# ── Product identity ───────────────────────────────────────────────────────
PRODUCT_NAME="CQOps"
PRODUCT_EXEC="cqops"
PRODUCT_TAGLINE="Less clicking. More radio."
PRODUCT_ICON="cqops"
PRODUCT_PUBLISHER="Szymon Porwolik"
PRODUCT_URL="https://github.com/szporwolik/cqops"
PRODUCT_LICENSE="Apache-2.0"
PRODUCT_CATEGORY="HamRadio"

# ── Short description (package managers, desktop Comment, tooltips) ─────────
SHORT_DESC="Fast, offline-first amateur radio logging for the terminal"

# ── Medium description (constrained fields, release notes) ─────────────────
MEDIUM_DESC="A fast, offline-first amateur radio logger for the terminal, built for portable operations, SOTA/POTA, contests, and club stations. Supports WSJT-X, Wavelog, GPS, APRS, multiple operators and logbooks, and a built-in browser dashboard."

# ── Full description (package long descriptions, README, website) ──────────
FULL_DESC="CQOps is a fast, offline-first amateur radio logger for the terminal, built for portable and field operations, SOTA/POTA activations, contests, and club stations with rotating operators. Log contacts manually or automatically through WSJT-X, switch operators and logbooks instantly, synchronize with Wavelog, and use GPS-aware station positioning with APRS support and a built-in browser dashboard. CQOps runs on Raspberry Pi, older laptops, low-power computers, local terminals, and remote systems over SSH."

# ── Desktop entry ─────────────────────────────────────────────────────────
DESKTOP_GENERIC_NAME="Amateur Radio Logger"
DESKTOP_CATEGORIES="HamRadio;Utility;Network;"
DESKTOP_KEYWORDS="ham radio;amateur radio;logger;logging;SOTA;POTA;contest;APRS;WSJT-X;Wavelog;"

# ── Windows metadata ──────────────────────────────────────────────────────
WIN_FILE_DESC="CQOps — Amateur Radio Logger"
WIN_DISPLAY_NAME="CQOps — Amateur Radio Logger"
