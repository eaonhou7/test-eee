import { getAmazonFinanceFxList } from '@/api/amazonFinanceFx'

export const amazonManagedCurrencyOptions = [
  { value: 'USD', label: '美元' },
  { value: 'EUR', label: '欧元' },
  { value: 'JPY', label: '日元' },
  { value: 'GBP', label: '英镑' },
  { value: 'AUD', label: '澳元' },
  { value: 'CAD', label: '加元' },
  { value: 'MXN', label: '墨西哥比索' },
  { value: 'CHF', label: '瑞士法郎' },
  { value: 'HKD', label: '港币' },
  { value: 'SGD', label: '新加坡元' },
  { value: 'NZD', label: '新西兰元' }
]

export const amazonCurrencyOptions = [
  ...amazonManagedCurrencyOptions,
  { value: 'CNY', label: '人民币' }
]

export const amazonFinanceSiteOptions = [
  { label: 'US', value: 'US' },
  { label: 'CA', value: 'CA' },
  { label: 'MX', value: 'MX' },
  { label: 'UK', value: 'UK' },
  { label: 'DE', value: 'DE' }
]

export const amazonListingSiteOptions = amazonFinanceSiteOptions.filter((item) => ['US', 'CA', 'MX'].includes(item.value))

export const amazonSiteDefaults = {
  US: {
    label: '美国站',
    marketplaceId: 'ATVPDKIKX0DER',
    currencyCode: 'USD',
    defaultLocale: 'en_US'
  },
  CA: {
    label: '加拿大站',
    marketplaceId: 'A2EUQ1WTGCTBG2',
    currencyCode: 'CAD',
    defaultLocale: 'en_US'
  },
  MX: {
    label: '墨西哥站',
    marketplaceId: 'A1AM78C64UM0Y8',
    currencyCode: 'MXN',
    defaultLocale: 'es_MX'
  },
  UK: {
    label: '英国站',
    marketplaceId: '',
    currencyCode: 'GBP',
    defaultLocale: 'en_GB'
  },
  DE: {
    label: '德国站',
    marketplaceId: '',
    currencyCode: 'EUR',
    defaultLocale: 'de_DE'
  }
}

export const normalizeAmazonSiteCode = (value, fallback = 'US') => {
  const normalized = String(value || '').trim().toUpperCase()
  return normalized || fallback
}

export const normalizeAmazonCurrencyCode = (value, fallback = 'USD') => {
  const normalized = String(value || '').trim().toUpperCase()
  return normalized || fallback
}

export const getAmazonSiteDefault = (siteCode) => amazonSiteDefaults[normalizeAmazonSiteCode(siteCode)] || amazonSiteDefaults.US

export const getAmazonSiteLabel = (siteCode) => {
  const normalized = normalizeAmazonSiteCode(siteCode, '')
  return amazonSiteDefaults[normalized]?.label || normalized || '站点'
}

export const getAmazonSiteCurrencyCode = (siteCode) => getAmazonSiteDefault(siteCode).currencyCode

export const getAmazonSiteMarketplaceId = (siteCode) => getAmazonSiteDefault(siteCode).marketplaceId

export const getAmazonSiteDefaultLocale = (siteCode) => getAmazonSiteDefault(siteCode).defaultLocale || 'en_US'

export const getAmazonCurrencyLabel = (currencyCode) => {
  const normalized = normalizeAmazonCurrencyCode(currencyCode, '')
  return amazonCurrencyOptions.find((item) => item.value === normalized)?.label || normalized
}

export const todayString = () => {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export const fetchLatestAmazonFxRate = async (currencyCode) => {
  const normalized = normalizeAmazonCurrencyCode(currencyCode)
  if (normalized === 'CNY') {
    return {
      currencyCode: 'CNY',
      rateToCny: 1,
      rateDate: todayString(),
      source: 'fixed'
    }
  }
  let res
  try {
    res = await getAmazonFinanceFxList({
      page: 1,
      pageSize: 1,
      currencyCode: normalized,
      dateTo: todayString()
    })
  } catch {
    return null
  }
  if (res.code !== 0) {
    return null
  }
  const row = (res.data?.list || []).find((item) => normalizeAmazonCurrencyCode(item.currencyCode) === normalized)
  if (!row || Number(row.rateToCny || 0) <= 0) {
    return null
  }
  return row
}

export const fetchLatestAmazonFxRateToCny = async (currencyCode) => {
  const row = await fetchLatestAmazonFxRate(currencyCode)
  const rate = Number(row?.rateToCny || 0)
  return rate > 0 ? rate : undefined
}

export const applyAmazonCurrencyFxRate = async (target, currencyCode, options = {}) => {
  if (!target) {
    return undefined
  }
  const currencyField = options.currencyField || 'currencyCode'
  const rateField = options.rateField || 'fxRateToCny'
  const normalized = normalizeAmazonCurrencyCode(currencyCode)
  target[currencyField] = normalized
  const rate = await fetchLatestAmazonFxRateToCny(normalized)
  target[rateField] = rate
  return rate
}

export const applyAmazonSiteCurrencyAndFx = async (target, siteCode, options = {}) => {
  if (!target) {
    return undefined
  }
  const siteField = options.siteField || 'siteCode'
  const normalizedSite = normalizeAmazonSiteCode(siteCode || target[siteField])
  target[siteField] = normalizedSite
  return applyAmazonCurrencyFxRate(target, getAmazonSiteCurrencyCode(normalizedSite), options)
}
