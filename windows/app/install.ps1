# K9 Web Protection — Install after build
# Run as Administrator after building with build.ps1.
# Copies the exe to Program Files and adds it to startup.

param(
    [switch]$Uninstall
)

$ErrorActionPreference = "Stop"
$appName    = "K9 Web Protection"
$exeName    = "K9WebProtection.exe"
$installDir = "C:\Program Files\K9 Web Protection"
$exeSrc     = "$PSScriptRoot\build\bin\$exeName"
$exeDst     = "$installDir\$exeName"
$regRunKey  = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run"
$regRunName = "K9WebProtection"

function Check-Admin {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    ([Security.Principal.WindowsPrincipal]$id).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}
if (-not (Check-Admin)) {
    Write-Host "ERROR: Run as Administrator." -ForegroundColor Red; exit 1
}

if ($Uninstall) {
    Write-Host "Uninstalling $appName..." -ForegroundColor Yellow
    Remove-ItemProperty -Path $regRunKey -Name $regRunName -ErrorAction SilentlyContinue
    Stop-Process -Name "K9WebProtection" -Force -ErrorAction SilentlyContinue
    Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "Uninstalled." -ForegroundColor Green
    exit 0
}

Write-Host "Installing $appName..." -ForegroundColor Cyan

if (-not (Test-Path $exeSrc)) {
    Write-Host "ERROR: Build first — run build.ps1" -ForegroundColor Red; exit 1
}

New-Item -ItemType Directory -Force $installDir | Out-Null
Copy-Item $exeSrc $exeDst -Force

# Register in startup (runs at login for all users)
Set-ItemProperty -Path $regRunKey -Name $regRunName -Value "`"$exeDst`""

Write-Host "Installed to: $exeDst" -ForegroundColor Green
Write-Host "Auto-start:   HKLM Run key added" -ForegroundColor Green
Write-Host "`nLaunch now? (y/n) " -NoNewline -ForegroundColor Cyan
$ans = Read-Host
if ($ans -eq 'y') { Start-Process $exeDst }
