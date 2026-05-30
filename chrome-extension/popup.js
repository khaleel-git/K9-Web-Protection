// K9 Web Protection — Popup

const SOCIAL_KEYS = [
  'facebook','instagram','twitter','reddit',
  'tiktok','youtube','snapchat','pinterest','linkedin',
]

// Mirrors SOCIAL_SITES in background.js — used to detect which toggle to flip
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

let settings      = {}
let currentDomain = ''

// ── Helpers ───────────────────────────────────────────────────────────────────
function el(id) { return document.getElementById(id) }

// Returns the SOCIAL_SITES_MAP key for a given hostname, or null
function getSocialKey(host) {
  host = host.toLowerCase().replace(/^www\./, '')
  for (const [key, domains] of Object.entries(SOCIAL_SITES_MAP)) {
    if (domains.some(d => host === d || host.endsWith('.' + d))) return key
  }
  return null
}

function ask(type, data) {
  return new Promise((resolve) => {
    try {
      chrome.runtime.sendMessage({ type, settings: data }, (response) => {
        if (chrome.runtime.lastError) {
          console.warn('K9:', chrome.runtime.lastError.message)
          resolve(type === 'GET_SETTINGS' ? {} : null)
          return
        }
        resolve(response || (type === 'GET_SETTINGS' ? {} : null))
      })
    } catch (e) {
      console.warn('K9 ask error:', e)
      resolve(type === 'GET_SETTINGS' ? {} : null)
    }
  })
}

async function save(patch) {
  const merged = { ...settings, ...patch }
  settings = merged
  await ask('SAVE_SETTINGS', merged)
}

// ── Allowlist rendering ───────────────────────────────────────────────────────
function renderAllowlist(list) {
  const section = el('allowed-section')
  const hdr     = el('allowed-hdr')
  if (!list || list.length === 0) {
    hdr.style.display     = 'none'
    section.style.display = 'none'
    section.innerHTML     = ''
    return
  }
  hdr.style.display     = 'block'
  section.style.display = 'block'
  section.innerHTML = list.map(domain =>
    `<div class="allowed-item">
       <span class="allowed-icon">✅</span>
       <span class="allowed-domain">${domain}</span>
       <button class="allowed-remove" data-domain="${domain}" title="Remove">×</button>
     </div>`
  ).join('')

  section.querySelectorAll('.allowed-remove').forEach(btn => {
    btn.addEventListener('click', () => removeAllowed(btn.dataset.domain))
  })
}

async function removeAllowed(domain) {
  const userAllowlist = (settings.userAllowlist || []).filter(d => d !== domain)
  await save({ userAllowlist })
  renderAllowlist(userAllowlist)

  // Update current-site buttons if the removed domain is the active tab
  if (domain === currentDomain) {
    el('btn-allow').classList.remove('active')
  }
}

// ── Load ──────────────────────────────────────────────────────────────────────
async function load() {
  settings = await ask('GET_SETTINGS') || {}
  const { stats = {}, blockSocial = {}, focusMode } = settings

  el('toggle-enabled').checked = settings.enabled !== false
  updateStatusBar(settings.enabled !== false, focusMode)

  el('stat-today').textContent = (stats.blockedToday || 0).toLocaleString()
  el('stat-total').textContent = (stats.totalBlocked  || 0).toLocaleString()

  el('toggle-adult').checked = settings.blockAdultContent !== false

  for (const key of SOCIAL_KEYS) {
    const input = el(`toggle-${key === 'youtube' ? 'youtube-social' : key}`)
    if (input) input.checked = blockSocial[key] === true
  }

  updateFocusBtn(focusMode)
  renderAllowlist(settings.userAllowlist || [])

  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
  if (tab?.url) {
    try {
      currentDomain = new URL(tab.url).hostname
      el('current-domain').textContent = currentDomain
      el('btn-block').classList.toggle('active', (settings.userBlocklist || []).includes(currentDomain))
      el('btn-allow').classList.toggle('active', (settings.userAllowlist || []).includes(currentDomain))
    } catch (_) {
      el('current-domain').textContent = 'Not a webpage'
    }
  }
}

// ── Status bar ────────────────────────────────────────────────────────────────
function updateStatusBar(on, focus) {
  const bar = el('status-bar')
  const lbl = el('status-label')
  bar.className = 'status-bar'
  if (!on)   { bar.classList.add('off');   lbl.textContent = '⚠ Protection Disabled'; return }
  if (focus) { bar.classList.add('focus'); lbl.textContent = '⏱ Focus Mode Active';   return }
  lbl.textContent = '✓ Protection Active'
}

// ── Focus Mode ────────────────────────────────────────────────────────────────
function updateFocusBtn(active) {
  const btn  = el('focus-btn')
  const icon = el('focus-icon')
  const lbl  = el('focus-label')
  const desc = el('focus-desc')
  btn.classList.toggle('active', !!active)
  icon.textContent = active ? '🔴' : '⏱'
  lbl.textContent  = active
    ? 'Focus Mode Active — Click to disable'
    : 'Focus Mode — Block All Distractions'
  desc.textContent = active
    ? 'All social media and distractions are blocked.'
    : 'Instantly blocks social media and adult content.'
}

async function toggleFocusMode() {
  const newState = !settings.focusMode
  await save({ focusMode: newState })
  settings.focusMode = newState
  updateFocusBtn(newState)
  updateStatusBar(settings.enabled !== false, newState)
}

// ── Block / Allow current site ────────────────────────────────────────────────
function blockCurrent() {
  if (!currentDomain || el('btn-block').classList.contains('active')) return
  const userBlocklist = [...(settings.userBlocklist || []), currentDomain]
  const userAllowlist = (settings.userAllowlist || []).filter(d => d !== currentDomain)
  el('btn-block').classList.add('active')
  el('btn-allow').classList.remove('active')
  save({ userBlocklist, userAllowlist })
  renderAllowlist(userAllowlist)
}

function allowCurrent() {
  if (!currentDomain || el('btn-allow').classList.contains('active')) return

  const userAllowlist = [...(settings.userAllowlist || []), currentDomain]
  const userBlocklist = (settings.userBlocklist || []).filter(d => d !== currentDomain)

  el('btn-allow').classList.add('active')
  el('btn-block').classList.remove('active')

  // If this is a social media site, flip its individual toggle off
  const patch = { userAllowlist, userBlocklist }
  const socialKey = getSocialKey(currentDomain)
  if (socialKey && settings.blockSocial?.[socialKey]) {
    const blockSocial = { ...(settings.blockSocial || {}), [socialKey]: false }
    patch.blockSocial = blockSocial
    settings.blockSocial = blockSocial
    const toggleId = `toggle-${socialKey === 'youtube' ? 'youtube-social' : socialKey}`
    const toggle = el(toggleId)
    if (toggle) toggle.checked = false
  }

  save(patch)
  renderAllowlist(userAllowlist)
}

// ── Event listeners ───────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  el('btn-block').addEventListener('click', blockCurrent)
  el('btn-allow').addEventListener('click', allowCurrent)
  el('focus-btn').addEventListener('click', toggleFocusMode)

  el('toggle-enabled').addEventListener('change', (e) => {
    const on = e.target.checked
    updateStatusBar(on, settings.focusMode)
    save({ enabled: on })
  })

  el('toggle-adult').addEventListener('change', e =>
    save({ blockAdultContent: e.target.checked })
  )

  document.querySelectorAll('[data-social]').forEach(input => {
    input.addEventListener('change', (e) => {
      const key       = e.target.dataset.social
      const blockSocial = { ...(settings.blockSocial || {}), [key]: e.target.checked }
      settings.blockSocial = blockSocial

      // Turning a social site ON should remove it from the allowlist
      if (e.target.checked) {
        const domains = SOCIAL_SITES_MAP[key] || []
        const userAllowlist = (settings.userAllowlist || [])
          .filter(d => !domains.some(sd => d === sd || d.endsWith('.' + sd)))
        if (userAllowlist.length !== (settings.userAllowlist || []).length) {
          settings.userAllowlist = userAllowlist
          renderAllowlist(userAllowlist)
          save({ blockSocial, userAllowlist })
          return
        }
      }

      save({ blockSocial })
    })
  })

  load().catch(e => console.error('K9 popup init error:', e))
})
