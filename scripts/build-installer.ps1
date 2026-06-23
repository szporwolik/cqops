$ErrorActionPreference = "Stop"
Push-Location (Split-Path -Parent $MyInvocation.MyCommand.Path)
Push-Location ..

$VERSION = Get-Content VERSION
$DIST_DIR = "dist"
New-Item -ItemType Directory -Force -Path $DIST_DIR | Out-Null

Write-Host "=== CQOps v$VERSION - Windows Installer Build ==="

$BUILD_DATE = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$LDFLAGS = "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$VERSION -X github.com/szporwolik/cqops/internal/version.BuildDate=$BUILD_DATE"
$env:CGO_ENABLED = "0"

Write-Host "[1/3] Building cqops.exe (windows/amd64)..."
go build -ldflags $LDFLAGS -o "build\cqops-windows-amd64.exe" ./cmd/cqops/
if ($LASTEXITCODE -ne 0) { throw "Build failed" }

Write-Host "[2/3] Locating NSIS compiler..."

$makensis = $null
$searchPaths = @(
    "C:\Program Files (x86)\NSIS\makensis.exe",
    "C:\Program Files\NSIS\makensis.exe",
    "$env:LOCALAPPDATA\Programs\NSIS\makensis.exe"
)
foreach ($p in $searchPaths) {
    if (Test-Path $p) { $makensis = $p; break }
}
if (-not $makensis) {
    $makensis = (Get-Command makensis -ErrorAction SilentlyContinue).Source
}
if (-not $makensis) {
    Write-Host "ERROR: makensis not found. Install NSIS: winget install NSIS.NSIS"
    Pop-Location
    Pop-Location
    exit 1
}
Write-Host "  makensis : $makensis"

# Generate .ico from .png if needed and ImageMagick is available
$iconPng = "assets\cqops.png"
$iconIco = "assets\cqops-icon.ico"
if ((Test-Path $iconPng) -and (-not (Test-Path $iconIco))) {
    $magick = $null
    # ImageMagick 7 uses "magick", ImageMagick 6 uses "convert"
    $magickPaths = @(
        "C:\Program Files\ImageMagick-7\magick.exe",
        "C:\Program Files\ImageMagick\magick.exe",
        "$env:LOCALAPPDATA\Programs\ImageMagick\magick.exe"
    )
    foreach ($mp in $magickPaths) {
        if (Test-Path $mp) { $magick = $mp; break }
    }
    if (-not $magick) {
        $magick = (Get-Command magick -ErrorAction SilentlyContinue).Source
    }
    if ($magick) {
        Write-Host "  icon     : generating $iconIco from $iconPng"
        & $magick "$iconPng" -resize 256x256 -define icon:auto-resize=256,128,64,48,32,16 "$iconIco" 2>&1 | Out-Null
        if ($LASTEXITCODE -ne 0 -or -not (Test-Path $iconIco)) {
            Write-Host "  icon     : conversion failed, installer will use default NSIS icon"
        }
    } else {
        Write-Host "  icon     : ImageMagick not found, skipping .ico (installer uses default icon)"
        Write-Host "           : To enable: winget install ImageMagick.ImageMagick"
    }
}

Write-Host "[3/3] Compiling installer..."
$rootPath = (Resolve-Path ".").Path
New-Item -ItemType Directory -Force -Path "$rootPath\dist" | Out-Null
$nsiArgs = @("/DVERSION=$VERSION", "/DROOT=$rootPath")
if (Test-Path "assets\cqops-icon.ico") {
    $icoPath = Join-Path $rootPath "assets\cqops-icon.ico"
    $nsiArgs += "/DICON_SRC=$icoPath"
}
& $makensis $nsiArgs "installer\cqops.nsi"
if ($LASTEXITCODE -ne 0) { throw "NSIS compilation failed" }

$installer = Get-ChildItem "$rootPath\dist\cqops-setup-$VERSION.exe"
$sizeMB = [math]::Round($installer.Length / 1048576, 1)
Write-Host "=== Done ==="
Write-Host "  Installer : $($installer.FullName)"
Write-Host "  Size      : $sizeMB MB"
Pop-Location
Pop-Location