import {
  DEFAULT_SETTINGS,
  buildApiUrl,
  createResponse,
  extract1688ProcurementTaskToken,
  extract1688TaskToken,
  is1688DetailUrl,
  is1688SearchUrl,
  isAmazonDetailUrl,
  isSupported1688Host,
  isSupportedAmazonHost,
  normalizeSettings
} from './shared/schema.js'

const MESSAGE_TYPES = {
  GET_SETTINGS: 'GET_SETTINGS',
  SAVE_SETTINGS: 'SAVE_SETTINGS',
  TEST_CONNECTION: 'TEST_CONNECTION',
  INSPECT_ACTIVE_TAB: 'INSPECT_ACTIVE_TAB',
  COLLECT_ACTIVE_TAB: 'COLLECT_ACTIVE_TAB',
  FETCH_IMAGE_AS_DATA_URL: 'FETCH_IMAGE_AS_DATA_URL',
  SET_1688_TASK_CONTEXT: 'SET_1688_TASK_CONTEXT',
  GET_1688_TASK_CONTEXT: 'GET_1688_TASK_CONTEXT',
  GET_1688_PROCUREMENT_TASK: 'GET_1688_PROCUREMENT_TASK',
  MARK_1688_PROCUREMENT_ITEM_COMPLETE: 'MARK_1688_PROCUREMENT_ITEM_COMPLETE',
  REPORT_1688_PROCUREMENT_TASK_STATE: 'REPORT_1688_PROCUREMENT_TASK_STATE',
  REPORT_1688_PROCUREMENT_TASK_RESULT: 'REPORT_1688_PROCUREMENT_TASK_RESULT'
}

const STORAGE_KEYS = {
  last1688TaskToken: 'collector1688:lastTaskToken',
  last1688ProcurementTaskToken: 'collector1688:lastProcurementTaskToken'
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  handleMessage(message, sender)
    .then((result) => sendResponse(result))
    .catch((error) => sendResponse(createResponse(false, { message: error.message || '执行失败' })))
  return true
})

async function handleMessage(message, sender) {
  switch (message?.type) {
    case MESSAGE_TYPES.GET_SETTINGS:
      return createResponse(true, { data: await getSettings() })
    case MESSAGE_TYPES.SAVE_SETTINGS:
      return saveSettings(message.payload || {})
    case MESSAGE_TYPES.TEST_CONNECTION:
      return testConnection()
    case MESSAGE_TYPES.INSPECT_ACTIVE_TAB:
      return inspectActiveTab(sender)
    case MESSAGE_TYPES.COLLECT_ACTIVE_TAB:
      return collectActiveTab(sender, message.payload || {})
    case MESSAGE_TYPES.FETCH_IMAGE_AS_DATA_URL:
      return fetchImageAsDataUrl(message.payload || {})
    case MESSAGE_TYPES.SET_1688_TASK_CONTEXT:
      return set1688TaskContext(sender, message.payload || {})
    case MESSAGE_TYPES.GET_1688_TASK_CONTEXT:
      return get1688TaskContext(sender, message.payload || {})
    case MESSAGE_TYPES.GET_1688_PROCUREMENT_TASK:
      return get1688ProcurementTask(sender, message.payload || {})
    case MESSAGE_TYPES.MARK_1688_PROCUREMENT_ITEM_COMPLETE:
      return mark1688ProcurementItemComplete(sender, message.payload || {})
    case MESSAGE_TYPES.REPORT_1688_PROCUREMENT_TASK_STATE:
      return report1688ProcurementTaskState(sender, message.payload || {})
    case MESSAGE_TYPES.REPORT_1688_PROCUREMENT_TASK_RESULT:
      return report1688ProcurementTaskResult(sender, message.payload || {})
    default:
      return createResponse(false, { message: '未知消息类型' })
  }
}

async function getSettings() {
  const stored = await chrome.storage.local.get(DEFAULT_SETTINGS)
  return normalizeSettings(stored)
}

async function saveSettings(settings) {
  const normalized = normalizeSettings(settings)
  await chrome.storage.local.set(normalized)
  return createResponse(true, { message: '配置已保存', data: normalized })
}

async function testConnection() {
  const settings = await getSettings()
  ensureSettings(settings)
  const { response, result, resolvedBaseUrl } = await requestJSON(settings.apiBaseUrl, '/amazon1688Collector/list', {
    method: 'POST',
    headers: buildHeaders(settings.apiToken),
    body: JSON.stringify({
      page: 1,
      pageSize: 1
    })
  })
  if (!response.ok || result?.code !== 0) {
    throw new Error(result?.msg || `连接测试失败 (${response.status})`)
  }
  return createResponse(true, {
    message: `连接测试成功，当前使用 ${resolvedBaseUrl}`,
    data: { resolvedBaseUrl }
  })
}

async function inspectActiveTab(sender) {
  const tab = await resolveTargetTab(sender)
  if (!tab?.id || !tab?.url) {
    return createResponse(false, { message: '未找到当前标签页' })
  }

  if (isSupportedAmazonHost(tab.url)) {
    if (!isAmazonDetailUrl(tab.url)) {
      return createResponse(false, {
        message: '当前页暂未自动识别为 Amazon 详情页，可尝试手动采集',
        canManualCollect: true
      })
    }
    const payload = await executeAmazonExtractor(tab.id, tab.url)
    return createResponse(true, {
      data: {
        pageType: 'amazon-detail',
        payload
      }
    })
  }

  if (isSupported1688Host(tab.url)) {
    if (is1688SearchUrl(tab.url)) {
      const taskContext = await resolve1688TaskContext({
        tabId: tab.id,
        url: tab.url
      })
      return createResponse(true, {
        data: {
          pageType: '1688-search',
          taskContext
        }
      })
    }
    if (is1688DetailUrl(tab.url)) {
      const taskContext = await resolve1688TaskContext({
        tabId: tab.id,
        url: tab.url
      })
      let payload = null
      try {
        payload = await execute1688Extractor(tab.id, tab.url)
      } catch (error) {
        payload = null
      }
      return createResponse(true, {
        data: {
          pageType: '1688-detail',
          taskContext,
          payload
        }
      })
    }
    return createResponse(false, { message: '当前页不是支持的 1688 图搜或详情页' })
  }

  return createResponse(false, { message: '当前页不是支持的 Amazon / 1688 页面' })
}

async function collectActiveTab(sender, payload = {}) {
  const settings = await getSettings()
  ensureSettings(settings)

  const tab = await resolveTargetTab(sender)
  if (!tab?.id || !tab?.url) {
    throw new Error('未找到当前标签页')
  }

  if (isSupportedAmazonHost(tab.url)) {
    if (!isAmazonDetailUrl(tab.url)) {
      throw new Error('当前页面不是 Amazon 详情页')
    }
    const detailPayload = await executeAmazonExtractor(tab.id, tab.url)
    const { response, result } = await requestJSON(settings.apiBaseUrl, '/amazonCollector/extension/upsertDetail', {
      method: 'POST',
      headers: buildHeaders(settings.apiToken),
      body: JSON.stringify(detailPayload)
    })
    if (!response.ok || result?.code !== 0) {
      throw new Error(result?.msg || `采集失败 (${response.status})`)
    }
    return createResponse(true, {
      message: '商品已采集到系统',
      payload: detailPayload,
      data: result?.data
    })
  }

  if (is1688DetailUrl(tab.url)) {
    const taskContext = await resolve1688TaskContext({
      tabId: tab.id,
      url: tab.url,
      preferToken: payload.taskToken
    })
    if (!taskContext?.taskToken) {
      throw new Error('未找到有效采集任务，请从系统重新打开 1688 图搜')
    }
    let detailPayload = null
    try {
      detailPayload = await execute1688Extractor(tab.id, tab.url)
      detailPayload.taskToken = taskContext.taskToken
      const { response, result } = await requestJSON(settings.apiBaseUrl, '/amazon1688Collector/extension/upsertDetail', {
        method: 'POST',
        headers: buildHeaders(settings.apiToken),
        body: JSON.stringify(detailPayload)
      })
      if (!response.ok || result?.code !== 0) {
        throw new Error(result?.msg || `采集失败 (${response.status})`)
      }

      const mergedContext = {
        ...taskContext,
        ...(result?.data || {}),
        taskToken: taskContext.taskToken,
        status: 'success'
      }
      await store1688TaskContext(mergedContext, tab.id)
      return createResponse(true, {
        message: `货物已采集到系统：${result?.data?.systemCode || taskContext.systemCode || '--'}`,
        payload: detailPayload,
        data: result?.data
      })
    } catch (error) {
      await safeReport1688TaskState(settings, taskContext.taskToken, 'failed', error.message || '采集失败')
      throw error
    }
  }

  throw new Error('当前页不是可采集的 Amazon / 1688 详情页')
}

async function set1688TaskContext(sender, payload = {}) {
  const tab = await resolveTargetTab(sender)
  const taskToken = String(payload.taskToken || extract1688TaskToken(tab?.url || '')).trim()
  if (!taskToken) {
    return createResponse(false, { message: '未找到 1688 采集任务 token' })
  }

  let context = {
    ...(await getStored1688TaskContext(taskToken)),
    taskToken,
    tabId: tab?.id || 0,
    lastSeenAt: new Date().toISOString()
  }

  const settings = await getSettings()
  if (settings.apiBaseUrl && settings.apiToken && payload.pageType === 'search') {
    try {
      const reported = await report1688TaskState(settings, taskToken, 'search_opened', '')
      context = {
        ...context,
        ...(reported || {})
      }
    } catch (error) {
      context = {
        ...context,
        status: context.status || 'pending'
      }
    }
  }

  await store1688TaskContext(context, tab?.id)
  return createResponse(true, { data: context })
}

async function get1688TaskContext(sender, payload = {}) {
  const tab = await resolveTargetTab(sender)
  const context = await resolve1688TaskContext({
    tabId: tab?.id,
    url: tab?.url,
    preferToken: payload.taskToken
  })
  if (!context) {
    return createResponse(false, { message: '未找到 1688 采集任务上下文' })
  }
  return createResponse(true, { data: context })
}

async function resolve1688TaskContext({ tabId, url, preferToken } = {}) {
  const directToken = String(preferToken || extract1688TaskToken(url || '')).trim()
  if (directToken) {
    const directContext = await getStored1688TaskContext(directToken)
    if (directContext) {
      return directContext
    }
    const fallback = { taskToken: directToken }
    if (tabId) {
      await store1688TaskContext(fallback, tabId)
    }
    return fallback
  }

  if (tabId) {
    const tabState = await chrome.storage.session.get(build1688TabKey(tabId))
    const tabContext = tabState?.[build1688TabKey(tabId)] || null
    if (tabContext?.taskToken) {
      return tabContext
    }
  }

  const lastState = await chrome.storage.session.get(STORAGE_KEYS.last1688TaskToken)
  const lastToken = String(lastState?.[STORAGE_KEYS.last1688TaskToken] || '').trim()
  if (!lastToken) {
    return null
  }
  return getStored1688TaskContext(lastToken)
}

async function getStored1688TaskContext(taskToken) {
  const token = String(taskToken || '').trim()
  if (!token) {
    return null
  }
  const state = await chrome.storage.session.get(build1688TaskKey(token))
  return state?.[build1688TaskKey(token)] || null
}

async function get1688ProcurementTask(sender, payload = {}) {
  const settings = await getSettings()
  ensureSettings(settings)
  const tab = await resolveTargetTab(sender)
  const taskToken = String(payload.taskToken || extract1688ProcurementTaskToken(tab?.url || '')).trim()
  if (!taskToken) {
    return createResponse(false, { message: '未找到 1688 采购任务 token' })
  }
  const stored = (await getStored1688ProcurementTask(taskToken)) || {}
  const { response, result } = await requestJSON(settings.apiBaseUrl, '/amazon1688Procurement/task/find', {
    method: 'GET',
    headers: buildHeaders(settings.apiToken)
  }, {
    taskToken
  })
  if (!response.ok || result?.code !== 0) {
    throw new Error(result?.msg || `查询采购任务失败 (${response.status})`)
  }
  const merged = {
    ...stored,
    taskToken,
    task: result?.data || null,
    completedGroupItemIds: stored.completedGroupItemIds || [],
    itemResults: stored.itemResults || {},
    lastLoadedAt: new Date().toISOString()
  }
  if (!merged.openedReported) {
    await report1688ProcurementTaskState(sender, {
      taskToken,
      status: 'opened',
      silent: true
    })
    merged.openedReported = true
  }
  await store1688ProcurementTask(merged)
  return createResponse(true, { data: merged })
}

async function mark1688ProcurementItemComplete(sender, payload = {}) {
  const tab = await resolveTargetTab(sender)
  const taskToken = String(payload.taskToken || extract1688ProcurementTaskToken(tab?.url || '')).trim()
  const groupItemId = Number(payload.groupItemId || 0)
  if (!taskToken || !groupItemId) {
    return createResponse(false, { message: '缺少采购任务标识或订单项' })
  }
  const stored = (await getStored1688ProcurementTask(taskToken)) || {}
  const completedSet = new Set(stored.completedGroupItemIds || [])
  completedSet.add(groupItemId)
  const itemResults = {
    ...(stored.itemResults || {}),
    [groupItemId]: {
      groupItemId,
      collectedProductId: Number(payload.collectedProductId || 0),
      selected1688SkuKey: String(payload.selected1688SkuKey || '').trim(),
      selected1688SkuAttrs: payload.selected1688SkuAttrs || {},
      purchaseQuantity: Number(payload.purchaseQuantity || 0)
    }
  }
  const nextItem = (stored.task?.items || []).find((item) => !completedSet.has(item.groupItemId)) || null
  const merged = {
    ...stored,
    completedGroupItemIds: Array.from(completedSet),
    itemResults,
    lastCompletedAt: new Date().toISOString()
  }
  await store1688ProcurementTask(merged)
  return createResponse(true, {
    data: {
      ...merged,
      nextItem,
      allCompleted: !nextItem
    }
  })
}

async function report1688ProcurementTaskState(sender, payload = {}) {
  const settings = await getSettings()
  ensureSettings(settings)
  const tab = await resolveTargetTab(sender)
  const taskToken = String(payload.taskToken || extract1688ProcurementTaskToken(tab?.url || '')).trim()
  if (!taskToken) {
    return createResponse(false, { message: '未找到 1688 采购任务 token' })
  }
  const { response, result } = await requestJSON(settings.apiBaseUrl, '/amazon1688Procurement/task/reportState', {
    method: 'POST',
    headers: buildHeaders(settings.apiToken),
    body: JSON.stringify({
      taskToken,
      status: String(payload.status || '').trim(),
      errorMessage: String(payload.errorMessage || '').trim()
    })
  })
  if (!response.ok || result?.code !== 0) {
    throw new Error(result?.msg || `上报采购任务状态失败 (${response.status})`)
  }
  if (payload.silent) {
    return createResponse(true, { data: result?.data || null })
  }
  return createResponse(true, { message: '采购任务状态已更新', data: result?.data || null })
}

async function report1688ProcurementTaskResult(sender, payload = {}) {
  const settings = await getSettings()
  ensureSettings(settings)
  const tab = await resolveTargetTab(sender)
  const taskToken = String(payload.taskToken || extract1688ProcurementTaskToken(tab?.url || '')).trim()
  if (!taskToken) {
    return createResponse(false, { message: '未找到 1688 采购任务 token' })
  }
  const stored = (await getStored1688ProcurementTask(taskToken)) || {}
  const requestPayload = {
    taskToken,
    status: String(payload.status || 'success').trim(),
    orderNo1688: String(payload.orderNo1688 || '').trim(),
    orderUrl: String(payload.orderUrl || tab?.url || '').trim(),
    errorMessage: String(payload.errorMessage || '').trim(),
    items: Object.values(stored.itemResults || {})
  }
  const { response, result } = await requestJSON(settings.apiBaseUrl, '/amazon1688Procurement/extension/reportResult', {
    method: 'POST',
    headers: buildHeaders(settings.apiToken),
    body: JSON.stringify(requestPayload)
  })
  if (!response.ok || result?.code !== 0) {
    throw new Error(result?.msg || `回传采购结果失败 (${response.status})`)
  }
  await store1688ProcurementTask({
    ...stored,
    taskToken,
    reportedAt: new Date().toISOString(),
    result: result?.data || null
  })
  return createResponse(true, { message: '采购结果已回传', data: result?.data || null })
}

async function store1688TaskContext(context, tabId) {
  if (!context?.taskToken) {
    return
  }
  const storagePayload = {
    [build1688TaskKey(context.taskToken)]: context,
    [STORAGE_KEYS.last1688TaskToken]: context.taskToken
  }
  if (tabId) {
    storagePayload[build1688TabKey(tabId)] = context
  }
  await chrome.storage.session.set(storagePayload)
}

async function report1688TaskState(settings, taskToken, status, errorMessage) {
  const { response, result } = await requestJSON(settings.apiBaseUrl, '/amazon1688Collector/task/reportState', {
    method: 'POST',
    headers: buildHeaders(settings.apiToken),
    body: JSON.stringify({
      taskToken,
      status,
      errorMessage
    })
  })
  if (!response.ok || result?.code !== 0) {
    throw new Error(result?.msg || `任务状态上报失败 (${response.status})`)
  }
  return result?.data || null
}

async function safeReport1688TaskState(settings, taskToken, status, errorMessage) {
  try {
    const data = await report1688TaskState(settings, taskToken, status, errorMessage)
    if (data?.taskToken) {
      await store1688TaskContext(data)
    }
  } catch (error) {
    return null
  }
  return null
}

async function executeAmazonExtractor(tabId, tabUrl) {
  await chrome.scripting.executeScript({
    target: { tabId },
    files: ['extractors/amazon-detail.js']
  })

  const [result] = await chrome.scripting.executeScript({
    target: { tabId },
    func: () => {
      if (typeof globalThis.__amazonCollectorExtractDetail !== 'function') {
        return { __collectorError: '提取器未加载成功' }
      }
      try {
        return globalThis.__amazonCollectorExtractDetail()
      } catch (error) {
        return { __collectorError: error?.message || '页面提取失败' }
      }
    }
  })

  if (result?.result?.__collectorError) {
    throw new Error(result.result.__collectorError)
  }

  const payload = result?.result
  if (!payload?.asin || !payload?.siteCode) {
    throw new Error('当前页面未提取到有效 ASIN 或站点信息')
  }
  return payload
}

async function execute1688Extractor(tabId, tabUrl) {
  await chrome.scripting.executeScript({
    target: { tabId },
    files: ['extractors/alibaba-1688-detail.js'],
    world: 'MAIN'
  })

  const [result] = await chrome.scripting.executeScript({
    target: { tabId },
    world: 'MAIN',
    func: () => {
      if (typeof globalThis.__amazonCollectorExtract1688Detail !== 'function') {
        return { __collectorError: '1688 提取器未加载成功' }
      }
      try {
        return globalThis.__amazonCollectorExtract1688Detail()
      } catch (error) {
        return { __collectorError: error?.message || '1688 页面提取失败' }
      }
    }
  })

  if (result?.result?.__collectorError) {
    throw new Error(result.result.__collectorError)
  }

  const payload = result?.result
  if (!payload?.offerId || !payload?.title) {
    throw new Error('当前页面未提取到有效 offerId 或标题')
  }
  if (!payload?.mainImageUrl) {
    throw new Error('当前页面未提取到真实商品主图，请等待详情页图片加载完成后重试')
  }
  if (!Array.isArray(payload?.skuOffers) || !payload.skuOffers.length) {
    throw new Error('当前页面未提取到 SKU 报价，请等待规格价格库存加载完成后重试')
  }
  payload.productUrl = payload.productUrl || tabUrl
  return payload
}

async function fetchImageAsDataUrl(payload = {}) {
  const imageUrl = String(payload.imageUrl || '').trim()
  if (!/^https?:\/\//i.test(imageUrl)) {
    throw new Error('图片链接无效，无法触发 1688 以图搜索')
  }

  const response = await fetch(imageUrl, {
    credentials: 'omit',
    headers: {
      Accept: 'image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8'
    }
  })
  if (!response.ok) {
    throw new Error(`图片下载失败 (${response.status})，无法触发 1688 以图搜索`)
  }

  const contentType = String(response.headers.get('content-type') || '').split(';')[0].trim().toLowerCase()
  if (contentType && !contentType.startsWith('image/')) {
    throw new Error(`图片地址返回了非图片内容：${contentType}`)
  }

  const blob = await response.blob()
  if (!blob.size) {
    throw new Error('图片内容为空，无法触发 1688 以图搜索')
  }
  if (blob.size > 10 * 1024 * 1024) {
    throw new Error('图片超过 10MB，无法触发 1688 以图搜索')
  }

  const resolvedType = blob.type || contentType || 'image/jpeg'
  const arrayBuffer = await blob.arrayBuffer()
  return createResponse(true, {
    data: {
      imageUrl,
      dataUrl: `data:${resolvedType};base64,${arrayBufferToBase64(arrayBuffer)}`,
      contentType: resolvedType,
      fileName: buildImageFileName(imageUrl, resolvedType),
      size: blob.size
    }
  })
}

async function resolveTargetTab(sender) {
  if (sender?.tab?.id) {
    return sender.tab
  }
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
  return tab
}

function arrayBufferToBase64(buffer) {
  const bytes = new Uint8Array(buffer)
  const chunkSize = 0x8000
  let binary = ''
  for (let i = 0; i < bytes.length; i += chunkSize) {
    const chunk = bytes.subarray(i, i + chunkSize)
    binary += String.fromCharCode(...chunk)
  }
  return btoa(binary)
}

function buildImageFileName(rawUrl, contentType) {
  const extByType = {
    'image/jpeg': 'jpg',
    'image/jpg': 'jpg',
    'image/png': 'png',
    'image/webp': 'webp',
    'image/gif': 'gif',
    'image/avif': 'avif',
    'image/svg+xml': 'svg'
  }
  try {
    const pathname = new URL(rawUrl).pathname
    const fileName = pathname.split('/').filter(Boolean).pop() || ''
    if (/\.(jpe?g|png|webp|gif|avif|svg)$/i.test(fileName)) {
      return fileName
    }
  } catch (error) {
    // Keep the fallback below when the source URL is unusual.
  }
  return `1688-image-search.${extByType[contentType] || 'jpg'}`
}

function buildHeaders(apiToken) {
  return {
    'Content-Type': 'application/json',
    'x-token': apiToken
  }
}

function ensureSettings(settings) {
  if (!settings.apiBaseUrl) {
    throw new Error('请先在插件设置中填写后台 API Base URL')
  }
  if (!settings.apiToken) {
    throw new Error('请先在插件设置中填写 API Token')
  }
}

function build1688TaskKey(taskToken) {
  return `collector1688:task:${String(taskToken || '').trim()}`
}

function build1688TabKey(tabId) {
  return `collector1688:tab:${tabId}`
}

function build1688ProcurementTaskKey(taskToken) {
  return `collector1688:procurement:${String(taskToken || '').trim()}`
}

async function getStored1688ProcurementTask(taskToken) {
  const token = String(taskToken || '').trim()
  if (!token) {
    return null
  }
  const state = await chrome.storage.session.get(build1688ProcurementTaskKey(token))
  return state?.[build1688ProcurementTaskKey(token)] || null
}

async function store1688ProcurementTask(context) {
  if (!context?.taskToken) {
    return
  }
  await chrome.storage.session.set({
    [build1688ProcurementTaskKey(context.taskToken)]: context,
    [STORAGE_KEYS.last1688ProcurementTaskToken]: context.taskToken
  })
}

async function parseJSON(response) {
  try {
    return await response.json()
  } catch (error) {
    return null
  }
}

async function requestJSON(baseUrl, apiPath, options, query) {
  const candidates = buildBaseUrlCandidates(baseUrl)
  let lastResponse = null
  let lastResult = null
  let lastError = null

  for (const candidate of candidates) {
    try {
      const response = await fetch(withQuery(buildApiUrl(candidate, apiPath), query), options)
      const result = await parseJSON(response)
      if (response.status === 404 && candidates.length > 1) {
        lastResponse = response
        lastResult = result
        continue
      }
      return {
        response,
        result,
        resolvedBaseUrl: candidate
      }
    } catch (error) {
      lastError = error
    }
  }

  if (lastError) {
    throw lastError
  }
  if (lastResponse) {
    return {
      response: lastResponse,
      result: lastResult,
      resolvedBaseUrl: candidates[candidates.length - 1]
    }
  }
  throw new Error('请求未发出')
}

function withQuery(rawUrl, query) {
  if (!query || typeof query !== 'object' || Array.isArray(query)) {
    return rawUrl
  }
  try {
    const url = new URL(rawUrl)
    Object.entries(query).forEach(([key, value]) => {
      if (value === null || typeof value === 'undefined' || value === '') {
        return
      }
      url.searchParams.set(key, String(value))
    })
    return url.toString()
  } catch (error) {
    return rawUrl
  }
}

function buildBaseUrlCandidates(baseUrl) {
  const normalized = String(baseUrl || '').trim().replace(/\/+$/, '')
  if (!normalized) {
    return []
  }
  const candidates = [normalized]
  if (!/\/api$/i.test(normalized)) {
    candidates.push(`${normalized}/api`)
  }
  return [...new Set(candidates)]
}
