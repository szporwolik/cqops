# Copilot Instructions for CQOps
CQOps is a fast, minimal Go TUI ham radio logger built with Bubble Tea v2, Bubbles, and Lip Gloss.

The application targets normal desktops but must also stay usable on low-end machines, Raspberry Pi-class devices, small portable screens, and field/portable ham radio setups. Prefer simple, fast, maintainable code over clever abstractions.

## Module and Dependencies (read first)

- Go module path: `github.com/szporwolik/cqops`. Use this exact path for `-ldflags -X` and imports — do **not** invent variants (e.g. `sq8r`).
- The Charm v2 stack is imported from the **`charm.land`** namespace, NOT `github.com/charmbracelet`:
  - `charm.land/bubbletea/v2` (aliased `tea`)
  - `charm.land/bubbles/v2/...` (e.g. `charm.land/bubbles/v2/textinput`, `/table`, `/viewport`, `/help`, `/key`)
  - `charm.land/lipgloss/v2`
  - When adding a Charm component, match the existing `charm.land/.../v2` imports. Never `go get github.com/charmbracelet/bubbles` (that is the wrong/older module and will break the build).
- Other key deps already in `go.mod`: `farmergreg/adif` + `spec` (ADIF), `ftl/hamradio` (grid/locator/distance), `k0swe/wsjtx-go` (WSJT-X UDP), `spf13/cobra` (CLI), `modernc.org/sqlite` (pure-Go SQLite, no cgo), `NimbleMarkets/ntcharts` (charts), `gen2brain/beeep` (desktop notifications), `gopkg.in/yaml.v3` (config), `golang.org/x/text` (Unicode normalization for ADIF).
- Do not add new dependencies unless they clearly remove complexity or improve correctness. Prefer the Charm ecosystem and the standard library.

## Core Principles

- Keep the app fast, small, and reliable.
- Preserve existing behavior unless explicitly fixing a bug.
- Prefer simple Go code that is easy to read and test.
- Avoid unnecessary dependencies.
- Avoid large rewrites unless specifically requested.
- Do not introduce network calls, sleeps, or blocking operations in `View()`.
- Keep rendering cheap and mostly pure.
- Cache expensive rendering when needed, especially map/image/table-like output.
- Never touch real user config, real logs, or real databases in tests.

## Bubble Tea v2 Architecture
Use idiomatic Bubble Tea v2 patterns.

Prefer:

- `tea.Model`, `tea.Cmd`, `Update`, `View`, and `Init` structure.
- Small focused components with their own update/view logic where appropriate.
- `tea.Batch` when multiple commands must be returned.
- Clear separation between root orchestration and component logic.
- Keeping `model.go` as root orchestration only.

Avoid:

- monolithic `Update()` functions.
- deeply mixed business logic and rendering logic.
- silently dropping commands.
- doing expensive work inside `View()`.
- blocking calls in `Update()` or `View()`.
- old Bubble Tea v1 patterns.
- hand-crafted terminal hacks when Bubble Tea/Bubbles/Lip Gloss can do it properly.

Dialog/modal behavior must block underlying input while active.

Resize handling must be robust and must not trigger expensive recalculation repeatedly unless the relevant inputs changed.

## Bubbles and Lip Gloss Usage
Use the Charm ecosystem instead of hand-crafted UI code.

Prefer:

- `bubbles/textinput` for editable text fields.
- `bubbles/table` for tables such as recent QSOs.
- `bubbles/help` and `bubbles/key` for key bindings and help text.
- `bubbles/viewport` for scrollable text/log views.
- Lip Gloss for:layout
- borders
- padding
- alignment
- truncation
- width/height calculations
- colors
- active/inactive states

Avoid:

- manual ANSI escape sequences.
- manual border drawing.
- repeated string padding logic.
- duplicate color definitions.
- hardcoded terminal sizes.
- wrapping long table values where truncation is required.

All styles should come from the existing centralized style/theme system. Do not scatter colors or Lip Gloss styles throughout unrelated files.

## Performance Requirements
CQOps should feel instant on low-end hardware.

Important constraints:

- Keep startup fast.
- Keep `View()` cheap.
- Avoid repeated allocation-heavy rendering loops.
- Avoid unnecessary goroutines.
- Avoid repeated DB/config/network access during render.
- Avoid recalculating map/table/layout content if inputs did not change.
- Avoid complex generic abstractions for simple TUI code.
- Prefer explicit, direct code over overly abstract framework-like code.

The app should remain comfortable on:

- Raspberry Pi-class hardware.
- older laptops/netbooks.
- small portable monitors.
- terminal sessions over SSH/VNC/RDP.

## Project Structure Expectations
Keep code grouped by concern.

### Package boundaries (keep domain logic out of the UI)

- `internal/qso` — domain: `QSO` struct, ADIF encode/decode, band/frequency mapping, mode/submode tables, callsign/locator validation, station defaults. New domain rules belong here, not in `internal/tui`.
- `internal/store` — SQLite: open, migrate, queries (QSO, DXC, PSK, stats, Wavelog). All DB access goes through this package.
- `internal/config` — YAML config, logbooks, paths, timezone, defaults.
- `internal/app` — aggregate that wires config + DB + WSJT-X lifecycle. Owns startup/shutdown.
- `internal/applog` — structured logging (slog with file rotation).
- `internal/version` — version resolution (ldflags → VERSION file fallback).
- `internal/cli` — Cobra commands (including non-interactive `qso`/`log` mode).
- `internal/{qrz,wavelog,wsjtx,rig/flrig,rig/rigctld,dxc,psk,solar,ref}` — integrations (network/UDP/HTTP/file). Must fail safely and stay independent of the UI.
- `internal/tui` — presentation only. It orchestrates and renders; it should not own ADIF formatting, band math, or schema details.

Dependency direction is one-way: `tui`/`cli` → `app` → `{config, store, qso, integrations}`. Domain packages (`qso`, `store`) must not import `tui`. Do not create circular dependencies.

### `internal/tui` file organization (~94 files, 34 test files)

Key files by responsibility:

- `model.go` — root model, `Init`, `Update`, `View`, high-level orchestration.
- `update_handlers.go`, `update_keys.go`, `update_screens.go`, `update_cycle.go` — update routing.
- `qso_form_view.go`, `qso_form_update.go`, `qso_form_validation.go`, `form_nav.go` — QSO form.
- `qso_lifecycle.go` — save/refresh lifecycle.
- `partner_view.go` — partner screen and map cache.
- `recentqsos.go`, `logbook_editor.go`, `logbook_editor_*.go` — QSO table and editor.
- `statusbar.go`, `tabbar.go`, `helpbar.go`, `toast.go` — UI chrome.
- `render.go`, `render_cache.go`, `layout.go`, `styles.go` — rendering infrastructure.
- `wavelog_integration.go` — Wavelog status/upload/lookup.
- `wsjtx_integration.go` — WSJT-X status and ADIF logging.
- `flrig_integration.go`, `flrig_interface.go` — flrig polling/result handling.
- `callbook_integration.go` — QRZ/callbook lookup.
- `dxc_*.go` — DX Cluster table, filters, keys, tune, state.
- `psk_*.go`, `solar_*.go` — PSK Reporter and solar data.
- `ref_integration.go` — SOTA/POTA/WWFF/IOTA reference search.
- `bpl_*.go` — band plan / broadcast presets.
- `health_checks.go` — internet/time/version checks.
- `wizard.go` — first-run setup wizard.
- `*_menu.go` — sub-screen menus (logbook, rig, contest, general, integration, notifications, main).
- `*_dialog.go` — modal dialogs (confirm, spot).

Do not move code into random files. If adding a file, name it by responsibility.

## Tests Are Mandatory
This project has a significant regression suite. Keep it healthy.

Before considering work complete, run:

```
go fmt ./...
go vet ./...
go test ./...
```
For release-style verification, also run:

```
go build -ldflags "-s -w" -o build/cqops ./cmd/cqops/
```
On Windows:

```
go build -ldflags "-s -w" -o build\cqops.exe ./cmd/cqops/
```
When changing behavior, add or update tests.

Current test coverage includes:

- layout helpers
- dialogs (confirm, spot)
- recent QSO table / logbook editor table
- QSO form rendering/navigation/autofill/retain/validation
- partner view and map cache
- flrig integration via fake client
- WSJT-X ADIF parsing and auto-log
- QSO save/refresh/delete using temporary SQLite databases
- Wavelog upload/status/private lookup/download using `httptest.Server`
- QRZ lookup behavior via function seam
- PSK Reporter and solar data result handling
- ADIF import (validation, persistence, Wavelog status)
- Editor upload (single/batch/missing fields)
- DXC table, band/mode/time filters
- Form navigation (column-aware Tab/arrows)
- Grid/bearing/distance computation
- Screen routing
- Station form validation
- Wizard validation
- Contest menu and contest QSO fields
- Favorites management
- Configuration save
- Download recovery and result rendering

Test files (run `go test ./...` for the authoritative current count; the list below may drift):

`internal/tui/` (34 test files):
- Core: `render_test.go`, `layout_test.go`, `confirmdialog_test.go`, `screen_routing_test.go`
- QSO form: `qso_form_test.go`, `qso_form_validation_test.go`, `form_nav_test.go`
- QSO lifecycle: `qso_lifecycle_test.go`, `adif_import_test.go`, `adif_persistence_test.go`
- Partner/map: `partner_view_test.go`, `map_test.go`, `grid_test.go`
- Wavelog: `wavelog_integration_test.go`, `editor_upload_test.go`, `download_recovery_test.go`, `download_result_render_test.go`
- QRZ: `callbook_integration_test.go`
- WSJT-X: `wsjtx_integration_test.go`
- flrig: `flrig_integration_test.go`
- DXC: `dxc_table_test.go`, `dxc_band_filter_test.go`, `dxc_mode_filter_test.go`, `dxc_time_filter_test.go`
- PSK/Solar: `psk_reporter_test.go`, `psk_solar_result_test.go`
- Editor/logbook: `logbook_menu_test.go`, `logbook_editor_contest_test.go`
- Config/wizard: `config_save_test.go`, `wizard_validation_test.go`, `station_form_validation_test.go`
- Other: `favorites_test.go`, `contest_menu_test.go`

`internal/qso/`:
- `band_test.go` — band/frequency logic
- `modetable_test.go` — mode/submode tables
- `callsign_test.go`, `locator_test.go`, `validate_test.go`, `adif_test.go`, `import_validate_test.go`

Tests must not require:

- live internet
- real QRZ credentials
- real Wavelog server
- real flrig server
- real WSJT-X UDP socket
- real user config
- real user database
- absolute local file paths

Use:

- `t.TempDir()` for temporary files/databases.
- `t.Cleanup()` for cleanup and restoring seams.
- `httptest.Server` for HTTP tests.
- fake clients/interfaces for integrations.
- behavior-focused tests instead of brittle full-screen snapshots.

## SQLite and Data Safety
Never use or modify real user data in tests.

For database tests:

- use temporary SQLite files.
- initialize schema through production store initialization functions.
- close DBs with cleanup.
- keep Wavelog disabled unless specifically testing Wavelog behavior through mocks.
- never depend on user logbooks or local config.

## Integration Rules
Integrations must fail safely.

Wavelog:

- disabled path must be safe.
- upload errors must not break local save.
- local QSO save must not depend on Wavelog success.
- tests should use `httptest.Server`.

QRZ/callbook:

- disabled or missing credentials must be safe.
- lookup errors must not panic.
- do not overwrite existing useful data unless current behavior explicitly does so.
- tests should use the existing QRZ lookup seam, not live QRZ.

flrig:

- disabled path must be safe.
- no live flrig server in tests.
- use `FlrigClient` interface and fake clients.

WSJT-X:

- no live UDP in tests.
- ADIF parsing and logging should be tested with local data/temp DB.
- WSJT-X logging must refresh local RecentQSOs independently of Wavelog upload.

## Comments and Documentation
Write comments only where they help future maintenance.

Good comments explain:

- non-obvious Bubble Tea command flow.
- why a cache exists and what invalidates it.
- why a test seam exists.
- tricky integration behavior.
- ADIF/WSJT-X edge cases.
- disabled integration behavior.

Avoid comments that:

- restate obvious code.
- mention temporary refactoring phases.
- reference old architecture.
- preserve outdated TODOs.
- describe removed code.

## Dead Code and Cleanup
Before adding new abstractions, check whether old code should be removed.

Look for:

- unused functions
- unused fields
- duplicate helpers
- duplicate styles
- stale comments
- commented-out code
- temporary scripts
- old compatibility shims
- unreachable branches

Remove only clearly safe dead code.

Do not remove exported symbols, config fields, or compatibility behavior unless you understand the consequences.

## UI/UX Requirements
The UI should remain compact, readable, and useful for ham radio operation.

Important details:

- long table values should truncate, not wrap.
- empty values should render consistently.
- active focus must be visible.
- dialogs should be centered and compact.
- no full-screen black modal blocks.
- status indicators should be clear but not noisy.
- small terminals must not panic.
- portable/field operation usability matters.

## Code Style
Prefer:

- small focused functions.
- explicit names.
- direct control flow.
- clear error handling.
- table-driven tests where useful.
- narrow interfaces for test seams.

Avoid:

- large abstractions.
- unnecessary generics.
- hidden global state.
- complex dependency injection frameworks.
- adding dependencies for simple tasks.
- changing behavior without tests.
- weakening production code only to make tests easier.

## Versioning and Release Builds
The version is single-sourced from the `VERSION` file and embedded at build time.

- To bump the version: edit `VERSION` only. Do not hardcode version strings elsewhere.
- The binary resolves its version via `internal/version`: the `-X` ldflags value wins, otherwise it falls back to reading the `VERSION` file next to the executable.
- When embedding via ldflags, the variable path must be exactly:
  `github.com/szporwolik/cqops/internal/version.Version` — using any other module path (e.g. `sq8r`) silently fails to embed and leaves the version as `dev`.

Correct release build (reads `VERSION`):

```
# Unix
go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$(cat VERSION)" -o build/cqops ./cmd/cqops/
```
```powershell
# Windows PowerShell
go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$(Get-Content VERSION)" -o build\cqops.exe ./cmd/cqops/
```

Prefer the helper scripts (`scripts/build.sh`, `scripts/build.ps1`) or `make build`, which already wire the correct module path. Do not bump the version unless explicitly asked.

## Completion Checklist
Before finishing any substantial change:

1. Confirm behavior is preserved.
2. Add or update tests if behavior changed.
3. Run `gofmt`.
4. Run `go vet ./...`.
5. Run `go test ./...`.
6. Run a build.
7. Check for dead code or stale comments.
8. Summarize what changed and what was intentionally left unchanged.

The default goal is: working, fast, minimal, tested, idiomatic Bubble Tea v2 code.

## Compact missing-input presentation

When band or mode are not yet entered in the QSO form, render:

```
Band       —
Mode       —
```

Do not add verbose instructional text such as `awaiting frequency` or `set frequency`. The field labels already communicate what is missing.

Do not show `Band + mode`, `DXCC + band`, or `DXCC + mode` combination rows until the required QSO fields are available.

For the current state without band/mode, render:

```
Call       KI6NAZ · NEW             QSOs       0 · first contact
DXCC       291 · WORKED             DXCC log   summary unavailable
Grid       DM03 · NEW
Band       —
Mode       —
```

When band and mode become available:

```
Call       KI6NAZ · NEW             QSOs       0 · first contact
Band       20m · NEW                DXCC log   248 QSOs · 9 bands · 5 modes
Mode       FT8 · NEW                Last DXCC  2026-07-11 · 20m FT8
Band+Mode  20m FT8 · NEW            Bands      20m×74 · 40m×52 · +7
DXCC       291 · WORKED             Modes      FT8×96 · SSB×71 · +3
Grid       DM03 · NEW
```

Do not add explanatory text after an em dash merely to fill unused space.

## Local worked-DXCC index

Create a fast local worked-status index so CQOps does not need to query or scan Wavelog, ADIF files, or the complete QSO table every time the operator enters a callsign.

The local CQOps database must become the primary fast source for:

- worked DXCC,
- worked DXCC on band,
- worked DXCC on mode,
- worked DXCC on band + mode,
- worked grid,
- worked call,
- call history statistics,
- DXCC history statistics.

Remote systems remain synchronization sources, not mandatory real-time dependencies for every lookup.

The index must be updated from:

- Wavelog ADIF download or synchronization,
- manually entered QSOs,
- WSJT-X logged QSOs,
- ADIF imports,
- future supported digital-mode integrations,
- QSO edits,
- QSO deletions,
- logbook or station-profile changes.

### Architecture

Keep the normalized local QSO table as the source of truth.

Build one or more materialized summary/index tables for fast worked-status queries. Do not treat the summary table as the only copy of QSO data.

Suggested tables:

```
worked_dxcc
worked_call
worked_grid
```

A possible `worked_dxcc` schema:

```sql
CREATE TABLE worked_dxcc (
    station_profile_id   INTEGER NOT NULL,
    logbook_id           INTEGER NOT NULL,

    dxcc                  INTEGER NOT NULL,
    band                  TEXT NOT NULL DEFAULT '',
    mode                  TEXT NOT NULL DEFAULT '',
    submode               TEXT NOT NULL DEFAULT '',

    qso_count             INTEGER NOT NULL DEFAULT 0,
    first_qso_utc         TEXT,
    last_qso_utc          TEXT,

    confirmed_count       INTEGER NOT NULL DEFAULT 0,

    data_quality          INTEGER NOT NULL DEFAULT 0,
    is_complete           INTEGER NOT NULL DEFAULT 0,
    source_mask           INTEGER NOT NULL DEFAULT 0,

    updated_at            TEXT NOT NULL,

    PRIMARY KEY (
        station_profile_id,
        logbook_id,
        dxcc,
        band,
        mode,
        submode
    )
);
```

The exact schema may be adjusted to the existing CQOps database design, but it must support indexed lookups for:

- DXCC only,
- DXCC + band,
- DXCC + mode/submode,
- DXCC + band + mode/submode.

Add appropriate indexes only after checking existing schema and query plans.

### Aggregation levels

Store or efficiently derive these aggregation levels:

1. DXCC
2. DXCC + band
3. DXCC + canonical mode/submode
4. DXCC + band + canonical mode/submode

Do not require a scan of all QSOs to answer:

```
Is DXCC 291 worked?
Is DXCC 291 worked on 20m?
Is DXCC 291 worked using FT8?
Is DXCC 291 worked on 20m FT8?
```

Equivalent fast local indexes should exist for:

- normalized callsign,
- normalized four-character grid.

### Incremental updates

Update the index incrementally whenever a QSO is:

- created manually,
- accepted from WSJT-X,
- imported from ADIF,
- downloaded from a remote logbook,
- edited,
- deleted,
- moved between logbooks or station profiles.

For insertion:

1. normalize the QSO,
2. resolve available DXCC, band, mode and grid data,
3. update relevant aggregate rows,
4. update count, first QSO and last QSO,
5. retain source and data-quality metadata.

For deletion or edits:

- decrement or rebuild only the affected aggregate keys,
- recalculate first/last timestamps when the removed QSO was an edge record,
- never allow negative counts,
- remove aggregate rows whose count becomes zero.

Use transactions so the QSO table and worked indexes cannot become inconsistent.

### Bulk synchronization and ADIF imports

For large Wavelog downloads or ADIF imports:

- do not update every summary row with an expensive independent query,
- import normalized QSOs in a transaction,
- collect affected keys,
- update aggregates in batches,
- or rebuild the affected logbook index with grouped SQL.

Provide a rebuild operation:

```
RebuildWorkedIndex(logbookID, stationProfileID)
```

Use it after:

- initial database migration,
- complete Wavelog synchronization,
- large ADIF import,
- detected inconsistency,
- normalization-rule changes.

The rebuild must be deterministic and safe to run repeatedly.

### Partial and incomplete QSO data

Imported QSOs may not contain complete information.

Handle incomplete data conservatively.

#### DXCC resolution

Use this priority:

1. valid DXCC value stored in the QSO,
2. reliable imported provider DXCC value,
3. deterministic callsign-prefix resolution,
4. cached callbook result with known provenance,
5. unresolved.

Do not guess DXCC from country text alone when the value is ambiguous.

Store how DXCC was resolved, for example:

```
explicit
imported
prefix
callbook
unknown
```

A QSO with unresolved DXCC must remain in the normal QSO log but must not be inserted into a specific worked-DXCC row.

#### Band resolution

Use:

1. valid ADIF band,
2. derive band from valid frequency,
3. unresolved.

Do not invent a band when both values are absent or inconsistent.

#### Mode resolution

Use CQOps canonical mode/submode normalization.

Examples:

- `MFSK + FT8` becomes canonical `FT8`,
- `MFSK + FT4` becomes canonical `FT4`,
- preserve meaningful submode distinctions,
- avoid counting the same QSO simultaneously as unrelated modes.

If mode is unknown, the QSO may still contribute to:

- worked DXCC,
- worked DXCC + band,

but not to mode-specific aggregates.

#### Grid resolution

Normalize valid Maidenhead locators.

For worked-grid status, use the configured precision, defaulting to four characters:

```
DM03xu -> DM03
```

Invalid or missing grids must not contribute to worked-grid aggregates.

### Data coverage and confidence

The absence of an aggregate row does not always prove that an entity is new.

Track local log coverage so CQOps can distinguish:

- definitely worked,
- definitely new within a complete log,
- unknown because local data is incomplete.

Suggested coverage metadata:

```go
type LogCoverage struct {
    LogbookID             int64
    StationProfileID      int64
    LastFullSyncAt        *time.Time
    LastIncrementalSyncAt *time.Time
    FullSyncCompleted     bool
    ImportedQSOCount      int
    UnresolvedDXCCCount   int
    MissingBandCount      int
    MissingModeCount      int
    SourceErrors          []string
}
```

Worked-state rules:

- if a matching aggregate exists with `qso_count > 0`:
  return `WORKED`
- if no aggregate exists and local coverage is known complete:
  return `NEW`
- if no aggregate exists but coverage is partial, stale, failed or unknown:
  return `UNKNOWN`

Never report `NEW` only because:

- Wavelog synchronization failed,
- only part of an ADIF file was imported,
- the local database has not completed its first full synchronization,
- many relevant QSOs have unresolved DXCC values.

A positive `WORKED` result remains valid even with incomplete coverage.

### Local-first query strategy

Use this lookup order:

1. local worked index,
2. local normalized QSO data if verification is required,
3. remote source only when local coverage is incomplete or a refresh was explicitly requested.

The normal logging-screen lookup must not depend on live Wavelog response time.

Expected behaviour:

- local index result appears immediately,
- remote synchronization may later improve statistics,
- stale asynchronous results must not overwrite a newer callsign lookup.

### Source provenance

Every imported or synchronized QSO should preserve provenance:

- local manual,
- WSJT-X,
- ADIF import,
- remote logbook instance,
- other integration.

A duplicate QSO present locally and remotely must count once in merged statistics.

Do not identify duplicates solely by source.

Use the existing CQOps QSO identity/deduplication rules based on normalized:

- station profile,
- contacted callsign,
- UTC timestamp with appropriate tolerance,
- band,
- mode/submode,
- frequency where available.

### Consistency checking

Add an inexpensive consistency mechanism.

Examples:

- aggregate QSO counts cannot be negative,
- `WORKED` cannot coexist with confirmed-empty history,
- aggregate first timestamp must not be later than last timestamp,
- aggregate rows must reference an existing logbook and station profile.

Optionally store an index schema/version number.

When normalization or aggregation rules change, mark the index for rebuild.

### UI integration

The `Worked` panel should primarily use the local index.

Do not mention implementation details such as `worked_dxcc` in the UI.

The source title may remain:

```
Worked · Local + qso.cqops.com
```

But the operator-facing result should appear immediately from local data.

If remote synchronization is incomplete, show a subtle source warning only when it affects certainty.

Examples:

```
DXCC       291 · WORKED
```

or:

```
DXCC       291 · UNKNOWN
```

Do not show `NEW` when local coverage is incomplete.

## Required tests for worked-DXCC index

Add tests for:

- full ADIF import builds worked-DXCC index,
- Wavelog full synchronization rebuilds the index,
- incremental remote synchronization updates only affected keys,
- manually logged QSO updates the index,
- WSJT-X QSO updates the index,
- QSO edit moves counts between aggregates,
- QSO deletion removes or updates aggregates,
- first and last QSO recalculation,
- duplicate local and remote QSO counts once,
- explicit DXCC value,
- prefix-derived DXCC,
- unresolved DXCC,
- band derived from frequency,
- missing band and frequency,
- canonical FT8 and FT4 normalization,
- missing mode still updates non-mode aggregates,
- six-character grid reduced to configured precision,
- missing or invalid grid,
- complete coverage with no match returns `NEW`,
- incomplete coverage with no match returns `UNKNOWN`,
- incomplete coverage with a match returns `WORKED`,
- failed synchronization never creates false `NEW`,
- index rebuild is idempotent,
- transaction rollback leaves QSO and index consistent,
- no negative aggregate counts,
- `Band —` and `Mode —` render without verbose guidance,
- no empty combination rows before band/mode are available.

The result should provide instant local worked-DXCC checks while remaining honest when imported or synchronized log data is incomplete.
