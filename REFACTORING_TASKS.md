# CQOPS TUI Refactoring Tasks

## Architecture Observations

### Current State (Good)
- Project uses Bubble Tea v2, Bubbles v2, Lip Gloss v2 — modern Charm ecosystem
- `model.go` serves as the root Bubble Tea Model with Update/View
- Screen-based navigation via `screenKind` enum
- `Layout` struct in `layout.go` for measured layout (not hardcoded)
- `Palette` + `Styles` in `styles.go` for centralized theming
- `KeyMap` in `keybindings.go` for centralized key bindings
- `DialogModel` in `confirmdialog.go` — reusable modal component
- `RecentQSOs` in `recentqsos.go` — read-only bubbles/table component
- Sub-models (menus, choosers, editor) each implement tea.Model

### Issues Found

#### Dead/Unused Code
1. **`confirmQuit` field** in Model struct — always uses `confirm *DialogModel` instead
2. **`fieldNames` variable** — defined but never referenced
3. **`editorRow()` function** in logbook_editor.go — unused (table built with buildTable + column defs)
4. **`th` alias** for `P` — legacy, adds confusion

#### Duplicate/Redundant Code
5. **`Confirm` struct** in confirm.go duplicates `DialogModel` in confirmdialog.go — both handle yes/no prompts
6. **Legacy style aliases** in styles.go: `errorStyle`, `cursorStyle`, `formLabelStyle`, `SectionStyle`, etc. — these alias `S.*` members and should be either removed or made private consistently
7. **`BarStyle`, `TitleStyle`, `HeaderStyle`** etc — public aliases of `S.*` — used inconsistently
8. **`renderProfileLine()` and `renderProfileBar()`** — profileLine does raw string building, profileBar wraps it with alignment; could be merged
9. **Content height calculations duplicated** across menus: `h - 4` pattern repeated in main_menu, general_menu, log_viewer, etc. Should use layout.MeasureLayout or a helper.
10. **Key bindings in `ActiveBindings()`** duplicate some from `DefaultKeyMap()`
11. **`QSOFormBox`, `RecentQSOsBox`, `MapBox`** all use `NormalBorder()` with `TextDim` — could share a helper

#### Manual UI Code (Should Use Lip Gloss / Bubbles)
12. **`section()` in render.go** uses `strings.Repeat("─", rem)` — should use Lip Gloss
13. **`fit()` and `clamp()` in render.go** — redundant; `clamp` is unused (grep shows no callers except definition)
14. **`trunc()` in render.go** — redundant with `truncate()`; `trunc` is unused (only definition, no callers)
15. **`toAny()` in render.go** — unused
16. **Manual `strings.Join(parts, " · ")`** in renderProfileLine — could use Lip Gloss JoinHorizontal
17. **`osc8Link()`** uses raw ANSI escape `\x1b]8;;` — could be wrapped in a lipgloss style helper
18. **`tern()` function** — simple, but could be replaced with inline conditional

#### Architecture Improvements Needed
19. **Model.Update() too large** — handles all screen routing, key handling, sub-model dispatch, tick processing. Should extract screen routing into separate methods.
20. **Sub-models have inline `width`/`height` management** - every Update method manually sets `m.subModel.width = m.width`. Could use a common pattern.
21. **`logbook_menu.go` uses inline styles** that duplicate styles.go entries
22. **`general_menu.go` uses inline `cursorStyle`, `formLabelStyle`** — aliases but local to file
23. **`callbook_menu.go` uses inline styles** — should use `S.*` directly
24. **`integration_menu.go` uses inline styles** — should use `S.*` directly
25. **Hardcoded terminal minimum** 75x24 in View() — should be configurable or use actual min

#### Resize/Performance
26. **map rendering happens in View()** via `viewPartner()` — could be cached
27. **table rebuilt every View()** in `RecentQSOs.View()` — by design (read-only), but column calcs could be cached when width unchanged

#### Style Consistency
28. **`Confirm` in confirm.go** uses `S.ConfirmMsg`, `S.ConfirmBtn`, etc.
29. **`DialogModel` in confirmdialog.go** uses its own local `dialogBoxStyle`, `dialogTitleStyle`, etc. — duplicates `S.ConfirmBox`, `S.ConfirmTitle`, etc.
30. **`errorStyle`** used in View() but defined as alias in styles.go — inconsistent with `ErrorStyle` used elsewhere

---

## Refactoring Steps — RESULTS

### Phase 1: Clean Dead Code ✅ COMPLETED
- [x] 1. Remove `confirmQuit` from Model struct
- [x] 2. `fieldNames` — KEPT (used at line ~1829 for QSO form labels)
- [x] 3. Remove `editorRow()` unused function in logbook_editor.go
- [x] 4. Remove `th` alias in styles.go
- [x] 5. Remove `trunc()` unused function in render.go
- [x] 6. Remove `toAny()` unused function in render.go
- [x] 7. `clamp()` — KEPT (used in status bar rendering)
- [x] 8. Build verified ✅

### Phase 2: Consolidate Duplicate Styles & Aliases ✅ COMPLETED
- [x] 9. Audited all style alias usage across codebase
- [x] 10. Private aliases KEPT (widely used, ~50+ usages; they are the canonical short form)
- [x] 11. Public aliases KEPT (used by external code)
- [x] 12. Consolidated `dialogBoxStyle` etc. in confirmdialog.go → uses `S.ConfirmBox`, `S.ConfirmTitle`, etc.
- [x] 13. Added `S.ConfirmHint` style; updated `S.ConfirmBtn`, `S.ConfirmBtnDim`, `S.ConfirmDanger` padding to (0,2)
- [x] 14. Build verified ✅

### Phase 3: Unify Confirmation Dialogs ✅ COMPLETED
- [x] 15. Entire `confirm.go` file DELETED — all code was dead (never called)
- [x] 16. No callers existed — `NewConfirm`, `RenderConfirmOverlay`, `Confirm` struct all unused
- [x] 17. `DialogModel` is now the single canonical dialog component
- [x] 18. Build verified ✅

### Phase 4: Refactor Key Bindings ⏭️ SKIPPED
- [ ] 19. Key bindings work correctly — refactoring risk outweighs benefit
- [ ] 20. Existing pattern is consistent and functional

### Phase 5: Extract Sub-Components ⏭️ SKIPPED
- [ ] 22-24. Method extraction attempted but hit Unicode matching issues in replacement tool
- [ ] Methods remain in model.go with clear section comments — functionally identical

### Phase 6: Clean Model.go Update() ⏭️ SKIPPED
- [ ] 26-27. Update() is large but well-structured — full refactoring risk outweighs benefit
- [ ] Screen routing already follows a clean switch-based pattern

### Phase 7: Final Polish ✅ COMPLETED
- [x] 29. Ran `go fmt` — no changes needed
- [x] 30. Ran `go vet` — PASSED
- [x] 31. Ran `go test ./...` — ALL PASSED
- [x] 32. Ran build — SUCCESS
- [x] 33. Binary builds and links correctly
- [x] 34. This file updated

---

## Summary of Changes Made

### Files Modified:
- `internal/tui/model.go` — removed `confirmQuit` dead field
- `internal/tui/styles.go` — removed `th` alias; updated Confirm* styles (padding, background, added ConfirmHint)
- `internal/tui/render.go` — removed dead `trunc()` and `toAny()` functions
- `internal/tui/logbook_editor.go` — removed dead `editorRow()` function
- `internal/tui/confirmdialog.go` — replaced local `dialog*Style` vars with `S.Confirm*` styles

### Files Deleted:
- `internal/tui/confirm.go` — entirely dead code (old `Confirm` struct, `RenderConfirmOverlay`, `Sheet`)

### Code Intentionally Kept:
- `fieldNames` — used for QSO form field labels
- `clamp()` — used in status bar for callsign/logbook clamping
- `tern()`, `fit()`, `osc8Link()`, `fillBody()`, `section()` — actively used utility functions
- `editorColTiers`, `editorColWidths`, `editorColValue` — used by logbook editor table
- Private style aliases (`errorStyle`, `cursorStyle`, etc.) — widely used (~50+ usages), provide concise style access
- Public style aliases (`TitleStyle`, `ErrorStyle`, etc.) — used by external code

### What Was NOT Changed (Intentionally):
- No CLI behavior changes
- No config file changes
- No SQLite/database changes
- No ADIF parsing changes
- No Wavelog integration changes
- No WSJT-X integration changes
- No flrig/rig integration changes
- No keybinding changes
- No business logic changes
- No platform compatibility changes

### Bubble Tea / Bubbles / Lip Gloss Components Now Used:
- `bubbles/textinput` — QSO form fields, station form, rig form
- `bubbles/table` — recent QSOs table, logbook editor table
- `bubbles/viewport` — log viewer
- `bubbles/help` + `bubbles/key` — key bindings and help bar
- `lipgloss` — all layout, borders, colors, alignment, compositing
- `DialogModel` — unified modal confirmation dialog

### Build/Test Results:
- `go fmt` — no changes needed
- `go vet` — PASSED
- `go test ./...` — ALL PASSED (qso package tests cached)
- `go build` — SUCCESS


---

## Phase 2 — Component extraction and Update/View simplification

### Method Extraction ✅ COMPLETED
- [x] Extract `headerView` + `statusDotStyled` + `renderStatusBar` + `renderToastBar` + `windowTitle` → `statusbar.go`
- [x] Extract `tabView` + `renderTabBar` + `renderProfileLine` + `renderProfileBar` → `tabbar.go`
- [x] Extract `helpView` + `renderHelpBar` → `helpbar.go`
- [x] 181 lines removed from model.go into 3 focused component files
- [x] Build verified ✅

### Update() Split ✅ COMPLETED
- [x] `handleTick()` — tick processing, ADIF ingestion, health checks
- [x] `handleAsyncMessages()` — inetResultMsg, wlStatusMsg, wlUploadResultMsg, flrigResultMsg
- [x] `handleGlobalKeys()` — F1-F10 function keys, Delete, Lookup
- [x] `handleFormKey()` — QSO form key bindings (retain, save, cycle, etc.)
- [x] `handlePendingRequests()` — needRefresh, qrzNeed, wlNeed
- [x] `handleChooserUpdate()` through `handleLogViewUpdate()` — 9 screen-specific handlers
- [x] Update() reduced from ~220 lines to ~70 lines
- [x] Build verified ✅

### Layout Helpers ✅ COMPLETED
- [x] Added `FixedZoneHeight = 4` constant (replaces magic number everywhere)
- [x] Added `contentHeight()` helper
- [x] Added `safeWidth()`, `safeHeight()` helpers
- [x] Added `emptyState()`, `renderSectionTitle()`, `truncWithEllipsis()` wrappers
- [x] Replaced all 17+ instances of `h - 4` pattern across 7 files
- [x] Build verified ✅

### Second Dead Code Pass ✅ COMPLETED
- [x] Removed unused `key` import from model.go after extraction
- [x] Removed unused `version` import from model.go after extraction
- [x] No new dead code found

### Files Changed This Phase
| File | Change |
|------|--------|
| `model.go` | Removed ~230 lines of rendering + key handling code; simplified Update() |
| `statusbar.go` | NEW — status bar rendering (headerView, statusDotStyled, renderStatusBar, renderToastBar, windowTitle) |
| `tabbar.go` | NEW — tab bar rendering (tabView, renderProfileLine, renderProfileBar, renderTabBar) |
| `helpbar.go` | NEW — help bar rendering (helpView, renderHelpBar) |
| `update_handlers.go` | NEW — Update() sub-handlers and screen routing methods |
| `render.go` | Added layout helpers (FixedZoneHeight, contentHeight, safeWidth, safeHeight, etc.) |
| `main_menu.go` | Replaced `h - 4` with `contentHeight(h)` |
| `general_menu.go` | Replaced `h - 4` with `contentHeight(h)` |
| `callbook_menu.go` | Replaced `h - 4` with `contentHeight(h)` |
| `integration_menu.go` | Replaced `h - 4` with `contentHeight(h)` |
| `log_viewer.go` | Replaced `height - 4` with `contentHeight()` |
| `logbook_menu.go` | Replaced `h - 4` with `contentHeight(h)` |
| `logbook_editor.go` | Replaced `height - 4` with `contentHeight()` |
| `rig_menu.go` | Replaced `h - 4` with `contentHeight(h)` |

### Build/Test Results
| Check | Result |
|-------|--------|
| `go fmt` | render.go, update_handlers.go formatted |
| `go vet` | ✅ PASSED |
| `go test ./...` | ✅ ALL PASSED |
| `go build` | ✅ SUCCESS |

### Remaining Risks / Future TODOs
- **RecentQSOs table**: Uses `bubbles/table` correctly, but column width calculations are recalculated on every `View()`. Could cache when width is unchanged (minor perf improvement).
- **Map rendering**: ASCII map content is regenerated in `viewPartner()` during `View()`. Could be cached for responsiveness on RPi devices.
- **Screen routing switch**: Still in `Update()` but delegated to named handlers — still ~30 lines, acceptable.
- **Sub-model width/height**: Each handler manually sets `m.subModel.width = m.width`. Could use a `setWidgetSize()` helper pattern.
- **No tui package tests**: Adding component-level Bubble Tea tests would improve safety for future refactoring.

- WSJT-X integration callbacks (thread-safe with mutex)
- SQLite database operations
- Wavelog upload pipeline
- flrig polling and connection management
- QSO form focus handling and retain behavior
- ADIF parsing from WSJT-X
- Config file save/load
- Multi-platform builds (Linux/Windows/macOS)
