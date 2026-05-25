<template>
  <div class="h-full overflow-auto bg-slate-50/70 dark:bg-slate-950">
    <div class="mx-auto flex max-w-[1440px] flex-col gap-6 p-4 lg:p-6">
      <section class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500 dark:text-slate-400">AMAZON DASHBOARD</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">Amazon 经营看板</h1>
          <p class="text-sm text-slate-600 dark:text-slate-300">
            {{ currentDateLabel }} · 今日 / 昨日经营指标、待处理事项、库存预警与销量利润趋势
          </p>
          <p class="text-xs text-slate-500 dark:text-slate-400">
            时区 {{ dashboard.meta.timezone || '--' }} · 利润口径 {{ profitBasisLabel }}
          </p>
        </div>

        <el-form :inline="true" :model="filters" class="!mb-0 flex flex-wrap items-center gap-y-2">
          <el-form-item label="店铺" class="!mb-0">
            <el-select v-model="filters.storeId" clearable filterable class="!w-52" placeholder="全部店铺">
              <el-option
                v-for="store in storeOptions"
                :key="store.id"
                :label="store.storeName"
                :value="store.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="站点" class="!mb-0">
            <el-select v-model="filters.siteCode" class="!w-32" placeholder="全部站点">
              <el-option
                v-for="site in siteOptions"
                :key="site.value || 'all'"
                :label="site.label"
                :value="site.value"
              />
            </el-select>
          </el-form-item>
          <el-form-item class="!mb-0">
            <el-button type="primary" @click="handleQuery">查询</el-button>
            <el-tooltip content="刷新数据" placement="top">
              <el-button :icon="Refresh" circle :loading="refreshing" @click="handleRefresh" />
            </el-tooltip>
          </el-form-item>
        </el-form>
      </section>

      <el-skeleton v-if="loading" animated :rows="12" />

      <section v-else-if="errorMessage" class="rounded-lg border border-rose-200 bg-white p-6 dark:border-rose-900/60 dark:bg-slate-900">
        <el-result icon="error" title="首页数据加载失败" :sub-title="errorMessage">
          <template #extra>
            <el-button type="primary" @click="fetchDashboard">重试</el-button>
          </template>
        </el-result>
      </section>

      <template v-else>
        <section class="grid grid-cols-1 gap-4 xl:grid-cols-3">
          <article class="min-h-[188px] rounded-lg border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium text-slate-500 dark:text-slate-400">订单数</p>
                <p class="mt-3 text-3xl font-semibold text-slate-900 dark:text-slate-100">
                  {{ formatInteger(dashboard.summary.today.orderCount) }}
                </p>
              </div>
              <span class="rounded-md bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300">
                {{ orderComparisonLabel }}
              </span>
            </div>
            <div class="mt-6 grid grid-cols-2 gap-4 text-sm">
              <div class="rounded-lg bg-slate-50 px-3 py-3 dark:bg-slate-800/80">
                <div class="text-slate-500 dark:text-slate-400">今日</div>
                <div class="mt-2 text-xl font-semibold text-slate-900 dark:text-slate-100">
                  {{ formatInteger(dashboard.summary.today.orderCount) }}
                </div>
              </div>
              <div class="rounded-lg bg-slate-50 px-3 py-3 dark:bg-slate-800/80">
                <div class="text-slate-500 dark:text-slate-400">昨日</div>
                <div class="mt-2 text-xl font-semibold text-slate-900 dark:text-slate-100">
                  {{ formatInteger(dashboard.summary.yesterday.orderCount) }}
                </div>
              </div>
            </div>
          </article>

          <article class="min-h-[188px] rounded-lg border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium text-slate-500 dark:text-slate-400">销售额</p>
                <p class="mt-3 text-3xl font-semibold text-slate-900 dark:text-slate-100">
                  {{ salesHeadline }}
                </p>
              </div>
              <span class="rounded-md bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">
                按币种
              </span>
            </div>
            <div class="mt-6 grid grid-cols-1 gap-3 text-sm xl:grid-cols-2">
              <div class="rounded-lg bg-slate-50 px-3 py-3 dark:bg-slate-800/80">
                <div class="text-slate-500 dark:text-slate-400">今日</div>
                <div class="mt-2 space-y-1.5">
                  <div v-for="line in todaySalesLines" :key="`today-${line}`" class="break-all font-medium text-slate-900 dark:text-slate-100">
                    {{ line }}
                  </div>
                </div>
              </div>
              <div class="rounded-lg bg-slate-50 px-3 py-3 dark:bg-slate-800/80">
                <div class="text-slate-500 dark:text-slate-400">昨日</div>
                <div class="mt-2 space-y-1.5">
                  <div v-for="line in yesterdaySalesLines" :key="`yesterday-${line}`" class="break-all font-medium text-slate-900 dark:text-slate-100">
                    {{ line }}
                  </div>
                </div>
              </div>
            </div>
          </article>

          <article class="min-h-[188px] rounded-lg border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
            <div class="flex items-start justify-between gap-3">
              <div class="flex items-center gap-2">
                <p class="text-sm font-medium text-slate-500 dark:text-slate-400">预估利润</p>
                <el-tooltip content="按订单项映射 listing 利润档汇总" placement="top">
                  <el-icon class="text-slate-400"><InfoFilled /></el-icon>
                </el-tooltip>
              </div>
              <span class="rounded-md bg-sky-50 px-2.5 py-1 text-xs font-medium text-sky-700 dark:bg-sky-500/10 dark:text-sky-300">
                CNY
              </span>
            </div>
            <div class="mt-3 text-3xl font-semibold text-slate-900 dark:text-slate-100">
              {{ formatCny(dashboard.summary.today.estimatedProfitCny) }}
            </div>
            <div class="mt-6 grid grid-cols-2 gap-4 text-sm">
              <div class="rounded-lg bg-slate-50 px-3 py-3 dark:bg-slate-800/80">
                <div class="text-slate-500 dark:text-slate-400">今日</div>
                <div class="mt-2 text-xl font-semibold text-slate-900 dark:text-slate-100">
                  {{ formatCny(dashboard.summary.today.estimatedProfitCny) }}
                </div>
              </div>
              <div class="rounded-lg bg-slate-50 px-3 py-3 dark:bg-slate-800/80">
                <div class="text-slate-500 dark:text-slate-400">昨日</div>
                <div class="mt-2 text-xl font-semibold text-slate-900 dark:text-slate-100">
                  {{ formatCny(dashboard.summary.yesterday.estimatedProfitCny) }}
                </div>
              </div>
            </div>
          </article>
        </section>

        <section class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-5">
          <article
            v-for="item in statusCards"
            :key="item.key"
            class="min-h-[148px] rounded-lg border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900"
          >
            <p class="text-sm font-medium text-slate-500 dark:text-slate-400">{{ item.title }}</p>
            <p class="mt-4 text-3xl font-semibold text-slate-900 dark:text-slate-100">{{ formatInteger(item.value) }}</p>
            <p class="mt-4 text-sm text-slate-500 dark:text-slate-400">{{ item.description }}</p>
          </article>
        </section>

        <section class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900">
          <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 class="text-lg font-semibold text-slate-900 dark:text-slate-100">销量 / 利润趋势</h2>
              <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">销量按商品件数统计，利润按 CNY 预估展示。</p>
            </div>
            <el-radio-group v-model="trendRange" size="default">
              <el-radio-button :label="7">7 天</el-radio-button>
              <el-radio-button :label="30">30 天</el-radio-button>
            </el-radio-group>
          </div>

          <div class="mt-6 h-[360px]">
            <el-empty
              v-if="!hasTrendData"
              description="最近区间暂无趋势数据"
              :image-size="80"
              class="h-full justify-center"
            />
            <gva-chart v-else :options="trendOptions" height="360px" />
          </div>
        </section>
      </template>
    </div>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { InfoFilled, Refresh } from '@element-plus/icons-vue'
  import { getAmazonDashboardOverview } from '@/api/amazonDashboard'
  import { getAmazonStoreList } from '@/api/amazonStore'
  import GvaChart from '@/components/charts/index.vue'

  defineOptions({
    name: 'Dashboard'
  })

  const createEmptyDashboard = () => ({
    filters: {
      storeId: 0,
      siteCode: ''
    },
    summary: {
      today: {
        orderCount: 0,
        sales: [],
        estimatedProfitCny: 0
      },
      yesterday: {
        orderCount: 0,
        sales: [],
        estimatedProfitCny: 0
      }
    },
    pending: {
      fbmOrders: 0,
      exceptionOrders: 0,
      needProcurement: 0
    },
    alerts: {
      lowStock: 0,
      outOfStock: 0
    },
    trend: [],
    meta: {
      timezone: '',
      profitBasis: ''
    }
  })

  const filters = ref({
    storeId: undefined,
    siteCode: ''
  })
  const loading = ref(true)
  const refreshing = ref(false)
  const errorMessage = ref('')
  const dashboard = ref(createEmptyDashboard())
  const storeOptions = ref([])
  const trendRange = ref(7)

  const siteOptions = [
    { label: '全部', value: '' },
    { label: 'US', value: 'US' },
    { label: 'CA', value: 'CA' },
    { label: 'MX', value: 'MX' }
  ]

  const currentDateLabel = computed(() => {
    try {
      return new Date().toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
      })
    } catch (e) {
      return new Date().toISOString().slice(0, 10)
    }
  })

  const todaySalesLines = computed(() => buildSalesLines(dashboard.value.summary.today.sales))
  const yesterdaySalesLines = computed(() => buildSalesLines(dashboard.value.summary.yesterday.sales))

  const salesHeadline = computed(() => {
    const count = dashboard.value.summary.today.sales.length
    if (count <= 0) {
      return '0'
    }
    if (count === 1) {
      return formatSalesAmount(dashboard.value.summary.today.sales[0])
    }
    return `${count} 个币种`
  })

  const profitBasisLabel = computed(() => {
    if (dashboard.value.meta.profitBasis === 'estimated_listing_profile') {
      return '利润档预估'
    }
    return dashboard.value.meta.profitBasis || '--'
  })

  const orderComparisonLabel = computed(() => {
    return formatComparison(dashboard.value.summary.today.orderCount, dashboard.value.summary.yesterday.orderCount)
  })

  const statusCards = computed(() => [
    {
      key: 'fbm',
      title: '待处理 FBM',
      value: dashboard.value.pending.fbmOrders,
      description: '等待进入 FBM 履约流程'
    },
    {
      key: 'exception',
      title: '异常订单',
      value: dashboard.value.pending.exceptionOrders,
      description: '有异常代码或履约失败'
    },
    {
      key: 'procurement',
      title: '需采购',
      value: dashboard.value.pending.needProcurement,
      description: '待处理采购或需补单'
    },
    {
      key: 'lowStock',
      title: '库存预警',
      value: dashboard.value.alerts.lowStock,
      description: '可售库存 1-10 的 SKU'
    },
    {
      key: 'outOfStock',
      title: '断货预警',
      value: dashboard.value.alerts.outOfStock,
      description: '当前可售库存为 0 的 SKU'
    }
  ])

  const visibleTrend = computed(() => {
    const trend = dashboard.value.trend || []
    return trend.slice(-trendRange.value)
  })

  const hasTrendData = computed(() => {
    return visibleTrend.value.some((item) => item.orderCount || item.unitsSold || item.estimatedProfitCny)
  })

  const trendOptions = computed(() => {
    const trend = visibleTrend.value
    return {
      color: ['#0f766e', '#f59e0b'],
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'shadow'
        },
        formatter(params) {
          const points = Array.isArray(params) ? params : [params]
          const date = points[0]?.axisValue || '--'
          const source = trend.find((item) => item.date === date)
          const lines = [`${date}`, `订单数：${formatInteger(source?.orderCount || 0)}`]
          points.forEach((point) => {
            if (point.seriesName === '销量件数') {
              lines.push(`销量件数：${formatInteger(point.value || 0)}`)
            }
            if (point.seriesName === '预估利润 CNY') {
              lines.push(`预估利润：${formatCny(point.value || 0)}`)
            }
          })
          return lines.join('<br/>')
        }
      },
      grid: {
        left: 16,
        right: 16,
        top: 24,
        bottom: 24,
        containLabel: true
      },
      legend: {
        top: 0,
        right: 0,
        textStyle: {
          color: '#64748b'
        }
      },
      xAxis: {
        type: 'category',
        data: trend.map((item) => item.date),
        axisLine: {
          lineStyle: {
            color: '#cbd5e1'
          }
        },
        axisLabel: {
          color: '#64748b',
          formatter: (value) => value.slice(5)
        }
      },
      yAxis: [
        {
          type: 'value',
          name: '销量',
          nameTextStyle: {
            color: '#64748b'
          },
          axisLabel: {
            color: '#64748b'
          },
          splitLine: {
            lineStyle: {
              color: '#e2e8f0'
            }
          }
        },
        {
          type: 'value',
          name: '利润',
          nameTextStyle: {
            color: '#64748b'
          },
          axisLabel: {
            color: '#64748b',
            formatter: (value) => `${value}`
          },
          splitLine: {
            show: false
          }
        }
      ],
      series: [
        {
          name: '销量件数',
          type: 'bar',
          barMaxWidth: 28,
          data: trend.map((item) => item.unitsSold || 0),
          itemStyle: {
            borderRadius: [6, 6, 0, 0]
          }
        },
        {
          name: '预估利润 CNY',
          type: 'line',
          yAxisIndex: 1,
          smooth: true,
          symbolSize: 8,
          data: trend.map((item) => item.estimatedProfitCny || 0),
          lineStyle: {
            width: 3
          }
        }
      ]
    }
  })

  const fetchStoreOptions = async () => {
    try {
      const res = await getAmazonStoreList({ page: 1, pageSize: 200 })
      if (res.code === 0) {
        storeOptions.value = res.data.list || []
      }
    } catch (error) {
      storeOptions.value = []
    }
  }

  const fetchDashboard = async () => {
    errorMessage.value = ''
    try {
      const res = await getAmazonDashboardOverview({
        storeId: filters.value.storeId,
        siteCode: filters.value.siteCode
      })
      if (res.code !== 0) {
        throw new Error(res.msg || '首页数据加载失败')
      }
      dashboard.value = {
        ...createEmptyDashboard(),
        ...res.data
      }
    } catch (error) {
      dashboard.value = createEmptyDashboard()
      errorMessage.value = error?.message || '首页数据加载失败'
    }
  }

  const handleQuery = async () => {
    refreshing.value = true
    try {
      await fetchDashboard()
    } finally {
      refreshing.value = false
    }
  }

  const handleRefresh = async () => {
    refreshing.value = true
    try {
      await fetchDashboard()
    } finally {
      refreshing.value = false
    }
  }

  const formatInteger = (value) => {
    return Number(value || 0).toLocaleString('zh-CN')
  }

  const formatNumber = (value) => {
    return Number(value || 0).toLocaleString('zh-CN', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    })
  }

  const formatCny = (value) => {
    return `CNY ${formatNumber(value)}`
  }

  const formatSalesAmount = (item) => {
    return `${item.currencyCode || '--'} ${formatNumber(item.amount)}`
  }

  const buildSalesLines = (items) => {
    if (!items || items.length === 0) {
      return ['0']
    }
    return items.map((item) => formatSalesAmount(item))
  }

  const formatComparison = (todayValue, yesterdayValue) => {
    const today = Number(todayValue || 0)
    const yesterday = Number(yesterdayValue || 0)
    if (yesterday === 0) {
      return today === 0 ? '较昨日 0%' : '较昨日 --'
    }
    const ratio = ((today - yesterday) / yesterday) * 100
    const sign = ratio > 0 ? '+' : ''
    return `较昨日 ${sign}${ratio.toFixed(1)}%`
  }

  onMounted(async () => {
    loading.value = true
    try {
      await Promise.allSettled([fetchStoreOptions(), fetchDashboard()])
    } finally {
      loading.value = false
    }
  })
</script>

<style lang="scss" scoped>
  :deep(.el-radio-group) {
    display: inline-flex;
  }
</style>
