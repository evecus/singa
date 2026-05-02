<template>
  <div class="page">
    <div class="topbar">
      <span class="topbar-title">规则集</span>
      <div class="topbar-right">
        <button class="btn btn-ghost btn-sm" @click="showHub = true">规则集中心</button>
        <button class="btn btn-ghost btn-sm" @click="showAdd = true">＋ 新增</button>
        <button class="btn btn-primary btn-sm"
          :disabled="rsStore.updating" @click="updateAll">
          {{ rsStore.updating ? '更新中…' : '↻ 更新全部' }}
        </button>
      </div>
    </div>
    <div class="page">

      <!-- view toggle -->
      <div class="flex items-center gap-2 mb-3">
        <div class="seg" style="width:120px">
          <button class="seg-btn" :class="{ on: view==='grid' }" @click="view='grid'">⊞ 网格</button>
          <button class="seg-btn" :class="{ on: view==='list' }" @click="view='list'">☰ 列表</button>
        </div>
        <span class="text-xs text-muted">共 {{ rsStore.local.length }} 个本地规则集</span>
      </div>

      <!-- Update results -->
      <div v-if="rsStore.results.length" class="card mb-3">
        <div class="card-title">上次更新结果</div>
        <div v-for="r in rsStore.results" :key="r.file"
          class="flex items-center gap-2 text-xs py-1 border-b" style="border-color:var(--border)">
          <span :class="r.error ? 'text-red' : 'text-green'">{{ r.error ? '✕' : '✓' }}</span>
          <span class="monospace flex-1">{{ r.file }}</span>
          <span class="text-muted">{{ r.error || r.mirror }}</span>
        </div>
      </div>

      <!-- Empty -->
      <div v-if="!rsStore.local.length" class="card">
        <div class="empty">
          暂无本地规则集<br>
          <button class="btn btn-primary mt-3" @click="updateAll">↓ 立即下载内置规则集</button>
        </div>
      </div>

      <!-- Grid view -->
      <div v-else-if="view === 'grid'" class="rs-grid">
        <div v-for="rs in rsStore.local" :key="rs.file" class="rs-card">
          <div class="rs-name">{{ rs.file }}</div>
          <div class="rs-meta">
            文件格式：{{ rs.format || 'binary' }}<br>
            更新时间：{{ fmtTime(rs.updatedAt) }}<br>
            大小：{{ fmtSize(rs.size) }}
          </div>
          <div class="rs-acts">
            <button class="btn btn-ghost btn-sm" @click="updateOne(rs.file)">↻ 更新</button>
            <button class="btn btn-danger btn-sm" @click="deleteRs(rs.file)">删除</button>
          </div>
        </div>
      </div>

      <!-- List view -->
      <div v-else class="card" style="padding:0">
        <table style="width:100%;border-collapse:collapse">
          <thead>
            <tr style="background:var(--surface2);font-size:11px;color:var(--text3);text-transform:uppercase;letter-spacing:.05em">
              <th style="padding:8px 14px;text-align:left;font-weight:600">文件名</th>
              <th style="padding:8px 14px;text-align:left;font-weight:600">格式</th>
              <th style="padding:8px 14px;text-align:left;font-weight:600">更新时间</th>
              <th style="padding:8px 14px;text-align:left;font-weight:600">大小</th>
              <th style="padding:8px 14px;text-align:right;font-weight:600">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="rs in rsStore.local" :key="rs.file"
              style="border-top:1px solid var(--border);font-size:13px">
              <td style="padding:9px 14px;font-family:var(--mono);font-weight:600">{{ rs.file }}</td>
              <td style="padding:9px 14px;color:var(--text3)">{{ rs.format || 'binary' }}</td>
              <td style="padding:9px 14px;color:var(--text3)">{{ fmtTime(rs.updatedAt) }}</td>
              <td style="padding:9px 14px;font-family:var(--mono)">{{ fmtSize(rs.size) }}</td>
              <td style="padding:9px 14px;text-align:right">
                <div class="flex gap-2" style="justify-content:flex-end">
                  <button class="btn btn-ghost btn-sm" @click="updateOne(rs.file)">↻</button>
                  <button class="btn btn-danger btn-sm" @click="deleteRs(rs.file)">删除</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

    </div>

    <!-- ── Hub modal ─────────────────────────────────────────────────── -->
    <div v-if="showHub" class="mask" @click.self="showHub=false">
      <div class="modal" style="max-width:1000px;max-height:85vh">
        <div class="modal-head">
          <div>
            <div>规则集中心</div>
            <div class="text-xs text-muted" style="margin-top:2px">
              规则集数量：{{ hubTotal }} · 来源：MetaCubeX/meta-rules-dat
            </div>
          </div>
          <div class="flex gap-2 items-center">
            <button class="btn btn-ghost btn-sm" @click="loadHub">↻ 更新列表</button>
            <button class="btn-icon" @click="showHub=false">✕</button>
          </div>
        </div>
        <div style="padding:10px 20px;border-bottom:1px solid var(--border);display:flex;flex-direction:column;gap:8px">
          <input class="input" v-model="hubSearch" placeholder="搜索规则集名称…" />
          <div style="display:flex;align-items:center;gap:6px;flex-wrap:wrap">
            <span style="font-size:11px;color:var(--text3)">数据源：</span>
            <button
              v-for="(m, i) in HUB_MIRRORS" :key="i"
              class="btn btn-sm"
              :class="activeMirrorIdx === i ? 'btn-primary' : 'btn-ghost'"
              style="font-size:11px;padding:2px 8px"
              @click="setMirror(i)">
              {{ m.label }}
            </button>
          </div>
        </div>
        <div style="flex:1;overflow-y:auto;padding:14px 20px;max-height:calc(85vh - 160px)">
          <div v-if="hubLoading" class="empty">加载中…</div>
          <div v-else-if="hubErr" class="alert alert-error">{{ hubErr }}</div>
          <div v-else class="rs-grid" style="grid-template-columns:repeat(4,1fr)">
            <div v-for="item in currentHubPage" :key="item.name" class="rs-card">
              <div class="flex items-center gap-2 mb-1">
                <span class="rs-name" style="margin:0;flex:1">{{ item.name }}</span>
                <span class="text-xs" style="background:var(--accent-bg);color:var(--accent);padding:1px 6px;border-radius:4px">
                  {{ item.category }}
                </span>
              </div>
              <div class="rs-meta">规则数量：{{ item.count ?? '—' }}</div>
              <div class="rs-acts">
                <button class="btn btn-ghost btn-sm"
                  @click="addFromHub(item, 'source')">添加 源文件</button>
                <button class="btn btn-primary btn-sm"
                  :class="{ 'btn-ghost': isAdded(item.name, 'binary') }"
                  @click="addFromHub(item, 'binary')">
                  {{ isAdded(item.name, 'binary') ? '已添加' : '添加 二进制' }}
                </button>
              </div>
            </div>
          </div>
        </div>
        <!-- Pagination -->
        <div class="flex items-center gap-2" style="padding:10px 20px;border-top:1px solid var(--border)">
          <button class="btn btn-ghost btn-sm" :disabled="hubPage<=1" @click="hubPage--">‹</button>
          <span class="text-xs text-muted">第 {{ hubPage }} / {{ hubTotalPages }} 页</span>
          <button class="btn btn-ghost btn-sm" :disabled="hubPage>=hubTotalPages" @click="hubPage++">›</button>
          <button class="btn btn-ghost btn-sm ml-auto" @click="showHub=false">关闭</button>
        </div>
      </div>
    </div>

    <!-- ── 新增规则集 modal ──────────────────────────────────────────── -->
    <div v-if="showAdd" class="mask" @click.self="showAdd=false">
      <div class="modal" style="max-width:460px">
        <div class="modal-head">
          <div>新增规则集</div>
          <button class="btn-icon" @click="showAdd=false">✕</button>
        </div>
        <div style="padding:16px 20px;display:flex;flex-direction:column;gap:12px">
          <div class="field">
            <label class="field-label">文件名（含扩展名，如 myrules.srs）</label>
            <input class="input input-mono" v-model="addFile"
              placeholder="myrules.srs" @keyup.enter="doAdd" />
          </div>
          <div class="field">
            <label class="field-label">下载链接</label>
            <input class="input input-mono" v-model="addUrl"
              placeholder="https://..." @keyup.enter="doAdd" />
          </div>
          <div v-if="addErr" class="alert alert-error" style="font-size:12px">{{ addErr }}</div>
        </div>
        <div style="padding:10px 20px;border-top:1px solid var(--border);display:flex;gap:8px;justify-content:flex-end">
          <button class="btn btn-ghost btn-sm" @click="showAdd=false">取消</button>
          <button class="btn btn-primary btn-sm" :disabled="addLoading" @click="doAdd">
            {{ addLoading ? '下载中…' : '保存并下载' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'

import { api } from '../api.js'
import { useRulesetsStore } from '../stores.js'

const rsStore = useRulesetsStore()
const view    = ref('grid')

// Hub
const showHub    = ref(false)
const hubLoading = ref(false)
const hubErr     = ref('')
const hubSearch  = ref('')
const hubPage    = ref(1)
const hubPageSize = 16
const hubData    = ref([])
const hubTotal   = computed(() => hubData.value.length)

const filteredHub = computed(() => {
  const q = hubSearch.value.trim().toLowerCase()
  return q ? hubData.value.filter(x => x.name.toLowerCase().includes(q)) : hubData.value
})

const hubTotalPages = computed(() => Math.max(1, Math.ceil(filteredHub.value.length / hubPageSize)))

const currentHubPage = computed(() => {
  const start = (hubPage.value - 1) * hubPageSize
  return filteredHub.value.slice(start, start + hubPageSize)
})

watch(hubSearch, () => { hubPage.value = 1 })

// sing-full.json mirrors (GUI-for-Cores/Ruleset-Hub)
const HUB_MIRRORS = [
  { label: 'jsDelivr (CF)', url: 'https://testingcf.jsdelivr.net/gh/GUI-for-Cores/Ruleset-Hub@latest/sing-full.json' },
  { label: 'jsDelivr',      url: 'https://cdn.jsdelivr.net/gh/GUI-for-Cores/Ruleset-Hub@latest/sing-full.json' },
  { label: 'GitHub',        url: 'https://github.com/GUI-for-Cores/Ruleset-Hub/releases/download/latest/sing-full.json' },
  { label: 'ghproxy',       url: 'https://ghproxy.com/https://github.com/GUI-for-Cores/Ruleset-Hub/releases/download/latest/sing-full.json' },
]
const activeMirrorIdx = ref(parseInt(localStorage.getItem('hubMirrorIdx') || '0'))

function setMirror(idx) {
  activeMirrorIdx.value = idx
  localStorage.setItem('hubMirrorIdx', String(idx))
  loadHub()
}

// Hub data from GUI-for-Cores/Ruleset-Hub sing-full.json
// 通过后端代理请求，避免浏览器跨域/网络限制
async function loadHub() {
  hubLoading.value = true; hubErr.value = ''
  const userChoice = activeMirrorIdx.value
  const order = [
    userChoice,
    ...HUB_MIRRORS.map((_, i) => i).filter(i => i !== userChoice)
  ]
  let lastErr = ''
  for (const idx of order) {
    const mirrorUrl = HUB_MIRRORS[idx].url
    try {
      const r = await api('GET', '/rulesets/fetch-hub?url=' + encodeURIComponent(mirrorUrl))
      const d = r
      hubData.value = (d.list || []).map(item => ({
        name: item.name,
        count: item.count,
        category: item.type,
        binaryUrl: (item.type === 'geoip' ? d.geoip : d.geosite) + item.name + '.srs',
        sourceUrl:  (item.type === 'geoip' ? d.geoip : d.geosite) + item.name + '.json',
      }))
      // 保持用户选择的镜像不变，不因为自动重试而覆盖
      hubLoading.value = false
      return
    } catch (e) {
      lastErr = `[${HUB_MIRRORS[idx].label}] ${e.message}`
    }
  }
  hubErr.value = '所有镜像均加载失败，请手动切换镜像或检查网络。最后错误：' + lastErr
  hubLoading.value = false
}

watch(showHub, v => { if (v && !hubData.value.length) loadHub() })

function isAdded(name, fmt) {
  const fname = name + (fmt === 'binary' ? '.srs' : '.json')
  return rsStore.local.some(r => r.file === fname)
}

async function addFromHub(item, fmt) {
  const url  = fmt === 'binary' ? item.binaryUrl : item.sourceUrl
  const file = item.name + (fmt === 'binary' ? '.srs' : '.json')
  try {
    await api('POST', '/rulesets/download', { url, file })
    await rsStore.scan()
  } catch (e) { alert('添加失败: ' + e.message) }
}

// Local operations
const ghProxy = ref(localStorage.getItem('ghProxy') || '')

// 新增规则集
const showAdd   = ref(false)
const addFile   = ref('')
const addUrl    = ref('')
const addErr    = ref('')
const addLoading = ref(false)

async function doAdd() {
  addErr.value = ''
  const file = addFile.value.trim()
  const url  = addUrl.value.trim()
  if (!file) { addErr.value = '请输入文件名'; return }
  if (!url)  { addErr.value = '请输入下载链接'; return }
  if (!file.endsWith('.srs') && !file.endsWith('.json')) {
    addErr.value = '文件名必须以 .srs 或 .json 结尾'
    return
  }
  addLoading.value = true
  try {
    await api('POST', '/rulesets/download', { url, file })
    await rsStore.scan()
    showAdd.value = false
    addFile.value = ''
    addUrl.value  = ''
  } catch (e) {
    addErr.value = '下载失败：' + e.message
  }
  addLoading.value = false
}
async function updateAll() {
  await rsStore.updateAll(ghProxy.value)
}
async function updateOne(file) {
  try {
    await api('POST', '/update-rules', { proxy: ghProxy.value, files: [file] })
    await rsStore.scan()
  } catch (e) { alert(e.message) }
}
async function deleteRs(file) {
  if (!confirm(`删除规则集 ${file}？`)) return
  await rsStore.deleteRuleset(file)
}

function fmtTime(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  const now = Date.now()
  const diff = now - d.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff/60000) + ' 分钟前'
  if (diff < 86400000*2) return Math.floor(diff/3600000) + ' 小时前'
  if (diff < 86400000*30) return Math.floor(diff/86400000) + ' 天前'
  if (diff < 86400000*60) return '上个月'
  return d.toLocaleDateString('zh-CN')
}

function fmtSize(bytes) {
  if (!bytes) return '—'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024*1024) return (bytes/1024).toFixed(1) + ' KB'
  return (bytes/1024/1024).toFixed(2) + ' MB'
}

onMounted(() => rsStore.scan())
</script>

<style scoped>
@media (max-width: 640px) {
  /* topbar-right wraps automatically via global style */
}
</style>
