<template>
  <div class="page">
    <div class="topbar">
      <span class="topbar-title">仪表盘</span>
      <div class="topbar-right">
        <span class="text-xs text-muted monospace">{{ now }}</span>
        <button v-if="!isRunning" class="btn btn-primary btn-sm"
          :disabled="!canStart || starting" @click="doStart">
          {{ starting ? '启动中…' : '▶ 启动' }}
        </button>
        <button v-else class="btn btn-danger btn-sm" @click="doStop">
          ⏹ 停止
        </button>
      </div>
    </div>
    <div class="page" style="display:flex;flex-direction:column;gap:16px">

      <!-- ── Stats row ─────────────────────────────────────────────── -->
      <div class="grid-4">
        <div class="stat-widget">
          <span class="stat-label">状态</span>
          <span class="stat-value" :class="stateClass">{{ stateLabel }}</span>
          <span class="stat-sub">sing-box core</span>
        </div>
        <div class="stat-widget">
          <span class="stat-label">PID</span>
          <span class="stat-value">{{ status.pid || '—' }}</span>
          <span class="stat-sub">进程 ID</span>
        </div>
        <div class="stat-widget">
          <span class="stat-label">内存</span>
          <span class="stat-value">{{ memStr }}</span>
          <span class="stat-sub">RSS 占用</span>
        </div>
        <div class="stat-widget">
          <span class="stat-label">版本</span>
          <span class="stat-value" style="font-size:14px">{{ sbVersion || '未安装' }}</span>
          <span class="stat-sub">sing-box</span>
        </div>
      </div>

      <!-- ── Control + Info ────────────────────────────────────────── -->
      <div class="grid-2" style="align-items:start">

        <!-- Left: Control panel -->
        <div style="display:flex;flex-direction:column;gap:14px">

          <!-- Config selector -->
          <div class="card">
            <div class="card-title">选择配置</div>
            <div class="field" style="margin-bottom:10px">
              <label class="field-label">配置模式</label>
              <div class="mode-grid">
                <div class="mode-card" :class="{ on: params.configMode === 'node' }"
                  @click="params.configMode = 'node'">
                  <div class="mode-card-icon">🔗</div>
                  <div class="mode-card-name">单节点</div>
                  <div class="mode-card-desc">从节点列表选择</div>
                </div>
                <div class="mode-card" :class="{ on: params.configMode === 'profile' || params.configMode === 'upload' }"
                  @click="params.configMode = 'profile'">
                  <div class="mode-card-icon">📄</div>
                  <div class="mode-card-name">配置文件</div>
                  <div class="mode-card-desc">生成或上传的配置</div>
                </div>
              </div>
            </div>

            <!-- Node selector -->
            <div v-if="params.configMode === 'node'" class="field" style="margin-bottom:10px">
              <label class="field-label">选择节点</label>
              <button class="select" style="text-align:left;cursor:pointer" @click="showNodePicker=true">
                <template v-if="selectedNodeLabel">{{ selectedNodeLabel }}</template>
                <span v-else style="color:var(--text3)">— 点击选择节点 —</span>
              </button>
            </div>

            <!-- Profile / upload selector -->
            <div v-if="params.configMode === 'profile' || params.configMode === 'upload'" class="field" style="margin-bottom:10px">
              <label class="field-label">选择配置文件</label>
              <button class="select" style="text-align:left;cursor:pointer" @click="showProfilePicker=true">
                <template v-if="selectedProfileLabel">{{ selectedProfileLabel }}</template>
                <span v-else style="color:var(--text3)">— 点击选择配置 —</span>
              </button>
            </div>
          </div>

          <!-- Route mode + blockAds: single-node only -->
          <div v-if="params.configMode === 'node'" class="card">
            <div class="card-title">路由模式</div>
            <div class="field" style="margin-bottom:10px">
              <div class="seg">
                <button class="seg-btn" :class="{ on: params.routeMode === 'whitelist' }"
                  @click="params.routeMode = 'whitelist'">🇨🇳 大陆白名单</button>
                <button class="seg-btn" :class="{ on: params.routeMode === 'gfwlist' }"
                  @click="params.routeMode = 'gfwlist'">📋 GFW列表</button>
                <button class="seg-btn" :class="{ on: params.routeMode === 'global' }"
                  @click="params.routeMode = 'global'">🌍 全局</button>
              </div>
            </div>
            <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px">
              <div class="toggle" :class="{ on: params.blockAds }"
                @click="params.blockAds = !params.blockAds"></div>
              <span>广告拦截</span>
            </label>
            <div class="field-hint" style="margin-top:10px">
              代理模式、局域网代理、IPv6 在
              <router-link to="/settings" style="color:var(--accent)">设置</router-link>
              中配置，当前：<strong>{{ currentProxyModeLabel }}</strong>
            </div>
          </div>

          <div v-if="startErr" class="alert alert-error">{{ startErr }}</div>
        </div>

        <!-- Right: Status info + log -->
        <div style="display:flex;flex-direction:column;gap:14px">

          <!-- Runtime info -->
          <div class="card" v-if="isRunning">
            <div class="card-title">运行信息</div>
            <div class="info-table">
              <span class="info-k">代理模式</span>
              <span class="info-v">{{ status.proxyMode || '—' }}</span>
              <span class="info-k">路由模式</span>
              <span class="info-v">{{ status.routeMode || '—' }}</span>
              <template v-if="status.ports?.mixed">
                <span class="info-k">HTTP/SOCKS5</span>
                <span class="info-v">127.0.0.1:{{ status.ports.mixed }}</span>
              </template>
              <template v-if="status.ports?.tproxy">
                <span class="info-k">TProxy</span>
                <span class="info-v">:{{ status.ports.tproxy }}</span>
              </template>
              <template v-if="status.ports?.dns">
                <span class="info-k">DNS</span>
                <span class="info-v">:{{ status.ports.dns }}</span>
              </template>
            </div>
          </div>

          <!-- Mini log -->
          <div class="card" style="padding:0;overflow:hidden">
            <div class="card-title-row" style="padding:12px 14px 0">
              <span class="card-title" style="margin:0">实时日志</span>
              <button class="btn btn-ghost btn-sm" @click="logsStore.clear()">清空</button>
            </div>
            <div class="log-panel" style="height:220px;border-radius:0;box-shadow:none;background:#0f1117">
              <div class="log-toolbar">
                <div v-if="isRunning" class="log-dot"></div>
                <span class="log-label">{{ isRunning ? 'LIVE' : 'IDLE' }}</span>
              </div>
              <div class="log-body" ref="logEl">
                <span v-if="!logsStore.logs.length" class="log-empty">等待日志…</span>
                <span v-for="(l, i) in logsStore.logs.slice(-80)" :key="i"
                  class="log-line" :class="logCls(l)">{{ l }}<br></span>
              </div>
            </div>
          </div>

        </div>
      </div>

    </div>

    <!-- ── 配置文件选择弹窗 ────────────────────────────────────────────── -->
    <div v-if="showProfilePicker" class="mask" @click.self="showProfilePicker=false">
      <div class="modal" style="max-width:520px;max-height:80vh;display:flex;flex-direction:column">
        <div class="modal-head">
          <span>选择配置文件</span>
          <button class="btn-icon" @click="showProfilePicker=false">✕</button>
        </div>
        <div style="padding:12px 16px;overflow-y:auto;flex:1">

          <!-- 生成配置 -->
          <div class="np-group-title">生成配置（{{ profilesStore.profiles.length }}）</div>
          <div v-if="!profilesStore.profiles.length" class="empty" style="padding:8px 0;font-size:12px">
            暂无配置，前往<router-link to="/profiles" style="color:var(--accent)">配置文件</router-link>页新增
          </div>
          <div v-else style="display:flex;flex-direction:column;gap:6px">
            <button v-for="prof in profilesStore.profiles" :key="prof.id"
              class="profile-pick-item" :class="{ on: params.profileId === prof.id && params.configMode === 'profile' }"
              :disabled="!prof.wizardConfig"
              @click="pickProfile(prof)">
              <span style="font-size:16px">📄</span>
              <div style="flex:1;text-align:left">
                <div style="font-weight:600;font-size:13px">{{ prof.name }}</div>
                <div style="font-size:11px;color:var(--text3)">
                  <span v-if="!prof.wizardConfig" style="color:var(--red)">未完成配置</span>
                </div>
              </div>
              <span v-if="params.profileId === prof.id && params.configMode === 'profile'"
                style="color:var(--accent);font-size:14px">✓</span>
            </button>
          </div>

          <!-- 上传配置 -->
          <div class="np-group-title" style="margin-top:14px">上传配置</div>
          <button class="profile-pick-item" :class="{ on: params.configMode === 'upload' }"
            @click="pickUpload">
            <span style="font-size:16px">📁</span>
            <div style="flex:1;text-align:left">
              <div style="font-weight:600;font-size:13px">已上传的 JSON 配置</div>
              <div style="font-size:11px;color:var(--text3)">使用上传配置文件页面上传的 config.json</div>
            </div>
            <span v-if="params.configMode === 'upload'"
              style="color:var(--accent);font-size:14px">✓</span>
          </button>

        </div>
        <div style="padding:10px 16px;border-top:1px solid var(--border);text-align:right">
          <button class="btn btn-ghost btn-sm" @click="showProfilePicker=false">关闭</button>
        </div>
      </div>
    </div>

    <!-- ── 节点选择弹窗 ──────────────────────────────────────────────── -->
    <div v-if="showNodePicker" class="mask" @click.self="showNodePicker=false">
      <div class="modal" style="max-width:580px;max-height:80vh;display:flex;flex-direction:column">
        <div class="modal-head">
          <span>选择节点</span>
          <button class="btn-icon" @click="showNodePicker=false">✕</button>
        </div>
        <div style="padding:12px 16px;overflow-y:auto;flex:1">

          <!-- 导入节点 -->
          <div class="np-group-title">导入节点（{{ nodesStore.nodes.length }}）</div>
          <div v-if="!nodesStore.nodes.length" class="empty" style="padding:8px 0;font-size:12px">暂无导入节点</div>
          <div v-else class="np-grid">
            <button v-for="n in nodesStore.nodes" :key="n.id"
              class="np-item" :class="{ on: params.nodeId === n.id }"
              @click="pickNode(n.id, '['+n.type+'] '+n.name)">
              <span class="np-type">{{ n.type }}</span>
              <span class="np-name">{{ n.name }}</span>
              <span class="np-addr">{{ n.server }}</span>
            </button>
          </div>

          <!-- 各订阅节点 -->
          <template v-for="sub in subsStore.subs" :key="sub.id">
            <div class="np-group-title" style="margin-top:14px;display:flex;align-items:center;gap:8px">
              <span>订阅：{{ sub.name }}（{{ sub.nodeCount || 0 }}）</span>
              <button v-if="!subProxyCache[sub.id]" class="btn btn-ghost btn-sm"
                style="font-size:11px" @click="loadSubProxies(sub.id)">
                展开节点
              </button>
              <button v-else class="btn btn-ghost btn-sm"
                style="font-size:11px" @click="delete subProxyCache[sub.id]">
                收起
              </button>
            </div>
            <div v-if="subProxyCache[sub.id]" class="np-grid">
              <button v-for="(p, idx) in subProxyCache[sub.id]" :key="idx"
                class="np-item"
                :class="{ on: params.configMode === 'subnode' && params.subscriptionId === sub.id && params.subNodeIdx === idx }"
                @click="pickSubNode(sub.id, idx, p)">
                <span class="np-type">{{ p.type }}</span>
                <span class="np-name">{{ p.name || p.tag || '节点'+(idx+1) }}</span>
                <span class="np-addr">{{ p.server }}</span>
              </button>
            </div>
          </template>

        </div>
        <div style="padding:10px 16px;border-top:1px solid var(--border);text-align:right">
          <button class="btn btn-ghost btn-sm" @click="showNodePicker=false">关闭</button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { api } from '../api.js'
import { useStatusStore, useNodesStore, useSubsStore, useLogsStore, useProfilesStore } from '../stores.js'

const statusStore   = useStatusStore()
const profilesStore = useProfilesStore()
const subsStore     = useSubsStore()
const nodesStore    = useNodesStore()
const logsStore     = useLogsStore()

const status    = computed(() => statusStore.status)
const isRunning = computed(() => statusStore.isRunning)

const sbVersion = ref('')
const memStr    = ref('—')
const now       = ref('')
const starting  = ref(false)
const startErr  = ref('')
const logEl     = ref(null)

const stateLabel = computed(() => ({
  running:'运行中', stopped:'已停止', error:'错误',
}[status.value.state] || status.value.state))
const stateClass = computed(() => ({
  running:'green', stopped:'', error:'red',
}[status.value.state] || ''))

// Global proxy settings loaded from backend
const proxySettings = ref({ tcpMode: 'off', udpMode: 'off', lanProxy: false, ipv6: false })

const currentProxyModeLabel = computed(() => {
  const ps = proxySettings.value
  if (ps.tcpMode === 'tun' || ps.udpMode === 'tun') return 'TUN 虚拟网卡'
  if (ps.tcpMode === 'tproxy' || ps.udpMode === 'tproxy') return 'TPROXY'
  if (ps.tcpMode === 'redir') return 'redir'
  return '系统代理'
})

const params = reactive({
  configMode: 'node',
  nodeId: '',
  subscriptionId: '',
  subNodeIdx: -1,
  profileId: '',
  routeMode: 'whitelist',
  blockAds: true,
})

const canStart = computed(() => {
  if (params.configMode === 'node') return !!params.nodeId
  if (params.configMode === 'subnode') return !!params.subscriptionId && params.subNodeIdx >= 0
  if (params.configMode === 'profile') return !!params.profileId
  if (params.configMode === 'upload') return true
  return false
})

// ── Profile picker ────────────────────────────────────────────────────────
const showProfilePicker    = ref(false)
const selectedProfileLabel = ref('')

function pickProfile(prof) {
  params.configMode = 'profile'
  params.profileId  = prof.id
  selectedProfileLabel.value = '📄 ' + prof.name
  showProfilePicker.value = false
}

function pickUpload() {
  params.configMode = 'upload'
  params.profileId  = ''
  selectedProfileLabel.value = '📁 已上传的 JSON 配置'
  showProfilePicker.value = false
}

// ── Node picker ───────────────────────────────────────────────────────────
const showNodePicker    = ref(false)
const selectedNodeLabel = ref('')
const subProxyCache     = reactive({})

async function loadSubProxies(subId) {
  try { subProxyCache[subId] = await subsStore.getProxies(subId) } catch {}
}

function pickNode(id, label) {
  params.configMode = 'node'
  params.nodeId = id
  selectedNodeLabel.value = label
  showNodePicker.value = false
}

function pickSubNode(subId, idx, proxy) {
  params.configMode = 'subnode'
  params.subscriptionId = subId
  params.subNodeIdx = idx
  const name = proxy.name || proxy.tag || ('节点' + (idx + 1))
  selectedNodeLabel.value = `[${proxy.type}] ${name}`
  showNodePicker.value = false
}

async function doStart() {
  starting.value = true; startErr.value = ''
  try {
    await api('POST', '/start', {
      configMode:     params.configMode,
      nodeId:         params.nodeId,
      subscriptionId: params.subscriptionId,
      subNodeIdx:     params.subNodeIdx,
      profileId:      params.profileId,
      routeMode:      params.routeMode,
      blockAds:       params.blockAds,
    })
    await statusStore.fetch()
    logsStore.startSSE()
  } catch (e) { startErr.value = e.message }
  finally { starting.value = false }
}

async function doStop() {
  await statusStore.stop()
  logsStore.stopSSE()
}

function logCls(l) {
  const s = l.toLowerCase()
  if (s.includes('error') || s.includes('fatal')) return 'err'
  if (s.includes('warn')) return 'warn'
  if (s.includes('info')) return 'info'
  return ''
}

// Auto-scroll log
watch(() => logsStore.logs.length, () => {
  nextTick(() => { if (logEl.value) logEl.value.scrollTop = logEl.value.scrollHeight })
})

// Clock & memory
async function readMem() {
  if (!status.value.pid) { memStr.value = '—'; return }
  try {
    const r = await fetch(`/proc/${status.value.pid}/status`)
    const t = await r.text()
    const m = /VmRSS:\s+(\d+)/.exec(t)
    if (m) memStr.value = (parseInt(m[1]) / 1024).toFixed(1) + ' MB'
  } catch {
    // /proc not directly accessible via fetch; use status endpoint workaround
    memStr.value = isRunning.value ? '运行中' : '—'
  }
}

let clockTimer = null
onMounted(async () => {
  try { const r = await api('GET', '/singbox/version'); sbVersion.value = r.version } catch {}
  try { const r = await api('GET', '/proxy-settings'); proxySettings.value = r } catch {}
  profilesStore.load()
  nodesStore.load()
  subsStore.load()
  // Pre-fill from saved status
  if (status.value.configMode) params.configMode = status.value.configMode
  if (status.value.nodeId)     params.nodeId     = status.value.nodeId
  if (status.value.routeMode)  params.routeMode  = status.value.routeMode
  clockTimer = setInterval(() => {
    now.value = new Date().toLocaleTimeString('zh-CN')
  }, 1000)
  now.value = new Date().toLocaleTimeString('zh-CN')
})
onUnmounted(() => clearInterval(clockTimer))
</script>

<style scoped>
.profile-pick-item {
  display: flex; align-items: center; gap: 10px; padding: 10px 12px; width: 100%;
  background: var(--surface2); border: 1.5px solid var(--border2); border-radius: var(--radius);
  cursor: pointer; transition: all .12s; text-align: left;
}
.profile-pick-item:hover:not(:disabled) { border-color: var(--accent); }
.profile-pick-item.on { border-color: var(--accent); background: var(--accent-bg); }
.profile-pick-item:disabled { opacity: .5; cursor: not-allowed; }
.np-group-title {
  font-size: 12px; font-weight: 700; color: var(--text2);
  padding: 6px 0 4px; border-bottom: 1px solid var(--border); margin-bottom: 6px;
}
.np-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 6px; margin-bottom: 4px;
}
.np-item {
  display: flex; flex-direction: column; gap: 2px; padding: 7px 10px;
  background: var(--surface2); border: 1.5px solid var(--border2); border-radius: var(--radius);
  cursor: pointer; text-align: left; transition: all .12s;
}
.np-item:hover { border-color: var(--accent); }
.np-item.on { border-color: var(--accent); background: var(--accent-bg); }
.np-type { font-size: 10px; font-weight: 700; color: var(--accent); font-family: var(--mono); }
.np-name { font-size: 12px; font-weight: 600; color: var(--text1); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.np-addr { font-size: 10px; color: var(--text3); font-family: var(--mono); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

@media (max-width: 640px) {
  .profile-pick-item { padding: 8px 10px; }
  .np-grid { grid-template-columns: 1fr 1fr; }
}
@media (max-width: 400px) {
  .np-grid { grid-template-columns: 1fr; }
}

</style>
