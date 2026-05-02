<template>
  <div class="page">
    <div class="log-inner">
      <div class="log-topbar">
        <span class="log-live" :class="{ live: isRunning }">
          {{ isRunning ? '● LIVE' : '○ IDLE' }}
        </span>
        <button class="log-clear" @click="logsStore.clear()">清空</button>
      </div>
      <div class="log-body" ref="logEl" @scroll="onLogScroll">
        <div v-if="logsStore.logs.length === 0" class="log-empty">等待核心启动…</div>
        <div v-for="(line, i) in logsStore.logs" :key="i" class="log-line">
          <span class="log-arr">›</span>
          <span class="log-txt" :class="logClass(line)">{{ line }}</span>
        </div>
      </div>
      <div class="log-foot">{{ logsStore.logs.length }} lines</div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { useStatusStore, useLogsStore } from '../stores.js'

const statusStore = useStatusStore()
const logsStore   = useLogsStore()
const isRunning   = computed(() => statusStore.isRunning)
const logEl       = ref(null)
const autoScroll  = ref(true)

function logClass(line) {
  const l = line.toLowerCase()
  if (l.includes('error') || l.includes('fatal')) return 'l-err'
  if (l.includes('warn'))  return 'l-warn'
  if (l.includes('info'))  return 'l-info'
  return ''
}

function onLogScroll() {
  if (!logEl.value) return
  const el = logEl.value
  autoScroll.value = el.scrollTop + el.clientHeight >= el.scrollHeight - 20
}

watch(() => logsStore.logs.length, () => {
  if (autoScroll.value) nextTick(() => {
    if (logEl.value) logEl.value.scrollTop = logEl.value.scrollHeight
  })
})
</script>

<style scoped>
/* LogsView uses its own full-height layout, no topbar — already mobile-friendly */
</style>
