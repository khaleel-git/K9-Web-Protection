# Public BlockerFlow Lists and K9 Built From Them: Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish BlockerFlow's two shipped lists in a public repo that always matches the stable channel, and make K9 (Mac and Windows) embed those lists from a pinned, signature-checked version.

**Architecture:** A new BlockerFlow script (`public_repo.py`) downloads the stable pack from `lists.blockerflow.net`, checks it against the signed manifest, renders the repo files (plain, hosts, AdBlock, plus the byte-identical shipped files), commits and tags. `publish.py` calls it after every stable change. In K9, `lists/fetch_blockerflow.py` reads a pinned tag from the public repo, checks the same signature, and writes K9's `pornography` and `proxy-avoidance` domain lists. `build.py` then writes one identical database for both apps.

**Tech Stack:** Python 3 stdlib plus the `openssl` CLI (signature checks, as `publish.py` already does), `unittest`, Go 1.24 (`go test`), GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-11-19-public-blocklists-repo-design.md` (read it before any task).

## Global Constraints

- Three repos, each on its own branch:
  - **BlockerFlow:** `~/Documents/blockerflow`. Its checkout is on `marketing/play-store-aso` with uncommitted work, so never touch it. Work in a new worktree `~/Documents/blockerflow-public-lists` on branch `feature/public-lists-repo`, cut from `master`.
  - **Public repo:** `~/Documents/blockerflow-lists`, a new local git repo for `khaleel-git/blockerflow-lists`.
  - **K9:** this repo, on branch `feature/blockerflow-lists`, cut from `master`.
- Ask the user before creating the GitHub repo, before any `git push`, and before any merge. Commits to local feature branches are fine.
- Signature: ECDSA P-256 with SHA-256 over the payload's UTF-8 bytes, DER-encoded, base64 in `signature`. Verify with `openssl dgst -sha256 -verify <pem> -signature <der> <payload>`. The public key is BlockerFlow `master:android/app/src/main/res/raw/lists_public_key.pem`.
- Pack source: `https://lists.blockerflow.net/channels/stable.json` and `packs/<version>/<path>.gz`. Send the header `User-Agent: blockerflow-lists-sync/1.0`, because Cloudflare returns 403 for Python's default UA.
- Only the stable channel ever reaches the public repo. Never beta.
- Shipped domain files have `#` header lines and `domain|source_mask` rows. This applies to both `pornography/domains.txt` and `web-proxy/domains.txt`.
- `domains.len` is never published.
- Tags are `v<version>`, for example `v20261105.8`. Never move an existing tag.
- The first public version is stable `20261105.8`. Today's K9 pin is `20261105.8`.
- If any check fails, write nothing: no partial files, no commit.

## Review Focus

1. **Bad or partial download.** One file fails its sha256 or a request times out mid-fetch. Expected: the script stops with a clear message, and no file in the public repo or in K9's `lists/categories/` changes. Tests: Task 2 `test_render_refuses_bad_hash` and Task 5 `test_load_pack_hash_mismatch_writes_nothing`.
2. **Same version run twice.** For example, re-running after a failed push. Expected: no new commit, no duplicate tag, and the push is retried. Test: Task 3 `test_sync_is_idempotent`.
3. **Rollback.** `publish.py --promote <older>`. Expected: the repo content returns to the older pack in a new commit, and the older tag stays where it was. Test: Task 3 `test_rollback_keeps_existing_tag`.
4. **Messy shipped rows.** CRLF line endings, uppercase letters, trailing spaces, duplicates after the mask is stripped, blank lines, `#` lines. Expected: output is unique, lowercase, sorted, and has no blank lines. Tests: Task 2 `test_strip_masks_normalises` and the same test in Task 5.
5. **A pack that adds or drops a file.** Expected: the public repo deletes stale files and adds new ones. K9 fails clearly if one of the two domain files it needs is missing from the manifest. Tests: Task 3 `test_write_tree_removes_stale_files` and Task 5 `test_load_pack_missing_entry`.

---

### Task 1: Licensing check (gate)

**Files:**
- Create: `~/Documents/blockerflow-lists/NOTICE`
- Create: `~/Documents/blockerflow-lists/LICENSE` (and `LICENSES/` if more than one license applies)

- [ ] **Step 1:** Check the current redistribution terms of each source behind the two lists:
  - Block List Project (`porn.txt`)
  - StevenBlack `extensions/porn/clefspeare13` and `extensions/porn/sinfonietta`
  - OISD NSFW (`nsfw.oisd.nl`)
  - HaGeZi NSFW
  - UT1 (`adult` keywords and URLs, `redirector`)

  The sources are defined in BlockerFlow `master:shared/blocklists/sync.py` and `build_categories.py`. Use each project's own license file or terms page.
- [ ] **Step 2:** If any source forbids redistribution or a derived public list, **stop**. Report to the user which source it is, the exact terms, and the options (drop that source from the public build, or don't publish). Do not continue to Task 4's push.
- [ ] **Step 3:** Otherwise write `LICENSE` and `NOTICE`:
  - `NOTICE` names each source, its license, its URL, and which public files derive from it.
  - Expected outcome: GPL-3.0 for `pornography/`, CC BY-SA 4.0 for `web-proxy/` and any UT1-derived keywords. Follow what the terms actually require.
- [ ] **Step 4:** Report the licensing decision to the user in one paragraph before Task 4.

### Task 2: Render the public repo tree (BlockerFlow, pure functions)

**Files:**
- Create: `shared/blocklists/public_repo.py`
- Test: `shared/blocklists/test_public_repo.py`

**Interfaces:**
- Produces:
  - `class PackError(Exception)`
  - `verify_manifest(manifest_text: str, public_key_pem: str) -> dict`
    - Returns the payload dict. Raises `PackError` on a bad signature or JSON, or on any file path that is absolute or contains `..`.
  - `strip_masks(data: bytes) -> list[str]`
  - `render_tree(manifest_text: str, payload: dict, files: dict[str, bytes]) -> dict[str, bytes]`
    - Keys are repo-relative paths. Raises `PackError` if any file's sha256 or size differs from the payload.

- [ ] **Step 1: Write the failing tests** (`unittest`). `setUpClass` generates a throwaway P-256 key with `openssl ecparam -name prime256v1 -genkey` and signs fixture payloads with it.
  - `test_verify_accepts_good_signature`: returns the payload.
  - `test_verify_rejects_tampered_payload`: raises `PackError`.
  - `test_verify_rejects_path_traversal`: a file path `../x` raises.
  - `test_strip_masks_normalises`: input `b"# h\r\nB.com|3\r\na.com|1\n\nb.com|8 \n"` gives `["a.com", "b.com"]`.
  - `test_render_tree_layout`: a pack with `pornography/domains.txt`, `pornography/keywords.txt`, `pornography/domains.len` and `web-proxy/domains.txt` renders exactly these keys:
    - `pornography/domains.txt`, `pornography/domains-with-sources.txt`, `pornography/hosts.txt`, `pornography/adblock.txt`, `pornography/keywords.txt`
    - `web-proxy/domains.txt`, `web-proxy/domains-with-sources.txt`, `web-proxy/hosts.txt`, `web-proxy/adblock.txt`
    - `manifest.json`

    There must be no `domains.len`. `domains-with-sources.txt` and `keywords.txt` must be byte-identical to the input, and `manifest.json` must equal `manifest_text`.
  - `test_render_formats`: for the input domains `a.com` and `b.com`:
    - `domains.txt` body lines are `a.com` and `b.com`.
    - `hosts.txt` body lines are `0.0.0.0 a.com` and `0.0.0.0 b.com`.
    - `adblock.txt` starts with `[Adblock Plus 2.0]` and its body lines are `||a.com^` and `||b.com^`.
    - Header lines start with `#` (`!` in adblock) and contain the version and `https://github.com/khaleel-git/blockerflow-lists`.
  - `test_render_refuses_bad_hash`: one file's bytes changed raises `PackError`.
- [ ] **Step 2:** Run `python3 -m unittest shared/blocklists/test_public_repo.py -v`. Expected: the tests fail (module or functions missing).
- [ ] **Step 3: Implement the three functions** in `public_repo.py`. Write the header text exactly as follows, using `!` instead of `#` in adblock files:
  ```
  # BlockerFlow lists: <category> (<plain domains|hosts file|AdBlock>)
  # Version <version>, built <created>
  # https://github.com/khaleel-git/blockerflow-lists (license: see NOTICE)
  ```
- [ ] **Step 4:** Run the tests again. Expected: all pass.
- [ ] **Step 5:** Commit: `blocklists: render the public lists repo from a signed pack`.

### Task 3: Fetch, write, commit (BlockerFlow), and hook into publish.py

**Files:**
- Modify: `shared/blocklists/public_repo.py`
- Modify: `shared/blocklists/publish.py` (`upload()` after the stable manifest upload, and `main()` for `--promote`)
- Test: `shared/blocklists/test_public_repo.py`

**Interfaces:**
- Consumes: Task 2 functions.
- Produces:
  - `fetch_pack(base_url: str, public_key_pem: str, channel: str = "stable") -> tuple[str, dict, dict[str, bytes]]`
    - Downloads, gunzips, and verifies everything before returning.
  - `write_tree(repo_dir: str, tree: dict[str, bytes]) -> bool`
    - Returns True if anything changed.
    - Deletes any top-level entry not in the tree, except `.git`, `.github`, `README.md`, `LICENSE`, `LICENSES`, `NOTICE`, `blockerflow_public_key.pem`.
  - `commit_and_tag(repo_dir: str, payload: dict, push: bool) -> str`
    - Returns `"committed"` or `"unchanged"`.
    - Commit message: `Lists <version> (stable seq <seq>)`.
    - Creates tag `v<version>` only if it doesn't exist.
    - Pushes the branch and tags when `push` is set, even if nothing changed.
  - `sync(repo_dir: str, *, base_url: str = BASE_URL, public_key_path: str = PUBLIC_KEY, dry_run: bool = False, push: bool = True) -> str`
  - CLI: `python3 shared/blocklists/public_repo.py [--repo DIR] [--dry-run] [--no-push] [--base-url URL]`
    - `--repo` defaults to `$BLOCKERFLOW_LISTS_REPO` or `~/Documents/blockerflow-lists`.
    - `--dry-run` writes the files and commits nothing.

- [ ] **Step 1: Write the failing tests.** Each test uses a temp git repo and a `file://` or `http.server` fixture serving a signed pack.
  - `test_sync_writes_and_tags`: tag `v20990101.1` exists afterwards, and the files match `render_tree`.
  - `test_sync_is_idempotent`: a second `sync` returns `"unchanged"`, the commit count stays the same, and there is one tag.
  - `test_rollback_keeps_existing_tag`:
    1. Sync pack A, then pack B, then A again.
    2. The tree equals A's tree, there are 3 commits, and `v<A>` still points at the first commit.
  - `test_write_tree_removes_stale_files`: a stale `piracy/domains.txt` is deleted, and `README.md` and `NOTICE` are kept.
  - `test_dry_run_commits_nothing`.
  - `test_fetch_failure_changes_nothing`: the server returns 500 for one pack file. `sync` raises `PackError`, and the working tree and HEAD are unchanged.
- [ ] **Step 2:** Run the tests. Expected: they fail.
- [ ] **Step 3: Implement** `fetch_pack`, `write_tree`, `commit_and_tag`, `sync` and the CLI. Use `urllib.request` with the UA from Global Constraints and a 60 s timeout. Run git with `subprocess`.
- [ ] **Step 4: Hook into `publish.py`.**
  - When to call it: after a successful non-dry-run upload where `channel == "stable"` (this covers `--promote`, since that calls `upload(..., "stable", ...)`).
  - What to call: `public_repo.sync(os.environ.get("BLOCKERFLOW_LISTS_REPO", "~/Documents/blockerflow-lists" expanded))`.
  - On failure: catch the exception and print `public repo is behind: <error>. Re-run: python3 shared/blocklists/public_repo.py`. Exit with status 0, because the R2 publish succeeded.
  - On a dry run: print `dry run: public repo not updated`.
- [ ] **Step 5:** Run the tests again. Expected: all pass. Then run `python3 shared/blocklists/publish.py --help`. Expected: no import error.
- [ ] **Step 6:** Commit: `blocklists: keep the public lists repo in step with stable`.

### Task 4: Create the public repo content (first version 20261105.8)

**Files:**
- Create in `~/Documents/blockerflow-lists/`:
  - `README.md`
  - the Task 1 `LICENSE` and `NOTICE`
  - the generated tree

- [ ] **Step 1:** `git init -b main ~/Documents/blockerflow-lists`, then commit `LICENSE` and `NOTICE`.
- [ ] **Step 2: Write `README.md`.** It covers:
  - What the lists are, with a link to https://www.blockerflow.net.
  - A table of every file and its format, including `keywords.txt` columns `keyword|weight|requiresBoundary` and that `weight >= 100` means high confidence.
  - Raw URLs of the form `https://raw.githubusercontent.com/khaleel-git/blockerflow-lists/main/<path>`, with instructions for Pi-hole (`domains.txt`), AdGuard and uBlock Origin (`adblock.txt`), and a hosts file (`hosts.txt`).
  - How to verify: download `manifest.json` and the public key, run the `openssl` command, compare the sha256 of `domains-with-sources.txt`.
  - Add the public key to the repo as `blockerflow_public_key.pem` (already on `write_tree`'s keep-list).
- [ ] **Step 3:** Run `python3 shared/blocklists/public_repo.py --no-push` from the BlockerFlow worktree. Expected: `committed`, tag `v20261105.8`, `pornography/domains.txt` with about 1.1 M lines, and `web-proxy/domains.txt` with about 19 k lines.
- [ ] **Step 4: Check that the files work in a real tool.**
  - `grep -c '^0.0.0.0 ' pornography/hosts.txt` equals the body line count of `domains.txt`.
  - `head` each file and confirm the header is correct.
- [ ] **Step 5: Ask the user** to approve creating the public `khaleel-git/blockerflow-lists` repo and pushing `main` plus tags. Only on a yes:
  ```bash
  gh repo create khaleel-git/blockerflow-lists --public --source ~/Documents/blockerflow-lists --push
  git -C ~/Documents/blockerflow-lists push --tags
  ```

### Task 5: K9 fetch script

**Files:**
- Create: `lists/fetch_blockerflow.py`
- Create: `lists/blockerflow_public_key.pem` (copy of BlockerFlow `master:android/app/src/main/res/raw/lists_public_key.pem`)
- Create: `lists/blockerflow-lists.version` (content: `20261105.8` followed by a newline)
- Test: `lists/tests/test_fetch_blockerflow.py`

**Interfaces:**
- Produces:
  - `class PackError(Exception)`
  - `verify_manifest(manifest_text: str, public_key_pem: str) -> dict`: same behaviour as Task 2.
  - `strip_masks(data: bytes) -> list[str]`: same behaviour as Task 2.
  - `MAPPING = {"pornography/domains.txt": ("pornography/domains-with-sources.txt", "pornography"), "web-proxy/domains.txt": ("web-proxy/domains-with-sources.txt", "proxy-avoidance")}`
    - Format: manifest path → (public repo path, K9 category).
  - `load_pack(version: str, read: Callable[[str], bytes], public_key_pem: str) -> dict[str, list[str]]`
    - `read` takes a repo path and returns its bytes at tag `v<version>`.
    - Returns K9 category → domains.
    - Raises `PackError` on a bad signature, on payload `version != version`, on a manifest entry from `MAPPING` that is missing, or on a sha256 mismatch.
  - `write_categories(version: str, domains: dict[str, list[str]]) -> None`
    - Writes `lists/categories/<cat>/domains.txt` with the header `# From blockerflow-lists v<version>. Do not edit; run lists/fetch_blockerflow.py`, then one domain per line.
  - `check_categories(version: str, domains: dict[str, list[str]]) -> list[str]`
    - Returns the names of categories whose committed file differs.
  - CLI: `python3 lists/fetch_blockerflow.py [--latest] [--check] [--no-build] [--from-dir PATH]`
    - Default: read the pin, fetch from `https://raw.githubusercontent.com/khaleel-git/blockerflow-lists/v<version>/<path>`, write, then run `build.py`.
    - `--latest`: look up the tags at `https://api.github.com/repos/khaleel-git/blockerflow-lists/tags`, take the highest `vYYYYMMDD.N` comparing `(int date, int n)`, and rewrite the pin.
    - `--check`: write nothing. Exit 1 and print the category names if they differ.
    - `--from-dir`: read from a local clone with `git -C PATH show v<version>:<path>`, so the result is the tagged version and not whatever is checked out (used before the repo is public). Tests pass their own `read` callable.

- [ ] **Step 1: Write the failing tests.** Use a fixture directory with a pack signed by a throwaway key, same approach as Task 2.
  - `test_load_pack_maps_categories`: returns the keys `pornography` and `proxy-avoidance` with stripped, sorted domains.
  - `test_load_pack_hash_mismatch_writes_nothing`: one changed byte raises `PackError`, and the category files are unchanged.
  - `test_load_pack_missing_entry`: a manifest without `web-proxy/domains.txt` raises `PackError` naming the path.
  - `test_load_pack_version_mismatch`: raises.
  - `test_strip_masks_normalises`: same input and expected output as in Task 2.
  - `test_check_detects_hand_edit`: write the categories, append a line to one file, and `check_categories` returns that category.

  Tests point the category output directory at a temp dir, through a module-level `CAT_DIR` the tests can override.
- [ ] **Step 2:** Run `python3 -m unittest discover -s lists/tests -v`. Expected: the tests fail.
- [ ] **Step 3:** Implement the script.
- [ ] **Step 4:** Run the tests again. Expected: all pass.
- [ ] **Step 5:** Commit: `lists: fetch pornography and proxy lists from blockerflow-lists`.

### Task 6: One database for both apps

**Files:**
- Modify: `lists/build.py`
  - `DB_DIR` becomes `DB_DIRS = [<mac>/app/internal/database, <windows>/app/internal/database]`.
  - `build()` writes all four files into each directory.
  - Update the docstring, which wrongly says `{ "a": [...] }`.
- Modify: `lists/sync.py`
  - Remove the `pornography` domain sources (keep its `urls` and `keywords` sources).
  - Remove the whole `proxy-avoidance` entry.
  - In `sync_category`, print the "all domain sources failed" warning only when the category has domain sources configured.
- Modify: `lists/categories/pornography/urls.txt`: append the Windows-only URL entries under the comment `# ── From the old Windows urls.json (Reddit, Twitter, 4chan, …) ──`.
- Modify: `lists/categories/p2p/urls.txt` (create it if missing): add `iptorrents.com/torrents`.
- Modify: `mac/app/internal/database/database.go` and `windows/app/internal/database/database.go`, identically: add `func (d *Database) CategoryCount(cat string) int`.
- Create: `mac/app/internal/database/database_test.go` and `windows/app/internal/database/database_test.go`, identical.
- Regenerate: `{mac,windows}/app/internal/database/*.json` and `lists/categories/{pornography,proxy-avoidance}/domains.txt`.

**Interfaces:**
- Produces: `CategoryCount(cat string) int`, used by Task 7.

- [ ] **Step 1: Write the failing Go test** `TestEmbeddedBlockerFlowListSizes`:
  - `DB.CategoryCount("pornography")` is between 800000 and 3000000.
  - `DB.CategoryCount("proxy-avoidance")` is between 15000 and 100000.
  - Also `TestDomainsJSONIsCategoryKeyed`: the embedded `domains.json` has a `pornography` key and no single-letter keys.
- [ ] **Step 2:** Run `cd windows/app && go test ./internal/database/`. Expected: FAIL (`CategoryCount` is undefined).
- [ ] **Step 3: Move the Windows-only URL entries.** Do this before regenerating, while `windows/app/internal/database/urls.json` still holds them. Write a throwaway one-off script, kept out of the repo. It reads every value of that JSON, skips entries already in `pornography/urls.txt`, sends `iptorrents.com/torrents` to `p2p/urls.txt`, and appends all other entries to `pornography/urls.txt` in their original order. About 4,275 entries land in `urls.txt`. Make the `build.py` and `sync.py` edits and add `CategoryCount`.
- [ ] **Step 4: Regenerate.** Run `python3 lists/fetch_blockerflow.py --from-dir ~/Documents/blockerflow-lists`, which fetches, then runs `build.py`. Then check:
  - `cmp` of each of the four JSON files, mac against windows, is identical.
  - `python3 lists/fetch_blockerflow.py --from-dir ~/Documents/blockerflow-lists --check` exits 0.
- [ ] **Step 5:** Run `cd windows/app && go test -mod=mod ./...` and `cd mac/app && go test ./...`. Expected: PASS on both.
- [ ] **Step 6:** Commit: `lists: both apps embed BlockerFlow's lists and one shared database`.

### Task 7: Windows level behaviour tests

**Files:**
- Create: `windows/app/internal/proxy/levels_test.go`

**Interfaces:**
- Consumes: `database.DB.CategoryCount`, `(*Proxy).decide(pol *Policy, host string, r *http.Request) verdict`, `LevelCategories`, `IsBuiltinAllowed`, `IsCommunicationAllowed`, `isWebProxy`.

- [ ] **Step 1: Write a test helper and the tests.**
  - `sampleDomain(t, cat string, avoid []string) string` reads `../database/domains.json`. It returns the first domain of `cat` that:
    - is not `IsBuiltinAllowed`, `IsCommunicationAllowed` or `isWebProxy`;
    - is not a direct IP;
    - is not in any category of `avoid` (check with `database.DB.BlocksDomainInCategories`).

    It returns `""` if there is none. Requests are `CONNECT host:443`, so URL checks don't run.
  - `TestNamedLevelsBlockEachOfTheirCategories`: for `high`, `default`, `moderate` and `minimal`, for every category in `LevelCategories[level]` except `unrated` (empty today), `decide` with `Policy{FilterLevel: level}` returns `verdictBlock` for `sampleDomain(t, cat, nil)`. A `""` sample fails the test, naming the category.
  - `TestMinimalLeavesGamblingAlone`: `d := sampleDomain(t, "gambling", LevelCategories[LevelMinimal])` passes at `minimal` and is blocked at `moderate`.
  - `TestMonitorBlocksNoDatabaseDomain`: a pornography sample passes at `monitor`.
- [ ] **Step 2:** Run `cd windows/app && go test -mod=mod ./internal/proxy/ -run 'Levels|Minimal|Monitor' -v`. Expected: PASS. Then run the same tests against `master`'s `domains.json` (`git stash` or a scratch worktree) to confirm they FAIL there, which shows they catch the old bug. Report both results.
- [ ] **Step 3:** Commit: `windows: test that each filter level blocks its categories`.

### Task 8: CI, docs, release note

**Files:**
- Create: `.github/workflows/lists-check.yml`
  - Runs on `ubuntu-latest`, on pushes and pull requests touching `lists/**`, `mac/app/internal/database/**` or `windows/app/internal/database/**`.
  - Step 1: `python3 -m unittest discover -s lists/tests`.
  - Step 2: `python3 lists/fetch_blockerflow.py --check`.
  - Step 3: `cmp` of the four JSON files, mac against windows.
  - Then `cd mac/app && go test ./internal/database/`. That package has no darwin-only imports; confirm first, and drop this step if it does.
- Modify: `lists/README.md`: where `pornography` and `proxy-avoidance` come from now, how to bump the pin (`--latest`), and that `sync.py` no longer touches them.
- Modify: `README.md`: one short paragraph saying the pornography and proxy lists come from BlockerFlow's public lists, with a link.
- Modify: `RELEASE.md`: add a section at the top with this content:
  > **Windows:** your filter level now blocks every category it lists (before, only Custom with adult blocking used the built-in domain database). Pornography and web-proxy lists now come from BlockerFlow's tested lists. The proxy-avoidance category no longer includes DNS-over-HTTPS resolvers and VPN hosts.

- [ ] **Step 1:** Write the workflow and docs.
- [ ] **Step 2:** Validate the workflow with `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/lists-check.yml'))"`, or `actionlint` if it's installed. The `--check` step can only pass once Task 4 is pushed; until then, note this in the PR description.
- [ ] **Step 3:** Commit: `lists: CI check for the pinned BlockerFlow lists; docs and release note`.

### Task 9: Final verification

- [ ] Run every test suite and record the outputs:
  - BlockerFlow: `python3 -m unittest shared/blocklists/test_public_repo.py`
  - K9: `python3 -m unittest discover -s lists/tests`
  - Windows: `cd windows/app && go test -mod=mod -count=1 ./...`
  - Mac: `cd mac/app && go test ./...`
- [ ] Build both apps: `cd mac/app && go build ./...` and `GOOS=windows go build -mod=mod ./...` in `windows/app`. The frontend `dist` must exist; run `npm ci && npm run build` in each `frontend/` first if it's missing.
- [ ] Report the Windows binary size before and after.
- [ ] Ask the user before pushing either branch or opening PRs.
