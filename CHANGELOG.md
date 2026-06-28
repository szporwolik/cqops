# Changelog

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
