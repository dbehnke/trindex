<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between border-b border-border pb-4 mb-6">
      <div class="flex items-center gap-3">
        <div class="w-1.5 h-6 bg-secondary shadow-[0_0_8px_#bf00ff]"></div>
        <h2 class="text-2xl font-bold text-text-dark tracking-wider">Vector Explorer</h2>
      </div>
      <button @click="showCreateModal = true" class="btn-primary">
        <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
        </svg>
        Insert Vector
      </button>
    </div>

    <div class="flex gap-4 p-4 card bg-[#030308] border-border-dark mb-6">
      <div class="flex-1 max-w-xs">
        <label class="block text-xs uppercase tracking-widest text-text-muted mb-2">Namespace Filter</label>
        <select v-model="filterNamespace" class="input font-mono text-sm">
          <option value="">[*] ALL_NAMESPACES</option>
          <option v-for="ns in namespaces" :key="ns" :value="ns">[{{ ns }}]</option>
        </select>
      </div>
      
      <div class="w-48">
        <label class="block text-xs uppercase tracking-widest text-text-muted mb-2">Sort Order</label>
        <select v-model="sortOrder" class="input font-mono text-sm">
          <option value="desc">DESC (Newest)</option>
          <option value="asc">ASC (Oldest)</option>
        </select>
      </div>
    </div>

    <div v-if="loading" class="text-center py-12 font-mono text-primary animate-pulse">
      [System] >> Retrieving memory vectors...
    </div>

    <div v-else-if="memories.length === 0" class="card text-center py-16 border-dashed border-border-dark">
      <svg class="w-16 h-16 mx-auto text-text-muted mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"></path>
      </svg>
      <p class="text-text-muted font-mono tracking-widest">[ ERR_NO_VECTORS_FOUND ]</p>
    </div>

    <div v-else class="space-y-4">
      <div
        v-for="memory in memories"
        :key="memory.id"
        class="card bg-[#050510] hover:bg-[#080814] relative overflow-hidden group border-border-dark hover:border-primary/50 transition-all duration-300"
      >
        <!-- Neon side strip -->
        <div class="absolute left-0 top-0 bottom-0 w-1 bg-gradient-to-b from-primary/80 to-secondary/80 shadow-[0_0_10px_rgba(57,_255,_20,_0.5)]"></div>

        <div class="flex items-start justify-between pl-4">
          <div class="flex-1">
            <div class="flex items-center gap-3 mb-3">
              <span class="text-xs font-mono text-text-muted">ID:</span>
              <span class="text-xs font-mono tracking-wider text-text-dark">{{ memory.id }}</span>
              <span class="px-2 py-0.5 rounded text-[10px] font-mono uppercase tracking-widest bg-[rgba(191,_0,_255,_0.15)] text-secondary border border-secondary/30 ml-2">
                {{ memory.namespace }}
              </span>
            </div>
            
            <p class="text-text-dark text-sm font-mono leading-relaxed whitespace-pre-wrap bg-[#020205] p-4 rounded-md border border-border/50">
              {{ memory.content }}
            </p>
            
            <div class="flex items-center gap-4 mt-4">
              <div class="flex items-center gap-1.5 text-xs text-text-muted font-mono">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                </svg>
                Timestamp: <span class="text-primary/80">{{ formatDate(memory.created_at) }}</span>
              </div>
            </div>
          </div>
          
          <button
            @click="deleteMemory(memory.id)"
            class="p-2 ml-4 text-text-muted hover:text-error hover:bg-[rgba(255,_0,_60,_0.1)] rounded transition-colors group/btn"
            title="Delete Vector"
          >
            <svg class="w-5 h-5 drop-shadow-[0_0_2px_rgba(255,_0,_60,_0)] group-hover/btn:drop-shadow-[0_0_5px_#ff003c]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
            </svg>
          </button>
        </div>
      </div>

      <div class="flex items-center justify-between pt-6 border-t border-border-dark mt-8">
        <button
          @click="page--"
          :disabled="page === 0"
          class="btn-secondary"
          :class="{ 'opacity-30 cursor-not-allowed hover:shadow-none hover:bg-transparent': page === 0 }"
        >
          &lt; Prev_Page
        </button>
        <span class="text-xs font-mono tracking-widest text-text-muted uppercase">Page [{{ page + 1 }}]</span>
        <button
          @click="page++"
          :disabled="memories.length < limit"
          class="btn-secondary"
          :class="{ 'opacity-30 cursor-not-allowed hover:shadow-none hover:bg-transparent': memories.length < limit }"
        >
          Next_Page &gt;
        </button>
      </div>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50">
      <div class="card bg-[#050510] border border-primary/50 shadow-[0_0_30px_rgba(57,_255,_20,_0.1)] w-full max-w-2xl mx-4">
        <div class="flex items-center justify-between border-b border-border pb-4 mb-6">
          <h3 class="text-lg font-mono font-bold text-primary tracking-widest uppercase">>>> Initialize_New_Vector</h3>
          <button @click="showCreateModal = false" class="text-text-muted hover:text-primary transition-colors">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>

        <form @submit.prevent="createMemory" class="space-y-6">
          <div>
            <label class="block text-xs uppercase font-mono tracking-widest text-text-muted mb-2">Payload Data (Content)</label>
            <textarea
              v-model="newMemory.content"
              class="textarea font-mono h-32 text-text-dark"
              placeholder="Enter context string here..."
              required
            ></textarea>
          </div>

          <div>
            <label class="block text-xs uppercase font-mono tracking-widest text-text-muted mb-2">Target Namespace</label>
            <input
              v-model="newMemory.namespace"
              type="text"
              class="input font-mono text-text-dark"
              placeholder="default"
            >
          </div>

          <div v-if="error" class="p-4 border border-error/50 rounded-md bg-[rgba(255,_0,_60,_0.1)] text-error font-mono text-sm shadow-[0_0_10px_rgba(255,_0,_60,_0.2)]">
            [ERR] {{ error }}
          </div>

          <div class="flex justify-end gap-4 pt-4 border-t border-border mt-6">
            <button type="button" @click="showCreateModal = false" class="btn-secondary">
              Abort
            </button>
            <button type="submit" class="btn-primary" :disabled="creating">
              {{ creating ? 'Executing...' : 'Execute Input' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useMemoryStore } from '../stores/memory'

const store = useMemoryStore()

const filterNamespace = ref('')
const sortOrder = ref('desc')
const page = ref(0)
const limit = 20
const showCreateModal = ref(false)
const creating = ref(false)
const newMemory = ref({ content: '', namespace: '' })

const memories = computed(() => store.memories)
const loading = computed(() => store.loading)
const error = computed(() => store.error)
const namespaces = computed(() => store.stats ? Object.keys(store.stats.by_namespace) : [])

onMounted(() => {
  store.fetchStats()
  loadMemories()
})

watch([filterNamespace, sortOrder, page], () => {
  loadMemories()
})

async function loadMemories() {
  await store.fetchMemories({
    namespace: filterNamespace.value,
    limit,
    offset: page.value * limit,
    order: sortOrder.value
  })
}

async function createMemory() {
  creating.value = true
  try {
    await store.createMemory({
      content: newMemory.value.content,
      namespace: newMemory.value.namespace || 'default'
    })
    showCreateModal.value = false
    newMemory.value = { content: '', namespace: '' }
    loadMemories()
  } finally {
    creating.value = false
  }
}

async function deleteMemory(id) {
  if (!confirm('Are you sure you want to delete this memory?')) return
  await store.deleteMemory(id)
}

function formatDate(dateString) {
  return new Date(dateString).toLocaleString()
}
</script>
