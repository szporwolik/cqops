# Release Checklist

## v0.5.3

### Required automated validation

- [x] `go fmt ./...` — clean
- [x] `go vet ./...` — clean
- [x] `go test ./...` — all pass (310+ tests)
- [x] `go test -race ./...` — no races
- [x] `go build -ldflags "-s -w" -o build/cqops ./cmd/cqops/` — succeeds (~16M stripped)

### Binary smoke tests

- [ ] `./build/cqops --version` — displays `0.5.3`
- [ ] `./build/cqops --help` — displays CLI help
- [ ] Confirm first-run wizard creates config with `0600` permissions
- [ ] Confirm app starts and displays QSO form

### Manual field tests

- [ ] Log one manual SSB/FM QSO
- [ ] Log one FT8/FT4 QSO from ADIF or WSJT-X path
- [ ] Import small ADIF file via logbook editor
- [ ] Export or upload to Wavelog, if configured
- [ ] Test Wavelog download, verify result screen shows counts
- [ ] Test DXC band filter cycling (Home/End keys)
- [ ] Test DXC time filter cycling (PgUp/PgDown keys)
- [ ] Test DXC mode filter cycling (Insert/Delete keys)
- [ ] Verify QSO form shows validation hints for invalid call/grid/freq/band/mode
- [ ] Verify StationForm shows validation hints for invalid callsign/locator
- [ ] Quit and restart app, confirm data persists

### Integration smoke tests (offline/disconnected-safe)

- [ ] Start with Wavelog disabled — no crash on upload attempt
- [ ] Start with WSJT-X disabled — no crash
- [ ] Start with flrig disabled — no crash
- [ ] Start with DXC disabled — no crash
- [ ] Start with QRZ disabled — no crash

### Hardware/environment tests

- [ ] App runs on target weak hardware / potato PC
- [ ] App runs on Raspberry Pi-class device
- [ ] Terminal size behavior at 80x24
- [ ] Terminal size behavior at 128x32
- [ ] Terminal session over SSH

### Security

- [ ] No API keys or secrets appear in help text, logs, or screenshots
- [ ] Config file permissions: `0600`
- [ ] Wavelog: HTTPS enforced in config validation

### Release steps

- [ ] Ensure working tree is clean except intended release files
- [ ] Commit release prep: `git commit -m "Release v0.5.3"`
- [ ] Tag: `git tag -a v0.5.3 -m "CQOps v0.5.3"`
- [ ] Push branch and tag: `git push origin main --tags`
- [ ] Attach Linux/Windows/macOS binaries
- [ ] Publish release notes from CHANGELOG.md

### Platform builds

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$(cat VERSION)" -o build/cqops-linux-amd64 ./cmd/cqops/

# Linux arm64 (Raspberry Pi)
GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$(cat VERSION)" -o build/cqops-linux-arm64 ./cmd/cqops/

# macOS amd64
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$(cat VERSION)" -o build/cqops-darwin-amd64 ./cmd/cqops/

# macOS arm64
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$(cat VERSION)" -o build/cqops-darwin-arm64 ./cmd/cqops/
```
