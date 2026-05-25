<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON 采集池</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">采集商品列表</h1>
          <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
            接收 Chrome 插件从 Amazon 详情页采集的商品数据，集中查看主图、价格、卖点、属性、变体和原始 JSON，
            并支持重试图片入库。
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
          <el-input v-model="searchInfo.keyword" clearable placeholder="标题 / ASIN / 品牌 / 卖家" />
        </el-form-item>
        <el-form-item label="站点">
          <el-select v-model="searchInfo.siteCode" clearable class="!w-32">
            <el-option label="US" value="US" />
            <el-option label="CA" value="CA" />
            <el-option label="MX" value="MX" />
          </el-select>
        </el-form-item>
        <el-form-item label="采集状态">
          <el-select v-model="searchInfo.collectStatus" clearable class="!w-36">
            <el-option label="成功" value="success" />
            <el-option label="告警" value="warning" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-form-item>
        <el-form-item label="品牌">
          <el-input v-model="searchInfo.brand" clearable placeholder="品牌名称" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="searchInfo.categoryLeaf" clearable filterable class="!w-48" placeholder="请选择分类">
            <el-option
              v-for="option in categoryOptions"
              :key="option.value"
              :label="`${option.label} (${option.count})`"
              :value="option.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="分类关键词">
          <el-input v-model="searchInfo.categoryKeyword" clearable placeholder="类目路径关键词" />
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
              <button
                v-if="resolveImageUrl(row.mainImageUrl)"
                type="button"
                class="rounded-lg border border-slate-200 bg-slate-50 p-0 transition hover:border-sky-500 hover:shadow-sm dark:border-slate-700 dark:bg-slate-900/60"
                @click="openRiskEditor(row)"
              >
                <el-image
                  :src="resolveImageUrl(row.mainImageUrl)"
                  fit="cover"
                  class="h-16 w-16 rounded-lg"
                />
              </button>
              <div
                v-else
                class="flex h-16 w-16 items-center justify-center rounded-lg border border-dashed border-slate-300 text-xs text-slate-400 dark:border-slate-700 dark:text-slate-500"
              >
                无图
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="标题 / ASIN" min-width="280">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <span class="font-medium text-slate-900 dark:text-slate-100">{{ row.title || '--' }}</span>
              <div class="flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
                <el-button
                  v-if="row.productUrl && row.asin"
                  type="primary"
                  link
                  class="!h-auto !min-h-0 !p-0 !text-xs"
                  @click="openProductUrl(row.productUrl)"
                >
                  ASIN {{ row.asin }}
                </el-button>
                <span v-else>ASIN {{ row.asin || '--' }}</span>
                <span v-if="row.parentAsin">父 ASIN {{ row.parentAsin }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="站点" width="90">
          <template #default="{ row }">
            <el-tag size="small">{{ row.siteCode || '--' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="brand" label="品牌" min-width="120" />
        <el-table-column label="价格" width="120">
          <template #default="{ row }">{{ formatPrice(row.priceAmount, row.currencyCode) }}</template>
        </el-table-column>
        <el-table-column label="评分" width="140">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <span>{{ formatRating(row.ratingValue) }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400">{{ formatReviewCount(row.reviewCount) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="分类" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">{{ row.categoryPathText || row.categoryLeaf || '--' }}</template>
        </el-table-column>
        <el-table-column prop="bsrText" label="BSR / 类目排名" min-width="220" show-overflow-tooltip />
        <el-table-column prop="sellerName" label="卖家" min-width="140" show-overflow-tooltip />
        <el-table-column label="状态" width="200">
          <template #default="{ row }">
            <div class="flex flex-col gap-1">
              <el-tag size="small" :type="getCollectStatusType(row.collectStatus)">
                {{ getCollectStatusLabel(row.collectStatus) }}
              </el-tag>
              <el-tag size="small" :type="getInfringementStatusType(row.infringementStatus)">
                {{ getInfringementStatusLabel(row.infringementStatus) }}
              </el-tag>
              <span class="text-xs text-slate-500 dark:text-slate-400">
                {{ row.imageCount || 0 }} 张图
                <template v-if="row.collectWarnings?.length"> / {{ row.collectWarnings.length }} 条告警</template>
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
              <el-dropdown trigger="click" @command="(command) => handleCollectorRowAction(command, row)">
                <el-button type="primary" link>更多</el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="sync">同步到商品上架管理</el-dropdown-item>
                    <el-dropdown-item command="risk">侵权校验</el-dropdown-item>
                    <el-dropdown-item command="rebind">重试图片入库</el-dropdown-item>
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

    <el-drawer v-model="drawerVisible" title="采集商品详情" size="86%" destroy-on-close>
      <template v-if="detail">
        <div class="flex flex-col gap-6">
          <section class="rounded-lg border border-slate-200 bg-slate-50 p-5 dark:border-slate-700 dark:bg-slate-800/60">
            <div class="mb-4 flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
              <div class="flex gap-4">
                <el-image
                  v-if="resolveImageUrl(detail.mainImageUrl)"
                  :src="resolveImageUrl(detail.mainImageUrl)"
                  :preview-src-list="previewImageList"
                  fit="cover"
                  preview-teleported
                  class="h-28 w-28 rounded-xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60"
                />
                <div class="space-y-2">
                  <h2 class="text-lg font-semibold text-slate-900 dark:text-slate-100">{{ detail.title || '--' }}</h2>
                  <div class="flex flex-wrap gap-2 text-sm text-slate-500 dark:text-slate-400">
                    <el-tag size="small">{{ detail.siteCode || '--' }}</el-tag>
                    <el-tag size="small" type="info">{{ detail.asin || '--' }}</el-tag>
                    <el-tag size="small" :type="getCollectStatusType(detail.collectStatus)">{{ getCollectStatusLabel(detail.collectStatus) }}</el-tag>
                    <el-tag size="small" :type="getInfringementStatusType(detail.infringementStatus)">{{ getInfringementStatusLabel(detail.infringementStatus) }}</el-tag>
                  </div>
                  <p class="text-sm text-slate-600 dark:text-slate-300">
                    品牌 {{ detail.brand || '--' }} / 卖家 {{ detail.sellerName || '--' }} / 发货方式 {{ detail.fulfillmentChannel || '--' }}
                    / 发货时效 {{ detail.deliveryEstimateText || '--' }}
                  </p>
                  <div class="flex flex-wrap gap-3 text-sm text-slate-600 dark:text-slate-300">
                    <span>价格 {{ formatPrice(detail.priceAmount, detail.currencyCode) }}</span>
                    <span>划线价 {{ formatPrice(detail.listPriceAmount, detail.currencyCode) }}</span>
                    <span>评分 {{ formatRating(detail.ratingValue) }}</span>
                    <span>评论 {{ formatReviewCount(detail.reviewCount) }}</span>
                  </div>
                </div>
              </div>
              <div class="flex gap-3">
                <el-button v-if="detail.productUrl" type="primary" plain @click="openProductUrl(detail.productUrl)">打开 Amazon</el-button>
                <el-button type="warning" @click="rebindImages(detail)" :loading="rebindLoadingId === detail.id">重试图片入库</el-button>
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
                <el-descriptions-item label="ASIN">{{ detail.asin || '--' }}</el-descriptions-item>
                <el-descriptions-item label="父 ASIN">{{ detail.parentAsin || '--' }}</el-descriptions-item>
                <el-descriptions-item label="Marketplace ID">{{ detail.marketplaceId || '--' }}</el-descriptions-item>
                <el-descriptions-item label="发货方式">{{ detail.fulfillmentChannel || '--' }}</el-descriptions-item>
                <el-descriptions-item label="发货时效">{{ detail.deliveryEstimateText || '--' }}</el-descriptions-item>
                <el-descriptions-item label="侵权状态">
                  <el-tag size="small" :type="getInfringementStatusType(detail.infringementStatus)">{{ getInfringementStatusLabel(detail.infringementStatus) }}</el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="折扣文案">{{ detail.discountText || '--' }}</el-descriptions-item>
                <el-descriptions-item label="采集时间">{{ formatDateTime(detail.lastCollectedAt) }}</el-descriptions-item>
              </el-descriptions>
            </div>

            <div class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
              <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">类目与排名 / 侵权截图</div>
              <div class="space-y-3 text-sm text-slate-600 dark:text-slate-300">
                <div>
                  <div class="mb-1 font-medium text-slate-900 dark:text-slate-100">类目路径</div>
                  <div class="flex flex-wrap gap-2">
                    <el-tag v-for="category in detail.categoryPath || []" :key="category" size="small" type="info">{{ category }}</el-tag>
                    <span v-if="!detail.categoryPath?.length">--</span>
                  </div>
                </div>
                <div>
                  <div class="mb-1 font-medium text-slate-900 dark:text-slate-100">BSR 文案</div>
                  <div>{{ detail.bsrText || '--' }}</div>
                </div>
                <div>
                  <div class="mb-1 font-medium text-slate-900 dark:text-slate-100">侵权截图</div>
                  <el-image
                    v-if="resolveImageUrl(detail.infringementScreenshot?.url)"
                    :src="resolveImageUrl(detail.infringementScreenshot?.url)"
                    :preview-src-list="[resolveImageUrl(detail.infringementScreenshot?.url)]"
                    fit="cover"
                    preview-teleported
                    class="h-28 w-28 rounded-lg border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60"
                  />
                  <div v-else>--</div>
                </div>
              </div>
            </div>
          </section>

          <section class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
            <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">卖点与描述</div>
            <div class="grid gap-4 xl:grid-cols-2">
              <div>
                <div class="mb-2 text-sm font-medium text-slate-900 dark:text-slate-100">Bullet Points</div>
                <ul v-if="detail.bulletPoints?.length" class="list-disc space-y-2 pl-5 text-sm text-slate-600 dark:text-slate-300">
                  <li v-for="bullet in detail.bulletPoints" :key="bullet">{{ bullet }}</li>
                </ul>
                <div v-else class="text-sm text-slate-500 dark:text-slate-400">暂无卖点</div>
              </div>
              <div>
                <div class="mb-2 text-sm font-medium text-slate-900 dark:text-slate-100">Description</div>
                <div class="whitespace-pre-wrap rounded-lg bg-slate-50 p-3 text-sm text-slate-600 dark:bg-slate-900/40 dark:text-slate-300">
                  {{ detail.descriptionText || '--' }}
                </div>
              </div>
            </div>
            <div class="mt-4">
              <div class="mb-2 text-sm font-medium text-slate-900 dark:text-slate-100">A+ HTML</div>
              <div class="max-h-64 overflow-auto rounded-lg bg-slate-50 p-3 text-xs text-slate-600 dark:bg-slate-900/40 dark:text-slate-300">
                <pre class="whitespace-pre-wrap break-words">{{ detail.aplusHtml || '--' }}</pre>
              </div>
            </div>
          </section>

          <section class="grid gap-4 xl:grid-cols-2">
            <div class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
              <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">属性参数</div>
              <el-empty v-if="!attributeEntries.length" description="暂无属性参数" />
              <div v-else class="grid gap-3 md:grid-cols-2">
                <div
                  v-for="[key, value] in attributeEntries"
                  :key="key"
                  class="rounded-lg bg-slate-50 p-3 text-sm dark:bg-slate-900/40"
                >
                  <div class="font-medium text-slate-900 dark:text-slate-100">{{ key }}</div>
                  <div class="mt-1 whitespace-pre-wrap break-words text-slate-600 dark:text-slate-300">{{ formatObjectValue(value) }}</div>
                </div>
              </div>
            </div>

            <div class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
              <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">变体概览</div>
              <el-empty v-if="!variantEntries.length" description="暂无变体信息" />
              <div v-else class="grid gap-3 md:grid-cols-2">
                <div
                  v-for="[key, value] in variantEntries"
                  :key="key"
                  class="rounded-lg bg-slate-50 p-3 text-sm dark:bg-slate-900/40"
                >
                  <div class="font-medium text-slate-900 dark:text-slate-100">{{ key }}</div>
                  <div class="mt-1 whitespace-pre-wrap break-words text-slate-600 dark:text-slate-300">{{ formatObjectValue(value) }}</div>
                </div>
              </div>
            </div>
          </section>

          <section class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
            <div class="mb-3 flex items-center justify-between">
              <div class="text-base font-semibold text-slate-900 dark:text-slate-100">全部图片</div>
              <span class="text-sm text-slate-500 dark:text-slate-400">{{ detail.images?.length || 0 }} 张</span>
            </div>
            <el-empty v-if="!detail.images?.length" description="暂无图片" />
            <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
              <div
                v-for="image in detail.images"
                :key="image.id"
                class="rounded-xl border border-slate-200 p-3 dark:border-slate-700"
              >
                <el-image
                  v-if="resolveImageUrl(image.file?.url || image.originalUrl)"
                  :src="resolveImageUrl(image.file?.url || image.originalUrl)"
                  :preview-src-list="previewImageList"
                  fit="cover"
                  preview-teleported
                  class="h-40 w-full rounded-lg bg-slate-50 dark:bg-slate-900/40"
                />
                <div class="mt-3 space-y-1 text-sm text-slate-600 dark:text-slate-300">
                  <div class="flex items-center gap-2">
                    <el-tag v-if="image.isMain" size="small" type="success">主图</el-tag>
                    <el-tag size="small" :type="getCollectStatusType(image.materialStatus)">{{ getMaterialStatusLabel(image.materialStatus) }}</el-tag>
                  </div>
                  <div class="break-all text-xs text-slate-500 dark:text-slate-400">{{ image.originalUrl || '--' }}</div>
                  <div v-if="image.materialError" class="text-xs text-red-500">{{ image.materialError }}</div>
                </div>
              </div>
            </div>
          </section>

          <section class="rounded-lg border border-slate-200 p-5 dark:border-slate-700">
            <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">原始 JSON</div>
            <pre class="max-h-[520px] overflow-auto rounded-xl bg-slate-50 p-4 text-xs text-slate-700 dark:bg-slate-900/40 dark:text-slate-300">{{ formattedRawPayload }}</pre>
          </section>
        </div>
      </template>
    </el-drawer>

    <el-drawer v-model="riskDrawerVisible" title="编辑采集商品" size="520px" destroy-on-close>
      <template v-if="riskForm.id">
        <div class="flex flex-col gap-5">
          <div class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
            <div class="flex gap-4">
              <el-image
                v-if="resolveImageUrl(riskForm.mainImageUrl)"
                :src="resolveImageUrl(riskForm.mainImageUrl)"
                fit="cover"
                class="h-24 w-24 rounded-xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60"
              />
              <div class="min-w-0 flex-1 space-y-2">
                <div class="text-base font-semibold text-slate-900 dark:text-slate-100">{{ riskForm.title || '--' }}</div>
                <div class="flex flex-wrap gap-2 text-sm text-slate-500 dark:text-slate-400">
                  <el-tag size="small">{{ riskForm.siteCode || '--' }}</el-tag>
                  <el-button
                    v-if="riskForm.productUrl && riskForm.asin"
                    type="primary"
                    link
                    class="!h-auto !min-h-0 !p-0"
                    @click="openProductUrl(riskForm.productUrl)"
                  >
                    ASIN {{ riskForm.asin }}
                  </el-button>
                  <span v-else>ASIN {{ riskForm.asin || '--' }}</span>
                </div>
              </div>
            </div>
          </div>

          <el-form label-width="96px">
            <el-form-item label="图片状态">
              <el-tag :type="getCollectStatusType(riskForm.collectStatus)">{{ getCollectStatusLabel(riskForm.collectStatus) }}</el-tag>
            </el-form-item>
            <el-form-item label="是否侵权">
              <el-radio-group v-model="riskForm.infringementStatus">
                <el-radio-button label="unknown">未知侵权</el-radio-button>
                <el-radio-button label="infringed">已侵权</el-radio-button>
                <el-radio-button label="clear">未侵权</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="侵权截图">
              <div class="flex flex-col gap-3">
                <div class="flex items-center gap-3">
                  <el-upload
                    :show-file-list="false"
                    accept="image/*"
                    :http-request="uploadRiskScreenshot"
                  >
                    <el-button type="primary" plain :loading="riskUploading">上传截图</el-button>
                  </el-upload>
                  <el-button
                    v-if="riskForm.infringementScreenshotFileId"
                    type="danger"
                    plain
                    @click="clearRiskScreenshot"
                  >
                    移除截图
                  </el-button>
                </div>
                <el-image
                  v-if="resolveImageUrl(riskForm.infringementScreenshot?.url)"
                  :src="resolveImageUrl(riskForm.infringementScreenshot?.url)"
                  :preview-src-list="[resolveImageUrl(riskForm.infringementScreenshot?.url)]"
                  fit="cover"
                  preview-teleported
                  class="h-36 w-36 rounded-xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60"
                />
                <div v-else class="text-sm text-slate-500 dark:text-slate-400">未上传截图</div>
              </div>
            </el-form-item>
          </el-form>

          <div class="flex justify-end gap-3">
            <el-button @click="riskDrawerVisible = false">取消</el-button>
            <el-button type="primary" :loading="riskSaving" @click="saveRisk">保存</el-button>
          </div>
        </div>
      </template>
    </el-drawer>

    <el-dialog v-model="syncDialogVisible" title="同步到商品上架管理" width="560px" destroy-on-close>
      <el-form label-width="96px">
        <el-form-item label="采集商品">
          <div class="text-sm text-slate-600 dark:text-slate-300">
            {{ syncDialogRow?.title || syncDialogRow?.asin || '--' }}
          </div>
        </el-form-item>
        <el-form-item label="店铺">
          <el-select v-model="syncForm.storeId" filterable clearable placeholder="请选择店铺" class="w-full">
            <el-option
              v-for="store in storeOptions"
              :key="store.id"
              :label="`${store.storeName} / ${store.authStatus === 'authorized' ? '已授权' : '未授权'}`"
              :value="store.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="模板">
          <el-select v-model="syncForm.templateId" filterable clearable placeholder="请选择模板" class="w-full">
            <el-option
              v-for="template in availableTemplateOptions"
              :key="template.id"
              :label="`${template.name} / ${template.siteCode} / ${template.productType}`"
              :value="template.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="syncDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="syncSaving" @click="syncToListing">同步</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getUrl } from '@/utils/image'
import { uploadFile } from '@/api/fileUploadAndDownload'
import {
  deleteAmazonCollectedProduct,
  downloadAmazonCollectorExtension,
  findAmazonCollectedProduct,
  getAmazonCollectedProductCategories,
  getAmazonCollectedProductList,
  rebindAmazonCollectedProductImages,
  syncAmazonCollectedProductToListing,
  updateAmazonCollectedProductRisk
} from '@/api/amazonCollector'
import { getAmazonStoreList } from '@/api/amazonStore'
import { getAmazonTemplateList } from '@/api/amazonTemplate'
import { normalizeBlobResponse, triggerBlobDownload } from '@/utils/blobDownload'

const tableData = ref([])
const total = ref(0)
const drawerVisible = ref(false)
const riskDrawerVisible = ref(false)
const syncDialogVisible = ref(false)
const detail = ref(null)
const detailLoadingId = ref(0)
const rebindLoadingId = ref(0)
const riskSaving = ref(false)
const riskUploading = ref(false)
const syncSaving = ref(false)
const categoryOptions = ref([])
const storeOptions = ref([])
const templateOptions = ref([])
const syncDialogRow = ref(null)

const syncForm = ref({
  storeId: undefined,
  templateId: undefined
})

const createRiskForm = () => ({
  id: 0,
  title: '',
  asin: '',
  siteCode: '',
  productUrl: '',
  mainImageUrl: '',
  collectStatus: '',
  infringementStatus: 'unknown',
  infringementScreenshotFileId: null,
  infringementScreenshot: null
})

const riskForm = ref(createRiskForm())

const searchInfo = ref({
  page: 1,
  pageSize: 10,
  keyword: '',
  siteCode: '',
  collectStatus: '',
  brand: '',
  categoryLeaf: '',
  categoryKeyword: ''
})

const fetchTable = async () => {
  const res = await getAmazonCollectedProductList(searchInfo.value)
  if (res.code === 0) {
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
    searchInfo.value.page = res.data.page || searchInfo.value.page
    searchInfo.value.pageSize = res.data.pageSize || searchInfo.value.pageSize
  }
}

const resetSearch = () => {
  searchInfo.value = {
    page: 1,
    pageSize: 10,
    keyword: '',
    siteCode: '',
    collectStatus: '',
    brand: '',
    categoryLeaf: '',
    categoryKeyword: ''
  }
  fetchCategoryOptions()
  fetchTable()
}

const handleCurrentChange = (page) => {
  searchInfo.value.page = page
  fetchTable()
}

const handleSizeChange = (pageSize) => {
  searchInfo.value.pageSize = pageSize
  searchInfo.value.page = 1
  fetchTable()
}

const fetchCollectorDetail = async (id) => {
  const res = await findAmazonCollectedProduct({ id })
  if (res.code !== 0) {
    throw new Error(res.msg || '加载详情失败')
  }
  return res.data
}

const openDetail = async (row) => {
  detailLoadingId.value = row.id
  try {
    detail.value = await fetchCollectorDetail(row.id)
    drawerVisible.value = true
  } finally {
    detailLoadingId.value = 0
  }
}

const handleCollectorRowAction = async (command, row) => {
  switch (command) {
    case 'sync':
      await openSyncDialog(row)
      return
    case 'risk':
      await openRiskEditor(row)
      return
    case 'rebind':
      await rebindImages(row)
      return
    case 'delete':
      await removeRow(row)
      return
    default:
      return
  }
}

const openRiskEditor = async (row) => {
  detailLoadingId.value = row.id
  try {
    const data = await fetchCollectorDetail(row.id)
    riskForm.value = {
      id: data.id,
      title: data.title || '',
      asin: data.asin || '',
      siteCode: data.siteCode || '',
      productUrl: data.productUrl || '',
      mainImageUrl: data.mainImageUrl || '',
      collectStatus: data.collectStatus || '',
      infringementStatus: data.infringementStatus || 'unknown',
      infringementScreenshotFileId: data.infringementScreenshotFileId || null,
      infringementScreenshot: data.infringementScreenshot || null
    }
    riskDrawerVisible.value = true
  } finally {
    detailLoadingId.value = 0
  }
}

const removeRow = async (row) => {
  await ElMessageBox.confirm(`确认删除采集商品 ${row.asin || row.title || row.id} 吗？`, '删除确认', {
    type: 'warning'
  })
  const res = await deleteAmazonCollectedProduct({ id: row.id })
  if (res.code === 0) {
    ElMessage.success('删除成功')
    if (detail.value?.id === row.id) {
      drawerVisible.value = false
      detail.value = null
    }
    fetchTable()
  }
}

const rebindImages = async (row) => {
  rebindLoadingId.value = row.id
  try {
    const res = await rebindAmazonCollectedProductImages({ id: row.id })
    if (res.code === 0) {
      ElMessage.success('图片入库已重试')
      if (detail.value?.id === row.id) {
        detail.value = res.data
      }
      fetchTable()
    }
  } finally {
    rebindLoadingId.value = 0
  }
}

const uploadRiskScreenshot = async (options) => {
  riskUploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', options.file)
    const res = await uploadFile(formData)
    const file = res?.data?.file
    const fileId = Number(file?.id || file?.ID || 0)
    const fileUrl = String(file?.url || file?.Url || '').trim()
    const fileName = String(file?.name || file?.Name || options.file?.name || '').trim()
    const fileKey = String(file?.key || file?.Key || '').trim()
    if (res.code === 0 && fileId > 0) {
      riskForm.value.infringementScreenshotFileId = fileId
      riskForm.value.infringementScreenshot = {
        id: fileId,
        name: fileName,
        url: fileUrl,
        key: fileKey
      }
      options.onSuccess?.(riskForm.value.infringementScreenshot)
      ElMessage.success('截图上传成功')
      return
    }
    throw new Error(res.msg || '截图上传失败')
  } catch (error) {
    options.onError?.(error)
    ElMessage.error(error.message || '截图上传失败')
  } finally {
    riskUploading.value = false
  }
}

const clearRiskScreenshot = () => {
  riskForm.value.infringementScreenshotFileId = null
  riskForm.value.infringementScreenshot = null
}

const fetchCategoryOptions = async () => {
  const res = await getAmazonCollectedProductCategories({
    siteCode: searchInfo.value.siteCode || undefined
  })
  if (res.code === 0) {
    categoryOptions.value = res.data || []
  }
}

const fetchStoreOptions = async () => {
  const res = await getAmazonStoreList({
    page: 1,
    pageSize: 200,
    isEnabled: true
  })
  if (res.code === 0) {
    storeOptions.value = res.data.list || []
  }
}

const fetchTemplateOptions = async () => {
  const res = await getAmazonTemplateList({
    page: 1,
    pageSize: 200
  })
  if (res.code === 0) {
    templateOptions.value = res.data.list || []
  }
}

const downloadCollectorExtension = async () => {
  const res = await downloadAmazonCollectorExtension()
  const { blob, fileName } = await normalizeBlobResponse(res, 'amazon-collector-latest.zip')
  triggerBlobDownload(blob, fileName)
  ElMessage.success('采集助手下载成功')
}

const openSyncDialog = async (row) => {
  syncDialogRow.value = row
  syncForm.value = {
    storeId: storeOptions.value.find((item) => item.authStatus === 'authorized')?.id,
    templateId: availableTemplateOptions.value[0]?.id
  }
  syncDialogVisible.value = true
}

const syncToListing = async () => {
  if (!syncDialogRow.value?.id) {
    return
  }
  if (!syncForm.value.storeId) {
    ElMessage.warning('请选择店铺')
    return
  }
  if (!syncForm.value.templateId) {
    ElMessage.warning('请选择模板')
    return
  }
  syncSaving.value = true
  try {
    const res = await syncAmazonCollectedProductToListing({
      id: syncDialogRow.value.id,
      storeId: syncForm.value.storeId,
      templateId: syncForm.value.templateId
    })
    if (res.code === 0) {
      ElMessage.success(`已同步到商品上架管理，商品组ID：${res.data.familyId}`)
      syncDialogVisible.value = false
      fetchTable()
    }
  } finally {
    syncSaving.value = false
  }
}

const saveRisk = async () => {
  if (!riskForm.value.id) {
    return
  }
  riskSaving.value = true
  try {
    const res = await updateAmazonCollectedProductRisk({
      id: riskForm.value.id,
      infringementStatus: riskForm.value.infringementStatus,
      infringementScreenshotFileId: riskForm.value.infringementScreenshotFileId
    })
    if (res.code === 0) {
      riskForm.value.infringementStatus = res.data.infringementStatus || 'unknown'
      riskForm.value.infringementScreenshotFileId = res.data.infringementScreenshotFileId || null
      riskForm.value.infringementScreenshot = res.data.infringementScreenshot || null
      if (detail.value?.id === res.data.id) {
        detail.value = res.data
      }
      riskDrawerVisible.value = false
      ElMessage.success('侵权状态已保存')
      fetchTable()
    }
  } finally {
    riskSaving.value = false
  }
}

const resolveImageUrl = (url) => {
  const formatted = getUrl(String(url || '').trim())
  return formatted || ''
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
      return '采集图片成功'
    case 'warning':
      return '采集图片告警'
    case 'failed':
      return '采集图片失败'
    default:
      return '未知状态'
  }
}

const getInfringementStatusType = (status) => {
  switch (status) {
    case 'infringed':
      return 'danger'
    case 'clear':
      return 'success'
    default:
      return 'info'
  }
}

const getInfringementStatusLabel = (status) => {
  switch (status) {
    case 'infringed':
      return '已侵权'
    case 'clear':
      return '未侵权'
    default:
      return '未知侵权'
  }
}

const getMaterialStatusLabel = (status) => {
  switch (status) {
    case 'success':
      return '已入库'
    case 'failed':
      return '入库失败'
    case 'pending':
      return '待处理'
    default:
      return '未知'
  }
}

const formatPrice = (price, currencyCode) => {
  if (price === null || typeof price === 'undefined') {
    return '--'
  }
  return `${currencyCode || ''} ${Number(price).toFixed(2)}`.trim()
}

const formatRating = (rating) => {
  if (rating === null || typeof rating === 'undefined') {
    return '--'
  }
  return `${Number(rating).toFixed(1)} / 5`
}

const formatReviewCount = (count) => {
  if (count === null || typeof count === 'undefined') {
    return '无评论数'
  }
  return `${count} 条评论`
}

const formatDateTime = (value) => {
  if (!value) {
    return '--'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString('zh-CN', { hour12: false })
}

const openProductUrl = (productUrl) => {
  if (!productUrl) {
    return
  }
  window.open(productUrl, '_blank', 'noopener,noreferrer')
}

const formatObjectValue = (value) => {
  if (Array.isArray(value)) {
    return value.map((item) => sanitizeCollectorText(item)).join(' / ')
  }
  if (value && typeof value === 'object') {
    return JSON.stringify(value, null, 2)
  }
  if (value === null || typeof value === 'undefined' || value === '') {
    return '--'
  }
  return sanitizeCollectorText(String(value))
}

const sanitizeCollectorText = (value) => {
  const raw = String(value || '').trim()
  if (!raw) {
    return '--'
  }
  const withoutScriptNoise = raw
    .replace(/\s+(?:var\s+[A-Za-z_$][\w$]*\s*;|P\.when\(|ue\.count\().*$/is, '')
    .replace(/^(\d+(?:\.\d+)?)\s+\1(\s+out of 5 stars(?:\s*\([^)]*\))?.*)$/i, '$1$2')
    .replace(/\s+/g, ' ')
    .trim()
  return withoutScriptNoise || '--'
}

const attributeEntries = computed(() => Object.entries(detail.value?.specAttributes || {}))
const variantEntries = computed(() => Object.entries(detail.value?.variantSummary || {}))
const previewImageList = computed(() => {
  const list = (detail.value?.images || [])
    .map((image) => resolveImageUrl(image.file?.url || image.originalUrl))
    .filter(Boolean)
  return [...new Set(list)]
})
const formattedRawPayload = computed(() => JSON.stringify(detail.value?.rawPayload || {}, null, 2))
const availableTemplateOptions = computed(() => {
  const siteCode = String(syncDialogRow.value?.siteCode || '').trim().toUpperCase()
  if (!siteCode) {
    return templateOptions.value
  }
  return templateOptions.value.filter((item) => String(item.siteCode || '').trim().toUpperCase() === siteCode)
})

watch(
  () => searchInfo.value.siteCode,
  () => {
    fetchCategoryOptions()
  }
)

onMounted(() => {
  fetchCategoryOptions()
  fetchStoreOptions()
  fetchTemplateOptions()
  fetchTable()
})
</script>
