import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'
import vm from 'node:vm'
import { fileURLToPath } from 'node:url'

const scriptPath = fileURLToPath(new URL('./content-1688.js', import.meta.url))
const scriptSource = fs.readFileSync(scriptPath, 'utf8')

function loadHooks() {
  const initialUrl = new URL('https://s.1688.com/shen/sell_offer.htm?tab=imageSearch')
  const window = {
    __AMAZON_COLLECTOR_1688_TEST__: true,
    location: {
      href: initialUrl.href,
      host: initialUrl.host,
      pathname: initialUrl.pathname
    }
  }
  vm.runInNewContext(scriptSource, {
    URL,
    window
  }, {
    filename: scriptPath
  })
  assert.ok(window.__amazonCollector1688TestHooks, 'expected content script test hooks')
  return window.__amazonCollector1688TestHooks
}

test('1688 source image task url triggers upload flow', () => {
  const hooks = loadHooks()
  const sourceImageUrl = 'https://m.media-amazon.com/images/I/source.jpg'
  const searchUrl = `https://s.1688.com/shen/sell_offer.htm?tab=imageSearch&__gva1688Task=task-1&__gva1688Image=${encodeURIComponent(sourceImageUrl)}`

  assert.equal(hooks.getSourceImageAddressFromUrl(searchUrl), sourceImageUrl)
  assert.equal(hooks.isUploadedImageSearchResultPage(searchUrl), false)
  assert.equal(hooks.shouldUploadImageForSearch(searchUrl), true)
})

test('1688 oss image result page is safe to decorate', () => {
  const hooks = loadHooks()
  const resultUrl = 'https://s.1688.com/shen/sell_offer.htm?tab=imageSearch&imageType=oss&imageAddress=img/search/O1CN01abc.jpg&__gva1688Task=task-1'

  assert.equal(hooks.getSourceImageAddressFromUrl(resultUrl), '')
  assert.equal(hooks.isUploadedImageSearchResultPage(resultUrl), true)
  assert.equal(hooks.isInvalidUploadedImageSearchResultPage(resultUrl), false)
  assert.equal(hooks.shouldUploadImageForSearch(resultUrl), false)
})

test('1688 mtop imageId result page is safe to decorate', () => {
  const hooks = loadHooks()
  const resultUrl = 'https://s.1688.com/shen/sell_offer.htm?tab=imageSearch&imageAddress=&imageId=1792708757601593298&spm=a26352.13672862.searchbox.input&imageIdList=1792708757601593298&__gva1688Task=task-1'

  assert.equal(hooks.getSourceImageAddressFromUrl(resultUrl), '')
  assert.equal(hooks.isUploadedImageSearchResultPage(resultUrl), true)
  assert.equal(hooks.isInvalidUploadedImageSearchResultPage(resultUrl), false)
  assert.equal(hooks.shouldUploadImageForSearch(resultUrl), false)
})

test('1688 numeric oss imageAddress is treated as invalid result page', () => {
  const hooks = loadHooks()
  const badResultUrl = 'https://s.1688.com/shen/sell_offer.htm?tab=imageSearch&imageType=oss&imageAddress=1792708757601593298&__gva1688Task=task-1'

  assert.equal(hooks.isUploadedImageSearchResultPage(badResultUrl), false)
  assert.equal(hooks.isInvalidUploadedImageSearchResultPage(badResultUrl), true)
  assert.equal(hooks.shouldUploadImageForSearch(badResultUrl), false)
})

test('1688 native imageAddress with external url is not treated as a result page', () => {
  const hooks = loadHooks()
  const sourceImageUrl = 'https://img.alicdn.com/example-main.jpg'
  const legacyUrl = `https://s.1688.com/shen/sell_offer.htm?tab=imageSearch&imageAddress=${encodeURIComponent(sourceImageUrl)}&__gva1688Task=task-1`

  assert.equal(hooks.getSourceImageAddressFromUrl(legacyUrl), sourceImageUrl)
  assert.equal(hooks.isUploadedImageSearchResultPage(legacyUrl), false)
  assert.equal(hooks.shouldUploadImageForSearch(legacyUrl), true)
})

test('1688 mtop numeric image id is used as imageId, not imageAddress', () => {
  const hooks = loadHooks()
  const imageId = '1792708757601593298'
  const target = hooks.getUsableUploadResultTarget({ imageId })
  const resultUrl = hooks.build1688ImageIdResultUrl(imageId, 'task-1')
  const parsed = new URL(resultUrl)

  assert.equal(target.type, 'imageId')
  assert.equal(target.value, imageId)
  assert.equal(hooks.getUsableUploadImageAddress({ imageId: '1792708757601593298' }), '')
  assert.equal(hooks.getUsableUploadImageAddress({ key: 'img/search/O1CN01abc.jpg' }), 'img/search/O1CN01abc.jpg')
  assert.equal(hooks.getUsableUploadImageAddress({ imageUrl: 'https://cbu01.alicdn.com/img/example.jpg' }), '')
  assert.equal(parsed.searchParams.get('imageAddress'), '')
  assert.equal(parsed.searchParams.get('imageId'), imageId)
  assert.equal(parsed.searchParams.get('imageIdList'), imageId)
  assert.equal(parsed.searchParams.get('__gva1688Task'), 'task-1')
})
