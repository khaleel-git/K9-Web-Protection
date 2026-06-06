# K10 Web Protection — Production Build
# Requires Go, Node, Wails, and NSIS installed.
# Run as Administrator.

$ErrorActionPreference = "Stop"
$appDir    = Split-Path -Parent $MyInvocation.MyCommand.Path
$nsisExe   = "C:\Program Files (x86)\NSIS\makensis.exe"
$nsiScript = "$appDir\build\windows\installer\project.nsi"
$exePath   = "$appDir\build\bin\K10WebProtection.exe"
$setup     = "$appDir\build\bin\K10WebProtection-setup.exe"

Set-Location $appDir

# ── Sync database JSON files from Mac source ──────────────────────────────────
Write-Host "`nSyncing database files..." -ForegroundColor Cyan
$macDb  = "$appDir\..\..\mac\app\internal\database"
$winDb  = "$appDir\internal\database"
foreach ($f in @("domains.json","urls.json","url-patterns.json","multi-words.json")) {
    $src = Join-Path $macDb $f
    $dst = Join-Path $winDb $f
    if (Test-Path $src) {
        Copy-Item $src $dst -Force
        Write-Host "  Copied $f" -ForegroundColor DarkGray
    } else {
        Write-Warning "  Missing $f in mac database dir — build may fail"
    }
}

# ── SignPath config ────────────────────────────────────────────────────────────
# OrgId and slugs are safe to store here.
# Token is read from the SIGNPATH_TOKEN environment variable — never commit a token to git.
#
# Set the token once in your PowerShell session before building:
#   $env:SIGNPATH_TOKEN = "your-token-here"
#
# Or set it permanently in Windows:
#   [System.Environment]::SetEnvironmentVariable("SIGNPATH_TOKEN","<token>","User")

$SignPathOrgId   = "13d0a0e1-fd9c-4c57-bd9d-e419053173ad"
$SignPathToken   = $env:SIGNPATH_TOKEN
$SignPathProject = "K10-Web-Protection"
$SignPathPolicy  = "K10-Web-Protection"

function Sign-WithSignPath {
    param([string]$Path)

    if (-not $SignPathToken) {
        Write-Host "  Signing skipped — run: `$env:SIGNPATH_TOKEN = `"your-token`"" -ForegroundColor DarkGray
        return
    }

    if (-not (Get-Module -ListAvailable -Name SignPath -ErrorAction SilentlyContinue)) {
        Write-Host "  Installing SignPath PowerShell module..." -ForegroundColor Cyan
        Install-Module -Name SignPath -Force -Scope CurrentUser
    }

    Write-Host "  Submitting to SignPath: $([IO.Path]::GetFileName($Path))..." -ForegroundColor Cyan

    Submit-SigningRequest `
        -OrganizationId    $SignPathOrgId `
        -ApiToken          $SignPathToken `
        -ProjectSlug       $SignPathProject `
        -SigningPolicySlug $SignPathPolicy `
        -InputArtifactPath  $Path `
        -OutputArtifactPath $Path `
        -WaitForCompletion

    Write-Host "  Signed OK" -ForegroundColor Green
}

# ── Build ──────────────────────────────────────────────────────────────────────
Write-Host "`nBuilding K10 Web Protection..." -ForegroundColor Cyan
go mod tidy
wails build -platform windows/amd64 -webview2 embed -skipbindings

Write-Host "`nSigning executable..." -ForegroundColor Cyan
Sign-WithSignPath $exePath

# ── Installer ──────────────────────────────────────────────────────────────────
Write-Host "`nBuilding installer..." -ForegroundColor Cyan
& $nsisExe $nsiScript

Write-Host "`nSigning installer..." -ForegroundColor Cyan
Sign-WithSignPath $setup

# ── Done ───────────────────────────────────────────────────────────────────────
$sizeMB = [math]::Round((Get-Item $setup).Length / 1MB, 1)
Write-Host "`nDone!" -ForegroundColor Green
Write-Host "Installer: $setup ($sizeMB MB)"
