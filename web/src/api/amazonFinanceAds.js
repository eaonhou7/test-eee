import service from '@/utils/request'

export const getAmazonFinanceAdsList = (data) => {
  return service({
    url: '/amazonFinanceAds/list',
    method: 'post',
    data
  })
}

export const importAmazonFinanceAds = (data) => {
  return service({
    url: '/amazonFinanceAds/import',
    method: 'post',
    data
  })
}
