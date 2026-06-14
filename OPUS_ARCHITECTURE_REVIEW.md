# OPUS Architecture & Refactoring Review — CQOPS

Working document for a first-pass architecture, ecosystem, and refactoring review
before the next version bump. Status is updated as work proceeds.

- Project: CQOPS — fast, minimal ham radio logging TUI (Go, Bubble Tea v2)
- Current version: `0.2.0`
- Baseline: `go build`, `go vet`, `go test ./...` all green; 111 test functions.

---

## 1. Current architecture summary

CQOPS follows a clean, layered design with the Elm/Bubble Tea pattern at the UI edge:

```
cmd/cqops/main.go        → thin entry, calls cli.Execute()
internal/cli/*           → Cobra commands (root, qso/log, logbook, rig, config, reset, version)
internal/app/app.go      → App aggregate: config + DB + WSJT-X listener lifecycle
internal/config/*        → YAML config, logbooks, paths, timezone, defaults
internal/store/*         → SQLite (modernc, pure-Go): open, migrate, queries
internal/qso/*           → domain: QSO struct, ADIF, band, modetable, fill, validate
internal/qrz, wavelog,
         wsjtx, rig/*    → integrations (network/UDP/HTTP), fail-safe
internal/tui/*           → Bubble Tea v2 presentation layer (root Model + components)
internal/applog          → structured logging
internal/version         → embedded version string
```

Dependency direction is healthy: `tui` and `cli` depend on `app`/domain/integrations;
domain (`qso`, `store`) has no UI dependencies. No circular dependencies observed.

## 2. Package/module map

| Package | Role | Notes |
|---|---|---|
| `cmd/cqops` | entry | 12 lines, ideal |
| `internal/cli` | CLI commands | `qso.go` largest (311) — manual logging mode lives here |
| `internal/app` | orchestration aggregate | clean; owns DB + WSJT-X lifecycle |
| `internal/config` | config/logbooks/paths | YAML via `gopkg.in/yaml.v3` |
| `internal/store` | SQLite persistence | slice-of-strings migrations, idempotent |
| `internal/qso` | domain logic | ADIF (farmergreg/adif), band/mode tables, validation, fill |
| `internal/qrz` | QRZ XML callbook | function seam for tests |
| `internal/wavelog` | Wavelog HTTP API | tested via httptest |
| `internal/wsjtx` | WSJT-X UDP listener | callback-based |
| `internal/rig/{flrig,rigctld}` | rig control | flrig via interface seam |
| `internal/tui` | presentation | 40+ files, component-per-concern |

## 3. Current TUI component map

| Component | File | Bubbles used | Handcrafted |
|---|---|---|---|
| Root model / orchestration | model.go, update_handlers.go | help, textinput | screen routing |
| QSO form | qso_form_view/update.go | textinput | field cycling, RST autofill |
| Recent QSOs (read-only) | recentqsos.go | table | tiered column layout |
| Partner / map | partner_view.go, map.go, grid.go | — | mercator ASCII map (cached) |
| Confirm dialog | confirmdialog.go | — | reusable `DialogModel` (good) |
| Log viewer | log_viewer.go | viewport | — (clean) |
| Logbook editor | logbook_editor.go | table, textinput | 34-field edit form, dialogs |
| Main menu | main_menu.go | — | cursor + render |
| Logbook chooser | logbook_menu.go | — | cursor + render |
| Rig chooser | rig_menu.go | — | cursor + render (dup of logbook) |
| General menu | general_menu.go | — | single-item cursor |
| Callbook menu | callbook_menu.go | textinput | checkbox/button/focus |
| Integration menu | integration_menu.go | textinput | checkbox/button/visibility focus |
| Station/Rig forms | station_form.go, rig_form.go | textinput | focus cycling |
| Wizard | wizard.go | (reuses forms) | step nav, timezone list |
| Status/tab/help bars | statusbar.go, tabbar.go, helpbar.go | help | Lip Gloss layout (good) |
| Toasts | toast.go | — | overlay compositor |

## 4. Handcrafted areas found

1. **Menu cursor logic** (`main_menu`, `logbook_menu`, `rig_menu`, `general_menu`):
   each hand-rolls `if cursor>0 {cursor--}` / `if cursor<len-1 {cursor++}` plus manual
   item rendering. `logbook_menu` and `rig_menu` are near-identical (list/edit/create/delete).
2. **Form focus cycling**: `station_form` uses a verbose 12-case `switch` on `.Focused()`;
   `rig_form` uses the cleaner `(focus+1) % n` index pattern. Inconsistent.
3. **Checkbox / button widgets** (`callbook_menu`, `integration_menu`, `wizard`):
   `"[x]"`/`"[ ]"` and `"[ Test ]"` rendered as strings with manual cursor highlight.
4. **ASCII world map** (`map.go`): handcrafted mercator projection over static ASCII art.
   No suitable maintained Go library exists for this niche; correctly cached.
5. **Tiered table column layout** (`recentqsos.go`): bespoke width distribution on top of
   `bubbles/table`. Reasonable; `table` has no responsive-column feature.

## 5. Ecosystem / library replacement candidates

| Candidate | Could replace | Status | Verdict |
|---|---|---|---|
| `charm.land/bubbles/v2 list` | menu cursor logic | available (already imported pkg family) | **Defer** — menus are tiny (1–5 items) and embedded in mode-machines; `list` brings filtering/pagination/styling overhead and behavior change risk for little gain. A tiny shared helper is lower-risk. |
| `huh` (forms) | callbook/integration/station forms | **Not available under `charm.land/v2` namespace**; upstream `charm.land/huh` would be a new heavy dependency and would fight the existing screen state-machine | **Reject** — heavy, behavior change, controllability loss. |
| `glamour` (markdown) | — | n/a, no markdown rendering need | **Reject** |
| `reflow` | truncation/wrapping | Lip Gloss v2 already handles width/truncate; `truncate()` helper is trivial | **Reject** — no benefit. |
| SQLite migration lib (golang-migrate, goose) | `store/migrations.go` | current migrations are idempotent slice-of-strings, ~30 lines | **Reject** — adds dependency + files for no correctness gain. |
| ADIF lib | already uses `farmergreg/adif/v5` | mature | **Keep** — already adopted. |
| `ftl/hamradio/locator` | grid/distance/azimuth | already used in `grid.go` | **Keep** — already adopted. |

**Guiding rule applied:** do not add a dependency just because it exists. None of the
candidates clearly reduce complexity or improve correctness enough to justify adoption
now. The project is already idiomatic on Bubbles where it counts (table, textinput,
viewport, help, key).

## 6. Dead / redundant code candidates

Verified by usage scan (uses outside definition site):

- `styles.go` package-level aliases with **zero** uses: `TitleStyle`, `titleStyle`,
  `HeaderStyle`, `headerStyle`, `WarningStyle`, `BarStyle`, `ActiveTabStyle`,
  `InactiveTabStyle`, `DisabledTabStyle`. (Package-level `var` aliases are not flagged by
  the compiler/`unused` linter, so these silently linger.)
- `confirmdialog.go`: `DialogOptions(...)` helper — **zero** uses (only `DangerOption`
  and direct `Option{}` literals are used).

Not removed (intentionally kept): exported domain APIs, config/DB fields, `DangerOption`
(used by logbook editor), `clamp`/`fit`/`section`/`fillBody` layout helpers (all used).

## 7. Performance hotspots

- `View()` is cheap; the only expensive render (ASCII map) is cached via
  `partnerMapCache` + `partnerMapCacheSig`, invalidated on resize and input change. Good.
- `RecentQSOs.View()` rebuilds the `table` each frame by design; documented as
  microsecond-cost and acceptable on Pi-class hardware. Verified — no change needed.
- Tick interval 200ms; flrig polled every 15 ticks (~3s); health checks every 300 ticks.
  Network work is always off the UI thread via `tea.Cmd`. Good.
- No blocking calls or DB/network access found in `View()`. Good.

## 8. Bubble Tea v2 compatibility findings

- Uses `tea.View`, `tea.NewView`, `AltScreen`, `WindowTitle`, `tea.KeyPressMsg`,
  `tea.WindowSizeMsg` — all current v2 idioms. No v1 patterns (`tea.KeyMsg`, string view).
- Async ops correctly return `tea.Cmd`; `tea.Batch` used to combine; commands are
  threaded through handlers (not silently dropped) — verified in `Update` and handlers.
- Active dialog blocks all input except key presses routed to the dialog (correct modal).
- Resize invalidates the map cache; layout recomputed from measured zones (no magic nums).

## 9. Test coverage gaps

- Strong coverage on: layout, dialog, recent QSO table, QSO form, partner/map cache,
  QSO save/refresh (temp SQLite), Wavelog (httptest), QRZ (seam), WSJT-X ADIF, flrig (fake).
- Gaps (acceptable for now, noted): menu cursor components (`main_menu`, `rig_menu`,
  `logbook_menu`) have no direct unit tests; CLI `log add` path is untested.

## 10. Proposed refactoring plan (this pass)

Low-risk, high-confidence only (preserve all behavior):

1. **Remove verified dead code**: unused style aliases + `DialogOptions`. (done below)
2. Leave menu/form/library decisions as documented recommendations — they are
   medium-risk behavior changes not justified for a minimal-footprint app right now.

Deferred (documented, not done — would change behavior/footprint without clear payoff):
shared menu-list helper, `huh` adoption, `bubbles/list` migration, StationForm focus refactor.

## 11. Risk assessment

- Dead-code removal: **very low** — symbols proven unused; compiler + tests confirm.
- No public API, config schema, DB schema, or CLI surface touched.

## 12. Changes completed

- Removed 9 unused style aliases from `internal/tui/styles.go`:
  `TitleStyle`, `titleStyle`, `HeaderStyle`, `headerStyle`, `WarningStyle`, `BarStyle`,
  `ActiveTabStyle`, `InactiveTabStyle`, `DisabledTabStyle`.
- Removed unused `DialogOptions` helper from `internal/tui/confirmdialog.go`.

## 13. Intentionally skipped items

- `bubbles/list` migration for menus (tiny menus; behavior-change risk > value).
- `huh` adoption (heavy/unavailable in v2 namespace; controllability loss).
- SQLite migration library (current approach is correct and minimal).
- Moving `grid.go`/`map.go` math out of `tui` (presentation-coupled; low payoff).
- StationForm focus refactor (works; cosmetic).

## 14. Final verification results

- `gofmt -l internal cmd`: clean (no files need formatting)
- `go vet ./...`: clean
- `go test ./...`: pass — 111 test functions, `internal/tui` ~1.37s
- release build: `go build -ldflags "-s -w" -o build\cqops.exe ./cmd/cqops/` → OK (~14.6 MB)

## 15. Version-bump readiness

- Version string embedded from `VERSION` (`0.2.0`) via `-X .../version.Version`; status bar
  reads it. No code change needed to bump — edit `VERSION` and tag.
- Config schema: **unchanged**. DB schema: **unchanged**. CLI surface: **unchanged**.
- Binary builds clean with release ldflags on Windows.
- **Recommendation:** project is ready for a version-bump review. This pass is a safe
  cleanup (dead-code only); no behavior changed.

### Recommended release notes (draft)
- Internal cleanup: removed dead style aliases and an unused dialog helper.
- Added `OPUS_ARCHITECTURE_REVIEW.md` documenting architecture, ecosystem evaluation,
  and a prioritized (deferred) refactoring backlog for future passes.
- No user-facing behavior, config, database, or CLI changes.
