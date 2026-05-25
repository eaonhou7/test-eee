<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.28em] text-slate-500">AMAZON LOGISTICS LIBRARY</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">物流报价库</h1>
          <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
            上传云途、燕文或三态报价 Excel，解析产品代码/产品号并落库；主列表支持搜索、分页、详情、费率明细和历史版本查看。
          </p>
        </div>
        <div class="gva-btn-list !mb-0">
          <el-button type="primary" @click="openUploadDialog">上传报价 Excel</el-button>
          <el-button @click="fetchTableData">刷新列表</el-button>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchTableData">
        <el-form-item label="服务商">
          <el-select v-model="searchInfo.provider" clearable class="!w-32" placeholder="全部">
            <el-option label="云途" value="yuntu" />
            <el-option label="燕文" value="yanwen" />
            <el-option label="三态" value="santai" />
          </el-select>
        </el-form-item>
        <el-form-item label="国家">
          <el-input v-model="searchInfo.country_label" clearable placeholder="美国 / 德国 / 日本..." class="!w-40" />
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="产品代码 / 渠道 / Sheet / 文件名" />
        </el-form-item>
        <el-form-item label="状态范围">
          <el-select v-model="searchInfo.active_scope" class="!w-36">
            <el-option label="当前版本" value="current" />
            <el-option label="历史版本" value="history" />
            <el-option label="全部" value="all" />
          </el-select>
        </el-form-item>
        <el-form-item label="平台">
          <el-select v-model="searchInfo.platform" clearable class="!w-36" placeholder="不限">
            <el-option v-for="item in platformOptions" :key="item" :label="item" :value="item" />
          </el-select>
        </el-form-item>
        <el-form-item label="物流商">
          <el-input v-model="searchInfo.logistics_provider" clearable placeholder="DHL / 云途 / 燕文..." />
        </el-form-item>
        <el-form-item label="生效时间">
          <el-date-picker
            v-model="searchInfo.effective_range"
            type="daterange"
            value-format="YYYY-MM-DD"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
          />
        </el-form-item>
        <el-form-item label="上传时间">
          <el-date-picker
            v-model="searchInfo.uploaded_range"
            type="daterange"
            value-format="YYYY-MM-DD"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchTableData">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <div class="mb-4 text-sm text-slate-500 dark:text-slate-400">
        当前筛选：{{ searchSummary }}
      </div>

      <el-table :data="tableData" stripe>
        <el-table-column prop="provider" label="服务商" width="90" />
        <el-table-column prop="product_code" label="产品代码/产品号" min-width="150" />
        <el-table-column prop="country_label" label="国家" min-width="110" />
        <el-table-column label="时效" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.transit_time || '--' }}
          </template>
        </el-table-column>
        <el-table-column prop="product_code_type" label="代码类型" min-width="120" />
        <el-table-column prop="channel_name" label="渠道名" min-width="240" show-overflow-tooltip />
        <el-table-column prop="logistics_provider" label="物流商" min-width="120" />
        <el-table-column label="平台" min-width="100">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ row.platform || '全部' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sheet_name" label="Sheet" min-width="220" show-overflow-tooltip />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.is_active ? 'success' : 'info'">{{ row.is_active ? 'current' : 'history' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="生效时间" min-width="170">
          <template #default="{ row }">
            {{ formatDate(row.effective_at) || row.effective_text_raw || '--' }}
          </template>
        </el-table-column>
        <el-table-column prop="source_file_name" label="来源文件" min-width="220" show-overflow-tooltip />
        <el-table-column label="最近上传" min-width="170">
          <template #default="{ row }">
            {{ formatDate(row.uploaded_at) || '--' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDetail(row)">查看详情</el-button>
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

    <el-dialog v-model="uploadDialogVisible" title="上传物流报价 Excel" width="560px" destroy-on-close>
      <el-form label-width="110px">
        <el-form-item label="服务商类型">
          <el-select v-model="uploadForm.provider" class="!w-full">
            <el-option label="云途" value="yuntu" />
            <el-option label="燕文" value="yanwen" />
            <el-option label="三态" value="santai" />
          </el-select>
        </el-form-item>
        <el-form-item label="Excel 文件">
          <input accept=".xlsx" type="file" @change="onUploadFileChange" />
          <div class="mt-2 text-sm text-slate-500 dark:text-slate-400">{{ uploadFile?.name || '未选择文件' }}</div>
        </el-form-item>
      </el-form>

      <el-alert v-if="uploadResult" :type="uploadResult.status === 'success' ? 'success' : 'warning'" :closable="false" class="mt-4">
        <template #title>
          <div class="space-y-1 text-sm">
            <div>批次号：{{ uploadResult.batch_id }}</div>
            <div>识别渠道：{{ uploadResult.parsed_channel_count || 0 }}，费率行：{{ uploadResult.parsed_rate_row_count || 0 }}，触达产品：{{ uploadResult.touched_product_count || 0 }}</div>
            <div v-if="uploadResult.failure_reason">失败原因：{{ uploadResult.failure_reason }}</div>
          </div>
        </template>
      </el-alert>

      <template #footer>
        <el-button @click="closeUploadDialog">取消</el-button>
        <el-button type="primary" :loading="uploadLoading" @click="submitUpload">开始上传</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" title="渠道详情" size="78%" destroy-on-close>
      <div v-if="detailData" class="flex flex-col gap-5 p-1">
        <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="服务商">{{ detailData.provider }}</el-descriptions-item>
            <el-descriptions-item label="产品代码/产品号">{{ detailData.product_code || '--' }}</el-descriptions-item>
            <el-descriptions-item label="代码类型">{{ detailData.product_code_type || '--' }}</el-descriptions-item>
            <el-descriptions-item label="逻辑产品键">{{ detailData.logical_product_key }}</el-descriptions-item>
            <el-descriptions-item label="渠道名">{{ detailData.channel_name }}</el-descriptions-item>
            <el-descriptions-item label="国家">{{ detailData.country_label || '--' }}</el-descriptions-item>
            <el-descriptions-item label="物流商">{{ detailData.logistics_provider }}</el-descriptions-item>
            <el-descriptions-item label="平台">{{ detailData.platform || '全部' }}</el-descriptions-item>
            <el-descriptions-item label="Sheet">{{ detailData.sheet_name }}</el-descriptions-item>
            <el-descriptions-item label="计价类型">{{ detailData.rate_kind || '--' }}</el-descriptions-item>
            <el-descriptions-item label="体积重除数">{{ detailData.volume_divisor || '--' }}</el-descriptions-item>
            <el-descriptions-item label="时效">{{ detailData.transit_time || '--' }}</el-descriptions-item>
            <el-descriptions-item label="生效时间">{{ formatDate(detailData.effective_at) || detailData.effective_text_raw || '--' }}</el-descriptions-item>
            <el-descriptions-item label="来源文件">{{ detailData.source_file_name || '--' }}</el-descriptions-item>
            <el-descriptions-item label="上传时间">{{ formatDate(detailData.uploaded_at) || '--' }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="detailData.is_active ? 'success' : 'info'">{{ detailData.is_active ? 'current' : 'history' }}</el-tag>
            </el-descriptions-item>
          </el-descriptions>

          <div class="mt-4 grid gap-3 lg:grid-cols-3">
            <div class="rounded border border-slate-200 bg-white p-3 text-sm text-slate-700 dark:border-slate-700 dark:bg-slate-900/70 dark:text-slate-300">
              <div class="font-medium text-slate-900 dark:text-slate-100">标签</div>
              <div class="mt-2 flex flex-wrap gap-2">
                <el-tag v-for="tag in detailData.tags || []" :key="tag" size="small">{{ tag }}</el-tag>
                <span v-if="!(detailData.tags || []).length" class="text-slate-400 dark:text-slate-500">--</span>
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

        <el-tabs v-model="detailTab">
          <el-tab-pane label="费率明细" name="rates">
            <el-table :data="rateRows" stripe>
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
          </el-tab-pane>

          <el-tab-pane label="版本历史" name="versions">
            <el-table :data="versionList" stripe>
              <el-table-column prop="batch_id" label="批次号" width="90" />
              <el-table-column prop="product_code" label="产品代码/产品号" min-width="140" />
              <el-table-column label="平台" min-width="100">
                <template #default="{ row }">
                  <el-tag size="small" effect="plain">{{ row.platform || '全部' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="100">
                <template #default="{ row }">
                  <el-tag :type="row.is_active ? 'success' : 'info'">{{ row.is_active ? 'current' : 'history' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="生效时间" min-width="170">
                <template #default="{ row }">
                  {{ formatDate(row.effective_at) || row.effective_text_raw || '--' }}
                </template>
              </el-table-column>
              <el-table-column prop="source_file_name" label="来源文件" min-width="220" show-overflow-tooltip />
              <el-table-column label="上传时间" min-width="170">
                <template #default="{ row }">
                  {{ formatDate(row.uploaded_at) || '--' }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="120" fixed="right">
                <template #default="{ row }">
                  <el-button type="primary" link @click="switchVersion(row)">查看此版本</el-button>
                </template>
              </el-table-column>
            </el-table>
            <div class="gva-pagination">
              <el-pagination
                layout="total, sizes, prev, pager, next, jumper"
                :current-page="versionPage.page"
                :page-size="versionPage.pageSize"
                :page-sizes="[10, 20, 50]"
                :total="versionPage.total"
                @current-change="handleVersionCurrentChange"
                @size-change="handleVersionSizeChange"
              />
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
  import { computed, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'

  import {
    uploadAmazonLogisticsWorkbook,
    getAmazonLogisticsChannelPage,
    getAmazonLogisticsChannelDetail,
    getAmazonLogisticsRateRowPage,
    getAmazonLogisticsVersionPage
  } from '@/api/amazonLogisticsLibrary'
  import { formatDate } from '@/utils/format'

  defineOptions({
    name: 'AmazonLogisticsLibrary'
  })

  const searchInfo = reactive({
    provider: '',
    country_label: '',
    keyword: '',
    active_scope: 'current',
    platform: '',
    logistics_provider: '',
    effective_range: [],
    uploaded_range: [],
    page: 1,
    pageSize: 10
  })
  const tableData = ref([])
  const total = ref(0)
  const uploadDialogVisible = ref(false)
  const uploadLoading = ref(false)
  const uploadFile = ref(null)
  const uploadResult = ref(null)
  const uploadForm = reactive({
    provider: 'yuntu'
  })
  const detailVisible = ref(false)
  const detailData = ref(null)
  const detailTab = ref('rates')
  const rateRows = ref([])
  const versionList = ref([])
  const ratePage = reactive({ page: 1, pageSize: 10, total: 0 })
  const versionPage = reactive({ page: 1, pageSize: 10, total: 0 })
  const platformOptions = ['全部', 'Amazon', '沃尔玛', 'Temu', 'SHEIN', 'TikTok', 'eBay', 'Shopify', 'Wayfair', 'Target', 'AliExpress', 'Shopee', 'Lazada']

  const searchSummary = computed(() => {
    const parts = []
    parts.push(`范围：${searchInfo.active_scope}`)
    if (searchInfo.provider) parts.push(`服务商：${searchInfo.provider}`)
    if (searchInfo.country_label) parts.push(`国家：${searchInfo.country_label}`)
    if (searchInfo.keyword) parts.push(`关键词：${searchInfo.keyword}`)
    if (searchInfo.platform) parts.push(`平台：${searchInfo.platform}`)
    if (searchInfo.logistics_provider) parts.push(`物流商：${searchInfo.logistics_provider}`)
    return parts.join(' / ')
  })

  const detailWarnings = computed(() => {
    if (!detailData.value) return []
    return [...(detailData.value.warnings || []), ...(detailData.value.unresolved_fees || [])]
  })

  const buildSearchPayload = () => ({
    page: searchInfo.page,
    pageSize: searchInfo.pageSize,
    provider: searchInfo.provider,
    country_label: searchInfo.country_label,
    keyword: searchInfo.keyword,
    active_scope: searchInfo.active_scope,
    platform: searchInfo.platform,
    logistics_provider: searchInfo.logistics_provider,
    effective_date_start: searchInfo.effective_range?.[0] || '',
    effective_date_end: searchInfo.effective_range?.[1] || '',
    uploaded_date_start: searchInfo.uploaded_range?.[0] || '',
    uploaded_date_end: searchInfo.uploaded_range?.[1] || ''
  })

  const fetchTableData = async () => {
    const res = await getAmazonLogisticsChannelPage(buildSearchPayload())
    if (res.code === 0) {
      tableData.value = res.data.list || []
      total.value = res.data.total || 0
      searchInfo.page = res.data.page || searchInfo.page
      searchInfo.pageSize = res.data.pageSize || searchInfo.pageSize
    }
  }

  const resetSearch = () => {
    searchInfo.provider = ''
    searchInfo.country_label = ''
    searchInfo.keyword = ''
    searchInfo.active_scope = 'current'
    searchInfo.platform = ''
    searchInfo.logistics_provider = ''
    searchInfo.effective_range = []
    searchInfo.uploaded_range = []
    searchInfo.page = 1
    searchInfo.pageSize = 10
    fetchTableData()
  }

  const handleCurrentChange = (page) => {
    searchInfo.page = page
    fetchTableData()
  }

  const handleSizeChange = (pageSize) => {
    searchInfo.pageSize = pageSize
    searchInfo.page = 1
    fetchTableData()
  }

  const openUploadDialog = () => {
    uploadDialogVisible.value = true
    uploadResult.value = null
    uploadFile.value = null
    uploadForm.provider = 'yuntu'
  }

  const closeUploadDialog = () => {
    uploadDialogVisible.value = false
    uploadLoading.value = false
    uploadFile.value = null
    uploadResult.value = null
  }

  const onUploadFileChange = (event) => {
    uploadFile.value = event.target.files?.[0] || null
  }

  const submitUpload = async () => {
    if (!uploadFile.value) {
      ElMessage.error('请选择 Excel 文件')
      return
    }
    uploadLoading.value = true
    try {
      const res = await uploadAmazonLogisticsWorkbook(uploadForm.provider, uploadFile.value)
      uploadResult.value = res.data || null
      if (res.code === 0) {
        ElMessage.success('上传并解析成功')
        fetchTableData()
      }
    } catch (error) {
      ElMessage.error(error?.response?.data?.msg || '上传失败')
    } finally {
      uploadLoading.value = false
    }
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

  const fetchVersionList = async (provider, logicalProductKey) => {
    const res = await getAmazonLogisticsVersionPage({
      provider,
      logical_product_key: logicalProductKey,
      page: versionPage.page,
      pageSize: versionPage.pageSize
    })
    if (res.code === 0) {
      versionList.value = res.data.list || []
      versionPage.total = res.data.total || 0
      versionPage.page = res.data.page || versionPage.page
      versionPage.pageSize = res.data.pageSize || versionPage.pageSize
    }
  }

  const loadDetail = async (channelVersionId, options = {}) => {
    const res = await getAmazonLogisticsChannelDetail({ channelVersionId })
    if (res.code !== 0) return
    detailData.value = res.data
    if (!options.skipPagingReset) {
      ratePage.page = 1
      versionPage.page = 1
    }
    await fetchRateRows(channelVersionId)
    await fetchVersionList(res.data.provider, res.data.logical_product_key)
  }

  const openDetail = async (row) => {
    detailVisible.value = true
    detailTab.value = 'rates'
    await loadDetail(row.id)
  }

  const switchVersion = async (row) => {
    detailTab.value = 'rates'
    await loadDetail(row.id, { skipPagingReset: true })
  }

  const handleRateCurrentChange = (page) => {
    ratePage.page = page
    if (detailData.value) {
      fetchRateRows(detailData.value.id)
    }
  }

  const handleRateSizeChange = (pageSize) => {
    ratePage.pageSize = pageSize
    ratePage.page = 1
    if (detailData.value) {
      fetchRateRows(detailData.value.id)
    }
  }

  const handleVersionCurrentChange = (page) => {
    versionPage.page = page
    if (detailData.value) {
      fetchVersionList(detailData.value.provider, detailData.value.logical_product_key)
    }
  }

  const handleVersionSizeChange = (pageSize) => {
    versionPage.pageSize = pageSize
    versionPage.page = 1
    if (detailData.value) {
      fetchVersionList(detailData.value.provider, detailData.value.logical_product_key)
    }
  }

  fetchTableData()
</script>
