# K10 Web Protection — Feature Plan

**Status legend:** ✅ Done · 🔧 Frontend only (backend needed) · ❌ Not started

---

## Setup Sidebar — All 9 Items

1. Web Categories to Block ✅
2. Time Restrictions ✅ (per-day schedule, auto-enables Focus Mode)
3. Web Site Exceptions ✅
4. Blocking Effects 🔧 (custom message wired; styled block page already in proxy)
5. URL Keywords ✅ (HTTP + HTTPS)
6. Safe Search ✅
7. Advanced ✅ (proxy port + autostart)
8. Password/Email ✅
9. K10 Update 🔧 (shows DB sizes; Check Now is a stub)

---

## Distribution / Installer ✅

`.pkg` installer produced by `build-pkg.sh`. Double-click install for end users.
- `pkg-scripts/preinstall` — stops services and unlocks files before upgrade
- `pkg-scripts/postinstall` — signs app, registers LaunchAgents, wires PF rules, locks files
- `pkg-resources/distribution.xml` + `welcome.html` — macOS Installer UI
- Unsigned (ad-hoc codesign). For Gatekeeper-free distribution: sign with Developer ID Installer + notarize.

---

## QUIC / HTTP3 Bypass Blocking ✅

Chrome falls back to QUIC (HTTP/3 over UDP 443) when its TCP CONNECT is blocked, bypassing the proxy entirely. Fixed with a PF anchor rule:
- `/etc/pf.anchors/k10webprotection`: `block drop out quick proto udp to any port 443`
- Anchor wired into `/etc/pf.conf` and reloaded at install time via `pfctl -f /etc/pf.conf`
- Watchdog re-applies the anchor every 10s if it gets flushed
- `quick` keyword ensures the rule fires before any later `pass` rules

---

## Focus Mode ✅

Separate top-nav tab. Blocks active social-media sites. User can toggle each site on/off, add custom sites, and set a countdown timer. Also triggers automatically during Time Restriction windows.

---

## Top Blocked Categories Chart ✅

Category is now stored at block time via `database.CategoryFor()` lookup — no more regex guessing. All 29 database categories have display names (Pornography, Adult, Nudity, Alt. Sexuality, Gambling, Malware, P2P / Torrents, Proxy Bypass, Violence / Hate, Extreme Content, Illegal Drugs, Dating, Hacking, Phishing, Suspicious, Alcohol, Tobacco, Weapons, Abortion, Messaging, Social Media, Forums, Image Search, Alt. Spirituality, LGBT, Intimate Apparel, Sex Education, Personal Pages, Unrated).

---

## Recent Activity Tagging ✅

Activity rows use the stored `entry.category` field (same database lookup), with `domainToCategory()` regex as fallback for older entries.

---

## Time Restrictions ✅

Per-day schedule (Mon–Sun). Each day has a From/Until time window and an Enabled toggle. When the current time falls inside a window, Focus Mode activates automatically (social-media blocking only). Overnight windows (e.g. 22:00→06:00) are supported. Defaults: Mon–Fri 08:00–22:00 enabled, Sat–Sun disabled.

---

## 1. K10 Update / Auto-Update ❌

**Setup → K10 Update:** "Check Now" is a stub.

**Backend needed:**
- `GetVersion() VersionInfo { Version, DBDate, DBDomains string }` on App
- `CheckForUpdate() (hasUpdate bool, message string, err error)` — HTTP call to a version endpoint
- Optional: auto-download updated blocklist

**Frontend:** Wire "Check Now" button to `go().CheckForUpdate()`.

---

## 2. Blocking Effects — Block Page Customisation 🔧

Styled HTML block page is already served by the proxy. The custom message field is wired to `SaveAdvancedSettings`.

**Remaining (frontend):**
- "Blank Page" option (send empty 200)
- Theme/colour picker if desired

---

## 3. Enable/Disable UX Polish 🔧

- Rename "Logout" button to "Lock Admin" or "Disable…"
- Distinguish "lock admin panel" from "disable protection" in the confirmation dialog
- Password re-prompt in header for re-enabling

---

## 4. Password-Protected Settings Saves 🔧

`SaveContentSettings` and `SaveAdvancedSettings` are called with `''` as password.

**Fix (frontend):**
- Before any `Save*` call, check `go().HasPassword()`
- If set, prompt user and pass password as first arg
- Show error on wrong password

---

## 5. Filter Level Persistence ❌

Dashboard "Filter Level" card does not reflect High/Moderate/Custom selections after restart.

**Backend needed:**
- `FilterLevel string` already in Config — ensure it round-trips via `GetContentSettings()` / `GetStatus()`

**Frontend:** `loadDashboard()` — use `cs.filterLevel` directly.

---

## 6. Real Page-Content Keyword Scanning ❌

**Current state:** `database.BlocksKeyword(rawURL)` is named and documented as "page-content keyword phrases" but it only checks the raw request URL string — not the actual page body. It fires at `proxy.go:234` for HTTP requests before the response is read.

**Why it matters:** URL-based keyword matching misses cases where a blocked keyword appears only in rendered page content, not in the URL (e.g. a search results page whose URL is `google.com/search?q=safe+query` but whose body contains blocked terms).

**What real page-content scanning requires:**
- MITM HTTPS interception: generate a per-session cert for the target host, TLS-terminate the connection, read the decrypted response body. This is a significant privacy/security surface.
- Content-type gating: only scan `text/html` and `text/plain` responses; skip binary, images, video.
- Streaming scan: buffer enough of the body to keyword-scan, then stream the rest — can't buffer entire responses.
- Performance budget: string.Contains over 20k keywords per response body is expensive; needs an Aho-Corasick or similar multi-pattern matcher.
- Certificate trust: user must install the K10 CA cert into macOS Keychain (already partially done for SafeSearch MITM in `mitm.go`).

**Suggested approach:**
1. Reuse the existing `initMITM()` / SafeSearch MITM infrastructure in `mitm.go`.
2. Extend `blockTunnel` → `mitmIntercept` to optionally scan the decrypted body for keyword matches.
3. Gate behind a new config flag `ScanPageContent bool` (off by default — it's a privacy escalation).
4. Replace linear scan with Aho-Corasick (e.g. `github.com/cloudflare/ahocorasick`) for O(n+m) body scan.

**Rename note:** `BlocksKeyword()` in `database.go` should be renamed to `BlocksURLKeyword()` to accurately reflect that it scans URLs, not page content. Update the call site at `proxy.go:234` and the comment at `database.go:130`.

---

---

# Security Architecture Upgrades — Priority Roadmap

These are the features that determine whether K10 is a real filter or a speed bump. Ordered by impact and feasibility.

---

## P1 (CRITICAL) — VPN / Tor Bypass Prevention ❌

**Why this is #1:** A VPN app running at the system level routes all traffic through an encrypted tunnel that completely bypasses the proxy. The user just installs ProtonVPN or Mullvad and your entire filter is invisible. This is the single easiest bypass and the most common one.

**How a VPN bypasses the proxy:**
The macOS system proxy setting (what K10 sets) is a convention — apps can ignore it. A VPN installs a `utun` (userspace tunnel) network interface and rewrites the system routing table so all packets go through the VPN before they ever reach the proxy. K10 never sees the traffic.

**Layers of defense (implement all, highest-ROI first):**

### Layer 1 — Block VPN protocol ports with PF (Packet Filter) — Easiest
macOS has a built-in BSD packet filter (`/etc/pf.conf`). Load rules at startup via LaunchDaemon (already have one).

Ports to block:
```
# OpenVPN
block out proto {tcp udp} to any port 1194
# WireGuard  
block out proto udp to any port 51820
# L2TP/IPSec
block out proto udp to any port {500, 4500}
# PPTP
block out proto tcp to any port 1723
# Shadowsocks / V2Ray common ports
block out proto {tcp udp} to any port {8388, 10086}
```
**Limitation:** VPN providers increasingly use port 443 (HTTPS) to blend in. Blocks naive VPN clients but not hardened ones.

**Implementation:** Add pf rule loading to `install.sh`. Write rules to `/etc/pf.anchors/k10webprotection`. Load via `pfctl -a k10webprotection -f` in the LaunchDaemon.

### Layer 2 — Block VPN provider domains — Already partially done
The `proxy-avoidance` category already blocks DoH/VPN domains from UT1. Extend it:
- Add explicit entries for top VPN provider domains: `protonvpn.com`, `mullvad.net`, `nordvpn.com`, `expressvpn.com`, `ipvanish.com`, `privatevpn.com`, `surfshark.com`, `cyberghost.com`, `pia.com` (Private Internet Access), `tunnelbear.com`
- Block their download/auth endpoints = VPN can't authenticate even if installed

**Implementation:** Add a manual `lists/categories/proxy-avoidance/domains.txt` supplement with these domains. `sync.py` uses `merge_into()` for URLs so manual entries survive a sync.

### Layer 3 — Detect and alert on VPN tunnel interfaces — Medium effort
When a VPN connects, macOS creates a `utun0`/`utun1`/`utun2` network interface. A background goroutine can poll `net.Interfaces()` and alert (or force-disconnect) when a new `utun` interface appears that wasn't there at startup.

```go
// In a goroutine in app.go, poll every 30s:
func (a *App) watchForVPN() {
    baseline := utunInterfaces()
    for range time.Tick(30 * time.Second) {
        current := utunInterfaces()
        if new := current - baseline; len(new) > 0 {
            // alert user / log / optionally kill the interface
        }
    }
}
func utunInterfaces() map[string]bool {
    ifaces, _ := net.Interfaces()
    m := make(map[string]bool)
    for _, i := range ifaces {
        if strings.HasPrefix(i.Name, "utun") {
            m[i.Name] = true
        }
    }
    return m
}
```

### Layer 4 — macOS Network Extension (Full OS-level interception) — Hard, highest coverage
A `NEPacketTunnelProvider` or `NEFilterDataProvider` intercepts ALL traffic at the kernel level — VPNs, non-proxy-aware apps, everything. This is how Circle, Bark, and enterprise MDM filters work.

**Blocker:** Requires an Apple entitlement (`com.apple.developer.networking.networkextension`) that must be requested from Apple for App Store distribution. For sideloaded/local builds, the entitlement can be self-signed with SIP disabled — not practical for end users.

**Verdict:** Layer 1 + 2 + 3 are implementable now. Layer 4 requires Apple partnership.

### Tor specifically
Tor uses port 9001 (relay) and 9030 (directory). Block both in PF. Also block `torproject.org` and known Tor bridge providers in the proxy-avoidance list. Meek bridges (Tor traffic disguised as HTTPS to `ajax.aspnetcdn.com` etc.) are harder — these are indistinguishable from normal HTTPS.

---

## P2 — Non-Browser App Traffic Interception ❌

**Why this is #2:** Mobile games, Discord, Twitter app, Reddit app, Telegram — these are native macOS/iOS apps that don't respect the system HTTP proxy. They open raw TCP/TLS connections directly. K10's proxy never sees them.

**What "system proxy" actually covers:**
- Safari, Chrome, Firefox (when system proxy is enabled in browser settings) ✓
- Terminal `curl`/`wget` if `http_proxy` env var is set ✓
- Apps that explicitly call `CFNetworkCopySystemProxySettings()` ✓
- Most Electron apps ✓
- **Native Swift/Obj-C apps using `URLSession`**: depends — some respect, some don't
- **Games, Discord, Telegram, custom TCP clients**: generally do NOT ✓✗

**Approaches:**

### Option A — DNS-level filtering (covers all apps, best ROI for P2)
If you control DNS, you control all apps — every app must resolve a hostname before connecting. See **P4** for the DNS architecture. DNS filtering alone doesn't see HTTPS content but it blocks the domain entirely, which is often sufficient.

**This is the single highest-leverage architectural change.** DNS + proxy together cover ~90% of cases.

### Option B — Transparent proxy via PF (pf redirect)
PF can redirect all outbound TCP port 80/443 traffic to a local port regardless of where the app sends it. This makes the proxy "transparent" — apps don't need to know about it.

```
# Redirect all port 80/443 to k10 proxy on port 8080
rdr pass on en0 proto tcp to port {80, 443} -> 127.0.0.1 port 8080
```

**Limitation:** HTTPS transparent proxying requires MITM (the proxy terminates TLS on behalf of the app, which requires the CA cert to be trusted). Apps with certificate pinning (Signal, banking apps) will break.

### Option C — Network Extension (same as P1 Layer 4)
Full solution but same Apple entitlement barrier.

**Recommended path:** Implement DNS filtering (P4) as the primary catch-all for all apps, and use PF transparent redirect for HTTP traffic from non-proxy-aware apps.

---

## P3 — Real Page-Content Keyword Scanning ❌

*(Detailed spec already above in Feature #6 — promoted to Priority 3 here.)*

**Key addition since original spec:** Once full MITM is in place for P2 (transparent proxy), page-content scanning becomes cheap to add — the TLS is already being terminated. The Aho-Corasick scanner can be bolted onto the existing MITM response pipeline.

**Short summary of what's needed:**
1. Full HTTPS MITM (transparent proxy via PF + per-host cert generation in `mitm.go`)
2. Response body buffering for `text/html` content types only
3. Aho-Corasick multi-pattern matcher replacing the current linear keyword scan
4. Config flag `ScanPageContent bool` — off by default (privacy escalation)

---

## P4 — Real-Time DNS Categorization (New Domain Coverage) ❌

**Why this matters:** Blocklists lag. A new porn site registered today won't appear in any list for days or weeks. DNS-based filtering can query a cloud categorization API on every new domain lookup — if the cloud says "pornography", block it even though it's not in the local list.

**Two sub-features, implement in order:**

### Sub-feature A — Local DNS Resolver (intercept all app DNS queries)
Run a DNS resolver on `127.0.0.1:53` (or `127.0.0.1:5353` redirected via PF) that:
1. Receives every DNS query from every app on the machine
2. Checks the hostname against the local blocklist database first (fast, no network)
3. If not in the local list, forwards to upstream DNS (8.8.8.8 or user-configured)
4. Returns `NXDOMAIN` or `0.0.0.0` for blocked domains

**Effect:** Blocks at DNS = affects all apps, all protocols, including native apps that bypass the HTTP proxy.

**Implementation:**
- Use `github.com/miekg/dns` (pure Go, well-maintained DNS library)
- Bind on startup: `dns.ListenAndServe("127.0.0.1:5353", "udp", handler)`
- PF rule to redirect port 53 → 5353: `rdr pass proto udp to port 53 -> 127.0.0.1 port 5353`
- Set `127.0.0.1` as the system DNS server on the active network interface at startup (reverse on stop)

**Complication:** Changing system DNS requires root. The LaunchDaemon (already running as root for the proxy) can do this.

### Sub-feature B — Cloud Categorization API for Unknown Domains
When a domain isn't in the local database, query a categorization API:

**Free/cheap options:**
- **Cloudflare Radar API** — domain categorization, free tier
- **Google Safe Browsing API** — malware/phishing only, free
- **IPQualityScore** — domain reputation, freemium
- **Quad9 DNS** — malware-blocking DNS (just use their resolver as upstream instead of 8.8.8.8, zero code)

**Simplest version:** Switch the upstream DNS from `8.8.8.8` to `9.9.9.9` (Quad9) or `1.1.1.3` (Cloudflare for Families). These block known malware/phishing at the DNS level for free, with no API integration needed.

**Full version (cloud lookup):**
```go
type CloudCategorizer interface {
    Categorize(domain string) ([]string, error) // returns category names
}
// Cache results in memory (LRU) + on disk to avoid re-querying the same domain
```
Cache TTL: 24h for clean domains, 1h for blocked (in case a site is cleaned up).

---

---

# Next-Level Architecture — Full Coverage Target

To reach ~90%+ accuracy (vs the current ~60-70% for known established sites), K10 needs at minimum 60% of the following. Listed in order of coverage impact:

| Feature | Coverage impact | Effort | Status |
|---------|----------------|--------|--------|
| DNS-level filtering (P4A) | Very high — all apps, all protocols | Medium | ❌ |
| VPN port blocking via PF (P1 L1) | High — stops most VPN clients | Low | ❌ |
| Full HTTPS MITM — transparent proxy | High — sees inside HTTPS for all apps | High | ❌ |
| ML image/video classification | Very high — catches visual content on any domain | Very high | ❌ |
| Cloud DNS categorization (P4B) | High — catches new/unlisted domains | Medium | ❌ |
| Page-content keyword scan (P3) | Medium — text-only, misses images | Medium | ❌ |
| VPN interface detection (P1 L3) | Medium — detects but doesn't prevent | Low | ❌ |
| Non-browser transparent proxy (P2B) | Medium — HTTP apps only | Medium | ❌ |
| Network Extension (P1 L4 / P2C) | Maximum — kernel-level, unbypassable | Very high + Apple entitlement | ❌ |

### The 60% target means implementing, at minimum:
1. **DNS-level filtering** (P4A) — single biggest architectural upgrade
2. **VPN port blocking** (P1 Layer 1+2) — low effort, high impact
3. **Cloud DNS upstream** (Quad9/Cloudflare for Families) — zero-code new-domain coverage
4. **VPN interface monitoring** (P1 Layer 3) — medium effort, detects and alerts

With just those four, K10 goes from "browser filter" to "machine-level filter" — covering native apps, games, and most VPN clients without requiring an Apple entitlement or full MITM.

### What ML image classification would look like
The gap DNS + proxy can never close: a domain not in any list serving images via a CDN. The only way to catch this is to classify the image itself.

- **Local option:** Apple's `CoreML` + a pre-trained NSFW model (e.g. open-source Yahoo Open NSFW converted to CoreML). Runs on-device, no privacy concern, ~50ms per image on Apple Silicon.
- **Cloud option:** Google Vision SafeSearch API, AWS Rekognition — accurate but adds latency and cost per image.
- **Integration point:** In the MITM pipeline, after TLS termination, intercept `Content-Type: image/*` responses and run the classifier. If score > threshold, serve a placeholder instead.
- **Caveat:** Only works for HTTP and HTTPS-intercepted connections. Encrypted video streams (HLS, DASH over HTTPS) cannot be classified without buffering the entire video.

This is the architecture that Covenant Eyes, Bark, and Circle approach — and it's what separates a serious parental filter from a domain blocklist.
