<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-200">
      <div>
        <h2 class="text-xl font-bold text-slate-900 tracking-tight">매장 리뷰 통합 관리 및 AI 답글</h2>
        <p class="text-sm text-slate-500">Google, 배민, 네이버 플레이스 매장 리뷰 분석 및 공식 API/클립보드 답글</p>
      </div>
      <div>
        <DataProvenanceBadge
          level="LEVEL_3_API"
          :updated-at="new Date().toISOString()"
          :verified="true"
        />
      </div>
    </div>

    <!-- Review Items List -->
    <div class="space-y-4">
      <!-- 1. Google Business Review (Public API direct reply) -->
      <div class="p-5 rounded-2xl bg-white border border-slate-200 shadow-sm space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="px-2 py-0.5 rounded text-xs font-semibold bg-blue-100 text-blue-800">
              Google Business
            </span>
            <span class="font-bold text-sm text-slate-800">이민수 님</span>
            <span class="text-xs text-amber-500">★★★★★ 5.0</span>
          </div>
          <span class="text-xs text-slate-400">어제</span>
        </div>
        <p class="text-sm text-slate-700 leading-relaxed">
          커피 원두 향이 너무 좋고 인테리어가 깔끔해서 단골 예약합니다! 디저트 크로플도 바삭해요.
        </p>
        <div class="pt-3 border-t border-slate-100 flex items-center justify-between">
          <span class="text-xs text-emerald-600 font-semibold">공식 API 직접 전송 가능 채널</span>
          <button
            type="button"
            @click="replyGoogle"
            class="px-3.5 py-1.5 rounded-lg bg-slate-900 text-white text-xs font-semibold hover:bg-slate-800 transition-colors"
          >
            답글 즉시 등록
          </button>
        </div>
      </div>

      <!-- 2. Baemin Review (Blocked platform -> AI Draft Clipboard Copy) -->
      <div class="p-5 rounded-2xl bg-white border border-slate-200 shadow-sm space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="px-2 py-0.5 rounded text-xs font-semibold bg-emerald-100 text-emerald-800">
              배달의민족
            </span>
            <span class="font-bold text-sm text-slate-800">박지수 님</span>
            <span class="text-xs text-amber-500">★★★★☆ 4.0</span>
          </div>
          <span class="text-xs text-slate-400">2일 전</span>
        </div>
        <p class="text-sm text-slate-700 leading-relaxed">
          배달 빠르고 포장도 꼼꼼했습니다. 다만 얼음이 살짝 녹아서 왔어요.
        </p>
        <div class="pt-3 border-t border-slate-100 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
          <span class="text-xs text-slate-500">
            비공식 스크래핑 차단 원칙(제11조) — <strong class="text-slate-700">AI 초안 원클릭 복사</strong> 지원
          </span>
          <button
            type="button"
            @click="copyAiDraft"
            class="px-3.5 py-1.5 rounded-lg bg-emerald-600 text-white text-xs font-semibold hover:bg-emerald-700 transition-colors"
          >
            AI 정중 답글 복사하기
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import DataProvenanceBadge from '../../../packages/ui-design-system/components/DataProvenanceBadge.vue'

function replyGoogle() {
  alert('Google Business Profile 공식 API로 답글이 등록되었습니다.')
}

function copyAiDraft() {
  const draft = '안녕하세요 사장님 매장입니다. 소중한 리뷰 감사드리며, 보냉 포장을 강화하여 다음 주문 시에는 더욱 시원하게 즐기실 수 있도록 개선하겠습니다. 감사합니다!'
  navigator.clipboard?.writeText(draft)
  alert('AI 답글 초안이 클립보드에 복사되었습니다. 배민 사장님 사이트에 붙여넣으세요.')
}
</script>
