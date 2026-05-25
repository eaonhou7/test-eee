<template>
  <div class="flex flex-col gap-6">
    <section class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.3em] text-slate-500">AMAZON SUPPORT INBOX</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">Amazon 客服消息</h1>
          <p class="max-w-4xl text-sm text-slate-600 dark:text-slate-300">
            汇总 Buyer Message、售后、退货、差评与 A-to-Z 工单，统一做 SLA 跟踪、上下文查看、模板回复和人工确认发送。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <el-button type="primary" @click="openCaseDialog()">新建工单</el-button>
          <el-button @click="triggerImport">批量导入</el-button>
          <el-button @click="openTemplateManager">模板管理</el-button>
          <el-button @click="fetchCaseList">刷新</el-button>
        </div>
      </div>

      <div class="mt-5 grid gap-3 md:grid-cols-5">
        <button
          v-for="card in summaryCards"
          :key="card.key"
          type="button"
          class="rounded-lg border px-4 py-4 text-left transition"
          :class="summaryCardClass(card.key)"
          @click="applySummaryFilter(card.key)"
        >
          <div class="text-xs text-slate-500 dark:text-slate-400">{{ card.label }}</div>
          <div class="mt-2 text-2xl font-semibold text-slate-900 dark:text-slate-100">{{ card.count }}</div>
        </button>
      </div>
    </section>

    <section class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchCaseList">
        <el-form-item label="店铺">
          <el-select v-model="searchInfo.storeId" clearable filterable class="!w-52">
            <el-option v-for="store in storeOptions" :key="store.id" :label="store.storeName" :value="store.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="站点">
          <el-select v-model="searchInfo.siteCode" clearable class="!w-32">
            <el-option label="US" value="US" />
            <el-option label="CA" value="CA" />
            <el-option label="MX" value="MX" />
          </el-select>
        </el-form-item>
        <el-form-item label="案例类型">
          <el-select v-model="searchInfo.caseType" clearable class="!w-40">
            <el-option v-for="item in caseTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="已读">
          <el-select v-model="searchInfo.readStatus" clearable class="!w-32">
            <el-option label="未读" value="unread" />
            <el-option label="已读" value="read" />
          </el-select>
        </el-form-item>
        <el-form-item label="处理状态">
          <el-select v-model="searchInfo.handlingStatus" clearable class="!w-36">
            <el-option label="待处理" value="pending" />
            <el-option label="处理中" value="processing" />
            <el-option label="已关闭" value="closed" />
          </el-select>
        </el-form-item>
        <el-form-item label="SLA">
          <el-select v-model="searchInfo.slaBucket" clearable class="!w-36">
            <el-option label="正常" value="normal" />
            <el-option label="即将超时" value="warning" />
            <el-option label="已超时" value="overdue" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" clearable placeholder="主题 / 买家 / 外部案例ID / 摘要" class="!w-72" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchCaseList">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </section>

    <section class="grid gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
      <div class="min-h-[760px] overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60">
        <div class="border-b border-slate-200 px-4 py-3 text-sm font-medium text-slate-900 dark:border-slate-700 dark:text-slate-100">
          收件箱
        </div>
        <div v-loading="listLoading" class="max-h-[760px] overflow-y-auto">
          <template v-if="caseList.length">
            <button
              v-for="item in caseList"
              :key="item.id"
              type="button"
              class="w-full border-b border-slate-200 px-4 py-4 text-left transition last:border-b-0 dark:border-slate-700"
              :class="activeCaseId === item.id ? 'bg-slate-50 dark:bg-slate-800/60' : 'hover:bg-slate-50 dark:hover:bg-slate-800/40'"
              @click="openCase(item)"
            >
              <div class="flex items-start gap-3">
                <span
                  class="mt-1 h-2.5 w-2.5 shrink-0 rounded-full"
                  :class="item.readStatus === 'unread' ? 'bg-sky-500' : 'bg-slate-200 dark:bg-slate-700'"
                />
                <div class="min-w-0 flex-1">
                  <div class="flex items-start justify-between gap-3">
                    <div class="truncate font-medium text-slate-900 dark:text-slate-100">{{ item.subject || '--' }}</div>
                    <el-tag size="small" :type="slaTagType(item.slaBucket)">{{ slaLabel(item.slaBucket) }}</el-tag>
                  </div>
                  <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-slate-500 dark:text-slate-400">
                    <span>{{ item.storeName || `Store #${item.storeId}` }}</span>
                    <span>{{ caseTypeLabel(item.caseType) }}</span>
                    <span>{{ item.siteCode || '--' }}</span>
                  </div>
                  <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-slate-500 dark:text-slate-400">
                    <span>{{ item.buyerName || '--' }}</span>
                    <span>{{ item.amazonOrderId || item.amazonRmaId || '--' }}</span>
                  </div>
                  <div class="mt-2 line-clamp-2 text-sm text-slate-600 dark:text-slate-300">{{ item.latestExcerpt || '--' }}</div>
                  <div class="mt-2 flex items-center justify-between gap-3 text-xs text-slate-500 dark:text-slate-400">
                    <span>{{ handlingStatusLabel(item.handlingStatus) }}</span>
                    <span>{{ formatRemaining(item.remainingMinutes, item.dueAt) }}</span>
                  </div>
                </div>
              </div>
            </button>
          </template>
          <el-empty v-else description="暂无客服消息" :image-size="88" class="py-20" />
        </div>
      </div>

      <div class="min-h-[760px] rounded-lg border border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900/60">
        <el-skeleton v-if="detailLoading" animated :rows="12" class="p-6" />
        <template v-else-if="activeDetail">
          <div class="border-b border-slate-200 px-6 py-5 dark:border-slate-700">
            <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
              <div class="space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <el-tag size="small" :type="caseTypeTagType(activeDetail.caseType)">{{ caseTypeLabel(activeDetail.caseType) }}</el-tag>
                  <el-tag size="small" :type="slaTagType(activeDetail.slaBucket)">{{ slaLabel(activeDetail.slaBucket) }}</el-tag>
                  <el-tag size="small" :type="activeDetail.readStatus === 'unread' ? 'warning' : 'success'">
                    {{ activeDetail.readStatus === 'unread' ? '未读' : '已读' }}
                  </el-tag>
                  <el-tag size="small" :type="handlingTagType(activeDetail.handlingStatus)">{{ handlingStatusLabel(activeDetail.handlingStatus) }}</el-tag>
                </div>
                <div class="text-2xl font-semibold text-slate-900 dark:text-slate-100">{{ activeDetail.subject || '--' }}</div>
                <div class="flex flex-wrap gap-x-4 gap-y-1 text-sm text-slate-500 dark:text-slate-400">
                  <span>{{ activeDetail.storeName || '--' }}</span>
                  <span>{{ activeDetail.siteCode || '--' }}</span>
                  <span>{{ activeDetail.amazonOrderId || '--' }}</span>
                  <span v-if="activeDetail.amazonRmaId">{{ activeDetail.amazonRmaId }}</span>
                </div>
              </div>
              <div class="flex flex-wrap gap-3">
                <el-button :loading="detailActionLoading === 'actions'" @click="refreshCaseActions">刷新动作</el-button>
                <el-button :disabled="activeDetail.readStatus !== 'unread'" @click="markCurrentRead">标记已读</el-button>
                <el-button @click="markCurrentPending">标记待处理</el-button>
                <el-button type="danger" plain @click="closeCurrentCase">关闭工单</el-button>
              </div>
            </div>
          </div>

          <div class="grid gap-6 p-6">
            <section class="rounded-lg border border-slate-200 p-4 dark:border-slate-700">
              <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
                <div class="text-base font-semibold text-slate-900 dark:text-slate-100">消息时间线</div>
                <div class="text-sm text-slate-500 dark:text-slate-400">
                  首次收件 {{ activeDetail.firstReceivedAt || '--' }} / SLA {{ activeDetail.dueAt || '--' }}
                </div>
              </div>
              <div class="space-y-3">
                <div
                  v-for="message in activeDetail.messages"
                  :key="message.id"
                  class="rounded-lg border px-4 py-3 dark:border-slate-700"
                  :class="message.role === 'agent' ? 'border-sky-200 bg-sky-50/70 dark:border-sky-900/60 dark:bg-sky-900/10' : 'border-slate-200 bg-slate-50 dark:bg-slate-900/40'"
                >
                  <div class="flex flex-wrap items-center justify-between gap-3">
                    <div class="flex flex-wrap items-center gap-2 text-sm font-medium text-slate-900 dark:text-slate-100">
                      <span>{{ messageRoleLabel(message.role) }}</span>
                      <el-tag size="small" type="info">{{ messageChannelLabel(message.channel) }}</el-tag>
                      <el-tag size="small" :type="sendStatusTagType(message.sendStatus)">{{ sendStatusLabel(message.sendStatus) }}</el-tag>
                    </div>
                    <div class="text-xs text-slate-500 dark:text-slate-400">{{ message.sentAt || message.createdAt || '--' }}</div>
                  </div>
                  <div class="mt-2 whitespace-pre-wrap text-sm text-slate-700 dark:text-slate-200">{{ message.bodyPlain || '--' }}</div>
                  <div v-if="message.errorMessage" class="mt-2 text-xs text-rose-500">{{ message.errorMessage }}</div>
                </div>
              </div>
            </section>

            <section class="grid gap-4 xl:grid-cols-[1.4fr_1fr]">
              <div class="rounded-lg border border-slate-200 p-4 dark:border-slate-700">
                <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">订单上下文</div>
                <template v-if="activeDetail.orderContext">
                  <el-descriptions :column="2" border>
                    <el-descriptions-item label="订单号">{{ activeDetail.orderContext.amazonOrderId || '--' }}</el-descriptions-item>
                    <el-descriptions-item label="状态">{{ activeDetail.orderContext.orderStatus || '--' }}</el-descriptions-item>
                    <el-descriptions-item label="履约">{{ activeDetail.orderContext.fulfillmentType || '--' }}</el-descriptions-item>
                    <el-descriptions-item label="工作流">{{ activeDetail.orderContext.workflowStatus || '--' }}</el-descriptions-item>
                    <el-descriptions-item label="买家">{{ activeDetail.orderContext.buyerName || '--' }}</el-descriptions-item>
                    <el-descriptions-item label="邮箱">{{ activeDetail.orderContext.buyerEmail || '--' }}</el-descriptions-item>
                    <el-descriptions-item label="金额">{{ formatPrice(activeDetail.orderContext.orderTotalAmount, activeDetail.orderContext.currencyCode) }}</el-descriptions-item>
                    <el-descriptions-item label="下单时间">{{ activeDetail.orderContext.purchaseDate || '--' }}</el-descriptions-item>
                    <el-descriptions-item label="收货地址" :span="2">
                      {{ formatAddress(activeDetail.orderContext.address) }}
                    </el-descriptions-item>
                  </el-descriptions>
                </template>
                <el-empty v-else description="暂无订单上下文" :image-size="72" />
              </div>

              <div class="rounded-lg border border-slate-200 p-4 dark:border-slate-700">
                <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">物流上下文</div>
                <div v-if="activeDetail.orderContext?.shipments?.length" class="space-y-3">
                  <div
                    v-for="shipment in activeDetail.orderContext.shipments"
                    :key="shipment.id"
                    class="rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 text-sm dark:border-slate-700 dark:bg-slate-900/40"
                  >
                    <div class="font-medium text-slate-900 dark:text-slate-100">{{ shipment.provider || shipment.carrierName || '--' }}</div>
                    <div class="mt-1 text-slate-600 dark:text-slate-300">{{ shipment.shippingMethod || shipment.channelName || '--' }}</div>
                    <div class="mt-2 text-xs text-slate-500 dark:text-slate-400">运单 {{ shipment.trackingNo || '--' }} / {{ shipment.status || '--' }}</div>
                  </div>
                </div>
                <el-empty v-else description="暂无物流上下文" :image-size="72" />
              </div>
            </section>

            <section class="rounded-lg border border-slate-200 p-4 dark:border-slate-700">
              <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">商品上下文</div>
              <div v-if="activeDetail.orderContext?.items?.length" class="space-y-3">
                <div
                  v-for="item in activeDetail.orderContext.items"
                  :key="item.id"
                  class="rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 dark:border-slate-700 dark:bg-slate-900/40"
                >
                  <div class="flex flex-wrap items-center gap-3">
                    <div class="font-medium text-slate-900 dark:text-slate-100">{{ item.sellerSku || '--' }}</div>
                    <div class="text-sm text-slate-500 dark:text-slate-400">{{ item.asin || '--' }}</div>
                    <div class="text-sm text-slate-500 dark:text-slate-400">x{{ item.quantityOrdered || 0 }}</div>
                  </div>
                  <div class="mt-2 text-sm text-slate-600 dark:text-slate-300">{{ item.title || '--' }}</div>
                  <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-slate-400">
                    <span>金额 {{ formatPrice(item.itemPriceAmount, item.currencyCode) }}</span>
                    <span>采购 {{ item.purchaseStatus || '--' }}</span>
                    <span>供货 {{ item.supplySource || '--' }}</span>
                  </div>
                </div>
              </div>
              <el-empty v-else description="暂无商品上下文" :image-size="72" />
            </section>

            <section
              v-if="activeDetail.returnContext"
              class="rounded-lg border border-slate-200 p-4 dark:border-slate-700"
            >
              <div class="mb-3 text-base font-semibold text-slate-900 dark:text-slate-100">退货上下文</div>
              <el-descriptions :column="2" border>
                <el-descriptions-item label="退货单号">{{ activeDetail.returnContext.amazonRmaId || '--' }}</el-descriptions-item>
                <el-descriptions-item label="申请状态">{{ activeDetail.returnContext.returnRequestStatus || '--' }}</el-descriptions-item>
                <el-descriptions-item label="退款金额">{{ formatPrice(activeDetail.returnContext.refundAmount, activeDetail.returnContext.refundCurrency) }}</el-descriptions-item>
                <el-descriptions-item label="退货运单">{{ activeDetail.returnContext.trackingId || '--' }}</el-descriptions-item>
                <el-descriptions-item label="申请时间">{{ activeDetail.returnContext.returnRequestDate || '--' }}</el-descriptions-item>
                <el-descriptions-item label="异常">{{ activeDetail.returnContext.exceptionMessage || '--' }}</el-descriptions-item>
              </el-descriptions>
            </section>

            <section class="rounded-lg border border-slate-200 p-4 dark:border-slate-700">
              <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
                <div>
                  <div class="text-base font-semibold text-slate-900 dark:text-slate-100">回复区</div>
                  <div class="text-sm text-slate-500 dark:text-slate-400">
                    模板填充后人工确认，再复制草稿或按 Amazon 可用动作直发。
                  </div>
                </div>
                <el-button link type="primary" @click="refreshCaseActions">刷新可用动作</el-button>
              </div>

              <div class="grid gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
                <div class="space-y-4">
                  <el-form label-position="top">
                    <el-form-item label="回复模板">
                      <el-select v-model="selectedTemplateId" filterable clearable class="!w-full" placeholder="选择模板">
                        <el-option
                          v-for="template in replyTemplateOptions"
                          :key="template.id"
                          :label="template.name"
                          :value="template.id"
                        />
                      </el-select>
                    </el-form-item>
                  </el-form>

                  <div v-if="selectedTemplate" class="space-y-3">
                    <div
                      v-for="field in selectedTemplate.variableSchema || []"
                      :key="field.key"
                      class="space-y-1"
                    >
                      <div class="text-xs text-slate-500 dark:text-slate-400">{{ field.label || field.key }}</div>
                      <el-input
                        v-model="replyVariables[field.key]"
                        :placeholder="field.placeholder || `填写${field.label || field.key}`"
                      />
                    </div>
                  </div>
                  <el-empty v-else description="请选择模板" :image-size="72" />
                </div>

                <div class="space-y-4">
                  <div class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-900/40">
                    <div class="text-sm font-medium text-slate-900 dark:text-slate-100">预览主题</div>
                    <div class="mt-2 text-sm text-slate-600 dark:text-slate-300">{{ previewSubject || '--' }}</div>
                  </div>
                  <div class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-900/40">
                    <div class="text-sm font-medium text-slate-900 dark:text-slate-100">预览正文</div>
                    <div class="mt-2 whitespace-pre-wrap text-sm text-slate-600 dark:text-slate-300">{{ previewBody || '--' }}</div>
                  </div>
                  <div class="flex flex-wrap gap-3">
                    <el-button
                      type="primary"
                      :disabled="!selectedTemplate"
                      :loading="replyLoading === 'copy'"
                      @click="copyReplyDraft"
                    >
                      复制草稿
                    </el-button>
                    <el-button
                      v-if="canDirectSend"
                      type="warning"
                      :loading="replyLoading === 'send'"
                      @click="sendDirectReply"
                    >
                      发送到 Amazon
                    </el-button>
                    <div v-else-if="selectedTemplate?.deliveryMode === 'amazon_direct'" class="self-center text-xs text-amber-500">
                      当前案例未返回与模板匹配的 Amazon 直发动作
                    </div>
                  </div>
                </div>
              </div>
            </section>
          </div>
        </template>
        <el-empty v-else description="请选择左侧消息查看详情" :image-size="96" class="py-24" />
      </div>
    </section>

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

    <input ref="importInputRef" type="file" accept=".xlsx" class="hidden" @change="onImportSelected" />

    <el-dialog v-model="caseDialogVisible" title="新建客服工单" width="760px" destroy-on-close>
      <el-form :model="caseForm" label-width="120px">
        <div class="grid gap-4 md:grid-cols-2">
          <el-form-item label="店铺">
            <el-select v-model="caseForm.storeId" filterable class="!w-full">
              <el-option v-for="store in storeOptions" :key="store.id" :label="store.storeName" :value="store.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="站点">
            <el-select v-model="caseForm.siteCode" class="!w-full">
              <el-option label="US" value="US" />
              <el-option label="CA" value="CA" />
              <el-option label="MX" value="MX" />
            </el-select>
          </el-form-item>
          <el-form-item label="案例类型">
            <el-select v-model="caseForm.caseType" class="!w-full">
              <el-option v-for="item in caseTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="首次收件">
            <el-date-picker
              v-model="caseForm.firstReceivedAt"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              class="!w-full"
            />
          </el-form-item>
          <el-form-item label="订单ID">
            <el-input-number v-model="caseForm.orderId" :min="1" class="!w-full" />
          </el-form-item>
          <el-form-item label="退货ID">
            <el-input-number v-model="caseForm.returnOrderId" :min="1" class="!w-full" />
          </el-form-item>
          <el-form-item label="买家姓名">
            <el-input v-model="caseForm.buyerName" />
          </el-form-item>
          <el-form-item label="买家邮箱">
            <el-input v-model="caseForm.buyerEmail" />
          </el-form-item>
        </div>
        <el-form-item label="外部案例ID">
          <el-input v-model="caseForm.externalCaseId" />
        </el-form-item>
        <el-form-item label="主题">
          <el-input v-model="caseForm.subject" />
        </el-form-item>
        <el-form-item label="客户消息">
          <el-input v-model="caseForm.messageBody" type="textarea" :rows="5" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="caseForm.notes" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="caseDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="caseSubmitting" @click="submitCase">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="templateManagerVisible" title="回复模板管理" width="1100px" destroy-on-close>
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div class="text-sm text-slate-500 dark:text-slate-400">维护售后、退货、差评和 A-to-Z 回复模板。</div>
        <div class="flex gap-3">
          <el-button type="primary" @click="openTemplateEditor()">新建模板</el-button>
          <el-button @click="fetchTemplates">刷新</el-button>
        </div>
      </div>
      <el-table :data="templateList" stripe>
        <el-table-column prop="name" label="模板名称" min-width="180" />
        <el-table-column prop="code" label="编码" min-width="170" />
        <el-table-column label="案例类型" width="130">
          <template #default="{ row }">{{ caseTypeLabel(row.caseType) }}</template>
        </el-table-column>
        <el-table-column label="投递方式" width="130">
          <template #default="{ row }">{{ deliveryModeLabel(row.deliveryMode) }}</template>
        </el-table-column>
        <el-table-column label="内置" width="90">
          <template #default="{ row }">{{ row.isBuiltin ? '是' : '否' }}</template>
        </el-table-column>
        <el-table-column label="启用" width="90">
          <template #default="{ row }">{{ row.isEnabled ? '是' : '否' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <div class="flex gap-2">
              <el-button type="primary" link @click="openTemplateEditor(row)">编辑</el-button>
              <el-button type="danger" link :disabled="row.isBuiltin" @click="removeTemplate(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="templateEditVisible" :title="templateEditForm.id ? '编辑模板' : '新建模板'" width="860px" destroy-on-close>
      <el-form :model="templateEditForm" label-width="120px">
        <div class="grid gap-4 md:grid-cols-2">
          <el-form-item label="编码">
            <el-input v-model="templateEditForm.code" :disabled="templateEditForm.isBuiltin" />
          </el-form-item>
          <el-form-item label="名称">
            <el-input v-model="templateEditForm.name" />
          </el-form-item>
          <el-form-item label="案例类型">
            <el-select v-model="templateEditForm.caseType" class="!w-full">
              <el-option v-for="item in caseTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="投递方式">
            <el-select v-model="templateEditForm.deliveryMode" class="!w-full">
              <el-option label="复制草稿" value="manual_copy" />
              <el-option label="Amazon直发" value="amazon_direct" />
            </el-select>
          </el-form-item>
          <el-form-item label="Amazon动作键">
            <el-input v-model="templateEditForm.amazonActionKey" :disabled="templateEditForm.deliveryMode !== 'amazon_direct'" />
          </el-form-item>
          <el-form-item label="排序">
            <el-input-number v-model="templateEditForm.sort" :min="1" class="!w-full" />
          </el-form-item>
        </div>
        <el-form-item label="主题模板">
          <el-input v-model="templateEditForm.subjectTemplate" />
        </el-form-item>
        <el-form-item label="正文模板">
          <el-input v-model="templateEditForm.bodyTemplate" type="textarea" :rows="5" />
        </el-form-item>
        <el-form-item label="变量定义">
          <div class="w-full space-y-3">
            <div
              v-for="(item, index) in templateEditForm.variableSchema"
              :key="`${item.key}-${index}`"
              class="grid gap-3 rounded-lg border border-slate-200 p-3 dark:border-slate-700 md:grid-cols-[1fr_1fr_120px_1fr_40px]"
            >
              <el-input v-model="item.key" placeholder="变量 key" />
              <el-input v-model="item.label" placeholder="显示名称" />
              <el-select v-model="item.required">
                <el-option label="选填" :value="false" />
                <el-option label="必填" :value="true" />
              </el-select>
              <el-input v-model="item.placeholder" placeholder="占位提示" />
              <el-button type="danger" link @click="templateEditForm.variableSchema.splice(index, 1)">删</el-button>
            </div>
            <el-button @click="addTemplateVariable">新增变量</el-button>
          </div>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="templateEditForm.isEnabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="templateEditVisible = false">取消</el-button>
        <el-button type="primary" :loading="templateSubmitting" @click="submitTemplate">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'

import { getAmazonStoreList } from '@/api/amazonStore'
import {
  closeAmazonSupportCase,
  findAmazonSupportCase,
  getAmazonSupportInboxList,
  importAmazonSupportWorkbook,
  markAmazonSupportCasePending,
  markAmazonSupportCaseRead,
  refreshAmazonSupportActions,
  saveAmazonSupportCase,
  sendAmazonSupportReply
} from '@/api/amazonSupportInbox'
import {
  deleteAmazonSupportTemplate,
  getAmazonSupportTemplateList,
  saveAmazonSupportTemplate
} from '@/api/amazonSupportTemplate'

defineOptions({
  name: 'AmazonSupportInbox'
})

const router = useRouter()
const route = useRoute()

const listLoading = ref(false)
const detailLoading = ref(false)
const detailActionLoading = ref('')
const replyLoading = ref('')
const caseSubmitting = ref(false)
const templateSubmitting = ref(false)

const caseList = ref([])
const total = ref(0)
const summary = ref({
  allCount: 0,
  unreadCount: 0,
  warningCount: 0,
  overdueCount: 0,
  pendingCount: 0
})
const activeCaseId = ref(0)
const activeDetail = ref(null)
const storeOptions = ref([])
const templateList = ref([])
const selectedTemplateId = ref(undefined)
const replyVariables = ref({})

const caseDialogVisible = ref(false)
const templateManagerVisible = ref(false)
const templateEditVisible = ref(false)
const importInputRef = ref(null)

const searchInfo = ref({
  page: 1,
  pageSize: 20,
  storeId: undefined,
  siteCode: '',
  caseType: '',
  readStatus: '',
  handlingStatus: '',
  slaBucket: '',
  keyword: ''
})

const caseForm = ref(createCaseForm())
const templateEditForm = ref(createTemplateForm())
const summaryFilter = ref('all')

const caseTypeOptions = [
  { label: 'Buyer Message', value: 'buyer_message' },
  { label: '售后', value: 'after_sales' },
  { label: '退货', value: 'return' },
  { label: '差评', value: 'negative_feedback' },
  { label: 'A-to-Z', value: 'a_to_z' }
]

const summaryCards = computed(() => [
  { key: 'all', label: '全部', count: summary.value.allCount || 0 },
  { key: 'unread', label: '未读', count: summary.value.unreadCount || 0 },
  { key: 'warning', label: '即将超时', count: summary.value.warningCount || 0 },
  { key: 'overdue', label: '已超时', count: summary.value.overdueCount || 0 },
  { key: 'pending', label: '待处理', count: summary.value.pendingCount || 0 }
])

const selectedTemplate = computed(() => templateList.value.find((item) => item.id === selectedTemplateId.value))

const replyTemplateOptions = computed(() => {
  if (!activeDetail.value) return templateList.value
  if (activeDetail.value.caseType === 'buyer_message') {
    return templateList.value.filter((item) => item.isEnabled)
  }
  return templateList.value.filter((item) => item.isEnabled && item.caseType === activeDetail.value.caseType)
})

const matchedAction = computed(() => {
  if (!selectedTemplate.value || !activeDetail.value) return null
  return (activeDetail.value.actionAvailability || []).find((item) => item.actionKey === selectedTemplate.value.amazonActionKey) || null
})

const canDirectSend = computed(() => Boolean(selectedTemplate.value?.deliveryMode === 'amazon_direct' && matchedAction.value))

const previewSubject = computed(() => renderTemplate(selectedTemplate.value?.subjectTemplate || '', replyVariables.value))
const previewBody = computed(() => renderTemplate(selectedTemplate.value?.bodyTemplate || '', replyVariables.value))

watch(selectedTemplateId, () => {
  resetReplyVariables()
})

const fetchStores = async () => {
  const res = await getAmazonStoreList({ page: 1, pageSize: 200 })
  if (res.code === 0) {
    storeOptions.value = res.data.list || []
  }
}

const fetchTemplates = async () => {
  const res = await getAmazonSupportTemplateList({ page: 1, pageSize: 200 })
  if (res.code === 0) {
    templateList.value = res.data.list || []
  }
}

const fetchCaseList = async () => {
  listLoading.value = true
  try {
    const res = await getAmazonSupportInboxList(searchInfo.value)
    if (res.code === 0) {
      caseList.value = res.data.list || []
      total.value = res.data.total || 0
      summary.value = res.data.summary || summary.value
      if (activeCaseId.value && !caseList.value.some((item) => item.id === activeCaseId.value)) {
        activeCaseId.value = 0
        activeDetail.value = null
      }
      const queryCaseId = Number(route.query.caseId || 0)
      if (queryCaseId) {
        await loadCaseDetail(queryCaseId)
      } else if (!activeCaseId.value && caseList.value.length) {
        await loadCaseDetail(caseList.value[0].id)
      }
    }
  } finally {
    listLoading.value = false
  }
}

const loadCaseDetail = async (caseId, options = {}) => {
  if (!caseId) return
  if (!options.force && activeCaseId.value === caseId && activeDetail.value) return
  activeCaseId.value = Number(caseId)
  detailLoading.value = true
  try {
    const res = await findAmazonSupportCase({ id: caseId })
    if (res.code === 0) {
      activeDetail.value = res.data
      selectedTemplateId.value = undefined
      replyVariables.value = {}
      await router.replace({
        query: {
          ...route.query,
          caseId: String(caseId),
          compose: undefined
        }
      })
    }
  } finally {
    detailLoading.value = false
  }
}

const openCase = async (item) => {
  await loadCaseDetail(item.id)
}

const applySummaryFilter = (key) => {
  summaryFilter.value = key
  searchInfo.value.page = 1
  searchInfo.value.readStatus = ''
  searchInfo.value.handlingStatus = ''
  searchInfo.value.slaBucket = ''
  switch (key) {
    case 'unread':
      searchInfo.value.readStatus = 'unread'
      break
    case 'warning':
      searchInfo.value.slaBucket = 'warning'
      break
    case 'overdue':
      searchInfo.value.slaBucket = 'overdue'
      break
    case 'pending':
      searchInfo.value.handlingStatus = 'pending'
      break
    default:
      break
  }
  fetchCaseList()
}

const summaryCardClass = (key) => {
  if (summaryFilter.value === key) {
    return 'border-sky-300 bg-sky-50 dark:border-sky-700 dark:bg-sky-900/20'
  }
  return 'border-slate-200 hover:border-slate-300 hover:bg-slate-50 dark:border-slate-700 dark:hover:border-slate-500 dark:hover:bg-slate-800/50'
}

const resetSearch = async () => {
  searchInfo.value = {
    page: 1,
    pageSize: 20,
    storeId: undefined,
    siteCode: '',
    caseType: '',
    readStatus: '',
    handlingStatus: '',
    slaBucket: '',
    keyword: ''
  }
  summaryFilter.value = 'all'
  await fetchCaseList()
}

const handleCurrentChange = (page) => {
  searchInfo.value.page = page
  fetchCaseList()
}

const handleSizeChange = (pageSize) => {
  searchInfo.value.pageSize = pageSize
  searchInfo.value.page = 1
  fetchCaseList()
}

const openCaseDialog = (prefill = null) => {
  caseForm.value = createCaseForm(prefill || buildPrefillFromRoute())
  caseDialogVisible.value = true
}

const submitCase = async () => {
  caseSubmitting.value = true
  try {
    const payload = {
      storeId: caseForm.value.storeId,
      siteCode: caseForm.value.siteCode,
      caseType: caseForm.value.caseType,
      sourceType: caseForm.value.sourceType,
      sourceRefType: caseForm.value.sourceRefType,
      orderId: caseForm.value.orderId ? Number(caseForm.value.orderId) : undefined,
      returnOrderId: caseForm.value.returnOrderId ? Number(caseForm.value.returnOrderId) : undefined,
      externalCaseId: caseForm.value.externalCaseId,
      subject: caseForm.value.subject,
      buyerName: caseForm.value.buyerName,
      buyerEmail: caseForm.value.buyerEmail,
      firstReceivedAt: caseForm.value.firstReceivedAt,
      messageBody: caseForm.value.messageBody,
      notes: caseForm.value.notes
    }
    const res = await saveAmazonSupportCase(payload)
    if (res.code === 0) {
      ElMessage.success('工单已保存')
      caseDialogVisible.value = false
      await fetchCaseList()
      if (res.data?.id) {
        await loadCaseDetail(res.data.id)
      }
    }
  } finally {
    caseSubmitting.value = false
  }
}

const triggerImport = () => {
  importInputRef.value?.click()
}

const onImportSelected = async (event) => {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return
  const res = await importAmazonSupportWorkbook(file)
  if (res.code === 0) {
    ElMessage.success(`导入完成，成功 ${res.data.successRows || 0} 条，失败 ${res.data.failedRows || 0} 条`)
    fetchCaseList()
  }
}

const markCurrentRead = async () => {
  if (!activeDetail.value) return
  const res = await markAmazonSupportCaseRead({ id: activeDetail.value.id })
  if (res.code === 0) {
    activeDetail.value = res.data
    fetchCaseList()
  }
}

const markCurrentPending = async () => {
  if (!activeDetail.value) return
  const res = await markAmazonSupportCasePending({ id: activeDetail.value.id })
  if (res.code === 0) {
    activeDetail.value = res.data
    fetchCaseList()
  }
}

const closeCurrentCase = async () => {
  if (!activeDetail.value) return
  await ElMessageBox.confirm('确认关闭当前客服工单吗？', '关闭工单', {
    type: 'warning'
  })
  const res = await closeAmazonSupportCase({ id: activeDetail.value.id })
  if (res.code === 0) {
    activeDetail.value = res.data
    fetchCaseList()
  }
}

const refreshCaseActions = async () => {
  if (!activeDetail.value) return
  detailActionLoading.value = 'actions'
  try {
    const res = await refreshAmazonSupportActions({ caseId: activeDetail.value.id })
    if (res.code === 0) {
      activeDetail.value.actionAvailability = res.data || []
      await loadCaseDetail(activeDetail.value.id, { force: true })
      ElMessage.success('可用动作已刷新')
    }
  } finally {
    detailActionLoading.value = ''
  }
}

const copyReplyDraft = async () => {
  if (!selectedTemplate.value || !activeDetail.value) return
  replyLoading.value = 'copy'
  try {
    const res = await sendAmazonSupportReply({
      caseId: activeDetail.value.id,
      templateId: selectedTemplate.value.id,
      deliveryMode: 'manual_copy',
      variables: replyVariables.value
    })
    if (res.code === 0) {
      try {
        await copyText(`${res.data.renderedSubject || ''}\n\n${res.data.renderedBody || ''}`.trim())
        ElMessage.success('草稿已复制')
      } catch (error) {
        ElMessage.warning(error?.message || '草稿已生成，但复制到剪贴板失败')
      }
      await loadCaseDetail(activeDetail.value.id, { force: true })
      await fetchCaseList()
    }
  } finally {
    replyLoading.value = ''
  }
}

const sendDirectReply = async () => {
  if (!selectedTemplate.value || !activeDetail.value || !matchedAction.value) return
  replyLoading.value = 'send'
  try {
    const res = await sendAmazonSupportReply({
      caseId: activeDetail.value.id,
      templateId: selectedTemplate.value.id,
      deliveryMode: 'amazon_direct',
      actionKey: matchedAction.value.actionKey,
      actionPath: matchedAction.value.path,
      variables: replyVariables.value
    })
    if (res.code === 0) {
      ElMessage.success(res.data.sendStatus === 'sent' ? '消息已发送到 Amazon' : '已记录为手工跟进')
      await loadCaseDetail(activeDetail.value.id, { force: true })
      await fetchCaseList()
    }
  } finally {
    replyLoading.value = ''
  }
}

const resetReplyVariables = () => {
  const nextValues = {}
  const defaults = defaultReplyVariables()
  ;(selectedTemplate.value?.variableSchema || []).forEach((item) => {
    nextValues[item.key] = defaults[item.key] || ''
  })
  replyVariables.value = nextValues
}

const defaultReplyVariables = () => {
  if (!activeDetail.value) return {}
  const detail = activeDetail.value
  return {
    buyer_name: detail.buyerName || '',
    buyer_email: detail.buyerEmail || '',
    amazon_order_id: detail.amazonOrderId || '',
    amazon_rma_id: detail.amazonRmaId || '',
    store_name: detail.storeName || '',
    site_code: detail.siteCode || '',
    case_subject: detail.subject || '',
    external_case_id: detail.externalCaseId || '',
    order_status: detail.orderContext?.orderStatus || '',
    tracking_no: detail.orderContext?.shipments?.[0]?.trackingNo || '',
    return_status: detail.returnContext?.returnRequestStatus || '',
    return_tracking_id: detail.returnContext?.trackingId || ''
  }
}

const openTemplateManager = async () => {
  await fetchTemplates()
  templateManagerVisible.value = true
}

const openTemplateEditor = (row = null) => {
  templateEditForm.value = createTemplateForm(row)
  templateEditVisible.value = true
}

const addTemplateVariable = () => {
  templateEditForm.value.variableSchema.push({
    key: '',
    label: '',
    required: false,
    placeholder: ''
  })
}

const submitTemplate = async () => {
  templateSubmitting.value = true
  try {
    const payload = {
      id: templateEditForm.value.id,
      code: templateEditForm.value.code,
      name: templateEditForm.value.name,
      caseType: templateEditForm.value.caseType,
      deliveryMode: templateEditForm.value.deliveryMode,
      amazonActionKey: templateEditForm.value.amazonActionKey,
      subjectTemplate: templateEditForm.value.subjectTemplate,
      bodyTemplate: templateEditForm.value.bodyTemplate,
      variableSchema: (templateEditForm.value.variableSchema || []).filter((item) => item.key),
      isEnabled: templateEditForm.value.isEnabled,
      sort: templateEditForm.value.sort
    }
    const res = await saveAmazonSupportTemplate(payload)
    if (res.code === 0) {
      ElMessage.success('模板已保存')
      templateEditVisible.value = false
      fetchTemplates()
    }
  } finally {
    templateSubmitting.value = false
  }
}

const removeTemplate = async (row) => {
  await ElMessageBox.confirm(`确认删除模板 ${row.name} 吗？`, '删除模板', { type: 'warning' })
  const res = await deleteAmazonSupportTemplate({ id: row.id })
  if (res.code === 0) {
    ElMessage.success('模板已删除')
    fetchTemplates()
  }
}

const buildPrefillFromRoute = () => {
  if (route.query.compose !== '1') return null
  return {
    storeId: Number(route.query.storeId || 0) || undefined,
    siteCode: route.query.siteCode || '',
    caseType: route.query.caseType || 'after_sales',
    orderId: Number(route.query.orderId || 0) || undefined,
    returnOrderId: Number(route.query.returnOrderId || 0) || undefined,
    buyerName: route.query.buyerName || '',
    buyerEmail: route.query.buyerEmail || '',
    subject: route.query.subject || buildPrefillSubject(route.query),
    externalCaseId: '',
    messageBody: '',
    notes: '',
    firstReceivedAt: formatCurrentDateTime(),
    sourceType: 'manual',
    sourceRefType: route.query.returnOrderId ? 'return' : 'order'
  }
}

const buildPrefillSubject = (query) => {
  if (query.amazonRmaId) return `退货 ${query.amazonRmaId} 客服跟进`
  if (query.amazonOrderId) return `订单 ${query.amazonOrderId} 客服跟进`
  return 'Amazon 客服跟进'
}

const initializeFromRoute = async () => {
  const prefill = buildPrefillFromRoute()
  if (prefill) {
    openCaseDialog(prefill)
  }
  const queryCaseId = Number(route.query.caseId || 0)
  if (queryCaseId) {
    await loadCaseDetail(queryCaseId)
  }
}

const copyText = async (value) => {
  if (navigator?.clipboard?.writeText && document.hasFocus()) {
    try {
      await navigator.clipboard.writeText(value)
      return
    } catch {}
  }
  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', 'readonly')
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  try {
    if (document.execCommand('copy')) {
      return
    }
  } finally {
    document.body.removeChild(textarea)
  }
  throw new Error('草稿已生成，但当前浏览器不允许写入剪贴板')
}

const formatCurrentDateTime = () => {
  const now = new Date()
  const pad = (value) => String(value).padStart(2, '0')
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`
}

const renderTemplate = (template, variables) => {
  return String(template || '').replace(/\{\{\s*([a-zA-Z0-9_]+)\s*\}\}/g, (_, key) => variables?.[key] || '')
}

function createCaseForm(overrides = null) {
  return {
    storeId: undefined,
    siteCode: '',
    caseType: 'after_sales',
    orderId: undefined,
    returnOrderId: undefined,
    buyerName: '',
    buyerEmail: '',
    subject: '',
    externalCaseId: '',
    messageBody: '',
    notes: '',
    firstReceivedAt: '',
    sourceType: 'manual',
    sourceRefType: '',
    ...overrides
  }
}

function createTemplateForm(row = null) {
  return {
    id: row?.id,
    code: row?.code || '',
    name: row?.name || '',
    caseType: row?.caseType || 'after_sales',
    deliveryMode: row?.deliveryMode || 'manual_copy',
    amazonActionKey: row?.amazonActionKey || '',
    subjectTemplate: row?.subjectTemplate || '',
    bodyTemplate: row?.bodyTemplate || '',
    isBuiltin: row?.isBuiltin || false,
    isEnabled: row?.isEnabled ?? true,
    sort: row?.sort || 100,
    variableSchema: row?.variableSchema?.length
      ? row.variableSchema.map((item) => ({
        key: item.key || '',
        label: item.label || '',
        required: Boolean(item.required),
        placeholder: item.placeholder || ''
      }))
      : []
  }
}

const caseTypeLabel = (value) => caseTypeOptions.find((item) => item.value === value)?.label || value || '--'

const caseTypeTagType = (value) => {
  switch (value) {
    case 'buyer_message':
      return 'primary'
    case 'return':
      return 'success'
    case 'negative_feedback':
      return 'danger'
    case 'a_to_z':
      return 'warning'
    default:
      return 'info'
  }
}

const handlingStatusLabel = (value) => {
  switch (value) {
    case 'processing':
      return '处理中'
    case 'closed':
      return '已关闭'
    default:
      return '待处理'
  }
}

const handlingTagType = (value) => {
  switch (value) {
    case 'processing':
      return 'warning'
    case 'closed':
      return 'info'
    default:
      return 'success'
  }
}

const slaLabel = (value) => {
  switch (value) {
    case 'overdue':
      return '已超时'
    case 'warning':
      return '即将超时'
    default:
      return '正常'
  }
}

const slaTagType = (value) => {
  switch (value) {
    case 'overdue':
      return 'danger'
    case 'warning':
      return 'warning'
    default:
      return 'success'
  }
}

const messageRoleLabel = (value) => {
  switch (value) {
    case 'agent':
      return '客服'
    case 'internal':
      return '内部备注'
    default:
      return '买家'
  }
}

const messageChannelLabel = (value) => {
  switch (value) {
    case 'amazon':
      return 'Amazon'
    case 'manual_copy':
      return '复制草稿'
    case 'internal':
      return '内部'
    default:
      return '导入'
  }
}

const sendStatusLabel = (value) => {
  switch (value) {
    case 'sent':
      return '已发送'
    case 'copied':
      return '已复制'
    case 'failed':
      return '失败'
    case 'fallback_manual':
      return '转手工发送'
    default:
      return '草稿'
  }
}

const sendStatusTagType = (value) => {
  switch (value) {
    case 'sent':
      return 'success'
    case 'copied':
      return 'primary'
    case 'failed':
      return 'danger'
    case 'fallback_manual':
      return 'warning'
    default:
      return 'info'
  }
}

const deliveryModeLabel = (value) => (value === 'amazon_direct' ? 'Amazon直发' : '复制草稿')

const formatPrice = (value, currencyCode) => {
  if (value === null || typeof value === 'undefined') return '--'
  return `${currencyCode || ''} ${Number(value).toFixed(2)}`.trim()
}

const formatAddress = (address) => {
  if (!address) return '--'
  return [
    address.recipientName,
    address.addressLine1,
    address.addressLine2,
    address.addressLine3,
    address.city,
    address.stateOrRegion,
    address.postalCode,
    address.countryCode
  ].filter(Boolean).join(', ') || '--'
}

const formatRemaining = (minutes, dueAt) => {
  if (!dueAt) return '--'
  const value = Number(minutes || 0)
  if (value < 0) return `超时 ${Math.abs(value)} 分钟`
  if (value < 60) return `${value} 分钟后截止`
  return `${Math.floor(value / 60)} 小时后截止`
}

onMounted(async () => {
  await fetchStores()
  await fetchTemplates()
  await initializeFromRoute()
  await fetchCaseList()
})
</script>
