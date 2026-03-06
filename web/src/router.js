import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from './views/Dashboard.vue'
import MemoryBrowser from './views/MemoryBrowser.vue'
import Search from './views/Search.vue'
import Stats from './views/Stats.vue'
import Login from './views/Login.vue'
import Settings from './views/Settings.vue'

const routes = [
  { path: '/login', name: 'Login', component: Login },
  { path: '/', name: 'Dashboard', component: Dashboard, meta: { requiresAuth: true } },
  { path: '/memories', name: 'MemoryBrowser', component: MemoryBrowser, meta: { requiresAuth: true } },
  { path: '/search', name: 'Search', component: Search, meta: { requiresAuth: true } },
  { path: '/stats', name: 'Stats', component: Stats, meta: { requiresAuth: true } },
  { path: '/settings', name: 'Settings', component: Settings, meta: { requiresAuth: true } }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const isAuthenticated = !!localStorage.getItem('trindex_api_key')

  if (to.meta.requiresAuth && !isAuthenticated) {
    next({ name: 'Login' })
  } else if (to.name === 'Login' && isAuthenticated) {
    next({ name: 'Dashboard' })
  } else {
    next()
  }
})

export default router
