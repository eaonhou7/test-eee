import service from '@/utils/request'

export const createAmazon1688CollectTask = (data) => {
  return service({
    url: '/amazon1688Collector/task/create',
    method: 'post',
    data
  })
}

export const reportAmazon1688CollectTaskState = (data, options = {}) => {
  return service({
    url: '/amazon1688Collector/task/reportState',
    method: 'post',
    data,
    ...options
  })
}

export const createAmazon1688RepairTask = (data) => {
  return service({
    url: '/amazon1688Collector/repair/createTask',
    method: 'post',
    data
  })
}

export const upsertAmazon1688CollectedProductFromExtension = (data, options = {}) => {
  return service({
    url: '/amazon1688Collector/extension/upsertDetail',
    method: 'post',
    data,
    ...options
  })
}

export const getAmazon1688CollectedProductList = (data) => {
  return service({
    url: '/amazon1688Collector/list',
    method: 'post',
    data
  })
}

export const findAmazon1688CollectedProduct = (params) => {
  return service({
    url: '/amazon1688Collector/find',
    method: 'get',
    params
  })
}

export const deleteAmazon1688CollectedProduct = (data) => {
  return service({
    url: '/amazon1688Collector/delete',
    method: 'delete',
    data
  })
}

export const downloadAmazon1688CollectorExtension = (params) => {
  return service({
    url: '/amazon1688Collector/downloadLatest',
    method: 'get',
    params,
    responseType: 'blob'
  })
}

export const upsertAmazon1688BindingVariantMapping = (data) => {
  return service({
    url: '/amazon1688Collector/binding/upsertVariantMapping',
    method: 'post',
    data
  })
}
