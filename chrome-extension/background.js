// K9 Web Protection — Background Service Worker

const BLOCKED_URL      = chrome.runtime.getURL('blocked.html')
const DYNAMIC_ID_START = 10000

// Social media domain map — keyed by toggle name
const SOCIAL_SITES = {
  facebook:  ['facebook.com', 'fb.com', 'messenger.com', 'facebook.net'],
  instagram: ['instagram.com'],
  twitter:   ['twitter.com', 'x.com', 't.co'],
  reddit:    ['reddit.com', 'redd.it'],
  tiktok:    ['tiktok.com', 'tiktokv.com'],
  youtube:   ['youtube.com', 'youtu.be', 'youtube-nocookie.com'],
  snapchat:  ['snapchat.com'],
  pinterest: ['pinterest.com', 'pin.it'],
  linkedin:  ['linkedin.com'],
}

// ── Defaults ──────────────────────────────────────────────────────────────────
const DEFAULTS = {
  enabled:           true,
  blockAdultContent: true,
  blockImageSearch:  false,
  blockYouTube:      false,
  focusMode:         false,
  blockSocial: {
    facebook: false, instagram: false, twitter: false, reddit: false,
    tiktok: false, youtube: false, snapchat: false, pinterest: false, linkedin: false,
  },
  userBlocklist: [],
  userAllowlist: [],
  userKeywords:  [],
  stats: {
    blockedToday:  0,
    totalBlocked:  0,
    lastResetDate: new Date().toDateString(),
    topBlocked:    [],
  },
}

// ── Install ───────────────────────────────────────────────────────────────────
chrome.runtime.onInstalled.addListener(async () => {
  const existing = await chrome.storage.local.get(null)
  const merged   = { ...DEFAULTS, ...existing }
  // deep-merge blockSocial so new keys appear
  merged.blockSocial = { ...DEFAULTS.blockSocial, ...(existing.blockSocial || {}) }
  await chrome.storage.local.set(merged)
  await syncDynamicRules(merged)
  updateBadge(merged)
})

// ── Stats ─────────────────────────────────────────────────────────────────────
async function maybeResetDaily() {
  const { stats } = await chrome.storage.local.get('stats')
  if (!stats) return
  if (stats.lastResetDate !== new Date().toDateString()) {
    stats.blockedToday  = 0
    stats.lastResetDate = new Date().toDateString()
    await chrome.storage.local.set({ stats })
  }
}

function incrementBlocked(domain) {
  chrome.storage.local.get('stats', ({ stats = DEFAULTS.stats }) => {
    stats.blockedToday++
    stats.totalBlocked++
    const entry = stats.topBlocked.find(e => e.domain === domain)
    if (entry) { entry.count++ }
    else { stats.topBlocked.push({ domain, count: 1 }) }
    stats.topBlocked.sort((a, b) => b.count - a.count)
    stats.topBlocked = stats.topBlocked.slice(0, 10)
    chrome.storage.local.set({ stats })
  })
}

// ── Badge ─────────────────────────────────────────────────────────────────────
function updateBadge(s) {
  if (!s.enabled) {
    chrome.action.setBadgeText({ text: 'OFF' })
    chrome.action.setBadgeBackgroundColor({ color: '#888' })
  } else if (s.focusMode) {
    chrome.action.setBadgeText({ text: '⏱' })
    chrome.action.setBadgeBackgroundColor({ color: '#ff6b35' })
  } else {
    chrome.action.setBadgeText({ text: '' })
  }
}

// ── Dynamic DNR rules ─────────────────────────────────────────────────────────
async function syncDynamicRules(s) {
  const existing  = await chrome.declarativeNetRequest.getDynamicRules()
  const removeIds = existing.map(r => r.id)
  const addRules  = []
  let id = DYNAMIC_ID_START

  // NOTE: extensionPath must NOT contain query strings — Chrome rejects them
  // and silently drops the entire updateDynamicRules call.
  // The blocked.html page detects the reason from the blocked URL itself.
  const push = (filter) => {
    addRules.push({
      id: id++,
      priority: 3,
      action: { type: 'redirect', redirect: { extensionPath: '/blocked.html' } },
      condition: { urlFilter: filter, resourceTypes: ['main_frame'] },
    })
  }

  // User block list
  for (const domain of (s.userBlocklist || [])) {
    const d = domain.replace(/^https?:\/\//, '').replace(/^www\./, '').split('/')[0]
    if (d) push(`||${d}`)
  }

  // Social media — individual toggles
  const blockSocial = s.blockSocial || {}
  for (const [key, domains] of Object.entries(SOCIAL_SITES)) {
    if (!blockSocial[key] && !s.focusMode) continue
    if (key === 'youtube' && !blockSocial.youtube && !s.focusMode) continue
    for (const d of domains) push(`||${d}`)
  }

  // Focus mode also blocks image search
  if (s.focusMode || s.blockImageSearch) {
    push('google.com/imghp')
    push('google.com/search?*tbm=isch')
    push('bing.com/images')
    push('images.google.')
    push('duckduckgo.com/*iax=images')
  }

  // blockYouTube toggle (content section, separate from social)
  if (s.blockYouTube && !blockSocial.youtube && !s.focusMode) {
    for (const d of SOCIAL_SITES.youtube) push(`||${d}`)
  }

  // Allow list — highest priority
  for (const domain of (s.userAllowlist || [])) {
    const d = domain.replace(/^https?:\/\//, '').replace(/^www\./, '').split('/')[0]
    if (!d) continue
    addRules.push({
      id: id++, priority: 10,
      action: { type: 'allow' },
      condition: { urlFilter: `||${d}`, resourceTypes: ['main_frame'] },
    })
  }

  await chrome.declarativeNetRequest.updateDynamicRules({ removeRuleIds: removeIds, addRules })

  // Static ruleset (adult content)
  try {
    const enable  = s.enabled && s.blockAdultContent
    await chrome.declarativeNetRequest.updateEnabledRulesets({
      enableRulesetIds:  enable ? ['k9_blocklist'] : [],
      disableRulesetIds: enable ? [] : ['k9_blocklist'],
    })
  } catch (_) {}
}

// ── Core URL check (full nav + pushState) ─────────────────────────────────────
async function checkUrl(tabId, rawUrl) {
  if (!rawUrl || rawUrl.startsWith(BLOCKED_URL)) return

  const s = await chrome.storage.local.get([
    'enabled', 'blockImageSearch', 'blockYouTube', 'blockSocial',
    'focusMode', 'userKeywords', 'userAllowlist',
  ])
  if (!s.enabled) return

  const url = rawUrl.toLowerCase()
  let host
  try { host = new URL(rawUrl).hostname.toLowerCase() } catch { return }

  // Allow-list
  for (const d of (s.userAllowlist || [])) {
    if (host === d || host.endsWith('.' + d)) return
  }

  const redirect = (reason) => {
    chrome.tabs.update(tabId, {
      url: `${BLOCKED_URL}?url=${encodeURIComponent(rawUrl)}&reason=${reason}`,
    })
  }

  // Social media — individual + focus mode
  const blockSocial = s.blockSocial || {}
  for (const [key, domains] of Object.entries(SOCIAL_SITES)) {
    const blocked = blockSocial[key] || s.focusMode ||
      (key === 'youtube' && s.blockYouTube)
    if (!blocked) continue
    for (const d of domains) {
      if (host === d || host.endsWith('.' + d)) { redirect('social'); return }
    }
  }

  // Image search (pushState — Google Images tab click)
  if (s.blockImageSearch || s.focusMode) {
    if (
      (url.includes('google.') && (url.includes('/imghp') || url.includes('tbm=isch'))) ||
      url.includes('bing.com/images') ||
      url.includes('images.google.') ||
      (url.includes('duckduckgo.com') && url.includes('iax=images'))
    ) { redirect('imgsearch'); return }
  }

  // User keywords
  for (const kw of (s.userKeywords || [])) {
    if (kw && url.includes(kw.toLowerCase())) { redirect('keyword'); return }
  }
}

// ── Navigation listeners ───────────────────────────────────────────────────────
chrome.webNavigation.onBeforeNavigate.addListener(async (d) => {
  if (d.frameId !== 0) return
  // stats tracking for blocked redirects
  if (d.url.startsWith(BLOCKED_URL)) {
    try {
      const blocked = new URL(d.url).searchParams.get('url')
      if (blocked) incrementBlocked(new URL(blocked).hostname)
    } catch (_) {}
    return
  }
  await checkUrl(d.tabId, d.url)
})

// pushState — Google Images tab, YouTube page changes, Twitter SPA, etc.
chrome.webNavigation.onHistoryStateUpdated.addListener(async (d) => {
  if (d.frameId !== 0) return
  await checkUrl(d.tabId, d.url)
})

// ── Messages from popup ───────────────────────────────────────────────────────
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
  }
})
