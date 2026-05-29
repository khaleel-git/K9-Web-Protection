# 🛡️ K9 Web Protection

A free, open-source content filtering tool built for the community — by the community. K9 helps you stay sober from pornography and harmful content by filtering it at the OS level, across every browser and app.

> *"Inspired by the original K9 Web Protection. We are giving back to the internet community."*

---

## What it does

K9 blocks adult content, harmful websites, and configurable keywords at two independent system levels so switching browsers, using incognito mode, or trying to bypass it via other apps doesn't work.

| Layer | How | Scope |
|-------|-----|-------|
| **Layer 1 — Hosts** | Writes to `/etc/hosts` | Blocks domains for every app on the system, even without internet |
| **Layer 2 — Proxy** | Local HTTPS proxy on `127.0.0.1:8080` | Blocks URLs, keywords, image search, YouTube, and the built-in adult database |

Both layers activate together when you click **Enable Protection**.

---

## Platforms

| Platform | Status | Tech Stack |
|----------|--------|------------|
| **macOS** | ✅ Active — v2.0.0 | Go + Wails (native desktop app) |
| **Windows** | 🔧 v1.1 (Python) | Python + PyInstaller |

---

## macOS — v2.0.0

The Mac app is a fully native desktop application built with Go and Wails. No Python, no screenshots, no screen recording permission required.

### Features

- **Content Blocking** — master toggle for the built-in adult database (thousands of domains + URL patterns + keywords)
- **Block image & video search** — blocks Google Images, Bing Images, and similar
- **Block YouTube** — full YouTube block with a single toggle
- **Safe Search enforcement** — forces safe search on Google, Bing, and DuckDuckGo
- **Block List / Allow List** — add your own domains on top of the built-in database
- **Keywords** — block any URL containing specific words or phrases
- **Focus Mode** — timed lock (30 min → 8 hr) that prevents disabling during a session
- **Disable Delay** — require a waiting period (1–48 hrs) before protection can be turned off
- **Accountability Partner** — store a partner's email for accountability
- **Password Protection** — password-gate all destructive actions
- **Watchdog** — a separate LaunchAgent re-locks files and re-enables the proxy every 10 seconds

### Quick start

```bash
# Build from source
cd mac/app
./setup-dev.sh      # install Go, Wails, npm deps (once)
wails build         # produces build/bin/K9 Web Protection.app

# Install
cd ..
sudo bash install.sh
```

📖 **[Full macOS guide →](mac/README.md)**

---

## Windows — v1.1

The Windows version uses a Python-based approach with NudeNet AI vision for visual detection.

📖 **[Full Windows guide →](windows/README.md)**

---

## Project structure

```
K9-Web-Protection/
├── mac/                              # macOS (v2.0.0 — Go + Wails)
│   ├── app/                          # Wails application source
│   │   ├── main.go                   # App entry point & window config
│   │   ├── app.go                    # All Go ↔ frontend bindings
│   │   ├── internal/
│   │   │   ├── config/config.go      # Settings, blocklists, stats (JSON)
│   │   │   ├── proxy/proxy.go        # Layer 2: HTTPS proxy engine
│   │   │   ├── hosts/hosts.go        # Layer 1: /etc/hosts management
│   │   │   └── database/             # Built-in blocklists (embedded)
│   │   └── frontend/                 # Vite + vanilla JS UI
│   ├── domains.json                  # Built-in domain blocklist
│   ├── urls.json                     # Built-in URL pattern blocklist
│   ├── multi-words.json              # Built-in keyword list
│   ├── install.sh                    # One-step installer for end users
│   ├── com.k9webprotection.plist     # LaunchAgent (app)
│   ├── com.k9webprotection.watchdog.plist # LaunchAgent (watchdog)
│   ├── k9_watchdog.sh                # Watchdog script
│   └── README.md                     # macOS documentation
├── windows/                          # Windows (v1.1 — Python)
│   ├── main.py                       # Windows core engine
│   ├── k9.bat                        # Self-healing watchdog
│   ├── k9-launcher.vbs               # Stealth background launcher
│   ├── domains.json                  # Windows blocklist database
│   ├── urls.json                     # Windows URL pattern database
│   ├── multi-words.json              # Windows keyword triggers
│   └── README.md                     # Windows documentation
└── lists/                            # Shared master lists
    ├── Keywords/                     # Keyword sources
    └── Urls/                         # URL sources
```

---

## Why no screenshots

The original v1.x used screenshots + OCR + NudeNet AI vision to detect content. This worked but caused two major problems:

1. While the script was running, **macOS blocked the user from taking their own screenshots or recording video** (Screen Recording permission conflict)
2. Taking a full screenshot every second was heavy on CPU and RAM

v2.0 (Mac) solves this by operating at the **network level** instead:

- The HTTPS proxy intercepts requests before content reaches the screen
- No Screen Recording permission needed
- Runs at near-zero CPU when nothing is being blocked

---

## Tech stack

| Component | macOS v2.0 | Windows v1.1 |
|-----------|-----------|--------------|
| **UI** | Wails (Go + WebView) | — (background only) |
| **Proxy** | Custom Go HTTP/HTTPS proxy | Python `mitmproxy` |
| **Blocklists** | Embedded JSON (Go `embed.FS`) | JSON files |
| **AI vision** | *(Phase 2)* | NudeNet + ONNX Runtime |
| **OS integration** | `networksetup`, `/etc/hosts`, osascript | pywin32 + Win32 API |
| **Persistence** | LaunchAgent + Watchdog + `uchg` locks | Registry `Run` key + Batch watchdog |
| **Build** | `wails build` → `.app` + DMG | PyInstaller → `.exe` |

---

## Privacy

K9 processes everything **100% locally**. No data, URLs, screenshots, or statistics leave your machine. The built-in blocklists are embedded in the binary at build time.

---

## Contributing

PRs are welcome. The most impactful areas:

- **Blocklist improvements** — add domains/URLs/keywords to `mac/domains.json`, `mac/urls.json`, `mac/multi-words.json`
- **Windows v2** — port the Go + Wails approach to Windows (replace the Python version)
- **HTTPS inspection** — implement TLS interception in the Go proxy for full URL-level HTTPS blocking

---

## License

Open Source — free for personal and community use.

---

**macOS Version:** 2.0.0 | **Windows Version:** 1.1 | **Updated:** May 2026
