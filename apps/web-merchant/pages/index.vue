<template>
  <div class="space-y-8">
    <!-- Top Greeting & Provenance Hero Header -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-6 border-b border-slate-200">
      <div>
        <div class="flex items-center gap-2">
          <h2 class="text-2xl font-bold text-slate-900 tracking-tight">BN카페 역삼역점</h2>
          <span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800 border border-emerald-300">
            소유권 인증 완료
          </span>
        </div>
        <p class="text-sm text-slate-500 mt-1">오늘의 실시간 승인·입금예정 및 대사 마감 브리핑</p>
      </div>

      <div class="flex items-center gap-3">
        <DataProvenanceBadge
          level="LEVEL_4_RECONCILED"
          :updated-at="currentTime"
          :verified="true"
        />
        <button
          type="button"
          @click="refreshData"
          class="px-3.5 py-2 rounded-xl bg-white border border-slate-200 text-slate-700 hover:bg-slate-50 text-xs font-medium shadow-sm transition-all"
        >
          새로고침
        </button>
      </div>
    </div>

    <!-- 4 Key Financial Metric Cards (Strict separation of Authorized, Scheduled, Reconciled, Expenses) -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
      <!-- 1. Authorized Revenue (승인 매출) -->
      <div class="p-5 rounded-2xl bg-white border border-slate-200 shadow-sm flex flex-col justify-between">
        <div>
          <div class="flex items-center justify-between text-xs text-slate-500 mb-2">
            <span>오늘 승인 매출</span>
            <span class="text-blue-600 font-medium">카드·배달·POS</span>
          </div>
          <FinancialAmount
            value="1850000"
            category="authorized"
            size="lg"
            unit="원"
          />
        </div>
        <div class="pt-3 border-t border-slate-100 text-xs text-slate-400 mt-4">
          총 42건 승인 완료
        </div>
      </div>

      <!-- 2. Scheduled Settlement (입금 예정) -->
      <div class="p-5 rounded-2xl bg-white border border-slate-200 shadow-sm flex flex-col justify-between">
        <div>
          <div class="flex items-center justify-between text-xs text-slate-500 mb-2">
            <span>내일 입금 예정</span>
            <span class="text-teal-600 font-medium">수수료 차감 후</span>
          </div>
          <FinancialAmount
            value="1822250"
            category="scheduled"
            size="lg"
            unit="원"
          />
        </div>
        <div class="pt-3 border-t border-slate-100 text-xs text-slate-400 mt-4">
          카드사 공제 수수료 -27,750원
        </div>
      </div>

      <!-- 3. Reconciled Deposit (통장 입금 완료) -->
      <div class="p-5 rounded-2xl bg-white border border-slate-200 shadow-sm flex flex-col justify-between ring-1 ring-emerald-500/20">
        <div>
          <div class="flex items-center justify-between text-xs text-slate-500 mb-2">
            <span>오늘 실제 입금 완료</span>
            <span class="text-emerald-700 font-bold">100% 대사 일치</span>
          </div>
          <FinancialAmount
            value="1452000"
            category="reconciled"
            size="lg"
            unit="원"
          />
        </div>
        <div class="pt-3 border-t border-slate-100 text-xs text-emerald-600 font-medium mt-4">
          신한카드 외 4건 계좌 입금 확인
        </div>
      </div>

      <!-- 4. Unpaid / Delayed Candidate (미지급 진단) -->
      <div class="p-5 rounded-2xl bg-white border border-slate-200 shadow-sm flex flex-col justify-between">
        <div>
          <div class="flex items-center justify-between text-xs text-slate-500 mb-2">
            <span>미입금 진단 경보</span>
            <span class="text-emerald-600 font-bold">이상 없음</span>
          </div>
          <FinancialAmount
            value="0"
            category="neutral"
            size="lg"
            unit="원"
          />
        </div>
        <div class="pt-3 border-t border-slate-100 text-xs text-slate-400 mt-4">
          지급 지연 및 보류 의심 0건
        </div>
      </div>
    </div>

    <!-- Quick Navigation Tabs -->
    <div class="flex border-b border-slate-200 gap-6 text-sm font-semibold">
      <NuxtLink to="/" class="pb-3 border-b-2 border-slate-900 text-slate-900">
        대시보드 종합
      </NuxtLink>
      <NuxtLink to="/reconciliation" class="pb-3 border-b-2 border-transparent text-slate-500 hover:text-slate-800 transition-colors">
        정산·입금 대사 (Level 4)
      </NuxtLink>
      <NuxtLink to="/tax" class="pb-3 border-b-2 border-transparent text-slate-500 hover:text-slate-800 transition-colors">
        부가세 예측 (세무팩)
      </NuxtLink>
      <NuxtLink to="/reviews" class="pb-3 border-b-2 border-transparent text-slate-500 hover:text-slate-800 transition-colors">
        리뷰·CRM 마케팅
      </NuxtLink>
    </div>

    <!-- Level 2 Universal Settlement File Upload Hero -->
    <div class="p-6 rounded-2xl bg-slate-900 text-white flex flex-col md:flex-row items-center justify-between gap-6 shadow-sm">
      <div class="space-y-2">
        <div class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full bg-blue-900/60 border border-blue-700 text-blue-300 text-xs font-semibold">
          <span>Level 2 파일 파서</span>
        </div>
        <h3 class="text-lg font-bold tracking-tight">배달의민족·쿠팡이츠·여신협회 정산서 1초 업로드</h3>
        <p class="text-xs text-slate-400 leading-relaxed max-w-xl">
          사장님의 비밀번호를 요구하지 않는 정직한 장부 시스템입니다. 사장님 사이트에서 다운로드한 엑셀/CSV 정산 명세서를 올리시면 11개 금융 속성(중개수수료, 배달비, 광고비, 실정산액)을 1원 단위로 자동 분석합니다.
        </p>
      </div>

      <div class="shrink-0 flex gap-3">
        <label class="cursor-pointer inline-flex items-center gap-2 px-5 py-3 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-slate-950 font-bold text-sm shadow transition-all">
          <input type="file" class="hidden" accept=".csv,.xlsx" @change="handleFileUpload" />
          <span>정산 파일 올리기</span>
        </label>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import DataProvenanceBadge from '../../../packages/ui-design-system/components/DataProvenanceBadge.vue'
import FinancialAmount from '../../../packages/ui-design-system/components/FinancialAmount.vue'

const currentTime = ref(new Date().toISOString())

function refreshData() {
  currentTime.value = new Date().toISOString()
}

function handleFileUpload(e: Event) {
  const target = e.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    alert(`${target.files[0].name} 파일이 선택되었습니다. 백엔드 Level 2 파서로 전송됩니다.`)
  }
}
</script>
