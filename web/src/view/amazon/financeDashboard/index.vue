<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON FINANCE DASHBOARD</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">Amazon 财务概览</h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            汇总订单利润、未结清应收应付、未对账结算、广告花费和汇率覆盖。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <el-button @click="fetchAll">刷新</el-button>
          <el-button type="primary" plain @click="openFxDialog">汇率覆盖</el-button>
          <el-button type="primary" @click="openAdsDialog">导入广告</el-button>
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
            <el-option v-for="site in amazonFinanceSiteOptions" :key="site.value" :label="site.label" :value="site.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="口径">
          <el-segmented v-model="searchInfo.basisType" :options="basisOptions" />
        </el-form-item>
        <el-form-item label="日期视图">
          <el-segmented v-model="searchInfo.dateView" :options="dateViewOptions" />
        </el-form-item>
        <el-form-item label="成本状态">
          <el-select v-model="searchInfo.actuality" clearable class="!w-36">
            <el-option label="全部" value="" />
            <el-option label="仅估算" value="estimated" />
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
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="fetchAll">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-skeleton :loading="loading" animated :rows="8">
      <template #default>
        <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="text-sm text-slate-500 dark:text-slate-400">销售额</div>
            <div class="mt-2 text-3xl font-semibold text-slate-900 dark:text-slate-100">{{ formatCny(overview.revenueCny) }}</div>
            <div class="mt-2 text-xs text-slate-500 dark:text-slate-400">订单数 {{ overview.orderCount || 0 }}</div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="text-sm text-slate-500 dark:text-slate-400">毛利润</div>
            <div class="mt-2 text-3xl font-semibold" :class="profitClass(overview.grossProfitCny)">
              {{ formatCny(overview.grossProfitCny) }}
            </div>
            <div class="mt-2 text-xs text-slate-500 dark:text-slate-400">净利润 {{ formatCny(overview.netProfitCny) }}</div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="text-sm text-slate-500 dark:text-slate-400">估算订单</div>
            <div class="mt-2 text-3xl font-semibold text-slate-900 dark:text-slate-100">{{ overview.estimatedOrderCount || 0 }}</div>
            <div class="mt-2 text-xs text-slate-500 dark:text-slate-400">待落实际费用的订单</div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="text-sm text-slate-500 dark:text-slate-400">未回收应收</div>
            <div class="mt-2 text-3xl font-semibold text-amber-600">{{ formatCny(overview.openReceivableCny) }}</div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="text-sm text-slate-500 dark:text-slate-400">未结清应付</div>
            <div class="mt-2 text-3xl font-semibold text-rose-600">{{ formatCny(overview.openPayableCny) }}</div>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="text-sm text-slate-500 dark:text-slate-400">未匹配结算 / 未落单广告</div>
            <div class="mt-2 text-3xl font-semibold text-slate-900 dark:text-slate-100">
              {{ overview.unmatchedSettlementLines || 0 }} / {{ overview.unallocatedAdsLines || 0 }}
            </div>
          </div>
        </section>

        <section class="mt-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
          <div class="mb-4 flex items-center justify-between">
            <div>
              <div class="text-base font-semibold text-slate-900 dark:text-slate-100">近 14 日利润趋势</div>
              <div class="text-sm text-slate-500 dark:text-slate-400">按当前口径和日期视图聚合。</div>
            </div>
          </div>
          <el-table :data="overview.recentTrend || []" stripe>
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

        <section class="mt-4 grid gap-4 xl:grid-cols-2">
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="mb-4 flex items-center justify-between">
              <div>
                <div class="text-base font-semibold text-slate-900 dark:text-slate-100">最新广告费用</div>
                <div class="text-sm text-slate-500 dark:text-slate-400">查看最近导入的广告花费与归因状态。</div>
              </div>
              <el-button type="primary" plain @click="openAdsDialog">导入广告</el-button>
            </div>
            <el-table :data="adsList" stripe>
              <el-table-column prop="adDate" label="日期" width="110" />
              <el-table-column label="广告维度" min-width="220">
                <template #default="{ row }">
                  <div class="flex flex-col gap-1">
                    <span class="font-medium text-slate-900 dark:text-slate-100">{{ row.campaignName || '--' }}</span>
                    <span class="text-xs text-slate-500 dark:text-slate-400">{{ row.sellerSku || row.asin || '--' }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="花费" width="140">
                <template #default="{ row }">{{ formatMoney(row.spendOriginal, row.currencyCode) }}</template>
              </el-table-column>
              <el-table-column label="人民币" width="130">
                <template #default="{ row }">{{ formatCny(row.spendCny) }}</template>
              </el-table-column>
              <el-table-column label="归因状态" width="120">
                <template #default="{ row }">
                  <el-tag size="small" :type="matchTagType(row.allocationStatus)">{{ matchLabel(row.allocationStatus) }}</el-tag>
                </template>
              </el-table-column>
            </el-table>
          </div>

          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="mb-4 flex items-center justify-between">
              <div>
                <div class="text-base font-semibold text-slate-900 dark:text-slate-100">汇率覆盖</div>
                <div class="text-sm text-slate-500 dark:text-slate-400">手工覆盖高于自动汇率，修改后会触发回算。</div>
              </div>
              <el-button type="primary" plain @click="openFxDialog">新增覆盖</el-button>
            </div>
            <el-table :data="fxList" stripe>
              <el-table-column prop="rateDate" label="日期" width="120" />
              <el-table-column prop="currencyCode" label="币种" width="90" />
              <el-table-column prop="rateToCny" label="汇率" width="120" />
              <el-table-column label="来源" width="120">
                <template #default="{ row }">
                  <el-tag size="small" :type="row.manualOverride ? 'warning' : 'info'">
                    {{ row.manualOverride ? '手工' : row.source || '--' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="reason" label="原因" min-width="180" />
            </el-table>
          </div>
        </section>
      </template>
    </el-skeleton>

    <el-dialog v-model="fxDialogVisible" title="汇率覆盖" width="520px" destroy-on-close>
      <el-form label-width="96px">
        <el-form-item label="币种">
          <el-select v-model="fxForm.currencyCode" class="!w-full">
            <el-option v-for="currency in financeFxCurrencyOptions" :key="currency.value" :label="currency.value" :value="currency.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="日期">
          <el-date-picker v-model="fxForm.rateDate" type="date" value-format="YYYY-MM-DD" class="!w-full" />
        </el-form-item>
        <el-form-item label="汇率">
          <el-input-number v-model="fxForm.rateToCny" :precision="6" :min="0" :step="0.01" class="!w-full" />
        </el-form-item>
        <el-form-item label="原因">
          <el-input v-model="fxForm.reason" type="textarea" :rows="3" placeholder="覆盖原因" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="fxDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="fxSaving" @click="submitFxOverride">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="adsDialogVisible" title="导入广告费用" width="1080px" destroy-on-close>
      <el-form label-width="92px">
        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <el-form-item label="店铺">
            <el-select v-model="adsForm.storeId" filterable class="!w-full">
              <el-option v-for="store in storeOptions" :key="store.id" :label="store.storeName" :value="store.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="站点">
            <el-select v-model="adsForm.siteCode" class="!w-full" @change="onAdsSiteChange">
              <el-option v-for="site in amazonFinanceSiteOptions" :key="site.value" :label="site.label" :value="site.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="广告账户">
            <el-input v-model="adsForm.accountName" />
          </el-form-item>
          <el-form-item label="币种">
            <el-select v-model="adsForm.currencyCode" class="!w-full" @change="onAdsCurrencyChange">
              <el-option v-for="currency in amazonCurrencyOptions" :key="currency.value" :label="currency.value" :value="currency.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="汇率">
            <el-input-number v-model="adsForm.fxRateToCny" :precision="6" :min="0" class="!w-full" />
          </el-form-item>
        </div>
        <el-form-item label="来源">
          <el-input v-model="adsForm.source" placeholder="manual / ads_api" />
        </el-form-item>
      </el-form>

      <div class="mb-3 flex justify-end">
        <el-button type="primary" plain @click="addAdsLine">新增广告行</el-button>
      </div>
      <el-table :data="adsForm.lines" border max-height="360">
        <el-table-column label="日期" min-width="140">
          <template #default="{ row }">
            <el-date-picker v-model="row.adDate" type="date" value-format="YYYY-MM-DD" class="!w-full" />
          </template>
        </el-table-column>
        <el-table-column label="订单ID" min-width="120">
          <template #default="{ row }">
            <el-input-number v-model="row.orderId" :min="0" controls-position="right" class="!w-full" />
          </template>
        </el-table-column>
        <el-table-column label="SKU / ASIN" min-width="220">
          <template #default="{ row }">
            <div class="grid gap-2">
              <el-input v-model="row.sellerSku" placeholder="Seller SKU" />
              <el-input v-model="row.asin" placeholder="ASIN" />
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Campaign" min-width="180">
          <template #default="{ row }">
            <el-input v-model="row.campaignName" />
          </template>
        </el-table-column>
        <el-table-column label="花费" min-width="120">
          <template #default="{ row }">
            <el-input-number v-model="row.spendOriginal" :min="0" :precision="2" class="!w-full" />
          </template>
        </el-table-column>
        <el-table-column label="点击 / 订单" min-width="160">
          <template #default="{ row }">
            <div class="grid gap-2">
              <el-input-number v-model="row.clicks" :min="0" controls-position="right" class="!w-full" />
              <el-input-number v-model="row.attributedOrders" :min="0" controls-position="right" class="!w-full" />
            </div>
          </template>
        </el-table-column>
        <el-table-column label="归因销售额" min-width="140">
          <template #default="{ row }">
            <el-input-number v-model="row.attributedSales" :min="0" :precision="2" class="!w-full" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="88" fixed="right">
          <template #default="{ $index }">
            <el-button type="danger" link @click="removeAdsLine($index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <el-button @click="adsDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="adsSaving" @click="submitAdsImport">导入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

import { getAmazonStoreList } from '@/api/amazonStore'
import { getAmazonFinanceDashboardOverview } from '@/api/amazonFinanceDashboard'
import { getAmazonFinanceFxList, saveAmazonFinanceFxOverride } from '@/api/amazonFinanceFx'
import { getAmazonFinanceAdsList, importAmazonFinanceAds } from '@/api/amazonFinanceAds'
import {
  amazonCurrencyOptions,
  amazonFinanceSiteOptions,
  amazonManagedCurrencyOptions,
  applyAmazonCurrencyFxRate,
  applyAmazonSiteCurrencyAndFx
} from '@/utils/amazonCurrency'

defineOptions({
  name: 'AmazonFinanceDashboard'
})

const loading = ref(false)
const fxSaving = ref(false)
const adsSaving = ref(false)
const fxDialogVisible = ref(false)
const adsDialogVisible = ref(false)
const storeOptions = ref([])
const overview = ref({
  recentTrend: []
})
const fxList = ref([])
const adsList = ref([])

const basisOptions = [
  { label: '权责', value: 'accrual' },
  { label: '现金', value: 'cash' }
]

const dateViewOptions = [
  { label: '下单日', value: 'purchase' },
  { label: '发货日', value: 'shipment' }
]

const financeFxCurrencyOptions = amazonManagedCurrencyOptions

const createDefaultFxForm = () => ({
  currencyCode: 'USD',
  rateDate: '',
  rateToCny: 7.1,
  reason: ''
})

const createEmptyAdsLine = () => ({
  adDate: '',
  orderId: undefined,
  sellerSku: '',
  asin: '',
  campaignName: '',
  spendOriginal: 0,
  clicks: 0,
  attributedOrders: 0,
  attributedSales: 0
})

const searchInfo = reactive({
  storeId: undefined,
  siteCode: '',
  basisType: 'accrual',
  dateView: 'purchase',
  actuality: '',
  dateRange: []
})

const fxForm = reactive(createDefaultFxForm())

const adsForm = reactive({
  storeId: undefined,
  siteCode: 'US',
  accountName: '',
  currencyCode: 'USD',
  fxRateToCny: undefined,
  source: 'manual',
  lines: [createEmptyAdsLine()]
})

const buildDatePayload = () => ({
  dateFrom: searchInfo.dateRange?.[0] || '',
  dateTo: searchInfo.dateRange?.[1] || ''
})

const fetchStoreOptions = async () => {
  const res = await getAmazonStoreList({ page: 1, pageSize: 200 })
  if (res.code === 0) {
    storeOptions.value = res.data.list || []
  }
}

const fetchOverview = async () => {
  const res = await getAmazonFinanceDashboardOverview({
    storeId: searchInfo.storeId,
    siteCode: searchInfo.siteCode,
    basisType: searchInfo.basisType,
    dateView: searchInfo.dateView,
    actuality: searchInfo.actuality,
    ...buildDatePayload()
  })
  if (res.code === 0) {
    overview.value = res.data || { recentTrend: [] }
  }
}

const fetchFX = async () => {
  const res = await getAmazonFinanceFxList({
    page: 1,
    pageSize: 12
  })
  if (res.code === 0) {
    fxList.value = res.data.list || []
  }
}

const fetchAds = async () => {
  const res = await getAmazonFinanceAdsList({
    page: 1,
    pageSize: 8,
    storeId: searchInfo.storeId,
    siteCode: searchInfo.siteCode,
    ...buildDatePayload()
  })
  if (res.code === 0) {
    adsList.value = res.data.list || []
  }
}

const fetchAll = async () => {
  loading.value = true
  try {
    await Promise.all([fetchOverview(), fetchFX(), fetchAds()])
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  searchInfo.storeId = undefined
  searchInfo.siteCode = ''
  searchInfo.basisType = 'accrual'
  searchInfo.dateView = 'purchase'
  searchInfo.actuality = ''
  searchInfo.dateRange = []
  fetchAll()
}

const resetFxForm = () => {
  Object.assign(fxForm, createDefaultFxForm())
}

const resetAdsForm = () => {
  adsForm.storeId = undefined
  adsForm.siteCode = 'US'
  adsForm.accountName = ''
  adsForm.currencyCode = 'USD'
  adsForm.fxRateToCny = undefined
  adsForm.source = 'manual'
  adsForm.lines = [createEmptyAdsLine()]
}

const openFxDialog = () => {
  resetFxForm()
  fxDialogVisible.value = true
}

const openAdsDialog = () => {
  resetAdsForm()
  adsDialogVisible.value = true
  applyAdsSiteFx(false)
}

const showMissingFxRate = (currencyCode) => {
  ElMessage.warning(`${currencyCode} 暂无可用汇率，请先在汇率管理维护`)
}

const applyAdsSiteFx = async (showWarning = true) => {
  const rate = await applyAmazonSiteCurrencyAndFx(adsForm, adsForm.siteCode)
  if (!rate && showWarning) {
    showMissingFxRate(adsForm.currencyCode)
  }
}

const applyAdsCurrencyFx = async (showWarning = true) => {
  const rate = await applyAmazonCurrencyFxRate(adsForm, adsForm.currencyCode)
  if (!rate && showWarning) {
    showMissingFxRate(adsForm.currencyCode)
  }
}

const onAdsSiteChange = () => {
  applyAdsSiteFx()
}

const onAdsCurrencyChange = () => {
  applyAdsCurrencyFx()
}

const addAdsLine = () => {
  adsForm.lines.push(createEmptyAdsLine())
}

const removeAdsLine = (index) => {
  adsForm.lines.splice(index, 1)
}

const submitFxOverride = async () => {
  fxSaving.value = true
  try {
    const res = await saveAmazonFinanceFxOverride({
      currencyCode: fxForm.currencyCode,
      rateDate: fxForm.rateDate,
      rateToCny: Number(fxForm.rateToCny || 0),
      reason: fxForm.reason
    })
    if (res.code === 0) {
      ElMessage.success('汇率覆盖已保存')
      fxDialogVisible.value = false
      resetFxForm()
      await fetchAll()
    }
  } finally {
    fxSaving.value = false
  }
}

const submitAdsImport = async () => {
  adsSaving.value = true
  try {
    const res = await importAmazonFinanceAds({
      ...adsForm,
      lines: adsForm.lines.map((line) => ({
        ...line,
        orderId: normalizeNullableNumber(line.orderId)
      }))
    })
    if (res.code === 0) {
      ElMessage.success('广告费用已导入')
      adsDialogVisible.value = false
      resetAdsForm()
      await fetchAll()
    }
  } finally {
    adsSaving.value = false
  }
}

const normalizeNullableNumber = (value) => {
  const numeric = Number(value)
  return numeric > 0 ? numeric : undefined
}

const formatMoney = (value, currencyCode) => {
  if (value === null || typeof value === 'undefined') return '--'
  return `${currencyCode || ''} ${Number(value).toFixed(2)}`.trim()
}

const formatCny = (value) => `CNY ${Number(value || 0).toFixed(2)}`

const profitClass = (value) => {
  if (Number(value || 0) > 0) return 'text-emerald-600'
  if (Number(value || 0) < 0) return 'text-rose-600'
  return 'text-slate-900 dark:text-slate-100'
}

const matchLabel = (value) => {
  switch (value) {
    case 'exact':
      return '精确'
    case 'manual':
      return '手工'
    case 'fuzzy':
      return '模糊'
    case 'unmatched':
      return '未落单'
    default:
      return value || '待处理'
  }
}

const matchTagType = (value) => {
  switch (value) {
    case 'exact':
      return 'success'
    case 'manual':
      return 'warning'
    case 'fuzzy':
      return 'info'
    case 'unmatched':
      return 'danger'
    default:
      return 'info'
  }
}

onMounted(async () => {
  await fetchStoreOptions()
  await fetchAll()
})
</script>
