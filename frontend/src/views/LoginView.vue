<script setup>
import { ref, nextTick } from 'vue'
import { useAuth } from '../stores/auth.js'

const { login } = useAuth()

const isRegister = ref(false)
const form = ref({ username: '', password: '' })
const error = ref('')
const loading = ref(false)

const toggleMode = () => {
  isRegister.value = !isRegister.value
  error.value = ''
}

const submit = async () => {
  const u = form.value.username.trim()
  const p = form.value.password
  if (!u || !p) {
    error.value = '请填写用户名和密码'
    return
  }
  loading.value = true
  error.value = ''

  try {
    const path = isRegister.value ? '/api/v1/auth/register' : '/api/v1/auth/login'
    const res = await fetch(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: u, password: p }),
    })
    const data = await res.json()
    if (data.code !== 200) {
      error.value = data.message || '请求失败'
      loading.value = false
      return
    }
    login(data.token, u)
  } catch (e) {
    error.value = '网络错误，请确认后端服务已启动'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="overlay">
    <div class="card">
      <div class="icon">📊</div>
      <h2>{{ isRegister ? '创建账号' : '欢迎回来' }}</h2>
      <p class="sub">{{ isRegister ? '注册后即可查看作品数据排行榜' : '登录以查看你的作品数据排行榜' }}</p>

      <div class="field">
        <label>用户名</label>
        <input
          v-model="form.username"
          placeholder="输入用户名"
          @keyup.enter="submit"
          autofocus
        />
      </div>
      <div class="field">
        <label>密码</label>
        <input
          v-model="form.password"
          type="password"
          placeholder="输入密码"
          @keyup.enter="submit"
        />
      </div>

      <p v-if="error" class="err">{{ error }}</p>

      <button class="btn-primary" @click="submit" :disabled="loading">
        {{ loading ? '请稍候…' : (isRegister ? '注册' : '登录') }}
      </button>

      <p class="switch">
        <a @click="toggleMode">
          {{ isRegister ? '已有账号？去登录' : '没有账号？去注册' }}
        </a>
      </p>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed; inset: 0;
  display: flex; align-items: center; justify-content: center;
  background: var(--bg);
}
.card {
  background: var(--card);
  border-radius: var(--radius);
  padding: 48px 40px;
  width: 400px;
  max-width: 90vw;
  box-shadow: var(--shadow-md);
  text-align: center;
}
.icon { font-size: 40px; margin-bottom: 12px; }
h2 {
  font-size: 26px; font-weight: 700;
  letter-spacing: -0.3px; margin-bottom: 4px;
}
.sub {
  font-size: 14px; color: var(--text2);
  margin-bottom: 32px;
}
.field {
  text-align: left; margin-bottom: 18px;
}
.field label {
  display: block; font-size: 11px; font-weight: 700;
  color: var(--text2); text-transform: uppercase;
  letter-spacing: 0.6px; margin-bottom: 6px;
}
.field input {
  width: 100%; padding: 13px 16px;
  border: 1.5px solid var(--sep); border-radius: var(--radius-sm);
  font-size: 16px; transition: border var(--transition), box-shadow var(--transition);
  background: #fafafa;
}
.field input:focus {
  border-color: var(--blue);
  background: #fff;
  box-shadow: 0 0 0 3px var(--blue-light);
}
.err {
  color: var(--red); font-size: 13px;
  margin: -4px 0 14px;
}
.btn-primary {
  width: 100%; padding: 13px;
  background: var(--blue); color: #fff;
  border-radius: var(--radius-sm);
  font-size: 16px; font-weight: 600;
  transition: opacity var(--transition);
  cursor: pointer;
}
.btn-primary:hover:not(:disabled) { opacity: 0.88; }
.btn-primary:active:not(:disabled) { opacity: 0.72; }
.btn-primary:disabled { opacity: 0.5; cursor: default; }
.switch {
  margin-top: 18px; font-size: 14px; color: var(--text2);
}
.switch a {
  color: var(--blue); cursor: pointer;
  font-weight: 500;
}
.switch a:hover { text-decoration: underline; }
</style>
