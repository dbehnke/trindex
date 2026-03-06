<template>
  <div class="space-y-6">
    <div class="flex items-center gap-3 mb-8 border-b border-border pb-4">
      <div class="w-1.5 h-6 bg-primary shadow-[0_0_8px_#39ff14]"></div>
      <h2 class="text-2xl font-bold text-text-dark tracking-wider">System Analytics</h2>
    </div>

    <div v-if="loading" class="text-center py-12 text-text-muted font-mono animate-pulse">
      > Compiling statistics...
    </div>

    <div v-else-if="stats" class="space-y-6">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div class="card relative overflow-hidden group">
          <div class="absolute top-0 right-0 w-12 h-12 bg-primary opacity-5 rounded-bl-full group-hover:opacity-10 transition-opacity"></div>
          <p class="text-xs uppercase tracking-wider text-text-muted mb-1 font-mono">Total Vectors</p>
          <p class="text-3xl font-bold text-text-dark font-mono">{{ formatNumber(stats.total_memories) }}</p>
        </div>

        <div class="card relative overflow-hidden group">
          <div class="absolute top-0 right-0 w-12 h-12 bg-primary opacity-5 rounded-bl-full group-hover:opacity-10 transition-opacity"></div>
          <p class="text-xs uppercase tracking-wider text-text-muted mb-1 font-mono">Velocity (24h)</p>
          <p class="text-3xl font-bold text-primary font-mono drop-shadow-[0_0_5px_rgba(57,_255,_20,_0.5)]">+{{ formatNumber(stats.recent_24h) }}</p>
        </div>

        <div class="card relative overflow-hidden group">
          <div class="absolute top-0 right-0 w-12 h-12 bg-secondary opacity-5 rounded-bl-full group-hover:opacity-10 transition-opacity"></div>
          <p class="text-xs uppercase tracking-wider text-text-muted mb-1 font-mono">Active Namespaces</p>
          <p class="text-3xl font-bold text-secondary font-mono drop-shadow-[0_0_5px_rgba(191,_0,_255,_0.5)]">{{ Object.keys(stats.by_namespace).length }}</p>
        </div>

        <div class="card relative overflow-hidden group">
          <div class="absolute top-0 right-0 w-12 h-12 bg-warning opacity-5 rounded-bl-full group-hover:opacity-10 transition-opacity"></div>
          <p class="text-xs uppercase tracking-wider text-text-muted mb-1 font-mono">Unique Tags</p>
          <p class="text-3xl font-bold text-warning font-mono">{{ stats.top_tags?.length || 0 }}</p>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
        <div class="card">
          <h3 class="text-sm uppercase tracking-widest text-text-muted border-b border-border pb-3 mb-5 font-mono">Volume by Namespace</h3>
          
          <div v-if="namespaceChartData.length === 0" class="text-center py-8 text-text-muted font-mono italic">
            [ No data fragments found ]
          </div>
          
          <div v-else class="space-y-4">
            <div
              v-for="item in namespaceChartData"
              :key="item.name"
              class="space-y-2 group"
            >
              <div class="flex items-center justify-between">
                <span class="text-xs font-mono tracking-wider text-text-dark truncate max-w-[60%]">{{ item.name }}</span>
                <span class="text-xs font-mono text-text-muted group-hover:text-primary transition-colors">
                  {{ item.count }} <span class="text-primary/60">({{ item.percentage.toFixed(1) }}%)</span>
                </span>
              </div>
              
              <div class="h-1 bg-[#1a1a3a] rounded-full overflow-hidden">
                <div
                  class="h-full bg-secondary rounded-full shadow-[0_0_8px_#bf00ff] transition-all duration-1000"
                  :style="{ width: item.percentage + '%' }"
                ></div>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <h3 class="text-sm uppercase tracking-widest text-text-muted border-b border-border pb-3 mb-5 font-mono">Hardware & Core Info</h3>
          
          <div class="space-y-1 font-mono text-sm bg-[#020205] p-4 rounded-md border border-border/50">
            <div class="flex justify-between py-2 border-b border-border-dark/50 hover:bg-[#050510] px-2 -mx-2 rounded transition-colors">
              <span class="text-text-muted">> Model_Core:</span>
              <span class="text-primary tracking-wide">{{ stats.embedding_model }}</span>
            </div>
            
            <div class="flex justify-between py-2 border-b border-border-dark/50 hover:bg-[#050510] px-2 -mx-2 rounded transition-colors">
              <span class="text-text-muted">> Dimensions:</span>
              <span class="text-text-dark font-bold">{{ stats.embed_dimensions }}</span>
            </div>
            
            <div class="flex justify-between py-2 border-b border-border-dark/50 hover:bg-[#050510] px-2 -mx-2 rounded transition-colors">
              <span class="text-text-muted">> Origin_Timestamp:</span>
              <span class="text-text-dark">{{ stats.oldest_memory ? formatDate(stats.oldest_memory) : 'NULL' }}</span>
            </div>
            
            <div class="flex justify-between py-2 hover:bg-[#050510] px-2 -mx-2 rounded transition-colors">
              <span class="text-text-muted">> Latest_Write:</span>
              <span class="text-text-dark">{{ stats.newest_memory ? formatDate(stats.newest_memory) : 'NULL' }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="card">
        <h3 class="text-sm uppercase tracking-widest text-text-muted border-b border-border pb-3 mb-5 font-mono">Global Tag Index</h3>
        
        <div v-if="!stats.top_tags?.length" class="text-center py-8 text-text-muted font-mono italic">
          [ Index Empty ]
        </div>
        
        <div v-else class="flex flex-wrap gap-3">
          <span
            v-for="(tag, index) in stats.top_tags"
            :key="tag"
            class="px-3 py-1.5 rounded-sm text-xs font-mono font-medium border transition-all duration-300 hover:-translate-y-0.5"
            :class="getTagClass(index)"
          >
            #{{ tag }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useMemoryStore } from '../stores/memory'

const store = useMemoryStore()

const stats = computed(() => store.stats)
const loading = computed(() => store.loading)

const namespaceChartData = computed(() => {
  if (!stats.value?.by_namespace) return []
  
  const total = stats.value.total_memories || 1
  return Object.entries(stats.value.by_namespace)
    .map(([name, count]) => ({
      name,
      count,
      percentage: (count / total) * 100
    }))
    .sort((a, b) => b.count - a.count)
})

onMounted(() => {
  store.fetchStats()
})

function formatNumber(num) {
  return new Intl.NumberFormat().format(num || 0)
}

function formatDate(dateString) {
  return new Date(dateString).toLocaleString()
}

// Map tags to devtool neon colors for variety
function getTagClass(index) {
  const classes = [
    'bg-[rgba(57,_255,_20,_0.1)] text-primary border-primary/30 hover:bg-[rgba(57,_255,_20,_0.2)] hover:shadow-[0_0_8px_rgba(57,_255,_20,_0.4)]',
    'bg-[rgba(191,_0,_255,_0.1)] text-secondary border-secondary/30 hover:bg-[rgba(191,_0,_255,_0.2)] hover:shadow-[0_0_8px_rgba(191,_0,_255,_0.4)]',
    'bg-[rgba(255,_179,_0,_0.1)] text-warning border-warning/30 hover:bg-[rgba(255,_179,_0,_0.2)] hover:shadow-[0_0_8px_rgba(255,_179,_0,_0.4)]',
    'bg-[rgba(255,_0,_60,_0.1)] text-error border-error/30 hover:bg-[rgba(255,_0,_60,_0.2)] hover:shadow-[0_0_8px_rgba(255,_0,_60,_0.4)]'
  ]
  return classes[index % classes.length]
}
</script>
