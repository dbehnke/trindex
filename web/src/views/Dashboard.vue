<template>
  <div class="space-y-6">
    <h2 class="text-2xl font-bold text-text">Dashboard</h2>

    <div v-if="loading" class="text-center py-12">
      Loading...
    </div>

    <template v-else-if="stats">
      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div class="card">
          <div class="flex items-center gap-4">
            <div class="p-3 rounded-lg bg-blue-100 text-primary">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
              </svg>
            </div>
            <div>
              <p class="text-sm text-text-muted">Total Memories</p>
              <p class="text-2xl font-bold">{{ formatNumber(stats.total_memories) }}</p>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="flex items-center gap-4">
            <div class="p-3 rounded-lg bg-green-100 text-success">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
            </div>
            <div>
              <p class="text-sm text-text-muted">Last 24 Hours</p>
              <p class="text-2xl font-bold">{{ formatNumber(stats.recent_24h) }}</p>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="flex items-center gap-4">
            <div class="p-3 rounded-lg bg-purple-100 text-purple-600">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"></path>
              </svg>
            </div>
            <div>
              <p class="text-sm text-text-muted">Namespaces</p>
              <p class="text-2xl font-bold">{{ Object.keys(stats.by_namespace).length }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div class="card">
          <h3 class="text-lg font-semibold mb-4">Memories by Namespace</h3>
          <div class="space-y-3">
            <div
              v-for="(count, namespace) in stats.by_namespace"
              :key="namespace"
              class="flex items-center justify-between"
            >
              <span class="text-sm font-medium">{{ namespace }}</span>
              <div class="flex items-center gap-3">
                <div class="w-32 h-2 bg-gray-200 rounded-full overflow-hidden">
                  <div
                    class="h-full bg-primary rounded-full"
                    :style="{ width: getNamespacePercentage(count) + '%' }"
                  ></div>
                </div>
                <span class="text-sm text-text-muted w-12 text-right">{{ count }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <h3 class="text-lg font-semibold mb-4">Top Tags</h3>
          <div class="flex flex-wrap gap-2">
            <span
              v-for="tag in stats.top_tags"
              :key="tag"
              class="px-3 py-1 rounded-full text-sm bg-blue-100 text-primary"
            >
              {{ tag }}
            </span>
            <p v-if="!stats.top_tags?.length" class="text-text-muted">No tags found</p>
          </div>

          <div class="mt-6 pt-6 border-t border-border">
            <h4 class="text-sm font-medium text-text-muted mb-2">Embedding Model</h4>
            <p class="text-sm">{{ stats.embedding_model }}</p>
            <p class="text-xs text-text-muted">{{ stats.embed_dimensions }} dimensions</p>
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
