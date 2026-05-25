import service from '@/utils/request'

export const getAmazonStoreList = (data) => {
  return service({
    url: '/amazonStore/list',
    method: 'post',
    data
  })
}

export const findAmazonStore = (params) => {
  return service({
    url: '/amazonStore/find',
    method: 'get',
    params
  })
}

export const saveAmazonStore = (data) => {
  return service({
    url: '/amazonStore/upsert',
    method: 'post',
    data
  })
}

export const deleteAmazonStore = (data) => {
  return service({
    url: '/amazonStore/delete',
    method: 'post',
    data
  })
}

export const startAmazonStoreAuth = (params) => {
  return service({
    url: '/amazonStore/authStart',
    method: 'get',
    params
  })
}

export const testAmazonStoreConnection = (data) => {
  return service({
    url: '/amazonStore/testConnection',
    method: 'post',
    data
  })
}

export const syncAmazonStoreOrdersNow = (data) => {
  return service({
    url: '/amazonStore/syncOrdersNow',
    method: 'post',
    data
  })
}
