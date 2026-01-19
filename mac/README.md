# K9 Web Protection for macOS (Persistent AI Blocker)

A high-security, AI-powered porn blocker for macOS. This version implements **Integrity Enforcement** and **Circular Persistence** to ensure the protection is virtually impossible to bypass, uninstall, or terminate.

---

## 🔒 Persistence & Integrity Architecture

The system uses a triple-layered defense mechanism to maintain "Unstoppable" status:

1. **Circular Watchdog**: Two separate LaunchAgents monitor each other. If the AI Engine is killed, the Watchdog restarts it. If the Watchdog is stopped, macOS restarts it.
2. **Integrity Enforcer (`k9_watchdog.sh`)**: A high-level background script that continuously monitors the binary and the service, re-applying `uchg` (immutable) flags every 10 seconds.
3. **File Immutability**: System-level locks make core files undeletable even by Admin users.

---

## 🚀 Deployment & Persistence

### 1. The Watchdog Script

**Path:** `/usr/local/bin/k9_watchdog.sh`

This script ensures the `uchg` (immutable) flags are active and the service is bootstrapped.

```bash
#!/bin/bash
# K9 Integrity Enforcer
BINARY="/Applications/K9 Web Protection.app/Contents/MacOS/K9 Web Protection"
PLIST="/Library/LaunchAgents/com.k9webprotection.plist"

while true; do
    # Re-apply immutable locks if removed
    if [[ $(ls -lO "$BINARY" | grep -c "uchg") -eq 0 ]]; then
        sudo chflags uchg "$BINARY"
    fi
    if [[ $(ls -lO "$PLIST" | grep -c "uchg") -eq 0 ]]; then
        sudo chflags uchg "$PLIST"
    fi

    # Ensure the AI Engine Service is LOADED
    if ! launchctl list | grep -q "com.k9webprotection"; then
        launchctl bootstrap gui/$(id -u) "$PLIST" 2>/dev/null
    fi
    sleep 10
done

```

### 2. The Persistence Services

**Path:** `/Library/LaunchAgents/`

| File | Purpose |
| --- | --- |
| **`com.k9webprotection.plist`** | Runs the primary AI Engine binary with high priority (`Nice -20`). |
| **`com.k9webprotection.watchdog.plist`** | Ensures the Watchdog script is always running. |

---

## 🛡️ Activating "Unstoppable" Mode

Once files are in their respective paths, run the following commands to lock the system:

```bash
# Apply immutable locks
sudo chflags uchg "/Applications/K9 Web Protection.app"
sudo chflags uchg "/Library/LaunchAgents/com.k9webprotection.plist"
sudo chflags uchg "/Library/LaunchAgents/com.k9webprotection.watchdog.plist"
sudo chflags uchg "/usr/local/bin/k9_watchdog.sh"

```

---

## 🛠 Maintenance & Removal

Standard uninstallation will fail. To perform maintenance or updates:

1. **Unlock Files**:
`sudo chflags nouchg /Applications/K9\ Web\ Protection.app`
(Repeat for `.plist` and `.sh` files).
2. **Unload Services**:
`launchctl bootout gui/$(id -u) /Library/LaunchAgents/com.k9webprotection.watchdog.plist`
3. **Terminate**:
`killall -9 "K9 Web Protection"`

---

## 📂 Project Structure (Source)

```text
mac/
├── main.py                          # Core AI Monitoring Engine
├── k9_watchdog.sh                   # Installed to /usr/local/bin/
├── com.k9webprotection.plist        # Installed to /Library/LaunchAgents/
├── com.k9webprotection.watchdog.plist # Installed to /Library/LaunchAgents/
├── domains.json                     # Hard-block database
├── urls.json                        # URL pattern database
├── multi-words.json                 # AI-trigger keywords
├── features/                        # Modular detection features
└── Heaven_Icon.icns                 # App branding icon

```

---

## ⚠️ Requirements

* **Accessibility Permissions**: Must be enabled for `K9 Web Protection` in *System Settings > Privacy & Security*.
* **Tesseract OCR**: Must be installed via Homebrew (`brew install tesseract`).

---

**Version**: 1.1 | **Updated**: January 2026 | **Platform**: macOS (Apple Silicon & Intel)