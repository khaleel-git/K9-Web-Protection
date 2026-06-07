# K10 Web Protection — Uninstaller
# Run as Administrator: powershell -ExecutionPolicy Bypass -File uninstall.ps1
# Equivalent to mac/uninstall.sh

$ErrorActionPreference = 'SilentlyContinue'

$InstallDir  = 'C:\Program Files\K10 Web Protection'
$ExeDst      = "$InstallDir\K10WebProtection.exe"
$WatchdogDst = "$InstallDir\k10_watchdog.ps1"

function Write-Step { param($n, $t) Write-Host "[$n] $t..." -ForegroundColor Cyan }
function Write-OK   { Write-Host "  Done." -ForegroundColor Green }

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "  K10 Web Protection — Uninstaller"
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host ""

# ── Elevate if not already Administrator ──────────────────────────────────────
if (-not ([Security.Principal.WindowsPrincipal]
        [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
        [Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Start-Process powershell -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`"" `
        -Verb RunAs -Wait
    exit
}

# ── 1. Stop watchdog and main process ─────────────────────────────────────────
Write-Step "1/5" "Stopping services"
Stop-ScheduledTask   -TaskName 'K10WebProtection-Watchdog'
Get-Process -Name 'K10WebProtection' | Stop-Process -Force
Start-Sleep -Milliseconds 800
Write-OK

# ── 2. Remove Task Scheduler tasks ────────────────────────────────────────────
Write-Step "2/5" "Removing scheduled tasks"
Unregister-ScheduledTask -TaskName 'K10WebProtection'          -Confirm:$false
Unregister-ScheduledTask -TaskName 'K10WebProtection-Watchdog' -Confirm:$false
Write-OK

# ── 3. Remove Windows Firewall rule ──────────────────────────────────────────
Write-Step "3/5" "Removing firewall rules"
netsh advfirewall firewall delete rule name='K10 Block QUIC' 2>$null | Out-Null
Write-OK

# ── 4. Clear system proxy and clean hosts file ────────────────────────────────
Write-Step "4/5" "Clearing proxy and hosts entries"

$key = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings'
Set-ItemProperty -Path $key -Name ProxyEnable -Value 0

# Notify WinINet that proxy settings changed
try {
    Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class WI2{[DllImport("wininet.dll")]public static extern bool InternetSetOption(IntPtr h,int o,IntPtr b,int l);}
'@ -ErrorAction Stop
    [WI2]::InternetSetOption([IntPtr]::Zero,39,[IntPtr]::Zero,0)|Out-Null
    [WI2]::InternetSetOption([IntPtr]::Zero,37,[IntPtr]::Zero,0)|Out-Null
} catch {}

# Remove K10 hosts entries
$hostsPath = "$env:SystemRoot\System32\drivers\etc\hosts"
if (Test-Path $hostsPath) {
    $c = [System.IO.File]::ReadAllText($hostsPath)
    $c = [System.Text.RegularExpressions.Regex]::Replace(
        $c, '(?s)\r?\n# K10-Web-Protection START.*?# K10-Web-Protection END\r?\n?', '')
    $c = [System.Text.RegularExpressions.Regex]::Replace(
        $c, '(?s)\r?\n# K10-SafeSearch START.*?# K10-SafeSearch END\r?\n?', '')
    [System.IO.File]::WriteAllText($hostsPath, $c)
    ipconfig /flushdns | Out-Null
}

# Remove K10 CA from Windows trust stores
Get-ChildItem Cert:\LocalMachine\Root |
    Where-Object { $_.Subject -like '*K10 Web Protection*' } |
    Remove-Item -ErrorAction SilentlyContinue
Get-ChildItem Cert:\CurrentUser\Root |
    Where-Object { $_.Subject -like '*K10 Web Protection*' } |
    Remove-Item -ErrorAction SilentlyContinue
Write-OK

# ── 5. Unlock and remove files ────────────────────────────────────────────────
Write-Step "5/5" "Removing files"

foreach ($f in @($ExeDst, $WatchdogDst)) {
    if (Test-Path $f) { icacls $f /remove:d Everyone 2>$null | Out-Null }
}

if (Test-Path $InstallDir) {
    Remove-Item -Path $InstallDir -Recurse -Force
}

# Remove shortcuts
$menuDir  = Join-Path ([Environment]::GetFolderPath('CommonPrograms')) 'K10 Web Protection'
$deskLink = Join-Path ([Environment]::GetFolderPath('CommonDesktopDirectory')) 'K10 Web Protection.lnk'
Remove-Item -Path $menuDir  -Recurse -Force
Remove-Item -Path $deskLink -Force

# Remove Add/Remove Programs entry
Remove-Item -Path 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\K10WebProtection' `
    -Recurse -Force
Write-OK

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "  K10 Web Protection uninstalled." -ForegroundColor Green
Write-Host "  Config kept at %APPDATA%\.k10webprotection — remove manually if needed."
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host ""
