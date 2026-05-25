export const parseContentDispositionFileName = (contentDisposition, fallbackName = 'download.xlsx') => {
  const value = String(contentDisposition || '')
  const utf8Match = value.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1]).replace(/["']/g, '')
    } catch (error) {
      return utf8Match[1].replace(/["']/g, '')
    }
  }

  const plainMatch = value.match(/filename="?([^"]+)"?/i)
  if (plainMatch?.[1]) {
    return plainMatch[1].replace(/["']/g, '')
  }
  return fallbackName
}

export const normalizeBlobResponse = async (response, fallbackFileName = 'download.xlsx') => {
  const blob = response instanceof Blob ? response : response?.data instanceof Blob ? response.data : null
  if (!blob) {
    throw new Error('下载失败，未获取到文件内容')
  }

  const headers = response?.headers || {}
  const contentType = String(headers['content-type'] || blob.type || '').toLowerCase()
  if (contentType.includes('application/json') || contentType.includes('text/plain')) {
    const text = await blob.text()
    try {
      const parsed = JSON.parse(text)
      throw new Error(parsed.msg || '下载失败')
    } catch (error) {
      throw new Error(error?.message || text || '下载失败')
    }
  }

  return {
    blob,
    fileName: parseContentDispositionFileName(headers['content-disposition'], fallbackFileName)
  }
}

export const triggerBlobDownload = (blob, fileName) => {
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}
