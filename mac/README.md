# K10 Web Protection — macOS

> Free, open-source parental control and web filtering for macOS.
> Blocks adult content, malware, and distracting websites — silently, persistently, and without a monthly subscription.

**Platform:** macOS 12 Monterey or later · Apple Silicon (M1/M2/M3/M4) and Intel

---

## Install

Download **`K10WebProtection-1.0.0.pkg`** from the [Releases](../../releases) page and double-click it.

The installer copies the app, registers the background services, and sets up all firewall rules automatically. No manual steps required.

### Gatekeeper warning

macOS will block the `.pkg` on first open with:

> *"K10WebProtection-1.0.0.pkg" Not Opened — Apple could not verify it is free of malware…*

This is expected for software not signed with a paid Apple Developer certificate. Fix it one of two ways:

**Option A — System Settings:**
1. Click **Done** on the warning dialog
2. Open **System Settings → Privacy & Security**
3. Scroll to the **Security** section → click **Open Anyway**
4. Enter your Mac password, then double-click the `.pkg` again

**Option B — Terminal:**
```bash
xattr -d com.apple.quarantine ~/Downloads/K10WebProtection-1.0.0.pkg
```
Then double-click the `.pkg` — no warning will appear.

---

## Default password

```
k9.khaleel.eu
```

Change it immediately: **Setup → Password & Settings → New Password**

---

## How it works

K10 runs a local HTTP/HTTPS proxy on port 8080 and sets it as the system proxy. Every browser request passes through it. Three independent layers work together:

| Layer | Mechanism | What it blocks |
|-------|-----------|----------------|
| **Content Proxy** | Local HTTP/HTTPS proxy (port 8080) | Categories, domains, URL patterns, keywords |
| **QUIC Firewall** | PF packet filter (UDP 443) | Chrome HTTP/3 bypass attempts |
| **MITM TLS** | Per-host certificate generation | Serves block pages over HTTPS; enforces SafeSearch |

---

## Features

- **932,000+ domain database** across 29 categories (pornography, malware, phishing, gambling, hacking, P2P, proxy bypass, violence, drugs, weapons, social media, and more)
- **Four filter levels**: Minimal → Moderate → Default → High, plus fully Custom
- **Focus Mode** — block social media sites on demand with a countdown timer
- **Time Restrictions** — per-day schedules (e.g. 08:00–22:00 Mon–Fri)
- **SafeSearch enforcement** on Google and Bing via HTTPS interception
- **Website exceptions** — per-domain allow and block overrides
- **URL keyword filtering** — block any URL containing a word or phrase
- **Activity log** with Top Blocked Categories chart
- **Password-protected** admin panel — all changes require authentication
- **Tamper-resistant** — LaunchAgent, watchdog, and immutable file flags keep it running

---

## Uninstall

**The correct way — use the in-app uninstaller:**

1. Open **K10 Web Protection**
2. Go to **Setup → Password & Settings**
3. Scroll to **Danger Zone** → click **Uninstall K10 Web Protection…**
4. Enter your admin password

The app closes and removes itself completely — app, LaunchAgents, firewall rules, system proxy, and config.

> **Do not drag the app to Trash.** K10 locks its own files to resist tampering. Dragging it will fail and may leave the system proxy enabled, breaking internet access.

**Force-remove if the app is already gone:**
```bash
sudo bash -c '
launchctl bootout gui/$(id -u) /Library/LaunchAgents/com.k10webprotection.watchdog.plist 2>/dev/null
launchctl bootout gui/$(id -u) /Library/LaunchAgents/com.k10webprotection.plist 2>/dev/null
chflags -R nouchg "/Applications/K10 Web Protection.app" 2>/dev/null
rm -rf "/Applications/K10 Web Protection.app"
rm -f /Library/LaunchAgents/com.k10webprotection.plist
rm -f /Library/LaunchAgents/com.k10webprotection.watchdog.plist
rm -f /usr/local/bin/k10_watchdog.sh
pfctl -a k10webprotection -F rules 2>/dev/null
sed -i "" "/k10webprotection/d" /etc/pf.conf 2>/dev/null
rm -f /etc/pf.anchors/k10webprotection
networksetup -listallnetworkservices | grep -v "An asterisk" | grep -v "^$" | while read svc; do
  networksetup -setwebproxystate "$svc" off
  networksetup -setsecurewebproxystate "$svc" off
done
rm -rf ~/.k10webprotection
'
```

---

## If internet breaks

If the proxy crashes while the system proxy is still active, all internet access fails.

**Quickest fix — Terminal:**
```bash
networksetup -listallnetworkservices | grep -v "An asterisk" | grep -v "^$" | while read svc; do
  networksetup -setwebproxystate "$svc" off
  networksetup -setsecurewebproxystate "$svc" off
done
```

**Or via System Settings:** Network → select your connection → Details → Proxies → uncheck Web Proxy (HTTP) and Secure Web Proxy (HTTPS).

---

## Build from source

### Prerequisites

```bash
brew install go
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Build the distributable .pkg

```bash
cd mac
bash build-pkg.sh
# Output: mac/K10WebProtection-1.0.0.pkg
```

For a universal binary (Apple Silicon + Intel):
```bash
bash build-pkg.sh --universal
```

### Dev mode (hot reload)

```bash
cd mac/app
wails dev
```

### Update the blocklist database

```bash
cd lists
python3 sync.py      # fetch latest from upstream sources
python3 build.py     # rebuild domains.json
cd ../mac/app
wails build          # re-embed the updated database
```

---

## Project structure

```
mac/
├── app/                                # Go + Wails application
│   ├── app.go                          # Business logic & frontend bindings
│   ├── shutdown_darwin.go              # macOS shutdown/restart handling
│   ├── internal/
│   │   ├── config/config.go            # Persistent config (~/.k10webprotection/)
│   │   ├── proxy/
│   │   │   ├── proxy.go                # HTTP/HTTPS proxy + block logic
│   │   │   ├── mitm.go                 # TLS MITM — block pages + SafeSearch
│   │   │   └── categories.go           # Filter levels, bypass detection
│   │   ├── database/                   # Embedded blocklists (29 categories, 932k+ domains)
│   │   └── hosts/hosts.go              # /etc/hosts management (SafeSearch IPs)
│   └── frontend/                       # Vite + vanilla JS UI
├── pkg-scripts/
│   ├── preinstall                      # Stops services before upgrade
│   └── postinstall                     # Configures system after install
├── pkg-resources/
│   ├── distribution.xml                # macOS Installer UI
│   └── welcome.html
├── com.k10webprotection.plist          # LaunchAgent — auto-starts the app
├── com.k10webprotection.watchdog.plist # LaunchAgent — runs the watchdog
├── k10_watchdog.sh                     # Re-locks files, re-enables proxy, re-applies PF
├── build-pkg.sh                        # Builds the distributable .pkg
├── install.sh                          # Manual installer (dev/source builds)
└── uninstall.sh                        # Standalone uninstaller script
```

---

## Persistence model

| Mechanism | Purpose |
|-----------|---------|
| LaunchAgent (`KeepAlive: Crashed`) | Restarts app after force-kill |
| Watchdog (every 10s) | Re-applies `uchg` locks, re-bootstraps LaunchAgent, re-enables system proxy |
| `uchg` flag | Prevents deletion of binary and plists without root + explicit unlock |
| PF firewall rule | Blocks QUIC (UDP 443) — survives reboots via `/etc/pf.conf` |
| Password gate | Required for disable, uninstall, and all settings changes |

---

**Version:** 1.0.0 | **Platform:** macOS 12+ (Apple Silicon & Intel) | **License:** Open Source
