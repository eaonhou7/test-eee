import service from '@/utils/request'

export const findAmazon1688ProcurementTask = (params) => {
  return service({
    url: '/amazon1688Procurement/task/find',
    method: 'get',
    params
  })
}
