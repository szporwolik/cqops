# CQOps

<p align="center">
  <img src="assets/other/gh-logo.png" alt="CQOps logo" width="480">
</p>

[![release](https://img.shields.io/github/v/release/szporwolik/cqops?include_prereleases&label=release&color=1f6feb)](https://github.com/szporwolik/cqops/releases)
[![go](https://img.shields.io/badge/Go-1.26-00ADD8)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)

A small, fast, offline-first amateur radio logger for the terminal. Built for portable and field operations where every watt and every CPU cycle counts.

CQOps is a personal project for my own use and my local club. It's not a full-featured desktop logger — for that, see [Log4OM](https://www.log4om.com/) (Windows), [QLog](https://github.com/foldynl/QLog) (Linux), or [Wavelog](https://www.wavelog.com/) (self-hosted web). CQOps fills a different niche: a lightweight, dependency-minimal CLI tool that runs on a Raspberry Pi, an old laptop, or any "potato PC" without a GUI — perfect for off-grid portable ops, SOTA/POTA, and fast keystroke-driven logging.

## Author

Szymon Porwolik — [szymon.porwolik.com](https://szymon.porwolik.com/)

## Features

- **Fast keyboard logging** — three-column form, Enter to log, Tab ↹ Col / ↑↓ Row navigation, auto date/time, DUPE! detection with two-press override, New Call / New DXCC badges
- **Multi-rig with flrig** — per-rig flrig and WSJT-X config; auto-fills freq, mode, power, split (VFO A/B → Freq/Freq RX)
- **QRZ, DXCC & SCP** — Ins triggers callbook lookup; auto-fills name, QTH, grid, country; prefix-based DXCC and live callsign autocomplete
- **Wavelog cloud sync** — upload, download, duplicate detection, station profile cycling
- **DX Cluster & PSK Reporter** — live spots with filters, spot-to-rig tuning, real-time propagation map
- **Contest logging** — exchange markers (@rst @serial etc.), auto-derived STX/SRX/STX_STRING/SRX_STRING, per-contest QSO filtering
- **Offline-first** — SQLite, REF database (SOTA/POTA/WWFF/IOTA), solar data cached hourly; `--offline` flag for fully disconnected ops
- **ADIF 3.1.7** — full import/export with all station and contest fields
- **Partner view** — distance, bearing, world map, QRZ photo, Wavelog private lookup
- **Raspberry Pi ready** — Windows, Linux, macOS, ARM; runs on potato PCs over SSH

## Screenshots

<p align="center">
  <img src="assets/screenshots/screen-shot-01.png?v=2" width="49%" alt="Screen 1">
  <img src="assets/screenshots/screen-shot-02.png?v=2" width="49%" alt="Screen 2">
</p>
<p align="center">
  <img src="assets/screenshots/screen-shot-03.png?v=2" width="49%" alt="Screen 3">
  <img src="assets/screenshots/screen-shot-04.png?v=2" width="49%" alt="Screen 4">
</p>

## Requirements

- Go 1.26+
- Terminal with 80×24 minimum
- WSJT-X 2.6+ (optional, for automatic digital mode logging)
- flrig (optional, for rig control and spot-to-radio tuning)
- Internet connection (optional, for DX cluster, QRZ, Wavelog, solar data, PSK Reporter)

## Build

The version from the `VERSION` file is always embedded in the binary.

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops

# Build for current platform (output in build/)
make build

# Or cross-compile for all platforms
make build-all

# Run tests
make test

# Lint (requires golangci-lint)
make lint
```

Binaries are placed in the `build/` directory (git-ignored). For smaller binaries, install UPX and run `upx --best build/cqops`.

### Without make (Windows / manual)

```powershell
# Windows PowerShell
.\scripts\build.ps1
```
```bash
# Linux / macOS
./scripts/build.sh
```

Or a one-liner:

```bash
go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$(cat VERSION)" -o build/cqops ./cmd/cqops/
```

## Releases

A GitHub Actions workflow (`.github/workflows/release.yml`) automates the release process. Before triggering it:

1. Update the version in **`VERSION`** (plain version number, e.g. `0.1.1`).
2. Run the **Create Release** workflow from the [Actions tab](https://github.com/szporwolik/cqops/actions) — it builds binaries for all 6 platforms (Windows, Linux, macOS; amd64 + arm64) and creates a tagged GitHub release with auto-generated notes from merged pull requests.

## Usage

```bash
cqops                  # Start interactive TUI (the only way to use CQOps)
cqops --offline        # Start in offline mode (skip all network checks)
cqops --debug          # Enable debug logging
cqops --logbook <name> # Start with a specific logbook
cqops version          # Print version
cqops --help           # Show flags

Flags:
  -o, --offline        Run in offline mode (skip all network checks)
  -d, --debug          Enable debug logging
  -l, --logbook string Logbook name to use
```

## Dependencies

**Core:**
- [Bubble Tea v2](https://charm.land/bubbletea) — Terminal UI framework
- [Bubbles v2](https://charm.land/bubbles) — TUI components (text input, table, viewport)
- [Lip Gloss v2](https://charm.land/lipgloss) — Terminal styling and layout
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [modernc.org/sqlite](https://modernc.org/sqlite) — Pure Go SQLite (no CGO)
- [ntcharts](https://github.com/NimbleMarkets/ntcharts) — Map rendering
- [yaml.v3](https://gopkg.in/yaml.v3) — YAML config parsing
- [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) — Terminal I/O

**Integrations:**
- [wsjtx-go](https://github.com/k0swe/wsjtx-go) — WSJT-X UDP protocol
- [farmergreg/adif](https://github.com/farmergreg/adif) + [farmergreg/spec](https://github.com/farmergreg/spec) — ADIF 3.1.7 parsing/writing & spec types
- [ftl/hamradio](https://github.com/ftl/hamradio) — Grid locator, distance math, DXCC prefix lookup (CTY.DAT)
- [gen2brain/beeep](https://github.com/gen2brain/beeep) — Desktop notifications

**Data:**
- [country-files.com](https://www.country-files.com/) — CTY.DAT DXCC prefix database by Jim Reisert AD1C (public domain factual data, loaded and cached locally)
- [Super Check Partial](https://www.supercheckpartial.com/) — SCP callsign database by Stu Phillips K6TU (public domain contest data, loaded and cached locally)
- [hamqsl.com](https://www.hamqsl.com/) — Solar conditions data (SFI, SSN, A/K indices) via Paul L Herrman N0NBH 
- [PSK Reporter](https://pskreporter.info/) — Real-time propagation spot data by Philip Gladstone
- [SOTA](https://www.sota.org.uk/) — Summits On The Air summit list (public data, loaded and cached locally)
- [POTA](https://pota.app/) — Parks On The Air park list (public data, loaded and cached locally)
- [WWFF](https://wwff.co/) — World Wide Flora & Fauna directory (public data, loaded and cached locally)
- [IOTA](https://www.iota-world.org/) — Islands On The Air directory (downloaded by the user at runtime from iota-world.org for personal non-commercial use; loaded and cached locally)

All licenses are permissive (MIT, Apache 2.0, BSD-3). IOTA data is used under the personal non-commercial terms published by RSGB IOTA Ltd. See `licenses/` directory.

## Contributing

This is a personal project. Issues are welcome, and pull requests are accepted — please open them against the `dev` branch.

## License

[Apache-2.0](https://www.apache.org/licenses/LICENSE-2.0)

Copyright (C) 2025-2026 Szymon Porwolik
