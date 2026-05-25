import service from '@/utils/request'

export const requestUSLogisticsQuotes = (data) => {
  return service({
    url: '/amazonLogistics/quoteUS',
    method: 'post',
    data
  })
}
