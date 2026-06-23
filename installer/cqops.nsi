; =============================================================================
; CQOps NSIS Installer — Amateur Radio Logging TUI
; Build: makensis /DVERSION=0.8.6 installer\cqops.nsi
; =============================================================================

Unicode true
ManifestDPIAware true

!ifndef VERSION
  !define VERSION "dev"
!endif

!define PRODUCT_NAME "CQOps"
!define PRODUCT_PUBLISHER "Szymon Porwolik"
!define PRODUCT_WEB_SITE "https://github.com/szporwolik/cqops"

; Paths — override via /DROOT=... /DBIN_SRC=... /DICON_SRC=... on makensis CLI.
!ifndef ROOT
  !define ROOT "${__FILEDIR__}\.."
!endif
!ifndef BIN_SRC
  !define BIN_SRC "${ROOT}\build\cqops-windows-amd64.exe"
!endif
!ifndef ICON_SRC
  !define ICON_SRC "${ROOT}\assets\cqops-icon.ico"
!endif
!ifndef ICON_FILENAME
  !define ICON_FILENAME "cqops-icon.ico"
!endif

Name "${PRODUCT_NAME} ${VERSION}"
OutFile "${ROOT}\dist\cqops-setup-${VERSION}.exe"
InstallDir "$PROGRAMFILES64\${PRODUCT_NAME}"
InstallDirRegKey HKLM "Software\${PRODUCT_NAME}" "InstallDir"
RequestExecutionLevel admin
SetCompressor /SOLID lzma

; -----------------------------------------------------------------------------
; Modern UI 2
; -----------------------------------------------------------------------------
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"

!insertmacro GetParameters
!insertmacro GetOptions

; -----------------------------------------------------------------------------
; Pages
; -----------------------------------------------------------------------------
!define MUI_ABORTWARNING
!if /FileExists "${ICON_SRC}"
  !define MUI_ICON "${ICON_SRC}"
  !define MUI_UNICON "${ICON_SRC}"
!endif

!insertmacro MUI_PAGE_WELCOME
!if /FileExists "${ROOT}\LICENSE"
  !insertmacro MUI_PAGE_LICENSE "${ROOT}\LICENSE"
!endif
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

; -----------------------------------------------------------------------------
; Install Section
; -----------------------------------------------------------------------------
Section "Install"
  SetOutPath "$INSTDIR"

  File "${BIN_SRC}"

  ; Copy the icon so the Control Panel uninstall entry can reference it.
  ; The .exe itself has the icon embedded via go-winres, so Start Menu
  ; shortcuts use the exe as their icon source.
  !if /FileExists "${ICON_SRC}"
    File "${ICON_SRC}"
  !endif

  ; Create a launcher batch file that keeps the terminal open on error.
  FileOpen $0 "$INSTDIR\cqops.cmd" w
  FileWrite $0 "@echo off$\r$\n"
  FileWrite $0 "setlocal$\r$\n"
  FileWrite $0 'set "CQOPS_HOME=$INSTDIR"$\r$\n'
  FileWrite $0 '"$INSTDIR\cqops-windows-amd64.exe" %*$\r$\n'
  FileWrite $0 "if %errorlevel% neq 0 (echo. & echo CQOps exited with error code %errorlevel% & pause)$\r$\n"
  FileClose $0

  ; Create Start Menu shortcut.
  ; Targets the .exe directly — the icon is embedded via go-winres so
  ; Windows Terminal shows the CQOps icon in the tab.
  ; The .exe pauses on error (panic or startup failure) so the user can
  ; read the message before the window closes.
  CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\CQOps.lnk" \
    "$INSTDIR\cqops-windows-amd64.exe" \
    "" \
    "$INSTDIR\cqops-windows-amd64.exe" 0
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall CQOps.lnk" \
    "$INSTDIR\uninstall.exe"

  ; Optional: copy a README / changelog
  File /nonfatal "${ROOT}\README.md"
  File /nonfatal "${ROOT}\CHANGELOG.md"

  ; Register uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Registry — install info
  WriteRegStr HKLM "Software\${PRODUCT_NAME}" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "Software\${PRODUCT_NAME}" "Version" "${VERSION}"

  ; Registry — uninstall info (Control Panel)
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" \
    "DisplayName" "${PRODUCT_NAME} — Amateur Radio Logging TUI"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" \
    "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" \
    "DisplayIcon" "$INSTDIR\cqops-windows-amd64.exe,0"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" \
    "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" \
    "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" \
    "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" \
    "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" \
    "NoRepair" 1

  ; Estimate size
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" \
    "EstimatedSize" "$0"

  ; Add to PATH — per-machine, append if not present
  ; Save current PATH, append $INSTDIR, broadcast WM_SETTINGCHANGE
  ReadRegStr $0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
  ${If} $0 != ""
    StrCpy $1 "$0;$INSTDIR"
    ${If} $0 == $1
      ; Already present — skip
    ${Else}
      WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path" "$1"
      SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=500
    ${EndIf}
  ${EndIf}
SectionEnd

; -----------------------------------------------------------------------------
; Uninstall Section
; -----------------------------------------------------------------------------
Section "Uninstall"
  ; Remove Start Menu shortcuts
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\CQOps.lnk"
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall CQOps.lnk"
  RMDir  "$SMPROGRAMS\${PRODUCT_NAME}"

  ; Remove installed files
  Delete "$INSTDIR\cqops-windows-amd64.exe"
  Delete "$INSTDIR\cqops.cmd"
  Delete "$INSTDIR\uninstall.exe"
  Delete "$INSTDIR\cqops-icon.ico"
  Delete "$INSTDIR\README.md"
  Delete "$INSTDIR\CHANGELOG.md"
  RMDir  "$INSTDIR"

  ; Remove registry keys
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
  DeleteRegKey HKLM "Software\${PRODUCT_NAME}"
SectionEnd
