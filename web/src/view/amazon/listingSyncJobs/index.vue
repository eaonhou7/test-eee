<template>
  <div class="flex flex-col gap-6">
    <section class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON LISTING SYNC JOBS</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">价格库存回传任务</h1>
          <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
            查看 Amazon 价格、库存、备货天数和配送模板批量回传任务，并同步 FBA 实际库存。
          </p>
        </div>
        <div class="gva-btn-list !mb-0">
          <el-button @click="fetchList">刷新</el-button>
          <el-button type="primary" plain :disabled="!searchInfo.storeId" @click="handleResyncFbaInventory">同步 FBA 实际库存</el-button>
        </div>
      </div>
    </section>

    <section class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchList">
        <el-form-item label="店铺">
          <el-select v-model="searchInfo.storeId" clearable filterable class="!w-56">
            <el-option
              v-for="store in storeOptions"
              :key="store.id"
              :label="store.storeName || `店铺${store.id}`"
              :value="store.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="处理状态">
          <el-select v-model="searchInfo.processingStatus" clearable class="!w-40">
            <el-option label="处理中" value="processing" />
            <el-option label="已完成" value="completed" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-form-item>
        <el-form-item label="提交状态">
          <el-select v-model="searchInfo.submitStatus" clearable class="!w-40">
            <el-option label="提交中" value="submitting" />
            <el-option label="已提交" value="submitted" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="Feed ID / SKU / 站点" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchList">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </section>

    <section class="gva-table-box">
      <el-table :data="jobList" row-key="id" border>
        <el-table-column prop="id" label="任务ID" width="90" />
        <el-table-column label="店铺" min-width="160">
          <template #default="{ row }">
            {{ getStoreName(row.storeId) }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="180">
          <template #default="{ row }">
            <div class="flex flex-wrap gap-2">
              <el-tag size="small" :type="statusTagType(row.processingStatus)">
                {{ statusLabel(row.processingStatus) }}
              </el-tag>
              <el-tag size="small" :type="statusTagType(row.submitStatus)">
                {{ statusLabel(row.submitStatus) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="feedId" label="Feed ID" min-width="220" />
        <el-table-column label="字段范围" min-width="220">
          <template #default="{ row }">
            <div class="flex flex-wrap gap-2">
              <el-tag v-for="scope in row.fieldScopes || []" :key="scope" size="small" type="info">
                {{ fieldScopeLabel(scope) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="issueSummary" label="结果摘要" min-width="220" />
        <el-table-column prop="submittedAt" label="提交时间" min-width="180" />
        <el-table-column prop="finishedAt" label="完成时间" min-width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <div class="flex items-center gap-2">
              <el-button type="primary" link @click="openDetail(row)">详情</el-button>
              <el-button type="primary" link @click="refreshJob(row)">刷新状态</el-button>
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
    </section>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'

import {
  getAmazonListingSyncList,
  refreshAmazonListingSyncStatus,
  resyncAmazonFbaInventory
} from '@/api/amazonListingSync'
import { getAmazonStoreList } from '@/api/amazonStore'

defineOptions({
  name: 'AmazonListingSyncJobManager'
})

const router = useRouter()

const storeOptions = ref([])
const jobList = ref([])
const total = ref(0)

const searchInfo = reactive({
  page: 1,
  pageSize: 10,
  storeId: undefined,
  processingStatus: '',
  submitStatus: '',
  keyword: ''
})

const fetchStores = async () => {
  const res = await getAmazonStoreList({
    page: 1,
    pageSize: 200
  })
  storeOptions.value = res.data?.list || []
}

const fetchList = async () => {
  const res = await getAmazonListingSyncList(searchInfo)
  jobList.value = res.data?.list || []
  total.value = res.data?.total || 0
}

const resetSearch = () => {
  searchInfo.page = 1
  searchInfo.pageSize = 10
  searchInfo.storeId = undefined
  searchInfo.processingStatus = ''
  searchInfo.submitStatus = ''
  searchInfo.keyword = ''
  fetchList()
}

const handleCurrentChange = (page) => {
  searchInfo.page = page
  fetchList()
}

const handleSizeChange = (pageSize) => {
  searchInfo.page = 1
  searchInfo.pageSize = pageSize
  fetchList()
}

const getStoreName = (storeId) => {
  const store = storeOptions.value.find((item) => Number(item.id) === Number(storeId || 0))
  return store?.storeName || `店铺${storeId || '--'}`
}

const fieldScopeLabel = (scope) => {
  switch (scope) {
    case 'price':
      return '价格'
    case 'inventory':
      return '库存'
    case 'leadTimeToShip':
      return '备货天数'
    case 'merchantShippingGroup':
      return '配送模板'
    default:
      return scope || '--'
  }
}

const statusLabel = (status) => {
  switch (status) {
    case 'submitting':
      return '提交中'
    case 'submitted':
      return '已提交'
    case 'processing':
      return '处理中'
    case 'completed':
      return '已完成'
    case 'failed':
      return '失败'
    default:
      return status || '--'
  }
}

const statusTagType = (status) => {
  switch (status) {
    case 'submitted':
    case 'completed':
      return 'success'
    case 'submitting':
    case 'processing':
      return 'warning'
    case 'failed':
      return 'danger'
    default:
      return 'info'
  }
}

const openDetail = (row) => {
  router.push({
    name: 'amazonListingSyncJobDetail',
    params: { id: row.id }
  })
}

const refreshJob = async (row) => {
  const res = await refreshAmazonListingSyncStatus({ id: row.id })
  if (res.code === 0) {
    ElMessage.success('任务状态已刷新')
    await fetchList()
  }
}

const handleResyncFbaInventory = async () => {
  const res = await resyncAmazonFbaInventory({ storeId: searchInfo.storeId })
  if (res.code === 0) {
    ElMessage.success('FBA 实际库存同步已触发')
  }
}

onMounted(() => {
  fetchStores()
  fetchList()
})
</script>
