<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON FINANCE REPORTS</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">Amazon 利润报表</h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            查看日报、周报、月报以及订单级利润快照，支持按站点、SKU、ASIN 和成本状态筛选。
          </p>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchAll">
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
            <el-option label="UK" value="UK" />
            <el-option label="DE" value="DE" />
          </el-select>
        </el-form-item>
        <el-form-item label="SKU">
          <el-input v-model="searchInfo.sellerSku" clearable class="!w-44" />
        </el-form-item>
        <el-form-item label="ASIN">
          <el-input v-model="searchInfo.asin" clearable class="!w-44" />
        </el-form-item>
        <el-form-item label="口径">
          <el-segmented v-model="searchInfo.basisType" :options="basisOptions" />
        </el-form-item>
        <el-form-item label="视图">
          <el-segmented v-model="searchInfo.dateView" :options="dateViewOptions" />
        </el-form-item>
        <el-form-item label="粒度">
          <el-select v-model="searchInfo.grain" class="!w-32">
            <el-option label="日" value="day" />
            <el-option label="周" value="week" />
            <el-option label="月" value="month" />
          </el-select>
        </el-form-item>
        <el-form-item label="日期范围">
          <el-date-picker
            v-model="searchInfo.dateRange"
            type="daterange"
            value-format="YYYY-MM-DD"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            class="!w-72"
          />
        </el-form-item>
        <el-form-item label="成本状态">
          <el-select v-model="searchInfo.actuality" clearable class="!w-36">
            <el-option label="全部" value="" />
            <el-option label="仅估算" value="estimated" />
          </el-select>
        </el-form-item>
        <el-form-item label="仅未对账">
          <el-switch v-model="searchInfo.onlyUnmatched" />
        </el-form-item>
        <el-form-item label="仅未结清">
          <el-switch v-model="searchInfo.onlyOutstanding" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="fetchAll">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-skeleton :loading="loading" animated :rows="8">
      <template #default>
        <section class="grid gap-4 md:grid-cols-3">
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="text-sm text-slate-500 dark:text-slate-400">销售额</div>
            <div class="mt-2 text-3xl font-semibold text-slate-900 dark:text-slate-100">{{ formatCny(summary.totals?.revenueCny) }}</div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="text-sm text-slate-500 dark:text-slate-400">毛利润</div>
            <div class="mt-2 text-3xl font-semibold" :class="profitClass(summary.totals?.grossProfitCny)">
              {{ formatCny(summary.totals?.grossProfitCny) }}
            </div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="text-sm text-slate-500 dark:text-slate-400">净利润</div>
            <div class="mt-2 text-3xl font-semibold" :class="profitClass(summary.totals?.netProfitCny)">
              {{ formatCny(summary.totals?.netProfitCny) }}
            </div>
          </div>
        </section>

        <section class="mt-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
          <div class="mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">期间汇总</div>
          <el-table :data="summary.rows || []" stripe>
            <el-table-column prop="periodStart" label="开始日期" min-width="120" />
            <el-table-column prop="periodEnd" label="结束日期" min-width="120" />
            <el-table-column prop="ordersCount" label="订单数" width="100" />
            <el-table-column label="销售额" min-width="140">
              <template #default="{ row }">{{ formatCny(row.revenueCny) }}</template>
            </el-table-column>
            <el-table-column label="毛利润" min-width="140">
              <template #default="{ row }">
                <span :class="profitClass(row.grossProfitCny)">{{ formatCny(row.grossProfitCny) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="净利润" min-width="140">
              <template #default="{ row }">
                <span :class="profitClass(row.netProfitCny)">{{ formatCny(row.netProfitCny) }}</span>
              </template>
            </el-table-column>
          </el-table>
        </section>

        <section class="mt-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
          <div class="mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">订单利润快照</div>
          <el-table :data="orders" stripe>
            <el-table-column label="订单" min-width="220">
              <template #default="{ row }">
                <div class="flex flex-col gap-1">
                  <span class="font-medium text-slate-900 dark:text-slate-100">{{ row.amazonOrderId || '--' }}</span>
                  <span class="text-xs text-slate-500 dark:text-slate-400">Store #{{ row.storeId || '--' }} / {{ row.siteCode || '--' }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="businessDate" label="业务日期" min-width="120" />
            <el-table-column label="销售额" min-width="120">
              <template #default="{ row }">{{ formatCny(row.revenueCny) }}</template>
            </el-table-column>
            <el-table-column label="毛利润" min-width="120">
              <template #default="{ row }"><span :class="profitClass(row.grossProfitCny)">{{ formatCny(row.grossProfitCny) }}</span></template>
            </el-table-column>
            <el-table-column label="净利润" min-width="120">
              <template #default="{ row }"><span :class="profitClass(row.netProfitCny)">{{ formatCny(row.netProfitCny) }}</span></template>
            </el-table-column>
            <el-table-column label="估算成本" min-width="120">
              <template #default="{ row }">{{ formatCny(row.estimatedCostCny) }}</template>
            </el-table-column>
            <el-table-column label="结算 / 应收" min-width="180">
              <template #default="{ row }">
                {{ statusLabel(row.settlementMatchStatus) }} / {{ statusLabel(row.receivableStatus) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button type="primary" link @click="openOrderProfit(row)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="gva-pagination">
            <el-pagination
              layout="total, sizes, prev, pager, next, jumper"
              :current-page="searchInfo.page"
              :page-size="searchInfo.pageSize"
              :page-sizes="[10, 20, 50, 100]"
              :total="ordersTotal"
              @current-change="handleCurrentChange"
              @size-change="handleSizeChange"
            />
          </div>
        </section>
      </template>
    </el-skeleton>

    <el-drawer v-model="orderDetailVisible" title="订单利润详情" size="72%">
      <el-table :data="orderProfitRows" stripe>
        <el-table-column prop="basisType" label="口径" width="100" />
        <el-table-column prop="dateView" label="视图" width="100" />
        <el-table-column prop="businessDate" label="业务日期" min-width="120" />
        <el-table-column label="毛利润" min-width="120">
          <template #default="{ row }">{{ formatCny(row.grossProfitCny) }}</template>
        </el-table-column>
        <el-table-column label="净利润" min-width="120">
          <template #default="{ row }">{{ formatCny(row.netProfitCny) }}</template>
        </el-table-column>
        <el-table-column label="成本拆解" min-width="620">
          <template #default="{ row }">
            <div class="grid gap-2 md:grid-cols-2 xl:grid-cols-3 text-sm text-slate-600 dark:text-slate-300">
              <div>采购：{{ formatCny(row.costBreakdown?.procurementCostCny) }}</div>
              <div>头程：{{ formatCny(row.costBreakdown?.firstLegCostCny) }}</div>
              <div>佣金：{{ formatCny(row.costBreakdown?.amazonReferralFeeCny) }}</div>
              <div>FBA配送：{{ formatCny(row.costBreakdown?.fbaFulfillmentFeeCny) }}</div>
              <div>仓储：{{ formatCny(row.costBreakdown?.storageFeeCny) }}</div>
              <div>广告：{{ formatCny(row.costBreakdown?.adCostCny) }}</div>
              <div>提现：{{ formatCny(row.costBreakdown?.withdrawalFeeCny) }}</div>
              <div>卡费：{{ formatCny(row.costBreakdown?.cardFeeCny) }}</div>
              <div>退货损耗：{{ formatCny(row.costBreakdown?.returnLossCny) }}</div>
              <div>退款：{{ formatCny(row.costBreakdown?.refundCostCny) }}</div>
              <div>赔付：{{ formatCny(row.costBreakdown?.compensationCny) }}</div>
              <div>补偿：{{ formatCny(row.costBreakdown?.reimbursementCny) }}</div>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'

import { getAmazonStoreList } from '@/api/amazonStore'
import {
  findAmazonFinanceOrderProfit,
  getAmazonFinanceReportOrders,
  getAmazonFinanceReportSummary
} from '@/api/amazonFinanceReport'

defineOptions({
  name: 'AmazonFinanceReports'
})

const loading = ref(false)
const storeOptions = ref([])
const summary = ref({ rows: [], totals: {} })
const orders = ref([])
const ordersTotal = ref(0)
const orderDetailVisible = ref(false)
const orderProfitRows = ref([])

const basisOptions = [
  { label: '权责', value: 'accrual' },
  { label: '现金', value: 'cash' }
]

const dateViewOptions = [
  { label: '下单日', value: 'purchase' },
  { label: '发货日', value: 'shipment' }
]

const searchInfo = reactive({
  page: 1,
  pageSize: 10,
  storeId: undefined,
  siteCode: '',
  sellerSku: '',
  asin: '',
  basisType: 'accrual',
  dateView: 'purchase',
  grain: 'day',
  actuality: '',
  onlyUnmatched: false,
  onlyOutstanding: false,
  dateRange: []
})

const fetchStoreOptions = async () => {
  const res = await getAmazonStoreList({ page: 1, pageSize: 200 })
  if (res.code === 0) {
    storeOptions.value = res.data.list || []
  }
}

const buildPayload = () => ({
  ...searchInfo,
  dateFrom: searchInfo.dateRange?.[0] || '',
  dateTo: searchInfo.dateRange?.[1] || ''
})

const fetchAll = async () => {
  loading.value = true
  try {
    const [summaryRes, ordersRes] = await Promise.all([
      getAmazonFinanceReportSummary(buildPayload()),
      getAmazonFinanceReportOrders(buildPayload())
    ])
    if (summaryRes.code === 0) {
      summary.value = summaryRes.data || { rows: [], totals: {} }
    }
    if (ordersRes.code === 0) {
      orders.value = ordersRes.data.list || []
      ordersTotal.value = ordersRes.data.total || 0
    }
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  searchInfo.page = 1
  searchInfo.pageSize = 10
  searchInfo.storeId = undefined
  searchInfo.siteCode = ''
  searchInfo.sellerSku = ''
  searchInfo.asin = ''
  searchInfo.basisType = 'accrual'
  searchInfo.dateView = 'purchase'
  searchInfo.grain = 'day'
  searchInfo.actuality = ''
  searchInfo.onlyUnmatched = false
  searchInfo.onlyOutstanding = false
  searchInfo.dateRange = []
  fetchAll()
}

const handleCurrentChange = (page) => {
  searchInfo.page = page
  fetchAll()
}

const handleSizeChange = (pageSize) => {
  searchInfo.page = 1
  searchInfo.pageSize = pageSize
  fetchAll()
}

const openOrderProfit = async (row) => {
  const res = await findAmazonFinanceOrderProfit({ orderId: row.orderId })
  if (res.code === 0) {
    orderProfitRows.value = res.data || []
    orderDetailVisible.value = true
  }
}

const formatCny = (value) => `CNY ${Number(value || 0).toFixed(2)}`

const profitClass = (value) => {
  if (Number(value || 0) > 0) return 'text-emerald-600'
  if (Number(value || 0) < 0) return 'text-rose-600'
  return 'text-slate-900 dark:text-slate-100'
}

const statusLabel = (value) => {
  switch (value) {
    case 'exact':
      return '已精确'
    case 'manual':
      return '已手工'
    case 'unmatched':
      return '未匹配'
    case 'settled':
      return '已结清'
    case 'partial':
      return '部分'
    case 'open':
      return '开放'
    default:
      return value || '--'
  }
}

onMounted(async () => {
  await fetchStoreOptions()
  await fetchAll()
})
</script>
