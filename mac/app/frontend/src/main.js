// K9 Web Protection — Frontend
import { EventsOn } from '../wailsjs/runtime/runtime.js'
const go = () => window.go?.main?.App

// ── Navigation ────────────────────────────────────────────────────────────────
document.querySelectorAll('.nav-item').forEach(item => {
  item.addEventListener('click', () => {
    const page = item.dataset.page
    document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'))
    document.querySelectorAll('.page').forEach(p => p.classList.remove('active'))
    item.classList.add('active')
    document.getElementById(`page-${page}`).classList.add('active')
    if (page === 'dashboard') loadDashboard()
    if (page === 'blocklist') loadBlocklist()
    if (page === 'allowlist') loadAllowlist()
    if (page === 'keywords')  loadKeywords()
    if (page === 'settings')  loadSettings()
  })
})

// ── Auth helper ───────────────────────────────────────────────────────────────
let _pendingAuth = null

async function openAuthModal(title, desc, action, onCancel) {
  const hasPass = await go().HasPassword()
  if (!hasPass) { await action(''); return }
  _pendingAuth = { action, onCancel: onCancel || null }
  document.getElementById('modal-auth-title').textContent = title
  document.getElementById('modal-auth-desc').textContent = desc
  document.getElementById('auth-pw').value = ''
  openModal('modal-auth')
}

async function confirmAuth() {
  if (!_pendingAuth) return
  try {
    await _pendingAuth.action(document.getElementById('auth-pw').value)
    _pendingAuth = null
    closeModal()
  } catch (e) {
    document.getElementById('auth-pw').value = ''
    document.getElementById('auth-pw').focus()
    toast(String(e), 'error')
  }
}

function cancelAuth() {
  if (_pendingAuth?.onCancel) _pendingAuth.onCancel()
  _pendingAuth = null
  closeModal()
}
window.confirmAuth = confirmAuth
window.cancelAuth = cancelAuth

// ── Toast ─────────────────────────────────────────────────────────────────────
function toast(msg, type = 'success') {
  const el = document.getElementById('toast')
  el.textContent = msg
  el.className = `toast show ${type}`
  setTimeout(() => el.classList.remove('show'), 2800)
}

// ── Modal ─────────────────────────────────────────────────────────────────────
function openModal(id) {
  document.getElementById('modal-overlay').classList.add('visible')
  document.getElementById(id).classList.add('visible')
}
function closeModal() {
  document.getElementById('modal-overlay').classList.remove('visible')
  document.querySelectorAll('.modal').forEach(m => m.classList.remove('visible'))
}
window.closeModal = closeModal

// ── Dashboard ─────────────────────────────────────────────────────────────────
async function loadDashboard() {
  const s = await go().GetStatus()

  const active = s.proxyRunning && s.layer1Active
  document.getElementById('status-hero').className = `status-hero ${active ? 'active' : 'inactive'}`
  document.getElementById('status-icon').textContent = active ? '🛡️' : '⚠️'
  document.getElementById('status-text').textContent = active ? 'Protection Active' : 'Protection Inactive'
  document.getElementById('status-sub').textContent  = active
    ? `Proxy on port ${s.proxyPort} · All browsers protected`
    : 'Your browsing is not currently filtered'
  document.getElementById('btn-enable').style.display  = active ? 'none' : 'inline-flex'
  document.getElementById('btn-disable').style.display = active ? 'inline-flex' : 'none'

  document.getElementById('stat-today').textContent = s.blockedToday.toLocaleString()
  document.getElementById('stat-total').textContent = s.totalBlocked.toLocaleString()

  const l1 = document.getElementById('layer1-card')
  l1.className = `stat-card ${s.layer1Active ? 'ok' : 'off'}`
  document.getElementById('stat-layer1').textContent = s.layer1Active ? 'Active' : 'Inactive'

  const l2 = document.getElementById('layer2-card')
  l2.className = `stat-card ${s.proxyRunning ? 'ok' : 'off'}`
  document.getElementById('stat-layer2').textContent = s.proxyRunning ? 'Running' : 'Stopped'

  document.getElementById('stat-db').textContent =
    `${(s.dbDomains + s.dbUrls).toLocaleString()} rules`
  document.getElementById('stat-db-sub').textContent =
    `${s.dbDomains.toLocaleString()} domains · ${s.dbUrls.toLocaleString()} URL patterns · ${s.dbKeywords.toLocaleString()} keywords`

  const topCard = document.getElementById('top-blocked-card')
  const topList = document.getElementById('top-blocked-list')
  if (s.topBlocked?.length > 0) {
    topCard.style.display = 'block'
    topList.innerHTML = s.topBlocked
      .sort((a, b) => b.count - a.count).slice(0, 8)
      .map(e => `<li><span>${escapeHtml(e.domain)}</span><span class="count">${e.count}</span></li>`)
      .join('')
  } else {
    topCard.style.display = 'none'
  }
}

async function enableProtection() {
  try { await go().EnableProtection(); toast('Protection enabled ✓'); loadDashboard() }
  catch (e) { toast(String(e), 'error') }
}
window.enableProtection = enableProtection

async function disableProtection() {
  const hasPass = await go().HasPassword()
  document.getElementById('disable-pw-field').style.display = hasPass ? 'block' : 'none'
  document.getElementById('disable-pw').value = ''
  openModal('modal-disable')
}
window.disableProtection = disableProtection

async function confirmDisable() {
  try {
    await go().DisableProtection(document.getElementById('disable-pw').value)
    closeModal(); toast('Protection disabled'); loadDashboard()
  } catch (e) { toast(String(e), 'error') }
}
window.confirmDisable = confirmDisable

// ── Block list ────────────────────────────────────────────────────────────────
async function loadBlocklist() {
  const data = await go().GetBlocklist()
  document.getElementById('db-domains-count').textContent =
    `${data.builtInDomains.toLocaleString()} domains`
  document.getElementById('db-urls-count').textContent =
    `${data.builtInUrls.toLocaleString()} URL patterns`
  const user = data.userAdded || []
  document.getElementById('user-blocklist-label').textContent =
    `Your additions (${user.length})`
  renderTags('blocklist-tags', user, removeFromBlocklist)
  document.getElementById('blocklist-user-empty').style.display =
    user.length === 0 ? 'block' : 'none'
}

async function addToBlocklist() {
  const input = document.getElementById('blocklist-input')
  const val = input.value.trim()
  if (!val) return
  try { await go().AddToBlocklist(val); input.value = ''; toast(`Blocked: ${val}`); loadBlocklist() }
  catch (e) { toast(String(e), 'error') }
}
window.addToBlocklist = addToBlocklist

async function removeFromBlocklist(domain) {
  try { await go().RemoveFromBlocklist(domain); toast(`Removed: ${domain}`); loadBlocklist() }
  catch (e) { toast(String(e), 'error') }
}

document.getElementById('blocklist-input')
  .addEventListener('keydown', e => { if (e.key === 'Enter') addToBlocklist() })

// ── Allow list ────────────────────────────────────────────────────────────────
async function loadAllowlist() {
  const list = await go().GetAllowlist()
  document.getElementById('allowlist-empty').style.display =
    (!list || list.length === 0) ? 'block' : 'none'
  renderTags('allowlist-tags', list || [], removeFromAllowlist)
}

async function addToAllowlist() {
  const input = document.getElementById('allowlist-input')
  const val = input.value.trim()
  if (!val) return
  try { await go().AddToAllowlist(val); input.value = ''; toast(`Allowed: ${val}`); loadAllowlist() }
  catch (e) { toast(String(e), 'error') }
}
window.addToAllowlist = addToAllowlist

async function removeFromAllowlist(domain) {
  try { await go().RemoveFromAllowlist(domain); toast(`Removed: ${domain}`); loadAllowlist() }
  catch (e) { toast(String(e), 'error') }
}

document.getElementById('allowlist-input')
  .addEventListener('keydown', e => { if (e.key === 'Enter') addToAllowlist() })

// ── Keywords ──────────────────────────────────────────────────────────────────
async function loadKeywords() {
  const data = await go().GetKeywords()
  document.getElementById('db-keywords-count').textContent =
    `${data.builtInCount.toLocaleString()} built-in keywords always active`
  const user = data.userAdded || []
  document.getElementById('user-keywords-label').textContent =
    `Your additions (${user.length})`
  renderTags('keywords-tags', user, removeKeyword)
  document.getElementById('keywords-user-empty').style.display =
    user.length === 0 ? 'block' : 'none'
}

async function addKeyword() {
  const input = document.getElementById('keyword-input')
  const val = input.value.trim()
  if (!val) return
  try { await go().AddKeyword(val); input.value = ''; toast(`Added: ${val}`); loadKeywords() }
  catch (e) { toast(String(e), 'error') }
}
window.addKeyword = addKeyword

async function removeKeyword(kw) {
  try { await go().RemoveKeyword(kw); toast(`Removed: ${kw}`); loadKeywords() }
  catch (e) { toast(String(e), 'error') }
}

document.getElementById('keyword-input')
  .addEventListener('keydown', e => { if (e.key === 'Enter') addKeyword() })

// ── Settings ──────────────────────────────────────────────────────────────────
async function loadSettings() {
  const content = await go().GetContentSettings()
  setToggle('toggle-adult',      content.blockAdultContent)
  setToggle('toggle-imgsearch',  content.blockImageSearch)
  setToggle('toggle-youtube',    content.blockYouTube)
  setToggle('toggle-safesearch', content.safeSearch)
}

function setToggle(id, on) {
  const el = document.getElementById(id)
  if (el) el.classList.toggle('on', on)
}

async function toggleContent(name) {
  const map = {
    adult: 'toggle-adult', imgsearch: 'toggle-imgsearch',
    youtube: 'toggle-youtube', safesearch: 'toggle-safesearch',
  }
  const id = map[name]
  if (!id) return
  const el = document.getElementById(id)
  const turningOff = el.classList.contains('on')

  if (turningOff) {
    await openAuthModal(
      'Password Required',
      'Enter your password to turn off this filter.',
      async (pw) => {
        el.classList.remove('on')
        await saveContentNow(pw)
      },
      null
    )
  } else {
    el.classList.add('on')
    await saveContentNow('')
  }
}
window.toggleContent = toggleContent

async function saveContentNow(pw) {
  try {
    await go().SaveContentSettings(pw, {
      blockAdultContent: document.getElementById('toggle-adult').classList.contains('on'),
      blockImageSearch:  document.getElementById('toggle-imgsearch').classList.contains('on'),
      blockYouTube:      document.getElementById('toggle-youtube').classList.contains('on'),
      safeSearch:        document.getElementById('toggle-safesearch').classList.contains('on'),
    })
    toast('Saved ✓')
  } catch (e) {
    loadSettings()
    toast(String(e), 'error')
  }
}

async function savePassword() {
  const current = document.getElementById('pw-current').value
  const next    = document.getElementById('pw-new').value
  const confirm = document.getElementById('pw-confirm').value
  if (next !== confirm) { toast('Passwords do not match', 'error'); return }
  try {
    await go().SetPassword(current, next)
    document.getElementById('pw-current').value = ''
    document.getElementById('pw-new').value = ''
    document.getElementById('pw-confirm').value = ''
    toast(next === '' ? 'Password removed' : 'Password saved ✓')
  } catch (e) { toast(String(e), 'error') }
}
window.savePassword = savePassword

async function showUninstall() {
  const hasPass = await go().HasPassword()
  document.getElementById('uninstall-pw-field').style.display = hasPass ? 'block' : 'none'
  document.getElementById('uninstall-pw').value = ''
  openModal('modal-uninstall')
}
window.showUninstall = showUninstall

async function confirmUninstall() {
  try {
    await go().Uninstall(document.getElementById('uninstall-pw').value)
    closeModal(); toast('K9 uninstalled. Please remove the app from /Applications.')
  } catch (e) { toast(String(e), 'error') }
}

window.confirmUninstall = confirmUninstall

// ── Helpers ───────────────────────────────────────────────────────────────────
function renderTags(containerId, items, onRemove) {
  const el = document.getElementById(containerId)
  if (!items || items.length === 0) { el.innerHTML = ''; return }
  el.innerHTML = items.map(item => `
    <span class="tag">${escapeHtml(item)}
      <button class="tag-remove" title="Remove"
        onclick="(${onRemove.toString()})('${escapeHtml(item).replace(/'/g,"\\'")}')">×</button>
    </span>`).join('')
}

function escapeHtml(s) {
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;')
    .replace(/>/g,'&gt;').replace(/"/g,'&quot;')
}

// ── Kill / quit guard ─────────────────────────────────────────────────────────
EventsOn('kill-requested', () => {
  document.getElementById('kill-pw').value = ''
  openModal('modal-kill')
})

async function confirmKill() {
  try {
    await go().ConfirmQuit(document.getElementById('kill-pw').value)
    closeModal()
  } catch (e) { toast(String(e), 'error') }
}
window.confirmKill = confirmKill

// ── Init ──────────────────────────────────────────────────────────────────────
window.addEventListener('load', () => {
  const init = () => { window.go?.main?.App ? loadDashboard() : setTimeout(init, 100) }
  init()
})
setInterval(() => {
  if (document.getElementById('page-dashboard')?.classList.contains('active')
      && window.go?.main?.App) loadDashboard()
}, 10000)
