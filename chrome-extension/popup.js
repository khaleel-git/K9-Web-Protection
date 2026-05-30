// K9 Web Protection — Popup

let settings  = {}
let currentDomain = ''

// ── Load ──────────────────────────────────────────────────────────────────────
async function load() {
  settings = await new Promise(resolve =>
    chrome.runtime.sendMessage({ type: 'GET_SETTINGS' }, resolve)
  )
  const { stats = {} } = settings

  // Master toggle
  document.getElementById('toggle-enabled').checked = settings.enabled !== false
  updateStatusBar(settings.enabled !== false)

  // Stats
  document.getElementById('stat-today').textContent = (stats.blockedToday || 0).toLocaleString()
  document.getElementById('stat-total').textContent = (stats.totalBlocked  || 0).toLocaleString()

  // Content toggles
  document.getElementById('toggle-adult').checked   = settings.blockAdultContent !== false
  document.getElementById('toggle-images').checked  = settings.blockImageSearch  === true
  document.getElementById('toggle-youtube').checked = settings.blockYouTube      === true

  // Current tab domain
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
  if (tab?.url) {
    try {
      currentDomain = new URL(tab.url).hostname
      document.getElementById('current-domain').textContent = currentDomain

      const blocked  = (settings.userBlocklist  || []).includes(currentDomain)
      const allowed  = (settings.userAllowlist || []).includes(currentDomain)
      document.getElementById('btn-block').classList.toggle('active', blocked)
      document.getElementById('btn-allow').classList.toggle('active', allowed)
    } catch (_) {
      document.getElementById('current-domain').textContent = 'Not a webpage'
    }
  }
}

// ── Status bar ────────────────────────────────────────────────────────────────
function updateStatusBar(on) {
  const bar = document.getElementById('status-bar')
  const lbl = document.getElementById('status-label')
  if (on) {
    bar.classList.remove('off')
    lbl.textContent = '✓ Protection Active'
  } else {
    bar.classList.add('off')
    lbl.textContent = '⚠ Protection Disabled'
  }
}

// ── Save settings ─────────────────────────────────────────────────────────────
async function save(patch) {
  const merged = { ...settings, ...patch }
  settings = merged
  await new Promise(resolve =>
    chrome.runtime.sendMessage({ type: 'SAVE_SETTINGS', settings: merged }, resolve)
  )
}

// ── Master toggle ─────────────────────────────────────────────────────────────
document.getElementById('toggle-enabled').addEventListener('change', e => {
  const on = e.target.checked
  updateStatusBar(on)
  save({ enabled: on })
})

// ── Content toggles ───────────────────────────────────────────────────────────
document.getElementById('toggle-adult').addEventListener('change', e => {
  save({ blockAdultContent: e.target.checked })
})
document.getElementById('toggle-images').addEventListener('change', e => {
  save({ blockImageSearch: e.target.checked })
})
document.getElementById('toggle-youtube').addEventListener('change', e => {
  save({ blockYouTube: e.target.checked })
})

// ── Block / Allow current site ────────────────────────────────────────────────
function blockCurrent() {
  if (!currentDomain) return
  const btn = document.getElementById('btn-block')
  if (btn.classList.contains('active')) return  // already blocked

  const userBlocklist = [...(settings.userBlocklist || [])]
  const userAllowlist = (settings.userAllowlist || []).filter(d => d !== currentDomain)
  if (!userBlocklist.includes(currentDomain)) userBlocklist.push(currentDomain)

  btn.classList.add('active')
  document.getElementById('btn-allow').classList.remove('active')
  save({ userBlocklist, userAllowlist })
}

function allowCurrent() {
  if (!currentDomain) return
  const btn = document.getElementById('btn-allow')
  if (btn.classList.contains('active')) return  // already allowed

  const userAllowlist = [...(settings.userAllowlist || [])]
  const userBlocklist = (settings.userBlocklist || []).filter(d => d !== currentDomain)
  if (!userAllowlist.includes(currentDomain)) userAllowlist.push(currentDomain)

  btn.classList.add('active')
  document.getElementById('btn-block').classList.remove('active')
  save({ userAllowlist, userBlocklist })
}

function openOptions() {
  chrome.runtime.openOptionsPage?.()
}

// ── Init ──────────────────────────────────────────────────────────────────────
load()
