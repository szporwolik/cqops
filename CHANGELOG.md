# Changelog

## v0.5.3 - 2026-06-18

### Summary
Pre-release after a 31-pass refactoring, validation, and test-hardening cycle. The TUI model was reduced from ~176 flat fields to ~53 grouped fields. Test coverage in the tui package increased from ~80 to 310+ tests. Three real UX bugs were found and fixed.

### Added
- Inline StationForm validation hints for callsign and locator (⚠ Invalid callsign / Invalid locator).
- Inline QSO form validation hints for callsign, grid, frequency, band, mode, and submode.
- Wavelog download/import/retry/result render test coverage (19 tests).
- DXC DB-backed band/time/mode filter cycle tests (63 tests).
- PSK Reporter and Solar result-handler tests.
- Wavelog ADIF import validation: `ValidateImportRecord` rejects bad core records and cleans non-fatal garbage.
- `dxc_test_helpers.go` — shared DXC test helper consolidation.

### Changed
- Reduced TUI model flat state by grouping runtime, UI, lookup, render, photo, rig, DXC, PSK, and solar state into 6 focused structs.
- Split DXC table, filter, key, and tune logic into focused files (`dxc_table.go`, `dxc_filter.go`, `dxc_keys.go`, `dxc_tune.go`, `dxc_state.go`).
- Split store queries by concern: `queries_qso.go`, `queries_wavelog.go`, `queries_stats.go`, `queries_psk.go`, `queries_dxc.go`.
- Moved flrig mode index lookup from standalone function onto `rigState` as `modeIndex()`.
- Improved Wavelog download error handling: mid-download `dlErr` is now captured immediately instead of silently dropped.
- Wavelog download result screen now trims whitespace-only error messages before displaying failure dialog.
- Station config wizard and normal save both validate via `config.Validate()` before committing.

### Fixed
- Wavelog download failures were logged internally but never shown in the editor result state (bug found in Pass 27).
- Whitespace-only Wavelog download error strings produced misleading blank failure dialogs (Pass 29/30).
- Invalid Wavelog-imported ADIF records could previously store bad core data (Pass 16/18).
- flrig XML-RPC string construction hardened with `encoding/xml` marshaling (Pass 1).
- FT8 and FT4 not in import-only normalization table — standalone ADIF modes were not normalized (Pass 6).
- Model partner data was incorrectly cleared on WSJT-X log (Pass 8).

### Security / Reliability
- Config files saved with `0600` permissions.
- Wavelog HTTPS enforced in config validation.
- SQLite: WAL mode, `_busy_timeout=5000`, and 3-retry insert loop for `SQLITE_BUSY`.
- No API keys or secrets rendered in Wavelog download result screens.
- `go test -race ./...` clean.

### Test Coverage
- TUI test count increased from ~80 to 310+.
- DXC filters: 63 DB-backed/key-driven tests (band: 18, time: 19, mode: 15, table/keys: 11).
- Wavelog download/recovery/render: 19 tests.
- Station/Config validation: 25 tests.
- ADIF persistence/import: 25 tests with temp SQLite databases.
- Logbook editor batch upload: 7 tests.
- All tests deterministic and offline — no real network, DB, or API key dependencies.

### Known Gaps
- DXC active connection lifecycle (connect/reconnect/disconnect/timeout) not yet tested.
- Logbook editor edit-form validation hints not yet implemented (Pass 20/21 pattern).
- flrig polling timeout/reconnect edge cases need more tests.
- Real hardware field testing (Raspberry Pi, portable setup) still required.
