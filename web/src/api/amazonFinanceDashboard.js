import service from '@/utils/request'

export const getAmazonFinanceDashboardOverview = (data) => {
  return service({
    url: '/amazonFinanceDashboard/overview',
    method: 'post',
    data
  })
}
