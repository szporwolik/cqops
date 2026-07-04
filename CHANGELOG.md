# Changelog

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
