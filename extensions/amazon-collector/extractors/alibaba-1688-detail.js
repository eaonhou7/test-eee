(function () {
  const DETAIL_PATH_PATTERN = /(?:^|\/)offer\/(\d+)\.html(?:[/?#]|$)/i
  const IMAGE_REJECT_PATTERN = /(logo|icon|sprite|loading|placeholder|avatar|qrcode|qr-code|wangwang|chat|service|toolbar|ai-assistant|assistant|empty|default)/i
  const PRODUCT_IMAGE_PATTERN = /(cbu01|cbuimg|imgextra|ibank|uploaded|alicdn\.com\/img|alicdn\.com\/bao)/i
  const IMAGE_CONTEXT_REJECT_PATTERN = /(toolbar|side|sidebar|service|chat|wangwang|qr|qrcode|shop-card|seller-card|company-card|ai-assistant|float|fixed|collect|favorite|recommend|similar|guess|广告|推荐|相似|看了又看|店铺|公司|客服|旺旺|二维码)/i
  const IMAGE_CONTEXT_DETAIL_PATTERN = /(detail|description|desc|商品详情|详情|描述|参数详情)/i
  const IMAGE_CONTEXT_SKU_PATTERN = /(sku|spec|sale-prop|saleprop|颜色|规格|款式|型号|尺寸|尺码)/i
  const IMAGE_CONTEXT_GALLERY_PATTERN = /(main|primary|preview|gallery|album|carousel|swipe|thumb|thumbnail|image-view|imageview|主图|首图|预览|相册|轮播|图集)/i
  const SKU_ROW_PRICE_PATTERN = /[￥¥]\s*\d+(?:\.\d+)?/
  const SKU_ROW_STOCK_PATTERN = /(库存|可售|现货|有货)\s*\d+/

  globalThis.__amazonCollectorExtract1688Detail = function () {
    if (window.location.host !== 'detail.1688.com' || !DETAIL_PATH_PATTERN.test(window.location.pathname)) {
      throw new Error('当前页面不是 1688 详情页')
    }

    const candidateRoots = collectCandidateRoots()
    const structuredData = collectStructuredDataRoots()
    const roots = [...candidateRoots, ...structuredData]

    const offerId = extractOfferIdFromPath() || firstStringValue(findValuesByKeys(roots, ['offerid', 'offerId', 'offerNum']))
    const title = cleanText(
      pickText([
        'h1',
        '[class*="title"] h1',
        '[class*="subject"]',
        '[data-title]'
      ]) ||
      firstStringValue(findValuesByKeys(roots, ['subject', 'title', 'offerTitle', 'name']))
    )

    const productImages = extractProductImages(roots)
    const mainImageUrl = productImages.mainImageUrl
    const galleryImageUrls = productImages.galleryImageUrls.filter((value) => value && value !== mainImageUrl)
    const detailImageUrls = productImages.detailImageUrls.filter(Boolean)

    const priceText = cleanText(
      pickText([
        '[class*="price"]',
        '[data-price]',
        '.price',
        '.cost-price'
      ]) ||
      firstStringValue(findValuesByKeys(roots, ['priceDisplay', 'priceText', 'priceRange', 'displayPrice']))
    )

    const specAttributes = normalizeAttributeMap(pickBestObject(findValuesByKeys(roots, ['specAttr', 'specAttrs', 'specifications', 'attributes', 'props'])))
    const productAttributes = extractProductAttributes(roots, specAttributes)
    const skuAttributes = extractSkuAttributes(roots)
    const skuOffers = extractSkuOffers(roots, skuAttributes)
    const sellerMetrics = extractSellerMetrics()
    const serviceLabels = extractServiceLabels()
    const logisticsInfo = extractLogisticsInfo(roots)
    const salesStats = extractSalesStats(roots)
    const tierPrices = extractTierPrices(roots)
    const videoUrls = extractVideoUrls()
    const packageInfo = extractPackageInfo(roots)
    const detailSections = extractDetailSections(roots)
    const detailText = cleanText(detailSections.map((section) => section.text).filter(Boolean).join('\n'))
    const descriptionHtml = extractDescriptionHtml(detailSections)
    const categoryPath = uniqueStrings([
      ...findArrayStrings(findValuesByKeys(roots, ['categoryPath', 'crumbs', 'breadcrumbs', 'breadCrumbs'])),
      ...selectAll('a[href*="sort"], a[href*="category"]')
        .map((node) => cleanText(node.textContent))
    ]).slice(0, 12)

    const payload = {
      taskToken: '',
      offerId,
      title,
      productUrl: window.location.href,
      mainImageUrl,
      galleryImageUrls,
      detailImageUrls,
      priceText,
      priceMin: parseMinPrice(priceText, roots, skuOffers),
      priceMax: parseMaxPrice(priceText, roots, skuOffers),
      currencyCode: firstStringValue(findValuesByKeys(roots, ['currencyCode', 'currency'])) || inferCurrencyCode(priceText),
      minOrderQuantity: parseNumberValue(
        pickText([
          '[class*="moq"]',
          '[class*="min-order"]',
          '[class*="order-quantity"]'
        ]) || firstStringValue(findValuesByKeys(roots, ['minOrderQuantity', 'mixWholeSale', 'beginAmount']))
      ),
      orderUnit: cleanText(
        firstStringValue(findValuesByKeys(roots, ['orderUnit', 'unit', 'saleUnit'])) ||
        pickText(['[class*="unit"]', '[class*="order-unit"]'])
      ),
      sellerCompany: cleanText(
        pickText([
          '[class*="company"]',
          '[data-company-name]'
        ]) || firstStringValue(findValuesByKeys(roots, ['companyName', 'sellerCompany', 'company']))
      ),
      shopName: cleanText(
        pickText([
          '[class*="shop-name"]',
          '[class*="shopName"]',
          '[data-shop-name]'
        ]) || firstStringValue(findValuesByKeys(roots, ['shopName', 'shop', 'sellerNick']))
      ),
      sellerUrl: normalizeUrl(
        pickAttribute([
          'a[href*="company"]',
          'a[href*="factory"]'
        ], 'href') || firstStringValue(findValuesByKeys(roots, ['companyUrl', 'sellerUrl']))
      ),
      shopUrl: normalizeUrl(
        pickAttribute([
          'a[href*="shop.1688.com"]',
          'a[href*="1688.com/page/offerlist"]'
        ], 'href') || firstStringValue(findValuesByKeys(roots, ['shopUrl']))
      ),
      origin: cleanText(
        pickText([
          '[class*="origin"]',
          '[class*="send-address"]',
          '[class*="address"]'
        ]) || firstStringValue(findValuesByKeys(roots, ['origin', 'sendAddress', 'deliveryPlace']))
      ),
      freightText: cleanText(
        pickText([
          '[class*="freight"]',
          '[class*="postage"]'
        ]) || firstStringValue(findValuesByKeys(roots, ['freightText', 'postage', 'carriage']))
      ),
      categoryPath,
      specAttributes,
      productAttributes,
      packageInfo,
      skuAttributes,
      skuOffers,
      detailSections,
      detailText,
      descriptionHtml,
      collectWarnings: collectWarnings({
        offerId,
        title,
        mainImageUrl,
        skuOffers
      }),
      rawPayload: {
        currentUrl: window.location.href,
        host: window.location.host,
        extractedAt: new Date().toISOString(),
        title,
        priceText,
        categoryPath,
        structuredRoots: structuredData.length,
        windowRoots: candidateRoots.length,
        imageExtraction: {
          main: mainImageUrl,
          galleryCount: galleryImageUrls.length,
          detailCount: detailImageUrls.length,
          mainSource: productImages.mainSource,
          candidates: productImages.debugCandidates
        },
        skuExtraction: {
          attributeCount: skuAttributes.length,
          offerCount: skuOffers.length
        },
        sellerMetrics,
        serviceLabels,
        logisticsInfo,
        salesStats,
        tierPrices,
        videoUrls,
        packageInfo,
        specAttributes,
        productAttributes,
        detailSections,
        detailText,
        skuAttributes,
        skuOffers
      }
    }

    return payload
  }

  function collectCandidateRoots() {
    return [
      window.__INIT_DATA__,
      window.__INITIAL_STATE__,
      window.__PRELOADED_STATE__,
      window.__GLOBAL_DATA__,
      window.__APP_INITIAL_STATE__,
      window.__data__,
      window.__page_data__,
      window.__NUXT__,
      window.__NEXT_DATA__,
      window._page_data_,
      window.__offerDetailData,
      window.__offer_detail_data__,
      window.__STORE_DATA__
    ].filter(Boolean)
  }

  function collectStructuredDataRoots() {
    return selectAll('script[type="application/ld+json"], script[type="application/json"]')
      .map((node) => {
        try {
          return JSON.parse(node.textContent || '')
        } catch (error) {
          return null
        }
      })
      .filter(Boolean)
  }

  function extractProductImages(roots) {
    const mainCandidateObjects = [
      ...findImageCandidates(roots, (key, path) => isTrustedMainImageKey(key, path), 'state-main'),
      ...collectImageCandidatesFromSelectors([
        '[class*="main"][class*="image"]',
        '[class*="main-image"]',
        '[class*="mainImage"]',
        '[class*="image-view"]',
        '[class*="imageView"]',
        '[class*="preview"]',
        '[class*="gallery"] [class*="active"]',
        '[class*="album"] [class*="active"]'
      ], 'dom-main')
    ]
    const galleryCandidateObjects = [
      ...mainCandidateObjects,
      ...findImageCandidates(roots, (key, path) => isTrustedGalleryImageKey(key, path), 'state-gallery'),
      ...collectImageCandidatesFromSelectors([
        '[class*="gallery"]',
        '[class*="album"]',
        '[class*="carousel"]',
        '[class*="swipe"]',
        '[class*="thumb"]',
        '[class*="thumbnail"]',
        '[class*="image-view"]',
        '[class*="imageView"]'
      ], 'dom-gallery')
    ]
    const detailCandidateObjects = [
      ...findImageCandidates(roots, (key, path) => isTrustedDetailImageKey(key, path), 'state-detail'),
      ...collectImageCandidatesFromSelectors([
        '#desc-lazyload-container',
        '[class*="detail"] [class*="content"]',
        '[class*="detail-content"]',
        '[class*="description"]',
        '[class*="desc"]'
      ], 'dom-detail')
    ]

    const scoredGallery = uniqueImageCandidates([...mainCandidateObjects, ...galleryCandidateObjects])
      .filter((candidate) => candidate.score >= 30 && !candidate.isDetail && !candidate.isSku)
      .sort((a, b) => b.score - a.score)
    const scoredMain = uniqueImageCandidates(mainCandidateObjects)
      .filter((candidate) => candidate.score >= 45 && !candidate.isDetail && !candidate.isSku)
      .sort((a, b) => b.score - a.score)
    const mainCandidate = scoredMain[0] || scoredGallery[0] || null
    const mainImageUrl = mainCandidate?.url || ''
    const detailCandidates = uniqueImageCandidates(detailCandidateObjects)
      .filter((candidate) => candidate.score >= 20)
      .sort((a, b) => b.score - a.score)
    return {
      mainImageUrl,
      mainSource: mainCandidate?.source || '',
      galleryImageUrls: uniqueStrings(scoredGallery.map((candidate) => candidate.url)).filter((value) => value !== mainImageUrl),
      detailImageUrls: uniqueStrings(detailCandidates.map((candidate) => candidate.url)).filter((value) => value !== mainImageUrl),
      debugCandidates: [...scoredMain, ...scoredGallery, ...detailCandidates]
        .map((candidate) => ({
          url: candidate.url,
          source: candidate.source,
          score: candidate.score,
          context: candidate.context,
          width: candidate.width,
          height: candidate.height,
          isDetail: candidate.isDetail,
          isSku: candidate.isSku
        }))
        .slice(0, 30)
    }
  }

  function collectImageCandidatesFromSelectors(selectors, source) {
    const results = []
    selectors.forEach((selector) => {
      selectAll(selector).forEach((node) => {
        const context = getNodeContextText(node)
        if (!nodeLooksLikeProductImageContext(node, context)) {
          return
        }
        extractImageUrlsFromNode(node).forEach((value) => {
          if (isLikelyProductImage(value, node)) {
            results.push(buildImageCandidate(value, {
              source,
              context,
              node,
              score: scoreImageCandidate(value, node, context, source)
            }))
          }
        })
      })
    })
    return results
  }

  function nodeLooksLikeProductImageContext(node, initialContext) {
    if (IMAGE_CONTEXT_REJECT_PATTERN.test(initialContext || '')) {
      return false
    }
    let current = node
    let depth = 0
    while (current && depth < 5) {
      const marker = cleanText([
        current.id,
        current.className,
        current.getAttribute?.('aria-label'),
        current.getAttribute?.('data-spm')
      ].filter(Boolean).join(' '))
      if (IMAGE_CONTEXT_REJECT_PATTERN.test(marker)) {
        return false
      }
      current = current.parentElement
      depth += 1
    }
    return true
  }

  function getNodeContextText(node) {
    const parts = []
    let current = node
    let depth = 0
    while (current && depth < 4) {
      parts.push(
        current.id,
        current.className,
        current.getAttribute?.('aria-label'),
        current.getAttribute?.('data-spm'),
        current.getAttribute?.('data-module-name')
      )
      current = current.parentElement
      depth += 1
    }
    return cleanText(parts.filter(Boolean).join(' '))
  }

  function scoreImageCandidate(rawUrl, node, context, source) {
    let score = 0
    const lowerContext = String(context || '')
    if (/main/i.test(source || '')) {
      score += 45
    }
    if (/gallery/i.test(source || '')) {
      score += 35
    }
    if (/state/i.test(source || '')) {
      score += 10
    }
    if (IMAGE_CONTEXT_GALLERY_PATTERN.test(lowerContext)) {
      score += 25
    }
    if (IMAGE_CONTEXT_DETAIL_PATTERN.test(lowerContext)) {
      score -= /detail/i.test(source || '') ? 0 : 45
    }
    if (IMAGE_CONTEXT_SKU_PATTERN.test(lowerContext)) {
      score -= 35
    }
    if (IMAGE_CONTEXT_REJECT_PATTERN.test(lowerContext)) {
      score -= 80
    }
    const size = getImageNodeSize(node)
    if (size.width >= 300 || size.height >= 300) {
      score += 15
    } else if (Math.max(size.width, size.height) >= 160) {
      score += 8
    } else if (size.width > 0 && size.height > 0) {
      score -= 20
    }
    if (isLikelyProductImage(rawUrl, node)) {
      score += 8
    }
    return score
  }

  function getImageNodeSize(node) {
    const image = node?.tagName === 'IMG' ? node : node?.querySelector?.('img')
    const rect = node?.getBoundingClientRect?.()
    return {
      width: Number(image?.naturalWidth || image?.clientWidth || image?.width || rect?.width || 0),
      height: Number(image?.naturalHeight || image?.clientHeight || image?.height || rect?.height || 0)
    }
  }

  function buildImageCandidate(rawUrl, options = {}) {
    const url = normalizeUrl(rawUrl)
    const size = getImageNodeSize(options.node)
    const context = cleanText(options.context || '')
    return {
      url,
      source: options.source || '',
      context,
      score: Number(options.score || 0),
      width: size.width,
      height: size.height,
      isDetail: IMAGE_CONTEXT_DETAIL_PATTERN.test(context) || /detail/i.test(options.source || ''),
      isSku: IMAGE_CONTEXT_SKU_PATTERN.test(context) || /sku/i.test(options.source || '')
    }
  }

  function uniqueImageCandidates(values) {
    const result = []
    const seen = new Set()
    values.forEach((candidate) => {
      if (!candidate?.url || seen.has(candidate.url)) {
        return
      }
      seen.add(candidate.url)
      result.push(candidate)
    })
    return result
  }

  function isTrustedMainImageKey(key, path) {
    const text = `${key || ''} ${path || ''}`
    if (/sku|seller|shop|logo|avatar|recommend|similar|detail|desc/i.test(text)) {
      return false
    }
    return /(main|primary|首图|主图).*(image|img|pic|url)|^(mainImage|mainImg|mainPic|mainImageUrl|offerImgUrl)$/i.test(key || '')
  }

  function isTrustedGalleryImageKey(key, path) {
    const text = `${key || ''} ${path || ''}`
    if (/sku|seller|shop|logo|avatar|recommend|similar|detail|desc/i.test(text)) {
      return false
    }
    return /(gallery|carousel|swipe|album|imageList|images|bigPic|offerImages|productImages|mainImages|主图|相册|图集)/i.test(text)
  }

  function isTrustedDetailImageKey(key, path) {
    const text = `${key || ''} ${path || ''}`
    return /(detail|description|descImage|descImgs|moduleImages|offerDetail|详情|描述)/i.test(text)
  }

  function extractImageUrlsFromNode(node) {
    const nodes = node.matches?.('img,source') ? [node] : []
    const children = node.querySelectorAll ? Array.from(node.querySelectorAll('img,source')) : []
    const values = []
    ;[...nodes, ...children].forEach((image) => {
      values.push(
        image.getAttribute('src'),
        image.getAttribute('data-src'),
        image.getAttribute('data-lazy-src'),
        image.getAttribute('data-ks-lazyload'),
        image.getAttribute('data-img'),
        image.getAttribute('data-url'),
        image.getAttribute('data-original'),
        image.getAttribute('srcset')
      )
    })
    const style = node.getAttribute?.('style') || ''
    const matchedBackground = style.match(/url\((['"]?)(.*?)\1\)/i)
    if (matchedBackground?.[2]) {
      values.push(matchedBackground[2])
    }
    return values
      .flatMap(splitSrcset)
      .map((value) => normalizeUrl(value))
      .filter(Boolean)
  }

  function splitSrcset(value) {
    return String(value || '')
      .split(',')
      .map((item) => item.trim().split(/\s+/)[0])
      .filter(Boolean)
  }

  function isLikelyProductImage(rawUrl, node) {
    const value = normalizeUrl(rawUrl)
    if (!value || !/^https?:\/\//i.test(value)) {
      return false
    }
    const lower = value.toLowerCase()
    if (IMAGE_REJECT_PATTERN.test(lower) || /\/\/g\.alicdn\.com\//i.test(lower) || /\/tfs\//i.test(lower)) {
      return false
    }
    if (!looksLikeImageUrl(value) && !PRODUCT_IMAGE_PATTERN.test(value)) {
      return false
    }
    if (node && node.tagName === 'IMG') {
      const width = Number(node.naturalWidth || node.clientWidth || node.width || 0)
      const height = Number(node.naturalHeight || node.clientHeight || node.height || 0)
      if (width > 0 && height > 0 && Math.max(width, height) < 80) {
        return false
      }
    }
    return true
  }

  function extractSellerMetrics() {
    const text = cleanText(document.body?.innerText || '')
    return compactObject({
      repurchaseRate: extractLabelValue(text, '店铺回头率'),
      serviceScore: extractLabelValue(text, '店铺服务分'),
      onTimeDispatchRate: extractLabelValue(text, '准时发货率'),
      favorableRate: extractLabelValue(text, '店铺好评率')
    })
  }

  function extractServiceLabels() {
    const text = cleanText(document.body?.innerText || '')
    return [
      '先采后付',
      '退货包运费',
      '7天无理由退货',
      '晚发必赔',
      '极速退款',
      '承诺48小时发货',
      '官方包退货',
      '跨境铺货',
      '一件代发'
    ].filter((label) => text.includes(label))
  }

  function extractLogisticsInfo(roots) {
    const text = cleanText(document.body?.innerText || '')
    return compactObject({
      origin: extractLabelValue(text, '发货地') || firstStringValue(findValuesByKeys(roots, ['origin', 'sendAddress', 'deliveryPlace'])),
      destination: extractTextByPattern(text, /送至\s*([^|，。；\s]{2,20})/),
      freightText: extractLabelValue(text, '运费') || firstStringValue(findValuesByKeys(roots, ['freightText', 'postage', 'carriage'])),
      dispatchPromise: extractTextByPattern(text, /(承诺\s*\d+\s*小时发货)/)
    })
  }

  function extractSalesStats(roots) {
    const text = cleanText(document.body?.innerText || '')
    return compactObject({
      soldText: extractTextByPattern(text, /已售\s*[\d,.万+]+\s*[^\s，。；|]*/),
      minOrderText: extractTextByPattern(text, /\d+\s*个起批/) || firstStringValue(findValuesByKeys(roots, ['beginAmount', 'minOrderQuantity', 'mixWholeSale']))
    })
  }

  function extractTierPrices(roots) {
    const candidates = findValuesByKeys(roots, ['priceRangeList', 'priceRanges', 'ladderPrices', 'ladderPriceList', 'wholesalePriceList'])
    return candidates.flatMap((value) => {
      if (Array.isArray(value)) {
        return value.map(normalizeTierPrice).filter((item) => Object.keys(item).length)
      }
      if (value && typeof value === 'object') {
        return [normalizeTierPrice(value)].filter((item) => Object.keys(item).length)
      }
      return []
    }).slice(0, 20)
  }

  function normalizeTierPrice(value) {
    if (!value || typeof value !== 'object') {
      return {}
    }
    return compactObject({
      beginAmount: firstValueByPossibleKeys(value, ['beginAmount', 'startQuantity', 'minQuantity', 'quantity']),
      price: firstValueByPossibleKeys(value, ['price', 'salePrice', 'discountPrice', 'amount']),
      priceText: firstValueByPossibleKeys(value, ['priceText', 'displayPrice', 'priceDisplay'])
    })
  }

  function extractVideoUrls() {
    return uniqueStrings([
      ...selectAll('video[src], video source[src], source[src]')
        .map((node) => normalizeUrl(node.getAttribute('src'))),
      ...selectAll('[data-video-url], [data-video], [video-url]')
        .map((node) => normalizeUrl(node.getAttribute('data-video-url') || node.getAttribute('data-video') || node.getAttribute('video-url')))
    ]).filter((url) => /^https?:\/\//i.test(url)).slice(0, 12)
  }

  function extractPackageInfo(roots) {
    const fromState = normalizePackageInfo(pickBestObject(findValuesByKeys(roots, ['packageInfo', 'packingInfo', 'packing', 'skuPackageInfo', 'packageWeight', 'packageSize'])))
    const fromDom = normalizePackageInfo(parseAttributeSectionsByLabels(['包装信息', '包装参数', '包装规格', '物流包装', '装箱信息', 'packing', 'package']))
    return compactObject({
      ...fromState,
      ...fromDom,
      rawText: cleanText(fromDom.rawText || fromState.rawText || '').slice(0, 1000)
    })
  }

  function extractProductAttributes(roots, specAttributes) {
    const fromState = findValuesByKeys(roots, [
      'productAttributes',
      'productProps',
      'offerAttributes',
      'offerProperties',
      'detailProps',
      'productParams',
      'attributes',
      'props',
      'specifications'
    ])
      .map(normalizeAttributeMap)
      .reduce((merged, value) => ({ ...merged, ...value }), {})
    const fromDom = parseAttributeSectionsByLabels(['商品属性', '产品属性', '商品参数', '产品参数', '规格参数', '属性参数'])
    return compactObject({
      ...clonePlainObject(specAttributes),
      ...fromState,
      ...fromDom
    })
  }

  function extractDetailSections(roots) {
    const fromState = findValuesByKeys(roots, ['detailSections', 'detailModules', 'moduleList', 'offerDetail', 'description', 'descriptionHtml', 'detailHtml', 'desc'])
      .flatMap(normalizeDetailSectionValue)
    const fromDom = selectAll([
      '#desc-lazyload-container',
      '[class*="offer-detail"]',
      '[class*="detail"] [class*="content"]',
      '[class*="detail-content"]',
      '[class*="description"]',
      '[class*="desc"]'
    ].join(','))
      .filter((node) => nodeLooksLikeDetailSection(node))
      .map(normalizeDetailSectionNode)
      .filter(Boolean)
    return uniqueObjectsByKey([...fromDom, ...fromState], (section) => `${section.title}|${section.text}|${(section.imageUrls || []).join('|')}`).slice(0, 40)
  }

  function normalizeDetailSectionValue(value) {
    if (!value) {
      return []
    }
    if (typeof value === 'string') {
      const text = cleanText(stripHtml(value))
      return text || /<img/i.test(value) ? [{
        title: '',
        text,
        html: value,
        imageUrls: extractImageUrlsFromHtml(value)
      }] : []
    }
    if (Array.isArray(value)) {
      return value.flatMap(normalizeDetailSectionValue)
    }
    if (typeof value === 'object') {
      const html = cleanText(firstValueByPossibleKeys(value, ['html', 'content', 'desc', 'description', 'detailHtml']))
      const text = cleanText(firstValueByPossibleKeys(value, ['text', 'titleText', 'contentText'])) || stripHtml(html)
      const title = cleanText(firstValueByPossibleKeys(value, ['title', 'name', 'label', 'moduleName']))
      const imageUrls = uniqueStrings([
        ...extractImageUrlsFromHtml(html),
        ...findImagesInPlainObject(value)
      ])
      const section = compactObject({
        title,
        text,
        html,
        imageUrls
      })
      return Object.keys(section).length ? [section] : []
    }
    return []
  }

  function normalizeDetailSectionNode(node) {
    const html = cleanText(node.innerHTML || '')
    const text = cleanText(node.innerText || node.textContent || stripHtml(html))
    const title = cleanText(node.querySelector?.('h1,h2,h3,h4,[class*="title"]')?.textContent || '')
    const imageUrls = uniqueStrings(extractImageUrlsFromNode(node).filter((value) => isLikelyProductImage(value)))
    const section = compactObject({
      title,
      text: text.slice(0, 4000),
      html,
      imageUrls
    })
    return Object.keys(section).length ? section : null
  }

  function nodeLooksLikeDetailSection(node) {
    const text = cleanText(node.innerText || node.textContent || '')
    const context = getNodeContextText(node)
    if (IMAGE_CONTEXT_REJECT_PATTERN.test(context)) {
      return false
    }
    return IMAGE_CONTEXT_DETAIL_PATTERN.test(context) || text.includes('商品详情') || text.includes('产品详情') || text.length > 80
  }

  function parseAttributeSectionsByLabels(labels) {
    const result = {}
    const rawTexts = []
    const normalizedLabels = labels.map((label) => normalizeKey(label))
    selectAll('table,dl,ul,ol,section,div')
      .filter((node) => {
        const text = cleanText(node.innerText || node.textContent || '')
        if (!text || text.length > 5000) {
          return false
        }
        const marker = cleanText([
          node.id,
          node.className,
          node.getAttribute?.('data-module-name'),
          text.slice(0, 80)
        ].filter(Boolean).join(' '))
        const normalizedMarker = normalizeKey(marker)
        return normalizedLabels.some((label) => normalizedMarker.includes(label))
      })
      .forEach((node) => {
        const text = cleanText(node.innerText || node.textContent || '')
        if (text) {
          rawTexts.push(text)
        }
        Object.assign(result, parseAttributePairsFromNode(node))
      })
    if (rawTexts.length) {
      result.rawText = uniqueStrings(rawTexts).join('\n').slice(0, 2000)
    }
    return compactObject(result)
  }

  function parseAttributePairsFromNode(node) {
    const result = {}
    selectAllWithin(node, 'tr').forEach((row) => {
      const cells = selectAllWithin(row, 'th,td')
      if (cells.length >= 2) {
        const key = cleanText(cells[0].innerText || cells[0].textContent)
        const value = cleanText(cells.slice(1).map((cell) => cell.innerText || cell.textContent).join(' '))
        assignAttributePair(result, key, value)
      }
    })
    selectAllWithin(node, 'dt').forEach((term) => {
      const key = cleanText(term.innerText || term.textContent)
      const value = cleanText(term.nextElementSibling?.innerText || term.nextElementSibling?.textContent || '')
      assignAttributePair(result, key, value)
    })
    selectAllWithin(node, 'li,p,span,div').forEach((child) => {
      const text = cleanText(child.innerText || child.textContent || '')
      parseAttributePairsFromText(text, result)
    })
    parseAttributePairsFromText(cleanText(node.innerText || node.textContent || ''), result)
    return compactObject(result)
  }

  function parseAttributePairsFromText(text, result = {}) {
    String(text || '')
      .split(/[\n\r]+| {2,}|；|;/)
      .map(cleanText)
      .filter(Boolean)
      .forEach((line) => {
        const matched = line.match(/^([^:：]{1,24})[:：]\s*(.{1,200})$/)
        if (!matched) {
          return
        }
        assignAttributePair(result, matched[1], matched[2])
      })
    return result
  }

  function assignAttributePair(result, rawKey, rawValue) {
    const key = cleanText(rawKey).replace(/^[：:]+|[：:]+$/g, '')
    const value = cleanText(rawValue)
    if (!key || !value || key === value || /商品属性|产品属性|包装信息|商品详情/.test(key)) {
      return
    }
    if (key.length > 40 || value.length > 500) {
      return
    }
    result[key] = value
  }

  function normalizeAttributeMap(value) {
    if (!value) {
      return {}
    }
    if (Array.isArray(value)) {
      return value.reduce((merged, item) => ({ ...merged, ...normalizeAttributeMap(item) }), {})
    }
    if (typeof value !== 'object') {
      return {}
    }
    const key = cleanText(firstValueByPossibleKeys(value, ['name', 'key', 'attributeName', 'propName', 'title', 'label']))
    const attrValue = cleanText(firstValueByPossibleKeys(value, ['value', 'valueName', 'text', 'content', 'desc']))
    if (key && attrValue) {
      return { [key]: attrValue }
    }
    return Object.entries(value).reduce((merged, [entryKey, entryValue]) => {
      if (entryValue === null || typeof entryValue === 'undefined') {
        return merged
      }
      if (typeof entryValue === 'string' || typeof entryValue === 'number' || typeof entryValue === 'boolean') {
        assignAttributePair(merged, entryKey, entryValue)
        return merged
      }
      if (Array.isArray(entryValue)) {
        const nested = normalizeAttributeMap(entryValue)
        if (Object.keys(nested).length) {
          merged[entryKey] = nested
        }
      }
      return merged
    }, {})
  }

  function normalizePackageInfo(value) {
    const normalized = normalizeAttributeMap(value)
    const rawText = cleanText(normalized.rawText || Object.entries(normalized).map(([key, child]) => `${key}: ${formatPlainValue(child)}`).join(' '))
    const dimensionText = /\d+(?:\.\d+)?\s*[xX*×]\s*\d+(?:\.\d+)?/.test(rawText) ? rawText : ''
    return compactObject({
      ...normalized,
      weightKg: parseWeightKg(firstMatchingAttributeValue(normalized, ['重量', '毛重', '净重', '包装重量', 'weight', 'weightKg']) || rawText),
      lengthCm: parseLengthCm(firstMatchingAttributeValue(normalized, ['长度', '长', '包装长度', 'length', 'lengthCm']) || dimensionText, 0),
      widthCm: parseLengthCm(firstMatchingAttributeValue(normalized, ['宽度', '宽', '包装宽度', 'width', 'widthCm']) || dimensionText, 1),
      heightCm: parseLengthCm(firstMatchingAttributeValue(normalized, ['高度', '高', '包装高度', 'height', 'heightCm']) || dimensionText, 2),
      packageQuantity: parseNumberValue(firstMatchingAttributeValue(normalized, ['装箱数量', '包装数量', '每箱数量', 'quantity', 'packageQuantity'])),
      packageUnit: cleanText(firstMatchingAttributeValue(normalized, ['包装单位', '单位', 'packageUnit'])),
      rawText
    })
  }

  function firstMatchingAttributeValue(values, keys) {
    const normalizedKeys = keys.map(normalizeKey)
    for (const [key, value] of Object.entries(values || {})) {
      const normalized = normalizeKey(key)
      if (normalizedKeys.some((candidate) => normalized.includes(candidate))) {
        return value
      }
    }
    return ''
  }

  function parseWeightKg(value) {
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value
    }
    const text = cleanText(value)
    const matched = text.match(/(\d+(?:\.\d+)?)\s*(kg|千克|公斤|g|克)/i) || text.match(/(\d+(?:\.\d+)?)/)
    if (!matched) {
      return null
    }
    const number = Number(matched[1])
    if (!Number.isFinite(number)) {
      return null
    }
    const unit = String(matched[2] || '').toLowerCase()
    if (unit === 'g' || unit === '克') {
      return roundNumber(number / 1000, 4)
    }
    return roundNumber(number, 4)
  }

  function parseLengthCm(value, index) {
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value
    }
    const text = cleanText(value)
    const dimensions = text.match(/\d+(?:\.\d+)?/g) || []
    if (!dimensions.length || index >= dimensions.length) {
      return null
    }
    const number = Number(dimensions[index])
    if (!Number.isFinite(number)) {
      return null
    }
    if (/mm|毫米/i.test(text)) {
      return roundNumber(number / 10, 4)
    }
    if (/m|米/i.test(text) && !/cm|厘米/i.test(text) && !/mm|毫米/i.test(text)) {
      return roundNumber(number * 100, 4)
    }
    return roundNumber(number, 4)
  }

  function roundNumber(value, precision) {
    const factor = Math.pow(10, precision || 2)
    return Math.round(value * factor) / factor
  }

  function extractLabelValue(text, label) {
    return extractTextByPattern(text, new RegExp(`${escapeRegExp(label)}\\s*[:：]?\\s*([\\d.]+\\s*%?|[\\d.]+\\s*分|[^\\s，。；|]{1,24})`))
  }

  function extractTextByPattern(text, pattern) {
    const match = String(text || '').match(pattern)
    return cleanText(match?.[1] || match?.[0] || '')
  }

  function escapeRegExp(value) {
    return String(value || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  }

  function extractSkuAttributes(roots) {
    const fromState = findValuesByKeys(roots, [
      'skuAttrs',
      'skuAttributes',
      'saleProps',
      'salePropList',
      'skuProps',
      'skuPropList',
      'saleProp',
      'skuPropertyList'
    ]).flatMap(normalizeSkuAttributeValue)

    const fromDom = inferSkuAttributesFromDom()
    return uniqueObjectsByKey([...fromState, ...fromDom], (item) => cleanText(item.name || item.propName || JSON.stringify(item)))
  }

  function normalizeSkuAttributeValue(value) {
    if (!value) {
      return []
    }
    if (Array.isArray(value)) {
      return value
        .map(normalizeSkuAttribute)
        .filter((item) => item && cleanText(item.name || item.propName || item.label))
    }
    if (typeof value === 'object') {
      if (Array.isArray(value.values) || Array.isArray(value.valueList) || Array.isArray(value.children)) {
        const normalized = normalizeSkuAttribute(value)
        return normalized ? [normalized] : []
      }
      return Object.entries(value).flatMap(([key, child]) => {
        if (Array.isArray(child)) {
          return [{
            name: cleanText(key),
            values: child.map(normalizeSkuAttributeOption).filter(Boolean)
          }]
        }
        if (child && typeof child === 'object') {
          const normalized = normalizeSkuAttribute(child)
          if (normalized && !normalized.name) {
            normalized.name = cleanText(key)
          }
          return normalized ? [normalized] : []
        }
        return []
      })
    }
    return []
  }

  function normalizeSkuAttribute(value) {
    if (!value || typeof value !== 'object') {
      return null
    }
    const name = cleanText(firstValueByPossibleKeys(value, ['name', 'propName', 'attributeName', 'title', 'label']))
    const values = firstArrayByPossibleKeys(value, ['values', 'valueList', 'children', 'options', 'items'])
      .map(normalizeSkuAttributeOption)
      .filter(Boolean)
    return {
      ...clonePlainObject(value),
      name,
      values
    }
  }

  function normalizeSkuAttributeOption(value) {
    if (!value) {
      return null
    }
    if (typeof value === 'string' || typeof value === 'number') {
      return { name: cleanText(value) }
    }
    if (typeof value !== 'object') {
      return null
    }
    return {
      ...clonePlainObject(value),
      name: cleanText(firstValueByPossibleKeys(value, ['name', 'value', 'valueName', 'title', 'label', 'text'])),
      imageUrl: firstImageFromObject(value)
    }
  }

  function inferSkuAttributesFromDom() {
    const offers = extractSkuOffersFromDom()
    if (!offers.length) {
      return []
    }
    const name = inferSkuAttributeName(offers.map((offer) => offer.attributeText).join(' '))
    return [{
      name,
      values: offers.map((offer) => ({
        name: offer.attributeText,
        imageUrl: offer.imageUrl || ''
      }))
    }]
  }

  function extractSkuOffers(roots, skuAttributes) {
    const stateOffers = extractSkuOffersFromState(roots, skuAttributes)
    const domOffers = extractSkuOffersFromDom()
    return mergeSkuOffers([...stateOffers, ...domOffers])
  }

  function extractSkuOffersFromState(roots, skuAttributes) {
    const values = findKeyedValuesByKeys(roots, [
      'skuOffers',
      'skuMap',
      'skuInfoMap',
      'skuList',
      'skuInfos',
      'skuInfoList',
      'priceBySku',
      'skuPrice',
      'skuPriceList',
      'skuPriceMap',
      'offerSkuMap',
      'offerSkuList',
      'skuModel'
    ])
    const rows = []
    values.forEach((entry) => {
      flattenSkuValue(entry.value, entry.key).forEach((row) => rows.push(row))
    })
    return rows
      .map((row, index) => normalizeSkuOffer(row.value, row.mapKey, index, skuAttributes))
      .filter(Boolean)
  }

  function flattenSkuValue(value, parentKey) {
    if (!value) {
      return []
    }
    if (Array.isArray(value)) {
      return value.map((item) => ({ value: item, mapKey: '' }))
    }
    if (typeof value !== 'object') {
      return []
    }
    if (looksLikeSkuOfferObject(value)) {
      return [{ value, mapKey: '' }]
    }
    return Object.entries(value).flatMap(([key, child]) => {
      if (!child) {
        return []
      }
      if (Array.isArray(child)) {
        return child.map((item) => ({ value: item, mapKey: key }))
      }
      if (typeof child === 'object') {
        return [{ value: child, mapKey: key }]
      }
      if (/price|stock|count|amount/i.test(parentKey || key)) {
        return [{ value: { [parentKey || key]: child }, mapKey: key }]
      }
      return []
    })
  }

  function looksLikeSkuOfferObject(value) {
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
      return false
    }
    const keys = Object.keys(value).map(normalizeKey)
    return keys.some((key) => /(skuid|skukey|specid|price|saleprice|discountprice|stock|canbookcount|amountonsale|inventory)/i.test(key))
  }

  function normalizeSkuOffer(rawValue, mapKey, index, skuAttributes) {
    if (!rawValue || typeof rawValue !== 'object') {
      return null
    }
    const row = clonePlainObject(rawValue)
    const skuId = cleanText(firstValueByPossibleKeys(row, ['skuId', 'skuID', 'sku', 'specId', 'id', 'skuIdStr', 'offerSkuId']))
    const explicitSkuKey = cleanText(firstValueByPossibleKeys(row, ['skuKey', 'key', 'specKey', 'specIdStr'])) || cleanText(mapKey) || skuId
    const attrs = extractSkuOfferAttrs(row, explicitSkuKey, skuAttributes)
    const attributeText = cleanText(
      firstValueByPossibleKeys(row, ['attributeText', 'specText', 'skuName', 'specName', 'name', 'title', 'label']) ||
      formatSkuAttrs(attrs) ||
      normalizeSkuKeyText(explicitSkuKey)
    )
    const rawPrice = firstValueByPossibleKeys(row, ['price', 'salePrice', 'discountPrice', 'retailPrice', 'offerPrice', 'priceText', 'priceDisplay', 'amount', 'priceCent'])
    const price = parseNumberValue(rawPrice)
    const priceText = cleanText(firstValueByPossibleKeys(row, ['priceText', 'priceDisplay'])) || formatPriceText(price, rawPrice)
    const rawStock = firstValueByPossibleKeys(row, ['canBookCount', 'stock', 'amountOnSale', 'inventory', 'quantity', 'availableStock', 'sellableQuantity', 'stockText'])
    const stock = parseNumberValue(rawStock)
    const stockText = cleanText(firstValueByPossibleKeys(row, ['stockText', 'inventoryText'])) || formatStockText(stock, rawStock)
    const imageUrl = firstImageFromObject(row)

    const hasSkuIdentity = Boolean(skuId || explicitSkuKey || attributeText)
    const hasCommercialData = Boolean(priceText || stockText || typeof price === 'number' || typeof stock === 'number')
    if (!hasSkuIdentity || !hasCommercialData) {
      return null
    }
    return compactObject({
      skuId,
      skuKey: explicitSkuKey || `sku-${index + 1}`,
      attrs,
      attributeText,
      price,
      priceText,
      stock,
      stockText,
      imageUrl
    })
  }

  function extractSkuOfferAttrs(row, skuKey, skuAttributes) {
    const direct = firstObjectByPossibleKeys(row, ['specAttrs', 'attrs', 'skuAttrs', 'attributes', 'spec', 'saleProps', 'props'])
    if (direct && Object.keys(direct).length) {
      return clonePlainObject(direct)
    }
    const attributeText = cleanText(firstValueByPossibleKeys(row, ['attributeText', 'specText', 'skuName', 'specName', 'name', 'title', 'label'])) || normalizeSkuKeyText(skuKey)
    if (attributeText) {
      return { [inferSkuAttributeName(attributeText, skuAttributes)]: attributeText }
    }
    return {}
  }

  function extractSkuOffersFromDom() {
    const rows = selectAll('div,li,tr')
      .filter((node) => isSmallestSkuRow(node))
      .map((node, index) => normalizeDomSkuRow(node, index))
      .filter(Boolean)
    return uniqueObjectsByKey(rows, (row) => `${row.attributeText}|${row.priceText}|${row.stockText}`)
  }

  function isSmallestSkuRow(node) {
    const text = cleanText(node.innerText || node.textContent)
    if (!isPotentialSkuRowText(text)) {
      return false
    }
    if (text.length > 260) {
      return false
    }
    return !Array.from(node.children || []).some((child) => {
      const childText = cleanText(child.innerText || child.textContent)
      return childText && childText.length < text.length && isPotentialSkuRowText(childText)
    })
  }

  function isPotentialSkuRowText(text) {
    return SKU_ROW_PRICE_PATTERN.test(text || '') && SKU_ROW_STOCK_PATTERN.test(text || '')
  }

  function normalizeDomSkuRow(node, index) {
    const text = cleanText(node.innerText || node.textContent)
    const priceMatch = text.match(SKU_ROW_PRICE_PATTERN)
    const stockMatch = text.match(/(库存|可售|现货|有货)\s*(\d+(?:\.\d+)?)\s*([\u4e00-\u9fa5A-Za-z]*)?/)
    if (!priceMatch || !stockMatch) {
      return null
    }
    const priceText = cleanText(priceMatch[0])
    const price = parseNumberValue(priceText)
    const stock = parseNumberValue(stockMatch[2])
    const stockText = cleanText(stockMatch[0])
    const attributeText = extractAttributeTextFromSkuRow(text, priceMatch.index || 0, node)
    if (!attributeText) {
      return null
    }
    const imageUrl = extractImageUrlsFromNode(node).find((value) => isLikelyProductImage(value, node)) || ''
    const attrName = inferSkuAttributeName(attributeText)
    return compactObject({
      skuId: cleanText(node.getAttribute?.('data-sku-id') || node.getAttribute?.('data-id') || ''),
      skuKey: cleanText(node.getAttribute?.('data-sku-id') || node.getAttribute?.('data-id') || attributeText) || `sku-${index + 1}`,
      attrs: { [attrName]: attributeText },
      attributeText,
      price,
      priceText,
      stock,
      stockText,
      imageUrl
    })
  }

  function extractAttributeTextFromSkuRow(text, priceIndex, node) {
    const prefix = cleanText(text.slice(0, priceIndex))
      .replace(/^(颜色|规格|款式|型号|尺寸|尺码)\s*[:：]?\s*/i, '')
      .replace(/^[\-+×x\d\s]+/, '')
      .trim()
    if (prefix) {
      const parts = prefix.split(/\s+/).filter((part) => !/^[\-+×x\d]+$/.test(part))
      return cleanText(parts[parts.length - 1] || prefix)
    }
    const image = node.querySelector?.('img')
    return cleanText(image?.getAttribute('alt') || image?.getAttribute('title') || '')
  }

  function inferSkuAttributeName(text, skuAttributes) {
    const known = (skuAttributes || []).map((item) => cleanText(item.name || item.propName || item.label)).find(Boolean)
    if (known) {
      return known
    }
    if (/色|红|粉|蓝|紫|绿|黄|黑|白|灰|橙|棕/.test(text || '')) {
      return '颜色'
    }
    if (/码|尺寸|cm|mm|inch|英寸|大号|小号|中号/i.test(text || '')) {
      return '尺寸'
    }
    return '规格'
  }

  function mergeSkuOffers(values) {
    const result = []
    const seen = new Set()
    values.forEach((value, index) => {
      if (!value) {
        return
      }
      const normalized = {
        ...value,
        skuKey: cleanText(value.skuKey || value.skuId || `sku-${index + 1}`)
      }
      const key = cleanText(normalized.skuKey || normalized.attributeText || JSON.stringify(normalized.attrs || {}))
      if (!key || seen.has(key)) {
        return
      }
      seen.add(key)
      result.push(normalized)
    })
    return result
  }

  function findValuesByKeys(roots, keys) {
    return findKeyedValuesByKeys(roots, keys).map((item) => item.value)
  }

  function findKeyedValuesByKeys(roots, keys) {
    const normalizedKeys = keys.map((key) => normalizeKey(key))
    const results = []
    for (const root of roots) {
      scanObject(root, (value, key) => {
        if (!key) {
          return
        }
        if (normalizedKeys.includes(normalizeKey(key))) {
          results.push({ key, value })
        }
      })
    }
    return results
  }

  function findImageCandidates(roots, matcher, source) {
    const results = []
    for (const root of roots) {
      scanObject(root, (value, key, parent, path) => {
        const pathText = (path || []).join('.')
        if (!matcher(String(key || ''), pathText, parent)) {
          return
        }
        const context = pathText
        if (typeof value === 'string') {
          const normalized = normalizeUrl(value)
          if (isLikelyProductImage(normalized)) {
            results.push(buildImageCandidate(normalized, {
              source,
              context,
              score: scoreStateImageCandidate(String(key || ''), pathText, source)
            }))
          }
          return
        }
        if (Array.isArray(value)) {
          value.forEach((item) => {
            if (typeof item === 'string') {
              const normalized = normalizeUrl(item)
              if (isLikelyProductImage(normalized)) {
                results.push(buildImageCandidate(normalized, {
                  source,
                  context,
                  score: scoreStateImageCandidate(String(key || ''), pathText, source)
                }))
              }
            }
            if (item && typeof item === 'object') {
              const image = firstImageFromObject(item)
              if (image) {
                results.push(buildImageCandidate(image, {
                  source,
                  context,
                  score: scoreStateImageCandidate(String(key || ''), pathText, source)
                }))
              }
            }
          })
        }
        if (value && typeof value === 'object' && !Array.isArray(value)) {
          const image = firstImageFromObject(value)
          if (image) {
            results.push(buildImageCandidate(image, {
              source,
              context,
              score: scoreStateImageCandidate(String(key || ''), pathText, source)
            }))
          }
        }
      })
    }
    return results
  }

  function scoreStateImageCandidate(key, path, source) {
    let score = /main/i.test(source || '') ? 65 : 45
    const text = `${key || ''} ${path || ''}`
    if (/gallery|album|imageList|offerImages|mainImages|主图|相册|图集/i.test(text)) {
      score += 15
    }
    if (/detail|desc|description|详情/i.test(text)) {
      score -= /detail/i.test(source || '') ? 0 : 45
    }
    if (/sku|seller|shop|logo|avatar|recommend|similar/i.test(text)) {
      score -= 60
    }
    return score
  }

  function scanObject(root, visitor) {
    const queue = [{ value: root, path: [] }]
    const seen = new WeakSet()
    let inspected = 0
    while (queue.length && inspected < 8000) {
      const entry = queue.shift()
      const current = entry?.value
      if (!current || typeof current !== 'object') {
        continue
      }
      if (seen.has(current)) {
        continue
      }
      seen.add(current)
      inspected += 1
      if (Array.isArray(current)) {
        current.forEach((item, index) => queue.push({ value: item, path: [...entry.path, String(index)] }))
        continue
      }
      Object.entries(current).forEach(([key, value]) => {
        const nextPath = [...entry.path, key]
        visitor(value, key, current, nextPath)
        if (value && typeof value === 'object') {
          queue.push({ value, path: nextPath })
        }
      })
    }
  }

  function pickBestObject(values) {
    return values.find((value) => value && typeof value === 'object' && !Array.isArray(value)) || {}
  }

  function findArrayStrings(values) {
    return values.flatMap((value) => {
      if (!Array.isArray(value)) {
        return []
      }
      return value.flatMap((item) => {
        if (typeof item === 'string') {
          return cleanText(item)
        }
        if (item && typeof item === 'object') {
          return Object.values(item).map((child) => cleanText(child)).filter(Boolean)
        }
        return []
      })
    })
  }

  function pickText(selectors) {
    for (const selector of selectors) {
      const node = document.querySelector(selector)
      const value = cleanText(node?.textContent || '')
      if (value) {
        return value
      }
    }
    return ''
  }

  function pickAttribute(selectors, attribute) {
    for (const selector of selectors) {
      const node = document.querySelector(selector)
      const value = normalizeUrl(node?.getAttribute?.(attribute) || '')
      if (value) {
        return value
      }
    }
    return ''
  }

  function selectAll(selector) {
    return Array.from(document.querySelectorAll(selector))
  }

  function selectAllWithin(node, selector) {
    return node?.querySelectorAll ? Array.from(node.querySelectorAll(selector)) : []
  }

  function extractDescriptionHtml(detailSections) {
    const sectionHtml = (detailSections || []).map((section) => cleanText(section.html || '')).find(Boolean)
    if (sectionHtml) {
      return sectionHtml
    }
    const node = document.querySelector('[class*="detail"] [class*="content"]') || document.querySelector('#desc-lazyload-container')
      || document.querySelector('[class*="detail-content"]')
    if (node?.innerHTML) {
      return node.innerHTML.trim()
    }
    return ''
  }

  function extractImageUrlsFromHtml(html) {
    const results = []
    String(html || '').replace(/<(?:img|source)[^>]+(?:src|data-src|data-lazy-src|data-ks-lazyload|data-original)=["']([^"']+)["'][^>]*>/gi, (_, url) => {
      const normalized = normalizeUrl(url)
      if (isLikelyProductImage(normalized)) {
        results.push(normalized)
      }
      return ''
    })
    String(html || '').replace(/url\((['"]?)(.*?)\1\)/gi, (_, quote, url) => {
      const normalized = normalizeUrl(url)
      if (isLikelyProductImage(normalized)) {
        results.push(normalized)
      }
      return ''
    })
    return uniqueStrings(results)
  }

  function findImagesInPlainObject(value) {
    const results = []
    scanObject(value, (child, key, parent, path) => {
      const pathText = (path || []).join('.')
      if (!/(image|img|pic|url|图片|图)/i.test(`${key || ''} ${pathText}`)) {
        return
      }
      if (typeof child === 'string' && isLikelyProductImage(child)) {
        results.push(normalizeUrl(child))
      }
      if (Array.isArray(child)) {
        child.forEach((item) => {
          if (typeof item === 'string' && isLikelyProductImage(item)) {
            results.push(normalizeUrl(item))
          }
        })
      }
    })
    return uniqueStrings(results)
  }

  function stripHtml(value) {
    return cleanText(String(value || '')
      .replace(/<script[\s\S]*?<\/script>/gi, ' ')
      .replace(/<style[\s\S]*?<\/style>/gi, ' ')
      .replace(/<[^>]+>/g, ' '))
  }

  function formatPlainValue(value) {
    if (value === null || typeof value === 'undefined') {
      return ''
    }
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
      return String(value)
    }
    try {
      return JSON.stringify(value)
    } catch (error) {
      return String(value)
    }
  }

  function extractOfferIdFromPath() {
    return window.location.pathname.match(DETAIL_PATH_PATTERN)?.[1] || ''
  }

  function parseMinPrice(priceText, roots, skuOffers) {
    const direct = parseNumberValue(firstStringValue(findValuesByKeys(roots, ['priceMin', 'beginPrice', 'minPrice', 'minSalePrice'])))
    if (typeof direct === 'number') {
      return direct
    }
    const prices = extractSkuPrices(skuOffers)
    if (prices.length) {
      return Math.min(...prices)
    }
    return parseNumberFromRange(priceText, 0)
  }

  function parseMaxPrice(priceText, roots, skuOffers) {
    const direct = parseNumberValue(firstStringValue(findValuesByKeys(roots, ['priceMax', 'endPrice', 'maxPrice', 'maxSalePrice'])))
    if (typeof direct === 'number') {
      return direct
    }
    const prices = extractSkuPrices(skuOffers)
    if (prices.length) {
      return Math.max(...prices)
    }
    return parseNumberFromRange(priceText, 1)
  }

  function extractSkuPrices(skuOffers) {
    return (skuOffers || [])
      .map((offer) => parseNumberValue(offer?.price ?? offer?.priceText))
      .filter((value) => typeof value === 'number')
  }

  function parseNumberFromRange(text, index) {
    const numbers = String(text || '').match(/\d+(?:\.\d+)?/g) || []
    if (!numbers.length) {
      return null
    }
    const value = Number(numbers[Math.min(index, numbers.length - 1)])
    return Number.isFinite(value) ? value : null
  }

  function parseNumberValue(value) {
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value
    }
    const matched = String(value || '').match(/\d+(?:\.\d+)?/)
    if (!matched) {
      return null
    }
    const parsed = Number(matched[0])
    return Number.isFinite(parsed) ? parsed : null
  }

  function formatPriceText(price, rawValue) {
    if (cleanText(rawValue).includes('¥') || cleanText(rawValue).includes('￥')) {
      return cleanText(rawValue)
    }
    if (typeof price === 'number') {
      return `¥${price}`
    }
    return ''
  }

  function formatStockText(stock, rawValue) {
    if (/库存|可售|现货|有货/.test(cleanText(rawValue))) {
      return cleanText(rawValue)
    }
    if (typeof stock === 'number') {
      return `库存${stock}`
    }
    return ''
  }

  function inferCurrencyCode(priceText) {
    if (/[￥¥]/.test(priceText || '')) {
      return 'CNY'
    }
    return ''
  }

  function firstStringValue(values) {
    for (const value of values) {
      if (typeof value === 'string' && cleanText(value)) {
        return cleanText(value)
      }
      if (typeof value === 'number' && Number.isFinite(value)) {
        return String(value)
      }
    }
    return ''
  }

  function firstValueByPossibleKeys(object, keys) {
    if (!object || typeof object !== 'object') {
      return ''
    }
    const normalized = Object.fromEntries(Object.entries(object).map(([key, value]) => [normalizeKey(key), value]))
    for (const key of keys) {
      const value = normalized[normalizeKey(key)]
      if (value !== null && typeof value !== 'undefined' && cleanText(value)) {
        return value
      }
    }
    return ''
  }

  function firstObjectByPossibleKeys(object, keys) {
    if (!object || typeof object !== 'object') {
      return null
    }
    const normalized = Object.fromEntries(Object.entries(object).map(([key, value]) => [normalizeKey(key), value]))
    for (const key of keys) {
      const value = normalized[normalizeKey(key)]
      if (value && typeof value === 'object' && !Array.isArray(value)) {
        return value
      }
    }
    return null
  }

  function firstArrayByPossibleKeys(object, keys) {
    if (!object || typeof object !== 'object') {
      return []
    }
    const normalized = Object.fromEntries(Object.entries(object).map(([key, value]) => [normalizeKey(key), value]))
    for (const key of keys) {
      const value = normalized[normalizeKey(key)]
      if (Array.isArray(value)) {
        return value
      }
    }
    return []
  }

  function firstImageFromObject(object) {
    if (!object || typeof object !== 'object') {
      return ''
    }
    const direct = firstValueByPossibleKeys(object, [
      'imageUrl',
      'image',
      'img',
      'pic',
      'picUrl',
      'skuImageUrl',
      'skuPicUrl',
      'originalImage',
      'url'
    ])
    if (typeof direct === 'string' && isLikelyProductImage(direct)) {
      return normalizeUrl(direct)
    }
    for (const value of Object.values(object)) {
      if (typeof value === 'string' && isLikelyProductImage(value)) {
        return normalizeUrl(value)
      }
    }
    return ''
  }

  function normalizeUrl(rawUrl) {
    let value = String(rawUrl || '').trim()
    if (!value) {
      return ''
    }
    value = value.replace(/&amp;/g, '&')
    if (/^\/\//.test(value)) {
      value = `https:${value}`
    }
    try {
      return new URL(value, window.location.href).toString()
    } catch (error) {
      return value
    }
  }

  function looksLikeImageUrl(value) {
    return /^https?:\/\//i.test(value || '') && /\.(?:jpg|jpeg|png|webp|avif)(?:[?#].*)?$/i.test(value || '')
  }

  function normalizeSkuKeyText(value) {
    return cleanText(String(value || '')
      .replace(/[>;|]+/g, ' / ')
      .replace(/\d+:\d+/g, '')
      .replace(/[_-]+/g, ' '))
  }

  function formatSkuAttrs(attrs) {
    if (!attrs || typeof attrs !== 'object') {
      return ''
    }
    return cleanText(Object.values(attrs).map((value) => {
      if (value && typeof value === 'object') {
        return firstValueByPossibleKeys(value, ['name', 'value', 'valueName', 'title', 'label'])
      }
      return value
    }).filter(Boolean).join(' / '))
  }

  function compactObject(value) {
    return Object.fromEntries(Object.entries(value).filter(([, child]) => {
      if (child === null || typeof child === 'undefined') {
        return false
      }
      if (typeof child === 'string') {
        return child.trim() !== ''
      }
      if (Array.isArray(child)) {
        return child.length > 0
      }
      if (typeof child === 'object') {
        return Object.keys(child).length > 0
      }
      return true
    }))
  }

  function clonePlainObject(value) {
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
      return {}
    }
    return { ...value }
  }

  function cleanText(value) {
    return String(value || '').replace(/\s+/g, ' ').trim()
  }

  function uniqueStrings(values) {
    const result = []
    const seen = new Set()
    values.forEach((value) => {
      const normalized = cleanText(value)
      if (!normalized || seen.has(normalized)) {
        return
      }
      seen.add(normalized)
      result.push(normalized)
    })
    return result
  }

  function uniqueObjectsByKey(values, resolver) {
    const result = []
    const seen = new Set()
    values.forEach((value) => {
      const key = resolver(value)
      if (!key || seen.has(key)) {
        return
      }
      seen.add(key)
      result.push(value)
    })
    return result
  }

  function normalizeKey(value) {
    return String(value || '').replace(/[^a-z0-9\u4e00-\u9fa5]/gi, '').toLowerCase()
  }

  function collectWarnings(payload) {
    const warnings = []
    if (!payload.offerId) {
      warnings.push('未提取到 offerId')
    }
    if (!payload.title) {
      warnings.push('未提取到标题')
    }
    if (!payload.mainImageUrl) {
      warnings.push('未提取到真实商品主图')
    }
    if (!Array.isArray(payload.skuOffers) || !payload.skuOffers.length) {
      warnings.push('未提取到 SKU 报价')
    }
    return warnings
  }
})()
