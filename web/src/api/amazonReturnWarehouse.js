import service from '@/utils/request'

export const getAmazonReturnWarehouseList = (data) => {
  return service({
    url: '/amazonReturnWarehouse/list',
    method: 'post',
    data
  })
}

export const findAmazonReturnWarehouse = (params) => {
  return service({
    url: '/amazonReturnWarehouse/find',
    method: 'get',
    params
  })
}

export const saveAmazonReturnWarehouse = (data) => {
  return service({
    url: '/amazonReturnWarehouse/save',
    method: 'post',
    data
  })
}

export const deleteAmazonReturnWarehouse = (data) => {
  return service({
    url: '/amazonReturnWarehouse/delete',
    method: 'post',
    data
  })
}
