<template>
  <div class="page">
    <div class="topbar">
      <span class="topbar-title">设置</span>
    </div>
    <div class="page" style="display:flex;flex-direction:column;gap:16px">

      <!-- ── sing-box 核心 ─────────────────────────────────────────── -->
      <div class="card">
        <div class="card-title">sing-box 核心</div>
        <div class="info-table" style="margin-bottom:12px">
          <span class="info-k">已安装版本</span>
          <span class="info-v">{{ sbInfo.version || '未安装' }}</span>
          <span class="info-k">架构</span>
          <span class="info-v">{{ sbInfo.arch || '—' }}</span>
          <span class="info-k">系统</span>
          <span class="info-v">{{ sbInfo.osName || '—' }} / {{ sbInfo.libc || '—' }}</span>
        </div>

        <div class="grid-2 gap-3" style="margin-bottom:10px">
          <div class="field">
            <label class="field-label">构建版本</label>
            <div class="seg">
              <button class="seg-btn" :class="{ on: sbFlavor==='official' }"
                @click="sbFlavor='official'">官方版</button>
              <button class="seg-btn" :class="{ on: sbFlavor==='ref1nd' }"
                @click="sbFlavor='ref1nd'">reF1nd 版</button>
            </div>
            <div class="field-hint">
              官方版：SagerNet/sing-box<br>
              reF1nd 版：含额外补丁
            </div>
          </div>
          <div class="field">
            <label class="field-label">版本号</label>
            <div class="seg" style="margin-bottom:6px">
              <button class="seg-btn" :class="{ on: sbVerMode==='latest' }"
                @click="sbVerMode='latest'">最新</button>
              <button class="seg-btn" :class="{ on: sbVerMode==='custom' }"
                @click="sbVerMode='custom'">指定</button>
            </div>
            <input v-if="sbVerMode==='custom'" class="input input-mono"
              v-model="sbVerInput" placeholder="例如 1.13.2" />
          </div>
        </div>

        <div class="flex gap-2">
          <button class="btn btn-ghost" @click="checkVersion" :disabled="sbChecking">
            {{ sbChecking ? '检测中…' : '↺ 检测版本' }}
          </button>
          <button class="btn btn-primary" @click="installSb" :disabled="sbInstalling">
            {{ sbInstalling ? '下载中…' : '↓ 下载/更新核心' }}
          </button>
        </div>
        <div v-if="sbMsg" class="alert mt-2" :class="sbMsgClass">{{ sbMsg }}</div>
      </div>

      <!-- ── 代理模式细化 ───────────────────────────────────────────── -->
      <div class="card">
        <div class="card-title">代理模式细化</div>
        <div class="field-hint" style="margin-bottom:12px">
          此处设置将覆盖仪表盘快速选择的代理模式，提供更精细的 TCP/UDP 控制。
        </div>
        <div class="grid-2 gap-3">
          <div class="field">
            <label class="field-label">TCP 代理模式</label>
            <div class="seg" style="flex-direction:column;border-radius:var(--radius);overflow:hidden">
              <button v-for="m in tcpModeOpts" :key="m.v"
                class="seg-btn" :class="{ on: tcpMode===m.v }"
                style="border-right:none;border-bottom:1px solid var(--border2);text-align:left;padding:8px 12px"
                @click="tcpMode=m.v">
                <span style="font-weight:700">{{ m.l }}</span>
                <span style="font-size:10px;color:var(--text3);margin-left:6px">{{ m.desc }}</span>
              </button>
            </div>
          </div>
          <div class="field">
            <label class="field-label">UDP 代理模式</label>
            <div class="seg" style="flex-direction:column;border-radius:var(--radius);overflow:hidden">
              <button v-for="m in udpModeOpts" :key="m.v"
                class="seg-btn" :class="{ on: udpMode===m.v }"
                style="border-right:none;border-bottom:1px solid var(--border2);text-align:left;padding:8px 12px"
                @click="udpMode=m.v">
                <span style="font-weight:700">{{ m.l }}</span>
                <span style="font-size:10px;color:var(--text3);margin-left:6px">{{ m.desc }}</span>
              </button>
            </div>
          </div>
        </div>
        <div class="alert alert-info mt-3 text-xs">
          当前组合模式：<strong>{{ resolvedProxyMode }}</strong>
        </div>
        <div style="display:flex;flex-direction:column;gap:8px;margin-top:12px">
          <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px">
            <div class="toggle" :class="{ on: lanProxy }" @click="lanProxy=!lanProxy"></div>
            <span>局域网代理</span>
          </label>
          <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px">
            <div class="toggle" :class="{ on: ipv6 }" @click="ipv6=!ipv6"></div>
            <span>IPv6 支持</span>
          </label>
        </div>
        <button class="btn btn-ghost btn-sm mt-2" @click="saveProxyMode">保存偏好</button>
        <div v-if="proxyModeMsg" class="alert alert-success mt-2 text-xs">{{ proxyModeMsg }}</div>
      </div>

      <!-- ── 规则集更新 ─────────────────────────────────────────────── -->
      <div class="card">
        <div class="card-title">规则集更新</div>
        <div class="field" style="margin-bottom:10px">
          <label class="field-label">GitHub 代理加速</label>
          <div class="field-row">
            <input class="input input-mono" v-model="ghProxy"
              placeholder="https://ghfast.top/" />
            <button class="btn btn-ghost btn-sm" @click="saveProxy">保存</button>
          </div>
          <div class="field-hint">
            用于下载规则集和 sing-box 核心，留空自动尝试内置镜像
          </div>
          <div class="flex gap-2 mt-2" style="flex-wrap:wrap">
            <span class="field-hint">预设：</span>
            <button v-for="p in proxyPresets" :key="p"
              class="btn btn-ghost btn-sm" style="font-family:var(--mono);font-size:10px"
              @click="ghProxy=p">{{ p }}</button>
          </div>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-primary" @click="updateRules" :disabled="updatingRules">
            {{ updatingRules ? '更新中…' : '↻ 更新规则集' }}
          </button>
        </div>
        <div v-if="rulesMsg" class="alert mt-2" :class="rulesMsgClass">{{ rulesMsg }}</div>
        <div v-if="rulesResults.length" style="margin-top:8px;display:flex;flex-direction:column;gap:2px">
          <div v-for="r in rulesResults" :key="r.file"
            class="flex gap-2 items-center text-xs py-1"
            style="border-bottom:1px solid var(--border)">
            <span :class="r.error ? 'text-red' : 'text-green'">{{ r.error ? '✕' : '✓' }}</span>
            <span class="monospace flex-1">{{ r.file }}</span>
            <span class="text-muted">{{ r.error || r.mirror }}</span>
          </div>
        </div>
      </div>

      <!-- ── Inbound 端口配置 ──────────────────────────────────────── -->
      <div class="card">
        <div class="card-title">Inbound 端口配置</div>
        <div class="field-hint" style="margin-bottom:12px">
          各 inbound 监听端口及 TUN 网卡名称。保存后下次启动核心时生效，影响全部配置模式。
        </div>
        <div class="grid-2 gap-3" style="margin-bottom:10px">
          <div class="field">
            <label class="field-label">DNS 端口（dns-in）</label>
            <input class="input input-mono" type="number" v-model.number="ib.dnsPort" placeholder="5356" />
          </div>
          <div class="field">
            <label class="field-label">混合代理端口（mixed-in）</label>
            <input class="input input-mono" type="number" v-model.number="ib.mixedPort" placeholder="2081" />
          </div>
          <div class="field">
            <label class="field-label">Redirect 端口（redirect-in）</label>
            <input class="input input-mono" type="number" v-model.number="ib.redirectPort" placeholder="7892" />
          </div>
          <div class="field">
            <label class="field-label">TProxy 端口（tproxy-in）</label>
            <input class="input input-mono" type="number" v-model.number="ib.tproxyPort" placeholder="7893" />
          </div>
          <div class="field">
            <label class="field-label">TUN 网卡名称</label>
            <input class="input input-mono" v-model="ib.tunInterface" placeholder="singa" />
          </div>
          <div class="field">
            <label class="field-label">TUN 地址（每行一个）</label>
            <textarea class="textarea input-mono" v-model="ib.tunAddressText" rows="2"
              placeholder="172.31.0.1/30&#10;fdfe:dcba:9876::1/126"></textarea>
          </div>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-ghost btn-sm" @click="saveSingaSettings">保存</button>
          <span v-if="singaMsg" class="text-xs" :class="singaMsg.startsWith('✓') ? 'text-green':'text-red'"
            style="align-self:center">{{ singaMsg }}</span>
        </div>
      </div>

      <!-- ── Experimental ──────────────────────────────────────────────── -->
      <div class="card">
        <div class="card-title">Experimental</div>
        <div class="field-hint" style="margin-bottom:12px">
          覆盖全部配置模式的 experimental 块（cache_file + clash_api）。
        </div>

        <div style="margin-bottom:14px">
          <div class="field-label" style="margin-bottom:6px">缓存文件</div>
          <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px;margin-bottom:8px">
            <div class="toggle" :class="{ on: exp.cacheEnabled }" @click="exp.cacheEnabled=!exp.cacheEnabled"></div>
            <span>启用缓存</span>
          </label>
          <div class="field" v-if="exp.cacheEnabled">
            <label class="field-label">缓存路径</label>
            <input class="input input-mono" v-model="exp.cachePath" placeholder="cache.db" />
          </div>
        </div>

        <div style="margin-bottom:10px">
          <div class="field-label" style="margin-bottom:6px">Clash API</div>
          <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px;margin-bottom:8px">
            <div class="toggle" :class="{ on: exp.clashAPIEnabled }" @click="exp.clashAPIEnabled=!exp.clashAPIEnabled"></div>
            <span>启用 Clash API</span>
          </label>
          <div v-if="exp.clashAPIEnabled" class="grid-2 gap-3">
            <div class="field">
              <label class="field-label">监听地址</label>
              <input class="input input-mono" v-model="exp.clashAPIListen" placeholder="0.0.0.0:9090" />
            </div>
            <div class="field">
              <label class="field-label">UI 路径</label>
              <input class="input input-mono" v-model="exp.clashAPIUI" placeholder="ui" />
            </div>
            <div class="field">
              <label class="field-label">UI 下载地址</label>
              <input class="input input-mono" v-model="exp.clashAPIUIURL"
                placeholder="https://fastly.jsdelivr.net/gh/..." />
            </div>
            <div class="field">
              <label class="field-label">下载出站（outbound tag）</label>
              <input class="input input-mono" v-model="exp.clashAPIDetour" placeholder="direct" />
            </div>
            <div class="field">
              <label class="field-label">默认模式</label>
              <div class="seg">
                <button v-for="m in clashModes" :key="m.v"
                  class="seg-btn" :class="{ on: exp.clashAPIMode===m.v }"
                  @click="exp.clashAPIMode=m.v">{{ m.l }}</button>
              </div>
            </div>
          </div>
        </div>

        <div class="flex gap-2">
          <button class="btn btn-ghost btn-sm" @click="saveSingaSettings">保存</button>
          <span v-if="singaMsg" class="text-xs" :class="singaMsg.startsWith('✓') ? 'text-green':'text-red'"
            style="align-self:center">{{ singaMsg }}</span>
        </div>
      </div>

      <!-- ── 日志 ──────────────────────────────────────────────────────── -->
      <div class="card">
        <div class="card-title">日志</div>
        <div class="field-hint" style="margin-bottom:12px">
          覆盖全部配置模式的 log 块。
        </div>
        <label class="flex items-center gap-2" style="cursor:pointer;font-size:13px;margin-bottom:10px">
          <div class="toggle" :class="{ on: !logCfg.disabled }" @click="logCfg.disabled=!logCfg.disabled"></div>
          <span>启用日志</span>
        </label>
        <div v-if="!logCfg.disabled" class="field" style="margin-bottom:10px">
          <label class="field-label">日志等级</label>
          <div class="seg">
            <button v-for="l in logLevels" :key="l"
              class="seg-btn" :class="{ on: logCfg.level===l }"
              @click="logCfg.level=l">{{ l }}</button>
          </div>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-ghost btn-sm" @click="saveSingaSettings">保存</button>
          <span v-if="singaMsg" class="text-xs" :class="singaMsg.startsWith('✓') ? 'text-green':'text-red'"
            style="align-self:center">{{ singaMsg }}</span>
        </div>
      </div>

      <!-- ── 局域网 IP 过滤 ─────────────────────────────────────────── -->
      <div class="card">
        <div class="card-title">局域网 IP 过滤</div>
        <div class="field-hint" style="margin-bottom:12px">
          仅在开启局域网代理时生效。
        </div>
        <div class="mode-grid" style="margin-bottom:10px">
          <div v-for="m in ipfModes" :key="m.v"
            class="mode-card" :class="{ on: ipfMode===m.v }"
            @click="ipfMode=m.v">
            <div class="mode-card-icon">{{ m.icon }}</div>
            <div class="mode-card-name">{{ m.name }}</div>
            <div class="mode-card-desc">{{ m.desc }}</div>
          </div>
        </div>
        <div class="field" style="margin-bottom:10px">
          <label class="field-label">IP 列表（空格或换行分隔，支持 CIDR）</label>
          <textarea class="textarea" v-model="ipfIPs" rows="3"
            :disabled="ipfMode==='off'"
            placeholder="192.168.1.0/24 10.0.0.100"></textarea>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-primary" @click="saveIPFilter">保存</button>
          <span v-if="ipfMsg" class="text-xs" :class="ipfMsg.startsWith('✓') ? 'text-green':'text-red'"
            style="align-self:center">{{ ipfMsg }}</span>
        </div>
      </div>

      <!-- ── 关于 ─────────────────────────────────────────────────── -->
      <div class="card">
        <div class="card-title">关于</div>
        <div class="info-table">
          <span class="info-k">singa 版本</span><span class="info-v">v2.0</span>
          <span class="info-k">项目主页</span>
          <span class="info-v">
            <a href="https://github.com/singa" target="_blank" style="color:var(--accent)">
              github.com/singa
            </a>
          </span>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'

// ── sing-box ──────────────────────────────────────────────────────────────
const sbInfo      = ref({})
const sbFlavor    = ref('official')
const sbVerMode   = ref('latest')
const sbVerInput  = ref('')
const sbChecking  = ref(false)
const sbInstalling = ref(false)
const sbMsg       = ref('')
const sbMsgClass  = ref('')

async function checkVersion() {
  sbChecking.value = true; sbMsg.value = ''
  try { sbInfo.value = await api('GET', '/singbox/version') }
  catch (e) { sbMsg.value = '✕ ' + e.message; sbMsgClass.value = 'alert-error' }
  finally { sbChecking.value = false }
}

async function installSb() {
  sbInstalling.value = true; sbMsg.value = ''
  const ver = sbVerMode.value === 'custom' ? sbVerInput.value.trim() : 'latest'
  try {
    const r = await api('POST', '/singbox/install', {
      proxy: ghProxy.value, flavor: sbFlavor.value, version: ver || 'latest',
    })
    sbMsg.value = `✓ 安装成功：${r.version}`; sbMsgClass.value = 'alert-success'
    await checkVersion()
  } catch (e) { sbMsg.value = '✕ ' + e.message; sbMsgClass.value = 'alert-error' }
  finally { sbInstalling.value = false }
}

// ── Proxy mode ────────────────────────────────────────────────────────────
const tcpMode  = ref('off')
const udpMode  = ref('off')
const lanProxy = ref(false)
const ipv6     = ref(false)

const tcpModeOpts = [
  { v: 'off',    l: '禁用',   desc: '不透明代理 TCP' },
  { v: 'redir',  l: 'redir',  desc: 'iptables REDIRECT（旧方案）' },
  { v: 'tproxy', l: 'tproxy', desc: 'iptables TPROXY（推荐 Linux）' },
  { v: 'tun',    l: 'tun',    desc: 'TUN 虚拟网卡（跨平台）' },
]
const udpModeOpts = [
  { v: 'off',    l: '禁用',   desc: '不透明代理 UDP' },
  { v: 'tproxy', l: 'tproxy', desc: 'iptables TPROXY UDP' },
  { v: 'tun',    l: 'tun',    desc: 'TUN 虚拟网卡' },
]

const resolvedProxyMode = computed(() => {
  if (tcpMode.value === 'tun' || udpMode.value === 'tun') return 'tun'
  if (tcpMode.value === 'tproxy' || udpMode.value === 'tproxy') return 'tproxy'
  if (tcpMode.value === 'redir') return 'redirect'
  return 'system_proxy（仅系统代理）'
})

const proxyModeMsg = ref('')
async function saveProxyMode() {
  try {
    await api('POST', '/proxy-settings', {
      tcpMode:  tcpMode.value,
      udpMode:  udpMode.value,
      lanProxy: lanProxy.value,
      ipv6:     ipv6.value,
    })
    proxyModeMsg.value = '✓ 已保存'
  } catch (e) {
    proxyModeMsg.value = '✕ ' + e.message
  }
  setTimeout(() => { proxyModeMsg.value = '' }, 2000)
}

// ── Rules ─────────────────────────────────────────────────────────────────
const ghProxy      = ref(localStorage.getItem('ghProxy') || '')
const updatingRules = ref(false)
const rulesMsg     = ref('')
const rulesMsgClass = ref('')
const rulesResults  = ref([])
const proxyPresets  = [
  'https://ghfast.top/',
  'https://gh-proxy.com/',
  'https://ghproxy.it/',
  'https://gh.ddlc.top/',
]

function saveProxy() {
  localStorage.setItem('ghProxy', ghProxy.value)
}

async function updateRules() {
  updatingRules.value = true; rulesMsg.value = ''; rulesResults.value = []
  try {
    const r = await api('POST', '/update-rules', { proxy: ghProxy.value })
    rulesResults.value = r.results || []
    if (r.failed === 0) {
      rulesMsg.value = `✓ 全部 ${r.total} 个规则集更新成功`
      rulesMsgClass.value = 'alert-success'
    } else if (r.failed < r.total) {
      rulesMsg.value = `⚠ ${r.total - r.failed}/${r.total} 个成功`
      rulesMsgClass.value = 'alert-warn'
    } else {
      rulesMsg.value = '✕ 全部更新失败'
      rulesMsgClass.value = 'alert-error'
    }
  } catch (e) {
    rulesMsg.value = '✕ ' + e.message; rulesMsgClass.value = 'alert-error'
  } finally { updatingRules.value = false }
}

// ── Singa Settings (inbound / experimental / log) ─────────────────────────
const singaMsg = ref('')

const ib = ref({
  dnsPort: 5356, mixedPort: 2081, redirectPort: 7892, tproxyPort: 7893,
  tunInterface: 'singa', tunAddressText: '172.31.0.1/30\nfdfe:dcba:9876::1/126',
})

const exp = ref({
  cacheEnabled: true, cachePath: 'cache.db',
  clashAPIEnabled: false, clashAPIListen: '', clashAPIUI: 'ui',
  clashAPIUIURL: '', clashAPIDetour: 'direct', clashAPIMode: 'rule',
})

const logCfg = ref({ disabled: true, level: 'warn' })

const clashModes = [{ v: 'rule', l: '规则' }, { v: 'global', l: '全局' }, { v: 'direct', l: '直连' }]
const logLevels  = ['trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic']

async function loadSingaSettings() {
  try {
    const r = await api('GET', '/singa-settings')
    if (r.inbound) {
      ib.value.dnsPort      = r.inbound.dnsPort      || 5356
      ib.value.mixedPort    = r.inbound.mixedPort    || 2081
      ib.value.redirectPort = r.inbound.redirectPort || 7892
      ib.value.tproxyPort   = r.inbound.tproxyPort   || 7893
      ib.value.tunInterface = r.inbound.tunInterface || 'singa'
      ib.value.tunAddressText = (r.inbound.tunAddress || ['172.31.0.1/30','fdfe:dcba:9876::1/126']).join('\n')
    }
    if (r.experimental) {
      exp.value.cacheEnabled    = r.experimental.cacheEnabled !== false
      exp.value.cachePath       = r.experimental.cachePath    || 'cache.db'
      exp.value.clashAPIEnabled = !!r.experimental.clashAPIEnabled
      exp.value.clashAPIListen  = r.experimental.clashAPIListen  || ''
      exp.value.clashAPIUI      = r.experimental.clashAPIUI      || 'ui'
      exp.value.clashAPIUIURL   = r.experimental.clashAPIUIURL   || ''
      exp.value.clashAPIDetour  = r.experimental.clashAPIDetour  || 'direct'
      exp.value.clashAPIMode    = r.experimental.clashAPIMode    || 'rule'
    }
    if (r.log) {
      logCfg.value.disabled = r.log.disabled !== false
      logCfg.value.level    = r.log.level || 'warn'
    }
  } catch {}
}

async function saveSingaSettings() {
  try {
    await api('POST', '/singa-settings', {
      inbound: {
        dnsPort:      ib.value.dnsPort,
        mixedPort:    ib.value.mixedPort,
        redirectPort: ib.value.redirectPort,
        tproxyPort:   ib.value.tproxyPort,
        tunInterface: ib.value.tunInterface,
        tunAddress:   ib.value.tunAddressText.split(/[\s,]+/).filter(Boolean),
      },
      experimental: {
        cacheEnabled:    exp.value.cacheEnabled,
        cachePath:       exp.value.cachePath,
        clashAPIEnabled: exp.value.clashAPIEnabled,
        clashAPIListen:  exp.value.clashAPIListen,
        clashAPIUI:      exp.value.clashAPIUI,
        clashAPIUIURL:   exp.value.clashAPIUIURL,
        clashAPIDetour:  exp.value.clashAPIDetour,
        clashAPIMode:    exp.value.clashAPIMode,
      },
      log: {
        disabled: logCfg.value.disabled,
        level:    logCfg.value.level,
      },
    })
    singaMsg.value = '✓ 已保存'
  } catch (e) {
    singaMsg.value = '✕ ' + e.message
  }
  setTimeout(() => { singaMsg.value = '' }, 2000)
}

// ── IP Filter ─────────────────────────────────────────────────────────────
const ipfMode = ref('off')
const ipfIPs  = ref('')
const ipfMsg  = ref('')
const ipfModes = [
  { v: 'off',       icon: '○', name: '关闭',  desc: '不过滤任何 IP' },
  { v: 'blacklist', icon: '✕', name: '黑名单', desc: '列表内不代理' },
  { v: 'whitelist', icon: '✓', name: '白名单', desc: '仅列表内代理' },
]

async function loadIPFilter() {
  try {
    const r = await api('GET', '/ip-filter')
    ipfMode.value = r.mode || 'off'
    ipfIPs.value  = r.ips  || ''
  } catch {}
}

async function saveIPFilter() {
  try {
    await api('POST', '/ip-filter', { mode: ipfMode.value, ips: ipfIPs.value })
    ipfMsg.value = '✓ 已保存'
    setTimeout(() => { ipfMsg.value = '' }, 2000)
  } catch (e) { ipfMsg.value = '✕ ' + e.message }
}

onMounted(() => {
  checkVersion()
  loadIPFilter()
  loadSingaSettings()
  api('GET', '/proxy-settings').then(r => {
    tcpMode.value  = r.tcpMode  || 'off'
    udpMode.value  = r.udpMode  || 'off'
    lanProxy.value = !!r.lanProxy
    ipv6.value     = !!r.ipv6
  }).catch(() => {})
})
</script>

<style scoped>
@media (max-width: 640px) {
  /* grid-2 already goes single column via global style */
}
</style>
