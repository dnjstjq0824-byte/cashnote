<template>
  <div style="min-height: 100vh; background-color: #0F172A; color: #F8FAFC; font-family: sans-serif; padding: 2rem;">
    <header style="border-bottom: 1px solid #1E293B; padding-bottom: 1.5rem; margin-bottom: 2rem; display: flex; justify-content: space-between; align-items: center;">
      <div>
        <h1 style="font-size: 1.5rem; font-weight: bold; margin: 0;">BN Merchant OS 관리자·파트너 콘솔</h1>
        <p style="font-size: 0.875rem; color: #94A3B8; margin-top: 0.25rem;">전사 가맹점, 4-Eyes 승인, 감사 로그 해시체인, Break-Glass 열람 통제</p>
      </div>
      <div style="background: rgba(5, 150, 105, 0.2); border: 1px solid #059669; color: #6EE7B7; padding: 0.25rem 0.75rem; border-radius: 9999px; font-size: 0.75rem; font-weight: 600;">
        4-EYES SECURE MODE ACTIVE
      </div>
    </header>

    <main style="display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 1.5rem;">
      <div style="background: #1E293B; border: 1px solid #334155; border-radius: 1rem; padding: 1.5rem;">
        <h3 style="font-size: 1.125rem; font-weight: 600; margin-top: 0;">사업자 소유권 심사 대기</h3>
        <p style="font-size: 2rem; font-weight: bold; color: #38BDF8; margin: 0.5rem 0;">0건</p>
        <p style="font-size: 0.75rem; color: #94A3B8;">국세청 진위확인 및 1원 계좌인증 정상 연동 중</p>
      </div>

      <div style="background: #1E293B; border: 1px solid #334155; border-radius: 1rem; padding: 1.5rem;">
        <h3 style="font-size: 1.125rem; font-weight: 600; margin-top: 0;">입금 불일치 대사 경보</h3>
        <p style="font-size: 2rem; font-weight: bold; color: #34D399; margin: 0.5rem 0;">0건</p>
        <p style="font-size: 0.75rem; color: #94A3B8;">불변 복식원장 차변==대변 100% 일치</p>
      </div>

      <div style="background: #1E293B; border: 1px solid #334155; border-radius: 1rem; padding: 1.5rem;">
        <h3 style="font-size: 1.125rem; font-weight: 600; margin-top: 0;">외부 API 규제 게이트</h3>
        <p style="font-size: 2rem; font-weight: bold; color: #FBBF24; margin: 0.5rem 0;">5개 활성</p>
        <p style="font-size: 0.75rem; color: #94A3B8;">배달3사 비공식 스크래핑 전면 차단 (`BLOCKED`)</p>
      </div>
    </main>
  </div>
</template>
