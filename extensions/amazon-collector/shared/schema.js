export const AMAZON_SITE_MAP = {
  'www.amazon.com': {
    siteCode: 'US',
    marketplaceId: 'ATVPDKIKX0DER'
  },
  'www.amazon.ca': {
    siteCode: 'CA',
    marketplaceId: 'A2EUQ1WTGCTBG2'
  },
  'www.amazon.com.mx': {
    siteCode: 'MX',
    marketplaceId: 'A1AM78C64UM0Y8'
  }
}

export const SUPPORTED_1688_HOSTS = ['s.1688.com', 'detail.1688.com']
export const GVA_1688_TASK_PARAM = '__gva1688Task'
export const GVA_1688_PROCUREMENT_TASK_PARAM = '__gva1688ProcurementTask'

export const DEFAULT_SETTINGS = {
  apiBaseUrl: '',
  apiToken: ''
}

export const isSupportedAmazonHost = (rawUrl = '') => {
  try {
    const url = new URL(rawUrl)
    return Boolean(AMAZON_SITE_MAP[url.host])
  } catch (error) {
    return false
  }
}

export const AMAZON_DETAIL_PATH_PATTERN = /(?:^|\/)(?:dp|gp\/product|gp\/aw\/d)\/([A-Z0-9]{10})(?:[/?#]|$)/i
export const ALIBABA_1688_DETAIL_PATH_PATTERN = /(?:^|\/)offer\/(\d+)\.html(?:[/?#]|$)/i

export const isAmazonDetailUrl = (rawUrl = '') => {
  try {
    const url = new URL(rawUrl)
    return AMAZON_DETAIL_PATH_PATTERN.test(url.pathname)
  } catch (error) {
    return false
  }
}

export const isSupported1688Host = (rawUrl = '') => {
  try {
    const url = new URL(rawUrl)
    return SUPPORTED_1688_HOSTS.includes(url.host)
  } catch (error) {
    return false
  }
}

export const is1688SearchUrl = (rawUrl = '') => {
  try {
    const url = new URL(rawUrl)
    return url.host === 's.1688.com' && (
      /\/youyuan\/index\.htm$/i.test(url.pathname) ||
      /\/shen\/sell_offer\.html?$/i.test(url.pathname)
    )
  } catch (error) {
    return false
  }
}

export const is1688DetailUrl = (rawUrl = '') => {
  try {
    const url = new URL(rawUrl)
    return url.host === 'detail.1688.com' && ALIBABA_1688_DETAIL_PATH_PATTERN.test(url.pathname)
  } catch (error) {
    return false
  }
}

export const extract1688TaskToken = (rawUrl = '') => {
  try {
    const url = new URL(rawUrl)
    return String(url.searchParams.get(GVA_1688_TASK_PARAM) || '').trim()
  } catch (error) {
    return ''
  }
}

export const extract1688ProcurementTaskToken = (rawUrl = '') => {
  try {
    const url = new URL(rawUrl)
    return String(url.searchParams.get(GVA_1688_PROCUREMENT_TASK_PARAM) || '').trim()
  } catch (error) {
    return ''
  }
}

export const append1688TaskToken = (rawUrl = '', taskToken = '') => {
  const trimmedToken = String(taskToken || '').trim()
  if (!trimmedToken) {
    return rawUrl
  }
  try {
    const url = new URL(rawUrl, window.location.origin)
    url.searchParams.set(GVA_1688_TASK_PARAM, trimmedToken)
    return url.toString()
  } catch (error) {
    return rawUrl
  }
}

export const normalizeSettings = (settings = {}) => {
  return {
    apiBaseUrl: String(settings.apiBaseUrl || '').trim().replace(/\/+$/, ''),
    apiToken: String(settings.apiToken || '').trim()
  }
}

export const buildApiUrl = (baseUrl, path) => {
  const normalizedBase = String(baseUrl || '').trim().replace(/\/+$/, '')
  if (!normalizedBase) {
    return ''
  }
  return `${normalizedBase}${path}`
}

export const createResponse = (ok, data = {}) => ({ ok, ...data })
