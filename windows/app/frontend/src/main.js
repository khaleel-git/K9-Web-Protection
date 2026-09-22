// K10 Web Protection — Professional UI wired to Go backend
import { EventsOn } from '../wailsjs/runtime/runtime.js'
import { initI18n, t, applyI18n, getLang, setLang } from './i18n/index.js'

const go = () => window.go?.main?.App

// Resolves once window.go is attached.
function whenBackendReady(timeoutMs = 10000) {
  return new Promise((resolve, reject) => {
    if (go()) { resolve(go()); return }
    const started = Date.now()
    const poll = () => {
      if (go()) { resolve(go()); return }
      if (Date.now() - started > timeoutMs) { reject(new Error('backend unavailable')); return }
      setTimeout(poll, 100)
    }
    poll()
  })
}

const loc = () => (getLang() === 'he' ? 'he-IL' : 'en-US')

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
  } else if (tabName === 'focusmode') {
    sidebar.style.display = 'none'
    showPage('focusmode')
    loadFocusMode()
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
    if (page === 'time')       loadTimeRestrictions()
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
  const [s, rules, cs] = await Promise.all([
    go().GetStatus(),
    go().GetRules(),
    go().GetContentSettings(),
  ])

  const l1ok   = s.layer1Active
  const l2ok   = s.proxyRunning
  const active = l2ok // protection is the proxy; hosts is an extra layer with its own warnings

  // ── Header badge ──
  const badge = document.getElementById('k9StatusDot')
  if (active) { badge.textContent = t('nav.status.active');   badge.className = 'status-badge active' }
  else        { badge.textContent = t('nav.status.inactive'); badge.className = 'status-badge' }

  // ── Inactive warning bar ──
  const inactiveBar = document.getElementById('protection-inactive-bar')
  if (inactiveBar) inactiveBar.style.display = active ? 'none' : 'flex'
  const btnEnable  = document.getElementById('btn-enable')
  const btnDisable = document.getElementById('btn-disable')
  if (btnEnable)  btnEnable.style.display  = active ? 'none'       : 'inline-flex'
  if (btnDisable) btnDisable.style.display = active ? 'inline-flex' : 'none'

  // ── Stat cards ──
  document.getElementById('stat-today').textContent = s.blockedToday.toLocaleString(loc())
  document.getElementById('stat-total').textContent = s.totalBlocked.toLocaleString(loc())

  const warnBar = document.getElementById('settings-warning-bar')
  const w = settingsWarnings(s.diagnostics)
  if (warnBar) {
    const lines = [...w.settings, ...w.hosts]
    warnBar.style.display = lines.length ? 'flex' : 'none'
    if (lines.length) lines.push(t(w.settings.length ? 'toast.settings.warning' : 'toast.hosts.warning'))
    document.getElementById('settings-warning-text').textContent = lines.join(' · ')
    const retry = document.getElementById('btn-retry-hosts')
    if (retry) retry.style.display = w.hosts.length && l2ok ? 'inline-flex' : 'none'
  }

  const allowCount = rules?.allow?.length ?? 0
  const blockCount = rules?.block?.length ?? 0
  const excEl = document.getElementById('stat-exceptions')
  if (excEl) excEl.innerHTML =
    `${allowCount} <span style="font-size:9px;color:#888;font-weight:600">${esc(t('dash.exceptions.allow'))}</span>&nbsp;&nbsp;` +
    `${blockCount} <span style="font-size:9px;color:#888;font-weight:600">${esc(t('dash.exceptions.block'))}</span>`

  const fl = cs.filterLevel || 'default'
  const levelEl    = document.getElementById('stat-filter-level')
  const levelSubEl = document.getElementById('stat-filter-sub')
  if (levelEl)    levelEl.textContent    = LEVEL_NAMES.includes(fl) ? t('level.' + fl + '.name') : fl
  if (levelSubEl) levelSubEl.textContent = levelSubText(fl)

  // ── DB chips ──
  const setText = (id, val) => { const el = document.getElementById(id); if (el) el.textContent = val }
  setText('db-domains',  s.dbDomains.toLocaleString(loc()))
  setText('db-urls',     s.dbUrls.toLocaleString(loc()))
  setText('db-keywords', s.dbKeywords.toLocaleString(loc()))

  // ── Protection modules ──
  const setMod = (dotId, valId, on, idle) => {
    const dot = document.getElementById(dotId)
    const val = document.getElementById(valId)
    if (dot) dot.className = 'prot-dot ' + (on ? 'on' : idle ? 'idle' : 'off')
    if (val) {
      val.textContent = on ? t('dash.module.active') : idle ? t('dash.module.notNeeded') : t('dash.module.inactive')
      val.className = 'prot-val ' + (on ? 'active' : idle ? 'idle' : 'inactive')
    }
  }
  setMod('dot-web-protection', 'mod-web-protection', active)
  setMod('dot-malware',        'mod-malware',        l2ok)
  setMod('dot-safesearch',     'mod-safesearch',     cs.safeSearch !== false)
  setMod('dot-https',          'mod-https',          l2ok)
  setMod('dot-dns',            'mod-dns',            l1ok, s.layer1Idle)

  // ── Top blocked categories bar chart (from topBlocked domain data) ──
  renderTopCategoriesChart(s.topBlocked)

  // ── Recent blocked activity (placeholder — real data needs backend work, see plan.md) ──
  renderRecentActivity(s.topBlocked)
}

// ── Categories ────────────────────────────────────────────────────────────────
// Static fallback mirror of internal/proxy/categories.go LevelCategories.
const LEVEL_NAMES = ['high', 'default', 'moderate', 'minimal', 'monitor', 'custom']

const LEVEL_CATEGORIES = {
  high: [
    'pornography', 'adult-mature', 'alternative-sexuality', 'alternative-spirituality',
    'abortion', 'alcohol', 'chat-im', 'extreme', 'gambling', 'hacking', 'illegal-drugs',
    'intimate-apparel', 'lgbt', 'malware-spyware', 'newsgroups-forums', 'nudity',
    'open-image-search', 'p2p', 'personal-pages', 'personals-dating', 'phishing',
    'proxy-avoidance', 'sex-education', 'social-networking', 'suspicious', 'tobacco', 'unrated',
    'violence-hate', 'weapons'
  ],
  default: [
    'pornography', 'adult-mature', 'alternative-sexuality', 'extreme', 'gambling', 'hacking',
    'illegal-drugs', 'intimate-apparel', 'nudity', 'personals-dating', 'phishing',
    'malware-spyware', 'proxy-avoidance', 'sex-education', 'abortion', 'suspicious',
    'violence-hate'
  ],
  moderate: [
    'pornography', 'adult-mature', 'gambling', 'hacking', 'illegal-drugs', 'phishing',
    'malware-spyware', 'extreme', 'violence-hate', 'suspicious'
  ],
  minimal: [
    'pornography', 'phishing', 'malware-spyware'
  ],
  monitor: [],
}

const KNOWN_CATEGORIES = new Set([
  'pornography', 'adult-mature', 'nudity', 'alternative-sexuality', 'sex-education',
  'social-networking', 'chat-im', 'gambling', 'malware-spyware', 'phishing', 'hacking',
  'violence-hate', 'extreme', 'illegal-drugs', 'p2p', 'proxy-avoidance', 'alcohol', 'tobacco',
  'weapons', 'abortion', 'personals-dating', 'intimate-apparel', 'newsgroups-forums',
  'open-image-search', 'personal-pages', 'alternative-spirituality', 'lgbt', 'suspicious',
  'unrated', 'streaming', 'proxy-p2p', 'ads-tracking', 'other'
])

let levelCategories = LEVEL_CATEGORIES
let levelCategoriesPromise = null

// Go randomises map key order; only the arrays are order-stable.
function loadLevelCategories() {
  if (levelCategoriesPromise) return levelCategoriesPromise
  levelCategoriesPromise = whenBackendReady()
    .then(app => app.GetLevelCategories())
    .then(map => {
      if (map && typeof map === 'object') levelCategories = map
      return levelCategories
    })
    .catch(err => {
      console.warn('[i18n] GetLevelCategories unavailable — using the static fallback, which may have drifted:', err)
      return levelCategories
    })
  return levelCategoriesPromise
}

function categoriesForLevel(level) {
  const list = levelCategories[level]
  if (Array.isArray(list)) return list
  console.warn('[i18n] level "' + level + '" missing from GetLevelCategories — using the static fallback')
  return LEVEL_CATEGORIES[level] || []
}

function levelSubText(level) {
  if (level === 'high')    return t('level.high.sub')
  if (level === 'monitor') return t('level.monitor.sub')
  if (level === 'custom')  return t('level.custom.sub')
  if (level === 'minimal') return t('level.minimal.sub')
  const list = categoriesForLevel(level)
  if (!list.length) return ''
  return t('level.sub.count', { count: list.length })
}

// Driven by the DOM's fixed box order, never by the response's key order.
async function renderLevelCategoryLists() {
  await loadLevelCategories()
  document.querySelectorAll('.cat-cols[data-level]').forEach(box => {
    box.innerHTML = categoriesForLevel(box.dataset.level)
      .map(id => `<span class="cat-item">${esc(t('cat.' + id))}</span>`)
      .join('\n')
  })
  const catPage = document.getElementById('page-categories')
  if (catPage) applyI18n(catPage)
}

function domainToCategory(domain) {
  const d = domain.toLowerCase()
  if (/facebook|instagram|twitter|x\.com|tiktok|snapchat|reddit|pinterest|tumblr|linkedin|threads|bereal|vk\.com|weibo/.test(d)) return 'social-networking'
  if (/whatsapp|telegram|discord|signal|messenger|viber|wechat|line\.me|skype|slack/.test(d)) return 'chat-im'
  if (/youtube|twitch|netflix|hulu|disneyplus|tubi|dailymotion|vimeo|spotify|soundcloud/.test(d)) return 'streaming'
  if (/pornhub|xhamster|xnxx|xvideos|onlyfans|porn|xxx|adult|nudity|phncdn|brazzers|redtube|youporn|sex\.com/.test(d)) return 'pornography'
  if (/casino|poker|slots|betway|bet365|draftkings|fanduel|gambl|bwin|1xbet|betfair/.test(d)) return 'gambling'
  if (/malware|trojan|spyware|adware|ransomware|botnet|exploit|payload/.test(d)) return 'malware-spyware'
  if (/phish|scam|fraud|fake|spoof/.test(d)) return 'phishing'
  if (/hate|terror|jihadist|extremis|violen/.test(d)) return 'violence-hate'
  if (/drug|weed|cannabis|cocaine|heroin|narco/.test(d)) return 'illegal-drugs'
  if (/proxy|vpn|tor\.|torproject|pirate|thepirate|1337x|rarbg|torrent|magnet/.test(d)) return 'proxy-p2p'
  if (/doubleclick|adnxs|googlesyndication|outbrain|taboola|ads\.|tracking\.|analytics\./.test(d)) return 'ads-tracking'
  return 'other'
}

function entryCategory(e) {
  if (e.category && KNOWN_CATEGORIES.has(e.category)) return e.category
  return domainToCategory(e.domain)
}

function renderTopCategoriesChart(topBlocked) {
  const el = document.getElementById('top-categories-bars')
  if (!el) return
  if (!topBlocked?.length) {
    el.innerHTML = `<div class="dash-bar-row"><span class="dash-bar-label" style="width:auto;color:#aaa">${esc(t('activity.empty.no-data'))}</span></div>`
    return
  }
  // Aggregate domain block counts into categories
  const catCounts = {}
  for (const e of topBlocked) {
    const cat = entryCategory(e)
    catCounts[cat] = (catCounts[cat] || 0) + e.count
  }
  const sorted = Object.entries(catCounts).sort((a, b) => b[1] - a[1]).slice(0, 5)
  const max = sorted[0]?.[1] || 1
  const colors = ['#991b1b', '#1e40af', '#92400e', '#6b21a8', '#854d0e']
  el.innerHTML = sorted.map(([cat, count], i) => {
    const pct = Math.round(count / max * 100)
    return `<div class="dash-bar-row">
      <span class="dash-bar-label">${esc(t('cat.' + cat))}</span>
      <div class="dash-bar-track"><div class="dash-bar-fill" style="width:${pct}%;background:${colors[i]}"></div></div>
      <span class="dash-bar-val">${count}</span>
    </div>`
  }).join('')
}

function fmtTime(iso) {
  if (!iso) return '—:—'
  const d = new Date(iso)
  if (isNaN(d)) return '—:—'
  return d.toLocaleTimeString(loc(), { hour: '2-digit', minute: '2-digit', hour12: false })
}

// Identifier -> CSS class for recent-activity tinting.
const CAT_CLASS_SETS = [
  ['cat-porn', new Set(['pornography', 'adult-mature', 'nudity', 'alternative-sexuality', 'sex-education'])],
  ['cat-gamble', new Set(['gambling'])],
  ['cat-malware', new Set(['malware-spyware', 'suspicious', 'hacking'])],
  ['cat-phish', new Set(['phishing'])],
  ['cat-violence', new Set(['violence-hate', 'extreme'])],
  ['cat-social', new Set(['social-networking', 'chat-im'])],
]

function renderRecentActivity(topBlocked) {
  const el = document.getElementById('recent-activity')
  if (!el) return
  if (!topBlocked?.length) {
    el.innerHTML = `<div class="dash-recent-row"><span class="dash-recent-url muted">${esc(t('activity.empty.none-recorded'))}</span></div>`
    return
  }
  const catClass = (e) => {
    const cat = entryCategory(e)
    for (const [cls, ids] of CAT_CLASS_SETS) {
      if (ids.has(cat)) return cls
    }
    return 'cat-other'
  }
  el.innerHTML = topBlocked.slice(0, 8).map(e =>
    `<div class="dash-recent-row">
      <span class="dash-recent-time ltr-text">${fmtTime(e.lastSeen)}</span>
      <span class="dash-recent-url ltr-text">${esc(e.domain)}</span>
      <span class="dash-recent-cat ${catClass(e)}">${esc(t('cat.' + entryCategory(e)))}</span>
    </div>`
  ).join('')
}

async function clearBlockedLog() {
  try {
    await go().ClearStats()
    loadDashboard()
  } catch (e) { notify(errText(e), 'err') }
}
window.clearBlockedLog = clearBlockedLog

function verifyProtection() {
  if (!window.go?.main?.App) return
  go().GetStatus().then(s => {
    const ok = s.proxyRunning && (s.layer1Active || s.layer1Idle)
    notify(ok ? t('toast.verify.ok') : t('toast.verify.fail'), ok ? 'ok' : 'err')
  })
}
window.verifyProtection = verifyProtection

async function enableProtection() {
  try { await go().EnableProtection(); notify(t('toast.protection.enabled'), 'ok') }
  catch (e) { notify(errText(e), 'err') }
  finally { loadDashboard() }
}
window.enableProtection = enableProtection

async function retryHosts() {
  const btn = document.getElementById('btn-retry-hosts')
  if (btn) btn.disabled = true
  try {
    const ap = await go().RetryHosts()
    const w = applyWarnings(ap)
    if (w.length) notify(errText(w.join(' · ')), 'err')
    else notify(t('toast.hosts.updated'), 'ok')
  } catch (e) { notify(errText(e), 'err') }
  finally { if (btn) btn.disabled = false; loadDashboard() }
}
window.retryHosts = retryHosts

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
    closeModal(); notify(t('toast.protection.disabled'), 'ok')
  } catch (e) {
    notify(errText(e), 'err')
    const s = await go().GetStatus().catch(() => null)
    if (s && !s.proxyRunning) closeModal() // turned off, only the cleanup failed
  } finally { loadDashboard() }
}
window.confirmDisable = confirmDisable

// ── Activity ──────────────────────────────────────────────────────────────────
async function loadActivity() {
  const s = await go().GetStatus()
  document.getElementById('gen-total').textContent = s.totalBlocked.toLocaleString(loc())
  document.getElementById('gen-today').textContent = s.blockedToday.toLocaleString(loc())

  const container = document.getElementById('activity-rows')
  if (!s.topBlocked?.length) {
    container.innerHTML = `<div class="act-row"><span class="muted">${esc(t('activity.empty.none-recorded'))}</span></div>`
    return
  }
  container.innerHTML = s.topBlocked
    .sort((a, b) => b.count - a.count)
    .map(e => `<div class="act-row">
      <span style="color:var(--red)">&#x29B8; <span class="ltr-text">${esc(e.domain)}</span></span>
      <span>${e.count}</span>
    </div>`)
    .join('')
}

async function loadCategories() {
  const s = await go().GetContentSettings()
  let level = s.filterLevel || 'default'

  document.querySelectorAll('[id^="setting-"]').forEach(el => el.classList.remove('selected'))
  const row = document.getElementById('setting-' + level)
  if (row) row.classList.add('selected')

  const radio = document.getElementById('level-' + level)
  if (radio) radio.checked = true

  document.getElementById('cat-adult').checked      = s.blockAdultContent !== false
  document.getElementById('cat-youtube').checked    = s.blockYouTube === true
  document.getElementById('cat-safesearch').checked = s.safeSearch !== false

  updateCategoryUI()
  renderLevelCategoryLists()
}

function updateCategoryUI() {
  const checked = document.querySelector('input[name="level"]:checked')?.value || 'default'
  document.querySelectorAll('[id^="setting-"]').forEach(el => el.classList.remove('selected'))
  const row = document.getElementById('setting-' + checked)
  if (row) row.classList.add('selected')
  const showHide = (id, show) => { const el = document.getElementById(id); if (el) el.style.display = show ? 'block' : 'none' }
  showHide('cat-list-high',     checked === 'high')
  showHide('cat-list-default',  checked === 'default')
  showHide('cat-list-moderate', checked === 'moderate')
  showHide('cat-list-minimal',  checked === 'minimal')
  showHide('custom-cats',       checked === 'custom')
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

  // Standard levels — no password needed
  if (['high', 'default', 'moderate', 'minimal'].includes(level)) {
    try {
      await go().SetFilterLevel(level)
      notify(t('toast.categories.saved'), 'ok')
      loadDashboard()
    } catch (e) { notify(errText(e), 'err') }
    return
  }

  // Monitor / Custom — password required
  let blockAdultContent = false, blockYouTube = false, safeSearch = true, blockImageSearch = false
  if (level === 'custom') {
    blockAdultContent = document.getElementById('cat-adult').checked
    blockYouTube      = document.getElementById('cat-youtube').checked
    safeSearch        = document.getElementById('cat-safesearch').checked
  }
  const hasPw = await go().HasPassword()
  const pw = hasPw ? await requirePassword(t('modal.require.save-categories')) : ''
  if (pw === null) return
  try {
    await go().SaveContentSettings(pw, { filterLevel: level, blockAdultContent, blockYouTube, safeSearch, blockImageSearch })
    notify(t('toast.categories.saved'), 'ok')
    loadDashboard()
  } catch (e) { notify(errText(e), 'err') }
}
window.saveCategories = saveCategories

// ── Exceptions ────────────────────────────────────────────────────────────────
async function loadExceptions() {
  renderRules(await go().GetRules())
}

function renderRules(v) {
  renderRuleList('blocklist-items', v?.block, 'block', v?.display)
  renderRuleList('allowlist-items', v?.allow, 'allow', v?.display)
  const el = document.getElementById('exc-apply-status')
  if (!el) return
  const lines = [v?.notice, ...applyWarnings(v?.apply), v?.apply?.hostsInfo].filter(Boolean)
  if (v?.apply?.hostsPartial) lines.push(t('content.exceptions.hosts-partial'))
  el.style.display = lines.length ? 'block' : 'none'
  el.innerHTML = lines.map(l => `<div dir="auto">${esc(l)}</div>`).join('')
}

function renderRuleList(id, rules, kind, display) {
  const el = document.getElementById(id)
  if (!rules?.length) {
    el.innerHTML = `<span style="color:#888; font-style:italic">${esc(t('content.exceptions.empty'))}</span>`
    return
  }
  el.innerHTML = rules.map(r =>
    `<div class="exc-item" style="font-size:12px; padding:2px 0; display:flex; align-items:center; gap:6px">
       <span style="color:${kind === 'block' ? '#cc2222' : '#228B22'}; font-weight:bold">&#x29B8;</span>
       <bdi class="exc-domain" style="color:#003E7E">${esc(display?.[r.domain] || r.domain)}</bdi>
       ${r.includeSubdomains ? `<span class="exc-badge">${esc(t('content.exceptions.badge-subdomains'))}</span>` : ''}
       <a href="#" class="exc-remove" data-kind="${kind}" data-domain="${esc(r.domain)}" title="${esc(t('content.exceptions.remove'))}"
          style="color:#cc2222; font-weight:bold; font-size:14px; text-decoration:none; margin-inline-start:auto">&times;</a>
     </div>`
  ).join('')
}

function applyWarnings(ap) {
  return [ap?.configError, ap?.hostsError].filter(Boolean)
}

// The raw load error stays in Diagnostics; the bar shows a translated line.
function settingsWarnings(d) {
  const ap = d?.lastApply || {}
  return {
    settings: [d?.loadError && t('dash.warn.load-error'), ap.configError].filter(Boolean),
    hosts: [ap.hostsError].filter(Boolean),
  }
}

// Success only when every layer applied the change.
function notifyApplied(v, okKey) {
  const w = [v?.notice, ...applyWarnings(v?.apply)].filter(Boolean)
  if (w.length) notify(errText(w.join(' · ')), 'err')
  else notify([t(okKey), v?.apply?.hostsInfo].filter(Boolean).join(' '), 'ok')
}

async function askPassword(titleKey) {
  return (await go().HasPassword()) ? requirePassword(t(titleKey)) : ''
}

async function addToBlocklist() {
  const input = document.getElementById('listTb-0')
  const val = input.value.trim()
  if (!val) return
  const sub = document.getElementById('listSub-0')?.checked === true
  try {
    const v = await go().AddBlockRule(val, sub)
    input.value = ''
    document.getElementById('listSub-0').checked = false
    renderRules(v); notifyApplied(v, 'toast.blocklist.added')
  } catch (e) { notify(errText(e), 'err') }
}
async function removeBlockRule(domain) {
  const pw = await askPassword('modal.require.remove-block')
  if (pw === null) return
  try { const v = await go().RemoveBlockRule(pw, domain); renderRules(v); notifyApplied(v, 'toast.entry.removed') }
  catch (e) { notify(errText(e), 'err') }
}
async function addToAllowlist() {
  const input = document.getElementById('listTb-1')
  const val = input.value.trim()
  if (!val) return
  const sub = document.getElementById('listSub-1')?.checked === true
  const pw = await askPassword('modal.require.add-allow')
  if (pw === null) return
  try {
    const v = await go().AddAllowRule(pw, val, sub)
    input.value = ''
    document.getElementById('listSub-1').checked = false
    renderRules(v); notifyApplied(v, 'toast.allowlist.added')
  } catch (e) { notify(errText(e), 'err') }
}
async function removeAllowRule(domain) {
  try { const v = await go().RemoveAllowRule(domain); renderRules(v); notifyApplied(v, 'toast.entry.removed') }
  catch (e) { notify(errText(e), 'err') }
}
window.addToBlocklist = addToBlocklist
window.addToAllowlist = addToAllowlist

document.getElementById('page-exceptions')?.addEventListener('click', e => {
  const a = e.target.closest?.('.exc-remove')
  if (!a) return
  e.preventDefault()
  if (a.dataset.kind === 'block') removeBlockRule(a.dataset.domain)
  else removeAllowRule(a.dataset.domain)
})
document.getElementById('listTb-0')?.addEventListener('keydown', e => { if (e.key === 'Enter') addToBlocklist() })
document.getElementById('listTb-1')?.addEventListener('keydown', e => { if (e.key === 'Enter') addToAllowlist() })

// ── Keywords ──────────────────────────────────────────────────────────────────
async function loadKeywords() {
  const data = await go().GetKeywords()
  document.getElementById('kw-builtin-desc').textContent =
    t('content.keywords.builtin', { count: data.builtInCount.toLocaleString(loc()) })
  const el = document.getElementById('kw-items')
  if (!data.userAdded?.length) {
    el.innerHTML = `<span style="color:#888; font-style:italic">${esc(t('content.keywords.empty'))}</span>`
    return
  }
  el.innerHTML = data.userAdded.map(kw =>
    `<div style="font-size:12px; padding:2px 0; display:flex; align-items:center; gap:6px">
       <span style="color:#cc6600; font-weight:bold">${esc(t('common.glyph.list-bullet'))}</span>
       <span class="ltr-text">${esc(kw)}</span>
       <a href="#" style="color:#cc2222; font-weight:bold; font-size:14px; text-decoration:none; margin-inline-start:4px"
          onclick="removeKeyword('${esc(kw).replace(/'/g,"\\'")}'); return false;">&times;</a>
     </div>`
  ).join('')
}

async function addKeyword() {
  const input = document.getElementById('kw-input')
  const val = input.value.trim()
  if (!val) return
  try { await go().AddKeyword(val); input.value = ''; loadKeywords(); notify(t('toast.keyword.added')) }
  catch (e) { notify(errText(e), 'err') }
}
async function removeKeyword(kw) {
  await go().RemoveKeyword(kw); loadKeywords(); notify(t('toast.keyword.removed'))
}
async function saveKeywords() { notify(t('toast.keywords.auto-saved'), 'ok') }
window.addKeyword = addKeyword
window.removeKeyword = removeKeyword
window.saveKeywords = saveKeywords
document.getElementById('kw-input')?.addEventListener('keydown', e => { if (e.key === 'Enter') addKeyword() })

// ── Safe Search ───────────────────────────────────────────────────────────────
async function loadSafeSearch() {
  const s = await go().GetContentSettings()
  const cb = document.getElementById('cb-safesearch')
  if (cb) cb.checked = s.safeSearch !== false
}

async function saveSafeSearch() {
  const on = document.getElementById('cb-safesearch').checked
  const pw = on ? '' : await askPassword('modal.require.safesearch-off')
  if (pw === null) { loadSafeSearch(); return }
  try {
    await go().SetSafeSearch(pw, on)
    notify(t('toast.safesearch.saved'), 'ok')
  } catch (e) { notify(errText(e), 'err'); loadSafeSearch() }
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
  if (next !== confirm) { notify(t('toast.password.mismatch'), 'err'); return }
  try {
    await go().SetPassword(current, next)
    document.getElementById('pw-current').value = ''
    document.getElementById('pw-new').value     = ''
    document.getElementById('pw-confirm').value = ''
    notify(next ? t('toast.password.saved') : t('toast.password.removed'), 'ok')
  } catch (e) { notify(errText(e), 'err') }
}
async function removePassword() {
  const current = document.getElementById('pw-current').value
  try { await go().SetPassword(current, ''); notify(t('toast.password.removed'), 'ok') }
  catch (e) { notify(errText(e), 'err') }
}
window.savePassword = savePassword
window.removePassword = removePassword

async function saveAdvancedSettings() {
  const delay = parseInt(document.getElementById('setting-delay').value)
  try {
    const adv = await go().GetAdvancedSettings()
    await go().SaveAdvancedSettings('', { ...adv, disableDelayHours: delay })
    notify(t('toast.settings.saved'), 'ok')
  } catch (e) { notify(errText(e), 'err') }
}
window.saveAdvancedSettings = saveAdvancedSettings

// ── Language ──────────────────────────────────────────────────────────────────
// Go is the source of truth; localStorage is only a pre-paint hint.

function syncLanguageSelect() {
  const sel = document.getElementById('setting-language')
  if (sel) sel.value = getLang()
}

// Guard against a reload cycle if the backend keeps disagreeing with what setLang persisted.
const LANG_RELOAD_KEY = 'k10.langReload'

async function reconcileLanguage() {
  let app, lang
  try {
    app = await whenBackendReady()
    lang = await app.GetLanguage()
  } catch (e) {
    console.warn('[i18n] GetLanguage unavailable — keeping the locally hinted language:', e)
    return
  }
  // Empty means the user has never chosen: persist what the environment resolved to, no reload needed.
  if (!lang) {
    try { await app.SetLanguage(getLang()) }
    catch (e) { console.warn('[i18n] could not persist the detected language:', e) }
    try { sessionStorage.removeItem(LANG_RELOAD_KEY) } catch (e) { /* storage unavailable */ }
    syncLanguageSelect()
    return
  }
  if (lang === getLang()) {
    try { sessionStorage.removeItem(LANG_RELOAD_KEY) } catch (e) { /* storage unavailable */ }
    syncLanguageSelect()
    return
  }
  // Reload rather than patch: the second load rebuilds imperative text too, which applyI18n cannot.
  let reloaded = false
  try { reloaded = sessionStorage.getItem(LANG_RELOAD_KEY) === lang } catch (e) { /* storage unavailable */ }
  if (reloaded) {
    console.warn('[i18n] backend still reports "' + lang + '" after a reload — not reloading again')
    syncLanguageSelect()
    return
  }
  try { sessionStorage.setItem(LANG_RELOAD_KEY, lang) } catch (e) { /* storage unavailable */ }
  setLang(lang)
  window.location.reload()
}

// Persist to Go first, then reload so every rendered list is rebuilt.
async function changeLanguage(lang) {
  if (!lang || lang === getLang()) return
  try {
    await whenBackendReady()
  } catch (e) {
    notify(t('toast.backend.missing'), 'err')
    syncLanguageSelect()
    return
  }
  try {
    await go().SetLanguage(lang)
  } catch (e) {
    notify(errText(e), 'err')
    syncLanguageSelect()
    return
  }
  setLang(lang)
  window.location.reload()
}
window.changeLanguage = changeLanguage

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
    notify(t('toast.advanced.saved'), 'ok')
  } catch (e) { notify(errText(e), 'err') }
}
async function installCA() {
  const btn = document.getElementById('btn-install-ca')
  const status = document.getElementById('ca-status')
  btn.disabled = true
  status.textContent = t('advanced.ca.installing')
  try {
    await go().InstallCACert()
    status.style.color = '#1a8a3a'
    status.textContent = t('advanced.ca.installed')
  } catch (e) {
    status.style.color = '#cc3333'
    status.textContent = errText(e)
  }
  btn.disabled = false
}

window.loadAdvanced = loadAdvanced
window.saveAdvanced = saveAdvanced
window.installCA = installCA

// ── Focus Mode ────────────────────────────────────────────────────────────────

let _focusEndTime = 0
let _focusCountdown = null

async function loadFocusMode() {
  if (!window.go?.main?.App) return
  const [fm, sites] = await Promise.all([go().GetFocusMode(), go().GetFocusSites()])
  renderFocusModeStatus(fm)
  renderFocusSitesList(sites)
  if (fm.active) {
    _focusEndTime = Date.now() + fm.remaining * 1000
    clearInterval(_focusCountdown)
    _focusCountdown = setInterval(_tickFocusCountdown, 1000)
  } else {
    clearInterval(_focusCountdown)
  }
}

function _tickFocusCountdown() {
  const rem = Math.max(0, Math.round((_focusEndTime - Date.now()) / 1000))
  if (rem <= 0) {
    clearInterval(_focusCountdown)
    renderFocusModeStatus({ active: false, remaining: 0 })
  } else {
    _updateFocusStatusUI(true, rem)
  }
}

function renderFocusModeStatus(fm) {
  _updateFocusStatusUI(fm.active, fm.remaining)
}

function _updateFocusStatusUI(active, remaining) {
  const dot   = document.getElementById('focus-status-dot')
  const text  = document.getElementById('focus-status-text')
  const stop  = document.getElementById('focus-stop-btn')
  const start = document.getElementById('focus-start-section')
  if (!dot) return

  if (active) {
    const m = Math.floor(remaining / 60), s = remaining % 60
    const timeStr = m > 0
      ? t('apps.time.minutes-seconds', { m, s: s.toString().padStart(2, '0') })
      : t('apps.time.seconds', { s })
    dot.style.background  = '#1a8a3a'
    text.textContent      = t('apps.status.active', { time: timeStr })
    text.style.color      = '#1a8a3a'
    if (stop)  stop.style.display  = 'inline-flex'
    if (start) start.style.opacity = '0.4'
  } else {
    dot.style.background  = '#aaa'
    text.textContent      = t('apps.status.inactive')
    text.style.color      = '#555'
    if (stop)  stop.style.display  = 'none'
    if (start) start.style.opacity = '1'
  }
}

function renderFocusSitesList(sites) {
  const el = document.getElementById('focus-sites-list')
  if (!el) return
  if (!sites?.length) {
    el.innerHTML = `<div style="padding:8px;color:#aaa;font-style:italic;font-size:11px">${esc(t('apps.sites.empty'))}</div>`
    return
  }
  el.innerHTML = sites.map(s => {
    const checked = s.active ? 'checked' : ''
    const del = !s.builtin
      ? `<a href="#" style="color:#cc2222;font-size:14px;font-weight:bold;text-decoration:none;margin-inline-start:auto;padding:0 8px;flex-shrink:0"
           onclick="removeFocusSite('${esc(s.domain).replace(/'/g,"\\'")}'); return false;">&times;</a>`
      : '<span style="width:28px;flex-shrink:0"></span>'
    return `<div style="display:flex;align-items:center;gap:8px;padding:5px 10px;border-bottom:1px solid #e8ecf2">
      <input type="checkbox" ${checked} style="flex-shrink:0;cursor:pointer"
        onchange="toggleFocusSite('${esc(s.domain).replace(/'/g,"\\'")}', this.checked)">
      <span class="ltr-text" style="font-size:11px;font-weight:600;color:${s.active ? 'var(--navy)' : '#999'};flex:1">${esc(s.domain)}</span>
      <span style="font-size:9px;font-weight:700;padding:1px 6px;border-radius:2px;background:${s.active ? '#e6f4eb' : '#f5f5f5'};color:${s.active ? '#1a8a3a' : '#aaa'};flex-shrink:0">${esc(s.active ? t('apps.site.block') : t('apps.site.allow'))}</span>
      ${del}
    </div>`
  }).join('')
}

async function startFocusMode() {
  const minutes = parseInt(document.getElementById('focus-duration')?.value || '30')
  try {
    await go().StartFocusMode(minutes)
    notify(minutes < 60
      ? t('toast.focus.started-min', { count: minutes })
      : t('toast.focus.started-hr',  { count: minutes / 60 }), 'ok')
    await loadFocusMode()
  } catch (e) { notify(errText(e), 'err') }
}

async function stopFocusMode() {
  const hasPw = await go().HasPassword()
  const pw = hasPw ? await requirePassword(t('modal.require.stop-focus')) : ''
  if (pw === null) return
  try {
    await go().StopFocusMode(pw)
    clearInterval(_focusCountdown)
    notify(t('toast.focus.stopped'), 'ok')
    await loadFocusMode()
  } catch (e) { notify(errText(e), 'err') }
}

async function toggleFocusSite(domain, active) {
  try {
    await go().SetFocusSiteActive(domain, active)
    const sites = await go().GetFocusSites()
    renderFocusSitesList(sites)
  } catch (e) { notify(errText(e), 'err') }
}

async function addFocusSite() {
  const input = document.getElementById('focus-add-input')
  const val = input.value.trim()
  if (!val) return
  try {
    await go().AddFocusSite(val)
    input.value = ''
    const sites = await go().GetFocusSites()
    renderFocusSitesList(sites)
    notify(t('toast.focus.site-added'), 'ok')
  } catch (e) { notify(errText(e), 'err') }
}

async function removeFocusSite(domain) {
  try {
    await go().RemoveFocusSite(domain)
    const sites = await go().GetFocusSites()
    renderFocusSitesList(sites)
    notify(t('toast.focus.site-removed'), 'ok')
  } catch (e) { notify(errText(e), 'err') }
}

window.startFocusMode  = startFocusMode
window.stopFocusMode   = stopFocusMode
window.toggleFocusSite = toggleFocusSite
window.addFocusSite    = addFocusSite
window.removeFocusSite = removeFocusSite

document.getElementById('focus-add-input')?.addEventListener('keydown', e => {
  if (e.key === 'Enter') addFocusSite()
})

// ── Time Restrictions ─────────────────────────────────────────────────────────
function toggleTimeRestrictions() {
  const enabled = document.getElementById('cb-time-enabled')?.checked
  const showHide = (id, show) => { const el = document.getElementById(id); if (el) el.style.display = show ? 'block' : 'none' }
  showHide('time-schedule', enabled)
  showHide('time-disabled-msg', !enabled)
}
window.toggleTimeRestrictions = toggleTimeRestrictions

async function loadTimeRestrictions() {
  if (!window.go?.main?.App) return
  try {
    const tr = await go().GetTimeRestrictions()
    const cbEnabled = document.getElementById('cb-time-enabled')
    if (cbEnabled) cbEnabled.checked = tr.enabled
    toggleTimeRestrictions()
    const days = tr.days || []
    days.forEach(d => {
      const row = document.querySelector(`.time-day-row[data-day="${d.day}"]`)
      if (!row) return
      const fromEl = row.querySelector('.tr-from')
      const toEl   = row.querySelector('.tr-to')
      const cbEl   = row.querySelector('.tr-enabled')
      if (fromEl) fromEl.value = d.from || '08:00'
      if (toEl)   toEl.value   = d.to   || '22:00'
      if (cbEl)   cbEl.checked = !!d.enabled
    })
    const timePage = document.getElementById('page-time')
    if (timePage) applyI18n(timePage)
  } catch (e) { notify(errText(e), 'err') }
}
window.loadTimeRestrictions = loadTimeRestrictions

async function saveTimeRestrictions() {
  if (!window.go?.main?.App) { notify(t('toast.backend.missing'), 'err'); return }
  try {
    const enabled = !!document.getElementById('cb-time-enabled')?.checked
    const days = []
    document.querySelectorAll('.time-day-row[data-day]').forEach(row => {
      const day    = row.dataset.day
      const from   = row.querySelector('.tr-from')?.value  || '08:00'
      const to     = row.querySelector('.tr-to')?.value    || '22:00'
      const enCb   = row.querySelector('.tr-enabled')
      const dayEnabled = enCb ? enCb.checked : false
      days.push({ day, from, to, enabled: dayEnabled })
    })
    await go().SaveTimeRestrictions({ enabled, days })
    notify(t('toast.time.saved'))
  } catch (e) { notify(errText(e), 'err') }
}
window.saveTimeRestrictions = saveTimeRestrictions

// ── Blocking Effects (frontend-only until backend is built) ───────────────────
function loadBlockingEffects() {
  const msgEl = document.getElementById('eff-custom-msg')
  if (msgEl) msgEl.value = ''
}
async function saveBlockingEffects() {
  const msg = document.getElementById('eff-custom-msg')?.value?.trim()
  if (!window.go?.main?.App) { notify(t('toast.effects.saved-stub'), 'ok'); return }
  try {
    const adv = await go().GetAdvancedSettings()
    await go().SaveAdvancedSettings('', { ...adv, blockedMessage: msg || adv.blockedMessage })
    notify(t('toast.effects.saved'), 'ok')
  } catch (e) { notify(errText(e), 'err') }
}
window.loadBlockingEffects = loadBlockingEffects
window.saveBlockingEffects = saveBlockingEffects

// ── K10 Update ────────────────────────────────────────────────────────────────
async function loadUpdate() {
  if (!window.go?.main?.App) return
  const s = await go().GetStatus()
  const dbEl = document.getElementById('upd-db-size')
  if (dbEl) dbEl.textContent = t('settings.update.db-summary', {
    domains:  s.dbDomains.toLocaleString(loc()),
    urls:     s.dbUrls.toLocaleString(loc()),
    keywords: s.dbKeywords.toLocaleString(loc()),
  })
  renderDiagnostics(s.diagnostics)
}

function renderDiagnostics(d) {
  const body = document.getElementById('diag-rows')
  if (!body || !d) return
  const ltr = v => `<bdi class="ltr-text">${esc(v)}</bdi>`
  const text = v => `<span dir="auto">${esc(v)}</span>`
  const ap = d.lastApply || {}
  let hosts = ap.hostsError || (ap.hostsApplied ? t('settings.diag.hosts.ok') : (ap.hostsInfo || t('settings.diag.hosts.off')))
  if (ap.hostsApplied && ap.hostsPartial) hosts += ' · ' + t('content.exceptions.hosts-partial')
  const skipped = [...(d.skippedLegacy || []), ...(d.skippedRules || [])]
  const rows = [
    ['settings.diag.path', ltr(d.settingsPath)],
    ['settings.diag.source', text(t('settings.diag.source.' + d.settingsSource))],
    d.migratedFrom && ['settings.diag.migrated-from', ltr(d.migratedFrom)],
    d.backupPath && ['settings.diag.backup', ltr(d.backupPath)],
    d.loadError && ['settings.diag.load-error', ltr(d.loadError)],
    d.loadError && ['settings.diag.recover', text(t('settings.diag.restore-hint'))],
    ['settings.diag.saving', text(d.readOnly ? t('settings.diag.read-only') : (ap.configError || t('settings.diag.ok')))],
    skipped.length && ['settings.diag.skipped', skipped.map(ltr).join(', ')],
    ['settings.diag.hosts', text(hosts)],
  ].filter(Boolean)
  let html = rows.map(([key, val]) => `<tr><td class="lbl">${esc(t(key))}</td><td>${val}</td></tr>`).join('')
  if (ap.closedTunnels > 0) {
    html += `<tr><td colspan="2" dir="auto">${esc(t('settings.diag.tunnels', { count: ap.closedTunnels.toLocaleString(loc()) }))}</td></tr>`
  }
  body.innerHTML = html
  const reload = document.getElementById('diag-reload')
  if (reload) reload.style.display = d.readOnly ? 'block' : 'none'
}
async function reloadSettings() {
  const btn = document.getElementById('btn-reload-settings')
  if (btn) btn.disabled = true
  try { renderDiagnostics(await go().ReloadSettings()); notify(t('toast.settings.reloaded'), 'ok') }
  catch (e) { notify(errText(e), 'err') }
  finally { if (btn) btn.disabled = false; loadUpdate() }
}
window.reloadSettings = reloadSettings
function checkForUpdate() {
  notify(t('toast.update.stub'), 'ok')
}
window.loadUpdate = loadUpdate
window.checkForUpdate = checkForUpdate

// ── Uninstall ─────────────────────────────────────────────────────────────────
async function showUninstall() {
  const hasPw = await go().HasPassword()
  const pw = hasPw ? await requirePassword(t('modal.require.uninstall')) : ''
  if (pw === null) return
  try { await go().Uninstall(pw || ''); notify(t('toast.uninstall.started'), 'ok') }
  catch (e) { notify(errText(e), 'err') }
}
window.showUninstall = showUninstall

// ── Utility ───────────────────────────────────────────────────────────────────
// FSI…PDI so an error mixing Hebrew, Latin identifiers and punctuation stays intact in an RTL container.
function errText(e) {
  return '\u2068' + String(e).replace(/^Error: /, '') + '\u2069'
}

function esc(s) {
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')
}

// ── Generic password confirm modal ────────────────────────────────────────────
let _pwConfirmCancel = null
function requirePassword(title) {
  if (_pwConfirmCancel) return Promise.resolve(null) // a prompt is already open
  const heading = title || t('modal.pw.title')
  return new Promise(resolve => {
    document.getElementById('pwConfirmTitle').textContent = heading
    document.getElementById('pwConfirmInput').value = ''
    document.getElementById('pwConfirmErr').style.display = 'none'
    document.getElementById('pwConfirmBg').classList.add('show')
    setTimeout(() => document.getElementById('pwConfirmInput').focus(), 50)
    _pwConfirmCancel = () => { document.getElementById('pwConfirmBg').classList.remove('show'); resolve(null) }
    document.getElementById('pwConfirmOkBtn').onclick = () => {
      const pw = document.getElementById('pwConfirmInput').value
      document.getElementById('pwConfirmBg').classList.remove('show')
      _pwConfirmCancel = null
      resolve(pw)
    }
  })
}
function closePwConfirm() {
  if (_pwConfirmCancel) { _pwConfirmCancel(); _pwConfirmCancel = null }
}
document.getElementById('pwConfirmInput')?.addEventListener('keydown', e => {
  if (e.key === 'Enter') document.getElementById('pwConfirmOkBtn').click()
  if (e.key === 'Escape') closePwConfirm()
})
window.closePwConfirm = closePwConfirm

// ── Quit modal ────────────────────────────────────────────────────────────────
async function showQuitModal() {
  const hasPw = await go().HasPassword()
  document.getElementById('quit-pw-row').style.display = hasPw ? 'block' : 'none'
  document.getElementById('quit-pw').value = ''
  document.getElementById('quit-err').style.display = 'none'
  document.getElementById('quitModalBg').classList.add('show')
  if (hasPw) setTimeout(() => document.getElementById('quit-pw').focus(), 50)
  if (!hasPw) setTimeout(confirmQuit, 0) // no password set → quit immediately
}

function closeQuitModal() { document.getElementById('quitModalBg').classList.remove('show') }

async function confirmQuit() {
  const pw = document.getElementById('quit-pw')?.value ?? ''
  try {
    await go().ConfirmQuit(pw)
  } catch (e) {
    const errEl = document.getElementById('quit-err')
    errEl.textContent = errText(e)
    errEl.style.display = 'block'
    document.getElementById('quit-pw').select()
  }
}

document.getElementById('quit-pw')?.addEventListener('keydown', e => { if (e.key === 'Enter') confirmQuit() })
window.closeQuitModal = closeQuitModal
window.confirmQuit = confirmQuit
window.showQuitModal = showQuitModal

// ── Init ──────────────────────────────────────────────────────────────────────
function bootstrapI18n() {
  initI18n()
  syncLanguageSelect()
  applyI18n()
  renderLevelCategoryLists()
  reconcileLanguage()
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', bootstrapI18n)
} else {
  bootstrapI18n()
}

window.addEventListener('load', () => {
  const init = () => {
    if (window.go?.main?.App) {
      loadDashboard()
      EventsOn('quit-requested', showQuitModal)
    } else { setTimeout(init, 100) }
  }
  init()
})

setInterval(() => {
  const home = document.getElementById('page-home')
  if (home?.style.display !== 'none' && window.go?.main?.App) loadDashboard()
}, 10000)
