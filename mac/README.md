# K10 Web Protection — macOS

A free, open-source parental control and web filtering app for macOS. Built with Go and Wails, it runs silently in the background and protects against adult content, harmful websites, and configurable keyword matches.

---

## How it works

K10 uses two independent layers of protection:

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

Change it in **Settings → Password** after first launch.

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

# Output: mac/app/build/bin/K10 Web Protection.app
```

---

## Install

```bash
cd mac
bash install.sh
```

The installer:
1. Copies `K10 Web Protection.app` to `/Applications`
2. Installs the watchdog script to `/usr/local/bin/k10_watchdog.sh`
3. Loads both LaunchAgents (app + watchdog) for the current user
4. Locks the binary and plists with the `uchg` immutable flag

---

## Installing on an unsigned Mac (no Apple Developer certificate)

This app is not signed with an Apple Developer certificate. macOS Gatekeeper will block it on first launch.

### Step 1 — Remove the quarantine flag

The installer does this automatically via `xattr -cr`. If needed manually:

```bash
sudo xattr -cr "/Applications/K10 Web Protection.app"
```

### Step 2 — Allow the app in System Settings

If macOS still blocks the app:

1. Open **System Settings → Privacy & Security**
2. Scroll to the **Security** section
3. You will see: *"K10WebProtection was blocked because it is not from an identified developer"*
4. Click **Allow Anyway**
5. Open the app again and click **Open**

> This only needs to be done once.

### Step 3 — Verify it is running

After opening the app, confirm both layers are active on the Dashboard:
- **Layer 1 — Hosts**: Active
- **Layer 2 — Proxy**: Running

---

## Proxy — known issues & internet recovery

The Layer 2 proxy routes all HTTP/HTTPS traffic through `127.0.0.1:8080`. If the proxy process stops unexpectedly while the system proxy setting is still enabled, **all internet access will fail**.

### How to fix internet if it breaks

**Option 1 — Open K10 and disable protection:**
1. Open `K10 Web Protection.app`
2. Click **Disable Protection** (enter the password)

**Option 2 — Turn off proxy via System Settings:**
1. Open **System Settings → Network → Details → Proxies**
2. Uncheck **Web Proxy (HTTP)** and **Secure Web Proxy (HTTPS)**

**Option 3 — Terminal (fastest):**

```bash
networksetup -listallnetworkservices | grep -v "An asterisk" | grep -v "^$" | while read svc; do
  networksetup -setwebproxystate "$svc" off
  networksetup -setsecurewebproxystate "$svc" off
done
```

### Check whether the proxy is running

```bash
lsof -i :8080
curl -x http://127.0.0.1:8080 http://example.com -I --max-time 3
```

### Change the proxy port

```bash
nano ~/.k10webprotection/config.json
# Change "proxyPort": 8080 to any free port, then restart the app
```

---

## Uninstall

Use the **Uninstall** button inside the app. It requires the password and removes all components cleanly.

To manually force-remove:

```bash
# 1. Unlock files
sudo chflags -R nouchg "/Applications/K10 Web Protection.app"
sudo chflags nouchg /Library/LaunchAgents/com.k10webprotection.plist
sudo chflags nouchg /Library/LaunchAgents/com.k10webprotection.watchdog.plist

# 2. Stop services
RUID=$(id -u $(logname))
launchctl bootout gui/$RUID /Library/LaunchAgents/com.k10webprotection.plist
launchctl bootout gui/$RUID /Library/LaunchAgents/com.k10webprotection.watchdog.plist

# 3. Remove files
sudo rm -rf "/Applications/K10 Web Protection.app"
sudo rm -f /Library/LaunchAgents/com.k10webprotection.plist
sudo rm -f /Library/LaunchAgents/com.k10webprotection.watchdog.plist
sudo rm -f /usr/local/bin/k10_watchdog.sh
rm -rf ~/.k10webprotection

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
│   │   ├── config/config.go           # Persistent config (~/.k10webprotection/)
│   │   ├── proxy/proxy.go             # Layer 2: HTTPS proxy
│   │   ├── hosts/hosts.go             # Layer 1: /etc/hosts management
│   │   └── database/                  # Built-in blocklists (29 categories)
│   └── frontend/                      # Vite + vanilla JS UI
├── com.k10webprotection.plist          # LaunchAgent — runs the app
├── com.k10webprotection.watchdog.plist # LaunchAgent — runs the watchdog
├── k10_watchdog.sh                     # Watchdog: re-locks files, re-enables proxy
└── install.sh                          # One-step installer
```

---

## Persistence model

- **LaunchAgent** (`KeepAlive: Crashed`) restarts the app after force-kills
- **Watchdog** runs every 10 seconds: re-applies `uchg` locks, re-bootstraps the LaunchAgent, re-enables the system proxy if disabled
- **`uchg` flag** on the binary and plists prevents deletion without root + explicit unlock
- **Password gate** on all destructive actions (disable, uninstall, change filters)

---

**Version:** 2.0.0 | **Platform:** macOS 12+ (Apple Silicon & Intel) | **License:** Open Source
