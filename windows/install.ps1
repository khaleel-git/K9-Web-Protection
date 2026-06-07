# K10 Web Protection — Installer
# Run once after building: powershell -ExecutionPolicy Bypass -File install.ps1
# Equivalent to mac/install.sh

$ErrorActionPreference = 'Stop'

$ScriptDir   = Split-Path -Parent $MyInvocation.MyCommand.Path
$ExeSrc      = "$ScriptDir\app\build\bin\K10WebProtection.exe"
$WatchdogSrc = "$ScriptDir\k10_watchdog.ps1"
$InstallDir  = 'C:\Program Files\K10 Web Protection'
$ExeDst      = "$InstallDir\K10WebProtection.exe"
$WatchdogDst = "$InstallDir\k10_watchdog.ps1"
$ProxyPort   = 8080

function Write-Step { param($n, $t) Write-Host "[$n] $t..." -ForegroundColor Cyan }
function Write-OK   { Write-Host "  Done." -ForegroundColor Green }
function Write-Fail { param($m) Write-Host "  ERROR: $m" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "  K10 Web Protection — Installer"
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host ""

# ── Elevate if not already Administrator ──────────────────────────────────────
if (-not ([Security.Principal.WindowsPrincipal]
        [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
        [Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "  Administrator privileges required. Re-launching elevated..." -ForegroundColor Yellow
    Start-Process powershell -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`"" `
        -Verb RunAs -Wait
    exit
}

# ── Pre-flight checks ─────────────────────────────────────────────────────────
if (-not (Test-Path $ExeSrc)) {
    Write-Fail "Executable not found at $ExeSrc`n  Run 'wails build' inside the app/ directory first."
}

# ── 1. Stop any running instance ──────────────────────────────────────────────
Write-Step "1/6" "Stopping running instance"
Stop-ScheduledTask -TaskName 'K10WebProtection-Watchdog' -ErrorAction SilentlyContinue
Get-Process -Name 'K10WebProtection' -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Milliseconds 800
Write-OK

# ── 2. Copy files ─────────────────────────────────────────────────────────────
Write-Step "2/6" "Installing to $InstallDir"

# Unlock existing files before overwriting
foreach ($f in @($ExeDst, $WatchdogDst)) {
    if (Test-Path $f) { icacls $f /remove:d Everyone 2>$null | Out-Null }
}

New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Copy-Item -Path $ExeSrc      -Destination $ExeDst      -Force
Copy-Item -Path $WatchdogSrc -Destination $WatchdogDst -Force

# Register in Add/Remove Programs
$uninstKey = 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\K10WebProtection'
New-Item -Path $uninstKey -Force | Out-Null
Set-ItemProperty -Path $uninstKey -Name DisplayName     -Value 'K10 Web Protection'
Set-ItemProperty -Path $uninstKey -Name UninstallString -Value "`"$ScriptDir\uninstall.ps1`""
Set-ItemProperty -Path $uninstKey -Name DisplayVersion  -Value '1.0.0'
Set-ItemProperty -Path $uninstKey -Name Publisher       -Value 'K10 Web Protection'
Set-ItemProperty -Path $uninstKey -Name DisplayIcon     -Value $ExeDst
Set-ItemProperty -Path $uninstKey -Name NoModify        -Value 1
Set-ItemProperty -Path $uninstKey -Name NoRepair        -Value 1
Write-OK

# ── 3. Create shortcuts ───────────────────────────────────────────────────────
Write-Step "3/6" "Creating shortcuts"
$shell   = New-Object -ComObject WScript.Shell
$menuDir = Join-Path ([Environment]::GetFolderPath('CommonPrograms')) 'K10 Web Protection'
New-Item -ItemType Directory -Path $menuDir -Force | Out-Null

$lnk = $shell.CreateShortcut("$menuDir\K10 Web Protection.lnk")
$lnk.TargetPath = $ExeDst; $lnk.WorkingDirectory = $InstallDir; $lnk.Save()

$desk = $shell.CreateShortcut(
    (Join-Path ([Environment]::GetFolderPath('CommonDesktopDirectory')) 'K10 Web Protection.lnk'))
$desk.TargetPath = $ExeDst; $desk.WorkingDirectory = $InstallDir; $desk.Save()
Write-OK

# ── 4. Task Scheduler — auto-start + restart on crash ────────────────────────
Write-Step "4/6" "Registering Task Scheduler tasks"

$action    = New-ScheduledTaskAction -Execute $ExeDst
$trigger   = New-ScheduledTaskTrigger -AtLogOn
$settings  = New-ScheduledTaskSettingsSet `
    -RestartCount 999 -RestartInterval (New-TimeSpan -Seconds 12) `
    -ExecutionTimeLimit ([TimeSpan]::Zero) -MultipleInstances IgnoreNew
$principal = New-ScheduledTaskPrincipal -GroupId 'BUILTIN\Users' -RunLevel Highest
Register-ScheduledTask -TaskName 'K10WebProtection' `
    -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null

$wAction    = New-ScheduledTaskAction -Execute 'powershell.exe' `
    -Argument "-NonInteractive -NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File `"$WatchdogDst`""
$wTrigger   = New-ScheduledTaskTrigger -AtStartup
$wSettings  = New-ScheduledTaskSettingsSet `
    -ExecutionTimeLimit ([TimeSpan]::Zero) `
    -RestartCount 999 -RestartInterval (New-TimeSpan -Seconds 30)
$wPrincipal = New-ScheduledTaskPrincipal -UserId 'SYSTEM' -RunLevel Highest -LogonType ServiceAccount
Register-ScheduledTask -TaskName 'K10WebProtection-Watchdog' `
    -Action $wAction -Trigger $wTrigger -Settings $wSettings -Principal $wPrincipal -Force | Out-Null
Write-OK

# ── 5. Block QUIC (UDP 443) via Windows Firewall ──────────────────────────────
Write-Step "5/6" "Blocking QUIC/UDP 443 via Windows Firewall"
netsh advfirewall firewall delete rule name='K10 Block QUIC' 2>$null | Out-Null
netsh advfirewall firewall add rule name='K10 Block QUIC' `
    dir=out protocol=UDP localport=443 action=block | Out-Null
Write-OK

# ── 6. Lock files ─────────────────────────────────────────────────────────────
Write-Step "6/6" "Locking files"
icacls $ExeDst      /deny 'Everyone:(DE,WD,AD)' 2>$null | Out-Null
icacls $WatchdogDst /deny 'Everyone:(DE,WD,AD)' 2>$null | Out-Null
Write-OK

# ── Start the app ─────────────────────────────────────────────────────────────
Write-Host ""
Start-ScheduledTask -TaskName 'K10WebProtection-Watchdog' -ErrorAction SilentlyContinue
Start-Process -FilePath $ExeDst

Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "  K10 Web Protection installed successfully!" -ForegroundColor Green
Write-Host "  Open the app and click Enable Protection."
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host ""
