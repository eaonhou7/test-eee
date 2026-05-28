import service from '@/utils/request'

export const getAmazonFinanceQuestionList = (data) => {
  return service({
    url: '/amazonFinanceQuestion/list',
    method: 'post',
    data
  })
}

export const findAmazonFinanceQuestion = (params) => {
  return service({
    url: '/amazonFinanceQuestion/find',
    method: 'get',
    params
  })
}

export const getAmazonFinanceQuestionTypes = () => {
  return service({
    url: '/amazonFinanceQuestion/types',
    method: 'get'
  })
}

export const saveAmazonFinanceQuestion = (data) => {
  return service({
    url: '/amazonFinanceQuestion/save',
    method: 'post',
    data
  })
}
