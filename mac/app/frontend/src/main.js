// K9 Web Protection — Professional UI wired to Go backend

const go = () => window.go?.main?.App

// ── Navigation ────────────────────────────────────────────────────────────────
function showTab(tabName) {
  // Update main menu active state
  document.querySelectorAll('#mainMenu li').forEach(li => li.classList.remove('selected'))
  const tabLi = document.getElementById('tab-' + tabName)
  if (tabLi) tabLi.classList.add('selected')

  // Hide all pages
  document.querySelectorAll('[id^="page-"]').forEach(p => p.style.display = 'none')

  const sidebar = document.getElementById('subMenu')
  const sidebarSetup   = document.getElementById('sidebar-setup')
  const sidebarReports = document.getElementById('sidebar-reports')

  if (tabName === 'home') {
    sidebar.style.display = 'none'
    showPage('home')
    loadDashboard()
  } else if (tabName === 'reports') {
    sidebar.style.display = 'flex'
    sidebarSetup.style.display   = 'none'
    sidebarReports.style.display = 'block'
    clearSideLinks()
    document.querySelector('#item-summary a').classList.add('selected')
    showPage('reports')
    loadActivity()
  } else if (tabName === 'setup') {
    sidebar.style.display = 'flex'
    sidebarSetup.style.display   = 'block'
    sidebarReports.style.display = 'none'
    showPage('categories')
    setSideActive('categories')
    loadCategories()
  } else if (tabName === 'help') {
    sidebar.style.display = 'none'
    showPage('help')
  }
}

function showPage(name) {
  document.querySelectorAll('[id^="page-"]').forEach(p => p.style.display = 'none')
  const pg = document.getElementById('page-' + name)
  if (pg) pg.style.display = 'block'
}

function setSideActive(name) {
  clearSideLinks()
  const item = document.getElementById('item-' + name) ||
               document.querySelector(`[data-page="${name}"]`)?.parentElement
  if (item) item.classList.add('selected')
}

function clearSideLinks() {
  document.querySelectorAll('#subMenu li').forEach(li => li.classList.remove('selected'))
}

// Tab clicks
document.querySelectorAll('.navLink').forEach(a => {
  a.addEventListener('click', e => {
    e.preventDefault()
    showTab(a.dataset.tab)
  })
})

// Sidebar clicks
document.querySelectorAll('.sideLink').forEach(a => {
  a.addEventListener('click', e => {
    e.preventDefault()
    const page = a.dataset.page
    clearSideLinks()
    a.parentElement.classList.add('selected')
    showPage(page)
    if (page === 'categories') loadCategories()
    if (page === 'exceptions') loadExceptions()
    if (page === 'keywords')   loadKeywords()
    if (page === 'safesearch') loadSafeSearch()
    if (page === 'password')   loadPasswordSettings()
    if (page === 'advanced')   loadAdvanced()
    if (page === 'update')     loadUpdate()
    if (page === 'effects')    loadBlockingEffects()
  })
})

// ── Notification bar ──────────────────────────────────────────────────────────
let notifyTimer
function notify(msg, type = 'ok') {
  const bar = document.getElementById('notifyBar')
  bar.textContent = type === 'ok' ? '✓ ' + msg : '✖ ' + msg
  bar.className = `notifyBar show ${type}`
  clearTimeout(notifyTimer)
  notifyTimer = setTimeout(() => bar.classList.remove('show'), 3000)
}

// ── Dashboard ─────────────────────────────────────────────────────────────────
async function loadDashboard() {
  if (!window.go?.main?.App) return
  const [s, bl, al, cs] = await Promise.all([
    go().GetStatus(),
    go().GetBlocklist(),
    go().GetAllowlist(),
    go().GetContentSettings(),
  ])

  const l1ok   = s.layer1Active
  const l2ok   = s.proxyRunning
  const active = l1ok && l2ok

  // ── Header badge ──
  const badge = document.getElementById('k9StatusDot')
  if (active) { badge.textContent = 'Active';   badge.className = 'status-badge active' }
  else        { badge.textContent = 'Inactive'; badge.className = 'status-badge' }

  // ── Inactive warning bar ──
  const inactiveBar = document.getElementById('protection-inactive-bar')
  if (inactiveBar) inactiveBar.style.display = active ? 'none' : 'flex'
  const btnEnable  = document.getElementById('btn-enable')
  const btnDisable = document.getElementById('btn-disable')
  if (btnEnable)  btnEnable.style.display  = active ? 'none'       : 'inline-flex'
  if (btnDisable) btnDisable.style.display = active ? 'inline-flex' : 'none'

  // ── Stat cards ──
  document.getElementById('stat-today').textContent = s.blockedToday.toLocaleString()
  document.getElementById('stat-total').textContent = s.totalBlocked.toLocaleString()

  const allowCount = al?.length ?? 0
  const blockCount = bl?.userAdded?.length ?? 0
  const excEl = document.getElementById('stat-exceptions')
  if (excEl) excEl.innerHTML =
    `${allowCount} <span style="font-size:9px;color:#888;font-weight:600">allow</span>&nbsp;&nbsp;` +
    `${blockCount} <span style="font-size:9px;color:#888;font-weight:600">block</span>`

  const levelLabel = cs.blockAdultContent ? 'Default' : 'Minimal'
  const levelSubEl = document.getElementById('stat-filter-sub')
  const levelEl    = document.getElementById('stat-filter-level')
  if (levelEl) levelEl.textContent = levelLabel
  if (levelSubEl) levelSubEl.textContent = cs.blockAdultContent ? '18 categories blocked' : 'Threats only'

  // ── DB chips ──
  const setText = (id, val) => { const el = document.getElementById(id); if (el) el.textContent = val }
  setText('db-domains',  s.dbDomains.toLocaleString())
  setText('db-urls',     s.dbUrls.toLocaleString())
  setText('db-keywords', s.dbKeywords.toLocaleString())

  // ── Protection modules ──
  const setMod = (dotId, valId, on) => {
    const dot = document.getElementById(dotId)
    const val = document.getElementById(valId)
    if (dot) dot.className = 'prot-dot ' + (on ? 'on' : 'off')
    if (val) { val.textContent = on ? 'Active' : 'Inactive'; val.className = 'prot-val ' + (on ? 'active' : 'inactive') }
  }
  setMod('dot-web-protection', 'mod-web-protection', active)
  setMod('dot-malware',        'mod-malware',        l2ok)
  setMod('dot-safesearch',     'mod-safesearch',     cs.safeSearch !== false)
  setMod('dot-https',          'mod-https',          l2ok)
  setMod('dot-dns',            'mod-dns',            l1ok)

  // ── Top blocked categories bar chart (from topBlocked domain data) ──
  renderTopCategoriesChart(s.topBlocked)

  // ── Recent blocked activity (placeholder — real data needs backend work, see plan.md) ──
  renderRecentActivity(s.topBlocked)
}

function renderTopCategoriesChart(topBlocked) {
  const el = document.getElementById('top-categories-bars')
  if (!el) return
  if (!topBlocked?.length) {
    el.innerHTML = '<div class="dash-bar-row"><span class="dash-bar-label" style="width:auto;color:#aaa">No data yet</span></div>'
    return
  }
  const top5 = topBlocked.slice(0, 5)
  const max  = top5[0]?.count || 1
  const colors = ['#991b1b','#92400e','#1e40af','#854d0e','#6b21a8']
  el.innerHTML = top5.map((e, i) => {
    const label = e.domain.length > 12 ? e.domain.slice(0, 11) + '…' : e.domain
    const pct = Math.round(e.count / max * 100)
    return `<div class="dash-bar-row">
      <span class="dash-bar-label">${esc(label)}</span>
      <div class="dash-bar-track"><div class="dash-bar-fill" style="width:${pct}%;background:${colors[i]}"></div></div>
      <span class="dash-bar-val">${e.count}</span>
    </div>`
  }).join('')
}

function fmtTime(iso) {
  if (!iso) return '—:—'
  const d = new Date(iso)
  if (isNaN(d)) return '—:—'
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })
}

function renderRecentActivity(topBlocked) {
  const el = document.getElementById('recent-activity')
  if (!el) return
  if (!topBlocked?.length) {
    el.innerHTML = '<div class="dash-recent-row"><span class="dash-recent-url muted">No activity recorded yet.</span></div>'
    return
  }
  const catClass = (domain) => {
    if (/porn|xxx|adult|sex|hub|babes|eliteb/.test(domain)) return 'cat-porn'
    if (/casino|gambl|poker|slot/.test(domain)) return 'cat-gamble'
    if (/malware|trojan|virus|hack/.test(domain)) return 'cat-malware'
    if (/phish|fake/.test(domain)) return 'cat-phish'
    if (/hate|violen/.test(domain)) return 'cat-violence'
    return 'cat-other'
  }
  const catLabel = (domain) => {
    if (/porn|xxx|adult|sex|hub|babes|eliteb/.test(domain)) return 'Pornography'
    if (/casino|gambl|poker|slot/.test(domain)) return 'Gambling'
    if (/malware|trojan|virus|hack/.test(domain)) return 'Malware'
    if (/phish|fake/.test(domain)) return 'Phishing'
    if (/hate|violen/.test(domain)) return 'Violence'
    return 'Blocked'
  }
  el.innerHTML = topBlocked.slice(0, 8).map(e =>
    `<div class="dash-recent-row">
      <span class="dash-recent-time">${fmtTime(e.lastSeen)}</span>
      <span class="dash-recent-url">${esc(e.domain)}</span>
      <span class="dash-recent-cat ${catClass(e.domain)}">${catLabel(e.domain)}</span>
    </div>`
  ).join('')
}

function verifyProtection() {
  if (!window.go?.main?.App) return
  go().GetStatus().then(s => {
    const ok = s.layer1Active && s.proxyRunning
    notify(ok ? 'All modules verified — protection is active.' : 'Warning: protection is not fully active.', ok ? 'ok' : 'err')
  })
}
window.verifyProtection = verifyProtection

async function enableProtection() {
  try { await go().EnableProtection(); notify('Protection enabled.', 'ok'); loadDashboard() }
  catch (e) { notify(String(e), 'err') }
}
window.enableProtection = enableProtection

// ── Disable modal ─────────────────────────────────────────────────────────────
async function showDisableModal() {
  const hasPw = await go().HasPassword()
  document.getElementById('modal-pw-row').style.display = hasPw ? 'block' : 'none'
  document.getElementById('disable-pw').value = ''
  document.getElementById('modalBg').classList.add('show')
}
window.showDisableModal = showDisableModal

function closeModal() { document.getElementById('modalBg').classList.remove('show') }
window.closeModal = closeModal

async function confirmDisable() {
  const pw = document.getElementById('disable-pw').value
  try {
    await go().DisableProtection(pw)
    closeModal(); notify('Protection disabled.', 'ok'); loadDashboard()
  } catch (e) { notify(String(e), 'err') }
}
window.confirmDisable = confirmDisable

// ── Activity ──────────────────────────────────────────────────────────────────
async function loadActivity() {
  const s = await go().GetStatus()
  document.getElementById('gen-total').textContent = s.totalBlocked.toLocaleString()
  document.getElementById('gen-today').textContent = s.blockedToday.toLocaleString()

  const container = document.getElementById('activity-rows')
  if (!s.topBlocked?.length) {
    container.innerHTML = '<div class="act-row"><span class="muted">No activity recorded yet.</span></div>'
    return
  }
  container.innerHTML = s.topBlocked
    .sort((a, b) => b.count - a.count)
    .map(e => `<div class="act-row">
      <span style="color:var(--red)">&#x29B8; ${esc(e.domain)}</span>
      <span>${e.count}</span>
    </div>`)
    .join('')
}

// ── Categories ────────────────────────────────────────────────────────────────
async function loadCategories() {
  const s = await go().GetContentSettings()
  const level = s.blockAdultContent ? 'default' : 'minimal'

  document.querySelectorAll('[id^="setting-"]').forEach(el => el.classList.remove('selected'))
  const row = document.getElementById('setting-' + level)
  if (row) row.classList.add('selected')

  const radio = document.getElementById('level-' + level)
  if (radio) radio.checked = true

  document.getElementById('cat-adult').checked      = s.blockAdultContent !== false
  document.getElementById('cat-youtube').checked    = s.blockYouTube === true
  document.getElementById('cat-safesearch').checked = s.safeSearch !== false

  updateCategoryUI()
}

function updateCategoryUI() {
  const checked = document.querySelector('input[name="level"]:checked')?.value || 'default'
  document.querySelectorAll('[id^="setting-"]').forEach(el => el.classList.remove('selected'))
  const row = document.getElementById('setting-' + checked)
  if (row) row.classList.add('selected')
  const showHide = (id, show) => { const el = document.getElementById(id); if (el) el.style.display = show ? 'block' : 'none' }
  showHide('cat-list-high',    checked === 'high')
  showHide('cat-list-default', checked === 'default')
  showHide('custom-cats',      checked === 'custom')
}

document.querySelectorAll('.radioLink').forEach(a => {
  a.addEventListener('click', e => {
    e.preventDefault()
    const radioId = a.id.replace('radio-', 'level-')
    const radio = document.getElementById(radioId)
    if (radio) { radio.checked = true; updateCategoryUI() }
  })
})

async function saveCategories() {
  const level = document.querySelector('input[name="level"]:checked')?.value || 'default'
  let blockAdultContent = true, blockYouTube = false, safeSearch = true, blockImageSearch = false
  if (level === 'high')     { blockAdultContent = true;  blockYouTube = true; }
  if (level === 'moderate') { blockAdultContent = true;  blockYouTube = false; }
  if (level === 'minimal')  { blockAdultContent = false; blockYouTube = false; }
  if (level === 'monitor')  { blockAdultContent = false; blockYouTube = false; }
  if (level === 'custom')   {
    blockAdultContent = document.getElementById('cat-adult').checked
    blockYouTube      = document.getElementById('cat-youtube').checked
    safeSearch        = document.getElementById('cat-safesearch').checked
  }
  try {
    await go().SaveContentSettings('', { blockAdultContent, blockYouTube, safeSearch, blockImageSearch })
    notify('Category settings saved.', 'ok')
  } catch (e) { notify(String(e), 'err') }
}
window.saveCategories = saveCategories

// ── Exceptions ────────────────────────────────────────────────────────────────
async function loadExceptions() {
  const bl = await go().GetBlocklist()
  renderList('blocklist-items', bl.userAdded, removeFromBlocklist, 'red')
  const al = await go().GetAllowlist()
  renderList('allowlist-items', al, removeFromAllowlist, 'green')
}

function renderList(id, items, removeFn, color) {
  const el = document.getElementById(id)
  if (!items?.length) {
    el.innerHTML = '<span style="color:#888; font-style:italic">No entries in this list</span>'
    return
  }
  el.innerHTML = items.map(item =>
    `<div style="font-size:12px; padding:2px 0; display:flex; align-items:center; gap:6px">
       <span style="color:${color === 'red' ? '#cc2222' : '#228B22'}; font-weight:bold">&#x29B8;</span>
       <span style="color:#003E7E">${esc(item)}</span>
       <a href="#" style="color:#cc2222; font-weight:bold; font-size:14px; text-decoration:none; margin-left:4px"
          onclick="(${removeFn.name})('${esc(item).replace(/'/g,"\\'")}'); return false;">&times;</a>
     </div>`
  ).join('')
}

async function addToBlocklist() {
  const input = document.getElementById('listTb-0')
  const val = input.value.trim()
  if (!val) return
  try { await go().AddToBlocklist(val); input.value = ''; loadExceptions(); notify('Added to Always Block list.') }
  catch (e) { notify(String(e), 'err') }
}
async function removeFromBlocklist(domain) {
  await go().RemoveFromBlocklist(domain); loadExceptions(); notify('Removed.')
}
async function addToAllowlist() {
  const input = document.getElementById('listTb-1')
  const val = input.value.trim()
  if (!val) return
  try { await go().AddToAllowlist(val); input.value = ''; loadExceptions(); notify('Added to Always Allow list.') }
  catch (e) { notify(String(e), 'err') }
}
async function removeFromAllowlist(domain) {
  await go().RemoveFromAllowlist(domain); loadExceptions(); notify('Removed.')
}
window.addToBlocklist = addToBlocklist
window.addToAllowlist = addToAllowlist

document.getElementById('listTb-0').addEventListener('keydown', e => { if (e.key === 'Enter') addToBlocklist() })
document.getElementById('listTb-1').addEventListener('keydown', e => { if (e.key === 'Enter') addToAllowlist() })

// ── Keywords ──────────────────────────────────────────────────────────────────
async function loadKeywords() {
  const data = await go().GetKeywords()
  document.getElementById('kw-builtin-desc').textContent =
    `Built-in: ${data.builtInCount.toLocaleString()} keyword entries (always active)`
  const el = document.getElementById('kw-items')
  if (!data.userAdded?.length) {
    el.innerHTML = '<span style="color:#888; font-style:italic">No custom keywords added.</span>'
    return
  }
  el.innerHTML = data.userAdded.map(kw =>
    `<div style="font-size:12px; padding:2px 0; display:flex; align-items:center; gap:6px">
       <span style="color:#cc6600; font-weight:bold">&#x25B6;</span>
       <span>${esc(kw)}</span>
       <a href="#" style="color:#cc2222; font-weight:bold; font-size:14px; text-decoration:none; margin-left:4px"
          onclick="removeKeyword('${esc(kw).replace(/'/g,"\\'")}'); return false;">&times;</a>
     </div>`
  ).join('')
}

async function addKeyword() {
  const input = document.getElementById('kw-input')
  const val = input.value.trim()
  if (!val) return
  try { await go().AddKeyword(val); input.value = ''; loadKeywords(); notify('Keyword added.') }
  catch (e) { notify(String(e), 'err') }
}
async function removeKeyword(kw) {
  await go().RemoveKeyword(kw); loadKeywords(); notify('Keyword removed.')
}
async function saveKeywords() { notify('Custom keywords are saved automatically.', 'ok') }
window.addKeyword = addKeyword
window.removeKeyword = removeKeyword
window.saveKeywords = saveKeywords
document.getElementById('kw-input').addEventListener('keydown', e => { if (e.key === 'Enter') addKeyword() })

// ── Safe Search ───────────────────────────────────────────────────────────────
async function loadSafeSearch() {
  const s = await go().GetContentSettings()
  const cb = document.getElementById('cb-safesearch')
  if (cb) cb.checked = s.safeSearch !== false
}

async function saveSafeSearch() {
  const on = document.getElementById('cb-safesearch').checked
  try {
    const s = await go().GetContentSettings()
    await go().SaveContentSettings('', { ...s, safeSearch: on })
    notify('Safe Search setting saved.', 'ok')
  } catch (e) { notify(String(e), 'err') }
}
window.saveSafeSearch = saveSafeSearch

// ── Password ──────────────────────────────────────────────────────────────────
async function loadPasswordSettings() {
  const adv   = await go().GetAdvancedSettings()
  const proxy = await go().GetProxySettings()
  document.getElementById('setting-delay').value = String(adv.disableDelayHours || 0)
}

async function savePassword() {
  const current = document.getElementById('pw-current').value
  const next    = document.getElementById('pw-new').value
  const confirm = document.getElementById('pw-confirm').value
  if (next !== confirm) { notify('Passwords do not match.', 'err'); return }
  try {
    await go().SetPassword(current, next)
    document.getElementById('pw-current').value = ''
    document.getElementById('pw-new').value     = ''
    document.getElementById('pw-confirm').value = ''
    notify(next ? 'Password saved.' : 'Password removed.', 'ok')
  } catch (e) { notify(String(e), 'err') }
}
async function removePassword() {
  const current = document.getElementById('pw-current').value
  try { await go().SetPassword(current, ''); notify('Password removed.', 'ok') }
  catch (e) { notify(String(e), 'err') }
}
window.savePassword = savePassword
window.removePassword = removePassword

async function saveAdvancedSettings() {
  const delay = parseInt(document.getElementById('setting-delay').value)
  try {
    const adv = await go().GetAdvancedSettings()
    await go().SaveAdvancedSettings('', { ...adv, disableDelayHours: delay })
    notify('Settings saved.', 'ok')
  } catch (e) { notify(String(e), 'err') }
}
window.saveAdvancedSettings = saveAdvancedSettings

// ── Advanced page ─────────────────────────────────────────────────────────────
async function loadAdvanced() {
  if (!window.go?.main?.App) return
  const [proxy, adv] = await Promise.all([go().GetProxySettings(), go().GetAdvancedSettings()])
  const portEl = document.getElementById('adv-proxy-port')
  const autoEl = document.getElementById('adv-autostart')
  if (portEl) portEl.value = proxy.proxyPort
  if (autoEl) autoEl.value = proxy.autoStart ? 'true' : 'false'
}
async function saveAdvanced() {
  const port     = parseInt(document.getElementById('adv-proxy-port')?.value || 2372)
  const autoStart = document.getElementById('adv-autostart')?.value === 'true'
  try {
    await go().SaveProxySettings({ proxyPort: port, autoStart })
    notify('Advanced settings saved.', 'ok')
  } catch (e) { notify(String(e), 'err') }
}
async function installCA() {
  const btn = document.getElementById('btn-install-ca')
  const status = document.getElementById('ca-status')
  btn.disabled = true
  status.textContent = 'Installing…'
  try {
    await go().InstallCACert()
    status.style.color = '#1a8a3a'
    status.textContent = 'Trusted successfully. Restart your browser.'
  } catch (e) {
    status.style.color = '#cc3333'
    status.textContent = String(e)
  }
  btn.disabled = false
}

window.loadAdvanced = loadAdvanced
window.saveAdvanced = saveAdvanced
window.installCA = installCA

// ── Time Restrictions (frontend-only until backend is built) ──────────────────
function toggleTimeRestrictions() {
  const enabled = document.getElementById('cb-time-enabled')?.checked
  const showHide = (id, show) => { const el = document.getElementById(id); if (el) el.style.display = show ? 'block' : 'none' }
  showHide('time-schedule', enabled)
  showHide('time-disabled-msg', !enabled)
}
window.toggleTimeRestrictions = toggleTimeRestrictions

// ── Blocking Effects (frontend-only until backend is built) ───────────────────
function loadBlockingEffects() {
  const msgEl = document.getElementById('eff-custom-msg')
  if (msgEl) msgEl.value = ''
}
async function saveBlockingEffects() {
  const msg = document.getElementById('eff-custom-msg')?.value?.trim()
  if (!window.go?.main?.App) { notify('Blocking effects saved (backend not yet implemented).', 'ok'); return }
  try {
    const adv = await go().GetAdvancedSettings()
    await go().SaveAdvancedSettings('', { ...adv, blockedMessage: msg || adv.blockedMessage })
    notify('Blocking effects saved.', 'ok')
  } catch (e) { notify(String(e), 'err') }
}
window.loadBlockingEffects = loadBlockingEffects
window.saveBlockingEffects = saveBlockingEffects

// ── K10 Update ────────────────────────────────────────────────────────────────
async function loadUpdate() {
  if (!window.go?.main?.App) return
  const s = await go().GetStatus()
  const dbEl = document.getElementById('upd-db-size')
  if (dbEl) dbEl.innerHTML = `<strong>${s.dbDomains.toLocaleString()}</strong> domains, <strong>${s.dbUrls.toLocaleString()}</strong> URL patterns, <strong>${s.dbKeywords.toLocaleString()}</strong> keywords`
}
function checkForUpdate() {
  notify('Update check requires backend support — see plan.md item 8.', 'ok')
}
window.loadUpdate = loadUpdate
window.checkForUpdate = checkForUpdate

// ── Uninstall ─────────────────────────────────────────────────────────────────
async function showUninstall() {
  const hasPw = await go().HasPassword()
  const pw = hasPw ? prompt('Enter password to uninstall K9 Web Protection:') : ''
  if (pw === null) return
  try { await go().Uninstall(pw || ''); notify('K9 uninstalled. Please delete the app from /Applications.', 'ok') }
  catch (e) { notify(String(e), 'err') }
}
window.showUninstall = showUninstall

// ── Utility ───────────────────────────────────────────────────────────────────
function esc(s) {
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')
}

// ── Init ──────────────────────────────────────────────────────────────────────
window.addEventListener('load', () => {
  const init = () => {
    if (window.go?.main?.App) { loadDashboard() }
    else { setTimeout(init, 100) }
  }
  init()
})

setInterval(() => {
  const home = document.getElementById('page-home')
  if (home?.style.display !== 'none' && window.go?.main?.App) loadDashboard()
}, 10000)
