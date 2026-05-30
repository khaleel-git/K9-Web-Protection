// K9 Web Protection — Blocked page

// Social domain → key map (mirrors SOCIAL_SITES in background.js)
const SOCIAL_SITES_MAP = {
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

// Flat list for quick isSocial check
const SOCIAL_DOMAINS = Object.values(SOCIAL_SITES_MAP).flat()

const QUOTES = [
  'Your future self will thank you for this moment of discipline.',
  "Every minute not scrolling is a minute invested in yourself.",
  "Champions are made in the moments they want to quit — but don't.",
  'The less you respond to distractions, the more peace you will have.',
  'Small disciplines repeated daily lead to great achievements.',
  "Your goals don't care about your newsfeed.",
  'Discipline is choosing between what you want now and what you want most.',
  'Focus is the superpower of the 21st century.',
  'The successful warrior is the average person with laser-like focus.',
  'What you do today can improve all your tomorrows.',
]

function el(id) { return document.getElementById(id) }

function isSocial(host) {
  return SOCIAL_DOMAINS.some(d => host === d || host.endsWith('.' + d))
}

// Returns the social key for a host, or null
function getSocialKey(host) {
  host = host.toLowerCase().replace(/^www\./, '')
  for (const [key, domains] of Object.entries(SOCIAL_SITES_MAP)) {
    if (domains.some(d => host === d || host.endsWith('.' + d))) return key
  }
  return null
}

// ── Parse params ──────────────────────────────────────────────────────────────
const params     = new URLSearchParams(location.search)
const blockedURL = params.get('url') || ''

let blockedHost = ''
try { blockedHost = new URL(blockedURL).hostname.toLowerCase() } catch (_) {}

const reason = params.get('reason') ||
  (isSocial(blockedHost) ? 'social' : 'adult')

// ── Configure page content ────────────────────────────────────────────────────
if (blockedHost) el('domain').textContent = blockedHost

if (reason === 'social') {
  document.title = 'Stay Focused — K9 Web Protection'
  document.body.classList.add('social')
  el('icon').textContent    = '📵'
  el('title').textContent   = 'Stay Focused'
  el('message').textContent = 'Social media is blocked. Use this time for something that matters.'
  const q = el('quote')
  q.textContent   = '"' + QUOTES[Math.floor(Math.random() * QUOTES.length)] + '"'
  q.style.display = 'block'
} else if (reason === 'keyword') {
  el('icon').textContent    = '🔑'
  el('title').textContent   = 'Keyword Blocked'
  el('message').textContent = 'This URL matched a blocked keyword.'
} else {
  el('message').textContent = 'This website is blocked by K9 Web Protection.'
}

// ── Allow this site ───────────────────────────────────────────────────────────
async function allowSite() {
  if (!blockedHost) return
  const btn = el('btn-allow')
  btn.textContent = 'Adding…'
  btn.disabled    = true

  try {
    // 1. Update allowlist in storage
    const data = await chrome.storage.local.get('userAllowlist')
    const list  = data.userAllowlist || []
    if (!list.includes(blockedHost)) list.push(blockedHost)
    await chrome.storage.local.set({ userAllowlist: list })

    // 2. If this is a social media site, also turn off its individual toggle
    //    so the popup shows it as disabled when next opened.
    const socialKey = getSocialKey(blockedHost)
    if (socialKey) {
      const { blockSocial = {} } = await chrome.storage.local.get('blockSocial')
      if (blockSocial[socialKey]) {
        blockSocial[socialKey] = false
        await chrome.storage.local.set({ blockSocial })
      }
    }

    // 3. Add allow rule directly for immediate effect
    const existing = await chrome.declarativeNetRequest.getDynamicRules()
    const maxId    = existing.length ? Math.max(...existing.map(r => r.id)) : 20000
    await chrome.declarativeNetRequest.updateDynamicRules({
      addRules: [{
        id:       maxId + 1,
        priority: 10,
        action:   { type: 'allow' },
        condition: { urlFilter: `||${blockedHost}`, resourceTypes: ['main_frame'] },
      }],
    })

    // 4. Tell background to do a full sync so changes persist
    chrome.runtime.sendMessage({ type: 'SAVE_SETTINGS', settings: { userAllowlist: list } })

    // 5. Navigate to the originally blocked URL
    window.location.href = blockedURL || '/'
  } catch (e) {
    console.error('K9 allowSite error:', e)
    btn.textContent = 'Error — try again'
    btn.disabled    = false
  }
}

// ── Bind buttons ──────────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  el('btn-back').addEventListener('click',  () => history.back())
  el('btn-allow').addEventListener('click', allowSite)
})
