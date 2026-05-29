# K9 Web Protection — macOS

A free, open-source parental control and web filtering app for macOS. Built with Go and Wails, it runs silently in the background and protects against adult content, harmful websites, and configurable keyword matches.

**Download:** [k9.khaleel.eu](https://k9.khaleel.eu)

---

## How it works

K9 uses two independent layers of protection:

| Layer | Mechanism | What it blocks |
|-------|-----------|----------------|
| **Layer 1** | `/etc/hosts` entries | Domains — works for every app, even offline |
| **Layer 2** | Local HTTPS proxy (port 8080) | URLs, keywords, adult databases, image search, YouTube |

Both layers activate when you click **Enable Protection** in the app.

---

## Default password

The app ships with a default password so filters are protected from the moment it is installed:

```
k9.khaleel.eu
```

Change it in **Settings → Uninstall Protection** after first launch.

---

## Requirements

- macOS 12 Monterey or later
- Apple Silicon or Intel Mac
- No additional dependencies — everything is self-contained

---

## Build from source

```bash
# 1. Install Go and Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 2. Build
cd mac/app
wails build

# Output: mac/app/build/bin/K9 Web Protection.app
```

---

## Install

```bash
cd mac
sudo bash install.sh
```

The installer:
1. Copies `K9 Web Protection.app` to `/Applications`
2. Installs the watchdog script to `/usr/local/bin/k9_watchdog.sh`
3. Loads both LaunchAgents (app + watchdog) for the current user
4. Locks the binary and plists with the `uchg` immutable flag

---

## Installing on an unsigned Mac (no Apple Developer certificate)

This app is not signed with an Apple Developer certificate (costs €99/year). macOS Gatekeeper will block it on first launch. Follow these steps to install it anyway.

### Step 1 — Remove the quarantine flag

After running the installer, strip the quarantine attribute that macOS adds to downloaded files:

```bash
sudo xattr -cr "/Applications/K9 Web Protection.app"
```

The installer already does this automatically, but run it manually if you downloaded the DMG directly.

### Step 2 — Allow the app in System Settings

If macOS still blocks the app when you open it:

1. Open **System Settings → Privacy & Security**
2. Scroll down to the **Security** section
3. You will see: *"K9WebProtection was blocked because it is not from an identified developer"*
4. Click **Allow Anyway**
5. Open the app again — click **Open** in the confirmation dialog

> This only needs to be done once. After approval macOS remembers the choice.

### Step 3 — Verify it is running

After opening the app, confirm both layers are active on the Dashboard:
- **Layer 1 — Hosts**: Active
- **Layer 2 — Proxy**: Running

### Troubleshooting Gatekeeper

If System Settings does not show the "Allow Anyway" button, run this once to allow apps from anywhere (re-enables Gatekeeper manually after install):

```bash
# Temporarily disable Gatekeeper
sudo spctl --master-disable

# Install and open K9, then re-enable Gatekeeper
sudo spctl --master-enable
```

---

## Proxy — known issues & internet recovery

The Layer 2 proxy routes all HTTP/HTTPS traffic through `127.0.0.1:8080`. If the proxy process stops unexpectedly while the system proxy setting is still enabled, **all internet access will fail** with a "connection refused" error.

### Why this can happen

| Scenario | Result |
|----------|--------|
| App closes normally | Proxy setting cleared automatically ✓ |
| App force-killed (`kill -9`) | LaunchAgent restarts app within 12 s ✓ |
| App deleted from Applications | Proxy stays on, internet breaks ✗ |
| System crash / hard reboot | On next login app auto-starts and re-syncs ✓ |
| Port 8080 already in use | Proxy fails to bind, internet breaks ✗ |

### How to fix internet if it breaks

**Option 1 — Open K9 and disable protection:**
1. Open `K9 Web Protection.app`
2. Click **Disable Protection** (enter the password)
3. Internet is restored immediately

**Option 2 — Turn off proxy via System Settings:**
1. Open **System Settings → Network**
2. Select your active connection (Wi-Fi or Ethernet)
3. Click **Details → Proxies**
4. Uncheck **Web Proxy (HTTP)** and **Secure Web Proxy (HTTPS)**
5. Click OK

**Option 3 — Fix via Terminal (fastest):**

```bash
# Turn off proxy for all network services at once
networksetup -listallnetworkservices | grep -v "An asterisk" | grep -v "^$" | while read svc; do
  networksetup -setwebproxystate "$svc" off
  networksetup -setsecurewebproxystate "$svc" off
done
```

### Check whether the proxy is actually running

```bash
# Should show a process listening on port 8080
lsof -i :8080

# Or check via curl — a 403 response means K9 is running and blocking
curl -x http://127.0.0.1:8080 http://example.com -I --max-time 3
```

### If port 8080 is already used by another app

Change K9's port in **Settings → (not available in current UI — edit config directly)**:

```bash
# Edit config and change proxyPort
nano ~/.k9webprotection/config.json
# Change "proxyPort": 8080 → "proxyPort": 8181 (or any free port)
```

Then restart the app.

---

## Uninstall

Use the **Uninstall** button inside the app (**Settings → Danger Zone**). It requires the password and cleanly removes all components.

To manually force-remove:

```bash
# 1. Unlock files
sudo chflags -R nouchg "/Applications/K9 Web Protection.app"
sudo chflags nouchg /Library/LaunchAgents/com.k9webprotection.plist
sudo chflags nouchg /Library/LaunchAgents/com.k9webprotection.watchdog.plist

# 2. Stop services
RUID=$(id -u $(logname))
launchctl bootout gui/$RUID /Library/LaunchAgents/com.k9webprotection.plist
launchctl bootout gui/$RUID /Library/LaunchAgents/com.k9webprotection.watchdog.plist

# 3. Remove files
sudo rm -rf "/Applications/K9 Web Protection.app"
sudo rm -f /Library/LaunchAgents/com.k9webprotection.plist
sudo rm -f /Library/LaunchAgents/com.k9webprotection.watchdog.plist
sudo rm -f /usr/local/bin/k9_watchdog.sh
rm -rf ~/.k9webprotection

# 4. Clear system proxy
networksetup -listallnetworkservices | grep -v "An asterisk" | grep -v "^$" | while read svc; do
  networksetup -setwebproxystate "$svc" off
  networksetup -setsecurewebproxystate "$svc" off
done
```

---

## Project structure

```
mac/
├── app/                               # Go + Wails application
│   ├── main.go                        # App entry point
│   ├── app.go                         # Business logic & frontend bindings
│   ├── internal/
│   │   ├── config/config.go           # Persistent config (JSON)
│   │   ├── proxy/proxy.go             # Layer 2: HTTPS proxy
│   │   ├── hosts/hosts.go             # Layer 1: /etc/hosts management
│   │   └── database/                  # Built-in blocklists
│   └── frontend/                      # Vite + vanilla JS UI
├── com.k9webprotection.plist          # LaunchAgent — runs the app
├── com.k9webprotection.watchdog.plist # LaunchAgent — runs the watchdog
├── k9_watchdog.sh                     # Watchdog: re-locks files, re-enables proxy
├── install.sh                         # One-step installer
└── k9-web-protection-logo.webp        # App icon source
```

---

## Persistence model

The app is designed to be difficult to bypass:

- **LaunchAgent** (`KeepAlive: Crashed`) restarts the app automatically after force-kills
- **Watchdog** runs every 10 seconds: re-applies `uchg` locks, re-bootstraps the LaunchAgent if unloaded, re-enables the system proxy if disabled
- **`uchg` flag** on the binary and plists prevents deletion without root + explicit unlock
- **Password gate** on all destructive actions (disable, uninstall, change filters)

---

**Version:** 2.0.0 | **Platform:** macOS 12+ (Apple Silicon & Intel) | **License:** Open Source
