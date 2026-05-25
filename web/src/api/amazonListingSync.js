import service from '@/utils/request'

export const previewAmazonListingSync = (data) => {
  return service({
    url: '/amazonListingSync/preview',
    method: 'post',
    data
  })
}

export const submitAmazonListingSync = (data) => {
  return service({
    url: '/amazonListingSync/submit',
    method: 'post',
    data
  })
}

export const getAmazonListingSyncList = (data) => {
  return service({
    url: '/amazonListingSync/list',
    method: 'post',
    data
  })
}

export const findAmazonListingSync = (params) => {
  return service({
    url: '/amazonListingSync/find',
    method: 'get',
    params
  })
}

export const refreshAmazonListingSyncStatus = (data) => {
  return service({
    url: '/amazonListingSync/refreshStatus',
    method: 'post',
    data
  })
}

export const resyncAmazonFbaInventory = (data) => {
  return service({
    url: '/amazonListingSync/resyncFbaInventory',
    method: 'post',
    data
  })
}
