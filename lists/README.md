# K9 Web Protection — Blocklists

Category-based blocklist source data, compiled into the embedded Go database.

## Quick Start

```bash
# 1. Fetch latest from all upstream sources (requires internet)
python3 lists/sync.py --no-build

# 2. Compile into the embedded database (default level)
python3 lists/build.py --level default

# 3. Rebuild the Go binary to embed the new data
cd mac/app && go build ./...
```

---

## Directory Structure

```
lists/
  categories/          ← one folder per category
    pornography/
      domains.txt      ← one hostname per line  (full source list, no cap)
      keywords.txt     ← one phrase per line
    gambling/
      domains.txt
      keywords.txt
    ...
  levels/              ← JSON manifests — which categories each level activates
    minimal.json
    moderate.json
    default.json
    high.json
    monitor.json
    custom.json
  sync.py              ← fetches from upstream sources → updates categories/
  build.py             ← compiles categories/ → mac/app/internal/database/
  README.md            ← this file
```

---

## Levels

| Level    | Description                                             | Embedded domains |
|----------|---------------------------------------------------------|-----------------|
| **High**     | All default + chat, newsgroups, social, alcohol, etc.  | ~210k            |
| **Default**  | Adult content, security threats, illegal, sexually-related | **~200k** ✓  |
| **Moderate** | Adult content, security threats, illegal activity      | ~185k            |
| **Minimal**  | Pornography and security threats only                  | ~160k            |
| **Monitor**  | Allows all — logs traffic only (no blocking)           | 0                |
| **Custom**   | User-selected categories (stored in app config)        | —                |

---

## Sources (actively maintained)

### Pornography — `categories/pornography/`
| Source | Notes | Domains |
|--------|-------|---------|
| [Block List Project — porn](https://github.com/blocklistproject/Lists) | CI/CD updated, ~500k entries | 500k |
| [StevenBlack/hosts — porn-only](https://github.com/StevenBlack/hosts) | Widely used consolidated list | 77k |
| [OISD NSFW](https://oisd.nl) | Zero false-positives focus, updated hourly | 299k |
| UT1 adult | Université Toulouse 1 adult URLs | 1k |
| **Embedded (capped at 100k)** | Priority order: BLP → StevenBlack → OISD → UT1 | **100k** |

### Malware / Spyware — `categories/malware-spyware/`
| Source | Notes | Domains |
|--------|-------|---------|
| [Block List Project — malware](https://github.com/blocklistproject/Lists) | Community maintained | 435k |
| [URLhaus](https://urlhaus.abuse.ch) | abuse.ch, active malware URLs | ~500 |
| [HaGeZi TIF](https://github.com/hagezi/dns-blocklists) | Threat Intelligence Feeds, 6h updates | 1.2M |
| **Embedded (capped at 30k)** | | **30k** |

### Phishing — `categories/phishing/`
| Source | Notes | Domains |
|--------|-------|---------|
| [Block List Project — phishing](https://github.com/blocklistproject/Lists) | | 435k |
| [Block List Project — abuse](https://github.com/blocklistproject/Lists) | | 25k |
| [Block List Project — scam](https://github.com/blocklistproject/Lists) | | 1.3k |
| [UT1 — phishing](https://github.com/olbat/ut1-blacklists) | | 688k |
| **Embedded (capped at 30k)** | | **30k** |

### Gambling — `categories/gambling/`
| Source | Notes | Domains |
|--------|-------|---------|
| [Block List Project — gambling](https://github.com/blocklistproject/Lists) | | 2.5k |
| [UT1 — gambling](https://github.com/olbat/ut1-blacklists) | | 32k |
| [UT1 — arjel](https://github.com/olbat/ut1-blacklists) | French gambling regulator list | 69 |
| **Embedded (capped at 20k)** | | **20k** |

### Proxy Avoidance — `categories/proxy-avoidance/`
| Source | Notes | Domains |
|--------|-------|---------|
| [HaGeZi — DoH bypass](https://github.com/hagezi/dns-blocklists) | Encrypted DNS bypass | 3.9k |
| [UT1 — DoH](https://github.com/olbat/ut1-blacklists) | | 3k |
| [UT1 — VPN](https://github.com/olbat/ut1-blacklists) | | 6k |
| **Embedded (capped at 9k)** | | **9k** |

### Other Categories (UT1 + curated)
| Category | Source | Embedded |
|---|---|---|
| personals-dating | UT1 dating | 6.5k |
| hacking | UT1 hacking + warez | 1.8k |
| illegal-drugs | UT1 drogue | 603 |
| chat-im | UT1 chat | 268 |
| social-networking | UT1 social_networks | 715 |
| intimate-apparel | UT1 lingerie | 193 |
| sex-education | UT1 sexual_education | 15 |
| alternative-spirituality | UT1 sect | 144 |
| newsgroups-forums | UT1 forums | 205 |
| adult-mature | UT1 adult | 635 |
| violence-hate | Curated | 29 |
| extreme | Curated | 18 |
| nudity | Curated | 17 |
| abortion | Curated | 19 |
| suspicious | Curated | 21 |
| alcohol | Curated | 36 |
| tobacco | Curated | 39 |
| weapons | Curated | 45 |
| p2p | Curated | 34 |
| open-image-search | Curated | 14 |
| personal-pages | Curated | 20 |
| lgbt | Curated | 17 |
| alternative-sexuality | Curated | 16 |

---

## build.py — Compile Categories into Database

```bash
python3 lists/build.py                    # default level
python3 lists/build.py --level high       # high level
python3 lists/build.py --level all        # build all levels
python3 lists/build.py --stats            # print per-category file vs embedded counts
```

**Binary size**: ~15 MB (with 200k embedded domains for default level).
Full source files in `categories/` are not size-limited and total ~1M+ domains.

---

## sync.py — Fetch Latest from Upstream

```bash
python3 lists/sync.py                     # fetch all, then rebuild default
python3 lists/sync.py --no-build          # fetch only, skip rebuild
python3 lists/sync.py --category phishing # refresh one category only
```

Overwrites category `domains.txt` files with fresh data. Does not touch `keywords.txt` (those are curated).

---

## Source Attribution

| Project | License | URL |
|---------|---------|-----|
| Block List Project | Unlicense | https://github.com/blocklistproject/Lists |
| StevenBlack/hosts | MIT | https://github.com/StevenBlack/hosts |
| OISD | CC0 | https://oisd.nl |
| HaGeZi DNS Blocklists | MIT | https://github.com/hagezi/dns-blocklists |
| URLhaus | — | https://urlhaus.abuse.ch |
| UT1 Blacklists | CC-BY-SA 4.0 | https://github.com/olbat/ut1-blacklists |
