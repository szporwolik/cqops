$ErrorActionPreference = "Stop"

$INSTALL_DIR = "$env:LOCALAPPDATA\Programs\cqops"
$START_MENU = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs"

Write-Host "=== CQOPS Uninstaller (Windows) ==="

Remove-Item "$INSTALL_DIR\cqops.exe" -Force -ErrorAction SilentlyContinue
Remove-Item "$INSTALL_DIR" -Force -ErrorAction SilentlyContinue
Write-Host "  Removed binary and directory"

Remove-Item "$START_MENU\CQOPS.lnk" -Force -ErrorAction SilentlyContinue
Write-Host "  Removed Start Menu shortcut"

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$userPath = ($userPath -split ";" | Where-Object { $_ -ne $INSTALL_DIR }) -join ";"
[Environment]::SetEnvironmentVariable("Path", $userPath, "User")
Write-Host "  Removed from user PATH"

Write-Host "`nCQOPS uninstalled."
