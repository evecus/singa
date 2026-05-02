import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from './api.js'

// ── Status ──────────────────────────────────────────────────────────────────
export const useStatusStore = defineStore('status', () => {
  const status = ref({ state: 'stopped', pid: 0, ports: {} })
  const isRunning = computed(() => status.value.state === 'running')

  async function fetch() {
    try { status.value = await api('GET', '/status') } catch {}
  }
  async function stop() {
    await api('POST', '/stop')
    await fetch()
  }
  return { status, isRunning, fetch, stop }
})

// ── Logs ────────────────────────────────────────────────────────────────────
export const useLogsStore = defineStore('logs', () => {
  const logs = ref([])
  let src = null
  function startSSE() {
    if (src) src.close()
    src = new EventSource('/api/logs')
    src.onmessage = e => {
      logs.value.push(e.data)
      if (logs.value.length > 3000) logs.value.splice(0, 500)
    }
  }
  function stopSSE() { if (src) { src.close(); src = null } }
  function clear()   { logs.value = [] }
  return { logs, startSSE, stopSSE, clear }
})

// ── Nodes ────────────────────────────────────────────────────────────────────
export const useNodesStore = defineStore('nodes', () => {
  const nodes = ref([])
  async function load() {
    try { nodes.value = await api('GET', '/nodes') } catch {}
  }
  async function importNodes(text) {
    const res = await api('POST', '/nodes/import', { text })
    await load()
    return res
  }
  async function deleteNode(id) {
    await api('DELETE', '/nodes/' + id)
    nodes.value = nodes.value.filter(n => n.id !== id)
  }
  return { nodes, load, importNodes, deleteNode }
})

// ── Subscriptions ────────────────────────────────────────────────────────────
export const useSubsStore = defineStore('subs', () => {
  const subs     = ref([])
  const updating = ref({})

  async function load() {
    try { subs.value = await api('GET', '/subscriptions') } catch {}
  }
  async function add(name, url, wizardConfig) {
    const s = await api('POST', '/subscriptions', { name, url, wizardConfig })
    subs.value.push(s)
    return s
  }
  async function updateMeta(id, payload) {
    const s = await api('PATCH', '/subscriptions/' + id, payload)
    const i = subs.value.findIndex(x => x.id === id)
    if (i >= 0) subs.value[i] = s
    return s
  }
  async function update(id) {
    updating.value = { ...updating.value, [id]: true }
    try {
      const s = await api('POST', `/subscriptions/${id}/update`)
      const i = subs.value.findIndex(x => x.id === id)
      if (i >= 0) subs.value[i] = s
      return s
    } finally {
      const n = { ...updating.value }; delete n[id]; updating.value = n
    }
  }
  async function remove(id) {
    await api('DELETE', '/subscriptions/' + id)
    subs.value = subs.value.filter(s => s.id !== id)
  }
  async function getProxies(id) {
    return api('GET', `/subscriptions/${id}/proxies`)
  }
  async function deleteProxy(subId, idx) {
    await api('DELETE', `/subscriptions/${subId}/proxies/${idx}`)
    await load()
  }
  return { subs, updating, load, add, updateMeta, update, remove, getProxies, deleteProxy }
})

// ── Rulesets ─────────────────────────────────────────────────────────────────
export const useRulesetsStore = defineStore('rulesets', () => {
  // local rulesets: array of { file, mirror, updatedAt, size }
  const local   = ref([])
  const updating = ref(false)
  const results  = ref([])

  async function scan() {
    try { local.value = await api('GET', '/rulesets') } catch {}
  }
  async function updateAll(proxy) {
    updating.value = true; results.value = []
    try {
      const res = await api('POST', '/update-rules', { proxy })
      results.value = res.results || []
      await scan()
      return res
    } finally { updating.value = false }
  }
  async function deleteRuleset(file) {
    await api('DELETE', '/rulesets/' + encodeURIComponent(file))
    local.value = local.value.filter(r => r.file !== file)
  }
  return { local, updating, results, scan, updateAll, deleteRuleset }
})

// ── Profiles ──────────────────────────────────────────────────────────────────
export const useProfilesStore = defineStore('profiles', () => {
  const profiles = ref([])

  async function load() {
    try { profiles.value = await api('GET', '/profiles') } catch {}
  }
  async function add(name, wizardConfig) {
    const p = await api('POST', '/profiles', { name, wizardConfig })
    profiles.value.push(p)
    return p
  }
  async function updateMeta(id, payload) {
    const p = await api('PATCH', '/profiles/' + id, payload)
    const i = profiles.value.findIndex(x => x.id === id)
    if (i >= 0) profiles.value[i] = p
    return p
  }
  async function remove(id) {
    await api('DELETE', '/profiles/' + id)
    profiles.value = profiles.value.filter(p => p.id !== id)
  }
  return { profiles, load, add, updateMeta, remove }
})
