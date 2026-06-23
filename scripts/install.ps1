$ErrorActionPreference = "Stop"

$VERSION = Get-Content "$PSScriptRoot\..\VERSION"
$APP_NAME = "CQOps"
$BUILD_DIR = "$PSScriptRoot\..\build"
$INSTALL_DIR = "$env:LOCALAPPDATA\Programs\cqops"
$BIN = "$INSTALL_DIR\cqops.exe"
$START_MENU = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs"

Write-Host "=== CQOps v$VERSION Installer (Windows) ==="
New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null

$exe = Get-ChildItem "$BUILD_DIR\cqops-windows-amd64.exe" -ErrorAction SilentlyContinue
if (-not $exe) {
    Write-Host "Building cqops..."
    & "$PSScriptRoot\build.ps1"
    $exe = Get-ChildItem "$BUILD_DIR\cqops-windows-amd64.exe"
}
Copy-Item $exe.FullName $BIN -Force
Write-Host "  Binary : $BIN"

$shortcut = "$START_MENU\CQOps.lnk"
$WScriptShell = New-Object -ComObject WScript.Shell
$Shortcut = $WScriptShell.CreateShortcut($shortcut)
$Shortcut.TargetPath = $BIN
$Shortcut.WorkingDirectory = "$env:USERPROFILE"
$Shortcut.Description = "CQOps - Amateur Radio Logging"
$Shortcut.Save()
Write-Host "  Menu   : Start Menu → CQOps"

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$INSTALL_DIR*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$INSTALL_DIR", "User")
    Write-Host "  PATH   : added (new terminals only)"
}

Write-Host "`nCQOps v$VERSION installed. Run 'cqops' or use the Start Menu."
Write-Host "Uninstall: .\scripts\uninstall.ps1"
