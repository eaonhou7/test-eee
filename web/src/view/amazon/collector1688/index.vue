<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">1688 货物采集池</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">1688 货物采集池</h1>
          <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
            接收浏览器插件从 1688 图搜与详情页采集的货物数据，查看价格区间、MOQ、SKU 报价、规格参数和当前绑定的系统 code。
          </p>
        </div>
        <div class="gva-btn-list !mb-0">
          <el-button @click="downloadCollectorExtension">下载 Amazon / 1688 采集助手</el-button>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchTable">
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="标题 / offerId / 系统 code / 店铺" />
        </el-form-item>
        <el-form-item label="采集状态">
          <el-select v-model="searchInfo.collectStatus" clearable class="!w-36">
            <el-option label="成功" value="success" />
            <el-option label="告警" value="warning" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-form-item>
        <el-form-item label="绑定状态">
          <el-select v-model="searchInfo.bindingStatus" clearable class="!w-36">
            <el-option label="已激活" value="active" />
            <el-option label="未激活" value="inactive" />
          </el-select>
        </el-form-item>
        <el-form-item label="系统 code">
          <el-input v-model="searchInfo.systemCode" clearable placeholder="SKU / 系统编码" />
        </el-form-item>
        <el-form-item label="店铺/公司">
          <el-input v-model="searchInfo.shopKeyword" clearable placeholder="店铺或公司名" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchTable">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table :data="tableData" row-key="id" stripe>
        <el-table-column label="主图" width="120">
          <template #default="{ row }">
            <div class="flex items-center justify-center">
              <el-image
                v-if="resolveCachedImageUrl(row.mainImageUrl)"
                :src="resolveCachedImageUrl(row.mainImageUrl)"
                fit="cover"
                class="h-16 w-16 rounded-lg border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60"
              />
              <div
                v-else
                class="flex h-16 w-16 items-center justify-center rounded-lg border border-dashed border-slate-300 text-xs text-slate-400 dark:border-slate-700 dark:text-slate-500"
              >
                无图
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="标题 / Offer ID" min-width="280">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <span class="font-medium text-slate-900 dark:text-slate-100">{{ row.title || '--' }}</span>
              <div class="flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
                <span>Offer {{ row.offerId || '--' }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="systemCodeText" label="系统 code" min-width="180" show-overflow-tooltip />
        <el-table-column label="店铺 / 公司" min-width="220">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <span>{{ row.shopName || '--' }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400">{{ row.sellerCompany || '--' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="价格 / MOQ" min-width="180">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <span>{{ formatPriceRange(row) }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400">MOQ {{ formatMoq(row.minOrderQuantity, row.orderUnit) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="categoryPathText" label="分类" min-width="220" show-overflow-tooltip />
        <el-table-column label="状态" width="200">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <el-tag size="small" :type="getCollectStatusType(row.collectStatus)">
                {{ getCollectStatusLabel(row.collectStatus) }}
              </el-tag>
              <span class="text-xs text-slate-500 dark:text-slate-400">
                {{ row.imageCount || 0 }} 张图
                <template v-if="row.bindings?.length"> / {{ row.bindings.length }} 条绑定</template>
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="采集时间" min-width="180">
          <template #default="{ row }">
            <div class="flex flex-col gap-1 text-sm">
              <span>{{ formatDateTime(row.lastCollectedAt) }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400">首次 {{ formatDateTime(row.collectedAt) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <div class="flex items-center gap-2">
              <el-button type="primary" link @click="openDetail(row)">查看详情</el-button>
              <el-dropdown trigger="click" @command="(command) => handleRowAction(command, row)">
                <el-button type="primary" link>更多</el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="open">打开 1688</el-dropdown-item>
                    <el-dropdown-item command="repair">重新采集修复</el-dropdown-item>
                    <el-dropdown-item command="delete" class="!text-rose-500">删除</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
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

    <el-drawer v-model="drawerVisible" title="1688 货物详情" size="88%" destroy-on-close>
      <template v-if="detail">
        <div class="flex flex-col gap-6">
          <section class="rounded-lg border border-slate-200 bg-slate-50 p-5 dark:border-slate-700 dark:bg-slate-800/60">
            <div class="mb-4 flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
              <div class="flex gap-4">
                <el-image
                  v-if="resolveCachedImageUrl(detail.mainImageUrl)"
                  :src="resolveCachedImageUrl(detail.mainImageUrl)"
                  :preview-src-list="previewImageList"
                  fit="cover"
                  preview-teleported
                  class="h-28 w-28 rounded-xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60"
                />
                <div class="space-y-2">
                  <h2 class="text-lg font-semibold text-slate-900 dark:text-slate-100">{{ detail.title || '--' }}</h2>
                  <div class="flex flex-wrap gap-2 text-sm text-slate-500 dark:text-slate-400">
                    <el-tag size="small" type="info">Offer {{ detail.offerId || '--' }}</el-tag>
                    <el-tag size="small" :type="getCollectStatusType(detail.collectStatus)">{{ getCollectStatusLabel(detail.collectStatus) }}</el-tag>
                  </div>
                  <p class="text-sm text-slate-600 dark:text-slate-300">
                    店铺 {{ detail.shopName || '--' }} / 公司 {{ detail.sellerCompany || '--' }} / 发货地 {{ detail.origin || '--' }}
                  </p>
                  <div class="flex flex-wrap gap-3 text-sm text-slate-600 dark:text-slate-300">
                    <span>价格 {{ formatPriceRange(detail) }}</span>
                    <span>MOQ {{ formatMoq(detail.minOrderQuantity, detail.orderUnit) }}</span>
                    <span>运费 {{ detail.freightText || '--' }}</span>
                  </div>
                </div>
              </div>
              <div class="flex gap-3">
                <el-button v-if="detail.productUrl" type="primary" plain @click="openProductUrl(detail.productUrl)">打开 1688</el-button>
                <el-button type="warning" plain @click="createRepairTask(detail)">重新采集修复</el-button>
              </div>
            </div>

            <div v-if="detail.collectWarnings?.length" class="mt-4">
              <div class="mb-2 text-sm font-medium text-slate-900 dark:text-slate-100">采集告警</div>
              <div class="flex flex-wrap gap-2">
                <el-tag v-for="warning in detail.collectWarnings" :key="warning" type="warning">{{ warning }}</el-tag>
              </div>
            </div>
          </section>

          <section class="grid gap-4 xl:grid-cols-2">
            <div class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
              <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">基础信息</div>
              <el-descriptions :column="1" border>
                <el-descriptions-item label="Offer ID">{{ detail.offerId || '--' }}</el-descriptions-item>
                <el-descriptions-item label="价格文案">{{ detail.priceText || '--' }}</el-descriptions-item>
                <el-descriptions-item label="分类">{{ detail.categoryPathText || '--' }}</el-descriptions-item>
                <el-descriptions-item label="店铺">{{ detail.shopName || '--' }}</el-descriptions-item>
                <el-descriptions-item label="公司">{{ detail.sellerCompany || '--' }}</el-descriptions-item>
                <el-descriptions-item label="采集时间">{{ formatDateTime(detail.lastCollectedAt) }}</el-descriptions-item>
              </el-descriptions>
            </div>

            <div class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
              <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">绑定信息</div>
              <div v-if="detail.bindings?.length" class="space-y-3">
                <div
                  v-for="binding in detail.bindings"
                  :key="binding.id"
                  class="rounded-lg border border-slate-200 p-3 text-sm dark:border-slate-700"
                >
                  <div class="mb-2 flex items-center gap-2">
                    <el-tag size="small" :type="binding.isActive ? 'success' : 'info'">{{ binding.isActive ? '激活' : '历史' }}</el-tag>
                    <el-tag size="small" :type="binding.mappingStatus === 'mapped' ? 'success' : 'warning'">
                      {{ binding.mappingStatus === 'mapped' ? '已映射规格' : '待映射规格' }}
                    </el-tag>
                    <span class="font-medium">{{ binding.systemCode || '--' }}</span>
                  </div>
                  <div class="text-slate-500 dark:text-slate-400">
                    Listing Item {{ binding.listingItemId }} / Family {{ binding.listingFamilyId }} / Task {{ binding.taskId }}
                  </div>
                  <div class="mt-3 grid gap-3 xl:grid-cols-[minmax(0,1fr)_auto]">
                    <el-select
                      v-model="variantDrafts[binding.id].selectedSkuKey"
                      filterable
                      clearable
                      placeholder="选择 1688 规格 SKU"
                    >
                      <el-option
                        v-for="option in skuOfferOptions"
                        :key="option.key"
                        :label="option.label"
                        :value="option.key"
                      />
                    </el-select>
                    <el-button
                      type="primary"
                      :disabled="!variantDrafts[binding.id]?.selectedSkuKey"
                      @click="saveVariantMapping(binding)"
                    >
                      保存规格映射
                    </el-button>
                  </div>
                  <div class="mt-2 text-xs text-slate-500 dark:text-slate-400">
                    当前规格：{{ binding.selectedSkuKey || '--' }}
                    <template v-if="Object.keys(binding.selectedSkuAttrs || {}).length">
                      / {{ formatJsonValue(binding.selectedSkuAttrs) }}
                    </template>
                  </div>
                </div>
              </div>
              <el-empty v-else description="暂无绑定信息" :image-size="80" />
            </div>
          </section>

          <section class="grid gap-4 xl:grid-cols-2">
            <div class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
              <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">规格属性</div>
              <el-descriptions :column="1" border>
                <template v-if="specAttributeEntries.length">
                  <el-descriptions-item v-for="entry in specAttributeEntries" :key="entry.key" :label="entry.key">{{ entry.value }}</el-descriptions-item>
                </template>
                <el-descriptions-item v-else label="提示">暂无规格属性</el-descriptions-item>
              </el-descriptions>
            </div>

            <div class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
              <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">SKU 报价</div>
              <el-table :data="detail.skuOffers || []" size="small" border max-height="320">
                <el-table-column label="SKU" min-width="160">
                  <template #default="{ row }">{{ formatJsonValue(row.skuId || row.skuKey || row.sku || row.specId || row.id) }}</template>
                </el-table-column>
                <el-table-column label="图片" width="86">
                  <template #default="{ row }">
                    <el-image
                      v-if="resolveCachedImageUrl(row.imageUrl || row.skuImageUrl || row.skuPicUrl)"
                      :src="resolveCachedImageUrl(row.imageUrl || row.skuImageUrl || row.skuPicUrl)"
                      fit="cover"
                      class="h-12 w-12 rounded bg-slate-100 dark:bg-slate-900/60"
                    />
                    <span v-else>--</span>
                  </template>
                </el-table-column>
                <el-table-column label="属性" min-width="220">
                  <template #default="{ row }">{{ formatJsonValue(row.attributeText || row.specAttrs || row.attrs || row.spec || row.name) }}</template>
                </el-table-column>
                <el-table-column label="价格" min-width="140">
                  <template #default="{ row }">{{ formatJsonValue(row.priceText || row.price || row.amount) }}</template>
                </el-table-column>
                <el-table-column label="库存" min-width="120">
                  <template #default="{ row }">{{ formatJsonValue(row.stockText || row.canBookCount || row.stock || row.amountOnSale) }}</template>
                </el-table-column>
              </el-table>
            </div>
          </section>

          <section class="grid gap-4 xl:grid-cols-2">
            <div class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
              <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">商品属性</div>
              <el-descriptions :column="1" border>
                <template v-if="productAttributeEntries.length">
                  <el-descriptions-item v-for="entry in productAttributeEntries" :key="entry.key" :label="entry.key">{{ entry.value }}</el-descriptions-item>
                </template>
                <el-descriptions-item v-else label="提示">暂无商品属性</el-descriptions-item>
              </el-descriptions>
            </div>

            <div class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
              <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">包装信息</div>
              <el-descriptions :column="1" border>
                <template v-if="packageInfoEntries.length">
                  <el-descriptions-item v-for="entry in packageInfoEntries" :key="entry.key" :label="entry.key">{{ entry.value }}</el-descriptions-item>
                </template>
                <el-descriptions-item v-else label="提示">暂无包装信息</el-descriptions-item>
              </el-descriptions>
            </div>
          </section>

          <section class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
            <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">商品详情</div>
            <div v-if="detailSectionItems.length" class="space-y-5">
              <div v-for="(section, index) in detailSectionItems" :key="`${section.title || 'detail'}-${index}`" class="space-y-3">
                <div v-if="section.title" class="text-sm font-medium text-slate-900 dark:text-slate-100">{{ section.title }}</div>
                <pre v-if="section.text" class="whitespace-pre-wrap break-words text-sm leading-6 text-slate-600 dark:text-slate-300">{{ section.text }}</pre>
                <div v-if="section.imageUrls?.length" class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                  <el-image
                    v-for="imageUrl in section.imageUrls"
                    :key="imageUrl"
                    :src="resolveCachedImageUrl(imageUrl)"
                    fit="cover"
                    class="h-40 w-full rounded-lg bg-slate-100 dark:bg-slate-900/60"
                  />
                </div>
              </div>
            </div>
            <pre v-else-if="fallbackDetailText" class="whitespace-pre-wrap break-words text-sm leading-6 text-slate-600 dark:text-slate-300">{{ fallbackDetailText }}</pre>
            <el-empty v-else description="暂无商品详情" :image-size="80" />
          </section>

          <section class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
            <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">图片列表</div>
            <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <div
                v-for="image in detail.images || []"
                :key="image.id"
                class="rounded-lg border border-slate-200 p-3 dark:border-slate-700"
                >
                <el-image
                  v-if="resolveCachedImageUrl(image.file?.url)"
                  :src="resolveCachedImageUrl(image.file?.url)"
                  fit="cover"
                  class="h-40 w-full rounded-lg bg-slate-100 dark:bg-slate-900/60"
                />
                <div class="mt-2 text-xs text-slate-500 dark:text-slate-400">
                  {{ image.imageType || '--' }} / {{ image.isMain ? '主图' : '普通图' }}
                </div>
              </div>
            </div>
          </section>

          <section class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
            <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">原始 JSON</div>
            <pre class="overflow-auto rounded-lg bg-slate-900 p-4 text-xs text-slate-100">{{ formattedRawPayload }}</pre>
          </section>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'

  import {
    createAmazon1688RepairTask,
    deleteAmazon1688CollectedProduct,
    downloadAmazon1688CollectorExtension,
    findAmazon1688CollectedProduct,
    getAmazon1688CollectedProductList,
    upsertAmazon1688BindingVariantMapping
  } from '@/api/amazon1688Collector'
  import { normalizeBlobResponse, triggerBlobDownload } from '@/utils/blobDownload'
  import { getUrl } from '@/utils/image'

  defineOptions({
    name: 'Amazon1688CollectorPool'
  })

  const searchInfo = ref({
    page: 1,
    pageSize: 10,
    keyword: '',
    collectStatus: '',
    bindingStatus: '',
    systemCode: '',
    shopKeyword: ''
  })
  const tableData = ref([])
  const total = ref(0)
  const drawerVisible = ref(false)
  const detail = ref(null)
  const variantDrafts = ref({})

  const previewImageList = computed(() => (detail.value?.images || [])
    .map((image) => resolveCachedImageUrl(image?.file?.url))
    .filter(Boolean))

  const specAttributeEntries = computed(() => {
    const value = detail.value?.specAttributes || {}
    return Object.entries(value).map(([key, rawValue]) => ({
      key,
      value: formatJsonValue(rawValue)
    }))
  })

  const productAttributeEntries = computed(() => mapObjectEntries(detail.value?.productAttributes))
  const packageInfoEntries = computed(() => mapObjectEntries(detail.value?.packageInfo))
  const detailSectionItems = computed(() => (detail.value?.detailSections || [])
    .map((section) => ({
      title: String(section?.title || '').trim(),
      text: String(section?.text || '').trim() || stripHtml(section?.html || ''),
      html: String(section?.html || '').trim(),
      imageUrls: normalizeDetailImageUrls(section?.imageUrls)
    }))
    .filter((section) => section.title || section.text || section.html || section.imageUrls.length))
  const fallbackDetailText = computed(() => {
    const text = String(detail.value?.detailText || '').trim()
    if (text) {
      return text
    }
    return stripHtml(detail.value?.descriptionHtml || '')
  })
  const formattedRawPayload = computed(() => JSON.stringify(detail.value?.rawPayload || {}, null, 2))
  const skuOfferOptions = computed(() => (detail.value?.skuOffers || []).map((offer, index) => ({
    key: resolveSkuOfferKey(offer, index),
    attrs: resolveSkuOfferAttrs(offer),
    label: formatSkuOfferLabel(offer, index)
  })))

  const mapObjectEntries = (value = {}) => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
      return []
    }
    return Object.entries(value).map(([key, rawValue]) => ({
      key,
      value: formatJsonValue(rawValue)
    }))
  }

  const normalizeDetailImageUrls = (value) => {
    if (!Array.isArray(value)) {
      return []
    }
    return value
      .map((url) => resolveCachedImageUrl(url))
      .filter(Boolean)
  }

  const stripHtml = (value) => String(value || '')
    .replace(/<script[\s\S]*?<\/script>/gi, '')
    .replace(/<style[\s\S]*?<\/style>/gi, '')
    .replace(/<[^>]+>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()

  const fetchTable = async () => {
    const res = await getAmazon1688CollectedProductList(searchInfo.value)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
    searchInfo.value.page = res.data?.page || searchInfo.value.page
    searchInfo.value.pageSize = res.data?.pageSize || searchInfo.value.pageSize
  }

  const resetSearch = async () => {
    searchInfo.value = {
      page: 1,
      pageSize: 10,
      keyword: '',
      collectStatus: '',
      bindingStatus: '',
      systemCode: '',
      shopKeyword: ''
    }
    await fetchTable()
  }

  const handleCurrentChange = async (page) => {
    searchInfo.value.page = page
    await fetchTable()
  }

  const handleSizeChange = async (pageSize) => {
    searchInfo.value.pageSize = pageSize
    searchInfo.value.page = 1
    await fetchTable()
  }

  const loadDetail = async (id) => {
    const res = await findAmazon1688CollectedProduct({ id })
    detail.value = res.data || null
    initializeVariantDrafts(detail.value)
  }

  const openDetail = async (row) => {
    await loadDetail(row.id)
    drawerVisible.value = true
  }

  const handleRowAction = async (command, row) => {
    switch (command) {
      case 'open':
        openProductUrl(row.productUrl)
        return
      case 'repair':
        await createRepairTask(row)
        return
      case 'delete':
        await deleteRow(row)
        return
      default:
        return
    }
  }

  const deleteRow = async (row) => {
    await ElMessageBox.confirm(`确认删除 1688 采集商品 ${row.offerId || row.title || row.id} 吗？`, '删除确认', {
      type: 'warning'
    })
    await deleteAmazon1688CollectedProduct({ id: row.id })
    ElMessage.success('删除成功')
    if (detail.value?.id === row.id) {
      drawerVisible.value = false
      detail.value = null
    }
    await fetchTable()
  }

  const downloadCollectorExtension = async () => {
    const res = await downloadAmazon1688CollectorExtension()
    const { blob, fileName } = await normalizeBlobResponse(res, 'amazon-collector-latest.zip')
    triggerBlobDownload(blob, fileName)
    ElMessage.success('采集助手下载成功')
  }

  const openProductUrl = (productUrl) => {
    const target = String(productUrl || '').trim()
    if (!target) {
      ElMessage.warning('当前记录缺少 1688 链接')
      return
    }
    window.open(target, '_blank', 'noopener,noreferrer')
  }

  const createRepairTask = async (row) => {
    const collectedProductId = Number(row?.id || row?.collectedProductId || 0)
    const offerId = String(row?.offerId || '').trim()
    if (!collectedProductId && !offerId) {
      ElMessage.warning('当前记录缺少修复采集标识')
      return
    }
    const res = await createAmazon1688RepairTask({
      collectedProductId,
      offerId
    })
    const target = String(res.data?.detailUrl || res.data?.searchUrl || '').trim()
    if (!target) {
      ElMessage.warning('修复任务已创建，但缺少打开链接')
      return
    }
    ElMessage.success('修复任务已创建，请在打开的 1688 页面完成采集')
    window.open(target, '_blank', 'noopener,noreferrer')
  }

  const resolveImageUrl = (url) => {
    const formatted = getUrl(String(url || '').trim())
    if (!formatted) {
      return ''
    }
    if (/^(https?:|data:|blob:)/i.test(formatted)) {
      return formatted
    }
    if (typeof window === 'undefined') {
      return formatted
    }
    return new URL(formatted, window.location.origin).href
  }

  const resolveCachedImageUrl = (url) => {
    const resolved = resolveImageUrl(url)
    if (!resolved || isBlocked1688ImageUrl(resolved)) {
      return ''
    }
    return resolved
  }

  const isBlocked1688ImageUrl = (url) => {
    try {
      const host = new URL(url, window.location.origin).host.toLowerCase()
      return host.includes('1688.com') || host.includes('alicdn.com')
    } catch (error) {
      return false
    }
  }

  const formatPriceRange = (row) => {
    const text = String(row?.priceText || '').trim()
    if (text) {
      return text
    }
    const min = row?.priceMin
    const max = row?.priceMax
    const currency = String(row?.currencyCode || '').trim()
    if (typeof min === 'number' && typeof max === 'number') {
      return `${currency} ${min.toFixed(2)} - ${max.toFixed(2)}`.trim()
    }
    if (typeof min === 'number') {
      return `${currency} ${min.toFixed(2)}`.trim()
    }
    return '--'
  }

  const formatMoq = (value, unit) => {
    if (typeof value !== 'number') {
      return '--'
    }
    return `${value}${String(unit || '').trim()}`.trim()
  }

  const formatDateTime = (value) => {
    if (!value) {
      return '--'
    }
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) {
      return value
    }
    return date.toLocaleString()
  }

  const getCollectStatusType = (status) => {
    switch (status) {
      case 'success':
        return 'success'
      case 'warning':
        return 'warning'
      case 'failed':
        return 'danger'
      default:
        return 'info'
    }
  }

  const getCollectStatusLabel = (status) => {
    switch (status) {
      case 'success':
        return '采集成功'
      case 'warning':
        return '采集告警'
      case 'failed':
        return '采集失败'
      default:
        return '未知状态'
    }
  }

  const formatJsonValue = (value) => {
    if (value === null || typeof value === 'undefined' || value === '') {
      return '--'
    }
    if (typeof value === 'string') {
      return value
    }
    try {
      return JSON.stringify(value)
    } catch (error) {
      return String(value)
    }
  }

  const initializeVariantDrafts = (payload) => {
    const nextDrafts = {}
    const optionMap = Object.fromEntries((payload?.skuOffers || []).map((offer, index) => {
      const option = {
        key: resolveSkuOfferKey(offer, index),
        attrs: resolveSkuOfferAttrs(offer)
      }
      return [option.key, option]
    }))
    ;(payload?.bindings || []).forEach((binding) => {
      const selectedKey = binding.selectedSkuKey || ''
      nextDrafts[binding.id] = {
        selectedSkuKey: selectedKey,
        selectedSkuAttrs: optionMap[selectedKey]?.attrs || binding.selectedSkuAttrs || {}
      }
    })
    variantDrafts.value = nextDrafts
  }

  const resolveSkuOfferKey = (offer, index) => {
    return String(offer?.skuId || offer?.skuKey || offer?.sku || offer?.specId || offer?.id || `sku-${index + 1}`).trim()
  }

  const resolveSkuOfferAttrs = (offer) => {
    const rawValue = offer?.specAttrs || offer?.attrs || offer?.spec || offer?.attributeText || offer?.name || {}
    if (rawValue && typeof rawValue === 'object' && !Array.isArray(rawValue)) {
      return rawValue
    }
    return { label: formatJsonValue(rawValue) }
  }

  const formatSkuOfferLabel = (offer, index) => {
    const key = resolveSkuOfferKey(offer, index)
    const attrs = formatJsonValue(offer?.attributeText || resolveSkuOfferAttrs(offer))
    const price = formatJsonValue(offer?.priceText || offer?.price || offer?.amount)
    const stock = formatJsonValue(offer?.stockText || offer?.canBookCount || offer?.stock || offer?.amountOnSale)
    return [key, attrs, price, stock].filter((item) => item && item !== '--').join(' / ')
  }

  const saveVariantMapping = async (binding) => {
    const draft = variantDrafts.value[binding.id]
    if (!draft?.selectedSkuKey) {
      ElMessage.warning('请先选择 1688 规格 SKU')
      return
    }
    const option = skuOfferOptions.value.find((item) => item.key === draft.selectedSkuKey)
    const res = await upsertAmazon1688BindingVariantMapping({
      bindingId: binding.id,
      selectedSkuKey: draft.selectedSkuKey,
      selectedSkuAttrs: option?.attrs || draft.selectedSkuAttrs || {}
    })
    if (res.code === 0) {
      ElMessage.success('规格映射已保存')
      await loadDetail(detail.value.id)
    }
  }

  onMounted(() => {
    fetchTable()
  })
</script>
