import service from '@/utils/request'

export const getAmazonFinanceReceivables = (data) => {
  return service({
    url: '/amazonFinanceArap/receivables',
    method: 'post',
    data
  })
}

export const getAmazonFinancePayables = (data) => {
  return service({
    url: '/amazonFinanceArap/payables',
    method: 'post',
    data
  })
}

export const getAmazonFinancePayments = (data) => {
  return service({
    url: '/amazonFinanceArap/payments',
    method: 'post',
    data
  })
}

export const saveAmazonFinancePayment = (data) => {
  return service({
    url: '/amazonFinanceArap/savePayment',
    method: 'post',
    data
  })
}
