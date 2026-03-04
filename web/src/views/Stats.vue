<template>
  <div class="space-y-6">
    <h2 class="text-2xl font-bold text-text">Statistics</h2>

    <div v-if="loading" class="text-center py-12">
      Loading statistics...
    </div>

    <div v-else-if="stats" class="space-y-6">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div class="card">
          <p class="text-sm text-text-muted mb-1">Total Memories</p>
          <p class="text-3xl font-bold">{{ formatNumber(stats.total_memories) }}</p>
        </div>

        <div class="card">
          <p class="text-sm text-text-muted mb-1">Last 24 Hours</p>
          <p class="text-3xl font-bold text-success">{{ formatNumber(stats.recent_24h) }}</p>
        </div>

        <div class="card">
          <p class="text-sm text-text-muted mb-1">Namespaces</p>
          <p class="text-3xl font-bold text-primary">{{ Object.keys(stats.by_namespace).length }}</p>
        </div>

        <div class="card">
          <p class="text-sm text-text-muted mb-1">Top Tags</p>
          <p class="text-3xl font-bold text-purple-600">{{ stats.top_tags?.length || 0 }}</p>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div class="card">
          <h3 class="text-lg font-semibold mb-6">Namespace Distribution</h3>
          
          <div v-if="namespaceChartData.length === 0" class="text-center py-8 text-text-muted">
            No data available
          </div>
          
          <div v-else class="space-y-4">
            <div
              v-for="item in namespaceChartData"
              :key="item.name"
              class="space-y-2"
            >
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium">{{ item.name }}</span>
                <span class="text-sm text-text-muted">
                  {{ item.count }} ({{ item.percentage.toFixed(1) }}%)
                </span>
              </div>
              
              <div class="h-2 bg-gray-200 rounded-full overflow-hidden">
                <div
                  class="h-full bg-primary rounded-full transition-all duration-500"
                  :style="{ width: item.percentage + '%' }"
                ></div>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <h3 class="text-lg font-semibold mb-6">System Information</h3>
          
          <div class="space-y-4">
            <div class="flex justify-between py-3 border-b border-border">
              <span class="text-text-muted">Embedding Model</span>
              <span class="font-medium">{{ stats.embedding_model }}</span>
            </div>
            
            <div class="flex justify-between py-3 border-b border-border">
              <span class="text-text-muted">Dimensions</span>
              <span class="font-medium">{{ stats.embed_dimensions }}</span>
            </div>
            
            <div class="flex justify-between py-3 border-b border-border">
              <span class="text-text-muted">Oldest Memory</span>
              <span class="font-medium">{{ stats.oldest_memory ? formatDate(stats.oldest_memory) : 'N/A' }}</span>
            </div>
            
            <div class="flex justify-between py-3">
              <span class="text-text-muted">Newest Memory</span>
              <span class="font-medium">{{ stats.newest_memory ? formatDate(stats.newest_memory) : 'N/A' }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="card">
        <h3 class="text-lg font-semibold mb-6">Top Tags</h3>
        
        <div v-if="!stats.top_tags?.length" class="text-center py-8 text-text-muted">
          No tags found
        </div>
        
        <div v-else class="flex flex-wrap gap-3">
          <span
            v-for="(tag, index) in stats.top_tags"
            :key="tag"
            class="px-4 py-2 rounded-full text-sm font-medium transition-colors"
            :class="getTagClass(index)"
          >
            {{ tag }}
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

function getTagClass(index) {
  const classes = [
    'bg-blue-100 text-blue-700',
    'bg-green-100 text-green-700',
    'bg-yellow-100 text-yellow-700',
    'bg-purple-100 text-purple-700',
    'bg-pink-100 text-pink-700',
    'bg-indigo-100 text-indigo-700',
    'bg-red-100 text-red-700',
    'bg-orange-100 text-orange-700',
    'bg-teal-100 text-teal-700',
    'bg-cyan-100 text-cyan-700'
  ]
  return classes[index % classes.length]
}
</script>
