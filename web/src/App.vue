<template>
  <div class="app">
    <header class="header">
      <div class="header-left">
        <span class="logo-mark">◈</span>
        <span class="logo-text">singa</span>
        <span class="logo-sub">proxy manager</span>
      </div>
      <div class="header-right">
        <div class="status-badge" :class="statusClass">
          <span class="status-dot"></span>
          <span>{{ statusLabel }}</span>
        </div>
      </div>
    </header>

    <main class="main">
      <!-- ── Left panel: controls ── -->
      <section class="panel panel-left">

        <!-- Config mode tabs -->
        <div class="block">
          <div class="block-label">配置模式</div>
          <div class="tab-row">
            <button class="tab-btn" :class="{active: configMode==='node'}"
              :disabled="isRunning" @click="configMode='node'">
              ⬡ 节点模式
            </button>
            <button class="tab-btn" :class="{active: configMode==='upload'}"
              :disabled="isRunning" @click="configMode='upload'">
              ⬢ 上传配置
            </button>
          </div>
        </div>

        <!-- ── NODE MODE ── -->
        <template v-if="configMode==='node'">

          <!-- Node list -->
          <div class="block">
            <div class="block-label-row">
              <span class="block-label">节点</span>
              <button class="btn-icon" @click="showImport=true" title="导入节点">＋</button>
            </div>
            <div v-if="nodes.length===0" class="empty-hint">暂无节点，点击 ＋ 导入</div>
            <div v-else class="node-list">
              <div v-for="n in nodes" :key="n.id"
                class="node-item" :class="{selected: selectedNodeId===n.id}"
                @click="selectedNodeId=n.id">
                <div class="node-left">
                  <span class="node-proto" :class="'proto-'+n.protocol">{{ n.protocol }}</span>
                  <div class="node-info">
                    <span class="node-name">{{ n.name || n.address }}</span>
                    <span class="node-addr">{{ n.address }}:{{ n.port }}</span>
                  </div>
                </div>
                <button class="btn-del" :disabled="isRunning"
                  @click.stop="deleteNode(n.id)">✕</button>
              </div>
            </div>
          </div>

          <!-- Route mode -->
          <div class="block">
            <div class="block-label">路由模式</div>
            <div class="route-grid">
              <button v-for="r in routeModes" :key="r.value"
                class="route-btn" :class="{active: routeMode===r.value}"
                :disabled="isRunning" @click="routeMode=r.value">
                <span class="route-icon">{{ r.icon }}</span>
                <span class="route-name">{{ r.label }}</span>
                <span class="route-desc">{{ r.desc }}</span>
              </button>
            </div>
          </div>

        </template>

        <!-- ── UPLOAD MODE ── -->
        <template v-else>
          <div class="block">
            <div class="block-label">配置文件</div>
            <div class="drop-zone"
              :class="{'drop-over': isDragging, loaded: uploadInfo}"
              @dragover.prevent="isDragging=true"
              @dragleave="isDragging=false"
              @drop.prevent="onDrop"
              @click="$refs.fileInput.click()">
              <input ref="fileInput" type="file" accept=".json" style="display:none" @change="onFileChange"/>
              <span class="drop-icon">{{ uploadInfo ? '✓' : '⬆' }}</span>
              <span class="drop-text">{{ uploadInfo ? 'config.json 已加载' : '上传 config.json' }}</span>
              <span class="drop-hint">点击或拖拽</span>
            </div>
            <div v-if="uploadInfo && uploadInfo.inbounds.length" class="inbound-list">
              <div v-for="ib in uploadInfo.inbounds" :key="ib.tag"
                class="inbound-item" :class="{match: ib.type===proxyMode}">
                <span class="ib-type">{{ ib.type }}</span>
                <span class="ib-tag">{{ ib.tag||'—' }}</span>
                <span class="ib-port" v-if="ib.port">:{{ ib.port }}</span>
              </div>
            </div>
          </div>
        </template>

        <!-- Proxy mode (both modes) -->
        <div class="block">
          <div class="block-label">透明代理模式</div>
          <div class="mode-grid">
            <button v-for="m in proxyModes" :key="m.value"
              class="mode-btn" :class="{active: proxyMode===m.value}"
              :disabled="isRunning" @click="proxyMode=m.value">
              <span class="mode-icon">{{ m.icon }}</span>
              <span class="mode-name">{{ m.label }}</span>
              <span class="mode-desc">{{ m.desc }}</span>
            </button>
          </div>
        </div>

        <!-- Toggles -->
        <div class="block toggles-block">
          <Toggle v-model="lanProxy" :disabled="isRunning"
            label="局域网代理"
            :hint="lanProxy ? '代理局域网设备，开启 ip_forward' : '仅代理本机流量'" />
          <Toggle v-model="ipv6" :disabled="isRunning"
            label="IPv6 代理"
            :hint="ipv6 ? '同时代理 IPv6 流量' : '仅代理 IPv4 流量'" />
        </div>

        <!-- Actions -->
        <div class="block block-actions">
          <button v-if="!isRunning" class="btn-start"
            :disabled="startDisabled" @click="startCore">
            ▶ 启动
          </button>
          <button v-else class="btn-stop" @click="stopCore">
            ■ 停止
          </button>
        </div>

        <div v-if="errorMsg" class="error-bar">⚠ {{ errorMsg }}</div>

        <!-- Runtime info -->
        <div v-if="isRunning" class="info-table">
          <div class="info-row" v-for="(v,k) in runtimeInfo" :key="k">
            <span class="info-k">{{ k }}</span>
            <span class="info-v">{{ v }}</span>
          </div>
        </div>

      </section>

      <!-- ── Right panel: log terminal ── -->
      <section class="panel panel-right">
        <div class="log-header">
          <span class="block-label" style="margin:0">LOGS</span>
          <span class="log-live" :class="{live: isRunning}">
            {{ isRunning ? '● LIVE' : '○ IDLE' }}
          </span>
        </div>
        <div class="log-terminal" ref="logEl" @scroll="onLogScroll">
          <div v-if="logs.length===0" class="log-empty">等待核心启动…</div>
          <div v-for="(line,i) in logs" :key="i" class="log-line">
            <span class="log-arrow">›</span>
            <span class="log-text" :class="logClass(line)">{{ line }}</span>
          </div>
        </div>
        <div class="log-footer">
          <span class="mono">{{ logs.length }} lines</span>
          <button class="btn-clear" @click="logs=[]">清空</button>
        </div>
      </section>
    </main>

    <!-- Import modal -->
    <div v-if="showImport" class="modal-overlay" @click.self="showImport=false">
      <div class="modal">
        <div class="modal-header">
          <span>导入节点</span>
          <button class="modal-close" @click="showImport=false">✕</button>
        </div>
        <div class="modal-body">
          <p class="modal-hint">每行一个分享链接，支持 vmess / vless / trojan / ss / tuic / hy2</p>
          <textarea v-model="importText" class="import-textarea"
            placeholder="vmess://...&#10;vless://...&#10;tuic://...&#10;hy2://..." />
          <div v-if="importErrors.length" class="import-errors">
            <div v-for="(e,i) in importErrors" :key="i" class="import-error-line">⚠ {{ e }}</div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-cancel" @click="showImport=false">取消</button>
          <button class="btn-import" :disabled="!importText.trim()" @click="doImport">
            导入
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'

// ── Sub-component: Toggle ──────────────────────────────────────────────────
const Toggle = {
  props: ['modelValue', 'disabled', 'label', 'hint'],
  emits: ['update:modelValue'],
  template: `
    <label class="toggle-row" :class="{disabled}">
      <div class="toggle" :class="{'toggle-on': modelValue}"
        @click="!disabled && $emit('update:modelValue', !modelValue)">
        <div class="toggle-knob"></div>
      </div>
      <div class="toggle-info">
        <span class="toggle-label">{{ label }}</span>
        <span class="toggle-hint">{{ hint }}</span>
      </div>
    </label>
  `
}

// ── State ──────────────────────────────────────────────────────────────────
const configMode  = ref('node')
const proxyMode   = ref('tproxy')
const routeMode   = ref('whitelist')
const lanProxy    = ref(false)
const ipv6        = ref(false)
const selectedNodeId = ref('')

const nodes       = ref([])
const uploadInfo  = ref(null)
const status      = ref({ state: 'stopped' })
const logs        = ref([])
const errorMsg    = ref('')
const isDragging  = ref(false)
const showImport  = ref(false)
const importText  = ref('')
const importErrors = ref([])
const logEl       = ref(null)
const autoScroll  = ref(true)

let sseSource = null
let pollTimer = null

// ── Computed ───────────────────────────────────────────────────────────────
const isRunning = computed(() => status.value.state === 'running')

const startDisabled = computed(() => {
  if (configMode.value === 'node' && !selectedNodeId.value) return true
  if (configMode.value === 'upload' && !uploadInfo.value) return true
  return false
})

const statusClass = computed(() => ({
  'badge-running': status.value.state === 'running',
  'badge-error':   status.value.state === 'error',
  'badge-stopped': status.value.state === 'stopped',
}))

const statusLabel = computed(() => ({
  running: '运行中', error: '错误', stopped: '已停止'
}[status.value.state] || status.value.state))

const runtimeInfo = computed(() => {
  const s = status.value
  const rows = {}
  if (s.pid)        rows['PID']    = s.pid
  if (s.proxyMode)  rows['透明代理'] = s.proxyMode
  if (s.routeMode)  rows['路由']   = s.routeMode
  if (s.configMode) rows['模式']   = s.configMode === 'node' ? '节点' : '上传'
  rows['局域网'] = s.lanProxy ? 'on' : 'off'
  rows['IPv6']   = s.ipv6    ? 'on' : 'off'
  if (s.ports) {
    rows['端口(DNS)']    = s.ports.dns
    rows['端口(Mixed)']  = s.ports.mixed
    if (s.proxyMode === 'tproxy')   rows['端口(TProxy)']   = s.ports.tproxy
    if (s.proxyMode === 'redirect') rows['端口(Redirect)'] = s.ports.redirect
  }
  return rows
})

// ── Static config ──────────────────────────────────────────────────────────
const proxyModes = [
  { value: 'tproxy',       icon: '⬡', label: 'tproxy',       desc: 'TCP + UDP' },
  { value: 'redirect',     icon: '⬢', label: 'redirect',     desc: 'TCP only' },
  { value: 'tun',          icon: '⬣', label: 'tun',          desc: 'Virtual NIC' },
  { value: 'system_proxy', icon: '⬟', label: 'system proxy', desc: 'Env/gsettings' },
]
const routeModes = [
  { value: 'whitelist', icon: '◐', label: '绕过大陆', desc: '国内直连，其余代理' },
  { value: 'gfwlist',   icon: '◑', label: '仅代理 GFW', desc: '被墙域名代理，其余直连' },
  { value: 'global',    icon: '●', label: '全局代理',  desc: '所有流量走代理' },
]

// ── Helpers ────────────────────────────────────────────────────────────────
async function api(method, path, body) {
  const opts = { method, headers: {} }
  if (body) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }
  const res = await fetch('/api' + path, opts)
  const data = await res.json()
  if (!res.ok) throw new Error(data.error || res.statusText)
  return data
}

function logClass(line) {
  const l = line.toLowerCase()
  if (l.includes('error') || l.includes('fatal')) return 'log-err'
  if (l.includes('warn'))  return 'log-warn'
  if (l.includes('info'))  return 'log-info'
  return ''
}

// ── Nodes ──────────────────────────────────────────────────────────────────
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

// ── Config upload ──────────────────────────────────────────────────────────
async function uploadFile(file) {
  errorMsg.value = ''
  const fd = new FormData()
  fd.append('config', file)
  try {
    const res = await fetch('/api/config', { method: 'POST', body: fd })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error)
    uploadInfo.value = data
  } catch (e) { errorMsg.value = e.message }
}
function onFileChange(e) { if (e.target.files[0]) uploadFile(e.target.files[0]) }
function onDrop(e) { isDragging.value = false; if (e.dataTransfer.files[0]) uploadFile(e.dataTransfer.files[0]) }

// ── Core control ───────────────────────────────────────────────────────────
async function startCore() {
  errorMsg.value = ''
  try {
    const body = {
      configMode: configMode.value,
      proxyMode:  proxyMode.value,
      lanProxy:   lanProxy.value,
      ipv6:       ipv6.value,
      routeMode:  routeMode.value,
      nodeId:     selectedNodeId.value,
    }
    await api('POST', '/start', body)
    await pollStatus()
    startSSE()
  } catch (e) { errorMsg.value = e.message }
}

async function stopCore() {
  try { await api('POST', '/stop'); stopSSE(); await pollStatus() }
  catch (e) { errorMsg.value = e.message }
}

// ── Status ─────────────────────────────────────────────────────────────────
async function pollStatus() {
  try {
    status.value = await api('GET', '/status')
    if (status.value.error) errorMsg.value = status.value.error
  } catch {}
}

// ── SSE ────────────────────────────────────────────────────────────────────
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

// ── Lifecycle ──────────────────────────────────────────────────────────────
onMounted(async () => {
  await Promise.all([loadNodes(), pollStatus()])
  try { uploadInfo.value = await api('GET', '/config/info') } catch {}
  if (isRunning.value) {
    configMode.value = status.value.configMode || 'node'
    proxyMode.value  = status.value.proxyMode  || 'tproxy'
    routeMode.value  = status.value.routeMode  || 'whitelist'
    lanProxy.value   = status.value.lanProxy   || false
    ipv6.value       = status.value.ipv6       || false
    startSSE()
  }
  pollTimer = setInterval(pollStatus, 2000)
  if (logEl.value) logEl.value.addEventListener('scroll', onLogScroll)
})
onUnmounted(() => { stopSSE(); clearInterval(pollTimer) })
</script>

<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}

:root{
  --bg:#0d0f12;--surface:#13161b;--surface2:#1a1e26;
  --border:#252a35;--border2:#2f3545;
  --text:#c8cdd8;--text2:#6b7280;--text3:#3d4455;
  --accent:#4fffb0;--accent2:#00d4ff;--warn:#f59e0b;--err:#f87171;
  --mono:'IBM Plex Mono',monospace;--sans:'Space Grotesk',sans-serif;
}

body{background:var(--bg);color:var(--text);font-family:var(--sans);font-size:14px;min-height:100vh}
body::before{
  content:'';position:fixed;inset:0;pointer-events:none;z-index:0;
  background-image:linear-gradient(var(--border) 1px,transparent 1px),
    linear-gradient(90deg,var(--border) 1px,transparent 1px);
  background-size:40px 40px;opacity:0.25;
}
.app{position:relative;z-index:1;min-height:100vh;display:flex;flex-direction:column}

/* ── Header ── */
.header{display:flex;align-items:center;justify-content:space-between;
  padding:0 28px;height:54px;background:var(--surface);
  border-bottom:1px solid var(--border);position:sticky;top:0;z-index:10}
.header-left{display:flex;align-items:center;gap:10px}
.logo-mark{color:var(--accent);font-size:20px}
.logo-text{font-size:18px;font-weight:700;letter-spacing:.08em;color:#e8ecf4}
.logo-sub{font-family:var(--mono);font-size:10px;color:var(--text3);letter-spacing:.14em;text-transform:uppercase}
.status-badge{display:flex;align-items:center;gap:7px;padding:4px 12px;
  border-radius:2px;font-family:var(--mono);font-size:11px;font-weight:600;
  letter-spacing:.1em;text-transform:uppercase;border:1px solid;transition:all .25s}
.status-dot{width:6px;height:6px;border-radius:50%}
.badge-stopped{color:var(--text2);border-color:var(--border2)}
.badge-stopped .status-dot{background:var(--text3)}
.badge-running{color:var(--accent);border-color:var(--accent);background:rgba(79,255,176,.06)}
.badge-running .status-dot{background:var(--accent);animation:pulse 1.5s infinite}
.badge-error{color:var(--err);border-color:var(--err)}
.badge-error .status-dot{background:var(--err)}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.3}}

/* ── Layout ── */
.main{flex:1;display:grid;grid-template-columns:380px 1fr;
  gap:1px;background:var(--border);min-height:calc(100vh - 54px)}
.panel{background:var(--surface);padding:22px 20px;
  display:flex;flex-direction:column;gap:20px}
.panel-right{padding:0;background:#0a0c0f}

/* ── Blocks ── */
.block-label{font-family:var(--mono);font-size:10px;font-weight:600;
  letter-spacing:.18em;color:var(--text3);text-transform:uppercase;margin-bottom:10px}
.block-label-row{display:flex;justify-content:space-between;align-items:center;margin-bottom:10px}
.block-label-row .block-label{margin-bottom:0}

/* ── Tabs ── */
.tab-row{display:grid;grid-template-columns:1fr 1fr;gap:6px}
.tab-btn{padding:9px;background:var(--surface2);border:1px solid var(--border);
  border-radius:3px;color:var(--text2);cursor:pointer;
  font-family:var(--sans);font-size:13px;font-weight:500;transition:all .15s}
.tab-btn:hover:not(:disabled){border-color:var(--border2);color:var(--text)}
.tab-btn.active{border-color:var(--accent);color:var(--accent);background:rgba(79,255,176,.06)}
.tab-btn:disabled{opacity:.5;cursor:not-allowed}

/* ── Node list ── */
.empty-hint{font-size:12px;color:var(--text3);text-align:center;padding:16px 0;font-family:var(--mono)}
.node-list{display:flex;flex-direction:column;gap:4px;max-height:220px;overflow-y:auto}
.node-list::-webkit-scrollbar{width:3px}
.node-list::-webkit-scrollbar-thumb{background:var(--border2);border-radius:2px}
.node-item{display:flex;align-items:center;justify-content:space-between;
  padding:8px 10px;background:var(--surface2);border:1px solid var(--border);
  border-radius:3px;cursor:pointer;transition:all .15s}
.node-item:hover{border-color:var(--border2)}
.node-item.selected{border-color:var(--accent2);background:rgba(0,212,255,.05)}
.node-left{display:flex;align-items:center;gap:8px;min-width:0}
.node-proto{font-family:var(--mono);font-size:10px;font-weight:700;
  padding:2px 6px;border-radius:2px;flex-shrink:0;text-transform:uppercase}
.proto-vmess{background:rgba(79,255,176,.15);color:var(--accent)}
.proto-vless{background:rgba(0,212,255,.15);color:var(--accent2)}
.proto-trojan{background:rgba(245,158,11,.15);color:var(--warn)}
.proto-ss{background:rgba(167,139,250,.15);color:#a78bfa}
.proto-tuic{background:rgba(52,211,153,.15);color:#34d399}
.proto-hysteria2{background:rgba(248,113,113,.15);color:var(--err)}
.node-info{display:flex;flex-direction:column;min-width:0}
.node-name{font-size:13px;font-weight:500;color:var(--text);
  white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:200px}
.node-addr{font-family:var(--mono);font-size:10px;color:var(--text3)}
.btn-del{background:none;border:none;color:var(--text3);cursor:pointer;
  font-size:12px;padding:2px 6px;transition:color .15s;flex-shrink:0}
.btn-del:hover:not(:disabled){color:var(--err)}
.btn-del:disabled{opacity:.4;cursor:not-allowed}
.btn-icon{background:none;border:1px solid var(--border2);color:var(--accent);
  cursor:pointer;padding:3px 10px;border-radius:2px;font-size:16px;line-height:1;
  transition:all .15s}
.btn-icon:hover{background:rgba(79,255,176,.08);border-color:var(--accent)}

/* ── Route grid ── */
.route-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:6px}
.route-btn{display:flex;flex-direction:column;align-items:center;gap:2px;
  padding:10px 6px;background:var(--surface2);border:1px solid var(--border);
  border-radius:3px;cursor:pointer;transition:all .15s;color:var(--text2)}
.route-btn:hover:not(:disabled){border-color:var(--border2);color:var(--text)}
.route-btn.active{border-color:var(--accent);color:var(--accent);background:rgba(79,255,176,.06)}
.route-btn:disabled{opacity:.5;cursor:not-allowed}
.route-icon{font-size:18px;line-height:1}
.route-name{font-family:var(--mono);font-size:11px;font-weight:600}
.route-desc{font-size:10px;opacity:.7;text-align:center}

/* ── Drop zone ── */
.drop-zone{border:1.5px dashed var(--border2);border-radius:4px;padding:20px;
  display:flex;flex-direction:column;align-items:center;gap:4px;
  cursor:pointer;transition:all .2s;background:var(--surface2)}
.drop-zone:hover,.drop-zone.drop-over{border-color:var(--accent);background:rgba(79,255,176,.04)}
.drop-zone.loaded{border-style:solid;border-color:var(--accent)}
.drop-icon{font-size:22px;color:var(--text3)}
.drop-zone.loaded .drop-icon{color:var(--accent)}
.drop-text{font-size:13px;font-weight:500}
.drop-hint{font-size:11px;color:var(--text3);font-family:var(--mono)}
.inbound-list{margin-top:8px;display:flex;flex-direction:column;gap:3px}
.inbound-item{display:flex;align-items:center;gap:8px;padding:5px 9px;
  background:var(--surface2);border:1px solid var(--border);border-radius:2px;
  font-family:var(--mono);font-size:11px}
.inbound-item.match{border-color:var(--accent2)}
.ib-type{color:var(--accent2);font-weight:600;min-width:64px}
.ib-tag{color:var(--text2);flex:1}
.ib-port{color:var(--accent);font-weight:600}

/* ── Proxy mode grid ── */
.mode-grid{display:grid;grid-template-columns:1fr 1fr;gap:6px}
.mode-btn{display:flex;flex-direction:column;align-items:flex-start;gap:2px;
  padding:10px 12px;background:var(--surface2);border:1px solid var(--border);
  border-radius:3px;cursor:pointer;transition:all .15s;color:var(--text2);text-align:left}
.mode-btn:hover:not(:disabled){border-color:var(--border2);color:var(--text)}
.mode-btn.active{border-color:var(--accent);color:var(--accent);background:rgba(79,255,176,.06)}
.mode-btn:disabled{opacity:.5;cursor:not-allowed}
.mode-icon{font-size:18px;line-height:1;margin-bottom:2px}
.mode-name{font-family:var(--mono);font-size:11px;font-weight:600}
.mode-desc{font-size:10px;opacity:.7}

/* ── Toggles ── */
.toggles-block{gap:12px}
.toggle-row{display:flex;align-items:center;gap:12px;cursor:pointer}
.toggle-row.disabled{opacity:.5;pointer-events:none}
.toggle{width:42px;height:22px;background:var(--surface2);
  border:1px solid var(--border2);border-radius:11px;position:relative;
  transition:all .2s;flex-shrink:0;cursor:pointer}
.toggle.toggle-on{background:rgba(79,255,176,.15);border-color:var(--accent)}
.toggle-knob{position:absolute;top:3px;left:3px;width:14px;height:14px;
  background:var(--text3);border-radius:50%;transition:all .2s}
.toggle.toggle-on .toggle-knob{transform:translateX(20px);background:var(--accent)}
.toggle-info{display:flex;flex-direction:column;gap:1px}
.toggle-label{font-size:13px;font-weight:500;color:var(--text)}
.toggle-hint{font-size:11px;color:var(--text3)}

/* ── Actions ── */
.block-actions{flex-direction:row}
.btn-start,.btn-stop{flex:1;display:flex;align-items:center;justify-content:center;
  gap:8px;padding:11px;font-family:var(--sans);font-size:14px;font-weight:700;
  letter-spacing:.04em;border:none;border-radius:3px;cursor:pointer;transition:all .15s}
.btn-start{background:var(--accent);color:#0d0f12}
.btn-start:hover:not(:disabled){background:#7fffcc;transform:translateY(-1px)}
.btn-start:disabled{opacity:.4;cursor:not-allowed;transform:none}
.btn-stop{background:transparent;color:var(--err);border:1.5px solid var(--err)}
.btn-stop:hover{background:rgba(248,113,113,.1);transform:translateY(-1px)}

/* ── Error / info ── */
.error-bar{padding:9px 12px;background:rgba(248,113,113,.08);
  border:1px solid rgba(248,113,113,.3);border-radius:3px;
  color:var(--err);font-size:12px}
.info-table{display:flex;flex-direction:column;gap:1px}
.info-row{display:flex;justify-content:space-between;align-items:center;
  padding:6px 10px;background:var(--surface2)}
.info-row:first-child{border-radius:3px 3px 0 0}
.info-row:last-child{border-radius:0 0 3px 3px}
.info-k{font-family:var(--mono);font-size:10px;color:var(--text3);text-transform:uppercase;letter-spacing:.1em}
.info-v{font-family:var(--mono);font-size:12px;color:var(--accent2)}
.mono{font-family:var(--mono)}

/* ── Log panel ── */
.log-header{display:flex;justify-content:space-between;align-items:center;
  padding:13px 18px;border-bottom:1px solid var(--border)}
.log-live{font-family:var(--mono);font-size:10px;color:var(--text3);letter-spacing:.1em}
.log-live.live{color:var(--accent);animation:blink 1.5s step-end infinite}
@keyframes blink{50%{opacity:.3}}
.log-terminal{flex:1;overflow-y:auto;padding:14px 18px;
  font-family:var(--mono);font-size:12px;line-height:1.8;
  height:calc(100vh - 54px - 44px - 36px)}
.log-terminal::-webkit-scrollbar{width:3px}
.log-terminal::-webkit-scrollbar-thumb{background:var(--border2);border-radius:2px}
.log-empty{color:var(--text3);text-align:center;padding:48px 0;font-size:13px}
.log-line{display:flex;gap:9px}
.log-arrow{color:var(--text3);user-select:none;flex-shrink:0}
.log-text{color:var(--text2);word-break:break-all}
.log-err{color:var(--err)!important}
.log-warn{color:var(--warn)!important}
.log-info{color:var(--text)!important}
.log-footer{display:flex;justify-content:space-between;align-items:center;
  padding:7px 18px;border-top:1px solid var(--border);
  font-family:var(--mono);font-size:11px;color:var(--text3)}
.btn-clear{background:none;border:none;color:var(--text3);
  font-family:var(--mono);font-size:11px;cursor:pointer;padding:2px 6px;transition:color .15s}
.btn-clear:hover{color:var(--text)}

/* ── Modal ── */
.modal-overlay{position:fixed;inset:0;background:rgba(0,0,0,.6);
  z-index:100;display:flex;align-items:center;justify-content:center}
.modal{background:var(--surface);border:1px solid var(--border2);border-radius:6px;
  width:520px;max-width:95vw;display:flex;flex-direction:column;overflow:hidden}
.modal-header{display:flex;justify-content:space-between;align-items:center;
  padding:16px 20px;border-bottom:1px solid var(--border);
  font-weight:600;font-size:15px;color:var(--text)}
.modal-close{background:none;border:none;color:var(--text3);cursor:pointer;
  font-size:16px;padding:2px 6px;transition:color .15s}
.modal-close:hover{color:var(--text)}
.modal-body{padding:16px 20px;display:flex;flex-direction:column;gap:12px}
.modal-hint{font-size:12px;color:var(--text3);font-family:var(--mono)}
.import-textarea{width:100%;height:180px;background:var(--surface2);
  border:1px solid var(--border2);border-radius:3px;
  color:var(--text);font-family:var(--mono);font-size:12px;
  padding:10px;resize:vertical;outline:none;line-height:1.7}
.import-textarea:focus{border-color:var(--accent)}
.import-errors{display:flex;flex-direction:column;gap:3px;max-height:80px;overflow-y:auto}
.import-error-line{font-family:var(--mono);font-size:11px;color:var(--warn)}
.modal-footer{display:flex;justify-content:flex-end;gap:8px;
  padding:14px 20px;border-top:1px solid var(--border)}
.btn-cancel{padding:8px 18px;background:var(--surface2);border:1px solid var(--border2);
  color:var(--text2);border-radius:3px;cursor:pointer;font-family:var(--sans);transition:all .15s}
.btn-cancel:hover{border-color:var(--text2);color:var(--text)}
.btn-import{padding:8px 22px;background:var(--accent);color:#0d0f12;
  border:none;border-radius:3px;cursor:pointer;font-family:var(--sans);
  font-weight:700;transition:all .15s}
.btn-import:hover:not(:disabled){background:#7fffcc}
.btn-import:disabled{opacity:.4;cursor:not-allowed}

@media(max-width:800px){
  .main{grid-template-columns:1fr}
  .panel-right{height:380px}
  .log-terminal{height:280px}
}
</style>
