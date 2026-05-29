# K9 Web Protection — Windows

A free, open-source parental control and web filtering app for Windows. Built with Go and Wails, it installs as a standard Windows application and protects against adult content, harmful websites, and configurable keyword matches.

**Support:** [hello@khaleel.eu](mailto:hello@khaleel.eu)

---

## How it works

K9 uses two independent layers of protection:

| Layer | Mechanism | What it blocks |
|-------|-----------|----------------|
| **Layer 1** | `C:\Windows\System32\drivers\etc\hosts` | Domains — works for every app, even offline |
| **Layer 2** | Local HTTP proxy on `127.0.0.1:8080` | URLs, keywords, adult databases, image search, YouTube |

Both layers activate when you click **Enable Protection** in the app.

---

## Default password

The app ships with a default password so filters are protected from the moment it is installed:

```
k9.khaleel.eu
```

Change it in **Settings → Uninstall Protection** after first launch.

---

## Install (end users)

1. Download `K9WebProtection-setup.exe` from the [Releases](https://github.com/yourusername/K9-Web-Protection/releases) page
2. Double-click it and follow the wizard — it installs to `C:\Program Files\K9 Web Protection\`
3. K9 launches automatically and starts with Windows (HKLM Run key, all users)

> The installer requires **Administrator** privileges. Windows may show a UAC prompt — click **Yes**.

---

## Build from source

### Required tools

Install all of these before attempting a build:

| Tool | Version | Download |
|------|---------|----------|
| **Go** | 1.22 or later | [go.dev/dl](https://go.dev/dl/) |
| **Node.js** | 18 LTS or later | [nodejs.org](https://nodejs.org/) |
| **Wails CLI** | v2.12.0 | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| **NSIS** | 3.x | [nsis.sourceforge.io](https://nsis.sourceforge.io/Download) — install to `C:\Program Files (x86)\NSIS\` |
| **WebView2** | Bundled by Wails | No separate install needed — Wails embeds it |

> All builds must be run from an **Administrator PowerShell** because the hosts file and system proxy require admin rights.

### 1 — Install Go

Download and run the installer from [go.dev/dl](https://go.dev/dl/). After installing, open a new PowerShell window and verify:

```powershell
go version
# go version go1.22.x windows/amd64
```

### 2 — Install Node.js

Download the LTS installer from [nodejs.org](https://nodejs.org/). Verify:

```powershell
node --version   # v18.x or later
npm --version
```

### 3 — Install Wails

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Verify the install:

```powershell
wails version
# Wails CLI v2.12.0
```

If `wails` is not found after install, add the Go bin directory to your PATH:

```powershell
$env:PATH += ";$env:GOPATH\bin"
# Or permanently via System Properties → Environment Variables
```

### 4 — Install NSIS

Download the NSIS installer from [nsis.sourceforge.io](https://nsis.sourceforge.io/Download) and install it.
The build script expects it at `C:\Program Files (x86)\NSIS\makensis.exe`.

---

## Build the app

```powershell
# From the repo root — run as Administrator
powershell -ExecutionPolicy Bypass -File .\windows\build.ps1
```

Or step by step from `windows/app/`:

```powershell
cd windows\app

# Tidy Go modules
go mod tidy

# Build the Wails app (embeds WebView2 runtime)
wails build -platform windows/amd64 -webview2 embed

# Output: windows\app\build\bin\K9WebProtection.exe
```

> **Application Control policy error?** If Windows blocks the Wails bindings generator (`wailsbindings.exe`), add `-skipbindings` to skip regenerating Go→JS bindings. Only needed after changes to Go method signatures.
>
> ```powershell
> wails build -platform windows/amd64 -webview2 embed -skipbindings
> ```

---

## Create the installer with NSIS

After a successful Wails build, package it into a setup wizard:

```powershell
& "C:\Program Files (x86)\NSIS\makensis.exe" `
    windows\app\build\windows\installer\project.nsi
```

Output: `windows\app\build\bin\K9WebProtection-setup.exe`

### What the installer does

| Action | Detail |
|--------|--------|
| Installs to | `C:\Program Files\K9 Web Protection\` |
| Auto-start | Writes `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run` — starts for all users |
| Shortcuts | Start Menu + Desktop |
| Programs list | Appears in Add/Remove Programs with version info |
| Clean install | Removes any previous config (`%APPDATA%\K9WebProtection`) on each install |
| Uninstaller | `C:\Program Files\K9 Web Protection\Uninstall.exe` |

### NSIS script location

```
windows/app/build/windows/installer/project.nsi
```

Key variables at the top of the script you may want to edit:

```nsis
!define APP_NAME    "K9 Web Protection"
!define APP_VERSION "2.0.0"
!define APP_EXE     "K9WebProtection.exe"
```

---

## One-command build (app + installer)

```powershell
# Run as Administrator from the repo root
powershell -ExecutionPolicy Bypass -File .\windows\build.ps1
```

`windows\build.ps1` delegates to `windows\app\build.ps1` which runs:
1. `go mod tidy`
2. `wails build -platform windows/amd64 -webview2 embed`
3. Signs the `.exe` (if `$CertThumbprint` is set)
4. `makensis project.nsi`
5. Signs `K9WebProtection-setup.exe` (if `$CertThumbprint` is set)

---

## Code signing — remove "Unknown Publisher"

Without a certificate, Windows shows **"Unknown Publisher"** in the UAC prompt and SmartScreen may block the installer entirely. Code signing fixes this by proving the binary came from you and has not been tampered with.

### Why it matters

| State | UAC prompt | SmartScreen |
|-------|-----------|-------------|
| Unsigned | "Unknown Publisher" (red/orange) | "Windows protected your PC" — blocks run |
| OV certificate | Your name shown | Warning shown, but can click "More info → Run anyway" |
| EV certificate | Your name shown | No SmartScreen warning at all |

### Option A — SignPath.io (recommended for open source — free)

SignPath gives open-source projects a Microsoft-trusted code signing certificate at no cost.

1. Go to [signpath.io/product/open-source](https://signpath.io/product/open-source)
2. Create a free account and apply for open-source signing
3. Link your GitHub repository (must be public)
4. Approval takes 1–3 business days
5. Once approved, SignPath signs your build artifacts via their CI integration or API

SignPath can sign via a GitHub Actions step — no certificate file to manage locally.

### Option B — Azure Trusted Signing (~$9.99/month)

Microsoft's own signing service. Works for individuals without a registered company.

1. Create an Azure account at [portal.azure.com](https://portal.azure.com)
2. Search for **Trusted Signing** and create an account + certificate profile
3. Follow Microsoft's guide: [learn.microsoft.com/azure/trusted-signing](https://learn.microsoft.com/en-us/azure/trusted-signing/)
4. Install the Azure Trusted Signing plugin for signtool:
   ```powershell
   dotnet tool install --global Azure.CodeSigning.Dlib
   ```
5. Use the plugin in the signing step — the thumbprint is replaced by a JSON metadata file

### Option C — Commercial OV/EV certificate (~$100–500/year)

Buy from a Certificate Authority: [Sectigo](https://sectigo.com), [DigiCert](https://digicert.com), or [GlobalSign](https://globalsign.com).

- **OV (Organisation Validation)** — ~$100–200/year. Requires business identity verification. SmartScreen still warns initially but warning disappears after your file accumulates enough download history.
- **EV (Extended Validation)** — ~$300–500/year. Requires identity + hardware token (YubiKey). Bypasses SmartScreen warnings immediately.

### Signing the build (Options A and C — local certificate)

Once you have a certificate installed in the Windows Certificate Store:

**Step 1 — Find your certificate thumbprint:**

```powershell
Get-ChildItem Cert:\CurrentUser\My | Select-Object Subject, Thumbprint
# or for machine-wide (EV tokens):
Get-ChildItem Cert:\LocalMachine\My | Select-Object Subject, Thumbprint
```

**Step 2 — Set the thumbprint in `build.ps1`:**

Open `windows/app/build.ps1` and set:

```powershell
$CertThumbprint = "YOUR_THUMBPRINT_HERE"
```

**Step 3 — Run the build** — signing happens automatically after each artifact is built:

```powershell
powershell -ExecutionPolicy Bypass -File .\windows\build.ps1
```

The build script signs both the app exe and the installer exe using:

```powershell
signtool sign /sha1 <thumbprint> /tr http://timestamp.digicert.com /td sha256 /fd sha256 `
    /d "K9 Web Protection" /du "https://k9.khaleel.eu" K9WebProtection.exe
```

The `/tr` (timestamp server) flag is critical — it means the signature stays valid even after the certificate expires.

**Verify the signature:**

```powershell
signtool verify /pa /v .\K9WebProtection-setup.exe
```

Or right-click the exe → Properties → Digital Signatures tab.

### Installing signtool.exe

`signtool.exe` is part of the **Windows SDK**. Install it via:

```powershell
winget install Microsoft.WindowsSDK.10.0.22621
```

Or download from [developer.microsoft.com/windows/downloads/windows-sdk](https://developer.microsoft.com/en-us/windows/downloads/windows-sdk/). The build script auto-locates the latest `signtool.exe` in `C:\Program Files (x86)\Windows Kits\10\bin\`.

---

## Proxy — known issues & internet recovery

The Layer 2 proxy routes all HTTP/HTTPS traffic through `127.0.0.1:8080`. If the proxy process stops while the system proxy is still enabled, **all internet access will fail** with "connection refused".

| Scenario | Result |
|----------|--------|
| App closes normally | Proxy registry setting cleared automatically ✓ |
| NSIS uninstaller runs | Proxy disabled, proxy registry key removed ✓ |
| App killed via Task Manager | Run the app again and click Disable Protection |
| System crash / hard reboot | App auto-starts on next login and re-syncs ✓ |

### Fix internet if the proxy gets stuck on

**Option 1 — Open K9 and click Disable Protection**

**Option 2 — Registry (run as Administrator):**

```powershell
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" `
    /v ProxyEnable /t REG_DWORD /d 0 /f
```

**Option 3 — Internet Options (no admin needed):**
1. Open **Control Panel → Internet Options → Connections → LAN settings**
2. Uncheck **Use a proxy server for your LAN**
3. Click OK

---

## Uninstall

**From within the app** — open Settings → Danger Zone → Uninstall. The app stops protection, then silently launches the NSIS uninstaller which removes all files, shortcuts, and registry entries.

**From Windows** — open **Control Panel → Programs → Uninstall a program**, find **K9 Web Protection**, and click Uninstall.

**Manual removal** (if the uninstaller fails):

```powershell
# Stop the app
taskkill /f /im K9WebProtection.exe

# Remove from auto-start
reg delete "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v K9WebProtection /f

# Clear the system proxy
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyEnable /t REG_DWORD /d 0 /f

# Remove files
Remove-Item -Recurse -Force "C:\Program Files\K9 Web Protection"

# Remove config and stats
Remove-Item -Recurse -Force "$env:APPDATA\K9WebProtection"

# Remove from Programs list
reg delete "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\K9WebProtection" /f
reg delete "HKLM\SOFTWARE\K9WebProtection" /f

# Remove shortcuts
Remove-Item -Force "$env:PUBLIC\Desktop\K9 Web Protection.lnk"
Remove-Item -Recurse -Force "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\K9 Web Protection"
```

---

## Project structure

```
windows/
├── app/
│   ├── main.go                          # Wails entry point, window config (dark theme)
│   ├── app.go                           # All Go ↔ frontend bindings & business logic
│   ├── go.mod                           # Go module: k9webprotection, Go 1.22, Wails v2.12
│   ├── internal/
│   │   ├── config/config.go             # Persistent config at %APPDATA%\K9WebProtection\
│   │   ├── proxy/proxy.go               # Layer 2: HTTP proxy engine (port 8080)
│   │   ├── hosts/hosts.go               # Layer 1: hosts file management (elevation aware)
│   │   └── database/
│   │       ├── database.go              # Loads & queries the embedded blocklists
│   │       ├── domains.json             # ~674 KB — built-in domain blocklist
│   │       ├── urls.json                # ~163 KB — built-in URL pattern list
│   │       └── multi-words.json         # ~265 KB — built-in keyword list
│   ├── frontend/
│   │   ├── index.html                   # Single-page app shell
│   │   └── src/
│   │       ├── main.js                  # Dashboard, blocklist, settings logic
│   │       └── style.css                # Dark theme UI
│   ├── build/
│   │   ├── appicon.png                  # 256×256 app icon
│   │   ├── bin/                         # Build output (exe + setup.exe)
│   │   └── windows/
│   │       ├── icon.ico                 # Multi-resolution ICO (16–256 px)
│   │       ├── app.manifest             # UAC manifest: requireAdministrator
│   │       └── installer/
│   │           └── project.nsi          # NSIS installer script
│   └── build.ps1                        # Step-by-step build script (wails + makensis)
└── build.ps1                            # Root convenience script → delegates to app/build.ps1
```

---

## Config file location

```
%APPDATA%\K9WebProtection\config.json
```

The config stores blocklists, settings, password hash (bcrypt), and block statistics. The NSIS installer wipes this directory on each install for a clean slate.

---

**Version:** 2.0.0 | **Platform:** Windows 10/11 x64 | **Stack:** Go + Wails | **Support:** [hello@khaleel.eu](mailto:hello@khaleel.eu)
