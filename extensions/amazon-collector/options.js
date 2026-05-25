const apiBaseUrlInput = document.getElementById('apiBaseUrl')
const apiTokenInput = document.getElementById('apiToken')
const saveButton = document.getElementById('saveButton')
const testButton = document.getElementById('testButton')
const statusNode = document.getElementById('status')

init()

async function init() {
  const response = await chrome.runtime.sendMessage({ type: 'GET_SETTINGS' })
  if (response?.ok) {
    apiBaseUrlInput.value = response.data.apiBaseUrl || ''
    apiTokenInput.value = response.data.apiToken || ''
  }
}

saveButton.addEventListener('click', async () => {
  setStatus('正在保存配置...')
  const response = await chrome.runtime.sendMessage({
    type: 'SAVE_SETTINGS',
    payload: {
      apiBaseUrl: apiBaseUrlInput.value,
      apiToken: apiTokenInput.value
    }
  })
  if (!response?.ok) {
    setStatus(response?.message || '保存失败', 'error')
    return
  }
  setStatus(response.message || '配置已保存', 'success')
})

testButton.addEventListener('click', async () => {
  setStatus('正在测试连接...')
  await chrome.runtime.sendMessage({
    type: 'SAVE_SETTINGS',
    payload: {
      apiBaseUrl: apiBaseUrlInput.value,
      apiToken: apiTokenInput.value
    }
  })
  const response = await chrome.runtime.sendMessage({ type: 'TEST_CONNECTION' })
  if (!response?.ok) {
    setStatus(response?.message || '连接测试失败', 'error')
    return
  }
  setStatus(response.message || '连接测试成功', 'success')
})

function setStatus(message, tone = '') {
  statusNode.textContent = message
  statusNode.className = `status ${tone}`.trim()
}
