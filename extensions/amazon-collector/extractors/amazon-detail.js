(function () {
  const SITE_MAP = {
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

  globalThis.__amazonCollectorExtractDetail = function () {
    const currentUrl = window.location.href
    const siteMeta = SITE_MAP[window.location.host]
    if (!siteMeta) {
      throw new Error('当前站点不在支持范围内')
    }
    if (!/(?:^|\/)(?:dp|gp\/product|gp\/aw\/d)\/[A-Z0-9]{10}(?:[/?#]|$)/i.test(window.location.pathname)) {
      throw new Error('当前页面不是 Amazon 详情页')
    }

    const scriptText = collectScriptText()
    const mainImageUrl = findImageUrlFromScripts(scriptText) || pickText([
      '#landingImage',
      '#imgTagWrapperId img',
      '#main-image-container img'
    ], 'src')

    const galleryImageUrls = uniqueStrings([
      ...findGalleryImageUrls(scriptText),
      ...selectAll('#altImages img').map((image) => image.getAttribute('src') || image.getAttribute('data-old-hires')),
      ...selectAll('#main-image-container img').map((image) => image.getAttribute('src'))
    ])

    const title = cleanText(pickText([
      '#productTitle',
      '#title',
      '[data-cy="title-recipe"] h1'
    ]))
    const asin = extractASIN(scriptText) || extractASINFromPath() || cleanText(findDetailBulletValue('ASIN'))
    const priceText = cleanText(pickText([
      '#corePrice_feature_div .a-offscreen',
      '#price_inside_buybox',
      '#newAccordionRow_1 .a-color-price',
      '#tp_price_block_total_price_ww .a-offscreen'
    ]))
    const listPriceText = cleanText(pickText([
      '#corePriceDisplay_desktop_feature_div .basisPrice .a-offscreen',
      '#corePrice_feature_div .priceToPay .a-text-price .a-offscreen',
      '#corePrice_feature_div .a-price.a-text-price .a-offscreen'
    ]))
    const ratingText = cleanText(pickText([
      '#acrPopover',
      '#averageCustomerReviews span.a-icon-alt',
      '[data-hook="rating-out-of-text"]'
    ], 'title'))
    const reviewCountText = cleanText(pickText([
      '#acrCustomerReviewText',
      '#reviewsMedley #acrCustomerReviewText',
      '[data-hook="total-review-count"]'
    ]))

    const specAttributes = collectSpecificationTable()
    const categoryPath = collectCategoryPath()
    const browseNodes = collectBrowseNodes(scriptText, categoryPath)
    const bulletPoints = uniqueStrings(selectAll('#feature-bullets li').map((item) => cleanText(item.textContent)))
    const descriptionText = cleanText(pickText([
      '#productDescription',
      '#bookDescription_feature_div',
      '#feature-bullets'
    ]))
    const sellerName = cleanText(pickText([
      '#sellerProfileTriggerId',
      '#merchant-info a',
      '#offerDisplayFeatureText a',
      '#shipsFromSoldBy_feature_div a'
    ]))

    const payload = {
      siteCode: siteMeta.siteCode,
      marketplaceId: siteMeta.marketplaceId,
      asin,
      parentAsin: extractParentASIN(scriptText),
      title,
      brand: cleanText(pickText([
        '#bylineInfo',
        '#brand',
        'a#bylineInfo'
      ])).replace(/^Brand:\s*/i, ''),
      productUrl: currentUrl,
      priceAmount: parsePriceAmount(priceText),
      currencyCode: parseCurrencyCode(priceText) || siteCurrency(siteMeta.siteCode),
      listPriceAmount: parsePriceAmount(listPriceText),
      discountText: cleanText(pickText([
        '#dealBadgeSupportingText',
        '#savingsPercentage',
        '#corePrice_feature_div .savingsPercentage'
      ])),
      ratingValue: parseRatingValue(ratingText),
      reviewCount: parseInteger(reviewCountText),
      bsrText: extractBSRText(),
      categoryPath,
      browseNodes,
      sellerName,
      fulfillmentChannel: inferFulfillmentChannel(),
      deliveryEstimateText: extractDeliveryEstimateText(),
      bulletPoints,
      descriptionText,
      aplusHtml: extractAplusHTML(),
      specAttributes,
      variantSummary: collectVariantSummary(),
      mainImageUrl,
      galleryImageUrls,
      collectWarnings: collectWarnings({
        asin,
        title,
        mainImageUrl,
        priceText,
        bulletPoints,
        specAttributes
      }),
      rawPayload: {
        currentUrl,
        host: window.location.host,
        extractedAt: new Date().toISOString(),
        priceText,
        listPriceText,
        ratingText,
        reviewCountText,
        specAttributes,
        variantSummary: collectVariantSummary(),
        categoryPath,
        browseNodes,
        deliveryEstimateText: extractDeliveryEstimateText(),
        detailBullets: collectDetailBulletsMap(),
        scriptHints: extractScriptHints(scriptText)
      }
    }

    payload.images = normalizeImages(payload.mainImageUrl, payload.galleryImageUrls)
    return payload
  }

  function collectScriptText() {
    return selectAll('script')
      .map((item) => item.textContent || '')
      .join('\n')
  }

  function pickText(selectors, attribute) {
    for (const selector of selectors) {
      const node = document.querySelector(selector)
      if (!node) {
        continue
      }
      if (attribute) {
        const value = node.getAttribute(attribute)
        if (cleanText(value)) {
          return value
        }
      }
      const textValue = extractSanitizedText(node)
      if (textValue) {
        return textValue
      }
    }
    return ''
  }

  function selectAll(selector) {
    return Array.from(document.querySelectorAll(selector))
  }

  function cleanText(value) {
    return String(value || '').replace(/\s+/g, ' ').trim()
  }

  function extractSanitizedText(node) {
    if (!node) {
      return ''
    }
    const cloned = node.cloneNode(true)
    cloned.querySelectorAll('script, style, noscript, template').forEach((item) => item.remove())
    cloned.querySelectorAll('[aria-hidden="true"], .aok-hidden, .a-hidden').forEach((item) => item.remove())
    return cleanText(cloned.textContent)
  }

  function uniqueStrings(values) {
    const seen = new Set()
    return values
      .map((value) => cleanText(value))
      .filter((value) => value)
      .filter((value) => {
        if (seen.has(value)) {
          return false
        }
        seen.add(value)
        return true
      })
  }

  function extractASINFromPath() {
    const match = window.location.pathname.match(/(?:^|\/)(?:dp|gp\/product|gp\/aw\/d)\/([A-Z0-9]{10})(?:[/?#]|$)/i)
    return match ? match[1].toUpperCase() : ''
  }

  function extractASIN(scriptText) {
    return findScriptValue(scriptText, /"asin"\s*:\s*"([A-Z0-9]{10})"/i) ||
      findScriptValue(scriptText, /asin\s*=\s*"([A-Z0-9]{10})"/i)
  }

  function extractParentASIN(scriptText) {
    return findScriptValue(scriptText, /"parentAsin"\s*:\s*"([A-Z0-9]{10})"/i) ||
      findScriptValue(scriptText, /"parentASIN"\s*:\s*"([A-Z0-9]{10})"/i)
  }

  function findScriptValue(scriptText, pattern) {
    const match = scriptText.match(pattern)
    return match ? match[1] : ''
  }

  function parsePriceAmount(priceText) {
    const match = String(priceText || '').replace(/,/g, '').match(/(\d+(?:\.\d+)?)/)
    return match ? Number(match[1]) : null
  }

  function parseCurrencyCode(priceText) {
    const value = String(priceText || '')
    if (value.includes('$')) {
      if (window.location.host === 'www.amazon.ca') {
        return 'CAD'
      }
      if (window.location.host === 'www.amazon.com.mx') {
        return 'MXN'
      }
      return 'USD'
    }
    if (value.includes('MX$')) {
      return 'MXN'
    }
    return ''
  }

  function siteCurrency(siteCode) {
    switch (siteCode) {
      case 'CA':
        return 'CAD'
      case 'MX':
        return 'MXN'
      default:
        return 'USD'
    }
  }

  function parseRatingValue(ratingText) {
    const match = String(ratingText || '').match(/(\d+(?:\.\d+)?)/)
    return match ? Number(match[1]) : null
  }

  function parseInteger(rawValue) {
    const digits = String(rawValue || '').replace(/[^\d]/g, '')
    return digits ? Number(digits) : null
  }

  function collectCategoryPath() {
    return uniqueStrings([
      ...selectAll('#wayfinding-breadcrumbs_feature_div li a').map((node) => node.textContent),
      ...selectAll('#wayfinding-breadcrumbs_container li a').map((node) => node.textContent),
      ...selectAll('#nav-subnav a[aria-current="page"], #desktop-grid-1 span.a-list-item a').map((node) => node.textContent)
    ])
  }

  function collectBrowseNodes(scriptText, categoryPath) {
    const result = []
    const seen = new Set()
    const push = (id, name) => {
      const normalizedId = cleanText(id)
      const normalizedName = cleanText(name)
      const dedupeKey = `${normalizedId}|${normalizedName}`
      if ((!normalizedId && !normalizedName) || seen.has(dedupeKey)) {
        return
      }
      seen.add(dedupeKey)
      result.push({
        id: normalizedId,
        name: normalizedName
      })
    }

    selectAll('#wayfinding-breadcrumbs_feature_div li a, #wayfinding-breadcrumbs_container li a').forEach((node) => {
      const href = node.getAttribute('href') || ''
      const nodeId = href.match(/[?&]node=([0-9]+)/i)?.[1] || ''
      push(nodeId, node.textContent)
    })

    const scriptNodePattern = /"browseNodeId"\s*:\s*"?(\\?\d+)"?(?:[^{}]{0,160}?"displayName"\s*:\s*"([^"]+)")?/ig
    let scriptNodeMatch = scriptNodePattern.exec(scriptText)
    while (scriptNodeMatch) {
      push(scriptNodeMatch[1].replace(/\\/g, ''), scriptNodeMatch[2] || '')
      scriptNodeMatch = scriptNodePattern.exec(scriptText)
    }

    if (!result.length) {
      ;(categoryPath || []).forEach((name) => push('', name))
    }

    return result
  }

  function collectDetailBulletsMap() {
    const result = {}
    selectAll('#detailBullets_feature_div li, #detailBulletsWrapper_feature_div li').forEach((item) => {
      const label = extractSanitizedText(item.querySelector('.a-text-bold'))
      const value = cleanText(extractSanitizedText(item).replace(label, ''))
      if (label) {
        result[label.replace(/:$/, '')] = value
      }
    })
    return result
  }

  function findDetailBulletValue(labelName) {
    const detailMap = collectDetailBulletsMap()
    const entry = Object.entries(detailMap).find(([key]) => key.toLowerCase().includes(labelName.toLowerCase()))
    return entry ? entry[1] : ''
  }

  function collectSpecificationTable() {
    const result = {}
    const selectors = [
      '#productDetails_techSpec_section_1 tr',
      '#productDetails_detailBullets_sections1 tr',
      '#technicalSpecifications_section_1 tr',
      '#prodDetails tr'
    ]
    selectors.forEach((selector) => {
      selectAll(selector).forEach((row) => {
        const cells = row.querySelectorAll('th, td')
        if (cells.length < 2) {
          return
        }
        const key = extractSanitizedText(cells[0])
        const value = normalizeSpecificationValue(key, cells[cells.length - 1])
        if (key && value && !result[key]) {
          result[key] = value
        }
      })
    })
    return result
  }

  function normalizeSpecificationValue(key, cell) {
    if (!cell) {
      return ''
    }
    if (/customer reviews/i.test(key)) {
      return extractCustomerReviewText(cell)
    }
    return extractSanitizedText(cell)
  }

  function extractCustomerReviewText(cell) {
    const ratingText = uniqueStrings([
      cleanText(cell.querySelector('[data-hook="rating-out-of-text"]')?.textContent),
      cleanText(cell.querySelector('#acrPopover')?.getAttribute('title')),
      cleanText(cell.querySelector('.a-icon-alt')?.textContent)
    ])
      .sort((left, right) => right.length - left.length)[0] || ''
    const reviewCountText = cleanText(
      cell.querySelector('#acrCustomerReviewText, [data-hook="total-review-count"]')?.textContent
    )
    const fallback = compactRepeatedReviewText(extractSanitizedText(cell))
    if (ratingText && reviewCountText) {
      return `${ratingText} (${reviewCountText})`
    }
    return ratingText || reviewCountText || fallback
  }

  function compactRepeatedReviewText(value) {
    const text = cleanText(value)
      .replace(/\s+(?:var\s+[A-Za-z_$][\w$]*\s*;|P\.when\(|ue\.count\().*$/i, '')
      .trim()
    return text.replace(
      /^(\d+(?:\.\d+)?)\s+\1(\s+out of 5 stars(?:\s*\([^)]*\))?.*)$/i,
      '$1$2'
    )
  }

  function extractBSRText() {
    const detailMap = collectDetailBulletsMap()
    const candidate = Object.entries(detailMap).find(([key]) => /best sellers rank/i.test(key))
    if (candidate) {
      return candidate[1]
    }
    return cleanText(pickText([
      '#productDetails_detailBullets_sections1 tr td',
      '#detailBullets_feature_div'
    ]))
  }

  function inferFulfillmentChannel() {
    const merchantText = cleanText(document.querySelector('#merchant-info')?.textContent || '')
    const shipText = cleanText(document.querySelector('#shipsFromSoldBy_feature_div')?.textContent || '')
    const combined = `${merchantText} ${shipText}`.toLowerCase()
    if (combined.includes('fulfilled by amazon')) {
      return 'FBA'
    }
    if (combined.includes('ships from amazon') || combined.includes('sold by amazon')) {
      return 'AMAZON'
    }
    if (combined.includes('ships from') || combined.includes('sold by')) {
      return 'FBM'
    }
    return ''
  }

  function extractDeliveryEstimateText() {
    const selectors = [
      '#mir-layout-DELIVERY_BLOCK-slot-PRIMARY_DELIVERY_MESSAGE_LARGE',
      '#mir-layout-DELIVERY_BLOCK-slot-DELIVERY_MESSAGE',
      '#deliveryBlockMessage',
      '#deliveryBlock_feature_div',
      '#ddmDeliveryMessage',
      '#upsell-message',
      '[data-csa-c-delivery-price]'
    ]
    for (const selector of selectors) {
      const node = document.querySelector(selector)
      const value = extractSanitizedText(node)
      if (value) {
        return value
      }
    }
    return cleanText(document.querySelector('#mir-layout-DELIVERY_BLOCK')?.textContent || '')
  }

  function extractAplusHTML() {
    return document.querySelector('#aplus')?.innerHTML || ''
  }

  function collectVariantSummary() {
    const result = {}
    selectAll('#variation_color_name li img, #variation_style_name li img, #variation_size_name li').forEach((node) => {
      const label = cleanText(node.getAttribute('alt') || node.textContent)
      if (!label) {
        return
      }
      result.options = result.options || []
      if (!result.options.includes(label)) {
        result.options.push(label)
      }
    })

    const twisterLabels = selectAll('#twister .selection, #twisterPlus_feature_div .selection').map((node) => cleanText(node.textContent))
    if (twisterLabels.length) {
      result.selected = twisterLabels
    }
    return result
  }

  function findImageUrlFromScripts(scriptText) {
    const match = scriptText.match(/"mainUrl"\s*:\s*"([^"]+)"/i) ||
      scriptText.match(/"hiRes"\s*:\s*"([^"]+)"/i) ||
      scriptText.match(/"large"\s*:\s*"([^"]+)"/i)
    return normalizeImageUrl(match ? match[1] : '')
  }

  function findGalleryImageUrls(scriptText) {
    const results = []
    const pattern = /"hiRes"\s*:\s*"([^"]+)"/ig
    let match = pattern.exec(scriptText)
    while (match) {
      results.push(normalizeImageUrl(match[1]))
      match = pattern.exec(scriptText)
    }
    return results
  }

  function normalizeImageUrl(rawUrl) {
    const value = cleanText(rawUrl)
    if (!value) {
      return ''
    }
    return value
      .replace(/\\u0026/g, '&')
      .replace(/\\\//g, '/')
  }

  function normalizeImages(mainImageUrl, galleryImageUrls) {
    const result = []
    const seen = new Set()
    const push = (url, isMain = false) => {
      const normalized = normalizeImageUrl(url)
      if (!normalized || seen.has(normalized)) {
        return
      }
      seen.add(normalized)
      result.push({
        sort: result.length + 1,
        isMain,
        originalUrl: normalized
      })
    }

    push(mainImageUrl, true)
    galleryImageUrls.forEach((url) => push(url, false))
    if (!result.length) {
      return []
    }
    if (!result.some((item) => item.isMain)) {
      result[0].isMain = true
    }
    return result
  }

  function collectWarnings(payload) {
    const warnings = []
    if (!payload.asin) {
      warnings.push('未提取到 ASIN')
    }
    if (!payload.title) {
      warnings.push('未提取到标题')
    }
    if (!payload.mainImageUrl) {
      warnings.push('未提取到主图')
    }
    if (!payload.priceText) {
      warnings.push('未提取到价格')
    }
    if (!payload.bulletPoints?.length) {
      warnings.push('未提取到卖点')
    }
    if (!Object.keys(payload.specAttributes || {}).length) {
      warnings.push('未提取到属性参数')
    }
    return warnings
  }

  function extractScriptHints(scriptText) {
    return {
      asin: extractASIN(scriptText),
      parentAsin: extractParentASIN(scriptText),
      hasImageBlock: /ImageBlockATF/i.test(scriptText),
      hasTwister: /twister/i.test(scriptText)
    }
  }
})()
