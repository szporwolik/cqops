$VERSION = Get-Content VERSION
$COMMIT = try { git rev-parse --short HEAD 2>$null } catch { "" }
go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$VERSION -X github.com/szporwolik/cqops/internal/version.Commit=$COMMIT" -o build\cqops.exe ./cmd/cqops/
