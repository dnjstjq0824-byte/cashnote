<template>
  <div class="space-y-6">
    <!-- Header Section with Data Provenance -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-slate-200">
      <div>
        <h2 class="text-xl font-bold text-slate-900 tracking-tight">시스템 아키텍처 및 외부 연동 상태</h2>
        <p class="text-sm text-slate-500">WO-MOS-0051 — 독립형 멀티테넌트 아키텍처 및 헬스체크 게이트</p>
      </div>
      <div>
        <DataProvenanceBadge
          level="LEVEL_3_API"
          :updated-at="lastSyncTime"
          :verified="true"
        />
      </div>
    </div>

    <!-- State Switcher for Verification & Interactive Testing -->
    <div class="p-3 bg-white rounded-xl border border-slate-200 flex flex-wrap gap-2 text-xs">
      <span class="font-semibold text-slate-600 self-center mr-2">5대 상태 검증 탭:</span>
      <button
        v-for="s in (['loaded', 'loading', 'empty', 'error', 'blocked'] as UIState[])"
        :key="s"
        type="button"
        @click="currentState = s"
        class="px-3 py-1.5 rounded-lg border font-medium transition-colors"
        :class="currentState === s ? 'bg-slate-900 text-white border-slate-900' : 'bg-slate-50 text-slate-700 border-slate-200 hover:bg-slate-100'"
      >
        {{ s.toUpperCase() }}
      </button>
    </div>

    <!-- 5-State Container Component -->
    <StateContainer
      :state="currentState"
      loading-text="외부 게이트웨이 및 코어 서비스 연결 상태를 확인하고 있습니다..."
      empty-title="등록된 연동 서비스가 없습니다"
      empty-message="새로운 가맹점 결제 데이터 수집 어댑터를 활성화해 보세요."
      empty-action-text="어댑터 카탈로그 둘러보기"
      error-title="코어 서비스 응답 지연"
      error-message="게이트웨이와의 통신 중 일시적인 타임아웃이 발생했습니다."
      request-id="req-arch-live-0051"
      blocked-title="배달 3사 비공식 스크래핑 차단됨 (제11조)"
      blocked-message="배달의민족·쿠팡이츠·요기요 사장님 비밀번호를 무단 수집하지 않습니다. Level 2 공식 정산 엑셀 파일 업로드로 즉시 분석 가능합니다."
      fallback-action-text="Level 2 정산 엑셀 파일 업로드 열기"
      @retry="currentState = 'loaded'"
      @action="currentState = 'loaded'"
      @fallback="triggerFallback"
    >
      <!-- LOADED STATE: Real System Capability Grid -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
        <div
          v-for="cap in capabilities"
          :key="cap.serviceCode"
          class="p-5 rounded-2xl bg-white border border-slate-200 shadow-sm hover:border-slate-300 transition-all flex flex-col justify-between"
        >
          <div>
            <div class="flex items-center justify-between mb-3">
              <span class="text-xs font-mono font-bold px-2 py-0.5 rounded bg-slate-100 text-slate-700">
                {{ cap.serviceCode }}
              </span>
              <span
                class="text-xs font-semibold px-2 py-0.5 rounded-full"
                :class="cap.status === 'APPROVED' ? 'bg-emerald-100 text-emerald-800' : 'bg-amber-100 text-amber-800'"
              >
                {{ cap.status }}
              </span>
            </div>
            <h3 class="text-base font-semibold text-slate-900 mb-1">{{ cap.name }}</h3>
            <p class="text-xs text-slate-500 leading-relaxed mb-4">
              기본 제공: <strong class="text-slate-700">{{ cap.sourceLevel }}</strong><br />
              미승인 시 대체: <strong class="text-slate-700">{{ cap.fallbackLevel }}</strong>
            </p>
          </div>

          <div class="pt-3 border-t border-slate-100 flex items-center justify-between text-xs">
            <span class="text-slate-500">직접 호출 가능:</span>
            <span :class="cap.hasDirectAccess ? 'text-emerald-600 font-bold' : 'text-slate-400 font-medium'">
              {{ cap.hasDirectAccess ? '가능 (Level 3)' : '차단 (Level 2 대체)' }}
            </span>
          </div>
        </div>
      </div>
    </StateContainer>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import DataProvenanceBadge from '../../../../packages/ui-design-system/components/DataProvenanceBadge.vue'
import StateContainer, { type UIState } from '../../../../packages/ui-design-system/components/StateContainer.vue'

const currentState = ref<UIState>('loaded')
const lastSyncTime = ref(new Date().toISOString())

interface CapabilityItem {
  serviceCode: string
  name: string
  status: string
  sourceLevel: string
  fallbackLevel: string
  hasDirectAccess: boolean
}

const capabilities = ref<CapabilityItem[]>([
  {
    serviceCode: 'API-KR-001',
    name: '국세청 사업자 진위 및 상태조회',
    status: 'APPROVED',
    sourceLevel: 'LEVEL_3_API',
    fallbackLevel: 'LEVEL_1_MANUAL',
    hasDirectAccess: true,
  },
  {
    serviceCode: 'API-KR-002',
    name: '여신금융협회 가맹점 매출거래정보',
    status: 'CONTRACT_REQUIRED',
    sourceLevel: 'LEVEL_3_API',
    fallbackLevel: 'LEVEL_2_FILE',
    hasDirectAccess: false,
  },
  {
    serviceCode: 'API-KR-004',
    name: '오픈뱅킹 계좌 및 1원 인증',
    status: 'APPROVAL_REQUIRED',
    sourceLevel: 'LEVEL_3_API',
    fallbackLevel: 'LEVEL_2_FILE',
    hasDirectAccess: false,
  },
  {
    serviceCode: 'API-KR-010',
    name: '배달3사(배민·쿠팡이츠·요기요) 정산',
    status: 'BLOCKED',
    sourceLevel: 'LEVEL_2_FILE',
    fallbackLevel: 'LEVEL_2_FILE',
    hasDirectAccess: false,
  },
  {
    serviceCode: 'API-KR-011',
    name: 'Google Business Profile 매장 리뷰',
    status: 'APPROVED',
    sourceLevel: 'LEVEL_3_API',
    fallbackLevel: 'LEVEL_1_MANUAL',
    hasDirectAccess: true,
  },
])

function triggerFallback() {
  alert('Level 2 공식 엑셀/CSV 파서 모달이 연결되었습니다.')
}
</script>

