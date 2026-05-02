import { createRouter, createWebHashHistory } from 'vue-router'

// Static imports instead of lazy imports.
// Lazy imports (dynamic import()) cause Vite to emit separate chunk files
// with content-hash filenames (e.g. DashboardView-DVqpjB7B.js).
// When Go's embed.FS is built from an older dist/, these chunk files won't
// exist and the browser gets 404 → "Failed to fetch dynamically imported module".
// Using static imports bundles everything into a single file, eliminating
// the mismatch between index.html references and embedded assets.
import DashboardView from './views/DashboardView.vue'
import NodesView     from './views/NodesView.vue'
import ProfilesView  from './views/ProfilesView.vue'
import RulesetsView  from './views/RulesetsView.vue'
import SettingsView  from './views/SettingsView.vue'

const routes = [
  { path: '/',          redirect: '/dashboard' },
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
