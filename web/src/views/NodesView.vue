<template>
  <div class="page">
    <div class="topbar">
      <span class="topbar-title">节点与订阅</span>
      <div class="topbar-right">
        <div class="tabs" style="margin:0;border:none">
          <button class="tab-btn" :class="{ on: tab === 'nodes' }" @click="tab = 'nodes'">节点</button>
          <button class="tab-btn" :class="{ on: tab === 'subs'  }" @click="tab = 'subs'">订阅</button>
        </div>
      </div>
    </div>
    <div class="page">

      <!-- ═══════════════════════════════ NODES ═══════════════════════════ -->
      <template v-if="tab === 'nodes'">
        <div class="grid-2" style="align-items:start;gap:16px">

          <!-- Import panel -->
          <div class="card">
            <div class="card-title-row">
              <span class="card-title">导入节点</span>
            </div>
            <div class="field" style="margin-bottom:10px">
              <label class="field-label">粘贴分享链接（每行一个）</label>
              <textarea class="textarea" v-model="importText" rows="8"
                placeholder="vmess://...&#10;vless://...&#10;trojan://...&#10;ss://...&#10;tuic://...&#10;hy2://..."></textarea>
              <div class="field-hint">支持 vmess / vless / trojan / ss / tuic / hysteria2</div>
            </div>
            <div class="flex gap-2">
              <button class="btn btn-primary" :disabled="!importText.trim() || importing"
                @click="doImport">
                {{ importing ? '导入中…' : '+ 导入' }}
              </button>
              <button class="btn btn-ghost" @click="importText = ''">清空</button>
            </div>
            <div v-if="importErrs.length" class="alert alert-warn mt-2">
              <div v-for="(e,i) in importErrs" :key="i">⚠ {{ e }}</div>
            </div>
            <div v-if="importOk" class="alert alert-success mt-2">{{ importOk }}</div>
          </div>

          <!-- Node list -->
          <div class="card">
            <div class="card-title-row">
              <span class="card-title">节点列表 ({{ nodesStore.nodes.length }})</span>
              <button v-if="nodesStore.nodes.length" class="btn btn-ghost btn-sm"
                @click="confirmClearNodes">清空全部</button>
            </div>
            <div v-if="!nodesStore.nodes.length" class="empty">暂无节点</div>
            <div v-else class="node-list">
              <div v-for="n in nodesStore.nodes" :key="n.id" class="node-row">
                <span class="proto-badge" :class="'p-' + n.type">{{ n.type }}</span>
                <div class="node-meta">
                  <div class="node-name">{{ n.name }}</div>
                  <div class="node-addr">{{ n.server }}:{{ n.port }}</div>
                </div>
                <button class="btn-icon danger" @click="nodesStore.deleteNode(n.id)" title="删除">✕</button>
              </div>
            </div>
          </div>

        </div>
      </template>

      <!-- ═══════════════════════════════ SUBS ════════════════════════════ -->
      <template v-if="tab === 'subs'">
        <div style="display:flex;flex-direction:column;gap:12px">

          <!-- Add subscription -->
          <div class="card">
            <div class="card-title">添加订阅</div>
            <div class="grid-2 gap-3" style="margin-bottom:10px">
              <div class="field">
                <label class="field-label">订阅名称</label>
                <input class="input" v-model="newSub.name" placeholder="我的机场" />
              </div>
              <div class="field">
                <label class="field-label">订阅 URL</label>
                <input class="input input-mono" v-model="newSub.url" placeholder="https://..." />
              </div>
            </div>
            <div class="flex gap-2">
              <button class="btn btn-primary" :disabled="!newSub.url.trim() || addingSubQuick"
                @click="addSubQuick">
                {{ addingSubQuick ? '添加中…' : '+ 添加并拉取' }}
              </button>
            </div>
            <div v-if="addSubErr" class="alert alert-error mt-2">{{ addSubErr }}</div>
          </div>

          <!-- Subscription tabs + node list -->
          <div v-if="subsStore.subs.length" class="card">
            <!-- Subscription buttons row -->
            <div style="display:flex;flex-wrap:wrap;gap:8px;margin-bottom:14px">
              <button v-for="sub in subsStore.subs" :key="sub.id"
                class="sub-tab-btn" :class="{ on: activeSubId === sub.id }"
                @click="selectSub(sub.id)">
                {{ sub.name }}
                <span class="sub-tab-count">{{ sub.nodeCount || 0 }}</span>
              </button>
            </div>

            <!-- Active sub info bar -->
            <div v-if="activeSub" style="display:flex;align-items:center;gap:10px;margin-bottom:10px">
              <div style="flex:1;font-size:12px;color:var(--text3)">
                <template v-if="activeSub.error">⚠ {{ activeSub.error }}</template>
                <template v-else-if="activeSub.nodeCount">
                  共 {{ activeSub.nodeCount }} 个节点 · 更新于 {{ fmtTime(activeSub.updatedAt) }}
                </template>
                <template v-else>未拉取节点</template>
              </div>
              <button class="btn btn-ghost btn-sm"
                :disabled="!!subsStore.updating[activeSub.id]"
                @click="subsStore.update(activeSub.id)">
                {{ subsStore.updating[activeSub.id] ? '更新中…' : '↻ 更新' }}
              </button>
              <button class="btn btn-ghost btn-sm" @click="openEditSub(activeSub)">✎ 编辑</button>
              <button class="btn btn-ghost btn-sm danger" @click="subsStore.remove(activeSub.id); activeSubId=''">删除</button>
            </div>

            <!-- Node list for active sub -->
            <div v-if="activeSubId">
              <div v-if="subProxiesLoading" class="empty">加载中…</div>
              <div v-else-if="!subProxies.length" class="empty">
                {{ activeSub?.error ? '拉取失败' : '暂无节点，请先更新订阅' }}
              </div>
              <div v-else class="node-list">
                <div v-for="(p, idx) in subProxies" :key="idx" class="node-row">
                  <span class="proto-badge" :class="'p-' + (p.type||'').toLowerCase()">{{ p.type }}</span>
                  <div class="node-meta">
                    <div class="node-name">{{ p.name || p.tag || '节点 ' + (idx+1) }}</div>
                    <div class="node-addr">{{ p.server }}{{ p.server_port ? ':'+p.server_port : '' }}</div>
                  </div>
                  <button class="btn-icon danger" @click="deleteSubProxy(idx)" title="删除">✕</button>
                </div>
              </div>
            </div>
            <div v-else class="empty" style="padding:12px 0">点击上方订阅名称查看节点</div>
          </div>

          <div v-if="!subsStore.subs.length" class="card">
            <div class="empty">暂无订阅</div>
          </div>

        </div>
      </template>

    </div>

    <!-- Edit subscription modal -->
    <div v-if="showEditSub" class="mask" @click.self="showEditSub=false">
      <div class="modal" style="max-width:480px">
        <div class="modal-head">
          <span>编辑订阅</span>
          <button class="btn-icon" @click="showEditSub=false">✕</button>
        </div>
        <div style="padding:16px 20px;display:flex;flex-direction:column;gap:12px">
          <div class="field">
            <label class="field-label">订阅名称</label>
            <input class="input" v-model="editSubForm.name" placeholder="我的机场" />
          </div>
          <div class="field">
            <label class="field-label">订阅 URL</label>
            <input class="input input-mono" v-model="editSubForm.url" placeholder="https://..." />
          </div>
          <div v-if="editSubErr" class="alert alert-error">{{ editSubErr }}</div>
          <div class="flex gap-2" style="justify-content:flex-end">
            <button class="btn btn-ghost" @click="showEditSub=false">取消</button>
            <button class="btn btn-primary" :disabled="!editSubForm.url.trim() || editSubSaving"
              @click="saveEditSub">
              {{ editSubSaving ? '保存中…' : '保存' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useNodesStore, useSubsStore } from '../stores.js'

const nodesStore = useNodesStore()
const subsStore  = useSubsStore()

const tab = ref('nodes')

// Nodes
const importText = ref('')
const importErrs = ref([])
const importOk   = ref('')
const importing  = ref(false)

async function doImport() {
  if (!importText.value.trim()) return
  importing.value = true; importErrs.value = []; importOk.value = ''
  try {
    const res = await nodesStore.importNodes(importText.value.trim())
    importErrs.value = res.errors || []
    const cnt = res.imported || 0
    if (cnt > 0) {
      importOk.value = `✓ 成功导入 ${cnt} 个节点`
      importText.value = ''
    }
  } catch (e) { importErrs.value = [e.message] }
  finally { importing.value = false }
}

function confirmClearNodes() {
  if (!confirm(`确认删除全部 ${nodesStore.nodes.length} 个节点？`)) return
  ;[...nodesStore.nodes].forEach(n => nodesStore.deleteNode(n.id))
}

// Subscriptions
const newSub = reactive({ name: '', url: '' })
const addingSubQuick = ref(false)
const addSubErr = ref('')

// Active sub + proxies
const activeSubId       = ref('')
const subProxies        = ref([])
const subProxiesLoading = ref(false)
const activeSub         = computed(() => subsStore.subs.find(s => s.id === activeSubId.value))

async function selectSub(id) {
  if (activeSubId.value === id) { activeSubId.value = ''; subProxies.value = []; return }
  activeSubId.value = id
  subProxiesLoading.value = true
  subProxies.value = []
  try { subProxies.value = await subsStore.getProxies(id) } catch {}
  subProxiesLoading.value = false
}

async function deleteSubProxy(idx) {
  if (!confirm('确定删除该节点？')) return
  await subsStore.deleteProxy(activeSubId.value, idx)
  subProxies.value.splice(idx, 1)
}

async function addSubQuick() {
  if (!newSub.url.trim()) return
  addingSubQuick.value = true; addSubErr.value = ''
  try {
    const s = await subsStore.add(
      newSub.name.trim() || newSub.url.trim(),
      newSub.url.trim(),
      null,
    )
    newSub.name = ''; newSub.url = ''
    await subsStore.update(s.id)
  } catch (e) { addSubErr.value = e.message }
  finally { addingSubQuick.value = false }
}

// Edit subscription
const showEditSub  = ref(false)
const editSubForm  = reactive({ id: '', name: '', url: '' })
const editSubSaving = ref(false)
const editSubErr   = ref('')

function openEditSub(sub) {
  editSubForm.id   = sub.id
  editSubForm.name = sub.name
  editSubForm.url  = sub.url || ''
  editSubErr.value = ''
  showEditSub.value = true
}

async function saveEditSub() {
  if (!editSubForm.url.trim()) return
  editSubSaving.value = true; editSubErr.value = ''
  try {
    await subsStore.updateMeta(editSubForm.id, {
      name: editSubForm.name.trim() || editSubForm.url.trim(),
      url:  editSubForm.url.trim(),
    })
    showEditSub.value = false
  } catch (e) { editSubErr.value = e.message }
  finally { editSubSaving.value = false }
}

function fmtTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleString('zh-CN', { month:'2-digit', day:'2-digit', hour:'2-digit', minute:'2-digit' })
}

onMounted(() => { nodesStore.load(); subsStore.load() })
</script>

<style scoped>
.sub-tab-btn {
  padding: 5px 14px; border-radius: 20px; font-size: 13px; font-weight: 600;
  border: 1.5px solid var(--border2); background: var(--surface2); cursor: pointer;
  color: var(--text2); transition: all .15s; display: flex; align-items: center; gap: 6px;
}
.sub-tab-btn:hover { border-color: var(--accent); color: var(--accent); }
.sub-tab-btn.on { background: var(--accent); border-color: var(--accent); color: #fff; }
.sub-tab-count {
  font-size: 11px; padding: 1px 6px; border-radius: 10px;
  background: rgba(0,0,0,.12); font-weight: 400;
}
.sub-tab-btn.on .sub-tab-count { background: rgba(255,255,255,.25); }
.btn.danger { color: var(--red); }
.btn.danger:hover { background: rgba(239,68,68,.08); }
@media (max-width: 640px) {
  .sub-tab-btn { padding: 4px 10px; font-size: 12px; }
}
</style>
