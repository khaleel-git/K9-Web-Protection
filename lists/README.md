# K9 Web Protection: Blocklists

Category-based blocklist source data, compiled into the embedded Go database.

## Quick Start

```bash
# 1. Fetch the pinned BlockerFlow pornography + proxy-avoidance lists (requires internet)
python3 lists/fetch_blockerflow.py --no-build

# 2. Fetch latest from the other upstream sources (requires internet)
python3 lists/sync.py --no-build

# 3. Compile into the embedded database; writes the same four JSON files to both apps
python3 lists/build.py

# 4. Rebuild the Go binaries to embed the new data
cd mac/app && go build ./...
cd windows/app && go build ./...
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
  blockerflow-lists.version  ← pinned version of BlockerFlow's public lists
  blockerflow_public_key.pem ← key that verifies the signed manifest
  fetch_blockerflow.py ← fetches pornography + proxy-avoidance from the pinned public repo
  sync.py              ← fetches the other upstream sources → updates categories/
  build.py             ← compiles categories/ → mac/app/internal/database/ AND
                         windows/app/internal/database/ (identical JSON in both)
  tests/               ← unittest suite for the tooling (python3 -m unittest discover -s lists/tests)
  README.md            ← this file
```

---

## Levels

| Level    | Description                                             |
|----------|---------------------------------------------------------|
| **High**     | All default + chat, newsgroups, social, alcohol, etc.  |
| **Default**  | Adult content, security threats, illegal, sexually-related |
| **Moderate** | Adult content, security threats, illegal activity      |
| **Minimal**  | Pornography and security threats only                  |
| **Monitor**  | Allows all, logs traffic only (no blocking)           |
| **Custom**   | User-selected categories (stored in app config)        |

Domain totals per category come from `python3 lists/build.py --stats`. Both apps embed the same
database, and the Windows filter level blocks every category its level lists.

---

## Sources (actively maintained)

### Pornography: `categories/pornography/`
Comes from BlockerFlow's public lists, not from K9's own syncing. See
[BlockerFlow lists: pornography and proxy-avoidance](#blockerflow-lists-pornography-and-proxy-avoidance).
About 894k domains in the pinned source file, about 880k embedded after `build.py` normalises them
(strips `www.`, dedupes); exact totals come from `python3 lists/build.py --stats`. `sync.py` still
merges UT1 adult URLs and keywords into `pornography/urls.txt` and `keywords.txt` (merge only, it
never overwrites them).

### Malware / Spyware: `categories/malware-spyware/`
| Source | Notes | Domains |
|--------|-------|---------|
| [Block List Project, malware](https://github.com/blocklistproject/Lists) | Community maintained | 435k |
| [URLhaus](https://urlhaus.abuse.ch) | abuse.ch, active malware URLs | ~500 |
| [HaGeZi TIF](https://github.com/hagezi/dns-blocklists) | Threat Intelligence Feeds, 6h updates | 1.2M |
| **Embedded (capped at 30k)** | | **30k** |

### Phishing: `categories/phishing/`
| Source | Notes | Domains |
|--------|-------|---------|
| [Block List Project, phishing](https://github.com/blocklistproject/Lists) | | 435k |
| [Block List Project, abuse](https://github.com/blocklistproject/Lists) | | 25k |
| [Block List Project, scam](https://github.com/blocklistproject/Lists) | | 1.3k |
| [UT1, phishing](https://github.com/olbat/ut1-blacklists) | | 688k |
| **Embedded (capped at 30k)** | | **30k** |

### Gambling: `categories/gambling/`
| Source | Notes | Domains |
|--------|-------|---------|
| [Block List Project, gambling](https://github.com/blocklistproject/Lists) | | 2.5k |
| [UT1, gambling](https://github.com/olbat/ut1-blacklists) | | 32k |
| [UT1, arjel](https://github.com/olbat/ut1-blacklists) | French gambling regulator list | 69 |
| **Embedded (capped at 20k)** | | **20k** |

### Proxy Avoidance: `categories/proxy-avoidance/`
Comes from BlockerFlow's public web-proxy list (about 19k domains). It no longer includes
DNS-over-HTTPS resolvers or VPN hosts. See
[BlockerFlow lists](#blockerflow-lists-pornography-and-proxy-avoidance).

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

## BlockerFlow lists: pornography and proxy-avoidance

`categories/pornography/domains.txt` and `categories/proxy-avoidance/domains.txt` are generated
from a pinned, signed version of the public repo
[khaleel-git/blockerflow-lists](https://github.com/khaleel-git/blockerflow-lists). Do not edit
them by hand. `sync.py` no longer writes `domains.txt` for these two categories. (It still merges
UT1 adult URLs and keywords into `pornography/urls.txt` and `keywords.txt`; it only merges, it
never overwrites them.)

`fetch_blockerflow.py` reads the pin from `lists/blockerflow-lists.version`, verifies the repo's
signed `manifest.json` against `lists/blockerflow_public_key.pem`, checks every file against its
signed size and sha256, and only then writes the two category files.

```bash
python3 lists/fetch_blockerflow.py             # fetch the pinned version, write, run build.py
python3 lists/fetch_blockerflow.py --check     # write nothing; exit 1 if the files differ from the pin
python3 lists/fetch_blockerflow.py --no-build  # skip build.py
python3 lists/fetch_blockerflow.py --latest    # use the newest vYYYYMMDD.N tag and update the pin
```

To bump to a newer BlockerFlow release:

1. `python3 lists/fetch_blockerflow.py --latest` (updates the pin, the two category files, and
   runs `build.py`).
2. Commit the pin (`lists/blockerflow-lists.version`), the category files, and the regenerated
   JSON in both `mac/app/internal/database/` and `windows/app/internal/database/`.

CI (`.github/workflows/lists-check.yml`) runs the tooling tests, runs `--check`, and verifies
the Mac and Windows JSON files are identical.

---

## build.py: Compile Categories into Database

```bash
python3 lists/build.py                    # build once; copy the four JSON files to both apps
python3 lists/build.py --stats            # print per-category counts
```

Writes `domains.json`, `urls.json`, `url-patterns.json` and `multi-words.json` to both
`mac/app/internal/database/` and `windows/app/internal/database/`. Source files in
`categories/` are not size-limited.

---

## sync.py: Fetch Latest from Upstream

```bash
python3 lists/sync.py                     # fetch all, then rebuild default
python3 lists/sync.py --no-build          # fetch only, skip rebuild
python3 lists/sync.py --category phishing # refresh one category only
```

Overwrites category `domains.txt` files with fresh data, except `pornography` and
`proxy-avoidance`, which come from BlockerFlow's lists (see above). `urls.txt` and `keywords.txt`
are only merged into, never overwritten; the only such source today is UT1 adult for
`pornography/`.

---

## Source Attribution

| Project | License | URL |
|---------|---------|-----|
| Block List Project | Unlicense | https://github.com/blocklistproject/Lists |
| StevenBlack/hosts | MIT | https://github.com/StevenBlack/hosts |
| OISD | CC0 | https://oisd.nl |
| HaGeZi DNS Blocklists | MIT | https://github.com/hagezi/dns-blocklists |
| URLhaus |, | https://urlhaus.abuse.ch |
| UT1 Blacklists | CC-BY-SA 4.0 | https://github.com/olbat/ut1-blacklists |
| BlockerFlow lists: pornography | GPL-3.0 | https://github.com/khaleel-git/blockerflow-lists |
| BlockerFlow lists: web-proxy (proxy-avoidance) | CC BY-SA 4.0 | https://github.com/khaleel-git/blockerflow-lists |
