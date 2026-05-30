// K9 Web Protection — Background Service Worker
// Handles: dynamic rules, keyword blocking, stats, settings

const BLOCKED_URL   = chrome.runtime.getURL('blocked.html')
const DYNAMIC_ID_START = 10000  // dynamic rules start from 10000

// ── Default settings ──────────────────────────────────────────────────────────
const DEFAULTS = {
  enabled:           true,
  blockAdultContent: true,
  blockImageSearch:  false,
  blockYouTube:      false,
  safeSearch:        true,
  userBlocklist:     [],
  userAllowlist:     [],
  userKeywords:      [],
  stats: {
    blockedToday: 0,
    totalBlocked:  0,
    lastResetDate: new Date().toDateString(),
    topBlocked:    []
  }
}

// ── Initialise on install ─────────────────────────────────────────────────────
chrome.runtime.onInstalled.addListener(async () => {
  const existing = await chrome.storage.local.get(null)
  const merged = { ...DEFAULTS, ...existing }
  await chrome.storage.local.set(merged)
  await syncDynamicRules(merged)
  updateBadge(merged)
})

// ── Reset stats daily ─────────────────────────────────────────────────────────
async function maybeResetDaily() {
  const { stats } = await chrome.storage.local.get('stats')
  if (!stats) return
  if (stats.lastResetDate !== new Date().toDateString()) {
    stats.blockedToday  = 0
    stats.lastResetDate = new Date().toDateString()
    await chrome.storage.local.set({ stats })
  }
}

// ── Dynamic declarativeNetRequest rules from user's lists ─────────────────────
async function syncDynamicRules(settings) {
  const existing = await chrome.declarativeNetRequest.getDynamicRules()
  const removeIds = existing.map(r => r.id)

  const addRules = []
  let id = DYNAMIC_ID_START

  // User block list
  for (const domain of (settings.userBlocklist || [])) {
    const clean = domain.replace(/^https?:\/\//, '').replace(/^www\./, '').split('/')[0]
    if (!clean) continue
    addRules.push({
      id: id++,
      priority: 3,
      action: { type: 'redirect', redirect: { extensionPath: '/blocked.html' } },
      condition: { urlFilter: `||${clean}`, resourceTypes: ['main_frame'] }
    })
  }

  // Block YouTube toggle
  if (settings.blockYouTube) {
    for (const d of ['youtube.com', 'youtu.be', 'youtube-nocookie.com']) {
      addRules.push({
        id: id++,
        priority: 3,
        action: { type: 'redirect', redirect: { extensionPath: '/blocked.html' } },
        condition: { urlFilter: `||${d}`, resourceTypes: ['main_frame'] }
      })
    }
  }

  // Block image search toggle
  if (settings.blockImageSearch) {
    for (const filter of ['google.com/imghp', 'google.com/search?*tbm=isch', 'bing.com/images', 'images.google.']) {
      addRules.push({
        id: id++,
        priority: 3,
        action: { type: 'redirect', redirect: { extensionPath: '/blocked.html' } },
        condition: { urlFilter: filter, resourceTypes: ['main_frame'] }
      })
    }
  }

  // User allow list — highest priority, overrides everything
  for (const domain of (settings.userAllowlist || [])) {
    const clean = domain.replace(/^https?:\/\//, '').replace(/^www\./, '').split('/')[0]
    if (!clean) continue
    addRules.push({
      id: id++,
      priority: 10,
      action: { type: 'allow' },
      condition: { urlFilter: `||${clean}`, resourceTypes: ['main_frame'] }
    })
  }

  await chrome.declarativeNetRequest.updateDynamicRules({ removeRuleIds: removeIds, addRules })

  // Enable/disable the static ruleset based on blockAdultContent toggle
  const enabledRulesets  = settings.enabled && settings.blockAdultContent ? ['k9_blocklist'] : []
  const disabledRulesets = settings.enabled && settings.blockAdultContent ? [] : ['k9_blocklist']
  try {
    await chrome.declarativeNetRequest.updateEnabledRulesets({
      enableRulesetIds:  enabledRulesets,
      disableRulesetIds: disabledRulesets
    })
  } catch (_) {}
}

// ── Badge ─────────────────────────────────────────────────────────────────────
function updateBadge(settings) {
  if (!settings.enabled) {
    chrome.action.setBadgeText({ text: 'OFF' })
    chrome.action.setBadgeBackgroundColor({ color: '#888' })
  } else {
    chrome.action.setBadgeText({ text: '' })
  }
}

function incrementBlocked(domain) {
  chrome.storage.local.get('stats', ({ stats = DEFAULTS.stats }) => {
    stats.blockedToday++
    stats.totalBlocked++
    const entry = stats.topBlocked.find(e => e.domain === domain)
    if (entry) {
      entry.count++
    } else {
      stats.topBlocked.push({ domain, count: 1 })
    }
    stats.topBlocked.sort((a, b) => b.count - a.count)
    stats.topBlocked = stats.topBlocked.slice(0, 10)
    chrome.storage.local.set({ stats })
  })
}

// ── Navigation listener — keyword blocking ────────────────────────────────────
chrome.webNavigation.onBeforeNavigate.addListener(async (details) => {
  if (details.frameId !== 0) return  // main frame only

  const { enabled, userKeywords } = await chrome.storage.local.get(['enabled', 'userKeywords'])
  if (!enabled || !userKeywords?.length) return

  const url = details.url.toLowerCase()
  for (const kw of userKeywords) {
    if (url.includes(kw.toLowerCase())) {
      // Redirect to blocked page
      chrome.tabs.update(details.tabId, {
        url: `${BLOCKED_URL}?url=${encodeURIComponent(details.url)}&reason=keyword`
      })
      return
    }
  }
})

// Track blocked redirects for stats
chrome.webNavigation.onBeforeNavigate.addListener(async (details) => {
  if (details.frameId !== 0) return
  if (!details.url.startsWith(BLOCKED_URL)) return

  const params  = new URL(details.url).searchParams
  const blocked = params.get('url') || 'unknown'
  try {
    const domain = new URL(blocked).hostname
    incrementBlocked(domain)
  } catch (_) {}
})

// ── Messages from popup / content script ─────────────────────────────────────
chrome.runtime.onMessage.addListener((msg, _sender, respond) => {
  maybeResetDaily()

  switch (msg.type) {
    case 'GET_SETTINGS':
      chrome.storage.local.get(null, respond)
      return true

    case 'SAVE_SETTINGS':
      chrome.storage.local.set(msg.settings, async () => {
        const all = await chrome.storage.local.get(null)
        await syncDynamicRules(all)
        updateBadge(all)
        respond({ ok: true })
      })
      return true

    case 'BLOCKED':
      incrementBlocked(msg.domain)
      respond({ ok: true })
      return true
  }
})
