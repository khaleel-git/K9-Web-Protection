# K9 Web Protection

A free, open-source web filter that blocks adult content at the OS level, across every browser and app.

**Support:** [hello@khaleel.eu](mailto:hello@khaleel.eu)

---

## How it works

Two independent layers of protection run simultaneously:

| Layer | Mechanism | Blocks |
|-------|-----------|--------|
| Layer 1 - Hosts | System hosts file | Domains, system-wide, even offline |
| Layer 2 - Proxy | Local proxy on `127.0.0.1:8080` | URLs, keywords, image search, YouTube |

---

## Platforms

| Platform | Version | Stack |
|----------|---------|-------|
| macOS | v2.0.0 | Go + Wails |
| Windows | v2.0.0 | Go + Wails + NSIS |

---

## Default password

```
k9.khaleel.eu
```

Change it in **Settings → Uninstall Protection** after first launch.

---

## macOS

Native desktop app built with Go and Wails.

**Features:** content blocking, image/video search blocking, YouTube block, Safe Search enforcement, custom block/allow lists, keywords, Focus Mode, Disable Delay, password protection, watchdog auto-restart.

```bash
cd mac/app
wails build

cd ..
sudo bash install.sh
```

[Full macOS guide](mac/README.md)

---

## Windows

Native desktop app built with Go and Wails, packaged as a standard Windows setup wizard.

**Features:** content blocking, image/video search blocking, YouTube block, Safe Search enforcement, custom block/allow lists, keywords, password protection, auto-start for all users.

```powershell
# Run as Administrator
powershell -ExecutionPolicy Bypass -File .\windows\build.ps1
```

Or download `K9WebProtection-setup.exe` from [Releases](../../releases).

[Full Windows guide](windows/README.md)

---

## Project structure

```
K9-Web-Protection/
├── mac/
│   ├── app/                    # Go + Wails source
│   │   ├── main.go
│   │   ├── app.go
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   ├── proxy/
│   │   │   ├── hosts/
│   │   │   └── database/       # Embedded blocklists
│   │   └── frontend/           # Vite + JS UI
│   ├── install.sh
│   ├── com.k9webprotection.plist
│   ├── com.k9webprotection.watchdog.plist
│   └── k9_watchdog.sh
├── windows/
│   ├── app/                    # Go + Wails source
│   │   ├── main.go
│   │   ├── app.go
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   ├── proxy/
│   │   │   ├── hosts/
│   │   │   └── database/       # Embedded blocklists
│   │   ├── frontend/           # Vite + JS UI
│   │   └── build/windows/
│   │       ├── icon.ico
│   │       ├── app.manifest    # UAC requireAdministrator
│   │       └── installer/
│   │           └── project.nsi
│   └── build.ps1
└── lists/                      # Source blocklist files
```

---

## Privacy

Everything runs 100% locally. No data, URLs, or statistics leave your machine. Blocklists are embedded in the binary at build time.

---

## Contributing

PRs welcome. Most impactful areas:

- **Blocklist improvements** - add domains, URLs, or keywords to the `database/` files
- **HTTPS inspection** - TLS interception in the Go proxy for full HTTPS URL blocking

---

## License

Open Source - free for personal and community use.

---

v2.0.0 - macOS and Windows | [hello@khaleel.eu](mailto:hello@khaleel.eu)
