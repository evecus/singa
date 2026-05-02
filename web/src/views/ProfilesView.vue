<template>
  <div class="page">
    <div class="topbar">
      <span class="topbar-title">配置文件</span>
      <div class="topbar-right">
        <div class="tabs" style="margin:0;border:none">
          <button class="tab-btn" :class="{ on: tab === 'generate' }" @click="tab = 'generate'">生成配置</button>
          <button class="tab-btn" :class="{ on: tab === 'upload'   }" @click="tab = 'upload'">上传配置</button>
        </div>
      </div>
    </div>
    <div class="page">

      <!-- ═══════════════════════ GENERATE ════════════════════════════ -->
      <template v-if="tab === 'generate'">
        <div style="display:flex;flex-direction:column;gap:14px">

          <div class="alert alert-info" style="display:flex;align-items:center;justify-content:space-between;gap:12px">
            <span>通过向导逐步配置各项参数，生成可用的 sing-box 配置。每个配置可独立绑定一个订阅。</span>
            <button class="btn btn-primary btn-sm" style="white-space:nowrap" @click="openWizard(null)">
              ＋ 新增配置
            </button>
          </div>

          <div v-if="!profilesStore.profiles.length" class="card">
            <div class="empty">
              点击右上角「新增配置」创建第一个向导配置。
            </div>
          </div>

          <div v-for="prof in profilesStore.profiles" :key="prof.id" class="card">
            <div class="flex items-center gap-3 mb-3">
              <span style="font-size:22px">📄</span>
              <div>
                <div style="font-weight:700">{{ prof.name }}</div>
                <div class="text-xs text-muted monospace">
                  <template v-if="prof.updatedAt">{{ fmtTime(prof.updatedAt) }}</template>
                </div>
              </div>
              <div class="ml-auto flex gap-2">
                <button class="btn btn-ghost btn-sm" @click="validateProfile(prof)"
                  :disabled="!prof.wizardConfig || validating===prof.id">
                  {{ validating===prof.id ? '验证中…' : '✓ 验证' }}
                </button>
                <button class="btn btn-ghost btn-sm" @click="openWizard(prof)">
                  ✎ 编辑配置
                </button>
                <button class="btn btn-danger btn-sm" @click="deleteProfile(prof.id)">
                  删除
                </button>
              </div>
            </div>
            <div v-if="prof.wizardConfig" class="alert alert-success text-xs">
              ✓ 已完成配置，可从仪表盘启动
            </div>
            <div v-else class="alert alert-warn text-xs">
              尚未完成向导，点击「编辑配置」继续
            </div>
            <!-- Validation results -->
            <template v-if="validationResults[prof.id]">
              <div v-if="validationResults[prof.id].ok" class="alert alert-success text-xs mt-2">
                ✓ 配置验证通过，所有引用均有效
              </div>
              <div v-else class="alert alert-error mt-2">
                <div class="text-xs font-bold mb-1">⚠ 配置存在引用错误：</div>
                <div v-for="(e,i) in validationResults[prof.id].errors" :key="i"
                  class="text-xs" style="margin-top:3px">
                  <code style="background:rgba(0,0,0,.1);padding:0 4px;border-radius:3px">{{ e.location }}</code>
                  {{ e.message }}
                </div>
              </div>
            </template>
          </div>

        </div>
      </template>

      <!-- ═══════════════════════ UPLOAD ═══════════════════════════════ -->
      <template v-if="tab === 'upload'">
        <div style="display:flex;flex-direction:column;gap:14px">

          <div class="card">
            <div class="card-title">上传配置文件</div>
            <label class="upload-drop" :class="{ over: dragOver }"
              @dragover.prevent="dragOver=true" @dragleave="dragOver=false" @drop.prevent="onDrop">
              <input type="file" accept=".json" style="display:none" ref="fileInput" @change="onFileChange" />
              <span style="font-size:36px">📁</span>
              <span style="font-size:13px;color:var(--text3)">拖拽或点击上传 sing-box JSON 配置</span>
            </label>
            <div v-if="uploadErr" class="alert alert-error mt-2">{{ uploadErr }}</div>
            <div v-if="uploadOk"  class="alert alert-success mt-2">{{ uploadOk }}</div>
          </div>

          <div v-if="configInfo" class="card">
            <div class="card-title">已上传配置</div>
            <div class="info-table">
              <span class="info-k">状态</span><span class="info-v">已就绪</span>
              <span class="info-k">入站数量</span><span class="info-v">{{ configInfo.inbounds?.length || 0 }}</span>
            </div>
            <div class="flex gap-2 mt-3">
              <button class="btn btn-ghost btn-sm" @click="viewConfigJson">查看 JSON</button>
            </div>
          </div>

        </div>
      </template>

    </div>

    <!-- View JSON modal -->
    <div v-if="showJson" class="mask" @click.self="showJson=false">
      <div class="modal" style="max-width:680px">
        <div class="modal-head">
          <span>配置文件内容</span>
          <button class="btn-icon" @click="showJson=false">✕</button>
        </div>
        <div style="padding:0 20px 20px">
          <textarea class="textarea" :value="jsonText" readonly rows="22"
            style="font-size:11px"></textarea>
        </div>
      </div>
    </div>

    <ProfileWizard v-if="showWizard"
      :profile="editingProfile"
      :subs="subsStore.subs"
      @close="showWizard=false"
      @saved="onWizardSaved" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'
import { useSubsStore, useProfilesStore } from '../stores.js'
import ProfileWizard from '../components/ProfileWizard.vue'

const subsStore    = useSubsStore()
const profilesStore = useProfilesStore()
const tab = ref('generate')

// ── Upload ────────────────────────────────────────────────────────────────
const dragOver   = ref(false)
const uploadErr  = ref('')
const uploadOk   = ref('')
const configInfo = ref(null)
const showJson   = ref(false)
const jsonText   = ref('')
const fileInput  = ref(null)

async function loadConfigInfo() {
  try { configInfo.value = await api('GET', '/config/info') } catch {}
}
async function uploadFile(file) {
  uploadErr.value = ''; uploadOk.value = ''
  if (!file?.name.endsWith('.json')) { uploadErr.value = '请上传 .json 文件'; return }
  try {
    const fd = new FormData(); fd.append('config', file)
    const r = await fetch('/api/config', { method:'POST', body:fd })
    const d = await r.json()
    if (!r.ok) throw new Error(d.error)
    uploadOk.value = '✓ 上传成功'
    await loadConfigInfo()
  } catch (e) { uploadErr.value = '✕ ' + e.message }
}
function onFileChange(e) { uploadFile(e.target.files[0]) }
function onDrop(e)       { dragOver.value=false; uploadFile(e.dataTransfer.files[0]) }
async function viewConfigJson() {
  try { jsonText.value = await (await fetch('/api/config/raw')).text() }
  catch { jsonText.value = '(无法读取)' }
  showJson.value = true
}

// ── Generate / wizard ─────────────────────────────────────────────────────
const showWizard    = ref(false)
const editingProfile = ref(null)
const validating = ref('')  // profile id being validated
const validationResults = ref({})  // { [profId]: { ok, errors } }

function openWizard(prof) {
  editingProfile.value = prof
  showWizard.value = true
}

async function onWizardSaved(prof) {
  showWizard.value = false
  await profilesStore.load()
  // Auto-validate after saving
  const saved = profilesStore.profiles.find(p => p.id === (prof?.id || editingProfile.value?.id))
  if (saved?.wizardConfig) {
    await validateProfile(saved)
  }
}

async function validateProfile(prof) {
  if (!prof.wizardConfig) return
  validating.value = prof.id
  try {
    const res = await api('POST', '/profiles/validate', { wizardConfig: prof.wizardConfig })
    validationResults.value = { ...validationResults.value, [prof.id]: res }
  } catch (e) {
    validationResults.value = { ...validationResults.value, [prof.id]: {
      ok: false, errors: [{ location: 'network', message: e.message }]
    }}
  } finally {
    validating.value = ''
  }
}

async function deleteProfile(id) {
  if (!confirm('确定删除此配置？')) return
  await profilesStore.remove(id)
}

// ── Helpers ───────────────────────────────────────────────────────────────

function fmtTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleString('zh-CN', { month:'2-digit', day:'2-digit', hour:'2-digit', minute:'2-digit' })
}

onMounted(() => {
  subsStore.load()
  profilesStore.load()
  loadConfigInfo()
})
</script>

<style scoped>
.upload-drop {
  display: flex; flex-direction: column; align-items: center; gap: 10px;
  padding: 32px 20px; border: 2px dashed var(--border2); border-radius: var(--radius);
  cursor: pointer; transition: all .15s; margin-top: 8px; background: var(--surface2);
}
.upload-drop:hover, .upload-drop.over {
  border-color: var(--accent); background: var(--accent-bg);
}
@media (max-width: 640px) {
  .upload-drop { padding: 22px 14px; }
}
</style>
