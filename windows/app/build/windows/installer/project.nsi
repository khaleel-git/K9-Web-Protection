; K10 Web Protection — NSIS Installer
; Build:
;   makensis /V3 /DVERSION=1.0.0 project.nsi
;
; Files expected in build/bin/ before running:
;   K10WebProtection.exe    (wails build output)
;   k10_watchdog.ps1        (copied by build-pkg.ps1)
;
; Files in this directory:
;   setup-system.ps1        (post-install Task Sched / Firewall / icacls)
;   teardown-system.ps1     (pre-uninstall cleanup)

; ── Includes ───────────────────────────────────────────────────────────────────
!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "x64.nsh"

; ── Constants ─────────────────────────────────────────────────────────────────
!ifndef VERSION
  !define VERSION "1.0.0"
!endif

Name              "K10 Web Protection"
OutFile           "..\..\bin\K10WebProtection-${VERSION}-setup.exe"
InstallDir        "$PROGRAMFILES64\K10 Web Protection"
InstallDirRegKey  HKLM "Software\K10 Web Protection" "InstallDir"
RequestExecutionLevel admin
Unicode True

; ── MUI Settings ──────────────────────────────────────────────────────────────
!define MUI_ABORTWARNING
!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"

; Installer pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN          "$INSTDIR\K10WebProtection.exe"
!define MUI_FINISHPAGE_RUN_TEXT     "Launch K10 Web Protection"
!define MUI_FINISHPAGE_SHOWREADME   ""
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

; ── Version info embedded in the .exe ─────────────────────────────────────────
VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName"     "K10 Web Protection"
VIAddVersionKey "CompanyName"     "K10 Web Protection"
VIAddVersionKey "FileDescription" "K10 Web Protection Installer"
VIAddVersionKey "FileVersion"     "${VERSION}"
VIAddVersionKey "ProductVersion"  "${VERSION}"
VIAddVersionKey "LegalCopyright"  "K10 Web Protection"

; ── Installer Section ─────────────────────────────────────────────────────────
Section "K10 Web Protection" SecMain
    SectionIn RO  ; Required

    SetOutPath "$INSTDIR"
    SetOverwrite on

    ; Kill any running instance before overwriting the exe
    DetailPrint "Stopping existing instance..."
    nsExec::Exec 'taskkill /F /IM K10WebProtection.exe'
    Sleep 800

    ; Unlock existing files before overwriting (in case of reinstall)
    nsExec::Exec 'icacls "$INSTDIR\K10WebProtection.exe" /remove:d Everyone'
    nsExec::Exec 'icacls "$INSTDIR\k10_watchdog.ps1"    /remove:d Everyone'

    ; ── Copy main files ────────────────────────────────────────────────────────
    DetailPrint "Copying files..."
    File "..\..\bin\K10WebProtection.exe"
    File "..\..\bin\k10_watchdog.ps1"
    File "setup-system.ps1"
    File "teardown-system.ps1"

    ; ── Write uninstaller ──────────────────────────────────────────────────────
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; ── Registry: app location ────────────────────────────────────────────────
    WriteRegStr HKLM "Software\K10 Web Protection" "InstallDir" "$INSTDIR"
    WriteRegStr HKLM "Software\K10 Web Protection" "Version"    "${VERSION}"

    ; ── Registry: Add/Remove Programs ─────────────────────────────────────────
    WriteRegStr   HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\K10WebProtection" \
        "DisplayName"     "K10 Web Protection"
    WriteRegStr   HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\K10WebProtection" \
        "UninstallString" '"$INSTDIR\Uninstall.exe"'
    WriteRegStr   HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\K10WebProtection" \
        "DisplayVersion"  "${VERSION}"
    WriteRegStr   HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\K10WebProtection" \
        "Publisher"       "K10 Web Protection"
    WriteRegStr   HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\K10WebProtection" \
        "DisplayIcon"     "$INSTDIR\K10WebProtection.exe"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\K10WebProtection" \
        "NoModify"        1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\K10WebProtection" \
        "NoRepair"        1

    ; ── Shortcuts ─────────────────────────────────────────────────────────────
    DetailPrint "Creating shortcuts..."
    CreateDirectory "$SMPROGRAMS\K10 Web Protection"
    CreateShortcut  "$SMPROGRAMS\K10 Web Protection\K10 Web Protection.lnk" \
        "$INSTDIR\K10WebProtection.exe" "" "$INSTDIR\K10WebProtection.exe" 0
    CreateShortcut  "$SMPROGRAMS\K10 Web Protection\Uninstall.lnk" \
        "$INSTDIR\Uninstall.exe"
    CreateShortcut  "$DESKTOP\K10 Web Protection.lnk" \
        "$INSTDIR\K10WebProtection.exe" "" "$INSTDIR\K10WebProtection.exe" 0

    ; ── System integration (Task Scheduler, Firewall, file locking) ───────────
    DetailPrint "Configuring system integration..."
    nsExec::ExecToLog 'powershell.exe -NonInteractive -NoProfile -ExecutionPolicy Bypass \
        -File "$INSTDIR\setup-system.ps1" "$INSTDIR"'
    Pop $0
    ${If} $0 != 0
        DetailPrint "Warning: system integration step returned $0"
    ${EndIf}

    ; setup-system.ps1 is a one-shot script — remove after running
    Delete "$INSTDIR\setup-system.ps1"

    ; ── Start watchdog ────────────────────────────────────────────────────────
    DetailPrint "Starting watchdog..."
    nsExec::Exec 'schtasks /run /tn "K10WebProtection-Watchdog"'

SectionEnd

; ── Uninstaller Section ───────────────────────────────────────────────────────
Section "Uninstall"

    ; ── System cleanup (must run before files are removed) ────────────────────
    DetailPrint "Cleaning up system integration..."
    nsExec::ExecToLog 'powershell.exe -NonInteractive -NoProfile -ExecutionPolicy Bypass \
        -File "$INSTDIR\teardown-system.ps1"'

    ; ── Remove files ──────────────────────────────────────────────────────────
    DetailPrint "Removing files..."
    Delete "$INSTDIR\K10WebProtection.exe"
    Delete "$INSTDIR\k10_watchdog.ps1"
    Delete "$INSTDIR\teardown-system.ps1"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir  "$INSTDIR"

    ; ── Remove shortcuts ──────────────────────────────────────────────────────
    Delete "$SMPROGRAMS\K10 Web Protection\*.lnk"
    RMDir  "$SMPROGRAMS\K10 Web Protection"
    Delete "$DESKTOP\K10 Web Protection.lnk"

    ; ── Remove registry ───────────────────────────────────────────────────────
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\K10WebProtection"
    DeleteRegKey HKLM "Software\K10 Web Protection"

SectionEnd
