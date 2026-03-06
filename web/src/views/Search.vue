<template>
  <div class="space-y-6">
    <div class="flex items-center gap-3 mb-8 border-b border-border pb-4">
      <div class="w-1.5 h-6 bg-primary shadow-[0_0_8px_#39ff14]"></div>
      <h2 class="text-2xl font-bold text-text-dark tracking-wider">Semantic Querying</h2>
    </div>

    <div class="card bg-[#030308] border-primary/40 shadow-[0_0_15px_rgba(57,_255,_20,_0.05)]">
      <form @submit.prevent="performSearch" class="space-y-6">
        <div class="flex gap-4 items-stretch relative">
          <!-- glowing decorative background for the search bar -->
          <div class="absolute inset-x-0 -inset-y-0 bg-primary/20 blur-xl opacity-30 rounded-lg pointer-events-none"></div>
          
          <div class="relative w-full flex-1">
            <span class="absolute inset-y-0 left-0 flex items-center pl-4 text-primary font-mono select-none">
              > _
            </span>
            <input
              v-model="query"
              type="text"
              class="block w-full bg-[#050510] border-2 border-primary/50 rounded-lg py-4 pl-12 pr-4 text-lg text-text-dark placeholder-text-muted/60 focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary focus:shadow-[0_0_20px_rgba(57,_255,_20,_0.25)] transition-all font-mono"
              placeholder="Enter search query..."
              required
            >
          </div>
          
          <button type="submit" class="btn-primary border-2 px-8 text-lg hover:shadow-[0_0_25px_rgba(57,_255,_20,_0.4)] relative z-10" :disabled="loading">
            <svg v-if="!loading" class="w-6 h-6 mr-3 drop-shadow-[0_0_5px_#39ff14]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
            </svg>
            <span v-else class="mr-3 animate-pulse">|</span>
            {{ loading ? 'Querying...' : 'EXECUTE' }}
          </button>
        </div>

        <div class="flex gap-6 mt-4">
          <div class="flex-1 max-w-xs">
            <label class="block text-xs uppercase tracking-widest text-text-muted mb-2 font-mono">Target Namespace</label>
            <select v-model="namespaceFilter" class="input font-mono text-sm bg-[#050510]">
              <option value="">[*] ALL_NAMESPACES</option>
              <option v-for="ns in namespaces" :key="ns" :value="ns">[{{ ns }}]</option>
            </select>
          </div>

          <div class="w-48">
            <label class="block text-xs uppercase tracking-widest text-text-muted mb-2 font-mono">Limit (Top-K)</label>
            <select v-model="topK" class="input font-mono text-sm bg-[#050510]">
              <option :value="5">TOP_5</option>
              <option :value="10">TOP_10</option>
              <option :value="20">TOP_20</option>
              <option :value="50">TOP_50</option>
            </select>
          </div>
        </div>
      </form>
    </div>

    <div v-if="error" class="p-4 border border-error/50 rounded-md bg-[rgba(255,_0,_60,_0.1)] text-error font-mono text-sm shadow-[0_0_10px_rgba(255,_0,_60,_0.2)]">
      [ERR] {{ error }}
    </div>

    <div v-if="hasSearched && !loading">
      <p class="text-xs uppercase font-mono tracking-widest text-text-muted mb-6 px-1">
        Found <span class="text-text-dark">{{ results.length }}</span> results
        <span v-if="namespacesSearched.length"> in [{{ namespacesSearched.join(', ') }}]</span>
      </p>

      <div v-if="results.length === 0" class="card text-center py-16 border-dashed border-border-dark">
        <svg class="w-16 h-16 mx-auto text-text-muted mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
        </svg>
        <p class="text-text-muted font-mono tracking-widest">[ ERR_NO_MATCHES ]</p>
      </div>

      <div v-else class="space-y-4">
        <div
          v-for="result in results"
          :key="result.id"
          class="card bg-[#050510] hover:bg-[#080814] relative overflow-hidden group border-border-dark hover:border-border transition-all duration-300"
        >
          <!-- Neon side strip based on score -->
          <div 
            class="absolute left-0 top-0 bottom-0 w-1 shadow-[0_0_10px_currentColor] opacity-50"
            :class="getSideStripClass(result.score)"
          ></div>

          <div class="flex items-start justify-between pl-4">
            <div class="flex-1 pr-6">
              <div class="flex items-center gap-3 mb-3">
                <span class="text-xs font-mono text-text-muted">ID:</span>
                <span class="text-xs font-mono tracking-wider text-text-dark">{{ result.id }}</span>
                <span class="px-2 py-0.5 rounded text-[10px] font-mono uppercase tracking-widest bg-[rgba(191,_0,_255,_0.15)] text-secondary border border-secondary/30 ml-2">
                  {{ result.namespace }}
                </span>
              </div>
              
              <p class="text-text-dark text-sm font-mono leading-relaxed whitespace-pre-wrap bg-[#020205] p-4 rounded-md border border-border/50">
                {{ result.content }}
              </p>
              
              <div class="flex items-center gap-4 mt-4">
                <div class="flex items-center gap-1.5 text-xs text-text-muted font-mono">
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                  </svg>
                  Indexed: <span class="text-primary/80">{{ formatDate(result.created_at) }}</span>
                </div>
              </div>
            </div>

            <div class="flex flex-col items-end gap-3 justify-center min-w-[100px] h-full pt-2">
              <div
                class="px-3 py-1.5 rounded bg-[#030308] border shrink-0 text-center shadow-lg font-mono text-lg tracking-wider"
                :class="getScoreBadgeClass(result.score)"
              >
                {{ (result.score * 100).toFixed(1) }}%
              </div>
              <button
                @click="deleteResult(result.id)"
                class="mt-auto p-2 text-text-muted hover:text-error hover:bg-[rgba(255,_0,_60,_0.1)] rounded transition-colors group/btn"
                title="Delete Vector"
              >
                <svg class="w-4 h-4 drop-shadow-[0_0_2px_rgba(255,_0,_60,_0)] group-hover/btn:drop-shadow-[0_0_5px_#ff003c]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
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

function getScoreBadgeClass(score) {
  if (score >= 0.8) return 'border-primary/50 text-primary shadow-[0_0_8px_rgba(57,_255,_20,_0.4)]'
  if (score >= 0.6) return 'border-warning/50 text-warning shadow-[0_0_8px_rgba(255,_179,_0,_0.4)]'
  return 'border-border-dark text-text-muted'
}

function getSideStripClass(score) {
  if (score >= 0.8) return 'bg-primary text-primary'
  if (score >= 0.6) return 'bg-warning text-warning'
  return 'bg-text-muted text-text-muted'
}

function formatDate(dateString) {
  return new Date(dateString).toLocaleString()
}
</script>
