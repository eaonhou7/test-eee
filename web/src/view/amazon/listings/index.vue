<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON 上架管理</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">商品上架管理</h1>
          <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
            管理 Amazon 列表页和详情页所需的父子变体、北美三站价格库存、多语言文案与图片资源，并按模板直接导出可上传 Excel。
          </p>
        </div>
        <div class="gva-btn-list !mb-0">
          <el-button type="warning" plain :disabled="!selectedRows.length" @click="openSelectedRuiguanQuery">睿观查侵权</el-button>
          <el-button type="primary" @click="openCreateDrawer">新建商品</el-button>
          <el-button type="primary" plain :disabled="!selectedRows.length" @click="openListingSyncDialog">批量回传价格库存</el-button>
          <el-button @click="openListingSyncJobs">回传任务</el-button>
          <el-button :disabled="!selectedRows.length" @click="validateSelectedRows">批量校验</el-button>
          <el-button type="success" :disabled="!selectedRows.length" @click="exportSelectedRows">导出选中</el-button>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchTree">
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="SKU / 商品组 / 产品类型" />
        </el-form-item>
        <el-form-item label="站点">
          <el-select v-model="searchInfo.siteCode" clearable class="!w-32">
            <el-option v-for="site in amazonListingSiteOptions" :key="site.value" :label="site.label" :value="site.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchInfo.status" clearable class="!w-36">
            <el-option label="草稿" value="draft" />
            <el-option label="启用" value="active" />
            <el-option label="归档" value="archived" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchTree">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table
        :data="treeData"
        row-key="id"
        stripe
        default-expand-all
        :tree-props="{ children: 'children' }"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="52" />
        <el-table-column label="商品 / 变体" min-width="240">
          <template #default="{ row }">
            <div class="flex flex-col">
              <span class="font-medium text-slate-900 dark:text-slate-100">{{ row.label }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400">{{ getProductTypeLabel(row.productType) || '--' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="sku" label="SKU" min-width="160" />
        <el-table-column label="主图" width="140">
          <template #default="{ row }">
            <div class="flex items-center justify-center">
              <el-image
                v-if="resolveImageUrl(row.mainImageUrl)"
                :src="resolveImageUrl(row.mainImageUrl)"
                :preview-src-list="[resolveImageUrl(row.mainImageUrl)]"
                fit="cover"
                preview-teleported
                class="h-16 w-16 rounded-lg border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-900/60"
              />
              <div
                v-else
                class="flex h-16 w-16 items-center justify-center rounded-lg border border-dashed border-slate-300 text-xs text-slate-400 dark:border-slate-700 dark:text-slate-500"
              >
                无主图
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="利润摘要" min-width="220">
          <template #default="{ row }">
            <div v-if="hasProfitSummary(row)" class="flex flex-col gap-1">
              <div class="flex items-center gap-2">
                <el-tag size="small" :type="getProfitStatusTagType(row.profitStatus)">
                  {{ `${row.profitSummarySiteCode || '--'} ${getProfitModeLabel(row.profitSummaryMode) || '未试算'}` }}
                </el-tag>
              </div>
              <span class="text-sm font-medium" :class="getProfitStatusTextClass(row.profitStatus)">
                {{ formatProfitMoney(row.profitNetProfitCny) }}
              </span>
              <span class="text-xs text-slate-500 dark:text-slate-400">净利率 {{ formatProfitPercent(row.profitNetMarginRate) }}</span>
            </div>
            <span v-else class="text-xs text-slate-400 dark:text-slate-500">未试算</span>
          </template>
        </el-table-column>
        <el-table-column label="角色" width="120">
          <template #default="{ row }">{{ getRoleLabel(row.role) }}</template>
        </el-table-column>
        <el-table-column label="变体主题" min-width="160">
          <template #default="{ row }">{{ getVariationThemeLabel(row.variationTheme) || '--' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">{{ getStatusLabel(row.status) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <div class="flex items-center gap-2">
              <el-button type="primary" link @click="openEditDrawer(row)">编辑</el-button>
              <el-dropdown trigger="click" @command="(command) => handleRowAction(command, row)">
                <el-button type="primary" link>更多</el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="ruiguan">睿观查询</el-dropdown-item>
                    <el-dropdown-item command="collect1688" :disabled="!resolveImageUrl(row.mainImageUrl)">货物采集</el-dropdown-item>
                    <el-dropdown-item command="publish">发布到 Amazon</el-dropdown-item>
                    <el-dropdown-item command="validate">校验</el-dropdown-item>
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

    <el-drawer
      v-model="drawerVisible"
      size="92%"
      destroy-on-close
      :show-close="false"
      :before-close="closeDrawer"
    >
      <template #header>
        <div class="flex items-center justify-between">
          <div>
            <div class="text-lg font-semibold text-slate-900 dark:text-slate-100">{{ formMode === 'create' ? '新建商品族' : '编辑商品族' }}</div>
            <div class="text-sm text-slate-500 dark:text-slate-400">支持独立款、父体、子体结构，统一维护站点价格库存、详情页文案、图片和模板映射。</div>
          </div>
          <div class="flex gap-3">
            <el-button @click="downloadHomeTemplate()">下载家居默认模板</el-button>
            <el-button @click="closeDrawer">取消</el-button>
            <el-button @click="validateDraft">校验草稿</el-button>
            <el-button type="primary" :loading="saveLoading" @click="saveDraft">保存</el-button>
          </div>
        </div>
      </template>

      <div class="flex flex-col gap-6 pr-2 pt-1">
        <section class="rounded-lg border border-slate-200 bg-slate-50 p-5 dark:border-slate-700 dark:bg-slate-800/60">
          <div class="mb-2 text-base font-semibold text-slate-900 dark:text-slate-100">商品组信息</div>
          <div class="mb-4 grid gap-2 text-sm text-slate-600 dark:text-slate-300 md:grid-cols-2 xl:grid-cols-4">
            <div>商品组名称：用于 Amazon 列表页后台管理识别同一组商品。</div>
            <div>产品类型：决定详情页属性字段、模板列头和导出规则。</div>
            <div>变体主题：定义父子体按颜色、尺寸等哪个维度切换。</div>
            <div>父 SKU：子体关联父体时使用；独立款可留空。</div>
          </div>
          <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <el-input v-model="formModel.family.familyName" placeholder="请输入商品组名称" />
            <el-select
              v-model="formModel.family.productType"
              filterable
              allow-create
              default-first-option
              clearable
              placeholder="请选择或输入产品类型"
            >
              <el-option v-for="option in productTypeOptions" :key="option" :label="getProductTypeLabel(option)" :value="option" />
            </el-select>
            <el-select
              v-model="formModel.family.variationTheme"
              filterable
              allow-create
              default-first-option
              clearable
              placeholder="请选择变体主题"
            >
              <el-option v-for="option in variationThemeOptions" :key="option" :label="getVariationThemeLabel(option)" :value="option" />
            </el-select>
            <el-input v-model="formModel.family.parentSku" placeholder="请输入父 SKU（可自动回填）" />
            <el-select v-model="formModel.family.status" placeholder="状态">
              <el-option label="草稿" value="draft" />
              <el-option label="启用" value="active" />
              <el-option label="归档" value="archived" />
            </el-select>
          </div>
          <el-input v-model="formModel.family.remark" type="textarea" :rows="2" class="mt-4" placeholder="备注" />
        </section>

        <section class="gva-btn-list !mb-0">
          <el-button @click="addItem('standalone')">新增独立款</el-button>
          <el-button @click="addItem('parent')">新增父体</el-button>
          <el-button @click="addItem('child')">新增子体</el-button>
        </section>

        <el-empty v-if="!formModel.items.length" description="还没有商品，请先新增独立款、父体或子体" />

        <el-collapse v-model="activePanels">
          <el-collapse-item
            v-for="(item, itemIndex) in formModel.items"
            :key="item.__key"
            :name="item.__key"
          >
            <template #title>
              <div class="flex min-w-0 flex-col">
                <span class="font-medium text-slate-900 dark:text-slate-100">{{ item.sku || `未命名${getRoleLabel(item.role)}` }}</span>
                <span class="text-xs tracking-[0.2em] text-slate-500 dark:text-slate-400">{{ getRoleLabel(item.role) }}</span>
              </div>
            </template>

            <div class="flex flex-col gap-5 pt-2">
              <div class="grid gap-2 text-sm text-slate-600 dark:text-slate-300 md:grid-cols-2 xl:grid-cols-4">
                <div>商品角色：决定是 Amazon 独立款、父体还是子体。</div>
                <div>SKU：用于列表识别、库存管理和 Excel 导出唯一定位。</div>
                <div>品牌/商品状况：对应 Amazon 详情页基础信息。</div>
                <div>外部商品编码：对应 UPC、EAN 等官方识别码。</div>
              </div>
              <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                <el-select v-model="item.role" placeholder="角色">
                  <el-option label="独立款" value="standalone" />
                  <el-option label="父体" value="parent" />
                  <el-option label="子体" value="child" />
                </el-select>
                <el-input v-model="item.sku" placeholder="请输入 SKU" />
                <el-input v-model="item.brand" placeholder="请输入品牌名称" />
                <el-select v-model="item.conditionType" filterable allow-create default-first-option clearable placeholder="请选择商品状况">
                  <el-option v-for="option in conditionTypeOptions" :key="option" :label="getConditionTypeLabel(option)" :value="option" />
                </el-select>
                <el-select v-model="item.externalProductIdType" filterable clearable placeholder="请选择外部商品编码类型">
                  <el-option v-for="option in externalProductIdTypeOptions" :key="option" :label="getExternalProductIdTypeLabel(option)" :value="option" />
                </el-select>
                <el-input v-model="item.externalProductId" placeholder="请输入外部商品编码" />
                <el-input v-model="item.merchantSuggestedAsin" placeholder="请输入建议 ASIN（可选）" />
                <el-select v-model="item.status" placeholder="状态">
                  <el-option label="草稿" value="draft" />
                  <el-option label="启用" value="active" />
                  <el-option label="归档" value="archived" />
                </el-select>
              </section>

              <div class="flex justify-end">
                <el-button type="danger" link @click="removeItem(itemIndex)">删除当前商品</el-button>
              </div>

              <el-tabs>
                <el-tab-pane label="基础信息">
                  <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                    <div v-for="field in getCommonFields(item)" :key="`${item.__key}-${field.fieldKey}`">
                      <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">{{ getFieldDisplayLabel(field) }}</div>
                      <div v-if="getFieldHelpText(field)" class="mb-2 text-xs text-slate-500 dark:text-slate-400">{{ getFieldHelpText(field) }}</div>
                      <dynamic-field-input
                        v-model="item.commonAttributes[field.fieldKey]"
                        :field="field"
                        :fallback-options="getFieldOptions(field)"
                      />
                    </div>
                  </div>
                </el-tab-pane>

                <el-tab-pane label="变体字段">
                  <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                    <div v-for="field in getVariationFields(item)" :key="`${item.__key}-${field.fieldKey}`">
                      <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">{{ getFieldDisplayLabel(field) }}</div>
                      <div v-if="getFieldHelpText(field)" class="mb-2 text-xs text-slate-500 dark:text-slate-400">{{ getFieldHelpText(field) }}</div>
                      <dynamic-field-input
                        v-model="item.variationAttributes[field.fieldKey]"
                        :field="field"
                        :fallback-options="getFieldOptions(field)"
                      />
                    </div>
                  </div>
                </el-tab-pane>

                <el-tab-pane label="共享图片">
                  <div class="mb-3 text-sm text-slate-600 dark:text-slate-300">共享图片会优先用在 Amazon 列表页主图、详情页图集，以及 1688 图搜选款入口。</div>
                  <image-asset-editor v-model="item.sharedImages" title="共享图片" />
                </el-tab-pane>

                <el-tab-pane label="站点信息">
                  <div class="mb-4 flex justify-end">
                    <el-button size="small" @click="addMarketplace(item)">新增站点</el-button>
                  </div>

                  <div
                    v-for="(binding, bindingIndex) in item.marketplaces"
                    :key="binding.__key"
                    class="mb-6 rounded-lg border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900/60"
                  >
                    <div class="mb-4 flex items-center justify-between">
                      <div>
                        <div class="text-base font-semibold text-slate-900 dark:text-slate-100">{{ getSiteBindingTitle(binding.siteCode) }}</div>
                        <div class="text-xs text-slate-500 dark:text-slate-400">模板决定导出列和校验规则；价格、库存和语言内容只作用于当前站点。</div>
                      </div>
                      <div class="flex items-center gap-3">
                        <el-button size="small" @click="downloadSelectedTemplate(binding)">{{ binding.templateId ? '下载所选模板' : '下载家居默认模板' }}</el-button>
                        <el-button type="danger" link @click="removeMarketplace(item, bindingIndex)">删除站点</el-button>
                      </div>
                    </div>

                    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                      <el-select v-model="binding.storeId" filterable clearable placeholder="请选择发布店铺">
                        <el-option
                          v-for="store in storeOptions"
                          :key="store.id"
                          :label="getStoreOptionLabel(store)"
                          :value="store.id"
                        />
                      </el-select>
                      <el-select v-model="binding.templateId" filterable placeholder="请选择模板" @change="onTemplateChange(binding)">
                        <el-option
                          v-for="template in templateOptions"
                          :key="template.id"
                          :label="getTemplateOptionLabel(template)"
                          :value="template.id"
                        />
                      </el-select>
                      <el-input v-model="binding.marketplaceId" placeholder="请输入站点标识（Marketplace ID）" />
                      <el-select v-model="binding.siteCode" placeholder="站点" @change="onMarketplaceSiteChange(binding)">
                        <el-option v-for="site in amazonListingSiteOptions" :key="site.value" :label="site.label" :value="site.value" />
                      </el-select>
                      <el-input v-model="binding.currencyCode" placeholder="请输入站点币种，例如 USD" @change="onMarketplaceCurrencyChange(binding)" />
                      <el-input-number v-model="binding.offerPrice" :min="0" :precision="2" class="!w-full" placeholder="售价" />
                      <el-input-number v-model="binding.salePrice" :min="0" :precision="2" class="!w-full" placeholder="促销价" />
                      <el-input-number v-model="binding.quantity" :min="0" class="!w-full" placeholder="库存数量" />
                      <el-input-number v-model="binding.leadTimeToShip" :min="0" class="!w-full" placeholder="备货天数" />
                    </div>

                    <el-input
                      v-model="binding.merchantShippingGroup"
                      class="mt-4"
                      placeholder="请输入配送模板名称"
                    />

                    <div class="mt-4 grid gap-4 md:grid-cols-2">
                      <div class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
                        <div class="mb-2 text-sm font-medium text-slate-900 dark:text-slate-100">最近价格库存回传</div>
                        <div class="space-y-1 text-sm text-slate-600 dark:text-slate-300">
                          <div>状态：{{ getListingSyncStatusLabel(binding.lastPriceInventorySyncStatus) }}</div>
                          <div>时间：{{ binding.lastPriceInventorySyncAt || '--' }}</div>
                          <div>消息：{{ binding.lastPriceInventorySyncMessage || '--' }}</div>
                        </div>
                      </div>
                      <div class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
                        <div class="mb-2 text-sm font-medium text-slate-900 dark:text-slate-100">FBA 实际库存</div>
                        <div class="space-y-1 text-sm text-slate-600 dark:text-slate-300">
                          <div>可售：{{ binding.remoteFbaAvailableQuantity ?? '--' }}</div>
                          <div>预留：{{ binding.remoteFbaReservedQuantity ?? '--' }}</div>
                          <div>在途：{{ binding.remoteFbaInboundQuantity ?? '--' }}</div>
                          <div>同步：{{ binding.lastRemoteInventorySyncAt || '--' }}</div>
                          <div>异常：{{ binding.lastRemoteInventorySyncError || '--' }}</div>
                        </div>
                      </div>
                    </div>

                    <profit-trial-panel
                      v-model="binding.profitProfile"
                      class="mt-5"
                      :site-code="binding.siteCode"
                      :currency-code="binding.currencyCode"
                      :offer-price="binding.offerPrice"
                    />

                    <div class="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                      <div v-for="field in getMarketplaceFields(binding)" :key="`${binding.__key}-${field.fieldKey}`">
                        <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">{{ getFieldDisplayLabel(field) }}</div>
                        <div v-if="getFieldHelpText(field)" class="mb-2 text-xs text-slate-500 dark:text-slate-400">{{ getFieldHelpText(field) }}</div>
                        <dynamic-field-input
                          v-model="binding.marketplaceAttributes[field.fieldKey]"
                          :field="field"
                          :fallback-options="getFieldOptions(field)"
                        />
                      </div>
                    </div>

                    <div class="mt-6 rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
                      <div class="mb-3 flex items-center justify-between">
                        <div>
                          <div class="text-sm font-medium text-slate-800 dark:text-slate-100">语言内容</div>
                          <div class="text-xs text-slate-500 dark:text-slate-400">标题、卖点、描述和搜索词会写入 Amazon 详情页本地化文案。</div>
                        </div>
                        <el-button size="small" @click="addLocale(binding)">新增语言</el-button>
                      </div>

                      <div
                        v-for="(locale, localeIndex) in binding.locales"
                        :key="locale.__key"
                        class="mb-5 rounded-lg border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900/80"
                      >
                        <div class="mb-4 flex items-center justify-between">
                          <div class="flex items-center gap-3">
                            <el-select v-model="locale.localeCode" class="!w-36">
                              <el-option
                                v-for="option in localeOptions"
                                :key="option.value"
                                :label="option.label"
                                :value="option.value"
                                :disabled="isLocaleOptionDisabled(binding, locale, option.value)"
                              />
                            </el-select>
                            <span class="text-sm text-slate-500 dark:text-slate-400">本地化字段与模板字段联动</span>
                          </div>
                          <el-button type="danger" link @click="removeLocale(binding, localeIndex)">删除语言</el-button>
                        </div>

                        <div class="grid gap-4">
                          <el-input v-model="locale.itemName" placeholder="请输入商品标题（用于 Amazon 详情页标题）" />
                          <el-input
                            :model-value="locale.bulletPoints.join('\n')"
                            type="textarea"
                            :rows="4"
                            placeholder="每行一个卖点，展示在 Amazon 详情页卖点区"
                            @update:model-value="updateBulletPoints(locale, $event)"
                          />
                          <el-input v-model="locale.productDescription" type="textarea" :rows="4" placeholder="请输入详情描述，用于 Amazon 详情页长描述" />
                          <el-input
                            :model-value="locale.searchTerms.join('\n')"
                            type="textarea"
                            :rows="3"
                            placeholder="每行一个搜索关键词，用于 Amazon 站内搜索"
                            @update:model-value="updateSearchTerms(locale, $event)"
                          />
                        </div>

                        <div class="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                          <div v-for="field in getLocaleFields(binding, locale)" :key="`${locale.__key}-${field.fieldKey}`">
                            <div class="mb-1 text-sm text-slate-600 dark:text-slate-300">{{ getFieldDisplayLabel(field) }}</div>
                            <div v-if="getFieldHelpText(field)" class="mb-2 text-xs text-slate-500 dark:text-slate-400">{{ getFieldHelpText(field) }}</div>
                            <dynamic-field-input
                              v-model="locale.localizedAttributes[field.fieldKey]"
                              :field="field"
                              :fallback-options="getFieldOptions(field)"
                            />
                          </div>
                        </div>
                      </div>
                    </div>

                    <div class="mt-5">
                      <image-asset-editor v-model="binding.images" title="站点图片" />
                    </div>
                  </div>
                </el-tab-pane>
              </el-tabs>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>
    </el-drawer>

    <el-dialog v-model="publishDialogVisible" title="发布到 Amazon" width="760px" destroy-on-close>
      <div class="flex flex-col gap-5">
        <el-alert
          title="发布按商品组（family）提交到 Amazon，系统会先做预检，再生成 JSON_LISTINGS_FEED。"
          type="info"
          :closable="false"
          show-icon
        />

        <el-form label-width="96px">
          <el-form-item label="店铺">
            <el-select v-model="publishForm.storeId" filterable clearable class="w-full" placeholder="请选择已授权店铺" @change="loadPublishPreview">
              <el-option
                v-for="store in authorizedStoreOptions"
                :key="store.id"
                :label="getStoreOptionLabel(store)"
                :value="store.id"
              />
            </el-select>
          </el-form-item>
        </el-form>

        <div v-if="publishPreview" class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
          <div class="mb-3 flex items-center justify-between">
            <div class="text-base font-semibold text-slate-900 dark:text-slate-100">发布预检</div>
            <el-tag :type="publishPreview.valid ? 'success' : 'danger'">
              {{ publishPreview.valid ? '预检通过' : '预检未通过' }}
            </el-tag>
          </div>
          <div class="grid gap-3 text-sm text-slate-600 dark:text-slate-300 md:grid-cols-2">
            <div>店铺：{{ getStoreNameById(publishPreview.storeId) || '--' }}</div>
            <div>Feed 类型：{{ publishPreview.feedType || '--' }}</div>
            <div>站点：{{ (publishPreview.siteCodes || []).join(' / ') || '--' }}</div>
            <div>Marketplace ID：{{ (publishPreview.marketplaceIds || []).join(' / ') || '--' }}</div>
          </div>

          <div class="mt-4">
            <div class="mb-2 text-sm font-medium text-slate-800 dark:text-slate-100">预检结果</div>
            <el-empty v-if="!publishPreview.issues?.length" description="没有发现阻断问题" :image-size="64" />
            <div v-else class="flex flex-col gap-2">
              <el-alert
                v-for="(issue, index) in publishPreview.issues"
                :key="`${issue.level}-${index}`"
                :title="issue.message"
                :type="issue.level === 'error' ? 'error' : 'warning'"
                :closable="false"
                show-icon
              />
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="publishDialogVisible = false">取消</el-button>
        <el-button :loading="publishPreviewLoading" @click="loadPublishPreview">重新预检</el-button>
        <el-button
          type="primary"
          :loading="publishSubmitting"
          :disabled="!publishForm.storeId || !publishPreview?.valid"
          @click="submitPublishJob"
        >
          确认发布
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="listingSyncDialogVisible" title="批量回传价格库存" width="860px" destroy-on-close>
      <div class="flex flex-col gap-5">
        <el-alert
          title="价格库存回传按站点 SKU 粒度提交到 Amazon。FBM 库存统一回传 9999，FBA 只同步实际库存展示，不回传数量。"
          type="info"
          :closable="false"
          show-icon
        />

        <el-form label-width="110px">
          <el-form-item label="店铺">
            <el-select
              v-model="listingSyncForm.storeId"
              filterable
              clearable
              class="w-full"
              placeholder="请选择已授权店铺"
              @change="loadListingSyncPreview"
            >
              <el-option
                v-for="store in authorizedStoreOptions"
                :key="store.id"
                :label="getStoreOptionLabel(store)"
                :value="store.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="字段范围">
            <div class="flex flex-wrap gap-2">
              <el-tag v-for="scope in listingSyncFieldScopes" :key="scope" size="small" type="info">
                {{ getListingSyncFieldScopeLabel(scope) }}
              </el-tag>
            </div>
          </el-form-item>
        </el-form>

        <div v-if="listingSyncPreview" class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/60">
          <div class="mb-3 flex items-center justify-between">
            <div class="text-base font-semibold text-slate-900 dark:text-slate-100">回传预检</div>
            <el-tag :type="listingSyncPreview.valid ? 'success' : 'danger'">
              {{ listingSyncPreview.valid ? '预检通过' : '预检未通过' }}
            </el-tag>
          </div>
          <div class="grid gap-3 text-sm text-slate-600 dark:text-slate-300 md:grid-cols-2">
            <div>店铺：{{ getStoreNameById(listingSyncPreview.storeId) || '--' }}</div>
            <div>Feed 类型：{{ listingSyncPreview.feedType || '--' }}</div>
            <div>可回传记录：{{ listingSyncPreview.recordCount || 0 }}</div>
            <div>跳过记录：{{ listingSyncPreview.skippedCount || 0 }}</div>
            <div>站点：{{ (listingSyncPreview.siteCodes || []).join(' / ') || '--' }}</div>
            <div>Marketplace ID：{{ (listingSyncPreview.marketplaceIds || []).join(' / ') || '--' }}</div>
          </div>

          <div class="mt-4">
            <div class="mb-2 text-sm font-medium text-slate-800 dark:text-slate-100">预检问题</div>
            <el-empty v-if="!listingSyncPreview.issues?.length" description="没有发现阻断问题" :image-size="64" />
            <div v-else class="flex flex-col gap-2">
              <el-alert
                v-for="(issue, index) in listingSyncPreview.issues"
                :key="`${issue.level}-${index}`"
                :title="issue.message"
                :type="issue.level === 'error' ? 'error' : 'warning'"
                :closable="false"
                show-icon
              />
            </div>
          </div>

          <div class="mt-4">
            <div class="mb-2 text-sm font-medium text-slate-800 dark:text-slate-100">即将回传的站点 SKU</div>
            <el-table :data="listingSyncPreview.records || []" max-height="280" border>
              <el-table-column prop="sku" label="SKU" min-width="160" />
              <el-table-column prop="siteCode" label="站点" width="100" />
              <el-table-column label="履约" width="100">
                <template #default="{ row }">{{ getProfitModeLabel(row.fulfillmentMode) || '--' }}</template>
              </el-table-column>
              <el-table-column label="价格 / 库存" min-width="180">
                <template #default="{ row }">
                  {{ row.pushedOfferPrice ?? '--' }} / {{ row.pushedQuantity ?? '--' }}
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="listingSyncDialogVisible = false">取消</el-button>
        <el-button :loading="listingSyncPreviewLoading" @click="loadListingSyncPreview">重新预检</el-button>
        <el-button
          type="primary"
          :loading="listingSyncSubmitting"
          :disabled="!listingSyncForm.storeId || !listingSyncPreview?.valid"
          @click="submitListingSyncJob"
        >
          确认回传
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useRouter } from 'vue-router'

  import {
    deleteAmazonListing,
    exportAmazonListingSelected,
    findAmazonListing,
    getAmazonListingTree,
    saveAmazonListing,
    validateAmazonListingItem,
    validateAmazonListingSelected
  } from '@/api/amazonListing'
  import { createAmazon1688CollectTask } from '@/api/amazon1688Collector'
  import { previewAmazonListingPublish, submitAmazonListingPublish } from '@/api/amazonListingPublish'
  import { previewAmazonListingSync, submitAmazonListingSync } from '@/api/amazonListingSync'
  import { getAmazonStoreList } from '@/api/amazonStore'
  import { downloadAmazonTemplateWorkbook, findAmazonTemplate, getAmazonTemplateList } from '@/api/amazonTemplate'
  import { normalizeBlobResponse, triggerBlobDownload } from '@/utils/blobDownload'
  import {
    amazonListingSiteOptions,
    fetchLatestAmazonFxRateToCny,
    getAmazonSiteDefaultLocale,
    getAmazonSiteLabel,
    getAmazonSiteMarketplaceId,
    getAmazonSiteCurrencyCode,
    normalizeAmazonCurrencyCode,
    normalizeAmazonSiteCode
  } from '@/utils/amazonCurrency'
  import { getUrl } from '@/utils/image'
  import DynamicFieldInput from './components/DynamicFieldInput.vue'
  import ImageAssetEditor from './components/ImageAssetEditor.vue'
  import ProfitTrialPanel from './components/ProfitTrialPanel.vue'

  defineOptions({
    name: 'AmazonListingManager'
  })

  const router = useRouter()

  const listingStatusOptions = ['draft', 'active', 'archived']
  const siteCodeOptions = amazonListingSiteOptions.map((item) => item.value)
  const listingStatusLabelMap = {
    draft: '草稿',
    active: '启用',
    archived: '归档'
  }
  const roleLabelMap = {
    standalone: '独立款',
    parent: '父体',
    child: '子体'
  }
  const conditionTypeLabelMap = {
    new_new: '全新',
    used_like_new: '二手近新',
    used_very_good: '二手非常好',
    used_good: '二手良好',
    used_acceptable: '二手可接受',
    collectible_like_new: '收藏级近新',
    collectible_very_good: '收藏级非常好',
    collectible_good: '收藏级良好',
    collectible_acceptable: '收藏级可接受',
    refurbished: '翻新'
  }
  const externalProductIdTypeLabelMap = {
    UPC: 'UPC 条码',
    EAN: 'EAN 条码',
    GTIN: 'GTIN 编码',
    GCID: 'GCID 编码',
    ISBN: 'ISBN 书号'
  }
  const variationThemeLabelMap = {
    SizeName: '尺寸（SizeName）',
    ColorName: '颜色（ColorName）',
    SizeColor: '尺寸+颜色（SizeColor）',
    StyleName: '款式（StyleName）',
    StyleSize: '款式+尺寸（StyleSize）',
    FlavorName: '风味（FlavorName）',
    PatternName: '图案（PatternName）',
    Configuration: '配置（Configuration）',
    PackSize: '包装数量（PackSize）',
    MaterialType: '材质（MaterialType）',
    UnitCount: '件数（UnitCount）',
    ScentName: '香型（ScentName）'
  }
  const localeLabelMap = {
    en_US: '英语（美国）',
    en_CA: '英语（加拿大）',
    fr_CA: '法语（加拿大）',
    es_MX: '西班牙语（墨西哥）'
  }
  const localeOptions = [
    { label: '英语（美国）', value: 'en_US' },
    { label: '英语（加拿大）', value: 'en_CA' },
    { label: '法语（加拿大）', value: 'fr_CA' },
    { label: '西班牙语（墨西哥）', value: 'es_MX' }
  ]
  const builtinHomeTemplateCodeMap = {
    US: 'builtin-home-default-us',
    CA: 'builtin-home-default-ca',
    MX: 'builtin-home-default-mx'
  }
  const conditionTypeOptions = [
    'new_new',
    'used_like_new',
    'used_very_good',
    'used_good',
    'used_acceptable',
    'collectible_like_new',
    'collectible_very_good',
    'collectible_good',
    'collectible_acceptable',
    'refurbished'
  ]
  const externalProductIdTypeOptions = ['UPC', 'EAN', 'GTIN', 'GCID', 'ISBN']
  const variationThemePresetOptions = [
    'SizeName',
    'ColorName',
    'SizeColor',
    'StyleName',
    'StyleSize',
    'FlavorName',
    'PatternName',
    'Configuration',
    'PackSize',
    'MaterialType',
    'UnitCount',
    'ScentName'
  ]

  const treeData = ref([])
  const total = ref(0)
  const selectedRows = ref([])
  const drawerVisible = ref(false)
  const saveLoading = ref(false)
  const formMode = ref('create')
  const activePanels = ref([])
  const storeOptions = ref([])
  const templateOptions = ref([])
  const templateCache = reactive({})
  const listingDetailCache = reactive({})
  const publishDialogVisible = ref(false)
  const publishPreviewLoading = ref(false)
  const publishSubmitting = ref(false)
  const publishPreview = ref(null)
  const listingSyncDialogVisible = ref(false)
  const listingSyncPreviewLoading = ref(false)
  const listingSyncSubmitting = ref(false)
  const listingSyncPreview = ref(null)
  const publishForm = reactive({
    familyId: 0,
    storeId: undefined
  })
  const listingSyncFieldScopes = ['price', 'inventory', 'leadTimeToShip', 'merchantShippingGroup']
  const listingSyncForm = reactive({
    storeId: undefined
  })

  const getStatusLabel = (status) => listingStatusLabelMap[status] || status || '--'
  const getRoleLabel = (role) => roleLabelMap[role] || role || '--'
  const getProfitModeLabel = (mode) => {
    if (!mode) {
      return ''
    }
    const normalized = String(mode).trim().toLowerCase()
    if (normalized === 'fba') {
      return 'FBA'
    }
    if (normalized === 'fbm') {
      return 'FBM'
    }
    return String(mode).toUpperCase()
  }
  const getConditionTypeLabel = (value) => conditionTypeLabelMap[value] || value
  const getExternalProductIdTypeLabel = (value) => externalProductIdTypeLabelMap[value] || value
  const getVariationThemeLabel = (value) => variationThemeLabelMap[value] || value || ''
  const getLocaleLabel = (value) => localeLabelMap[value] || value
  const getSiteLabel = (siteCode) => getAmazonSiteLabel(siteCode)
  const getSiteBindingTitle = (siteCode) => `${getSiteLabel(siteCode)}绑定`
  const hasProfitSummary = (row) => typeof row?.profitNetProfitCny === 'number' && typeof row?.profitNetMarginRate === 'number'
  const formatProfitMoney = (value) => typeof value === 'number' ? `CNY ${value.toFixed(2)}` : '--'
  const formatProfitPercent = (value) => typeof value === 'number' ? `${(value * 100).toFixed(2)}%` : '--'
  const getProfitStatusTagType = (status) => {
    if (status === 'success') {
      return 'success'
    }
    if (status === 'warning') {
      return 'warning'
    }
    if (status === 'danger') {
      return 'danger'
    }
    return 'info'
  }
  const getProfitStatusTextClass = (status) => {
    if (status === 'success') {
      return 'text-emerald-500'
    }
    if (status === 'warning') {
      return 'text-amber-500'
    }
    if (status === 'danger') {
      return 'text-rose-500'
    }
    return 'text-slate-900 dark:text-slate-100'
  }
  const getProductTypeLabel = (value) => {
    if (!value) {
      return ''
    }
    if (String(value).toLowerCase() === 'home') {
      return '家居（home）'
    }
    return value
  }
  const getTemplateOptionLabel = (template) => {
    const title = template?.name || template?.code || '未命名模板'
    const site = getSiteLabel(template?.siteCode)
    const type = getProductTypeLabel(template?.productType || '')
    return [title, site, type].filter(Boolean).join(' / ')
  }
  const getStoreOptionLabel = (store) => {
    if (!store) {
      return '未选择店铺'
    }
    return `${store.storeName || `店铺${store.id}`}${store.authStatus === 'authorized' ? '' : '（未授权）'}`
  }
  const getStoreNameById = (storeId) => {
    const store = storeOptions.value.find((item) => Number(item.id) === Number(storeId || 0))
    return store?.storeName || ''
  }
  const getListingSyncStatusLabel = (status) => {
    switch (status) {
      case 'completed':
        return '已完成'
      case 'failed':
        return '失败'
      case 'processing':
        return '处理中'
      case 'submitted':
        return '已提交'
      default:
        return status || '--'
    }
  }
  const getListingSyncFieldScopeLabel = (scope) => {
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

  const normalizeOptions = (values = []) =>
    Array.from(
      new Set(
        values
          .map((item) => String(item || '').trim())
          .filter(Boolean)
      )
    )

  const productTypeOptions = computed(() =>
    normalizeOptions([
      ...templateOptions.value.map((item) => item.productType),
      formModel.family.productType
    ])
  )

  const variationThemeOptions = computed(() =>
    normalizeOptions([formModel.family.variationTheme, ...variationThemePresetOptions])
  )
  const authorizedStoreOptions = computed(() =>
    (storeOptions.value || []).filter((item) => item.authStatus === 'authorized' && item.isEnabled !== false)
  )

  const findDefaultHomeTemplate = (siteCode = 'US') => {
    const normalizedSiteCode = String(siteCode || 'US').toUpperCase()
    return templateOptions.value.find((template) => template.code === builtinHomeTemplateCodeMap[normalizedSiteCode]) ||
      templateOptions.value.find((template) => template.siteCode === normalizedSiteCode && String(template.productType || '').toLowerCase() === 'home')
  }

  const searchInfo = reactive({
    page: 1,
    pageSize: 10,
    keyword: '',
    siteCode: '',
    status: ''
  })

  const createEmptyImage = () => ({
    slotCode: '',
    fileId: 0,
    imageUrl: '',
    sort: 1,
    isPrimary: false
  })

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

  const cloneProfitResult = (result) => {
    if (!result || typeof result !== 'object') {
      return null
    }
    return {
      revenuePrice: toOptionalNumber(result.revenuePrice),
      revenueCurrencyCode: String(result.revenueCurrencyCode || '').trim(),
      saleCny: toOptionalNumber(result.saleCny),
      commissionCny: toOptionalNumber(result.commissionCny),
      adCostCny: toOptionalNumber(result.adCostCny),
      fixedCostCny: toOptionalNumber(result.fixedCostCny),
      grossProfitCny: toOptionalNumber(result.grossProfitCny),
      netProfitCny: toOptionalNumber(result.netProfitCny),
      netMarginRate: toOptionalNumber(result.netMarginRate),
      roiRate: toOptionalNumber(result.roiRate),
      breakEvenPrice: toOptionalNumber(result.breakEvenPrice),
      breakEvenCurrencyCode: String(result.breakEvenCurrencyCode || '').trim(),
      costBreakdown: {
        procurementCostCny: toOptionalNumber(result.costBreakdown?.procurementCostCny),
        firstLegCostCny: toOptionalNumber(result.costBreakdown?.firstLegCostCny),
        fbaFulfillmentFeeCny: toOptionalNumber(result.costBreakdown?.fbaFulfillmentFeeCny),
        fbmLastMileCostCny: toOptionalNumber(result.costBreakdown?.fbmLastMileCostCny),
        otherCostCny: toOptionalNumber(result.costBreakdown?.otherCostCny),
        commissionCny: toOptionalNumber(result.costBreakdown?.commissionCny),
        adCostCny: toOptionalNumber(result.costBreakdown?.adCostCny),
        fixedCostCny: toOptionalNumber(result.costBreakdown?.fixedCostCny)
      }
    }
  }

  const normalizeProfitProfile = (profile) => {
    const base = createEmptyProfitProfile()
    const current = profile && typeof profile === 'object' ? profile : {}
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
      result: cloneProfitResult(current.result)
    }
  }

  const serializeProfitProfile = (profile) => {
    const normalized = normalizeProfitProfile(profile)
    if (!normalized.fulfillmentMode) {
      return undefined
    }
    return {
      id: normalized.id || 0,
      fulfillmentMode: normalized.fulfillmentMode,
      costCurrencyCode: 'CNY',
      exchangeRateToCny: toOptionalNumber(normalized.exchangeRateToCny),
      referralFeeRate: toOptionalNumber(normalized.referralFeeRate),
      adCostRate: toOptionalNumber(normalized.adCostRate),
      procurementCostCny: toOptionalNumber(normalized.procurementCostCny),
      firstLegCostCny: toOptionalNumber(normalized.firstLegCostCny),
      fbaFulfillmentFeeCny: toOptionalNumber(normalized.fbaFulfillmentFeeCny),
      fbmLastMileCostCny: toOptionalNumber(normalized.fbmLastMileCostCny),
      otherCostCny: toOptionalNumber(normalized.otherCostCny)
    }
  }

  const mergeLocaleEntry = (current, incoming) => ({
    ...current,
    id: current?.id || incoming?.id || 0,
    itemName: current?.itemName || incoming?.itemName || '',
    bulletPoints: current?.bulletPoints?.length ? current.bulletPoints : [...(incoming?.bulletPoints || [])],
    productDescription: current?.productDescription || incoming?.productDescription || '',
    searchTerms: current?.searchTerms?.length ? current.searchTerms : [...(incoming?.searchTerms || [])],
    localizedAttributes: {
      ...(incoming?.localizedAttributes || {}),
      ...(current?.localizedAttributes || {})
    }
  })

  const dedupeLocales = (locales = []) => {
    const result = []
    const indexMap = new Map()
    locales.forEach((locale) => {
      const localeCode = String(locale?.localeCode || '').trim()
      const normalized = {
        ...locale,
        localeCode,
        bulletPoints: [...(locale?.bulletPoints || [])],
        searchTerms: [...(locale?.searchTerms || [])],
        localizedAttributes: { ...(locale?.localizedAttributes || {}) }
      }
      if (!localeCode) {
        result.push(normalized)
        return
      }
      if (indexMap.has(localeCode)) {
        const existingIndex = indexMap.get(localeCode)
        result[existingIndex] = mergeLocaleEntry(result[existingIndex], normalized)
        return
      }
      indexMap.set(localeCode, result.length)
      result.push(normalized)
    })
    return result
  }

  const getUnusedLocaleOption = (binding, currentLocaleCode = '') =>
    localeOptions.find((option) =>
      option.value === currentLocaleCode ||
      !(binding?.locales || []).some((locale) => locale.localeCode === option.value)
    )

  const isLocaleOptionDisabled = (binding, locale, optionValue) =>
    (binding?.locales || []).some((entry) => entry !== locale && entry.localeCode === optionValue)

  const createEmptyLocale = (localeCode = 'en_US') => ({
    __key: `locale-${Date.now()}-${Math.random()}`,
    id: 0,
    localeCode,
    itemName: '',
    bulletPoints: [],
    productDescription: '',
    searchTerms: [],
    localizedAttributes: {}
  })

  const createEmptyMarketplace = (siteCode = 'US') => {
    const normalizedSiteCode = normalizeAmazonSiteCode(siteCode)
    return {
      __key: `market-${Date.now()}-${Math.random()}`,
      id: 0,
      storeId: authorizedStoreOptions.value[0]?.id,
      templateId: undefined,
      marketplaceId: getAmazonSiteMarketplaceId(normalizedSiteCode),
      siteCode: normalizedSiteCode,
      currencyCode: getAmazonSiteCurrencyCode(normalizedSiteCode),
      offerPrice: undefined,
      salePrice: undefined,
      quantity: undefined,
      leadTimeToShip: undefined,
      merchantShippingGroup: '',
      lastPriceInventorySyncAt: '',
      lastPriceInventorySyncStatus: '',
      lastPriceInventorySyncMessage: '',
      remoteFbaAvailableQuantity: undefined,
      remoteFbaReservedQuantity: undefined,
      remoteFbaInboundQuantity: undefined,
      lastRemoteInventorySyncAt: '',
      lastRemoteInventorySyncError: '',
      marketplaceAttributes: {},
      profitProfile: createEmptyProfitProfile(),
      locales: [createEmptyLocale(getAmazonSiteDefaultLocale(normalizedSiteCode))],
      images: [{ ...createEmptyImage(), slotCode: 'MAIN', isPrimary: true }]
    }
  }

  const createEmptyItem = (role = 'standalone') => ({
    __key: `item-${Date.now()}-${Math.random()}`,
    id: 0,
    parentItemId: undefined,
    role,
    sku: '',
    brand: '',
    conditionType: 'new_new',
    externalProductIdType: '',
    externalProductId: '',
    merchantSuggestedAsin: '',
    commonAttributes: {},
    variationAttributes: {},
    status: 'draft',
    sharedImages: [{ ...createEmptyImage(), slotCode: 'MAIN', isPrimary: true }],
    marketplaces: [createEmptyMarketplace('US')]
  })

  const formModel = reactive({
    family: {
      id: 0,
      familyName: '',
      productType: '',
      variationTheme: '',
      parentSku: '',
      status: 'draft',
      remark: ''
    },
    items: []
  })

  const fetchTree = async () => {
    const res = await getAmazonListingTree(searchInfo)
    treeData.value = res.data?.list || []
    total.value = res.data?.total || 0
  }

  const fetchTemplateOptions = async () => {
    const res = await getAmazonTemplateList({
      page: 1,
      pageSize: 200
    })
    templateOptions.value = res.data?.list || []
    if (drawerVisible.value) {
      await applyDefaultTemplatesToForm()
    }
  }

  const fetchStoreOptions = async () => {
    const res = await getAmazonStoreList({
      page: 1,
      pageSize: 200
    })
    storeOptions.value = res.data?.list || []
    const defaultStoreId = authorizedStoreOptions.value[0]?.id
    if (defaultStoreId && drawerVisible.value) {
      formModel.items.forEach((item) => {
        ;(item.marketplaces || []).forEach((binding) => {
          if (!binding.storeId) {
            binding.storeId = defaultStoreId
          }
        })
      })
    }
  }

  const resetSearch = () => {
    searchInfo.page = 1
    searchInfo.pageSize = 10
    searchInfo.keyword = ''
    searchInfo.siteCode = ''
    searchInfo.status = ''
    fetchTree()
  }

  const handleCurrentChange = (page) => {
    searchInfo.page = page
    fetchTree()
  }

  const handleSizeChange = (pageSize) => {
    searchInfo.page = 1
    searchInfo.pageSize = pageSize
    fetchTree()
  }

  const handleSelectionChange = (rows) => {
    selectedRows.value = rows || []
  }

  const resetForm = () => {
    formModel.family.id = 0
    formModel.family.familyName = ''
    formModel.family.productType = ''
    formModel.family.variationTheme = ''
    formModel.family.parentSku = ''
    formModel.family.status = 'draft'
    formModel.family.remark = ''
    formModel.items = [createEmptyItem('standalone')]
    activePanels.value = [formModel.items[0].__key]
  }

  const openCreateDrawer = async () => {
    formMode.value = 'create'
    resetForm()
    drawerVisible.value = true
    await applyDefaultTemplatesToForm()
    await syncAllMarketplaceRates({ showWarning: false })
  }

  const openEditDrawer = async (row) => {
    const res = await findAmazonListing({ familyId: row.familyId })
    const data = res.data || {}
    formMode.value = 'edit'
    formModel.family.id = data.id || row.familyId
    formModel.family.familyName = data.familyName || ''
    formModel.family.productType = data.productType || ''
    formModel.family.variationTheme = data.variationTheme || ''
    formModel.family.parentSku = data.parentSku || ''
    formModel.family.status = data.status || 'draft'
    formModel.family.remark = data.remark || ''
    formModel.items = (data.items || []).map((item) => ({
      __key: `item-${item.id || Math.random()}`,
      ...item,
      commonAttributes: { ...(item.commonAttributes || {}) },
      variationAttributes: { ...(item.variationAttributes || {}) },
      sharedImages: (item.sharedImages || []).map((image) => ({ ...image })),
      marketplaces: (item.marketplaces || []).map((binding) => ({
        __key: `market-${binding.id || Math.random()}`,
        ...binding,
        storeId: binding.storeId || undefined,
        marketplaceAttributes: { ...(binding.marketplaceAttributes || {}) },
        profitProfile: normalizeProfitProfile(binding.profitProfile),
        locales: (binding.locales || []).map((locale) => ({
          __key: `locale-${locale.id || Math.random()}`,
          ...locale,
          localizedAttributes: { ...(locale.localizedAttributes || {}) },
          bulletPoints: [...(locale.bulletPoints || [])],
          searchTerms: [...(locale.searchTerms || [])]
        })),
        images: (binding.images || []).map((image) => ({ ...image }))
      }))
    }))
    activePanels.value = formModel.items.map((item) => item.__key)
    drawerVisible.value = true
    await applyDefaultTemplatesToForm()
    await syncAllMarketplaceRates({ showWarning: false, resetTemplate: false })
  }

  const closeDrawer = () => {
    drawerVisible.value = false
  }

  const addItem = (role) => {
    const item = createEmptyItem(role)
    if (role === 'child') {
      const parent = formModel.items.find((entry) => entry.role === 'parent')
      if (parent?.id) {
        item.parentItemId = parent.id
      }
    }
    formModel.items.push(item)
    activePanels.value = [...activePanels.value, item.__key]
  }

  const removeItem = (itemIndex) => {
    formModel.items.splice(itemIndex, 1)
  }

  const addMarketplace = (item) => {
    const binding = createEmptyMarketplace('US')
    item.marketplaces.push(binding)
    applyDefaultTemplateForBinding(binding).then(() => syncMarketplaceSiteDefaults(binding, { showWarning: false, resetTemplate: false }))
  }

  const removeMarketplace = (item, index) => {
    item.marketplaces.splice(index, 1)
  }

  const showMissingFxRate = (currencyCode) => {
    ElMessage.warning(`${currencyCode} 暂无可用汇率，请先在汇率管理维护`)
  }

  const syncMarketplaceProfitRate = async (binding, options = {}) => {
    if (!binding) {
      return undefined
    }
    const showWarning = options.showWarning !== false
    const currencyCode = normalizeAmazonCurrencyCode(binding.currencyCode || getAmazonSiteCurrencyCode(binding.siteCode))
    binding.currencyCode = currencyCode
    const token = Number(binding.__fxToken || 0) + 1
    binding.__fxToken = token
    const rate = await fetchLatestAmazonFxRateToCny(currencyCode)
    if (binding.__fxToken !== token) {
      return undefined
    }
    binding.profitProfile = normalizeProfitProfile({
      ...(binding.profitProfile || {}),
      exchangeRateToCny: rate
    })
    if (!rate && showWarning) {
      showMissingFxRate(currencyCode)
    }
    return rate
  }

  const syncMarketplaceSiteDefaults = async (binding, options = {}) => {
    if (!binding) {
      return
    }
    const showWarning = options.showWarning !== false
    const resetTemplate = options.resetTemplate !== false
    const siteCode = normalizeAmazonSiteCode(binding.siteCode)
    binding.siteCode = siteCode
    binding.marketplaceId = getAmazonSiteMarketplaceId(siteCode) || binding.marketplaceId || ''
    binding.currencyCode = getAmazonSiteCurrencyCode(siteCode)
    if (!(binding.locales || []).length) {
      binding.locales = [createEmptyLocale(getAmazonSiteDefaultLocale(siteCode))]
    }
    if (resetTemplate && binding.templateId) {
      const template = templateCache[binding.templateId] || templateOptions.value.find((item) => item.id === binding.templateId)
      if (template?.siteCode && template.siteCode !== siteCode) {
        binding.templateId = undefined
        binding.marketplaceAttributes = {}
      }
    }
    await syncMarketplaceProfitRate(binding, { showWarning })
    if (resetTemplate && !binding.templateId) {
      await applyDefaultTemplateForBinding(binding)
    }
  }

  const syncAllMarketplaceRates = async (options = {}) => {
    const tasks = []
    formModel.items.forEach((item) => {
      ;(item.marketplaces || []).forEach((binding) => {
        tasks.push(syncMarketplaceSiteDefaults(binding, options))
      })
    })
    await Promise.all(tasks)
  }

  const onMarketplaceSiteChange = (binding) => {
    syncMarketplaceSiteDefaults(binding)
  }

  const onMarketplaceCurrencyChange = (binding) => {
    syncMarketplaceProfitRate(binding)
  }

  const addLocale = (binding) => {
    const nextLocale = getUnusedLocaleOption(binding)
    if (!nextLocale) {
      ElMessage.warning('当前站点支持的语言已全部添加，请勿重复新增')
      return
    }
    binding.locales.push(createEmptyLocale(nextLocale.value))
  }

  const removeLocale = (binding, index) => {
    binding.locales.splice(index, 1)
  }

  const updateBulletPoints = (locale, value) => {
    locale.bulletPoints = value
      .split('\n')
      .map((item) => item.trim())
      .filter(Boolean)
  }

  const updateSearchTerms = (locale, value) => {
    locale.searchTerms = value
      .split('\n')
      .map((item) => item.trim())
      .filter(Boolean)
  }

  const getTemplateDetail = async (templateId) => {
    if (!templateId) {
      return null
    }
    if (!templateCache[templateId]) {
      const res = await findAmazonTemplate({ id: templateId })
      templateCache[templateId] = res.data || null
    }
    return templateCache[templateId]
  }

  const getListingFamilyDetail = async (familyId) => {
    const normalizedFamilyId = Number(familyId || 0)
    if (!normalizedFamilyId) {
      return null
    }
    if (!listingDetailCache[normalizedFamilyId]) {
      const res = await findAmazonListing({ familyId: normalizedFamilyId })
      listingDetailCache[normalizedFamilyId] = res.data || null
    }
    return listingDetailCache[normalizedFamilyId]
  }

  const onTemplateChange = async (binding) => {
    const detail = await getTemplateDetail(binding.templateId)
    if (!detail) {
      return
    }
    const siteCode = normalizeAmazonSiteCode(detail.siteCode || binding.siteCode)
    binding.marketplaceId = detail.marketplaceId || getAmazonSiteMarketplaceId(siteCode) || binding.marketplaceId
    binding.siteCode = siteCode
    binding.currencyCode = getAmazonSiteCurrencyCode(siteCode)
    binding.locales = dedupeLocales(binding.locales || [])
    const locales = detail.supportedLocales || []
    if (!binding.locales.length && locales.length) {
      binding.locales = locales.map((localeCode) => createEmptyLocale(localeCode))
    }
    await syncMarketplaceProfitRate(binding, { showWarning: false })
    if (!formModel.family.productType && detail.productType) {
      formModel.family.productType = detail.productType
    }
  }

  const applyDefaultTemplateForBinding = async (binding) => {
    if (!binding || binding.templateId) {
      return
    }
    const template = findDefaultHomeTemplate(binding.siteCode || 'US')
    if (!template?.id) {
      return
    }
    binding.templateId = template.id
    await onTemplateChange(binding)
  }

  const applyDefaultTemplatesToForm = async () => {
    for (const item of formModel.items) {
      for (const binding of item.marketplaces || []) {
        if (!binding.templateId) {
          await applyDefaultTemplateForBinding(binding)
        }
      }
    }
  }

  const getTemplateFieldsByScope = (item, scope, binding, locale) => {
    const templateIds = binding ? [binding.templateId] : item.marketplaces.map((entry) => entry.templateId).filter(Boolean)
    const seen = new Set()
    const fields = []
    templateIds.forEach((templateId) => {
      const detail = templateCache[templateId]
      if (!detail) {
        return
      }
      ;(detail.fields || []).forEach((field) => {
        if (field.scope !== scope || !field.enabled) {
          return
        }
        if (scope === 'locale' && locale && field.localeCode && field.localeCode !== locale.localeCode) {
          return
        }
        if (seen.has(field.fieldKey)) {
          return
        }
        seen.add(field.fieldKey)
        fields.push(field)
      })
    })
    return fields
  }

  const getCommonFields = (item) => getTemplateFieldsByScope(item, 'common')
  const getVariationFields = (item) => getTemplateFieldsByScope(item, 'variation')
  const getMarketplaceFields = (binding) => getTemplateFieldsByScope({ marketplaces: [binding] }, 'marketplace', binding)
  const getLocaleFields = (binding, locale) => getTemplateFieldsByScope({ marketplaces: [binding] }, 'locale', binding, locale)

  const normalizeFieldMatcher = (field) =>
    `${field.fieldKey || ''}${field.columnHeader || ''}${field.amazonPath || ''}`
      .toLowerCase()
      .replace(/[^a-z0-9]/g, '')

  const getFieldOptions = (field) => {
    const matcher = normalizeFieldMatcher(field)
    if (!matcher) {
      return []
    }
    if (
      matcher.includes('producttype') ||
      matcher.includes('feedproducttype') ||
      matcher.includes('itemtype') ||
      matcher.includes('itemtypekeyword') ||
      matcher.includes('category') ||
      matcher.includes('classification')
    ) {
      return productTypeOptions.value
    }
    if (matcher.includes('variationtheme')) {
      return variationThemeOptions.value
    }
    if (matcher.includes('conditiontype')) {
      return conditionTypeOptions
    }
    if (matcher.includes('externalproductidtype') || matcher.includes('productidtype')) {
      return externalProductIdTypeOptions
    }
    if (matcher.includes('parentage')) {
      return ['parent', 'child']
    }
    if (matcher.includes('relationshiptype')) {
      return ['variation']
    }
    if (matcher.includes('sitecode')) {
      return siteCodeOptions
    }
    if (matcher.includes('status')) {
      return listingStatusOptions
    }
    return []
  }

  const getFieldGuide = (field) => {
    const matcher = normalizeFieldMatcher(field)
    if (matcher.includes('producttype') || matcher.includes('itemtypekeyword')) {
      return {
        label: '产品类型',
        help: '决定 Amazon 详情页属性范围、模板列头和导出规则。'
      }
    }
    if (matcher.includes('variationtheme')) {
      return {
        label: '变体主题',
        help: '定义 Amazon 详情页规格切换按颜色、尺寸等哪个维度展示。'
      }
    }
    if (matcher.includes('itemname') || matcher.includes('title')) {
      return {
        label: '商品标题',
        help: '用于 Amazon 搜索结果列表标题和详情页主标题。'
      }
    }
    if (matcher.includes('brand')) {
      return {
        label: '品牌',
        help: '用于 Amazon 详情页品牌展示和品牌筛选。'
      }
    }
    if (matcher.includes('conditiontype')) {
      return {
        label: '商品状况',
        help: '用于 Amazon 详情页状态展示，例如全新、二手、翻新。'
      }
    }
    if (matcher.includes('externalproductidtype')) {
      return {
        label: '外部商品编码类型',
        help: '指定 UPC、EAN 等编码类别，用于 Amazon 商品身份识别。'
      }
    }
    if (matcher.includes('externalproductid')) {
      return {
        label: '外部商品编码',
        help: '填写 UPC、EAN 等实际编码值，用于建档和识别。'
      }
    }
    if (matcher.includes('merchantsuggestedasin')) {
      return {
        label: '建议 ASIN',
        help: '适用于已知目标 ASIN 的跟卖或关联场景。'
      }
    }
    if (matcher.includes('mainimageurl')) {
      return {
        label: '主图链接',
        help: '用于 Amazon 列表主图、详情页主图和 1688 图搜选款。'
      }
    }
    if (matcher.includes('otherimageurl')) {
      return {
        label: '附图链接',
        help: '用于 Amazon 详情页图集补充展示。'
      }
    }
    if (matcher.includes('standardprice') || matcher.includes('offerprice')) {
      return {
        label: '售价',
        help: '用于当前站点 Amazon 列表页和详情页的基础售价展示。'
      }
    }
    if (matcher.includes('saleprice')) {
      return {
        label: '促销价',
        help: '用于 Amazon 前台活动价格或折扣展示。'
      }
    }
    if (matcher.includes('quantity')) {
      return {
        label: '库存数量',
        help: '用于 Amazon 可售库存控制和导出。'
      }
    }
    if (matcher.includes('merchantshippinggroup')) {
      return {
        label: '配送模板',
        help: '用于 Amazon 后台配送模板匹配和运费时效设置。'
      }
    }
    if (matcher.includes('leadtimetoship')) {
      return {
        label: '备货天数',
        help: '用于 Amazon 预计发货时效计算。'
      }
    }
    if (matcher.includes('bulletpoint')) {
      return {
        label: '卖点',
        help: '用于 Amazon 详情页卖点区展示核心卖点。'
      }
    }
    if (matcher.includes('productdescription') || matcher.includes('description')) {
      return {
        label: '详情描述',
        help: '用于 Amazon 详情页长描述区域。'
      }
    }
    if (matcher.includes('searchterms') || matcher.includes('generickeywords')) {
      return {
        label: '搜索关键词',
        help: '用于 Amazon 站内搜索匹配，不直接展示给买家。'
      }
    }
    if (matcher.includes('parentage')) {
      return {
        label: '父子关系',
        help: '用于标记当前记录是父体还是子体，决定变体结构。'
      }
    }
    if (matcher.includes('parentsku')) {
      return {
        label: '父 SKU',
        help: '用于子体关联对应父体。'
      }
    }
    if (matcher.includes('relationshiptype')) {
      return {
        label: '关系类型',
        help: '用于 Amazon 识别父子体之间的变体关系。'
      }
    }
    if (matcher.includes('colorname')) {
      return {
        label: '颜色',
        help: '用于 Amazon 详情页颜色规格切换。'
      }
    }
    if (matcher.includes('sizename')) {
      return {
        label: '尺寸',
        help: '用于 Amazon 详情页尺寸规格切换。'
      }
    }
    if (matcher.includes('material')) {
      return {
        label: '材质',
        help: '用于 Amazon 详情页规格参数展示。'
      }
    }
    if (field?.scope === 'marketplace') {
      return {
        label: field?.fieldLabel || '站点字段',
        help: '仅作用于当前 Amazon 站点的价格、库存、配送或站点属性。'
      }
    }
    if (field?.scope === 'locale') {
      return {
        label: field?.fieldLabel || '语言字段',
        help: '用于 Amazon 详情页不同语言的标题、卖点、描述或搜索词。'
      }
    }
    if (field?.scope === 'variation') {
      return {
        label: field?.fieldLabel || '变体字段',
        help: '用于 Amazon 父子体区分，如颜色、尺寸、款式等。'
      }
    }
    if (field?.scope === 'image') {
      return {
        label: field?.fieldLabel || '图片字段',
        help: '用于 Amazon 主图、副图和详情图资源映射。'
      }
    }
    return {
      label: field?.fieldLabel || field?.columnHeader || field?.fieldKey || '模板字段',
      help: '该字段来自模板中心配置，用于 Amazon 列表/详情页属性展示、校验或导出。'
    }
  }

  const getFieldDisplayLabel = (field) => {
    const guide = getFieldGuide(field)
    const source = String(field?.fieldLabel || '').trim()
    if (!source || /^[A-Za-z0-9_ -]+$/.test(source)) {
      return guide.label
    }
    return source
  }

  const getFieldHelpText = (field) => getFieldGuide(field).help

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

  const isPublicImageUrl = (url) => /^https?:\/\//i.test(String(url || '').trim())
  const isAlibabaImageUrl = (url) => {
    const value = String(url || '').trim().toLowerCase()
    return value.includes('alicdn.com') || value.includes('alibaba.com') || value.includes('1688.com')
  }

  const get1688ImageSearchUrl = (imageUrl, taskToken = '') => {
    const trimmed = String(imageUrl || '').trim()
    const url = new URL('https://s.1688.com/shen/sell_offer.htm')
    url.searchParams.set('tab', 'imageSearch')
    if (String(taskToken || '').trim()) {
      url.searchParams.set('__gva1688Task', String(taskToken || '').trim())
    }
    if (trimmed) {
      url.searchParams.set('__gva1688Image', trimmed)
    }
    return url.toString()
  }

  const isLocalOrPrivateHost = (hostname) => {
    const host = String(hostname || '').trim().toLowerCase()
    if (!host) {
      return false
    }
    if (['localhost', '127.0.0.1', '0.0.0.0', '::1'].includes(host)) {
      return true
    }
    if (/^10\./.test(host) || /^192\.168\./.test(host)) {
      return true
    }
    const private172Match = host.match(/^172\.(\d+)\./)
    return private172Match ? Number(private172Match[1]) >= 16 && Number(private172Match[1]) <= 31 : false
  }

  const isLocalOrPrivateImageUrl = (url) => {
    try {
      const parsed = new URL(String(url || '').trim())
      return isLocalOrPrivateHost(parsed.hostname)
    } catch (error) {
      return false
    }
  }

  const openPendingExternalWindow = () => {
    const opened = window.open('', '_blank')
    if (!opened) {
      return null
    }
    try {
      opened.document.write('<!doctype html><title>Opening 1688</title><body style="font:14px system-ui;padding:24px;color:#475569">Opening 1688 image search...</body>')
      opened.document.close()
    } catch (error) {
      // The placeholder is best-effort; navigation below is the important part.
    }
    return opened
  }

  const navigateExternalWindow = (opened, url) => {
    if (opened && !opened.closed) {
      try {
        opened.opener = null
      } catch (error) {
        // Ignore browsers that do not allow clearing opener on about:blank.
      }
      try {
        opened.location.replace(url)
        return true
      } catch (error) {
        try {
          opened.location.href = url
          return true
        } catch (innerError) {
          return false
        }
      }
    }
    return Boolean(window.open(url, '_blank', 'noopener,noreferrer'))
  }

  const fallbackCopyText = (text) => {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.setAttribute('readonly', 'readonly')
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    textarea.style.left = '-9999px'
    document.body.appendChild(textarea)
    textarea.select()
    const copied = document.execCommand('copy')
    document.body.removeChild(textarea)
    return copied
  }

  const copyText = async (text) => {
    if (!text) {
      return false
    }
    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(text)
        return true
      } catch (error) {
        return fallbackCopyText(text)
      }
    }
    return fallbackCopyText(text)
  }

  const RUIGUAN_DETECT_URL = 'https://detect.eric-bot.com/eric'

  const getRuiguanDetectUrl = () => RUIGUAN_DETECT_URL

  const getAttributeText = (values = {}, keys = []) => {
    if (!values || typeof values !== 'object') {
      return ''
    }
    for (const key of keys) {
      const value = values[key]
      if (String(value || '').trim()) {
        return String(value).trim()
      }
    }
    return ''
  }

  const getPreferredRuiguanItem = (detail, row) => {
    const items = Array.isArray(detail?.items) ? detail.items : []
    return items.find((item) => String(item?.sku || '').trim() === String(row?.sku || '').trim()) ||
      items.find((item) => item?.role !== 'parent') ||
      items[0] ||
      null
  }

  const buildRuiguanSearchKeyword = (detail, row) => {
    const preferredItem = getPreferredRuiguanItem(detail, row)
    const locales = preferredItem?.marketplaces?.flatMap((binding) => binding?.locales || []) || []
    const localeTitle = locales.find((locale) => String(locale?.itemName || '').trim())?.itemName || ''
    const commonTitle = getAttributeText(preferredItem?.commonAttributes, ['itemName', 'item_name', 'title', 'productTitle'])
    const variationTitle = getAttributeText(preferredItem?.variationAttributes, ['itemName', 'item_name', 'title'])
    const brand = String(preferredItem?.brand || '').trim()
    const productType = String(detail?.productType || row?.productType || '').trim()
    const familyName = String(detail?.familyName || row?.label || '').trim()
    const title = String(localeTitle || commonTitle || variationTitle || familyName).trim()

    const parts = []
    if (brand && !title.toLowerCase().includes(brand.toLowerCase())) {
      parts.push(brand)
    }
    if (title) {
      parts.push(title)
    }
    if (productType && !parts.some((part) => part.toLowerCase().includes(productType.toLowerCase()))) {
      parts.push(productType)
    }

    return parts
      .map((item) => String(item || '').trim())
      .filter(Boolean)
      .slice(0, 3)
      .join(' ')
  }

  const buildRuiguanDetectPrompt = (detail, row) => {
    const preferredItem = getPreferredRuiguanItem(detail, row)
    const keyword = buildRuiguanSearchKeyword(detail, row)
    const brand = String(preferredItem?.brand || '').trim()
    const productType = String(detail?.productType || row?.productType || '').trim()
    const familyName = String(detail?.familyName || row?.label || '').trim()
    const sku = String(preferredItem?.sku || row?.sku || '').trim()
    const siteCodes = Array.from(new Set(
      (preferredItem?.marketplaces || [])
        .map((binding) => String(binding?.siteCode || '').trim().toUpperCase())
        .filter(Boolean)
    ))
    const imageUrl = resolveImageUrl(row?.mainImageUrl)
    const lines = ['请帮我检测这个 Amazon 商品是否存在侵权风险，并优先排查外观专利、商标和版权问题。']

    if (keyword) {
      lines.push(`商品关键词：${keyword}`)
    }
    if (brand) {
      lines.push(`品牌：${brand}`)
    }
    if (productType) {
      lines.push(`产品类型：${productType}`)
    }
    if (familyName) {
      lines.push(`商品组：${familyName}`)
    }
    if (sku) {
      lines.push(`SKU：${sku}`)
    }
    if (siteCodes.length) {
      lines.push(`目标站点：${siteCodes.join(' / ')}`)
    }
    if (imageUrl && isPublicImageUrl(imageUrl)) {
      lines.push(`主图链接：${imageUrl}`)
    }

    return lines.join('\n')
  }

  const getPreferredStoreIdForFamily = (detail) => {
    const bindings = (detail?.items || []).flatMap((item) => item?.marketplaces || [])
    return bindings.find((binding) => Number(binding?.storeId || 0) > 0)?.storeId || authorizedStoreOptions.value[0]?.id
  }

  const openPublishDialog = async (row) => {
    const familyId = Number(row?.familyId || 0)
    if (!familyId) {
      ElMessage.warning('当前记录缺少商品组信息，无法发起发布')
      return
    }
    publishForm.familyId = familyId
    publishPreview.value = null
    const detail = await getListingFamilyDetail(familyId)
    publishForm.storeId = getPreferredStoreIdForFamily(detail)
    publishDialogVisible.value = true
    if (publishForm.storeId) {
      await loadPublishPreview()
    }
  }

  const loadPublishPreview = async () => {
    if (!publishForm.familyId) {
      return
    }
    if (!publishForm.storeId) {
      publishPreview.value = null
      return
    }
    publishPreviewLoading.value = true
    try {
      const res = await previewAmazonListingPublish({
        familyId: publishForm.familyId,
        storeId: publishForm.storeId
      })
      publishPreview.value = res.data || null
    } finally {
      publishPreviewLoading.value = false
    }
  }

  const submitPublishJob = async () => {
    if (!publishForm.familyId || !publishForm.storeId) {
      ElMessage.warning('请先选择店铺并完成预检')
      return
    }
    publishSubmitting.value = true
    try {
      const res = await submitAmazonListingPublish({
        familyId: publishForm.familyId,
        storeId: publishForm.storeId
      })
      ElMessage.success(`发布任务已创建，Feed ID：${res.data?.feedId || '--'}`)
      publishDialogVisible.value = false
    } finally {
      publishSubmitting.value = false
    }
  }

  const openListingSyncJobs = () => {
    router.push({ name: 'amazonListingSyncJobManager' })
  }

  const openListingSyncDialog = async () => {
    if (!selectedRows.value.length) {
      ElMessage.warning('请先选择至少一条商品')
      return
    }
    listingSyncPreview.value = null
    listingSyncForm.storeId = authorizedStoreOptions.value[0]?.id
    listingSyncDialogVisible.value = true
    if (listingSyncForm.storeId) {
      await loadListingSyncPreview()
    }
  }

  const loadListingSyncPreview = async () => {
    if (!listingSyncForm.storeId) {
      listingSyncPreview.value = null
      return
    }
    listingSyncPreviewLoading.value = true
    try {
      const payload = buildSelectedPayload()
      const res = await previewAmazonListingSync({
        storeId: listingSyncForm.storeId,
        familyIds: payload.familyIds,
        itemIds: payload.itemIds,
        fieldScopes: listingSyncFieldScopes
      })
      listingSyncPreview.value = res.data || null
    } finally {
      listingSyncPreviewLoading.value = false
    }
  }

  const submitListingSyncJob = async () => {
    if (!listingSyncForm.storeId || !listingSyncPreview.value?.valid) {
      ElMessage.warning('请先选择店铺并完成预检')
      return
    }
    listingSyncSubmitting.value = true
    try {
      const payload = buildSelectedPayload()
      const res = await submitAmazonListingSync({
        storeId: listingSyncForm.storeId,
        familyIds: payload.familyIds,
        itemIds: payload.itemIds,
        fieldScopes: listingSyncFieldScopes
      })
      ElMessage.success(`价格库存回传任务已创建，任务ID：${res.data?.id || '--'}`)
      listingSyncDialogVisible.value = false
      if (res.data?.id) {
        router.push({
          name: 'amazonListingSyncJobDetail',
          params: { id: res.data.id }
        })
      }
    } finally {
      listingSyncSubmitting.value = false
    }
  }

  const openRuiguanQuery = async (row) => {
    let detail = null
    try {
      detail = await getListingFamilyDetail(row?.familyId)
    } catch (error) {
      detail = null
    }

    const keyword = buildRuiguanSearchKeyword(detail, row)
    const prompt = buildRuiguanDetectPrompt(detail, row)
    const url = getRuiguanDetectUrl()
    window.open(url, '_blank', 'noopener,noreferrer')

    const copied = await copyText(prompt)
    if (copied) {
      ElMessage.success(keyword
        ? `已打开睿观侵权检测，并复制检测内容：${keyword}`
        : '已打开睿观侵权检测，并复制检测内容，请直接粘贴到睿观继续查询')
      return
    }
    ElMessage.warning(keyword
      ? '已打开睿观侵权检测，请把商品关键词或主图链接粘贴到睿观继续查询'
      : '已打开睿观侵权检测，但未提取到有效商品信息，请在睿观手动补充标题或图片链接')
  }

  const openSelectedRuiguanQuery = async () => {
    if (!selectedRows.value.length) {
      ElMessage.warning('请先选中一条商品再进行侵权查询')
      return
    }
    if (selectedRows.value.length > 1) {
      ElMessage.warning('当前仅按第一条选中商品发起侵权查询')
    }
    await openRuiguanQuery(selectedRows.value[0])
  }

  const handleRowAction = async (command, row) => {
    switch (command) {
      case 'ruiguan':
        await openRuiguanQuery(row)
        return
      case 'collect1688':
        await start1688Collect(row)
        return
      case 'publish':
        await openPublishDialog(row)
        return
      case 'validate':
        await validateFamily(row)
        return
      case 'delete':
        await deleteFamilyRow(row)
        return
      default:
        return
    }
  }

  const start1688Collect = async (row) => {
    const pendingWindow = openPendingExternalWindow()
    const listingItemId = Number(row?.id || 0)
    if (!listingItemId) {
      pendingWindow?.close?.()
      ElMessage.warning('当前记录缺少商品ID，无法发起货物采集')
      return
    }

    const systemCode = String(row?.sku || '').trim()
    if (!systemCode) {
      pendingWindow?.close?.()
      ElMessage.warning('当前商品缺少 SKU，无法作为系统 code 发起采集')
      return
    }

    const imageUrl = resolveImageUrl(row?.mainImageUrl)
    if (!imageUrl) {
      pendingWindow?.close?.()
      ElMessage.warning('当前商品没有可用主图')
      return
    }

    if (!isPublicImageUrl(imageUrl)) {
      pendingWindow?.close?.()
      ElMessage.warning('当前图片地址不是公网可访问链接，1688 图搜无法直接使用')
      return
    }

    if (isLocalOrPrivateImageUrl(imageUrl)) {
      ElMessage.info('当前主图为本地缓存地址，将通过采集助手补传图片触发 1688 图搜')
    }

    if (!isAlibabaImageUrl(imageUrl)) {
      ElMessage.warning('当前主图不是阿里系图片地址，1688 图搜识别率可能较低')
    }

    try {
      const res = await createAmazon1688CollectTask({
        listingItemId,
        systemCode,
        mainImageUrl: imageUrl
      })
      const searchURL = String(res.data?.searchUrl || '').trim() || get1688ImageSearchUrl(imageUrl, res.data?.taskToken)
      if (!navigateExternalWindow(pendingWindow, searchURL)) {
        pendingWindow?.close?.()
        await copyText(searchURL)
        ElMessage.warning('浏览器拦截了 1688 图搜窗口，已复制链接，请粘贴到 Chrome 打开')
        return
      }
      ElMessage.success('已打开 1688 图搜，请选择商品；进入详情页后会自动采集，如失败可点击页面上的“采集货物”按钮。')
    } catch (error) {
      pendingWindow?.close?.()
      throw error
    }
  }

  const downloadTemplateFile = async (params, fallbackFileName, successMessage) => {
    const res = await downloadAmazonTemplateWorkbook(params)
    const { blob, fileName } = await normalizeBlobResponse(res, fallbackFileName)
    triggerBlobDownload(blob, fileName)
    ElMessage.success(successMessage)
  }

  const downloadHomeTemplate = async (siteCode = 'US') => {
    await downloadTemplateFile(
      { preset: 'home', siteCode },
      `amazon-home-template-${String(siteCode || 'US').toLowerCase()}.xlsx`,
      '家居默认模板下载成功'
    )
  }

  const downloadSelectedTemplate = async (binding) => {
    if (!binding?.templateId) {
      await downloadHomeTemplate(binding?.siteCode || 'US')
      return
    }
    const template = templateOptions.value.find((item) => item.id === binding.templateId)
    await downloadTemplateFile(
      { id: binding.templateId },
      `${template?.code || 'amazon-template'}.xlsx`,
      '模板下载成功'
    )
  }

  const serializeItem = (item) => ({
    id: item.id || 0,
    parentItemId: item.parentItemId || undefined,
    role: item.role,
    sku: item.sku,
    brand: item.brand,
    conditionType: item.conditionType,
    externalProductIdType: item.externalProductIdType,
    externalProductId: item.externalProductId,
    merchantSuggestedAsin: item.merchantSuggestedAsin,
    commonAttributes: item.commonAttributes || {},
    variationAttributes: item.variationAttributes || {},
    status: item.status,
    sharedImages: (item.sharedImages || []).map((image, index) => ({
      ...image,
      sort: index + 1
    })),
    marketplaces: (item.marketplaces || []).map((binding) => ({
      id: binding.id || 0,
      storeId: binding.storeId || undefined,
      templateId: binding.templateId,
      marketplaceId: binding.marketplaceId,
      siteCode: binding.siteCode,
      currencyCode: binding.currencyCode,
      offerPrice: binding.offerPrice,
      salePrice: binding.salePrice,
      quantity: binding.quantity,
      leadTimeToShip: binding.leadTimeToShip,
      merchantShippingGroup: binding.merchantShippingGroup,
      marketplaceAttributes: binding.marketplaceAttributes || {},
      profitProfile: serializeProfitProfile(binding.profitProfile),
      locales: dedupeLocales(binding.locales || []).map((locale) => ({
        id: locale.id || 0,
        localeCode: locale.localeCode,
        itemName: locale.itemName,
        bulletPoints: locale.bulletPoints || [],
        productDescription: locale.productDescription,
        searchTerms: locale.searchTerms || [],
        localizedAttributes: locale.localizedAttributes || {}
      })),
      images: (binding.images || []).map((image, index) => ({
        ...image,
        sort: index + 1
      }))
    }))
  })

  const buildPayload = () => ({
    family: {
      ...formModel.family
    },
    items: formModel.items.map(serializeItem)
  })

  const saveDraft = async () => {
    saveLoading.value = true
    try {
      await Promise.all(
        formModel.items.flatMap((item) =>
          item.marketplaces.map((binding) => (binding.templateId ? getTemplateDetail(binding.templateId) : Promise.resolve()))
        )
      )
      await saveAmazonListing(buildPayload())
      ElMessage.success('商品已保存')
      drawerVisible.value = false
      fetchTree()
    } finally {
      saveLoading.value = false
    }
  }

  const validateDraft = async () => {
    await Promise.all(
      formModel.items.flatMap((item) =>
        item.marketplaces.map((binding) => (binding.templateId ? getTemplateDetail(binding.templateId) : Promise.resolve()))
      )
    )
    const res = await validateAmazonListingItem(buildPayload())
    showValidationResult(res.data)
  }

  const validateFamily = async (row) => {
    const res = await validateAmazonListingSelected({
      familyIds: [row.familyId],
      itemIds: []
    })
    showValidationResult(res.data)
  }

  const validateSelectedRows = async () => {
    const res = await validateAmazonListingSelected(buildSelectedPayload())
    showValidationResult(res.data)
  }

  const showValidationResult = (data) => {
    const errors = data?.errors || []
    const warnings = data?.warnings || []
    if (!errors.length && !warnings.length) {
      ElMessage.success('校验通过')
      return
    }
    const lines = [...errors.map((item) => `错误: ${item.message}`), ...warnings.map((item) => `提示: ${item.message}`)]
    ElMessageBox.alert(lines.join('<br/>'), '校验结果', {
      dangerouslyUseHTMLString: true,
      confirmButtonText: '知道了'
    })
  }

  const buildSelectedPayload = () => {
    const familyIds = []
    const itemIds = []
    selectedRows.value.forEach((row) => {
      if (row.familyId) {
        familyIds.push(row.familyId)
      }
      if (row.id) {
        itemIds.push(row.id)
      }
    })
    return {
      familyIds: Array.from(new Set(familyIds)),
      itemIds: Array.from(new Set(itemIds))
    }
  }

  const exportSelectedRows = async () => {
    const res = await exportAmazonListingSelected(buildSelectedPayload())
    const url = res.data?.downloadUrl
    if (url) {
      window.open(url, '_blank')
      ElMessage.success(`导出文件已生成：${res.data?.fileName}`)
    }
  }

  const deleteFamilyRow = async (row) => {
    await ElMessageBox.confirm('确认删除整个变体族吗？这会删除当前 family 下所有商品与站点绑定。', '删除商品', {
      type: 'warning'
    })
    await deleteAmazonListing({ familyId: row.familyId })
    ElMessage.success('删除成功')
    fetchTree()
  }

  onMounted(() => {
    fetchTree()
    fetchTemplateOptions()
    fetchStoreOptions()
  })
</script>
