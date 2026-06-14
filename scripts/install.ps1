$ErrorActionPreference = "Stop"

$VERSION = Get-Content "$PSScriptRoot\..\VERSION"
$APP_NAME = "CQOPS"
$BUILD_DIR = "$PSScriptRoot\..\build"
$INSTALL_DIR = "$env:LOCALAPPDATA\Programs\cqops"
$BIN = "$INSTALL_DIR\cqops.exe"
$START_MENU = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs"

Write-Host "=== CQOPS v$VERSION Installer (Windows) ==="
New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null

$exe = Get-ChildItem "$BUILD_DIR\cqops-windows-amd64.exe" -ErrorAction SilentlyContinue
if (-not $exe) {
    Write-Host "Building cqops..."
    & "$PSScriptRoot\build.ps1"
    $exe = Get-ChildItem "$BUILD_DIR\cqops-windows-amd64.exe"
}
Copy-Item $exe.FullName $BIN -Force
Write-Host "  Binary : $BIN"

$shortcut = "$START_MENU\CQOPS.lnk"
$WScriptShell = New-Object -ComObject WScript.Shell
$Shortcut = $WScriptShell.CreateShortcut($shortcut)
$Shortcut.TargetPath = $BIN
$Shortcut.WorkingDirectory = "$env:USERPROFILE"
$Shortcut.Description = "CQOPS � Amateur Radio Logging"
$Shortcut.Save()
Write-Host "  Menu   : Start Menu � CQOPS"

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$INSTALL_DIR*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$INSTALL_DIR", "User")
    Write-Host "  PATH   : added (new terminals only)"
}

Write-Host "`nCQOPS v$VERSION installed. Run 'cqops' or use the Start Menu."
Write-Host "Uninstall: .\scripts\uninstall.ps1"
