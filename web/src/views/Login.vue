<template>
  <div class="flex items-center justify-center min-h-[70vh]">
    <div class="bg-surface border border-border-primary p-8 rounded-lg shadow-glow-primary max-w-md w-full">
      <h2 class="text-2xl font-bold mb-6 text-primary glow-text text-center">SYSTEM ACCESS</h2>
      
      <form @submit.prevent="handleLogin" class="space-y-6">
        <div>
          <label class="block text-sm font-medium text-text-muted mb-2">API Key / Authorization Token</label>
          <input 
            v-model="apiKey" 
            type="password" 
            class="w-full bg-background border border-border-primary rounded px-4 py-3 text-text focus:outline-none focus:border-primary focus:shadow-glow-primary font-mono transition-all"
            placeholder="trindex_..."
            required
          />
        </div>
        
        <button 
          type="submit" 
          class="w-full bg-primary/10 hover:bg-primary/20 text-primary border border-primary px-6 py-3 rounded uppercase tracking-wider font-bold transition-all shadow-glow-primary hover:shadow-glow-primary-lg"
        >
          Authenticate
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const apiKey = ref('')

function handleLogin() {
  if (apiKey.value) {
    localStorage.setItem('trindex_api_key', apiKey.value)
    router.push('/')
  }
}
</script>
