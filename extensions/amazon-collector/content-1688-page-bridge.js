(function () {
  const SOURCE = 'amazon-collector-1688'

  if (window.__amazonCollector1688PageBridgeInstalled) {
    return
  }
  window.__amazonCollector1688PageBridgeInstalled = true

  window.addEventListener('message', async (event) => {
    if (event.source !== window) {
      return
    }
    const request = event.data || {}
    if (request.source !== SOURCE || request.type !== 'UPLOAD_IMAGE_WITH_MTOP') {
      return
    }

    try {
      const imageBase64 = extractBase64(request.dataUrl)
      const mtop = await waitForMtop()
      mtop.config.prefix = 'h5api'
      mtop.config.subDomain = 'm'
      mtop.config.mainDomain = '1688.com'

      const result = await mtop.request({
        api: 'mtop.1688.imageService.putImage',
        data: JSON.stringify({
          imageBase64,
          appName: 'searchImageUpload',
          appKey: 'pvvljh1grxcmaay2vgpe9nb68gg9ueg2'
        }),
        ecode: '0',
        v: '1.0',
        type: 'POST'
      })
      const imageId = String(result?.data?.imageId || '').trim()
      const imageAddress = String(result?.data?.imageAddress || '').trim()
      const imageUrl = String(result?.data?.imageUrl || result?.data?.url || '').trim()
      const key = String(result?.data?.key || result?.data?.objectKey || result?.data?.name || '').trim()
      if (!imageId && !imageAddress && !imageUrl && !key) {
        throw new Error((result?.ret || []).join(';') || '1688 未返回 imageId')
      }
      window.postMessage({
        source: SOURCE,
        type: 'UPLOAD_IMAGE_WITH_MTOP_RESULT',
        requestId: request.requestId,
        ok: true,
        imageId,
        imageAddress,
        imageUrl,
        key
      }, '*')
    } catch (error) {
      window.postMessage({
        source: SOURCE,
        type: 'UPLOAD_IMAGE_WITH_MTOP_RESULT',
        requestId: request.requestId,
        ok: false,
        message: error?.message || '1688 图片上传失败'
      }, '*')
    }
  })

  function extractBase64(dataUrl) {
    const value = String(dataUrl || '').trim()
    const marker = 'base64,'
    const index = value.indexOf(marker)
    const base64 = index >= 0 ? value.slice(index + marker.length) : value
    if (!base64) {
      throw new Error('图片内容为空')
    }
    return base64
  }

  function waitForMtop() {
    return new Promise((resolve, reject) => {
      const deadline = Date.now() + 10000
      const timer = window.setInterval(() => {
        const mtop = window.lib?.mtop
        if (mtop?.request && mtop?.config) {
          window.clearInterval(timer)
          resolve(mtop)
          return
        }
        if (Date.now() > deadline) {
          window.clearInterval(timer)
          reject(new Error('1688 mtop 未就绪'))
        }
      }, 200)
    })
  }
})()
