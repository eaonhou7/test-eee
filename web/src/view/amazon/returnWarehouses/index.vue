<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">RETURN WAREHOUSES</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">退货仓库管理</h1>
          <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
            维护回仓地址簿，按国家、站点和默认仓优先级为退货处置自动选仓。
          </p>
        </div>
        <div class="gva-btn-list !mb-0">
          <el-button type="primary" @click="openDialog()">新增仓库</el-button>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchTable">
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="仓库名 / 联系人 / 电话" />
        </el-form-item>
        <el-form-item label="国家">
          <el-select v-model="searchInfo.countryCode" clearable class="!w-36">
            <el-option label="US" value="US" />
            <el-option label="CA" value="CA" />
            <el-option label="MX" value="MX" />
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
        <el-table-column prop="name" label="仓库名称" min-width="160" />
        <el-table-column prop="countryCode" label="国家" width="100" />
        <el-table-column label="站点范围" min-width="160">
          <template #default="{ row }">
            <div class="flex flex-wrap gap-2">
              <el-tag v-for="site in row.siteScopes || []" :key="site" size="small">{{ site }}</el-tag>
              <span v-if="!(row.siteScopes || []).length">全部站点</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="联系人" min-width="200">
          <template #default="{ row }">
            <div class="text-sm">
              <div>{{ row.contactName || '--' }}</div>
              <div class="text-xs text-slate-500 dark:text-slate-400">{{ row.phone || '--' }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="地址" min-width="260">
          <template #default="{ row }">
            <div class="text-sm text-slate-700 dark:text-slate-300">
              {{ formatAddress(row) }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="优先级" width="90" prop="priority" />
        <el-table-column label="状态" width="140">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <el-tag :type="row.isEnabled ? 'success' : 'info'">{{ row.isEnabled ? '启用' : '停用' }}</el-tag>
              <el-tag v-if="row.isDefault" type="warning" size="small">默认仓</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="180" fixed="right">
          <template #default="{ row }">
            <div class="flex gap-2">
              <el-button type="primary" link @click="openDialog(row)">编辑</el-button>
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑仓库' : '新增仓库'" width="760px" destroy-on-close>
      <el-form label-width="120px">
        <div class="grid gap-4 md:grid-cols-2">
          <el-form-item label="仓库名称">
            <el-input v-model="form.name" placeholder="请输入仓库名称" />
          </el-form-item>
          <el-form-item label="国家">
            <el-select v-model="form.countryCode" class="w-full">
              <el-option label="US" value="US" />
              <el-option label="CA" value="CA" />
              <el-option label="MX" value="MX" />
            </el-select>
          </el-form-item>
          <el-form-item label="站点范围" class="md:col-span-2">
            <el-select v-model="form.siteScopes" multiple filterable class="w-full" placeholder="留空表示全部">
              <el-option label="US" value="US" />
              <el-option label="CA" value="CA" />
              <el-option label="MX" value="MX" />
            </el-select>
          </el-form-item>
          <el-form-item label="联系人">
            <el-input v-model="form.contactName" />
          </el-form-item>
          <el-form-item label="电话">
            <el-input v-model="form.phone" />
          </el-form-item>
          <el-form-item label="地址1" class="md:col-span-2">
            <el-input v-model="form.addressLine1" />
          </el-form-item>
          <el-form-item label="地址2" class="md:col-span-2">
            <el-input v-model="form.addressLine2" />
          </el-form-item>
          <el-form-item label="地址3" class="md:col-span-2">
            <el-input v-model="form.addressLine3" />
          </el-form-item>
          <el-form-item label="城市">
            <el-input v-model="form.city" />
          </el-form-item>
          <el-form-item label="州/省">
            <el-input v-model="form.stateOrRegion" />
          </el-form-item>
          <el-form-item label="邮编">
            <el-input v-model="form.postalCode" />
          </el-form-item>
          <el-form-item label="优先级">
            <el-input-number v-model="form.priority" class="w-full" :min="1" :step="1" controls-position="right" />
          </el-form-item>
          <el-form-item label="默认仓">
            <el-switch v-model="form.isDefault" />
          </el-form-item>
          <el-form-item label="启用">
            <el-switch v-model="form.isEnabled" />
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveWarehouse">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  deleteAmazonReturnWarehouse,
  getAmazonReturnWarehouseList,
  saveAmazonReturnWarehouse
} from '@/api/amazonReturnWarehouse'

defineOptions({
  name: 'AmazonReturnWarehouseManager'
})

const tableData = ref([])
const total = ref(0)
const dialogVisible = ref(false)
const saving = ref(false)

const searchInfo = ref({
  page: 1,
  pageSize: 10,
  keyword: '',
  countryCode: ''
})

const createForm = () => ({
  id: 0,
  name: '',
  countryCode: 'US',
  siteScopes: [],
  contactName: '',
  phone: '',
  addressLine1: '',
  addressLine2: '',
  addressLine3: '',
  city: '',
  stateOrRegion: '',
  postalCode: '',
  priority: 100,
  isDefault: false,
  isEnabled: true
})

const form = ref(createForm())

const fetchTable = async () => {
  const res = await getAmazonReturnWarehouseList(searchInfo.value)
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
    countryCode: ''
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
      siteScopes: row.siteScopes || []
    }
    : createForm()
  dialogVisible.value = true
}

const saveWarehouse = async () => {
  saving.value = true
  try {
    const res = await saveAmazonReturnWarehouse(form.value)
    if (res.code === 0) {
      ElMessage.success('保存成功')
      dialogVisible.value = false
      fetchTable()
    }
  } finally {
    saving.value = false
  }
}

const removeRow = async (row) => {
  await ElMessageBox.confirm(`确认删除退货仓 ${row.name} 吗？`, '删除确认', {
    type: 'warning'
  })
  const res = await deleteAmazonReturnWarehouse({ id: row.id })
  if (res.code === 0) {
    ElMessage.success('删除成功')
    fetchTable()
  }
}

const formatAddress = (row) => {
  return [row.addressLine1, row.addressLine2, row.addressLine3, row.city, row.stateOrRegion, row.postalCode].filter(Boolean).join(', ') || '--'
}

onMounted(() => {
  fetchTable()
})
</script>
