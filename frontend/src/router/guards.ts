import type { Router } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/authStore'

export function setupGuards(router: Router) {
  router.beforeEach((to) => {
    const authStore = useAuthStore()
    if (to.meta.requiresAuth && !authStore.token) {
      return { path: '/login', query: { redirect: to.fullPath } }
    }
    if (to.meta.requiresAdmin && !authStore.isAdmin()) {
      ElMessage.warning('仅管理员可访问')
      return { path: '/products' }
    }
    return true
  })
}
