<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.24em] text-slate-500">AMAZON LOGISTICS</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">美国直发物流比价</h1>
          <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
            当前页面只读取数据库中的已激活报价版本。Excel 上传、历史版本和渠道明细请到物流报价库维护。
          </p>
        </div>
        <div class="gva-btn-list !mb-0">
          <el-button type="primary" @click="goToLibrary">去报价库管理数据</el-button>
          <el-button @click="resetForm">重置条件</el-button>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <div class="mb-4">
        <h2 class="text-lg font-semibold text-slate-900 dark:text-slate-100">查询参数</h2>
        <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">尺寸可以不填；如果填写，长宽高必须全部填写。</p>
      </div>

      <el-form label-position="top" @submit.prevent>
        <div class="grid gap-4 2xl:grid-cols-[minmax(0,640px)_minmax(0,1fr)]">
          <div class="grid gap-4 md:grid-cols-3">
            <el-form-item label="商品重量 (KG)">
              <el-input-number v-model="form.weight_kg" :min="0.01" :precision="3" :step="0.05" class="!w-full" />
            </el-form-item>
            <el-form-item label="平台">
              <el-select v-model="form.platform" class="!w-full">
                <el-option v-for="item in platformOptions" :key="item" :label="item" :value="item" />
              </el-select>
            </el-form-item>
            <el-form-item label="是否带电">
              <el-switch v-model="form.contains_battery" inline-prompt active-text="是" inactive-text="否" />
            </el-form-item>
          </div>

          <div class="grid gap-4 md:grid-cols-3">
            <el-form-item label="长 (cm)">
              <el-input-number v-model="form.length_cm" :min="1" :precision="1" :step="1" class="!w-full" />
            </el-form-item>
            <el-form-item label="宽 (cm)">
              <el-input-number v-model="form.width_cm" :min="1" :precision="1" :step="1" class="!w-full" />
            </el-form-item>
            <el-form-item label="高 (cm)">
              <el-input-number v-model="form.height_cm" :min="1" :precision="1" :step="1" class="!w-full" />
            </el-form-item>
          </div>
        </div>

        <div class="mt-1 rounded border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-700 dark:bg-slate-800/60">
          <div class="grid gap-3 md:grid-cols-2">
            <div>
              <div class="text-xs uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">体积重</div>
              <div class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">{{ formatWeight(volumetricWeight) }}</div>
            </div>
            <div>
              <div class="text-xs uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">计费基准重</div>
              <div class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">{{ formatWeight(billableBaseWeight) }}</div>
            </div>
          </div>
          <div class="mt-3 text-xs text-slate-500 dark:text-slate-400">
            预览按长(cm) × 宽(cm) × 高(cm) / 8000 估算；实际报价按各渠道入库的体积重规则计算。
          </div>
        </div>

        <div class="gva-btn-list !mb-0 mt-2">
          <el-button type="primary" :loading="loading" @click="submit">查询最低报价</el-button>
          <el-button @click="clearDimensions">清空尺寸</el-button>
        </div>
      </el-form>
    </div>

    <div class="gva-table-box">
      <div class="mb-4 flex items-center justify-between gap-4">
        <div>
          <h2 class="text-lg font-semibold text-slate-900 dark:text-slate-100">最低报价概览</h2>
          <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">分别展示云途、燕文与全局最低报价。</p>
        </div>
        <span class="text-sm text-slate-500 dark:text-slate-400">共 {{ quotes.length }} 条报价</span>
      </div>

      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <article
          v-for="card in lowestCards"
          :key="card.key"
          class="rounded border border-slate-200 bg-slate-50 p-5 dark:border-slate-700 dark:bg-slate-800/60"
          :class="card.value ? 'cursor-pointer transition hover:border-blue-400 hover:bg-blue-50/70 dark:hover:border-blue-500 dark:hover:bg-blue-950/30' : ''"
          :tabindex="card.value ? 0 : -1"
          @click="openQuoteDetail(card.value)"
          @keyup.enter="openQuoteDetail(card.value)"
        >
          <div class="text-sm text-slate-500 dark:text-slate-400">{{ card.label }}</div>
          <div class="mt-3 text-3xl font-semibold text-slate-900 dark:text-slate-100">{{ formatPrice(card.value?.price_cny) }}</div>
          <div class="mt-3 text-sm font-medium text-slate-700 dark:text-slate-200">{{ card.value?.logistics_provider || '--' }}</div>
          <div class="mt-1 text-xs text-slate-500 dark:text-slate-400">渠道码：{{ card.value?.service_code || '--' }}</div>
          <div class="mt-1 text-xs text-slate-500 dark:text-slate-400">平台：{{ card.value?.platform || '全部' }}</div>
          <div class="mt-1 text-xs text-slate-500 dark:text-slate-400">时效：{{ card.value?.transit_time || '--' }}</div>
          <div class="mt-1 text-sm text-slate-500 dark:text-slate-400">{{ card.value?.channel_name || '暂无结果' }}</div>
          <div class="mt-2 text-xs text-slate-400 dark:text-slate-500">状态：{{ card.value?.price_status || '--' }}</div>
        </article>
      </div>
    </div>

    <div class="gva-table-box">
      <div class="mb-4">
        <h2 class="text-lg font-semibold text-slate-900 dark:text-slate-100">数据来源</h2>
        <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">仅展示数据库当前激活版本的摘要。</p>
      </div>

      <div class="grid gap-3 md:grid-cols-3">
        <div
          v-for="item in sourceCards"
          :key="item.label"
          class="rounded border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60"
        >
          <div class="text-sm font-medium text-slate-900 dark:text-slate-100">{{ item.label }}</div>
          <div class="mt-2 break-all text-sm text-slate-600 dark:text-slate-300">{{ item.source?.latest_file_name || '暂无已激活文件' }}</div>
          <div class="mt-2 text-xs text-slate-500 dark:text-slate-400">
            批次：{{ item.source?.active_batch_count ?? 0 }} · 渠道：{{ item.source?.active_channel_count ?? 0 }}
          </div>
          <div class="mt-1 text-xs text-slate-500 dark:text-slate-400">
            最近上传：{{ formatDate(item.source?.latest_uploaded_at) || '--' }}
          </div>
        </div>
      </div>

      <div v-if="providerErrorEntries.length" class="mt-4 flex flex-col gap-3">
        <el-alert
          v-for="[provider, message] in providerErrorEntries"
          :key="provider"
          type="warning"
          :closable="false"
          :title="`${provider} 数据源异常：${message}`"
        />
      </div>
    </div>

    <div class="gva-table-box">
      <div class="mb-4">
        <h2 class="text-lg font-semibold text-slate-900 dark:text-slate-100">全量排序结果</h2>
        <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">默认按价格升序展示。</p>
      </div>

      <el-table :data="quotes" stripe empty-text="查询后显示渠道报价">
        <el-table-column prop="provider" label="来源" min-width="90" />
        <el-table-column prop="logistics_provider" label="物流商" min-width="120" />
        <el-table-column prop="platform" label="平台" min-width="100" />
        <el-table-column label="渠道" min-width="260">
          <template #default="{ row }">
            <el-button type="primary" link class="!h-auto !p-0 !whitespace-normal !text-left" @click="openQuoteDetail(row)">
              {{ row.channel_name || '--' }}
            </el-button>
          </template>
        </el-table-column>
        <el-table-column prop="service_code" label="产品代码" min-width="120" />
        <el-table-column prop="transit_time" label="时效" min-width="140" />
        <el-table-column label="价格" min-width="120">
          <template #default="{ row }">
            {{ formatPrice(row.price_cny) }}
          </template>
        </el-table-column>
        <el-table-column label="体积重" min-width="110">
          <template #default="{ row }">
            {{ formatWeight(row.volumetric_weight_kg) }}
          </template>
        </el-table-column>
        <el-table-column prop="price_status" label="状态" min-width="100" />
        <el-table-column label="计费重" min-width="110">
          <template #default="{ row }">
            {{ formatWeight(row.billable_weight_kg) }}
          </template>
        </el-table-column>
        <el-table-column prop="source_mode" label="数据源" min-width="100" />
        <el-table-column label="标签" min-width="220">
          <template #default="{ row }">
            <div class="flex flex-wrap gap-1">
              <el-tag v-for="tag in row.channel_tags || []" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
              <span v-if="!(row.channel_tags || []).length" class="text-xs text-slate-400 dark:text-slate-500">--</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <div v-if="warningList.length" class="gva-table-box">
      <h2 class="text-lg font-semibold text-slate-900 dark:text-slate-100">风险与提示</h2>
      <div class="mt-4 flex flex-col gap-3">
        <el-alert v-for="warning in warningList" :key="warning" type="info" :closable="false" :title="warning" />
      </div>
    </div>

    <el-drawer v-model="detailVisible" title="渠道详情" size="78%" destroy-on-close>
      <div v-if="selectedQuote" v-loading="detailLoading" class="flex flex-col gap-5 p-1">
        <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <h3 class="text-lg font-semibold text-slate-900 dark:text-slate-100">渠道结果</h3>
              <div class="mt-1 text-sm text-slate-500 dark:text-slate-400">
                {{ selectedQuote.logistics_provider || '--' }} · {{ selectedQuote.channel_name || '--' }}
              </div>
            </div>
            <div class="text-left lg:text-right">
              <div class="text-3xl font-semibold text-slate-900 dark:text-slate-100">{{ formatPrice(selectedQuote.price_cny) }}</div>
              <div class="mt-1 text-xs text-slate-500 dark:text-slate-400">状态：{{ selectedQuote.price_status || '--' }}</div>
            </div>
          </div>

          <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <div class="rounded border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/70">
              <div class="text-xs uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">渠道码</div>
              <div class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">{{ selectedQuote.service_code || '--' }}</div>
            </div>
            <div class="rounded border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/70">
              <div class="text-xs uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">平台</div>
              <div class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">{{ selectedQuote.platform || '全部' }}</div>
            </div>
            <div class="rounded border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/70">
              <div class="text-xs uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">计费重</div>
              <div class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">{{ formatWeight(selectedQuote.billable_weight_kg) }}</div>
            </div>
            <div class="rounded border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/70">
              <div class="text-xs uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">体积重</div>
              <div class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">{{ formatWeight(selectedQuote.volumetric_weight_kg) }}</div>
            </div>
          </div>

          <div class="mt-4 grid gap-3 xl:grid-cols-2">
            <div class="rounded border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/70">
              <div class="font-medium text-slate-900 dark:text-slate-100">费用结果</div>
              <div class="mt-3 grid gap-2 text-sm text-slate-600 dark:text-slate-300 sm:grid-cols-2">
                <div v-for="item in feeBreakdownRows" :key="item.label" class="flex items-center justify-between gap-3 rounded bg-slate-50 px-3 py-2 dark:bg-slate-800/80">
                  <span>{{ item.label }}</span>
                  <span class="font-medium text-slate-900 dark:text-slate-100">{{ formatPrice(item.value) }}</span>
                </div>
              </div>
            </div>
            <div class="rounded border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/70">
              <div class="font-medium text-slate-900 dark:text-slate-100">计算过程</div>
              <div class="mt-3 flex flex-col gap-2 text-sm text-slate-600 dark:text-slate-300">
                <span v-for="note in calculationNotes" :key="note">{{ formatCalculationNote(note) }}</span>
                <span v-if="!calculationNotes.length" class="text-slate-400 dark:text-slate-500">--</span>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
          <h3 class="mb-3 text-lg font-semibold text-slate-900 dark:text-slate-100">渠道详情</h3>
          <el-descriptions v-if="detailData" :column="2" border>
            <el-descriptions-item label="服务商">{{ detailData.provider }}</el-descriptions-item>
            <el-descriptions-item label="产品代码/产品号">{{ detailData.product_code || selectedQuote.service_code || '--' }}</el-descriptions-item>
            <el-descriptions-item label="代码类型">{{ detailData.product_code_type || '--' }}</el-descriptions-item>
            <el-descriptions-item label="逻辑产品键">{{ detailData.logical_product_key || '--' }}</el-descriptions-item>
            <el-descriptions-item label="渠道名">{{ detailData.channel_name || selectedQuote.channel_name || '--' }}</el-descriptions-item>
            <el-descriptions-item label="国家">{{ detailData.country_label || '--' }}</el-descriptions-item>
            <el-descriptions-item label="物流商">{{ detailData.logistics_provider || selectedQuote.logistics_provider || '--' }}</el-descriptions-item>
            <el-descriptions-item label="平台">{{ detailData.platform || selectedQuote.platform || '全部' }}</el-descriptions-item>
            <el-descriptions-item label="Sheet">{{ detailData.sheet_name || selectedQuote.sheet_name || '--' }}</el-descriptions-item>
            <el-descriptions-item label="计价类型">{{ detailData.rate_kind || '--' }}</el-descriptions-item>
            <el-descriptions-item label="体积重除数">{{ detailData.volume_divisor || '--' }}</el-descriptions-item>
            <el-descriptions-item label="时效">{{ detailData.transit_time || selectedQuote.transit_time || '--' }}</el-descriptions-item>
            <el-descriptions-item label="生效时间">{{ formatDate(detailData.effective_at) || detailData.effective_text_raw || '--' }}</el-descriptions-item>
            <el-descriptions-item label="来源文件">{{ detailData.source_file_name || '--' }}</el-descriptions-item>
            <el-descriptions-item label="上传时间">{{ formatDate(detailData.uploaded_at) || '--' }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="detailData.is_active ? 'success' : 'info'">{{ detailData.is_active ? 'current' : 'history' }}</el-tag>
            </el-descriptions-item>
          </el-descriptions>
          <el-empty v-else description="暂无渠道详情" />

          <div class="mt-4 grid gap-3 lg:grid-cols-3">
            <div class="rounded border border-slate-200 bg-white p-3 text-sm text-slate-700 dark:border-slate-700 dark:bg-slate-900/70 dark:text-slate-300">
              <div class="font-medium text-slate-900 dark:text-slate-100">标签</div>
              <div class="mt-2 flex flex-wrap gap-2">
                <el-tag v-for="tag in detailTags" :key="tag" size="small">{{ tag }}</el-tag>
                <span v-if="!detailTags.length" class="text-slate-400 dark:text-slate-500">--</span>
              </div>
            </div>
            <div class="rounded border border-slate-200 bg-white p-3 text-sm text-slate-700 dark:border-slate-700 dark:bg-slate-900/70 dark:text-slate-300 lg:col-span-2">
              <div class="font-medium text-slate-900 dark:text-slate-100">Warnings / 未决费用</div>
              <div class="mt-2 flex flex-col gap-1">
                <span v-for="warning in detailWarnings" :key="warning">{{ warning }}</span>
                <span v-if="!detailWarnings.length" class="text-slate-400 dark:text-slate-500">--</span>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
          <h3 class="mb-3 text-lg font-semibold text-slate-900 dark:text-slate-100">费率明细</h3>
          <el-table :data="rateRows" stripe empty-text="暂无费率明细">
            <el-table-column prop="sequence_no" label="#" width="70" />
            <el-table-column prop="zone" label="分区" min-width="110" />
            <el-table-column prop="weight_min_kg" label="最小重量" min-width="110" />
            <el-table-column prop="weight_max_kg" label="最大重量" min-width="110" />
            <el-table-column prop="rate_per_kg" label="元/KG" min-width="100" />
            <el-table-column prop="handling_fee_cny" label="处理费" min-width="100" />
            <el-table-column prop="registration_fee_cny" label="挂号费" min-width="100" />
            <el-table-column prop="first_weight_kg" label="首重(KG)" min-width="100" />
            <el-table-column prop="first_price_cny" label="首重价" min-width="100" />
            <el-table-column prop="continue_weight_kg" label="续重(KG)" min-width="100" />
            <el-table-column prop="continue_price_cny" label="续重价" min-width="100" />
            <el-table-column prop="volume_ratio_min" label="体积比下限" min-width="110" />
            <el-table-column prop="volume_ratio_max" label="体积比上限" min-width="110" />
            <el-table-column prop="billable_weight_mode" label="计重口径" min-width="110" />
            <el-table-column prop="rate_label_raw" label="原始档位" min-width="180" show-overflow-tooltip />
          </el-table>
          <div class="gva-pagination">
            <el-pagination
              layout="total, sizes, prev, pager, next, jumper"
              :current-page="ratePage.page"
              :page-size="ratePage.pageSize"
              :page-sizes="[10, 20, 50]"
              :total="ratePage.total"
              @current-change="handleRateCurrentChange"
              @size-change="handleRateSizeChange"
            />
          </div>
        </section>
      </div>
      <el-empty v-else description="请选择报价渠道" />
    </el-drawer>
  </div>
</template>

<script setup>
  import { computed, reactive, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { ElMessage } from 'element-plus'

  import { requestUSLogisticsQuotes } from '@/api/amazonLogistics'
  import {
    getAmazonLogisticsChannelDetail,
    getAmazonLogisticsRateRowPage
  } from '@/api/amazonLogisticsLibrary'
  import { formatDate } from '@/utils/format'

  defineOptions({
    name: 'AmazonLogisticsQuote'
  })

  const router = useRouter()
  const loading = ref(false)
  const result = ref(null)
  const detailVisible = ref(false)
  const detailLoading = ref(false)
  const selectedQuote = ref(null)
  const detailData = ref(null)
  const rateRows = ref([])
  const ratePage = reactive({ page: 1, pageSize: 10, total: 0 })
  const form = reactive({
    weight_kg: 0.3,
    platform: '全部',
    contains_battery: false,
    length_cm: undefined,
    width_cm: undefined,
    height_cm: undefined
  })
  const platformOptions = ['全部', 'Amazon', '沃尔玛', 'Temu', 'SHEIN', 'TikTok', 'eBay', 'Shopify', 'Wayfair', 'Target', 'AliExpress', 'Shopee', 'Lazada']

  const quotes = computed(() => result.value?.quotes || [])
  const providerErrors = computed(() => result.value?.provider_errors || {})
  const providerErrorEntries = computed(() => Object.entries(providerErrors.value))
  const lowestCards = computed(() => [
    { key: 'yuntu', label: '云途最低', value: result.value?.provider_lowest?.yuntu || null },
    { key: 'yanwen', label: '燕文最低', value: result.value?.provider_lowest?.yanwen || null },
    { key: 'santai', label: '三态最低', value: result.value?.provider_lowest?.santai || null },
    { key: 'overall', label: '全局最低', value: result.value?.overall_lowest || null }
  ])
  const sourceCards = computed(() => {
    const sources = result.value?.sources || {}
    return [
      { label: '云途数据源', source: sources.yuntu || null },
      { label: '燕文数据源', source: sources.yanwen || null },
      { label: '三态数据源', source: sources.santai || null }
    ]
  })
  const warningList = computed(() => {
    const warnings = new Set()
    quotes.value.forEach((item) => {
      ;(item.warnings || []).forEach((warning) => warnings.add(warning))
      ;(item.fee_breakdown?.unresolved_fees || []).forEach((warning) => warnings.add(warning))
    })
    providerErrorEntries.value.forEach(([, message]) => warnings.add(message))
    return Array.from(warnings)
  })
  const feeBreakdownRows = computed(() => {
    const breakdown = selectedQuote.value?.fee_breakdown
    const valueOf = (key) => (breakdown ? breakdown[key] ?? 0 : undefined)
    return [
      { label: '总价', value: selectedQuote.value?.price_cny },
      { label: '基础运费', value: valueOf('base_charge_cny') },
      { label: '处理费', value: valueOf('handling_fee_cny') },
      { label: '挂号费', value: valueOf('registration_fee_cny') },
      { label: '附加费', value: valueOf('mandatory_fee_cny') }
    ]
  })
  const calculationNotes = computed(() => uniqueStrings(selectedQuote.value?.fee_breakdown?.calculation_notes || []))
  const detailTags = computed(() => uniqueStrings([
    ...(selectedQuote.value?.channel_tags || []),
    ...(detailData.value?.tags || [])
  ]))
  const detailWarnings = computed(() => uniqueStrings([
    ...(selectedQuote.value?.warnings || []),
    ...(selectedQuote.value?.fee_breakdown?.unresolved_fees || []),
    ...(detailData.value?.warnings || []),
    ...(detailData.value?.unresolved_fees || [])
  ]))
  const volumetricWeight = computed(() => {
    if (![form.length_cm, form.width_cm, form.height_cm].every((value) => typeof value !== 'undefined')) {
      return null
    }
    return (Number(form.length_cm) * Number(form.width_cm) * Number(form.height_cm)) / 8000
  })
  const billableBaseWeight = computed(() => {
    if (volumetricWeight.value === null) {
      return form.weight_kg
    }
    return Math.max(Number(form.weight_kg) || 0, volumetricWeight.value)
  })

  const formatPrice = (value) => {
    if (value === null || typeof value === 'undefined') {
      return '--'
    }
    return `¥${Number(value).toFixed(2)}`
  }

  const formatWeight = (value) => {
    if (value === null || typeof value === 'undefined') {
      return '--'
    }
    return `${Number(value).toFixed(4)} KG`
  }

  const formatCalculationNote = (note) => {
    if (note.startsWith('actual_weight=')) {
      return `实际重量：${note.replace('actual_weight=', '')}`
    }
    if (note.startsWith('volumetric_weight=')) {
      return `体积重：${note.replace('volumetric_weight=', '')}`
    }
    if (note.startsWith('billable_base=max(actual, volumetric)=')) {
      return `计费基准：实际重量与体积重取较大值 = ${note.replace('billable_base=max(actual, volumetric)=', '')}`
    }
    if (note.startsWith('min_billable=')) {
      return `最低计费重：${note.replace('min_billable=', '')}`
    }
    if (note.startsWith('rounded_step=')) {
      return `进位步长：${note.replace('rounded_step=', '')}`
    }
    return note
  }

  const clearDimensions = () => {
    form.length_cm = undefined
    form.width_cm = undefined
    form.height_cm = undefined
  }

  const resetForm = () => {
    form.weight_kg = 0.3
    form.platform = '全部'
    form.contains_battery = false
    clearDimensions()
    result.value = null
  }

  const uniqueStrings = (values = []) => Array.from(new Set(values.filter(Boolean)))

  const goToLibrary = () => {
    router.push('/layout/amazon-logistics/logisticsLibrary')
  }

  const resetDetailPaging = () => {
    ratePage.page = 1
    ratePage.pageSize = 10
    ratePage.total = 0
    rateRows.value = []
  }

  const fetchRateRows = async (channelVersionId) => {
    const res = await getAmazonLogisticsRateRowPage({
      channelVersionId,
      page: ratePage.page,
      pageSize: ratePage.pageSize
    })
    if (res.code === 0) {
      rateRows.value = res.data.list || []
      ratePage.total = res.data.total || 0
      ratePage.page = res.data.page || ratePage.page
      ratePage.pageSize = res.data.pageSize || ratePage.pageSize
    }
  }

  const loadQuoteDetail = async (channelVersionId) => {
    detailLoading.value = true
    try {
      const detailRes = await getAmazonLogisticsChannelDetail({ channelVersionId })
      if (detailRes.code === 0) {
        detailData.value = detailRes.data || null
      }
      await fetchRateRows(channelVersionId)
    } finally {
      detailLoading.value = false
    }
  }

  const openQuoteDetail = async (quote) => {
    if (!quote) return
    selectedQuote.value = quote
    detailData.value = null
    resetDetailPaging()
    detailVisible.value = true
    if (!quote.channel_version_id) {
      ElMessage.warning('当前报价缺少渠道详情ID')
      return
    }
    await loadQuoteDetail(quote.channel_version_id)
  }

  const handleRateCurrentChange = (page) => {
    ratePage.page = page
    if (selectedQuote.value?.channel_version_id) {
      fetchRateRows(selectedQuote.value.channel_version_id)
    }
  }

  const handleRateSizeChange = (pageSize) => {
    ratePage.pageSize = pageSize
    ratePage.page = 1
    if (selectedQuote.value?.channel_version_id) {
      fetchRateRows(selectedQuote.value.channel_version_id)
    }
  }

  const submit = async () => {
    const hasAnyDimension = [form.length_cm, form.width_cm, form.height_cm].some((value) => typeof value !== 'undefined')
    const hasAllDimensions = [form.length_cm, form.width_cm, form.height_cm].every((value) => typeof value !== 'undefined')
    if (hasAnyDimension && !hasAllDimensions) {
      ElMessage.error('长宽高要么都不填，要么全部填写')
      return
    }

    loading.value = true
    try {
      const response = await requestUSLogisticsQuotes({
        weight_kg: form.weight_kg,
        platform: form.platform,
        contains_battery: form.contains_battery,
        length_cm: form.length_cm,
        width_cm: form.width_cm,
        height_cm: form.height_cm
      })

      result.value = response.data || null
      if (response.code !== 0) {
        return
      }
      if (!(response.data?.quotes || []).length) {
        ElMessage.warning('没有匹配到可报价渠道')
      }
    } catch (error) {
      ElMessage.error(error?.response?.data?.msg || '物流比价失败')
    } finally {
      loading.value = false
    }
  }
</script>
