#!/usr/bin/env python3
"""
K9 Web Protection — blocklist sync script.

Fetches the latest data from actively maintained upstream sources:
  - Block List Project   (github.com/blocklistproject/Lists)
  - StevenBlack/hosts   (github.com/StevenBlack/hosts)
  - OISD NSFW           (oisd.nl)
  - HaGeZi              (github.com/hagezi/dns-blocklists)
  - URLhaus              (urlhaus.abuse.ch)

Run this to refresh category files, then run build.py to recompile the DB.

Usage:
  python3 lists/sync.py                   # fetch all, then build default level
  python3 lists/sync.py --no-build        # fetch only, skip build step
  python3 lists/sync.py --category porn   # fetch only the porn category
"""

import argparse
import os
import re
import subprocess
import sys
import urllib.request
from typing import Callable

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
CAT_DIR  = os.path.join(BASE_DIR, "categories")

# ── parser helpers ─────────────────────────────────────────────────────────────

def _parse_hosts(text: str) -> list[str]:
    """Parse hosts-file format: '0.0.0.0 domain.com' or '127.0.0.1 domain.com'."""
    domains: list[str] = []
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split()
        if len(parts) >= 2 and parts[0] in ("0.0.0.0", "127.0.0.1"):
            d = parts[1].lower()
            if d and d not in ("0.0.0.0", "localhost", "broadcasthost"):
                domains.append(d)
    return domains


def _parse_adblock(text: str) -> list[str]:
    """Parse AdBlock/AdGuard format: '||domain.com^'."""
    domains: list[str] = []
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("!") or line.startswith("#"):
            continue
        m = re.match(r'^\|\|([a-z0-9\.\-]+)\^', line)
        if m:
            domains.append(m.group(1).lower())
    return domains


def _parse_plain(text: str) -> list[str]:
    """Parse plain domain-per-line format (HaGeZi, UT1 domains, etc.)."""
    domains: list[str] = []
    for line in text.splitlines():
        d = line.strip().lower()
        if not d or d.startswith("#"):
            continue
        if d.startswith("www."):
            d = d[4:]
        if "." in d and " " not in d:
            domains.append(d)
    return domains


def _parse_url_patterns(text: str) -> list[str]:
    """Parse plain URL-pattern or keyword list (one entry per line, no dot filter)."""
    items: list[str] = []
    for line in text.splitlines():
        item = line.strip().lower()
        if not item or item.startswith("#"):
            continue
        items.append(item)
    return items


# ── fetch helper ───────────────────────────────────────────────────────────────

def fetch(url: str, parser: Callable[[str], list[str]],
          cap: int = 0, label: str = "") -> list[str]:
    label = label or url.split("/")[-1]
    print(f"    Fetching {label} ...", end=" ", flush=True)
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "K9-sync/1.0"})
        with urllib.request.urlopen(req, timeout=30) as r:
            text = r.read().decode("utf-8", errors="ignore")
        items = parser(text)
        if cap and len(items) > cap:
            items = items[:cap]
        print(f"{len(items):,} entries")
        return items
    except Exception as e:
        print(f"FAILED ({e})")
        return []


def merge_into(category: str, filename: str, new_items: list[str]):
    """Merge new_items into an existing category file (dedup, preserves manual entries)."""
    path = os.path.join(CAT_DIR, category, filename)
    os.makedirs(os.path.dirname(path), exist_ok=True)

    existing: set[str] = set()
    if os.path.exists(path):
        with open(path) as f:
            for line in f:
                d = line.strip().lower()
                if d and not d.startswith("#"):
                    existing.add(d)

    combined = sorted(existing | set(new_items))
    with open(path, "w") as f:
        f.write("\n".join(combined) + "\n")

    added = len(set(new_items) - existing)
    print(f"    → {path.split('lists/')[-1]}: {len(combined):,} total (+{added:,} new)")


def overwrite(category: str, filename: str, items: list[str]):
    """Write sorted, deduped entries to a category file (overwrites)."""
    path = os.path.join(CAT_DIR, category, filename)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    unique = sorted(set(d.lower() for d in items if d and not d.startswith("#")))
    with open(path, "w") as f:
        f.write("\n".join(unique) + "\n")
    print(f"    → {path.split('lists/')[-1]}: {len(unique):,} entries")


# ── source definitions ─────────────────────────────────────────────────────────

BLP  = "https://raw.githubusercontent.com/blocklistproject/Lists/main"
SB   = "https://raw.githubusercontent.com/StevenBlack/hosts/master"
OISD = "https://nsfw.oisd.nl"
HGZ  = "https://raw.githubusercontent.com/hagezi/dns-blocklists/main/domains"
UT1  = "https://raw.githubusercontent.com/olbat/ut1-blacklists/master/blacklists"
URLH = "https://urlhaus.abuse.ch/downloads/hostfile"

SOURCES: dict[str, dict] = {

    # ── Pornography ────────────────────────────────────────────────────────────
    "pornography": [
        {"type": "domains", "label": "Block List Project — porn",    "url": f"{BLP}/porn.txt",                     "parser": _parse_hosts,        "cap": 0},
        {"type": "domains", "label": "StevenBlack — porn-only",      "url": f"{SB}/alternates/porn-only/hosts",    "parser": _parse_hosts,        "cap": 0},
        {"type": "domains", "label": "OISD NSFW",                    "url": OISD,                                  "parser": _parse_adblock,      "cap": 0},
        {"type": "urls",    "label": "UT1 — adult (urls)",           "url": f"{UT1}/adult/urls",                   "parser": _parse_url_patterns, "cap": 0},
        {"type": "keywords","label": "UT1 — adult (keywords)",       "url": f"{UT1}/adult/keywords",               "parser": _parse_url_patterns, "cap": 0},
    ],

    # ── Malware / Spyware ──────────────────────────────────────────────────────
    "malware-spyware": [
        {"type": "domains", "label": "Block List Project — malware", "url": f"{BLP}/malware.txt",                  "parser": _parse_hosts,        "cap": 50_000},
        {"type": "domains", "label": "URLhaus — hostfile",           "url": URLH,                                  "parser": _parse_hosts,        "cap": 0},
        {"type": "domains", "label": "HaGeZi — Threat Intel Feeds", "url": f"{HGZ}/tif.txt",                      "parser": _parse_plain,        "cap": 50_000},
    ],

    # ── Phishing ───────────────────────────────────────────────────────────────
    "phishing": [
        {"type": "domains", "label": "Block List Project — phishing","url": f"{BLP}/phishing.txt",                 "parser": _parse_hosts,        "cap": 50_000},
        {"type": "domains", "label": "Block List Project — abuse",   "url": f"{BLP}/abuse.txt",                   "parser": _parse_hosts,        "cap": 30_000},
        {"type": "domains", "label": "Block List Project — scam",    "url": f"{BLP}/scam.txt",                    "parser": _parse_hosts,        "cap": 30_000},
        {"type": "domains", "label": "UT1 — phishing",               "url": f"{UT1}/phishing/domains",            "parser": _parse_plain,        "cap": 50_000},
        {"type": "urls",    "label": "UT1 — phishing (urls)",        "url": f"{UT1}/phishing/urls",               "parser": _parse_url_patterns, "cap": 0},
    ],

    # ── Gambling ───────────────────────────────────────────────────────────────
    "gambling": [
        {"type": "domains", "label": "Block List Project — gambling","url": f"{BLP}/gambling.txt",                 "parser": _parse_hosts,        "cap": 0},
        {"type": "domains", "label": "UT1 — gambling",               "url": f"{UT1}/gambling/domains",            "parser": _parse_plain,        "cap": 0},
        {"type": "domains", "label": "UT1 — arjel (FR gambling reg)","url": f"{UT1}/arjel/domains",              "parser": _parse_plain,        "cap": 0},
    ],

    # ── Hacking ────────────────────────────────────────────────────────────────
    "hacking": [
        {"type": "domains", "label": "UT1 — hacking",                "url": f"{UT1}/hacking/domains",             "parser": _parse_plain,        "cap": 0},
        {"type": "domains", "label": "UT1 — warez",                  "url": f"{UT1}/warez/domains",               "parser": _parse_plain,        "cap": 0},
        {"type": "urls",    "label": "UT1 — hacking (urls)",         "url": f"{UT1}/hacking/urls",                "parser": _parse_url_patterns, "cap": 0},
    ],

    # ── Proxy Avoidance ────────────────────────────────────────────────────────
    "proxy-avoidance": [
        {"type": "domains", "label": "HaGeZi — DoH bypass",         "url": f"{HGZ}/doh.txt",                     "parser": _parse_plain,        "cap": 0},
        {"type": "domains", "label": "UT1 — DoH",                   "url": f"{UT1}/doh/domains",                 "parser": _parse_plain,        "cap": 0},
        {"type": "domains", "label": "UT1 — VPN",                   "url": f"{UT1}/vpn/domains",                 "parser": _parse_plain,        "cap": 0},
    ],

    # ── Personals / Dating ────────────────────────────────────────────────────
    "personals-dating": [
        {"type": "domains", "label": "UT1 — dating",                 "url": f"{UT1}/dating/domains",              "parser": _parse_plain,        "cap": 0},
    ],

    # ── Illegal Drugs ─────────────────────────────────────────────────────────
    "illegal-drugs": [
        {"type": "domains", "label": "UT1 — drogue",                 "url": f"{UT1}/drogue/domains",              "parser": _parse_plain,        "cap": 0},
    ],

    # ── Chat / IM ─────────────────────────────────────────────────────────────
    "chat-im": [
        {"type": "domains", "label": "UT1 — chat",                   "url": f"{UT1}/chat/domains",                "parser": _parse_plain,        "cap": 0},
    ],

    # ── Social Networking ─────────────────────────────────────────────────────
    "social-networking": [
        {"type": "domains", "label": "UT1 — social_networks",        "url": f"{UT1}/social_networks/domains",     "parser": _parse_plain,        "cap": 0},
    ],

    # ── Intimate Apparel / Swimsuit ───────────────────────────────────────────
    "intimate-apparel": [
        {"type": "domains", "label": "UT1 — lingerie",               "url": f"{UT1}/lingerie/domains",            "parser": _parse_plain,        "cap": 0},
    ],

    # ── Sex Education ─────────────────────────────────────────────────────────
    "sex-education": [
        {"type": "domains", "label": "UT1 — sexual_education",       "url": f"{UT1}/sexual_education/domains",    "parser": _parse_plain,        "cap": 0},
    ],

    # ── Alternative Spirituality ──────────────────────────────────────────────
    "alternative-spirituality": [
        {"type": "domains", "label": "UT1 — sect",                   "url": f"{UT1}/sect/domains",                "parser": _parse_plain,        "cap": 0},
    ],

    # ── Newsgroups / Forums ───────────────────────────────────────────────────
    "newsgroups-forums": [
        {"type": "domains", "label": "UT1 — forums",                 "url": f"{UT1}/forums/domains",              "parser": _parse_plain,        "cap": 0},
    ],
}


# ── runner ─────────────────────────────────────────────────────────────────────

def sync_category(name: str, sources: list[dict]):
    print(f"\n[{name}]")
    domains:  list[str] = []
    urls:     list[str] = []
    keywords: list[str] = []

    for src in sources:
        typ   = src.get("type", "domains")
        items = fetch(src["url"], src["parser"], src.get("cap", 0), src["label"])
        if typ == "urls":
            urls.extend(items)
        elif typ == "keywords":
            keywords.extend(items)
        else:
            domains.extend(items)

    overwrite(name, "domains.txt", domains)
    if urls:
        # merge preserves hand-curated entries already in urls.txt
        merge_into(name, "urls.txt", urls)
    if keywords:
        merge_into(name, "keywords.txt", keywords)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--no-build",  action="store_true", help="Skip build step")
    parser.add_argument("--category",  default="all", help="Sync a single category")
    args = parser.parse_args()

    if args.category == "all":
        for cat, sources in SOURCES.items():
            sync_category(cat, sources)
    else:
        if args.category not in SOURCES:
            sys.exit(f"Unknown category: {args.category}. Available: {list(SOURCES)}")
        sync_category(args.category, SOURCES[args.category])

    if not args.no_build:
        print("\nRebuilding default-level database …")
        build_script = os.path.join(BASE_DIR, "build.py")
        subprocess.run([sys.executable, build_script, "--level", "default"], check=True)
        print("Done. Re-run 'go build' in mac/app/ to embed the updated lists.")
    else:
        print("\nSkipped build. Run: python3 lists/build.py --level default")


if __name__ == "__main__":
    main()
