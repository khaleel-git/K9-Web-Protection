#!/usr/bin/env python3
"""
K9 Web Protection — blocklist build script.

Reads category domain/keyword files and level manifests, then writes
the embedded JSON files consumed by mac/app/internal/database/.

Usage:
  python3 lists/build.py [--level default]   # build one level (default)
  python3 lists/build.py --level high        # build high level
  python3 lists/build.py --level all         # build all levels sequentially
  python3 lists/build.py --stats             # print category sizes, no build

The output JSON format matches what database.go expects:
  { "a": ["aaa.com", ...], "b": ["bbb.com", ...], ... }

Per-category caps keep the embedded binary to ~10 MB.
Full source files live in lists/categories/ and are updated by sync.py.
"""

import argparse
import json
import os
import sys
from collections import defaultdict

BASE_DIR   = os.path.dirname(os.path.abspath(__file__))
CAT_DIR    = os.path.join(BASE_DIR, "categories")
LEVEL_DIR  = os.path.join(BASE_DIR, "levels")
DB_DIR     = os.path.join(BASE_DIR, "..", "mac", "app", "internal", "database")

KEYWORD_LIMIT = 20_000

# Per-category domain caps for the embedded binary.
# Full source files are kept in categories/ with no cap.
# Priority: Block List Project > StevenBlack > OISD > UT1 (sync.py fetches in this order)
CATEGORY_CAPS: dict[str, int] = {
    "pornography":                  0,  # no cap — full 790k list for maximum coverage
    "malware-spyware":         50_000,
    "phishing":                50_000,
    "gambling":                     0,  # no cap — 34k, small enough
    "proxy-avoidance":              0,  # no cap — 11k, small enough
    "personals-dating":             0,  # no cap — 6.5k, small enough
    "hacking":                      0,  # no cap — 1.8k, small enough
    # Small categories — embed all
    "illegal-drugs":            0,
    "chat-im":                  0,
    "social-networking":        0,
    "intimate-apparel":         0,
    "sex-education":            0,
    "alternative-spirituality": 0,
    "newsgroups-forums":        0,
    "adult-mature":             0,
    "alternative-sexuality":    0,
    "extreme":                  0,
    "nudity":                   0,
    "abortion":                 0,
    "suspicious":               0,
    "violence-hate":            0,
    "alcohol":                  0,
    "tobacco":                  0,
    "weapons":                  0,
    "p2p":                      0,
    "open-image-search":        0,
    "personal-pages":           0,
    "lgbt":                     0,
    "unrated":                  0,
}


def load_level(name: str) -> list[str]:
    path = os.path.join(LEVEL_DIR, f"{name}.json")
    with open(path) as f:
        return json.load(f)["categories"]


def load_domains(categories: list[str]) -> list[str]:
    seen: set[str] = set()
    domains: list[str] = []
    for cat in categories:
        path = os.path.join(CAT_DIR, cat, "domains.txt")
        if not os.path.exists(path):
            continue
        cat_domains: list[str] = []
        with open(path) as f:
            for line in f:
                d = line.strip().lower()
                if not d or d.startswith("#"):
                    continue
                for pfx in ("https://", "http://", "www."):
                    if d.startswith(pfx):
                        d = d[len(pfx):]
                d = d.rstrip("/")
                if d and d not in seen:
                    seen.add(d)
                    cat_domains.append(d)
        cap = CATEGORY_CAPS.get(cat, 0)
        if cap and len(cat_domains) > cap:
            cat_domains = _even_sample(cat_domains, cap)
        domains.extend(cat_domains)
    return domains


def _even_sample(items: list[str], cap: int) -> list[str]:
    """
    Return up to `cap` entries distributed evenly across first-character buckets.
    This ensures 'p' domains (e.g. pornpics.de) are included even when the
    total list is much larger than the cap and sorted alphabetically.
    """
    from collections import defaultdict
    buckets: dict[str, list[str]] = defaultdict(list)
    for item in items:
        key = item[0] if item and (item[0].isalpha() or item[0].isdigit()) else "_"
        buckets[key].append(item)

    result: list[str] = []
    bucket_keys = sorted(buckets.keys())
    per_bucket = max(1, cap // len(bucket_keys))

    # First pass: take up to per_bucket from each bucket
    for key in bucket_keys:
        result.extend(buckets[key][:per_bucket])

    # Second pass: fill remaining slots from buckets that had more
    remaining = cap - len(result)
    if remaining > 0:
        for key in bucket_keys:
            extra = buckets[key][per_bucket:]
            take = min(remaining, len(extra))
            if take > 0:
                result.extend(extra[:take])
                remaining -= take
            if remaining <= 0:
                break

    return result[:cap]


def load_keywords(categories: list[str]) -> list[str]:
    seen: set[str] = set()
    keywords: list[str] = []
    for cat in categories:
        path = os.path.join(CAT_DIR, cat, "keywords.txt")
        if not os.path.exists(path):
            continue
        with open(path) as f:
            for line in f:
                kw = line.strip().lower()
                if not kw or kw.startswith("#"):
                    continue
                if kw not in seen:
                    seen.add(kw)
                    keywords.append(kw)
    return keywords[:KEYWORD_LIMIT]


def to_alpha_json(items: list[str]) -> dict[str, list[str]]:
    buckets: dict[str, list[str]] = defaultdict(list)
    for item in items:
        key = item[0] if item and item[0].isalpha() else "0"
        buckets[key].append(item)
    return dict(sorted(buckets.items()))


def write_db(domains: list[str], keywords: list[str], dest_dir: str):
    os.makedirs(dest_dir, exist_ok=True)

    with open(os.path.join(dest_dir, "domains.json"), "w") as f:
        json.dump(to_alpha_json(domains), f, indent=2)

    with open(os.path.join(dest_dir, "multi-words.json"), "w") as f:
        json.dump(to_alpha_json(keywords), f, indent=2)

    with open(os.path.join(dest_dir, "urls.json"), "w") as f:
        json.dump({}, f, indent=2)

    print(f"  Written to {dest_dir}")
    print(f"    domains  : {len(domains):,}")
    print(f"    keywords : {len(keywords):,}")


def build(level_name: str):
    print(f"\nBuilding level: {level_name}")
    categories = load_level(level_name)
    if not categories:
        print("  No categories — writing empty database.")
        write_db([], [], DB_DIR)
        return
    print(f"  Categories ({len(categories)}): {', '.join(categories)}")
    domains  = load_domains(categories)
    keywords = load_keywords(categories)
    write_db(domains, keywords, DB_DIR)


def print_stats():
    print(f"\n{'Category':<30} {'File domains':>14} {'Embedded cap':>14}")
    print("-" * 62)
    total_file = total_embed = 0
    for cat in sorted(os.listdir(CAT_DIR)):
        path = os.path.join(CAT_DIR, cat, "domains.txt")
        if not os.path.exists(path):
            continue
        with open(path) as f:
            count = sum(1 for l in f if l.strip() and not l.startswith("#"))
        cap = CATEGORY_CAPS.get(cat, 0)
        embed = min(count, cap) if cap else count
        print(f"  {cat:<28} {count:>14,} {embed:>14,}")
        total_file  += count
        total_embed += embed
    print("-" * 62)
    print(f"  {'TOTAL':<28} {total_file:>14,} {total_embed:>14,}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--level",  default="default")
    parser.add_argument("--stats",  action="store_true")
    args = parser.parse_args()

    if args.stats:
        print_stats()
        sys.exit(0)

    if args.level == "all":
        for lv in ("minimal", "moderate", "default", "high", "monitor"):
            build(lv)
        print("\nDone. Re-run 'go build' in mac/app/ to embed updated lists.")
    else:
        build(args.level)
        print("\nDone. Re-run 'go build' in mac/app/ to embed updated lists.")
