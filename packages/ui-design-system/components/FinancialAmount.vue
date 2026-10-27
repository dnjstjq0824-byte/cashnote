<template>
  <span
    class="inline-block font-mono tracking-tight font-semibold"
    :class="[colorClass, sizeClass]"
    style="font-variant-numeric: tabular-nums;"
  >
    <span v-if="prefix" class="mr-0.5 text-xs opacity-70">{{ prefix }}</span>
    <span>{{ formattedValue }}</span>
    <span v-if="unit" class="ml-0.5 text-xs font-normal opacity-80">{{ unit }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export type FinanceCategory = 'authorized' | 'scheduled' | 'reconciled' | 'warning' | 'danger' | 'expense' | 'neutral'

const props = withDefaults(
  defineProps<{
    value: number | string
    category?: FinanceCategory
    size?: 'sm' | 'md' | 'lg' | 'hero'
    prefix?: string
    unit?: string
    largeFont?: boolean
  }>(),
  {
    category: 'neutral',
    size: 'md',
    prefix: '',
    unit: '원',
    largeFont: false,
  }
)

const colorClass = computed(() => {
  switch (props.category) {
    case 'reconciled':
      return 'text-emerald-700'
    case 'authorized':
      return 'text-blue-700'
    case 'scheduled':
      return 'text-teal-700'
    case 'warning':
      return 'text-amber-700'
    case 'danger':
      return 'text-rose-700'
    case 'expense':
      return 'text-slate-600'
    case 'neutral':
    default:
      return 'text-slate-900'
  }
})

const sizeClass = computed(() => {
  if (props.largeFont) {
    switch (props.size) {
      case 'hero':
        return 'text-4xl font-extrabold'
      case 'lg':
        return 'text-2xl'
      case 'sm':
        return 'text-base'
      case 'md':
      default:
        return 'text-lg'
    }
  }

  switch (props.size) {
    case 'hero':
      return 'text-3xl font-extrabold'
    case 'lg':
      return 'text-xl'
    case 'sm':
      return 'text-sm'
    case 'md':
    default:
      return 'text-base'
  }
})

const formattedValue = computed(() => {
  const num = typeof props.value === 'string' ? parseFloat(props.value) : props.value
  if (isNaN(num)) return '0'
  return new Intl.NumberFormat('ko-KR').format(num)
})
</script>
