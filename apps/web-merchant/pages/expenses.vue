<template>
  <div class="min-h-screen bg-slate-900 text-slate-100 p-6 space-y-6">
    <!-- Header -->
    <header class="flex items-center justify-between border-b border-slate-800 pb-4">
      <div>
        <div class="flex items-center space-x-3">
          <h1 class="text-2xl font-bold text-white tracking-tight">매장 지출 및 거래처 관리 (Expenses & Vendors)</h1>
          <span class="px-2.5 py-0.5 text-xs font-semibold rounded-full bg-indigo-500/20 text-indigo-400 border border-indigo-500/30">
            Phase 2 Level 1~2
          </span>
        </div>
        <p class="text-sm text-slate-400 mt-1">
          임차료·인건비 등 고정비와 식자재·배달소모품 등 변동비를 분리하고 세무 적격증빙을 관리합니다.
        </p>
      </div>
      <div class="flex items-center space-x-3">
        <DataProvenanceBadge source-level="LEVEL_1_MANUAL" :verified="true" :updated-at="lastUpdatedAt" />
        <button @click="showVendorModal = true" class="px-3.5 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg text-sm font-medium transition">
          + 거래처 등록
        </button>
        <button @click="showExpenseModal = true" class="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-sm font-medium shadow-lg transition">
          + 지출 등록
        </button>
      </div>
    </header>

    <!-- Expense Structure Statistics Cards -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div class="bg-slate-800/80 rounded-xl p-5 border border-slate-700 shadow-sm">
        <div class="text-xs font-semibold text-slate-400 uppercase tracking-wider">총 지출 금액 (Total Expenses)</div>
        <div class="text-3xl font-extrabold text-white mt-2">₩{{ formatNumber(summary.total) }}</div>
        <div class="text-xs text-slate-400 mt-1">공급가액 ₩{{ formatNumber(summary.supply) }} + 부가세 ₩{{ formatNumber(summary.vat) }}</div>
      </div>

      <div class="bg-slate-800/80 rounded-xl p-5 border border-slate-700 shadow-sm">
        <div class="text-xs font-semibold text-blue-400 uppercase tracking-wider">고정비용 (Fixed: 임차료/인건비)</div>
        <div class="text-3xl font-extrabold text-blue-300 mt-2">₩{{ formatNumber(summary.fixed) }}</div>
        <div class="text-xs text-slate-400 mt-1">전체 지출의 약 {{ summary.fixedRate }}% 점유</div>
      </div>

      <div class="bg-slate-800/80 rounded-xl p-5 border border-slate-700 shadow-sm">
        <div class="text-xs font-semibold text-amber-400 uppercase tracking-wider">변동비용 (Variable: 식자재/소모품)</div>
        <div class="text-3xl font-extrabold text-amber-300 mt-2">₩{{ formatNumber(summary.variable) }}</div>
        <div class="text-xs text-slate-400 mt-1">매출 연동 비용 {{ summary.variableRate }}%</div>
      </div>
    </div>

    <!-- Main Content Tabs -->
    <div class="bg-slate-800/60 rounded-xl border border-slate-700/80 p-5 space-y-4">
      <div class="flex items-center justify-between border-b border-slate-700 pb-3">
        <div class="flex space-x-2">
          <button
            @click="activeTab = 'expenses'"
            :class="['px-4 py-2 rounded-lg text-sm font-medium transition', activeTab === 'expenses' ? 'bg-slate-700 text-emerald-400 shadow' : 'text-slate-400 hover:text-white']"
          >
            지출 내역 목록 (Expenses)
          </button>
          <button
            @click="activeTab = 'vendors'"
            :class="['px-4 py-2 rounded-lg text-sm font-medium transition', activeTab === 'vendors' ? 'bg-slate-700 text-emerald-400 shadow' : 'text-slate-400 hover:text-white']"
          >
            등록 거래처 목록 (Vendors)
          </button>
        </div>
      </div>

      <!-- Tab 1: Expense List -->
      <div v-if="activeTab === 'expenses'" class="overflow-x-auto">
        <table class="w-full text-left text-sm text-slate-300">
          <thead class="text-xs uppercase bg-slate-900/60 text-slate-400 border-b border-slate-700">
            <tr>
              <th class="py-3 px-4">지출일자</th>
              <th class="py-3 px-4">지출항목</th>
              <th class="py-3 px-4">거래처명</th>
              <th class="py-3 px-4">적격증빙</th>
              <th class="py-3 px-4">결제수단</th>
              <th class="py-3 px-4 text-right">공급가액</th>
              <th class="py-3 px-4 text-right">부가세</th>
              <th class="py-3 px-4 text-right">합계금액</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800">
            <tr v-for="exp in expenses" :key="exp.id" class="hover:bg-slate-750">
              <td class="py-3 px-4 text-slate-400">{{ formatDate(exp.expense_date) }}</td>
              <td class="py-3 px-4">
                <span :class="['px-2 py-0.5 text-xs rounded-full font-medium', exp.is_fixed ? 'bg-blue-900/40 text-blue-300 border border-blue-500/30' : 'bg-amber-900/40 text-amber-300 border border-amber-500/30']">
                  {{ exp.category_label }}
                </span>
              </td>
              <td class="py-3 px-4 font-medium text-white">{{ exp.vendor_name }}</td>
              <td class="py-3 px-4">
                <span class="text-xs px-2 py-0.5 bg-slate-900 text-slate-300 rounded border border-slate-700">{{ exp.proof_type_label }}</span>
              </td>
              <td class="py-3 px-4 text-xs text-slate-400">{{ exp.payment_method }}</td>
              <td class="py-3 px-4 text-right font-mono text-slate-300">₩{{ formatNumber(exp.supply_amount) }}</td>
              <td class="py-3 px-4 text-right font-mono text-slate-400">₩{{ formatNumber(exp.vat_amount) }}</td>
              <td class="py-3 px-4 text-right font-mono font-bold text-white">₩{{ formatNumber(exp.total_amount) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Tab 2: Vendor List -->
      <div v-if="activeTab === 'vendors'" class="overflow-x-auto">
        <table class="w-full text-left text-sm text-slate-300">
          <thead class="text-xs uppercase bg-slate-900/60 text-slate-400 border-b border-slate-700">
            <tr>
              <th class="py-3 px-4">상호명</th>
              <th class="py-3 px-4">사업자등록번호</th>
              <th class="py-3 px-4">대표자명</th>
              <th class="py-3 px-4">업태/종목</th>
              <th class="py-3 px-4">연락처</th>
              <th class="py-3 px-4">비고</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800">
            <tr v-for="v in vendors" :key="v.id" class="hover:bg-slate-750">
              <td class="py-3 px-4 font-bold text-white">{{ v.trade_name }}</td>
              <td class="py-3 px-4 font-mono text-emerald-400">{{ formatBRN(v.business_number) }}</td>
              <td class="py-3 px-4 text-slate-300">{{ v.representative }}</td>
              <td class="py-3 px-4 text-slate-400 text-xs">{{ v.category }}</td>
              <td class="py-3 px-4 text-slate-400 font-mono text-xs">{{ v.phone }}</td>
              <td class="py-3 px-4 text-slate-500 text-xs">{{ v.memo || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Vendor Registration Modal -->
    <div v-if="showVendorModal" class="fixed inset-0 bg-black/70 flex items-center justify-center p-4 z-50">
      <div class="bg-slate-800 border border-slate-700 rounded-xl p-6 w-full max-w-md space-y-4">
        <div class="flex items-center justify-between border-b border-slate-700 pb-3">
          <h3 class="text-lg font-bold text-white">신규 거래처 등록</h3>
          <button @click="showVendorModal = false" class="text-slate-400 hover:text-white">&times;</button>
        </div>

        <div class="space-y-3 text-sm">
          <div>
            <label class="block text-slate-400 mb-1">상호명 (필수)</label>
            <input v-model="newVendor.trade_name" placeholder="(주)식자재유통" class="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-white" />
          </div>
          <div>
            <label class="block text-slate-400 mb-1">사업자등록번호 (10자리)</label>
            <input v-model="newVendor.business_number" placeholder="123-45-67890" class="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-white font-mono" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-slate-400 mb-1">대표자명</label>
              <input v-model="newVendor.representative" placeholder="홍길동" class="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-white" />
            </div>
            <div>
              <label class="block text-slate-400 mb-1">업종/분류</label>
              <input v-model="newVendor.category" placeholder="식자재도매" class="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-white" />
            </div>
          </div>
          <div>
            <label class="block text-slate-400 mb-1">연락처</label>
            <input v-model="newVendor.phone" placeholder="010-1234-5678" class="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-white" />
          </div>
        </div>

        <div class="flex justify-end space-x-2 pt-3 border-t border-slate-700">
          <button @click="showVendorModal = false" class="px-4 py-2 bg-slate-700 text-white rounded text-sm">취소</button>
          <button @click="saveVendor" class="px-4 py-2 bg-emerald-600 text-white font-medium rounded text-sm">등록</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';

const lastUpdatedAt = new Date().toISOString();
const activeTab = ref('expenses');
const showVendorModal = ref(false);
const showExpenseModal = ref(false);

const summary = ref({
  total: '4950000',
  supply: '4500000',
  vat: '450000',
  fixed: '4000000',
  fixedRate: '80.8',
  variable: '950000',
  variableRate: '19.2'
});

const expenses = ref([
  { id: '1', expense_date: '2026-02-01', category_label: '임차료', is_fixed: true, vendor_name: '강남부동산빌딩', proof_type_label: '전자세금계산서', payment_method: '은행이체', supply_amount: '1500000', vat_amount: '150000', total_amount: '1650000' },
  { id: '2', expense_date: '2026-02-01', category_label: '인건비', is_fixed: true, vendor_name: '주방 매니저', proof_type_label: '급여이체', payment_method: '은행이체', supply_amount: '2500000', vat_amount: '0', total_amount: '2500000' },
  { id: '3', expense_date: '2026-02-02', category_label: '식자재', is_fixed: false, vendor_name: '(주)에스프레소원두', proof_type_label: '사업용카드', payment_method: '신용카드', supply_amount: '800000', vat_amount: '80000', total_amount: '880000' }
]);

const vendors = ref([
  { id: 'v1', trade_name: '(주)에스프레소원두', business_number: '1234567890', representative: '이원두', category: '식자재/원두', phone: '02-123-4567', memo: '주2회 정기 납품' },
  { id: 'v2', trade_name: '강남부동산빌딩', business_number: '2345678901', representative: '김건물', category: '부동산임대', phone: '02-987-6543', memo: '매월 말일 임대료 청구' }
]);

const newVendor = ref({
  trade_name: '',
  business_number: '',
  representative: '',
  category: '',
  phone: ''
});

function formatNumber(v: string | number) {
  return Number(v || 0).toLocaleString();
}

function formatDate(d: string) {
  return d ? d.substring(0, 10) : '';
}

function formatBRN(brn: string) {
  if (!brn || brn.length !== 10) return brn;
  return `${brn.substring(0, 3)}-${brn.substring(3, 5)}-${brn.substring(5)}`;
}

function saveVendor() {
  if (!newVendor.value.trade_name) {
    alert('상호명을 입력하세요.');
    return;
  }
  vendors.value.push({
    id: `v${vendors.value.length + 1}`,
    trade_name: newVendor.value.trade_name,
    business_number: newVendor.value.business_number.replace(/[^0-9]/g, ''),
    representative: newVendor.value.representative,
    category: newVendor.value.category,
    phone: newVendor.value.phone,
    memo: ''
  });
  showVendorModal.value = false;
  newVendor.value = { trade_name: '', business_number: '', representative: '', category: '', phone: '' };
}
</script>
