# K9 Web Protection v2.0 — Windows & macOS

> *Inspired by the original K9 Web Protection. Free, open-source, built for the community.*

Both platforms are now native desktop apps built with Go and Wails. No Python. No screenshots. No screen recording permission. Protection runs entirely at the network level.

---

## What's New in v2.0

- **Rebuilt from scratch** — Go + Wails replaces the Python/screenshot engine on both platforms
- **Native desktop UI** — Dashboard, Block List, Allow List, Keywords, Settings
- **No screen recording permission** — protection runs at the OS network level
- **Works in every browser** — Chrome, Firefox, Safari, Brave, any browser, even incognito
- **Built-in database** — thousands of domains, URL patterns, and keywords embedded in the binary
- **Content toggles** — individually switch adult content, image search, YouTube, and Safe Search
- **Password protection** — password-gate all destructive actions
- **Focus Mode** — timed lock (30 min to 8 hours) that prevents disabling during a session *(macOS)*
- **Disable Delay** — require a waiting period before protection can be turned off *(macOS)*
- **Watchdog** — background agent re-locks files and re-enables the proxy every 10 seconds *(macOS)*
- **Auto-start for all users** — HKLM Run key so protection starts with Windows *(Windows)*

---

## How it works

Two independent protection layers activate together:

| Layer | Mechanism | What it blocks |
|-------|-----------|----------------|
| Layer 1 - Hosts | System hosts file | Domains — every app on the system, even offline |
| Layer 2 - Proxy | Local HTTP proxy on `127.0.0.1:8080` | URLs, keywords, adult database, image search, YouTube |

Both layers survive browser switches, incognito mode, and app changes.

---

## Downloads

| File | Platform | Description |
|------|----------|-------------|
| `K9WebProtection-setup.exe` | Windows 10/11 | Double-click to install — standard setup wizard |
| `K9-Web-Protection-macOS.dmg` | macOS 12+ | Drag to Applications, run `install.sh` |

**Default password:** `k9.khaleel.eu` — change it in Settings after first launch.

---

## Windows Install (v2.0)

1. Download `K9WebProtection-setup.exe`
2. Double-click and follow the setup wizard
3. K9 launches automatically and starts with Windows

The installer requires Administrator privileges. Windows may show a UAC prompt — click Yes.

**If SmartScreen warns about unknown publisher:**
Click **More info** then **Run anyway**. The app is open source — you can review the full source code on GitHub. A verified publisher certificate is pending approval.

---

## macOS Install (v2.0)

**Step 1 — Run the installer:**
```bash
sudo bash /Volumes/K9\ Web\ Protection/install.sh
```

The installer copies the app to `/Applications`, configures the system proxy, installs the LaunchAgent for auto-start, and locks the binary.

**Step 2 — If Gatekeeper blocks the app:**
```bash
sudo xattr -cr "/Applications/K9 Web Protection.app"
```
Then: **System Settings → Privacy & Security → Allow Anyway**

**Step 3 — Open the app and click Enable Protection.**

---

## Requirements

**Windows v2.0:** Windows 10 or 11 (64-bit) — no additional dependencies

**macOS v2.0:** macOS 12 Monterey or later — Apple Silicon or Intel — no additional dependencies

---

## Privacy

- All processing is 100% local — no data, URLs, or stats leave your machine
- Built-in blocklists are embedded in the binary at build time
- No telemetry, no accounts, no internet required to function
- Config stored locally: `%APPDATA%\K9WebProtection\` (Windows) or `~/.k9webprotection/` (macOS)

---

## If your internet stops after enabling

**Windows** — run in PowerShell:
```powershell
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyEnable /t REG_DWORD /d 0 /f
```
Or: **Control Panel → Internet Options → Connections → LAN settings → uncheck Use a proxy server**

**macOS** — run in Terminal:
```bash
networksetup -listallnetworkservices | grep -v "An asterisk" | grep -v "^$" | while read svc; do
  networksetup -setwebproxystate "$svc" off
  networksetup -setsecurewebproxystate "$svc" off
done
```
Or: **System Settings → Network → Details → Proxies → uncheck Web Proxy and Secure Web Proxy**

---

## Build from source

**Windows:**
```powershell
# Requires Go, Node.js, Wails, and NSIS — see windows/README.md
powershell -ExecutionPolicy Bypass -File .\windows\build.ps1
```

**macOS:**
```bash
# Requires Go and Wails
cd mac/app
wails build
cd ..
sudo bash install.sh
```

---

## Known issues

- **Port conflict** — if something else uses port 8080, the proxy will fail to start. Change the port in Settings.
- **Unsigned binary (macOS)** — Gatekeeper warning on first launch. Follow Step 2 above.
- **Unsigned installer (Windows)** — SmartScreen warning until publisher certificate is verified. Application submitted.

---

## Documentation

- [Windows Full Guide](https://github.com/khaleel-git/K9-Web-Protection/blob/master/windows/README.md)
- [macOS Full Guide](https://github.com/khaleel-git/K9-Web-Protection/blob/master/mac/README.md)

---

## License

Open source — free for personal and community use.

**Disclaimer:** K9 is a tool, not a complete solution. It works best combined with intention, accountability, and community support.

---

**Windows:** v2.0.0 - Go + Wails + NSIS
**macOS:** v2.0.0 - Go + Wails
**Support:** hello@khaleel.eu
**Updated:** May 2026
