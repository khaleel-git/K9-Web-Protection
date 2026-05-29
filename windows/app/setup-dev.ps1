# K9 Web Protection — Windows Dev Setup
# Run this once to install Go, Node, Wails, then launch the dev server.
# Must be run as Administrator (right-click → Run as Administrator).

param(
    [switch]$BuildOnly   # skip install checks, just build
)

$ErrorActionPreference = "Stop"

function Check-Admin {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    $p  = [Security.Principal.WindowsPrincipal]$id
    return $p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Check-Admin)) {
    Write-Host "ERROR: Please run this script as Administrator." -ForegroundColor Red
    exit 1
}

Write-Host "`n=== K9 Web Protection — Dev Setup ===" -ForegroundColor Cyan

# ── 1. Go ──────────────────────────────────────────────────────────────────────
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "`n[1/3] Installing Go via winget..." -ForegroundColor Yellow
    winget install --id GoLang.Go --accept-source-agreements --accept-package-agreements
    # Refresh PATH
    $env:PATH = [System.Environment]::GetEnvironmentVariable("PATH","Machine") + ";" +
                [System.Environment]::GetEnvironmentVariable("PATH","User")
} else {
    Write-Host "`n[1/3] Go already installed: $(go version)" -ForegroundColor Green
}

# ── 2. Node.js ─────────────────────────────────────────────────────────────────
if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Host "`n[2/3] Installing Node.js via winget..." -ForegroundColor Yellow
    winget install --id OpenJS.NodeJS.LTS --accept-source-agreements --accept-package-agreements
    $env:PATH = [System.Environment]::GetEnvironmentVariable("PATH","Machine") + ";" +
                [System.Environment]::GetEnvironmentVariable("PATH","User")
} else {
    Write-Host "`n[2/3] Node already installed: $(node --version)" -ForegroundColor Green
}

# ── 3. Wails CLI ───────────────────────────────────────────────────────────────
if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {
    Write-Host "`n[3/3] Installing Wails CLI..." -ForegroundColor Yellow
    go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
    $env:PATH = "$env:USERPROFILE\go\bin;" + $env:PATH
} else {
    Write-Host "`n[3/3] Wails already installed: $(wails version)" -ForegroundColor Green
}

# ── Change to app directory ────────────────────────────────────────────────────
$appDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $appDir

# ── go mod tidy ────────────────────────────────────────────────────────────────
Write-Host "`nRunning go mod tidy..." -ForegroundColor Yellow
go mod tidy

if ($BuildOnly) {
    # ── Production build ────────────────────────────────────────────────────────
    Write-Host "`nBuilding K9 Web Protection (Windows)..." -ForegroundColor Yellow
    wails build -platform windows/amd64
    Write-Host "`nBuild complete! Output: build\bin\K9WebProtection.exe" -ForegroundColor Green
} else {
    # ── Dev mode ────────────────────────────────────────────────────────────────
    Write-Host "`nStarting dev server (wails dev)..." -ForegroundColor Cyan
    Write-Host "The app window will open automatically.`n"
    wails dev
}
