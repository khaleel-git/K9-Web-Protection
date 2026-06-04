// K9 Web Protection — Classic Admin UI

const go = () => window.go?.main?.App

// ── Tab navigation ────────────────────────────────────────────────────────────
document.querySelectorAll('.k9-tab').forEach(tab => {
  tab.addEventListener('click', () => {
    const tabName  = tab.dataset.tab
    const firstPage = tab.dataset.page

    document.querySelectorAll('.k9-tab').forEach(t => t.classList.remove('active'))
    tab.classList.add('active')

    // Show/hide sidebar
    const sidebar = document.getElementById('k9-sidebar')
    const sidebarSetup    = document.getElementById('sidebar-setup')
    const sidebarActivity = document.getElementById('sidebar-activity')

    if (tabName === 'home') {
      sidebar.style.display = 'none'
      showPage('home')
    } else if (tabName === 'activity') {
      sidebar.style.display = 'block'
      sidebarSetup.style.display    = 'none'
      sidebarActivity.style.display = 'block'
      showPage('activity')
      loadActivity()
    } else if (tabName === 'setup') {
      sidebar.style.display = 'block'
      sidebarSetup.style.display    = 'block'
      sidebarActivity.style.display = 'none'
      showPage(firstPage)
    }
  })
})

// Sidebar link navigation
document.querySelectorAll('.sidebar-link').forEach(link => {
  link.addEventListener('click', () => {
    const page = link.dataset.page
    document.querySelectorAll('.sidebar-link').forEach(l => l.classList.remove('active'))
    link.classList.add('active')
    showPage(page)

    // Load data for each section
    if (page === 'setup-exceptions') loadExceptions()
    if (page === 'setup-keywords')   loadKeywords()
    if (page === 'setup-categories') loadCategories()
    if (page === 'setup-password')   loadPasswordSettings()
    if (page === 'activity')         loadActivity()
  })
})

function showPage(id) {
  document.querySelectorAll('.page').forEach(p => p.classList.remove('active'))
  const page = document.getElementById('page-' + id)
  if (page) page.classList.add('active')
}

// ── Status message ────────────────────────────────────────────────────────────
let msgTimer
function msg(text, type = 'success') {
  const el = document.getElementById('k9-msg')
  el.textContent = text
  el.className = `k9-msg show ${type}`
  clearTimeout(msgTimer)
  msgTimer = setTimeout(() => el.classList.remove('show'), 3000)
}

// ── Dashboard / Home ──────────────────────────────────────────────────────────
async function loadDashboard() {
  if (!window.go?.main?.App) return
  const s = await go().GetStatus()

  const l1ok = s.layer1Active
  const l2ok = s.proxyRunning

  document.getElementById('status-layer1').innerHTML = l1ok
    ? '<span class="dot-active">●</span> Active'
    : '<span class="dot-inactive">●</span> Inactive'
  document.getElementById('status-layer2').innerHTML = l2ok
    ? '<span class="dot-active">●</span> Running'
    : '<span class="dot-inactive">●</span> Stopped'
  document.getElementById('status-port').textContent    = s.proxyPort
  document.getElementById('stat-today').textContent     = s.blockedToday.toLocaleString()
  document.getElementById('stat-total').textContent     = s.totalBlocked.toLocaleString()
  document.getElementById('stat-domains').textContent   = s.dbDomains.toLocaleString()
  document.getElementById('stat-urls').textContent      = s.dbUrls.toLocaleString()
  document.getElementById('stat-keywords').textContent  = s.dbKeywords.toLocaleString()

  const active = l1ok && l2ok
  document.getElementById('btn-enable').style.display  = active ? 'none' : 'inline-block'
  document.getElementById('btn-disable').style.display = active ? 'inline-block' : 'none'

  const pill = document.getElementById('header-status')
  if (active) { pill.textContent = '● Active';   pill.className = 'k9-status-pill' }
  else        { pill.textContent = '● Inactive'; pill.className = 'k9-status-pill inactive' }
}

async function enableProtection() {
  try {
    await go().EnableProtection()
    msg('Protection enabled successfully.', 'success')
    loadDashboard()
  } catch (e) { msg(String(e), 'error') }
}
window.enableProtection = enableProtection

// ── Disable modal ─────────────────────────────────────────────────────────────
async function showDisableModal() {
  const hasPw = await go().HasPassword()
  document.getElementById('disable-pw-row').style.display = hasPw ? 'block' : 'none'
  document.getElementById('disable-pw').value = ''
  document.getElementById('modal-overlay').classList.add('show')
}
window.showDisableModal = showDisableModal

function closeModal() {
  document.getElementById('modal-overlay').classList.remove('show')
}
window.closeModal = closeModal

async function confirmDisable() {
  const pw = document.getElementById('disable-pw').value
  try {
    await go().DisableProtection(pw)
    closeModal()
    msg('Protection disabled.', 'error')
    loadDashboard()
  } catch (e) { msg(String(e), 'error') }
}
window.confirmDisable = confirmDisable

// ── Activity ──────────────────────────────────────────────────────────────────
async function loadActivity() {
  const s = await go().GetStatus()
  document.getElementById('gen-total').textContent = s.totalBlocked.toLocaleString()
  document.getElementById('gen-today').textContent = s.blockedToday.toLocaleString()

  const tbody = document.getElementById('activity-categories')
  if (!s.topBlocked?.length) {
    tbody.innerHTML = '<tr><td colspan="2" class="empty-row">No activity recorded yet.</td></tr>'
    return
  }
  tbody.innerHTML = s.topBlocked
    .sort((a, b) => b.count - a.count)
    .map(e => `<tr>
      <td><span class="dot-inactive">⊘</span> ${escHtml(e.domain)}</td>
      <td style="text-align:right">${e.count}</td>
    </tr>`).join('')
}

// ── Web Categories ────────────────────────────────────────────────────────────
async function loadCategories() {
  const s = await go().GetContentSettings()
  // Map settings to level
  const level = s.blockAdultContent ? 'default' : 'minimal'
  document.querySelectorAll('input[name="level"]').forEach(r => {
    r.checked = (r.value === level)
  })
  document.getElementById('cat-adult').checked      = s.blockAdultContent !== false
  document.getElementById('cat-youtube').checked    = s.blockYouTube === true
  document.getElementById('cat-safesearch').checked = s.safeSearch !== false
  updateLevelUI()
}

function updateLevelUI() {
  const val = document.querySelector('input[name="level"]:checked')?.value || 'default'
  document.querySelectorAll('.level-row').forEach(row => row.classList.remove('selected'))
  const activeRow = document.getElementById('level-' + val)
  if (activeRow) activeRow.classList.add('selected')
  document.getElementById('custom-section').style.display = val === 'custom' ? 'block' : 'none'
}
document.querySelectorAll('input[name="level"]').forEach(r =>
  r.addEventListener('change', updateLevelUI)
)

async function saveCategories() {
  const level = document.querySelector('input[name="level"]:checked')?.value || 'default'
  let blockAdult = true, blockYouTube = false, safeSearch = true
  if (level === 'high')     { blockAdult = true;  blockYouTube = true; }
  if (level === 'moderate') { blockAdult = true;  blockYouTube = false; }
  if (level === 'minimal')  { blockAdult = false; blockYouTube = false; }
  if (level === 'custom')   {
    blockAdult   = document.getElementById('cat-adult').checked
    blockYouTube = document.getElementById('cat-youtube').checked
    safeSearch   = document.getElementById('cat-safesearch').checked
  }
  try {
    await go().SaveContentSettings({ blockAdultContent: blockAdult, blockYouTube, safeSearch, blockImageSearch: false })
    msg('Category settings saved.', 'success')
  } catch (e) { msg(String(e), 'error') }
}
window.saveCategories = saveCategories

// ── Web Site Exceptions ───────────────────────────────────────────────────────
async function loadExceptions() {
  const bl = await go().GetBlocklist()
  renderExceptionList('blocklist-items', bl.userAdded, removeFromBlocklist, 'red')
  const al = await go().GetAllowlist()
  renderExceptionList('allowlist-items', al, removeFromAllowlist, 'green')
}

function renderExceptionList(id, items, removeFn, color) {
  const el = document.getElementById(id)
  if (!items?.length) {
    el.innerHTML = '<div class="empty-row">No entries in this list</div>'
    return
  }
  el.innerHTML = items.map(item =>
    `<div class="exception-item">
       <span style="color:${color === 'red' ? 'var(--red)' : 'var(--green)'}">⊘ ${escHtml(item)}</span>
       <button class="remove-btn" data-item="${escHtml(item)}" data-fn="${removeFn.name}">✕</button>
     </div>`
  ).join('')
  el.querySelectorAll('.remove-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      if (btn.dataset.fn === 'removeFromBlocklist') removeFromBlocklist(btn.dataset.item)
      else removeFromAllowlist(btn.dataset.item)
    })
  })
}

async function addToBlocklist() {
  const input = document.getElementById('blocklist-input')
  const val = input.value.trim()
  if (!val) return
  try { await go().AddToBlocklist(val); input.value = ''; loadExceptions(); msg('Added to block list.') }
  catch (e) { msg(String(e), 'error') }
}
async function removeFromBlocklist(domain) {
  await go().RemoveFromBlocklist(domain); loadExceptions(); msg('Removed from block list.')
}
async function addToAllowlist() {
  const input = document.getElementById('allowlist-input')
  const val = input.value.trim()
  if (!val) return
  try { await go().AddToAllowlist(val); input.value = ''; loadExceptions(); msg('Added to allow list.') }
  catch (e) { msg(String(e), 'error') }
}
async function removeFromAllowlist(domain) {
  await go().RemoveFromAllowlist(domain); loadExceptions(); msg('Removed from allow list.')
}

function saveExceptions() { msg('Exceptions saved.', 'success') }
window.addToBlocklist = addToBlocklist
window.addToAllowlist = addToAllowlist
window.saveExceptions = saveExceptions

document.getElementById('blocklist-input').addEventListener('keydown', e => { if (e.key === 'Enter') addToBlocklist() })
document.getElementById('allowlist-input').addEventListener('keydown', e => { if (e.key === 'Enter') addToAllowlist() })

// ── URL Keywords ──────────────────────────────────────────────────────────────
async function loadKeywords() {
  const data = await go().GetKeywords()
  document.getElementById('kw-builtin-count').textContent = data.builtInCount.toLocaleString() + ' entries (always active)'
  const el = document.getElementById('keyword-items')
  if (!data.userAdded?.length) {
    el.innerHTML = '<div class="empty-row">No custom keywords added.</div>'
    return
  }
  el.innerHTML = data.userAdded.map(kw =>
    `<div class="keyword-item">
       <span>🔑 ${escHtml(kw)}</span>
       <button class="remove-btn" data-kw="${escHtml(kw)}">✕</button>
     </div>`
  ).join('')
  el.querySelectorAll('.remove-btn').forEach(btn => {
    btn.addEventListener('click', () => removeKeyword(btn.dataset.kw))
  })
}

async function addKeyword() {
  const input = document.getElementById('keyword-input')
  const val = input.value.trim()
  if (!val) return
  try { await go().AddKeyword(val); input.value = ''; loadKeywords(); msg('Keyword added.') }
  catch (e) { msg(String(e), 'error') }
}
async function removeKeyword(kw) {
  await go().RemoveKeyword(kw); loadKeywords(); msg('Keyword removed.')
}
window.addKeyword = addKeyword
document.getElementById('keyword-input').addEventListener('keydown', e => { if (e.key === 'Enter') addKeyword() })

// ── Password / Security ───────────────────────────────────────────────────────
async function loadPasswordSettings() {
  const adv   = await go().GetAdvancedSettings()
  const proxy = await go().GetProxySettings()
  document.getElementById('setting-delay').value = String(adv.disableDelayHours || 0)
  document.getElementById('setting-port').value  = proxy.proxyPort
}

async function savePassword() {
  const current = document.getElementById('pw-current').value
  const next    = document.getElementById('pw-new').value
  const confirm = document.getElementById('pw-confirm').value
  if (next !== confirm) { msg('Passwords do not match.', 'error'); return }
  try {
    await go().SetPassword(current, next)
    document.getElementById('pw-current').value = ''
    document.getElementById('pw-new').value     = ''
    document.getElementById('pw-confirm').value = ''
    msg(next === '' ? 'Password removed.' : 'Password saved.', 'success')
  } catch (e) { msg(String(e), 'error') }
}
window.savePassword = savePassword

async function saveAdvancedSettings() {
  try {
    const delay = parseInt(document.getElementById('setting-delay').value)
    const adv   = await go().GetAdvancedSettings()
    await go().SaveAdvancedSettings({ ...adv, disableDelayHours: delay })
    msg('Settings saved.', 'success')
  } catch (e) { msg(String(e), 'error') }
}
window.saveAdvancedSettings = saveAdvancedSettings

async function saveProxySettings() {
  const port = parseInt(document.getElementById('setting-port').value)
  try {
    await go().SaveProxySettings({ proxyPort: port, autoStart: true })
    msg('Proxy settings saved.', 'success')
  } catch (e) { msg(String(e), 'error') }
}
window.saveProxySettings = saveProxySettings

// ── Uninstall ─────────────────────────────────────────────────────────────────
async function showUninstall() {
  const hasPw = await go().HasPassword()
  const pw = hasPw ? prompt('Enter password to uninstall K9:') : ''
  if (pw === null) return
  try {
    await go().Uninstall(pw || '')
    msg('K9 uninstalled. Please remove the app from /Applications.', 'error')
  } catch (e) { msg(String(e), 'error') }
}
window.showUninstall = showUninstall

// ── Utility ───────────────────────────────────────────────────────────────────
function escHtml(s) {
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')
}

// ── Init ──────────────────────────────────────────────────────────────────────
window.addEventListener('load', () => {
  const init = () => {
    if (window.go?.main?.App) {
      loadDashboard()
      loadCategories()
    } else {
      setTimeout(init, 100)
    }
  }
  init()
})

setInterval(() => {
  const homePage = document.getElementById('page-home')
  if (homePage?.classList.contains('active') && window.go?.main?.App) loadDashboard()
}, 10000)
