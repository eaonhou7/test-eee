<template>
  <div class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
    <div class="mb-4 flex flex-col gap-2 xl:flex-row xl:items-center xl:justify-between">
      <div>
        <div class="text-sm font-semibold text-slate-900 dark:text-slate-100">利润试算</div>
        <div class="text-xs text-slate-500 dark:text-slate-400">当前站点售价自动参与计算，固定成本统一按人民币 CNY 录入。</div>
      </div>
      <div class="flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
        <span>{{ siteCode ? `${siteCode} 站` : '当前站点' }}</span>
        <span>售价：{{ formatMoney(offerPrice, currencyCode || '--') }}</span>
        <span v-if="calculating">正在试算...</span>
      </div>
    </div>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <div>
        <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">履约模式</div>
        <el-select v-model="localProfile.fulfillmentMode" class="!w-full" clearable placeholder="请选择 FBA 或 FBM">
          <el-option label="FBA（亚马逊配送）" value="fba" />
          <el-option label="FBM（卖家自发货）" value="fbm" />
        </el-select>
      </div>
      <div>
        <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">汇率</div>
        <el-input-number
          v-model="localProfile.exchangeRateToCny"
          :min="0"
          :precision="4"
          :step="0.01"
          class="!w-full"
          placeholder="1 单位站点币种 = ? CNY"
        />
      </div>
      <div>
        <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">平台佣金率（%）</div>
        <el-input-number v-model="localProfile.referralFeeRate" :min="0" :max="99.99" :precision="2" :step="0.1" class="!w-full" />
      </div>
      <div>
        <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">广告占比（%）</div>
        <el-input-number v-model="localProfile.adCostRate" :min="0" :max="99.99" :precision="2" :step="0.1" class="!w-full" />
      </div>
      <div>
        <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">采购成本（CNY）</div>
        <el-input-number v-model="localProfile.procurementCostCny" :min="0" :precision="2" :step="1" class="!w-full" />
      </div>
      <div>
        <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">头程成本（CNY）</div>
        <el-input-number v-model="localProfile.firstLegCostCny" :min="0" :precision="2" :step="1" class="!w-full" />
      </div>
      <div v-if="localProfile.fulfillmentMode === 'fba'">
        <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">FBA 配送费（CNY）</div>
        <el-input-number v-model="localProfile.fbaFulfillmentFeeCny" :min="0" :precision="2" :step="1" class="!w-full" />
      </div>
      <div v-if="localProfile.fulfillmentMode === 'fbm'">
        <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">尾程派送费（CNY）</div>
        <el-input-number v-model="localProfile.fbmLastMileCostCny" :min="0" :precision="2" :step="1" class="!w-full" />
      </div>
      <div>
        <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">其他成本（CNY）</div>
        <el-input-number v-model="localProfile.otherCostCny" :min="0" :precision="2" :step="1" class="!w-full" />
      </div>
    </div>

    <el-alert
      v-if="alertMessage"
      :title="alertMessage"
      :type="alertType"
      :closable="false"
      show-icon
      class="mt-4"
    />

    <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-5">
      <div v-for="card in metricCards" :key="card.label" class="rounded-lg border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/70">
        <div class="text-xs text-slate-500 dark:text-slate-400">{{ card.label }}</div>
        <div class="mt-2 text-lg font-semibold" :class="card.className">{{ card.value }}</div>
      </div>
    </div>

    <div class="mt-4 rounded-lg border border-dashed border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900/70">
      <div class="mb-3 text-sm font-medium text-slate-800 dark:text-slate-100">成本拆解</div>
      <div class="grid gap-3 text-sm md:grid-cols-2 xl:grid-cols-4">
        <div v-for="entry in breakdownItems" :key="entry.label" class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-800/60">
          <div class="text-xs text-slate-500 dark:text-slate-400">{{ entry.label }}</div>
          <div class="mt-1 font-medium text-slate-800 dark:text-slate-100">{{ entry.value }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
  import { computed, nextTick, ref, watch } from 'vue'
  import { useDebounceFn } from '@vueuse/core'

  import { calculateAmazonListingProfit } from '@/api/amazonListingProfit'

  const props = defineProps({
    modelValue: {
      type: Object,
      default: () => ({})
    },
    siteCode: {
      type: String,
      default: ''
    },
    currencyCode: {
      type: String,
      default: ''
    },
    offerPrice: {
      type: Number,
      default: undefined
    }
  })

  const emit = defineEmits(['update:modelValue'])

  const createEmptyProfitProfile = () => ({
    id: 0,
    fulfillmentMode: '',
    costCurrencyCode: 'CNY',
    exchangeRateToCny: undefined,
    referralFeeRate: 15,
    adCostRate: 0,
    procurementCostCny: undefined,
    firstLegCostCny: undefined,
    fbaFulfillmentFeeCny: undefined,
    fbmLastMileCostCny: undefined,
    otherCostCny: undefined,
    validationStatus: 'unconfigured',
    validationMessage: '请选择履约模式后再试算',
    result: null
  })

  const toOptionalNumber = (value) => {
    if (value === '' || value === null || typeof value === 'undefined') {
      return undefined
    }
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : undefined
  }

  const cloneCostBreakdown = (value = {}) => ({
    procurementCostCny: toOptionalNumber(value.procurementCostCny),
    firstLegCostCny: toOptionalNumber(value.firstLegCostCny),
    fbaFulfillmentFeeCny: toOptionalNumber(value.fbaFulfillmentFeeCny),
    fbmLastMileCostCny: toOptionalNumber(value.fbmLastMileCostCny),
    otherCostCny: toOptionalNumber(value.otherCostCny),
    commissionCny: toOptionalNumber(value.commissionCny),
    adCostCny: toOptionalNumber(value.adCostCny),
    fixedCostCny: toOptionalNumber(value.fixedCostCny)
  })

  const cloneResult = (value) => {
    if (!value || typeof value !== 'object') {
      return null
    }
    return {
      revenuePrice: toOptionalNumber(value.revenuePrice),
      revenueCurrencyCode: String(value.revenueCurrencyCode || '').trim(),
      saleCny: toOptionalNumber(value.saleCny),
      commissionCny: toOptionalNumber(value.commissionCny),
      adCostCny: toOptionalNumber(value.adCostCny),
      fixedCostCny: toOptionalNumber(value.fixedCostCny),
      grossProfitCny: toOptionalNumber(value.grossProfitCny),
      netProfitCny: toOptionalNumber(value.netProfitCny),
      netMarginRate: toOptionalNumber(value.netMarginRate),
      roiRate: toOptionalNumber(value.roiRate),
      breakEvenPrice: toOptionalNumber(value.breakEvenPrice),
      breakEvenCurrencyCode: String(value.breakEvenCurrencyCode || '').trim(),
      costBreakdown: cloneCostBreakdown(value.costBreakdown)
    }
  }

  const normalizeProfile = (value) => {
    const base = createEmptyProfitProfile()
    const current = value && typeof value === 'object' ? value : {}
    return {
      ...base,
      id: Number(current.id || 0),
      fulfillmentMode: String(current.fulfillmentMode || '').trim().toLowerCase(),
      costCurrencyCode: 'CNY',
      exchangeRateToCny: toOptionalNumber(current.exchangeRateToCny),
      referralFeeRate: toOptionalNumber(current.referralFeeRate) ?? 15,
      adCostRate: toOptionalNumber(current.adCostRate) ?? 0,
      procurementCostCny: toOptionalNumber(current.procurementCostCny),
      firstLegCostCny: toOptionalNumber(current.firstLegCostCny),
      fbaFulfillmentFeeCny: toOptionalNumber(current.fbaFulfillmentFeeCny),
      fbmLastMileCostCny: toOptionalNumber(current.fbmLastMileCostCny),
      otherCostCny: toOptionalNumber(current.otherCostCny),
      validationStatus: String(current.validationStatus || base.validationStatus),
      validationMessage: String(current.validationMessage || base.validationMessage),
      result: cloneResult(current.result)
    }
  }

  const localProfile = ref(createEmptyProfitProfile())
  const calculating = ref(false)
  let suppressWatch = false
  let suppressReleaseToken = 0
  let requestToken = 0

  const getProfileSignature = (value) => JSON.stringify(normalizeProfile(value))

  const releaseSuppressWatch = () => {
    const token = ++suppressReleaseToken
    nextTick(() => {
      if (token === suppressReleaseToken) {
        suppressWatch = false
      }
    })
  }

  const applyLocalProfile = (value, shouldEmit = true) => {
    const normalized = normalizeProfile(value)
    if (getProfileSignature(normalized) !== getProfileSignature(localProfile.value)) {
      suppressWatch = true
      localProfile.value = normalized
      releaseSuppressWatch()
    }
    if (shouldEmit) {
      emit('update:modelValue', normalizeProfile(localProfile.value))
    }
  }

  watch(
    () => props.modelValue,
    (value) => {
      applyLocalProfile(value, false)
    },
    { deep: true, immediate: true }
  )

  const resetToUnconfigured = (message = '请选择履约模式后再试算') => {
    applyLocalProfile({
      ...localProfile.value,
      validationStatus: 'unconfigured',
      validationMessage: message,
      result: null
    })
  }

  const calculateProfit = async () => {
    if (!localProfile.value.fulfillmentMode) {
      resetToUnconfigured()
      return
    }

    const currentToken = ++requestToken
    calculating.value = true
    try {
      const res = await calculateAmazonListingProfit({
        siteCode: props.siteCode,
        currencyCode: props.currencyCode,
        offerPrice: toOptionalNumber(props.offerPrice),
        profitProfile: {
          id: localProfile.value.id || 0,
          fulfillmentMode: localProfile.value.fulfillmentMode,
          costCurrencyCode: 'CNY',
          exchangeRateToCny: toOptionalNumber(localProfile.value.exchangeRateToCny),
          referralFeeRate: toOptionalNumber(localProfile.value.referralFeeRate),
          adCostRate: toOptionalNumber(localProfile.value.adCostRate),
          procurementCostCny: toOptionalNumber(localProfile.value.procurementCostCny),
          firstLegCostCny: toOptionalNumber(localProfile.value.firstLegCostCny),
          fbaFulfillmentFeeCny: toOptionalNumber(localProfile.value.fbaFulfillmentFeeCny),
          fbmLastMileCostCny: toOptionalNumber(localProfile.value.fbmLastMileCostCny),
          otherCostCny: toOptionalNumber(localProfile.value.otherCostCny)
        }
      })
      if (currentToken !== requestToken) {
        return
      }
      applyLocalProfile(res.data || localProfile.value)
    } catch (error) {
      if (currentToken !== requestToken) {
        return
      }
      applyLocalProfile({
        ...localProfile.value,
        validationStatus: 'invalid',
        validationMessage: error?.response?.data?.msg || error?.message || '利润试算失败，请稍后重试',
        result: null
      })
    } finally {
      if (currentToken === requestToken) {
        calculating.value = false
      }
    }
  }

  const debouncedCalculate = useDebounceFn(calculateProfit, 350)

  watch(
    () => [
      props.siteCode,
      props.currencyCode,
      toOptionalNumber(props.offerPrice),
      localProfile.value.fulfillmentMode,
      toOptionalNumber(localProfile.value.exchangeRateToCny),
      toOptionalNumber(localProfile.value.referralFeeRate),
      toOptionalNumber(localProfile.value.adCostRate),
      toOptionalNumber(localProfile.value.procurementCostCny),
      toOptionalNumber(localProfile.value.firstLegCostCny),
      toOptionalNumber(localProfile.value.fbaFulfillmentFeeCny),
      toOptionalNumber(localProfile.value.fbmLastMileCostCny),
      toOptionalNumber(localProfile.value.otherCostCny)
    ],
    () => {
      if (suppressWatch) {
        return
      }
      emit('update:modelValue', normalizeProfile(localProfile.value))
      if (!localProfile.value.fulfillmentMode) {
        resetToUnconfigured()
        return
      }
      debouncedCalculate()
    }
  )

  const alertType = computed(() => (localProfile.value.validationStatus === 'invalid' ? 'error' : 'info'))
  const alertMessage = computed(() => (localProfile.value.validationStatus === 'valid' ? '' : localProfile.value.validationMessage || ''))

  const formatMoney = (value, currency = 'CNY') => {
    const parsed = toOptionalNumber(value)
    if (typeof parsed === 'undefined') {
      return '--'
    }
    return `${String(currency || 'CNY').toUpperCase()} ${parsed.toFixed(2)}`
  }

  const formatPercent = (value) => {
    const parsed = toOptionalNumber(value)
    if (typeof parsed === 'undefined') {
      return '--'
    }
    return `${(parsed * 100).toFixed(2)}%`
  }

  const resultStatusClass = computed(() => {
    const margin = toOptionalNumber(localProfile.value?.result?.netMarginRate)
    if (typeof margin === 'undefined') {
      return 'text-slate-900 dark:text-slate-100'
    }
    if (margin < 0) {
      return 'text-rose-500'
    }
    if (margin < 0.1) {
      return 'text-amber-500'
    }
    return 'text-emerald-500'
  })

  const metricCards = computed(() => {
    const result = localProfile.value?.result || null
    return [
      { label: '毛利额', value: formatMoney(result?.grossProfitCny, 'CNY'), className: resultStatusClass.value },
      { label: '净利额', value: formatMoney(result?.netProfitCny, 'CNY'), className: resultStatusClass.value },
      { label: '净利率', value: formatPercent(result?.netMarginRate), className: resultStatusClass.value },
      { label: '保本售价', value: formatMoney(result?.breakEvenPrice, result?.breakEvenCurrencyCode || props.currencyCode || '--'), className: 'text-slate-900 dark:text-slate-100' },
      { label: 'ROI', value: formatPercent(result?.roiRate), className: resultStatusClass.value }
    ]
  })

  const breakdownItems = computed(() => {
    const breakdown = localProfile.value?.result?.costBreakdown || {}
    const mode = localProfile.value.fulfillmentMode
    return [
      { label: '采购成本', value: formatMoney(breakdown.procurementCostCny, 'CNY') },
      { label: '头程成本', value: formatMoney(breakdown.firstLegCostCny, 'CNY') },
      { label: mode === 'fbm' ? '尾程派送费' : 'FBA 配送费', value: formatMoney(mode === 'fbm' ? breakdown.fbmLastMileCostCny : breakdown.fbaFulfillmentFeeCny, 'CNY') },
      { label: '其他成本', value: formatMoney(breakdown.otherCostCny, 'CNY') },
      { label: '平台佣金', value: formatMoney(breakdown.commissionCny, 'CNY') },
      { label: '广告成本', value: formatMoney(breakdown.adCostCny, 'CNY') },
      { label: '固定成本合计', value: formatMoney(breakdown.fixedCostCny, 'CNY') },
      { label: '销售额折算', value: formatMoney(localProfile.value?.result?.saleCny, 'CNY') }
    ]
  })
</script>
