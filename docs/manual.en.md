# CQOps User Manual

## Table of Contents

1. [About CQOps](#about-cqops)
2. [Installation](#installation)
3. [First-Run Wizard](#first-run-wizard)
4. [Main Screen Layout](#main-screen-layout)
5. [QSO Logging](#qso-logging)
6. [Contest Logging](#contest-logging)
7. [Logbook Editor](#logbook-editor)
8. [Configuration](#configuration)
9. [Integrations](#integrations)
10. [Key Bindings Reference](#key-bindings-reference)
11. [Troubleshooting](#troubleshooting)

---

## About CQOps

CQOps is a fast, offline-first amateur radio logger for the terminal. It's built for portable and field operations where every watt and every CPU cycle counts — runs comfortably on a Raspberry Pi, an old laptop, or any "potato PC" without a GUI.

**Key design goals:**
- Keyboard-driven, fast logging
- Works on low-end hardware (Raspberry Pi-class)
- Offline-first with optional cloud sync (Wavelog)
- Pure-Go, single binary, no CGO / no system dependencies
- Runs over SSH, VNC, RDP, or directly in a terminal

---

## Installation

### Windows

Download the installer (`cqops-setup.exe`) from [GitHub Releases](https://github.com/szporwolik/cqops/releases). It adds CQOps to the Start Menu and PATH.

Or download the standalone `cqops.exe` and run it from any terminal.

### Linux

**Debian/Ubuntu (`.deb`):**
```bash
sudo dpkg -i cqops_*.deb
```

**Fedora/RHEL (`.rpm`):**
```bash
sudo rpm -i cqops-*.rpm
```

**Arch Linux (`.pkg.tar.zst`):**
```bash
sudo pacman -U cqops-*.pkg.tar.zst
```

**From source:**
```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
make build
```

### macOS

Download the binary from [GitHub Releases](https://github.com/szporwolik/cqops/releases) and place it in your PATH.

### Requirements

- Terminal with minimum 80×24 characters (80×43 recommended)
- Go 1.26+ (build from source only)
- WSJT-X 2.6+ (optional, for automatic digital mode logging)

### Command-Line Options

```bash
cqops              # Normal start (TUI)
cqops --offline    # Start without any network activity
cqops --version    # Print version and exit
cqops --help       # Show help
```

---

## First-Run Wizard

On first launch, CQOps opens a setup wizard that guides you through:

1. **Station** — callsign, Maidenhead grid locator, IARU region, CQ/ITU zone, continent
2. **Rig** — radio model, antenna, power, backend (flrig or Hamlib rigctld)
3. **QRZ.com** — username and API password (optional, for callbook lookups)
4. **Wavelog** — URL, API key, station profile ID (optional, for cloud sync)
5. **DX Cluster** — host, port, login credentials (optional)
6. **Time Zone** — your local IANA timezone (e.g. `Europe/Warsaw`)

You can skip any integration — everything works offline. All settings can be changed later via the configuration screens.

**Navigation in wizard:** Tab / Shift+Tab between fields, Enter to proceed, Esc to skip or go back.

---

## Main Screen Layout

```
┌─ Status Bar ───────────────────────────────────────────────────────┐
│ SP9SPM │ DEBUG │ DIGI (Yaesu FTDx10) │ OP: SP9SPM │ 14:32 UTC │
├────────────────────────────────────────────────────────────────────┤
│ Net ● WSJT ● Rig ● WL ● QRZ ● DXC ●                                │
├─ Tab Bar ──────────────────────────────────────────────────────────┤
│ [QSO] [Recent] [Partner] [DX Cluster] [PSK] [Solar] [Log] [BPL]   │
├─ Main Content Area ────────────────────────────────────────────────┤
│                                                                     │
│  (QSO form, table, map, etc. — changes with active tab)            │
│                                                                     │
├─ Help Bar ─────────────────────────────────────────────────────────┤
│ F1 QSO  F2 Recent  F3 Partner  F4 DXC  F5 PSK  F6 Solar  F7 BPL   │
└─────────────────────────────────────────────────────────────────────┘
```

### Status Bar

Shows your callsign, active logbook name, active rig name (with model), active operator, and current time in both UTC and local time.

### Integration Status Dots

| Dot  | Meaning                                           |
|------|---------------------------------------------------|
| ● Green  | Connected / online                            |
| ● Red    | Disconnected / error                           |
| ● Amber  | Connecting / transitional state                |
| ○ Grey   | Disabled / not configured                      |

- **Net** — Internet connectivity
- **WSJT** — WSJT-X UDP connection status
- **Rig** — flrig or Hamlib rigctld connection
- **WL** — Wavelog API reachability
- **QRZ** — QRZ.com subscription status
- **DXC** — DX Cluster telnet connection

### Tab Bar

Press **F1–F8** or click tabs to switch between screens:

| Key | Screen        | Purpose                                        |
|-----|---------------|------------------------------------------------|
| F1  | QSO           | QSO entry form                                 |
| F2  | Recent        | Recent QSOs table with inline edit             |
| F3  | Partner       | Callbook data, map, stats for current callsign |
| F4  | DX Cluster    | Live DX spots with filters                     |
| F5  | PSK Reporter  | PSK spot table and propagation map             |
| F6  | Solar         | Solar flux, A/K indices, conditions            |
| F7  | Band Plan     | Band plan browser (HAM, VHF/UHF, CB, broadcast)|
| F8  | Logbook Editor| QSO management, ADIF import/export, Wavelog    |

### Help Bar

The bottom row shows the most important key bindings for the current screen. Press **?** to see all available shortcuts.

---

## QSO Logging

The QSO form (F1) is the primary logging interface. It's a three-column form with fields grouped logically.

### Form Navigation

| Key            | Action                                      |
|----------------|---------------------------------------------|
| Tab            | Move to next field (column-aware)           |
| Shift+Tab      | Move to previous field                      |
| ↑ / ↓          | Move up/down within a column                |
| Ctrl+S         | Save QSO (from any field)                   |
| Enter          | Save QSO (from last field)                  |
| Esc            | Clear form / cancel                         |
| Ins            | Trigger QRZ callbook lookup                 |
| Space          | Toggle retain comment / cycle operator      |
| Ctrl+O         | Cycle through configured operators          |

### Form Fields

| Column 1 (Callsign) | Column 2 (Report)  | Column 3 (Details)  |
|---------------------|--------------------|---------------------|
| Callsign            | RST Sent           | Name                |
| Band                | RST Received       | QTH                 |
| Frequency (MHz)     | Mode               | Grid locator        |
| Frequency RX (MHz)  | Submode            | Comment             |
|                     |                    | Notes               |
|                     |                    | SOTA / POTA / WWFF  |
|                     |                    | IOTA                |

### Logging a QSO

1. Fill in the callsign (auto-uppercased).
2. Band, frequency, mode, and submode are auto-filled from rig if connected (flrig/Hamlib).
3. Date and time are set to current UTC automatically.
4. Press **Tab** through fields or use **↑/↓** to navigate.
5. Press **Enter** to save. The form validates fields and warns about duplicates.

### Duplicate Detection

If a callsign/band/mode combination already exists in the logbook for today, CQOps shows a **DUPE!** warning. Press **Enter** again to confirm and save anyway, or **Esc** to cancel.

### Badges

After saving a QSO, the form may show badges next to the callsign field:
- **New Call** — first time working this callsign in this logbook
- **New DXCC** — first time working this DXCC entity in this logbook

### Auto-Fill from Rig

When a rig is connected (flrig or Hamlib), these fields auto-populate:
- Frequency
- Frequency RX (if split mode detected)
- Mode / Submode

### Auto-Fill from QRZ

Press **Ins** with the callsign field filled to trigger a QRZ lookup. If configured, it fills:
- Name, QTH, Grid locator, Country, CQ zone, ITU zone, DXCC, Continent

### Retain Comment

Press **Space** on the "Retain comment" toggle in the status area, or use the toggle in the form. When enabled, the Comment field keeps its value between QSOs — useful for contest or event logging where every QSO shares the same comment.

### WSJT-X Auto-Logging

When WSJT-X is connected via UDP, QSOs are automatically logged:
- The ADIF record from WSJT-X is parsed and saved.
- Duplicates are detected and silently skipped.
- Mode, submode, frequency, and operator are taken from WSJT-X.
- Auto-logged QSOs appear in Recent QSOs immediately.

---

## Contest Logging

CQOps supports contest logging with exchange fields and serial number tracking.

### Setting Up a Contest

1. Go to **Logbook Editor** (F8).
2. Press **Ins** to create a new contest (or select an existing one).
3. Fill in:
   - **Name** — your label for this contest
   - **Contest date** — date of the contest
   - **Contest ID** — ADIF contest identifier (e.g. `AADX-CW`, `CQ-WW-SSB`)
   - **Exchange Sent** — template for sent exchange (e.g. `@rst @serial`)
   - **Exchange Received** — template for received exchange (e.g. `@rst @serial`)
   - **Prefill exchange** — auto-fill next expected exchange values
4. Make the contest active.

### Exchange Markers

| Marker      | Replaced With                               |
|-------------|---------------------------------------------|
| `@rst`      | RST Sent / RST Received value               |
| `@serial`   | Auto-incrementing serial number             |
| `@call`     | Your callsign                               |
| `@grid`     | Your grid locator                           |
| `@name`     | Your name (from operator profile)           |

### During Contest

- The QSO form gains **STX** (sent exchange) and **SRX** (received exchange) fields.
- Serial numbers auto-increment after each logged QSO.
- STX_STRING and SRX_STRING are derived from exchange templates.
- Recent QSOs table filters to show only contest QSOs when a contest is active.
- The contest menu shows current serial number and QSO count.

### After Contest

- Export QSOs to ADIF with all contest fields preserved.
- Contest QSOs have the `CONTEST_ID` field in ADIF export.
- Upload to Wavelog preserves contest metadata.

---

## Logbook Editor

The Logbook Editor (F8) is the central hub for managing QSOs, ADIF operations, and Wavelog sync.

### Editor Features

| Action              | Key / Menu                                   |
|---------------------|----------------------------------------------|
| View all QSOs       | Table displayed on open                      |
| Edit a QSO          | **Enter** on a row → inline edit form        |
| Delete a QSO        | **Del** on selected row (with confirmation)  |
| Filter QSOs         | Ctrl+F → type callsign/band/mode to filter   |
| ADIF Import         | File menu → Import ADIF                      |
| ADIF Export         | File menu → Export ADIF (all or filtered)    |
| Wavelog Download    | Wavelog menu → Download contacts             |
| Wavelog Upload      | Wavelog menu → Upload unsent QSOs            |

### Inline Editing

1. Select a QSO row with **↑/↓**.
2. Press **Enter** to open the inline edit form.
3. Edit fields — Tab/Shift+Tab to navigate.
4. Press **Ctrl+S** or **Enter** on the last field to save.
5. Changes are reflected in Recent QSOs immediately.

### ADIF Import

- Supports ADIF 3.1.7 format.
- Validates records before import.
- Detects and skips duplicates (by callsign/band/mode/date/time key).
- Shows import summary: total records, imported, duplicates, errors.
- Imported QSOs are tagged for Wavelog upload if Wavelog is configured.

### ADIF Export

- Exports to ADIF 3.1.7 format.
- All standard ADIF fields plus contest fields.
- Option to export filtered results or entire logbook.

### Wavelog Download

Downloads QSOs from a Wavelog server into the local logbook.

- **Incremental**: Only fetches QSOs newer than the last download. The `last_fetched_id` is saved per logbook and used on subsequent downloads.
- **Duplicate-safe**: Already-imported QSOs are detected and skipped.
- **Resumable**: If interrupted, the stored ID is not updated — next download picks up where it left off.
- **Purge-aware**: Purging the logbook resets the fetch ID to 0 (full re-download).

### Wavelog Upload

Uploads locally saved QSOs to Wavelog.

- **Batch upload**: 50 QSOs per HTTP request.
- **Status tracking**: Each QSO shows its Wavelog status (not sent / sent / error).
- **Duplicate handling**: Server-side duplicates are detected and individual QSOs are retried.
- **Operator/grid mismatch**: Detected during upload with normalize-and-retry flow.
- **Race-free refresh**: Recent QSOs table updates immediately after upload completes.

---

## Configuration

All settings live in `~/.config/cqops/config.yaml`.

### Multi-Logbook Support

CQOps supports multiple logbooks — useful for separating club station, portable, and home logging.

- Switch active logbook in the Logbook Editor (F8 → cycle with key).
- Each logbook has its own station, Wavelog, contest, and operator settings.
- QSO form, Recent QSOs, and Partner view reflect the active logbook.

### Multi-Operator Support

- Operator profiles stored in config with callsign and name.
- Per-logbook active operator selectable via **Ctrl+O** or space-toggle.
- Logged OPERATOR field follows the active operator.
- Wavelog upload uses the active operator's callsign.

### Multi-Rig Support

- Configure multiple rigs with different backends (flrig or Hamlib rigctld).
- Per-rig: model, antenna, power, rotor config.
- Switch active rig in the rig menu.
- WSJT-X can be enabled per-rig with UDP host/port.

### Encrypted Secrets

Since v0.8.7, sensitive credentials are stored encrypted:

- **Secrets file**: `~/.config/cqops/secrets.enc` (0600 permissions)
- **Encryption**: AES-256-GCM with machine-tied key
- **Protected data**: QRZ password, DXC login, Wavelog API keys
- **Auto-migration**: Plaintext secrets from older configs migrate on first run
- **Recovery**: If the secrets file is corrupted, the app starts normally and shows a warning — re-enter credentials via the UI

### Configuration Screens

Access configuration screens from the Logbook Editor menu or press **Esc** from various screens to access menus:

- **Station** — callsign, grid, CQ/ITU zone, IARU region, references
- **Rig** — model, antenna, power, backend, frequency/mode defaults
- **Wavelog** — URL, API key, station profile ID
- **QRZ** — username, password
- **DX Cluster** — host, port, login
- **Notifications** — enable/disable toasts for specific events
- **General** — timezone, distance units, map rendering, debug mode

---

## Integrations

### Wavelog

Cloud logging platform integration. Supports:
- **Upload**: Send local QSOs to Wavelog (batch of 50, with retry).
- **Download**: Fetch QSOs from Wavelog (incremental, idempotent).
- **Private lookup**: Check worked/confirmed status for a callsign before logging.

Credentials: URL, API key, and station profile ID — per logbook.

### QRZ.com

Callbook lookup service. Requires a QRZ XML subscription.

- **Ins** on the QSO form triggers lookup for the current callsign.
- Fills: name, QTH, grid, country, CQ zone, ITU zone, DXCC, continent, IOTA.
- Photo displayed in Partner view (F3).

### flrig

Rig control via flrig's XML-RPC interface (HTTP).

- Auto-fills frequency, mode, and power.
- Detects split operation (VFO A → Freq, VFO B → Freq RX).
- Status dot on the integration bar.

Configuration: flrig host and port (default `localhost:12345`).

### Hamlib Rigctld

Rig control via Hamlib's TCP daemon.

- Frequency, mode, VFO, split, and power queries.
- Graceful VFO name query fallback for radios that don't support it (e.g. Xiegu).
- Rotor control via Hamlib rotctld (azimuth, elevation, stop).

Configuration: hamlib radio host/port and optional rotor host/port.

### WSJT-X

Automatic digital mode QSO logging via UDP.

- Listens for WSJT-X UDP messages on configured host/port.
- Parses ADIF from `QsoLogged` messages and saves QSOs.
- TX status indicator (green dot when transmitting).
- Frequency and mode sync from WSJT-X to the QSO form.
- Auto-logged QSOs inherit contest ID when a contest is active.
- Operator is taken from WSJT-X; mismatch with active operator triggers a warning.

### DX Cluster

Live DX spot feed via telnet.

- Connect to any DX cluster node (default: `dxspots.com:7300`).
- **Spotted-by filter**: Press **S** to filter spots where you are the spotting station.
- **Band filter**: Cycle through band filters.
- **Mode filter**: Cycle through mode filters (CW, SSB, FT8, etc.).
- **Time filter**: Show spots from last 15/30/60 minutes or all.
- **Continent filter**: Filter by continent of the spotted station.
- **Spot dialog**: **Enter** on a spot to open a dialog with details and actions.
- **Tune to spot**: From the spot dialog, tune your rig to the spot frequency.
- **Log from spot**: From the spot dialog, pre-fill the QSO form with the spot's callsign and frequency.

### PSK Reporter

Propagation reporting and visualization.

- Spot table with callsign, frequency, SNR, grid, distance, and time.
- Band, time, and mode filters.
- **Map view**: ASCII-art world map showing propagation paths from your station.
- Per-callsign fetch timestamps with 5-minute cooldown across logbook cycles.

### Solar Data

Solar-terrestrial data from hamqsl.com.

- Solar flux index (SFI), sunspot number (SN).
- A-index and K-index.
- Conditions summary (band-by-band propagation forecast).
- Cached hourly for offline use.
- Displayed on the Solar tab (F6) and optionally as a panel on the QSO screen.

### REF Database

Reference database for SOTA, POTA, WWFF, and IOTA.

- Search references by prefix, name, or designator.
- Auto-fill SOTA/POTA/WWFF/IOTA fields in the QSO form.
- Reference information displayed in Partner view.

### Band Plan Browser

Browse amateur and broadcast band plans.

- **HAM bands**: 160m through 23cm with mode-specific sub-bands.
- **VHF/UHF**: 2m, 70cm, and higher bands.
- **CB / PMR446**: Citizen Band and PMR446 channels.
- **Broadcast**: AM, FM, and SW broadcast bands with preset stations (BBC, VOA, etc.).
- **Export**: Band plan can be exported as Markdown.
- **Tune**: Tune your rig to a selected frequency/preset.

---

## Key Bindings Reference

### Global

| Key          | Action                                  |
|--------------|-----------------------------------------|
| F1           | QSO form                                |
| F2           | Recent QSOs table                       |
| F3           | Partner view                            |
| F4           | DX Cluster                              |
| F5           | PSK Reporter                            |
| F6           | Solar data                              |
| F7           | Band plan browser                       |
| F8           | Logbook editor                          |
| ?            | Show all key bindings (help overlay)    |
| Ctrl+C / Esc | Quit (from main screen, Esc opens menu) |

### QSO Form (F1)

| Key       | Action                          |
|-----------|---------------------------------|
| Tab       | Next field (column-aware)       |
| Shift+Tab | Previous field                  |
| ↑ / ↓     | Move within column              |
| Enter     | Save QSO (from last field)      |
| Ctrl+S    | Save QSO (from any field)       |
| Esc       | Clear form / cancel             |
| Ins       | QRZ callbook lookup             |
| Space     | Toggle retain comment / operator|
| Ctrl+O    | Cycle active operator           |

### Recent QSOs (F2)

| Key       | Action                          |
|-----------|---------------------------------|
| ↑ / ↓     | Navigate rows                   |
| Home/End  | Jump to first/last row          |
| Enter     | Open inline editor for selected |
| Del       | Delete selected QSO             |

### Logbook Editor (F8)

| Key       | Action                          |
|-----------|---------------------------------|
| ↑ / ↓     | Navigate rows                   |
| Enter     | Edit selected QSO               |
| Del       | Delete selected QSO             |
| Ins       | Create new (contest/operator)   |
| Ctrl+F    | Filter QSOs                     |
| Esc       | Back to menu / close filter     |

### DX Cluster (F4)

| Key       | Action                          |
|-----------|---------------------------------|
| ↑ / ↓     | Navigate spots                  |
| Enter     | Open spot dialog                |
| S         | Toggle spotted-by filter        |
| B         | Cycle band filter               |
| M         | Cycle mode filter               |
| T         | Cycle time filter               |
| C         | Cycle continent filter          |

### Partner View (F3)

| Key       | Action                          |
|-----------|---------------------------------|
| ↑ / ↓     | Scroll stats / info             |
| Space     | Cycle display mode              |

---

## Troubleshooting

### The app doesn't start

- Check that your terminal is at least 80×24 characters.
- On Windows, run from Windows Terminal, not the legacy `cmd.exe` console.
- Try `cqops --offline` to rule out network issues during startup.
- Check logs in `~/.config/cqops/cqops.log`.

### Rig not connecting

- **flrig**: Verify flrig is running and the port matches (default `12345`).
- **Hamlib**: Verify rigctld is running and the TCP port is correct.
- Check the Rig status dot — amber means connecting, red means failed.
- Suppressed reconnect toasts are normal — CQOps silently retries in the background.

### WSJT-X not auto-logging

- Verify WSJT-X is configured to send UDP to the correct host/port.
- In WSJT-X: Settings → Reporting → UDP Server.
- Check the WSJT status dot — should be green when WSJT-X is running.
- WSJT-X must be version 2.6 or newer.

### Wavelog upload fails

- Verify URL, API key, and station profile ID in config.
- Check the WL status dot — green means reachable.
- Upload errors are shown as toasts; the QSO remains saved locally.
- Individual QSO failures don't block the rest of the batch.

### Config file issues

- Config is at `~/.config/cqops/config.yaml` (Linux/macOS) or `%APPDATA%\cqops\config.yaml` (Windows).
- Secrets are in `secrets.enc` in the same directory.
- If the config is corrupted, delete it and restart — the wizard will create a fresh one.
- The `last_fetched_id` field only appears after a successful Wavelog download.

### Performance issues

- CQOps is designed for low-end hardware. If it feels slow:
  - Disable map rendering in General settings.
  - Disable solar panel on QSO screen.
  - Close unused tabs (DXC, PSK).
  - Run with `--offline` if network is unreliable.

### Reporting Bugs

Please report issues on [GitHub Issues](https://github.com/szporwolik/cqops/issues) with:
- CQOps version (`cqops --version`)
- Operating system and terminal emulator
- Steps to reproduce
- Any relevant log output from `~/.config/cqops/cqops.log`
