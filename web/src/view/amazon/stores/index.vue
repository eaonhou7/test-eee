<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON 店铺中心</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">店铺管理</h1>
          <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
            维护多店铺 Amazon SP-API 授权、启用站点、手动 refresh token 与订单同步入口。
          </p>
        </div>
        <div class="gva-btn-list !mb-0">
          <el-button @click="downloadAuthorizationDoc">下载授权文档</el-button>
          <el-button type="primary" @click="openDialog()">新增店铺</el-button>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchTable">
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="店铺名称 / Seller ID / SP ID" />
        </el-form-item>
        <el-form-item label="授权状态">
          <el-select v-model="searchInfo.authStatus" clearable class="!w-36">
            <el-option label="已授权" value="authorized" />
            <el-option label="未授权" value="unauthorized" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchTable">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table :data="tableData" row-key="id" stripe>
        <el-table-column prop="storeName" label="店铺名称" min-width="180" />
        <el-table-column prop="region" label="区域" width="90" />
        <el-table-column label="Seller / SP ID" min-width="200">
          <template #default="{ row }">
            <div class="flex flex-col gap-1 text-sm">
              <span>{{ row.sellerId || '--' }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400">{{ row.sellingPartnerId || '--' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="启用站点" min-width="180">
          <template #default="{ row }">
            <div class="flex flex-wrap gap-2">
              <el-tag v-for="site in row.enabledMarketplaces || []" :key="site" size="small">{{ site }}</el-tag>
              <span v-if="!(row.enabledMarketplaces || []).length">--</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="授权状态" width="120">
          <template #default="{ row }">
            <el-tag :type="row.authStatus === 'authorized' ? 'success' : 'info'">
              {{ row.authStatus === 'authorized' ? '已授权' : '未授权' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最后同步" min-width="160">
          <template #default="{ row }">{{ row.lastOrderSyncAt || '--' }}</template>
        </el-table-column>
        <el-table-column label="操作" min-width="360" fixed="right">
          <template #default="{ row }">
            <div class="flex gap-2">
              <el-button type="primary" link @click="openDialog(row)">编辑</el-button>
              <el-button type="primary" link @click="startAuth(row)">去授权</el-button>
              <el-button type="success" link @click="testConnection(row)">测试连接</el-button>
              <el-button type="warning" link @click="syncOrders(row)">立即拉单</el-button>
              <el-button type="danger" link @click="removeRow(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div class="gva-pagination">
        <el-pagination
          layout="total, sizes, prev, pager, next, jumper"
          :current-page="searchInfo.page"
          :page-size="searchInfo.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑店铺' : '新增店铺'" width="620px" destroy-on-close>
      <el-form label-width="108px">
        <el-form-item label="店铺名称">
          <el-input v-model="form.storeName" placeholder="请输入店铺名称" />
        </el-form-item>
        <el-form-item label="区域">
          <el-select v-model="form.region" class="w-full">
            <el-option label="NA" value="NA" />
            <el-option label="EU" value="EU" />
            <el-option label="FE" value="FE" />
          </el-select>
        </el-form-item>
        <el-form-item label="Seller ID">
          <el-input v-model="form.sellerId" placeholder="可选，授权后也可回填" />
        </el-form-item>
        <el-form-item label="SP 卖家ID">
          <el-input v-model="form.sellingPartnerId" placeholder="可选，授权回调会回填" />
        </el-form-item>
        <el-form-item label="启用站点">
          <el-select v-model="form.enabledMarketplaces" multiple filterable class="w-full">
            <el-option label="US / ATVPDKIKX0DER" value="ATVPDKIKX0DER" />
            <el-option label="CA / A2EUQ1WTGCTBG2" value="A2EUQ1WTGCTBG2" />
            <el-option label="MX / A1AM78C64UM0Y8" value="A1AM78C64UM0Y8" />
          </el-select>
        </el-form-item>
        <el-form-item label="Refresh Token">
          <el-input v-model="form.refreshToken" type="textarea" :rows="4" placeholder="可选：直接粘贴 refresh token，或保存后点击“去授权”完成授权" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.isEnabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveStore">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { triggerBlobDownload } from '@/utils/blobDownload'
import {
  deleteAmazonStore,
  getAmazonStoreList,
  saveAmazonStore,
  startAmazonStoreAuth,
  syncAmazonStoreOrdersNow,
  testAmazonStoreConnection
} from '@/api/amazonStore'

const AUTHORIZATION_DOC_PATH = 'docs/amazon-store-authorization-guide.html'
const AUTHORIZATION_DOC_NAME = 'Amazon店铺授权详细文档.html'

const tableData = ref([])
const total = ref(0)
const dialogVisible = ref(false)
const saving = ref(false)

const searchInfo = ref({
  page: 1,
  pageSize: 10,
  keyword: '',
  authStatus: ''
})

const createForm = () => ({
  id: 0,
  storeName: '',
  region: 'NA',
  sellerId: '',
  sellingPartnerId: '',
  enabledMarketplaces: ['ATVPDKIKX0DER'],
  refreshToken: '',
  isEnabled: true
})

const form = ref(createForm())

const fetchTable = async () => {
  const res = await getAmazonStoreList(searchInfo.value)
  if (res.code === 0) {
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  }
}

const resetSearch = () => {
  searchInfo.value = {
    page: 1,
    pageSize: 10,
    keyword: '',
    authStatus: ''
  }
  fetchTable()
}

const handleCurrentChange = (page) => {
  searchInfo.value.page = page
  fetchTable()
}

const handleSizeChange = (size) => {
  searchInfo.value.pageSize = size
  searchInfo.value.page = 1
  fetchTable()
}

const downloadAuthorizationDoc = async () => {
  try {
    const url = new URL(`${import.meta.env.BASE_URL}${AUTHORIZATION_DOC_PATH}`, window.location.origin)
    const response = await fetch(url.toString())
    if (!response.ok) {
      throw new Error('授权文档不存在')
    }
    const blob = await response.blob()
    triggerBlobDownload(blob, AUTHORIZATION_DOC_NAME)
    ElMessage.success('授权文档下载成功')
  } catch (error) {
    ElMessage.error(error?.message || '授权文档下载失败')
  }
}

const openDialog = (row) => {
  form.value = row
    ? {
        id: row.id,
        storeName: row.storeName,
        region: row.region || 'NA',
        sellerId: row.sellerId || '',
        sellingPartnerId: row.sellingPartnerId || '',
        enabledMarketplaces: [...(row.enabledMarketplaces || [])],
        refreshToken: '',
        isEnabled: row.isEnabled !== false
      }
    : createForm()
  dialogVisible.value = true
}

const saveStore = async () => {
  saving.value = true
  try {
    const res = await saveAmazonStore(form.value)
    if (res.code === 0) {
      ElMessage.success('店铺保存成功')
      dialogVisible.value = false
      fetchTable()
    }
  } finally {
    saving.value = false
  }
}

const startAuth = async (row) => {
  const res = await startAmazonStoreAuth({ id: row.id })
  if (res.code === 0 && res.data.authorizeUrl) {
    window.open(res.data.authorizeUrl, '_blank', 'noopener,noreferrer')
    ElMessage.success('已打开 Amazon 授权页面')
  }
}

const testConnection = async (row) => {
  const res = await testAmazonStoreConnection({ id: row.id })
  if (res.code === 0) {
    ElMessage.success(`连接成功，可访问站点：${(res.data.marketplaceCodes || []).join(', ') || '已授权'}`)
    fetchTable()
  }
}

const syncOrders = async (row) => {
  const res = await syncAmazonStoreOrdersNow({ id: row.id })
  if (res.code === 0) {
    ElMessage.success(`同步完成，共处理 ${res.data.ordersSynced || 0} 笔订单`)
    fetchTable()
  }
}

const removeRow = async (row) => {
  await ElMessageBox.confirm(`确认删除店铺 ${row.storeName} 吗？`, '删除确认', { type: 'warning' })
  const res = await deleteAmazonStore({ id: row.id })
  if (res.code === 0) {
    ElMessage.success('删除成功')
    fetchTable()
  }
}

onMounted(() => {
  fetchTable()
})
</script>
