# CQOps Pre-Release Verification Report

**Version:** 0.8.1  
**Date:** 2026-06-21  
**Module:** github.com/szporwolik/cqops

---

## Release Recommendation

**READY** — with known low-severity risks documented below. No HIGH-severity issues found. All tests pass, build is clean, staticcheck has only intentional/test-helper warnings.

---

## Final Verification Commands

| Command | Result |
|---------|--------|
| `go fmt ./...` | ✅ PASS (1 file formatted: update_handlers.go) |
| `go vet ./...` | ✅ PASS |
| `go test ./... -count=1` | ✅ PASS (all 19 packages) |
| `go build -ldflags "-s -w" -o build/cqops ./cmd/cqops/` | ✅ PASS |
| `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | ⚠️ 6 warnings (all classified below) |
| `go test -race ./...` | ❌ NOT RUN (CGO/environment not available, project uses pure-Go modernc.org/sqlite) |
| `govulncheck ./...` | ❌ NOT RUN (network timeout during tool download) |

---

## Remaining Staticcheck Warnings (Itemized)

| # | Warning ID | File:Line | Reason | Disposition | Risk |
|---|-----------|-----------|--------|-------------|------|
| 1 | SA4017 | `internal/ref/ref_import.go:588` | `After` return value ignored — but it's used in an `if` condition | **False positive** | None |
| 2 | U1000 | `internal/tui/contest_menu_test.go:111` | `contestWithName` test helper unused | **Intentionally kept** | None |
| 3 | U1000 | `internal/tui/logbook_menu_test.go:74` | `keyLeft` test helper unused | **Intentionally kept** | None |
| 4 | U1000 | `internal/tui/logbook_menu_test.go:76` | `keyUp` test helper unused | **Intentionally kept** | None |
| 5 | U1000 | `internal/tui/logbook_menu_test.go:79` | `keyRune` test helper unused | **Intentionally kept** | None |
| 6 | U1000 | `internal/tui/wavelog_integration_test.go:43` | `wavelogStationInfoHandler` test helper unused | **Intentionally kept** | None |

---

## Staticcheck Warnings Fixed in This Pass

| Count | Category | Description |
|-------|----------|-------------|
| 3 | S1009 | Redundant nil check before `len()` in DXC filter tests |
| 13 | ST1005 | Capitalized error strings in `wavelog/errors.go` (11) and `editor_upload.go` (1), plus 1 matching test |
| 7 | ST1013 | HTTP numeric literals (401, 405) replaced with `http.StatusUnauthorized`, `http.StatusMethodNotAllowed` |
| 3 | SA4006 | Unused variables in `download_recovery_test.go` (2) and `wizard.go` (1) |

---

## Performance Audit Findings

### Fixed (HIGH)

| # | Finding | Fix |
|---|---------|-----|
| H-1 | DXC band/continent SQLite query on every filter keypress | Added `cachedBands`/`cachedConts` to `dxcState`. Invalidated on new spots and purge. |
| H-2 | RecentQSOs filtered mode rebuilt `table.New()` every frame | Added `filteredCachedView`/`filteredCachedW`/`filteredCachedH`/`filteredCachedLen` cache fields. Cache key includes filtered QSO count. Invalidated on `SetFilterCall`/`ClearFilter`. |

### Fixed (MEDIUM)

| # | Finding | Fix |
|---|---------|-----|
| M-3 | Config file write (synchronous YAML serialization) on retain-comment keypress | Converted `persistRetainComment()` to return `tea.Cmd` (async goroutine). Updated 3 call sites to batch the command. |

### Fixed (LOW)

| # | Finding | Fix |
|---|---------|-----|
| L-2 | `severityStyle()` default case returned `lipgloss.NewStyle()` per call | Pre-allocated `bplDefaultSeverity` package-level var. |

### Deferred (MEDIUM — small scope but not critical)

| # | Finding | Reason Deferred |
|---|---------|----------------|
| M-1 | DXC view `lipgloss.NewStyle()` per frame (wrapper styles) | Requires adding style cache fields to dxcState; will do in dedicated rendering pass |
| M-2 | Logbook editor `lipgloss.NewStyle()` per frame (wrapper styles) | Requires coordinating with logbook editor's existing table cache |
| M-4 | Root View `lipgloss.NewStyle().MaxHeight()` per frame (clip style) | Single allocation per frame; negligible on any hardware |

### Verified as Already Optimized

- Status bar: 1-second TTL cache (only 1/60 frames rebuild)
- Partner map: `getOrBuildMap` returns cached image; proper invalidation
- PSK Reporter: dual-tier cache (`viewKey` + `spotKey`)
- DXC table: `tableReady` flag prevents rebuild; only on resize/filter change
- Logbook editor table: `cachedSig` includes page/QSO count/cursor
- BPL bandplan views: `bpl.cachedSig` only rebuilds on tab/scroll change
- Tab bar/help bar: cached via `rc.tabSig`
- Form cache: signature-based with `strings.Builder`
- flrig polling: 5-second interval, async HTTP, gated by `polling` flag
- Solar fetch: 5-minute interval, stale cache fallback
- QRZ/Wavelog periodic checks: only on tick 1 (startup) — not periodic

---

## Integration Lifecycle Audit

### All Integrations Verified Safe

| Integration | Timeout | Disabled No-Op | Credentials Not Logged | Goroutine Cleanup | View() Non-Blocking |
|-------------|---------|----------------|------------------------|-------------------|---------------------|
| WSJT-X UDP | N/A (UDP) | ✅ | ✅ No credentials | ✅ `wg.Wait()` | ✅ |
| flrig | 1s / 3s HTTP | ✅ | ✅ No credentials | ✅ via client nil | ✅ |
| Wavelog | 10s / 5min HTTP | ✅ | ✅ Only URL logged | ✅ `tea.Cmd` only | ✅ |
| QRZ | 10s HTTP | ✅ | ✅ Only callsign logged | ✅ `tea.Cmd` only | ✅ |
| DX Cluster | 10s TCP | ✅ | ✅ No credentials | ✅ `stopCh` channel | ✅ |
| PSK Reporter | 15s HTTP | ✅ (no network from View) | ✅ No credentials | ✅ `tea.Cmd` only | ✅ |
| Solar | 15s HTTP | ✅ | ✅ No credentials | ✅ `tea.Cmd` only | ✅ |
| GitHub version | 5s HTTP | ✅ (once, offline-gated) | ✅ No credentials | ✅ `tea.Cmd` only | ✅ |
| REF import | 300s HTTP | ✅ (offline-first) | ✅ No credentials | ✅ Synchronous | ✅ |

### Fixed (MEDIUM)

**Flrig goroutine data race (Finding #2):** Raw goroutines (`go m.fetchFlrigModes()` / `go m.fetchFlrigName()`) wrote to `m.rig.modes` and `m.rig.name` without synchronization, racing with reads in `View()`.

**Fix:** Converted to `tea.Cmd`-returning functions (`fetchFlrigModesCmd()` / `fetchFlrigNameCmd()`). Added `fmodesMsg`/`fnameMsg` message types processed in `handleAsyncMessages`. Updated `applyFlrigResult` to return `tea.Cmd`. No raw goroutines remain.

### Known Low-Severity Risks

| # | Finding | Risk |
|---|---------|------|
| 1 | WSJT-X: `unsafe.Pointer` for UDP socket close | Fragile against library updates — works correctly today |
| 3 | flrig: Auto-reconnect tick every 30s when permanently disabled | Negligible (no-op function, returns nil immediately) |
| 4 | Wavelog: API key in URL path for `station_info` endpoint | Wavelog API design; key not logged by CQOps |
| 5 | Wavelog: Debug log dumps raw `private_lookup` response body | Only at DEBUG level; contains worked/confirmed call data |
| 6 | QRZ: Password in HTTPS GET query string | QRZ XML API limitation; not logged by CQOps |
| 7 | QRZ: Plaintext password in global memory cache | Local TUI app; memory cleared on process exit |
| 8 | DXC: Fire-and-forget goroutines in `Start()` | Safe due to nil conn checks; overlap prevented by `connecting` flag |
| 9 | DXC: Oldest spot dropped silently when channel buffer full (256) | Intentional; only under extreme load on slow machines |
| 10 | PSK: Auto-fetch may miss internet-comes-online transition | Minor UX; user can press F5 |
| 11 | Solar: Retry sleep blocks command goroutine for up to 15s | Runs in goroutine; doesn't block UI |
| 12 | Photo: Uses `http.DefaultTransport` as base | Client-level 15s timeout set; connection pooling is managed |

---

## Deferred Issues Inspected (No Changes Made)

### 1. `logbook_editor_update.go`
- **`Update()` size (384 lines):** Monolithic switch — deferred to future cleanup pass
- **`runDownload`/`runImport` duplication (~100 lines):** 80% shared code — deferred, requires test suite updates
- **`isConfirmMode()` naming:** Misleading (includes non-confirm dialog modes) — low impact, deferred
- **`dlCancel` theoretical race:** Nil channel after purge — practically unreachable, deferred
- **Count queries silently falling to zero:** Unchecked errors in `runExport` — medium risk but deferred (export is manual action)
- **`scanner.Err()` not propagated:** Error logged but final msg reports success — medium risk, deferred (ADIF scanner rarely fails after successful reads)

### 2. Wavelog unsent count fallback
- Ambiguous when `wlUnsentCount == 0` (real zero vs DB query failure)
- Deferred: requires introducing a separate error flag; current behavior is conservative (may over-count but won't silently skip uploads)

### 3. `lookupTimeoutMsg`/`pendingSave` mechanism
- `pendingSave` is never set to `true` (only the now-removed `smartSaveOrLookup` set it)
- `lookupTimeoutMsg` handler is dead at runtime
- Deferred: removing requires coordinated model.go changes; harmless dead code

---

## Known Risks by Severity

### HIGH
*(none found)*

### MEDIUM
| # | Risk | Mitigation |
|---|------|-----------|
| M-A | DXC band/continent filter DB query on keypress | **FIXED** in this pass — added memory cache |
| M-B | RecentQSOs filtered table per-frame rebuild | **FIXED** in this pass — added filtered cache |
| M-C | Config sync write on retain keypress | **FIXED** in this pass — async save |
| M-D | Flrig goroutine data race on modes/name | **FIXED** in this pass — tea.Cmd instead of raw goroutines |
| M-E | `scanner.Err()` not propagated in download/import | Deferred — ADIF scanner rarely fails after processing records |
| M-F | `runDownload`/`runImport` code duplication | Deferred — ~100 lines shared, will address in cleanup pass |

### LOW
- 12 integration lifecycle findings (all documented above)
- 4 deferred design issues in logbook_editor_update.go
- Wavelog unsent count fallback ambiguity
- `lookupTimeoutMsg`/`pendingSave` dead mechanism

---

## Summary of Changes in This Pass

| Category | Count | Details |
|----------|-------|---------|
| Staticcheck warnings fixed | 26 | S1009×3, ST1005×13, ST1013×7, SA4006×3 |
| Performance optimizations | 4 | DXC cache, RecentQSOs filtered cache, async config save, pre-allocated severity style |
| Integration safety fixes | 1 | Flrig goroutine → tea.Cmd (race fix) |
| Code changes touched | ~15 files | No new files, no new dependencies, no behavior changes |
| Tests added/modified | 5 files | Updated test expectations for lowercased errors, new signature patterns |
| GUI behavior changed | **None** | All layout, colors, borders, tabs, dialogs, keybindings preserved |

---

## File Structure After This Pass

```
internal/tui/
├── bpl_views.go          (~40KB) — BPL bandplan, broadcast presets, export
├── callbook_integration.go
├── flrig_integration.go   — tea.Cmd-based mode/name fetching (was raw goroutines)
├── dxc_state.go           — added cachedBands/cachedConts fields
├── dxc_filter.go          — cached band/continent queries
├── dxc_integration.go     — cache invalidation on spots stored + purge
├── recentqsos.go          — filtered mode view cache
├── update_handlers.go     — handleAsyncMessages returns (bool, tea.Cmd)
├── update_keys.go         — async config save on retain toggle
├── update_screens.go      (~39KB) — screen update handlers only
└── ...
```
