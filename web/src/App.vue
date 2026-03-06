<template>
  <div class="min-h-screen bg-background text-text font-mono">
    <header v-if="$route.name !== 'Login'" class="border-b border-border-primary bg-surface shadow-[0_4px_20px_rgba(57,_255,_20,_0.05)] relative z-10">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-16">
          <div class="flex items-center gap-4">
            <svg class="w-8 h-8 text-primary drop-shadow-[0_0_8px_rgba(57,_255,_20,_0.8)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"></path>
            </svg>
            <h1 class="text-xl font-bold tracking-wider text-text-dark drop-shadow-[0_0_2px_rgba(255,_255,_255,_0.5)]">TRIN<span class="text-primary glow-text">DEX</span></h1>
          </div>
          
          <div class="flex items-center gap-4 text-xs">
             <div class="flex items-center gap-2 px-3 py-1 rounded-full bg-[rgba(57,_255,_20,_0.05)] border border-primary text-primary">
                <span class="w-2 h-2 rounded-full bg-primary shadow-[0_0_5px_#39ff14]"></span>
                System Active
             </div>
          </div>
        </div>
      </div>
    </header>

    <div class="flex max-w-7xl mx-auto align-top items-stretch">
      <aside v-if="$route.name !== 'Login'" class="w-64 hidden md:block py-6 px-4 border-r border-[#1a1a3a] min-h-[calc(100vh-4rem)]">
        <nav class="space-y-2">
          <router-link
            v-for="item in navItems"
            :key="item.path"
            :to="item.path"
            :class="['nav-link', { 'active': $route.path === item.path }]"
          >
            <span v-html="item.icon" class="mr-3 opacity-80"></span>
            {{ item.name }}
          </router-link>
        </nav>
      </aside>

      <main class="flex-1 py-8 px-6">
        <router-view></router-view>
      </main>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'

const navItems = [
  {
    name: 'Dashboard',
    path: '/',
    icon: '<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"></path></svg>'
  },
  {
    name: 'Vectors',
    path: '/memories',
    icon: '<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path></svg>'
  },
  {
    name: 'Query',
    path: '/search',
    icon: '<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>'
  },
  {
    name: 'Analytics',
    path: '/stats',
    icon: '<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"></path></svg>'
  },
  {
    name: 'Settings',
    path: '/settings',
    icon: '<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path></svg>'
  }
]

onMounted(() => {
  // Ensure we enforce dark mode class on HTML document for potential tailwind classes outside App
  document.documentElement.classList.add('dark')
})
</script>
