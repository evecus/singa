<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-logo">
        <img src="/favicon.svg" style="width:48px;height:48px" alt="singa" />
        <div class="login-brand">singa</div>
      </div>

      <!-- Setup mode -->
      <template v-if="needsSetup">
        <div class="login-title">首次启动，请创建账号</div>
        <div class="field" style="margin-bottom:10px">
          <label class="field-label">用户名</label>
          <input class="input" v-model="form.username" placeholder="admin" autocomplete="username" />
        </div>
        <div class="field" style="margin-bottom:10px">
          <label class="field-label">密码</label>
          <input class="input" v-model="form.password" type="password" placeholder="至少 6 位" autocomplete="new-password" />
        </div>
        <div class="field" style="margin-bottom:16px">
          <label class="field-label">确认密码</label>
          <input class="input" v-model="form.confirm" type="password" placeholder="再次输入密码" autocomplete="new-password" />
        </div>
        <button class="btn btn-primary" style="width:100%" :disabled="loading" @click="doSetup">
          {{ loading ? '创建中…' : '创建账号' }}
        </button>
      </template>

      <!-- Login mode -->
      <template v-else>
        <div class="login-title">登录</div>
        <div class="field" style="margin-bottom:10px">
          <label class="field-label">用户名</label>
          <input class="input" v-model="form.username" placeholder="admin" autocomplete="username"
            @keydown.enter="doLogin" />
        </div>
        <div class="field" style="margin-bottom:16px">
          <label class="field-label">密码</label>
          <input class="input" v-model="form.password" type="password" placeholder="密码"
            autocomplete="current-password" @keydown.enter="doLogin" />
        </div>
        <button class="btn btn-primary" style="width:100%" :disabled="loading" @click="doLogin">
          {{ loading ? '登录中…' : '登录' }}
        </button>
      </template>

      <div v-if="errorMsg" class="alert alert-error mt-3">{{ errorMsg }}</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'
import { useAuthStore } from '../stores.js'

const router = useRouter()
const authStore = useAuthStore()

const needsSetup = ref(false)
const loading = ref(false)
const errorMsg = ref('')

const form = ref({ username: '', password: '', confirm: '' })

onMounted(async () => {
  const status = await api('GET', '/auth/status').catch(() => ({ enabled: false }))
  if (!status.enabled) {
    // Auth disabled, go straight to dashboard
    authStore.setToken('noauth')
    router.replace('/dashboard')
    return
  }
  needsSetup.value = status.needsSetup
})

async function doSetup() {
  errorMsg.value = ''
  if (!form.value.username) { errorMsg.value = '请填写用户名'; return }
  if (form.value.password.length < 6) { errorMsg.value = '密码至少 6 位'; return }
  if (form.value.password !== form.value.confirm) { errorMsg.value = '两次密码不一致'; return }
  loading.value = true
  try {
    const res = await api('POST', '/auth/setup', {
      username: form.value.username,
      password: form.value.password,
    })
    authStore.setToken(res.token)
    router.replace('/dashboard')
  } catch (e) {
    errorMsg.value = e.message
  } finally {
    loading.value = false
  }
}

async function doLogin() {
  errorMsg.value = ''
  if (!form.value.username || !form.value.password) {
    errorMsg.value = '请填写用户名和密码'; return
  }
  loading.value = true
  try {
    const res = await api('POST', '/auth/login', {
      username: form.value.username,
      password: form.value.password,
    })
    authStore.setToken(res.token)
    router.replace('/dashboard')
  } catch (e) {
    errorMsg.value = '用户名或密码错误'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg);
}

.login-card {
  width: 340px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 32px 28px;
  box-shadow: 0 8px 32px rgba(0,0,0,.12);
}

.login-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 24px;
}

.login-brand {
  font-size: 22px;
  font-weight: 700;
  letter-spacing: .5px;
  color: var(--text1);
}

.login-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text2);
  margin-bottom: 18px;
}
</style>
