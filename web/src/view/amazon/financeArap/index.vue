<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON FINANCE AR/AP</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">Amazon 应收应付</h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            追踪未回款订单、未结清账单和付款记录，支撑每日现金流核对。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <el-button @click="fetchCurrentTab">刷新</el-button>
          <el-button type="primary" @click="openPaymentDialog">登记付款</el-button>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchCurrentTab">
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
        <el-form-item label="状态">
          <el-select v-model="searchInfo.status" clearable class="!w-36">
            <el-option label="Open" value="open" />
            <el-option label="Partial" value="partial" />
            <el-option label="Settled" value="settled" />
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
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="备注 / 对手方" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchCurrentTab">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <el-tab-pane label="应收" name="receivables">
          <el-table :data="receivableRows" stripe>
            <el-table-column prop="sourceType" label="来源" width="120" />
            <el-table-column label="店铺 / 站点" width="140">
              <template #default="{ row }">{{ row.storeId || '--' }} / {{ row.siteCode || '--' }}</template>
            </el-table-column>
            <el-table-column prop="orderId" label="订单ID" min-width="100" />
            <el-table-column label="应收 / 已收 / 未收" min-width="260">
              <template #default="{ row }">
                {{ formatCny(row.amountCny) }} / {{ formatCny(row.receivedCny) }} / {{ formatCny(row.outstandingCny) }}
              </template>
            </el-table-column>
            <el-table-column prop="dueDate" label="到期日" min-width="120" />
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-tag size="small" :type="statusTagType(row.status)">{{ row.status || '--' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="notes" label="备注" min-width="180" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="应付" name="payables">
          <el-table :data="payableRows" stripe>
            <el-table-column prop="sourceType" label="来源" width="120" />
            <el-table-column label="对手方" min-width="180">
              <template #default="{ row }">{{ row.counterpartyName || '--' }}</template>
            </el-table-column>
            <el-table-column label="店铺 / 站点" width="140">
              <template #default="{ row }">{{ row.storeId || '--' }} / {{ row.siteCode || '--' }}</template>
            </el-table-column>
            <el-table-column label="应付 / 已付 / 未付" min-width="260">
              <template #default="{ row }">
                {{ formatCny(row.amountCny) }} / {{ formatCny(row.paidCny) }} / {{ formatCny(row.outstandingCny) }}
              </template>
            </el-table-column>
            <el-table-column prop="dueDate" label="到期日" min-width="120" />
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-tag size="small" :type="statusTagType(row.status)">{{ row.status || '--' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="notes" label="备注" min-width="180" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="付款记录" name="payments">
          <el-table :data="paymentRows" stripe>
            <el-table-column label="对手方" min-width="180">
              <template #default="{ row }">
                <div class="flex flex-col gap-1">
                  <span>{{ row.counterpartyName || '--' }}</span>
                  <span class="text-xs text-slate-500 dark:text-slate-400">{{ row.counterpartyType || '--' }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="金额" min-width="120">
              <template #default="{ row }">{{ formatMoney(row.amountOriginal, row.currencyCode) }}</template>
            </el-table-column>
            <el-table-column label="人民币" min-width="120">
              <template #default="{ row }">{{ formatCny(row.amountCny) }}</template>
            </el-table-column>
            <el-table-column label="手续费" min-width="160">
              <template #default="{ row }">
                {{ row.feeAmountCny ? formatCny(row.feeAmountCny) : '--' }}
              </template>
            </el-table-column>
            <el-table-column prop="paymentDate" label="付款日期" min-width="120" />
            <el-table-column prop="notes" label="备注" min-width="180" />
          </el-table>
        </el-tab-pane>
      </el-tabs>

      <div class="gva-pagination">
        <el-pagination
          layout="total, sizes, prev, pager, next, jumper"
          :current-page="pagination.page"
          :page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <el-dialog v-model="paymentVisible" title="登记付款" width="640px" destroy-on-close>
      <el-form label-width="110px">
        <div class="grid gap-4 md:grid-cols-2">
          <el-form-item label="店铺">
            <el-select v-model="paymentForm.storeId" filterable class="!w-full">
              <el-option v-for="store in storeOptions" :key="store.id" :label="store.storeName" :value="store.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="站点">
            <el-select v-model="paymentForm.siteCode" class="!w-full" @change="onPaymentSiteChange">
              <el-option v-for="site in amazonFinanceSiteOptions" :key="site.value" :label="site.label" :value="site.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="对手方类型">
            <el-input v-model="paymentForm.counterpartyType" placeholder="supplier / bank / platform" />
          </el-form-item>
          <el-form-item label="对手方名称">
            <el-input v-model="paymentForm.counterpartyName" />
          </el-form-item>
          <el-form-item label="关联账单类型">
            <el-input v-model="paymentForm.relatedBillType" placeholder="procurement / first_leg" />
          </el-form-item>
          <el-form-item label="关联账单ID">
            <el-input-number v-model="paymentForm.relatedBillId" :min="0" controls-position="right" class="!w-full" />
          </el-form-item>
          <el-form-item label="Settlement批次">
            <el-input-number v-model="paymentForm.relatedSettlementBatchId" :min="0" controls-position="right" class="!w-full" />
          </el-form-item>
          <el-form-item label="币种">
            <el-select v-model="paymentForm.currencyCode" class="!w-full" @change="onPaymentCurrencyChange">
              <el-option v-for="currency in amazonCurrencyOptions" :key="currency.value" :label="currency.value" :value="currency.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="付款金额">
            <el-input-number v-model="paymentForm.amountOriginal" :precision="2" :min="0" class="!w-full" />
          </el-form-item>
          <el-form-item label="汇率">
            <el-input-number v-model="paymentForm.fxRateToCny" :precision="6" :min="0" class="!w-full" />
          </el-form-item>
          <el-form-item label="费率 %">
            <el-input-number v-model="paymentForm.feeRate" :precision="4" :min="0" class="!w-full" />
          </el-form-item>
          <el-form-item label="手续费">
            <el-input-number v-model="paymentForm.feeAmountOriginal" :precision="2" :min="0" class="!w-full" />
          </el-form-item>
          <el-form-item label="付款日期">
            <el-date-picker v-model="paymentForm.paymentDate" type="date" value-format="YYYY-MM-DD" class="!w-full" />
          </el-form-item>
          <el-form-item label="备注">
            <el-input v-model="paymentForm.notes" type="textarea" :rows="3" />
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="paymentVisible = false">取消</el-button>
        <el-button type="primary" :loading="paymentSaving" @click="submitPayment">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

import { getAmazonStoreList } from '@/api/amazonStore'
import {
  getAmazonFinancePayables,
  getAmazonFinancePayments,
  getAmazonFinanceReceivables,
  saveAmazonFinancePayment
} from '@/api/amazonFinanceArap'
import {
  amazonCurrencyOptions,
  amazonFinanceSiteOptions,
  applyAmazonCurrencyFxRate,
  applyAmazonSiteCurrencyAndFx
} from '@/utils/amazonCurrency'

defineOptions({
  name: 'AmazonFinanceArap'
})

const storeOptions = ref([])
const activeTab = ref('receivables')
const receivableRows = ref([])
const payableRows = ref([])
const paymentRows = ref([])
const paymentVisible = ref(false)
const paymentSaving = ref(false)

const searchInfo = reactive({
  storeId: undefined,
  siteCode: '',
  status: '',
  keyword: '',
  dateRange: []
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const paymentForm = reactive({
  storeId: undefined,
  siteCode: 'US',
  counterpartyType: 'supplier',
  counterpartyName: '',
  relatedBillType: '',
  relatedBillId: undefined,
  relatedSettlementBatchId: undefined,
  currencyCode: 'USD',
  amountOriginal: 0,
  fxRateToCny: undefined,
  feeRate: undefined,
  feeAmountOriginal: undefined,
  paymentDate: '',
  notes: ''
})

const resetPaymentForm = () => {
  paymentForm.storeId = undefined
  paymentForm.siteCode = 'US'
  paymentForm.counterpartyType = 'supplier'
  paymentForm.counterpartyName = ''
  paymentForm.relatedBillType = ''
  paymentForm.relatedBillId = undefined
  paymentForm.relatedSettlementBatchId = undefined
  paymentForm.currencyCode = 'USD'
  paymentForm.amountOriginal = 0
  paymentForm.fxRateToCny = undefined
  paymentForm.feeRate = undefined
  paymentForm.feeAmountOriginal = undefined
  paymentForm.paymentDate = ''
  paymentForm.notes = ''
}

const openPaymentDialog = () => {
  resetPaymentForm()
  paymentVisible.value = true
  applyPaymentSiteFx(false)
}

const showMissingFxRate = (currencyCode) => {
  ElMessage.warning(`${currencyCode} 暂无可用汇率，请先在汇率管理维护`)
}

const applyPaymentSiteFx = async (showWarning = true) => {
  const rate = await applyAmazonSiteCurrencyAndFx(paymentForm, paymentForm.siteCode)
  if (!rate && showWarning) {
    showMissingFxRate(paymentForm.currencyCode)
  }
}

const applyPaymentCurrencyFx = async (showWarning = true) => {
  const rate = await applyAmazonCurrencyFxRate(paymentForm, paymentForm.currencyCode)
  if (!rate && showWarning) {
    showMissingFxRate(paymentForm.currencyCode)
  }
}

const onPaymentSiteChange = () => {
  applyPaymentSiteFx()
}

const onPaymentCurrencyChange = () => {
  applyPaymentCurrencyFx()
}

const fetchStoreOptions = async () => {
  const res = await getAmazonStoreList({ page: 1, pageSize: 200 })
  if (res.code === 0) {
    storeOptions.value = res.data.list || []
  }
}

const requestPayload = () => ({
  page: pagination.page,
  pageSize: pagination.pageSize,
  storeId: searchInfo.storeId,
  siteCode: searchInfo.siteCode,
  status: searchInfo.status,
  keyword: searchInfo.keyword,
  dateFrom: searchInfo.dateRange?.[0] || '',
  dateTo: searchInfo.dateRange?.[1] || ''
})

const fetchCurrentTab = async () => {
  const payload = requestPayload()
  let res
  if (activeTab.value === 'receivables') {
    res = await getAmazonFinanceReceivables(payload)
    if (res.code === 0) {
      receivableRows.value = res.data.list || []
    }
  } else if (activeTab.value === 'payables') {
    res = await getAmazonFinancePayables(payload)
    if (res.code === 0) {
      payableRows.value = res.data.list || []
    }
  } else {
    res = await getAmazonFinancePayments(payload)
    if (res.code === 0) {
      paymentRows.value = res.data.list || []
    }
  }
  if (res?.code === 0) {
    pagination.total = res.data.total || 0
  }
}

const handleTabChange = () => {
  pagination.page = 1
  fetchCurrentTab()
}

const handleCurrentChange = (page) => {
  pagination.page = page
  fetchCurrentTab()
}

const handleSizeChange = (pageSize) => {
  pagination.pageSize = pageSize
  pagination.page = 1
  fetchCurrentTab()
}

const resetSearch = () => {
  searchInfo.storeId = undefined
  searchInfo.siteCode = ''
  searchInfo.status = ''
  searchInfo.keyword = ''
  searchInfo.dateRange = []
  pagination.page = 1
  fetchCurrentTab()
}

const submitPayment = async () => {
  paymentSaving.value = true
  try {
    const res = await saveAmazonFinancePayment({
      ...paymentForm,
      relatedBillId: normalizeNullableNumber(paymentForm.relatedBillId),
      relatedSettlementBatchId: normalizeNullableNumber(paymentForm.relatedSettlementBatchId),
      fxRateToCny: normalizeNullableFloat(paymentForm.fxRateToCny),
      feeRate: normalizeNullableFloat(paymentForm.feeRate),
      feeAmountOriginal: normalizeNullableFloat(paymentForm.feeAmountOriginal)
    })
    if (res.code === 0) {
      ElMessage.success('付款记录已保存')
      paymentVisible.value = false
      resetPaymentForm()
      activeTab.value = 'payments'
      pagination.page = 1
      await fetchCurrentTab()
    }
  } finally {
    paymentSaving.value = false
  }
}

const normalizeNullableNumber = (value) => {
  const numeric = Number(value)
  return numeric > 0 ? numeric : undefined
}

const normalizeNullableFloat = (value) => {
  const numeric = Number(value)
  return numeric > 0 ? numeric : undefined
}

const formatMoney = (value, currencyCode) => `${currencyCode || ''} ${Number(value || 0).toFixed(2)}`.trim()
const formatCny = (value) => `CNY ${Number(value || 0).toFixed(2)}`

const statusTagType = (value) => {
  switch (value) {
    case 'settled':
      return 'success'
    case 'partial':
      return 'warning'
    default:
      return 'danger'
  }
}

onMounted(async () => {
  await fetchStoreOptions()
  await fetchCurrentTab()
})
</script>
