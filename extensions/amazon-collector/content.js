(function () {
  const BUTTON_ID = 'amazon-collector-floating-button'
  const TOAST_ID = 'amazon-collector-toast'
  const BADGE_ID = 'amazon-collector-floating-badge'

  if (!isSupportedAmazonPage() || document.getElementById(BUTTON_ID)) {
    return
  }

  const button = document.createElement('button')
  button.id = BUTTON_ID
  button.type = 'button'
  button.textContent = getButtonText()
  button.addEventListener('click', handleCollect)
  document.documentElement.appendChild(button)
  renderBadge()
  observeLocationChange()

  async function handleCollect() {
    const originalText = button.textContent
    button.disabled = true
    button.textContent = '采集中...'

    try {
      const response = await chrome.runtime.sendMessage({ type: 'COLLECT_ACTIVE_TAB' })
      if (!response?.ok) {
        showToast(response?.message || '采集失败', 'error')
      } else {
        showToast(response.message || '采集成功', 'success')
      }
    } catch (error) {
      showToast(error?.message || '采集失败', 'error')
    } finally {
      button.disabled = false
      button.textContent = getButtonText() || originalText
    }
  }

  function isSupportedAmazonPage() {
    return ['www.amazon.com', 'www.amazon.ca', 'www.amazon.com.mx'].includes(window.location.host)
  }

  function isAmazonDetailPage() {
    return /(?:^|\/)(?:dp|gp\/product|gp\/aw\/d)\/[A-Z0-9]{10}(?:[/?#]|$)/i.test(window.location.pathname)
  }

  function getButtonText() {
    return isAmazonDetailPage() ? '采集到系统' : '手动采集到系统'
  }

  function renderBadge() {
    let badge = document.getElementById(BADGE_ID)
    if (!badge) {
      badge = document.createElement('div')
      badge.id = BADGE_ID
      document.documentElement.appendChild(badge)
    }
    badge.textContent = isAmazonDetailPage()
      ? '已识别到 Amazon 详情页'
      : '未自动识别，仍可手动点击采集'
    badge.dataset.tone = isAmazonDetailPage() ? 'success' : 'warning'
  }

  function observeLocationChange() {
    let lastPath = window.location.pathname
    window.setInterval(() => {
      if (window.location.pathname === lastPath) {
        return
      }
      lastPath = window.location.pathname
      button.textContent = getButtonText()
      renderBadge()
    }, 800)
  }

  function showToast(message, tone) {
    let toast = document.getElementById(TOAST_ID)
    if (!toast) {
      toast = document.createElement('div')
      toast.id = TOAST_ID
      document.documentElement.appendChild(toast)
    }
    toast.textContent = message
    toast.dataset.tone = tone
    toast.classList.add('is-visible')
    window.clearTimeout(showToast.timer)
    showToast.timer = window.setTimeout(() => {
      toast.classList.remove('is-visible')
    }, 2600)
  }
})()
