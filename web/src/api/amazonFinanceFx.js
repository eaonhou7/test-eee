import service from '@/utils/request'

export const getAmazonFinanceFxList = (data) => {
  return service({
    url: '/amazonFinanceFx/list',
    method: 'post',
    data
  })
}

export const saveAmazonFinanceFxOverride = (data) => {
  return service({
    url: '/amazonFinanceFx/saveOverride',
    method: 'post',
    data
  })
}

export const refreshAmazonFinanceFxRates = () => {
  return service({
    url: '/amazonFinanceFx/refreshDailyRates',
    method: 'post'
  })
}
