<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs text-slate-500">AMAZON FINANCE FX</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">汇率管理</h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            管理 11 种主要流通货币兑人民币汇率，自动汇率每天刷新，手工覆盖后会触发财务回算。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <el-button :loading="loading" @click="fetchAll">刷新列表</el-button>
          <el-button type="primary" :loading="refreshing" @click="refreshDailyRates">立即刷新</el-button>
          <el-button type="primary" plain @click="openOverrideDialog()">手工覆盖</el-button>
        </div>
      </div>
    </div>

    <div class="gva-table-box">
      <div class="mb-4 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <div class="text-base font-semibold text-slate-900 dark:text-slate-100">今日汇率概览</div>
          <div class="text-sm text-slate-500 dark:text-slate-400">{{ today }} 兑人民币汇率</div>
        </div>
        <a
          class="text-sm text-blue-500 hover:text-blue-600"
          href="https://www.exchangerate-api.com"
          target="_blank"
          rel="noopener noreferrer"
        >
          Rates by ExchangeRate-API
        </a>
      </div>

      <el-skeleton :loading="todayLoading" animated :rows="3">
        <template #default>
          <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
            <div
              v-for="currency in currencyOptions"
              :key="currency.value"
              class="rounded-lg border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/50"
            >
              <div class="flex items-center justify-between gap-2">
                <div>
                  <div class="text-base font-semibold text-slate-900 dark:text-slate-100">{{ currency.value }}</div>
                  <div class="text-xs text-slate-500 dark:text-slate-400">{{ currency.label }}</div>
                </div>
                <el-tag size="small" :type="sourceTagType(todayRateMap[currency.value]?.source)">
                  {{ sourceLabel(todayRateMap[currency.value]?.source) }}
                </el-tag>
              </div>
              <div class="mt-3 text-xl font-semibold text-slate-900 dark:text-slate-100">
                {{ formatRate(todayRateMap[currency.value]?.rateToCny) }}
              </div>
              <div class="mt-1 text-xs text-slate-500 dark:text-slate-400">
                {{ todayRateMap[currency.value]?.updatedAt || todayRateMap[currency.value]?.createdAt || '--' }}
              </div>
            </div>
          </div>
        </template>
      </el-skeleton>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="searchTable">
        <el-form-item label="币种">
          <el-select v-model="searchInfo.currencyCode" clearable class="!w-36">
            <el-option v-for="currency in currencyOptions" :key="currency.value" :label="currency.value" :value="currency.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="来源">
          <el-select v-model="searchInfo.source" clearable class="!w-44">
            <el-option v-for="source in sourceOptions" :key="source.value" :label="source.label" :value="source.value" />
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
          <el-button type="primary" :loading="loading" @click="searchTable">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table :data="tableData" stripe v-loading="loading">
        <el-table-column prop="rateDate" label="日期" width="120" />
        <el-table-column prop="currencyCode" label="币种" width="100" />
        <el-table-column label="兑人民币" width="140">
          <template #default="{ row }">{{ formatRate(row.rateToCny) }}</template>
        </el-table-column>
        <el-table-column label="来源" width="150">
          <template #default="{ row }">
            <el-tag size="small" :type="sourceTagType(row.source)">{{ sourceLabel(row.source) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="手工覆盖" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.manualOverride ? 'warning' : 'info'">{{ row.manualOverride ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="原因" min-width="260" show-overflow-tooltip />
        <el-table-column prop="createdAt" label="创建时间" min-width="170" />
        <el-table-column prop="updatedAt" label="更新时间" min-width="170" />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openOverrideDialog(row)">手工覆盖</el-button>
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

    <el-dialog v-model="overrideVisible" title="手工覆盖汇率" width="540px" destroy-on-close>
      <el-form label-width="96px">
        <el-form-item label="币种">
          <el-select v-model="overrideForm.currencyCode" class="!w-full">
            <el-option
              v-for="currency in currencyOptions"
              :key="currency.value"
              :label="`${currency.value} - ${currency.label}`"
              :value="currency.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="日期">
          <el-date-picker v-model="overrideForm.rateDate" type="date" value-format="YYYY-MM-DD" class="!w-full" />
        </el-form-item>
        <el-form-item label="兑人民币">
          <el-input-number v-model="overrideForm.rateToCny" :precision="6" :min="0" :step="0.01" class="!w-full" />
        </el-form-item>
        <el-form-item label="原因">
          <el-input v-model="overrideForm.reason" type="textarea" :rows="3" placeholder="请输入覆盖原因" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="overrideVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitOverride">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import {
  getAmazonFinanceFxList,
  refreshAmazonFinanceFxRates,
  saveAmazonFinanceFxOverride
} from '@/api/amazonFinanceFx'
import { amazonManagedCurrencyOptions } from '@/utils/amazonCurrency'

defineOptions({
  name: 'AmazonFinanceFx'
})

const currencyOptions = amazonManagedCurrencyOptions

const sourceOptions = [
  { value: 'exchange_rate_api', label: '自动汇率' },
  { value: 'manual', label: '手工覆盖' },
  { value: 'carry_forward', label: '沿用历史' }
]

const loading = ref(false)
const todayLoading = ref(false)
const refreshing = ref(false)
const saving = ref(false)
const overrideVisible = ref(false)
const tableData = ref([])
const todayRates = ref([])
const total = ref(0)
const today = todayString()

const searchInfo = reactive({
  page: 1,
  pageSize: 10,
  currencyCode: '',
  source: '',
  dateRange: []
})

const overrideForm = reactive({
  currencyCode: 'USD',
  rateDate: today,
  rateToCny: 7.1,
  reason: ''
})

const todayRateMap = computed(() => {
  return todayRates.value.reduce((acc, row) => {
    acc[row.currencyCode] = row
    return acc
  }, {})
})

const buildListPayload = () => ({
  page: searchInfo.page,
  pageSize: searchInfo.pageSize,
  currencyCode: searchInfo.currencyCode,
  source: searchInfo.source,
  dateFrom: searchInfo.dateRange?.[0] || '',
  dateTo: searchInfo.dateRange?.[1] || ''
})

const fetchTable = async () => {
  loading.value = true
  try {
    const res = await getAmazonFinanceFxList(buildListPayload())
    if (res.code === 0) {
      tableData.value = res.data.list || []
      total.value = res.data.total || 0
      searchInfo.page = res.data.page || searchInfo.page
      searchInfo.pageSize = res.data.pageSize || searchInfo.pageSize
    }
  } finally {
    loading.value = false
  }
}

const fetchTodayRates = async () => {
  todayLoading.value = true
  try {
    const res = await getAmazonFinanceFxList({
      page: 1,
      pageSize: 20,
      dateFrom: today,
      dateTo: today
    })
    if (res.code === 0) {
      todayRates.value = res.data.list || []
    }
  } finally {
    todayLoading.value = false
  }
}

const fetchAll = async () => {
  await Promise.all([fetchTable(), fetchTodayRates()])
}

const searchTable = async () => {
  searchInfo.page = 1
  await fetchTable()
}

const resetSearch = () => {
  searchInfo.page = 1
  searchInfo.pageSize = 10
  searchInfo.currencyCode = ''
  searchInfo.source = ''
  searchInfo.dateRange = []
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

const refreshDailyRates = async () => {
  refreshing.value = true
  try {
    const res = await refreshAmazonFinanceFxRates()
    if (res.code === 0) {
      const data = res.data || {}
      const message = `刷新完成：自动 ${data.successCount || 0}，沿用 ${data.carryForwardCount || 0}，跳过手工 ${data.skippedManualCount || 0}，失败 ${data.failedCount || 0}`
      if (Array.isArray(data.errors) && data.errors.length > 0) {
        ElMessage.warning(message)
        await ElMessageBox.alert(data.errors.join('\n'), '刷新提示', { type: 'warning' })
      } else {
        ElMessage.success(message)
      }
      await fetchAll()
    }
  } finally {
    refreshing.value = false
  }
}

const openOverrideDialog = (row) => {
  overrideForm.currencyCode = row?.currencyCode || searchInfo.currencyCode || 'USD'
  overrideForm.rateDate = row?.rateDate || today
  overrideForm.rateToCny = Number(row?.rateToCny || 7.1)
  overrideForm.reason = row?.manualOverride ? row.reason || '' : ''
  overrideVisible.value = true
}

const submitOverride = async () => {
  saving.value = true
  try {
    const res = await saveAmazonFinanceFxOverride({
      currencyCode: overrideForm.currencyCode,
      rateDate: overrideForm.rateDate,
      rateToCny: Number(overrideForm.rateToCny || 0),
      reason: overrideForm.reason
    })
    if (res.code === 0) {
      ElMessage.success('手工汇率已保存')
      overrideVisible.value = false
      await fetchAll()
    }
  } finally {
    saving.value = false
  }
}

const sourceLabel = (value) => {
  switch (value) {
    case 'exchange_rate_api':
      return '自动'
    case 'manual':
      return '手工'
    case 'carry_forward':
      return '沿用'
    default:
      return value || '未生成'
  }
}

const sourceTagType = (value) => {
  switch (value) {
    case 'exchange_rate_api':
      return 'success'
    case 'manual':
      return 'warning'
    case 'carry_forward':
      return 'info'
    default:
      return ''
  }
}

const formatRate = (value) => {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric <= 0) return '--'
  return numeric.toFixed(6)
}

function todayString() {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

onMounted(fetchAll)
</script>
