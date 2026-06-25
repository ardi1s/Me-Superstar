<script setup>
import { useAuth } from './stores/auth.js'
import LoginView from './views/LoginView.vue'
import DashboardView from './views/DashboardView.vue'

const { isLoggedIn } = useAuth()
</script>

<template>
  <LoginView v-if="!isLoggedIn()" />
  <DashboardView v-else />
</template>
