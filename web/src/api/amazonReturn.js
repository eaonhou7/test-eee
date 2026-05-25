import service from '@/utils/request'

export const getAmazonReturnList = (data) => {
  return service({
    url: '/amazonReturn/list',
    method: 'post',
    data
  })
}

export const findAmazonReturn = (params) => {
  return service({
    url: '/amazonReturn/find',
    method: 'get',
    params
  })
}

export const resyncAmazonReturns = (data) => {
  return service({
    url: '/amazonReturn/resync',
    method: 'post',
    data
  })
}

export const recomputeAmazonReturnDecision = (data) => {
  return service({
    url: '/amazonReturn/recomputeDecision',
    method: 'post',
    data
  })
}

export const relinkAmazonReturnOriginalOrder = (data) => {
  return service({
    url: '/amazonReturn/relinkOriginalOrder',
    method: 'post',
    data
  })
}

export const confirmAmazonReturnRedirect = (data) => {
  return service({
    url: '/amazonReturn/confirmRedirect',
    method: 'post',
    data
  })
}

export const confirmAmazonReturnWarehouse = (data) => {
  return service({
    url: '/amazonReturn/confirmWarehouseReturn',
    method: 'post',
    data
  })
}

export const overrideAmazonReturnDecision = (data) => {
  return service({
    url: '/amazonReturn/overrideDecision',
    method: 'post',
    data
  })
}

export const releaseAmazonReturnRedirect = (data) => {
  return service({
    url: '/amazonReturn/releaseRedirect',
    method: 'post',
    data
  })
}
