import service from '@/utils/request'

export const getAmazonTemplateList = (data) => {
  return service({
    url: '/amazonTemplate/list',
    method: 'post',
    data
  })
}

export const findAmazonTemplate = (params) => {
  return service({
    url: '/amazonTemplate/find',
    method: 'get',
    params
  })
}

export const createAmazonTemplate = (data) => {
  return service({
    url: '/amazonTemplate/create',
    method: 'post',
    data
  })
}

export const updateAmazonTemplate = (data) => {
  return service({
    url: '/amazonTemplate/update',
    method: 'put',
    data
  })
}

export const deleteAmazonTemplate = (data) => {
  return service({
    url: '/amazonTemplate/delete',
    method: 'delete',
    data
  })
}

export const uploadAmazonTemplateWorkbook = (templateId, file) => {
  const formData = new FormData()
  formData.append('templateId', String(templateId))
  formData.append('file', file)
  return service({
    url: '/amazonTemplate/uploadWorkbook',
    method: 'post',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

export const parseAmazonTemplateWorkbook = (params) => {
  return service({
    url: '/amazonTemplate/parseWorkbook',
    method: 'get',
    params
  })
}

export const downloadAmazonTemplateWorkbook = (params) => {
  return service({
    url: '/amazonTemplate/downloadWorkbook',
    method: 'get',
    params,
    responseType: 'blob'
  })
}

export const saveAmazonTemplateFieldRules = (data) => {
  return service({
    url: '/amazonTemplate/saveFieldRules',
    method: 'post',
    data
  })
}
