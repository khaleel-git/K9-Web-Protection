# K9 Web Protection

A free, open-source parental control and web filter for macOS. Blocks adult content, malware, and distracting websites at the network level, across every browser, without a subscription.

**Contact:** [hello@khaleel.eu](mailto:hello@khaleel.eu)

> **This repository is no longer actively updated.**
> It is kept public as the open-source foundation that **BlockerFlow** was built on. The code here stays available, but I am not currently publishing new changes to it.
>
> **Need the latest BlockerFlow code?** The current source is not published here, but you are welcome to **request it from me directly** at [hello@khaleel.eu](mailto:hello@khaleel.eu).

---

## Try BlockerFlow: the app built from this project

This repo was used to create **[BlockerFlow](https://blockerflow.khaleel.eu/)**, a mobile app that blocks distracting and harmful content so you can stay focused.

| | |
|---|---|
| **Website** | [blockerflow.khaleel.eu](https://blockerflow.khaleel.eu/) |
| **Android (Google Play)** | [Get BlockerFlow on Google Play](https://play.google.com/store/apps/details?id=eu.khaleel.blockerflow) |

If you want protection and focus on your phone, with ongoing updates and support, use BlockerFlow. This repository was the starting point, and BlockerFlow is where I keep working.

[![Get it on Google Play](https://img.shields.io/badge/Google%20Play-Get%20BlockerFlow-3DDC84?logo=googleplay&logoColor=white)](https://play.google.com/store/apps/details?id=eu.khaleel.blockerflow)

---

## Blocklists

The pornography and proxy-avoidance lists in the Mac and Windows apps come from BlockerFlow's public lists, [khaleel-git/blockerflow-lists](https://github.com/khaleel-git/blockerflow-lists), pinned to a signed version (see [lists/README.md](lists/README.md)). Their licence terms (GPL-3.0 for pornography, CC BY-SA 4.0 for proxy-avoidance) are in [lists/blockerflow-lists/](lists/blockerflow-lists/README.md). This is a list refresh only: the repository is still not actively updated otherwise.

---

## Support

This project is free and open source. If it helped you, the best way to support my work is to **install [BlockerFlow](https://play.google.com/store/apps/details?id=eu.khaleel.blockerflow)** and share it with others.

> **[Donate via PayPal](https://www.paypal.com/paypalme/Khaleeleu)**: optional, and always appreciated

---

## How it works

K9 Web Protection installs a local HTTP/HTTPS proxy on your Mac and sets it as the system proxy. Every browser request (Safari, Chrome, Firefox, any app) passes through it before reaching the internet. Blocked sites never load; the user sees a clean block page instead.

Three independent layers work together:

### Layer 1: Content Proxy
A local proxy runs on `127.0.0.1:8080` and intercepts all HTTP and HTTPS traffic. It checks every request against:
- A built-in database of **932,000+ domains** across 29 categories (pornography, malware, phishing, gambling, hacking, P2P, proxy bypass, violence, drugs, weapons, social media, and more)
- User-defined block and allow lists
- URL keyword rules (block any URL containing a word or phrase)
- Focus Mode site list

For HTTPS, the proxy uses a CONNECT tunnel to see the hostname and block by domain. A built-in TLS MITM layer issues per-host certificates (signed by a locally-generated CA) to deliver the block page directly inside the HTTPS connection rather than showing a browser error.

### Layer 2: QUIC Firewall
Modern browsers (especially Chrome) use **HTTP/3 over UDP port 443**: a protocol called QUIC. Because QUIC runs over UDP rather than TCP, it bypasses the system HTTP proxy entirely and the content proxy never sees the request.

K9 installs a **PF (Packet Filter) firewall rule** at the macOS kernel level that drops all outbound UDP traffic on port 443. This forces browsers to fall back to TCP, where the content proxy can intercept and block them normally.

```
# /etc/pf.anchors/k10webprotection
block drop out quick proto udp to any port 443
```

The rule is wired into `/etc/pf.conf` and reloaded at each boot. A watchdog re-applies it every 10 seconds if it gets flushed.

### Layer 3: SafeSearch Enforcement
For Google and Bing, K9 performs full **HTTPS MITM interception** to:
- Inject `&safe=active` into every search request, forcing strict SafeSearch regardless of the user's account settings
- Block the SafeSearch preferences page so it cannot be turned off
- Redirect SafeSearch IPs via `/etc/hosts` as a secondary enforcement layer

---

## Tamper Resistance

K9 is designed to stay running even if someone tries to stop it:

| Mechanism | What it does |
|-----------|-------------|
| **LaunchAgent** (`KeepAlive: Crashed`) | Restarts the app automatically after a force-quit |
| **Watchdog** (runs every 10s) | Re-applies `uchg` file locks, re-bootstraps the LaunchAgent, re-enables the system proxy if disabled, re-applies PF rules if flushed |
| **`uchg` immutable flag** | Locks the binary and LaunchAgent plists so they cannot be deleted without root and an explicit unlock |
| **Password gate** | Every change (disable, uninstall, filter settings, password) requires the admin password |

---

## Features at a Glance

| Feature | Description |
|---------|-------------|
| **Content filtering** | 29 categories, 932k+ domains, four preset levels (Minimal / Moderate / Default / High) + Custom |
| **Focus Mode** | Block social media on demand with a countdown timer |
| **Time Restrictions** | Per-day schedules, block outside allowed hours (e.g. 08:00 to 22:00) |
| **SafeSearch** | Force Google and Bing into strict SafeSearch mode |
| **Website exceptions** | Per-domain allow and block overrides |
| **URL keywords** | Block any URL containing a word or phrase |
| **Activity log** | Every blocked request logged with domain, category, and timestamp |
| **In-app uninstaller** | One-click complete removal (app, services, firewall rules, config) |

---

## Platforms

| Platform | Status | Stack |
|----------|--------|-------|
| **macOS** | v1.0.0 | Go + Wails, `.pkg` installer |
| **Chrome extension** | Live on the Chrome Web Store (search "K9 Web Protection") | Manifest V3, source in [chrome-extension/](chrome-extension/) |
| Windows | In progress | Go + Wails + NSIS |

---

## Default password

```
k9.khaleel.eu
```

Change it immediately after install: **Setup → Password & Settings → New Password**

---

## macOS: Quick Start

Download `K10WebProtection-1.0.0.pkg` from [Releases](../../releases) and double-click it.

For build-from-source instructions: [mac/README.md](mac/README.md)

---

## Project structure

```
K9-Web-Protection_BlockerFlow/
├── mac/
│   ├── app/                        # Go + Wails source
│   │   ├── app.go                  # Business logic & bindings
│   │   ├── internal/
│   │   │   ├── config/             # Persistent config
│   │   │   ├── proxy/              # HTTP/HTTPS proxy + MITM
│   │   │   ├── hosts/              # /etc/hosts SafeSearch IPs
│   │   │   └── database/           # Embedded blocklists (932k+ domains)
│   │   └── frontend/               # Vite + vanilla JS UI
│   ├── pkg-scripts/                # .pkg pre/postinstall scripts
│   ├── build-pkg.sh                # Builds distributable .pkg
│   ├── install.sh                  # Manual installer
│   ├── uninstall.sh                # Force uninstaller
│   ├── k10_watchdog.sh             # Tamper-resistance watchdog
│   └── com.k10webprotection*.plist # LaunchAgents
└── lists/                          # Blocklist source files + build scripts
    ├── sync.py                     # Fetch latest from upstream sources
    ├── build.py                    # Compile domains.json for the app
    └── categories/                 # Per-category domain/URL/keyword files
```

---

## Privacy

Everything runs **100% locally on your device**. No browsing history, blocked domains, or personal data is ever sent to any server. The blocklist database is embedded in the binary at build time and works fully offline.

---

## Contributing

This repository is no longer actively maintained, so PRs may not be reviewed promptly. You are free to fork it and build your own version. For the latest BlockerFlow code, [email me](mailto:hello@khaleel.eu).

---

## License

Open Source, free for personal and community use.

---

**v1.0.0** · macOS · [hello@khaleel.eu](mailto:hello@khaleel.eu)
