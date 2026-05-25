<template>
  <div class="flex flex-col gap-3">
    <div class="flex justify-between">
      <span class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ title }}</span>
      <el-button size="small" @click="addRow">新增图片</el-button>
    </div>

    <div
      v-for="(image, index) in localValue"
      :key="image.id || image.fileId || `${index}-${image.slotCode || 'image'}`"
      class="rounded-lg border border-slate-200 bg-white p-3 dark:border-slate-700 dark:bg-slate-900/60"
    >
      <div class="grid gap-3 md:grid-cols-[140px_1fr_110px_110px_90px]">
        <el-input :model-value="image.slotCode" placeholder="槽位，如 MAIN/PT1" @update:model-value="updateRow(index, { slotCode: $event })" />
        <div class="flex gap-2">
          <el-input :model-value="image.imageUrl" placeholder="上传后自动回填" @update:model-value="updateRow(index, { imageUrl: $event })" />
          <el-upload
            :show-file-list="false"
            :http-request="(options) => uploadImage(index, options)"
            accept=".jpg,.jpeg,.png,.webp,.gif"
          >
            <el-button>上传</el-button>
          </el-upload>
        </div>
        <el-switch :model-value="Boolean(image.isPrimary)" active-text="主图" inactive-text="普通图" @update:model-value="updateRow(index, { isPrimary: $event })" />
        <el-input-number :model-value="image.sort" :min="1" class="!w-full" @update:model-value="updateRow(index, { sort: $event || index + 1 })" />
        <el-button type="danger" link @click="removeRow(index)">删除</el-button>
      </div>
      <img
        v-if="image.imageUrl"
        :src="resolveImageUrl(image.imageUrl)"
        class="mt-3 h-24 w-24 rounded-lg border border-slate-200 object-cover dark:border-slate-700"
      />
    </div>
  </div>
</template>

<script setup>
  import { computed } from 'vue'
  import { ElMessage } from 'element-plus'

  import { uploadAmazonListingImage } from '@/api/amazonListingImage'
  import { getUrl } from '@/utils/image'

  const props = defineProps({
    modelValue: {
      type: Array,
      default: () => []
    },
    title: {
      type: String,
      default: '图片列表'
    }
  })

  const emit = defineEmits(['update:modelValue'])

  const localValue = computed({
    get: () => props.modelValue || [],
    set: (value) => emit('update:modelValue', value)
  })

  const normalizeImages = (images = []) =>
    images.map((image, index) => ({
      slotCode: '',
      fileId: 0,
      imageUrl: '',
      sort: index + 1,
      isPrimary: index === 0,
      ...image,
      sort: Number(image?.sort) > 0 ? Number(image.sort) : index + 1
    }))

  const applyImages = (images) => {
    localValue.value = normalizeImages(images)
  }

  const addRow = () => {
    applyImages([
      ...localValue.value,
      {
        slotCode: '',
        fileId: 0,
        imageUrl: '',
        sort: localValue.value.length + 1,
        isPrimary: localValue.value.length === 0
      }
    ])
  }

  const removeRow = (index) => {
    const next = [...localValue.value]
    const [removed] = next.splice(index, 1)
    if (removed?.isPrimary && next.length && !next.some((item) => item.isPrimary)) {
      next[0] = {
        ...next[0],
        isPrimary: true
      }
    }
    applyImages(
      next.map((item, itemIndex) => ({
        ...item,
        sort: itemIndex + 1
      }))
    )
  }

  const updateRow = (index, patch = {}) => {
    const next = localValue.value.map((image, imageIndex) => {
      if (imageIndex !== index) {
        return patch.isPrimary ? { ...image, isPrimary: false } : { ...image }
      }
      return {
        ...image,
        ...patch
      }
    })
    applyImages(next)
  }

  const defaultSlotCode = (index, currentSlotCode = '') => {
    if (currentSlotCode) {
      return currentSlotCode
    }
    return index === 0 ? 'MAIN' : `PT${index}`
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

  const uploadImage = async (index, options) => {
    const current = localValue.value[index] || {}
    try {
      const res = await uploadAmazonListingImage(options.file)
      const payload = res.data || {}
      if (!payload.imageUrl) {
        throw new Error('上传成功但未返回图片地址')
      }
      updateRow(index, {
        fileId: payload.fileId || 0,
        imageUrl: payload.imageUrl,
        slotCode: defaultSlotCode(index, current.slotCode)
      })
      options.onSuccess?.(payload)
      ElMessage.success('图片上传成功')
    } catch (error) {
      options.onError?.(error)
      ElMessage.error(error?.response?.data?.msg || error?.message || '图片上传失败')
    }
  }
</script>
