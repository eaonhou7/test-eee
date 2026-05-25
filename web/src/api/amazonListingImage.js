import service from '@/utils/request'

export const uploadAmazonListingImage = (file) => {
  const formData = new FormData()
  formData.append('file', file)
  return service({
    url: '/amazonListingImage/upload',
    method: 'post',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

export const deleteAmazonListingImage = (data) => {
  return service({
    url: '/amazonListingImage/delete',
    method: 'delete',
    data
  })
}

export const sortAmazonListingImage = (data) => {
  return service({
    url: '/amazonListingImage/sort',
    method: 'post',
    data
  })
}
