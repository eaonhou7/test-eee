<template>
  <el-select
    v-if="hasOptions"
    :model-value="modelValue"
    filterable
    clearable
    class="!w-full"
    @update:model-value="updateValue"
  >
    <el-option v-for="item in resolvedOptions" :key="item" :label="item" :value="item" />
  </el-select>
  <el-switch
    v-else-if="field.dataType === 'boolean'"
    :model-value="Boolean(modelValue)"
    @update:model-value="updateValue"
  />
  <el-input-number
    v-else-if="field.dataType === 'integer' || field.dataType === 'number'"
    :model-value="toNumber(modelValue)"
    :precision="field.dataType === 'integer' ? 0 : 2"
    :step="field.dataType === 'integer' ? 1 : 0.1"
    class="!w-full"
    @update:model-value="updateValue"
  />
  <el-input
    v-else
    :model-value="modelValue"
    :type="isLongText ? 'textarea' : 'text'"
    :rows="isLongText ? 3 : undefined"
    @update:model-value="updateValue"
  />
</template>

<script setup>
  import { computed } from 'vue'

  const props = defineProps({
    field: {
      type: Object,
      required: true
    },
    modelValue: {
      type: [String, Number, Boolean],
      default: ''
    },
    fallbackOptions: {
      type: Array,
      default: () => []
    }
  })

  const emit = defineEmits(['update:modelValue'])

  const normalizeOptions = (values = []) =>
    Array.from(
      new Set(
        values
          .map((item) => String(item || '').trim())
          .filter(Boolean)
      )
    )

  const ruleOptions = computed(() => {
    const rule = props.field.rule || {}
    const candidates = [
      ...(Array.isArray(rule.options) ? rule.options : []),
      ...(Array.isArray(rule.enumValues) ? rule.enumValues : []),
      ...(Array.isArray(rule.choices) ? rule.choices : []),
      ...(Array.isArray(rule.values) ? rule.values : [])
    ]
    return normalizeOptions(candidates)
  })

  const resolvedOptions = computed(() =>
    normalizeOptions([
      ...((props.field.enumValues || [])),
      ...(ruleOptions.value || []),
      ...(props.fallbackOptions || [])
    ])
  )

  const hasOptions = computed(() => resolvedOptions.value.length > 0)
  const isLongText = computed(() => {
    const key = `${props.field.fieldKey || ''}${props.field.columnHeader || ''}`.toLowerCase()
    return key.includes('description') || key.includes('bullet') || key.includes('search')
  })

  const updateValue = (value) => {
    emit('update:modelValue', value)
  }

  const toNumber = (value) => {
    if (value === '' || value === null || typeof value === 'undefined') {
      return undefined
    }
    const parsed = Number(value)
    return Number.isNaN(parsed) ? undefined : parsed
  }
</script>
