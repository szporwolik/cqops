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
- Other key deps already vendored in `go.mod`: `farmergreg/adif` + `spec` (ADIF), `ftl/hamradio` (grid/locator/distance), `k0swe/wsjtx-go` (WSJT-X UDP), `spf13/cobra` (CLI), `modernc.org/sqlite` (pure-Go SQLite, no cgo).
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

- `internal/qso` — domain: `QSO` struct, ADIF encode, band/frequency mapping, mode/submode tables, validation, station defaults. New domain rules belong here, not in `internal/tui`.
- `internal/store` — SQLite: open, migrate, queries. All DB access goes through this package.
- `internal/config` — YAML config, logbooks, paths, timezone, defaults.
- `internal/app` — aggregate that wires config + DB + WSJT-X lifecycle. Owns startup/shutdown.
- `internal/{qrz,wavelog,wsjtx,rig}` — integrations (network/UDP/HTTP). Must fail safely and stay independent of the UI.
- `internal/cli` — Cobra commands (including non-interactive `qso`/`log` mode).
- `internal/tui` — presentation only. It orchestrates and renders; it should not own ADIF formatting, band math, or schema details.

Dependency direction is one-way: `tui`/`cli` → `app` → `{config, store, qso, integrations}`. Domain packages (`qso`, `store`) must not import `tui`. Do not create circular dependencies.

### `internal/tui` file organization

- `model.go` — root model, `Init`, `Update`, `View`, high-level orchestration.
- `update_handlers.go` — focused update routing/handlers.
- `qso_form_view.go` — QSO form rendering.
- `qso_form_update.go` — QSO form focus/update helpers.
- `qso_lifecycle.go` — save/refresh lifecycle.
- `partner_view.go` — partner screen and map cache.
- `statusbar.go`, `tabbar.go`, `helpbar.go` — top/bottom UI components.
- `wavelog_integration.go` — Wavelog status/upload/lookup orchestration.
- `wsjtx_integration.go` — WSJT-X status and ADIF logging.
- `flrig_integration.go` — flrig polling/result handling.
- `callbook_integration.go` — QRZ/callbook lookup.
- `health_checks.go` — internet/time/status checks.

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
- dialogs
- recent QSO table
- QSO form rendering/navigation/autofill/retain
- partner view and map cache
- flrig integration via fake client
- WSJT-X ADIF parsing
- QSO save/refresh using temporary SQLite databases
- Wavelog upload/status/private lookup using `httptest.Server`
- QRZ lookup behavior via function seam

Test files (run `go test ./...` for the authoritative current count; the list below may drift):

`internal/tui/`:
- `render_test.go` — layout helpers
- `confirmdialog_test.go` — unified dialog component
- `recentqsos_test.go` — recent QSO table
- `qso_form_test.go` — QSO form rendering/navigation/autofill/retain
- `partner_view_test.go` — partner view and map cache
- `qso_lifecycle_test.go` — QSO save/refresh with temp SQLite
- `wavelog_integration_test.go` — Wavelog upload/status/lookup via `httptest.Server`
- `callbook_integration_test.go` — QRZ lookup via function seam
- `wsjtx_integration_test.go` — WSJT-X ADIF parsing
- `flrig_integration_test.go` — flrig integration via fake client

`internal/qso/`:
- `band_test.go` — band/frequency logic
- `modetable_test.go` — mode/submode tables

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
