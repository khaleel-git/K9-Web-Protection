// K9 Web Protection — Popup

const SOCIAL_KEYS = ['facebook','instagram','twitter','reddit','tiktok','youtube','snapchat','pinterest','linkedin']

let settings      = {}
let currentDomain = ''

// ── Load all settings ─────────────────────────────────────────────────────────
async function load() {
  settings = await ask('GET_SETTINGS')
  const { stats = {}, blockSocial = {}, focusMode } = settings

  // Master toggle
  el('toggle-enabled').checked = settings.enabled !== false
  updateStatusBar(settings.enabled !== false, focusMode)

  // Stats
  el('stat-today').textContent = (stats.blockedToday || 0).toLocaleString()
  el('stat-total').textContent = (stats.totalBlocked  || 0).toLocaleString()

  // Content toggles
  el('toggle-adult').checked  = settings.blockAdultContent !== false
  el('toggle-images').checked = settings.blockImageSearch  === true

  // Social media toggles
  for (const key of SOCIAL_KEYS) {
    const input = el(`toggle-${key === 'youtube' ? 'youtube-social' : key}`)
    if (input) input.checked = blockSocial[key] === true
  }

  // Focus mode button
  updateFocusBtn(focusMode)

  // Current tab
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
  if (!on)    { bar.classList.add('off');   lbl.textContent = '⚠ Protection Disabled'; return }
  if (focus)  { bar.classList.add('focus'); lbl.textContent = '⏱ Focus Mode Active';   return }
  lbl.textContent = '✓ Protection Active'
}

// ── Focus Mode ────────────────────────────────────────────────────────────────
function updateFocusBtn(active) {
  const btn  = el('focus-btn')
  const icon = el('focus-icon')
  const lbl  = el('focus-label')
  const desc = el('focus-desc')

  btn.classList.toggle('active', active)
  icon.textContent = active ? '🔴' : '⏱'
  lbl.textContent  = active
    ? 'Focus Mode Active — Click to disable'
    : 'Focus Mode — Block All Distractions'
  desc.textContent = active
    ? 'All social media and distractions are blocked.'
    : 'Instantly blocks social media, image search, and adult content.'

  // Dim individual social toggles while focus mode is on (they're all overridden)
  document.querySelectorAll('.toggle-row[data-group="social"]')
    .forEach(row => row.classList.toggle('focus-active', active))
}

async function toggleFocusMode() {
  const newState = !settings.focusMode
  await save({ focusMode: newState })
  settings.focusMode = newState
  updateFocusBtn(newState)
  updateStatusBar(settings.enabled !== false, newState)
}
window.toggleFocusMode = toggleFocusMode

// ── Master toggle ─────────────────────────────────────────────────────────────
el('toggle-enabled').addEventListener('change', async (e) => {
  const on = e.target.checked
  updateStatusBar(on, settings.focusMode)
  await save({ enabled: on })
})

// ── Content toggles ───────────────────────────────────────────────────────────
el('toggle-adult').addEventListener('change', e => save({ blockAdultContent: e.target.checked }))
el('toggle-images').addEventListener('change', e => save({ blockImageSearch: e.target.checked }))

// ── Social media toggles ──────────────────────────────────────────────────────
document.querySelectorAll('[data-social]').forEach(input => {
  input.addEventListener('change', async (e) => {
    const key = e.target.dataset.social
    const blockSocial = { ...(settings.blockSocial || {}), [key]: e.target.checked }
    settings.blockSocial = blockSocial
    await save({ blockSocial })
  })
})

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
window.blockCurrent = blockCurrent
window.allowCurrent = allowCurrent

function openOptions() { chrome.runtime.openOptionsPage?.() }
window.openOptions = openOptions

// ── Helpers ───────────────────────────────────────────────────────────────────
async function save(patch) {
  const merged = { ...settings, ...patch }
  settings = merged
  await ask('SAVE_SETTINGS', merged)
}

function ask(type, data) {
  return new Promise(resolve =>
    chrome.runtime.sendMessage({ type, settings: data }, resolve)
  )
}

function el(id) { return document.getElementById(id) }

// ── Init ──────────────────────────────────────────────────────────────────────
load()
