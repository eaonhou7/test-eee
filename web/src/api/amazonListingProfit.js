import service from '@/utils/request'

export const calculateAmazonListingProfit = (data) => {
  return service({
    url: '/amazonListingProfit/calculate',
    method: 'post',
    data
  })
}
