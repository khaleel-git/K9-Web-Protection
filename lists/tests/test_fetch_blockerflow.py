import base64
import hashlib
import http.client
import json
import os
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
import fetch_blockerflow as fb  # noqa: E402

VERSION = "20261105.8"
PORN = (b"# header\n"
        b"B.example|3\n"
        b"a.example|1\n"
        b"\n"
        b"A.EXAMPLE|2\n"
        b"c.example\n")
PROXY = b"# header\nproxy.example|1\nvpn.example|2\n"


def _openssl(*args, **kw):
    subprocess.run(["openssl", *args], check=True, capture_output=True, **kw)


class Fixture:
    """A pack signed by a throwaway key, served from memory."""

    def __init__(self, tmp, version=VERSION, drop=None, tweak=None):
        key = os.path.join(tmp, "key.pem")
        pub = os.path.join(tmp, "pub.pem")
        _openssl("ecparam", "-name", "prime256v1", "-genkey", "-noout", "-out", key)
        _openssl("ec", "-in", key, "-pubout", "-out", pub)
        with open(pub) as f:
            self.public_key = f.read()
        shipped = {"pornography/domains.txt": PORN, "web-proxy/domains.txt": PROXY}
        self.repo = {
            "pornography/domains-with-sources.txt": PORN,
            "web-proxy/domains-with-sources.txt": PROXY,
        }
        files = [{"path": p, "sha256": hashlib.sha256(b).hexdigest(), "size": len(b)}
                 for p, b in shipped.items() if p != drop]
        payload = json.dumps({"version": version, "created": "2026-11-05T00:00:00Z", "seq": 8,
                              "files": files})
        pay = os.path.join(tmp, "payload")
        sig = os.path.join(tmp, "sig")
        with open(pay, "w") as f:
            f.write(payload)
        _openssl("dgst", "-sha256", "-sign", key, "-out", sig, pay)
        with open(sig, "rb") as f:
            signature = base64.b64encode(f.read()).decode()
        self.repo["manifest.json"] = json.dumps({"payload": payload, "signature": signature}).encode()
        if tweak:
            tweak(self.repo)

    def read(self, path):
        return self.repo[path]


class TestFetchBlockerflow(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = self._tmp.name
        self.addCleanup(self._tmp.cleanup)
        self.cat_dir = os.path.join(self.tmp, "categories")
        old = fb.CAT_DIR
        fb.CAT_DIR = self.cat_dir
        self.addCleanup(setattr, fb, "CAT_DIR", old)

    def fixture(self, **kw):
        return Fixture(self.tmp, **kw)

    def test_load_pack_maps_categories(self):
        fx = self.fixture()
        got = fb.load_pack(VERSION, fx.read, fx.public_key)
        self.assertEqual(set(got), {"pornography", "proxy-avoidance"})
        self.assertEqual(got["pornography"], ["a.example", "b.example", "c.example"])
        self.assertEqual(got["proxy-avoidance"], ["proxy.example", "vpn.example"])

    def test_load_pack_hash_mismatch_writes_nothing(self):
        fx = self.fixture(tweak=lambda r: r.update({"web-proxy/domains-with-sources.txt": PROXY + b"x"}))
        with self.assertRaises(fb.PackError):
            fb.load_pack(VERSION, fx.read, fx.public_key)
        self.assertFalse(os.path.exists(self.cat_dir))

    def test_load_pack_bad_signature(self):
        fx = self.fixture()
        other = Fixture(tempfile.mkdtemp(dir=self.tmp))
        with self.assertRaises(fb.PackError):
            fb.load_pack(VERSION, fx.read, other.public_key)

    def test_load_pack_missing_entry(self):
        fx = self.fixture(drop="web-proxy/domains.txt")
        with self.assertRaises(fb.PackError) as cm:
            fb.load_pack(VERSION, fx.read, fx.public_key)
        self.assertIn("web-proxy/domains.txt", str(cm.exception))

    def test_load_pack_version_mismatch(self):
        fx = self.fixture(version="20261105.7")
        with self.assertRaises(fb.PackError):
            fb.load_pack(VERSION, fx.read, fx.public_key)

    def test_strip_masks_normalises(self):
        self.assertEqual(fb.strip_masks(PORN), ["a.example", "b.example", "c.example"])
        self.assertEqual(fb.strip_masks(b"# only\n\n  \n"), [])

    def test_strip_masks_rejects_bad_utf8(self):
        with self.assertRaises(fb.PackError):
            fb.strip_masks(b"\xff\xfe\n")

    def test_write_categories_header_and_lines(self):
        fb.write_categories(VERSION, {"pornography": ["a.example", "b.example"], "proxy-avoidance": ["p.example"]})
        with open(os.path.join(self.cat_dir, "pornography", "domains.txt")) as f:
            self.assertEqual(f.read(), "# From blockerflow-lists v20261105.8 (GPL-3.0, see "
                                       "lists/blockerflow-lists/LICENSE). Do not edit; "
                                       "run lists/fetch_blockerflow.py\na.example\nb.example\n")
        with open(os.path.join(self.cat_dir, "proxy-avoidance", "domains.txt")) as f:
            self.assertEqual(f.read(), "# From blockerflow-lists v20261105.8 (CC BY-SA 4.0, see "
                                       "lists/blockerflow-lists/CC-BY-SA-4.0.txt). Do not edit; "
                                       "run lists/fetch_blockerflow.py\np.example\n")

    def test_check_detects_a_changed_licence_header(self):
        domains = {"pornography": ["a.example"], "proxy-avoidance": ["p.example"]}
        fb.write_categories(VERSION, domains)
        path = os.path.join(self.cat_dir, "pornography", "domains.txt")
        with open(path) as f:
            text = f.read()
        with open(path, "w") as f:
            f.write(text.replace("GPL-3.0", "MIT"))
        self.assertEqual(fb.check_categories(VERSION, domains), ["pornography"])

    def test_check_detects_hand_edit(self):
        fx = self.fixture()
        domains = fb.load_pack(VERSION, fx.read, fx.public_key)
        fb.write_categories(VERSION, domains)
        self.assertEqual(fb.check_categories(VERSION, domains), [])
        with open(os.path.join(self.cat_dir, "proxy-avoidance", "domains.txt"), "a") as f:
            f.write("hand.edit\n")
        self.assertEqual(fb.check_categories(VERSION, domains), ["proxy-avoidance"])

    def test_check_reports_missing_file(self):
        domains = {"pornography": ["a.example"], "proxy-avoidance": ["p.example"]}
        self.assertEqual(sorted(fb.check_categories(VERSION, domains)), ["pornography", "proxy-avoidance"])

    def test_latest_version_is_the_signed_manifest_on_main(self):
        # main holds the rolled-back version A; a newer tag B may exist, and must not matter.
        a = self.fixture(version="20261105.8")
        asked = []

        def read(path):
            asked.append(path)
            return a.read(path)
        self.assertEqual(fb.latest_version(a.public_key, read), "20261105.8")
        self.assertEqual(asked, ["manifest.json"])

    def test_latest_version_never_asks_for_tags(self):
        self.assertFalse(hasattr(fb, "TAGS_URL"))
        self.assertFalse(hasattr(fb, "TAG_RE"))

    def test_latest_version_bad_signature_fails(self):
        a = self.fixture()
        other = Fixture(tempfile.mkdtemp(dir=self.tmp))
        with self.assertRaises(fb.PackError):
            fb.latest_version(other.public_key, a.read)

    def test_latest_version_missing_manifest_fails(self):
        def read(path):
            raise fb.PackError("HTTP 404")
        with self.assertRaises(fb.PackError):
            fb.latest_version(self.fixture().public_key, read)

    def test_main_latest_reads_main_and_not_tags(self):
        # The manifest on main says A; the tag for A is fetched; the pin becomes A.
        a = self.fixture(version="20261105.8")
        pin = self._patch_main(a, pin_version="20250101.1")
        refs = []

        def make_reader(from_dir, ref):
            refs.append(ref)
            return a.read
        fb._make_reader = make_reader
        self.assertEqual(fb.main(["--latest", "--no-build"]), 0)
        self.assertEqual(refs, ["main", "v20261105.8"])
        with open(pin) as f:
            self.assertEqual(f.read(), "20261105.8\n")

    def test_main_latest_bad_signature_on_main_writes_nothing(self):
        good = self.fixture()
        evil = Fixture(tempfile.mkdtemp(dir=self.tmp), version="20261106.1")  # another key
        pin = self._patch_main(good, pin_version="20250101.1")
        fb._make_reader = lambda from_dir, ref: (evil.read if ref == "main" else good.read)
        self.assertEqual(fb.main(["--latest", "--no-build"]), 2)
        self.assertFalse(os.path.exists(self.cat_dir))
        with open(pin) as f:
            self.assertEqual(f.read(), "20250101.1\n")

    def test_main_latest_tag_that_fails_leaves_pin_alone(self):
        a = self.fixture(version="20261105.8")
        broken = self.fixture(tweak=lambda r: r.update({"web-proxy/domains-with-sources.txt": PROXY + b"x"}))
        pin = self._patch_main(a, pin_version="20250101.1")
        fb._make_reader = lambda from_dir, ref: (a.read if ref == "main" else broken.read)
        self.assertEqual(fb.main(["--latest", "--no-build"]), 2)
        self.assertFalse(os.path.exists(self.cat_dir))
        with open(pin) as f:
            self.assertEqual(f.read(), "20250101.1\n")

    def test_http_get_wraps_incomplete_read(self):
        class Resp:
            def __enter__(self):
                return self

            def __exit__(self, *a):
                return False

            def read(self):
                raise http.client.IncompleteRead(b"par", 10)
        with mock.patch.object(fb.urllib.request, "urlopen", lambda *a, **k: Resp()):
            with self.assertRaises(fb.PackError):
                fb._http_get("https://example.invalid/x")

    def test_main_short_read_exits_with_the_pack_error_code(self):
        self._patch_main(self.fixture())

        def short(path):
            raise fb.PackError("https://example.invalid/x: IncompleteRead(3 bytes read, 7 more expected)")
        fb._make_reader = lambda from_dir, ref: short
        self.assertEqual(fb.main(["--no-build"]), 2)

    def _patch_main(self, fx, pin_version=VERSION):
        pin = os.path.join(self.tmp, "pin")
        keyfile = os.path.join(self.tmp, "k.pem")
        with open(pin, "w") as f:
            f.write(pin_version + "\n")
        with open(keyfile, "w") as f:
            f.write(fx.public_key)
        for name, value in (("PIN_PATH", pin), ("KEY_PATH", keyfile),
                            ("_make_reader", lambda from_dir, ref: fx.read)):
            self.addCleanup(setattr, fb, name, getattr(fb, name))
            setattr(fb, name, value)
        return pin

    def test_main_check_exit_codes(self):
        self._patch_main(self.fixture())
        self.assertEqual(fb.main(["--check"]), 1)  # nothing written yet: differs
        self.assertFalse(os.path.exists(self.cat_dir))
        self.assertEqual(fb.main(["--no-build"]), 0)
        self.assertEqual(fb.main(["--check"]), 0)

    def test_main_failed_check_writes_nothing(self):
        fx = self.fixture(tweak=lambda r: r.update({"pornography/domains-with-sources.txt": PORN + b"x|1\n"}))
        pin = self._patch_main(fx, pin_version="20250101.1")
        self.assertEqual(fb.main(["--latest", "--no-build"]), 2)
        self.assertFalse(os.path.exists(self.cat_dir))
        with open(pin) as f:
            self.assertEqual(f.read(), "20250101.1\n")

    def test_main_latest_rewrites_pin(self):
        pin = self._patch_main(self.fixture(), pin_version="20250101.1")
        self.assertEqual(fb.main(["--latest", "--no-build"]), 0)
        with open(pin) as f:
            self.assertEqual(f.read(), VERSION + "\n")


if __name__ == "__main__":
    unittest.main()
