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


def build_category_db(dest_dir: str):
    """Build a category-keyed domains.json with ALL categories for runtime level filtering."""
    print("\nBuilding category-aware database (all categories)…")
    all_cats: dict[str, list[str]] = {}
    total = 0
    for cat in sorted(CATEGORY_CAPS.keys()):
        path = os.path.join(CAT_DIR, cat, "domains.txt")
        if not os.path.exists(path):
            continue
        seen: set[str] = set()
        domains: list[str] = []
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
                    domains.append(d)
        cap = CATEGORY_CAPS.get(cat, 0)
        if cap and len(domains) > cap:
            domains = _even_sample(domains, cap)
        all_cats[cat] = sorted(domains)
        total += len(domains)
        print(f"  {cat:<30} {len(domains):>10,}")

    os.makedirs(dest_dir, exist_ok=True)
    with open(os.path.join(dest_dir, "domains.json"), "w") as f:
        json.dump(all_cats, f, separators=(',', ':'))
    print(f"\n  Total domains : {total:,}")
    print(f"  Written to    : {dest_dir}/domains.json")


def _read_patterns(cat: str, filename: str) -> list[str]:
    """Read non-empty, non-comment lines from a category file."""
    path = os.path.join(CAT_DIR, cat, filename)
    if not os.path.exists(path):
        return []
    out: list[str] = []
    with open(path) as f:
        for line in f:
            p = line.strip().lower()
            if p and not p.startswith("#"):
                out.append(p)
    return out


def build_category_urls(dest_dir: str):
    """Build category-keyed urls.json from each category's urls.txt.

    urls.txt contains full-URL substring patterns including TLD patterns
    (.xxx, .porn) and path/query patterns (/porn/, ?q=porn).
    Checked against the full raw URL in the proxy.
    """
    print("\nBuilding category URL patterns (urls.json)…")
    result: dict[str, list[str]] = {}
    total = 0
    for cat in sorted(CATEGORY_CAPS.keys()):
        patterns = _read_patterns(cat, "urls.txt")
        if patterns:
            result[cat] = sorted(set(patterns))
            total += len(result[cat])
            print(f"  {cat:<30} {len(result[cat]):>8,} url patterns")
    os.makedirs(dest_dir, exist_ok=True)
    with open(os.path.join(dest_dir, "urls.json"), "w") as f:
        json.dump(result, f, separators=(',', ':'))
    print(f"\n  Total url patterns : {total:,}")
    print(f"  Written to         : {dest_dir}/urls.json")
    return total


def build_url_patterns(dest_dir: str):
    """Build category-keyed url-patterns.json from each category's url-patterns.txt.

    url-patterns.txt contains keyword wildcards checked against URL path+query.
    E.g. 'porn' blocks any URL whose path/query contains 'porn'.
    Category-aware: only fires for categories active at the current filter level.
    """
    print("\nBuilding URL keyword patterns (url-patterns.json)…")
    result: dict[str, list[str]] = {}
    total = 0
    for cat in sorted(CATEGORY_CAPS.keys()):
        patterns = _read_patterns(cat, "url-patterns.txt")
        if patterns:
            result[cat] = sorted(set(patterns))
            total += len(result[cat])
            print(f"  {cat:<30} {len(result[cat]):>8,} keyword patterns")
    os.makedirs(dest_dir, exist_ok=True)
    with open(os.path.join(dest_dir, "url-patterns.json"), "w") as f:
        json.dump(result, f, separators=(',', ':'))
    print(f"\n  Total keyword patterns : {total:,}")
    print(f"  Written to             : {dest_dir}/url-patterns.json")
    return total


def write_page_keywords(dest_dir: str):
    """Write multi-words.json (page content keyword phrases from all categories)."""
    all_cats = list(CATEGORY_CAPS.keys())
    keywords = load_keywords(all_cats)
    os.makedirs(dest_dir, exist_ok=True)
    with open(os.path.join(dest_dir, "multi-words.json"), "w") as f:
        json.dump(to_alpha_json(keywords), f, indent=2)
    print(f"  page keywords : {len(keywords):,}")
    return len(keywords)


def build(level_name: str):
    """Build all embedded database files."""
    print(f"\nNote: level '{level_name}' ignored — building full category-aware DB.")
    build_category_db(DB_DIR)
    print()
    n_urls     = build_category_urls(DB_DIR)
    n_patterns = build_url_patterns(DB_DIR)
    print()
    n_keywords = write_page_keywords(DB_DIR)
    print(f"\n  Summary: {n_urls:,} url patterns | {n_patterns:,} keyword patterns | {n_keywords:,} page keywords")


def print_stats():
    print(f"\n{'Category':<30} {'Domains':>10} {'URLs':>8} {'Patterns':>10} {'Keywords':>10}")
    print("-" * 72)
    for cat in sorted(os.listdir(CAT_DIR)):
        cat_dir = os.path.join(CAT_DIR, cat)
        if not os.path.isdir(cat_dir):
            continue
        def count_file(fn):
            p = os.path.join(cat_dir, fn)
            if not os.path.exists(p):
                return 0
            with open(p) as f:
                return sum(1 for l in f if l.strip() and not l.startswith("#"))
        d = count_file("domains.txt")
        u = count_file("urls.txt")
        p = count_file("url-patterns.txt")
        k = count_file("keywords.txt")
        if d or u or p or k:
            print(f"  {cat:<28} {d:>10,} {u:>8,} {p:>10,} {k:>10,}")
    print("-" * 72)


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
