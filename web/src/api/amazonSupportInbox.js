import service from '@/utils/request'

export const getAmazonSupportInboxList = (data) => {
  return service({
    url: '/amazonSupportInbox/list',
    method: 'post',
    data
  })
}

export const findAmazonSupportCase = (params) => {
  return service({
    url: '/amazonSupportInbox/find',
    method: 'get',
    params
  })
}

export const saveAmazonSupportCase = (data) => {
  return service({
    url: '/amazonSupportInbox/upsertCase',
    method: 'post',
    data
  })
}

export const markAmazonSupportCaseRead = (data) => {
  return service({
    url: '/amazonSupportInbox/markRead',
    method: 'post',
    data
  })
}

export const markAmazonSupportCasePending = (data) => {
  return service({
    url: '/amazonSupportInbox/markPending',
    method: 'post',
    data
  })
}

export const closeAmazonSupportCase = (data) => {
  return service({
    url: '/amazonSupportInbox/close',
    method: 'post',
    data
  })
}

export const refreshAmazonSupportActions = (data) => {
  return service({
    url: '/amazonSupportInbox/refreshActions',
    method: 'post',
    data
  })
}

export const sendAmazonSupportReply = (data) => {
  return service({
    url: '/amazonSupportInbox/sendReply',
    method: 'post',
    data
  })
}

export const importAmazonSupportWorkbook = (file) => {
  const formData = new FormData()
  formData.append('file', file)
  return service({
    url: '/amazonSupportInbox/importWorkbook',
    method: 'post',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}
