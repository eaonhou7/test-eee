<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON FINANCE COST BILLS</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">Amazon 成本账单</h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            录入采购、头程、提现、信用卡和手工调整账单，并回算到订单利润。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <el-button @click="fetchTable">刷新</el-button>
          <el-button type="primary" @click="openSaveDialog()">新增账单</el-button>
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
        <el-form-item label="账单类型">
          <el-select v-model="searchInfo.billType" clearable class="!w-44">
            <el-option label="采购" value="procurement" />
            <el-option label="头程" value="first_leg" />
            <el-option label="提现" value="withdrawal" />
            <el-option label="信用卡" value="card_fee" />
            <el-option label="手工调整" value="manual_adjustment" />
          </el-select>
        </el-form-item>
        <el-form-item label="付款状态">
          <el-select v-model="searchInfo.paymentStatus" clearable class="!w-40">
            <el-option label="未付" value="unpaid" />
            <el-option label="部分" value="partial" />
            <el-option label="已付" value="paid" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="账单号 / 供应商" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchTable">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table :data="tableData" stripe>
        <el-table-column label="账单" min-width="220">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <span class="font-medium text-slate-900 dark:text-slate-100">{{ row.billNo || '--' }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400">{{ row.vendorName || '--' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="120">
          <template #default="{ row }">{{ billTypeLabel(row.billType) }}</template>
        </el-table-column>
        <el-table-column label="店铺 / 站点" width="140">
          <template #default="{ row }">{{ row.storeId || '--' }} / {{ row.siteCode || '--' }}</template>
        </el-table-column>
        <el-table-column prop="billDate" label="账单日期" min-width="120" />
        <el-table-column label="原币金额" min-width="130">
          <template #default="{ row }">{{ formatMoney(row.totalAmountOriginal, row.currencyCode) }}</template>
        </el-table-column>
        <el-table-column label="人民币" min-width="120">
          <template #default="{ row }">{{ formatCny(row.totalAmountCny) }}</template>
        </el-table-column>
        <el-table-column label="付款状态" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="paymentTagType(row.paymentStatus)">{{ paymentStatusLabel(row.paymentStatus) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <div class="flex items-center gap-2">
              <el-button type="primary" link @click="openDetail(row)">详情</el-button>
              <el-button type="warning" link @click="openSaveDialog(row.id)">编辑</el-button>
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

    <el-drawer v-model="detailVisible" title="账单详情" size="74%">
      <template v-if="currentDetail">
        <div class="mb-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <div class="rounded-xl border border-slate-200 p-3 dark:border-slate-700">账单号：{{ currentDetail.billNo || '--' }}</div>
          <div class="rounded-xl border border-slate-200 p-3 dark:border-slate-700">类型：{{ billTypeLabel(currentDetail.billType) }}</div>
          <div class="rounded-xl border border-slate-200 p-3 dark:border-slate-700">供应商：{{ currentDetail.vendorName || '--' }}</div>
          <div class="rounded-xl border border-slate-200 p-3 dark:border-slate-700">总额：{{ formatCny(currentDetail.totalAmountCny) }}</div>
        </div>
        <el-table :data="currentDetail.lines || []" stripe max-height="620">
          <el-table-column prop="sellerSku" label="SKU" min-width="150" />
          <el-table-column prop="asin" label="ASIN" min-width="130" />
          <el-table-column prop="orderId" label="订单ID" min-width="100" />
          <el-table-column prop="orderItemId" label="订单项ID" min-width="110" />
          <el-table-column prop="quantity" label="数量" width="90" />
          <el-table-column label="原币金额" min-width="120">
            <template #default="{ row }">{{ formatMoney(row.amountOriginal, currentDetail.currencyCode) }}</template>
          </el-table-column>
          <el-table-column label="人民币" min-width="120">
            <template #default="{ row }">{{ formatCny(row.amountCny) }}</template>
          </el-table-column>
          <el-table-column label="分摊状态" width="120">
            <template #default="{ row }">
              <el-tag size="small" :type="allocationTagType(row.allocationStatus)">{{ allocationLabel(row.allocationStatus) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="allocationMessage" label="说明" min-width="180" />
        </el-table>
      </template>
    </el-drawer>

    <el-dialog v-model="saveVisible" :title="saveForm.id ? '编辑账单' : '新增账单'" width="1120px" destroy-on-close>
      <el-form label-width="92px">
        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <el-form-item label="账单类型">
            <el-select v-model="saveForm.billType" class="!w-full">
              <el-option label="采购" value="procurement" />
              <el-option label="头程" value="first_leg" />
              <el-option label="提现" value="withdrawal" />
              <el-option label="信用卡" value="card_fee" />
              <el-option label="手工调整" value="manual_adjustment" />
            </el-select>
          </el-form-item>
          <el-form-item label="账单号">
            <el-input v-model="saveForm.billNo" />
          </el-form-item>
          <el-form-item label="店铺">
            <el-select v-model="saveForm.storeId" filterable class="!w-full">
              <el-option v-for="store in storeOptions" :key="store.id" :label="store.storeName" :value="store.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="站点">
            <el-select v-model="saveForm.siteCode" class="!w-full" @change="onSaveSiteChange">
              <el-option v-for="site in amazonFinanceSiteOptions" :key="site.value" :label="site.label" :value="site.value" />
            </el-select>
          </el-form-item>
        </div>
        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <el-form-item label="供应商">
            <el-input v-model="saveForm.vendorName" />
          </el-form-item>
          <el-form-item label="币种">
            <el-select v-model="saveForm.currencyCode" class="!w-full" @change="onSaveCurrencyChange">
              <el-option v-for="currency in amazonCurrencyOptions" :key="currency.value" :label="currency.value" :value="currency.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="账单日期">
            <el-date-picker v-model="saveForm.billDate" type="date" value-format="YYYY-MM-DD" class="!w-full" />
          </el-form-item>
          <el-form-item label="到期日期">
            <el-date-picker v-model="saveForm.dueDate" type="date" value-format="YYYY-MM-DD" class="!w-full" />
          </el-form-item>
        </div>
        <div class="grid gap-4 md:grid-cols-2">
          <el-form-item label="汇率">
            <el-input-number v-model="saveForm.fxRateToCny" :precision="6" :min="0" class="!w-full" />
          </el-form-item>
          <el-form-item label="备注">
            <el-input v-model="saveForm.notes" />
          </el-form-item>
        </div>
      </el-form>

      <div class="mb-3 flex justify-end">
        <el-button type="primary" plain @click="addBillLine">新增账单行</el-button>
      </div>
      <el-table :data="saveForm.lines" border max-height="360">
        <el-table-column label="订单 / 订单项" min-width="180">
          <template #default="{ row }">
            <div class="grid gap-2">
              <el-input-number v-model="row.orderId" :min="0" controls-position="right" class="!w-full" />
              <el-input-number v-model="row.orderItemId" :min="0" controls-position="right" class="!w-full" />
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
        <el-table-column label="数量" min-width="100">
          <template #default="{ row }">
            <el-input-number v-model="row.quantity" :min="0" controls-position="right" class="!w-full" />
          </template>
        </el-table-column>
        <el-table-column label="原币金额" min-width="120">
          <template #default="{ row }">
            <el-input-number v-model="row.amountOriginal" :precision="2" :min="0" class="!w-full" />
          </template>
        </el-table-column>
        <el-table-column label="备注" min-width="180">
          <template #default="{ row }">
            <el-input v-model="row.notes" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="88">
          <template #default="{ $index }">
            <el-button type="danger" link @click="removeBillLine($index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <el-button @click="saveVisible = false">取消</el-button>
        <el-button type="primary" :loading="saveLoading" @click="submitSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

import { getAmazonStoreList } from '@/api/amazonStore'
import {
  findAmazonFinanceCostBill,
  getAmazonFinanceCostBillList,
  saveAmazonFinanceCostBill
} from '@/api/amazonFinanceCostBill'
import {
  amazonCurrencyOptions,
  amazonFinanceSiteOptions,
  applyAmazonCurrencyFxRate,
  applyAmazonSiteCurrencyAndFx
} from '@/utils/amazonCurrency'

defineOptions({
  name: 'AmazonFinanceCostBills'
})

const storeOptions = ref([])
const tableData = ref([])
const total = ref(0)
const detailVisible = ref(false)
const saveVisible = ref(false)
const saveLoading = ref(false)
const currentDetail = ref(null)

const searchInfo = reactive({
  page: 1,
  pageSize: 10,
  billType: '',
  storeId: undefined,
  siteCode: '',
  paymentStatus: '',
  keyword: ''
})

const saveForm = reactive({
  id: 0,
  billType: 'procurement',
  billNo: '',
  storeId: undefined,
  siteCode: 'US',
  vendorName: '',
  currencyCode: 'USD',
  billDate: '',
  dueDate: '',
  fxRateToCny: undefined,
  notes: '',
  lines: []
})

const fetchStoreOptions = async () => {
  const res = await getAmazonStoreList({ page: 1, pageSize: 200 })
  if (res.code === 0) {
    storeOptions.value = res.data.list || []
  }
}

const fetchTable = async () => {
  const res = await getAmazonFinanceCostBillList(searchInfo)
  if (res.code === 0) {
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  }
}

const resetSearch = () => {
  searchInfo.page = 1
  searchInfo.pageSize = 10
  searchInfo.billType = ''
  searchInfo.storeId = undefined
  searchInfo.siteCode = ''
  searchInfo.paymentStatus = ''
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
  const res = await findAmazonFinanceCostBill({ id: row.id })
  if (res.code === 0) {
    currentDetail.value = res.data
    detailVisible.value = true
  }
}

const resetSaveForm = () => {
  saveForm.id = 0
  saveForm.billType = 'procurement'
  saveForm.billNo = ''
  saveForm.storeId = undefined
  saveForm.siteCode = 'US'
  saveForm.vendorName = ''
  saveForm.currencyCode = 'USD'
  saveForm.billDate = ''
  saveForm.dueDate = ''
  saveForm.fxRateToCny = undefined
  saveForm.notes = ''
  saveForm.lines = []
}

const openSaveDialog = async (id) => {
  resetSaveForm()
  if (id) {
    const res = await findAmazonFinanceCostBill({ id })
    if (res.code === 0) {
      Object.assign(saveForm, {
        ...res.data,
        lines: (res.data.lines || []).map((line) => ({
          orderId: line.orderId,
          orderItemId: line.orderItemId,
          sellerSku: line.sellerSku,
          asin: line.asin,
          quantity: line.quantity,
          amountOriginal: line.amountOriginal,
          notes: line.notes
        }))
      })
    }
  } else {
    addBillLine()
    applySaveSiteFx(false)
  }
  saveVisible.value = true
}

const showMissingFxRate = (currencyCode) => {
  ElMessage.warning(`${currencyCode} 暂无可用汇率，请先在汇率管理维护`)
}

const applySaveSiteFx = async (showWarning = true) => {
  const rate = await applyAmazonSiteCurrencyAndFx(saveForm, saveForm.siteCode)
  if (!rate && showWarning) {
    showMissingFxRate(saveForm.currencyCode)
  }
}

const applySaveCurrencyFx = async (showWarning = true) => {
  const rate = await applyAmazonCurrencyFxRate(saveForm, saveForm.currencyCode)
  if (!rate && showWarning) {
    showMissingFxRate(saveForm.currencyCode)
  }
}

const onSaveSiteChange = () => {
  applySaveSiteFx()
}

const onSaveCurrencyChange = () => {
  applySaveCurrencyFx()
}

const addBillLine = () => {
  saveForm.lines.push({
    orderId: undefined,
    orderItemId: undefined,
    sellerSku: '',
    asin: '',
    quantity: 0,
    amountOriginal: 0,
    notes: ''
  })
}

const removeBillLine = (index) => {
  saveForm.lines.splice(index, 1)
}

const submitSave = async () => {
  saveLoading.value = true
  try {
    const res = await saveAmazonFinanceCostBill({
      ...saveForm,
      id: Number(saveForm.id || 0),
      fxRateToCny: saveForm.fxRateToCny ? Number(saveForm.fxRateToCny) : undefined,
      lines: saveForm.lines.map((line) => ({
        ...line,
        orderId: normalizeNullableNumber(line.orderId),
        orderItemId: normalizeNullableNumber(line.orderItemId)
      }))
    })
    if (res.code === 0) {
      ElMessage.success('账单已保存')
      saveVisible.value = false
      currentDetail.value = res.data
      detailVisible.value = true
      await fetchTable()
    }
  } finally {
    saveLoading.value = false
  }
}

const normalizeNullableNumber = (value) => {
  const numeric = Number(value)
  return numeric > 0 ? numeric : undefined
}

const formatMoney = (value, currencyCode) => `${currencyCode || ''} ${Number(value || 0).toFixed(2)}`.trim()
const formatCny = (value) => `CNY ${Number(value || 0).toFixed(2)}`

const billTypeLabel = (value) => {
  switch (value) {
    case 'procurement':
      return '采购'
    case 'first_leg':
      return '头程'
    case 'withdrawal':
      return '提现'
    case 'card_fee':
      return '信用卡'
    case 'manual_adjustment':
      return '手工调整'
    default:
      return value || '--'
  }
}

const paymentStatusLabel = (value) => {
  switch (value) {
    case 'paid':
      return '已付'
    case 'partial':
      return '部分'
    default:
      return '未付'
  }
}

const paymentTagType = (value) => {
  switch (value) {
    case 'paid':
      return 'success'
    case 'partial':
      return 'warning'
    default:
      return 'danger'
  }
}

const allocationLabel = (value) => {
  switch (value) {
    case 'exact':
      return '精确'
    case 'manual':
      return '手工'
    case 'fuzzy':
      return '模糊'
    case 'unmatched':
      return '未分配'
    default:
      return value || '待处理'
  }
}

const allocationTagType = (value) => {
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
