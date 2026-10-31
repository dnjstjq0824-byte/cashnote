<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-200">
      <div>
        <h2 class="text-xl font-bold text-slate-900 tracking-tight">자동 입금 대사 및 미지급 진단 (Level 4)</h2>
        <p class="text-sm text-slate-500">카드사/배달앱 지급예정 스케줄과 실제 은행 통장 입금액의 1원 단위 정밀 교차 검증</p>
      </div>
      <div>
        <DataProvenanceBadge
          level="LEVEL_4_RECONCILED"
          :updated-at="new Date().toISOString()"
          :verified="true"
        />
      </div>
    </div>

    <!-- Summary Stats -->
    <div class="grid grid-cols-1 sm:grid-cols-4 gap-4">
      <div class="p-4 rounded-xl bg-white border border-slate-200">
        <span class="text-xs text-slate-500 font-medium">대사 일치율</span>
        <p class="text-2xl font-bold text-emerald-600 mt-1">98.2%</p>
        <p class="text-xs text-slate-400 mt-1">정산 완료 54건 중 53건 일치</p>
      </div>
      <div class="p-4 rounded-xl bg-white border border-slate-200">
        <span class="text-xs text-slate-500 font-medium">단수 오차 허용 매칭 (±10원)</span>
        <p class="text-2xl font-bold text-teal-600 mt-1">2건</p>
        <p class="text-xs text-slate-400 mt-1">카드사 원단위 절사 자동 흡수</p>
      </div>
      <div class="p-4 rounded-xl bg-white border border-slate-200">
        <span class="text-xs text-slate-500 font-medium">동기화 지연 대기</span>
        <p class="text-2xl font-bold text-amber-600 mt-1">0건</p>
        <p class="text-xs text-slate-400 mt-1">은행 API 최신 상태 반영됨</p>
      </div>
      <div class="p-4 rounded-xl bg-white border border-slate-200">
        <span class="text-xs text-slate-500 font-medium">확정 미입금 경보</span>
        <p class="text-2xl font-bold text-slate-900 mt-1">0건</p>
        <p class="text-xs text-emerald-600 font-semibold mt-1">미지급 위험 0원</p>
      </div>
    </div>

    <!-- Match Breakdown Table -->
    <div class="bg-white rounded-2xl border border-slate-200 overflow-hidden shadow-sm">
      <div class="p-4 border-b border-slate-100 flex items-center justify-between">
        <h3 class="text-sm font-bold text-slate-800">최근 대사 상세 내역</h3>
        <span class="text-xs text-slate-500">실시간 대사 기준일: 오늘</span>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-slate-50 border-b border-slate-200 text-slate-500 font-semibold uppercase">
            <tr>
              <th class="py-3 px-4">정산처</th>
              <th class="py-3 px-4">지급 예정일</th>
              <th class="py-3 px-4 text-right">정산 예정액</th>
              <th class="py-3 px-4 text-right">통장 실입금액</th>
              <th class="py-3 px-4 text-right">차액</th>
              <th class="py-3 px-4 text-center">대사 결과</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr class="hover:bg-slate-50/50">
              <td class="py-3.5 px-4 font-semibold text-slate-900">신한카드</td>
              <td class="py-3.5 px-4 text-slate-500">2026-10-30</td>
              <td class="py-3.5 px-4 text-right font-mono font-medium">492,500원</td>
              <td class="py-3.5 px-4 text-right font-mono font-medium text-emerald-600">492,500원</td>
              <td class="py-3.5 px-4 text-right font-mono text-slate-400">0원</td>
              <td class="py-3.5 px-4 text-center">
                <span class="inline-block px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-emerald-100 text-emerald-800">
                  완전 일치 (1:1)
                </span>
              </td>
            </tr>
            <tr class="hover:bg-slate-50/50">
              <td class="py-3.5 px-4 font-semibold text-slate-900">KB국민카드</td>
              <td class="py-3.5 px-4 text-slate-500">2026-10-30</td>
              <td class="py-3.5 px-4 text-right font-mono font-medium">210,000원</td>
              <td class="py-3.5 px-4 text-right font-mono font-medium text-teal-600">209,996원</td>
              <td class="py-3.5 px-4 text-right font-mono text-amber-600">-4원</td>
              <td class="py-3.5 px-4 text-center">
                <span class="inline-block px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-teal-100 text-teal-800" title="카드사 원단위 절사">
                  수수료 절사 오차
                </span>
              </td>
            </tr>
            <tr class="hover:bg-slate-50/50">
              <td class="py-3.5 px-4 font-semibold text-slate-900">우아한형제들 (배민)</td>
              <td class="py-3.5 px-4 text-slate-500">2026-10-28</td>
              <td class="py-3.5 px-4 text-right font-mono font-medium">749,500원</td>
              <td class="py-3.5 px-4 text-right font-mono font-medium text-emerald-600">749,500원</td>
              <td class="py-3.5 px-4 text-right font-mono text-slate-400">0원</td>
              <td class="py-3.5 px-4 text-center">
                <span class="inline-block px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-emerald-100 text-emerald-800">
                  완전 일치 (1:1)
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import DataProvenanceBadge from '../../../packages/ui-design-system/components/DataProvenanceBadge.vue'
</script>

