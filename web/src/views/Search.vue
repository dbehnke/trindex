<template>
  <div class="space-y-6">
    <h2 class="text-2xl font-bold text-text">Semantic Search</h2>

    <div class="card">
      <form @submit.prevent="performSearch" class="space-y-4">
        <div class="flex gap-4">
          <input
            v-model="query"
            type="text"
            class="input flex-1"
            placeholder="Enter search query..."
            required
          >
          <button type="submit" class="btn-primary" :disabled="loading">
            <svg v-if="!loading" class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
            </svg>
            <span v-else class="mr-2">...</span>
            {{ loading ? 'Searching...' : 'Search' }}
          </button>
        </div>

        <div class="flex gap-4">
          <select v-model="namespaceFilter" class="input w-48">
            <option value="">All Namespaces</option>
            <option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</option>
          </select>

          <select v-model="topK" class="input w-32">
            <option :value="5">Top 5</option>
            <option :value="10">Top 10</option>
            <option :value="20">Top 20</option>
            <option :value="50">Top 50</option>
          </select>
        </div>
      </form>
    </div>

    <div v-if="error" class="p-4 rounded-lg bg-red-50 text-error">
      {{ error }}
    </div>

    <div v-if="hasSearched && !loading">
      <p class="text-sm text-text-muted mb-4">
        Found {{ results.length }} results
        <span v-if="namespacesSearched.length"> in namespaces: {{ namespacesSearched.join(', ') }}</span>
      </p>

      <div v-if="results.length === 0" class="card text-center py-12">
        <svg class="w-16 h-16 mx-auto text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
        </svg>
        <p class="text-text-muted">No results found</p>
      </div>

      <div v-else class="space-y-4">
        <div
          v-for="result in results"
          :key="result.id"
          class="card"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1">
              <p class="text-text whitespace-pre-wrap">{{ result.content }}</p>
              
              <div class="flex items-center gap-4 mt-4">
                <span class="px-2 py-1 rounded text-xs bg-blue-100 text-primary">
                  {{ result.namespace }}
                </span>
                <span class="text-xs text-text-muted">
                  {{ formatDate(result.created_at) }}
                </span>
              </div>
            </div>

            <div class="flex flex-col items-end gap-2">
              <div
                class="px-3 py-1 rounded-full text-sm font-medium"
                :class="getScoreClass(result.score)"
              >
                {{ (result.score * 100).toFixed(1) }}%
              </div>
              <button
                @click="deleteResult(result.id)"
                class="p-1 text-gray-400 hover:text-error transition-colors"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useMemoryStore } from '../stores/memory'

const store = useMemoryStore()

const query = ref('')
const namespaceFilter = ref('')
const topK = ref(10)
const hasSearched = ref(false)

const results = computed(() => store.searchResults)
const loading = computed(() => store.loading)
const error = computed(() => store.error)
const namespaces = computed(() => store.stats ? Object.keys(store.stats.by_namespace) : [])
const namespacesSearched = ref([])

onMounted(() => {
  store.fetchStats()
})

async function performSearch() {
  hasSearched.value = true
  const namespaces = namespaceFilter.value ? [namespaceFilter.value] : []
  const data = await store.search(query.value, namespaces, topK.value)
  namespacesSearched.value = data.namespaces_searched || []
}

async function deleteResult(id) {
  if (!confirm('Are you sure you want to delete this memory?')) return
  await store.deleteMemory(id)
  results.value = results.value.filter(r => r.id !== id)
}

function getScoreClass(score) {
  if (score >= 0.8) return 'bg-green-100 text-green-700'
  if (score >= 0.6) return 'bg-yellow-100 text-yellow-700'
  return 'bg-gray-100 text-gray-700'
}

function formatDate(dateString) {
  return new Date(dateString).toLocaleString()
}
</script>
