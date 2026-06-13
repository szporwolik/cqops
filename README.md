# CQOps

Cross-platform amateur radio logging tool written in Go.

Small, fast, offline-first, keyboard-friendly.

## Build

The version from the `VERSION` file is always embedded in the binary.

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops

# Build for current platform (output in build/)
make build

# Or cross-compile for all platforms
make build-all

# Run tests
make test

# Lint (requires golangci-lint)
make lint
```

Binaries are placed in the `build/` directory (git-ignored).

For smaller binaries, install UPX and run `upx --best build/cqops`.

### Manual build (without make)

```bash
# Linux / macOS
go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$(cat VERSION)" -o build/cqops ./cmd/cqops/
```

On Windows PowerShell:

```powershell
$ver = Get-Content VERSION
go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$ver" -o build/cqops.exe ./cmd/cqops/
```

Or use the platform scripts:

```powershell
.\scripts\build.ps1   # Windows
```
```bash
./scripts/build.sh      # Linux/macOS
```

## Usage

```bash
cqops                  # Start interactive TUI
cqops config show      # Show configuration
cqops log add --call SP9ABC --band 20m --freq 14.074 --mode FT8
cqops log list         # List recent QSOs
cqops logbook list     # List logbooks
cqops version          # Print version
cqops --help           # Show all commands
```

## License

Apache-2.0
