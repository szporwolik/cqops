# CQOps

<p align="center">
  <img src="assets/other/gh-logo.png" alt="CQOps logo" width="480">
</p>

[![release](https://img.shields.io/github/v/release/szporwolik/cqops?include_prereleases&label=release&color=1f6feb)](https://github.com/szporwolik/cqops/releases)
[![downloads](https://img.shields.io/github/downloads/szporwolik/cqops/total?color=1f6feb)](https://github.com/szporwolik/cqops/releases)
[![go](https://img.shields.io/badge/Go-1.26-00ADD8)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![docs](https://img.shields.io/badge/docs-cqops.com-1f6feb)](https://docs.cqops.com/)

A small, fast, offline-first amateur radio logger for the terminal — built mainly for portable/field ops on Raspberry Pi and low-power devices. Supports hotseat operator switching for club stations: multiple ops share rigs and logbooks, swapping in and out with a single keystroke (Ctrl+O).

CQOps is a personal project for my own use and my local club. It's not a full-featured desktop logger — for that, see [Log4OM](https://www.log4om.com/) (Windows), [QLog](https://github.com/foldynl/QLog) (Linux), or [Wavelog](https://www.wavelog.org/) (self-hosted web). CQOps fills a different niche: a lightweight, dependency-minimal CLI tool that runs on a Raspberry Pi, an old laptop, or any "potato PC" without a GUI — perfect for off-grid portable ops, SOTA/POTA, and club stations with rotating operators.

> 📖 **Full documentation, installation guides, and translations at [docs.cqops.com](https://docs.cqops.com/)** — available in English, Polski, Deutsch, Español, 日本語, Français, and Italiano.

## Author

Szymon Porwolik — [szymon.porwolik.com](https://szymon.porwolik.com/)

## Features

- **Fast keyboard logging** — three-column QSO form, Enter to log, dupe detection, badges
- **Multi-operator & club station** — hot-swap operators and logbooks with Ctrl+O / Ctrl+L
- **Multi-rig** — flrig and Hamlib rigctld support, Ctrl+R to cycle rigs
- **QRZ callbook** — Ins triggers lookup, auto-fills name, QTH, grid, country
- **Wavelog sync** — upload, incremental download, per-logbook configuration
- **Encrypted secrets** — AES-256-GCM, machine-tied key, never plaintext
- **DX Cluster & PSK Reporter** — live spots with filters, spot-to-rig tuning, ASCII propagation map
- **CQOps Live** — built-in browser dashboard with live map, recent QSOs, stats, QRZ photos, and top operators. Perfect for Field Day displays, club station screens, or remote monitoring — enable in F9 → Integrations
- **Contest logging** — exchange markers, auto serial numbers, ADIF contest ID
- **Offline-first** — SQLite, cached REF/Solar/DXCC data; `--offline` flag
- **ADIF 3.1.7** — full import/export, contest fields preserved
- **Raspberry Pi ready** — Windows, Linux, macOS, ARM; runs over SSH

See the [documentation](https://docs.cqops.com/) for detailed workflows, configuration, keyboard shortcuts, and troubleshooting.

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

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build        # Build for current platform (output in build/)
make build-all    # Cross-compile for all platforms
make test         # Run tests
```

For smaller binaries, install [UPX](https://upx.github.io/) and run `upx --best build/cqops`.

See the [documentation](https://docs.cqops.com/manual.en.html#download--installation) for pre-built downloads and platform-specific installation.

## Releases

Release notes are in [CHANGELOG.md](CHANGELOG.md). All releases are on the [GitHub Releases](https://github.com/szporwolik/cqops/releases) page.

Each release includes:

| Asset | Target |
|---|---|
| `cqops-setup.exe` | Windows installer (NSIS) |
| `cqops-windows-portable.zip` | Windows portable (no install, amd64) |
| `cqops_amd64.deb` | Debian / Ubuntu amd64 |
| `cqops_arm64.deb` | Debian / Ubuntu arm64 |
| `cqops_armhf.deb` | Debian / Ubuntu armhf (Raspberry Pi) |
| `cqops-linux-amd64.tar.gz` | Linux amd64 portable |
| `cqops-linux-arm64.tar.gz` | Linux arm64 portable |
| `cqops-linux-armhf.tar.gz` | Linux armhf portable |
| `cqops-darwin-amd64` | macOS amd64 (raw binary) |
| `cqops-darwin-arm64` | macOS arm64 (raw binary) |

See the [documentation](https://docs.cqops.com/manual.en.html#download--installation) for download links and install instructions per platform.

## Usage

```bash
cqops              # Start the TUI
cqops --offline    # Start without network activity
cqops --version    # Print version and exit
cqops --help       # Show help
```

Full usage guide, workflows, and keyboard shortcuts are in the [documentation](https://docs.cqops.com/).

**Integrations:**
- [wsjtx-go](https://github.com/k0swe/wsjtx-go) — WSJT-X UDP protocol
- [farmergreg/adif](https://github.com/farmergreg/adif) + [farmergreg/spec](https://github.com/farmergreg/spec) — ADIF 3.1.7 parsing/writing & spec types
- [ftl/hamradio](https://github.com/ftl/hamradio) — Grid locator, distance math, DXCC prefix lookup (CTY.DAT)
- [gen2brain/beeep](https://github.com/gen2brain/beeep) — Desktop notifications

**Data & third-party services**

*Reference data (loaded and cached locally):*
- [country-files.com](https://www.country-files.com/) — CTY.DAT DXCC prefix database by Jim Reisert AD1C (public domain factual data)
- [Super Check Partial](https://www.supercheckpartial.com/) — SCP callsign database by Stu Phillips K6TU (public domain contest data)
- [SOTA](https://www.sota.org.uk/) — Summits On The Air summit list (public data)
- [POTA](https://pota.app/) — Parks On The Air park list (public data)
- [WWFF](https://wwff.co/) — World Wide Flora & Fauna directory (public data)
- [IOTA](https://www.iota-world.org/) — Islands On The Air directory (personal non-commercial use per RSGB IOTA Ltd terms)

*Live data (online, cached locally):*
- [hamqsl.com](https://www.hamqsl.com/) — Solar conditions data (SFI, SSN, A/K indices) by Paul L Herrman N0NBH
- [PSK Reporter](https://pskreporter.info/) — Real-time propagation spot data by Philip Gladstone

*CQOps Live dashboard — map tiles, weather radar, Leaflet:*
- Map tiles: [OpenStreetMap](https://www.openstreetmap.org/copyright) — © OpenStreetMap contributors (ODbL).
- Weather radar overlay: [RainViewer](https://www.rainviewer.com/) public API (browser-side, optional, offline-safe). Attribution displayed on-map and in footer.
- Leaflet 1.9.4 bundled under BSD-2. See `licenses/LEAFLET-BSD2-LICENSE`.
- All services remain optional. CQOps and CQOps Live work offline with cached/local assets.
- Use of third-party services does not imply endorsement of CQOps by those projects.

All licenses are permissive (MIT, Apache 2.0, BSD-2, BSD-3). See `licenses/` directory.

## Contributing

This is a personal project. Issues are welcome, and pull requests are accepted — please open them against the `dev` branch.

## License

[Apache-2.0](https://www.apache.org/licenses/LICENSE-2.0)

Copyright (C) 2025-2026 Szymon Porwolik
