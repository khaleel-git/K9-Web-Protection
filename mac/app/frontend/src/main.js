// K9 Web Protection — Original UI wired to Go backend

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
    sidebar.style.display = 'block'
    sidebarSetup.style.display   = 'none'
    sidebarReports.style.display = 'block'
    clearSideLinks()
    document.querySelector('#item-summary a').classList.add('selected')
    showPage('reports')
    loadActivity()
  } else if (tabName === 'setup') {
    sidebar.style.display = 'block'
    sidebarSetup.style.display   = 'block'
    sidebarReports.style.display = 'none'
    showPage('categories')
    setSideActive('categories')
    loadCategories()
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
    if (page === 'password')   loadPasswordSettings()
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
  const s = await go().GetStatus()

  const l1ok = s.layer1Active
  const l2ok = s.proxyRunning
  const active = l1ok && l2ok

  document.getElementById('status-layer1').innerHTML = l1ok
    ? '<span style="color:green">&#x25CF;</span> Active'
    : '<span style="color:red">&#x25CF;</span> Inactive'
  document.getElementById('status-layer2').innerHTML = l2ok
    ? '<span style="color:green">&#x25CF;</span> Running'
    : '<span style="color:red">&#x25CF;</span> Stopped'
  document.getElementById('stat-today').textContent = s.blockedToday.toLocaleString()
  document.getElementById('stat-total').textContent = s.totalBlocked.toLocaleString()
  document.getElementById('stat-db').textContent =
    `${s.dbDomains.toLocaleString()} domains, ${s.dbUrls.toLocaleString()} URL patterns, ${s.dbKeywords.toLocaleString()} keywords`

  document.getElementById('btn-enable').style.display  = active ? 'none' : 'inline'
  document.getElementById('btn-disable').style.display = active ? 'inline' : 'none'

  const dot = document.getElementById('k9StatusDot')
  if (active) { dot.textContent = 'Active';   dot.className = 'active' }
  else        { dot.textContent = 'Inactive'; dot.className = '' }
}

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

  const tbody = document.getElementById('activity-rows')
  if (!s.topBlocked?.length) {
    tbody.innerHTML = '<tr><td colspan="2" style="padding:6px; color:#888; font-style:italic">No activity recorded yet.</td></tr>'
    return
  }
  let alt = false
  tbody.innerHTML = s.topBlocked
    .sort((a, b) => b.count - a.count)
    .map(e => {
      const row = `<tr ${alt ? 'style="background:#e8eeff"' : ''}>
        <td style="padding:4px 8px; color:#cc2222">&#x29B8; ${esc(e.domain)}</td>
        <td style="text-align:right; padding:4px 8px">${e.count}</td>
      </tr>`
      alt = !alt; return row
    }).join('')
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
  document.getElementById('custom-cats').style.display = checked === 'custom' ? 'block' : 'none'
  document.getElementById('cat-list-default').style.display = checked === 'default' ? 'block' : 'none'
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
    await go().SaveContentSettings({ blockAdultContent, blockYouTube, safeSearch, blockImageSearch })
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
window.addKeyword = addKeyword
window.removeKeyword = removeKeyword
document.getElementById('kw-input').addEventListener('keydown', e => { if (e.key === 'Enter') addKeyword() })

// ── Safe Search ───────────────────────────────────────────────────────────────
async function saveSafeSearch() {
  const on = document.getElementById('cb-safesearch').checked
  try {
    const s = await go().GetContentSettings()
    await go().SaveContentSettings({ ...s, safeSearch: on })
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
    await go().SaveAdvancedSettings({ ...adv, disableDelayHours: delay })
    notify('Settings saved.', 'ok')
  } catch (e) { notify(String(e), 'err') }
}
window.saveAdvancedSettings = saveAdvancedSettings

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
