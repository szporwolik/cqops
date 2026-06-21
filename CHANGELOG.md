# Changelog

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
