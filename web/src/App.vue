<template>
  <div class="app">

    <!-- ── Top bar ── -->
    <header class="topbar">
      <div class="brand">
        <span class="brand-icon">◈</span>
        <span class="brand-name">singa</span>
        <span class="brand-sub">PROXY MANAGER</span>
      </div>
      <nav class="tab-nav">
        <button class="tab-btn" :class="{active: tab==='config'}" @click="tab='config'">配置</button>
        <button class="tab-btn" :class="{active: tab==='logs'}"   @click="tab='logs'">日志</button>
      </nav>
      <div class="topbar-right">
        <div class="status-pill" :class="statusClass">
          <span class="status-dot"></span>
          <span>{{ statusLabel }}</span>
        </div>
        <button class="settings-btn" :class="{active: tab==='settings'}" @click="tab='settings'" title="设置">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="3"/>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
          </svg>
        </button>
      </div>
    </header>

    <!-- ── Config tab ── -->
    <div v-show="tab==='config'" class="tab-content">
      <div class="sidebar">

        <div class="section">
          <div class="section-title">配置模式</div>
          <div class="seg">
            <button class="seg-btn" :class="{on: configMode==='node'}"
              :disabled="isRunning" @click="configMode='node'">⬡ 节点模式</button>
            <button class="seg-btn" :class="{on: configMode==='upload'}"
              :disabled="isRunning" @click="configMode='upload'">⬢ 上传配置</button>
          </div>
        </div>

        <template v-if="configMode==='node'">
          <div class="section">
            <div class="section-title-row">
              <span class="section-title">节点</span>
              <button class="icon-btn" @click="showImport=true">＋ 导入</button>
            </div>
            <div v-if="nodes.length===0" class="empty-tip">暂无节点，点击「导入」添加</div>
            <div v-else class="node-list">
              <div v-for="n in nodes" :key="n.id"
                class="node-row" :class="{active: selectedNodeId===n.id}"
                @click="selectedNodeId=n.id">
                <span class="proto-badge" :class="'p-'+n.protocol">{{ n.protocol }}</span>
                <div class="node-meta">
                  <span class="node-name">{{ n.name || n.address }}</span>
                  <span class="node-addr">{{ n.address }}:{{ n.port }}</span>
                </div>
                <button class="del-btn" :disabled="isRunning" @click.stop="deleteNode(n.id)">✕</button>
              </div>
            </div>
          </div>

          <div class="section">
            <div class="section-title">路由模式</div>
            <div class="route-grid">
              <button v-for="r in routeModes" :key="r.value"
                class="route-btn" :class="{on: routeMode===r.value}"
                :disabled="isRunning" @click="routeMode=r.value">
                <span class="route-icon">{{ r.icon }}</span>
                <span class="route-name">{{ r.label }}</span>
                <span class="route-desc">{{ r.desc }}</span>
              </button>
            </div>
          </div>
        </template>

        <template v-else>
          <div class="section">
            <div class="section-title">配置文件</div>
            <div class="dropzone" :class="{over: isDragging, loaded: uploadInfo}"
              @dragover.prevent="isDragging=true" @dragleave="isDragging=false"
              @drop.prevent="onDrop" @click="$refs.fileInput.click()">
              <input ref="fileInput" type="file" accept=".json" style="display:none" @change="onFileChange"/>
              <span class="dz-icon">{{ uploadInfo ? '✓' : '↑' }}</span>
              <span class="dz-text">{{ uploadInfo ? 'config.json 已加载' : '点击或拖拽上传 config.json' }}</span>
            </div>
            <div v-if="uploadInfo && uploadInfo.inbounds && uploadInfo.inbounds.length" class="ib-list">
              <div v-for="ib in uploadInfo.inbounds" :key="ib.tag"
                class="ib-row" :class="{match: ib.type===proxyMode}">
                <span class="ib-type">{{ ib.type }}</span>
                <span class="ib-tag">{{ ib.tag || '—' }}</span>
                <span class="ib-port" v-if="ib.port">:{{ ib.port }}</span>
              </div>
            </div>
          </div>
        </template>

        <div class="section">
          <div class="section-title">透明代理模式</div>
          <div class="proxy-grid">
            <button v-for="m in proxyModes" :key="m.value"
              class="proxy-btn" :class="{on: proxyMode===m.value}"
              :disabled="isRunning" @click="proxyMode=m.value">
              <span class="proxy-icon">{{ m.icon }}</span>
              <span class="proxy-name">{{ m.label }}</span>
              <span class="proxy-desc">{{ m.desc }}</span>
            </button>
          </div>
        </div>

        <div class="section">
          <div class="section-title">网络选项</div>
          <div class="toggle-group">
            <label class="toggle-row" :class="{disabled: isRunning}">
              <div class="toggle-track" :class="{on: lanProxy}"
                @click="!isRunning && (lanProxy=!lanProxy)">
                <div class="toggle-thumb"></div>
              </div>
              <div class="toggle-labels">
                <span class="toggle-name">局域网代理</span>
                <span class="toggle-hint">{{ lanProxy ? '代理局域网设备，开启 ip_forward' : '仅代理本机流量' }}</span>
              </div>
            </label>
            <label class="toggle-row" :class="{disabled: isRunning}">
              <div class="toggle-track" :class="{on: ipv6}"
                @click="!isRunning && (ipv6=!ipv6)">
                <div class="toggle-thumb"></div>
              </div>
              <div class="toggle-labels">
                <span class="toggle-name">IPv6 代理</span>
                <span class="toggle-hint">{{ ipv6 ? '同时代理 IPv6 流量' : '仅代理 IPv4 流量' }}</span>
              </div>
            </label>
            <label class="toggle-row" :class="{disabled: isRunning || configMode==='upload'}">
              <div class="toggle-track" :class="{on: blockAds}"
                @click="!isRunning && configMode!=='upload' && (blockAds=!blockAds)">
                <div class="toggle-thumb"></div>
              </div>
              <div class="toggle-labels">
                <span class="toggle-name">去广告</span>
                <span class="toggle-hint">{{ configMode==='upload' ? '仅节点模式可用' : blockAds ? '拦截广告域名请求' : '不拦截广告' }}</span>
              </div>
            </label>
          </div>
        </div>

        <div class="section action-section">
          <button v-if="!isRunning" class="btn-start"
            :disabled="startDisabled" @click="startCore">
            ▶ 启动
          </button>
          <button v-else class="btn-stop" @click="stopCore">
            ■ 停止
          </button>
        </div>

        <div v-if="errorMsg" class="error-bar">⚠ {{ errorMsg }}</div>

        <div v-if="isRunning" class="section">
          <div class="section-title">运行状态</div>
          <div class="info-grid">
            <template v-for="(v,k) in runtimeInfo" :key="k">
              <span class="info-k">{{ k }}</span>
              <span class="info-v">{{ v }}</span>
            </template>
          </div>
        </div>

      </div>
    </div>

    <!-- ── Logs tab ── -->
    <div v-show="tab==='logs'" class="tab-content logpane">
      <div class="log-inner">
      <div class="log-topbar">
        <span class="log-live" :class="{live: isRunning}">
          {{ isRunning ? '● LIVE' : '○ IDLE' }}
        </span>
        <button class="log-clear" @click="logs=[]">清空</button>
      </div>
      <div class="log-body" ref="logEl" @scroll="onLogScroll">
        <div v-if="logs.length===0" class="log-empty">等待核心启动…</div>
        <div v-for="(line,i) in logs" :key="i" class="log-line">
          <span class="log-arr">›</span>
          <span class="log-txt" :class="logClass(line)">{{ line }}</span>
        </div>
      </div>
      <div class="log-foot">{{ logs.length }} lines</div>
      </div>
    </div>

    <!-- ── Settings tab ── -->
    <div v-show="tab==='settings'" class="tab-content">
      <div class="sidebar">

        <!-- sing-box core -->
        <div class="section">
          <div class="section-title">sing-box 核心</div>
          <div class="info-grid" v-if="sbInfo">
            <span class="info-k">已安装版本</span>
            <span class="info-v">{{ sbInfo.version || '未安装' }}</span>
            <span class="info-k">架构</span>
            <span class="info-v">{{ sbInfo.arch }}</span>
            <span class="info-k">系统</span>
            <span class="info-v">{{ sbInfo.osName }} / {{ sbInfo.libc }}</span>
          </div>
          <div class="section-title" style="margin-top:4px">选择版本</div>
          <div class="flavor-grid">
            <button class="flavor-btn" :class="{on: sbFlavor==='official'}" @click="sbFlavor='official'">
              <span class="flavor-name">官方版</span>
              <span class="flavor-desc">SagerNet/sing-box</span>
            </button>
            <button class="flavor-btn" :class="{on: sbFlavor==='ref1nd'}" @click="sbFlavor='ref1nd'">
              <span class="flavor-name">reF1nd 版</span>
              <span class="flavor-desc">reF1nd/sing-box-releases</span>
            </button>
          </div>
          <div class="section-title" style="margin-top:4px">选择版本号</div>
          <div class="version-selector">
            <label class="ver-opt" :class="{on: sbVersionMode==='latest'}" @click="sbVersionMode='latest'">
              <span class="ver-radio"></span>latest
            </label>
            <label class="ver-opt" :class="{on: sbVersionMode==='custom'}" @click="sbVersionMode='custom'">
              <span class="ver-radio"></span>自定义
            </label>
            <input
              v-if="sbVersionMode==='custom'"
              class="text-input ver-input"
              v-model="sbVersionInput"
              placeholder="请输入版本号"
              spellcheck="false"
            />
          </div>
          <div class="settings-actions">
            <button class="icon-btn secondary" @click="fetchSbVersion" :class="{loading: sbChecking}" :disabled="sbChecking">
              {{ sbChecking ? '检测中…' : '↺ 检测已安装版本' }}
            </button>
            <button class="icon-btn" @click="installSingbox" :class="{loading: sbInstalling}" :disabled="sbInstalling">
              {{ sbInstalling ? '下载中…' : '↓ 下载/更新核心' }}
            </button>
          </div>
          <div v-if="sbMsg" class="update-msg" :class="sbMsgClass">{{ sbMsg }}</div>
        </div>

        <!-- 规则集 -->
        <div class="section">
          <div class="section-title">规则集</div>
          <div class="settings-actions">
            <button class="icon-btn" @click="updateRules" :class="{loading: updatingRules}" :disabled="updatingRules">
              {{ updatingRules ? '更新中…' : '↻ 更新规则集' }}
            </button>
          </div>
          <div v-if="updateRulesMsg" class="update-msg" :class="updateRulesMsgClass">{{ updateRulesMsg }}</div>
          <div v-if="updateRulesDetail.length" class="rules-detail">
            <div v-for="r in updateRulesDetail" :key="r.file" class="rules-row" :class="{err: r.error}">
              <span class="rules-file">{{ r.file }}</span>
              <span class="rules-mirror">{{ r.error || r.mirror }}</span>
            </div>
          </div>
        </div>

        <!-- GitHub 代理 -->
        <div class="section">
          <div class="section-title">GitHub 代理加速</div>
          <p class="settings-hint">用于更新规则集和下载 sing-box 核心，留空则直连后自动尝试内置镜像</p>
          <div class="input-row">
            <input class="text-input" v-model="ghProxy" placeholder="https://your-proxy.com/" spellcheck="false"/>
            <button class="icon-btn" @click="saveProxy">保存</button>
          </div>
          <div class="proxy-presets">
            <span class="preset-label">预设：</span>
            <button v-for="p in proxyPresets" :key="p" class="preset-btn" @click="ghProxy=p">{{ p }}</button>
          </div>
        </div>

      </div>
    </div>

    <!-- ── Import modal ── -->
    <div v-if="showImport" class="mask" @click.self="closeImport">
      <div class="modal">
        <div class="modal-head">
          <span>导入节点</span>
          <button class="modal-x" @click="closeImport">✕</button>
        </div>
        <div class="modal-body">
          <p class="modal-hint">每行一个分享链接，支持 vmess / vless / trojan / ss / tuic / hy2</p>
          <textarea v-model="importText" class="import-ta"
            placeholder="vmess://...&#10;vless://...&#10;tuic://...&#10;hy2://..."></textarea>
          <div v-if="importErrors.length" class="import-errs">
            <div v-for="(e,i) in importErrors" :key="i" class="import-err">⚠ {{ e }}</div>
          </div>
        </div>
        <div class="modal-foot">
          <button class="btn-cancel" @click="closeImport">取消</button>
          <button class="btn-ok" :disabled="!importText.trim()" @click="doImport">导入</button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'

const tab            = ref('config')
const configMode     = ref('node')
const proxyMode      = ref('tproxy')
const routeMode      = ref('whitelist')
const lanProxy       = ref(false)
const ipv6           = ref(false)
const blockAds       = ref(false)

// Settings state
const ghProxy        = ref('')
const sbInfo         = ref(null)
const sbFlavor       = ref('official')
const sbChecking     = ref(false)
const sbInstalling   = ref(false)
const sbMsg          = ref('')
const sbMsgClass     = ref('')
const sbVersionMode  = ref('latest')  // 'latest' | 'custom'
const sbVersionInput = ref('')
const updatingRules      = ref(false)
const updateRulesMsg     = ref('')
const updateRulesMsgClass = ref('')
const updateRulesDetail  = ref([])

const proxyPresets = [
  'https://ghfast.top/',
  'https://gh-proxy.com/',
  'https://ghproxy.it/',
]

function saveProxy() {
  localStorage.setItem('ghProxy', ghProxy.value)
}

function loadProxy() {
  ghProxy.value = localStorage.getItem('ghProxy') || ''
}

async function fetchSbVersion() {
  sbChecking.value = true
  sbMsg.value = ''
  try {
    sbInfo.value = await api('GET', '/singbox/version')
  } catch (e) {
    sbMsg.value = '✕ ' + e.message
    sbMsgClass.value = 'msg-err'
  } finally {
    sbChecking.value = false
  }
}

async function installSingbox() {
  sbInstalling.value = true
  sbMsg.value = ''
  try {
    const version = sbVersionMode.value === 'custom' && sbVersionInput.value.trim()
      ? sbVersionInput.value.trim()
      : 'latest'
    const res = await api('POST', '/singbox/install', { proxy: ghProxy.value, flavor: sbFlavor.value, version })
    sbMsg.value = `✓ 安装成功：${res.version}`
    sbMsgClass.value = 'msg-ok'
    await fetchSbVersion()
  } catch (e) {
    sbMsg.value = '✕ ' + e.message
    sbMsgClass.value = 'msg-err'
  } finally {
    sbInstalling.value = false
  }
}

async function updateRules() {
  updatingRules.value = true
  updateRulesMsg.value = ''
  updateRulesDetail.value = []
  try {
    const res = await api('POST', '/update-rules', { proxy: ghProxy.value })
    updateRulesDetail.value = res.results || []
    if (res.failed === 0) {
      updateRulesMsg.value = `✓ 全部 ${res.total} 个规则集更新成功`
      updateRulesMsgClass.value = 'msg-ok'
    } else if (res.failed < res.total) {
      updateRulesMsg.value = `⚠ ${res.total - res.failed}/${res.total} 成功，${res.failed} 个失败`
      updateRulesMsgClass.value = 'msg-warn'
    } else {
      updateRulesMsg.value = `✕ 全部更新失败，请检查网络或设置代理`
      updateRulesMsgClass.value = 'msg-err'
    }
  } catch (e) {
    updateRulesMsg.value = `✕ ${e.message}`
    updateRulesMsgClass.value = 'msg-err'
  } finally {
    updatingRules.value = false
  }
}

const selectedNodeId = ref('')
const nodes        = ref([])
const uploadInfo   = ref(null)
const status       = ref({ state: 'stopped' })
const logs         = ref([])
const errorMsg     = ref('')
const isDragging   = ref(false)
const showImport   = ref(false)
const importText   = ref('')
const importErrors = ref([])
const logEl        = ref(null)
const autoScroll   = ref(true)

let sseSource = null
let pollTimer = null

const isRunning = computed(() => status.value.state === 'running')

const startDisabled = computed(() => {
  if (configMode.value === 'node' && !selectedNodeId.value) return true
  if (configMode.value === 'upload' && !uploadInfo.value) return true
  return false
})

const statusClass = computed(() => ({
  'pill-run':  status.value.state === 'running',
  'pill-err':  status.value.state === 'error',
  'pill-stop': status.value.state === 'stopped',
}))

const statusLabel = computed(() =>
  ({ running: '运行中', error: '错误', stopped: '已停止' }[status.value.state] || status.value.state)
)

const runtimeInfo = computed(() => {
  const s = status.value
  const r = {}
  if (s.pid)        r['PID']    = s.pid
  if (s.proxyMode)  r['透明代理'] = s.proxyMode
  if (s.routeMode)  r['路由']   = s.routeMode
  if (s.configMode) r['模式']   = s.configMode === 'node' ? '节点' : '上传'
  r['局域网'] = s.lanProxy ? 'on' : 'off'
  r['IPv6']   = s.ipv6    ? 'on' : 'off'
  if (s.ports) {
    r['DNS 端口']   = s.ports.dns
    r['Mixed 端口'] = s.ports.mixed
    if (s.proxyMode === 'tproxy')   r['TProxy 端口']   = s.ports.tproxy
    if (s.proxyMode === 'redirect') r['Redirect 端口'] = s.ports.redirect
  }
  return r
})

const proxyModes = [
  { value: 'tproxy',       icon: '⬡', label: 'tproxy',       desc: 'TCP + UDP' },
  { value: 'redirect',     icon: '⬢', label: 'redirect',     desc: 'TCP only' },
  { value: 'tun',          icon: '⬣', label: 'tun',          desc: 'Virtual NIC' },
  { value: 'system_proxy', icon: '⬟', label: 'system proxy', desc: 'Env/gsettings' },
]
const routeModes = [
  { value: 'whitelist', icon: '◐', label: '绕过大陆',    desc: '国内直连，其余代理' },
  { value: 'gfwlist',   icon: '◑', label: '仅代理 GFW', desc: '被墙域名代理，其余直连' },
  { value: 'global',    icon: '●', label: '全局代理',   desc: '所有流量走代理' },
]

async function api(method, path, body) {
  const opts = { method, headers: {} }
  if (body) { opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(body) }
  const res = await fetch('/api' + path, opts)
  const data = await res.json()
  if (!res.ok) throw new Error(data.error || res.statusText)
  return data
}

function logClass(line) {
  const l = line.toLowerCase()
  if (l.includes('error') || l.includes('fatal')) return 'l-err'
  if (l.includes('warn'))  return 'l-warn'
  if (l.includes('info'))  return 'l-info'
  return ''
}

async function loadNodes() {
  try { nodes.value = await api('GET', '/nodes') } catch {}
}

async function deleteNode(id) {
  try {
    await api('DELETE', '/nodes/' + id)
    nodes.value = nodes.value.filter(n => n.id !== id)
    if (selectedNodeId.value === id) selectedNodeId.value = ''
  } catch (e) { errorMsg.value = e.message }
}

async function doImport() {
  importErrors.value = []
  try {
    const res = await api('POST', '/nodes/import', { text: importText.value })
    nodes.value = [...nodes.value, ...(res.nodes || [])]
    importErrors.value = res.errors || []
    if (res.imported > 0) {
      importText.value = ''
      if (!importErrors.value.length) showImport.value = false
    }
  } catch (e) { importErrors.value = [e.message] }
}

function closeImport() { showImport.value = false; importErrors.value = [] }

async function uploadFile(file) {
  errorMsg.value = ''
  const fd = new FormData(); fd.append('config', file)
  try {
    const res = await fetch('/api/config', { method: 'POST', body: fd })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error)
    uploadInfo.value = data
  } catch (e) { errorMsg.value = e.message }
}
function onFileChange(e) { if (e.target.files[0]) uploadFile(e.target.files[0]) }
function onDrop(e) { isDragging.value = false; if (e.dataTransfer.files[0]) uploadFile(e.dataTransfer.files[0]) }

async function startCore() {
  errorMsg.value = ''
  try {
    await api('POST', '/start', {
      configMode: configMode.value,
      proxyMode:  proxyMode.value,
      lanProxy:   lanProxy.value,
      ipv6:       ipv6.value,
      blockAds:   blockAds.value,
      routeMode:  routeMode.value,
      nodeId:     selectedNodeId.value,
    })
    await pollStatus()
    startSSE()
  } catch (e) { errorMsg.value = e.message }
}

async function stopCore() {
  try { await api('POST', '/stop'); stopSSE(); await pollStatus() }
  catch (e) { errorMsg.value = e.message }
}

async function pollStatus() {
  try {
    status.value = await api('GET', '/status')
    if (status.value.error) errorMsg.value = status.value.error
  } catch {}
}

function startSSE() {
  if (sseSource) sseSource.close()
  sseSource = new EventSource('/api/logs')
  sseSource.onmessage = e => {
    logs.value.push(e.data)
    if (logs.value.length > 2000) logs.value.splice(0, 500)
    if (autoScroll.value) nextTick(() => {
      if (logEl.value) logEl.value.scrollTop = logEl.value.scrollHeight
    })
  }
}
function stopSSE() { if (sseSource) { sseSource.close(); sseSource = null } }
function onLogScroll() {
  if (!logEl.value) return
  const el = logEl.value
  autoScroll.value = el.scrollTop + el.clientHeight >= el.scrollHeight - 20
}

onMounted(async () => {
  loadProxy()
  await Promise.all([loadNodes(), pollStatus()])
  try { uploadInfo.value = await api('GET', '/config/info') } catch {}
  configMode.value     = status.value.configMode  || 'node'
  proxyMode.value      = status.value.proxyMode   || 'tproxy'
  routeMode.value      = status.value.routeMode   || 'whitelist'
  lanProxy.value       = status.value.lanProxy    || false
  ipv6.value           = status.value.ipv6        || false
  blockAds.value       = status.value.blockAds    || false
  if (status.value.nodeId) selectedNodeId.value = status.value.nodeId
  if (isRunning.value) startSSE()
  pollTimer = setInterval(pollStatus, 10000)
  if (logEl.value) logEl.value.addEventListener('scroll', onLogScroll)
  // Pre-fetch sing-box version for settings page
  fetchSbVersion()
})
onUnmounted(() => { stopSSE(); clearInterval(pollTimer) })
</script>

<style>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

:root {
  --bg:       #f4f6f8;
  --surface:  #ffffff;
  --border:   #e2e6ea;
  --border2:  #cdd3da;
  --text:     #1a1d23;
  --text2:    #5a6472;
  --text3:    #9ba5b0;
  --accent:   #00b37a;
  --accent-h: #00956a;
  --accent-bg:#e6f7f2;
  --blue:     #2563eb;
  --blue-bg:  #eff4ff;
  --red:      #e53e3e;
  --red-bg:   #fff5f5;
  --warn:     #d97706;
  --warn-bg:  #fffbeb;
  --radius:   8px;
  --mono:     'JetBrains Mono', 'Fira Code', 'IBM Plex Mono', monospace;
  --sans:     'Inter', 'PingFang SC', 'Helvetica Neue', sans-serif;
}

body {
  background: var(--bg);
  color: var(--text);
  font-family: var(--sans);
  font-size: 14px;
  min-height: 100vh;
  -webkit-font-smoothing: antialiased;
}

.app { display: flex; flex-direction: column; height: 100vh; overflow: hidden; }

/* topbar */
.topbar {
  flex-shrink: 0; display: flex; align-items: center;
  height: 52px; padding: 0 20px;
  background: var(--surface); border-bottom: 1px solid var(--border); gap: 0;
}
.brand { display: flex; align-items: center; gap: 8px; margin-right: 32px; }
.brand-icon { color: var(--accent); font-size: 20px; line-height: 1; }
.brand-name { font-size: 17px; font-weight: 700; letter-spacing: .03em; }
.brand-sub  {
  font-family: var(--mono); font-size: 10px; color: var(--text3);
  letter-spacing: .15em; text-transform: uppercase; margin-top: 1px;
}
.tab-nav { display: flex; align-items: center; gap: 2px; flex: 1; }
.tab-btn {
  padding: 6px 20px; background: transparent; border: none; border-radius: 6px;
  font-family: var(--sans); font-size: 14px; font-weight: 500; color: var(--text3);
  cursor: pointer; transition: all .15s;
}
.tab-btn:hover { color: var(--text2); background: var(--bg); }
.tab-btn.active { color: var(--accent); font-weight: 700; background: var(--accent-bg); }

.topbar-right { margin-left: auto; display: flex; align-items: center; gap: 10px; }

.settings-btn {
  width: 32px; height: 32px; border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  background: transparent; border: none; color: var(--text3);
  cursor: pointer; transition: all .15s;
}
.settings-btn:hover { background: var(--bg); color: var(--text2); }
.settings-btn.active { background: var(--accent-bg); color: var(--accent); }

.status-pill {
  display: flex; align-items: center; gap: 6px; padding: 4px 12px;
  border-radius: 20px; font-size: 12px; font-weight: 600; border: 1.5px solid;
}
.status-dot  { width: 7px; height: 7px; border-radius: 50%; }
.pill-stop   { color: var(--text3); border-color: var(--border2); }
.pill-stop .status-dot { background: var(--text3); }
.pill-run    { color: var(--accent); border-color: var(--accent); background: var(--accent-bg); }
.pill-run .status-dot  { background: var(--accent); animation: blink 1.4s infinite; }
.pill-err    { color: var(--red); border-color: var(--red); background: var(--red-bg); }
.pill-err .status-dot  { background: var(--red); }
@keyframes blink { 0%,100%{opacity:1} 50%{opacity:.3} }

.tab-content { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.tab-content .sidebar {
  flex: 1; overflow-y: auto; padding: 20px;
  display: flex; flex-direction: column; gap: 16px;
  max-width: 500px; width: 100%; margin: 0 auto;
}
.sidebar::-webkit-scrollbar { width: 4px; }
.sidebar::-webkit-scrollbar-thumb { background: var(--border2); border-radius: 2px; }

.section { display: flex; flex-direction: column; gap: 8px; }
.section-title {
  font-size: 11px; font-weight: 700; letter-spacing: .1em;
  text-transform: uppercase; color: var(--text3);
}
.section-title-row { display: flex; justify-content: space-between; align-items: center; }
.title-actions { display: flex; gap: 6px; align-items: center; }

.seg {
  display: grid; grid-template-columns: 1fr 1fr; gap: 0;
  border: 1.5px solid var(--border2); border-radius: var(--radius); overflow: hidden;
}
.seg-btn {
  padding: 9px; background: transparent; border: none; cursor: pointer;
  font-family: var(--sans); font-size: 13px; font-weight: 500; color: var(--text2);
  transition: all .15s;
}
.seg-btn:first-child { border-right: 1.5px solid var(--border2); }
.seg-btn:hover:not(:disabled) { background: var(--bg); color: var(--text); }
.seg-btn.on { background: var(--accent-bg); color: var(--accent); font-weight: 700; }
.seg-btn:disabled { opacity: .45; cursor: not-allowed; }

.icon-btn {
  display: flex; align-items: center; gap: 4px;
  padding: 4px 10px; background: var(--accent); color: #fff;
  border: none; border-radius: 6px; font-size: 12px; font-weight: 600;
  cursor: pointer; transition: background .15s;
}
.icon-btn:hover:not(:disabled) { background: var(--accent-h); }
.icon-btn:disabled { opacity: .5; cursor: not-allowed; }
.icon-btn.secondary {
  background: none; border: 1px solid var(--border2); color: var(--text3); font-weight: 500;
}
.icon-btn.secondary:hover:not(:disabled) { border-color: var(--accent); color: var(--accent); background: var(--accent-bg); }
.icon-btn.loading { background: none; border: 1px solid var(--accent); color: var(--accent); font-weight: 500; }

.update-msg {
  font-size: 11px; padding: 5px 8px; border-radius: 4px;
  font-family: var(--mono); line-height: 1.4; margin-top: 2px;
}
.msg-ok   { background: var(--accent-bg); color: var(--accent); }
.msg-warn { background: #fffbeb; color: var(--warn); }
.msg-err  { background: var(--red-bg); color: var(--red); }

.empty-tip { font-size: 12px; color: var(--text3); text-align: center; padding: 14px 0; }
.node-list { display: flex; flex-direction: column; gap: 5px; max-height: 200px; overflow-y: auto; }
.node-list::-webkit-scrollbar { width: 3px; }
.node-list::-webkit-scrollbar-thumb { background: var(--border2); border-radius: 2px; }

.node-row {
  display: flex; align-items: center; gap: 9px; padding: 9px 10px;
  border: 1.5px solid var(--border); border-radius: var(--radius);
  cursor: pointer; transition: all .15s; background: var(--bg);
}
.node-row:hover { border-color: var(--border2); background: var(--surface); }
.node-row.active { border-color: var(--accent); background: var(--accent-bg); }

.proto-badge {
  font-family: var(--mono); font-size: 9px; font-weight: 700;
  padding: 2px 6px; border-radius: 4px; flex-shrink: 0; text-transform: uppercase;
}
.p-vmess     { background: #e6f7f2; color: #00856a; }
.p-vless     { background: #eff4ff; color: #2563eb; }
.p-trojan    { background: #fffbeb; color: #d97706; }
.p-ss        { background: #f5f3ff; color: #7c3aed; }
.p-tuic      { background: #ecfdf5; color: #059669; }
.p-hysteria2 { background: #fff1f2; color: #e11d48; }

.node-meta { display: flex; flex-direction: column; flex: 1; min-width: 0; }
.node-name { font-size: 13px; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.node-addr { font-family: var(--mono); font-size: 10px; color: var(--text3); }

.del-btn {
  background: none; border: none; color: var(--text3); cursor: pointer;
  padding: 2px 5px; font-size: 12px; border-radius: 4px; transition: all .15s;
}
.del-btn:hover:not(:disabled) { color: var(--red); background: var(--red-bg); }
.del-btn:disabled { opacity: .35; cursor: not-allowed; }

.route-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; }
.route-btn {
  display: flex; flex-direction: column; align-items: center; gap: 3px;
  padding: 10px 6px; border: 1.5px solid var(--border); border-radius: var(--radius);
  background: var(--bg); cursor: pointer; transition: all .15s; color: var(--text2);
}
.route-btn:hover:not(:disabled) { border-color: var(--border2); color: var(--text); background: var(--surface); }
.route-btn.on { border-color: var(--accent); color: var(--accent); background: var(--accent-bg); }
.route-btn:disabled { opacity: .45; cursor: not-allowed; }
.route-icon { font-size: 18px; line-height: 1; }
.route-name { font-size: 12px; font-weight: 700; }
.route-desc { font-size: 10px; opacity: .7; text-align: center; line-height: 1.3; }

.dropzone {
  border: 2px dashed var(--border2); border-radius: var(--radius); padding: 20px;
  display: flex; flex-direction: column; align-items: center; gap: 6px;
  cursor: pointer; transition: all .2s; background: var(--bg); color: var(--text2);
}
.dropzone:hover, .dropzone.over { border-color: var(--accent); background: var(--accent-bg); color: var(--accent); }
.dropzone.loaded { border-style: solid; border-color: var(--accent); background: var(--accent-bg); color: var(--accent); }
.dz-icon { font-size: 24px; }
.dz-text { font-size: 13px; font-weight: 500; }

.ib-list { display: flex; flex-direction: column; gap: 4px; }
.ib-row {
  display: flex; align-items: center; gap: 8px; padding: 5px 9px;
  border: 1px solid var(--border); border-radius: 6px; background: var(--bg);
  font-family: var(--mono); font-size: 11px;
}
.ib-row.match { border-color: var(--accent); background: var(--accent-bg); }
.ib-type { color: var(--blue); font-weight: 700; min-width: 60px; }
.ib-tag  { color: var(--text2); flex: 1; }
.ib-port { color: var(--accent); font-weight: 700; }

.proxy-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
.proxy-btn {
  display: flex; flex-direction: column; align-items: flex-start; gap: 2px;
  padding: 10px 12px; border: 1.5px solid var(--border); border-radius: var(--radius);
  background: var(--bg); cursor: pointer; transition: all .15s; color: var(--text2); text-align: left;
}
.proxy-btn:hover:not(:disabled) { border-color: var(--border2); color: var(--text); background: var(--surface); }
.proxy-btn.on { border-color: var(--accent); color: var(--accent); background: var(--accent-bg); }
.proxy-btn:disabled { opacity: .45; cursor: not-allowed; }
.proxy-icon { font-size: 16px; line-height: 1; margin-bottom: 2px; }
.proxy-name { font-family: var(--mono); font-size: 11px; font-weight: 700; }
.proxy-desc { font-size: 10px; opacity: .7; }

.toggle-group { display: flex; flex-direction: column; gap: 10px; }
.toggle-row { display: flex; align-items: center; gap: 12px; cursor: pointer; }
.toggle-row.disabled { opacity: .45; pointer-events: none; }
.toggle-track {
  width: 44px; height: 24px; border-radius: 12px;
  background: var(--border2); position: relative; transition: background .2s; flex-shrink: 0; cursor: pointer;
}
.toggle-track.on { background: var(--accent); }
.toggle-thumb {
  position: absolute; top: 3px; left: 3px; width: 18px; height: 18px; border-radius: 50%;
  background: #fff; box-shadow: 0 1px 3px rgba(0,0,0,.2); transition: transform .2s;
}
.toggle-track.on .toggle-thumb { transform: translateX(20px); }
.toggle-labels { display: flex; flex-direction: column; gap: 1px; }
.toggle-name { font-size: 13px; font-weight: 600; color: var(--text); }
.toggle-hint { font-size: 11px; color: var(--text3); }

.action-section { flex-direction: row; }
.btn-start, .btn-stop {
  flex: 1; padding: 12px; font-family: var(--sans); font-size: 15px; font-weight: 700;
  border: none; border-radius: var(--radius); cursor: pointer; transition: all .15s;
  display: flex; align-items: center; justify-content: center; gap: 8px;
}
.btn-start { background: var(--accent); color: #fff; }
.btn-start:hover:not(:disabled) { background: var(--accent-h); transform: translateY(-1px); }
.btn-start:disabled { opacity: .4; cursor: not-allowed; transform: none; }
.btn-stop  { background: var(--red-bg); color: var(--red); border: 1.5px solid var(--red); }
.btn-stop:hover { background: #fed7d7; }

.error-bar {
  padding: 9px 12px; background: var(--red-bg);
  border: 1px solid #feb2b2; border-radius: var(--radius);
  color: var(--red); font-size: 12px; line-height: 1.5;
}

.info-grid {
  display: grid; grid-template-columns: auto 1fr; gap: 1px;
  background: var(--border); border-radius: var(--radius); overflow: hidden;
}
.info-k, .info-v { padding: 6px 10px; background: var(--bg); font-size: 12px; }
.info-k {
  font-family: var(--mono); color: var(--text3); font-size: 11px;
  text-transform: uppercase; letter-spacing: .05em;
}
.info-v { font-family: var(--mono); color: var(--blue); font-weight: 600; }

/* Settings */
.settings-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.flavor-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
.flavor-btn {
  display: flex; flex-direction: column; align-items: flex-start; gap: 2px;
  padding: 10px 12px; border: 1.5px solid var(--border); border-radius: var(--radius);
  background: var(--bg); cursor: pointer; transition: all .15s; color: var(--text2); text-align: left;
}
.flavor-btn:hover { border-color: var(--border2); color: var(--text); background: var(--surface); }
.flavor-btn.on { border-color: var(--accent); color: var(--accent); background: var(--accent-bg); }
.flavor-name { font-size: 13px; font-weight: 700; }
.flavor-desc { font-family: var(--mono); font-size: 10px; opacity: .7; }
.version-selector { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-bottom: 4px; }
.ver-opt { display: flex; align-items: center; gap: 5px; font-size: 13px; cursor: pointer; user-select: none; padding: 3px 0; }
.ver-opt .ver-radio { width: 13px; height: 13px; border-radius: 50%; border: 2px solid var(--border2); display: inline-flex; align-items: center; justify-content: center; transition: border-color .15s; }
.ver-opt.on .ver-radio { border-color: var(--accent); background: var(--accent); box-shadow: inset 0 0 0 2px var(--surface); }
.ver-input { flex: 1; min-width: 120px; font-size: 12px; padding: 4px 8px; }
.settings-hint { font-size: 12px; color: var(--text3); line-height: 1.5; }
.input-row { display: flex; gap: 8px; }
.text-input {
  flex: 1; padding: 7px 10px; border: 1.5px solid var(--border2); border-radius: var(--radius);
  font-family: var(--mono); font-size: 12px; color: var(--text); background: var(--bg);
  outline: none; transition: border-color .15s;
}
.text-input:focus { border-color: var(--accent); }
.proxy-presets { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; margin-top: 2px; }
.preset-label { font-size: 11px; color: var(--text3); }
.preset-btn {
  font-family: var(--mono); font-size: 10px; padding: 2px 8px;
  border: 1px solid var(--border2); border-radius: 4px;
  background: var(--bg); color: var(--text2); cursor: pointer; transition: all .15s;
}
.preset-btn:hover { border-color: var(--accent); color: var(--accent); background: var(--accent-bg); }

.rules-detail {
  display: flex; flex-direction: column; gap: 2px;
  max-height: 160px; overflow-y: auto; margin-top: 4px;
  border: 1px solid var(--border); border-radius: var(--radius); padding: 4px;
}
.rules-detail::-webkit-scrollbar { width: 3px; }
.rules-detail::-webkit-scrollbar-thumb { background: var(--border2); border-radius: 2px; }
.rules-row {
  display: flex; justify-content: space-between; gap: 8px;
  font-family: var(--mono); font-size: 10px; padding: 3px 6px; border-radius: 3px;
  color: var(--text2);
}
.rules-row.err { color: var(--red); background: var(--red-bg); }
.rules-file { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.rules-mirror { color: var(--text3); flex-shrink: 0; }
.rules-row.err .rules-mirror { color: var(--red); }

/* log pane */
.logpane { background: var(--bg); }
.log-inner {
  max-width: 900px; width: 100%; margin: 0 auto;
  display: flex; flex-direction: column; flex: 1; overflow: hidden;
  background: var(--surface); border-left: 1px solid var(--border); border-right: 1px solid var(--border);
}
.log-topbar {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 16px; border-bottom: 1px solid var(--border);
  flex-shrink: 0; background: var(--surface);
}
.log-live { font-family: var(--mono); font-size: 10px; color: var(--text3); letter-spacing: .1em; margin-right: auto; }
.log-live.live { color: var(--accent); animation: blink 1.4s infinite; }
.log-clear {
  background: none; border: 1px solid var(--border2); color: var(--text3);
  padding: 3px 10px; border-radius: 4px; cursor: pointer; font-size: 11px;
  font-family: var(--mono); transition: all .15s;
}
.log-clear:hover { border-color: var(--border2); color: var(--text); }
.log-body {
  flex: 1; overflow-y: auto; padding: 12px 16px;
  font-family: var(--mono); font-size: 12px; line-height: 1.9; background: var(--surface);
}
.log-body::-webkit-scrollbar { width: 4px; }
.log-body::-webkit-scrollbar-thumb { background: var(--border2); border-radius: 2px; }
.log-empty { color: var(--text3); text-align: center; padding: 60px 0; }
.log-line  { display: flex; gap: 8px; }
.log-arr   { color: var(--text3); user-select: none; flex-shrink: 0; }
.log-txt   { color: var(--text2); word-break: break-all; }
.l-err  { color: var(--red) !important; }
.l-warn { color: var(--warn) !important; }
.l-info { color: var(--text) !important; }
.log-foot {
  flex-shrink: 0; padding: 6px 16px; border-top: 1px solid var(--border);
  font-family: var(--mono); font-size: 10px; color: var(--text3); background: var(--surface);
}

/* modal */
.mask {
  position: fixed; inset: 0; background: rgba(0,0,0,.45); z-index: 100;
  display: flex; align-items: center; justify-content: center; padding: 16px;
}
.modal {
  background: var(--surface); border-radius: 12px; width: 100%; max-width: 500px;
  box-shadow: 0 20px 60px rgba(0,0,0,.25); overflow: hidden; display: flex; flex-direction: column;
}
.modal-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 16px 20px; border-bottom: 1px solid var(--border);
  font-size: 15px; font-weight: 700;
}
.modal-x {
  background: none; border: none; color: var(--text3); cursor: pointer;
  font-size: 16px; padding: 2px 6px; border-radius: 4px; transition: all .15s;
}
.modal-x:hover { background: var(--bg); color: var(--text); }
.modal-body { padding: 16px 20px; display: flex; flex-direction: column; gap: 10px; }
.modal-hint { font-size: 12px; color: var(--text3); }
.import-ta {
  width: 100%; height: 160px; resize: vertical; padding: 10px;
  border: 1.5px solid var(--border2); border-radius: var(--radius);
  font-family: var(--mono); font-size: 12px; color: var(--text);
  background: var(--bg); outline: none; line-height: 1.7;
}
.import-ta:focus { border-color: var(--accent); }
.import-errs { display: flex; flex-direction: column; gap: 3px; max-height: 80px; overflow-y: auto; }
.import-err  { font-family: var(--mono); font-size: 11px; color: var(--warn); }
.modal-foot {
  display: flex; justify-content: flex-end; gap: 8px;
  padding: 14px 20px; border-top: 1px solid var(--border);
}
.btn-cancel {
  padding: 8px 18px; background: var(--bg); border: 1.5px solid var(--border2);
  color: var(--text2); border-radius: var(--radius); cursor: pointer; font-family: var(--sans);
  transition: all .15s; font-size: 13px;
}
.btn-cancel:hover { border-color: var(--border2); color: var(--text); background: var(--surface); }
.btn-ok {
  padding: 8px 22px; background: var(--accent); color: #fff;
  border: none; border-radius: var(--radius); cursor: pointer;
  font-family: var(--sans); font-weight: 700; font-size: 13px; transition: all .15s;
}
.btn-ok:hover:not(:disabled) { background: var(--accent-h); }
.btn-ok:disabled { opacity: .4; cursor: not-allowed; }

@media (max-width: 768px) {
  .brand-sub { display: none; }
  .tab-content .sidebar { padding: 16px; }
}
</style>
