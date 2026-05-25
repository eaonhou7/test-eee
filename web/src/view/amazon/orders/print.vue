<template>
  <div class="mx-auto max-w-5xl bg-white px-6 py-8 text-slate-900">
    <div class="mb-6 flex items-center justify-between print:hidden">
      <div>
        <div class="text-xs tracking-[0.3em] text-slate-500">SYSTEM SHIPPING SLIP</div>
        <div class="text-2xl font-semibold">{{ detail?.amazonOrderId || `订单 #${orderID}` }}</div>
      </div>
      <div class="flex gap-3">
        <el-button @click="window.history.back()">返回</el-button>
        <el-button type="primary" @click="printPage">打印</el-button>
      </div>
    </div>

    <div v-if="detail" class="space-y-6">
      <section class="rounded-2xl border border-slate-300 p-5">
        <div class="mb-4 flex items-start justify-between gap-4">
          <div>
            <div class="text-sm text-slate-500">Amazon Order</div>
            <div class="text-xl font-semibold">{{ detail.amazonOrderId || '--' }}</div>
          </div>
          <div class="text-right text-sm">
            <div>{{ detail.siteCode || '--' }}</div>
            <div>{{ detail.purchaseDate || '--' }}</div>
          </div>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <div class="rounded-xl border border-slate-200 p-4">
            <div class="mb-2 text-sm font-semibold text-slate-700">收货信息</div>
            <div class="space-y-1 text-sm leading-6">
              <div>{{ detail.address?.recipientName || '--' }}</div>
              <div>{{ detail.address?.phone || '--' }}</div>
              <div>{{ formatAddress(detail.address) }}</div>
            </div>
          </div>
          <div class="rounded-xl border border-slate-200 p-4">
            <div class="mb-2 text-sm font-semibold text-slate-700">履约摘要</div>
            <div class="space-y-1 text-sm leading-6">
              <div>履约类型：{{ formatFulfillmentType(detail.fulfillmentType) }}</div>
              <div>工作流：{{ workflowLabel(detail.workflowStatus) }}</div>
              <div>采购状态：{{ statusLabel(detail.procurementStatus) }}</div>
              <div>物流状态：{{ statusLabel(detail.logisticsStatus) }}</div>
            </div>
          </div>
        </div>
      </section>

      <section class="rounded-2xl border border-slate-300 p-5">
        <div class="mb-3 text-lg font-semibold">订单项</div>
        <table class="w-full border-collapse text-sm">
          <thead>
            <tr class="bg-slate-100 text-left">
              <th class="border border-slate-300 px-3 py-2">SKU</th>
              <th class="border border-slate-300 px-3 py-2">标题</th>
              <th class="border border-slate-300 px-3 py-2">数量</th>
              <th class="border border-slate-300 px-3 py-2">1688采购单</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in detail.items || []" :key="item.id">
              <td class="border border-slate-300 px-3 py-2 align-top">{{ item.sellerSku || '--' }}</td>
              <td class="border border-slate-300 px-3 py-2 align-top">{{ item.title || '--' }}</td>
              <td class="border border-slate-300 px-3 py-2 align-top">{{ item.quantityOrdered || 0 }}</td>
              <td class="border border-slate-300 px-3 py-2 align-top">{{ item.purchaseOrderNo || '--' }}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <section class="grid gap-4 md:grid-cols-2">
        <div class="rounded-2xl border border-slate-300 p-5">
          <div class="mb-3 text-base font-semibold">采购组</div>
          <div v-if="detail.procurementGroups?.length" class="space-y-3 text-sm">
            <div v-for="group in detail.procurementGroups" :key="group.id" class="rounded-xl border border-slate-200 p-3">
              <div>{{ group.shopName || group.shopGroupKey }}</div>
              <div class="text-slate-500">1688 单号：{{ group.orderNo1688 || '--' }}</div>
            </div>
          </div>
          <div v-else class="text-sm text-slate-500">暂无采购组</div>
        </div>

        <div class="rounded-2xl border border-slate-300 p-5">
          <div class="mb-3 text-base font-semibold">物流单</div>
          <div v-if="detail.shipments?.length" class="space-y-3 text-sm">
            <div v-for="shipment in detail.shipments" :key="shipment.id" class="rounded-xl border border-slate-200 p-3">
              <div>{{ shipment.provider || '--' }} / {{ shipment.channelName || '--' }}</div>
              <div class="text-slate-500">运单号：{{ shipment.trackingNo || '--' }}</div>
              <div class="text-slate-500">揽收：{{ shipment.actualPickupAt || shipment.reservedPickupAt || '--' }}</div>
            </div>
          </div>
          <div v-else class="text-sm text-slate-500">暂无物流单</div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'

import { findAmazonOrder } from '@/api/amazonOrder'

defineOptions({
  name: 'AmazonOrderShipmentPrint'
})

const route = useRoute()
const detail = ref(null)
const orderID = ref(0)

const loadDetail = async () => {
  orderID.value = Number(route.params.id || route.query.id || 0)
  if (!orderID.value) {
    return
  }
  const res = await findAmazonOrder({ id: orderID.value })
  if (res.code === 0) {
    detail.value = res.data || null
    if (route.query.token) {
      await nextTick()
      window.setTimeout(() => {
        printPage()
      }, 120)
    }
  }
}

const printPage = () => {
  window.print()
}

const formatAddress = (address) => {
  if (!address) {
    return '--'
  }
  return [
    address.addressLine1,
    address.addressLine2,
    address.addressLine3,
    address.city,
    address.stateOrRegion,
    address.postalCode,
    address.countryCode
  ].filter(Boolean).join(' / ') || '--'
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
    case 'fulfillment_running':
      return '执行中'
    case 'fulfillment_completed':
      return '已完成'
    case 'fulfillment_failed':
      return '执行失败'
    case 'fba_closed':
      return 'FBA关闭'
    default:
      return value || '--'
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
    case 'created':
      return '已下单'
    case 'picked_up':
      return '已揽收'
    case 'submitted':
      return '已回传'
    default:
      return value || '--'
  }
}

watch(() => route.fullPath, () => {
  loadDetail()
}, { immediate: true })
</script>

<style scoped>
@media print {
  body {
    background: #fff;
  }
}
</style>
