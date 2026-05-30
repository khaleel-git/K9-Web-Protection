// K9 Web Protection — Content Script
// Runs at document_start for keyword-based blocking before the page renders.

;(async () => {
  const url = window.location.href.toLowerCase()

  // Ignore extension pages themselves
  if (url.startsWith('chrome-extension://') || url.startsWith('chrome://')) return

  const data = await chrome.storage.local.get(['enabled', 'userKeywords'])
  if (!data.enabled) return

  const keywords = data.userKeywords || []
  for (const kw of keywords) {
    if (!kw) continue
    if (url.includes(kw.toLowerCase())) {
      const blockedUrl = chrome.runtime.getURL('blocked.html') +
        '?url=' + encodeURIComponent(window.location.href) +
        '&reason=keyword'
      window.location.replace(blockedUrl)
      return
    }
  }
})()
