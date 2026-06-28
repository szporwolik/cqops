# CQOps

<p align="center">
  <img src="assets/other/gh-logo.png" alt="CQOps logo" width="480">
</p>

[![release](https://img.shields.io/github/v/release/szporwolik/cqops?include_prereleases&label=release&color=1f6feb)](https://github.com/szporwolik/cqops/releases)
[![downloads](https://img.shields.io/github/downloads/szporwolik/cqops/total?color=1f6feb)](https://github.com/szporwolik/cqops/releases)
[![go](https://img.shields.io/badge/Go-1.26-00ADD8)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)

A small, fast, offline-first amateur radio logger for the terminal. Built for portable and field operations where every watt and every CPU cycle counts.

CQOps is a personal project for my own use and my local club. It's not a full-featured desktop logger — for that, see [Log4OM](https://www.log4om.com/) (Windows), [QLog](https://github.com/foldynl/QLog) (Linux), or [Wavelog](https://www.wavelog.org/) (self-hosted web). CQOps fills a different niche: a lightweight, dependency-minimal CLI tool that runs on a Raspberry Pi, an old laptop, or any "potato PC" without a GUI — perfect for off-grid portable ops, SOTA/POTA, and fast keystroke-driven logging.

## Author

Szymon Porwolik — [szymon.porwolik.com](https://szymon.porwolik.com/)

## Features

- **Fast keyboard logging** — three-column form, Enter to log, Tab ↹ Col / ↑↓ Row navigation, auto date/time, DUPE! detection with two-press override, New Call / New DXCC badges
- **Multi-operator & club station** — per-logbook active operator with Ctrl+O hot-swap, operator profiles (callsign + name), logged OPERATOR field and Wavelog upload follow the active operator
- **Multi-rig with flrig & Hamlib** — per-rig flrig or hamlib rigctld config; auto-fills freq, mode, power, split (VFO A/B → Freq/Freq RX)
- **QRZ, DXCC & SCP** — Ins triggers callbook lookup; auto-fills name, QTH, grid, country; prefix-based DXCC and live callsign autocomplete
- **Wavelog cloud sync** — upload, download, duplicate detection, station profile cycling
- **Encrypted secrets** — QRZ password, Wavelog API keys, DXC login stored AES-256-GCM encrypted; never in plaintext config
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
- Hamlib rigctld (optional, for rig control and spot-to-radio tuning via TCP)
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

Release notes are published in [CHANGELOG.md](CHANGELOG.md).  
All releases are available on the [GitHub Releases](https://github.com/szporwolik/cqops/releases) page.

A GitHub Actions workflow (`.github/workflows/release.yml`) automates the release process. Before triggering it:

1. Update the version in **`VERSION`** (plain version number, e.g. `0.8.7`).
2. Run the **Create Release** workflow from the [Actions tab](https://github.com/szporwolik/cqops/actions) — it builds binaries for 6 platforms (linux/amd64, linux/arm64, linux/armhf, windows/amd64, darwin/amd64, darwin/arm64), Debian packages, portable archives, and a Windows installer, then creates a tagged GitHub release.

Each release includes:

| Asset | Target |
|---|---|
| `cqops-setup.exe` | Windows installer (NSIS) |
| `cqops-windows-portable.zip` | Windows portable (no install, amd64) |
| `cqops_X.Y.Z_linux_amd64.deb` | Debian / Ubuntu amd64 |
| `cqops_X.Y.Z_linux_arm64.deb` | Debian / Ubuntu arm64 |
| `cqops_X.Y.Z_linux_armhf.deb` | Debian / Ubuntu armhf (Raspberry Pi) |
| `cqops-linux-amd64.tar.gz` | Linux amd64 portable |
| `cqops-linux-arm64.tar.gz` | Linux arm64 portable |
| `cqops-linux-armhf.tar.gz` | Linux armhf portable |
| `cqops-darwin-amd64` | macOS amd64 (raw binary) |
| `cqops-darwin-arm64` | macOS arm64 (raw binary) |

### Building installers locally

```powershell
# Windows: NSIS installer (requires makensis + ImageMagick)
.\scripts\build-installer.ps1
```
```bash
# Linux: Debian packages (requires nfpm)
bash scripts/build-packages.sh
```

The Windows installer registers in Control Panel, adds Start Menu shortcuts, and integrates with `%PATH%`. The Linux packages install `cqops` to `/usr/bin/` with a `.desktop` entry and icon.

**Build-time tools used (not linked into the binary):**
[NSIS](https://nsis.sourceforge.io/) (zlib/libpng), [nfpm](https://nfpm.goreleaser.com/) (Apache 2.0), [go-winres](https://github.com/tc-hib/go-winres) (MIT), [ImageMagick](https://imagemagick.org/) (Apache 2.0).

## Usage

```bash
cqops                  # Start interactive TUI (the only way to use CQOps)
cqops --offline        # Start in offline mode (skip all network checks)
cqops --debug          # Enable debug logging
cqops version          # Print version
cqops --help           # Show flags

Flags:
  -o, --offline        Run in offline mode (skip all network checks)
  -d, --debug          Enable debug logging
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
- [golang.org/x/text](https://pkg.go.dev/golang.org/x/text) — Unicode normalization for ADIF

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
