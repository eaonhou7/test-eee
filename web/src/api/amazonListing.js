import service from '@/utils/request'

export const getAmazonListingTree = (data) => {
  return service({
    url: '/amazonListing/list',
    method: 'post',
    data
  })
}

export const findAmazonListing = (params) => {
  return service({
    url: '/amazonListing/find',
    method: 'get',
    params
  })
}

export const saveAmazonListing = (data) => {
  return service({
    url: '/amazonListing/save',
    method: 'post',
    data
  })
}

export const deleteAmazonListing = (data) => {
  return service({
    url: '/amazonListing/delete',
    method: 'delete',
    data
  })
}

export const validateAmazonListingItem = (data) => {
  return service({
    url: '/amazonListing/validateItem',
    method: 'post',
    data
  })
}

export const validateAmazonListingSelected = (data) => {
  return service({
    url: '/amazonListing/validateSelected',
    method: 'post',
    data
  })
}

export const exportAmazonListingSelected = (data) => {
  return service({
    url: '/amazonListing/exportSelected',
    method: 'post',
    data
  })
}

export const getAmazonFamilyList = (data) => {
  return service({
    url: '/amazonListingFamily/list',
    method: 'post',
    data
  })
}
