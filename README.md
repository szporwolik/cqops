# CQOps

[![release](https://img.shields.io/github/v/release/szporwolik/cqops?include_prereleases&label=release&color=1f6feb)](https://github.com/szporwolik/cqops/releases)
[![go](https://img.shields.io/badge/Go-1.26-00ADD8)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)

A small, fast, offline-first amateur radio logger for the terminal. Built for portable and field operations where every watt and every CPU cycle counts.

CQOps is a personal project written primarily for my own use and my local club. It is not intended to replace full-featured desktop loggers — if you need a complete shack management solution, check out [Ham Radio Deluxe](https://www.hamradiodeluxe.com/) (Windows), [CQRlog](https://www.cqrlog.com/) (Linux), or [Wavelog](https://www.wavelog.com/) (self-hosted web). CQOps fills a different niche: a lightweight, dependency-minimal CLI tool that runs happily on a Raspberry Pi, an old laptop, or any "potato PC" without a GUI. Perfect for off-grid portable ops, SOTA/POTA activations, and situations where you want fast keystroke-driven logging without the overhead of a desktop environment.

Repo: https://github.com/szporwolik/cqops

## Author

Szymon Porwolik — [szymon.porwolik.com](https://szymon.porwolik.com/)

## Features

- **WSJT-X integration** — automatic QSO logging from FT8/FT4 and other digital modes
- **flrig / rigctld support** — read frequency, mode, and band directly from your rig
- **QRZ.com callbook** — one-key callsign lookup with auto-fill of name, QTH, grid, and country
- **Wavelog cloud sync** — batch upload, duplicate detection, and station normalization
- **Full ADIF import/export** — compatible with any ADIF-based logging workflow
- **Terminal UI (TUI)** — keyboard-driven, works over SSH, no GUI required
- **Offline-first** — SQLite database, no internet required for core logging
- **Multi-logbook** — switch between station logs with per-logbook station profiles
- **Partner details view** — grid-to-grid distance, bearing, and world map
- **Cross-platform** — Windows, Linux, macOS — including ARM builds for Raspberry Pi and Apple Silicon

## Requirements

- Go 1.26+
- Terminal with 75×24 minimum (80×24 recommended)
- WSJT-X 2.6+ (optional, for automatic logging)
- flrig (optional, for rig control)

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

## Dependencies

**Core:**
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — Terminal UI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components (text input)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Terminal styling
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [modernc.org/sqlite](https://modernc.org/sqlite) — Pure Go SQLite (no CGO)
- [wsjtx-go](https://github.com/k0swe/wsjtx-go) — WSJT-X UDP protocol
- [farmergreg/adif](https://github.com/farmergreg/adif) — ADIF parsing/writing

## Contributing

This is a personal project. Issues are welcome, and pull requests are accepted — please open them against the `dev` branch.

## License

[Apache-2.0](https://www.apache.org/licenses/LICENSE-2.0)

Copyright (C) 2025-2026 Szymon Porwolik
