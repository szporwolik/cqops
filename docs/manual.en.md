---
title: CQOps User Manual
description: Practical guide to installing, setting up, and operating CQOps amateur radio logger
---

# CQOps User Manual

CQOps is a fast, keyboard-driven amateur radio logger for home, portable, contest, and club operation. It saves every QSO locally, so normal logging continues without an internet connection. Callbooks, Wavelog, DX Cluster, APRS, PSK Reporter, rig control, and other integrations are optional.

This manual covers everything needed to get on the air, manage contacts, and solve common problems without going into software-development details.

## Contents

1. [Install CQOps](#install-cqops)
2. [Set Up Your Station](#set-up-your-station)
3. [Log Your First QSO](#log-your-first-qso)
4. [Understand the Main Screen](#understand-the-main-screen)
5. [Everyday QSO Logging](#everyday-qso-logging)
6. [Logbooks Operators Rigs and Contests](#logbooks-operators-rigs-and-contests)
7. [Manage Your Log](#manage-your-log)
8. [Integrations](#integrations)
9. [Portable Operation](#portable-operation)
10. [Contest Operation](#contest-operation)
11. [CQOps Live Dashboard](#cqops-live-dashboard)
12. [Keyboard Reference](#keyboard-reference)
13. [Troubleshooting](#troubleshooting)
14. [Get Help](#get-help)

## Install CQOps

Download the latest release from <https://github.com/szporwolik/cqops/releases>.

### Windows

| Package | Recommended use |
|---|---|
| [Windows installer](https://github.com/szporwolik/cqops/releases/latest/download/cqops-setup.exe) | Best choice for most operators; adds CQOps to the Start Menu and command path |
| [Portable ZIP](https://github.com/szporwolik/cqops/releases/latest/download/cqops-windows-portable.zip) | Run without installing |

Windows Terminal is recommended. After installation, open a terminal and enter `cqops`.

### Debian Ubuntu Linux Mint and Pop OS

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install cqops
```

You can instead download an `amd64`, `arm64`, or `armhf` Debian package from the release page and install it with `sudo dpkg -i cqops_*.deb`.

### Fedora RHEL Rocky Linux and AlmaLinux

```bash
curl -1sLf 'https://dl.cloudsmith.io/public/szporwolik/cqops/setup.rpm.sh' | sudo -E bash
sudo dnf install cqops
```

### Arch Linux Manjaro and CachyOS

Install `cqops-bin` from the AUR, for example with `paru -S cqops-bin` or `yay -S cqops-bin`.

### macOS

Download `cqops-darwin-arm64` for Apple Silicon or `cqops-darwin-amd64` for an Intel Mac, then run:

```bash
chmod +x cqops-darwin-*
sudo mv cqops-darwin-* /usr/local/bin/cqops
```

### Before you start

CQOps needs a terminal window of at least 75 by 24 characters. A window of 80 by 43 characters or larger gives a better view.

| Command | Purpose |
|---|---|
| `cqops` | Start normally |
| `cqops --offline` | Start without internet services |
| `cqops --version` | Show the installed version |
| `cqops --help` | Show startup help |

## Set Up Your Station

The setup wizard opens on first launch. Only basic station information is required; online services can be added later.

### Station and logbook

Enter a logbook name, station callsign, Maidenhead locator, and continent. Press **Ctrl+A** to show optional fields such as SOTA, POTA, or WWFF references and CQ or ITU zones.

If you use Wavelog, you can also enter its address, API v2 token beginning with `wl2_`, and station profile.

### Rig

Give the rig profile a clear name and enter the radio, antenna, and normal power. Choose:

- **None** for manual frequency and mode entry;
- **flrig** if the station uses flrig;
- **Hamlib** if the station uses `rigctld`.

WSJT-X and rotor control can be added later.

### Finish the wizard

Review the summary and save. CQOps uses the computer's timezone automatically, while QSO date and time are recorded in UTC.

| Key | Wizard action |
|---|---|
| Tab / Shift+Tab | Move between fields |
| Space | Change a checkbox or selectable option |
| Enter | Continue or save |
| Esc | Return to the previous page |
| F10 | Quit |

Change settings later with **F9**.

## Log Your First QSO

1. Press **F1** to open the QSO screen.
2. Type the other station's callsign.
3. Check UTC date and time, frequency, band, mode, and reports.
4. Add the name, QTH, locator, reference, or comment if useful.
5. Press **Enter** to save.

With working rig control, CQOps can fill frequency, band, mode, and split frequency automatically.

If **DUPE!** appears, press **Enter** again to save the contact anyway or **Esc** to return to the form. A saved contact immediately appears in Recent QSOs.

**Before a long session:** check the active callsign, logbook, rig, operator, and contest in the status bars. This prevents contacts from being saved in the wrong operating context.

## Understand the Main Screen

| Key | Screen | Purpose |
|---|---|---|
| F1 | QSO | Enter QSOs and view recent contacts |
| F2 | Partner | Callbook information, map, statistics, and photo |
| F3 | APRS | Nearby APRS stations |
| F4 | DX Cluster | Browse and filter DX spots |
| F5 | PSK Reporter | Check recent digital-mode propagation |
| F6 | References | Find SOTA, POTA, WWFF, and IOTA references |
| F7 | Band Plan | Browse operating frequencies and presets |
| F8 | Logbook | Review, edit, import, export, and synchronize QSOs |
| F9 | Configuration | Configure the station and integrations |

Press **?** for help relevant to the current screen and **F10** to quit.

### Status indicators

The top bar shows the active logbook, rig, station callsign, operator, local time, UTC time, and connection status.

| Appearance | Meaning |
|---|---|
| White or normal | Connected or active |
| Yellow | Disabled, connecting, or waiting for data |
| Red | Connection or configuration problem |
| Highlighted WSJT | WSJT-X is transmitting |

GPS is yellow while waiting for a fix and white after acquiring one. **WL!** means that Wavelog is configured with an unsupported legacy key; replace it with an API v2 token beginning with `wl2_`.

## Everyday QSO Logging

### Complete the form

Use **Tab** and **Shift+Tab** to move through fields. The form includes:

- UTC date and time;
- callsign, name, QTH, and locator;
- RST sent and received;
- frequency, receive frequency, band, mode, and submode;
- transmit power;
- SOTA, POTA, WWFF, and IOTA references;
- contest exchanges when a contest is active;
- a free-text comment.

Use **PgUp** and **PgDn** to cycle band, mode, or submode. **Shift+Backspace** clears the current field; **Del** clears the entire form.

### Look up a callsign

Enter the callsign and press **Ins**. Depending on your configuration, CQOps can retrieve name, QTH, locator, country, zones, DXCC entity, continent, photo, and worked or confirmed status.

Configure QRZ.com, HamQTH, QRZ.RU, and Callook.info under **F9 → Callbook**. QRZ.com XML access requires a suitable subscription; HamQTH requires a free account; Callook focuses on US callsigns.

If both locators are known, CQOps shows distance and bearing. It can also flag a duplicate, a new callsign, or a new DXCC entity.

### Keep information for another QSO

- **Keep** preserves the comment after saving.
- **Retain** preserves the complete form.

Retain is useful for nets and consecutive contacts with similar details. Always check the callsign and time before saving again.

### Frequency favorites

CQOps has three favorites for frequency, mode, and band.

| Favorite | Recall | Save current settings |
|---|---|---|
| 1 | Alt+Ins | Alt+Shift+Ins |
| 2 | Alt+Home | Alt+Shift+Home |
| 3 | Alt+PgUp | Alt+Shift+PgUp |

### References and band plans

Press **F6** to search SOTA, POTA, WWFF, and IOTA references. A selected result can fill the QSO form.

Press **F7** to browse amateur, VHF/UHF, CB, PMR446, broadcast, and common portable frequencies. Selecting an entry can tune a connected rig.

## Logbooks Operators Rigs and Contests

CQOps keeps these operating contexts separate:

| Item | Examples | Switch with |
|---|---|---|
| Logbook | Home, portable, club, expedition | Ctrl+L |
| Operator | Individual club or field-day operators | Ctrl+O |
| Rig | HF, VHF, or portable station | Ctrl+R |
| Contest | No contest or a configured event | Ctrl+C |

Create and edit profiles under **F9**. Verify the status bar after switching.

Use separate logbooks when the station identity or operating purpose differs. Each can have its own station details, award references, Wavelog station, APRS identity, and contest settings.

At a shared station, create one profile per operator. The active operator's callsign is stored in the ADIF `OPERATOR` field. Create a rig profile for each radio or operating position, including radio, antenna, power, and any control connections.

## Manage Your Log

Press **F8** to open the Logbook screen.

### Correct or delete a QSO

Select a contact with the arrow keys and press **Enter** or **e**. Correct the fields, then press **Enter** and confirm. Press **Delete** to remove the selected QSO.

Read the confirmation carefully: if a contact already exists in Wavelog and CQOps is online, an edit or deletion can also be applied to the Wavelog copy.

### Import and export ADIF

| Action | Shortcut |
|---|---|
| Import ADIF | Ctrl+I |
| Export ADIF | Ctrl+E |

CQOps supports ADIF 3.1.7. Import validates records and skips duplicates. Export can include the whole active logbook or only the selected contest. Keep an exported ADIF file as a backup.

### Synchronize with Wavelog

CQOps always saves locally first. A failed upload does not remove the local QSO.

| Action | Shortcut |
|---|---|
| Upload QSOs not yet in Wavelog | Ctrl+W |
| Download new QSOs from Wavelog | Alt+W |

Configure the Wavelog address, API v2 token, and station profile for each logbook. Before the first synchronization, check that the Wavelog station matches the active CQOps logbook.

## Integrations

All integrations are optional and are configured under **F9**.

### Rig control

Rig control can read frequency, mode, power, and, where supported, split operation. With flrig, confirm that flrig is running and that the address and port match. With Hamlib, start `rigctld` for the correct radio and serial port before CQOps.

### WSJT-X

CQOps can receive completed QSOs from WSJT-X and add them to the active logbook. Before operating, verify that:

- the UDP address and port match in WSJT-X and CQOps;
- the correct logbook, operator, and contest are active;
- the WSJT indicator shows a working connection.

Duplicate WSJT-X messages are skipped. A warning appears if the WSJT-X operator differs from the active CQOps operator.

### DX Cluster

Press **F4** to browse spots and filter them by band, mode, spotter continent, or age.

| Key | Action |
|---|---|
| Enter | Put the spot into the QSO form, tune, and return to F1 |
| Space | Tune and remain on the cluster screen |
| Backspace | Clear filters |

On the QSO form, **Ctrl+S** sends a spot and **Ctrl+P** uses the closest displayed spot. Verify a callsign before spotting it.

### PSK Reporter

Press **F5** to see recent digital-mode reports and propagation by band, mode, and time. Internet access is required.

### GPS

CQOps can use a serial GPS or GPSD. With **Grid from GPS** enabled, the current position supplies the station locator for logging, distance calculations, APRS beacons, and the dashboard.

| Indicator | Meaning |
|---|---|
| Red GPS | Disconnected or incorrectly configured |
| Yellow GPS | Connected but no fix |
| White GPS | Fix acquired |

Choose 6, 8, or 10 locator characters as appropriate. Wait for a fix and check the locator before starting a portable activation.

### APRS

| Service | Connection | Internet required |
|---|---|---|
| APRS-IS | APRS internet network | Yes |
| KISS | Serial hardware TNC | No |
| KISS Server | Local or network TNC such as Dire Wolf | No for a local server |

Configure the service under **F9 → Integrations → APRS**. Configure callsign and SSID, symbol, comment, beacon interval, and range in the active logbook.

If transmission is not enabled, CQOps receives only and shows **APRS-RX**. The minimum automatic beacon interval is five minutes. Press **F3** to view nearby stations; select one and press **Enter** to use it in the QSO form. Press **b** to send an immediate beacon when transmission is configured.

### Rotor control

Rotor control through Hamlib `rotctld` is experimental. **Check the antenna limits before enabling movement and keep the stop command ready. Incorrect settings can damage station equipment.**

| Shortcut | Action |
|---|---|
| Alt+, / Alt+. | Decrease / increase azimuth by 5 degrees |
| Alt+' / Alt+; | Decrease / increase elevation by 5 degrees |
| Alt+\ | Point toward the calculated QSO bearing |
| Alt+/ | Stop movement |

### Solar conditions

CQOps can display solar and band-condition information from HamQSL. Live data requires internet access; the latest downloaded data remains available offline.

## Portable Operation

### Before leaving

1. Create or select the portable logbook.
2. Set the portable callsign, locator, antenna, and power.
3. Enter the activation reference or confirm it can be found with **F6**.
4. Test rig control, GPS, and WSJT-X if needed.
5. Run CQOps online once to refresh reference, solar, and prefix data.
6. Export an ADIF backup of important logs.

### In the field

Start with `cqops --offline` when internet access is absent or unreliable. Local logging, cached reference data, and directly connected equipment continue to work. Internet callbooks, Wavelog, DX Cluster, APRS-IS, PSK Reporter, weather, and live solar updates will not. Serial or local-network KISS APRS can still work.

### After the activation

1. Check the QSO count.
2. Export the activation to ADIF and keep a backup.
3. Reconnect to the internet.
4. Upload unsent QSOs to Wavelog with **F8 → Ctrl+W**, if used.
5. Submit the ADIF to the relevant award or activation program.

## Contest Operation

CQOps provides lightweight contest support for casual entries and contest contacts made during ordinary or portable operation. Use a dedicated contest logger for serious multi-operator, multi-radio, or advanced contesting.

Open **F9 → Contests**, press **Ins**, and enter the event name, date, official ADIF contest ID, starting serial, and exchange formats.

| Marker | Value inserted |
|---|---|
| `@rst` | Signal report |
| `@serial` | Next serial number |
| `@cqz` / `@mycqz` | Other station's / your CQ zone |
| `@itu` / `@myitu` | Other station's / your ITU zone |
| `@grid` / `@mygrid` | Other station's / your locator |

Press **Ctrl+C** to select the active contest. Exchange fields appear, serials advance automatically, and CQOps shows QSO count and rate information where space permits.

To export, activate the contest, open **F8**, apply the contest filter if needed, press **Ctrl+E**, and export only that contest. CQOps exports ADIF. Some organisers require Cabrillo or another format, which may need conversion outside CQOps.

## CQOps Live Dashboard

CQOps Live displays station activity in a browser. It suits club displays, field days, public events, or another shack screen.

1. Open **F9 → Integrations → HTTP Server**.
2. Enable the server and save.
3. Open `http://localhost:8073` on the CQOps computer.

The default address also permits devices on the same local network to use the CQOps computer's IP address and port 8073. Select `127.0.0.1` if access should be limited to the CQOps computer.

The dashboard can show the station being worked, today's contacts and paths, recent QSOs, totals, rates, operators, longest contacts, APRS stations, and propagation information. You can set an event title, subtitle, logo URL, and start date.

Do not expose the dashboard directly to the public internet unless the network is properly secured.

## Keyboard Reference

### Global

| Key | Action |
|---|---|
| F1 to F9 | Open the screen shown in the top menu |
| F10 | Quit |
| ? | Help |
| Ctrl+L / Ctrl+O | Switch logbook / operator |
| Ctrl+R / Ctrl+C | Switch rig / contest |
| Esc | Previous screen |

### QSO form

| Key | Action |
|---|---|
| Tab / Shift+Tab | Next / previous field |
| Enter | Save QSO or confirm a duplicate |
| Ins | Look up the callsign |
| PgUp / PgDn | Cycle band, mode, or submode |
| Shift+Backspace / Del | Clear field / clear form |
| Ctrl+S / Ctrl+P | Send a spot / use nearest spot |

### Logbook

| Key | Action |
|---|---|
| Arrow keys | Select a QSO |
| Enter or e | Edit selected QSO |
| Delete | Delete selected QSO |
| Ctrl+I / Ctrl+E | Import / export ADIF |
| Ctrl+W / Alt+W | Upload to / download from Wavelog |
| Ctrl+C | Change contest filter |
| Backspace | Clear search |

The help bar and **?** overlay list additional screen-specific commands.

## Troubleshooting

### CQOps does not start or the display is incomplete

- Enlarge the terminal to at least 75 by 24 characters.
- On Windows, use Windows Terminal.
- Try `cqops --offline` to rule out an unavailable online service.
- Check that the newest release is installed.

### The rig does not connect

- Confirm that the correct rig profile is active.
- For flrig, start flrig and verify its address and port; the usual port is 12345.
- For Hamlib, verify `rigctld`, radio model, serial port, speed, host, and port.
- Check whether another program has exclusive access to the radio's serial port.

### WSJT-X contacts are not logged

- Check **WSJT-X → Settings → Reporting → UDP Server**.
- Match the address and port with the active CQOps rig profile.
- Use WSJT-X 2.6 or newer.
- Check the WSJT indicator and active logbook, operator, and contest.

### Wavelog synchronization fails

- Confirm the URL and internet connection.
- Use an API v2 token beginning with `wl2_`.
- Verify the station profile for the active logbook.
- Check the **WL** indicator and retry from **F8**.

Your QSOs remain stored locally after a synchronization error.

### GPS connects but no locator appears

- Wait with a clear view of the sky for the first fix.
- Check the serial port and speed, or the GPSD address.
- Enable **Grid from GPS**.
- Confirm that the displayed locator is plausible before operating.

### APRS receives but does not transmit

- Enable APRS transmission and beaconing for the active logbook.
- Check callsign and SSID, interval, and service type.
- For KISS, verify the TNC and radio path.
- For APRS-IS, verify internet access and the callsign.

### Log file locations

| System | Location |
|---|---|
| Linux | `~/.local/share/cqops/logs/` |
| macOS | `~/Library/Application Support/cqops/logs/` |
| Windows | `%APPDATA%\cqops\logs\` |

## Get Help

Press **?** first and review the relevant troubleshooting section. If the issue remains:

1. Note the version from `cqops --version`.
2. Note the operating system and terminal application.
3. Record the exact steps that reproduce the issue.
4. Include the relevant log. Enable **F9 → General → Debug** first if the issue can be safely reproduced.
5. Remove passwords, API tokens, and other private information from attachments and screenshots.

Report issues at <https://github.com/szporwolik/cqops/issues>.
