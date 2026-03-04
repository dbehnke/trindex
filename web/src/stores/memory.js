import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

const API_BASE = '/api'

function getHeaders() {
  const apiKey = localStorage.getItem('trindex_api_key') || ''
  return {
    'Content-Type': 'application/json',
    'X-API-Key': apiKey
  }
}

export const useMemoryStore = defineStore('memory', () => {
  const memories = ref([])
  const searchResults = ref([])
  const stats = ref(null)
  const loading = ref(false)
  const error = ref(null)

  const totalMemories = computed(() => stats.value?.total_memories || 0)
  const byNamespace = computed(() => stats.value?.by_namespace || {})

  async function fetchMemories(params = {}) {
    loading.value = true
    error.value = null
    try {
      const queryParams = new URLSearchParams()
      if (params.namespace) queryParams.append('namespace', params.namespace)
      if (params.limit) queryParams.append('limit', params.limit)
      if (params.offset) queryParams.append('offset', params.offset)
      if (params.order) queryParams.append('order', params.order)

      const response = await fetch(`${API_BASE}/memories/?${queryParams}`, {
        headers: getHeaders()
      })
      if (!response.ok) throw new Error(await response.text())
      memories.value = await response.json()
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function fetchMemory(id) {
    const response = await fetch(`${API_BASE}/memories/${id}`, {
      headers: getHeaders()
    })
    if (!response.ok) throw new Error(await response.text())
    return response.json()
  }

  async function createMemory(data) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${API_BASE}/memories/`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify(data)
      })
      if (!response.ok) throw new Error(await response.text())
      const memory = await response.json()
      memories.value.unshift(memory)
      return memory
    } catch (e) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteMemory(id) {
    const response = await fetch(`${API_BASE}/memories/${id}`, {
      method: 'DELETE',
      headers: getHeaders()
    })
    if (!response.ok) throw new Error(await response.text())
    memories.value = memories.value.filter(m => m.id !== id)
  }

  async function search(query, namespaces = [], topK = 10) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${API_BASE}/search`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({
          query,
          namespaces,
          top_k: topK,
          threshold: 0.7
        })
      })
      if (!response.ok) throw new Error(await response.text())
      const data = await response.json()
      searchResults.value = data.results
      return data
    } catch (e) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchStats(namespace = '') {
    try {
      const queryParams = namespace ? `?namespace=${namespace}` : ''
      const response = await fetch(`${API_BASE}/stats${queryParams}`, {
        headers: getHeaders()
      })
      if (!response.ok) throw new Error(await response.text())
      stats.value = await response.json()
    } catch (e) {
      error.value = e.message
    }
  }

  return {
    memories,
    searchResults,
    stats,
    loading,
    error,
    totalMemories,
    byNamespace,
    fetchMemories,
    fetchMemory,
    createMemory,
    deleteMemory,
    search,
    fetchStats
  }
})
