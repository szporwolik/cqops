# CQOPS Release Smoke Test Checklist

Use this checklist before tagging a release. All items should pass on both Windows and Linux.

## First Run
- [ ] Clean first start (no config.yaml) launches wizard
- [ ] Wizard Step 1: Enter station name — shows "Station name is required" if empty
- [ ] Wizard Step 2: Enter rig name — shows "Rig name is required" if empty
- [ ] Wizard: Space cycles Wavelog stations when Station ID is focused
- [ ] Wizard: Complete all steps, config is saved
- [ ] Restart: config loads, no wizard

## Logbook Operations
- [ ] Create a new logbook
- [ ] Switch between logbooks (F8)
- [ ] Delete a logbook (with confirmation dialog)
- [ ] Cancel delete works
- [ ] Active logbook name shown in status bar

## QSO Operations
- [ ] Add QSO with callsign, band, mode — saves to RecentQSOs
- [ ] Dupe check shows confirmation dialog (press Enter twice)
- [ ] Edit QSO (Enter on RecentQSOs row)
- [ ] Delete QSO (Del on RecentQSOs row, then confirm)
- [ ] Cancel delete works
- [ ] Clear form (F9 or Del on empty form)

## QSO Form Navigation
- [ ] Tab cycles horizontally across columns
- [ ] Shift+Tab cycles backward
- [ ] Down/Up arrows navigate vertically (column-aware)
- [ ] Enter saves QSO
- [ ] Callsign exit triggers QRZ/Wavelog lookup (if enabled)
- [ ] Band/Mode cycle with +/- keys
- [ ] RST auto-fill based on mode
- [ ] Retain comment toggle (Ctrl+T)

## Contest Mode
- [ ] Ctrl+C cycles through active contests
- [ ] Contest info line shown on QSO screen
- [ ] Contest exchange fields visible when contest is active
- [ ] Exchange auto-fill: STX derived from ExchSent, SRX from ExchRcvd

## Partner View (F2)
- [ ] Shows callbook data (QRZ/Wavelog private lookup)
- [ ] Shows logbook stats (QSO count, first/last, bands)
- [ ] Shows Wavelog info if available
- [ ] Map renders correctly (Azimuthal equidistant)
- [ ] Photo loads if available (not shown if QRZ disabled)
- [ ] F2 with empty callsign: shows warning toast

## Station & Rig Config
- [ ] Station form: edit Name, Callsign, Locator, SOTA/POTA/WWFF refs
- [ ] Station form: IARU Region cycles 1-2-3
- [ ] Station form: Continent cycles through list
- [ ] Rig form: edit Rig Name, flrig/WSJT-X toggles
- [ ] Save: shows "Settings saved" toast
- [ ] Invalid callsign blocks save with error toast
- [ ] Invalid locator blocks save with error toast

## Integrations (General)
- [ ] Settings → Integrations: enable/disable QRZ, Wavelog, DXC, PSK, flrig, WSJT-X, REF
- [ ] Disabled integration is no-op (no errors, no network calls)
- [ ] Bad QRZ credentials: shows error toast, does not crash
- [ ] Bad Wavelog credentials: shows error toast, does not crash

## QRZ (F3 lookup or auto on callsign)
- [ ] Enabled with good credentials: fills Name, QTH, Grid, Country
- [ ] Disabled: no lookup, no error
- [ ] Bad credentials: error toast, form fields unchanged
- [ ] "Not found" callsign: no error, fields empty

## Wavelog
- [ ] Upload: QSO appears in Wavelog after save
- [ ] Private lookup: Wavelog data shown in partner view
- [ ] Station ID cycling in wizard (Space key)
- [ ] Download: fetches QSOs from Wavelog
- [ ] Disabled: no API calls

## flrig
- [ ] Enabled with running flrig: frequency/mode shown in status
- [ ] Disabled or flrig not running: shows offline dot, no errors
- [ ] Split mode detection: Freq Rx field populated
- [ ] Mode detection: CW, SSB, FT8, etc.

## WSJT-X
- [ ] Enabled with running WSJT-X: status dot green, auto-log works
- [ ] Disabled: no errors
- [ ] TX detection: "TX" shown in status bar

## DX Cluster (F4)
- [ ] Connect to cluster: spots appear
- [ ] Filters: Band (Home/End), Continent (\), Mode (Insert/Del), Time (PgUp/PgDn)
- [ ] Backspace clears all filters
- [ ] Enter on spot: fills QSO form, tunes rig
- [ ] Space on spot: tunes rig only
- [ ] Disabled: no connection attempt

## PSK Reporter (F5)
- [ ] Shows spots from PSK Reporter
- [ ] Band filter: Home/End cycles bands with spots
- [ ] Time filter: PgUp/PgDn cycles time windows
- [ ] Mode filter: Insert/Del cycles modes
- [ ] F5 refreshes data
- [ ] Map view if enabled

## Reference Database (F6)
- [ ] Search: enter ref/name, shows results
- [ ] Enter/Insert on result: adds to QSO form
- [ ] Up/Down/PgUp/PgDn navigate results
- [ ] Backspace clears search
- [ ] REF names shown on QSO screen when SOTA/POTA/WWFF/IOTA filled

## Band Plan (F7)
- [ ] Shows band plan for selected region
- [ ] Region cycling works
- [ ] BPL modes: HAM, VHF/UHF, CB, PMR446, Broadcast, CON
- [ ] Broadcast presets displayed
- [ ] Export to Markdown works

## Terminal & Resize
- [ ] Resize terminal: layout adjusts, no panic
- [ ] Very small terminal (below 75x24): shows "Terminal too small" message
- [ ] Normal terminal (80x24): all screens functional
- [ ] Large terminal (200x50): layout uses available space

## Error Recovery
- [ ] Kill flrig while CQOPS is running: shows offline status, reconnects
- [ ] Kill WSJT-X while CQOPS is running: shows offline status
- [ ] Network disconnect: shows offline indicator, recovers on reconnect
- [ ] Start CQOPS without config: wizard launches, can complete

## ADIF Import/Export
- [ ] Import ADIF file: records added to logbook
- [ ] Export ADIF: file created, readable by other tools
- [ ] Import with duplicates: dupes reported
- [ ] Import with errors: errors reported, valid records imported

## Release Build
- [ ] `go build -ldflags "-s -w" -o build/cqops ./cmd/cqops/` succeeds (Linux)
- [ ] `go build -ldflags "-s -w" -o build/cqops.exe ./cmd/cqops/` succeeds (Windows)
- [ ] `./build/cqops version` shows correct version
- [ ] `./build/cqops.exe version` shows correct version (Windows)
- [ ] `go test ./... -count=1` passes all packages
- [ ] `go vet ./...` passes
- [ ] `staticcheck ./...` has zero actionable findings
