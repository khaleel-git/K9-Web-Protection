# K10 Web Protection — Package Builder
# Produces a distributable Windows setup.exe installer via NSIS.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File build-pkg.ps1
#
# Requirements: Go, Node, Wails, NSIS (makensis in PATH or default install path)

$ErrorActionPreference = 'Stop'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$AppDir    = "$ScriptDir\app"
$Version   = '1.0.0'
$ExePath   = "$AppDir\build\bin\K10WebProtection.exe"
$SetupPath = "$AppDir\build\bin\K10WebProtection-$Version-setup.exe"
$NsiScript = "$AppDir\build\windows\installer\project.nsi"
$NsisExe   = if (Get-Command makensis -ErrorAction SilentlyContinue) {
                 (Get-Command makensis).Source
             } else {
                 'C:\Program Files (x86)\NSIS\makensis.exe'
             }

function Write-Step { param($n, $t) Write-Host "[$n] $t..." -ForegroundColor Cyan }
function Write-OK   { Write-Host "  Done." -ForegroundColor Green }

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "  K10 Web Protection — Package Builder v$Version"
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host ""

# ── Tool checks ───────────────────────────────────────────────────────────────
foreach ($tool in @('wails', 'go')) {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        Write-Host "Error: '$tool' not found in PATH." -ForegroundColor Red
        if ($tool -eq 'wails') {
            Write-Host "  Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
        }
        exit 1
    }
}
if (-not (Test-Path $NsisExe)) {
    Write-Host "Error: NSIS not found." -ForegroundColor Red
    Write-Host "  Download from https://nsis.sourceforge.io/Download"
    Write-Host "  Or: winget install NSIS.NSIS"
    exit 1
}

# ── 1. Sync database files ────────────────────────────────────────────────────
Write-Step "1/4" "Syncing database files from Mac source"
$macDb = "$ScriptDir\..\mac\app\internal\database"
$winDb = "$AppDir\internal\database"
foreach ($f in @('domains.json','urls.json','url-patterns.json','multi-words.json')) {
    $src = Join-Path $macDb $f
    $dst = Join-Path $winDb $f
    if (Test-Path $src) {
        Copy-Item $src $dst -Force
        Write-Host "  Synced $f" -ForegroundColor DarkGray
    } else {
        Write-Warning "  $f not found in mac database dir — build may fail"
    }
}
Write-OK

# ── 2. Build Wails app ────────────────────────────────────────────────────────
Write-Step "2/4" "Building app (windows/amd64)"
Set-Location $AppDir
wails build -platform windows/amd64 -webview2 embed -skipbindings
if (-not (Test-Path $ExePath)) {
    Write-Host "Build failed — $ExePath not found" -ForegroundColor Red; exit 1
}
Write-OK

# ── 3. Stage supporting files in build/bin ────────────────────────────────────
Write-Step "3/4" "Staging supporting files"
$binDir = "$AppDir\build\bin"
Copy-Item -Path "$ScriptDir\k10_watchdog.ps1" -Destination "$binDir\k10_watchdog.ps1" -Force
Write-Host "  Staged k10_watchdog.ps1" -ForegroundColor DarkGray
Write-OK

# ── 4. Build NSIS installer ───────────────────────────────────────────────────
Write-Step "4/4" "Building NSIS installer"
& $NsisExe /V3 "/DVERSION=$Version" $NsiScript
if ($LASTEXITCODE -ne 0) {
    Write-Host "NSIS build failed (exit $LASTEXITCODE)" -ForegroundColor Red; exit 1
}
Write-OK

Set-Location $ScriptDir

$sizeMB = [math]::Round((Get-Item $SetupPath).Length / 1MB, 1)
Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "  Package ready:  K10WebProtection-$Version-setup.exe  ($sizeMB MB)" -ForegroundColor Green
Write-Host ""
Write-Host "  To sign the installer with signtool:" -ForegroundColor Yellow
Write-Host "    signtool sign /fd sha256 /tr http://timestamp.digicert.com /td sha256 \"
Write-Host "      `"$SetupPath`""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host ""
