import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from './views/Dashboard.vue'
import MemoryBrowser from './views/MemoryBrowser.vue'
import Search from './views/Search.vue'
import Stats from './views/Stats.vue'

const routes = [
  { path: '/', name: 'Dashboard', component: Dashboard },
  { path: '/memories', name: 'MemoryBrowser', component: MemoryBrowser },
  { path: '/search', name: 'Search', component: Search },
  { path: '/stats', name: 'Stats', component: Stats }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
