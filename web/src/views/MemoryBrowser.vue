<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-2xl font-bold text-text">Memory Browser</h2>
      <button @click="showCreateModal = true" class="btn-primary">
        <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path>
        </svg>
        Add Memory
      </button>
    </div>

    <div class="flex gap-4">
      <select v-model="filterNamespace" class="input w-48">
        <option value="">All Namespaces</option>
        <option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</option>
      </select>
      
      <select v-model="sortOrder" class="input w-32">
        <option value="desc">Newest First</option>
        <option value="asc">Oldest First</option>
      </select>
    </div>

    <div v-if="loading" class="text-center py-12">
      Loading memories...
    </div>

    <div v-else-if="memories.length === 0" class="card text-center py-12">
      <svg class="w-16 h-16 mx-auto text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"></path>
      </svg>
      <p class="text-text-muted">No memories found</p>
    </div>

    <div v-else class="space-y-4">
      <div
        v-for="memory in memories"
        :key="memory.id"
        class="card hover:shadow-md transition-shadow"
      >
        <div class="flex items-start justify-between">
          <div class="flex-1">
            <p class="text-text whitespace-pre-wrap">{{ memory.content }}</p>
            <div class="flex items-center gap-4 mt-4">
              <span class="px-2 py-1 rounded text-xs bg-blue-100 text-primary">
                {{ memory.namespace }}
              </span>
              <span class="text-xs text-text-muted">
                {{ formatDate(memory.created_at) }}
              </span>
            </div>
          </div>
          
          <button
            @click="deleteMemory(memory.id)"
            class="p-2 text-gray-400 hover:text-error transition-colors"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
            </svg>
          </button>
        </div>
      </div>

      <div class="flex items-center justify-between pt-4">
        <button
          @click="page--"
          :disabled="page === 0"
          class="btn-secondary"
          :class="{ 'opacity-50 cursor-not-allowed': page === 0 }"
        >
          Previous
        </button>
        <span class="text-sm text-text-muted">Page {{ page + 1 }}</span>
        <button
          @click="page++"
          :disabled="memories.length < limit"
          class="btn-secondary"
          :class="{ 'opacity-50 cursor-not-allowed': memories.length < limit }"
        >
          Next
        </button>
      </div>
    </div>

    <div v-if="showCreateModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="card w-full max-w-2xl mx-4">
        <div class="flex items-center justify-between mb-6">
          <h3 class="text-xl font-semibold">Add New Memory</h3>
          <button @click="showCreateModal = false" class="text-gray-400 hover:text-text">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>

        <form @submit.prevent="createMemory" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-1">Content</label>
            <textarea
              v-model="newMemory.content"
              class="textarea"
              rows="4"
              placeholder="Enter memory content..."
              required
            ></textarea>
          </div>

          <div>
            <label class="block text-sm font-medium mb-1">Namespace</label>
            <input
              v-model="newMemory.namespace"
              type="text"
              class="input"
              placeholder="default"
            >
          </div>

          <div v-if="error" class="p-4 rounded-lg bg-red-50 text-error text-sm">
            {{ error }}
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <button type="button" @click="showCreateModal = false" class="btn-secondary">
              Cancel
            </button>
            <button type="submit" class="btn-primary" :disabled="creating">
              {{ creating ? 'Creating...' : 'Create Memory' }}
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
