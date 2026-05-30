// K9 Web Protection — Popup
// No inline event handlers — all binding done here to satisfy MV3 CSP.

const SOCIAL_KEYS = [
  'facebook','instagram','twitter','reddit',
  'tiktok','youtube','snapchat','pinterest','linkedin',
]

let settings      = {}
let currentDomain = ''

// ── Helpers ───────────────────────────────────────────────────────────────────
function el(id) { return document.getElementById(id) }

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

// ── Load ──────────────────────────────────────────────────────────────────────
async function load() {
  settings = await ask('GET_SETTINGS') || {}
  const { stats = {}, blockSocial = {}, focusMode } = settings

  el('toggle-enabled').checked = settings.enabled !== false
  updateStatusBar(settings.enabled !== false, focusMode)

  el('stat-today').textContent = (stats.blockedToday || 0).toLocaleString()
  el('stat-total').textContent = (stats.totalBlocked  || 0).toLocaleString()

  el('toggle-adult').checked  = settings.blockAdultContent !== false

  for (const key of SOCIAL_KEYS) {
    const id    = `toggle-${key === 'youtube' ? 'youtube-social' : key}`
    const input = el(id)
    if (input) input.checked = blockSocial[key] === true
  }

  updateFocusBtn(focusMode)

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
}

function allowCurrent() {
  if (!currentDomain || el('btn-allow').classList.contains('active')) return
  const userAllowlist = [...(settings.userAllowlist || []), currentDomain]
  const userBlocklist = (settings.userBlocklist || []).filter(d => d !== currentDomain)
  el('btn-allow').classList.add('active')
  el('btn-block').classList.remove('active')
  save({ userAllowlist, userBlocklist })
}

// ── Event listeners (no inline onclick — MV3 CSP requires this) ───────────────
document.addEventListener('DOMContentLoaded', () => {
  el('btn-block').addEventListener('click',  blockCurrent)
  el('btn-allow').addEventListener('click',  allowCurrent)
  el('focus-btn').addEventListener('click',  toggleFocusMode)

  el('toggle-enabled').addEventListener('change', (e) => {
    const on = e.target.checked
    updateStatusBar(on, settings.focusMode)
    save({ enabled: on })
  })

  el('toggle-adult').addEventListener('change', e => save({ blockAdultContent: e.target.checked }))

  document.querySelectorAll('[data-social]').forEach(input => {
    input.addEventListener('change', (e) => {
      const key        = e.target.dataset.social
      const blockSocial = { ...(settings.blockSocial || {}), [key]: e.target.checked }
      settings.blockSocial = blockSocial
      save({ blockSocial })
    })
  })

  load().catch(e => console.error('K9 popup init error:', e))
})
