$ErrorActionPreference = "Stop"

$VERSION = Get-Content VERSION
$BUILD_DATE = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$BUILD_DIR = "build"
New-Item -ItemType Directory -Force -Path $BUILD_DIR | Out-Null

$LDFLAGS = "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$VERSION -X github.com/szporwolik/cqops/internal/version.BuildDate=$BUILD_DATE"

$env:CGO_ENABLED = "0"

$targets = @(
    @{OS="windows"; Arch="amd64"; Ext=".exe"},
    @{OS="windows"; Arch="arm64"; Ext=".exe"},
    @{OS="linux";   Arch="amd64"; Ext=""},
    @{OS="linux";   Arch="arm64"; Ext=""},
    @{OS="darwin";  Arch="amd64"; Ext=""},
    @{OS="darwin";  Arch="arm64"; Ext=""}
)

foreach ($t in $targets) {
    $env:GOOS = $t.OS
    $env:GOARCH = $t.Arch
    $name = "cqops-$($t.OS)-$($t.Arch)$($t.Ext)"
    Write-Host "Building cqops $VERSION for $($t.OS)/$($t.Arch)..."
    go build -ldflags $LDFLAGS -o "$BUILD_DIR/$name" ./cmd/cqops
    if ($LASTEXITCODE -ne 0) { throw "Build failed for $($t.OS)/$($t.Arch)" }
}

Write-Host "Done. Binaries in $BUILD_DIR/"

# Install to GOPATH/bin (if --install flag passed)
if ($args -contains "--install") {
    Write-Host "Installing cqops $VERSION..."
    go install -ldflags $LDFLAGS ./cmd/cqops
    Write-Host "Installed to $(go env GOPATH)/bin/cqops.exe"
}
