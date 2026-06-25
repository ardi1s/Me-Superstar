<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuth } from '../stores/auth.js'

const { token, username, logout, authHeaders } = useAuth()

// ---- 数据状态 ----
const period = ref('7d')
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const accounts = ref([])
const selectedAccountID = ref(0)
const ranking = ref([])
const pageSize = 20

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// ---- 数据请求 ----
async function fetchAccounts() {
  try {
    const res = await fetch('/api/v1/accounts', { headers: authHeaders() })
    const data = await res.json()
    if (data.code === 200) accounts.value = data.data || []
  } catch (e) { /* 静默失败 */ }
}

async function fetchRanking() {
  loading.value = true
  try {
    let url = `/api/v1/works/top-fans?period=${period.value}&page=${page.value}&page_size=${pageSize}`
    if (selectedAccountID.value > 0) url += `&account_id=${selectedAccountID.value}`
    const res = await fetch(url, { headers: authHeaders() })
    const data = await res.json()
    if (data.code === 200 && data.data) {
      ranking.value = data.data.items || []
      total.value = data.data.total || 0
    } else {
      ranking.value = []
      total.value = 0
    }
  } catch (e) {
    ranking.value = []
    total.value = 0
  }
  loading.value = false
}

// ---- 交互（关键：每个操作独立重置状态再请求）----
function switchPeriod(p) {
  period.value = p
  page.value = 1
  fetchRanking()
}

function switchAccount() {
  page.value = 1
  fetchRanking()
}

function goPage(p) {
  if (p < 1 || p > totalPages.value) return
  page.value = p
  fetchRanking()
}

function doLogout() {
  logout()
  accounts.value = []
  ranking.value = []
}

// ---- 抖音授权入口 ----
function bindDouyin() {
  const w = window.open(
    `/api/v1/auth/douyin?token=${encodeURIComponent(token.value)}`,
    'douyin-auth',
    'width=600,height=700'
  )
  // 监听弹窗关闭后刷新账号列表
  const timer = setInterval(() => {
    if (w.closed) {
      clearInterval(timer)
      fetchAccounts()
    }
  }, 500)
}

// ---- 格式化 ----
function fmtNum(n) {
  if (!n && n !== 0) return '0'
  if (n >= 100000) return (n / 10000).toFixed(1) + '万'
  return Number(n).toLocaleString()
}

function fmtDate(s) {
  if (!s) return '-'
  const d = new Date(s)
  if (isNaN(d.getTime())) return '-'
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function imgFallback(e) {
  e.target.src = 'data:image/svg+xml,' + encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="52" height="70">' +
    '<rect fill="#e5e5ea" width="52" height="70" rx="6"/>' +
    '<text x="26" y="40" text-anchor="middle" fill="#aeaeb2" font-size="11" font-family="sans-serif">暂无</text>' +
    '</svg>'
  )
}

onMounted(() => {
  fetchAccounts()
  fetchRanking()
})
</script>

<template>
  <div class="app">
    <!-- 导航栏 -->
    <nav>
      <h1>📊 作品涨粉排行榜</h1>
      <div class="user-area">
        <button class="btn-primary" @click="bindDouyin">绑定抖音</button>
        <div class="avatar">{{ (username || 'U')[0].toUpperCase() }}</div>
        <span class="name">{{ username }}</span>
        <button class="btn-ghost" @click="doLogout">退出登录</button>
      </div>
    </nav>

    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="segmented">
        <button
          v-for="p in periods"
          :key="p.key"
          :class="{ active: period === p.key }"
          @click="switchPeriod(p.key)"
        >{{ p.label }}</button>
      </div>
      <select class="select-native" v-model="selectedAccountID" @change="switchAccount">
        <option :value="0">全部账号</option>
        <option v-for="acc in accounts" :key="acc.id" :value="acc.id">
          {{ acc.platform }} — {{ acc.platform_account_id }}
        </option>
      </select>
    </div>

    <!-- 表格卡片 -->
    <div class="card">
      <!-- 加载 -->
      <div v-if="loading" class="state">
        <div class="spinner"></div>
        <p>加载中…</p>
      </div>

      <!-- 空态 -->
      <div v-else-if="ranking.length === 0" class="state">
        <div class="state-icon">📭</div>
        <p>暂无排行数据</p>
        <p class="hint">完成抖音授权后，系统会自动同步作品数据</p>
      </div>

      <!-- 数据表格 -->
      <table v-else>
        <thead>
          <tr>
            <th style="width:68px"></th>
            <th>作品</th>
            <th>发布时间</th>
            <th>播放量</th>
            <th>涨粉</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in ranking" :key="item.work_id">
            <td class="cover-td">
              <img :src="item.cover_url" :alt="item.title" @error="imgFallback" />
            </td>
            <td class="title-td" :title="item.title">
              {{ item.title || '(无标题)' }}
            </td>
            <td class="muted-td">{{ fmtDate(item.publish_time) }}</td>
            <td class="num-td">{{ fmtNum(item.total_play_count) }}</td>
            <td class="fans-td">+{{ fmtNum(item.total_fans_added) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 分页 -->
    <div class="pager" v-if="total > 0 && !loading">
      <button :disabled="page <= 1" @click="goPage(page - 1)">← 上一页</button>
      <span>第 {{ page }} / {{ totalPages }} 页</span>
      <button :disabled="page >= totalPages" @click="goPage(page + 1)">下一页 →</button>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      periods: [
        { key: '1d', label: '昨日' },
        { key: '7d', label: '近 7 天' },
        { key: '30d', label: '近 30 天' },
      ],
    }
  },
}
</script>

<style scoped>
/* ====== 布局 ====== */
.app {
  max-width: 1100px;
  margin: 0 auto;
  padding: 36px 24px 80px;
}

/* ====== 导航 ====== */
nav {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 36px;
}
nav h1 {
  font-size: 32px; font-weight: 700;
  letter-spacing: -0.5px;
}
.user-area {
  display: flex; align-items: center; gap: 14px;
}
.avatar {
  width: 34px; height: 34px; border-radius: 50%;
  background: linear-gradient(135deg, #667eea, #764ba2);
  display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 15px; font-weight: 700;
}
.name {
  font-size: 15px; color: var(--text2);
}
.btn-primary {
  padding: 7px 18px;
  background: var(--blue); color: #fff;
  border-radius: 20px; font-size: 13px; font-weight: 600;
  transition: opacity var(--transition);
}
.btn-primary:hover { opacity: 0.88; }
.btn-primary:active { opacity: 0.72; }

.btn-ghost {
  padding: 7px 18px;
  background: var(--card); border: 1.5px solid var(--sep);
  border-radius: 20px; font-size: 13px; font-weight: 500;
  color: var(--text2); transition: all var(--transition);
}
.btn-ghost:hover {
  background: #f0f0f0; color: var(--text);
}

/* ====== 工具栏 ====== */
.toolbar {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 24px; flex-wrap: wrap; gap: 16px;
}

/* Segmented Control */
.segmented {
  display: flex; gap: 2px;
  background: #e5e5ea; border-radius: 10px;
  padding: 2px;
}
.segmented button {
  padding: 9px 24px;
  border-radius: 8px; background: transparent;
  font-size: 14px; font-weight: 500; color: var(--text);
  transition: all var(--transition);
  white-space: nowrap;
}
.segmented button.active {
  background: #fff;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08), 0 1px 0 rgba(0,0,0,0.04);
}
.segmented button:not(.active):hover { color: #000; }

/* 原生下拉 */
.select-native {
  padding: 10px 34px 10px 14px;
  border: 1.5px solid var(--sep); border-radius: var(--radius-sm);
  font-size: 14px; min-width: 220px;
  background: var(--card); color: var(--text);
  appearance: none; -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg width='10' height='6'%3E%3Cpath d='M1 1l4 4 4-4' stroke='%2386868b' stroke-width='1.5' fill='none'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 14px center;
}

/* ====== 卡片 ====== */
.card {
  background: var(--card); border-radius: var(--radius);
  box-shadow: var(--shadow-sm); overflow: hidden;
}

/* ====== 表格 ====== */
table { width: 100%; border-collapse: collapse; }
th {
  text-align: left; padding: 14px 20px;
  font-size: 11px; font-weight: 700; color: var(--text2);
  text-transform: uppercase; letter-spacing: 0.8px;
  border-bottom: 1px solid var(--sep);
  background: #fafafa;
}
td { padding: 14px 20px; border-bottom: 1px solid #f0f0f0; }
tr:last-child td { border-bottom: none; }
tr { transition: background 0.15s; }
tr:hover td { background: #fafafa; }

.cover-td img {
  width: 52px; height: 70px; object-fit: cover;
  border-radius: 6px; background: #f0f0f0; display: block;
}
.title-td {
  max-width: 360px;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  font-weight: 500; font-size: 15px;
}
.muted-td {
  font-size: 13px; color: var(--text2); white-space: nowrap;
}
.num-td {
  font-size: 14px; color: var(--text2);
  font-variant-numeric: tabular-nums; white-space: nowrap;
}
.fans-td {
  font-size: 16px; font-weight: 700; color: var(--blue);
  font-variant-numeric: tabular-nums; white-space: nowrap;
}

/* ====== 状态 ====== */
.state {
  text-align: center; padding: 72px 20px;
}
.state-icon { font-size: 48px; margin-bottom: 12px; }
.state p { font-size: 15px; color: var(--text2); }
.state .hint { font-size: 13px; color: var(--text3); margin-top: 4px; }

.spinner {
  width: 28px; height: 28px; margin: 0 auto 14px;
  border: 3px solid #e5e5ea;
  border-top-color: var(--blue);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* ====== 分页 ====== */
.pager {
  display: flex; align-items: center; justify-content: center;
  gap: 8px; padding: 24px 0 0;
}
.pager button {
  min-width: 40px; height: 36px; padding: 0 16px;
  border: 1.5px solid var(--sep); border-radius: 8px;
  background: var(--card); font-size: 13px; font-weight: 500;
  color: var(--text); transition: all var(--transition);
}
.pager button:disabled { opacity: 0.3; cursor: default; }
.pager button:not(:disabled):hover { background: #f0f0f0; }
.pager button:not(:disabled):active { background: #e5e5ea; }
.pager span { font-size: 13px; color: var(--text2); margin: 0 6px; }

/* ====== 响应式 ====== */
@media (max-width: 700px) {
  nav { flex-direction: column; align-items: flex-start; gap: 12px; }
  .toolbar { flex-direction: column; align-items: stretch; }
  .select-native { width: 100%; }
  th, td { padding: 12px 14px; }
}
</style>
