import service from '@/utils/request'

export const getAmazonOrderList = (data) => {
  return service({
    url: '/amazonOrder/list',
    method: 'post',
    data
  })
}

export const findAmazonOrder = (params) => {
  return service({
    url: '/amazonOrder/find',
    method: 'get',
    params
  })
}

export const resyncAmazonOrder = (data) => {
  return service({
    url: '/amazonOrder/resync',
    method: 'post',
    data
  })
}

export const startAmazonOrderFulfillment = (data) => {
  return service({
    url: '/amazonOrder/startFulfillment',
    method: 'post',
    data
  })
}

export const retryAmazonOrderFulfillment = (data) => {
  return service({
    url: '/amazonOrder/retryFulfillment',
    method: 'post',
    data
  })
}

export const printAmazonOrderSystemSlip = (data) => {
  return service({
    url: '/amazonOrder/printSystemSlip',
    method: 'post',
    data
  })
}

export const updateAmazonOrderPackageOverrides = (data) => {
  return service({
    url: '/amazonOrder/updatePackageOverrides',
    method: 'post',
    data
  })
}

export const manualAmazonOrderShipmentConfirm = (data) => {
  return service({
    url: '/amazonOrder/manualShipmentConfirm',
    method: 'post',
    data
  })
}

export const retryAmazonOrderShipmentConfirm = (data) => {
  return service({
    url: '/amazonOrder/retryShipmentConfirm',
    method: 'post',
    data
  })
}
