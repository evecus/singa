<template>
  <div class="app">
    <!-- Header -->
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
      <!-- Left column: controls -->
      <section class="panel panel-controls">

        <!-- Config upload -->
        <div class="block">
          <div class="block-label">CONFIG</div>
          <div
            class="drop-zone"
            :class="{ 'drop-zone--over': isDragging, 'drop-zone--loaded': configInfo }"
            @dragover.prevent="isDragging = true"
            @dragleave="isDragging = false"
            @drop.prevent="onDrop"
            @click="$refs.fileInput.click()"
          >
            <input ref="fileInput" type="file" accept=".json" style="display:none" @change="onFileChange" />
            <template v-if="!configInfo">
              <span class="drop-icon">⬆</span>
              <span class="drop-text">上传 config.json</span>
              <span class="drop-hint">点击或拖拽</span>
            </template>
            <template v-else>
              <span class="drop-icon loaded-icon">✓</span>
              <span class="drop-text">config.json 已加载</span>
              <span class="drop-hint">点击重新上传</span>
            </template>
          </div>

          <!-- Inbound summary -->
          <div v-if="configInfo && configInfo.inbounds.length" class="inbound-list">
            <div
              v-for="ib in configInfo.inbounds"
              :key="ib.tag || ib.type"
              class="inbound-item"
              :class="{ 'inbound-item--match': ib.type === selectedMode }"
            >
              <span class="inbound-type">{{ ib.type }}</span>
              <span class="inbound-tag">{{ ib.tag || '—' }}</span>
              <span class="inbound-port" v-if="ib.port">:{{ ib.port }}</span>
            </div>
          </div>
        </div>

        <!-- Proxy mode -->
        <div class="block">
          <div class="block-label">透明代理模式</div>
          <div class="mode-grid">
            <button
              v-for="m in modes"
              :key="m.value"
              class="mode-btn"
              :class="{ 'mode-btn--active': selectedMode === m.value }"
              :disabled="status.state === 'running'"
              @click="selectedMode = m.value"
            >
              <span class="mode-icon">{{ m.icon }}</span>
              <span class="mode-name">{{ m.label }}</span>
              <span class="mode-desc">{{ m.desc }}</span>
            </button>
          </div>
        </div>

        <!-- LAN proxy toggle -->
        <div class="block">
          <div class="block-label">局域网代理</div>
          <label class="toggle-row" :class="{ 'toggle-row--disabled': status.state === 'running' }">
            <div class="toggle" :class="{ 'toggle--on': lanProxy }" @click="status.state !== 'running' && (lanProxy = !lanProxy)">
              <div class="toggle-knob"></div>
            </div>
            <div class="toggle-info">
              <span class="toggle-label">{{ lanProxy ? '已启用' : '已禁用' }}</span>
              <span class="toggle-hint">{{ lanProxy ? '代理局域网设备流量，开启 ip_forward' : '仅代理本机流量' }}</span>
            </div>
          </label>
        </div>

        <!-- IPv6 toggle -->
        <div class="block">
          <div class="block-label">IPv6 代理</div>
          <label class="toggle-row" :class="{ 'toggle-row--disabled': status.state === 'running' }">
            <div class="toggle" :class="{ 'toggle--on': ipv6 }" @click="status.state !== 'running' && (ipv6 = !ipv6)">
              <div class="toggle-knob"></div>
            </div>
            <div class="toggle-info">
              <span class="toggle-label">{{ ipv6 ? '已启用' : '已禁用' }}</span>
              <span class="toggle-hint">{{ ipv6 ? '同时代理 IPv6 流量，下发 ip6 规则' : '仅代理 IPv4 流量' }}</span>
            </div>
          </label>
        </div>

        <!-- Action buttons -->
        <div class="block block-actions">
          <button
            v-if="status.state !== 'running'"
            class="btn btn-start"
            :disabled="!configInfo || uploading"
            @click="startCore"
          >
            <span>▶</span> 启动
          </button>
          <button
            v-else
            class="btn btn-stop"
            @click="stopCore"
          >
            <span>■</span> 停止
          </button>
        </div>

        <!-- Error message -->
        <div v-if="errorMsg" class="error-bar">
          <span>⚠</span> {{ errorMsg }}
        </div>

        <!-- Runtime info -->
        <div v-if="status.state === 'running'" class="info-table">
          <div class="info-row">
            <span class="info-k">PID</span>
            <span class="info-v mono">{{ status.pid }}</span>
          </div>
          <div class="info-row">
            <span class="info-k">模式</span>
            <span class="info-v mono">{{ status.mode }}</span>
          </div>
          <div class="info-row">
            <span class="info-k">端口</span>
            <span class="info-v mono">{{ status.port || '—' }}</span>
          </div>
          <div class="info-row">
            <span class="info-k">局域网</span>
            <span class="info-v mono">{{ status.lanProxy ? 'on' : 'off' }}</span>
          </div>
          <div class="info-row">
            <span class="info-k">IPv6</span>
            <span class="info-v mono">{{ status.ipv6 ? 'on' : 'off' }}</span>
          </div>
        </div>
      </section>

      <!-- Right column: log terminal -->
      <section class="panel panel-logs">
        <div class="block-label log-label">
          <span>LOGS</span>
          <span class="log-indicator" :class="{ 'log-indicator--live': status.state === 'running' }">
            {{ status.state === 'running' ? '● LIVE' : '○ IDLE' }}
          </span>
        </div>
        <div class="log-terminal" ref="logEl">
          <div v-if="logs.length === 0" class="log-empty">等待核心启动…</div>
          <div v-for="(line, i) in logs" :key="i" class="log-line">
            <span class="log-prefix">›</span>
            <span class="log-text" :class="logClass(line)">{{ line }}</span>
          </div>
        </div>
        <div class="log-footer">
          <span class="mono">{{ logs.length }} lines</span>
          <button class="btn-clear" @click="logs = []">清空</button>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'

const modes = [
  { value: 'tproxy',       icon: '⬡', label: 'tproxy',       desc: 'TCP + UDP' },
  { value: 'redirect',     icon: '⬢', label: 'redirect',     desc: 'TCP only' },
  { value: 'tun',          icon: '⬣', label: 'tun',          desc: 'Virtual NIC' },
  { value: 'system_proxy', icon: '⬟', label: 'system proxy', desc: 'Env / gsettings' },
]

const selectedMode = ref('tproxy')
const lanProxy = ref(false)
const ipv6 = ref(false)
const configInfo = ref(null)
const status = ref({ state: 'stopped', mode: '', port: 0, lanProxy: false, pid: 0, error: '' })
const logs = ref([])
const errorMsg = ref('')
const uploading = ref(false)
const isDragging = ref(false)
const logEl = ref(null)
const autoScroll = ref(true)

let sseSource = null
let pollTimer = null

const statusClass = computed(() => ({
  'badge-running': status.value.state === 'running',
  'badge-error':   status.value.state === 'error',
  'badge-stopped': status.value.state === 'stopped',
}))

const statusLabel = computed(() => ({
  running: '运行中',
  error:   '错误',
  stopped: '已停止',
}[status.value.state] || status.value.state))

function logClass(line) {
  const l = line.toLowerCase()
  if (l.includes('error') || l.includes('fatal')) return 'log-error'
  if (l.includes('warn')) return 'log-warn'
  if (l.includes('info')) return 'log-info'
  return ''
}

// ── API helpers ────────────────────────────────────────────────────
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

// ── Config upload ──────────────────────────────────────────────────
async function uploadFile(file) {
  uploading.value = true
  errorMsg.value = ''
  try {
    const fd = new FormData()
    fd.append('config', file)
    const res = await fetch('/api/config', { method: 'POST', body: fd })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error)
    configInfo.value = data
  } catch (e) {
    errorMsg.value = e.message
  } finally {
    uploading.value = false
  }
}

function onFileChange(e) {
  const f = e.target.files[0]
  if (f) uploadFile(f)
}

function onDrop(e) {
  isDragging.value = false
  const f = e.dataTransfer.files[0]
  if (f) uploadFile(f)
}

// ── Start / Stop ───────────────────────────────────────────────────
async function startCore() {
  errorMsg.value = ''
  try {
    await api('POST', '/start', { mode: selectedMode.value, lanProxy: lanProxy.value, ipv6: ipv6.value })
    await pollStatus()
    startSSE()
  } catch (e) {
    errorMsg.value = e.message
  }
}

async function stopCore() {
  errorMsg.value = ''
  try {
    await api('POST', '/stop')
    stopSSE()
    await pollStatus()
  } catch (e) {
    errorMsg.value = e.message
  }
}

// ── Status polling ─────────────────────────────────────────────────
async function pollStatus() {
  try {
    const s = await api('GET', '/status')
    status.value = s
    if (s.error) errorMsg.value = s.error
  } catch {}
}

function startPolling() {
  pollTimer = setInterval(pollStatus, 2000)
}

// ── SSE log stream ─────────────────────────────────────────────────
function startSSE() {
  if (sseSource) sseSource.close()
  sseSource = new EventSource('/api/logs')
  sseSource.onmessage = (e) => {
    logs.value.push(e.data)
    if (logs.value.length > 2000) logs.value.splice(0, 500)
    if (autoScroll.value) {
      nextTick(() => {
        if (logEl.value) logEl.value.scrollTop = logEl.value.scrollHeight
      })
    }
  }
  sseSource.onerror = () => {
    // Reconnect is automatic for EventSource
  }
}

function stopSSE() {
  if (sseSource) {
    sseSource.close()
    sseSource = null
  }
}

// ── Load initial config info ───────────────────────────────────────
async function loadConfigInfo() {
  try {
    const data = await api('GET', '/config/info')
    configInfo.value = data
  } catch {}
}

// ── Auto-scroll detection ──────────────────────────────────────────
function onLogScroll() {
  if (!logEl.value) return
  const el = logEl.value
  autoScroll.value = el.scrollTop + el.clientHeight >= el.scrollHeight - 20
}

onMounted(async () => {
  await loadConfigInfo()
  await pollStatus()
  if (status.value.state === 'running') startSSE()
  startPolling()
  if (logEl.value) logEl.value.addEventListener('scroll', onLogScroll)
})

onUnmounted(() => {
  stopSSE()
  clearInterval(pollTimer)
})

// Sync selectedMode and toggles to running state on load
watch(() => status.value.mode, (m) => { if (m) selectedMode.value = m }, { immediate: true })
watch(() => status.value.lanProxy, (v) => { if (status.value.state === 'running') lanProxy.value = v }, { immediate: true })
watch(() => status.value.ipv6, (v) => { if (status.value.state === 'running') ipv6.value = v }, { immediate: true })
</script>

<style>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

:root {
  --bg:       #0d0f12;
  --surface:  #13161b;
  --surface2: #1a1e26;
  --border:   #252a35;
  --border2:  #2f3545;
  --text:     #c8cdd8;
  --text2:    #6b7280;
  --text3:    #3d4455;
  --accent:   #4fffb0;
  --accent2:  #00d4ff;
  --warn:     #f59e0b;
  --error:    #f87171;
  --running:  #4fffb0;
  --mono:     'IBM Plex Mono', monospace;
  --sans:     'Space Grotesk', sans-serif;
}

body {
  background: var(--bg);
  color: var(--text);
  font-family: var(--sans);
  font-size: 14px;
  min-height: 100vh;
  overflow-x: hidden;
}

/* Subtle grid background */
body::before {
  content: '';
  position: fixed;
  inset: 0;
  background-image:
    linear-gradient(var(--border) 1px, transparent 1px),
    linear-gradient(90deg, var(--border) 1px, transparent 1px);
  background-size: 40px 40px;
  opacity: 0.3;
  pointer-events: none;
  z-index: 0;
}

.app { position: relative; z-index: 1; min-height: 100vh; display: flex; flex-direction: column; }

/* ── Header ── */
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
  height: 56px;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 10;
}
.header-left { display: flex; align-items: center; gap: 10px; }
.logo-mark { color: var(--accent); font-size: 20px; line-height: 1; }
.logo-text { font-size: 18px; font-weight: 700; letter-spacing: 0.08em; color: #e8ecf4; }
.logo-sub { font-family: var(--mono); font-size: 11px; color: var(--text3); letter-spacing: 0.12em; text-transform: uppercase; margin-top: 1px; }

.status-badge {
  display: flex; align-items: center; gap: 7px;
  padding: 5px 12px;
  border-radius: 2px;
  font-family: var(--mono); font-size: 11px; font-weight: 600;
  letter-spacing: 0.1em; text-transform: uppercase;
  border: 1px solid;
  transition: all 0.3s;
}
.status-dot { width: 6px; height: 6px; border-radius: 50%; }
.badge-stopped { color: var(--text2); border-color: var(--border2); }
.badge-stopped .status-dot { background: var(--text3); }
.badge-running { color: var(--accent); border-color: var(--accent); background: rgba(79,255,176,0.06); }
.badge-running .status-dot { background: var(--accent); animation: pulse 1.5s infinite; }
.badge-error { color: var(--error); border-color: var(--error); }
.badge-error .status-dot { background: var(--error); }
@keyframes pulse { 0%,100%{ opacity:1; } 50%{ opacity:0.3; } }

/* ── Main layout ── */
.main {
  flex: 1;
  display: grid;
  grid-template-columns: 360px 1fr;
  gap: 1px;
  background: var(--border);
  min-height: calc(100vh - 56px);
}

.panel {
  background: var(--surface);
  padding: 28px 24px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* ── Block ── */
.block-label {
  font-family: var(--mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  color: var(--text3);
  text-transform: uppercase;
  margin-bottom: 12px;
}

/* ── Drop zone ── */
.drop-zone {
  border: 1.5px dashed var(--border2);
  border-radius: 4px;
  padding: 24px 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  transition: all 0.2s;
  background: var(--surface2);
}
.drop-zone:hover, .drop-zone--over { border-color: var(--accent); background: rgba(79,255,176,0.04); }
.drop-zone--loaded { border-style: solid; border-color: var(--accent); }
.drop-icon { font-size: 22px; color: var(--text3); transition: color 0.2s; }
.drop-zone:hover .drop-icon, .drop-zone--over .drop-icon { color: var(--accent); }
.loaded-icon { color: var(--accent) !important; }
.drop-text { font-size: 13px; font-weight: 500; color: var(--text); }
.drop-hint { font-size: 11px; color: var(--text3); font-family: var(--mono); }

/* ── Inbound list ── */
.inbound-list { margin-top: 10px; display: flex; flex-direction: column; gap: 4px; }
.inbound-item {
  display: flex; align-items: center; gap: 8px;
  padding: 6px 10px;
  background: var(--surface2);
  border: 1px solid var(--border);
  border-radius: 3px;
  font-family: var(--mono); font-size: 12px;
}
.inbound-item--match { border-color: var(--accent2); background: rgba(0,212,255,0.05); }
.inbound-type { color: var(--accent2); font-weight: 600; min-width: 70px; }
.inbound-tag { color: var(--text2); flex: 1; }
.inbound-port { color: var(--accent); font-weight: 600; }

/* ── Mode grid ── */
.mode-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.mode-btn {
  display: flex; flex-direction: column; align-items: flex-start;
  gap: 2px; padding: 12px 14px;
  background: var(--surface2);
  border: 1px solid var(--border);
  border-radius: 3px;
  cursor: pointer;
  text-align: left;
  transition: all 0.15s;
  color: var(--text2);
}
.mode-btn:hover:not(:disabled) { border-color: var(--border2); color: var(--text); }
.mode-btn--active { border-color: var(--accent) !important; color: var(--accent) !important; background: rgba(79,255,176,0.06) !important; }
.mode-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.mode-icon { font-size: 18px; line-height: 1; margin-bottom: 4px; }
.mode-name { font-family: var(--mono); font-size: 12px; font-weight: 600; }
.mode-desc { font-size: 10px; opacity: 0.7; }

/* ── Toggle ── */
.toggle-row { display: flex; align-items: center; gap: 14px; cursor: pointer; }
.toggle-row--disabled { opacity: 0.5; pointer-events: none; }
.toggle {
  width: 44px; height: 24px;
  background: var(--surface2);
  border: 1px solid var(--border2);
  border-radius: 12px;
  position: relative;
  transition: all 0.2s;
  flex-shrink: 0;
  cursor: pointer;
}
.toggle--on { background: rgba(79,255,176,0.15); border-color: var(--accent); }
.toggle-knob {
  position: absolute; top: 3px; left: 3px;
  width: 16px; height: 16px;
  background: var(--text3);
  border-radius: 50%;
  transition: all 0.2s;
}
.toggle--on .toggle-knob { transform: translateX(20px); background: var(--accent); }
.toggle-info { display: flex; flex-direction: column; gap: 2px; }
.toggle-label { font-size: 13px; font-weight: 500; color: var(--text); }
.toggle-hint { font-size: 11px; color: var(--text3); }

/* ── Action buttons ── */
.block-actions { flex-direction: row; gap: 10px; }
.btn {
  flex: 1; display: flex; align-items: center; justify-content: center; gap: 8px;
  padding: 12px 20px;
  font-family: var(--sans); font-size: 14px; font-weight: 700;
  letter-spacing: 0.05em;
  border: none; border-radius: 3px; cursor: pointer;
  transition: all 0.15s;
}
.btn-start {
  background: var(--accent);
  color: #0d0f12;
}
.btn-start:hover:not(:disabled) { background: #7fffcc; transform: translateY(-1px); }
.btn-start:disabled { opacity: 0.4; cursor: not-allowed; transform: none; }
.btn-stop {
  background: transparent;
  color: var(--error);
  border: 1.5px solid var(--error);
}
.btn-stop:hover { background: rgba(248,113,113,0.1); transform: translateY(-1px); }

/* ── Error bar ── */
.error-bar {
  padding: 10px 14px;
  background: rgba(248,113,113,0.08);
  border: 1px solid rgba(248,113,113,0.3);
  border-radius: 3px;
  color: var(--error);
  font-size: 12px;
  display: flex; align-items: flex-start; gap: 8px;
}

/* ── Info table ── */
.info-table { display: flex; flex-direction: column; gap: 1px; }
.info-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 7px 12px;
  background: var(--surface2);
}
.info-row:first-child { border-radius: 3px 3px 0 0; }
.info-row:last-child { border-radius: 0 0 3px 3px; }
.info-k { font-family: var(--mono); font-size: 11px; color: var(--text3); text-transform: uppercase; letter-spacing: 0.1em; }
.info-v { font-family: var(--mono); font-size: 12px; color: var(--accent2); }
.mono { font-family: var(--mono); }

/* ── Log panel ── */
.panel-logs { padding: 0; background: #0a0c0f; }
.log-label {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 0;
}
.log-indicator { font-family: var(--mono); font-size: 10px; color: var(--text3); }
.log-indicator--live { color: var(--accent); animation: blink 1.5s step-end infinite; }
@keyframes blink { 50% { opacity: 0.3; } }

.log-terminal {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  font-family: var(--mono);
  font-size: 12px;
  line-height: 1.8;
  min-height: 0;
  height: calc(100vh - 56px - 48px - 36px);
}
.log-terminal::-webkit-scrollbar { width: 4px; }
.log-terminal::-webkit-scrollbar-track { background: transparent; }
.log-terminal::-webkit-scrollbar-thumb { background: var(--border2); border-radius: 2px; }

.log-empty { color: var(--text3); padding: 40px 0; text-align: center; font-size: 13px; }
.log-line { display: flex; gap: 10px; }
.log-prefix { color: var(--text3); user-select: none; flex-shrink: 0; }
.log-text { color: var(--text2); word-break: break-all; }
.log-error { color: var(--error) !important; }
.log-warn { color: var(--warn) !important; }
.log-info { color: var(--text) !important; }

.log-footer {
  display: flex; justify-content: space-between; align-items: center;
  padding: 8px 20px;
  border-top: 1px solid var(--border);
  font-family: var(--mono); font-size: 11px; color: var(--text3);
}
.btn-clear {
  background: none; border: none; color: var(--text3);
  font-family: var(--mono); font-size: 11px;
  cursor: pointer; padding: 2px 6px;
  transition: color 0.15s;
}
.btn-clear:hover { color: var(--text); }

/* ── Responsive ── */
@media (max-width: 800px) {
  .main { grid-template-columns: 1fr; }
  .panel-logs { height: 400px; }
  .log-terminal { height: 300px; }
}
</style>
