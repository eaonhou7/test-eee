<template>
  <div class="flex flex-col gap-6">
    <section class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <el-button link type="primary" class="!px-0" @click="goBack">返回退货列表</el-button>
          <p class="text-xs tracking-[0.3em] text-slate-500">RETURN DETAIL</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">
            {{ detail?.amazonRmaId || `退货 #${returnID}` }}
          </h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            查看退货原单关联、赠送阈值判断、转寄候选、回仓处置单和服务商执行状态。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <el-button @click="fetchDetail">刷新</el-button>
          <el-button type="primary" plain :disabled="!detail" @click="openSupportInbox">新建客服工单</el-button>
          <el-button type="warning" :loading="actionLoading === 'sync'" @click="handleResync">重跑同步</el-button>
        </div>
      </div>
    </section>

    <el-skeleton :loading="loading" animated :rows="10">
      <template #default>
        <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
          <div class="mb-4 flex flex-wrap items-center gap-2">
            <el-tag size="small" :type="linkTagType(detail?.linkStatus)">{{ linkLabel(detail?.linkStatus) }}</el-tag>
            <el-tag size="small" type="info">{{ detail?.returnRequestStatus || '--' }}</el-tag>
          </div>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="Amazon订单">{{ detail?.amazonOrderId || '--' }}</el-descriptions-item>
            <el-descriptions-item label="站点">{{ detail?.siteCode || '--' }}</el-descriptions-item>
            <el-descriptions-item label="退货类型">{{ detail?.returnType || '--' }}</el-descriptions-item>
            <el-descriptions-item label="Resolution">{{ detail?.resolution || '--' }}</el-descriptions-item>
            <el-descriptions-item label="退款金额">{{ formatPrice(detail?.refundAmount, detail?.refundCurrency) }}</el-descriptions-item>
            <el-descriptions-item label="面单费用">{{ formatPrice(detail?.labelCost, detail?.labelCurrency) }}</el-descriptions-item>
            <el-descriptions-item label="承运商">{{ detail?.carrier || '--' }}</el-descriptions-item>
            <el-descriptions-item label="退货运单">{{ detail?.trackingId || '--' }}</el-descriptions-item>
            <el-descriptions-item label="申请时间">{{ detail?.returnRequestDate || '--' }}</el-descriptions-item>
            <el-descriptions-item label="送达时间">{{ detail?.returnDeliveryDate || '--' }}</el-descriptions-item>
            <el-descriptions-item label="异常信息" :span="2">{{ detail?.exceptionMessage || '--' }}</el-descriptions-item>
          </el-descriptions>
        </section>

        <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
          <div class="mb-4 flex flex-col gap-2 xl:flex-row xl:items-center xl:justify-between">
            <div>
              <div class="text-base font-semibold text-slate-900 dark:text-slate-100">财务影响</div>
              <div class="text-sm text-slate-500 dark:text-slate-400">退款、面单费、处置费、货损和回收货值会同步影响订单真实利润。</div>
            </div>
            <el-tag size="small" :type="financeImpactTagType(detail?.financeImpact?.netImpactCny)">
              净影响 {{ formatPrice(detail?.financeImpact?.netImpactCny, 'CNY') }}
            </el-tag>
          </div>
          <div v-if="detail?.financeImpact" class="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
            <div class="rounded-xl border border-slate-200 p-4 dark:border-slate-700">
              <div class="text-xs text-slate-500 dark:text-slate-400">退款</div>
              <div class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">{{ formatPrice(detail.financeImpact.refundCny, 'CNY') }}</div>
            </div>
            <div class="rounded-xl border border-slate-200 p-4 dark:border-slate-700">
              <div class="text-xs text-slate-500 dark:text-slate-400">面单费</div>
              <div class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">{{ formatPrice(detail.financeImpact.labelFeeCny, 'CNY') }}</div>
            </div>
            <div class="rounded-xl border border-slate-200 p-4 dark:border-slate-700">
              <div class="text-xs text-slate-500 dark:text-slate-400">处置费</div>
              <div class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">{{ formatPrice(detail.financeImpact.dispositionFeeCny, 'CNY') }}</div>
            </div>
            <div class="rounded-xl border border-slate-200 p-4 dark:border-slate-700">
              <div class="text-xs text-slate-500 dark:text-slate-400">货损</div>
              <div class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">{{ formatPrice(detail.financeImpact.goodsLossCny, 'CNY') }}</div>
            </div>
            <div class="rounded-xl border border-slate-200 p-4 dark:border-slate-700">
              <div class="text-xs text-slate-500 dark:text-slate-400">可回收货值</div>
              <div class="mt-1 text-lg font-semibold text-emerald-600">{{ formatPrice(detail.financeImpact.recoveryCny, 'CNY') }}</div>
            </div>
          </div>
          <el-empty v-else description="暂无财务影响数据" :image-size="80" />
        </section>

        <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
          <div class="mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">退货项决策</div>
          <div class="space-y-4">
            <div
              v-for="item in detail?.items || []"
              :key="item.id"
              class="rounded-2xl border border-slate-200 p-4 dark:border-slate-700"
            >
              <div class="mb-3 flex flex-wrap items-center gap-2">
                <div class="font-medium text-slate-900 dark:text-slate-100">{{ item.sellerSku || '--' }}</div>
                <el-tag size="small" :type="decisionTagType(item.recommendedDecision)">
                  {{ decisionLabel(item.recommendedDecision) }}
                </el-tag>
                <el-tag size="small" :type="decisionStatusTagType(item.decisionStatus)">
                  {{ decisionStatusLabel(item.decisionStatus) }}
                </el-tag>
              </div>
              <div class="grid gap-3 text-sm text-slate-600 dark:text-slate-300 md:grid-cols-2 xl:grid-cols-4">
                <div>数量：{{ item.returnQuantity || 0 }}</div>
                <div>货值：{{ formatPrice(item.goodsValueCny, 'CNY') }}</div>
                <div>退货成本：{{ formatPrice(item.intakeFeeCny, 'CNY') }}</div>
                <div>近30天销量：{{ item.soldQtyLast30d || 0 }}</div>
                <div>原订单项：{{ item.originalOrderItemId || '--' }}</div>
                <div>目标订单：{{ item.targetOrderId || '--' }}</div>
                <div>目标仓：{{ item.targetWarehouseId || '--' }}</div>
                <div>关联置信度：{{ formatConfidence(item.linkConfidence) }}</div>
              </div>
              <div class="mt-3 text-sm text-slate-500 dark:text-slate-400">
                {{ item.decisionReason || item.exceptionMessage || '--' }}
              </div>

              <div v-if="item.redirectCandidate" class="mt-4 rounded-xl border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800/50 dark:bg-amber-900/20 dark:text-amber-200">
                候选转寄订单：{{ item.redirectCandidate.amazonOrderId || item.redirectCandidate.targetOrderId }}，数量 {{ item.redirectCandidate.quantity }}，
                近30天销量 {{ item.redirectCandidate.soldQtyLast30d }}。
              </div>

              <div v-if="item.disposition" class="mt-4 rounded-xl border border-slate-200 bg-slate-50 p-3 text-sm dark:border-slate-700 dark:bg-slate-900/40">
                <div class="font-medium text-slate-900 dark:text-slate-100">处置单</div>
                <div class="mt-2 grid gap-2 md:grid-cols-2">
                  <div>目标：{{ decisionLabel(item.disposition.targetType) }}</div>
                  <div>服务商：{{ item.disposition.providerName || '--' }}</div>
                  <div>服务商单号：{{ item.disposition.providerOrderNo || '--' }}</div>
                  <div>运单号：{{ item.disposition.providerTrackingNo || '--' }}</div>
                  <div>状态：{{ decisionStatusLabel(item.disposition.status) }}</div>
                  <div>总费用：{{ formatPrice(item.disposition.totalFeeCny, 'CNY') }}</div>
                </div>
                <div v-if="item.disposition.labelUrl" class="mt-2">
                  <el-button type="primary" link @click="openExternal(item.disposition.labelUrl)">打开面单</el-button>
                </div>
              </div>

              <div class="mt-4 flex flex-wrap gap-2">
                <el-button type="primary" plain @click="handleRecompute(item)">重算决策</el-button>
                <el-button
                  v-if="item.redirectCandidate"
                  type="warning"
                  :loading="actionLoading === `redirect-${item.id}`"
                  @click="handleConfirmRedirect(item)"
                >
                  确认转寄
                </el-button>
                <el-button
                  v-if="item.recommendedDecision === 'warehouse' || item.decisionStatus === 'exception'"
                  type="success"
                  :loading="actionLoading === `warehouse-${item.id}`"
                  @click="handleConfirmWarehouse(item)"
                >
                  确认回仓
                </el-button>
                <el-button
                  v-if="item.targetOrderId"
                  type="danger"
                  plain
                  :loading="actionLoading === `release-${item.id}`"
                  @click="handleReleaseRedirect(item)"
                >
                  释放转寄
                </el-button>
              </div>
            </div>
          </div>
        </section>
      </template>
    </el-skeleton>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import {
  confirmAmazonReturnRedirect,
  confirmAmazonReturnWarehouse,
  findAmazonReturn,
  recomputeAmazonReturnDecision,
  releaseAmazonReturnRedirect,
  resyncAmazonReturns
} from '@/api/amazonReturn'

defineOptions({
  name: 'AmazonReturnDetail'
})

const route = useRoute()
const router = useRouter()

const returnID = computed(() => Number(route.params.id || route.query.id || 0))
const detail = ref(null)
const loading = ref(false)
const actionLoading = ref('')

const fetchDetail = async () => {
  if (!returnID.value) return
  loading.value = true
  try {
    const res = await findAmazonReturn({ id: returnID.value })
    if (res.code === 0) {
      detail.value = res.data
    }
  } finally {
    loading.value = false
  }
}

const goBack = () => {
  router.push({ name: 'amazonReturnManager' })
}

const openSupportInbox = () => {
  if (!detail.value) {
    return
  }
  router.push({
    name: 'amazonSupportInbox',
    query: {
      compose: '1',
      caseType: 'return',
      storeId: String(detail.value.storeId || ''),
      siteCode: detail.value.siteCode || '',
      orderId: detail.value.orderId ? String(detail.value.orderId) : '',
      returnOrderId: String(detail.value.id || ''),
      amazonOrderId: detail.value.amazonOrderId || '',
      amazonRmaId: detail.value.amazonRmaId || ''
    }
  })
}

const handleResync = async () => {
  actionLoading.value = 'sync'
  try {
    const res = await resyncAmazonReturns({ storeId: detail.value?.storeId })
    if (res.code === 0) {
      ElMessage.success('已触发退货同步')
      await fetchDetail()
    }
  } finally {
    actionLoading.value = ''
  }
}

const handleRecompute = async (item) => {
  actionLoading.value = `recompute-${item.id}`
  try {
    const res = await recomputeAmazonReturnDecision({ returnItemId: item.id })
    if (res.code === 0) {
      detail.value = res.data
      ElMessage.success('重算完成')
    }
  } finally {
    actionLoading.value = ''
  }
}

const handleConfirmRedirect = async (item) => {
  actionLoading.value = `redirect-${item.id}`
  try {
    const res = await confirmAmazonReturnRedirect({
      returnItemId: item.id,
      targetOrderItemId: item.redirectCandidate?.targetOrderItemId
    })
    if (res.code === 0) {
      detail.value = res.data
      ElMessage.success('已创建转寄处置单')
    }
  } finally {
    actionLoading.value = ''
  }
}

const handleConfirmWarehouse = async (item) => {
  actionLoading.value = `warehouse-${item.id}`
  try {
    const res = await confirmAmazonReturnWarehouse({
      returnItemId: item.id,
      warehouseId: item.targetWarehouseId
    })
    if (res.code === 0) {
      detail.value = res.data
      ElMessage.success('已创建回仓处置单')
    }
  } finally {
    actionLoading.value = ''
  }
}

const handleReleaseRedirect = async (item) => {
  actionLoading.value = `release-${item.id}`
  try {
    const res = await releaseAmazonReturnRedirect({ returnItemId: item.id })
    if (res.code === 0) {
      detail.value = res.data
      ElMessage.success('已释放转寄')
    }
  } finally {
    actionLoading.value = ''
  }
}

const openExternal = (url) => {
  window.open(url, '_blank', 'noopener,noreferrer')
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

const decisionLabel = (value) => {
  switch (value) {
    case 'gift':
      return '直接赠送'
    case 'warehouse':
      return '退回仓库'
    case 'new_buyer':
      return '转寄新买家'
    case 'manual_review':
      return '人工复核'
    default:
      return value || '--'
  }
}

const decisionTagType = (value) => {
  switch (value) {
    case 'gift':
      return 'info'
    case 'warehouse':
      return 'success'
    case 'new_buyer':
      return 'warning'
    case 'manual_review':
      return 'danger'
    default:
      return 'info'
  }
}

const decisionStatusLabel = (value) => {
  switch (value) {
    case 'pending':
      return '待处理'
    case 'recommended':
      return '已建议'
    case 'confirmed':
      return '已确认'
    case 'closed':
      return '已关闭'
    case 'exception':
      return '异常'
    case 'created':
      return '已创建'
    case 'completed':
      return '已完成'
    case 'released':
      return '已释放'
    default:
      return value || '--'
  }
}

const decisionStatusTagType = (value) => {
  switch (value) {
    case 'recommended':
      return 'warning'
    case 'confirmed':
    case 'created':
      return 'primary'
    case 'closed':
    case 'completed':
      return 'success'
    case 'exception':
      return 'danger'
    default:
      return 'info'
  }
}

const formatPrice = (value, currencyCode) => {
  if (value === null || typeof value === 'undefined') return '--'
  return `${currencyCode || ''} ${Number(value).toFixed(2)}`.trim()
}

const financeImpactTagType = (value) => {
  if (Number(value || 0) > 0) return 'danger'
  if (Number(value || 0) < 0) return 'success'
  return 'info'
}

const formatConfidence = (value) => {
  if (value === null || typeof value === 'undefined') return '--'
  return `${(Number(value) * 100).toFixed(0)}%`
}

onMounted(() => {
  fetchDetail()
})
</script>
