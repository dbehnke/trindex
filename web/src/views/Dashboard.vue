<template>
  <div class="space-y-6">
    <div class="flex items-center gap-3 mb-8 border-b border-border pb-4">
      <div class="w-1.5 h-6 bg-primary shadow-[0_0_8px_#39ff14]"></div>
      <h2 class="text-2xl font-bold text-text-dark tracking-wider">Dashboard</h2>
    </div>

    <div v-if="loading" class="text-center py-12 text-text-muted font-mono animate-pulse">
      > Loading system status...
    </div>

    <template v-else-if="stats">
      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div class="card relative overflow-hidden group">
          <div class="absolute top-0 right-0 w-16 h-16 bg-primary opacity-5 rounded-bl-full group-hover:opacity-10 transition-opacity"></div>
          <div class="flex items-center gap-4 relative z-10">
            <div class="p-3 rounded-lg bg-[rgba(57,255,20,0.1)] text-primary ring-1 ring-primary/30">
              <svg class="w-6 h-6 drop-shadow-[0_0_5px_#39ff14]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
              </svg>
            </div>
            <div>
              <p class="text-xs uppercase tracking-wider text-text-muted mb-1">Total Memories</p>
              <p class="text-3xl font-bold text-text-dark">{{ formatNumber(stats.total_memories) }}</p>
            </div>
          </div>
        </div>

        <div class="card relative overflow-hidden group">
          <div class="absolute top-0 right-0 w-16 h-16 bg-primary opacity-5 rounded-bl-full group-hover:opacity-10 transition-opacity"></div>
          <div class="flex items-center gap-4 relative z-10">
            <div class="p-3 rounded-lg bg-[rgba(57,255,20,0.1)] text-primary ring-1 ring-primary/30">
              <svg class="w-6 h-6 drop-shadow-[0_0_5px_#39ff14]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
            </div>
            <div>
              <p class="text-xs uppercase tracking-wider text-text-muted mb-1">Last 24 Hours</p>
              <p class="text-3xl font-bold text-text-dark">+{{ formatNumber(stats.recent_24h) }}</p>
            </div>
          </div>
        </div>

        <div class="card relative overflow-hidden group">
          <div class="absolute top-0 right-0 w-16 h-16 bg-secondary opacity-5 rounded-bl-full group-hover:opacity-10 transition-opacity"></div>
          <div class="flex items-center gap-4 relative z-10">
            <div class="p-3 rounded-lg bg-[rgba(191,0,255,0.1)] text-secondary ring-1 ring-secondary/30">
              <svg class="w-6 h-6 drop-shadow-[0_0_5px_#bf00ff]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"></path>
              </svg>
            </div>
            <div>
              <p class="text-xs uppercase tracking-wider text-text-muted mb-1">Namespaces</p>
              <p class="text-3xl font-bold text-text-dark">{{ Object.keys(stats.by_namespace).length }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
        <div class="card">
          <h3 class="text-sm uppercase tracking-widest text-text-muted border-b border-border pb-3 mb-5">Memories by Namespace</h3>
          <div class="space-y-4">
            <div
              v-for="(count, namespace) in stats.by_namespace"
              :key="namespace"
              class="flex items-center justify-between group"
            >
              <span class="text-sm font-medium text-text-dark w-1/3 truncate">{{ namespace }}</span>
              <div class="flex items-center gap-4 flex-1">
                <div class="flex-1 h-1.5 bg-[#1a1a3a] rounded-full overflow-hidden">
                  <div
                    class="h-full bg-primary rounded-full shadow-[0_0_8px_#39ff14] transition-all duration-1000"
                    :style="{ width: getNamespacePercentage(count) + '%' }"
                  ></div>
                </div>
                <span class="text-sm text-text-muted font-mono w-16 text-right group-hover:text-primary transition-colors">{{ count }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="card flex flex-col">
          <h3 class="text-sm uppercase tracking-widest text-text-muted border-b border-border pb-3 mb-5">Top Tags</h3>
          <div class="flex flex-wrap gap-2 flex-1 content-start">
            <span
              v-for="tag in stats.top_tags"
              :key="tag"
              class="px-3 py-1 rounded-sm text-xs font-mono bg-[rgba(191,0,255,0.1)] text-secondary border border-secondary/30 hover:bg-secondary hover:text-white hover:shadow-[0_0_10px_#bf00ff] cursor-default transition-all duration-200"
            >
              {{ tag }}
            </span>
            <p v-if="!stats.top_tags?.length" class="text-text-muted text-sm italic">No tags found</p>
          </div>

          <div class="mt-6 pt-5 border-t border-border bg-[#030308] p-4 rounded-md">
            <div class="flex justify-between items-center mb-1">
              <h4 class="text-xs uppercase tracking-wider text-text-muted">Embedding Model</h4>
              <span class="w-2 h-2 rounded-full bg-primary animate-pulse shadow-[0_0_5px_#39ff14]"></span>
            </div>
            <p class="text-sm font-mono text-primary">{{ stats.embedding_model }}</p>
            <p class="text-xs text-text-muted mt-1">> Vector Dimensions: <span class="text-text-dark">{{ stats.embed_dimensions }}</span></p>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { onMounted, computed } from 'vue'
import { useMemoryStore } from '../stores/memory'

const store = useMemoryStore()

const stats = computed(() => store.stats)
const loading = computed(() => store.loading)

onMounted(() => {
  store.fetchStats()
})

function formatNumber(num) {
  return new Intl.NumberFormat().format(num || 0)
}

function getNamespacePercentage(count) {
  const total = stats.value?.total_memories || 1
  return (count / total) * 100
}
</script>
