<template>
  <div class="app">
    <!-- ── Mobile overlay ──────────────────────────────────────────── -->
    <div v-if="sidebarOpen" class="sidebar-overlay" @click="sidebarOpen = false"></div>

    <!-- ── Sidebar ─────────────────────────────────────────────────── -->
    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="sidebar-brand">
        <div class="brand-logo">S</div>
        <div class="brand-text">
          <span class="brand-name">singa</span>
          <span class="brand-ver">v{{ appVer }}</span>
        </div>
        <button class="sidebar-close" @click="sidebarOpen = false">✕</button>
      </div>

      <nav class="sidebar-nav">
        <span class="nav-section-label">主要</span>
        <RouterLink to="/dashboard" class="nav-item" active-class="active" @click="sidebarOpen = false">
          <span class="nav-icon">⬡</span><span>仪表盘</span>
        </RouterLink>
        <RouterLink to="/nodes" class="nav-item" active-class="active" @click="sidebarOpen = false">
          <span class="nav-icon">⬢</span><span>节点与订阅</span>
          <span v-if="totalNodes" class="nav-badge">{{ totalNodes }}</span>
        </RouterLink>
        <RouterLink to="/profiles" class="nav-item" active-class="active" @click="sidebarOpen = false">
          <span class="nav-icon">◈</span><span>配置文件</span>
        </RouterLink>

        <span class="nav-section-label">工具</span>
        <RouterLink to="/rulesets" class="nav-item" active-class="active" @click="sidebarOpen = false">
          <span class="nav-icon">⊞</span><span>规则集</span>
        </RouterLink>
        <RouterLink to="/settings" class="nav-item" active-class="active" @click="sidebarOpen = false">
          <span class="nav-icon">⚙</span><span>设置</span>
        </RouterLink>
      </nav>

      <div class="sidebar-status">
        <div class="status-row" :class="'status-' + statusStore.status.state">
          <div class="status-dot"></div>
          <span class="status-label">{{ stateLabel }}</span>
          <span v-if="statusStore.status.pid" class="text-xs text-muted monospace" style="margin-left:auto">
            {{ statusStore.status.pid }}
          </span>
        </div>
        <button v-if="authEnabled" class="nav-item" style="width:100%;margin-top:4px;color:var(--text3);font-size:12px" @click="doLogout">
          <span class="nav-icon" style="font-size:13px">⎋</span><span>退出登录</span>
        </button>
      </div>
    </aside>

    <!-- ── Main ───────────────────────────────────────────────────── -->
    <main class="main">
      <!-- 移动端悬浮汉堡按钮，叠在各页面 topbar 左侧 -->
      <button class="hamburger-fab" @click="sidebarOpen = true" aria-label="打开菜单">
        <span></span><span></span><span></span>
      </button>
      <RouterView />
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useStatusStore, useNodesStore, useSubsStore, useLogsStore, useAuthStore } from './stores.js'
import { api } from './api.js'

const statusStore = useStatusStore()
const nodesStore  = useNodesStore()
const subsStore   = useSubsStore()
const logsStore   = useLogsStore()
const authStore   = useAuthStore()
const router      = useRouter()

const authEnabled = ref(false)

async function checkAuthEnabled() {
  try {
    const s = await fetch('/api/auth/status')
    const d = await s.json()
    authEnabled.value = d.enabled
  } catch {}
}

async function doLogout() {
  try { await api('POST', '/auth/logout') } catch {}
  authStore.setToken('')
  router.push('/login')
}

const appVer = '2.0'
const sidebarOpen = ref(false)

const stateLabel = computed(() => ({
  running: '运行中', stopped: '已停止', error: '错误',
}[statusStore.status.state] || statusStore.status.state))

const totalNodes = computed(() =>
  nodesStore.nodes.length + subsStore.subs.reduce((a, s) => a + (s.nodeCount || 0), 0)
)

let poll = null
onMounted(async () => {
  checkAuthEnabled()
  await statusStore.fetch()
  await Promise.all([nodesStore.load(), subsStore.load()])
  if (statusStore.isRunning) logsStore.startSSE()
  poll = setInterval(statusStore.fetch, 8000)
})
onUnmounted(() => { clearInterval(poll); logsStore.stopSSE() })
</script>
