import service from '@/utils/request'

export const getAmazonFinanceCostBillList = (data) => {
  return service({
    url: '/amazonFinanceCostBill/list',
    method: 'post',
    data
  })
}

export const findAmazonFinanceCostBill = (params) => {
  return service({
    url: '/amazonFinanceCostBill/find',
    method: 'get',
    params
  })
}

export const saveAmazonFinanceCostBill = (data) => {
  return service({
    url: '/amazonFinanceCostBill/save',
    method: 'post',
    data
  })
}
