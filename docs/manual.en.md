---
title: CQOps User Manual
description: A practical guide to setting up CQOps, logging contacts, and operating at home or in the field
---

# CQOps User Manual

CQOps is a keyboard-operated amateur radio logger for home, portable, club, and casual contest use. QSOs are saved on your computer first; internet services are optional. Start with manual logging and add radio control or online services when you need them.

Menu names and field labels below follow the English interface. Shortcuts depend on the active screen: use the bottom help bar or **?** to check them.

## Contents

1. [Installation](#installation)
2. [First setup](#setup)
3. [Your first QSO](#first-qso)
4. [Screens and status](#screens)
5. [Everyday logging](#logging)
6. [Station profiles](#profiles)
7. [Logbook and backups](#logbook)
8. [Radio and digital modes](#radio)
9. [Online services](#online)
10. [GPS and APRS](#position)
11. [Portable operation](#portable)
12. [Contests](#contests)
13. [CQOps Live](#dashboard)
14. [Keyboard reference](#keys)
15. [Troubleshooting and help](#help)

<a id="installation"></a>

## Installation

Get CQOps from the [release page](https://github.com/szporwolik/cqops/releases). Use a terminal window at least 75 × 24 characters; 80 × 43 or larger is more comfortable.

| System | Installation |
|---|---|
| Windows | Download `cqops-setup.exe`, or extract `cqops-windows-portable.zip` for use without installation. Windows Terminal is recommended. |
| Debian, Ubuntu, Linux Mint, Pop!_OS | Download the appropriate `.deb`: `amd64` for most Intel/AMD PCs, `arm64` for 64-bit ARM, or `armhf` for 32-bit Raspberry Pi OS. Open it with your package installer. |
| Fedora, RHEL, Rocky, AlmaLinux | Use the repository commands below. |
| Arch, Manjaro, CachyOS | Install the AUR package with `paru -S cqops-bin` or `yay -S cqops-bin`. |
| Other Linux systems | Download the Linux `.tar.gz` for your processor from the release page and extract it. |
| macOS | Download `cqops-darwin-arm64` for Apple Silicon or `cqops-darwin-amd64` for Intel. Use the commands below. |

For Debian-based systems, you can instead install from the repository:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

For Fedora-based systems:

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

On macOS, run these commands in the download folder, using the exact downloaded filename in place of `FILE`:

```bash
chmod +x FILE
sudo mv FILE /usr/local/bin/cqops
```

Start with `cqops`, or run the extracted portable program. Use `cqops --offline` to start in offline mode, `cqops --version` to display the version, and `cqops --help` for startup options. Export your logs before updating.

<a id="setup"></a>

## First setup

The first-run wizard asks for a logbook name, station callsign, Maidenhead locator, and continent. The **station callsign** is the callsign you transmit; an **operator profile** identifies the person operating that station.

Press **Ctrl+A** for optional station references and CQ/ITU zones. Your own SOTA/POTA/WWFF reference belongs in the station/logbook settings; references in the QSO form describe the station you contact. Set the IARU region later in **F9 → Logbooks**.

Create a rig profile with a name, antenna, and power. Choose **None** for manual frequency and mode entry, **flrig**, or **Hamlib**. Skip optional connections until basic logging works.

Use **Tab / Shift+Tab** to move, **Space** to change selectable options, and the **Save & Next** button to continue. **Esc** goes back; **F10** quits. Review the summary and save. CQOps detects the computer's timezone; QSO dates and times are UTC. Check the computer clock before operating.

<a id="first-qso"></a>

## Your first QSO

1. Press **F1**. Check the active logbook, station callsign, operator, rig, and contest.
2. Enter the other station's callsign. Press **Ins** for a lookup if configured. Priority means data trust: defaults are QRZ.com (100) > HamQTH (90) > Callook.info (80) > QRZ.RU (70) > local logbook (60) > Wavelog (10), with CTY.DAT always last; lower-priority providers only fill fields that are still empty.
3. Check UTC date/time, frequency in MHz, band, mode, and sent/received reports.
4. Add any useful name, QTH, locator, reference, or comment.
5. Press **Enter**. The contact appears in Recent QSOs.

If **DUPE!** requests confirmation, press **Enter** again to save anyway or **Esc** to cancel the confirmation. A warning is a reason to check the contact, not proof that it should be discarded.

<a id="screens"></a>

## Screens and status

| Key | Screen | Purpose |
|---|---|---|
| F1 | QSO | Enter contacts; view recent QSOs |
| F2 | Partner | Callbook details, map, statistics, photo |
| F3 | APRS | Nearby stations |
| F4 | DX Cluster | Spots and filters |
| F5 | PSK Reporter | Digital-mode reception reports |
| F6 | References | SOTA, POTA, WWFF, IOTA search |
| F7 | Band Plan | Frequencies and operating presets |
| F8 | Logbook | Edit, import, export, synchronize |
| F9 | Configuration | Station and service settings |
| F10 | Quit | Exit CQOps |

The top bar identifies the active station setup and shows local time (**L**) and UTC (**Z**). Connection labels normally use white for active, yellow for disabled/connecting/waiting, and red for an error. WSJT is highlighted during transmission. **WL!** warns of an unsupported legacy Wavelog key.

<a id="logging"></a>

## Everyday logging

Use **Tab / Shift+Tab** between fields and **PgUp / PgDn** to cycle band, mode, or submode. **Shift+Backspace** clears the current field; **Del** clears the form. Check **Freq RX** when logging split operation.

**Keep** preserves the comment after saving. **Retain** preserves the whole form: check the callsign, time, reports, and references before saving the next contact. Contest exchange fields appear only with an active contest. Use **SIG / SIG Info** for other special-interest group details when needed.

With both locators known, CQOps shows distance and bearing. Callbook locations may describe a home station rather than its current portable location; verify them. New-call, new-DXCC, and duplicate badges help you assess a contact.

**F6** searches references by name or designator and can fill the contact's reference. **F7** browses band plans and can tune a connected radio. Entries are operating aids, not permission to transmit: check your licence privileges and local band plan.

Three shared favorites store frequency, mode, and band:

| Slot | Recall | Save current values |
|---|---|---|
| 1 | Alt+Ins | Alt+Shift+Ins |
| 2 | Alt+Home | Alt+Shift+Home |
| 3 | Alt+PgUp | Alt+Shift+PgUp |

<a id="profiles"></a>

## Station profiles

Create logbooks, operators, rigs, and contests in their **F9** menus; **Ins** adds an entry. From the QSO screen:

| Shortcut | Switch |
|---|---|
| Ctrl+L | Logbook |
| Ctrl+O | Operator |
| Ctrl+R | Rig |
| Ctrl+C | Contest |

Logbooks keep separate station details and Wavelog/APRS settings. Operator profiles store the person at the controls; the callsign is recorded in ADIF `OPERATOR`. Rig profiles store equipment, power, radio control, rotor, and WSJT-X settings. Verify the status bar after every switch, especially during automated digital logging.

Other **F9** menus cover general display/units/timezone, callbook providers, integrations, and notification sounds.

<a id="logbook"></a>

## Logbook and backups

In **F8**, select a QSO and press **Enter** or **e** to edit. Save with **Enter** and confirm. **Delete** removes the selected contact. Back up before bulk changes; **Ctrl+P** is the destructive purge-all command, not a search shortcut.

| F8 shortcut | Action |
|---|---|
| Ctrl+I | Import ADIF; validate records and skip duplicates |
| Ctrl+E | Export all contacts or a contest-filtered selection |
| Ctrl+W | Upload unsent contacts to Wavelog |
| Alt+W | Download from Wavelog |

Review the import summary and export selection. Imported contacts may subsequently be uploaded to Wavelog. CQOps supports ADIF 3.1.7 and preserves contest IDs and exchanges. Keep separate backups for every logbook, preferably on another device. ADIF backs up contacts, not all program settings or credentials.

Configuration is in `~/.config/cqops/config.yaml` on Linux/macOS and `%APPDATA%\cqops\config.yaml` on Windows. Credentials are stored separately in `secrets.enc`; re-enter them when moving to another computer. Do not delete configuration files as a first troubleshooting step.

<a id="radio"></a>

## Radio and digital modes

### Rig control

Under **F9 → Rigs**, select flrig or Hamlib and match the connection settings. Start flrig or `rigctld` first. flrig normally uses `localhost:12345`. Available frequency, mode, split, and power readings depend on the radio. With **None**, enter them manually.

### WSJT-X

Use WSJT-X 2.6 or newer. Match **Settings → Reporting → UDP Server** in WSJT-X with the UDP settings of the active CQOps rig profile. Log a completed test QSO in WSJT-X and confirm that it appears in CQOps.

Received QSOs use the active logbook and contest; duplicates are skipped. Check the operator and WSJT indicator before a session. CQOps warns about an operator mismatch. Wavelog upload can follow when configured. Select the appropriate Mode/Submode; CQOps exports FT8 as FT8, and FT4/FT2 as MFSK with the corresponding submode.

### Rotor control

Hamlib `rotctld` control is experimental. Verify direction and physical limits before use. Keep a safe way to stop movement: wrong settings can damage the antenna, rotor, or feed line.

| Shortcut | Action |
|---|---|
| Alt+, / Alt+. | Azimuth −5° / +5° |
| Alt+' / Alt+; | Elevation −5° / +5° |
| Alt+\ | Point toward the calculated bearing |
| Alt+/ | Stop movement |

<a id="online"></a>

## Online services

### Callbooks

Configure providers and priority in **F9 → Callbook**, then press **Ins** on the QSO form. CQOps tries enabled providers in order. Base-call fallback can look up a callsign without portable prefixes/suffixes; verify the returned location.

| Provider | Access |
|---|---|
| QRZ.com | XML subscription and credentials |
| HamQTH | Free account |
| QRZ.RU | API login, separate from website credentials |
| Callook.info | US callsigns; no account |

**F2** shows partner information. Photo availability depends on the provider and terminal; experimental **Kitty Graphics** in General settings needs a compatible terminal such as Kitty, Ghostty, or WezTerm.

### Wavelog

Configure the URL, API v2 token (`wl2_…`), and station profile per logbook. Legacy v1 keys are not accepted. Selecting a Wavelog station can fill local station details: verify callsign, locator, and references before saving.

QSOs are saved locally first. Failed uploads can be retried with **F8 → Ctrl+W**; **Alt+W** downloads contacts. When you open a linked QSO for editing, CQOps can refresh it from Wavelog. Online edits and deletions also affect the remote copy. Read the confirmation, especially when offline; do not assume a local-only change has reached Wavelog.


Club stations: use the owner's `wl2_` API key together with the **Shared club station** option in the logbook form. Synced contacts then become read-only — edits and deletions must be made on the Wavelog side, and CQOps never sends PATCH or DELETE for them. New contacts are uploaded normally and attributed to the active operator (falling back to the station callsign when no operator is selected). The API key is stored encrypted and, with this option on, is never displayed again once saved — leave the key field empty to keep it, type a new one to replace it.
### DX Cluster and propagation

Configure DX Cluster under Integrations and open **F4**. **b / c / m / t** filter band, spotter continent, mode, and age. **Backspace** clears filters. **Enter** fills the QSO form, tunes a connected rig, and returns to F1; **Space** tunes without leaving the cluster.

On F1, **Ctrl+S** opens the spot dialog and **Ctrl+P** takes the callsign from the closest displayed spot. Verify it before sending. **F5** displays PSK Reporter reception reports, not a guarantee of present propagation. The Solar panel shows HamQSL conditions; cached values may be old. **F5 is off by default — enable PSK Reporter in Integrations.**

<a id="position"></a>

## GPS and APRS

### GPS

Configure a serial GPS receiver or GPSD under Integrations. Enable **Grid from GPS** in station/logbook settings to use its locator for contacts, bearings, APRS, and the dashboard. Red GPS means an error, yellow means no fix, and white means a fix. Check the displayed locator before operating. Choose 6, 8, or 10 characters; more characters do not guarantee better receiver accuracy.

### APRS

| Service | Connection |
|---|---|
| APRS-IS | Internet APRS server |
| KISS | Serial hardware TNC and radio |
| KISS Server | TCP TNC such as Dire Wolf; can run locally |

Select the service in **F9 → Integrations → APRS**. Set callsign/SSID, symbol, comment, range, and beacon interval in **F9 → Logbooks → [active logbook] → APRS**. Enable **APRS TX** and **Send beacons** only if you intend to transmit. Receive-only operation shows **APRS-RX**. Position beacons disclose your location; check the position and intended audience first.

Automatic beacon intervals are at least five minutes. **F3** shows recently heard stations: arrows select, **Enter** fills the QSO form, **d / t / s** change distance/age/type filters, **Backspace** clears them, and **b** sends a configured beacon immediately. GPS-derived beacons require **Grid from GPS** and a valid position.

<a id="portable"></a>

## Portable operation

Before leaving, select the portable logbook; check callsign, locator, activation reference, rig, antenna, and power. Test the full station setup and run CQOps online to refresh cached reference and prefix data. Verify **F6** finds the references you need. Export a backup.

Without internet, local logging still works. `cqops --offline` skips network features; do not rely on live lookups or synchronization. Test any local-network equipment in your chosen startup mode before leaving. Cached information may be out of date.

Afterwards, check QSO count and references, export ADIF, keep a backup, and upload unsent contacts to Wavelog if used. Check each award program's required submission format; convert the export if necessary.

<a id="contests"></a>

## Contests

CQOps supports casual contest logging, exchanges, serial numbers, and rate information. It is not a full contest scoring or submission system. Use a dedicated logger for advanced contest operation.

In **F9 → Contests**, press **Ins** and set name, date, ADIF contest ID, starting serial, and sent/received exchange templates.

| Marker | Value |
|---|---|
| `@rst` | Sent or received report |
| `@serial` | Serial number |
| `@cqz` / `@mycqz` | Contact's / your CQ zone |
| `@itu` / `@myitu` | Contact's / your ITU zone |
| `@grid` / `@mygrid` | Contact's / your locator |

On **F1**, **Ctrl+C** switches contests. Check the exchange and next serial before transmitting. The status bar shows count, next serial, and timing; wider windows show more rate statistics. Return to no-contest operation when finished.

For export, open **F8**, select the contest filter with **Ctrl+C**, then use **Ctrl+E** and verify the export selection. Output is ADIF, not Cabrillo. Follow the organiser's required format and submission rules.

<a id="dashboard"></a>

## CQOps Live

Enable **F9 → Integrations → HTTP Server** and save with the **[ Save & Back ]** button (**Enter**/**Space**). On the CQOps computer, open `http://localhost:8073`.

The default address `0.0.0.0` allows local-network access, subject to the firewall. On another device, use the CQOps computer's IP address with port `8073`. Set `127.0.0.1` to restrict access to the CQOps computer. Keep it on a trusted network; do not forward the port to the internet.

The dashboard updates automatically with the current contact, QSO maps, recent contacts, rates, operators, APRS, and available propagation/weather data. Internet-dependent layers may be unavailable offline. Set Header 1, Header 2, Logo URL, and Event Start to customize an event display; the start date filters the displayed statistics and QSO lists.

<a id="keys"></a>

## Keyboard reference

| Screen | Keys | Action |
|---|---|---|
| General | ? / Esc / F10 | Help / back / quit |
| QSO | Tab / Shift+Tab | Next / previous field |
| QSO | Enter / Ins | Save / lookup |
| QSO | Shift+Backspace / Del | Clear field / form |
| QSO | Ctrl+L / Ctrl+O / Ctrl+R / Ctrl+C | Switch logbook / operator / rig / contest |
| Logbook | ↑ / ↓, PgUp / PgDn, Home / End | Select, change page, first / last row |
| Logbook | Enter or e / Delete | Edit / delete selected QSO |
| Logbook | Ctrl+I / Ctrl+E | Import / export ADIF |
| Logbook | Ctrl+W / Alt+W | Upload / download Wavelog |
| Logbook | Ctrl+C / Backspace | Contest filter / clear search |

Shortcuts are screen-specific: **Ctrl+C is not the quit command**. On laptops, function keys may require **Fn**. If a terminal captures a shortcut, check its keyboard settings and the CQOps help bar.

<a id="help"></a>

## Troubleshooting and help

| Problem | Check first |
|---|---|
| Startup or incomplete screen | Terminal size; Windows Terminal on Windows; try `cqops --offline` |
| Radio disconnected | Active rig profile; flrig/rigctld running; model, serial port, speed, host/port; another program holding the serial port |
| WSJT-X QSO missing | Matching UDP settings, WSJT indicator, completed QSO actually logged in WSJT-X, correct active logbook |
| Wavelog error | URL, `wl2_` token, station profile, internet; local QSOs remain safe |
| No GPS position | Port/speed or GPSD address, clear sky view, valid fix, Grid from GPS enabled |
| APRS not beaconing | Correct logbook, APRS TX and Send beacons, callsign/SSID, TNC/radio or internet connection |
| Dashboard unreachable | Server enabled, correct computer IP and port, local firewall; localhost only refers to the device running the browser |

For settings issues, use **F9** before changing files. Re-enter credentials if CQOps reports a secrets problem or you moved to another computer.

If needed, enable **F9 → General → Debug**, reproduce the issue safely, and collect the relevant diagnostic log; turn debug off afterwards.

| System | Diagnostic logs |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

Report problems at [GitHub Issues](https://github.com/szporwolik/cqops/issues). Include CQOps version, operating system, terminal, steps, and the relevant log. Remove passwords, API tokens, and private information before sharing.
