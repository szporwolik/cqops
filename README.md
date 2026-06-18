# CQOps

[![release](https://img.shields.io/github/v/release/szporwolik/cqops?include_prereleases&label=release&color=1f6feb)](https://github.com/szporwolik/cqops/releases)
[![go](https://img.shields.io/badge/Go-1.26-00ADD8)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)

A small, fast, offline-first amateur radio logger for the terminal. Built for portable and field operations where every watt and every CPU cycle counts.

CQOps is a personal project written primarily for my own use and my local club. It is not intended to replace full-featured desktop loggers — if you need a complete shack management solution, check out [Log4OM](https://www.log4om.com/) (Windows), [QLog](https://github.com/foldynl/QLog) (Linux), or [Wavelog](https://www.wavelog.com/) (self-hosted web). CQOps fills a different niche: a lightweight, dependency-minimal CLI tool that runs happily on a Raspberry Pi, an old laptop, or any "potato PC" without a GUI. Perfect for off-grid portable ops, SOTA/POTA activations, and situations where you want fast keystroke-driven logging without the overhead of a desktop environment.

Repo: https://github.com/szporwolik/cqops

## Author

Szymon Porwolik — [szymon.porwolik.com](https://szymon.porwolik.com/)

## Features

- **DX Cluster** — telnet connection to dxspider nodes, live spot table with band/mode/time filters, Enter to fill QSO form, Tab to tune rig to spot frequency and mode
- **Rig control via flrig** — read frequency/mode/band, tune rig from DX spots, automatic mode mapping (CW→CW-L, FT8→DATA-U, etc.)
- **WSJT-X integration** — automatic QSO logging from FT8/FT4 and other digital modes
- **PSK Reporter** — real-time propagation data with band/mode/time filters and world map
- **Solar conditions** — SFI, SSN, A/K indices, geomagnetic field from hamqsl.com with N0NBH threshold highlighting
- **QRZ.com callbook** — one-key callsign lookup with auto-fill of name, QTH, grid, and country
- **Wavelog cloud sync** — batch upload, duplicate detection, private lookup, and station normalization
- **Full ADIF import/export** — compatible with any ADIF-based logging workflow
- **Terminal UI (TUI)** — keyboard-driven, works over SSH, no GUI required
- **Offline-first** — SQLite database, no internet required for core logging
- **Multi-logbook** — switch between station logs with per-logbook station profiles
- **Partner details view** — grid-to-grid distance, bearing, and world map
- **Cross-platform** — Windows, Linux, macOS — including ARM builds for Raspberry Pi and Apple Silicon
- **Potato PC ready** — runs comfortably on Raspberry Pi-class hardware, old laptops, and portable monitors

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
cqops                  # Start interactive TUI
cqops config show      # Show configuration
cqops log add --call SP9ABC --band 20m --freq 14.074 --mode FT8
cqops log list         # List recent QSOs
cqops logbook list     # List logbooks
cqops version          # Print version
cqops --help           # Show all commands
```

### Key Bindings

| Key | Context | Action |
|---|---|---|
| `F1` | Global | QSO form |
| `F2` | Global | QRZ partner lookup |
| `F4` | Global | DX Cluster (spots table) |
| `F5` | Global | PSK Reporter (propagation) |
| `F7` | Global | Logbook editor |
| `F8` | Global | Configuration menu |
| `F9` | Global | Log viewer |
| `F10` | Global | Quit |

## Dependencies

**Core:**
- [Bubble Tea v2](https://charm.land/bubbletea) — Terminal UI framework
- [Bubbles v2](https://charm.land/bubbles) — TUI components (text input, table, viewport)
- [Lip Gloss v2](https://charm.land/lipgloss) — Terminal styling and layout
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [modernc.org/sqlite](https://modernc.org/sqlite) — Pure Go SQLite (no CGO)
- [ntcharts](https://github.com/NimbleMarkets/ntcharts) — Map rendering

**Integrations:**
- [wsjtx-go](https://github.com/k0swe/wsjtx-go) — WSJT-X UDP protocol
- [farmergreg/adif](https://github.com/farmergreg/adif) — ADIF parsing/writing
- [ftl/hamradio](https://github.com/ftl/hamradio) — Grid locator and distance math
- [gen2brain/beeep](https://github.com/gen2brain/beeep) — Desktop notifications

All licenses are permissive (MIT, Apache 2.0, BSD-3). See `licenses/` directory.

## Contributing

This is a personal project. Issues are welcome, and pull requests are accepted — please open them against the `dev` branch.

## License

[Apache-2.0](https://www.apache.org/licenses/LICENSE-2.0)

Copyright (C) 2025-2026 Szymon Porwolik
