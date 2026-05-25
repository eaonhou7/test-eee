import service from '@/utils/request'

export const getAmazonFinanceReportSummary = (data) => {
  return service({
    url: '/amazonFinanceReport/summary',
    method: 'post',
    data
  })
}

export const getAmazonFinanceReportOrders = (data) => {
  return service({
    url: '/amazonFinanceReport/orders',
    method: 'post',
    data
  })
}

export const findAmazonFinanceOrderProfit = (params) => {
  return service({
    url: '/amazonFinanceReport/orderProfit',
    method: 'get',
    params
  })
}
