import { ref, reactive } from 'vue'

// 简单的响应式认证状态，全局共享
const token = ref(localStorage.getItem('token') || '')
const username = ref(localStorage.getItem('username') || '')

export function useAuth() {
  const isLoggedIn = () => !!token.value

  const login = (t, u) => {
    token.value = t
    username.value = u
    localStorage.setItem('token', t)
    localStorage.setItem('username', u)
  }

  const logout = () => {
    token.value = ''
    username.value = ''
    localStorage.removeItem('token')
    localStorage.removeItem('username')
  }

  const authHeaders = () => ({
    'Content-Type': 'application/json',
    Authorization: 'Bearer ' + token.value,
  })

  return { token, username, isLoggedIn, login, logout, authHeaders }
}
