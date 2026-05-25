import service from '@/utils/request'

export const upsertAmazonCollectedProductFromExtension = (data, options = {}) => {
  return service({
    url: '/amazonCollector/extension/upsertDetail',
    method: 'post',
    data,
    ...options
  })
}

export const getAmazonCollectedProductList = (data) => {
  return service({
    url: '/amazonCollector/list',
    method: 'post',
    data
  })
}

export const findAmazonCollectedProduct = (params) => {
  return service({
    url: '/amazonCollector/find',
    method: 'get',
    params
  })
}

export const deleteAmazonCollectedProduct = (data) => {
  return service({
    url: '/amazonCollector/delete',
    method: 'delete',
    data
  })
}

export const rebindAmazonCollectedProductImages = (data) => {
  return service({
    url: '/amazonCollector/rebindImages',
    method: 'post',
    data
  })
}

export const updateAmazonCollectedProductRisk = (data) => {
  return service({
    url: '/amazonCollector/updateRisk',
    method: 'post',
    data
  })
}

export const getAmazonCollectedProductCategories = (params) => {
  return service({
    url: '/amazonCollector/categories',
    method: 'get',
    params
  })
}

export const downloadAmazonCollectorExtension = (params) => {
  return service({
    url: '/amazonCollector/downloadLatest',
    method: 'get',
    params,
    responseType: 'blob'
  })
}

export const syncAmazonCollectedProductToListing = (data) => {
  return service({
    url: '/amazonCollector/syncToListing',
    method: 'post',
    data
  })
}
