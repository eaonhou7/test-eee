import service from '@/utils/request'

export const getAmazonDashboardOverview = (data) => {
  return service({
    url: '/amazonDashboard/overview',
    method: 'post',
    data,
    donNotShowLoading: true
  })
}
