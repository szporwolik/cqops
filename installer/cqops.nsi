; =============================================================================
; CQOps NSIS Installer — Fast, offline-first ham radio logger
; Build: makensis /DVERSION=X.Y.Z installer\cqops.nsi
; =============================================================================

Unicode true
ManifestDPIAware true

!ifndef VERSION
  !define VERSION "dev"
!endif

!define PRODUCT_NAME "CQOps"
!define PRODUCT_PUBLISHER "Szymon Porwolik"
!define PRODUCT_WEB_SITE "https://github.com/szporwolik/cqops"
!define REG_UNINST "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
!define REG_APP "Software\${PRODUCT_NAME}"

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

Name "${PRODUCT_NAME} ${VERSION}"
OutFile "${ROOT}\dist\cqops-setup.exe"
InstallDir "$PROGRAMFILES64\${PRODUCT_NAME}"
InstallDirRegKey HKLM "${REG_APP}" "InstallDir"
RequestExecutionLevel admin
SetCompressor /SOLID lzma

; -----------------------------------------------------------------------------
; Modern UI 2 + helpers
; -----------------------------------------------------------------------------
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"
!include "WordFunc.nsh"

!insertmacro GetParameters
!insertmacro GetOptions

; -----------------------------------------------------------------------------
; Macros — idempotent machine PATH add / remove
; -----------------------------------------------------------------------------
!define ENV_HKLM 'HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"'

!macro AddToPath Dir
  Push $0
  Push $1
  ReadRegStr $0 ${ENV_HKLM} "Path"
  ${If} $0 == ""
    StrCpy $0 "${Dir}"
  ${Else}
    ${WordFind} "$0" "${Dir}" "E+1;" $1
    ${If} $1 == 0
      StrCpy $0 "$0;${Dir}"
    ${EndIf}
  ${EndIf}
  WriteRegExpandStr ${ENV_HKLM} "Path" "$0"
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=500
  Pop $1
  Pop $0
!macroend

!macro RemoveFromPath Dir
  Push $0
  Push $1
  Push $2
  ReadRegStr $0 ${ENV_HKLM} "Path"
  ${If} $0 != ""
    ; Delete every occurrence of Dir as a semicolon-delimited word.
    ${WordReplace} "$0" "${Dir}" "" "E+};" $1
    ; Clean up leading semicolon.
    StrCpy $2 $1 1
    ${If} $2 == ";"
      StrCpy $1 $1 "" 1
    ${EndIf}
    ; Clean up trailing semicolon.
    StrCpy $2 $1 "" -1
    ${If} $2 == ";"
      StrCpy $1 $1 -1
    ${EndIf}
    ; Collapse double semicolons (from adjacent deleted entries).
    ${WordReplace} "$1" ";;" ";" "E+};" $1
    WriteRegExpandStr ${ENV_HKLM} "Path" "$1"
    SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=500
  ${EndIf}
  Pop $2
  Pop $1
  Pop $0
!macroend

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

  ; Install binary as cqops.exe.
  File "/oname=cqops.exe" "${BIN_SRC}"

  ; Copy the icon.
  !if /FileExists "${ICON_SRC}"
    File "${ICON_SRC}"
  !endif

  ; Copy README into docs/.
  CreateDirectory "$INSTDIR\docs"
  SetOutPath "$INSTDIR\docs"
  File /nonfatal "${ROOT}\README.md"
  SetOutPath "$INSTDIR"

  ; Start Menu shortcuts.
  CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\CQOps.lnk" \
    "$INSTDIR\cqops.exe" "" "$INSTDIR\cqops.exe" 0
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\README.lnk" \
    "$INSTDIR\docs\README.md"
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall CQOps.lnk" \
    "$INSTDIR\uninstall.exe"

  ; Uninstaller.
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Registry — app info.
  WriteRegStr HKLM "${REG_APP}" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "${REG_APP}" "Version" "${VERSION}"

  ; Registry — uninstall info (Control Panel).
  WriteRegStr HKLM "${REG_UNINST}" "DisplayName" \
    "${PRODUCT_NAME} - Fast offline-first ham radio logger"
  WriteRegStr HKLM "${REG_UNINST}" "UninstallString" \
    '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKLM "${REG_UNINST}" "QuietUninstallString" \
    '"$INSTDIR\uninstall.exe" /S'
  WriteRegStr HKLM "${REG_UNINST}" "DisplayIcon" \
    "$INSTDIR\cqops.exe,0"
  WriteRegStr HKLM "${REG_UNINST}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${REG_UNINST}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr HKLM "${REG_UNINST}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegDWORD HKLM "${REG_UNINST}" "NoModify" 1
  WriteRegDWORD HKLM "${REG_UNINST}" "NoRepair" 1

  ; Estimated size.
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKLM "${REG_UNINST}" "EstimatedSize" "$0"

  ; Add to machine PATH.
  !insertmacro AddToPath "$INSTDIR"
SectionEnd

; -----------------------------------------------------------------------------
; Uninstall Section
; -----------------------------------------------------------------------------
Section "Uninstall"
  ; Remove from machine PATH.
  !insertmacro RemoveFromPath "$INSTDIR"

  ; Remove Start Menu shortcuts.
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\CQOps.lnk"
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\README.lnk"
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall CQOps.lnk"
  RMDir  "$SMPROGRAMS\${PRODUCT_NAME}"

  ; Remove installed files — does NOT touch user config or logs.
  Delete "$INSTDIR\cqops.exe"
  Delete "$INSTDIR\cqops-icon.ico"
  Delete "$INSTDIR\docs\README.md"
  RMDir  "$INSTDIR\docs"
  Delete "$INSTDIR\uninstall.exe"
  RMDir  "$INSTDIR"

  ; Remove registry keys.
  DeleteRegKey HKLM "${REG_UNINST}"
  DeleteRegKey HKLM "${REG_APP}"
SectionEnd

; -----------------------------------------------------------------------------
; Silent install/uninstall support
; -----------------------------------------------------------------------------
Function .onInit
  ${GetParameters} $R0
  ${GetOptions} $R0 "/S" $R1
  IfErrors +2
    SetSilent silent
FunctionEnd

Function un.onInit
  ${GetParameters} $R0
  ${GetOptions} $R0 "/S" $R1
  IfErrors +2
    SetSilent silent
FunctionEnd
