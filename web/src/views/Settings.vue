<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-2xl font-bold text-primary glow-text">API Keys & Settings</h2>
      <button 
        @click="isGeneratingFormOpen = !isGeneratingFormOpen" 
        class="bg-primary/10 hover:bg-primary/20 text-primary border border-primary px-4 py-2 rounded text-sm uppercase tracking-wider font-bold transition-all disabled:opacity-50"
      >
        {{ isGeneratingFormOpen ? 'Cancel' : '+ Generate New Key' }}
      </button>
    </div>

    <!-- Active Device Key -->
    <div class="bg-surface border border-border-primary p-6 rounded-lg shadow-glow-primary">
      <h3 class="text-lg font-semibold text-text-dark mb-4">Active Session</h3>
      <div class="flex items-center gap-4">
        <input 
          :value="activeKey.replace(/./g, '•')" 
          type="text" 
          readonly
          class="flex-1 bg-background border border-border-primary rounded px-4 py-2 text-text font-mono"
        />
        <button @click="logout" class="bg-red-500/10 hover:bg-red-500/20 text-red-500 border border-red-500 px-4 py-2 rounded text-sm uppercase tracking-wider font-bold transition-all hover:shadow-[0_0_15px_rgba(239,68,68,0.5)]">
          Logout
        </button>
      </div>
    </div>

    <!-- Generation Form Container -->
    <div v-if="isGeneratingFormOpen" class="bg-[#050510] border border-primary/50 p-6 rounded-lg shadow-[0_0_30px_rgba(57,_255,_20,_0.1)]">
      <h3 class="text-lg font-mono font-bold text-primary tracking-widest uppercase mb-4">>>> Initialize_New_Key</h3>
      
      <div v-if="!newGeneratedSecret" class="space-y-4">
        <div>
          <label class="block text-xs uppercase font-mono tracking-widest text-text-muted mb-2">Identifier / Name</label>
          <input 
            v-model="newKeyName" 
            type="text" 
            class="w-full bg-background border border-border-primary rounded px-4 py-2 text-text font-mono focus:border-primary focus:shadow-glow-primary focus:outline-none transition-all"
            placeholder="e.g., Python SDK, Mobile App"
            @keyup.enter="generateKey"
          />
        </div>
        <div class="flex justify-end gap-3">
          <button @click="isGeneratingFormOpen = false" class="btn-secondary">Cancel</button>
          <button @click="generateKey" :disabled="!newKeyName || isGenerating" class="btn-primary flex items-center gap-2">
            <svg v-if="isGenerating" class="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {{ isGenerating ? 'Executing...' : 'Generate' }}
          </button>
        </div>
      </div>

      <div v-else class="space-y-4">
        <div class="p-4 bg-primary/10 border border-primary rounded-md">
          <p class="text-sm text-primary mb-2 font-bold tracking-wider uppercase">Key Generated Successfully</p>
          <p class="text-xs text-text-muted mb-4">Please copy this secret key now. It will never be shown again.</p>
          
          <div class="flex items-center gap-2">
            <input 
              :value="newGeneratedSecret" 
              type="text" 
              readonly
              class="flex-1 bg-background border border-primary/50 text-white font-mono text-sm px-4 py-2 rounded focus:outline-none"
            />
            <button @click="copyToClipboard" class="btn-primary whitespace-nowrap">
              {{ copied ? 'Copied!' : 'Copy to Clipboard' }}
            </button>
          </div>
        </div>
        <div class="flex justify-end">
          <button @click="closeGeneration" class="btn-secondary">Done</button>
        </div>
      </div>
    </div>

    <!-- Key Manager -->
    <div class="bg-surface border border-border-primary rounded-lg overflow-hidden shadow-glow-primary">
      <div class="overflow-x-auto">
        <table class="w-full text-left border-collapse">
          <thead>
            <tr class="bg-background/50 border-b border-border-primary text-xs uppercase tracking-wider text-text-muted">
              <th class="px-6 py-4 font-medium">Name</th>
              <th class="px-6 py-4 font-medium">Created</th>
              <th class="px-6 py-4 font-medium">Last Used</th>
              <th class="px-6 py-4 font-medium text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-primary">
            <tr v-if="loading" class="hover:bg-primary/5 transition-colors">
              <td colspan="4" class="px-6 py-4 text-center text-text-muted">Loading constraints...</td>
            </tr>
            <tr v-else-if="keys.length === 0" class="hover:bg-primary/5 transition-colors">
              <td colspan="4" class="px-6 py-4 text-center text-text-muted">No API keys generated.</td>
            </tr>
            <tr v-else v-for="key in keys" :key="key.id" class="hover:bg-primary/5 transition-colors group">
              <td class="px-6 py-4">
                <div class="flex items-center gap-2">
                  <span class="w-2 h-2 rounded-full" :class="key.is_revoked ? 'bg-red-500 shadow-[0_0_5px_#ef4444]' : 'bg-primary shadow-[0_0_5px_#39ff14]'"></span>
                  <span class="font-medium" :class="key.is_revoked ? 'text-text-muted line-through' : 'text-text'">{{ key.name }}</span>
                </div>
              </td>
              <td class="px-6 py-4 text-sm text-text-muted font-mono">{{ formatDate(key.created_at) }}</td>
              <td class="px-6 py-4 text-sm text-text-muted font-mono">{{ formatDate(key.last_used_at) }}</td>
              <td class="px-6 py-4 text-right">
                <button 
                  v-if="!key.is_revoked"
                  @click="revokeKey(key.id)" 
                  class="text-red-500 hover:text-red-400 text-sm font-bold uppercase tracking-wider opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  Revoke
                </button>
                <span v-else class="text-xs text-red-500 uppercase tracking-wider font-bold bg-red-500/10 px-2 py-1 rounded">Revoked</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const activeKey = ref(localStorage.getItem('trindex_api_key') || '')
const keys = ref([])
const loading = ref(true)

const isGeneratingFormOpen = ref(false)
const isGenerating = ref(false)
const newKeyName = ref('')
const newGeneratedSecret = ref('')
const copied = ref(false)

const getHeaders = () => ({
  'Content-Type': 'application/json',
  'X-API-Key': activeKey.value
})

async function loadKeys() {
  try {
    const res = await fetch('/api/keys', { headers: getHeaders() })
    if (res.status === 401) {
      logout()
      return
    }
    const data = await res.json()
    keys.value = data || []
  } catch (e) {
    console.error('Failed to load keys', e)
  } finally {
    loading.value = false
  }
}

async function generateKey() {
  if (!newKeyName.value) return

  isGenerating.value = true
  try {
    const res = await fetch('/api/keys', {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ name: newKeyName.value })
    })
    const data = await res.json()
    if (data.secret) {
      newGeneratedSecret.value = data.secret
      await loadKeys()
    } else {
      alert('Failed to generate key: ' + JSON.stringify(data))
    }
  } catch (e) {
    alert('Error generating key: ' + e.message)
  } finally {
    isGenerating.value = false
  }
}

async function copyToClipboard() {
  try {
    await navigator.clipboard.writeText(newGeneratedSecret.value)
    copied.value = true
    setTimeout(() => copied.value = false, 2000)
  } catch (err) {
    console.error('Failed to copy text: ', err)
  }
}

function closeGeneration() {
  isGeneratingFormOpen.value = false
  newKeyName.value = ''
  newGeneratedSecret.value = ''
  copied.value = false
}

async function revokeKey(id) {
  if (!confirm('Are you sure you want to revoke this key? Any systems using it will immediately lose access.')) return
  
  try {
    const res = await fetch(`/api/keys/${id}`, {
      method: 'DELETE',
      headers: getHeaders()
    })
    if (res.status === 401) {
      logout()
      return
    }
    await loadKeys()
  } catch (e) {
    alert('Failed to revoke key')
  }
}

function logout() {
  localStorage.removeItem('trindex_api_key')
  router.push('/login')
}

function formatDate(isoString) {
  if (!isoString) return 'Never'
  return new Date(isoString).toLocaleString('en-US', {
    month: 'short', day: 'numeric', year: 'numeric',
    hour: 'numeric', minute: '2-digit'
  })
}

onMounted(() => {
  if (!activeKey.value) {
    router.push('/login')
    return
  }
  loadKeys()
})
</script>
