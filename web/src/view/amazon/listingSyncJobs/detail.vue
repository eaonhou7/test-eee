<template>
  <div class="flex flex-col gap-6">
    <section class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <el-button link type="primary" class="!px-0" @click="goBack">返回回传任务列表</el-button>
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON LISTING SYNC DETAIL</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">
            价格库存回传任务 #{{ jobID || '--' }}
          </h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            查看 Feed 提交状态、Amazon 返回结果，以及每个站点 SKU 的回传明细。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <el-button @click="fetchDetail">刷新</el-button>
          <el-button type="primary" :loading="refreshing" @click="handleRefreshStatus">刷新 Amazon 状态</el-button>
        </div>
      </div>
    </section>

    <el-skeleton :loading="loading" animated :rows="10">
      <template #default>
        <section class="grid gap-4 xl:grid-cols-[2fr_1fr]">
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="mb-4 flex flex-wrap gap-2">
              <el-tag size="small" :type="statusTagType(detail?.processingStatus)">
                {{ statusLabel(detail?.processingStatus) }}
              </el-tag>
              <el-tag size="small" :type="statusTagType(detail?.submitStatus)">
                {{ statusLabel(detail?.submitStatus) }}
              </el-tag>
            </div>
            <el-descriptions :column="2" border>
              <el-descriptions-item label="店铺">{{ detail?.storeId || '--' }}</el-descriptions-item>
              <el-descriptions-item label="Feed 类型">{{ detail?.feedType || '--' }}</el-descriptions-item>
              <el-descriptions-item label="Feed ID">{{ detail?.feedId || '--' }}</el-descriptions-item>
              <el-descriptions-item label="结果文档">{{ detail?.resultDocumentId || '--' }}</el-descriptions-item>
              <el-descriptions-item label="字段范围" :span="2">
                <div class="flex flex-wrap gap-2">
                  <el-tag v-for="scope in detail?.fieldScopes || []" :key="scope" size="small" type="info">
                    {{ fieldScopeLabel(scope) }}
                  </el-tag>
                </div>
              </el-descriptions-item>
              <el-descriptions-item label="提交时间">{{ detail?.submittedAt || '--' }}</el-descriptions-item>
              <el-descriptions-item label="完成时间">{{ detail?.finishedAt || '--' }}</el-descriptions-item>
              <el-descriptions-item label="结果摘要" :span="2">{{ detail?.issueSummary || '--' }}</el-descriptions-item>
              <el-descriptions-item label="错误信息" :span="2">{{ detail?.errorMessage || '--' }}</el-descriptions-item>
            </el-descriptions>
          </div>

          <div class="rounded-2xl border border-slate-200 bg-slate-50 p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/40">
            <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">任务摘要</div>
            <div class="space-y-3 text-sm text-slate-600 dark:text-slate-300">
              <div class="rounded-xl border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/60">
                <div class="mb-1 font-medium">记录数</div>
                <div>{{ detail?.records?.length || 0 }}</div>
              </div>
              <div class="rounded-xl border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/60">
                <div class="mb-1 font-medium">成功 / 失败 / 跳过 / 待处理</div>
                <div>{{ summarizeRecords(detail?.records || []) }}</div>
              </div>
            </div>
          </div>
        </section>

        <section class="grid gap-4 xl:grid-cols-2">
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">Feed Payload</div>
            <pre class="max-h-[420px] overflow-auto rounded-xl bg-slate-950/95 p-4 text-xs text-slate-100">{{ payloadText }}</pre>
          </div>
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">Amazon Response</div>
            <pre class="max-h-[420px] overflow-auto rounded-xl bg-slate-950/95 p-4 text-xs text-slate-100">{{ responseText }}</pre>
          </div>
        </section>

        <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
          <div class="mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">SKU 回传明细</div>
          <el-table :data="detail?.records || []" row-key="id" border>
            <el-table-column prop="sku" label="SKU" min-width="180" />
            <el-table-column prop="siteCode" label="站点" width="100" />
            <el-table-column prop="marketplaceId" label="Marketplace ID" min-width="180" />
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-tag size="small" :type="statusTagType(row.syncStatus)">{{ statusLabel(row.syncStatus) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="回传值" min-width="280">
              <template #default="{ row }">
                <div class="flex flex-col gap-1 text-sm">
                  <span>价格：{{ formatMoney(row.pushedOfferPrice) }}</span>
                  <span>库存：{{ row.pushedQuantity ?? '--' }}</span>
                  <span>备货天数：{{ row.pushedLeadTimeToShip ?? '--' }}</span>
                  <span>配送模板：{{ row.pushedMerchantShippingGroup || '--' }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="问题" min-width="320">
              <template #default="{ row }">
                <div class="flex flex-col gap-1 text-sm text-slate-600 dark:text-slate-300">
                  <span v-if="row.errorMessage">{{ row.errorMessage }}</span>
                  <span v-for="(issue, index) in row.issues || []" :key="index">
                    {{ issue.message || issue.description || JSON.stringify(issue) }}
                  </span>
                  <span v-if="!row.errorMessage && !(row.issues || []).length">--</span>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </section>
      </template>
    </el-skeleton>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'

import { findAmazonListingSync, refreshAmazonListingSyncStatus } from '@/api/amazonListingSync'

defineOptions({
  name: 'AmazonListingSyncJobDetail'
})

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const refreshing = ref(false)
const detail = ref(null)

const jobID = computed(() => Number(route.params.id || route.query.id || 0))
const payloadText = computed(() => JSON.stringify(detail.value?.payload || {}, null, 2))
const responseText = computed(() => JSON.stringify(detail.value?.response || {}, null, 2))

const fetchDetail = async () => {
  if (!jobID.value) {
    return
  }
  loading.value = true
  try {
    const res = await findAmazonListingSync({ id: jobID.value })
    if (res.code === 0) {
      detail.value = res.data || null
    }
  } finally {
    loading.value = false
  }
}

const goBack = () => {
  router.push({ name: 'amazonListingSyncJobManager' })
}

const handleRefreshStatus = async () => {
  refreshing.value = true
  try {
    const res = await refreshAmazonListingSyncStatus({ id: jobID.value })
    if (res.code === 0) {
      detail.value = res.data || null
      ElMessage.success('Amazon 状态已刷新')
    }
  } finally {
    refreshing.value = false
  }
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
    case 'pending':
      return '待处理'
    case 'skipped':
      return '已跳过'
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
    case 'pending':
      return 'warning'
    case 'failed':
      return 'danger'
    case 'skipped':
      return 'info'
    default:
      return 'info'
  }
}

const summarizeRecords = (records) => {
  const summary = {
    completed: 0,
    failed: 0,
    skipped: 0,
    pending: 0
  }
  records.forEach((record) => {
    if (record.syncStatus === 'completed') {
      summary.completed += 1
    } else if (record.syncStatus === 'skipped') {
      summary.skipped += 1
    } else if (record.syncStatus === 'pending' || record.syncStatus === 'processing') {
      summary.pending += 1
    } else {
      summary.failed += 1
    }
  })
  return `${summary.completed} / ${summary.failed} / ${summary.skipped} / ${summary.pending}`
}

const formatMoney = (value) => {
  if (value === null || typeof value === 'undefined') {
    return '--'
  }
  return Number(value).toFixed(2)
}

watch(jobID, () => {
  fetchDetail()
}, { immediate: true })
</script>
