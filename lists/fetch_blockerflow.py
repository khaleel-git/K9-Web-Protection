#!/usr/bin/env python3
"""
K9 Web Protection - fetch the pornography and proxy-avoidance lists from BlockerFlow.

K9 no longer maintains these two lists itself. It embeds the tested lists of a pinned version
of the public repo khaleel-git/blockerflow-lists (pin: lists/blockerflow-lists.version). The
repo's signed manifest.json is checked against lists/blockerflow_public_key.pem, and every
shipped file against its signed size and sha256, before anything is written.

Usage:
  python3 lists/fetch_blockerflow.py                  # fetch the pinned version, write, run build.py
  python3 lists/fetch_blockerflow.py --latest         # use the version signed on main and update the pin
  python3 lists/fetch_blockerflow.py --check          # write nothing; exit 1 if the files differ
  python3 lists/fetch_blockerflow.py --no-build       # skip build.py
  python3 lists/fetch_blockerflow.py --from-dir PATH  # read the tag (and main, for --latest) from a local clone (git show)
"""

import argparse
import base64
import hashlib
import http.client
import json
import os
import re
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
CAT_DIR = os.path.join(BASE_DIR, "categories")
PIN_PATH = os.path.join(BASE_DIR, "blockerflow-lists.version")
KEY_PATH = os.path.join(BASE_DIR, "blockerflow_public_key.pem")

REPO = "khaleel-git/blockerflow-lists"
RAW_URL = "https://raw.githubusercontent.com/" + REPO + "/{ref}/{path}"
USER_AGENT = "k9-blockerflow-lists/1.0"
HTTP_TIMEOUT = 60
OPENSSL_TIMEOUT = 30
VERSION_RE = re.compile(r"[0-9A-Za-z][0-9A-Za-z._-]*")

# manifest path -> (path in the public repo, K9 category). The manifest names the shipped files
# (domain|source_mask rows); in the public repo those are domains-with-sources.txt.
MAPPING = {
    "pornography/domains.txt": ("pornography/domains-with-sources.txt", "pornography"),
    "web-proxy/domains.txt": ("web-proxy/domains-with-sources.txt", "proxy-avoidance"),
}

# K9 category -> (licence name, licence text file in lists/blockerflow-lists/). Named in the header of
# each generated category file, because those files carry the licensed data.
LICENCES = {
    "pornography": ("GPL-3.0", "LICENSE"),
    "proxy-avoidance": ("CC BY-SA 4.0", "CC-BY-SA-4.0.txt"),
}


class PackError(Exception):
    pass


def _bad_path(path):
    return (not isinstance(path, str) or not path or path.startswith("/") or "\\" in path
            or ".." in path.split("/") or path == "manifest.json")


def _check_payload(payload):
    for field in ("version", "created"):
        if not isinstance(payload.get(field), str) or not payload[field]:
            raise PackError(f"malformed manifest: payload has no {field}")
    if not VERSION_RE.fullmatch(payload["version"]):
        raise PackError(f"malformed manifest: unsafe version {payload['version']!r}")
    if not isinstance(payload.get("seq"), int) or isinstance(payload["seq"], bool):
        raise PackError("malformed manifest: payload has no seq")
    for f in payload["files"]:
        if not isinstance(f, dict) or _bad_path(f.get("path")):
            raise PackError(f"unsafe file path in manifest: {f.get('path') if isinstance(f, dict) else f!r}")
        if (not isinstance(f.get("size"), int) or isinstance(f["size"], bool) or f["size"] < 0
                or not isinstance(f.get("sha256"), str)):
            raise PackError(f"malformed manifest: {f['path']} needs an integer size and a sha256")


# Duplicated in blockerflow/shared/blocklists/public_repo.py (verify_manifest); keep both in step.
def verify_manifest(manifest_text, public_key_pem):
    """Check the manifest's ECDSA P-256/SHA-256 signature and return the payload dict."""
    try:
        outer = json.loads(manifest_text)
        payload_text = outer["payload"]
        signature = base64.b64decode(outer["signature"], validate=True)
        payload = json.loads(payload_text)
        files = payload["files"]
        if not isinstance(payload, dict) or not isinstance(files, list):
            raise ValueError("payload is not an object with a file list")
    except (ValueError, KeyError, TypeError) as e:
        raise PackError(f"malformed manifest: {e}")
    with tempfile.TemporaryDirectory() as d:
        paths = {name: os.path.join(d, name) for name in ("key.pem", "sig", "payload")}
        for name, data in (("key.pem", public_key_pem.encode()), ("sig", signature),
                           ("payload", payload_text.encode())):
            with open(paths[name], "wb") as f:
                f.write(data)
        try:
            r = subprocess.run(["openssl", "dgst", "-sha256", "-verify", paths["key.pem"],
                                "-signature", paths["sig"], paths["payload"]], capture_output=True,
                               timeout=OPENSSL_TIMEOUT)
        except FileNotFoundError:
            raise PackError("openssl is not installed, so the manifest signature cannot be checked")
        except subprocess.TimeoutExpired:
            raise PackError("openssl timed out checking the manifest signature")
        if r.returncode != 0:
            raise PackError("manifest signature does not verify")
    _check_payload(payload)
    return payload


# Duplicated in blockerflow/shared/blocklists/public_repo.py (strip_masks); keep both in step.
def strip_masks(data):
    """`domain|source_mask` rows to a sorted, de-duplicated, lowercase list of domains."""
    domains = set()
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError:
        raise PackError("domain file is not valid UTF-8")
    for line in text.splitlines():
        line = line.strip()
        if line and not line.startswith("#"):
            domain = line.split("|", 1)[0].strip().lower()
            if domain:
                domains.add(domain)
    return sorted(domains)


def load_pack(version, read, public_key_pem):
    """Verify the pack at tag v<version> and return {K9 category: domains}. `read(repo_path)` gives
    the bytes of a file at that tag. Raises PackError on any failed check; nothing is written."""
    try:
        manifest_text = read("manifest.json").decode("utf-8")
    except UnicodeDecodeError:
        raise PackError("malformed manifest: not valid UTF-8")
    payload = verify_manifest(manifest_text, public_key_pem)
    if payload["version"] != version:
        raise PackError(f"manifest is version {payload['version']}, expected {version}")
    entries = {f["path"]: f for f in payload["files"]}
    domains = {}
    for manifest_path, (repo_path, category) in MAPPING.items():
        entry = entries.get(manifest_path)
        if entry is None:
            raise PackError(f"manifest has no entry for {manifest_path}")
        data = read(repo_path)
        if len(data) != entry["size"] or hashlib.sha256(data).hexdigest() != entry["sha256"]:
            raise PackError(f"{repo_path} does not match the manifest (size or sha256)")
        try:
            domains[category] = strip_masks(data)
        except PackError as e:
            raise PackError(f"{repo_path}: {e}")
        if not domains[category]:
            raise PackError(f"{repo_path} has no domains")
    return domains


def _header(version, category):
    name, text = LICENCES[category]
    return (f"# From blockerflow-lists v{version} ({name}, see lists/blockerflow-lists/{text}). "
            "Do not edit; run lists/fetch_blockerflow.py\n")


def _render(version, category, category_domains):
    return (_header(version, category) + "".join(d + "\n" for d in category_domains)).encode("utf-8")


def _cat_file(category):
    return os.path.join(CAT_DIR, category, "domains.txt")


def write_categories(version, domains):
    for category, items in domains.items():
        dest = _cat_file(category)
        os.makedirs(os.path.dirname(dest), exist_ok=True)
        tmp = dest + ".tmp"
        with open(tmp, "wb") as f:
            f.write(_render(version, category, items))
        os.replace(tmp, dest)


def check_categories(version, domains):
    """Names of the categories whose committed file differs from what the pack gives."""
    differs = []
    for category, items in domains.items():
        try:
            with open(_cat_file(category), "rb") as f:
                same = f.read() == _render(version, category, items)
        except OSError:
            same = False
        if not same:
            differs.append(category)
    return differs


# ── sources ───────────────────────────────────────────────────────────────────

def _http_get(url):
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    try:
        with urllib.request.urlopen(req, timeout=HTTP_TIMEOUT) as r:
            return r.read()
    except urllib.error.HTTPError as e:
        raise PackError(f"{url}: HTTP {e.code}")
    except urllib.error.URLError as e:
        raise PackError(f"{url}: {e.reason}")
    except http.client.HTTPException as e:  # IncompleteRead (a short read) and friends
        raise PackError(f"{url}: {e!r}")
    except OSError as e:
        raise PackError(f"{url}: {e}")


def latest_version(public_key_pem, read=None):
    """The version signed in main's manifest.json: what the public repo currently publishes as
    stable. Tags are not used: a rollback leaves the newer tag in place but moves main back.
    `read(repo_path)` gives the bytes of a file on main."""
    read = read or _make_reader(None, "main")
    try:
        manifest_text = read("manifest.json").decode("utf-8")
    except UnicodeDecodeError:
        raise PackError("malformed manifest on main: not valid UTF-8")
    return verify_manifest(manifest_text, public_key_pem)["version"]


def _make_reader(from_dir, ref):
    """Reader of repo files at `ref` ('main' or 'v<version>'): git show in a clone, or raw.githubusercontent."""
    if from_dir:
        clone = os.path.abspath(os.path.expanduser(from_dir))

        def read(path):
            try:
                r = subprocess.run(["git", "-C", clone, "show", f"{ref}:{path}"],
                                   capture_output=True, timeout=120)
            except FileNotFoundError:
                raise PackError("git is not installed")
            except subprocess.TimeoutExpired:
                raise PackError(f"git show {path} timed out")
            if r.returncode != 0:
                raise PackError(f"git show {ref}:{path} failed: {r.stderr.decode(errors='replace').strip()}")
            return r.stdout
        return read
    return lambda path: _http_get(RAW_URL.format(ref=ref, path=path))


def _read_pin():
    try:
        with open(PIN_PATH) as f:
            version = f.read().strip()
    except OSError as e:
        raise PackError(f"cannot read the version pin: {e}")
    if not VERSION_RE.fullmatch(version):
        raise PackError(f"unsafe version in {PIN_PATH}: {version!r}")
    return version


def main(argv=None):
    ap = argparse.ArgumentParser(description="Fetch K9's pornography and proxy-avoidance lists from blockerflow-lists.")
    ap.add_argument("--latest", action="store_true", help="use the version signed on main and update the pin")
    ap.add_argument("--check", action="store_true", help="write nothing; exit 1 if the category files differ")
    ap.add_argument("--no-build", action="store_true", help="do not run build.py afterwards")
    ap.add_argument("--from-dir", metavar="PATH", help="read the tag from a local clone with git show")
    args = ap.parse_args(argv)
    try:
        pinned = _read_pin()
        try:
            with open(KEY_PATH) as f:
                key = f.read()
        except OSError as e:
            raise PackError(f"cannot read the public key: {e}")
        version = latest_version(key, _make_reader(args.from_dir, "main")) if args.latest else pinned
        # The pin is only rewritten below, after this tag has loaded and verified in full.
        domains = load_pack(version, _make_reader(args.from_dir, f"v{version}"), key)
        counts = ", ".join(f"{c}: {len(d)}" for c, d in sorted(domains.items()))
        if args.check:
            differs = check_categories(version, domains)
            if differs:
                print(f"differs from blockerflow-lists v{version}: {', '.join(differs)}")
                return 1
            print(f"up to date with blockerflow-lists v{version} ({counts})")
            return 0
        write_categories(version, domains)
        if version != pinned:
            with open(PIN_PATH, "w") as f:
                f.write(version + "\n")
        print(f"wrote blockerflow-lists v{version} ({counts})")
    except PackError as e:
        print(f"error: {e}", file=sys.stderr)
        return 2
    if not args.no_build:
        subprocess.run([sys.executable, os.path.join(BASE_DIR, "build.py"), "--level", "default"], check=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())
