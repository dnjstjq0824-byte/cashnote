<template>
  <div class="w-full relative">
    <!-- 1. LOADING STATE -->
    <div v-if="state === 'loading'" class="w-full py-12 flex flex-col items-center justify-center space-y-4">
      <div class="w-8 h-8 border-4 border-slate-200 border-t-emerald-600 rounded-full animate-spin" />
      <p class="text-sm text-slate-500 font-medium tracking-tight">{{ loadingText || '데이터를 안전하게 동기화하고 있습니다...' }}</p>
    </div>

    <!-- 2. BLOCKED STATE (Regulatory / Partner Contract Gate) -->
    <div
      v-else-if="state === 'blocked'"
      class="w-full p-6 rounded-2xl bg-amber-50/70 border border-amber-200/80 flex flex-col items-start space-y-3"
    >
      <div class="flex items-center gap-2 text-amber-800 font-semibold text-base">
        <svg class="w-5 h-5 text-amber-600 shrink-0" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
        </svg>
        <span>{{ blockedTitle || '정식 연동 준비 중 (안내)' }}</span>
      </div>
      <p class="text-sm text-amber-900 leading-relaxed">
        {{ blockedMessage || '외부 플랫폼 계약 및 금융 인허가 심사 진행 중입니다. 사장님의 로그인 비밀번호를 무단 수집하지 않는 원칙을 준수합니다.' }}
      </p>
      <div v-if="fallbackActionText" class="pt-1">
        <button
          type="button"
          @click="$emit('fallback')"
          class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-amber-600 hover:bg-amber-700 text-white text-sm font-medium shadow-sm transition-colors duration-150"
        >
          {{ fallbackActionText }}
        </button>
      </div>
    </div>

    <!-- 3. ERROR STATE -->
    <div
      v-else-if="state === 'error'"
      class="w-full p-6 rounded-2xl bg-rose-50/70 border border-rose-200/80 flex flex-col items-start space-y-3"
    >
      <div class="flex items-center gap-2 text-rose-800 font-semibold text-base">
        <svg class="w-5 h-5 text-rose-600 shrink-0" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
        </svg>
        <span>{{ errorTitle || '일시적 처리 오류' }}</span>
      </div>
      <p class="text-sm text-rose-900 leading-relaxed">{{ errorMessage || '일시적인 장애가 발생했습니다. 잠시 후 다시 시도해 주세요.' }}</p>
      <div v-if="requestId" class="text-xs text-rose-600 font-mono">
        요청 ID: {{ requestId }}
      </div>
      <div class="pt-1">
        <button
          type="button"
          @click="$emit('retry')"
          class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-rose-600 hover:bg-rose-700 text-white text-sm font-medium shadow-sm transition-colors duration-150"
        >
          다시 시도
        </button>
      </div>
    </div>

    <!-- 4. EMPTY STATE -->
    <div
      v-else-if="state === 'empty'"
      class="w-full py-16 px-6 text-center rounded-2xl bg-slate-50/70 border border-slate-200/80 flex flex-col items-center justify-center space-y-3"
    >
      <div class="w-12 h-12 rounded-full bg-slate-100 flex items-center justify-center text-slate-400">
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
        </svg>
      </div>
      <h3 class="text-base font-semibold text-slate-800">{{ emptyTitle || '등록된 내역이 없습니다' }}</h3>
      <p class="text-sm text-slate-500 max-w-sm leading-relaxed">{{ emptyMessage || '새로운 매출을 등록하거나 정산 파일을 업로드하여 장부를 시작해 보세요.' }}</p>
      <div v-if="emptyActionText" class="pt-2">
        <button
          type="button"
          @click="$emit('action')"
          class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-slate-900 hover:bg-slate-800 text-white text-sm font-medium shadow-sm transition-colors duration-150"
        >
          {{ emptyActionText }}
        </button>
      </div>
    </div>

    <!-- 5. LOADED (SUCCESS) STATE -->
    <div v-else class="w-full">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
export type UIState = 'loading' | 'loaded' | 'empty' | 'error' | 'blocked'

defineProps<{
  state: UIState
  loadingText?: string
  emptyTitle?: string
  emptyMessage?: string
  emptyActionText?: string
  errorTitle?: string
  errorMessage?: string
  requestId?: string
  blockedTitle?: string
  blockedMessage?: string
  fallbackActionText?: string
}>()

defineEmits<{
  (e: 'retry'): void
  (e: 'action'): void
  (e: 'fallback'): void
}>()
</script>

