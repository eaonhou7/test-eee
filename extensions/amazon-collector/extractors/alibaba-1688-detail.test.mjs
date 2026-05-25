import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'
import vm from 'node:vm'

class FakeElement {
  constructor({ tagName = 'div', attrs = {}, text = '', html = '', children = [], naturalWidth = 0, naturalHeight = 0 } = {}) {
    this.tagName = tagName.toUpperCase()
    this.attrs = { ...attrs }
    this.id = attrs.id || ''
    this.className = attrs.class || attrs.className || ''
    this._text = text
    this._html = html
    this.children = children
    this.parentElement = null
    this.naturalWidth = naturalWidth
    this.naturalHeight = naturalHeight
    this.clientWidth = naturalWidth
    this.clientHeight = naturalHeight
    this.width = naturalWidth
    this.height = naturalHeight
    this.children.forEach((child) => {
      child.parentElement = this
    })
  }

  get textContent() {
    return this._text || this.children.map((child) => child.textContent).join(' ')
  }

  get innerText() {
    return this.textContent
  }

  get innerHTML() {
    return this._html || this.children.map((child) => child.innerHTML || child.textContent).join('')
  }

  getAttribute(name) {
    if (name === 'class') {
      return this.className
    }
    if (name === 'id') {
      return this.id
    }
    return this.attrs[name] || ''
  }

  getBoundingClientRect() {
    return {
      width: this.clientWidth || 0,
      height: this.clientHeight || 0
    }
  }

  matches(selector) {
    return selector.split(',').some((part) => matchSelectorPart(this, part.trim()))
  }

  querySelector(selector) {
    return this.querySelectorAll(selector)[0] || null
  }

  querySelectorAll(selector) {
    const result = []
    const visit = (node) => {
      if (matchesSelector(node, selector)) {
        result.push(node)
      }
      node.children.forEach(visit)
    }
    this.children.forEach(visit)
    return result
  }
}

class FakeDocument extends FakeElement {
  constructor(body) {
    super({ tagName: 'document', children: [body] })
    this.body = body
  }
}

function matchesSelector(node, selector) {
  return selector.split(',').some((part) => {
    const pieces = part.trim().split(/\s+/).filter(Boolean)
    if (!pieces.length) {
      return false
    }
    if (!matchSelectorPart(node, pieces[pieces.length - 1])) {
      return false
    }
    let current = node.parentElement
    for (let i = pieces.length - 2; i >= 0; i -= 1) {
      while (current && !matchSelectorPart(current, pieces[i])) {
        current = current.parentElement
      }
      if (!current) {
        return false
      }
      current = current.parentElement
    }
    return true
  })
}

function matchSelectorPart(node, part) {
  if (!part || part === '*') {
    return true
  }
  if (part.startsWith('#')) {
    return node.id === part.slice(1)
  }
  const tag = part.match(/^[a-z0-9-]+/i)?.[0]
  if (tag && node.tagName.toLowerCase() !== tag.toLowerCase()) {
    return false
  }
  const classMatches = [...part.matchAll(/\[class\*=["']([^"']+)["']\]/gi)]
  if (classMatches.some((match) => !String(node.className || '').toLowerCase().includes(match[1].toLowerCase()))) {
    return false
  }
  const attrContainsMatches = [...part.matchAll(/\[([a-z0-9_-]+)\*=["']([^"']+)["']\]/gi)]
  if (attrContainsMatches.some((match) => !String(node.getAttribute(match[1]) || '').toLowerCase().includes(match[2].toLowerCase()))) {
    return false
  }
  const attrMatches = [...part.matchAll(/\[([a-z0-9_-]+)(?:=["']([^"']+)["'])?\]/gi)]
  if (attrMatches.some((match) => {
    if (match[1].toLowerCase() === 'class') {
      return false
    }
    const value = node.getAttribute(match[1])
    return match[2] ? value !== match[2] : !value
  })) {
    return false
  }
  return true
}

function runExtractor({ state = {}, body }) {
  const document = new FakeDocument(body)
  const window = {
    location: {
      host: 'detail.1688.com',
      pathname: '/offer/1046906516992.html',
      href: 'https://detail.1688.com/offer/1046906516992.html'
    },
    __INIT_DATA__: state
  }
  const context = vm.createContext({
    console,
    document,
    URL,
    window
  })
  vm.runInContext(fs.readFileSync(new URL('./alibaba-1688-detail.js', import.meta.url), 'utf8'), context)
  return context.__amazonCollectorExtract1688Detail()
}

test('extracts trusted gallery image and keeps detail/sku images out of main image', () => {
  const mainUrl = 'https://cbu01.alicdn.com/img/ibank/O1CN-main.jpg'
  const galleryUrl = 'https://cbu01.alicdn.com/img/ibank/O1CN-gallery.jpg'
  const detailUrl = 'https://cbu01.alicdn.com/img/ibank/O1CN-detail.jpg'
  const skuUrl = 'https://cbu01.alicdn.com/img/ibank/O1CN-sku.jpg'
  const logoUrl = 'https://cbu01.alicdn.com/img/ibank/O1CN-logo.jpg'

  const body = new FakeElement({
    tagName: 'body',
    children: [
      new FakeElement({ tagName: 'h1', text: '测试商品' }),
      new FakeElement({
        attrs: { class: 'seller-card shop-card' },
        children: [new FakeElement({ tagName: 'img', attrs: { src: logoUrl }, naturalWidth: 500, naturalHeight: 500 })]
      }),
      new FakeElement({
        attrs: { class: 'detail-content' },
        text: '商品详情 这里是详情内容',
        children: [new FakeElement({ tagName: 'img', attrs: { src: detailUrl }, naturalWidth: 750, naturalHeight: 750 })]
      }),
      new FakeElement({
        attrs: { class: 'sku-row' },
        text: '粉色 ¥4.99 库存100个',
        children: [new FakeElement({ tagName: 'img', attrs: { src: skuUrl }, naturalWidth: 240, naturalHeight: 240 })]
      }),
      new FakeElement({
        attrs: { class: 'image-view product-gallery' },
        children: [
          new FakeElement({ tagName: 'img', attrs: { src: mainUrl }, naturalWidth: 800, naturalHeight: 800 }),
          new FakeElement({ tagName: 'img', attrs: { src: galleryUrl }, naturalWidth: 800, naturalHeight: 800 })
        ]
      })
    ]
  })

  const payload = runExtractor({
    state: {
      seller: { logoUrl },
      offer: {
        imageList: [mainUrl, galleryUrl],
        descImgs: [detailUrl]
      },
      skuModel: {
        skuInfoMap: {
          pink: {
            skuId: 'pink',
            skuKey: 'pink',
            priceText: '¥4.99',
            stockText: '库存100个',
            specAttrs: { 颜色: '粉色' },
            skuImageUrl: skuUrl
          }
        }
      },
      productAttributes: [{ name: '材质', value: 'PVC' }],
      packageInfo: {
        包装尺寸: '12*8*5cm',
        包装重量: '350g'
      }
    },
    body
  })

  assert.equal(payload.mainImageUrl, mainUrl)
  assert.ok(payload.galleryImageUrls.includes(galleryUrl))
  assert.ok(!payload.galleryImageUrls.includes(skuUrl))
  assert.ok(payload.detailImageUrls.includes(detailUrl))
  assert.equal(payload.skuOffers[0].imageUrl, skuUrl)
  assert.equal(payload.productAttributes['材质'], 'PVC')
  assert.equal(payload.packageInfo.weightKg, 0.35)
  assert.equal(payload.packageInfo.lengthCm, 12)
  assert.equal(payload.packageInfo.widthCm, 8)
  assert.equal(payload.packageInfo.heightCm, 5)
  assert.ok(payload.detailSections.length > 0)
})
