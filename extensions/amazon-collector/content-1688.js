(function () {
  const BUTTON_ID = 'amazon-collector-floating-button'
  const TOAST_ID = 'amazon-collector-toast'
  const BADGE_ID = 'amazon-collector-floating-badge'
  const COLLECT_TASK_PARAM = '__gva1688Task'
  const PROCUREMENT_TASK_PARAM = '__gva1688ProcurementTask'
  const PAGE_BRIDGE_SOURCE = 'amazon-collector-1688'

  let currentTaskContext = null
  let currentProcurementContext = null
  let autoActionKey = ''
  let imageSearchUploadKey = ''
  let pageBridgeInjected = false
  let linkObserver = null
  let activeButtonHandler = null

  if (!isSupported1688Page()) {
    return
  }

  init().catch((error) => {
    showToast(error?.message || '1688 页面初始化失败', 'error')
  })

  async function init() {
    await refreshPageState()
    observeLocationChange()
  }

  async function refreshPageState() {
    resetFloatingButton()
    ensureBadge()
    currentTaskContext = null
    currentProcurementContext = null
    disconnectLinkObserver()

    const procurementToken = getProcurementTaskTokenFromUrl()
    if (procurementToken) {
      if (is1688DetailPage()) {
        await handleProcurementDetailPage(procurementToken)
        renderProcurementBadge()
        return
      }
      if (is1688CheckoutPage()) {
        await handleProcurementCheckoutPage(procurementToken)
        renderProcurementBadge()
        return
      }
      renderBadge('当前采购页不支持自动处理', 'warning')
      return
    }

    if (is1688SearchPage()) {
      await handleSearchPage()
      renderCollectBadge()
      return
    }
    if (is1688DetailPage()) {
      await handleCollectDetailPage()
      renderCollectBadge()
      return
    }
    renderBadge('当前页面不是 1688 图搜、详情页或订单页', 'warning')
  }

  async function handleSearchPage() {
    const taskToken = getCollectTaskTokenFromUrl()
    if (!taskToken) {
      renderBadge('当前图搜页未携带采集任务', 'warning')
      return
    }
    try {
      const response = await chrome.runtime.sendMessage({
        type: 'SET_1688_TASK_CONTEXT',
        payload: {
          taskToken,
          pageType: 'search'
        }
      })
      if (response?.ok) {
        currentTaskContext = response.data || null
      } else {
        showToast(response?.message || '1688 采集任务上下文初始化失败', 'error')
      }
    } catch (error) {
      showToast(error?.message || '1688 采集任务上下文初始化失败', 'error')
    }
    decorateDetailLinks()
    observeDetailLinks()
    window.setTimeout(() => {
      ensureImageSearchTriggered(taskToken).catch((error) => {
        const message = error?.message || '1688 图搜补传图片失败'
        renderBadge(message, 'warning')
        showToast(message, 'error')
      })
    }, 1400)
  }

  async function handleCollectDetailPage() {
    currentTaskContext = await loadCollectTaskContext()
    ensureFloatingButton('采集货物', () => {
      collectDetail(true).catch((error) => {
        showToast(error?.message || '采集失败', 'error')
      })
    })
    if (!currentTaskContext?.taskToken) {
      return
    }
    const offerId = extractOfferIdFromPath()
    const currentKey = `collect:${currentTaskContext.taskToken}:${offerId || 'unknown'}`
    if (!offerId || autoActionKey === currentKey) {
      return
    }
    autoActionKey = currentKey
    window.setTimeout(() => {
      collectDetail(false).catch((error) => {
        showToast(error?.message || '自动采集失败', 'error')
      })
    }, 1600)
  }

  async function handleProcurementDetailPage(taskToken) {
    currentProcurementContext = await loadProcurementTask(taskToken)
    ensureFloatingButton('执行采购', () => {
      runProcurementForCurrentItem(true).catch((error) => {
        showToast(error?.message || '采购执行失败', 'error')
      })
    })
    if (!currentProcurementContext?.task?.items?.length) {
      return
    }
    const currentItem = resolveCurrentProcurementItem()
    if (!currentItem) {
      renderBadge('当前 1688 商品不在采购任务列表中', 'warning')
      return
    }
    const currentKey = `procurement:${taskToken}:${currentItem.groupItemId}`
    if (autoActionKey === currentKey) {
      return
    }
    autoActionKey = currentKey
    window.setTimeout(() => {
      runProcurementForCurrentItem(false).catch((error) => {
        showToast(error?.message || '自动采购失败', 'error')
      })
    }, 1800)
  }

  async function handleProcurementCheckoutPage(taskToken) {
    currentProcurementContext = await loadProcurementTask(taskToken)
    ensureFloatingButton('回传采购单号', () => {
      reportProcurementResult().catch((error) => {
        showToast(error?.message || '回传采购结果失败', 'error')
      })
    })
    const currentKey = `procurement-report:${taskToken}:${window.location.pathname}`
    if (autoActionKey === currentKey) {
      return
    }
    const orderNo = extractProcurementOrderNo()
    if (!orderNo) {
      return
    }
    autoActionKey = currentKey
    window.setTimeout(() => {
      reportProcurementResult().catch(() => {})
    }, 1200)
  }

  async function loadCollectTaskContext() {
    const directTaskToken = getCollectTaskTokenFromUrl()
    try {
      const response = await chrome.runtime.sendMessage({
        type: 'GET_1688_TASK_CONTEXT',
        payload: {
          taskToken: directTaskToken
        }
      })
      if (!response?.ok) {
        return directTaskToken ? { taskToken: directTaskToken } : null
      }
      return response.data || (directTaskToken ? { taskToken: directTaskToken } : null)
    } catch (error) {
      return directTaskToken ? { taskToken: directTaskToken } : null
    }
  }

  async function loadProcurementTask(taskToken) {
    const response = await chrome.runtime.sendMessage({
      type: 'GET_1688_PROCUREMENT_TASK',
      payload: {
        taskToken
      }
    })
    if (!response?.ok) {
      throw new Error(response?.message || '加载采购任务失败')
    }
    return response.data || null
  }

  async function collectDetail(manual) {
    if (!is1688DetailPage()) {
      throw new Error('当前页面不是 1688 详情页')
    }
    const button = document.getElementById(BUTTON_ID)
    const originalText = button?.textContent || '采集货物'
    setButtonState(button, true, manual ? '手动采集中...' : '采集中...')
    try {
      const response = await chrome.runtime.sendMessage({
        type: 'COLLECT_ACTIVE_TAB',
        payload: {
          taskToken: currentTaskContext?.taskToken || getCollectTaskTokenFromUrl(),
          manual
        }
      })
      if (!response?.ok) {
        throw new Error(response?.message || '采集失败')
      }
      if (response?.data?.systemCode) {
        currentTaskContext = {
          ...(currentTaskContext || {}),
          taskToken: response?.payload?.taskToken || currentTaskContext?.taskToken || getCollectTaskTokenFromUrl(),
          systemCode: response.data.systemCode
        }
      }
      renderCollectBadge()
      showToast(response.message || `货物已采集到系统：${currentTaskContext?.systemCode || '--'}`, 'success')
    } finally {
      setButtonState(button, false, originalText)
    }
  }

  async function runProcurementForCurrentItem(manual) {
    const taskToken = getProcurementTaskTokenFromUrl()
    if (!taskToken) {
      throw new Error('未找到采购任务 token')
    }
    if (!currentProcurementContext?.task?.items?.length) {
      currentProcurementContext = await loadProcurementTask(taskToken)
    }
    const currentItem = resolveCurrentProcurementItem()
    if (!currentItem) {
      throw new Error('当前页面不在采购任务商品列表中')
    }

    const button = document.getElementById(BUTTON_ID)
    const originalText = button?.textContent || '执行采购'
    setButtonState(button, true, manual ? '手动采购中...' : '采购中...')
    try {
      const attrTexts = extractProcurementAttrTexts(currentItem.selected1688SkuAttrs)
      for (const text of attrTexts) {
        await clickBestMatch(text)
        await sleep(180)
      }

      setPurchaseQuantity(currentItem.purchaseQuantity)
      await sleep(120)

      const clicked = clickActionButton(['加入进货单', '加入购物车', '立即订购', '立即购买'])
      if (!clicked) {
        throw new Error('未找到可点击的采购按钮')
      }
      await sleep(1200)

      const response = await chrome.runtime.sendMessage({
        type: 'MARK_1688_PROCUREMENT_ITEM_COMPLETE',
        payload: {
          taskToken,
          groupItemId: currentItem.groupItemId,
          collectedProductId: currentItem.collectedProductId,
          selected1688SkuKey: currentItem.selected1688SkuKey,
          selected1688SkuAttrs: currentItem.selected1688SkuAttrs || {},
          purchaseQuantity: currentItem.purchaseQuantity
        }
      })
      if (!response?.ok) {
        throw new Error(response?.message || '采购进度保存失败')
      }

      currentProcurementContext = response.data || currentProcurementContext
      if (response.data?.nextItem?.productUrl) {
        showToast('当前货物已加入采购流程，正在打开下一件商品', 'success')
        window.location.href = appendProcurementTaskParam(response.data.nextItem.productUrl, taskToken)
        return
      }

      showToast('采购项已全部加入购物车，请提交 1688 订单后在订单页回传采购单号', 'success')
      window.setTimeout(() => {
        window.location.href = appendProcurementTaskParam('https://cart.1688.com/cart.htm', taskToken)
      }, 900)
    } finally {
      setButtonState(button, false, originalText)
    }
  }

  async function reportProcurementResult() {
    const taskToken = getProcurementTaskTokenFromUrl()
    if (!taskToken) {
      throw new Error('未找到采购任务 token')
    }
    const orderNo = extractProcurementOrderNo()
    if (!orderNo) {
      throw new Error('当前页面未识别到 1688 采购单号，请在订单成功页或订单详情页重试')
    }
    const response = await chrome.runtime.sendMessage({
      type: 'REPORT_1688_PROCUREMENT_TASK_RESULT',
      payload: {
        taskToken,
        status: 'success',
        orderNo1688: orderNo,
        orderUrl: window.location.href
      }
    })
    if (!response?.ok) {
      throw new Error(response?.message || '采购结果回传失败')
    }
    showToast(response.message || `采购单号 ${orderNo} 已回传`, 'success')
    renderProcurementBadge(`采购单号 ${orderNo} 已回传`, 'success')
  }

  function resolveCurrentProcurementItem() {
    const items = currentProcurementContext?.task?.items || []
    const offerId = extractOfferIdFromPath()
    const currentUrl = normalizeComparableUrl(window.location.href)
    return items.find((item) => {
      const itemUrl = normalizeComparableUrl(item.productUrl)
      if (item.groupItemId && (currentProcurementContext?.completedGroupItemIds || []).includes(item.groupItemId)) {
        return false
      }
      return (offerId && String(item.offerId || '').trim() === offerId) || (itemUrl && itemUrl === currentUrl)
    }) || null
  }

  function renderCollectBadge() {
    if (is1688SearchPage()) {
      if (currentTaskContext?.systemCode) {
        renderBadge(`系统 code: ${currentTaskContext.systemCode}，等待选款`, 'success')
        return
      }
      renderBadge(currentTaskContext?.taskToken ? '已接收 1688 采集任务，等待选款' : '当前图搜页未携带采集任务', currentTaskContext?.taskToken ? 'success' : 'warning')
      return
    }
    if (is1688DetailPage()) {
      if (currentTaskContext?.systemCode) {
        renderBadge(`系统 code: ${currentTaskContext.systemCode}，详情页可自动或手动采集`, 'success')
        return
      }
      renderBadge('当前详情页未携带有效采集任务，可尝试手动采集', 'warning')
    }
  }

  function renderProcurementBadge(message, tone) {
    if (message) {
      renderBadge(message, tone)
      return
    }
    const completed = currentProcurementContext?.completedGroupItemIds?.length || 0
    const total = currentProcurementContext?.task?.items?.length || 0
    if (is1688DetailPage()) {
      renderBadge(`1688 采购执行中 ${completed}/${total}，当前页会自动选规格并加入购物车`, 'success')
      return
    }
    if (is1688CheckoutPage()) {
      renderBadge(`1688 采购已完成 ${completed}/${total}，请提交订单后回传采购单号`, 'warning')
      return
    }
    renderBadge('1688 采购任务已加载', 'success')
  }

  function ensureFloatingButton(label, handler) {
    let button = document.getElementById(BUTTON_ID)
    if (!button) {
      button = document.createElement('button')
      button.id = BUTTON_ID
      button.type = 'button'
      document.documentElement.appendChild(button)
    }
    if (activeButtonHandler) {
      button.removeEventListener('click', activeButtonHandler)
    }
    activeButtonHandler = handler
    button.textContent = label
    button.disabled = false
    button.addEventListener('click', activeButtonHandler)
    return button
  }

  function resetFloatingButton() {
    const button = document.getElementById(BUTTON_ID)
    if (!button) {
      return
    }
    if (activeButtonHandler) {
      button.removeEventListener('click', activeButtonHandler)
      activeButtonHandler = null
    }
    button.remove()
  }

  function setButtonState(button, disabled, text) {
    if (!button) {
      return
    }
    button.disabled = disabled
    if (text) {
      button.textContent = text
    }
  }

  function ensureBadge() {
    let badge = document.getElementById(BADGE_ID)
    if (!badge) {
      badge = document.createElement('div')
      badge.id = BADGE_ID
      document.documentElement.appendChild(badge)
    }
    return badge
  }

  function renderBadge(message, tone) {
    const badge = ensureBadge()
    badge.textContent = message
    badge.dataset.tone = tone || 'warning'
  }

  function decorateDetailLinks() {
    const taskToken = currentTaskContext?.taskToken || getCollectTaskTokenFromUrl()
    if (!taskToken) {
      return
    }
    Array.from(document.querySelectorAll('a[href]')).forEach((anchor) => {
      const href = anchor.getAttribute('href') || ''
      if (!isDetailOfferHref(href)) {
        return
      }
      anchor.href = appendQueryTaskParam(href, COLLECT_TASK_PARAM, taskToken)
    })
  }

  function observeDetailLinks() {
    disconnectLinkObserver()
    linkObserver = new MutationObserver(() => {
      decorateDetailLinks()
    })
    linkObserver.observe(document.documentElement, {
      childList: true,
      subtree: true
    })
    document.addEventListener('click', decorateClickLink, true)
  }

  async function ensureImageSearchTriggered(taskToken) {
    const imageAddress = getImageAddressFromUrl() || String(currentTaskContext?.mainImageUrl || '').trim()
    if (!taskToken || !imageAddress) {
      return
    }

    for (let i = 0; i < 5; i += 1) {
      if (hasImageSearchResults()) {
        decorateDetailLinks()
        return
      }
      await sleep(700)
    }

    const currentKey = `${taskToken}:${imageAddress}`
    if (imageSearchUploadKey === currentKey) {
      return
    }
    imageSearchUploadKey = currentKey
    renderBadge('图搜未自动触发，正在补传图片', 'warning')

    const response = await chrome.runtime.sendMessage({
      type: 'FETCH_IMAGE_AS_DATA_URL',
      payload: {
        imageUrl: imageAddress
      }
    })
    if (!response?.ok || !response?.data?.dataUrl) {
      throw new Error(response?.message || '图片下载失败，无法触发 1688 图搜')
    }

    try {
      const uploadResult = await uploadImageWith1688Mtop(response.data.dataUrl)
      if (uploadResult?.imageId) {
        renderBadge('已上传图片，正在进入 1688 图搜结果', 'success')
        window.location.href = build1688ImageResultUrl(uploadResult.imageId, taskToken)
        return
      }
    } catch (error) {
      renderBadge('mtop 上传失败，改用页面上传控件兜底', 'warning')
    }

    const file = dataUrlToFile(
      response.data.dataUrl,
      response.data.fileName || '1688-image-search.jpg',
      response.data.contentType || 'image/jpeg'
    )
    let input = findImageFileInput()
    if (!input) {
      clickImageSearchEntry()
      await sleep(500)
      input = findImageFileInput()
    }
    if (!input) {
      throw new Error('未找到 1688 图搜上传控件')
    }

    setFileInputFiles(input, file)
    renderBadge('已补传图片，等待 1688 图搜结果', 'success')
    showToast('已补传图片触发 1688 图搜', 'success')

    await sleep(1200)
    decorateDetailLinks()
    if (!hasImageSearchResults()) {
      clickActionButton(['搜索', '搜同款', '找相似'])
      await sleep(1200)
      decorateDetailLinks()
    }
  }

  async function uploadImageWith1688Mtop(dataUrl) {
    await ensurePageBridgeInjected()
    const requestId = `img-${Date.now()}-${Math.random().toString(16).slice(2)}`
    return new Promise((resolve, reject) => {
      const timeout = window.setTimeout(() => {
        window.removeEventListener('message', handleMessage)
        reject(new Error('1688 图片上传超时'))
      }, 15000)

      function handleMessage(event) {
        if (event.source !== window) {
          return
        }
        const data = event.data || {}
        if (data.source !== PAGE_BRIDGE_SOURCE || data.type !== 'UPLOAD_IMAGE_WITH_MTOP_RESULT' || data.requestId !== requestId) {
          return
        }
        window.clearTimeout(timeout)
        window.removeEventListener('message', handleMessage)
        if (!data.ok) {
          reject(new Error(data.message || '1688 图片上传失败'))
          return
        }
        resolve(data)
      }

      window.addEventListener('message', handleMessage)
      window.postMessage({
        source: PAGE_BRIDGE_SOURCE,
        type: 'UPLOAD_IMAGE_WITH_MTOP',
        requestId,
        dataUrl
      }, '*')
    })
  }

  async function ensurePageBridgeInjected() {
    if (pageBridgeInjected) {
      return
    }
    pageBridgeInjected = true
    const script = document.createElement('script')
    script.src = chrome.runtime.getURL('content-1688-page-bridge.js')
    script.async = false
    await new Promise((resolve, reject) => {
      script.onload = () => {
        script.remove()
        resolve()
      }
      script.onerror = () => {
        pageBridgeInjected = false
        script.remove()
        reject(new Error('1688 页面桥接脚本加载失败'))
      }
      ;(document.head || document.documentElement).appendChild(script)
    })
  }

  function build1688ImageResultUrl(imageId, taskToken) {
    const url = new URL('https://s.1688.com/shen/sell_offer.htm')
    url.searchParams.set('tab', 'imageSearch')
    url.searchParams.set('imageType', 'oss')
    url.searchParams.set('imageAddress', imageId)
    url.searchParams.set('spm', 'a261y.7663282.imagesearch.upload')
    url.searchParams.set('scene', 'same_shop_search')
    url.searchParams.set(COLLECT_TASK_PARAM, taskToken)
    return url.toString()
  }

  function getImageAddressFromUrl() {
    try {
      const currentUrl = new URL(window.location.href)
      return String(currentUrl.searchParams.get('imageAddress') || '').trim()
    } catch (error) {
      return ''
    }
  }

  function hasImageSearchResults() {
    return Array.from(document.querySelectorAll('a[href]'))
      .some((anchor) => isDetailOfferHref(anchor.getAttribute('href') || '') && isVisibleElement(anchor))
  }

  function findImageFileInput() {
    const inputs = Array.from(document.querySelectorAll('input[type="file"]'))
    if (!inputs.length) {
      return null
    }
    return inputs.find((input) => {
      const profile = `${input.accept || ''} ${input.name || ''} ${input.id || ''} ${input.className || ''}`.toLowerCase()
      return profile.includes('image') || profile.includes('img') || profile.includes('file') || profile.includes('upload')
    }) || inputs[0]
  }

  function clickImageSearchEntry() {
    return clickActionButton(['图片搜索', '以图搜', '搜图'])
  }

  function setFileInputFiles(input, file) {
    const transfer = new DataTransfer()
    transfer.items.add(file)
    input.files = transfer.files
    input.dispatchEvent(new Event('input', { bubbles: true }))
    input.dispatchEvent(new Event('change', { bubbles: true }))
  }

  function dataUrlToFile(dataUrl, fileName, contentType) {
    const parts = String(dataUrl || '').split(',')
    if (parts.length < 2) {
      throw new Error('图片内容格式无效')
    }
    const metaType = parts[0].match(/^data:([^;]+)/i)?.[1] || ''
    const binary = atob(parts.slice(1).join(','))
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i += 1) {
      bytes[i] = binary.charCodeAt(i)
    }
    return new File([bytes], fileName, {
      type: contentType || metaType || 'image/jpeg'
    })
  }

  function disconnectLinkObserver() {
    if (linkObserver) {
      linkObserver.disconnect()
      linkObserver = null
    }
    document.removeEventListener('click', decorateClickLink, true)
  }

  function decorateClickLink(event) {
    const taskToken = currentTaskContext?.taskToken || getCollectTaskTokenFromUrl()
    if (!taskToken) {
      return
    }
    const anchor = event.target?.closest?.('a[href]')
    if (!anchor) {
      return
    }
    const href = anchor.getAttribute('href') || ''
    if (!isDetailOfferHref(href)) {
      return
    }
    anchor.href = appendQueryTaskParam(href, COLLECT_TASK_PARAM, taskToken)
  }

  function appendProcurementTaskParam(rawHref, taskToken) {
    return appendQueryTaskParam(rawHref, PROCUREMENT_TASK_PARAM, taskToken)
  }

  function appendQueryTaskParam(rawHref, paramName, taskToken) {
    try {
      const url = new URL(rawHref, window.location.href)
      url.searchParams.set(paramName, taskToken)
      return url.toString()
    } catch (error) {
      return rawHref
    }
  }

  function getCollectTaskTokenFromUrl() {
    try {
      const currentUrl = new URL(window.location.href)
      return String(currentUrl.searchParams.get(COLLECT_TASK_PARAM) || '').trim()
    } catch (error) {
      return ''
    }
  }

  function getProcurementTaskTokenFromUrl() {
    try {
      const currentUrl = new URL(window.location.href)
      return String(currentUrl.searchParams.get(PROCUREMENT_TASK_PARAM) || '').trim()
    } catch (error) {
      return ''
    }
  }

  function extractOfferIdFromPath() {
    const matched = window.location.pathname.match(/offer\/(\d+)\.html/i)
    return matched?.[1] || ''
  }

  function isSupported1688Page() {
    return ['s.1688.com', 'detail.1688.com', 'cart.1688.com', 'trade.1688.com'].includes(window.location.host)
  }

  function is1688SearchPage() {
    return window.location.host === 's.1688.com' && (
      /\/youyuan\/index\.htm$/i.test(window.location.pathname) ||
      /\/shen\/sell_offer\.html?$/i.test(window.location.pathname)
    )
  }

  function is1688DetailPage() {
    return window.location.host === 'detail.1688.com' && /\/offer\/\d+\.html/i.test(window.location.pathname)
  }

  function is1688CheckoutPage() {
    return ['cart.1688.com', 'trade.1688.com'].includes(window.location.host)
  }

  function isDetailOfferHref(href) {
    try {
      const url = new URL(href, window.location.href)
      return url.host === 'detail.1688.com' && /\/offer\/\d+\.html/i.test(url.pathname)
    } catch (error) {
      return false
    }
  }

  function observeLocationChange() {
    let lastHref = window.location.href
    window.setInterval(() => {
      if (window.location.href === lastHref) {
        return
      }
      lastHref = window.location.href
      autoActionKey = ''
      imageSearchUploadKey = ''
      refreshPageState().catch((error) => {
        showToast(error?.message || '1688 页面刷新失败', 'error')
      })
    }, 800)
  }

  async function clickBestMatch(expectedText) {
    const target = String(expectedText || '').trim()
    if (!target) {
      return false
    }
    const candidates = Array.from(document.querySelectorAll('button, a, li, span, div, label'))
      .filter(isVisibleElement)
      .map((element) => ({
        element,
        text: normalizeText(element.innerText || element.textContent || '')
      }))
      .filter((item) => item.text && item.text.includes(normalizeText(target)))

    if (!candidates.length) {
      return false
    }
    candidates[0].element.click()
    return true
  }

  function setPurchaseQuantity(quantity) {
    const targetQuantity = Number(quantity || 0)
    if (!targetQuantity || targetQuantity <= 0) {
      return false
    }
    const input = Array.from(document.querySelectorAll('input'))
      .find((element) => {
        const text = `${element.name || ''} ${element.id || ''} ${element.placeholder || ''}`.toLowerCase()
        return isVisibleElement(element) && (element.type === 'number' || text.includes('qty') || text.includes('quantity') || text.includes('数量'))
      })
    if (!input) {
      return false
    }
    input.focus()
    input.value = String(targetQuantity)
    input.dispatchEvent(new Event('input', { bubbles: true }))
    input.dispatchEvent(new Event('change', { bubbles: true }))
    return true
  }

  function clickActionButton(texts) {
    const normalizedTexts = texts.map((text) => normalizeText(text))
    const candidates = Array.from(document.querySelectorAll('button, a, span, div'))
      .filter(isVisibleElement)
      .find((element) => normalizedTexts.some((text) => normalizeText(element.innerText || element.textContent || '').includes(text)))
    if (!candidates) {
      return false
    }
    candidates.click()
    return true
  }

  function extractProcurementAttrTexts(attrs) {
    if (!attrs || typeof attrs !== 'object') {
      return []
    }
    return Array.from(new Set(Object.values(attrs)
      .flatMap((value) => Array.isArray(value) ? value : [value])
      .map((value) => normalizeText(formatValue(value)))
      .filter(Boolean)))
  }

  function extractProcurementOrderNo() {
    const urlMatch = window.location.href.match(/(?:orderId|orderNo|bizOrderId)=([0-9]{8,})/i)
    if (urlMatch?.[1]) {
      return urlMatch[1]
    }
    const text = document.body?.innerText || ''
    const labelMatch = text.match(/(?:订单编号|订单号|采购单号)[^0-9]{0,8}([0-9]{8,})/)
    if (labelMatch?.[1]) {
      return labelMatch[1]
    }
    const looseMatch = text.match(/\b([0-9]{12,})\b/)
    return looseMatch?.[1] || ''
  }

  function normalizeComparableUrl(rawUrl) {
    try {
      const url = new URL(rawUrl, window.location.href)
      url.hash = ''
      url.searchParams.delete(COLLECT_TASK_PARAM)
      url.searchParams.delete(PROCUREMENT_TASK_PARAM)
      return url.toString()
    } catch (error) {
      return ''
    }
  }

  function normalizeText(value) {
    return String(value || '').replace(/\s+/g, '').trim().toLowerCase()
  }

  function formatValue(value) {
    if (value === null || typeof value === 'undefined') {
      return ''
    }
    if (typeof value === 'string') {
      return value
    }
    if (typeof value === 'object') {
      try {
        return Object.values(value).join(' ')
      } catch (error) {
        return ''
      }
    }
    return String(value)
  }

  function isVisibleElement(element) {
    if (!element) {
      return false
    }
    const style = window.getComputedStyle(element)
    const rect = element.getBoundingClientRect()
    return style.visibility !== 'hidden' && style.display !== 'none' && rect.width > 0 && rect.height > 0
  }

  function sleep(ms) {
    return new Promise((resolve) => window.setTimeout(resolve, ms))
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
    }, 3200)
  }
})()
