<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-200">
      <div>
        <h2 class="text-xl font-bold text-slate-900 tracking-tight">부가가치세 실시간 추정 계산기</h2>
        <p class="text-sm text-slate-500">홈택스 전자세금계산서·신용카드 매입전표를 반영한 실시간 예상 납부세액</p>
      </div>
      <div>
        <DataProvenanceBadge
          level="LEVEL_3_API"
          :updated-at="new Date().toISOString()"
          :verified="true"
        />
      </div>
    </div>

    <!-- Tax Hero Card -->
    <div class="p-6 rounded-2xl bg-white border border-slate-200 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-6">
      <div>
        <div class="flex items-center gap-2 mb-2">
          <span class="px-2.5 py-0.5 rounded text-xs font-bold bg-slate-900 text-white">2026년 2기 확정</span>
          <span class="text-xs text-slate-500">일반과세자 (음식점/카페)</span>
        </div>
        <p class="text-xs text-slate-500 font-medium">현재까지 예상 부가가치세 납부세액</p>
        <p class="text-3xl font-extrabold text-slate-900 font-mono mt-1">4,060,000 원</p>
      </div>

      <div class="p-4 rounded-xl bg-slate-50 border border-slate-200 text-xs space-y-1.5 min-w-[240px]">
        <div class="flex justify-between">
          <span class="text-slate-500">신고 납부기한:</span>
          <span class="font-bold text-slate-800">2027-01-25</span>
        </div>
        <div class="flex justify-between">
          <span class="text-slate-500">남은 일수:</span>
          <span class="font-bold text-emerald-600">86일 남음</span>
        </div>
        <div class="flex justify-between">
          <span class="text-slate-500">적격증빙 집계:</span>
          <span class="font-bold text-slate-800">12건 완료</span>
        </div>
      </div>
    </div>

    <!-- Calculation Formula Grid -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-5">
      <div class="p-5 rounded-2xl bg-white border border-slate-200 shadow-sm space-y-2">
        <h3 class="text-sm font-bold text-slate-800">1. 매출세액 (10%)</h3>
        <p class="text-xl font-bold text-blue-600 font-mono">+ 10,000,000원</p>
        <p class="text-xs text-slate-500">과세매출 공급가액 100,000,000원의 10%</p>
      </div>

      <div class="p-5 rounded-2xl bg-white border border-slate-200 shadow-sm space-y-2">
        <h3 class="text-sm font-bold text-slate-800">2. 공제가능 매입세액</h3>
        <p class="text-xl font-bold text-rose-600 font-mono">- 4,000,000원</p>
        <p class="text-xs text-slate-500">전자세금계산서 및 사업용 카드 매입전표</p>
      </div>

      <div class="p-5 rounded-2xl bg-white border border-slate-200 shadow-sm space-y-2">
        <h3 class="text-sm font-bold text-slate-800">3. 추가 감면 공제</h3>
        <p class="text-xl font-bold text-emerald-600 font-mono">- 1,940,000원</p>
        <p class="text-xs text-slate-500">신용카드발행 1.3%(104만) + 의제매입 9/109(90만)</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import DataProvenanceBadge from '../../../packages/ui-design-system/components/DataProvenanceBadge.vue'
</script>
