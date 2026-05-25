import service from '@/utils/request'

export const getAmazonSupportTemplateList = (data) => {
  return service({
    url: '/amazonSupportTemplate/list',
    method: 'post',
    data
  })
}

export const findAmazonSupportTemplate = (params) => {
  return service({
    url: '/amazonSupportTemplate/find',
    method: 'get',
    params
  })
}

export const saveAmazonSupportTemplate = (data) => {
  return service({
    url: '/amazonSupportTemplate/save',
    method: 'post',
    data
  })
}

export const deleteAmazonSupportTemplate = (data) => {
  return service({
    url: '/amazonSupportTemplate/delete',
    method: 'post',
    data
  })
}
