# K10 Web Protection — v1.0.0 for macOS

> **Free, open-source parental control and web filtering — built exclusively for macOS.**
> Blocks adult content, malware, and distracting websites silently and persistently in the background, without a monthly subscription.
>
> **Platform:** macOS 12 Monterey or later · Apple Silicon (M1/M2/M3/M4) and Intel

---

## Support This Project

K10 Web Protection is **free and open source**. To distribute it without Gatekeeper warnings, an Apple Developer ID is required — this costs **€99/year**.

> 💛 **[Donate via PayPal](https://www.paypal.com/paypalme/Khaleeleu)** — any amount appreciated
>
> 🍎 **Goal:** Raise €99 for an Apple Developer ID so future releases install without security warnings

---

## ⚠️ Important — Read Before Installing

### Gatekeeper Warning (macOS Security)

This build is **not signed with an Apple Developer certificate**. When you download and open the `.pkg`, macOS will block it with a dialog:

> *"K10WebProtection-1.0.0.pkg" Not Opened — Apple could not verify it is free of malware…*

**This is normal.** It is a standard macOS requirement for all software distributed outside the App Store. K10 Web Protection is open source — you can inspect every line of code in this repository.

---

### How to open it — two options

> **Note:** On macOS 13 Ventura and later, right-clicking → Open no longer bypasses this warning. Use one of the methods below.

---

**Option A — System Settings (no Terminal required)**

1. Click **Done** to close the warning (do not click "Move to Bin")
2. Open **System Settings** → **Privacy & Security**
3. Scroll down to the **Security** section
4. You will see: *"K10WebProtection-1.0.0.pkg was blocked"*
5. Click **Open Anyway** and enter your Mac password
6. Double-click the `.pkg` again — the installer opens normally

---

**Option B — Terminal (one command, fastest)**

Open Terminal (search "Terminal" in Spotlight) and paste:

```bash
xattr -d com.apple.quarantine ~/Downloads/K10WebProtection-1.0.0.pkg
```

Then double-click the `.pkg` — no warning will appear.

---

This is a one-time step. Once installed, the app runs silently with no further warnings.

---

## ⚠️ Important — How to Safely Remove K10

K10 Web Protection is designed to be **tamper-resistant** — it locks its own files to prevent easy removal. Because of this, **do not** try to drag the app to Trash. That will fail and may leave the system proxy enabled, breaking your internet.

**The correct way to uninstall:**

1. Open **K10 Web Protection**
2. Go to **Setup → Password & Settings**
3. Scroll to **Danger Zone**
4. Click **Uninstall K10 Web Protection…**
5. Enter your admin password — the app closes and removes itself completely

If K10 is already gone from `/Applications` and you need to force-clean:

```bash
# Run this in Terminal (will ask for your password)
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

## Download & Install

1. Download **`K10WebProtection-1.0.0.pkg`** from the release assets below
2. Right-click → **Open** → **Open** (see Gatekeeper warning above)
3. Follow the installer — enter your Mac password when prompted
4. K10 Web Protection opens automatically when done

---

## Default Admin Password

```
k9.khaleel.eu
```

Change it immediately after install: **Setup → Password & Settings → New Password**

---

## What It Does

K10 Web Protection sits silently between your browser and the internet. Every website request passes through it. Blocked sites never load — the user sees a clean block page instead.

It works across **Safari, Chrome, Firefox, and every other browser** because it operates at the network layer, not as a browser extension.

### Three layers of protection

| Layer | Technology | What it catches |
|-------|-----------|-----------------|
| **Content Proxy** | Local HTTP/HTTPS proxy on port 8080 | All web requests — blocks by category, domain, URL pattern, or keyword |
| **QUIC Firewall** | PF packet filter rule | Chrome's HTTP/3 (UDP) traffic that would otherwise bypass the proxy |
| **HTTPS Interception** | TLS MITM with per-host certificates | Delivers the block page over HTTPS; enforces SafeSearch on Google & Bing |

---

## Features

### Content Filtering
Block websites by category from a database of **932,000+ domains** across 29 categories:

| Category | Examples |
|----------|---------|
| Pornography | Adult video sites, escort sites |
| Adult / Mature | Mixed-adult content |
| Gambling | Online casinos, sports betting |
| Malware / Spyware | Known malware-hosting domains |
| Phishing | Fake banking, scam sites |
| Hacking | Exploit tools, warez |
| P2P / Piracy | Torrent sites |
| Proxy Bypass | VPN providers, web proxies, DoH servers |
| Violence / Hate | Extremist content |
| Illegal Drugs | Drug marketplaces |
| Weapons | Illegal firearms sites |
| Social Networking | Facebook, Twitter, TikTok, Reddit |
| Messaging / Chat | Chat platforms |
| Dating | Personals and dating sites |
| … and 15 more | Alcohol, Tobacco, Gambling, LGBT, etc. |

Choose a pre-set **Filter Level** or go fully custom:

| Level | What it blocks |
|-------|---------------|
| **Minimal** | Pornography + Malware + Phishing only |
| **Moderate** | Adds Gambling, Hacking, Drugs, Violence |
| **Default** | Adds Adult, Nudity, Alt. Sexuality, Dating, Proxy Bypass, and more |
| **High** | Everything — including Social Media, Forums, Image Search, YouTube |
| **Custom** | You choose exactly which categories to block |

---

### Focus Mode
Temporarily block distracting social media sites with one click. Built-in sites include Facebook, Instagram, Twitter/X, TikTok, YouTube, Reddit, Discord, Twitch, Snapchat, Pinterest, LinkedIn, Telegram, and more. Add your own custom sites anytime.

Set a **countdown timer** — Focus Mode automatically deactivates when time is up.

---

### Time Restrictions
Schedule when internet access is allowed. Set a daily time window (e.g. 08:00–22:00) for each day of the week independently. Outside those hours, all non-essential browsing is blocked.

- Supports overnight windows (e.g. 22:00 → 06:00)
- Enable or disable individual days
- Defaults: Mon–Fri 08:00–22:00, weekends configurable

---

### SafeSearch Enforcement
Forces Google and Bing into strict SafeSearch mode — even if someone is logged in to a Google account and has SafeSearch turned off. The SafeSearch preference screen is blocked so it cannot be changed.

---

### Website Exceptions
Override the filter for specific domains:
- **Allow list** — always let through (e.g. a school website that shares a domain with blocked content)
- **Block list** — always block, regardless of filter level (e.g. a specific game site)

---

### URL Keywords
Block any URL that contains a specific word or phrase — even on sites not in the database. Examples: `sex`, `nude`, `torrent`, `/adult/`. Applies to both HTTP and HTTPS connections.

---

### Activity Log
Every blocked request is recorded with the domain, category, and timestamp. View recent activity and a chart of your **Top Blocked Categories** to see what the filter is catching.

---

### Persistence & Tamper Resistance

K10 is designed to stay running even if someone tries to stop it:

- **LaunchAgent** restarts the app automatically after a force-quit
- **Watchdog process** re-enables the system proxy and re-applies firewall rules every 10 seconds if they are removed
- **Immutable flag (`uchg`)** on the binary and launch files prevents deletion without the admin password
- **Password gate** on all changes — disable, uninstall, change settings, and change the password all require the current admin password

---

## How to Use

### First Launch

After installation, K10 opens automatically. You'll see the **Dashboard**:

- **Blocked Today** — requests blocked since midnight
- **Total Blocked** — all-time count
- **Protection Modules** — status of each protection layer
- **Top Blocked Categories** — chart of what's being caught

### Changing the Filter Level

1. Go to **Setup → Web Categories to Block**
2. Select a level (Minimal / Moderate / Default / High / Custom)
3. Click **Save**

### Setting Up Time Restrictions

1. Go to **Setup → Time Restrictions**
2. Toggle **Enable Time Restrictions** on
3. Set a From/To time window for each day
4. Enable the days you want restricted
5. Click **Save**

### Starting Focus Mode

1. Click **Focus Mode** in the top navigation
2. Toggle the sites you want to block on/off
3. Optionally set a timer (30 min, 1 hour, 2 hours, etc.)
4. Click **Start Focus Mode**

### Adding a Custom Blocked Site

1. Go to **Setup → Web Site Exceptions**
2. Under **Block List**, type the domain (e.g. `reddit.com`)
3. Click **Add**

### Temporarily Disabling Protection

Click **Logout** → enter your password → protection pauses. Click **Enable Protection** to resume.

You can set a **Disable Delay** (Setup → Password & Settings) to require a waiting period before protection can be disabled — useful for preventing impulsive bypasses.

---

## Requirements

- macOS 12 Monterey or later
- Apple Silicon (M1/M2/M3/M4) or Intel Mac
- ~50 MB disk space
- No internet connection required after install (database is built-in)

---

## Uninstall

1. Open K10 Web Protection
2. Go to **Setup → Password & Settings**
3. Scroll to **Danger Zone**
4. Click **Uninstall K10 Web Protection…**
5. Enter your admin password

The app closes itself and removes everything: the application, launch agents, firewall rules, system proxy settings, and all configuration files.

---

## Troubleshooting

**Internet stopped working after a crash?**

Open System Settings → Network → select your connection → Details → Proxies → uncheck Web Proxy (HTTP) and Secure Web Proxy (HTTPS).

Or paste this into Terminal:
```bash
networksetup -listallnetworkservices | grep -v "An asterisk" | grep -v "^$" | while read svc; do
  networksetup -setwebproxystate "$svc" off
  networksetup -setsecurewebproxystate "$svc" off
done
```

**A site is blocked that shouldn't be?**

Go to Setup → Web Site Exceptions → Allow List → add the domain.

**A site is not being blocked?**

1. Check that the filter level includes the relevant category (Setup → Web Categories)
2. Add it manually: Setup → Web Site Exceptions → Block List

**The block page shows a certificate warning (`NET::ERR_CERT_AUTHORITY_INVALID`)?**

K10 installs its CA certificate automatically on first launch. If you still see this warning:
1. Go to **Setup → Safe Search → Install Certificate** and enter your Mac password
2. Fully quit and relaunch Chrome (`⌘Q`, not just close the window)
3. If the warning persists, open **Keychain Access** → **System** keychain → find "K10 Web Protection CA" → double-click → Trust → SSL → **Always Trust**

---

## Privacy

K10 Web Protection processes all traffic **locally on your Mac**. No browsing history, blocked sites, or personal data is sent to any server. The blocklist database ships with the app and is never uploaded or shared.

---

## Building from Source

```bash
# Prerequisites: Go 1.21+, Wails v2
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Build the .pkg installer
git clone https://github.com/your-username/K10-Web-Protection
cd K10-Web-Protection/mac
bash build-pkg.sh
```

See [README.md](README.md) for full build documentation.

---

## Changelog — v1.0.0

- Initial public release
- 932,000+ domain blocklist across 29 categories (Block List Project, StevenBlack, OISD NSFW, HaGeZi, URLhaus, UT1)
- Four preset filter levels (Minimal → High) plus fully custom mode
- Focus Mode with per-site toggles and countdown timer
- Time Restrictions with per-day schedules and overnight window support
- SafeSearch enforcement on Google and Bing via HTTPS interception
- QUIC/HTTP3 firewall rule — prevents Chrome from bypassing the proxy over UDP
- Complete in-app uninstaller
- Password-protected admin panel
- Activity log with Top Blocked Categories chart
- Tamper-resistant: LaunchAgent, watchdog, and immutable file flags

---

*K10 Web Protection is free and open source. Contributions welcome.*
