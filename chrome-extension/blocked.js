// K9 Web Protection — Blocked page
// External script only — no inline JS to satisfy MV3 CSP.

const SOCIAL_DOMAINS = [
  'facebook.com','fb.com','messenger.com','facebook.net',
  'instagram.com',
  'twitter.com','x.com','t.co',
  'reddit.com','redd.it',
  'tiktok.com','tiktokv.com',
  'youtube.com','youtu.be','youtube-nocookie.com',
  'snapchat.com',
  'pinterest.com','pin.it',
  'linkedin.com',
]

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

// ── Parse params ──────────────────────────────────────────────────────────────
const params     = new URLSearchParams(location.search)
const blockedURL = params.get('url') || ''

let blockedHost = ''
try { blockedHost = new URL(blockedURL).hostname.toLowerCase() } catch (_) {}

function isSocial(host) {
  return SOCIAL_DOMAINS.some(d => host === d || host.endsWith('.' + d))
}

const reason = params.get('reason') ||
  (isSocial(blockedHost) ? 'social' : 'adult')

// ── Configure page content ────────────────────────────────────────────────────
function el(id) { return document.getElementById(id) }

if (blockedHost) {
  el('domain').textContent = blockedHost
}

if (reason === 'social') {
  document.title = 'Stay Focused — K9 Web Protection'
  document.body.classList.add('social')
  el('icon').textContent    = '📵'
  el('title').textContent   = 'Stay Focused'
  el('message').textContent = 'Social media is blocked. Use this time for something that matters.'
  const q = el('quote')
  q.textContent   = '"' + QUOTES[Math.floor(Math.random() * QUOTES.length)] + '"'
  q.style.display = 'block'
} else if (reason === 'imgsearch') {
  el('icon').textContent    = '🔍'
  el('title').textContent   = 'Image Search Blocked'
  el('message').textContent = 'Image and video search is blocked by K9 Web Protection.'
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

    // 2. Add allow rule directly for immediate effect
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

    // 3. Tell background to do a full sync for future sessions
    chrome.runtime.sendMessage({ type: 'SAVE_SETTINGS', settings: { userAllowlist: list } })

    // 4. Navigate to the originally blocked URL
    window.location.href = blockedURL || '/'
  } catch (e) {
    console.error('K9 allowSite error:', e)
    btn.textContent = 'Error — try again'
    btn.disabled    = false
  }
}

// ── Bind buttons (no inline onclick — MV3 CSP) ────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  el('btn-back').addEventListener('click',  () => history.back())
  el('btn-allow').addEventListener('click', allowSite)
})
