# Public BlockerFlow block lists, and K9 built from them

Date: 2026-11-19
Status: draft, waiting for approval

## Why

BlockerFlow's block lists have been cleaned up and tested against false positives (the
`fpaudit` gates, the allowlist, the dead-domain pruning). K9 Web Protection still ships its
own older lists, and the Windows build ships a much smaller pornography list than the Mac
build (its `domains.json` is about 650 KB against the Mac's 20 MB, because `lists/build.py`
only writes `mac/app/internal/database/`).

Two outcomes are wanted:

1. **A public repo of BlockerFlow's lists that anyone can use**: K9 users, people running
   Pi-hole, AdGuard, uBlock Origin or a router, and other developers.
2. **K9 ships those lists with every build**, on both Mac and Windows.

## Goals

- A new public GitHub repo holding exactly the lists BlockerFlow ships, and nothing else.
- Each domain list in three formats: plain domains, hosts file, AdBlock syntax.
- The repo always matches what BlockerFlow's **stable** channel ships.
- K9's build pulls a pinned version of those lists, checks them, and rebuilds the embedded
  database for Mac and Windows alike.

## Non-goals

- No runtime list downloads in K9. Lists change only when K9 is rebuilt (decided).
- No K9-only categories in the public repo. K9's other 27 categories (gambling, malware,
  phishing, weapons, ...) stay in the K9 repo only (decided).
- No change to how the BlockerFlow apps get their lists. `lists.blockerflow.net` and the
  signed packs keep working exactly as they do now.

## Anti-goals (a failure even if it technically works)

- The public repo drifts from what BlockerFlow ships, or carries beta lists people treat
  as stable.
- A broken, truncated or tampered list reaches the public repo or a K9 build.
- The lists go public under terms that break an upstream source's license.

## What exists today

**BlockerFlow** (`~/Documents/blockerflow`, `shared/blocklists/`):

- `publish.py` builds the lists exactly as Android bundles them (`./gradlew :app:copyBlocklists`),
  runs the Android unit tests, gzips each file into `shared/blocklists/dist/<version>/`,
  signs a manifest with `~/.blockerflow/lists-signing.pem` (ECDSA P-256), and uploads it with
  Wrangler to the R2 bucket `blockerflow-lists`, served at `https://lists.blockerflow.net`.
- `--channel beta|stable` publishes a fresh pack; `--promote <version>` moves an existing pack
  to stable (also used for rollback).
- Manifest: `channels/<channel>.json` = `{"payload": "<json text>", "signature": "<base64>"}`.
  The payload lists each file's `path`, `sha256` and `size`, plus `seq` and `version`.
- Public key: `android/app/src/main/res/raw/lists_public_key.pem`.
- Live today: stable `20261105.8`, beta `20261105.9`.

What a pack (and so every BlockerFlow app) ships:

| File | Format |
|---|---|
| `pornography/domains.txt` | `hostname\|source_mask`, sorted, about 19 MB |
| `pornography/keywords.txt` | `keyword\|weight\|requiresBoundary` |
| `pornography/keywords_intl.txt` | international keywords |
| `pornography/keyword_tiers.txt` | `keyword\|strong` or `keyword\|context` |
| `pornography/adult_hostname_keywords.txt` | one brand root per line |
| `pornography/page_check_skip.txt` | hosts the page check skips |
| `pornography/domains.len` | internal line count used by the Android app |
| `web-proxy/domains.txt` | one hostname per line, about 360 KB |

Upstream sources behind those lists: Block List Project (porn), StevenBlack porn extensions
(clefspeare13, sinfonietta), OISD NSFW, HaGeZi NSFW, UT1 adult keywords and URLs, and UT1
for web-proxy (`build_categories.py`).

**K9** (this repo):

- `lists/categories/<category>/domains.txt` and `keywords.txt`, one entry per line.
- `lists/build.py --level <level>` compiles them into `domains.json`, `urls.json`,
  `url-patterns.json` and `multi-words.json` under `mac/app/internal/database/` only.
- `windows/app/internal/database/` and `mac/app/internal/database/` embed those files with
  `go:embed`. `database.go` is identical in both.
- K9 has `pornography` and `proxy-avoidance` categories; these map to BlockerFlow's
  `pornography` and `web-proxy`.
- No build script or CI workflow runs `build.py` or `sync.py`; the generated JSON is committed.

## Part 1: the public repo

**Name and home:** `khaleel-git/blockerflow-lists`, public (my call; not objected to). The
README introduces BlockerFlow and links to it.

**Contents:** exactly what BlockerFlow ships (decided), keywords included. Layout (my call;
the builder may adjust names):

```
pornography/
  domains.txt          one hostname per line (source_mask stripped)
  hosts.txt            0.0.0.0 <hostname>
  adblock.txt          ||<hostname>^
  keywords.txt         as shipped, keyword|weight|requiresBoundary
  keywords_intl.txt    as shipped
  keyword_tiers.txt    as shipped
  adult_hostname_keywords.txt  as shipped
  page_check_skip.txt  as shipped
web-proxy/
  domains.txt
  hosts.txt
  adblock.txt
manifest.json          the signed stable manifest this commit was made from
README.md              what each file is, its format, copy-paste links for Pi-hole,
                       AdGuard, uBlock Origin, hosts files; how to verify the signature
LICENSE / NOTICE       see "Licensing" below
```

- `domains.len` is not published; it is internal to the Android app (my call).
- `source_mask` is stripped from the public domain lists so they work in every tool
  (my call). The README says which sources feed each list.
- Keyword files keep the exact shipped format, and the README documents the columns,
  including that `weight >= 100` means high confidence.
- The three formats are generated from the same domain list, so they always hold the same
  hosts. Each file starts with a comment header (version, date, source link, license) in the
  comment syntax its format allows.

**Licensing (must be settled before the repo goes public):** the lists derive from sources
with different terms. Known: Block List Project is Unlicense, StevenBlack is MIT, HaGeZi is
GPL-3.0, UT1 is CC BY-SA 4.0. Unknown: OISD's terms for redistribution, and the licenses of
the clefspeare13 and sinfonietta StevenBlack extensions. The builder checks each source's
current terms first and proposes the repo's licensing (likely GPL-3.0 for `pornography/`,
CC BY-SA 4.0 for `web-proxy/` and any UT1-derived keywords, with a NOTICE crediting every
source). If any source forbids redistribution, stop and bring it back before publishing.
Nothing is pushed to a public repo until this is resolved.

## Part 2: keeping the repo in step with BlockerFlow

`publish.py` gains a step that updates the public repo **only when stable changes**: after
a successful `--channel stable` publish or a `--promote` (which also covers rollback). Beta
publishes never touch the public repo (my call, from the "stable only" anti-goal).

The step:

1. Downloads the pack just made stable from `lists.blockerflow.net` (`channels/stable.json`
   and `packs/<version>/*.gz`) and checks the signature and every sha256 against it (my
   call). It does not read `dist/<version>/`: `dist/` is gitignored and only exists on the
   machine that built the pack, and the signing key is not on every machine. Checking with
   the public key means the step needs no secret, can be run on its own, and can be tested
   anywhere.
2. Regenerates the repo's files from them (strip masks, write the three formats, copy the
   keyword files).
3. Commits to a local clone of the public repo with a message naming the version and
   `seq`, tags it `v<version>` (for example `v20261105.8`), and pushes.

Nothing is committed when the upload to R2 failed, and the R2 upload is not rolled back if
the git push fails; the script says the repo is behind and how to re-run only the repo step
(builder's call on the flag name). `--dry-run` writes the repo files but commits nothing.

The first commit is made from the current stable pack, `20261105.8`, downloaded and checked
the same way.

**Going live (my call):** the builder prepares the repo contents and shows the `--dry-run`
result, and asks before creating the public GitHub repo and before the first push. Commits
and pushes to the K9 repo are also asked for, not assumed. Building and signing new K9
installers (`.pkg`, Windows setup) is not part of this work.

## Part 3: K9 built from the repo

A new script in `lists/` (name is the builder's call) does, in order:

1. Reads the pinned version from a file in `lists/` (for example `lists/blockerflow-lists.version`).
   A `--latest` flag reads and verifies the signed `manifest.json` on the public repo's `main`
   (the current stable, which respects rollbacks) and rewrites the pin to its version.
2. Downloads that tag's `manifest.json` and files from the public repo.
3. Verifies the manifest signature with BlockerFlow's public key (copied into the K9 repo),
   and each file's sha256 against the manifest. Any mismatch stops the script with nothing
   written.
4. Writes `pornography/domains.txt` into `lists/categories/pornography/domains.txt` and
   `web-proxy/domains.txt` into `lists/categories/proxy-avoidance/domains.txt`, one hostname
   per line, replacing what is there.
5. Runs `build.py`.

The manifest hashes the shipped files (with `source_mask`), so the K9 script downloads the
shipped-format domain lists to check them. Either the public repo also carries the
shipped-format `domains.txt` (for example as `pornography/domains-with-sources.txt`), or K9
verifies against hashes the repo step adds to its own `manifest.json`. Builder's call, as
long as K9 can prove the lists are byte-for-byte what BlockerFlow signed.

`build.py` changes to write the database files into both `mac/app/internal/database/` and
`windows/app/internal/database/`, so the two apps embed identical lists.

**Proxy list (decided):** K9's `proxy-avoidance` domains are replaced outright by
BlockerFlow's `web-proxy` list, not merged with it. This drops K9's current list (11,691
hosts, mostly DNS-over-HTTPS resolvers such as `dns.google` and VPN hosts, from HaGeZi DoH and
UT1 `doh`/`vpn`); only 296 of those hosts are also in BlockerFlow's 19,068. `sync.py` stops
fetching those DoH and VPN domain sources for `proxy-avoidance`.

**Windows parity (decided):** both apps embed the same database. Today the Windows
`domains.json` is in an old format keyed by first letter (`{"a": [...], "b": [...]}`), so on
every named level Windows blocks none of the category domains; only Custom with adult
blocking on uses them. After this change Windows blocks every category its level lists, as
the Mac does. Windows' current `urls.json` holds about 4,300 Reddit and Twitter URL entries
the Mac file lacks; those are kept (moved into the K9 category source data so `build.py`
produces them for both apps). Needed with it:

- Windows Go tests that each named level now blocks a known domain of each of its categories,
  and that Monitor blocks nothing.
- A release note for Windows users that their level now blocks its full category set.

**Keywords in K9:** K9 keeps its own `keywords.txt` files; BlockerFlow's scored keywords are
not imported (my call). K9's matcher treats every keyword as a flat phrase, while
BlockerFlow's keywords depend on their weights and boundary flags, so importing them as plain
phrases would block innocent pages.

**K9's other categories** stay as they are and keep coming from `lists/sync.py`. `sync.py`
must no longer write `pornography/domains.txt` or `proxy-avoidance/domains.txt` (it can
still fetch their keywords and URL patterns).

**When it runs (my call):** by hand, like `build.py` today. A developer runs the script,
and the regenerated category files and database JSON are committed. The release scripts and
CI do not fetch anything. Instead, a CI check verifies that the committed pornography and
proxy domain lists match the pinned version's signed manifest, so a hand-edited or tampered
list fails CI.

**Checks:** the existing Go tests must pass on both apps. Add a test that the embedded
pornography and proxy-avoidance domain counts are within a sane range (catches an empty or
truncated list), and one that the Mac and Windows `domains.json` are identical.

**Size:** the Windows binary grows by about 20 MB of embedded data, matching the Mac.

The K9 README and `lists/README.md` explain where these two lists now come from and how to
bump the pin.

## Left to the builder

- File and flag names, the repo step's exact git mechanics, commit message wording.
- Whether files live as plain files in git or as release assets, as long as a stable raw URL
  per file exists for Pi-hole and AdGuard users.
- The README wording and the exact per-format header text.
- How the K9 download is done (raw URLs or a tag tarball).
