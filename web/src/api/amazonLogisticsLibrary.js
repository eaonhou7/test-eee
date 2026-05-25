import service from '@/utils/request'

export const uploadAmazonLogisticsWorkbook = (provider, file) => {
  const formData = new FormData()
  formData.append('provider', provider)
  formData.append('file', file)
  return service({
    url: '/amazonLogisticsLibrary/uploadWorkbook',
    method: 'post',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

export const getAmazonLogisticsChannelPage = (data) => {
  return service({
    url: '/amazonLogisticsLibrary/getChannelPage',
    method: 'post',
    data
  })
}

export const getAmazonLogisticsChannelDetail = (data) => {
  return service({
    url: '/amazonLogisticsLibrary/getChannelDetail',
    method: 'post',
    data
  })
}

export const getAmazonLogisticsRateRowPage = (data) => {
  return service({
    url: '/amazonLogisticsLibrary/getRateRowPage',
    method: 'post',
    data
  })
}

export const getAmazonLogisticsVersionPage = (data) => {
  return service({
    url: '/amazonLogisticsLibrary/getVersionPage',
    method: 'post',
    data
  })
}
