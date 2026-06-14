# Changelog

## 0.4.0-rc.1 (2026-06-14)

This is the first release candidate for the 0.4.0 series. It includes major
internal refactoring for safety, testability, and maintainability — with no
breaking changes to the config schema or user-visible behavior.

### Config & data safety

- **Config path tests** are fully isolated via `t.Setenv`/`t.TempDir()` and
  cross-platform (Linux, macOS, Windows) using `runtime.GOOS`-aware assertions.
- **Store tests** cover schema initialization, CRUD, Wavelog status, station
  field normalization, and counting with 81.7% coverage.
- **Config load/save round-trip** tests verify API keys, rig presets, and
  Wavelog settings survive YAML serialization.
- **EnsureConfig** tests verify first-run creation, existing config loading,
  malformed config rejection, and nil-logbooks handling — all isolated.

### WSJT-X lifecycle fixes

- **Config changes now actually restart** the listener (host/port changes
  were silently ignored before).
- **Generation guard** prevents stale UDP goroutines from invoking callbacks
  after a listener restart.
- **Restart avoidance**: `MaybeRestartWSJTX()` only restarts when the
  effective config (enabled/host/port) actually changes, minimizing UDP
  goroutine leaks from the external library limitation.
- **Mutex protection** on listener lifecycle state (`active`, `generation`,
  callback snapshots).

### Editor upload safety

- **Upload eligibility tests** cover missing config, missing required fields,
  all-sent, skip-detail, and mismatch detection for batch uploads.
- **Normalization tests** verify station-field normalization with a temporary
  SQLite DB — selected IDs are normalized, others untouched.

### Codebase health

- **284 tests** (up from baseline), zero data races, zero vet warnings.
- **No stale artifacts**, no `TODO`/`FIXME` cruft, no accidental debug prints.
- **Large file splits**: `logbook_editor.go` (842→178 lines) and
  `update_handlers.go` (599→111 lines) split into focused files.
- **Shared helpers** extracted for textinput styling, form navigation
  (wrapNext/wrapPrev/blurTextinputs), and cross-platform path expectations.

### Known limitations (this release)

- **WSJT-X UDP goroutine leak**: the `k0swe/wsjtx-go` library does not expose
  a socket-close mechanism. Each restart leaks one UDP goroutine/socket.
  Mitigated by restart-only-on-config-change (Phase 2I) and generation guard
  (Phase 2H). Acceptable for a desktop ham radio app where restarts are rare.
- **Flrig and Wavelog** should be smoke-tested against real hardware/accounts
  before public release.
