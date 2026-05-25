<template>
  <div class="flex flex-col gap-6">
    <section class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <el-button link type="primary" class="!px-0" @click="goBack">返回订单列表</el-button>
          <p class="text-xs tracking-[0.3em] text-slate-500">ORDER FULFILLMENT DETAIL</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">
            {{ detail?.amazonOrderId || `订单 #${orderID}` }}
          </h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            查看 Amazon 订单履约快照、1688 采购组、包裹档案、物流下单状态和 Amazon 发货回传结果。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <el-button @click="fetchDetail">刷新</el-button>
          <el-button type="primary" plain :disabled="!detail" @click="openSupportInbox">新建客服工单</el-button>
          <el-button
            v-if="canStart"
            type="warning"
            :loading="actionLoading === 'start'"
            @click="handleStart"
          >
            开始履约
          </el-button>
          <el-button
            v-if="canRetry"
            type="danger"
            plain
            :loading="actionLoading === 'retry'"
            @click="handleRetry"
          >
            重试履约
          </el-button>
          <el-button
            type="primary"
            plain
            :loading="actionLoading === 'print'"
            @click="handleSystemPrint"
          >
            打印系统发货单
          </el-button>
          <el-button
            v-if="canManualShipmentConfirm"
            type="primary"
            plain
            @click="manualShipmentDialogVisible = true"
          >
            手工录入运单并回传
          </el-button>
          <el-button
            v-if="detail?.printing?.officialPrintUrl"
            type="primary"
            @click="openOfficialPrint"
          >
            打开 Amazon 官方页
          </el-button>
        </div>
      </div>
    </section>

    <el-skeleton :loading="loading" animated :rows="10">
      <template #default>
        <section class="grid gap-4 xl:grid-cols-[2fr_1fr]">
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="mb-4 flex flex-wrap items-center gap-2">
              <el-tag size="small" :type="detail?.fulfillmentType === 'fbm' ? 'warning' : 'success'">
                {{ formatFulfillmentType(detail?.fulfillmentType) }}
              </el-tag>
              <el-tag size="small" :type="workflowTagType(detail?.workflowStatus)">
                {{ workflowLabel(detail?.workflowStatus) }}
              </el-tag>
              <el-tag size="small" :type="returnSummaryTagType(detail?.returnSummary?.status || detail?.returnSummaryStatus)">
                退货 {{ returnSummaryLabel(detail?.returnSummary?.status || detail?.returnSummaryStatus) }}
              </el-tag>
              <el-tag size="small" :type="statusTagType(detail?.procurementStatus)">
                采购 {{ statusLabel(detail?.procurementStatus) }}
              </el-tag>
              <el-tag size="small" :type="statusTagType(detail?.printStatus)">
                打印 {{ statusLabel(detail?.printStatus) }}
              </el-tag>
              <el-tag size="small" :type="statusTagType(detail?.logisticsStatus)">
                物流 {{ statusLabel(detail?.logisticsStatus) }}
              </el-tag>
              <el-tag size="small" :type="statusTagType(detail?.amazonFeedbackStatus)">
                Amazon {{ statusLabel(detail?.amazonFeedbackStatus) }}
              </el-tag>
            </div>
            <el-descriptions :column="2" border>
              <el-descriptions-item label="站点">{{ detail?.siteCode || '--' }}</el-descriptions-item>
              <el-descriptions-item label="Amazon 状态">{{ detail?.orderStatus || '--' }}</el-descriptions-item>
              <el-descriptions-item label="买家">{{ detail?.buyerName || '--' }}</el-descriptions-item>
              <el-descriptions-item label="邮箱">{{ detail?.buyerEmail || '--' }}</el-descriptions-item>
              <el-descriptions-item label="金额">{{ formatPrice(detail?.orderTotalAmount, detail?.currencyCode) }}</el-descriptions-item>
              <el-descriptions-item label="下单时间">{{ detail?.purchaseDate || '--' }}</el-descriptions-item>
              <el-descriptions-item label="最后同步">{{ detail?.lastSynchronizedAt || '--' }}</el-descriptions-item>
              <el-descriptions-item label="最后工作流">{{ detail?.lastWorkflowAt || '--' }}</el-descriptions-item>
              <el-descriptions-item label="异常代码">{{ detail?.exceptionCode || '--' }}</el-descriptions-item>
              <el-descriptions-item label="异常信息">{{ detail?.exceptionMessage || '--' }}</el-descriptions-item>
              <el-descriptions-item label="收货人">{{ detail?.address?.recipientName || '--' }}</el-descriptions-item>
              <el-descriptions-item label="电话">{{ detail?.address?.phone || '--' }}</el-descriptions-item>
              <el-descriptions-item label="收货地址" :span="2">
                {{ formatAddress(detail?.address) }}
              </el-descriptions-item>
            </el-descriptions>
          </div>

          <div class="rounded-2xl border border-slate-200 bg-slate-50 p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/40">
            <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">打印与回传</div>
            <div class="space-y-3 text-sm text-slate-600 dark:text-slate-300">
              <div class="rounded-xl border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/60">
                <div class="mb-1 font-medium">打印状态</div>
                <div>{{ statusLabel(detail?.printStatus) }}</div>
                <div class="text-xs text-slate-500 dark:text-slate-400">
                  系统发货单与 Amazon 官方页均保持手动触发
                </div>
              </div>
              <div class="rounded-xl border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/60">
                <div class="mb-1 font-medium">系统发货单</div>
                <div class="text-xs break-all text-slate-500 dark:text-slate-400">
                  {{ detail?.printing?.systemPrintUrl || '--' }}
                </div>
              </div>
              <div class="rounded-xl border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/60">
                <div class="mb-1 font-medium">Amazon 官方页</div>
                <div class="text-xs break-all text-slate-500 dark:text-slate-400">
                  {{ detail?.printing?.officialPrintUrl || '--' }}
                </div>
              </div>
              <div class="rounded-xl border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/60">
                <div class="mb-1 font-medium">Amazon 回传状态</div>
                <div>{{ statusLabel(detail?.amazonFeedbackStatus) }}</div>
                <div class="text-xs text-slate-500 dark:text-slate-400">
                  发货确认时间 {{ detail?.shipmentConfirmedAt || '--' }}
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
          <div class="mb-4 flex flex-col gap-2 xl:flex-row xl:items-center xl:justify-between">
            <div>
              <div class="text-base font-semibold text-slate-900 dark:text-slate-100">财务利润</div>
              <div class="text-sm text-slate-500 dark:text-slate-400">展示当前订单的权责/现金两套利润快照、结算匹配和应收状态。</div>
            </div>
            <div class="flex flex-wrap gap-2">
              <el-tag size="small" :type="financeStatusTagType(detail?.settlementMatchStatus)">
                结算 {{ formatFinanceStatus(detail?.settlementMatchStatus) }}
              </el-tag>
              <el-tag size="small" :type="financeStatusTagType(detail?.receivableStatus)">
                应收 {{ formatFinanceStatus(detail?.receivableStatus) }}
              </el-tag>
            </div>
          </div>

          <div v-if="financeSnapshots.length" class="grid gap-4 xl:grid-cols-2">
            <div
              v-for="snapshot in financeSnapshots"
              :key="`${snapshot.basisType}-${snapshot.dateView}`"
              class="rounded-2xl border border-slate-200 p-4 dark:border-slate-700"
            >
              <div class="mb-3 flex flex-wrap items-center gap-2">
                <el-tag size="small" :type="snapshot.basisType === 'cash' ? 'warning' : 'success'">
                  {{ snapshot.basisType === 'cash' ? '现金口径' : '权责口径' }}
                </el-tag>
                <el-tag size="small" :type="financeStatusTagType(snapshot.settlementMatchStatus)">
                  {{ formatFinanceStatus(snapshot.settlementMatchStatus) }}
                </el-tag>
                <span class="text-sm text-slate-500 dark:text-slate-400">{{ financeSnapshotDateLabel(snapshot) }}</span>
              </div>
              <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                <div class="rounded-xl bg-slate-50 p-3 dark:bg-slate-900/40">
                  <div class="text-xs text-slate-500 dark:text-slate-400">销售额</div>
                  <div class="mt-1 text-lg font-semibold text-slate-900 dark:text-slate-100">
                    {{ formatPrice(snapshot.costBreakdown?.revenueCny, 'CNY') }}
                  </div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3 dark:bg-slate-900/40">
                  <div class="text-xs text-slate-500 dark:text-slate-400">毛利润</div>
                  <div class="mt-1 text-lg font-semibold" :class="profitClass(snapshot.grossProfitCny)">
                    {{ formatPrice(snapshot.grossProfitCny, 'CNY') }}
                  </div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3 dark:bg-slate-900/40">
                  <div class="text-xs text-slate-500 dark:text-slate-400">净利润</div>
                  <div class="mt-1 text-lg font-semibold" :class="profitClass(snapshot.netProfitCny)">
                    {{ formatPrice(snapshot.netProfitCny, 'CNY') }}
                  </div>
                </div>
              </div>
              <div class="mt-4 grid gap-2 text-sm text-slate-600 dark:text-slate-300 md:grid-cols-2 xl:grid-cols-3">
                <div>采购：{{ formatPrice(snapshot.costBreakdown?.procurementCostCny, 'CNY') }}</div>
                <div>头程：{{ formatPrice(snapshot.costBreakdown?.firstLegCostCny, 'CNY') }}</div>
                <div>佣金：{{ formatPrice(snapshot.costBreakdown?.amazonReferralFeeCny, 'CNY') }}</div>
                <div>FBA配送：{{ formatPrice(snapshot.costBreakdown?.fbaFulfillmentFeeCny, 'CNY') }}</div>
                <div>仓储：{{ formatPrice(snapshot.costBreakdown?.storageFeeCny, 'CNY') }}</div>
                <div>广告：{{ formatPrice(snapshot.costBreakdown?.adCostCny, 'CNY') }}</div>
                <div>提现：{{ formatPrice(snapshot.costBreakdown?.withdrawalFeeCny, 'CNY') }}</div>
                <div>卡费：{{ formatPrice(snapshot.costBreakdown?.cardFeeCny, 'CNY') }}</div>
                <div>退货损耗：{{ formatPrice(snapshot.costBreakdown?.returnLossCny, 'CNY') }}</div>
                <div>退款：{{ formatPrice(snapshot.costBreakdown?.refundCostCny, 'CNY') }}</div>
                <div>补偿：{{ formatPrice(snapshot.costBreakdown?.reimbursementCny, 'CNY') }}</div>
                <div>赔付：{{ formatPrice(snapshot.costBreakdown?.compensationCny, 'CNY') }}</div>
                <div>已匹配结算：{{ formatPrice(snapshot.matchedSettlementCny, 'CNY') }}</div>
                <div>估算成本：{{ formatPrice(snapshot.estimatedCostCny, 'CNY') }}</div>
                <div>估算条目：{{ snapshot.estimatedEntryCount || 0 }}</div>
              </div>
            </div>
          </div>
          <el-empty v-else description="尚未生成财务利润快照" :image-size="80" />
        </section>

        <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
          <div class="mb-4 flex flex-col gap-2 xl:flex-row xl:items-center xl:justify-between">
            <div>
              <div class="text-base font-semibold text-slate-900 dark:text-slate-100">关联退货与转寄候选</div>
              <div class="text-sm text-slate-500 dark:text-slate-400">展示当前订单已关联的退货单，以及可用于“退货直发新买家”的候选记录。</div>
            </div>
          </div>

          <div v-if="detail?.linkedReturns?.length" class="space-y-4">
            <div
              v-for="returnOrder in detail.linkedReturns"
              :key="returnOrder.id"
              class="rounded-2xl border border-slate-200 p-4 dark:border-slate-700"
            >
              <div class="mb-3 flex flex-wrap items-center gap-2">
                <el-tag size="small" :type="returnSummaryTagType(returnOrder.linkStatus)">
                  {{ returnSummaryLabel(returnOrder.linkStatus) }}
                </el-tag>
                <span class="font-medium text-slate-900 dark:text-slate-100">{{ returnOrder.amazonRmaId || '--' }}</span>
                <span class="text-sm text-slate-500 dark:text-slate-400">{{ returnOrder.returnRequestStatus || '--' }}</span>
              </div>
              <div class="grid gap-2 text-sm text-slate-600 dark:text-slate-300 md:grid-cols-2">
                <div>退款：{{ formatPrice(returnOrder.refundAmount, returnOrder.refundCurrency) }}</div>
                <div>面单费：{{ formatPrice(returnOrder.labelCost, returnOrder.labelCurrency) }}</div>
                <div>退货运单：{{ returnOrder.trackingId || '--' }}</div>
                <div>异常：{{ returnOrder.exceptionMessage || '--' }}</div>
              </div>
              <div class="mt-3">
                <el-button type="primary" link @click="openReturnDetail(returnOrder)">打开退货详情</el-button>
              </div>
            </div>
          </div>
          <el-empty v-else description="暂无关联退货" :image-size="80" />

          <div v-if="detail?.returnRedirectCandidates?.length" class="mt-4">
            <div class="mb-3 text-sm font-medium text-slate-900 dark:text-slate-100">转寄候选</div>
            <div class="space-y-3">
              <div
                v-for="candidate in detail.returnRedirectCandidates"
                :key="`${candidate.returnItemId}-${candidate.targetOrderItemId}`"
                class="rounded-2xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-800/40 dark:bg-amber-900/20"
              >
                <div class="flex flex-wrap items-center gap-2">
                  <el-tag size="small" type="warning">退货直发新买家</el-tag>
                  <span class="font-medium text-slate-900 dark:text-slate-100">
                    退货项 {{ candidate.returnItemId }} -> 新单 {{ candidate.amazonOrderId || candidate.targetOrderId }}
                  </span>
                </div>
                <div class="mt-2 grid gap-2 text-sm text-slate-600 dark:text-slate-300 md:grid-cols-2">
                  <div>SKU：{{ candidate.sellerSku || '--' }}</div>
                  <div>数量：{{ candidate.quantity || 0 }}</div>
                  <div>近30天销量：{{ candidate.soldQtyLast30d || 0 }}</div>
                  <div>退货成本：{{ formatPrice(candidate.intakeFeeCny, 'CNY') }}</div>
                </div>
                <div class="mt-2 text-sm text-slate-500 dark:text-slate-400">
                  {{ candidate.reason || '--' }}
                </div>
                <div class="mt-3">
                  <el-button
                    type="warning"
                    size="small"
                    :loading="actionLoading === `redirect-${candidate.returnItemId}`"
                    @click="handleConfirmRedirectCandidate(candidate)"
                  >
                    确认转寄
                  </el-button>
                </div>
              </div>
            </div>
          </div>

          <div v-if="redirectReservedItems.length" class="mt-4">
            <div class="mb-3 text-sm font-medium text-slate-900 dark:text-slate-100">已锁定的退货转寄</div>
            <div class="space-y-3">
              <div
                v-for="item in redirectReservedItems"
                :key="item.id"
                class="rounded-2xl border border-slate-200 p-4 dark:border-slate-700"
              >
                <div class="flex flex-wrap items-center gap-2">
                  <el-tag size="small" type="primary">转寄占用中</el-tag>
                  <span class="font-medium text-slate-900 dark:text-slate-100">{{ item.sellerSku || `订单项 ${item.id}` }}</span>
                </div>
                <div class="mt-2 grid gap-2 text-sm text-slate-600 dark:text-slate-300 md:grid-cols-2">
                  <div>退货项：{{ item.reservedReturnItemId || '--' }}</div>
                  <div>状态：{{ item.returnRedirectStatus || '--' }}</div>
                </div>
                <div class="mt-3">
                  <el-button
                    type="danger"
                    plain
                    size="small"
                    :loading="actionLoading === `release-${item.reservedReturnItemId}`"
                    @click="handleReleaseRedirectReservation(item)"
                  >
                    回退正常采购
                  </el-button>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
          <div class="mb-4 flex items-center justify-between">
            <div>
              <div class="text-base font-semibold text-slate-900 dark:text-slate-100">订单项与绑定状态</div>
              <div class="text-sm text-slate-500 dark:text-slate-400">检查 SKU 是否已绑定 1688 商品、规格映射是否完整，以及采购单号回写情况。</div>
            </div>
          </div>

          <el-table :data="detail?.items || []" row-key="id" border>
            <el-table-column label="SKU / 标题" min-width="260">
              <template #default="{ row }">
                <div class="flex flex-col gap-1">
                  <span class="font-medium text-slate-900 dark:text-slate-100">{{ row.sellerSku || '--' }}</span>
                  <span class="text-xs text-slate-500 dark:text-slate-400">{{ row.title || '--' }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="数量" width="120">
              <template #default="{ row }">{{ row.quantityOrdered || 0 }} / 已发 {{ row.quantityShipped || 0 }}</template>
            </el-table-column>
            <el-table-column label="绑定状态" min-width="220">
              <template #default="{ row }">
                <div class="flex flex-col gap-1">
                  <el-tag size="small" :type="row.activeBindingId ? 'success' : 'danger'">
                    {{ row.activeBindingId ? '已绑定1688' : '未绑定1688' }}
                  </el-tag>
                  <span class="text-xs text-slate-500 dark:text-slate-400">
                    Listing {{ row.listingItemId || '--' }} / Binding {{ row.activeBindingId || '--' }}
                  </span>
                  <span class="text-xs text-slate-500 dark:text-slate-400">
                    规格 {{ row.selected1688SkuKey || '--' }}
                  </span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="1688 商品" min-width="220">
              <template #default="{ row }">
                <div class="flex flex-col gap-1">
                  <span>{{ row.boundProduct?.title || row.boundProduct?.offerId || '--' }}</span>
                  <span class="text-xs text-slate-500 dark:text-slate-400">
                    {{ row.boundProduct?.shopName || '--' }}
                  </span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="采购回写" min-width="160">
              <template #default="{ row }">
                <div class="flex flex-col gap-1">
                  <span>{{ row.purchaseOrderNo || '--' }}</span>
                  <span class="text-xs text-slate-500 dark:text-slate-400">
                    {{ statusLabel(row.purchaseStatus) }} / 数量 {{ row.purchaseQuantity || '--' }}
                  </span>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </section>

        <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
          <div class="mb-4 flex flex-col gap-2 xl:flex-row xl:items-center xl:justify-between">
            <div>
              <div class="text-base font-semibold text-slate-900 dark:text-slate-100">包裹档案覆盖</div>
              <div class="text-sm text-slate-500 dark:text-slate-400">如果 1688 推断的重量尺寸不完整，可在这里按订单项人工修正，修正后再重试履约。</div>
            </div>
            <el-button type="primary" :loading="packageSaving" @click="handleSavePackageOverrides">保存包裹信息</el-button>
          </div>

          <div class="grid gap-4 xl:grid-cols-2">
            <div
              v-for="draft in packageDrafts"
              :key="draft.orderItemId"
              class="rounded-2xl border border-slate-200 p-4 dark:border-slate-700"
            >
              <div class="mb-3">
                <div class="font-medium text-slate-900 dark:text-slate-100">{{ draft.sellerSku || '--' }}</div>
                <div class="text-xs text-slate-500 dark:text-slate-400">{{ draft.title || '--' }}</div>
              </div>
              <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
                <el-input-number v-model="draft.weightKg" :precision="3" :min="0" :step="0.01" controls-position="right" />
                <el-input-number v-model="draft.lengthCm" :precision="2" :min="0" :step="0.1" controls-position="right" />
                <el-input-number v-model="draft.widthCm" :precision="2" :min="0" :step="0.1" controls-position="right" />
                <el-input-number v-model="draft.heightCm" :precision="2" :min="0" :step="0.1" controls-position="right" />
                <el-switch
                  v-model="draft.containsBattery"
                  inline-prompt
                  active-text="带电"
                  inactive-text="普货"
                />
              </div>
            </div>
          </div>
        </section>

        <section class="grid gap-4 xl:grid-cols-2">
          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">1688 采购组</div>
            <div v-if="detail?.procurementGroups?.length" class="space-y-4">
              <div
                v-for="group in detail.procurementGroups"
                :key="group.id"
                class="rounded-2xl border border-slate-200 p-4 dark:border-slate-700"
              >
                <div class="mb-3 flex flex-wrap items-center gap-2">
                  <el-tag size="small" :type="statusTagType(group.status)">{{ statusLabel(group.status) }}</el-tag>
                  <el-tag size="small" :type="statusTagType(group.taskStatus)">任务 {{ statusLabel(group.taskStatus) }}</el-tag>
                  <span class="text-sm font-medium text-slate-900 dark:text-slate-100">{{ group.shopName || group.shopGroupKey }}</span>
                </div>
                <div class="grid gap-2 text-sm text-slate-600 dark:text-slate-300">
                  <div>任务令牌：{{ group.taskToken || '--' }}</div>
                  <div>1688 采购单：{{ group.orderNo1688 || '--' }}</div>
                  <div>开始时间：{{ group.startedAt || '--' }}</div>
                  <div>完成时间：{{ group.finishedAt || '--' }}</div>
                  <div>异常：{{ group.errorMessage || '--' }}</div>
                </div>
                <div class="mt-3 flex flex-wrap gap-2">
                  <el-button
                    v-if="group.orderUrl"
                    type="primary"
                    link
                    @click="openExternal(group.orderUrl)"
                  >
                    打开 1688 订单
                  </el-button>
                  <el-button type="warning" link @click="openProcurementTask(group)">打开 1688 采购任务</el-button>
                </div>
                <div class="mt-3 flex flex-wrap gap-2">
                  <el-tag
                    v-for="item in group.items || []"
                    :key="item.id"
                    size="small"
                    type="info"
                  >
                    {{ procurementItemLabel(item) }}
                  </el-tag>
                </div>
              </div>
            </div>
            <el-empty v-else description="尚未生成采购组" :image-size="80" />
          </div>

          <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900/60">
            <div class="mb-4 text-base font-semibold text-slate-900 dark:text-slate-100">物流单与揽收</div>
            <div v-if="detail?.shipments?.length" class="space-y-4">
              <div
                v-for="shipment in detail.shipments"
                :key="shipment.id"
                class="rounded-2xl border border-slate-200 p-4 dark:border-slate-700"
              >
                <div class="mb-3 flex flex-wrap items-center gap-2">
                  <el-tag size="small" :type="statusTagType(shipment.status)">{{ statusLabel(shipment.status) }}</el-tag>
                  <el-tag size="small" :type="statusTagType(shipment.amazonSubmitStatus)">
                    Amazon {{ statusLabel(shipment.amazonSubmitStatus) }}
                  </el-tag>
                </div>
                <div class="grid gap-2 text-sm text-slate-600 dark:text-slate-300">
                  <div>来源：{{ shipmentSourceLabel(shipment.source) }}</div>
                  <div>服务商：{{ shipment.provider || '--' }} / {{ shipment.channelName || '--' }}</div>
                  <div>承运商：{{ shipment.carrierCode || shipment.carrierName || '--' }}</div>
                  <div>发货方式：{{ shipment.shippingMethod || '--' }}</div>
                  <div>运单号：{{ shipment.trackingNo || '--' }}</div>
                  <div>发货时间：{{ shipment.shippedAt || '--' }}</div>
                  <div>预约揽收：{{ shipment.reservedPickupAt || '--' }}</div>
                  <div>实际揽收：{{ shipment.actualPickupAt || '--' }}</div>
                  <div>回传尝试：{{ shipment.amazonSubmitRetryCount || 0 }}</div>
                  <div>最后回传：{{ shipment.amazonSubmitAttemptedAt || '--' }}</div>
                  <div>回传异常：{{ shipment.amazonSubmitLastError || '--' }}</div>
                  <div>错误：{{ shipment.errorMessage || '--' }}</div>
                </div>
                <div class="mt-3 flex flex-wrap gap-3">
                  <el-button v-if="shipment.labelUrl" type="primary" link @click="openExternal(shipment.labelUrl)">打开面单</el-button>
                  <el-button
                    v-if="shipment.trackingNo && shipment.amazonSubmitStatus !== 'submitted'"
                    type="warning"
                    link
                    :loading="actionLoading === `retry-shipment-${shipment.id}`"
                    @click="handleRetryShipmentConfirm(shipment)"
                  >
                    重试回传
                  </el-button>
                </div>
              </div>
            </div>
            <el-empty v-else description="尚未创建物流单" :image-size="80" />
          </div>
        </section>

        <el-dialog v-model="manualShipmentDialogVisible" title="手工录入运单并回传" width="560px" destroy-on-close>
          <el-form label-width="110px">
            <el-form-item label="承运商代码">
              <el-input v-model="manualShipmentForm.carrierCode" placeholder="可选，例如 YUNEXPRESS" />
            </el-form-item>
            <el-form-item label="承运商名称">
              <el-input v-model="manualShipmentForm.carrierName" placeholder="必填，例如 云途" />
            </el-form-item>
            <el-form-item label="发货方式">
              <el-input v-model="manualShipmentForm.shippingMethod" placeholder="可选，例如 Standard" />
            </el-form-item>
            <el-form-item label="追踪号">
              <el-input v-model="manualShipmentForm.trackingNo" placeholder="必填" />
            </el-form-item>
            <el-form-item label="发货时间">
              <el-date-picker
                v-model="manualShipmentForm.shippedAt"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                class="!w-full"
                placeholder="不填则默认当前时间"
              />
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="manualShipmentDialogVisible = false">取消</el-button>
            <el-button
              type="primary"
              :loading="actionLoading === 'manual-shipment'"
              @click="handleManualShipmentConfirm"
            >
              保存并回传
            </el-button>
          </template>
        </el-dialog>
      </template>
    </el-skeleton>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'

import { findAmazon1688ProcurementTask } from '@/api/amazonProcurement'
import { confirmAmazonReturnRedirect, releaseAmazonReturnRedirect } from '@/api/amazonReturn'
import {
  findAmazonOrder,
  manualAmazonOrderShipmentConfirm,
  printAmazonOrderSystemSlip,
  retryAmazonOrderShipmentConfirm,
  retryAmazonOrderFulfillment,
  startAmazonOrderFulfillment,
  updateAmazonOrderPackageOverrides
} from '@/api/amazonOrder'

defineOptions({
  name: 'AmazonOrderFulfillmentDetail'
})

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const packageSaving = ref(false)
const actionLoading = ref('')
const detail = ref(null)
const packageDrafts = ref([])
const manualShipmentDialogVisible = ref(false)
const manualShipmentForm = reactive({
  carrierCode: '',
  carrierName: '',
  shippingMethod: '',
  trackingNo: '',
  shippedAt: ''
})

const orderID = computed(() => Number(route.params.id || route.query.id || 0))
const itemMap = computed(() => {
  const pairs = (detail.value?.items || []).map((item) => [item.id, item])
  return Object.fromEntries(pairs)
})
const financeSnapshots = computed(() =>
  [detail.value?.financeSnapshotAccrual, detail.value?.financeSnapshotCash].filter(Boolean)
)
const redirectReservedItems = computed(() =>
  (detail.value?.items || []).filter((item) => item.supplySource === 'return_redirect' && item.reservedReturnItemId)
)

const canStart = computed(() => detail.value?.fulfillmentType === 'fbm' && detail.value?.workflowStatus === 'fbm_pending')
const canRetry = computed(() => ['fulfillment_failed', 'fbm_exception'].includes(detail.value?.workflowStatus))
const canManualShipmentConfirm = computed(() =>
  detail.value?.fulfillmentType === 'fbm' &&
  ['Unshipped', 'PartiallyShipped'].includes(detail.value?.orderStatus) &&
  detail.value?.amazonFeedbackStatus !== 'submitted'
)

const fetchDetail = async () => {
  if (!orderID.value) {
    return
  }
  loading.value = true
  try {
    const res = await findAmazonOrder({ id: orderID.value })
    if (res.code === 0) {
      detail.value = res.data || null
      packageDrafts.value = (res.data?.items || []).map((item) => ({
        orderItemId: item.id,
        listingItemId: item.listingItemId,
        sellerSku: item.sellerSku,
        title: item.title,
        weightKg: numberOrNull(item.fulfillmentProfile?.weightKg),
        lengthCm: numberOrNull(item.fulfillmentProfile?.lengthCm),
        widthCm: numberOrNull(item.fulfillmentProfile?.widthCm),
        heightCm: numberOrNull(item.fulfillmentProfile?.heightCm),
        containsBattery: Boolean(item.fulfillmentProfile?.containsBattery)
      }))
    }
  } finally {
    loading.value = false
  }
}

const goBack = () => {
  router.push({ name: 'amazonOrderManager' })
}

const openSupportInbox = () => {
  if (!detail.value) {
    return
  }
  router.push({
    name: 'amazonSupportInbox',
    query: {
      compose: '1',
      caseType: 'after_sales',
      storeId: String(detail.value.storeId || ''),
      siteCode: detail.value.siteCode || '',
      orderId: String(detail.value.id || ''),
      buyerName: detail.value.buyerName || '',
      buyerEmail: detail.value.buyerEmail || '',
      amazonOrderId: detail.value.amazonOrderId || ''
    }
  })
}

const openReturnDetail = (row) => {
  router.push({
    name: 'amazonReturnDetail',
    params: { id: row.id }
  })
}

const handleConfirmRedirectCandidate = async (candidate) => {
  await ElMessageBox.confirm('确认使用当前退货件直发给这个待发货订单吗？系统会锁定该订单并跳过 1688 采购。', '确认转寄', {
    type: 'warning'
  })
  actionLoading.value = `redirect-${candidate.returnItemId}`
  try {
    const res = await confirmAmazonReturnRedirect({
      returnItemId: candidate.returnItemId,
      targetOrderItemId: candidate.targetOrderItemId
    })
    if (res.code === 0) {
      ElMessage.success('已确认转寄到当前订单')
      await fetchDetail()
    }
  } finally {
    actionLoading.value = ''
  }
}

const handleReleaseRedirectReservation = async (item) => {
  if (!item?.reservedReturnItemId) {
    return
  }
  await ElMessageBox.confirm('确认释放当前退货占用并恢复正常采购吗？', '回退正常采购', {
    type: 'warning'
  })
  actionLoading.value = `release-${item.reservedReturnItemId}`
  try {
    const res = await releaseAmazonReturnRedirect({ returnItemId: item.reservedReturnItemId })
    if (res.code === 0) {
      ElMessage.success('已恢复正常采购链路')
      await fetchDetail()
    }
  } finally {
    actionLoading.value = ''
  }
}

const handleStart = async () => {
  await ElMessageBox.confirm('确认启动当前订单的 FBM 履约流程吗？', '开始履约', { type: 'warning' })
  actionLoading.value = 'start'
  try {
    const res = await startAmazonOrderFulfillment({ id: orderID.value })
    if (res.code === 0) {
      ElMessage.success('履约任务已创建')
      await fetchDetail()
    }
  } finally {
    actionLoading.value = ''
  }
}

const handleRetry = async () => {
  await ElMessageBox.confirm('确认重试当前订单的履约流程吗？', '重试履约', { type: 'warning' })
  actionLoading.value = 'retry'
  try {
    const res = await retryAmazonOrderFulfillment({ id: orderID.value })
    if (res.code === 0) {
      ElMessage.success('履约已重新触发')
      await fetchDetail()
    }
  } finally {
    actionLoading.value = ''
  }
}

const handleSystemPrint = async () => {
  actionLoading.value = 'print'
  try {
    const res = await printAmazonOrderSystemSlip({ id: orderID.value })
    if (res.code === 0) {
      const url = buildPrintURL(res.data)
      if (url) {
        window.open(url, '_blank', 'noopener,noreferrer')
      }
      ElMessage.success('已生成打印页')
      await fetchDetail()
    }
  } finally {
    actionLoading.value = ''
  }
}

const resetManualShipmentForm = () => {
  manualShipmentForm.carrierCode = ''
  manualShipmentForm.carrierName = ''
  manualShipmentForm.shippingMethod = ''
  manualShipmentForm.trackingNo = ''
  manualShipmentForm.shippedAt = ''
}

const handleManualShipmentConfirm = async () => {
  if (!manualShipmentForm.carrierName.trim()) {
    ElMessage.warning('请填写承运商名称')
    return
  }
  if (!manualShipmentForm.trackingNo.trim()) {
    ElMessage.warning('请填写追踪号')
    return
  }
  actionLoading.value = 'manual-shipment'
  try {
    const res = await manualAmazonOrderShipmentConfirm({
      orderId: orderID.value,
      carrierCode: manualShipmentForm.carrierCode.trim(),
      carrierName: manualShipmentForm.carrierName.trim(),
      shippingMethod: manualShipmentForm.shippingMethod.trim(),
      trackingNo: manualShipmentForm.trackingNo.trim(),
      shippedAt: manualShipmentForm.shippedAt || ''
    })
    if (res.code === 0) {
      ElMessage.success(res.msg || '运单已保存')
      manualShipmentDialogVisible.value = false
      resetManualShipmentForm()
      await fetchDetail()
    }
  } finally {
    actionLoading.value = ''
  }
}

const handleRetryShipmentConfirm = async (shipment) => {
  actionLoading.value = `retry-shipment-${shipment.id}`
  try {
    const res = await retryAmazonOrderShipmentConfirm({ shipmentId: shipment.id })
    if (res.code === 0) {
      ElMessage.success(res.msg || '已触发重试')
      await fetchDetail()
    }
  } finally {
    actionLoading.value = ''
  }
}

const openOfficialPrint = () => {
  openExternal(detail.value?.printing?.officialPrintUrl)
}

const handleSavePackageOverrides = async () => {
  packageSaving.value = true
  try {
    const payload = {
      id: orderID.value,
      items: packageDrafts.value.map((draft) => ({
        orderItemId: draft.orderItemId,
        listingItemId: draft.listingItemId,
        weightKg: numberOrNull(draft.weightKg),
        lengthCm: numberOrNull(draft.lengthCm),
        widthCm: numberOrNull(draft.widthCm),
        heightCm: numberOrNull(draft.heightCm),
        containsBattery: typeof draft.containsBattery === 'boolean' ? draft.containsBattery : false
      }))
    }
    const res = await updateAmazonOrderPackageOverrides(payload)
    if (res.code === 0) {
      ElMessage.success('包裹信息已保存')
      await fetchDetail()
    }
  } finally {
    packageSaving.value = false
  }
}

const procurementItemLabel = (item) => {
  const orderItem = itemMap.value[item.orderItemId]
  return `${orderItem?.sellerSku || `Item#${item.orderItemId}`}${item.selected1688SkuKey ? ` / ${item.selected1688SkuKey}` : ''}`
}

const openExternal = (target) => {
  if (!target) {
    return
  }
  window.open(target, '_blank', 'noopener,noreferrer')
}

const openProcurementTask = async (group) => {
  if (!group?.taskToken) {
    ElMessage.warning('当前采购组缺少任务令牌')
    return
  }
  const res = await findAmazon1688ProcurementTask({ taskToken: group.taskToken })
  if (res.code !== 0) {
    return
  }
  const firstItem = (res.data?.items || []).find((item) => item.productUrl)
  if (!firstItem?.productUrl) {
    ElMessage.warning('当前采购任务缺少 1688 商品链接')
    return
  }
  openExternal(appendProcurementTaskParam(firstItem.productUrl, group.taskToken))
}

const buildPrintURL = (printing) => {
  const baseURL = String(printing?.systemPrintUrl || detail.value?.printing?.systemPrintUrl || '').trim()
  if (!baseURL) {
    return ''
  }
  const token = String(printing?.systemPrintToken || detail.value?.printing?.systemPrintToken || '').trim()
  if (!token) {
    return baseURL
  }
  const separator = baseURL.includes('?') ? '&' : '?'
  return `${baseURL}${separator}token=${encodeURIComponent(token)}`
}

const appendProcurementTaskParam = (rawUrl, taskToken) => {
  const target = String(rawUrl || '').trim()
  const token = String(taskToken || '').trim()
  if (!target || !token) {
    return target
  }
  try {
    const url = new URL(target)
    url.searchParams.set('__gva1688ProcurementTask', token)
    return url.toString()
  } catch (error) {
    return target
  }
}

const formatPrice = (price, currencyCode) => {
  if (price === null || typeof price === 'undefined') {
    return '--'
  }
  return `${currencyCode || ''} ${Number(price).toFixed(2)}`.trim()
}

const profitClass = (value) => {
  if (Number(value || 0) > 0) return 'text-emerald-600'
  if (Number(value || 0) < 0) return 'text-rose-600'
  return 'text-slate-900 dark:text-slate-100'
}

const formatFinanceStatus = (value) => {
  switch (value) {
    case 'exact':
      return '精确'
    case 'manual':
      return '手工'
    case 'fuzzy':
      return '模糊'
    case 'unmatched':
      return '未匹配'
    case 'settled':
      return '已结清'
    case 'partial':
      return '部分'
    case 'open':
      return '开放'
    default:
      return value || '--'
  }
}

const financeStatusTagType = (value) => {
  switch (value) {
    case 'exact':
    case 'settled':
      return 'success'
    case 'manual':
    case 'partial':
      return 'warning'
    case 'fuzzy':
      return 'info'
    case 'unmatched':
    case 'open':
      return 'danger'
    default:
      return 'info'
  }
}

const financeSnapshotDateLabel = (snapshot) => {
  if (!snapshot) return '--'
  if (snapshot.dateView === 'shipment') {
    return `发货日 ${snapshot.businessDate || snapshot.shipmentDate || '--'}`
  }
  return `下单日 ${snapshot.businessDate || snapshot.purchaseDate || '--'}`
}

const formatAddress = (address) => {
  if (!address) {
    return '--'
  }
  const values = [
    address.addressLine1,
    address.addressLine2,
    address.addressLine3,
    address.city,
    address.stateOrRegion,
    address.postalCode,
    address.countryCode
  ].filter(Boolean)
  return values.join(' / ') || '--'
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
    case 'fbm_waiting_return_redirect':
      return '待转寄'
    case 'fbm_return_redirect_shipped':
      return '转寄已发货'
    case 'fulfillment_running':
      return '执行中'
    case 'fulfillment_completed':
      return '已完成'
    case 'fulfillment_failed':
      return '执行失败'
    case 'fba_closed':
      return 'FBA关闭'
    case 'closed':
      return '已关闭'
    default:
      return value || '--'
  }
}

const workflowTagType = (value) => {
  switch (value) {
    case 'fbm_pending':
    case 'fbm_waiting_return_redirect':
      return 'warning'
    case 'fbm_exception':
    case 'fulfillment_failed':
      return 'danger'
    case 'fulfillment_running':
      return 'primary'
    case 'fulfillment_completed':
    case 'fbm_return_redirect_shipped':
    case 'fba_closed':
      return 'success'
    default:
      return 'info'
  }
}

const returnSummaryLabel = (value) => {
  switch (value) {
    case 'none':
      return '无退货'
    case 'open':
      return '开放'
    case 'processing':
      return '处理中'
    case 'closed':
      return '已关闭'
    case 'closed_returnless':
      return '赠送关闭'
    case 'exception':
      return '异常'
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

const returnSummaryTagType = (value) => {
  switch (value) {
    case 'none':
      return 'info'
    case 'open':
      return 'warning'
    case 'processing':
    case 'linked':
      return 'primary'
    case 'closed':
    case 'closed_returnless':
      return 'success'
    case 'exception':
    case 'missing_order':
    case 'missing_item':
    case 'ambiguous':
    case 'manual_review':
      return 'danger'
    default:
      return 'info'
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
    case 'skipped':
      return '已跳过'
    case 'created':
      return '已下单'
    case 'picked_up':
      return '已揽收'
    case 'submitted':
      return '已回传'
    case 'return_redirect_pending':
      return '待转寄'
    case 'return_redirect_booked':
      return '已转寄'
    case 'success':
      return '成功'
    case 'opened':
      return '已打开'
    default:
      return value || '--'
  }
}

const shipmentSourceLabel = (value) => {
  switch (value) {
    case 'auto_provider':
      return '自动物流'
    case 'manual_entry':
      return '手工录入'
    case 'return_redirect':
      return '退货转寄'
    default:
      return value || '--'
  }
}

const statusTagType = (value) => {
  switch (value) {
    case 'ready':
    case 'running':
      return 'primary'
    case 'completed':
    case 'created':
    case 'picked_up':
    case 'submitted':
    case 'return_redirect_booked':
    case 'success':
      return 'success'
    case 'failed':
      return 'danger'
    case 'blocked':
    case 'return_redirect_pending':
      return 'warning'
    default:
      return 'info'
  }
}

const numberOrNull = (value) => {
  if (value === null || typeof value === 'undefined' || value === '') {
    return null
  }
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}

watch(orderID, () => {
  fetchDetail()
}, { immediate: true })

watch(manualShipmentDialogVisible, (visible) => {
  if (!visible) {
    resetManualShipmentForm()
  }
})
</script>
