import { spawn } from 'node:child_process'
import { mkdtemp, rm } from 'node:fs/promises'
import { createRequire } from 'node:module'
import { tmpdir } from 'node:os'
import path from 'node:path'

const require = createRequire(new URL('../web/package.json', import.meta.url))
const WebSocket = require('ws')

const ROOT_URL = process.env.UI_URL || 'http://127.0.0.1:8080'
const CHROME_BIN = process.env.CHROME_BIN || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
const PORT = Number(process.env.CDP_PORT || 9331)
const TAG = `ui-crud-${Date.now().toString(36)}`

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

async function waitFor(fn, label, timeout = 15000, interval = 250) {
  const started = Date.now()
  let lastError
  while (Date.now() - started < timeout) {
    try {
      const value = await fn()
      if (value) return value
    } catch (error) {
      lastError = error
    }
    await sleep(interval)
  }
  throw new Error(`${label} timed out${lastError ? `: ${lastError.message}` : ''}`)
}

async function fetchJSON(url, options) {
  const response = await fetch(url, options)
  if (!response.ok) {
    throw new Error(`${url} responded ${response.status}`)
  }
  return response.json()
}

class CDP {
  constructor(wsUrl) {
    this.ws = new WebSocket(wsUrl)
    this.nextId = 1
    this.pending = new Map()
    this.handlers = new Map()
    this.ws.on('message', (data) => {
      const message = JSON.parse(String(data))
      if (message.id && this.pending.has(message.id)) {
        const { resolve, reject } = this.pending.get(message.id)
        this.pending.delete(message.id)
        if (message.error) reject(new Error(message.error.message))
        else resolve(message.result)
        return
      }
      const callbacks = this.handlers.get(message.method) || []
      for (const callback of callbacks) callback(message.params || {})
    })
  }

  async open() {
    await new Promise((resolve, reject) => {
      this.ws.once('open', resolve)
      this.ws.once('error', reject)
    })
  }

  on(method, callback) {
    const callbacks = this.handlers.get(method) || []
    callbacks.push(callback)
    this.handlers.set(method, callbacks)
  }

  send(method, params = {}) {
    const id = this.nextId++
    this.ws.send(JSON.stringify({ id, method, params }))
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject })
      setTimeout(() => {
        if (this.pending.has(id)) {
          this.pending.delete(id)
          reject(new Error(`${method} timed out`))
        }
      }, 20000)
    })
  }

  close() {
    this.ws.close()
  }
}

async function launchChrome() {
  const profileDir = await mkdtemp(path.join(tmpdir(), 'amazon-ui-chrome-'))
  const args = [
    `--remote-debugging-port=${PORT}`,
    `--user-data-dir=${profileDir}`,
    '--no-first-run',
    '--no-default-browser-check',
    '--disable-background-networking',
    '--disable-sync',
    '--disable-features=Translate,AutofillServerCommunication',
    '--window-size=1480,1120',
    'about:blank'
  ]
  const chrome = spawn(CHROME_BIN, args, { stdio: ['ignore', 'ignore', 'pipe'] })
  chrome.stderr.on('data', () => { })
  chrome.on('exit', (code, signal) => {
    if (code && !signal) console.error(`Chrome exited with code ${code}`)
  })
  await waitFor(() => fetchJSON(`http://127.0.0.1:${PORT}/json/version`), 'Chrome CDP')
  const target = await fetchJSON(`http://127.0.0.1:${PORT}/json/new?about:blank`, { method: 'PUT' })
  const cdp = new CDP(target.webSocketDebuggerUrl)
  await cdp.open()
  return { chrome, profileDir, cdp }
}

function serializable(value) {
  if (value === undefined) return undefined
  return JSON.stringify(value)
}

async function main() {
  const { chrome, profileDir, cdp } = await launchChrome()
  const events = {
    consoleErrors: [],
    exceptions: [],
    httpErrors: [],
    appErrors: [],
    loadingFailures: []
  }
  const responseByRequest = new Map()

  cdp.on('Runtime.consoleAPICalled', (params) => {
    if (params.type === 'error') {
      events.consoleErrors.push((params.args || []).map((arg) => arg.value || arg.description || '').join(' '))
    }
  })
  cdp.on('Runtime.exceptionThrown', (params) => {
    events.exceptions.push({
      text: params.exceptionDetails?.text || 'Runtime exception',
      description: params.exceptionDetails?.exception?.description || '',
      url: params.exceptionDetails?.url || '',
      line: params.exceptionDetails?.lineNumber,
      column: params.exceptionDetails?.columnNumber
    })
  })
  cdp.on('Network.responseReceived', (params) => {
    const url = params.response?.url || ''
    if (url.includes('/api/')) {
      responseByRequest.set(params.requestId, {
        url,
        status: params.response.status,
        mimeType: params.response.mimeType || ''
      })
      if (params.response.status >= 400) {
        events.httpErrors.push({ url, status: params.response.status })
      }
    }
  })
  cdp.on('Network.loadingFinished', (params) => {
    const meta = responseByRequest.get(params.requestId)
    if (!meta || !meta.mimeType.includes('json')) return
    void cdp.send('Network.getResponseBody', { requestId: params.requestId }).then((body) => {
      try {
        const parsed = JSON.parse(body.body)
        if (parsed && parsed.code !== undefined && parsed.code !== 0) {
          events.appErrors.push({
            url: meta.url.replace(ROOT_URL, ''),
            code: parsed.code,
            msg: parsed.msg || parsed.message || ''
          })
        }
      } catch { }
    }).catch(() => { })
  })
  cdp.on('Network.loadingFailed', (params) => {
    if (params.type !== 'Image') {
      events.loadingFailures.push({ id: params.requestId, errorText: params.errorText })
    }
  })

  const run = async (expression, label = 'evaluate', timeout = 20000) => {
    const result = await cdp.send('Runtime.evaluate', {
      expression,
      awaitPromise: true,
      returnByValue: true,
      timeout
    })
    if (result.exceptionDetails) {
      const detail = result.exceptionDetails.exception?.description || result.exceptionDetails.text
      throw new Error(`${label}: ${detail}`)
    }
    return result.result?.value
  }

  const call = (method, ...args) =>
    run(`window.__ui.${method}(${args.map(serializable).join(',')})`, method)

  const stepResults = []
  async function step(name, fn) {
    const started = Date.now()
    try {
      await fn()
      stepResults.push({ name, ok: true, ms: Date.now() - started })
      console.log(`PASS ${name}`)
    } catch (error) {
      const state = await run(`
          (() => {
            const visible = (el) => {
              if (!el) return false
              const style = getComputedStyle(el)
              const rect = el.getBoundingClientRect()
              return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0
            }
            const label = (el) => (el.innerText || el.textContent || el.getAttribute('aria-label') || '').replace(/\\s+/g, ' ').trim()
            return {
              href: location.href,
              hash: location.hash,
              title: document.title,
              route: document.querySelector('#app')?.__vue_app__?.config?.globalProperties?.$route
                ? {
                    name: document.querySelector('#app').__vue_app__.config.globalProperties.$route.name,
                    path: document.querySelector('#app').__vue_app__.config.globalProperties.$route.path,
                    matched: document.querySelector('#app').__vue_app__.config.globalProperties.$route.matched?.map((item) => ({
                      name: item.name,
                      path: item.path,
                      componentName: item.components?.default?.name || item.components?.default?.__name || String(item.components?.default || '').slice(0, 80)
                    }))
                  }
                : null,
              amazonRoutes: document.querySelector('#app')?.__vue_app__?.config?.globalProperties?.$router
                ?.getRoutes?.()
                ?.filter((item) => String(item.path).includes('amazon-logistics'))
                ?.map((item) => ({
                  name: item.name,
                  path: item.path,
                  componentName: item.components?.default?.name || item.components?.default?.__name || String(item.components?.default || '').slice(0, 80)
                })),
              buttons: [...document.querySelectorAll('button,.el-button,a')].filter(visible).map(label).filter(Boolean).slice(0, 80),
              body: label(document.body).slice(0, 2400)
            }
          })()
        `, 'capture failure state').catch((captureError) => ({ captureError: captureError.message }))
      console.log(`STATE ${name}: ${JSON.stringify(state, null, 2)}`)
      console.log(`BROWSER_EVENTS ${name}: ${JSON.stringify(events, null, 2)}`)
      stepResults.push({ name, ok: false, error: error.message, ms: Date.now() - started })
      console.log(`FAIL ${name}: ${error.message}`)
      throw error
    }
  }

  try {
    await cdp.send('Page.enable')
    await cdp.send('Page.setInterceptFileChooserDialog', { enabled: true }).catch(() => { })
    await cdp.send('Runtime.enable')
    await cdp.send('Network.enable')

    await step('UI 登录', async () => {
      await cdp.send('Page.navigate', { url: `${ROOT_URL}/#/login` })
      await cdp.send('Page.loadEventFired').catch(() => { })
      await run(`
        (async () => {
          const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
          const visible = (el) => {
            if (!el) return false
            const style = getComputedStyle(el)
            const rect = el.getBoundingClientRect()
            return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0
          }
          const set = (el, value) => {
            el.focus()
            const setter = Object.getOwnPropertyDescriptor(el instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype, 'value').set
            setter.call(el, value)
            el.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText', data: value }))
            el.dispatchEvent(new Event('change', { bubbles: true }))
          }
          const clickText = (text) => {
            const elements = [...document.querySelectorAll('button,.el-button,a')]
              .filter(visible)
              .filter((el) => (el.innerText || el.textContent || '').replace(/\\s+/g, ' ').trim().includes(text))
            if (!elements.length) throw new Error('button not found: ' + text)
            elements[0].scrollIntoView({ block: 'center', inline: 'center' })
            elements[0].click()
          }
          const waitUntil = async (fn, label) => {
            const started = Date.now()
            while (Date.now() - started < 15000) {
              if (fn()) return true
              await sleep(200)
            }
            throw new Error(label)
          }
          await waitUntil(() => document.querySelector('input[placeholder*="用户名"]'), 'login inputs missing')
          set(document.querySelector('input[placeholder*="用户名"]'), 'admin')
          set(document.querySelector('input[placeholder*="密码"]'), '123456')
          clickText('登 录')
          await waitUntil(() => !location.hash.includes('/login'), 'login did not redirect')
          return { hash: location.hash }
        })()
      `, 'login')
    })

    await run(`
      (() => {
        const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
        const text = (el) => (el?.innerText || el?.textContent || el?.getAttribute?.('aria-label') || el?.value || '').replace(/\\s+/g, ' ').trim()
        const visible = (el) => {
          if (!el) return false
          const style = getComputedStyle(el)
          const rect = el.getBoundingClientRect()
          return style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0' && rect.width > 0 && rect.height > 0
        }
        const all = (selector, root = document) => [...root.querySelectorAll(selector)].filter(visible)
        const activeDialog = () => {
          const dialogs = all('.el-dialog, .el-drawer, .el-message-box')
          return dialogs[dialogs.length - 1] || document
        }
        const formItem = (label, root = activeDialog()) => {
          const items = all('.el-form-item', root)
          const found = items.find((item) => text(item.querySelector('.el-form-item__label')) === label)
            || items.find((item) => text(item.querySelector('.el-form-item__label')).includes(label))
          if (!found) throw new Error('form item not found: ' + label)
          return found
        }
        const setNativeValue = (el, value) => {
          el.scrollIntoView({ block: 'center', inline: 'center' })
          el.focus()
          const proto = el instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype
          Object.getOwnPropertyDescriptor(proto, 'value').set.call(el, String(value))
          el.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText', data: String(value) }))
          el.dispatchEvent(new Event('change', { bubbles: true }))
          el.blur()
        }
        const clickEl = (el) => {
          if (!el) throw new Error('click target missing')
          el.scrollIntoView({ block: 'center', inline: 'center' })
          el.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }))
          el.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
          el.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }))
          el.click()
        }
        const findText = (label, selector = 'button,.el-button,a,.el-dropdown-menu__item,.el-tag,[role="button"]', root = document) => {
          const candidates = all(selector, root)
          return candidates.find((el) => text(el) === label) || candidates.find((el) => text(el).includes(label))
        }
        const waitUntil = async (fn, label, timeout = 15000) => {
          const started = Date.now()
          let last
          while (Date.now() - started < timeout) {
            try {
              last = fn()
              if (last) return last
            } catch (error) {
              last = error
            }
            await sleep(200)
          }
          throw new Error(label + (last instanceof Error ? ': ' + last.message : ''))
        }
        const waitIdle = () => waitUntil(() => !document.querySelector('.el-loading-mask:not([style*="display: none"])'), 'loading did not finish', 20000).catch(() => true)
        const button = (label, root = document) => {
          const target = findText(label, 'button,.el-button,a,[role="button"]', root)
          if (!target) throw new Error('button not found: ' + label)
          return target
        }
        const tableRow = (needle) => {
          const rows = all('.el-table__row')
          const found = rows.find((row) => text(row).includes(needle))
          if (!found) throw new Error('row not found: ' + needle)
          return found
        }
        window.__ui = {
          tag: ${JSON.stringify(TAG)},
          sleep,
          text: (selector) => text(document.querySelector(selector)),
          route: async (hash, marker) => {
            location.hash = hash
            await waitUntil(() => location.hash.includes(hash.replace(/^#/, '')), 'route did not change')
            await waitIdle()
            if (marker) await waitUntil(() => text(document.body).includes(marker), 'marker missing: ' + marker)
            await sleep(400)
            return { hash: location.hash, title: document.title }
          },
          clickText: async (label, rootSelector) => {
            const root = rootSelector ? document.querySelector(rootSelector) : document
            const target = await waitUntil(() => findText(label, 'button,.el-button,a,[role="button"],.el-tabs__item,.el-radio-button,.el-radio-button__inner', root || document), 'button not found: ' + label)
            clickEl(target)
            await sleep(300)
            return true
          },
          clickDialogText: async (label) => {
            const target = await waitUntil(() => findText(label, 'button,.el-button,a,[role="button"]', activeDialog()), 'dialog button not found: ' + label)
            clickEl(target)
            await sleep(300)
            return true
          },
          closeTopDialog: async () => {
            const dialogs = all('.el-dialog')
            const dialog = dialogs[dialogs.length - 1]
            if (!dialog) return true
            const close = dialog.querySelector('.el-dialog__headerbtn')
            if (!close) throw new Error('dialog close button missing')
            clickEl(close)
            await sleep(400)
            return true
          },
          closeTopDrawer: async () => {
            const drawers = all('.el-drawer')
            const drawer = drawers[drawers.length - 1]
            if (!drawer) return true
            const close = drawer.querySelector('.el-drawer__close-btn,.el-drawer__headerbtn,.el-drawer__close')
            if (!close) throw new Error('drawer close button missing')
            clickEl(close)
            await sleep(500)
            return true
          },
          input: async (label, value, scope = 'dialog') => {
            const root = scope === 'page' ? document : activeDialog()
            const item = formItem(label, root)
            const input = item.querySelector('textarea,input')
            if (!input) throw new Error('input missing for ' + label)
            setNativeValue(input, value)
            await sleep(80)
            return true
          },
          searchInput: async (label, value) => {
            const root = document.querySelector('.gva-search-box') || document
            const item = formItem(label, root)
            const input = item.querySelector('textarea,input')
            if (!input) throw new Error('search input missing for ' + label)
            setNativeValue(input, value)
            await sleep(80)
            return true
          },
          select: async (label, option, scope = 'dialog') => {
            const root = scope === 'page' ? document : activeDialog()
            const item = formItem(label, root)
            const select = item.querySelector('.el-select')
            if (!select) throw new Error('select missing for ' + label)
            clickEl(select.querySelector('.el-input__wrapper,input') || select)
            const optionEl = await waitUntil(() => {
              const options = all('.el-select-dropdown__item').filter((el) => !el.classList.contains('is-disabled'))
              return options.find((el) => text(el) === option) || options.find((el) => text(el).includes(option))
            }, 'option not found: ' + option)
            clickEl(optionEl)
            await sleep(250)
            return true
          },
          searchSelect: async (label, option) => window.__ui.select(label, option, 'page'),
          switchTo: async (label, desired, scope = 'dialog') => {
            const root = scope === 'page' ? document : activeDialog()
            const sw = formItem(label, root).querySelector('.el-switch')
            if (!sw) throw new Error('switch missing for ' + label)
            const checked = sw.classList.contains('is-checked') || sw.getAttribute('aria-checked') === 'true'
            if (checked !== desired) clickEl(sw)
            await sleep(100)
            return true
          },
          rowAction: async (needle, action) => {
            const row = tableRow(needle)
            clickEl(button(action, row))
            await sleep(400)
            return true
          },
          supportCaseCard: async (needle) => {
            const candidates = all('button')
            const found = candidates.find((el) => text(el).includes(needle))
            if (!found) throw new Error('case card not found: ' + needle)
            clickEl(found)
            await sleep(500)
            return true
          },
          dropdownRowAction: async (needle, menuText, itemText) => {
            const row = tableRow(needle)
            clickEl(button(menuText, row))
            const item = await waitUntil(() => findText(itemText, '.el-dropdown-menu__item'), 'dropdown item not found: ' + itemText)
            clickEl(item)
            await sleep(500)
            return true
          },
          confirm: async () => {
            const box = await waitUntil(() => activeDialog().classList?.contains('el-message-box') ? activeDialog() : document.querySelector('.el-message-box'), 'confirm box missing')
            clickEl(button('确定', box))
            await sleep(700)
            return true
          },
          waitText: async (needle, timeout = 15000) => {
            await waitUntil(() => text(document.body).includes(needle), 'text missing: ' + needle, timeout)
            return true
          },
          waitHash: async (needle, timeout = 15000) => {
            await waitUntil(() => location.hash.includes(needle), 'hash missing: ' + needle, timeout)
            return true
          },
          waitNoText: async (needle, timeout = 15000) => {
            const started = Date.now()
            while (Date.now() - started < timeout) {
              await sleep(250)
              if (!text(document.body).includes(needle)) return true
            }
            throw new Error('text still present: ' + needle)
          },
          rowExists: (needle) => !!all('.el-table__row').find((row) => text(row).includes(needle)),
          toastText: () => all('.el-message').map(text).join(' | '),
          activeText: () => text(activeDialog()),
          bodyIncludes: (needle) => text(document.body).includes(needle)
        }
        return true
      })()
    `, 'install helpers')

    const api = async (url, payload) => run(`
      fetch('/api${url}', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'x-token': localStorage.getItem('token') || ''
        },
        body: JSON.stringify(${JSON.stringify(payload || {})})
      }).then((res) => res.json())
    `, `api ${url}`)

    let storeId = 0
    let providerName = `${TAG}-provider`
    let warehouseName = `${TAG}-warehouse`
    let templateCode = `${TAG}-tpl`
    let supportTemplateCode = `${TAG}-reply`
    let caseSubject = `${TAG}-case`

    await step('店铺管理 CRUD 与搜索/重置', async () => {
      await call('route', '#/layout/amazon-store/storeManager', '店铺管理')
      await call('clickText', '下载授权文档')
      await call('clickText', '新增店铺')
      await call('input', '店铺名称', `${TAG}-store`)
      await call('select', '区域', 'EU')
      await call('input', 'Seller ID', `SELLER-${TAG}`)
      await call('input', 'SP 卖家ID', `SP-${TAG}`)
      await call('select', '启用站点', 'CA /')
      await call('input', 'Refresh Token', `refresh-token-${TAG}`)
      await call('switchTo', '启用', true)
      await call('clickDialogText', '保存')
      await call('searchInput', '关键词', `${TAG}-store`)
      await call('clickText', '查询')
      await call('waitText', `${TAG}-store`)
      await call('searchInput', '关键词', `${TAG}-store`)
      await call('clickText', '查询')
      await call('waitText', `${TAG}-store`)
      await call('searchSelect', '授权状态', '已授权')
      await call('clickText', '查询')
      await call('waitText', `${TAG}-store`)
      await call('clickText', '重置')
      await call('waitText', `${TAG}-store`)
      await call('rowAction', `${TAG}-store`, '编辑')
      await call('input', '店铺名称', `${TAG}-store-updated`)
      await call('select', '区域', 'NA')
      await call('switchTo', '启用', false)
      await call('clickDialogText', '保存')
      await call('waitText', `${TAG}-store-updated`)
      const list = await api('/amazonStore/list', { page: 1, pageSize: 10, keyword: `${TAG}-store-updated` })
      storeId = list?.data?.list?.[0]?.id || 0
      if (!storeId) throw new Error('saved store id missing')
    })

    await step('模板中心 CRUD 与下载/上传入口', async () => {
      await call('route', '#/layout/amazon-product/templateCenter', '模板中心')
      await call('clickText', '下载家居默认模板')
      await call('clickText', '新建模板')
      await call('clickDialogText', '下载家居默认模板')
      await call('input', '模板编码', templateCode)
      await call('input', '模板名称', `${TAG}-模板`)
      await call('input', '站点标识', 'ATVPDKIKX0DER')
      await call('select', '站点', 'US')
      await call('input', '产品类型', 'HOME')
      await call('input', '模板版本', 'v1')
      await call('input', '工作表名称', 'Template')
      await call('select', '状态', '草稿')
      await call('input', '列头行号', '1')
      await call('input', '数据起始行', '2')
      await call('select', '支持语言', '英语（美国）')
      await call('input', '备注', `note ${TAG}`)
      await call('clickDialogText', '保存')
      await call('searchInput', '关键词', templateCode)
      await call('clickText', '查询')
      await call('waitText', templateCode)
      await call('searchInput', '关键词', templateCode)
      await call('searchSelect', '站点', 'US')
      await call('searchSelect', '状态', '草稿')
      await call('clickText', '查询')
      await call('waitText', templateCode)
      await call('clickText', '重置')
      await call('waitText', templateCode)
      await call('rowAction', templateCode, '编辑')
      await call('input', '模板名称', `${TAG}-模板-更新`)
      await call('select', '状态', '启用')
      await call('clickDialogText', '保存')
      await call('waitText', `${TAG}-模板-更新`)
      await call('rowAction', templateCode, '下载模板')
      await call('rowAction', templateCode, '上传模板')
    })

    await step('退货服务商 CRUD、搜索、测试连接', async () => {
      await call('route', '#/layout/amazon-returns/returnProviders', '退货服务商管理')
      await call('clickText', '新增服务商')
      await call('input', '名称', providerName)
      await call('input', '编码', `provider_${TAG.replaceAll('-', '_')}`)
      await call('select', '报价模式', '手工')
      await call('input', '优先级', '8')
      await call('input', '基础URL', 'https://provider.example.test')
      await call('input', '报价路径', '/quote')
      await call('input', '创建路径', '/create')
      await call('input', '轨迹路径', '/tracking')
      await call('input', '鉴权头', 'Authorization')
      await call('input', '鉴权Token', `token-${TAG}`)
      await call('input', '处理费(CNY)', '3')
      await call('input', '基础费(CNY)', '11')
      await call('input', '每KG费(CNY)', '2')
      await call('select', '国家范围', 'US')
      await call('switchTo', '回仓支持', true)
      await call('switchTo', '转寄支持', true)
      await call('switchTo', '轨迹支持', true)
      await call('switchTo', '地址预填', true)
      await call('switchTo', '启用', true)
      await call('clickDialogText', '保存')
      await call('searchInput', '关键词', providerName)
      await call('clickText', '查询')
      await call('waitText', providerName)
      await call('searchInput', '关键词', providerName)
      await call('searchSelect', '报价模式', '手工')
      await call('clickText', '查询')
      await call('waitText', providerName)
      await call('rowAction', providerName, '测试连接')
      await sleep(600)
      await call('rowAction', providerName, '编辑')
      providerName = `${TAG}-provider-updated`
      await call('input', '名称', providerName)
      await call('input', '处理费(CNY)', '5')
      await call('switchTo', '地址预填', false)
      await call('clickDialogText', '保存')
      await call('waitText', providerName)
      await call('clickText', '重置')
      await call('waitText', providerName)
    })

    await step('退货仓库 CRUD 与搜索/开关', async () => {
      await call('route', '#/layout/amazon-returns/returnWarehouses', '退货仓库管理')
      await call('clickText', '新增仓库')
      await call('input', '仓库名称', warehouseName)
      await call('select', '国家', 'US')
      await call('select', '站点范围', 'US')
      await call('input', '联系人', `Contact ${TAG}`)
      await call('input', '电话', '15500001111')
      await call('input', '地址1', '100 Test Street')
      await call('input', '地址2', 'Suite 2')
      await call('input', '地址3', 'Dock 3')
      await call('input', '城市', 'Los Angeles')
      await call('input', '州/省', 'CA')
      await call('input', '邮编', '90001')
      await call('input', '优先级', '9')
      await call('switchTo', '默认仓', true)
      await call('switchTo', '启用', true)
      await call('clickDialogText', '保存')
      await call('searchInput', '关键词', warehouseName)
      await call('clickText', '查询')
      await call('waitText', warehouseName)
      await call('searchInput', '关键词', warehouseName)
      await call('searchSelect', '国家', 'US')
      await call('clickText', '查询')
      await call('waitText', warehouseName)
      await call('rowAction', warehouseName, '编辑')
      warehouseName = `${TAG}-warehouse-updated`
      await call('input', '仓库名称', warehouseName)
      await call('switchTo', '默认仓', false)
      await call('switchTo', '启用', false)
      await call('clickDialogText', '保存')
      await call('waitText', warehouseName)
      await call('clickText', '重置')
      await call('waitText', warehouseName)
    })

    await step('物流报价与报价库按钮/输入覆盖', async () => {
      await call('route', '#/layout/amazon-logistics/logisticsQuote', '美国直发物流比价')
      await call('input', '重量', '2.5', 'page')
      await call('input', '长', '12', 'page')
      await call('input', '宽', '10', 'page')
      await call('input', '高', '8', 'page')
      await call('switchTo', '是否带电', true, 'page')
      await call('clickText', '清空尺寸')
      await call('input', '长', '12', 'page')
      await call('input', '宽', '10', 'page')
      await call('input', '高', '8', 'page')
      await call('clickText', '查询最低报价')
      await call('waitText', '最低报价', 20000)
      await call('clickText', '重置条件')
      await call('clickText', '去报价库管理数据')
      await call('waitHash', 'logisticsLibrary')
      await call('waitText', '物流报价库', 15000)
      await call('clickText', '上传报价 Excel')
      await call('select', '服务商类型', '云途')
      await call('clickDialogText', '取消')
      await call('clickText', '刷新列表')
      await call('searchSelect', '服务商', '云途')
      await call('clickText', '查询')
      await call('clickText', '重置')
      await call('rowAction', '云途', '查看详情')
      await call('waitText', '费率明细', 15000)
      await call('clickText', '版本历史')
      await call('waitText', '版本历史', 15000)
      await call('clickText', '费率明细')
    })

    await step('客服工单与回复模板 CRUD', async () => {
      await call('route', '#/layout/amazon-support/supportInbox', 'Amazon 客服消息')
      await call('clickText', '模板管理')
      await call('clickText', '新建模板')
      await call('input', '编码', supportTemplateCode)
      await call('input', '名称', `${TAG}-回复模板`)
      await call('select', '案例类型', 'Buyer Message')
      await call('select', '投递方式', '复制草稿')
      await call('input', '排序', '6')
      await call('input', '主题模板', `Re: {{subject}} ${TAG}`)
      await call('input', '正文模板', `Hello {{buyer_name}}, ${TAG}`)
      await call('clickText', '新增变量')
      await call('switchTo', '启用', true)
      await call('clickDialogText', '保存')
      await call('waitText', supportTemplateCode)
      await call('rowAction', supportTemplateCode, '编辑')
      await call('input', '名称', `${TAG}-回复模板-更新`)
      await call('clickDialogText', '保存')
      await call('waitText', `${TAG}-回复模板-更新`)
      await call('closeTopDialog')
      await call('closeTopDialog')
      await call('clickText', '新建工单')
      await call('select', '店铺', `${TAG}-store-updated`)
      await call('select', '站点', 'US')
      await call('select', '案例类型', 'Buyer Message')
      await call('input', '买家姓名', `Buyer ${TAG}`)
      await call('input', '买家邮箱', `${TAG}@example.com`)
      await call('input', '外部案例ID', `CASE-${TAG}`)
      await call('input', '主题', caseSubject)
      await call('input', '客户消息', `Need help ${TAG}`)
      await call('input', '备注', `Note ${TAG}`)
      await call('clickDialogText', '保存')
      await call('searchInput', '关键词', caseSubject)
      await call('clickText', '查询')
      await call('waitText', caseSubject)
      await call('searchInput', '关键词', caseSubject)
      await call('searchSelect', '站点', 'US')
      await call('searchSelect', '案例类型', 'Buyer Message')
      await call('searchSelect', '已读', '未读')
      await call('searchSelect', '处理状态', '待处理')
      await call('clickText', '查询')
      await call('waitText', caseSubject)
      await call('supportCaseCard', caseSubject)
      await call('clickText', '刷新动作')
      await call('clickText', '标记已读')
      await sleep(400)
      await call('clickText', '标记待处理')
      await sleep(400)
      await call('select', '回复模板', `${TAG}-回复模板-更新`, 'page')
      await call('clickText', '复制草稿')
      await sleep(400)
      await call('clickText', '关闭工单')
      await call('confirm')
      await call('waitText', '已关闭')
      await call('clickText', '模板管理')
      await call('rowAction', supportTemplateCode, '删除')
      await call('confirm')
      await sleep(700)
    })

    await step('采集池插件来源数据的页面查询/详情/风险/删除', async () => {
      const asin = `B${Date.now().toString().slice(-9)}`
      const upsert = await api('/amazonCollector/extension/upsertDetail', {
        asin,
        siteCode: 'US',
        marketplaceId: 'ATVPDKIKX0DER',
        title: `${TAG}-采集商品`,
        brand: `${TAG}-brand`,
        productUrl: `https://www.amazon.com/dp/${asin}`,
        mainImageUrl: '',
        priceAmount: 12.34,
        currencyCode: 'USD',
        collectStatus: 'success',
        categoryPath: ['Home', 'Kitchen', TAG],
        categoryLeaf: TAG,
        sellerName: 'UI Seller',
        rawJson: { tag: TAG }
      })
      if (upsert.code !== 0) throw new Error(upsert.msg || 'collector upsert failed')
      await call('route', '#/layout/amazon-collection/collectorList', '采集商品列表')
      await call('searchInput', '关键词', TAG)
      await call('searchSelect', '站点', 'US')
      await call('searchSelect', '采集状态', '告警')
      await call('searchInput', '品牌', `${TAG}-brand`)
      await call('searchInput', '分类关键词', TAG)
      await call('clickText', '查询')
      await call('waitText', `${TAG}-采集商品`)
      await call('rowAction', `${TAG}-采集商品`, '查看详情')
      await call('waitText', '采集商品详情')
      await call('clickText', '重试图片入库')
      await sleep(400)
      await call('closeTopDrawer')
      await call('dropdownRowAction', `${TAG}-采集商品`, '更多', '侵权校验')
      await call('waitText', '编辑采集商品', 15000)
      await call('clickText', '已侵权')
      await call('clickText', '保存')
      await call('waitText', '已侵权', 15000)
      await call('dropdownRowAction', `${TAG}-采集商品`, '更多', '删除')
      await call('confirm')
      await sleep(700)
    })

    await step('清理本轮 UI 创建的数据', async () => {
      await call('route', '#/layout/amazon-returns/returnWarehouses', '退货仓库管理')
      await call('searchInput', '关键词', warehouseName)
      await call('clickText', '查询')
      await call('waitText', warehouseName)
      await call('rowAction', warehouseName, '删除')
      await call('confirm')

      await call('route', '#/layout/amazon-returns/returnProviders', '退货服务商管理')
      await call('searchInput', '关键词', providerName)
      await call('clickText', '查询')
      await call('waitText', providerName)
      await call('rowAction', providerName, '删除')
      await call('confirm')

      await call('route', '#/layout/amazon-product/templateCenter', '模板中心')
      await call('searchInput', '关键词', templateCode)
      await call('clickText', '查询')
      await call('waitText', templateCode)
      await call('rowAction', templateCode, '删除')
      await call('confirm')

      await call('route', '#/layout/amazon-store/storeManager', '店铺管理')
      await call('searchInput', '关键词', `${TAG}-store-updated`)
      await call('clickText', '查询')
      await call('waitText', `${TAG}-store-updated`)
      await call('rowAction', `${TAG}-store-updated`, '删除')
      await call('confirm')

      if (storeId) {
        const caseList = await api('/amazonSupportInbox/list', { page: 1, pageSize: 20, keyword: caseSubject })
        const caseId = caseList?.data?.list?.[0]?.id
        if (caseId) {
          await api('/amazonSupportInbox/close', { id: caseId })
        }
      }
    })

    await sleep(1000)
    const unexpectedAppErrors = events.appErrors.filter((item) => {
      if (item.url.includes('/amazonStore/testConnection')) return false
      if (item.url.includes('/amazonStore/syncOrdersNow')) return false
      if (item.url.includes('/amazonTemplate/downloadWorkbook') && String(item.msg || '').includes('字段规则')) return false
      return !String(item.msg || '').includes('cancel')
    })
    const summary = {
      tag: TAG,
      steps: stepResults,
      browserEvents: {
        consoleErrors: events.consoleErrors,
        exceptions: events.exceptions,
        httpErrors: events.httpErrors,
        appErrors: unexpectedAppErrors,
        loadingFailures: events.loadingFailures
      }
    }
    console.log(`UI_CRUD_PROBE_RESULT=${JSON.stringify(summary, null, 2)}`)
    if (stepResults.some((item) => !item.ok) || events.consoleErrors.length || events.exceptions.length || events.httpErrors.length || unexpectedAppErrors.length) {
      process.exitCode = 1
    }
  } finally {
    cdp.close()
    chrome.kill('SIGTERM')
    await sleep(500)
    await rm(profileDir, { recursive: true, force: true })
  }
}

main().catch((error) => {
  console.error(error.stack || error.message)
  process.exit(1)
})
