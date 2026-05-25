<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">RETURN PROVIDERS</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">退货服务商管理</h1>
          <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
            维护退货回仓与转寄服务商配置、人工报价参数和 API 连接状态。
          </p>
        </div>
        <div class="gva-btn-list !mb-0">
          <el-button type="primary" @click="openDialog()">新增服务商</el-button>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchTable">
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="名称 / 编码" />
        </el-form-item>
        <el-form-item label="报价模式">
          <el-select v-model="searchInfo.quoteMode" clearable class="!w-36">
            <el-option label="手工" value="manual" />
            <el-option label="API" value="api" />
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
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="code" label="编码" width="140" />
        <el-table-column label="报价模式" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="row.quoteMode === 'api' ? 'primary' : 'info'">
              {{ row.quoteMode === 'api' ? 'API' : '手工' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="适用国家" min-width="180">
          <template #default="{ row }">
            <div class="flex flex-wrap gap-2">
              <el-tag v-for="country in row.countryScopes || []" :key="country" size="small">{{ country }}</el-tag>
              <span v-if="!(row.countryScopes || []).length">全部</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="费用" min-width="180">
          <template #default="{ row }">
            <div class="text-sm">
              <div>处理费 {{ formatMoney(row.handlingFeeCny) }}</div>
              <div class="text-xs text-slate-500 dark:text-slate-400">
                基础费 {{ formatMoney(row.baseFeeCny) }} / 每KG {{ formatMoney(row.perKgFeeCny) }}
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="能力" min-width="180">
          <template #default="{ row }">
            <div class="flex flex-wrap gap-2">
              <el-tag size="small" type="success" v-if="row.supportsWarehouseReturn">回仓</el-tag>
              <el-tag size="small" type="warning" v-if="row.supportsBuyerRedirect">转寄</el-tag>
              <el-tag size="small" type="info" v-if="row.supportsTracking">轨迹</el-tag>
              <el-tag size="small" type="primary" v-if="row.supportsAddressPrefill">预填</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="row.isEnabled ? 'success' : 'info'">
              {{ row.isEnabled ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="240" fixed="right">
          <template #default="{ row }">
            <div class="flex gap-2">
              <el-button type="primary" link @click="openDialog(row)">编辑</el-button>
              <el-button type="success" link @click="testConnection(row)">测试连接</el-button>
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑服务商' : '新增服务商'" width="760px" destroy-on-close>
      <el-form label-width="120px">
        <div class="grid gap-4 md:grid-cols-2">
          <el-form-item label="名称">
            <el-input v-model="form.name" placeholder="请输入名称" />
          </el-form-item>
          <el-form-item label="编码">
            <el-input v-model="form.code" placeholder="如 yuntu_return" />
          </el-form-item>
          <el-form-item label="报价模式">
            <el-select v-model="form.quoteMode" class="w-full">
              <el-option label="手工" value="manual" />
              <el-option label="API" value="api" />
            </el-select>
          </el-form-item>
          <el-form-item label="优先级">
            <el-input-number v-model="form.priority" class="w-full" :min="1" :step="1" controls-position="right" />
          </el-form-item>
          <el-form-item label="基础URL" class="md:col-span-2">
            <el-input v-model="form.baseUrl" placeholder="https://provider.example.com" />
          </el-form-item>
          <el-form-item label="报价路径">
            <el-input v-model="form.quotePath" placeholder="/quote" />
          </el-form-item>
          <el-form-item label="创建路径">
            <el-input v-model="form.createPath" placeholder="/create" />
          </el-form-item>
          <el-form-item label="轨迹路径">
            <el-input v-model="form.trackingPath" placeholder="/tracking" />
          </el-form-item>
          <el-form-item label="鉴权头">
            <el-input v-model="form.authHeader" placeholder="Authorization" />
          </el-form-item>
          <el-form-item label="鉴权Token" class="md:col-span-2">
            <el-input v-model="form.authToken" type="textarea" :rows="2" placeholder="留空则沿用已保存 token" />
          </el-form-item>
          <el-form-item label="处理费(CNY)">
            <el-input-number v-model="form.handlingFeeCny" class="w-full" :min="0" :step="1" controls-position="right" />
          </el-form-item>
          <el-form-item label="基础费(CNY)">
            <el-input-number v-model="form.baseFeeCny" class="w-full" :min="0" :step="1" controls-position="right" />
          </el-form-item>
          <el-form-item label="每KG费(CNY)">
            <el-input-number v-model="form.perKgFeeCny" class="w-full" :min="0" :step="1" controls-position="right" />
          </el-form-item>
          <el-form-item label="国家范围" class="md:col-span-2">
            <el-select v-model="form.countryScopes" multiple filterable allow-create default-first-option class="w-full" placeholder="留空表示全部">
              <el-option label="US" value="US" />
              <el-option label="CA" value="CA" />
              <el-option label="MX" value="MX" />
            </el-select>
          </el-form-item>
          <el-form-item label="回仓支持">
            <el-switch v-model="form.supportsWarehouseReturn" />
          </el-form-item>
          <el-form-item label="转寄支持">
            <el-switch v-model="form.supportsBuyerRedirect" />
          </el-form-item>
          <el-form-item label="轨迹支持">
            <el-switch v-model="form.supportsTracking" />
          </el-form-item>
          <el-form-item label="地址预填">
            <el-switch v-model="form.supportsAddressPrefill" />
          </el-form-item>
          <el-form-item label="启用">
            <el-switch v-model="form.isEnabled" />
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveProvider">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  deleteAmazonReturnProvider,
  getAmazonReturnProviderList,
  saveAmazonReturnProvider,
  testAmazonReturnProviderConnection
} from '@/api/amazonReturnProvider'

defineOptions({
  name: 'AmazonReturnProviderManager'
})

const tableData = ref([])
const total = ref(0)
const dialogVisible = ref(false)
const saving = ref(false)

const searchInfo = ref({
  page: 1,
  pageSize: 10,
  keyword: '',
  quoteMode: ''
})

const createForm = () => ({
  id: 0,
  name: '',
  code: '',
  quoteMode: 'manual',
  baseUrl: '',
  quotePath: '',
  createPath: '',
  trackingPath: '',
  authHeader: '',
  authToken: '',
  handlingFeeCny: 0,
  baseFeeCny: 0,
  perKgFeeCny: 0,
  supportsBuyerRedirect: false,
  supportsWarehouseReturn: true,
  supportsTracking: true,
  supportsAddressPrefill: false,
  countryScopes: [],
  priority: 100,
  isEnabled: true
})

const form = ref(createForm())

const fetchTable = async () => {
  const res = await getAmazonReturnProviderList(searchInfo.value)
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
    quoteMode: ''
  }
  fetchTable()
}

const handleCurrentChange = (page) => {
  searchInfo.value.page = page
  fetchTable()
}

const handleSizeChange = (pageSize) => {
  searchInfo.value.pageSize = pageSize
  searchInfo.value.page = 1
  fetchTable()
}

const openDialog = (row) => {
  form.value = row
    ? {
      ...createForm(),
      ...row,
      countryScopes: row.countryScopes || []
    }
    : createForm()
  dialogVisible.value = true
}

const saveProvider = async () => {
  saving.value = true
  try {
    const res = await saveAmazonReturnProvider(form.value)
    if (res.code === 0) {
      ElMessage.success('保存成功')
      dialogVisible.value = false
      fetchTable()
    }
  } finally {
    saving.value = false
  }
}

const testConnection = async (row) => {
  const res = await testAmazonReturnProviderConnection({ id: row.id })
  if (res.code === 0) {
    ElMessage.success(res.data.message || '测试成功')
    fetchTable()
  }
}

const removeRow = async (row) => {
  await ElMessageBox.confirm(`确认删除退货服务商 ${row.name} 吗？`, '删除确认', {
    type: 'warning'
  })
  const res = await deleteAmazonReturnProvider({ id: row.id })
  if (res.code === 0) {
    ElMessage.success('删除成功')
    fetchTable()
  }
}

const formatMoney = (value) => {
  if (value === null || typeof value === 'undefined') {
    return '--'
  }
  return Number(value).toFixed(2)
}

onMounted(() => {
  fetchTable()
})
</script>
