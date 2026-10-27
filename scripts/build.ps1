$ErrorActionPreference = "Stop"

$VERSION = Get-Content VERSION
$COMMIT = try { git rev-parse --short HEAD 2>$null } catch { "" }
$BUILD_DATE = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$BUILD_DIR = "build"
New-Item -ItemType Directory -Force -Path $BUILD_DIR | Out-Null

# Ensure Go is on PATH (VS Code task shells can start before Go was installed/registered).
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    foreach ($p in @("$env:ProgramFiles\Go\bin", "${env:ProgramFiles(x86)}\Go\bin", "$env:LOCALAPPDATA\Programs\Go\bin", "$env:USERPROFILE\go\bin", "$env:USERPROFILE\scoop\apps\go\current\bin")) {
        if (Test-Path (Join-Path $p 'go.exe')) { $env:Path = "$p;$env:Path"; break }
    }
}

# Patch winres.json with current version before Windows builds
$winres = Get-Content winres\winres.json -Raw | ConvertFrom-Json
$ver4 = "$VERSION.0"
$winres.RT_MANIFEST.'#1'.'0409'.identity.version = $ver4
$winres.RT_VERSION.'#1'.'0000'.fixed.file_version = $ver4
$winres.RT_VERSION.'#1'.'0000'.fixed.product_version = $ver4
$winres.RT_VERSION.'#1'.'0000'.info.'0409'.FileVersion = $VERSION
$winres.RT_VERSION.'#1'.'0000'.info.'0409'.ProductVersion = $VERSION
$winres | ConvertTo-Json -Depth 10 | Set-Content winres\winres.json -Encoding UTF8
# Regenerate .syso if go-winres is available. Run from the repo root: the tool
# reads winres/winres.json relative to the working directory and must write the
# objects next to the binary (--out), otherwise the Windows build keeps the
# previously committed resources.
if (Get-Command go-winres -ErrorAction SilentlyContinue) {
    go-winres make --arch amd64,386 --out cmd/cqops/rsrc
}

$LDFLAGS = "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$VERSION -X github.com/szporwolik/cqops/internal/version.Commit=$COMMIT -X github.com/szporwolik/cqops/internal/version.BuildDate=$BUILD_DATE"

$env:CGO_ENABLED = "0"

# The loop below sets GOOS/GOARCH for each target. Remember the caller's values
# and restore them afterwards, so no build target leaks into the --install step
# (which must produce the binary this machine runs).
$savedLoopGOOS = $env:GOOS
$savedLoopGOARCH = $env:GOARCH

$targets = @(
    @{OS="windows"; Arch="amd64"; Ext=".exe"},
    @{OS="windows"; Arch="arm64"; Ext=".exe"},
    @{OS="linux";   Arch="amd64"; Ext=""},
    @{OS="linux";   Arch="arm64"; Ext=""},
    @{OS="darwin";  Arch="amd64"; Ext=""},
    @{OS="darwin";  Arch="arm64"; Ext=""}
)

try {
    foreach ($t in $targets) {
        $env:GOOS = $t.OS
        $env:GOARCH = $t.Arch
        $name = "cqops-$($t.OS)-$($t.Arch)$($t.Ext)"
        Write-Host "Building cqops $VERSION for $($t.OS)/$($t.Arch)..."
        go build -ldflags $LDFLAGS -o "$BUILD_DIR/$name" ./cmd/cqops
        if ($LASTEXITCODE -ne 0) { throw "Build failed for $($t.OS)/$($t.Arch)" }
    }
} finally {
    if ($savedLoopGOOS) { $env:GOOS = $savedLoopGOOS } else { Remove-Item Env:GOOS -ErrorAction SilentlyContinue }
    if ($savedLoopGOARCH) { $env:GOARCH = $savedLoopGOARCH } else { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue }
}

Write-Host "Done. Binaries in $BUILD_DIR/"

# Install to GOPATH/bin (if --install flag passed)
if ($args -contains "--install") {
    Write-Host "Installing cqops $VERSION..."
    # Install for the machine running this script: the loop above ends on
    # darwin/arm64, and an inherited GOOS/GOARCH from the caller's shell would
    # cross-compile too - either way the operator would keep running the old
    # cqops.exe while the script reported success. Pin the host platform, check
    # the exit code, verify the file, and restore the environment afterwards.
    $hostOS = "$(go env GOHOSTOS)".Trim()
    $hostArch = "$(go env GOHOSTARCH)".Trim()
    $savedInstallGOOS = $env:GOOS
    $savedInstallGOARCH = $env:GOARCH
    try {
        $env:GOOS = $hostOS
        $env:GOARCH = $hostArch
        go install -ldflags $LDFLAGS ./cmd/cqops
        if ($LASTEXITCODE -ne 0) {
            throw "go install failed for $hostOS/$hostArch (exit code $LASTEXITCODE)"
        }
    } finally {
        if ($savedInstallGOOS) { $env:GOOS = $savedInstallGOOS } else { Remove-Item Env:GOOS -ErrorAction SilentlyContinue }
        if ($savedInstallGOARCH) { $env:GOARCH = $savedInstallGOARCH } else { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue }
    }
    # `go install` writes to GOBIN when it is set and only falls back to
    # $GOPATH/bin when it is not, so ask go for the destination instead of
    # assuming the default: with a custom GOBIN the install succeeds but the
    # check below failed, and a stale cqops.exe in $GOPATH/bin was reported as
    # the freshly installed binary.
    $goBin = "$(go env GOBIN)".Trim()
    if (-not $goBin) {
        # GOPATH may list several directories; `go install` uses the first one.
        $firstGoPath = @("$(go env GOPATH)".Trim() -split [IO.Path]::PathSeparator)[0]
        $goBin = Join-Path $firstGoPath "bin"
    }
    $exe = "cqops"
    if ($hostOS -eq "windows") { $exe = "cqops.exe" }
    $installed = Join-Path $goBin $exe
    if (-not (Test-Path $installed)) {
        throw "go install finished but $installed is missing - the old binary is still in use"
    }
    Write-Host "Installed to $installed"
}
