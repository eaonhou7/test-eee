<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON RETURNS WORKBENCH</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">Amazon 退货工作台</h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            定时同步 Amazon 退货报表，自动关联原订单、计算赠送阈值、识别可转寄订单并生成回仓处置。
          </p>
        </div>
        <div class="gva-btn-list !mb-0">
          <el-button :loading="resyncLoading" type="warning" @click="handleResync">立即同步退货</el-button>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchTable">
        <el-form-item label="店铺">
          <el-select v-model="searchInfo.storeId" clearable filterable class="!w-52">
            <el-option v-for="store in storeOptions" :key="store.id" :label="store.storeName" :value="store.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="站点">
          <el-select v-model="searchInfo.siteCode" clearable class="!w-32">
            <el-option label="US" value="US" />
            <el-option label="CA" value="CA" />
            <el-option label="MX" value="MX" />
          </el-select>
        </el-form-item>
        <el-form-item label="关联状态">
          <el-select v-model="searchInfo.linkStatus" clearable class="!w-40">
            <el-option label="已关联" value="linked" />
            <el-option label="缺原单" value="missing_order" />
            <el-option label="缺订单项" value="missing_item" />
            <el-option label="歧义" value="ambiguous" />
            <el-option label="人工复核" value="manual_review" />
          </el-select>
        </el-form-item>
        <el-form-item label="建议动作">
          <el-select v-model="searchInfo.recommendedDecision" clearable class="!w-40">
            <el-option label="赠送" value="gift" />
            <el-option label="转寄新买家" value="new_buyer" />
            <el-option label="回仓" value="warehouse" />
            <el-option label="人工复核" value="manual_review" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="订单号 / RMA / 运单号" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchTable">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table :data="tableData" row-key="id" stripe>
        <el-table-column label="退货单" min-width="220">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <span class="font-medium text-slate-900 dark:text-slate-100">{{ row.amazonRmaId || '--' }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400">
                {{ row.amazonOrderId || '--' }} / {{ row.siteCode || '--' }}
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="申请状态" width="140">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.returnRequestStatus || '--' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="关联状态" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="linkTagType(row.linkStatus)">{{ linkLabel(row.linkStatus) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="建议汇总" min-width="220">
          <template #default="{ row }">
            <div class="flex flex-wrap gap-2">
              <el-tag v-if="row.decisionSummary?.gift" size="small" type="info">赠送 {{ row.decisionSummary.gift }}</el-tag>
              <el-tag v-if="row.decisionSummary?.new_buyer" size="small" type="warning">转寄 {{ row.decisionSummary.new_buyer }}</el-tag>
              <el-tag v-if="row.decisionSummary?.warehouse" size="small" type="success">回仓 {{ row.decisionSummary.warehouse }}</el-tag>
              <el-tag v-if="row.decisionSummary?.manual_review" size="small" type="danger">复核 {{ row.decisionSummary.manual_review }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="退款 / 面单费" min-width="180">
          <template #default="{ row }">
            <div class="text-sm">
              <div>{{ formatPrice(row.refundAmount, row.refundCurrency) }}</div>
              <div class="text-xs text-slate-500 dark:text-slate-400">
                面单费 {{ formatPrice(row.labelCost, row.labelCurrency) }}
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="returnRequestDate" label="申请时间" min-width="160" />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDetail(row)">详情</el-button>
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
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { getAmazonReturnList, resyncAmazonReturns } from '@/api/amazonReturn'
import { getAmazonStoreList } from '@/api/amazonStore'

defineOptions({
  name: 'AmazonReturnWorkbench'
})

const router = useRouter()
const tableData = ref([])
const total = ref(0)
const resyncLoading = ref(false)
const storeOptions = ref([])

const searchInfo = ref({
  page: 1,
  pageSize: 10,
  storeId: undefined,
  siteCode: '',
  linkStatus: '',
  recommendedDecision: '',
  keyword: ''
})

const fetchStoreOptions = async () => {
  const res = await getAmazonStoreList({ page: 1, pageSize: 200 })
  if (res.code === 0) {
    storeOptions.value = res.data.list || []
  }
}

const fetchTable = async () => {
  const res = await getAmazonReturnList(searchInfo.value)
  if (res.code === 0) {
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  }
}

const resetSearch = () => {
  searchInfo.value = {
    page: 1,
    pageSize: 10,
    storeId: undefined,
    siteCode: '',
    linkStatus: '',
    recommendedDecision: '',
    keyword: ''
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

const handleResync = async () => {
  resyncLoading.value = true
  try {
    const res = await resyncAmazonReturns({})
    if (res.code === 0) {
      ElMessage.success(`同步完成，新增/更新 ${res.data.recordsSynced || 0} 条退货记录`)
      fetchTable()
    }
  } finally {
    resyncLoading.value = false
  }
}

const openDetail = (row) => {
  router.push({
    name: 'amazonReturnDetail',
    params: { id: row.id }
  })
}

const linkLabel = (value) => {
  switch (value) {
    case 'linked':
      return '已关联'
    case 'missing_order':
      return '缺原单'
    case 'missing_item':
      return '缺订单项'
    case 'ambiguous':
      return '歧义'
    case 'manual_review':
      return '人工复核'
    default:
      return value || '--'
  }
}

const linkTagType = (value) => {
  switch (value) {
    case 'linked':
      return 'success'
    case 'missing_order':
    case 'missing_item':
    case 'ambiguous':
    case 'manual_review':
      return 'danger'
    default:
      return 'info'
  }
}

const formatPrice = (value, currencyCode) => {
  if (value === null || typeof value === 'undefined') {
    return '--'
  }
  return `${currencyCode || ''} ${Number(value).toFixed(2)}`.trim()
}

onMounted(async () => {
  await fetchStoreOptions()
  await fetchTable()
})
</script>
