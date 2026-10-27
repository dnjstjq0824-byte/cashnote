<template>
  <div class="min-h-screen bg-slate-50 text-slate-900 font-sans antialiased flex flex-col">
    <!-- Top Global Header -->
    <header class="bg-slate-900 text-white border-b border-slate-800 sticky top-0 z-50">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-xl bg-emerald-500 flex items-center justify-center font-bold text-slate-950 text-lg shadow-sm">
            BN
          </div>
          <div>
            <h1 class="text-base font-bold tracking-tight">BN Merchant OS</h1>
            <p class="text-xs text-slate-400">사장님 경영관리 플랫폼</p>
          </div>
        </div>

        <div class="flex items-center gap-4">
          <div class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-emerald-950/80 border border-emerald-800 text-emerald-300 text-xs font-medium">
            <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
            <span>Core API 연결됨</span>
          </div>
        </div>
      </div>
    </header>

    <!-- Main Content Area -->
    <main class="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <NuxtPage />
    </main>

    <!-- Footer -->
    <footer class="bg-white border-t border-slate-200 py-6 text-center text-xs text-slate-500">
      <p>© 2026 Booking Network — BN Merchant OS. 독립형 상용 SaaS 플랫폼 (제1조 독립 원칙 준수)</p>
    </footer>
  </div>
</template>
