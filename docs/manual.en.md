---
title: CQOps User Manual
description: User guide for installing, configuring, and using CQOps — a fast, terminal-first amateur radio logger
---

# CQOps User Manual

CQOps is a fast, terminal-first amateur radio logger for operators who want reliable keyboard logging with low system overhead. It is designed for shack use, portable operation, club stations, field days, and machines such as Raspberry Pi-class devices or older laptops.

CQOps always saves QSOs locally first. Internet-based integrations are optional.

## Contents

1. [What CQOps Is](#what-cqops-is)
2. [Download and Installation](#download-and-installation)
3. [First Launch](#first-launch)
4. [Log Your First QSO](#log-your-first-qso)
5. [Main Screen](#main-screen)
6. [Common Workflows](#common-workflows)
7. [QSO Logging](#qso-logging)
8. [Logbook Editor and ADIF](#logbook-editor-and-adif)
9. [Contests](#contests)
10. [Favorites, References, and Band Plans](#favorites-references-and-band-plans)
11. [Integrations](#integrations)
12. [CQOps Live Dashboard](#cqops-live-dashboard)
13. [Configuration](#configuration)
14. [Keyboard Shortcuts](#keyboard-shortcuts)
15. [Troubleshooting](#troubleshooting)
16. [Reporting Bugs](#reporting-bugs)

---

## What CQOps Is

CQOps is built around fast QSO entry, local-first logging, and practical field operation.

### Main ideas

- **Terminal-first operation** — optimized for keyboard use.
- **Offline-first logging** — local QSO logging works without internet access.
- **Low overhead** — suitable for Raspberry Pi-class systems, older laptops, and shared station PCs.
- **Portable design** — distributed as a single Go binary.
- **Multiple logbooks** — useful for personal, portable, contest, and club logs.
- **Multiple operators** — useful for hot-seat and shared club station workflows.
- **Multiple rigs** — each rig preset can keep its own backend and WSJT-X settings.
- **Optional integrations** — QRZ.com, Wavelog, DX Cluster, PSK Reporter, APRS, rig control, rotor control, solar data, and the CQOps Live browser dashboard.

Local logging does not require internet access. Network features are skipped in `--offline` mode.

### Who CQOps is for

CQOps is a good fit for:

- portable operators,
- SOTA and POTA activators,
- club stations,
- field day teams,
- operators who prefer a terminal workflow,
- stations that need quick switching between operators, logbooks, or rigs.

CQOps is not intended to replace every feature of a full desktop logger or a web-based logbook platform. It focuses on fast terminal logging, field operation, offline use, and shared-station workflows.

---

## Download and Installation

Browse all releases:

<https://github.com/szporwolik/cqops/releases>

### Windows

| Package | Link | Notes |
|---|---|---|
| Installer | [cqops-setup.exe](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Recommended for most users. Adds CQOps to the Start Menu and PATH. |
| Portable ZIP | [cqops-windows-portable.zip](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Extract and run without installing. |

Use **Windows Terminal** rather than the legacy console.

### Linux — Debian / Ubuntu

| Architecture | Link | Use for |
|---|---|---|
| amd64 | [cqops_amd64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_amd64.deb) | Most Intel/AMD PCs |
| arm64 | [cqops_arm64.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_arm64.deb) | 64-bit ARM systems |
| armhf | [cqops_armhf.deb](https://github.com/szporwolik/cqops/releases/latest/download/cqops_armhf.deb) | 32-bit Raspberry Pi OS |

Install the downloaded package:

```bash
sudo dpkg -i cqops_*.deb
```

### Linux — Portable Tarball

| Architecture | Link | Use for |
|---|---|---|
| amd64 | [cqops-linux-amd64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-amd64.tar.gz) | Most Intel/AMD PCs |
| arm64 | [cqops-linux-arm64.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-arm64.tar.gz) | 64-bit ARM systems |
| armhf | [cqops-linux-armhf.tar.gz](https://github.com/szporwolik/cqops/releases/latest/download/cqops-linux-armhf.tar.gz) | 32-bit Raspberry Pi OS |

### macOS

| Architecture | Link | Use for |
|---|---|---|
| Apple Silicon | [cqops-darwin-arm64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-arm64) | M1/M2/M3 Macs |
| Intel | [cqops-darwin-amd64](https://github.com/szporwolik/cqops/releases/latest/download/cqops-darwin-amd64) | Intel Macs |

Install manually:

```bash
chmod +x cqops-darwin-* && sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Build from source

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
make install
```

Source builds require Go 1.26 or newer.

### Terminal requirements

| Requirement | Value |
|---|---|
| Minimum terminal size | 80×24 characters |
| Recommended terminal size | 80×43 characters or larger |
| Recommended Windows terminal | Windows Terminal |

### Basic commands

```bash
cqops              # Start the TUI
cqops --offline    # Start without network activity
cqops --version    # Print version and exit
cqops --help       # Show help
```

---

## First Launch

On first launch, CQOps opens the setup wizard. Only the essential station information is required for local logging. Network integrations can be skipped and configured later.

### Wizard pages

| Page | What it configures |
|---|---|
| Station & Logbook | Initial logbook, station callsign, operator, grid locator, optional references and zones, Wavelog URL/API/station profile ID |
| Rig | Rig preset, model, antenna, power, backend, optional rotor, optional WSJT-X UDP settings |
| Integrations | QRZ.com lookup settings |
| General | IANA timezone |
| Summary | Review and save |

Supported rig backends are:

- None,
- flrig,
- Hamlib `rigctld`.

### Wizard navigation

| Key | Action |
|---|---|
| Ctrl+S | Validate and continue; on Summary, save and start CQOps |
| Esc | Go back |
| F10 | Quit |
| Tab / Shift+Tab | Move between fields |
| Space | Toggle checkboxes |

You can change wizard settings later with **F9**.

---

## Log Your First QSO

1. Start CQOps:

   ```bash
   cqops
   ```

2. Complete the setup wizard with at least your callsign and grid locator.

3. Open the QSO form with **F1**.

4. Enter the contact callsign. CQOps uppercases callsigns automatically.

5. Fill the remaining fields. If the active rig is connected through flrig or Hamlib, CQOps can fill frequency, band, mode, and submode automatically.

6. Press **Enter** or **Ctrl+S** to save.

7. If a **DUPE!** warning appears, press **Enter** again to save anyway, or **Esc** to cancel.

The saved QSO appears immediately in the Recent QSOs table.

---

## Main Screen

CQOps uses a fixed terminal layout:

```text
┌─ Status Bar ───────────────────────────────────────────────────────────────────┐
│  CQOps v0.8.9  Log Portable  Rig FTDx10  Call SP9MOA/P                         │
│  Net WSJT Hamlib DXC WL                                           23:00L 2100Z │
├─ Tab Bar ──────────────────────────────────────────────────────────────────────┤
│  F1 QSO   F2 QRZ   F4 DXC   F5 HRD   F6 REF   F7 BPL   F8 LOG   F9 CFG         │
├─ Main Content Area ────────────────────────────────────────────────────────────┤
│  QSO form, partner view, map, editor, dashboard data, or active screen content  │
├─ Help Bar ─────────────────────────────────────────────────────────────────────┤
│  ? Help • Enter Log QSO • F10 Quit                                              │
└────────────────────────────────────────────────────────────────────────────────┘
```

### Status bar

The status bar shows:

- CQOps version,
- active logbook,
- active rig,
- station callsign,
- active operator,
- integration status labels,
- local time marked as `L`,
- UTC time marked as `Z`.

Common labels include **Net**, **WSJT**, **Rig**, **Flrig**, **Hamlib**, **Rotator**, **DXC**, **WL**, and **GPS**. The GPS label follows the same colour convention — red when disconnected, yellow when connected but without a fix, white when a position fix is acquired.

| Color | Meaning |
|---|---|
| White/default | Connected or active |
| Yellow | Disabled, connecting, or expected offline |
| Red | Error or disconnected |
| Accent + bold | WSJT-X is transmitting |

### Main tabs

| Key | Tab | Screen |
|---|---|---|
| F1 | QSO | QSO form and Recent QSOs |
| F2 | QRZ | Partner view: callbook data, map, stats, photo |
| F4 | DXC | DX Cluster spots and filters |
| F5 | HRD | PSK Reporter spots and propagation map |
| F6 | REF | SOTA/POTA/WWFF/IOTA reference search |
| F7 | BPL | Band Plan Browser |
| F8 | LOG | Logbook editor, ADIF, Wavelog sync |
| F9 | CFG | Configuration menus |

The help bar shows shortcuts relevant to the active screen. Press **?** for the full help overlay.

---

## Common Workflows

### Portable, SOTA, or POTA operation

Before leaving home:

1. Run CQOps once with internet access.
2. Let CQOps download or refresh cached data such as solar data, REF data, and DXCC prefixes.
3. Check that the Solar panel shows data.
4. Check that REF search on **F6** returns results.

In the field:

1. Start CQOps in offline mode:

   ```bash
   cqops --offline
   ```

2. Log normally. QSOs are saved locally.
3. When back online, open **F8** and press **w** to upload unsent QSOs to Wavelog.

### Shared club station and hot-seat logging

1. Open **F9 → Operators**.
2. Press **Ins** to add operator profiles.
3. On the QSO form, press **Ctrl+O** to switch the active operator.
4. Check the active operator in the status bar before saving.
5. Use **Retain** when multiple operators need to log similar contacts without retyping the full form.

The active operator is saved in the ADIF `OPERATOR` field.

### Personal and club logbooks

1. Open **F9 → Logbooks**.
2. Press **Ins** to create each logbook.
3. On the QSO form, press **Ctrl+L** to switch the active logbook.
4. Check the active logbook in the status bar before saving.

Each logbook can keep its own station details, Wavelog settings, contest settings, and operators.

### Multiple rigs

1. Open **F9 → Rigs**.
2. Press **Ins** to create rig presets.
3. Select the backend: None, flrig, or Hamlib.
4. On the QSO form, press **Ctrl+R** to switch the active rig.

A rig preset can include backend, model, antenna, power, rotor settings, and WSJT-X UDP settings.

### WSJT-X digital operation

When WSJT-X UDP integration is enabled, CQOps can receive ADIF messages from WSJT-X and auto-log completed digital QSOs.

Auto-logged QSOs:

- are saved to the active logbook,
- appear in Recent QSOs immediately,
- skip duplicates,
- inherit the active contest ID,
- can be uploaded automatically to Wavelog when Wavelog is configured and reachable.

If the operator reported by WSJT-X does not match the active operator in CQOps, CQOps shows a warning.

Before long digital sessions, check:

- active logbook,
- active operator,
- active contest,
- WSJT-X status label.

### Wavelog sync

CQOps saves QSOs locally first. Wavelog sync is optional.

| Action | Where | Shortcut | Notes |
|---|---|---|---|
| Upload unsent QSOs | Logbook Editor | `w` | Uploads in batches of 50 |
| Download from Wavelog | Logbook Editor | `Ctrl+W` | Incremental download using `last_fetched_id` |

Upload status is tracked per QSO:

- not sent,
- sent,
- error.

If upload fails, the QSO remains in the local logbook and can be retried later. Purging a logbook resets the fetch ID to `0`, allowing a full re-download.

---

## QSO Logging

The QSO form is the main logging screen. Open it with **F1**.

CQOps can fill fields from:

| Source | Fields |
|---|---|
| flrig / Hamlib | Frequency, Freq RX if split, mode, submode |
| QRZ.com | Name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent |
| REF database | SOTA, POTA, WWFF, IOTA references |
| Wavelog lookup | Worked/confirmed status when configured |
| DXCC/prefix data | Prefix and country-related data |

### Form layout

| Left column | Middle column | Right column |
|---|---|---|
| Date UTC | Mode | Power W |
| Time UTC | Submode | Freq RX |
| Call | Name | SOTA Ref |
| RST sent | QTH | POTA Ref |
| RST rcvd | Grid | WWFF Ref |
| Frequency MHz | Country | IOTA |
| Band | SIG | SIG Info |
| Exch sent |  |  |
| Exch rcvd |  |  |

Exchange fields appear only when a contest is active.

The bottom row contains:

- **Comment**,
- **Keep** — preserves the Comment field between QSOs,
- **Retain** — preserves the whole form after saving.

Fields such as Band, Mode, and Submode can be cycled with **PgUp/PgDn**.

### Path, bearing, and badges

When both grid locators are known, CQOps shows distance and azimuth.

The QSO form can also show badges such as:

- **DUPE!**
- **New Call!**
- **New DXCC!**

### Saving

| Key | Action |
|---|---|
| Enter | Save QSO |
| Ctrl+S | Save QSO from any field |
| Esc | Cancel duplicate confirmation |
| Enter on DUPE confirmation | Save duplicate anyway |

---

## Logbook Editor and ADIF

Open the Logbook Editor with **F8**.

Use it for:

- QSO review,
- inline editing,
- deleting QSOs,
- ADIF import,
- ADIF export,
- Wavelog upload,
- Wavelog download,
- contest-related operations.

### Editing QSOs

1. Select a row with **↑/↓**.
2. Press **Enter** or **e**.
3. Edit the QSO.
4. Save with **Ctrl+S**.

Changes appear in Recent QSOs immediately.

### ADIF import and export

CQOps supports ADIF 3.1.7 import and export.

| Action | Shortcut |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

Import validates records, skips duplicates, and shows a summary. Imported QSOs are marked for Wavelog upload when Wavelog sync is configured.

Export can include all QSOs or contest-filtered QSOs. `CONTEST_ID` is preserved.

### Digital mode handling

Mode and submode handling follows ADIF 3.1.7 as described in this manual:

- FT8 is exported as a standalone mode.
- FT4 and FT2 are exported as MFSK with the appropriate submode.
- Imported legacy MFSK + FT8 records are normalised to standalone FT8.

The QSO form has separate **Mode** and **Submode** fields. Both can be cycled with **PgUp/PgDn**.

---

## Contests

Contests add exchange fields and serial handling to the QSO form.

Create or configure a contest in the Logbook Editor with **Ins**.

Contest configuration includes:

- contest name,
- date,
- ADIF contest ID,
- exchange templates.

### Template markers

| Marker | Replaced with |
|---|---|
| `@rst` | RST sent or received |
| `@serial` | Auto-incrementing serial number |
| `@call` | Your callsign |
| `@grid` | Your grid locator |
| `@name` | Operator name from the operator profile |

Press **Ctrl+C** to cycle the active contest.

When a contest is active:

- the QSO form shows exchange fields,
- serial numbers auto-increment,
- Recent QSOs can filter to contest QSOs,
- ADIF export preserves `CONTEST_ID`.

---

## Favorites, References, and Band Plans

### Favorites

Favorites store frequency, mode, and band presets in 10 slots.

| Shortcut | Action |
|---|---|
| Alt+0–9 | Recall a favorite |
| Alt+Shift+0–9 | Save current frequency, mode, and band to a favorite |

Favorites are stored in the configuration and are shared across logbooks.

Example:

1. Enter `145.55`.
2. Set mode to `FM`.
3. Set band to `2m`.
4. Press **Alt+Shift+1**.
5. Later, press **Alt+1** to recall the preset.

### REF Lookup

Open REF Lookup with **F6**.

It searches:

- SOTA,
- POTA,
- WWFF,
- IOTA.

You can search by prefix, name, or reference designator. Selected references can fill the QSO form.

### Band Plan Browser

Open the Band Plan Browser with **F7**.

It provides quick access to:

- amateur bands,
- VHF/UHF ranges,
- CB,
- PMR446,
- broadcast presets.

A selected frequency can be used to tune the active rig. Band plan data can also be exported as Markdown.

---

## Integrations

All integrations are optional. Local logging works without them.

### QRZ.com

QRZ.com lookup requires internet access and a QRZ XML subscription.

On the QSO form, press **Ins** to fill callbook fields such as:

- name,
- QTH,
- grid,
- country,
- CQ/ITU zones,
- DXCC,
- continent.

The Partner view on **F2** can show the operator photo when available.

### Wavelog

Wavelog integration supports:

- upload,
- incremental download,
- worked/confirmed lookup.

Wavelog is configured per active logbook with:

- URL,
- API key,
- station profile ID.

CQOps always saves QSOs locally first. Wavelog upload failure does not delete local data.

### flrig

flrig integration uses XML-RPC over HTTP.

Default endpoint:

```text
localhost:12345
```

CQOps can read:

- frequency,
- mode,
- power.

Split operation maps VFO A to Frequency and VFO B to Freq RX.

### Hamlib / rigctld

Hamlib rig control uses the `rigctld` TCP daemon.

Depending on radio and backend support, CQOps can query:

- frequency,
- mode,
- VFO,
- split,
- power.

CQOps handles missing VFO-name support gracefully where possible.

### Hamlib Rotor / rotctld

Rotor control uses Hamlib `rotctld`.

CQOps supports:

- azimuth,
- elevation,
- stop commands.

| Shortcut | Action |
|---|---|
| Ctrl+←/→ | Adjust azimuth by 5° |
| Ctrl+↑/↓ | Adjust elevation by 5° |
| Ctrl+A | Point rotor to calculated path bearing |
| Ctrl+F1 | Stop rotor |

### WSJT-X

WSJT-X integration uses UDP messages from WSJT-X. CQOps parses ADIF messages and can auto-log completed QSOs.

The rig label becomes accent-colored while WSJT-X is transmitting. If the operator reported by WSJT-X does not match the active operator, CQOps shows a warning.

### GPS

CQOps can read position from a GPS receiver and use it as the station grid
locator — ideal for portable, mobile, or field operations.

Two backends are supported:

- **Serial** — connects directly to a GPS receiver over a serial port
  (USB-to-serial, built-in COM port, or `/dev/ttyUSB0`).
- **GPSD** — connects to a [gpsd](https://gpsd.io/) server over TCP
  (default `127.0.0.1:2947`). Useful when the GPS is shared with other
  applications or accessed over the network.

The GPS status indicator in the status bar shows:

| Colour | Meaning |
|--------|---------|
| Red `GPS` | Disconnected / error |
| Yellow `GPS` | Connected, no fix yet |
| White `GPS` | Fix acquired, position locked |

When a fix is acquired, the station grid locator is replaced with the
GPS-derived position and marked `(GPS)` in the status line:

```
Rig SSB - FTDx10/Dipole  ·  Grid JO62TJ43PL (GPS)
```

Enable **Grid from GPS** in the Station & Logbook settings to use the
GPS grid for QSO logging, APRS beacons, the dashboard map, and distance
calculations.

**Grid precision** — configurable in the Integration menu (10, 8, or 6
characters). Default is 10-char (~25 m accuracy). The grid is always
computed at full precision internally and truncated to the configured
length at the usage layer.

### DX Cluster

DX Cluster integration uses telnet and requires internet access.

Default server:

```text
dxspots.com:7300
```

Filters include:

- band,
- continent,
- mode,
- age/time.

| Key | Action |
|---|---|
| Enter | Fill QSO form, tune rig, and return to QSO |
| Space | Tune rig and stay on DX Cluster |
| Backspace | Clear filters |

### PSK Reporter

PSK Reporter integration requires internet access.

It provides:

- propagation spots,
- band/time/mode filters,
- ASCII world map on **F5**.

### APRS

APRS integration uses a TCP connection to an APRS-IS server and requires internet access.

Default server:

```text
euro.aprs2.net:14580
```

CQOps can receive position reports from nearby stations and display them on the CQOps Live local map with:

- standard symbols,
- callsign popups,
- auto-fit view,
- configurable range circle.

CQOps can also send a periodic beacon with:

- station callsign,
- SSID,
- grid locator,
- optional comment.

APRS is configured per logbook in:

```text
F9 → Logbooks → [active logbook] → APRS
```

### Solar Data

Solar data comes from hamqsl.com and includes:

- SFI,
- sunspot number,
- A/K indices,
- band-by-band conditions.

Live updates require internet access. Cached data remains available offline after a successful fetch.

---

## CQOps Live Dashboard

CQOps Live is a built-in browser dashboard for real-time station activity.

It is useful for:

- field day public displays,
- club station screens,
- contest monitoring,
- watching the station from another room,
- event or fair booths.

### Enable the dashboard

1. Press **F9**.
2. Open **Integrations**.
3. Go to **HTTP Server**.
4. Enable **HTTP server**.
5. Optionally set address and port.
6. Press **Ctrl+S** to save.
7. Open the dashboard in a browser.

Default settings:

| Setting | Default |
|---|---|
| Address | `0.0.0.0` |
| Port | `8073` |
| Local URL | `http://localhost:8073` |

The server starts immediately after saving.

### Display modes

CQOps Live has two display modes.

#### Overview mode

Shown when no active callsign is being worked.

It displays:

- live Leaflet map,
- today's QSO markers,
- great-circle paths,
- recent QSOs table,
- station information,
- statistics,
- 5-minute, 15-minute, and 1-hour rate tracking,
- top operators,
- longest-distance QSOs.

#### Active / Now Working mode

Shown when a callsign is being worked.

It displays:

- large callsign,
- submode indicator,
- QRZ photo when available,
- band and mode badges,
- DUPE / NEW CALL / NEW DXCC indicators,
- distance and bearing,
- highlighted dashed map path from station grid to partner grid.

### Info box

The info box above the local map cycles every 5 seconds through modules:

- band conditions,
- solar activity,
- geomagnetic field,
- latest DX Cluster spot,
- PSK Reporter per-band report counts.

Band conditions always render full-width.

### Weather row

The weather row shows current Open-Meteo conditions for the station grid locator:

- temperature,
- wind,
- humidity,
- icon.

Weather data is fetched browser-side and degrades gracefully when offline.

### Local map

The right-side local map can show:

- APRS stations,
- standard APRS symbols,
- range circle,
- callsign popups,
- optional day/night terminator overlay,
- optional RainViewer weather radar overlay.

### Real-time updates and performance

CQOps Live updates through Server-Sent Events (SSE). No page refresh is needed.

The dashboard is designed for low-power hardware:

- browser handles map rendering,
- browser handles distance calculations,
- browser handles statistics,
- CQOps pushes lightweight JSON updates,
- when the HTTP server is disabled, no port is opened and dashboard goroutines do not run.

### Dashboard customization

In the HTTP Server integration form, you can configure:

| Field | Description |
|---|---|
| Header 1 | Main title shown in the page header and hero area. Falls back to “CQOps Live”. |
| Header 2 | Subtitle below the title. Falls back to “Fast, portable ham radio logger”. |
| Logo URL | Publicly accessible image URL shown in the top-left corner. Falls back to the CQOps logo. |
| Event Start | Date in `YYYY-MM-DD` format. Filters stats and QSO lists from that date onward. |

---

## Configuration

Open configuration with **F9**.

### Configuration files

| Platform | Config path |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Sensitive credentials are stored separately in `secrets.enc` in the same configuration directory.

Secrets are encrypted with a machine-tied key. When moving configuration to another machine, credentials must be entered again.

### Configuration menus

| Menu | Configures |
|---|---|
| Station | Callsign, grid, CQ/ITU zone, IARU region, references |
| Rig | Rig presets, model, antenna, power, backend, rotor, WSJT-X |
| Wavelog | URL, API key, station profile ID |
| QRZ | Username and password |
| DX Cluster | Host, port, login |
| Operators | Operator profiles |
| Logbooks | Station, Wavelog, contest, operator, and APRS settings per logbook |
| Notifications | QSO saved alerts, Wavelog status, dupe beep, error sounds |
| General | Timezone, distance units, map, debug mode |

### Multi-logbook

Use multiple logbooks for home, portable, contest, and club operation.

Press **Ctrl+L** to cycle the active logbook.

Each logbook keeps its own:

- station details,
- Wavelog settings,
- contest settings,
- operator settings.

### Multi-operator

Operator profiles contain:

- operator callsign,
- operator name.

Press **Ctrl+O** to cycle the active operator.

The active operator is saved in the ADIF `OPERATOR` field and follows Wavelog uploads.

### Multi-rig

Rig presets store:

- backend,
- model,
- antenna,
- power,
- rotor settings,
- WSJT-X settings.

Press **Ctrl+R** to cycle the active rig.

### Encrypted secrets

Since v0.8.7, credentials are stored encrypted.

| Item | Value |
|---|---|
| Secrets file | `secrets.enc` |
| Location | Same directory as `config.yaml` |
| Unix permissions | `0600` where supported |
| Encryption | AES-256-GCM with a machine-tied key |
| Protected data | QRZ password, DX Cluster login, Wavelog API keys |

Plaintext secrets from older configs migrate on first run.

If `secrets.enc` is corrupted, CQOps starts with a warning and asks you to re-enter credentials.

---

## Keyboard Shortcuts

### Global

| Key | Action |
|---|---|
| F1 | QSO form and Recent QSOs |
| F2 | Partner view |
| F4 | DX Cluster |
| F5 | PSK Reporter |
| F6 | REF Lookup |
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

### QSO form

| Key | Action |
|---|---|
| Tab | Next field |
| Shift+Tab | Previous field |
| ↑ / ↓ | Move within column |
| Enter | Save QSO, with duplicate confirmation if needed |
| Ctrl+S | Save QSO from any field |
| Del | Clear all form fields |
| Ins | Lookup: QRZ, Wavelog, DXCC, and duplicate check |
| PgUp / PgDn | Cycle band, mode, or submode |
| Ctrl+D | Open spot dialog |
| Ctrl+T | Toggle Keep Comment |
| Ctrl+←/→ | Adjust rotor azimuth by 5° |
| Ctrl+↑/↓ | Adjust rotor elevation by 5° |
| Ctrl+A | Point rotor to bearing from own grid to partner grid |
| Ctrl+F1 | Stop rotor |
| Alt+0–9 | Recall favorite |
| Alt+Shift+0–9 | Save current frequency, mode, and band as favorite |

### Logbook Editor

| Key | Action |
|---|---|
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

### DX Cluster

| Key | Action |
|---|---|
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

### Partner view

| Key | Action |
|---|---|
| F2 | Cycle Partner view → Photo → Back |
| Esc / F1 | Return to QSO form |

---

## Troubleshooting

### CQOps does not start

Check:

- terminal size is at least 80×24,
- Windows users are using Windows Terminal,
- network startup is not blocking by trying:

  ```bash
  cqops --offline
  ```

Check logs:

| Platform | Logs path |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

### Rig does not connect

For flrig:

- verify that flrig is running,
- verify the port in the active rig preset,
- default port is `12345`.

For Hamlib:

- verify that `rigctld` is running,
- verify host and port,
- check that your radio/backend supports the requested data.

Status labels help diagnose the issue:

| Color | Meaning |
|---|---|
| White/default | Connected |
| Yellow | Disabled or connecting |
| Red | Failed |

Reconnect toasts may be suppressed. CQOps can retry silently.

### WSJT-X does not auto-log

Check:

- WSJT-X **Settings → Reporting → UDP Server**,
- UDP host and port match the active rig preset in CQOps,
- WSJT-X 2.6 or newer is used,
- WSJT status label is active,
- active logbook is correct,
- active operator is correct.

### Wavelog upload fails

Check:

- Wavelog URL,
- API key,
- station profile ID,
- **WL** status label.

Upload errors are shown as toasts. QSOs remain saved locally even when upload fails. Individual QSO failures do not block the rest of the upload batch.

### Config file issues

Config file:

| Platform | Path |
|---|---|
| Linux / macOS | `~/.config/cqops/config.yaml` |
| Windows | `%APPDATA%\cqops\config.yaml` |

Secrets file:

```text
secrets.enc
```

The secrets file is stored in the same directory as `config.yaml`.

If the config is corrupted, move or delete it and restart CQOps. The setup wizard will create a fresh config.

The `last_fetched_id` field appears only after a successful Wavelog download.

### Performance issues

Try:

- disable map rendering in General settings,
- disable the Solar panel if not needed,
- avoid network-heavy screens such as DX Cluster and PSK Reporter when offline,
- use `cqops --offline` when the network is unreliable.

---

## Reporting Bugs

Before reporting a bug:

1. Enable **Debug mode** in **F9 → General → Debug**, or set:

   ```yaml
   debug: true
   ```

   in `config.yaml`.

2. Reproduce the issue.
3. Attach the relevant log.

Report issues on GitHub:

<https://github.com/szporwolik/cqops/issues>

Include:

- CQOps version from `cqops --version`,
- operating system,
- terminal emulator,
- steps to reproduce,
- relevant debug log.
