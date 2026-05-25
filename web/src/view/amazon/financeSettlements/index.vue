<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON FINANCE SETTLEMENTS</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">Amazon 结算对账</h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            导入结算单、自动匹配订单，并对未命中行做人工改配。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <el-button @click="fetchTable">刷新</el-button>
          <el-button type="primary" @click="openImportDialog">导入结算单</el-button>
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
            <el-option v-for="site in amazonFinanceSiteOptions" :key="site.value" :label="site.label" :value="site.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="匹配状态">
          <el-select v-model="searchInfo.matchStatus" clearable class="!w-40">
            <el-option label="待处理" value="pending" />
            <el-option label="精确" value="exact" />
            <el-option label="模糊" value="fuzzy" />
            <el-option label="手工" value="manual" />
            <el-option label="未匹配" value="unmatched" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="Settlement ID" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchTable">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table :data="tableData" stripe>
        <el-table-column label="结算单" min-width="220">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <span class="font-medium text-slate-900 dark:text-slate-100">{{ row.settlementId || '--' }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400">Store #{{ row.storeId || '--' }} / {{ row.siteCode || '--' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="期间" min-width="180">
          <template #default="{ row }">{{ row.postedStart || '--' }} ~ {{ row.postedEnd || '--' }}</template>
        </el-table-column>
        <el-table-column label="原币金额" min-width="120">
          <template #default="{ row }">{{ formatMoney(row.totalAmountOriginal, row.currencyCode) }}</template>
        </el-table-column>
        <el-table-column label="人民币" min-width="120">
          <template #default="{ row }">{{ formatCny(row.totalAmountCny) }}</template>
        </el-table-column>
        <el-table-column label="已匹配 / 未匹配" min-width="180">
          <template #default="{ row }">
            {{ formatCny(row.matchedAmountCny) }} / {{ formatCny(row.unmatchedAmountCny) }}
          </template>
        </el-table-column>
        <el-table-column label="匹配状态" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="matchTagType(row.matchStatus)">{{ matchLabel(row.matchStatus) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
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

    <el-drawer v-model="detailVisible" title="结算单详情" size="78%">
      <template v-if="currentDetail">
        <div class="mb-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <div class="rounded-xl border border-slate-200 p-3 dark:border-slate-700">结算单：{{ currentDetail.settlementId || '--' }}</div>
          <div class="rounded-xl border border-slate-200 p-3 dark:border-slate-700">店铺 / 站点：{{ currentDetail.storeId || '--' }} / {{ currentDetail.siteCode || '--' }}</div>
          <div class="rounded-xl border border-slate-200 p-3 dark:border-slate-700">总额：{{ formatCny(currentDetail.totalAmountCny) }}</div>
          <div class="rounded-xl border border-slate-200 p-3 dark:border-slate-700">状态：{{ matchLabel(currentDetail.matchStatus) }}</div>
        </div>
        <el-table :data="currentDetail.lines || []" stripe max-height="620">
          <el-table-column prop="postedAt" label="入账日" width="110" />
          <el-table-column prop="transactionType" label="类型" width="140" />
          <el-table-column label="订单 / 订单项" min-width="220">
            <template #default="{ row }">
              <div class="flex flex-col gap-1">
                <span>{{ row.amazonOrderId || '--' }}</span>
                <span class="text-xs text-slate-500 dark:text-slate-400">{{ row.amazonOrderItemId || row.orderItemId || '--' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="SKU / ASIN" min-width="180">
            <template #default="{ row }">
              <div class="flex flex-col gap-1">
                <span>{{ row.sellerSku || '--' }}</span>
                <span class="text-xs text-slate-500 dark:text-slate-400">{{ row.asin || '--' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="描述" min-width="180" />
          <el-table-column label="金额" width="120">
            <template #default="{ row }">{{ formatMoney(row.amountOriginal, row.currencyCode) }}</template>
          </el-table-column>
          <el-table-column label="匹配" width="140">
            <template #default="{ row }">
              <el-tag size="small" :type="matchTagType(row.matchStatus)">{{ matchLabel(row.matchStatus) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="说明" min-width="180">
            <template #default="{ row }">{{ row.matchReason || '--' }}</template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="row.matchStatus !== 'exact' && row.matchStatus !== 'manual'"
                type="warning"
                link
                @click="openMatchDialog(row)"
              >
                手工改配
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </el-drawer>

    <el-dialog v-model="importVisible" title="导入结算单" width="1120px" destroy-on-close>
      <el-form label-width="96px">
        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <el-form-item label="店铺">
            <el-select v-model="importForm.storeId" filterable class="!w-full">
              <el-option v-for="store in storeOptions" :key="store.id" :label="store.storeName" :value="store.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="站点">
            <el-select v-model="importForm.siteCode" class="!w-full" @change="onImportSiteChange">
              <el-option v-for="site in amazonFinanceSiteOptions" :key="site.value" :label="site.label" :value="site.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="Settlement ID">
            <el-input v-model="importForm.settlementId" />
          </el-form-item>
          <el-form-item label="币种">
            <el-select v-model="importForm.currencyCode" class="!w-full" @change="onImportCurrencyChange">
              <el-option v-for="currency in amazonCurrencyOptions" :key="currency.value" :label="currency.value" :value="currency.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="汇率">
            <el-input-number v-model="importForm.fxRateToCny" :precision="6" :min="0" class="!w-full" />
          </el-form-item>
        </div>
        <div class="grid gap-4 md:grid-cols-3">
          <el-form-item label="来源">
            <el-input v-model="importForm.source" />
          </el-form-item>
          <el-form-item label="开始日期">
            <el-date-picker v-model="importForm.postedStart" type="date" value-format="YYYY-MM-DD" class="!w-full" />
          </el-form-item>
          <el-form-item label="结束日期">
            <el-date-picker v-model="importForm.postedEnd" type="date" value-format="YYYY-MM-DD" class="!w-full" />
          </el-form-item>
        </div>
      </el-form>

      <div class="mb-3 flex justify-end">
        <el-button type="primary" plain @click="addImportLine">新增结算行</el-button>
      </div>
      <el-table :data="importForm.lines" border max-height="360">
        <el-table-column label="入账日" min-width="130">
          <template #default="{ row }">
            <el-date-picker v-model="row.postedAt" type="date" value-format="YYYY-MM-DD" class="!w-full" />
          </template>
        </el-table-column>
        <el-table-column label="类型" min-width="160">
          <template #default="{ row }">
            <el-input v-model="row.transactionType" placeholder="revenue / referral_fee" />
          </template>
        </el-table-column>
        <el-table-column label="订单 / 订单项" min-width="220">
          <template #default="{ row }">
            <div class="grid gap-2">
              <el-input v-model="row.amazonOrderId" placeholder="Amazon Order ID" />
              <el-input v-model="row.amazonOrderItemId" placeholder="Amazon Order Item ID" />
            </div>
          </template>
        </el-table-column>
        <el-table-column label="SKU / ASIN" min-width="180">
          <template #default="{ row }">
            <div class="grid gap-2">
              <el-input v-model="row.sellerSku" placeholder="Seller SKU" />
              <el-input v-model="row.asin" placeholder="ASIN" />
            </div>
          </template>
        </el-table-column>
        <el-table-column label="描述" min-width="180">
          <template #default="{ row }">
            <el-input v-model="row.description" />
          </template>
        </el-table-column>
        <el-table-column label="金额" min-width="130">
          <template #default="{ row }">
            <el-input-number v-model="row.amountOriginal" :precision="2" class="!w-full" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="88">
          <template #default="{ $index }">
            <el-button type="danger" link @click="removeImportLine($index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="importVisible = false">取消</el-button>
        <el-button type="primary" :loading="importLoading" @click="submitImport">导入</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="matchVisible" title="手工改配结算行" width="520px" destroy-on-close>
      <el-form label-width="100px">
        <el-form-item label="当前行">
          <div class="text-sm text-slate-500 dark:text-slate-400">
            {{ selectedLine?.transactionType || '--' }} / {{ selectedLine?.amazonOrderId || selectedLine?.sellerSku || '--' }}
          </div>
        </el-form-item>
        <el-form-item label="订单ID">
          <el-input-number v-model="matchForm.orderId" :min="0" controls-position="right" class="!w-full" />
        </el-form-item>
        <el-form-item label="订单项ID">
          <el-input-number v-model="matchForm.orderItemId" :min="0" controls-position="right" class="!w-full" />
        </el-form-item>
        <el-form-item label="原因">
          <el-input v-model="matchForm.reason" type="textarea" :rows="3" placeholder="记录人工匹配原因" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="matchVisible = false">取消</el-button>
        <el-button type="primary" :loading="matchLoading" @click="submitManualMatch">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

import { getAmazonStoreList } from '@/api/amazonStore'
import {
  findAmazonFinanceSettlement,
  getAmazonFinanceSettlementList,
  importAmazonFinanceSettlement,
  manualMatchAmazonFinanceSettlement
} from '@/api/amazonFinanceSettlement'
import {
  amazonCurrencyOptions,
  amazonFinanceSiteOptions,
  applyAmazonCurrencyFxRate,
  applyAmazonSiteCurrencyAndFx
} from '@/utils/amazonCurrency'

defineOptions({
  name: 'AmazonFinanceSettlements'
})

const storeOptions = ref([])
const tableData = ref([])
const total = ref(0)
const detailVisible = ref(false)
const importVisible = ref(false)
const matchVisible = ref(false)
const importLoading = ref(false)
const matchLoading = ref(false)
const currentDetail = ref(null)
const selectedLine = ref(null)

const searchInfo = reactive({
  page: 1,
  pageSize: 10,
  storeId: undefined,
  siteCode: '',
  matchStatus: '',
  keyword: ''
})

const createImportLine = () => ({
  postedAt: '',
  transactionType: 'revenue',
  amazonOrderId: '',
  amazonOrderItemId: '',
  sellerSku: '',
  asin: '',
  description: '',
  amountOriginal: 0
})

const importForm = reactive({
  storeId: undefined,
  siteCode: 'US',
  settlementId: '',
  currencyCode: 'USD',
  fxRateToCny: undefined,
  source: 'manual',
  postedStart: '',
  postedEnd: '',
  lines: [createImportLine()]
})

const matchForm = reactive({
  lineId: undefined,
  orderId: undefined,
  orderItemId: undefined,
  reason: ''
})

const fetchStoreOptions = async () => {
  const res = await getAmazonStoreList({ page: 1, pageSize: 200 })
  if (res.code === 0) {
    storeOptions.value = res.data.list || []
  }
}

const fetchTable = async () => {
  const res = await getAmazonFinanceSettlementList(searchInfo)
  if (res.code === 0) {
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  }
}

const resetSearch = () => {
  searchInfo.page = 1
  searchInfo.pageSize = 10
  searchInfo.storeId = undefined
  searchInfo.siteCode = ''
  searchInfo.matchStatus = ''
  searchInfo.keyword = ''
  fetchTable()
}

const handleCurrentChange = (page) => {
  searchInfo.page = page
  fetchTable()
}

const handleSizeChange = (pageSize) => {
  searchInfo.pageSize = pageSize
  searchInfo.page = 1
  fetchTable()
}

const openDetail = async (row) => {
  const res = await findAmazonFinanceSettlement({ id: row.id })
  if (res.code === 0) {
    currentDetail.value = res.data
    detailVisible.value = true
  }
}

const openImportDialog = () => {
  resetImportForm()
  importVisible.value = true
  applyImportSiteFx(false)
}

const resetImportForm = () => {
  importForm.storeId = undefined
  importForm.siteCode = 'US'
  importForm.settlementId = ''
  importForm.currencyCode = 'USD'
  importForm.fxRateToCny = undefined
  importForm.source = 'manual'
  importForm.postedStart = ''
  importForm.postedEnd = ''
  importForm.lines = [createImportLine()]
}

const resetMatchForm = () => {
  matchForm.lineId = undefined
  matchForm.orderId = undefined
  matchForm.orderItemId = undefined
  matchForm.reason = ''
  selectedLine.value = null
}

const addImportLine = () => {
  importForm.lines.push(createImportLine())
}

const removeImportLine = (index) => {
  importForm.lines.splice(index, 1)
}

const showMissingFxRate = (currencyCode) => {
  ElMessage.warning(`${currencyCode} 暂无可用汇率，请先在汇率管理维护`)
}

const applyImportSiteFx = async (showWarning = true) => {
  const rate = await applyAmazonSiteCurrencyAndFx(importForm, importForm.siteCode)
  if (!rate && showWarning) {
    showMissingFxRate(importForm.currencyCode)
  }
}

const applyImportCurrencyFx = async (showWarning = true) => {
  const rate = await applyAmazonCurrencyFxRate(importForm, importForm.currencyCode)
  if (!rate && showWarning) {
    showMissingFxRate(importForm.currencyCode)
  }
}

const onImportSiteChange = () => {
  applyImportSiteFx()
}

const onImportCurrencyChange = () => {
  applyImportCurrencyFx()
}

const submitImport = async () => {
  importLoading.value = true
  try {
    const res = await importAmazonFinanceSettlement(importForm)
    if (res.code === 0) {
      ElMessage.success('结算单已导入')
      importVisible.value = false
      resetImportForm()
      currentDetail.value = res.data
      detailVisible.value = true
      await fetchTable()
    }
  } finally {
    importLoading.value = false
  }
}

const openMatchDialog = (line) => {
  resetMatchForm()
  selectedLine.value = line
  matchForm.lineId = line.id
  matchForm.orderId = line.orderId
  matchForm.orderItemId = line.orderItemId
  matchForm.reason = line.matchReason || ''
  matchVisible.value = true
}

const submitManualMatch = async () => {
  matchLoading.value = true
  try {
    const res = await manualMatchAmazonFinanceSettlement({
      lineId: matchForm.lineId,
      matchType: 'manual',
      orderId: normalizeNullableNumber(matchForm.orderId),
      orderItemId: normalizeNullableNumber(matchForm.orderItemId),
      reason: matchForm.reason
    })
    if (res.code === 0) {
      ElMessage.success('结算行已改配')
      currentDetail.value = res.data
      matchVisible.value = false
      resetMatchForm()
      await fetchTable()
    }
  } finally {
    matchLoading.value = false
  }
}

const normalizeNullableNumber = (value) => {
  const numeric = Number(value)
  return numeric > 0 ? numeric : undefined
}

const formatMoney = (value, currencyCode) => `${currencyCode || ''} ${Number(value || 0).toFixed(2)}`.trim()
const formatCny = (value) => `CNY ${Number(value || 0).toFixed(2)}`

const matchLabel = (value) => {
  switch (value) {
    case 'exact':
      return '精确'
    case 'manual':
      return '手工'
    case 'fuzzy':
      return '模糊'
    case 'unmatched':
      return '未匹配'
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
  await fetchTable()
})
</script>
