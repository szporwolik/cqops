---
title: CQOps User Manual
description: Complete guide to installing, configuring, and using CQOps — a fast, terminal-first amateur radio logger
---

# CQOps User Manual

## Table of Contents

1. [About CQOps](#about-cqops)
2. [Download & Installation](#download--installation)
3. [First Launch — Setup Wizard](#first-launch--setup-wizard)
4. [Quick Start: Log Your First QSO](#quick-start-log-your-first-qso)
5. [Main Screen Overview](#main-screen-overview)
6. [Common Workflows](#common-workflows)
7. [Core Features](#core-features)
8. [Integrations](#integrations)
9. [Configuration Reference](#configuration-reference)
10. [Keyboard Shortcuts](#keyboard-shortcuts)
11. [Troubleshooting](#troubleshooting)

---

## About CQOps

CQOps is a fast, terminal-first amateur radio logger for operators who need speed, reliability, and low system overhead — in the shack, on a summit, at a field day, or at a shared club station.

**Offline-first.** Local QSO logging does not require internet access. Cached reference data, solar data, and DXCC prefixes remain available after they have been downloaded once. Network integrations such as Wavelog, QRZ.com, DX Cluster, and PSK Reporter require connectivity and are skipped in `--offline` mode.

**Built for field operation.** CQOps is QRP-ready, SOTA/POTA-friendly, and comfortable on Raspberry Pi-class machines, old laptops, and systems without a desktop environment.

**Club-station ready.** CQOps supports multiple logbooks, operator profiles, and rig presets. Switch the active logbook, active operator, or active rig with a single keystroke.

**Portable by design.** CQOps is a single binary written in Go. It has no CGO dependency and no required system services.

**Cross-platform.** Windows, Linux, and macOS are supported on amd64 and arm64.

### Who CQOps Is For

- Portable operators who need fast keyboard logging on low-power hardware.
- SOTA and POTA activators who log offline and upload later.
- Club stations with multiple operators sharing the same station.
- Field day teams using shared machines or Raspberry Pi-class hardware.
- Operators who prefer a terminal workflow over a desktop GUI.

CQOps is not intended to replace full desktop loggers or web-based logbook platforms. It focuses on fast terminal logging, field operation, offline use, and shared-station workflows.

---

## Download & Installation

> [Browse all releases →](https://github.com/szporwolik/cqops/releases)

### Windows

| Package | Link | Notes |
|---------|------|-------|
| **Installer** | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Recommended for most users. Adds CQOps to the Start Menu and PATH. |
| Portable ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Extract and run without installing. |

### Linux — Debian / Ubuntu

| Architecture | Link | Use for |
|-------------|------|---------|
| **amd64** | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | Most Intel/AMD PCs |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | 64-bit ARM systems |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | 32-bit Raspberry Pi OS |

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Portable Tarball

| Architecture | Link | Use for |
|-------------|------|---------|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | Most Intel/AMD PCs |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | 64-bit ARM systems |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | 32-bit Raspberry Pi OS |

### macOS

| Architecture | Link | Use for |
|-------------|------|---------|
| **Apple Silicon** | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | M1/M2/M3 Macs |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Intel Macs |

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### From Source

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build        # Build only; binary is written to build/
make install      # Build and install to the system
```

Source builds require Go 1.26 or newer.

### Requirements

- Terminal size: minimum 80×24 characters.
- Recommended terminal size: 80×43 or larger.
- A modern terminal emulator is recommended. On Windows, use Windows Terminal instead of the legacy console.

### Command-Line Options

```bash
cqops              # Start the TUI
cqops --offline    # Start without network activity
cqops --version    # Print version and exit
cqops --help       # Show help
```

---

## First Launch — Setup Wizard

On first run, CQOps opens a setup wizard for the essential station settings. Network integrations can be skipped; local logging works without them.

1. **Station & Logbook**  
   Configure the initial logbook, station callsign, operator, and grid locator. Optional fields include SOTA/POTA/WWFF references, IARU region, CQ/ITU zone, DXCC, and SIG/SIG Info. Wavelog setup is also available here: URL, API key, station profile ID, Update, and Test.

2. **Rig**  
   Configure a rig preset: name, model, antenna, power, and radio backend. Supported backends are None, flrig, and Hamlib rigctld. Optional settings include Hamlib rotor control and WSJT-X UDP integration.

3. **Integrations**  
   Configure QRZ.com callbook lookup: enable flag, username, masked password, and Test.

4. **General**  
   Select the IANA timezone. CQOps detects the system timezone by default and also provides a scrollable list.

5. **Summary**  
   Review the configuration. Press **Ctrl+S** to save and start CQOps.

**Wizard navigation:** **Ctrl+S** advances after validation. **Esc** goes back. **F10** quits. Space toggles checkboxes. Tab and Shift+Tab move between fields.

All wizard settings can be changed later from the configuration menu with **F9**.

---

## Quick Start: Log Your First QSO

1. **Install and run CQOps.**  
   Download the package for your platform, launch `cqops`, and complete the setup wizard with at least your callsign and grid locator.

2. **Use the QSO form.**  
   The QSO form opens on **F1**. Enter a callsign; CQOps uppercases it automatically. If the active rig is connected through flrig or Hamlib, frequency, band, mode, and submode are filled automatically. Date and time are set to current UTC.

3. **Move through fields.**  
   Use **Tab**, **Shift+Tab**, and **↑/↓**.

4. **Save the QSO.**  
   Press **Enter** or **Ctrl+S**. If a **DUPE!** warning appears, press **Enter** again to save anyway, or **Esc** to cancel.

The new QSO appears immediately in the Recent QSOs table below the form.

---

## Main Screen Overview

```text
┌─ Status Bar ───────────────────────────────────────────────────────────────────┐
│  CQOps v0.8.9  Log Portable  Rig FTDx10  Call SP9MOA/P                          │
│  Net WSJT Hamlib DXC WL                                            23:00L 2100Z │
├─ Tab Bar ───────────────────────────────────────────────────────────────────────┤
│ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮ ╭──────╮         │
│ │F1 QSO│ │F2 QRZ│ │F4 DXC│ │F5 HRD│ │F6 REF│ │F7 BPL│ │F8 LOG│ │F9 CFG│         │
├─ Main Content Area ─────────────────────────────────────────────────────────────┤
│                                                                                  │
│  QSO form, table, map, editor, or active screen content                          │
│                                                                                  │
├─ Help Bar ──────────────────────────────────────────────────────────────────────┤
│  ? Help • Enter Log QSO • F10 Quit                                               │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### Status Bar

The status bar shows the CQOps version, active logbook, active rig, station callsign, and active operator. The right side shows integration status labels and time in local (`L`) and UTC (`Z`).

**Label colors:**

| Color | Meaning |
|-------|---------|
| White/default | Connected or active |
| Yellow | Disabled, connecting, or expected offline |
| Red | Error or disconnected |
| Accent + bold | WSJT-X is transmitting |

Labels that may appear include **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, and **WL**.

### Tab Bar

| Key | Tab | Screen |
|-----|-----|--------|
| F1 | QSO | QSO form and Recent QSOs table |
| F2 | QRZ | Partner view: callbook data, map, stats, photo |
| F4 | DXC | DX Cluster spots and filters |
| F5 | HRD | PSK Reporter spots and propagation map |
| F6 | REF | SOTA/POTA/WWFF/IOTA reference search |
| F7 | BPL | Band plan browser |
| F8 | LOG | Logbook editor, ADIF, Wavelog sync |
| F9 | CFG | Configuration menus |

### Help Bar

The bottom row shows the most relevant shortcuts for the active screen. Press **?** for the full help overlay.

---

## Common Workflows

### Portable / SOTA / POTA Operation

1. **Before leaving home**, run CQOps once with internet access. This allows CQOps to populate caches such as solar data, REF data, and DXCC prefixes.
2. **Verify the cache** before going offline. Check that the Solar panel shows data and that REF search on **F6** returns results.
3. **In the field**, start CQOps with `cqops --offline`. Network activity is skipped, which avoids delays from unreachable services.
4. **Log normally.** Local logging works without internet.
5. **Upload later.** When back online, open the Logbook Editor with **F8** and press **w** to upload unsent QSOs to Wavelog.

### Shared Club Station & Hot-Seat Operation

1. **Add operator profiles:** open **F9 → Operators**, then press **Ins** for each operator. Enter their callsign and name.
2. **Switch the active operator:** on the QSO form, press **Ctrl+O**. The active operator is shown in the status bar and written to the `OPERATOR` field for saved QSOs.
3. **Use hot-seat logging:** operator A logs a QSO, operator B presses **Ctrl+O**, then logs under their own operator profile.
4. **Use Retain when needed:** enable **Retain** if multiple operators need to log the same contact without retyping the full form.

Before saving at a shared station, check the active logbook and active operator in the status bar.

### Private + Club Logbooks

Many operators maintain a personal logbook and one or more club logbooks.

1. **Create logbooks:** open **F9 → Logbooks**, then press **Ins** for each logbook.
2. **Switch the active logbook:** press **Ctrl+L** on the QSO form. The status bar shows the active logbook.
3. **Keep station data separate:** each logbook can have its own station callsign, Wavelog settings, contest settings, and operators.
4. **Dual-log quickly:** enable **Retain**, save the QSO in one logbook, press **Ctrl+L**, then save it again in the other logbook if appropriate.

### Multiple Rigs

1. **Create rig presets:** open **F9 → Rigs**, then press **Ins** for each rig.
2. **Set the backend:** use flrig or Hamlib for CAT-controlled radios. Use None for manually tuned radios.
3. **Switch the active rig:** press **Ctrl+R** on the QSO form.
4. **Operate mixed stations:** for example, use a CAT-controlled HF rig and a manual VHF/UHF rig in the same session.
5. **Configure WSJT-X per rig:** each rig preset can have its own WSJT-X UDP settings.

When the active rig has CAT control, CQOps can fill frequency, band, mode, and submode automatically. For manual rigs, enter them yourself.

### FT8 / WSJT-X Auto-Logging

When WSJT-X is connected through UDP, CQOps can log digital QSOs automatically from WSJT-X ADIF messages.

- Auto-logged QSOs are saved to the active logbook.
- Duplicate auto-logged QSOs are skipped.
- Auto-logged QSOs inherit the active contest ID.
- QSOs appear in Recent QSOs immediately.
- If Wavelog is configured and reachable, auto-logged QSOs can be uploaded automatically.
- If the WSJT-X operator does not match the active operator, CQOps shows a warning.

Check the active logbook, active operator, and active contest before long digital sessions.

### Wavelog Sync

Wavelog sync is optional. CQOps always saves QSOs locally first.

**Upload:** press **w** in the Logbook Editor (**F8**). CQOps uploads unsent QSOs in batches of 50 and tracks per-QSO status: not sent, sent, or error.

**Download:** press **Ctrl+W** in the Logbook Editor. Downloads are incremental. CQOps fetches QSOs newer than the saved `last_fetched_id` for the active logbook. Duplicates are skipped.

If a Wavelog upload fails, the QSO remains in the local logbook and can be retried later. Purging a logbook resets the fetch ID to `0`, allowing a full re-download.

---

## Core Features

### QSO Logging

The QSO form (**F1**) is the primary logging screen. It uses a three-column layout and can auto-fill fields from rig control, QRZ.com, Wavelog lookup, DXCC/prefix data, and REF databases.

**Form fields:**

| Left Column | Middle Column | Right Column |
|-------------|---------------|--------------|
| Date UTC | Mode **(▼)** | Power W |
| Time UTC | Submode **(▼)** | Freq RX |
| Call | Name | SOTA Ref |
| RST sent | QTH | POTA Ref |
| RST rcvd | Grid | WWFF Ref |
| Frequency MHz | Country | IOTA |
| Band **(▼)** | SIG | SIG Info |
| Exch sent ⚠️ | | |
| Exch rcvd ⚠️ | | |

⚠️ Exchange fields appear only when a contest is active. Fields marked **(▼)** cycle with **PgUp/PgDn**.

The bottom row contains:

- **Comment**
- **Keep** — preserves the Comment field between QSOs; toggle with **Ctrl+T**
- **Retain** — preserves the whole form after saving

The path/bearing line shows distance and azimuth when both grid locators are known. It can also show badges such as **DUPE!**, **New Call!**, and **New DXCC!**.

### Auto-Fill Sources

| Source | Fields |
|--------|--------|
| flrig / Hamlib | Frequency, Freq RX if split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| REF database | SOTA, POTA, WWFF, IOTA references |
| Wavelog lookup | Worked/confirmed status when configured |

### Contest Logging

Contests add exchange fields and serial handling to the QSO form.

Create or configure a contest in the Logbook Editor (**F8**) with **Ins**. Set the contest name, date, ADIF contest ID, and exchange templates.

Supported template markers:

| Marker | Replaced With |
|--------|---------------|
| `@rst` | RST sent or received |
| `@serial` | Auto-incrementing serial number |
| `@call` | Your callsign |
| `@grid` | Your grid locator |
| `@name` | Operator name from the operator profile |

Press **Ctrl+C** to cycle the active contest. When a contest is active:

- the QSO form shows exchange fields,
- serial numbers auto-increment,
- Recent QSOs can filter to contest QSOs,
- ADIF export preserves `CONTEST_ID`.

### Logbook Editor

The Logbook Editor (**F8**) is used for QSO management, ADIF import/export, Wavelog sync, and contest-related operations.

**Inline editing:** select a row with **↑/↓**, press **Enter** or **e**, edit the QSO, then save with **Ctrl+S**. Changes are reflected in Recent QSOs immediately.

### ADIF Import & Export

CQOps supports ADIF 3.1.7 import and export. Mode and submode handling follows the ADIF 3.1.7 spec: FT8 is exported as a standalone mode, while FT4 and FT2 are exported as MFSK with the appropriate submode. Imported legacy MFSK+FT8 records are normalised to standalone FT8 automatically.

- **Ctrl+I** imports an ADIF file, validates records, skips duplicates, and shows a summary.
- **Ctrl+E** exports QSOs. Export can include all QSOs or contest-filtered QSOs. Contest IDs are preserved.
- Imported QSOs are marked for Wavelog upload if Wavelog sync is configured.
- The QSO form's **Submode** field works alongside **Mode** — cycle both independently with **PgUp/PgDn**.

### Favorites

Favorites store frequency, mode, and band presets in 10 slots.

| Shortcut | Action |
|----------|--------|
| Alt+0–9 | Recall favorite slot |
| Alt+Shift+0–9 | Save current frequency/mode/band to slot |

Favorites are stored in the configuration and are shared across logbooks.

Example: for a Polish SOTA FM calling setup, enter `145.55`, set mode `FM`, set band `2m`, then press **Alt+Shift+1**. Later, press **Alt+1** to recall it.

### REF Lookup

The REF screen (**F6**) searches SOTA, POTA, WWFF, and IOTA references. Search by prefix, name, or reference designator. Selected references can fill the QSO form.

### Band Plan Browser

The Band Plan Browser (**F7**) provides quick access to amateur bands, VHF/UHF ranges, CB, PMR446, and broadcast presets. A selected frequency can be used to tune the active rig. Band plan data can also be exported as Markdown.

---

## Integrations

### QRZ.com

QRZ.com lookup requires internet access and a QRZ XML subscription.

Press **Ins** on the QSO form to fill callbook fields such as name, QTH, grid, country, CQ/ITU zones, DXCC, and continent. The Partner view (**F2**) can show the operator photo when available.

### Wavelog

Wavelog integration requires internet access. It supports upload, incremental download, and worked/confirmed lookup.

Wavelog is configured per active logbook with URL, API key, and station profile ID. CQOps always saves QSOs locally first; Wavelog upload failure does not lose data.

See [Wavelog Sync](#wavelog-sync).

### flrig

flrig integration uses XML-RPC over HTTP. The default endpoint is `localhost:12345`.

CQOps can read frequency, mode, and power from flrig. Split operation is mapped as VFO A to Frequency and VFO B to Freq RX.

### Hamlib / rigctld

Hamlib rig control uses the `rigctld` TCP daemon. CQOps can query frequency, mode, VFO, split, and power depending on radio support.

Some radios or Hamlib backends do not support every query. CQOps handles missing VFO-name support gracefully where possible.

### Hamlib Rotor / rotctld

Rotor control uses Hamlib `rotctld`. CQOps supports azimuth, elevation, and stop commands.

Useful shortcuts:

| Shortcut | Action |
|----------|--------|
| Ctrl+←/→ | Adjust azimuth by 5° |
| Ctrl+↑/↓ | Adjust elevation by 5° |
| Ctrl+A | Point rotor to calculated path bearing |
| Ctrl+F1 | Stop rotor |

### WSJT-X

WSJT-X integration uses UDP messages from WSJT-X. CQOps parses ADIF messages and can auto-log completed QSOs.

The rig label becomes accent-colored while WSJT-X is transmitting. If the operator reported by WSJT-X does not match the active operator, CQOps shows a warning.

See [FT8 / WSJT-X Auto-Logging](#ft8--wsjt-x-auto-logging).

### DX Cluster

DX Cluster integration uses a telnet connection and requires internet access. The default server is `dxspots.com:7300`.

Filters include band, continent, mode, and age/time. Press **Enter** on a spot to fill the QSO form, tune the active rig, and return to the QSO screen. Press **Space** to tune without filling the form. Press **Backspace** to clear filters.

### PSK Reporter

PSK Reporter integration requires internet access. It provides propagation spots, band/time/mode filters, and an ASCII world map on **F5**.

### APRS

APRS integration uses a TCP connection to an APRS-IS server and requires internet access. The default server is `euro.aprs2.net:14580`.

CQOps receives position reports from nearby stations and displays them on the CQOps Live dashboard local map with standard symbols, callsign popups, and an auto-fit view. A configurable range circle shows the beacon coverage area. A periodic beacon with the station callsign, SSID, grid locator, and optional comment can be sent.

APRS is configured per logbook in the station settings (**F9 → Logbooks → [active logbook] → APRS**).

### Solar Data

Solar data includes SFI, sunspot number, A/K indices, and band-by-band conditions from hamqsl.com. Live updates require internet access. Cached data remains available offline after a successful fetch.

### CQOps Live — Browser Dashboard

CQOps Live is a built-in web dashboard that displays your station activity in real time on any browser — perfect for field day public displays, club station screens, contest monitoring, or keeping an eye on the shack from another room.

**How to enable**

1. Press **F9** to open the main menu, then select **Integrations**.
2. Scroll down to the **HTTP Server** section and check **Enable HTTP server**.
3. Optionally set the address (default `0.0.0.0`) and port (default `8073`).
4. Press **Ctrl+S** to save. The server starts immediately.
5. Open `http://localhost:8073` (or the configured address) in any browser.

**What the dashboard shows**

The dashboard has two display modes that switch automatically:

- **Overview mode** (no active callsign): a live Leaflet map with today's QSO markers and great-circle paths, a recent QSOs table, station info, stats with 5-minute/15-minute/1-hour rate tracking, top operators, and longest-distance QSOs.
- **Active / Now Working mode** (callsign being worked): a prominent callsign display with submode indicator, QRZ photo (if available), band/mode badges, DUPE/NEW CALL/NEW DXCC indicators, distance and bearing, and a highlighted dashed line on the map from your station to the partner's grid.

The **info box** above the local map cycles through modules every 5 seconds: band conditions (day/night propagation per band group from HamQSL solar data), solar activity (SFI, sunspots), geomagnetic field (A/K indices), DX Cluster last spot, and PSK Reporter per-band report counts. Band conditions always renders full-width.

A **weather row** shows current conditions from Open-Meteo (temperature, wind, humidity, icon) for the station's grid locator. Weather data is fetched browser-side and degrades gracefully when offline.

The **local map** (right panel) shows APRS stations with standard symbols, a range circle, and callsign popups. A day/night terminator overlay and RainViewer weather radar are available as optional overlays.

All panels update in real time via Server-Sent Events (SSE) — no page refresh needed.

**Customisation**

In the HTTP Server integration form you can configure:

| Field | Description |
|-------|-------------|
| Header 1 | Main title shown in the page header and hero area. Falls back to "CQOps Live". |
| Header 2 | Subtitle below the title. Falls back to "Fast, portable ham radio logger". |
| Logo URL | A publicly-accessible image URL displayed in the top-left corner. Falls back to the CQOps logo. |
| Event Start | A date in `YYYY-MM-DD` format. When set, stats and QSO lists are filtered from this date onward — useful for multi-day events. |

**Performance**

The dashboard is designed for low-power hardware. The browser handles all map rendering, distance calculations, and statistics. The CQOps terminal app only pushes lightweight JSON updates via SSE. When the HTTP server is disabled, there is zero overhead — no port is opened and no dashboard goroutines run.

**Typical use cases**

- **Field day / contest public display**: connect a large screen or projector to show the live map and recent QSOs.
- **Club station information screen**: run on a dedicated monitor showing station activity to visitors.
- **Remote monitoring**: open the dashboard on a tablet or phone to watch station activity from another room.
- **Event / fair booth**: configure Header 1/2 and the club logo for a branded, professional display.

---

## Configuration Reference

CQOps configuration is stored in:

| Platform | Config path |
|----------|-------------|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Sensitive credentials are stored separately in `secrets.enc` in the same configuration directory. Secrets are encrypted with a machine-tied key, so credentials must be re-entered when moving a configuration to another machine.

Open configuration with **F9**.

| Menu | Configures |
|------|------------|
| Station | Callsign, grid, CQ/ITU zone, IARU region, references |
| Rig | Rig presets, model, antenna, power, backend, rotor, WSJT-X |
| Wavelog | URL, API key, station profile ID |
| QRZ | Username and password |
| DX Cluster | Host, port, login |
| Operators | Operator profiles: callsign and name |
| Logbooks | Station, Wavelog, contest, and operator settings per logbook |
| Notifications | QSO saved alerts, Wavelog status, dupe beep, error sounds |
| General | Timezone, distance units, map, debug mode |

### Multi-Logbook

Use multiple logbooks for home, portable, contest, and club operation. Press **Ctrl+L** to cycle the active logbook. Each logbook keeps its own station details, Wavelog settings, contest settings, and operator settings.

### Multi-Operator

Operator profiles contain an operator callsign and name. Press **Ctrl+O** to cycle the active operator. The active operator is saved in the ADIF `OPERATOR` field and follows Wavelog uploads.

### Multi-Rig

Rig presets store backend, model, antenna, power, rotor, and WSJT-X settings. Press **Ctrl+R** to cycle the active rig.

### Encrypted Secrets

Since v0.8.7, credentials are stored encrypted.

- **Secrets file:** `secrets.enc`
- **Location:** same directory as `config.yaml`
- **Unix permissions:** `0600` where supported
- **Encryption:** AES-256-GCM with a machine-tied key
- **Protected data:** QRZ password, DX Cluster login, Wavelog API keys
- **Migration:** plaintext secrets from older configs migrate on first run
- **Recovery:** if `secrets.enc` is corrupted, CQOps starts with a warning and asks you to re-enter credentials

---

## Keyboard Shortcuts

### Global

| Key | Action |
|-----|--------|
| F1 | QSO form and Recent QSOs |
| F2 | Partner view |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | REF lookup |
| F7 | Band Plan Browser |
| F8 | Logbook Editor |
| F9 | Configuration / main menu |
| F10 | Quit |
| Ctrl+F9 | Log viewer |
| ? | Help overlay |
| Ctrl+L | Cycle active logbook |
| Ctrl+R | Cycle active rig |
| Ctrl+C | Cycle active contest |
| Ctrl+O | Cycle active operator |
| Esc | Back to previous screen |

### QSO Form — F1

| Key | Action |
|-----|--------|
| Tab | Next field |
| Shift+Tab | Previous field |
| ↑ / ↓ | Move within column |
| Enter | Save QSO, with dupe confirmation if needed |
| Ctrl+S | Save QSO from any field |
| Del | Clear all form fields |
| Ins | Lookup: QRZ, Wavelog, DXCC, and dupe check |
| PgUp / PgDn | Cycle band, mode, or submode |
| Ctrl+D | Open spot dialog |
| Ctrl+T | Toggle Keep Comment |
| Ctrl+←/→ | Adjust rotor azimuth by 5° |
| Ctrl+↑/↓ | Adjust rotor elevation by 5° |
| Ctrl+A | Point rotor to bearing from own grid to partner grid |
| Ctrl+F1 | Stop rotor |
| Alt+0–9 | Recall favorite slot |
| Alt+Shift+0–9 | Save current frequency/mode/band to favorite slot |

### Logbook Editor — F8

| Key | Action |
|-----|--------|
| ↑ / ↓ | Navigate rows |
| PgUp / PgDn | Previous or next page |
| Home / End | First or last row |
| Enter / e | Edit selected QSO |
| Delete | Delete selected QSO |
| p | Purge all QSOs |
| Ctrl+C | Cycle contest filter |
| Ctrl+E | Export ADIF |
| Ctrl+I / Tab | Import ADIF |
| w | Upload unsent QSOs to Wavelog |
| Ctrl+W | Download contacts from Wavelog |
| Esc / F6 | Close editor and return to QSO form |

### DX Cluster — F4

| Key | Action |
|-----|--------|
| ↑ / ↓ | Navigate spots |
| Enter | Fill QSO form, tune rig, and return to QSO |
| Space | Tune rig to selected spot and stay on DX Cluster |
| Home | Cycle band filter forward |
| End | Cycle band filter backward |
| `\` | Cycle continent filter |
| Ins | Cycle mode filter forward |
| Del | Cycle mode filter backward |
| PgUp | Cycle time filter forward |
| PgDn | Cycle time filter backward |
| Backspace | Clear all filters |
| Esc / F4 | Return to QSO form |

### Partner View — F2

| Key | Action |
|-----|--------|
| F2 | Cycle Partner view → Photo → Back |
| Esc / F1 | Return to QSO form |

---

## Troubleshooting

### The app does not start

- Check that the terminal is at least 80×24 characters.
- On Windows, use Windows Terminal instead of the legacy `cmd.exe` console.
- Try `cqops --offline` to rule out network timeouts during startup.
- Check the logs directory:
  - Linux: `~/.local/share/cqops/logs/`
  - macOS: `~/Library/Application Support/cqops/logs/`
  - Windows: `%APPDATA%\cqops\logs\`

### Rig does not connect

- For flrig, verify that flrig is running and the port matches the CQOps rig preset. The default is `12345`.
- For Hamlib, verify that `rigctld` is running and that the host and port are correct.
- Check the status label: white/default means connected, yellow means disabled or connecting, and red means failed.
- Reconnect toasts may be suppressed; CQOps can retry silently.

### WSJT-X does not auto-log

- In WSJT-X, check **Settings → Reporting → UDP Server**.
- Make sure the UDP host and port match the active rig preset in CQOps.
- WSJT-X 2.6 or newer is recommended.
- Check that the WSJT status label is active when WSJT-X is running.
- Confirm that the active logbook and active operator are correct before operating.

### Wavelog upload fails

- Verify the Wavelog URL, API key, and station profile ID.
- Check the **WL** status label.
- Upload errors are shown as toasts.
- QSOs remain saved locally even when upload fails.
- Individual QSO failures do not block the rest of the upload batch.

### Config file issues

- Config file:
  - Linux/macOS: `~/.config/cqops/config.yaml`
  - Windows: `%APPDATA%\cqops\config.yaml`
- Secrets file: `secrets.enc` in the same directory.
- If the config is corrupted, move or delete it and restart CQOps. The setup wizard will create a fresh config.
- The `last_fetched_id` field appears only after a successful Wavelog download.

### Performance issues

- Disable map rendering in General settings.
- Disable the Solar panel if it is not needed.
- Close or avoid network-heavy screens such as DX Cluster and PSK Reporter when operating offline.
- Use `cqops --offline` when the network is unreliable or unavailable.

### Reporting Bugs

Before reporting a bug, enable **Debug mode** from **F9 → General → Debug**, or set `debug: true` in `config.yaml`. Reproduce the issue and attach the relevant log.

Report issues on [GitHub Issues](https://github.com/szporwolik/cqops/issues) with:

- CQOps version from `cqops --version`
- Operating system
- Terminal emulator
- Steps to reproduce
- Relevant debug log
