# Changelog

## v0.9.4 — 2026-07-21

> **DXC path line, branding consistency, and dashboard polish.** Smart spot filtering, DXCC badge fix, hamlib mode recognition, band plan tune fix, and unified product identity across all packaging channels.

### Cross-Platform Branding Consistency
- **Unified product identity**: all package descriptions, desktop entries, AUR metadata, Windows installer, executable resources, README, CLI help, dashboard headers, documentation, and CI tooling now share a single canonical branding source (`scripts/branding.sh`) with automated validation (`scripts/validate-branding.sh`).
- **Tagline**: *"Less clicking. More radio."* now appears in README, dashboard, and all 10 manual translations.
- **Descriptions**: replaced implementation-centric *"Fast, minimal Go TUI ham radio logger"* with the canonical *"Fast, offline-first amateur radio logger for the terminal"* across all channels.
- **Icon fix**: AUR `.desktop` entry now uses `Icon=cqops` instead of the generic `Icon=utilities-terminal`. Desktop file simplified to `Categories=Network;HamRadio;`.
- **Windows metadata**: `FileDescription`, `CompanyName`, `LegalCopyright`, and `ProductVersion` corrected in both `winres.json` and compiled `.syso` resources. `.syso` determinism verified in CI.
- **AUR improvements**: PKGBUILD and `.SRCINFO` share a single `pkgdesc` variable to prevent drift. Canonical `.desktop` file sourced from the tagged release instead of a duplicated inline heredoc. Real SHA-256 checksum for the desktop file.
- **CI validation**: new `branding-check.yml` workflow (6 jobs) — ShellCheck + branding validation, DEB/RPM build + inspection, Arch `makepkg --printsrcinfo`, `CGO_ENABLED=1 go test -race`, NSIS compilation + metadata assertions, `.syso` determinism.
- **9-language docs**: dashboard Header 2 fallback updated to the tagline in all manual translations. Non-English introductory paragraphs preserved (awaiting native-speaker translation pass).

### DXC Path Line — Smart Filtering
- **Same-continent filter**: spots above the QSO form now default to showing only spots heard from the same continent as the station (`SpotCont` matches configured `Continent`). Cascading fallback if no matches.
- **Same-mode filter**: spots are filtered by mode category — `CW`, `DIGI` (FT8/FT4/RTTY/PKT), or `PHONE` (SSB/AM/FM). Switches automatically when you change mode on the radio.
- **15-minute time window**: spots older than 15 minutes are excluded by default. Falls back to time-only if continent+mode filters produce no results.
- **Cache invalidation**: added generation counter (`rawGen`) so the line redraws immediately when new spots arrive, even when the 600-spot buffer is full.
- **Ctrl+P reference auto-fill**: cycling through on-frequency spots now parses the spot comment for SOTA/POTA/WWFF/IOTA references and auto-fills the QSO form.
- **DB index**: new composite index `idx_dxc_spots_band_time` on `dxc_spots(band, received_at)` for efficient band+time queries.

### Dashboard — New DXCC Badge Fix
- **Case-insensitive DXCC matching**: `countryWorkedBefore` now uses the same multi-strategy matching as the F2 partner page — DXCC entity number, case-insensitive country name, and prefix LIKE matching. Fixes "New DXCC" badge incorrectly showing when the DXCC was already worked but stored under a different country name casing (e.g. `SAUDI ARABIA` from Wavelog vs `Saudi Arabia` from QRZ).

### Mode & Rig Compatibility
- **Hamlib mode recognition**: `spotModeCategory` now recognizes `PKTUSB`, `PKTLSB`, `CWR`, `RTTYR`, `FMN`, `WFM`, `SSB`, and `PKT` variants — fixes mode filtering being silently disabled when the rig reports hamlib-specific mode names.
- **Rig mode normalization**: `rigModeMap` now maps `PKTUSB`/`PKTLSB`/`PKTFM` → `PKT` and `FMN` → `FM`.

### Fixes
- **Band plan SAT false positive**: tuning a broadcast station (e.g. "Radio Antena Satelor") no longer incorrectly selects FM mode. Satellite detection now uses word-boundary matching.
- **Default continent**: logbooks without an explicit `continent` field now default to `EU`, so the DXC path line continent filter works on upgraded configs.
- **Test suite**: two stale tests fixed. All 32 test packages pass.
- **Code cleanup**: removed dead `gpsTickMsg` type, converted callbook test handler to tagged switch, marked unused parameter.

### Under the Hood
- **~25 commits**, **~27 files changed**. New DB index added (idempotent `CREATE INDEX IF NOT EXISTS`). New CI workflow for cross-platform branding/packaging validation. No config migration needed from v0.9.2/v0.9.3.

## v0.9.3 — 2026-07-21

> **Polish release.** First-run wizard improvements, secret masking, navigation fixes, and AUR packaging complete.

### First-Run Wizard
- **Enter = save & next**: Enter now works as Ctrl+S in all wizard steps (Station, Rig, General, Summary). Wizard header shows `[Enter — save & next]` hint.
- **Operator field removed**: the wizard no longer shows the operator selector — operators don't exist yet during first-run setup.
- **Navigation fixes**: Tab/Shift+Tab now correctly wraps between fields. Fixed rig name field not receiving focus (missing `case rigFieldName` in `focusField()`). Fixed off-screen navigation when Wavelog or WSJT-X is disabled.

### Security — Secret Masking
- **Wavelog API key**: now masked as `***` in the station config form. Shows plaintext only when the field is focused for editing. Same for APRS passcode.
- **Masking logic**: secrets are masked on blur, revealed on focus, and re-masked before save. The underlying `secrets.enc` encryption was already in place — this change only affects UI display.

### UI Polish
- **Rotator hint**: the "Experimental feature — use with caution" text is now hidden on terminals narrower than 85 columns — no more wrapping.
- **AUR PKGBUILD**: now includes SVG app icon and `.desktop` entry so CQOps appears in the application menu with the proper icon on Arch-based systems.

### Under the Hood
- **~10 commits**, **5 files changed**. No config or database migration needed from v0.9.2.

## v0.9.2 — 2026-07-18

> **Maintenance release.** New packaging target and distribution polish — no application code changes from v0.9.1.

### Arch Linux / AUR
- **`cqops-bin` published on AUR**: Arch, Manjaro, CachyOS, and other Arch-based distros can now install via `yay -S cqops-bin`. PKGBUILD downloads the pre-built binary from GitHub Releases — no compilation needed.
- **Automated updates**: the release workflow now pushes PKGBUILD and `.SRCINFO` updates to AUR on every release, keeping `pkgver` and `sha256sums` in sync automatically.
- **AUR badge**: version shield added to README header alongside Cloudsmith.

### Distribution
- **README**: added installation section for Arch Linux, Manjaro, and CachyOS with AUR instructions.
- **Release workflow**: AUR publish job fixed — switched from `.deb` extraction to `.tar.gz`, added `.SRCINFO` generation, and configured `git` identity for automated commits.

### Under the Hood
- **~5 commits**, **2 files changed**. No config or database migration needed from v0.9.1. Same binary — only packaging and docs updated.

## v0.9.1 — 2026-07-16

> **Performance release.** CPU usage on low-end hardware (Raspberry Pi, old laptops, portable field setups) dropped from ~107% to ~13% — a 92% reduction.

### Performance — Potato PC Optimizations
- **FPS cap**: renderer limited to 20 FPS via `tea.WithFPS(20)`. CQOps is tick-driven at 1 Hz — higher frame rates provide no visual benefit but waste CPU on low-end hardware.
- **GPS polling merged**: GPS position now polls every 60 seconds inside the main tick instead of running a separate 1-second ticker. Eliminates one full Update→View cycle per second.
- **Dashboard stats event-driven**: today QSOs, aggregate stats, and recent QSOs are now recomputed only when a QSO is saved, logbook changes, or data is imported — not on a 5-second timer. Avoids 4 expensive COUNT/DISTINCT aggregation queries every 5 seconds.
- **WorkedSummary cache**: `GetWorkedSummary()` (16 SQL queries) was running inside `View()` on every frame. Now cached with a `call|grid|DXCC|country` key and only recomputed when inputs change. `viewPartner` CPU dropped from 16.68% to 2.61%.
- **stripANSI rewrite**: regex-based ANSI escape stripping replaced with a byte scanner including a fast-path for strings without escape sequences. Removes the `regexp` import from the dashboard push path. ~5× faster on low-end CPUs.
- **countryWorkedBefore cache**: per-country `COUNT(*)` query now cached per `(country, baseCall)` pair. Cleared on QSO save.
- **View() JoinVertical → plain newline**: the main view compositing (status + tabs + body + help) now uses `strings.Join` instead of `lipgloss.JoinVertical`. Avoids the Lip Gloss line-measurement pass on every frame.
- **Profiling**: new `--pprof` flag starts a `net/http/pprof` server on `127.0.0.1:6060` for CPU, heap, goroutine, mutex, and block profiling. Zero overhead when not used.

### New Callbook Provider — QRZ.RU
- **QRZ.RU**: free global callbook with name, QTH, grid, DXCC, and photo. Russian-hosted; good coverage for Eastern European and Asian callsigns. Config, secrets, menu integration, and dashboard support included.
- **HamQTH default images**: placeholder images from `hamqth.com/images/default/` (e.g. `paddle_and_notebook.jpg`) are now treated as "no image" so lower-priority callbooks (QRZ, Callook) can supply real photos. Genuine HamQTH operator photos pass through unchanged.

### UI Polish
- **Compact status bar**: operator callsign shown without parenthetical name (`Op SP9SPM` instead of `Op SP9SPM (Szymon)`). Log/Call/Op labels merged when values are identical — e.g. `Log/Call/Op SP9SPM` instead of three separate fields. Rig name shown when model differs from operator.
- **Partner identity line**: US state (short codes), short continent names, and local time for DX stations.
- **Worked panel**: distribution labels prefixed with scope (`Call Bands`, `DXCC Modes`, `Grid Grids`). Local logbook and Wavelog data merged into a single unified panel with compact row format.
- **DXC table**: new DXCC, band, and mode cells highlighted with color when they're new for the operator.
- **Scroll indicators**: BPL, DXC, and REF views now show `▲ more above` / `▼ more below` hints when content overflows.
- **REF view**: title header, scroll indicator, and proper table height constants added.
- **Logbook editor**: new search-by-call, country, and name filter bar.
- **PSK Reporter**: side-by-side layout replaced with DXC-style filter bar and full-width map.

### Fixes
- **Wavelog download key**: changed from `Ctrl+W` to `Alt+W` to avoid conflict with the `Ctrl+W` close-tab shortcut in terminal multiplexers.
- **Contest QSO matching**: all contest-related queries now match by `contest_adif_id` — fixes edge cases where QSOs from different contests with the same callsign were incorrectly cross-matched.
- **Big CTY**: longitude sign convention corrected (west positive → negative). Redundant DXCC grid auto-fill removed. DXCC lookup migrated from `cty.dat` binary format to Big CTY CSV with ADIF entity numbers. Full DXCC backfill after bulk Wavelog download.
- **Map**: station marker always renders on initial load. Map height cap removed — fills available space on tall terminals. Tile CRS correctly re-initialized on internet restore.
- **Dashboard**: SSE subscriber buffer increased from 16 to 128 to prevent event loss. Redundant operator/logbook SSE events suppressed. Force-push on reconnect and logbook/rig/contest/operator toggle. Debug mode propagated from TUI.
- **Internet detection**: adaptive polling with debounced offline detection; toast spam suppressed after first disconnect.
- **Help bar**: `operatorForm` cache key fixed for edit mode. `WL` abbreviation expanded to `Wavelog`.
- **Rotor**: `Ctrl+F1` stop binding removed; status dot resets correctly on stop.
- **Contest duration**: format simplified to minutes or `H:MM` — seconds dropped.
- **HTTP dashboard**: address text input replaced with a toggle (`127.0.0.1` / `0.0.0.0` / custom).
- **Station info**: `stripNonDigits` moved to `qso` package. Flrig/hamlib defaults and unused helpers removed. GeneralMenu cursor index fixed for 10-item list.

### Under the Hood
- **~70 commits**, **~45 files changed**. All 30 test packages pass. No new dependencies, zero cgo. Backward-compatible — no config or database migration needed from v0.9.0.

## v0.9.0 — 2026-07-14

> **Breaking change.** This release introduces `config_version: 1` and database `PRAGMA user_version = 1`. Config is auto-migrated on load (legacy keys → nested callbook structure). Database migrations are applied once and skipped on subsequent starts. No manual intervention required for upgrades from v0.8.7+; pre-v0.8.7 databases are re-backfilled automatically.

### Config v1 — Nested Callbook & Legacy Migration
- **Callbook group**: all callbook providers moved under `integrations.callbook:` (QRZ.com, HamQTH, Callook.info) with per-provider priority and credentials. The old flat keys (`QRZLegacy`, `qrzcom_callbook`, etc.) are auto-migrated on load.
- **Config versioning**: `config_version: 1` written on save. Future migrators can branch on this field.
- **Legacy key migration**: `picture_at_qrz_pane` → `picture_at_partner_pane`, `wavelog_sent` → `qso_sent`, `wavelog_errors` → `all_errors`. Old keys are silently upgraded.
- **CTY.DAT always-on**: the prefix-to-DXCC lookup now runs unconditionally — removed from the General menu toggle. Always available as the ultimate callbook fallback.

### Multi-Provider Callbook
- **HamQTH**: free global callbook with name, QTH, grid, country, CQ/ITU zones, and DXCC. Requires a free account.
- **Callook.info**: free US-focused callbook. No account needed — fast FCC lookups for US callsigns.
- **Priority cascading**: providers are tried in configured priority order. When a higher-priority provider fails or is disabled, the next is queried automatically.
- **Base-call fallback**: when enabled (default: on), CQOps also tries the base callsign (e.g. `SP9MOA` from `DL/SP9MOA/P`) if the full call returns no match from any provider.
- **Callbook menu**: new `F9 → Callbook` top-level menu — separate from the Integration menu. Configure providers, test connections, set priorities.
- **Wavelog lookup**: moved into the callbook pipeline as a provider with its own priority slot. Worked/confirmed status from Wavelog is merged with callbook data.

### Offline Resilience
- **Embedded world map**: a ~150 KB equirectangular map image is compiled into the binary. The dashboard falls back to it when internet is unavailable, using EPSG:4326 CRS for correct alignment.
- **Graceful degradation**: dashboard tiles, weather, radar, and QR codes all degrade cleanly offline. No broken images, no JS errors.
- **Offline toast suppression**: network-error toasts fire once on first detection, then go silent. Enables `--offline` for clean portable/field operation without notification spam.
- **Beep suppression**: desktop notification sounds are suppressed when offline — no alert-spam during field ops.

### Dashboard Overhaul
- **Dark theme**: full dark mode with OpenFreeMap dark tiles, dark-themed CSS, and localStorage persistence (applied before first paint). Theme names `bright`, `dark`, `yl`, and `hivis`.
- **Operator badges**: deterministic colour-hashed operator badges in the recent-QSOs table. Consistent colours per operator across sessions.
- **QR link**: configurable QR code link in the dashboard header — defaults to `docs.cqops.com`.
- **Top-QSOs table**: replaced the old definition list with a proper styled table matching the recent-QSOs column layout.
- **Disconnected overlay**: logo + elapsed-time timer when SSE connection drops. Auto-recovers on reconnect, including CRS reset for tile maps.
- **Mid-width responsive breakpoint**: stats panel and table columns adapt at intermediate widths (was missing between narrow and wide tiers).

### Database — Schema Consolidation & Versioning
- **Clean migrations**: all historical `ALTER TABLE` additions folded into the base `CREATE TABLE`. Removed the botched-migration recovery `DELETE` and startup messages to stderr. Migrations are silent and idempotent.
- **Schema versioning**: uses SQLite `PRAGMA user_version` (currently `1`) — migrations are skipped when the database is already at the current version. Future schema changes can target specific version gaps without re-running already-applied work.
- **DXCC column**: `dxcc` entity number added for remote stations (populated by callbook providers).

### Security Hardening
- **Dashboard default bind**: changed from `0.0.0.0` (all interfaces) to `127.0.0.1` (localhost only). Users who need LAN access must set the address explicitly — the safe default protects field operators on public networks.
- **Radar proxy sanitization**: `/radar-proxy/` endpoint now rejects paths containing `..` — prevents path traversal to the upstream CDN.

### Bug Fixes
- **Portable dupe detection**: `IsDuplicateQSO` now matches on `base_call` in addition to exact `call` — logging `SP9MOA` after `DL/SP9MOA/P` on the same band/mode/date correctly shows `DUPE!`.
- **Nil deref guards**: `cycleActiveContest()` and `cycleActiveOperator()` now check `m.App.Logbook != nil` before accessing logbook fields — prevents panic when the active logbook is unset.
- **PSK spot count**: `InsertPSKSpots` now returns `0` (not the pre-commit tally) when `tx.Commit()` fails — the caller no longer receives a misleading count of unpersisted inserts.
- **SSB submode**: force-corrected to `SSB` on frequency change to prevent stale submode values from carrying over between bands.
- **APRS beacon clamp**: interval clamped to 5–180 minutes with save-time validation — out-of-range values no longer produce beacon storms or silent failures.
- **ADIF STX_STRING/SRX_STRING**: now exports the exchange stripped of the RST prefix per ADIF spec (`STX=599` → `STX_STRING=001`, not `STX_STRING=599 001`).
- **Help bar**: `operatorForm` cache key fixed for edit mode — switching between logbook list and operator editor no longer shows stale shortcuts.
- **Wavelog lookup guard**: skips Wavelog result when only DXCC prefix data (not actual Wavelog worked/confirmed) was returned.

### Translations
- **9 languages**: English, Polski, Deutsch, Español, 日本語, Français, Italiano, **Português (BR)**, and **Русский**. All manuals updated with multi-provider callbook, offline map, and config restructure.
- **Exchange markers**: all manuals corrected for the 8-template-marker set (`@rst`, `@serial`, `@cqz`, `@mycqz`, `@itu`, `@myitu`, `@grid`, `@mygrid`).

### CI / Build
- **UPX compression**: binaries are now compressed with `upx --best` in CI — Linux amd64/arm64/armhf and Darwin amd64/arm64. Typical 40–60% size reduction.
- **Winget**: disabled until the manifest PR is accepted by Microsoft. Will re-enable as a fast-follow release.

### Under the Hood
- **~70 commits**, **~95 files changed**. Config auto-migration tested with real v0.8.x `config.yaml`. All 30 test packages pass. No new dependencies, no cgo, no runtime API changes for ADIF, Wavelog, WSJT-X, flrig, or rigctld backends.

## v0.8.13 — 2026-07-12

### Contest Statistics Panel
- **Live stats panel**: when a contest is active and the terminal is wide enough, a compact statistics panel appears to the right of the QSO form with a yellow border. Shows Rate (last 10/100 QSOs), Count (last 60m / current hour), Peak (best 1m/10m/60m sliding window), Avg (session average + duration).
- **Activity chart**: Unicode block-character (`█`) vertical bar chart showing QSOs per minute over the last 60 minutes, scaled to 4 rows.
- **Bottom status bar**: contest line shows ID, name, total QSOs, first QSO time, time since last QSO, next serial number, and on-air time. Responsive — fields hide on narrow terminals.
- **On-air time**: computed as sum of inter-QSO gaps shorter than 30 minutes — approximates active operating time vs idle.
- **Performance**: panel render is signature-cached (like solar panel) — rebuilds only when data changes, not every frame. Data refreshes every 5 seconds. Pre-sized allocations, no goroutine leaks.
- **Accurate totals**: TotalQSOs uses `COUNT(*)` query instead of `len(qsos)` — no longer capped at 1000 rows for active contests.
- **DB index**: composite index `idx_qsos_contest_date_time` so the ListQSOs contest query satisfies both WHERE and ORDER BY from a single index scan.

### REF Search — Diacritic and Case Insensitive
- **Unicode-aware search**: `normalizeForSearch()` strips diacritics and lowercases — `ćwilin`, `cwilin`, and `Ćwilin` all find `Ćwilin`. Uses `golang.org/x/text` NFD normalization.
- **search column**: new column in the refs table populated during rebuild. Existing databases auto-detect missing backfill and trigger a rebuild on next restart. Fallback preserves old behavior for unpopulated databases.
- **Backspace fix**: Backspace now works as normal character deletion in the REF search box. **Delete** key clears the entire search. Help overlay updated (`Del → Clear`).

### Keybinding Consistency Pass
- **Rotor**: removed `Ctrl+↑/↓` and `Ctrl+A` — `Alt+;`/`Alt+'`/`Alt+\` are the only rotor shortcuts now. No more conflict with rig tune or "select all" surprise.
- **DXC help**: continent filter label `\ → Sp Cont` (was vague `\ → Continent`).
- **Standardized**: all help bars now say `Space` instead of `Spc`.
- **Stale comment**: rotor handler doc fixed (was `Ctrl+R`, actually `Alt+\`).
- **Manual**: full keybinding section updated to match actual bindings. Favorites section corrected (3 slots via Alt+Ins/Home/PgUp, not 10 via Alt+0–9).

### Duration Display
- **Seconds dropped**: `formatDurationShort` now returns `H:MM` (≥1 hour) or `M` (<1 hour). Per-minute refresh makes seconds meaningless. `Sess 1:18` instead of `Sess 1:18:55`.

### ADIF Export — Contest Filenames
- **Contest-aware filenames**: when a contest filter is active, the exported filename includes the contest ADIF ID and date: `20260712_150405_sp9spm_IARU-HF_20260712.adi`.
- **OS-safe sanitization**: all filename-unsafe characters (`/ \ : * ? " < > |`) replaced with `-`, spaces with `_`.

### Distribution & Packaging
- **nfpm.yaml**: removed bogus `libc6` dependency (binary is statically linked, CGO_ENABLED=0). License corrected to Apache-2.0. RPM packaging support added.
- **Release workflow**: rewritten with `validate-version` job, RPM packages (x86_64 + aarch64), Cloudsmith publishing via OIDC, versionless `cqops_amd64.deb` for stable download links. SHA-256 checksums generated for all assets.
- **winres.json**: version auto-injected from `VERSION` file by build scripts. File description updated.
- **Package metadata**: descriptions in nfpm, NSIS installer, Windows resource file, install scripts, and .desktop file all updated to match README tone.
- **README**: new Installation section with WinGet, Cloudsmith APT/RPM, AUR, and Go methods. Release assets table includes RPM. Cloudsmith OSS hosting badge and attribution.

### Licenses
- Added `CHARM-X-TERM-MIT-LICENSE` for `github.com/charmbracelet/x/term`.
- Updated `third_party/NOTICE.md` with the new entry.

## v0.8.12 — 2026-07-12

### Recent QSOs Table — Full-Width + Smart Columns
- **Full terminal width**: the recent QSOs table is no longer capped at 140/200 columns. On large screens, it uses all available space — richer column tiers appear naturally and text-heavy columns (Name, QTH, DXCC) stop truncating. Small-screen behavior is unchanged.
- **Smart column caps**: every column has a reasonable maximum width — `Call` caps at 12, `Comment` at 30, `Band` at 7, `Mode` at 6, etc. Extra space on ultra-wide screens flows to text-heavy columns via iterative redistribution instead of blowing up short fields.
- **Notes column removed**: the rarely-used Notes column is removed from all tiers. Its 12-char allocation is redistributed to Name, QTH, Comment, and reference fields.
- **Reference fields breathe**: SOTA, POTA, WWFF, IOTA, SIG caps raised so they absorb leftover space on huge monitors instead of it all dumping to the last column.
- **Contest exchange columns**: when contest mode is active, `ExchSent` and `ExchRcvd` replace SOTA/POTA/WWFF/IOTA/SIG at the wide tiers. Non-contest behavior is unchanged.

### DXC Dupe Markers — Spotter-Aware
- **DXC table**: already-worked spots show a `D ` prefix before the callsign (dimmed) — visually distinct from new spots. A single batch query (`DXCDupeSet`) checks all spots against logged QSOs with zero per-spot DB access.
- **DXC path line**: dupe spots in the band-line above the QSO form use the same `D ` prefix convention for consistency.
- **Contest-aware**: in contest mode, dupe checks span the entire contest (48h+), not just today's date. Switching logbooks or contests invalidates the dupe cache automatically.
- **Instant refresh**: dupe markers update immediately after logging a QSO — no waiting for the next spot drain cycle.
- **SQLite covering indexes**: `idx_qsos_date_call_band_mode` and `idx_qsos_contest_call_band_mode` let SQLite answer dupe queries from the index alone, avoiding table scans on every DXC table rebuild.
- **Monochrome-safe**: all dupe markers use text characters (`D ` prefix), not just color, so they work on simple terminals and SSH sessions.

### IARU Region Fix
- **Region 0 default**: `Normalize()` now defaults unset `IARURegion` to 1 (Europe) for all logbooks. Previously, a missing config key silently mapped to Region 2 (widest) limits, causing incorrect out-of-band frequency warnings on 40m (red at 7.300 instead of 7.200 for EU stations).
- **Tests**: 40 new test cases for `IsInHamBand` covering all three IARU regions, band edges, and out-of-band frequencies.

### DXC Continent Filter Fix
- **Spotter continent**: the continent filter now operates on the spotter's continent (`SpotCont`) instead of the spotted station's continent (`DXCont`). Press `\` to filter for spots heard FROM a specific continent. Filter label updated to `Sp Cont`.

### UI Polish
- **Help bar decluttered**: `Ctrl+F` (Spot→Call), `Ctrl+↑` (Rig +step), and `Ctrl+↓` (Rig −step) removed from the default bottom bar. Still available via the `?` help overlay — keeps the bottom line clean on portable/small screens.
- **Dashboard favicon**: updated to the rebranded CQOps icon.

### Under the Hood
- **14 files changed**, 1 new test file (40 cases). No dependency changes, no config format changes, no breaking API changes.

## v0.8.11 — 2026-07-10

### Critical Fixes
- **Database orphan**: `NewID()` now uses deterministic SHA-256 hashing instead of `time.Now().UnixNano()`. Previously, running the wizard twice could produce different database filenames, leaving imported QSOs stranded in an orphaned file. The database is now also reopened after the wizard when the logbook changes.
- **WSJT-X auto-recovery**: tick handler no longer checks `rp.WsjtxEnabled` directly — it always delegates to `MaybeRestartWSJTX`, which has its own change-detection. This prevents the auto-recovery from re-enabling WSJT-X after the user intentionally disabled it.
- **Desktop notifications on Windows**: `desktopAvailable()` now returns `true` on Windows (`runtime.GOOS` check), fixing silent notification failures on Windows 10/11.

### Integration Fixes
- **Hamlib VFO name query spam**: VFO name query ("v" command) is attempted only once per connection via `vfoNameOK` flag. On rigs that don't support it (e.g., Xiegu G90), this eliminates 12,000+ retries per session.
- **Power clamping**: `clampRigPower()` applies `math.Floor` then clamps to the rig preset's max power, fixing a Xiegu G90 displaying 21W when set to 20W due to firmware rounding.
- **WSJT-X power priority**: `txPowerForWSJTX` now directly sets QSO power before `ApplyStationDefaults`. Previously, `ApplyStationDefaults` only filled empty fields, so the WSJT-X ADIF `tx_pwr=10W` survived even when the form showed 21W from hamlib.
- **Kitty guard**: `ensureKitty()` now checks `kittyTerminalEnv()` in addition to `picture.KittySupported()`, preventing false Kitty activation on terminals that pass the probe but lack true graphics support.
- **Dashboard HTTP/DXC mid-run enable**: stale backoff timers and incomplete state reset are now cleared unconditionally when disabled or offline, fixing services that wouldn't start after being toggled on mid-session with the "Enable when CQOps starts" checkbox off.
- **APRS double log**: removed redundant "APRS: connected" log from the client run-loop (the app-level `OnStatus` callback already logs it).

### Dashboard Performance
- **Active QSO dedup**: field-level cache comparison in `pushDashboardFast` skips `SetActiveQSO` and its debug log when nothing changed since the previous tick, reducing ~10,000 redundant pushes per session to 1 per change.
- **Partner dedup**: same field-level cache for partner lookups — `partner pushed` log fires only when QRZ/Wavelog data actually changes, not every tick.
- **Empty partner guard**: `partnerEmpty` flag prevents the "partner cleared" debug log from firing every tick when no call is entered.
- **Dashboard throttle**: `pushDashboardState` now throttles to every 2 ticks (~2s) instead of every tick (~1s). The dashboard SSE push is already change-detected, so the slightly slower poll rate is imperceptible while halving per-tick CPU overhead on low-end hardware.

### Log Cleanup
- **`!BADKEY` fixes**: structured log keys added for dashboard listening URL and APRS reconnect delay.
- **Duplicate debug logs**: `desktopAvailable()` result is cached via `sync.Once` — the "notify: desktop check" debug line fires once at startup instead of twice (or more on repeated calls).
- **Double cursor on Windows**: `\033[?25l` at startup hides the conhost block cursor, preventing a double-cursor artifact alongside Bubble Tea's text cursor.

### Under the Hood
- **14 commits**, 11 files changed (162 insertions, 46 deletions).

## v0.8.10 — 2026-07-09

### Kitty Graphics Protocol — Terminal-Native Images
- **Kitty graphics support** for partner map, PSK Reporter map, inline partner photo, and full-screen photo viewer (F2). No external viewer needed — images render directly in supporting terminals (Kitty, WezTerm, Konsole ≥24.08, Ghostty).
- **Graceful fallback**: ANSI half-block map + Unicode glyph photo placeholder on non-Kitty terminals. Zero configuration — `charm.land/bubbles/v2/picture` handles capability detection.
- **Kitty toggle** in General Settings (`kittyGraphics`) — can be disabled to force ANSI/glyph rendering.
- **Photo viewer**: full-screen image (F2) with ESC to close; Kitty protocol handles sizing and placement automatically.
- **Map cache**: Kitty image dimensions are tracked to avoid redundant re-encodes; only rebuilds when inputs (grid, grayline, window size) actually change.

### GPS Integration — Serial NMEA + GPSD
- **Serial GPS**: connect a USB/RS-232 NMEA receiver (e.g. u-blox) — configure port, baud rate, DTR/RTS in F9 → Integrations → GPS.
- **GPSD**: TCP connection to a local `gpsd` daemon with host/port configuration.
- **Grid precision control**: choose 6, 8, or 10-character Maidenhead grid (F9 → Integrations → GPS Precision). Controls accuracy of position shared via APRS beacons, QSO logging, and dashboard.
- **GPS Grid** toggle on station form: use GPS-derived grid instead of fixed station grid. Auto-updates on position change.
- **APRS beacon grid**: respects GPS precision setting — never transmits a more accurate grid than the user configured.
- **Dashboard weather**: falls back to GPS-derived coordinates when available for Open-Meteo location resolution.

### APRS — KISS TNC Support
- **KISS TNC** (serial/TCP): send APRS position beacons and receive packets via a hardware TNC or software modem (Direwolf, QtSoundModem). Configure in F9 → Integrations → APRS.
- **KISS Server** mode: connect to a remote KISS-over-TCP server for shared TNC access.
- **APRS-IS** (existing): unchanged — internet-based APRS reporting continues to work alongside KISS.
- **Station trails**: dashboard shows last 5 position points for each APRS station with directional arrows on the map.
- **AX.25/KISS tests**: comprehensive test coverage for frame encoding/decoding.

### Portable SOTA/POTA Starting Areas
- **New "Portable" tab** on the Band Plan screen (F7 → right-arrow to PORT). Per-IARU-region suggested CW and SSB starting areas for QRP/portable/SOTA/POTA operations (40m–10m).
- **Not official channels** — clearly labeled as suggestions. Always check band plans, listen, ask QRL, spot exact frequency.
- **Markdown export** (Ctrl+E) includes the Portable section.
- **Data sourced** from IARU Region 1/2/3 band plans and practical field reports.

### Dashboard Enhancements
- **Metric/imperial units**: temperature (°C/°F), wind speed (km|m/h, mph, kn), precipitation (mm/in) — configurable in F9 → General.
- **APRS station trails**: directional path history on the Leaflet map with marker arrows.
- **Wind speed & precipitation formatting**: unit-aware display in the weather module.
- **APRS map**: nearby station markers now use standard APRS symbol icons with improved popup positioning.

### Linux TTY & Bare Terminal Support
- **Bare TTY detection**: auto-detects `TERM=linux`, `XDG_SESSION_TYPE=tty`, or framebuffer console (no `$DISPLAY`).
- **Forced screen clear**: on bare TTYs, `tea.ClearScreen` is issued on every keypress at the outermost `Update()` level — unstoppable by screen handlers.
- **ANSI 16-color palette**: automatic fallback when terminal doesn't support 256 colors.
- **tmux auto-launch**: on Linux console (no desktop), CQOps auto-launches inside `tmux` for proper function-key support (F1–F12).
- **Window size probe**: terminal dimensions are probed at startup to eliminate resize flash on slow machines (Raspberry Pi).

### Map & Partner View Polish
- **Partner map centering**: map and legend now centered horizontally with `lipgloss.PlaceHorizontal`.
- **PSK map centering**: same centering applied to Heard/PSK pane.
- **Map width**: increased from 128→140 chars on large screens; uses full column width on partner page.
- **Inline photo**: properly positioned with asymmetric padding; cache respects `PictureAtQRZPane` toggle (no restart needed).
- **Kitty F2 viewer**: full-screen dimensions match content area; exit properly resets photo dimensions for inline view.

### Config Menu Redesign
- **Borderless menus**: all config choosers (logbook, rig, contest, operator, integration, notifications) use `menuBoxStyle` — no ANSI border escapes that corrupt Kitty graphics placement.
- **Viewport scrolling**: all menu list and edit views now use `bubbles/viewport` with auto-scroll that follows cursor focus.
- **PgUp/PgDown/Home/End** support in all viewport-backed menus.
- **Integration menu**: blank row between header and content for visual breathing room.

### Rig Power Handling
- **Power clamping**: rig power values are floored and clamped to the rig preset's configured maximum — a Xiegu G90 set to 20W will never display 21W due to firmware rounding.
- **WSJT-X power priority**: `txPowerForWSJTX` now directly sets the QSO power before `ApplyStationDefaults`, so the hamlib/flrig form value always wins over WSJT-X ADIF `tx_pwr` — fixes QSOs logged with 10W while the form showed 21W.

### Integration Lifecycle Fixes
- **HTTP server mid-run enable**: stale backoff timer is now reset when HTTP is disabled; server restarts when config is re-saved even if address/port haven't changed.
- **DXC mid-run enable**: `connecting` and `lastAttempt` state is fully reset on disable and internet loss — no more silent failures when toggling DXC on mid-session.
- **WSJT-X CQ transition**: form is cleared when the user starts calling CQ (DX call → empty + transmitting), removing the previous partner's data.

### Key Bindings & Navigation
- **Favorites**: Ctrl+V/B/N to recall favorites 1/2/3; Ctrl+Shift+V/B/N to save.
- **Rotor controls**: Alt+←/→/↑/↓ for azimuth and elevation (Alt only, no Ctrl required).
- **Pane navigation**: Ctrl+←/→ to switch between QSO form, Recent QSOs, and partner/map panes.
- **Comment retention**: Ctrl+K toggles keep-comment mode.
- **Form holding**: Ctrl+H toggles retain-form mode.
- **Tab shortcuts**: Alt+digit labels for Linux console compatibility.
- **Focusable item hints**: space-key indicator on toggles and buttons throughout menus.

### Wizard Cleanup
- **APRS section removed** from first-run wizard (callsign, passcode, TX beacon, interval, radius, symbol, comment, test button).
- **GPS Grid checkbox removed** from first-run wizard.
- Both remain fully available in the regular Settings → Station screen.

### Security & Safety
- **Single-instance guard**: file-lock prevents running two CQOps instances against the same config directory — protects SQLite from concurrent write corruption.
- **QRZ password sanitizing**: password is redacted in error log messages.

### Bug Fixes & Polish
- **Photo cache invalidation**: partner view cache now includes `PictureAtQRZPane` flag — toggling the setting mid-run no longer shows a stale empty column.
- **Toast simplification**: removed internal caching from ToastQueue; dedup window unchanged.
- **ADIF export**: bearing is validated before writing; contest exchange fields use standard ADIF keys.
- **Rig edit restart**: WSJT-X listener is immediately restarted when rig configuration changes, no app restart needed.
- **Rig preset duplication**: Ctrl+D in the rig chooser copies the selected preset.
- **Linux console**: `TERM=xterm-256color` is set as fallback for proper color and key support.
- **Config reset**: Ctrl+Alt+R with confirmation dialog resets configuration to defaults.
- **Cache reset**: Ctrl+Alt+C clears all render caches.
- **Terminal capability logging**: comprehensive environment diagnostics at startup for debugging Linux framebuffer console issues.

### Under the Hood
- **91 commits**, 92 files changed (10,189 insertions, 1,121 deletions).
- **ntcharts v2.2.0**: `picture.Model` and `pictureurl.Model` for Kitty graphics.
- **Dependencies bumped**: Bubble Tea v2, Bubbles v2, Lip Gloss v2, and all Charm ecosystem packages.
- **Build scripts**: `build.sh`/`build.ps1` use correct module path for ldflags version embedding.
- **All tests pass**: `go test ./...` — 34 TUI test files, comprehensive coverage for new GPS, APRS KISS, and power handling code.

## v0.8.9 — 2026-07-05

### CQOps Live — Built-in Browser Dashboard
- **Real-time web dashboard** with SSE push, Leaflet map, and live station display. Enable in F9 → Integrations, then open `http://localhost:8073` in any browser.
- **Live map** with QSO paths, active QSO tracking, partner photo display, day/night terminator overlay, and RainViewer weather radar.
- **Stats panel**: today's QSOs, unique calls, 5m/15m/60m rate tracking, top operators.
- **Recent QSOs table**: 7-row live feed with band/mode color badges, auto-scroll.
- **Band conditions module**: day/night propagation per band group (80–40m, 30–20m, 17–15m, 12–10m) from HamQSL solar data. Always renders full-width in the info box.
- **Solar & geomagnetic modules**: SFI, sunspots, A-index, K-index with color-coded condition thresholds.
- **DXC & PSK Reporter modules**: last spotted station, per-band report counts.
- **Weather row**: current conditions from Open-Meteo (temp, wind, humidity, icon) for the station's grid locator.
- **APRS integration**: nearby stations on the local map with standard APRS symbol icons, range circle, callsign popups, and auto-fit. Optional periodic position beacon with grid locator.
- **QRZ photos** displayed inline in the hero panel when available.
- **Responsive design**: FullHD+ optimized, breakpoints for small screens, narrow layouts, and short viewports. Works on Field Day projector displays.
- **Info box cycling**: modules rotate every 5 seconds, 1 or 2 columns depending on width.
- **Offline-safe**: all third-party services degrade gracefully; dashboard works with cached/local assets.

### ADIF 3.1.7 Compliance
- **FT8** is now exported as a standalone mode (not MFSK+FT8), per ADIF 3.1.7 spec.
- **FT4 and FT2** exported as MFSK with submode FT4/FT2.
- **Mode normalization**: `NormalizeMode` converts standalone FT4/FT2→MFSK+submode, and legacy MFSK+FT8→standalone FT8.
- **Submode display**: rig info and QSO form now include submode; dashboard shows submode via smart `submode||mode` fallback.

### Stats & Rate Calculation
- **Three-tier rate display**: 5-minute, 15-minute, and 1-hour rates replace the single `RatePerHour` field.
- **Rate query robustness**: uses `printf('%s%06s', qso_date, time_on)` for reliable time comparison, fixing off-by-window errors.
- **Stats fields** nowrap+ellipsis for clean overflow handling at any screen width.

### Dashboard UI Polish
- **21 band colors + 4 mode group colors** as CSS variables, used consistently across badges, pills, and table cells.
- **Premium styling**: border strength 0.22→0.35, shadow 0.07→0.12, badge backgrounds 0.08→0.22 for better visibility.
- **Consolidated breakpoints**: weather 8→4, height 4→2, width 3→2 for simpler maintenance.
- **UTC clock** now displays seconds (`23:26:23Z`).
- **Top QSOs** compact redesign: no trophy icons, no rank numbers, km without space, 9 items visible at FullHD+.

### Bug Fixes
- **SQLITE_BUSY on Wavelog status update**: `UpdateWavelogStatus` now retries 5 times with exponential backoff (100ms→1.6s), preventing "database is locked" errors from leaving the local status as "no" when the upload succeeded.
- **WSJT-X event channel overflow**: removed dead `Events` channel write that caused "dropping events" warnings every ~2.6k events. Channel kept initialized for external consumers.
- **HTTP server restart**: now only restarts when address, port, or enabled state changes — header/logo edits no longer trigger unnecessary restarts.
- **WSJT-X TX power**: added `>0` guard with `strconv.ParseFloat` to prevent zero-watt power from rig-in-RX state overwriting WSJT-X reported power.
- **Dashboard enrichment race**: `forcePushDashboardRecent` clears `lastRecentIDs` before pushing enriched QSOs, so country/grid updates from QRZ reach the browser immediately.
- **Top QSOs without grids**: removed `km>0` filter so QSOs without grid squares still appear in the top list.
- **Extra modules cycling**: `cycleExtraModule` now delegates to `updateExtraBox` (was calling itself inconsistently).

### Rebranding
- **New brand colors**: cyan `#08F8F8` and magenta `#F80868` replace the previous green palette.
- **App icon**: `$c` in cyan, `q` in magenta on a dark rounded background. Regenerated across all formats (PNG, XPM, ICO, .syso).
- **README overhaul**: architecture Mermaid diagram showing Station→CQOps→Internet/Dashboard/File I/O flow, platform badges, Quick Install section, tightened feature list, screenshot grouping.

### Refactoring & Cleanup
- **Dead code removal**: ASCII world map rendering, unused functions in `queries_qso.go`, `dxc_filter.go`, `operator_menu.go`, and `styles.go`.
- **Geo package**: coordinate conversion utilities moved from `map_ascii.go` to new `internal/tui/geo.go` with comprehensive tests.
- **8-char grid support**: latitude/longitude calculation now handles extended Maidenhead locators.
- **Duplicate QSO notification**: system beep on dupe detection (configurable via notifications menu).

## v0.8.8 — 2026-06-29

### Hamlib Rigctld — Robust Multi-Rig Support
- **VFO probe overhaul**: try `f VFOA` first to avoid blocking the serial mutex on backends that require VFO-prefixed commands (model 1042). Detect non-VFO backends by inspecting RPRT -1 suffix rejection. Probe timeout increased from 300ms to 2s for slow serial rigs.
- **Drain-before-RPRT fix**: `cmd()` now drains the character-mode repeat BEFORE checking RPRT errors. Previously an RPRT -11 on the `v` command skipped the drain, leaking stale data that poisoned all subsequent reads on the shared connection → permanent `freq=0`.
- **Frequency validation**: values ≤100 kHz (stale "USB", "RPRT 0", "0") now trigger an immediate connection drop instead of silently showing 0 Hz forever.
- **Power query**: non-fatal — no longer drops the shared TCP connection on failure. `powerVfoOK` flag remembers VFO-form rejection and skips retries. Backends that don't support `l VFOA RFPOWER` (model 1042) fall back silently.
- **Disconnected backoff**: polling interval increases from 1s to 10s when rigctld is unreachable, preventing rapid connect/drop cycles that flooded rigctld with TIME_WAIT connections.
- **Rig config menu**: selecting a different rig now immediately disconnects the old hamlib client and connects to the new rig's host:port (`needsRefresh` flag). Previously required exiting the menu first.

### DXC Cluster
- **Band sort on new spots**: cached sort band is reset when fresh spots arrive, so the active band filter re-sorts correctly instead of showing stale order.
- **Logbook switch**: cycling logbooks now auto-requests `SH/FDX 50` so the DXC table is never empty on a fresh logbook.

### QRZ & Wavelog Lookups
- **Completion-aware skip**: QRZ and Wavelog lookups now skip dispatch if already completed for the same call sign, eliminating redundant HTTP requests.
- **Mode normalization**: rig mode (USB/LSB) is normalized to canonical form (SSB) before storing as `wlLastMode`, preventing spurious "pending" state on the Partner screen.
- **Wavelog timeout**: dispatch time is now reset after timeout fires, preventing repeated timeout toasts for the same call.
- **Field navigation**: Wavelog data is only cleared when the normalized band or mode actually changes, not on every keystroke in the QSO form.

### PSK Reporter
- **Band marker colors**: migrated from ANSI 8-bit codes (9–15, rendered dull/grey on modern terminals) to the semantic RGB palette (Primary, Success, Warning, Accent, Info, Error) for clearly distinguishable band dots and legend labels.

### Band Plan
- **Markdown export** (`Ctrl+E` on F7): exports the full IARU Region band plan as `cqops_bandplan.md` in the config directory, with a `Generated by CQOps vX.Y.Z on YYYY-MM-DD` footer linking to cqops.com.
- **FT2 mode**: added to digital mode and spot keyword lists.

### Bug Fixes
- **Windows secrets test**: `TestSave_WritesWithCorrectPermissions` now skipped on Windows (Unix permission bits don't apply).
- **DXC spot fill**: `dxcFillFromSelected` only clears lookup state when the spot call differs from the current form call, preserving in-progress QRZ/Wavelog data.
- **Duplicate check**: mode is now normalized via `NormalizeRigMode` before querying, matching the stored format.

### Polishing
- Toast: always "Hamlib: connected" — the `--vfo` flag cannot be reliably detected from the protocol alone, and guessing wrong produced misleading warnings on both backends.

## v0.8.7 — 2026-06-28

### Encrypted Secrets Store
- **New `internal/secrets` package** — AES-256-GCM encrypted storage for passwords and API keys
- Secrets live in `~/.config/cqops/secrets.enc` (0600 permissions), never in plaintext `config.yaml`
- Key derived from `/etc/machine-id` (Linux) or hostname fallback — tied to the machine
- Auto-migration: plaintext secrets from existing configs migrate to encrypted store on first run
- Protected: QRZ password, DXC login, Wavelog API keys (per logbook)
- Graceful degradation: corruption or wrong-machine → app starts normally, warning toast shown, secrets re-enterable via UI
- Zero CPU overhead after startup: decrypted secrets cached in memory

### Paste Support
- Clipboard paste now works in the **wizard** (station form, rig form, QRZ credentials)
- Clipboard paste now works in the **logbook editor** (inline QSO editing — callsign, comment, notes, etc.)
- Clipboard paste now works in the **station editor** (logbook chooser → Wavelog section)
- All paste targets respect field formatting (uppercase for callsigns, locator normalization, etc.)

### Operator Editor Improvements
- Callsign auto-uppercased on every keystroke (matches StationForm behavior)
- Validation toast shown when leaving callsign field with non-standard value (no digit)
- Validation fires on Tab, Shift+Tab, Up, Down, paste, and save (Ctrl+S)

### Toast System Overhaul
- UTF-8 symbols replace text prefixes: ● (info), ✓ (success), ▲ (warning), ✗ (error)
- Symbols are geometric characters, not emoji — render correctly on B&W terminals
- All integration toasts now use `Integration: message` prefix format:
  - Solar, flrig, Hamlib, Internet, REF, Band Plan, Rig tune
  - QRZ/Wavelog errors, DXC spotted-by notifications

### Help Bar — Visible Key Bindings
- Ins (Create) and Del (Delete) now visible in the bottom bar for:
  - Rig config menu, logbook config menu, contest config menu, operator config menu
- Previously only accessible via the ? help overlay

### Bug Fixes (New)
- **Wavelog upload race**: Recent QSOs table now refreshes immediately after upload completes, no longer shows stale "not sent" status
- **Favorite recall**: frequency now trims trailing zeros (e.g. `14.250000` → `14.25`), matching ADIF export formatting
- **Config validation**: `EnsureConfig()` now applies encrypted secrets before validating, so the app starts correctly with secrets in `secrets.enc`

### Performance — ~70 optimizations across 5 rounds
- Render caches with signature-based invalidation: contest menu, PSK map, solar panel, help overlay, buildContestLine, helpSuffix
- `lipgloss.NewStyle()` eliminated from every hot path: root View() clip styles, DXC spacer/table wrappers, logbook editor dialogs/edit forms, confirm/spot dialog buttons, notifications menu, help overlay
- `fmt.Sprintf` replaced with `strings.Builder`+`strconv` in all cache keys: PSK Reporter, BPL views, logbook editor, QSO form path row, DXC filter info
- DXC: filter-aware spot cache with in-memory raw cache, pre-allocated query slices, `strconv.FormatFloat` for frequency format, `formatDXCSpotTime()` avoids `time.Format`
- PSK Reporter: async DB loading, cached spot map markers, table rowStyle caching
- BPL: precomputed line lists at startup, `bplFreqStr()`/`bplBwStr()` helpers using `strconv`
- RecentQSOs: pre-computed tier max widths at `init()`, O(1) tier lookup
- flrig: 5 goroutines → sequential XML-RPC calls (~10,800 fewer goroutine spawns per 3h session)
- Toast dedup (2s window), `Active()` dirty-flag cache, overlay content cache
- Other: invariant styles promoted to package-level vars, pre-compiled regexps, wizard formBox style cache, logbook download progress message cache

### Code Quality — ~30 fixes across 3 rounds
- Error handling: solar parse errors now logged, tune verify errors logged, import_validate errors include callsign context, WSJT-X event overflow warning
- Refactoring: 130-line lookup result switch extracted from `Update()` to `handleLookupResultMsg()`, shared `handleTuneResult()` for DXC/BPL tunes, `dxcCycleFilter()`/`dxcCycleFilterBack()` generic filter cycling, `clearQRZFields()` reused
- Default host/port constants in `config/`, deprecated `backend` field now warns, `FriendlyError` handles all HTTP codes
- Nil guard on `cycleActiveContest()`, WSJT-X toast nil guard

### Features
- Wider Recent QSOs table when solar panel active — shows Operator + WL columns on ≥166-col terminals
- `map.go` → `map_ascii.go` clarity rename

### Bug Fixes
- WSJT-X status dot now turns green immediately on connect (cache key missing `wsjtx.online`)
- DXC/BPL tune now works when WSJT-X is listening but not transmitting (`wsjtx.online` → `wsjtx.tx`)
- Rig connect toasts suppressed on reconnect loops (`vfoWarned` flag)
- Toast overlay no longer caches full composite (was hiding content on screen switch)
- `nfpm.yaml` fixed: removed invalid `glibc` depends, unnecessary `libsqlite3-0` recommends
- `build.ps1` fixed: removed invalid `GOARCH=armhf`

### Tests
- `store/migrations_test.go` — migration application + idempotency tests
- `internal/rotor/rotor_test.go` — `Status` zero-value test

### Packaging & Scripts
- `uninstall.sh` now matches install-specific PATH line instead of deleting any line containing "cqops"
- `installer/cqops.nsi` comment no longer hardcodes version
- Backup file `build/cqops.exe~` removed

## v0.8.6 — 2026-06-24

### Multi-Operator & Club Station Support
- Operator profiles in config (callsign + name), per-logbook active operator
- Ctrl+O hot-swap through configured operators, space-toggleable in station form
- Operator menu (create/edit/delete) with validation and all-logbook cascade
- WSJT-X auto-log preserves WSJT-X operator; warns on mismatch with active operator
- Wizard auto-creates operator entry from callsign during first-run setup

### Hamlib Rigctld Backend & Rotor Control
- Backend-agnostic rig architecture: flrig (HTTP) and hamlib rigctld (TCP) via shared `RigClient` interface
- Hamlib rigctld support: frequency, mode, VFO, split, power with graceful VFO name query fallback for Xiegu radios
- Hamlib rotctld rotor control backend with TUI integration (azimuth, elevation, stop)
- Per-rig rotor config in rig presets (hamlib host/port)
- VFO mode auto-detection for split-capable radios

### Windows Installer (NSIS)
- `installer/cqops.nsi` — Start Menu shortcuts, PATH integration, Control Panel uninstall entry, license page, solid LZMA compression
- `scripts/build-installer.ps1` — local build with auto `.ico` generation from `cqops.png` via ImageMagick
- Shortcut targets `.exe` directly; Windows Terminal shows embedded icon in tab/taskbar

### Linux Packages (nfpm)
- `nfpm.yaml` — deb, rpm, and archlinux (`pkg.tar.zst`) for amd64 + arm64
- `installer/cqops.desktop` — freedesktop entry with enriched keywords
- `scripts/build-packages.sh` — local cross-platform package build

### Embedded App Icon & Console Icon
- `winres/winres.json` — go-winres config with icon, manifest (DPI-aware, long-path, Win7+), version metadata
- `cmd/cqops/rsrc_windows_*.syso` — compiled Windows resources (icon + manifest)
- Runtime `setConsoleIcon()` via Win32 API — Windows Terminal tab shows CQOps icon

### Error Persistence
- Top-level `recover()` in `main.go` pauses on panic/startup failure so the terminal stays open

### WSJT-X Fixes
- `QsoLoggedMessage` (field-based) now constructs ADIF and saves — no more silently dropped QSOs
- WSJT-X auto-logged QSOs now inherit `ContestID` so they appear in RecentQSOs when a contest is active

### Bug Fixes (Audit)
- Fixed nil map panic when saving operator to uninitialized Operators map
- Fixed `config.Upgrade()` not stamping `State.Version` (was empty stub)
- Fixed invalid `DROP INDEX IF EXISTS` in SQLite migrations; added dedup DELETE before DXC UNIQUE index
- Fixed WSJT-X `unsafe.Pointer` usage with `recover()` and `Kind` check
- Fixed Wavelog `AllDuplicates` detection: iterates all Messages, defaults to false when empty
- Fixed DXC goroutine leak: `stopCh` checks in time.Sleep goroutines, exponential reconnect backoff
- Fixed `ListAllQSOs` OOM risk: internal pagination (500 per page)
- Fixed toast unbounded growth: capped at 20 items
- Fixed `--version` flag: prints version and exits without TUI
- Fixed default `Debug: true` → `false`
- Fixed logbook delete: synchronous `os.Remove` before toast
- Fixed double Wavelog lookup; added retry for QRZ lookups
- Fixed Maidenhead grid calculation (`LatLonToLocator` replaced with correct algorithm)
- Fixed PSK Reporter: per-callsign fetch timestamps with 5-minute cooldown across logbook cycles
- Fixed DXC: selected-row highlight spans full row; filter columns indicated via header
- Fixed DXC: show "DX Cluster not configured" toast when DXC is disabled (F4)
- Fixed bandplan export to match TUI data and formatting
- Fixed photo cache invalidation to reduce CPU usage during rendering
- Fixed photo loading state management in partner view

### Wavelog
- Chunked batch upload (50 QSOs per HTTP call) with individual fallback on duplicate errors
- Operator/grid mismatch detection during upload with normalize-and-retry flow

### Logging & Performance
- Size-based log rotation (10 MB) to prevent disk exhaustion

### CI & Build
- `.github/workflows/release.yml` — 3-job pipeline: build-unix (Go + nfpm), build-installer (Windows NSIS), publish (GitHub Release)
- `Makefile` — added `installer`, `packages`, `installer-all` targets
- `.gitignore` — added `dist/`

### Cleanup
- Removed dead code: `pendingSave`, `screenCON`, `handleCONUpdate`, `viewCON`, tabbar F3 CON, `lookupTimeoutMsg`, `openLogFile()`
- `if`/`else if` chains converted to tagged `switch` (wizard, rig_menu)
- Split inefficient `WriteString` concatenation in `bpl_views.go`
- Removed stale `flrig_integration.go` and `flrig_interface.go` (replaced by `rig_poll.go` + `rig_client.go`)
- Updated README with downloads badge, screenshots, and Unicode normalization package reference

## v0.8.5 — 2026-06-22 (First Public Release)

CQOps is a fast, minimal Go TUI ham radio logger built with Bubble Tea v2.
It targets normal desktops, Raspberry Pi-class hardware, and field/portable setups.

### Core Logging
- Full QSO form with callsign, band, frequency, mode/submode, RST, grid, name, QTH, country, comment, notes
- Automatic WSJT-X QSO logging via UDP ADIF with duplicate detection
- Manual QSO save with dupe check (press Enter twice to confirm)
- Logbook editor: view, edit, delete, filter, and export QSOs
- Recent QSOs table with automatic refresh on new/edit/delete
- Contest mode with exchange fields (STX/SRX), serial parsing, and contest filtering
- Retain comment toggle for quick repeated entries
- ADIF import/export with validation, Wavelog status tracking, and download recovery
- SOTA, POTA, WWFF, IOTA reference fields with auto-fill from REF database

### Integration Suite
- **Wavelog** — upload, private lookup (worked/confirmed status), full download/import
- **QRZ.com** — callbook lookup for name, QTH, grid, country, CQ/ITU zone, and photo
- **flrig** — frequency, mode, and split detection via XML-RPC
- **WSJT-X** — UDP listener, auto-log, TX status indicator, frequency/mode sync
- **DX Cluster** — telnet client with band/continent/mode/time filters, spot dialog, rig tuning
- **PSK Reporter** — spot table with band/time/mode filters and map view
- **Solar data** — hamqsl.com integration with solar flux, A-index, K-index display
- **REF database** — SOTA, POTA, WWFF, IOTA reference search and auto-fill
- **DXCC/CTY.DAT** — country/continent/CQ/ITU zone from callsign prefix
- **SCP** — Super Check Partial database for callsign completion

### TUI & UX
- Status bar with callsign, logbook, rig, operator, UTC/LT clock
- Integration status indicators (Net, WSJT, Rig, WL, QRZ, DXC) with green/red/amber dots
- Partner view with callbook data, logbook stats, azimuthal map, and photo
- Band plan browser (HAM, VHF/UHF, CB, PMR446, Broadcast) with markdown export
- Broadcast presets (BBC, VOA, etc.) with tune-to-frequency
- First-run wizard with station, rig, QRZ, Wavelog, and timezone setup
- Config screen for all integrations, notifications, and appearance
- Log viewer with scrollable text output
- Toast notification system with expiration
- Keyboard-driven navigation with help bar

### Technical
- Pure-Go SQLite via `modernc.org/sqlite` — no CGO, portable to any platform
- Bubble Tea v2 architecture with ~94 TUI source files, 34 test files
- Centralized style/theme system via Lip Gloss v2
- Render caching for expensive views (RecentQSOs, REF table, DXC filter-info, partner map)
- Cross-platform: Windows, Linux, macOS (amd64 + arm64)
- Graceful offline mode — all integrations fail safe when disabled or unreachable
- Structured logging with file rotation
- YAML config with multiple logbook support
- Version check against GitHub releases

### Performance
- Fast startup on Raspberry Pi-class hardware
- No blocking I/O in `View()` — all rendering is pure
- Cached table/map recomputation avoids allocation-heavy frames
- Network calls are async via `tea.Cmd`, never blocking updates

### License
Apache 2.0. See `LICENSE` and `licenses/` for third-party notices.
