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
