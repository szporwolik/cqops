$ErrorActionPreference = "Stop"

$VERSION = Get-Content VERSION
$BUILD_DIR = "build"
New-Item -ItemType Directory -Force -Path $BUILD_DIR | Out-Null

$LDFLAGS = "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$VERSION"

Write-Host "Building cqops $VERSION for windows/amd64..."
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags $LDFLAGS -o "$BUILD_DIR/cqops-windows-amd64.exe" ./cmd/cqops

Write-Host "Building cqops $VERSION for windows/arm64..."
$env:GOARCH = "arm64"
go build -ldflags $LDFLAGS -o "$BUILD_DIR/cqops-windows-arm64.exe" ./cmd/cqops

Write-Host "Done. Binaries in $BUILD_DIR/"
