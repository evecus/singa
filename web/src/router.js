import { createRouter, createWebHashHistory } from 'vue-router'

import DashboardView from './views/DashboardView.vue'
import NodesView     from './views/NodesView.vue'
import ProfilesView  from './views/ProfilesView.vue'
import RulesetsView  from './views/RulesetsView.vue'
import SettingsView  from './views/SettingsView.vue'
import LoginView     from './views/LoginView.vue'

const routes = [
  { path: '/',          redirect: '/dashboard' },
  { path: '/login',     component: LoginView, meta: { public: true } },
  { path: '/dashboard', component: DashboardView },
  { path: '/nodes',     component: NodesView },
  { path: '/profiles',  component: ProfilesView },
  { path: '/rulesets',  component: RulesetsView },
  { path: '/settings',  component: SettingsView },
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// Navigation guard: check auth before entering protected pages
router.beforeEach(async (to) => {
  if (to.meta.public) return true

  // Check auth status from backend
  try {
    const res = await fetch('/api/auth/status')
    const status = await res.json()

    if (!status.enabled) return true  // auth disabled

    const token = localStorage.getItem('singa_token')
    if (!token) {
      return '/login'
    }

    if (status.needsSetup) {
      return '/login'
    }

    // Verify token is still valid by checking a simple endpoint
    // We rely on the 401 handler in api.js for runtime expiry
    return true
  } catch {
    return true  // If we can't check, let it through
  }
})
