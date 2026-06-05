# K9 Web Protection — Feature Plan

Frontend is complete and wired to existing backend APIs where possible. Items below are features visible in the UI that need backend implementation, or existing wiring that needs fixing.

**Status legend:** ✅ Done · 🔧 Frontend only (backend needed) · ❌ Not started

---

## Setup Sidebar — All 9 Items ✅

All sidebar items are present and navigate to their pages:
1. Web Categories to Block ✅
2. Time Restrictions 🔧 (frontend shell only)
3. Web Site Exceptions ✅
4. Blocking Effects 🔧 (frontend shell only)
5. URL Keywords ✅
6. Safe Search ✅
7. Advanced ✅ (proxy port + autostart wired to Go)
8. Password/Email ✅
9. K10 Update 🔧 (frontend shell only)

---

## 1. Recent Blocked Activity Log ❌

**Dashboard:** "Recent Blocked Activity" panel shows top blocked domains without timestamps or category badges. Needs a proper activity log.

**Backend needed:**
- Add ring-buffer in config: `type ActivityEntry { Time time.Time; Domain string; Category string }`
- Record in proxy `onBlock` callback
- Add `GetRecentActivity() []ActivityEntry` on App

**Frontend:** `renderRecentActivity()` in `main.js` — replace domain heuristics with `go().GetRecentActivity()`.

---

## 2. Per-Category Block Counts (Top Blocked Categories Bar Chart) ❌

**Dashboard:** Bar chart shows top blocked *domains*, not semantic categories (Pornography, Malware, Gambling, etc.).

**Backend needed:**
- Map each blocked domain to its category at block time (proxy or DB layer)
- Store per-category hit counts in `config.Stats`
- Add `TopCategories []CategoryEntry` to `Status`

**Frontend:** `renderTopCategoriesChart()` — use `s.topCategories`.

---

## 3. Per-Category Activity Tagging ❌

Blocked activity rows need real category labels, not regex heuristics on domain names.

**Backend needed:** Same as item 2 — store `Category string` in `ActivityEntry`.

**Frontend:** `renderRecentActivity()` — use `entry.category` directly.

---

## 4. Filter Level Persistence ❌

**Dashboard:** Filter Level card always shows "Default" or "Minimal" based on `blockAdultContent` bool. Doesn't reflect High/Moderate/Custom selections.

**Backend needed:**
- Add `FilterLevel string` to `Config`
- Set it in `SaveContentSettings`
- Return it in `GetContentSettings()` or `GetStatus()`

**Frontend:** `loadDashboard()` — use `cs.filterLevel` instead of the boolean check.

---

## 5. Time Restrictions ❌

**Setup → Time Restrictions:** Page shows a per-day time picker UI but saving does nothing.

**Backend needed:**
- Add `TimeRestrictions map[string][2]string` to Config (e.g. `"Monday": ["08:00","22:00"]`)
- Add `TimeEnabled bool` to Config
- Proxy checks `time.Now()` against schedule before forwarding requests
- Add `GetTimeRestrictions()` / `SaveTimeRestrictions(schedule, enabled)` on App

**Frontend:** Wire `Save` button to `go().SaveTimeRestrictions(...)`.

---

## 6. Blocking Effects — Custom Block Page ❌

**Setup → Blocking Effects:** Shows dropdowns but only the custom message field is wired to `SaveAdvancedSettings`.

**Backend needed:**
- Proxy currently returns a plain-text response on blocked requests
- Implement a styled HTML block page (embed as template)
- Respect `BlockedMessage` from config as custom message text
- Support "Blank Page" option (empty 200 response)

**Frontend:** Already wired — `eff-custom-msg` saves via `go().SaveAdvancedSettings`.

---

## 7. K10 Update / Auto-Update ❌

**Setup → K10 Update:** Page shows database sizes but "Check Now" is a stub.

**Backend needed:**
- `GetVersion() VersionInfo { Version, DBDate, DBDomains string }` on App
- `CheckForUpdate() (hasUpdate bool, message string, err error)` — HTTP call to a version endpoint
- Optional: auto-download and apply updated blocklist database

**Frontend:** Wire `Check Now` button to `go().CheckForUpdate()` and update UI with result.

---

## 8. Enable/Disable UX Polish 🔧

**Current state:** When protection is inactive, a warning bar appears in the Home dashboard with "Enable Protection". The header Logout button opens the disable modal.

**Improvements needed (frontend):**
- Rename "Logout" to something clearer ("Lock Admin" or "Disable…")
- Add a confirmation screen distinguishing "lock the admin panel" from "disable protection"
- Consider a password-prompt in the header for re-enabling (prevents casual bypassing)

---

## 9. Password-Protected Settings Saves 🔧

**Current state:** `SaveContentSettings` and `SaveAdvancedSettings` are called with `''` as the password. This works when no password is set but silently fails when a password IS configured.

**Fix needed (frontend):**
- Before calling any `Save*` method that requires password, call `go().HasPassword()`
- If true, prompt user for password and pass it as first arg
- Show error notification if password is wrong

---

## 10. Safe Search Page — Load on Navigate ✅

Fixed: `loadSafeSearch()` is called when navigating to the Safe Search page, populating the checkbox from `GetContentSettings()`.
