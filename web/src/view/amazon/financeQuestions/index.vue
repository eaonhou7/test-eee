<template>
  <div>
    <div class="gva-table-box">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="space-y-2">
          <p class="text-xs text-slate-500">AMAZON FINANCE QUESTIONS</p>
          <h1 class="text-2xl font-semibold text-slate-900 dark:text-slate-100">问题列表</h1>
        </div>
      </div>
    </div>

    <div class="gva-search-box !pb-4">
      <el-form :inline="true" :model="searchInfo" @keyup.enter="searchTable">
        <el-form-item label="问题标题">
          <el-input v-model="searchInfo.title" clearable placeholder="标题模糊搜索" class="!w-64" />
        </el-form-item>
        <el-form-item label="问题类型">
          <el-select
            v-model="searchInfo.questionType"
            clearable
            filterable
            allow-create
            default-first-option
            class="!w-44"
          >
            <el-option v-for="item in questionTypeOptions" :key="item" :label="item" :value="item" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" :loading="loading" @click="searchTable">搜索</el-button>
          <el-button type="primary" icon="plus" @click="openDialog()">新增</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <el-table :data="tableData" stripe v-loading="loading" row-key="id">
        <el-table-column prop="title" label="问题标题" min-width="240" show-overflow-tooltip />
        <el-table-column prop="questionType" label="问题类型" width="150" />
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">{{ formatDate(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="更新时间" min-width="170">
          <template #default="{ row }">{{ formatDate(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDialog(row)">编辑</el-button>
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

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑问题' : '新增问题'" width="860px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="96px">
        <el-form-item label="标题" prop="title">
          <el-input v-model="form.title" placeholder="请输入问题标题" maxlength="255" show-word-limit />
        </el-form-item>
        <el-form-item label="问题类型" prop="questionType">
          <el-select
            v-model="form.questionType"
            filterable
            allow-create
            default-first-option
            class="!w-full"
            placeholder="请选择或输入问题类型"
          >
            <el-option v-for="item in questionTypeOptions" :key="item" :label="item" :value="item" />
          </el-select>
        </el-form-item>
        <el-form-item label="内容" prop="contentHtml">
          <div class="w-full">
            <RichEdit v-model="form.contentHtml" />
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

import {
  findAmazonFinanceQuestion,
  getAmazonFinanceQuestionList,
  saveAmazonFinanceQuestion
} from '@/api/amazonFinanceQuestion'
import RichEdit from '@/components/richtext/rich-edit.vue'

defineOptions({
  name: 'AmazonFinanceQuestionManager'
})

const questionTypeOptions = ['店铺创建', '收款账户']

const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const tableData = ref([])
const total = ref(0)
const formRef = ref()

const searchInfo = reactive({
  page: 1,
  pageSize: 10,
  title: '',
  questionType: ''
})

const createForm = () => ({
  id: 0,
  title: '',
  questionType: '',
  contentHtml: ''
})

const form = reactive(createForm())

const rules = {
  title: [{ required: true, message: '请输入问题标题', trigger: 'blur' }],
  questionType: [{ required: true, message: '请选择或输入问题类型', trigger: 'change' }]
}

const buildListPayload = () => ({
  page: searchInfo.page,
  pageSize: searchInfo.pageSize,
  title: searchInfo.title,
  questionType: searchInfo.questionType
})

const fetchTable = async () => {
  loading.value = true
  try {
    const res = await getAmazonFinanceQuestionList(buildListPayload())
    if (res.code === 0) {
      tableData.value = res.data.list || []
      total.value = res.data.total || 0
      searchInfo.page = res.data.page || searchInfo.page
      searchInfo.pageSize = res.data.pageSize || searchInfo.pageSize
    }
  } finally {
    loading.value = false
  }
}

const searchTable = async () => {
  searchInfo.page = 1
  await fetchTable()
}

const resetSearch = () => {
  searchInfo.page = 1
  searchInfo.pageSize = 10
  searchInfo.title = ''
  searchInfo.questionType = ''
  fetchTable()
}

const handleCurrentChange = (page) => {
  searchInfo.page = page
  fetchTable()
}

const handleSizeChange = (pageSize) => {
  searchInfo.pageSize = pageSize
  searchInfo.page = 1
  fetchTable()
}

const resetForm = () => {
  Object.assign(form, createForm())
  formRef.value?.clearValidate?.()
}

const openDialog = async (row) => {
  resetForm()
  if (row?.id) {
    const res = await findAmazonFinanceQuestion({ id: row.id })
    if (res.code !== 0) return
    Object.assign(form, {
      id: res.data.id || 0,
      title: res.data.title || '',
      questionType: res.data.questionType || '',
      contentHtml: res.data.contentHtml || ''
    })
  } else {
    form.questionType = questionTypeOptions[0]
  }
  dialogVisible.value = true
}

const submitForm = async () => {
  const valid = await formRef.value?.validate?.().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    const res = await saveAmazonFinanceQuestion({
      id: form.id,
      title: form.title,
      questionType: form.questionType,
      contentHtml: form.contentHtml
    })
    if (res.code === 0) {
      ElMessage.success('保存成功')
      dialogVisible.value = false
      await fetchTable()
    }
  } finally {
    saving.value = false
  }
}

const formatDate = (value) => {
  if (!value) return '--'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

onMounted(fetchTable)
</script>
