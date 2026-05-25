const elements = {
  pageLabel: document.getElementById('pageLabel'),
  primaryLabel: document.getElementById('primaryLabel'),
  contextLabel: document.getElementById('contextLabel'),
  pageValue: document.getElementById('pageValue'),
  primaryValue: document.getElementById('primaryValue'),
  titleValue: document.getElementById('titleValue'),
  priceValue: document.getElementById('priceValue'),
  contextValue: document.getElementById('contextValue'),
  status: document.getElementById('status'),
  collectButton: document.getElementById('collectButton')
}

let currentInspection = null
let collectable = false

init()

async function init() {
  elements.collectButton.disabled = true
  setStatus('正在识别当前页面...')
  const response = await chrome.runtime.sendMessage({ type: 'INSPECT_ACTIVE_TAB' })
  if (!response?.ok) {
    renderUnsupported(response?.message || '当前页面不在支持范围内', Boolean(response?.canManualCollect))
    return
  }
  currentInspection = response.data || null
  renderInspection(currentInspection)
}

elements.collectButton.addEventListener('click', async () => {
  if (!collectable) {
    return
  }
  elements.collectButton.disabled = true
  setStatus('正在采集到系统...')
  const response = await chrome.runtime.sendMessage({ type: 'COLLECT_ACTIVE_TAB' })
  if (!response?.ok) {
    setStatus(response?.message || '采集失败', 'error')
    elements.collectButton.disabled = false
    return
  }
  setStatus(response.message || '采集成功', 'success')
  elements.collectButton.disabled = false
})

function renderInspection(inspection) {
  const pageType = inspection?.pageType || ''
  if (pageType === 'amazon-detail') {
    const payload = inspection.payload || {}
    elements.pageLabel.textContent = '页面'
    elements.primaryLabel.textContent = 'ASIN'
    elements.contextLabel.textContent = '上下文'
    elements.pageValue.textContent = 'Amazon 详情页'
    elements.primaryValue.textContent = payload.asin || '--'
    elements.titleValue.textContent = payload.title || '--'
    elements.priceValue.textContent = formatPrice(payload.priceAmount, payload.currencyCode)
    elements.contextValue.textContent = `${payload.images?.length || 0} 张图 / ${payload.siteCode || '--'}`
    elements.collectButton.textContent = '立即采集'
    elements.collectButton.disabled = false
    collectable = true
    setStatus('页面识别完成', 'success')
    return
  }

  if (pageType === '1688-search') {
    const taskContext = inspection.taskContext || {}
    elements.pageLabel.textContent = '页面'
    elements.primaryLabel.textContent = '任务'
    elements.contextLabel.textContent = '系统 code'
    elements.pageValue.textContent = '1688 图搜页'
    elements.primaryValue.textContent = shortenToken(taskContext.taskToken)
    elements.titleValue.textContent = '已接收采集任务，等待选款'
    elements.priceValue.textContent = '--'
    elements.contextValue.textContent = taskContext.systemCode || '--'
    elements.collectButton.textContent = '等待选款'
    elements.collectButton.disabled = true
    collectable = false
    setStatus(taskContext.systemCode ? `已接收系统 code：${taskContext.systemCode}，等待选款` : '当前图搜页已接收采集任务', 'success')
    return
  }

  if (pageType === '1688-detail') {
    const payload = inspection.payload || {}
    const taskContext = inspection.taskContext || {}
    elements.pageLabel.textContent = '页面'
    elements.primaryLabel.textContent = 'Offer'
    elements.contextLabel.textContent = '系统 code'
    elements.pageValue.textContent = '1688 详情页'
    elements.primaryValue.textContent = payload.offerId || '--'
    elements.titleValue.textContent = payload.title || '当前详情页可自动或手动采集'
    elements.priceValue.textContent = payload.priceText || '--'
    elements.contextValue.textContent = taskContext.systemCode || '--'
    elements.collectButton.textContent = '采集货物'
    elements.collectButton.disabled = false
    collectable = true
    setStatus(taskContext.taskToken ? '详情页已准备采集，若自动采集失败可手动补采' : '当前详情页缺少采集任务，可尝试手动补采', taskContext.taskToken ? 'success' : 'error')
    return
  }

  renderUnsupported('当前页面不在支持范围内', false)
}

function renderUnsupported(message, canManualCollect = false) {
  elements.pageLabel.textContent = '页面'
  elements.primaryLabel.textContent = '主标识'
  elements.contextLabel.textContent = '上下文'
  elements.pageValue.textContent = '--'
  elements.primaryValue.textContent = '--'
  elements.titleValue.textContent = message
  elements.priceValue.textContent = '--'
  elements.contextValue.textContent = '--'
  elements.collectButton.textContent = canManualCollect ? '手动采集到系统' : '立即采集'
  elements.collectButton.disabled = !canManualCollect
  collectable = canManualCollect
  setStatus(message, canManualCollect ? 'warning' : 'error')
}

function setStatus(message, tone = '') {
  elements.status.textContent = message
  elements.status.className = `status ${tone}`.trim()
}

function formatPrice(price, currencyCode) {
  if (price === null || typeof price === 'undefined') {
    return '--'
  }
  return `${currencyCode || ''} ${Number(price).toFixed(2)}`.trim()
}

function shortenToken(value) {
  const text = String(value || '').trim()
  if (!text) {
    return '--'
  }
  if (text.length <= 12) {
    return text
  }
  return `${text.slice(0, 6)}...${text.slice(-4)}`
}
