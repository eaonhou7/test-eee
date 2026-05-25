import service from '@/utils/request'

export const getAmazonFinanceSettlementList = (data) => {
  return service({
    url: '/amazonFinanceSettlement/list',
    method: 'post',
    data
  })
}

export const findAmazonFinanceSettlement = (params) => {
  return service({
    url: '/amazonFinanceSettlement/find',
    method: 'get',
    params
  })
}

export const importAmazonFinanceSettlement = (data) => {
  return service({
    url: '/amazonFinanceSettlement/import',
    method: 'post',
    data
  })
}

export const manualMatchAmazonFinanceSettlement = (data) => {
  return service({
    url: '/amazonFinanceSettlement/manualMatch',
    method: 'post',
    data
  })
}
