import service from '@/utils/request'

export const getAmazonReturnProviderList = (data) => {
  return service({
    url: '/amazonReturnProvider/list',
    method: 'post',
    data
  })
}

export const findAmazonReturnProvider = (params) => {
  return service({
    url: '/amazonReturnProvider/find',
    method: 'get',
    params
  })
}

export const saveAmazonReturnProvider = (data) => {
  return service({
    url: '/amazonReturnProvider/save',
    method: 'post',
    data
  })
}

export const deleteAmazonReturnProvider = (data) => {
  return service({
    url: '/amazonReturnProvider/delete',
    method: 'post',
    data
  })
}

export const testAmazonReturnProviderConnection = (data) => {
  return service({
    url: '/amazonReturnProvider/testConnection',
    method: 'post',
    data
  })
}
