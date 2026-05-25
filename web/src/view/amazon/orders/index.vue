<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON FBM FULFILLMENT</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">Amazon 订单履约</h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            定时同步 Amazon 新订单，自动归档 FBA/FBM 履约类型、1688 绑定状态、采购组和物流进度。FBM 订单在详情页启动采购、打印和物流闭环。
          </p>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchTable">
        <el-form-item label="店铺">
          <el-select v-model="searchInfo.storeId" clearable filterable class="!w-52">
            <el-option
              v-for="store in storeOptions"
              :key="store.id"
              :label="store.storeName"
              :value="store.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="站点">
          <el-select v-model="searchInfo.siteCode" clearable class="!w-32">
            <el-option label="US" value="US" />
            <el-option label="CA" value="CA" />
            <el-option label="MX" value="MX" />
          </el-select>
        </el-form-item>
        <el-form-item label="Amazon状态">
          <el-input v-model="searchInfo.status" clearable placeholder="Unshipped / Shipped" />
        </el-form-item>
        <el-form-item label="履约类型">
          <el-select v-model="searchInfo.fulfillmentType" clearable class="!w-36">
            <el-option label="FBA" value="fba" />
            <el-option label="FBM" value="fbm" />
          </el-select>
        </el-form-item>
        <el-form-item label="工作流">
          <el-select v-model="searchInfo.workflowStatus" clearable class="!w-44">
            <el-option label="待履约" value="fbm_pending" />
            <el-option label="异常" value="fbm_exception" />
            <el-option label="待转寄" value="fbm_waiting_return_redirect" />
            <el-option label="转寄已发货" value="fbm_return_redirect_shipped" />
            <el-option label="执行中" value="fulfillment_running" />
            <el-option label="已完成" value="fulfillment_completed" />
            <el-option label="执行失败" value="fulfillment_failed" />
            <el-option label="FBA关闭" value="fba_closed" />
          </el-select>
        </el-form-item>
        <el-form-item label="退货摘要">
          <el-select v-model="searchInfo.returnSummaryStatus" clearable class="!w-40">
            <el-option label="无退货" value="none" />
            <el-option label="开放" value="open" />
            <el-option label="处理中" value="processing" />
            <el-option label="已关闭" value="closed" />
            <el-option label="异常" value="exception" />
          </el-select>
        </el-form-item>
        <el-form-item label="有转寄候选">
          <el-switch v-model="searchInfo.hasRedirectCandidate" />
        </el-form-item>
        <el-form-item label="仅异常">
          <el-switch v-model="searchInfo.exceptionOnly" />
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="订单号 / 买家 / 收货人" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchTable">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table :data="tableData" row-key="id" stripe>
        <el-table-column label="Amazon订单" min-width="220">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <span class="font-medium text-slate-900 dark:text-slate-100">{{ row.amazonOrderId || '--' }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400">
                {{ row.storeName || `Store #${row.storeId || '--'}` }} / {{ row.siteCode || '--' }}
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Amazon状态" width="120">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.orderStatus || '--' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="履约类型" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.fulfillmentType === 'fbm' ? 'warning' : 'success'">
              {{ formatFulfillmentType(row.fulfillmentType) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="工作流" min-width="240">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <el-tag size="small" :type="workflowTagType(row.workflowStatus)">
                {{ workflowLabel(row.workflowStatus) }}
              </el-tag>
              <span class="text-xs text-slate-500 dark:text-slate-400">
                采购 {{ statusLabel(row.procurementStatus) }} / 打印 {{ statusLabel(row.printStatus) }} / 物流 {{ statusLabel(row.logisticsStatus) }} / Amazon {{ statusLabel(row.amazonFeedbackStatus) }}
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="退货" min-width="140">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <el-tag size="small" :type="returnSummaryTagType(row.returnSummaryStatus)">
                {{ returnSummaryLabel(row.returnSummaryStatus) }}
              </el-tag>
              <span v-if="(row.returnRedirectCandidates || []).length" class="text-xs text-amber-500">
                有转寄候选
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="异常" min-width="220">
          <template #default="{ row }">
            <div v-if="row.exceptionCode" class="flex flex-col gap-1">
              <el-tag size="small" type="danger">{{ row.exceptionCode }}</el-tag>
              <span class="text-xs text-rose-500">{{ row.exceptionMessage || '--' }}</span>
            </div>
            <span v-else class="text-sm text-slate-400 dark:text-slate-500">无异常</span>
          </template>
        </el-table-column>
        <el-table-column label="买家 / 收货人" min-width="180">
          <template #default="{ row }">
            <div class="flex flex-col gap-1 text-sm">
              <span>{{ row.buyerName || '--' }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400">{{ row.address?.recipientName || '--' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="金额" width="130">
          <template #default="{ row }">{{ formatPrice(row.orderTotalAmount, row.currencyCode) }}</template>
        </el-table-column>
        <el-table-column prop="purchaseDate" label="下单时间" min-width="168" />
        <el-table-column prop="lastSynchronizedAt" label="最后同步" min-width="168" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <div class="flex items-center gap-2">
              <el-button type="primary" link @click="openDetail(row)">详情</el-button>
              <el-button
                v-if="canStart(row)"
                type="warning"
                link
                @click="startRowFulfillment(row)"
              >
                开始履约
              </el-button>
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
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRouter } from 'vue-router'

import {
  getAmazonOrderList,
  startAmazonOrderFulfillment
} from '@/api/amazonOrder'
import { getAmazonStoreList } from '@/api/amazonStore'

defineOptions({
  name: 'AmazonOrderFulfillmentList'
})

const router = useRouter()
const tableData = ref([])
const total = ref(0)
const storeOptions = ref([])

const searchInfo = ref({
  page: 1,
  pageSize: 10,
  storeId: undefined,
  siteCode: '',
  status: '',
  fulfillmentType: '',
  workflowStatus: '',
  returnSummaryStatus: '',
  hasRedirectCandidate: false,
  exceptionOnly: false,
  keyword: ''
})

const fetchStoreOptions = async () => {
  const res = await getAmazonStoreList({ page: 1, pageSize: 200 })
  if (res.code === 0) {
    storeOptions.value = res.data.list || []
  }
}

const fetchTable = async () => {
  const res = await getAmazonOrderList(searchInfo.value)
  if (res.code === 0) {
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  }
}

const resetSearch = async () => {
  searchInfo.value = {
    page: 1,
    pageSize: 10,
    storeId: undefined,
    siteCode: '',
    status: '',
    fulfillmentType: '',
    workflowStatus: '',
    returnSummaryStatus: '',
    hasRedirectCandidate: false,
    exceptionOnly: false,
    keyword: ''
  }
  await fetchTable()
}

const handleCurrentChange = async (page) => {
  searchInfo.value.page = page
  await fetchTable()
}

const handleSizeChange = async (pageSize) => {
  searchInfo.value.pageSize = pageSize
  searchInfo.value.page = 1
  await fetchTable()
}

const openDetail = (row) => {
  router.push({
    name: 'amazonOrderDetail',
    params: { id: row.id }
  })
}

const canStart = (row) => row?.fulfillmentType === 'fbm' && row?.workflowStatus === 'fbm_pending'

const startRowFulfillment = async (row) => {
  await ElMessageBox.confirm(`确认开始处理订单 ${row.amazonOrderId || row.id} 的 FBM 履约吗？`, '开始履约', {
    type: 'warning'
  })
  const res = await startAmazonOrderFulfillment({ id: row.id })
  if (res.code === 0) {
    ElMessage.success('履约任务已生成')
    await fetchTable()
    openDetail(row)
  }
}

const formatPrice = (price, currencyCode) => {
  if (price === null || typeof price === 'undefined') {
    return '--'
  }
  return `${currencyCode || ''} ${Number(price).toFixed(2)}`.trim()
}

const formatFulfillmentType = (value) => {
  switch (value) {
    case 'fba':
      return 'FBA'
    case 'fbm':
      return 'FBM'
    default:
      return '未知'
  }
}

const workflowLabel = (value) => {
  switch (value) {
    case 'fbm_pending':
      return '待履约'
    case 'fbm_exception':
      return '资料异常'
    case 'fbm_waiting_return_redirect':
      return '待转寄'
    case 'fbm_return_redirect_shipped':
      return '转寄已发货'
    case 'fulfillment_running':
      return '执行中'
    case 'fulfillment_completed':
      return '已完成'
    case 'fulfillment_failed':
      return '执行失败'
    case 'fba_closed':
      return 'FBA关闭'
    case 'closed':
      return '已关闭'
    default:
      return value || '--'
  }
}

const workflowTagType = (value) => {
  switch (value) {
    case 'fbm_pending':
    case 'fbm_waiting_return_redirect':
      return 'warning'
    case 'fbm_exception':
    case 'fulfillment_failed':
      return 'danger'
    case 'fulfillment_running':
      return 'primary'
    case 'fulfillment_completed':
    case 'fbm_return_redirect_shipped':
    case 'fba_closed':
      return 'success'
    default:
      return 'info'
  }
}

const returnSummaryLabel = (value) => {
  switch (value) {
    case 'none':
      return '无退货'
    case 'open':
      return '开放'
    case 'processing':
      return '处理中'
    case 'closed':
      return '已关闭'
    case 'closed_returnless':
      return '赠送关闭'
    case 'exception':
      return '异常'
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

const returnSummaryTagType = (value) => {
  switch (value) {
    case 'none':
      return 'info'
    case 'open':
      return 'warning'
    case 'processing':
    case 'linked':
      return 'primary'
    case 'closed':
    case 'closed_returnless':
      return 'success'
    case 'exception':
    case 'missing_order':
    case 'missing_item':
    case 'ambiguous':
    case 'manual_review':
      return 'danger'
    default:
      return 'info'
  }
}

const statusLabel = (value) => {
  switch (value) {
    case 'pending':
      return '待处理'
    case 'ready':
      return '就绪'
    case 'running':
      return '执行中'
    case 'completed':
      return '已完成'
    case 'failed':
      return '失败'
    case 'blocked':
      return '阻塞'
    case 'skipped':
      return '已跳过'
    case 'created':
      return '已下单'
    case 'picked_up':
      return '已揽收'
    case 'submitted':
      return '已回传'
    case 'success':
      return '成功'
    case 'opened':
      return '已打开'
    case 'return_redirect_pending':
      return '待转寄'
    case 'return_redirect_booked':
      return '已转寄'
    default:
      return value || '--'
  }
}

onMounted(async () => {
  await fetchStoreOptions()
  await fetchTable()
})
</script>
