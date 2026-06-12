# CQOps

Cross-platform amateur radio logging tool written in Go.

Small, fast, offline-first, keyboard-friendly.

## Build

```bash
git clone https://github.com/szporwolik/cqops.git
cd cqops
go build -o cqops ./cmd/cqops/
```

### Versioned build

The version is read from the `VERSION` file at runtime. For release builds, inject it via ldflags:

```bash
go build -ldflags "-X github.com/szporwolik/cqops/internal/cli.Version=$(cat VERSION)" -o cqops ./cmd/cqops/
```

On Windows PowerShell:

```powershell
$ver = Get-Content VERSION
go build -ldflags "-X github.com/szporwolik/cqops/internal/cli.Version=$ver" -o cqops.exe ./cmd/cqops/
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
