<template>
  <div
    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium tracking-wide transition-all duration-150"
    :class="badgeClass"
    :title="tooltipText"
  >
    <span class="w-1.5 h-1.5 rounded-full" :class="dotClass" />
    <span>{{ levelLabel }}</span>
    <span v-if="formattedTime" class="text-[10px] opacity-75 font-mono">
      {{ formattedTime }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export type ProvenanceLevel = 'LEVEL_1_MANUAL' | 'LEVEL_2_FILE' | 'LEVEL_3_API' | 'LEVEL_4_RECONCILED'

const props = defineProps<{
  level: ProvenanceLevel
  updatedAt?: string | Date
  verified?: boolean
}>()

const levelLabel = computed(() => {
  switch (props.level) {
    case 'LEVEL_1_MANUAL':
      return '수기 입력 (L1)'
    case 'LEVEL_2_FILE':
      return '정산서 파일 (L2)'
    case 'LEVEL_3_API':
      return '공식 연동 API (L3)'
    case 'LEVEL_4_RECONCILED':
      return '은행 입금 대사 완료 (L4)'
    default:
      return '출처 확인 중'
  }
})

const badgeClass = computed(() => {
  switch (props.level) {
    case 'LEVEL_4_RECONCILED':
      return 'bg-emerald-50 text-emerald-800 border border-emerald-200'
    case 'LEVEL_3_API':
      return 'bg-purple-50 text-purple-800 border border-purple-200'
    case 'LEVEL_2_FILE':
      return 'bg-blue-50 text-blue-800 border border-blue-200'
    case 'LEVEL_1_MANUAL':
    default:
      return 'bg-slate-100 text-slate-700 border border-slate-300'
  }
})

const dotClass = computed(() => {
  switch (props.level) {
    case 'LEVEL_4_RECONCILED':
      return 'bg-emerald-500'
    case 'LEVEL_3_API':
      return 'bg-purple-500'
    case 'LEVEL_2_FILE':
      return 'bg-blue-500'
    case 'LEVEL_1_MANUAL':
    default:
      return 'bg-slate-400'
  }
})

const tooltipText = computed(() => {
  return `${levelLabel.value} • 최종 검증: ${props.updatedAt || '방금 전'}`
})

const formattedTime = computed(() => {
  if (!props.updatedAt) return ''
  const d = new Date(props.updatedAt)
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
})
</script>
