import service from '@/utils/request'

export const previewAmazonListingPublish = (data) => {
  return service({
    url: '/amazonListingPublish/preview',
    method: 'post',
    data
  })
}

export const submitAmazonListingPublish = (data) => {
  return service({
    url: '/amazonListingPublish/submit',
    method: 'post',
    data
  })
}

export const getAmazonListingPublishList = (data) => {
  return service({
    url: '/amazonListingPublish/list',
    method: 'post',
    data
  })
}

export const findAmazonListingPublish = (params) => {
  return service({
    url: '/amazonListingPublish/find',
    method: 'get',
    params
  })
}
