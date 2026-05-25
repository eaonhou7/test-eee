<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div class="space-y-2">
          <p class="text-xs tracking-[0.28em] text-slate-500">AMAZON 模板中心</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">模板中心</h1>
          <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
            上传 Seller Central 官方模板，解析列头并配置字段规则；系统也会自动提供家居类目默认模板，供上架列表和详情页直接绑定与导出。
          </p>
        </div>
        <div class="gva-btn-list !mb-0">
          <el-button type="primary" @click="openCreateDialog">新建模板</el-button>
          <el-button @click="downloadHomeTemplate()">下载家居默认模板</el-button>
          <el-button @click="fetchTemplates">刷新列表</el-button>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="fetchTemplates">
        <el-form-item label="关键词">
          <el-input v-model="searchInfo.keyword" placeholder="模板编码 / 模板名称 / 产品类型" clearable />
        </el-form-item>
        <el-form-item label="站点">
          <el-select v-model="searchInfo.siteCode" clearable placeholder="全部站点" class="!w-32">
            <el-option label="US" value="US" />
            <el-option label="CA" value="CA" />
            <el-option label="MX" value="MX" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchInfo.status" clearable placeholder="全部状态" class="!w-36">
            <el-option label="草稿" value="draft" />
            <el-option label="启用" value="active" />
            <el-option label="归档" value="archived" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchTemplates">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table :data="tableData" stripe>
        <el-table-column prop="code" label="模板编码" min-width="150" />
        <el-table-column prop="name" label="模板名称" min-width="180" />
        <el-table-column prop="siteCode" label="站点" width="90" />
        <el-table-column prop="productType" label="产品类型" min-width="160" />
        <el-table-column prop="templateVersion" label="版本" width="120" />
        <el-table-column prop="workbookFileName" label="工作簿文件" min-width="180" show-overflow-tooltip />
        <el-table-column label="字段数" width="100">
          <template #default="{ row }">{{ row.fieldCount || 0 }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">{{ getStatusLabel(row.status) }}</template>
        </el-table-column>
        <el-table-column label="操作" min-width="360" fixed="right">
          <template #default="{ row }">
            <div class="flex flex-wrap gap-2">
              <el-button type="primary" link @click="openEditDialog(row)">编辑</el-button>
              <el-button type="primary" link @click="downloadTemplate(row)">下载模板</el-button>
              <el-button type="primary" link @click="triggerUpload(row)">上传模板</el-button>
              <el-button type="primary" link :disabled="!row.workbookFileId" @click="parseWorkbook(row)">解析列头</el-button>
              <el-button type="danger" link @click="deleteTemplateRow(row)">删除</el-button>
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

    <input
      ref="uploadInputRef"
      type="file"
      accept=".xlsx"
      class="hidden"
      @change="onWorkbookSelected"
    />

    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新建模板' : '编辑模板'"
      width="760px"
      destroy-on-close
    >
      <div class="mb-4 rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-900/60 dark:text-slate-300">
        <div>模板编码：后台唯一标识，用在 Amazon 模板列表筛选与站点绑定。</div>
        <div>产品类型：决定商品详情页字段、校验规则和导出的 Excel 列结构。</div>
        <div>工作表与行号：告诉系统模板文件的列头在哪一行、数据从哪一行开始写入。</div>
      </div>
      <el-form :model="formData" label-width="120px">
        <div class="grid gap-4 md:grid-cols-2">
          <el-form-item label="模板编码">
            <el-input v-model="formData.code" />
          </el-form-item>
          <el-form-item label="模板名称">
            <el-input v-model="formData.name" />
          </el-form-item>
          <el-form-item label="站点标识（Marketplace ID）">
            <el-input v-model="formData.marketplaceId" />
          </el-form-item>
          <el-form-item label="站点">
            <el-select v-model="formData.siteCode" class="!w-full">
              <el-option label="US" value="US" />
              <el-option label="CA" value="CA" />
              <el-option label="MX" value="MX" />
            </el-select>
          </el-form-item>
          <el-form-item label="产品类型">
            <el-input v-model="formData.productType" />
          </el-form-item>
          <el-form-item label="模板版本">
            <el-input v-model="formData.templateVersion" />
          </el-form-item>
          <el-form-item label="工作表名称">
            <el-input v-model="formData.sheetName" />
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="formData.status" class="!w-full">
              <el-option label="草稿" value="draft" />
              <el-option label="启用" value="active" />
              <el-option label="归档" value="archived" />
            </el-select>
          </el-form-item>
          <el-form-item label="列头行号">
            <el-input-number v-model="formData.headerRowIndex" :min="1" class="!w-full" />
          </el-form-item>
          <el-form-item label="数据起始行">
            <el-input-number v-model="formData.dataStartRowIndex" :min="2" class="!w-full" />
          </el-form-item>
        </div>
        <el-form-item label="支持语言">
          <el-select v-model="formData.supportedLocales" multiple filterable class="!w-full">
            <el-option label="英语（美国）" value="en_US" />
            <el-option label="英语（加拿大）" value="en_CA" />
            <el-option label="法语（加拿大）" value="fr_CA" />
            <el-option label="西班牙语（墨西哥）" value="es_MX" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="formData.notes" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="downloadHomeTemplate(formData.siteCode || 'US')">下载家居默认模板</el-button>
        <el-button v-if="dialogMode === 'edit' && formData.id" @click="downloadTemplate({ id: formData.id, code: formData.code, siteCode: formData.siteCode })">下载当前模板</el-button>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="submitTemplate">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="rulesDialogVisible" title="字段规则" width="1180px" destroy-on-close>
      <div class="mb-4 flex flex-wrap items-center gap-3 rounded border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-900/60 dark:text-slate-300">
        <span>工作表：{{ parseResult.sheetName || '--' }}</span>
        <span>列头行号：{{ parseResult.headerRowIndex || '--' }}</span>
        <span>数据起始行：{{ parseResult.dataStartRowIndex || '--' }}</span>
      </div>
      <div class="mb-4 rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-900/60 dark:text-slate-300">
        <div>字段键名：系统内部存储名，用于商品详情页动态表单、校验和导出映射。</div>
        <div>字段位置：说明该字段是基础信息、变体信息、站点信息、语言内容还是图片资源。</div>
        <div>是否必填：决定商品保存/导出前是否拦截；图片槽位用于把图片写入 Amazon 的主图或副图列。</div>
      </div>

      <el-table :data="fieldRules" max-height="560">
        <el-table-column prop="columnIndex" label="#" width="70" />
        <el-table-column prop="columnHeader" label="列头" min-width="180" />
        <el-table-column label="字段键名" min-width="180">
          <template #default="{ row }">
            <el-input v-model="row.fieldKey" />
          </template>
        </el-table-column>
        <el-table-column label="中文名称" min-width="160">
          <template #default="{ row }">
            <el-input v-model="row.fieldLabel" placeholder="用于 Amazon 详情页和编辑页展示的中文名" />
          </template>
        </el-table-column>
        <el-table-column label="字段位置" width="150">
          <template #default="{ row }">
            <el-select v-model="row.scope">
              <el-option label="基础信息" value="common" />
              <el-option label="变体字段" value="variation" />
              <el-option label="站点信息" value="marketplace" />
              <el-option label="语言内容" value="locale" />
              <el-option label="图片资源" value="image" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="语言代码" width="130">
          <template #default="{ row }">
            <el-input v-model="row.localeCode" placeholder="留空表示跟随站点语言" />
          </template>
        </el-table-column>
        <el-table-column label="数据类型" width="140">
          <template #default="{ row }">
            <el-select v-model="row.dataType">
              <el-option label="文本" value="string" />
              <el-option label="整数" value="integer" />
              <el-option label="数值" value="number" />
              <el-option label="布尔值" value="boolean" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="是否必填" width="140">
          <template #default="{ row }">
            <el-select v-model="row.requiredLevel">
              <el-option label="选填" value="optional" />
              <el-option label="必填" value="required" />
              <el-option label="条件必填" value="conditional" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="可选值" min-width="220">
          <template #default="{ row }">
            <el-select
              v-model="row.enumValues"
              multiple
              filterable
              allow-create
              default-first-option
              collapse-tags
              collapse-tags-tooltip
              class="!w-full"
              placeholder="回车新增可选值"
            >
              <el-option
                v-for="option in row.enumValues || []"
                :key="option"
                :label="option"
                :value="option"
              />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="图片槽位" width="140">
          <template #default="{ row }">
            <el-input v-model="row.imageSlot" placeholder="例如 MAIN、PT1、PT2" />
          </template>
        </el-table-column>
        <el-table-column label="启用" width="90">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" />
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <el-button @click="rulesDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="rulesLoading" @click="saveFieldRules">保存字段规则</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
  import { nextTick, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'

  import {
    createAmazonTemplate,
    deleteAmazonTemplate,
    downloadAmazonTemplateWorkbook,
    getAmazonTemplateList,
    parseAmazonTemplateWorkbook,
    saveAmazonTemplateFieldRules,
    updateAmazonTemplate,
    uploadAmazonTemplateWorkbook
  } from '@/api/amazonTemplate'
  import { normalizeBlobResponse, triggerBlobDownload } from '@/utils/blobDownload'

  defineOptions({
    name: 'AmazonTemplateCenter'
  })

  const templateStatusLabelMap = {
    draft: '草稿',
    active: '启用',
    archived: '归档'
  }

  const tableData = ref([])
  const total = ref(0)
  const dialogVisible = ref(false)
  const rulesDialogVisible = ref(false)
  const submitLoading = ref(false)
  const rulesLoading = ref(false)
  const uploadInputRef = ref()
  const uploadTemplateId = ref(0)
  const dialogMode = ref('create')
  const parseResult = ref({})
  const fieldRules = ref([])

  const searchInfo = reactive({
    page: 1,
    pageSize: 10,
    keyword: '',
    siteCode: '',
    status: ''
  })

  const formData = reactive({
    id: 0,
    code: '',
    name: '',
    marketplaceId: '',
    siteCode: 'US',
    productType: '',
    templateVersion: '',
    sheetName: '',
    headerRowIndex: 1,
    dataStartRowIndex: 2,
    supportedLocales: ['en_US'],
    status: 'draft',
    notes: ''
  })

  const resetForm = () => {
    formData.id = 0
    formData.code = ''
    formData.name = ''
    formData.marketplaceId = ''
    formData.siteCode = 'US'
    formData.productType = ''
    formData.templateVersion = ''
    formData.sheetName = ''
    formData.headerRowIndex = 1
    formData.dataStartRowIndex = 2
    formData.supportedLocales = ['en_US']
    formData.status = 'draft'
    formData.notes = ''
  }

  const getStatusLabel = (status) => templateStatusLabelMap[status] || status || '--'

  const downloadTemplateFile = async (params, fallbackFileName, successMessage) => {
    try {
      const res = await downloadAmazonTemplateWorkbook(params)
      const { blob, fileName } = await normalizeBlobResponse(res, fallbackFileName)
      triggerBlobDownload(blob, fileName)
      ElMessage.success(successMessage)
    } catch (error) {
      ElMessage.error(error?.message || '模板下载失败')
    }
  }

  const downloadHomeTemplate = async (siteCode = 'US') => {
    await downloadTemplateFile(
      { preset: 'home', siteCode },
      `amazon-home-template-${String(siteCode || 'US').toLowerCase()}.xlsx`,
      '家居默认模板下载成功'
    )
  }

  const downloadTemplate = async (row) => {
    const templateId = Number(row?.id || 0)
    if (!templateId) {
      await downloadHomeTemplate(row?.siteCode || formData.siteCode || 'US')
      return
    }
    await downloadTemplateFile(
      { id: templateId },
      `${row?.code || 'amazon-template'}.xlsx`,
      '模板下载成功'
    )
  }

  const fetchTemplates = async () => {
    const res = await getAmazonTemplateList(searchInfo)
    tableData.value = res.data?.list || []
    total.value = res.data?.total || 0
  }

  const resetSearch = () => {
    searchInfo.page = 1
    searchInfo.pageSize = 10
    searchInfo.keyword = ''
    searchInfo.siteCode = ''
    searchInfo.status = ''
    fetchTemplates()
  }

  const openCreateDialog = () => {
    dialogMode.value = 'create'
    resetForm()
    dialogVisible.value = true
  }

  const openEditDialog = (row) => {
    dialogMode.value = 'edit'
    formData.id = row.id
    formData.code = row.code
    formData.name = row.name
    formData.marketplaceId = row.marketplaceId
    formData.siteCode = row.siteCode
    formData.productType = row.productType
    formData.templateVersion = row.templateVersion
    formData.sheetName = row.sheetName
    formData.headerRowIndex = row.headerRowIndex
    formData.dataStartRowIndex = row.dataStartRowIndex
    formData.supportedLocales = [...(row.supportedLocales || [])]
    formData.status = row.status
    formData.notes = row.notes
    dialogVisible.value = true
  }

  const submitTemplate = async () => {
    submitLoading.value = true
    try {
      const payload = { ...formData }
      const action = dialogMode.value === 'create' ? createAmazonTemplate : updateAmazonTemplate
      await action(payload)
      ElMessage.success(dialogMode.value === 'create' ? '模板已创建' : '模板已更新')
      dialogVisible.value = false
      fetchTemplates()
    } finally {
      submitLoading.value = false
    }
  }

  const deleteTemplateRow = async (row) => {
    await ElMessageBox.confirm(`确认删除模板 ${row.name} 吗？`, '删除模板', {
      type: 'warning'
    })
    await deleteAmazonTemplate({ id: row.id })
    ElMessage.success('删除成功')
    fetchTemplates()
  }

  const triggerUpload = async (row) => {
    uploadTemplateId.value = row.id
    await nextTick()
    uploadInputRef.value?.click()
  }

  const onWorkbookSelected = async (event) => {
    const file = event.target.files?.[0]
    if (!file || !uploadTemplateId.value) {
      return
    }
    try {
      await uploadAmazonTemplateWorkbook(uploadTemplateId.value, file)
      ElMessage.success('模板文件上传成功')
      fetchTemplates()
    } finally {
      event.target.value = ''
      uploadTemplateId.value = 0
    }
  }

  const parseWorkbook = async (row) => {
    const res = await parseAmazonTemplateWorkbook({ id: row.id })
    parseResult.value = res.data || {}
    fieldRules.value = (res.data?.fields || []).map((item) => ({
      ...item,
      enumValues: normalizeEnumValues(item.enumValues)
    }))
    rulesDialogVisible.value = true
  }

  const normalizeEnumValues = (values = []) =>
    Array.from(
      new Set(
        (values || [])
          .map((item) => String(item || '').trim())
          .filter(Boolean)
      )
    )

  const saveFieldRules = async () => {
    rulesLoading.value = true
    try {
      await saveAmazonTemplateFieldRules({
        templateId: parseResult.value.templateId,
        fields: fieldRules.value.map((item) => ({
          ...item,
          enumValues: normalizeEnumValues(item.enumValues)
        }))
      })
      ElMessage.success('字段规则已保存')
      rulesDialogVisible.value = false
      fetchTemplates()
    } finally {
      rulesLoading.value = false
    }
  }

  const handleCurrentChange = (page) => {
    searchInfo.page = page
    fetchTemplates()
  }

  const handleSizeChange = (pageSize) => {
    searchInfo.page = 1
    searchInfo.pageSize = pageSize
    fetchTemplates()
  }

  fetchTemplates()
</script>
