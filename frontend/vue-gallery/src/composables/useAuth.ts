/**
 * Auth state management composable
 * Provides reactive auth state and login/logout methods
 */

import { ref, computed } from 'vue'
import {
  login as apiLogin,
  register as apiRegister,
  getCurrentUser,
  updateCurrentUserProfile,
  setToken,
  clearToken,
  isAuthenticated,
  ApiError,
} from '@/api/client'
import type { UserProfileUpdateRequest, UserResponse } from '@/api/client'

// Reactive state
const user = ref<UserResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

// Initialize user from token if exists
async function initAuth() {
  if (isAuthenticated()) {
    loading.value = true
    try {
      user.value = await getCurrentUser()
    } catch (e) {
      if (e instanceof ApiError) {
        error.value = e.message
      } else {
        error.value = '登录状态已失效，请重新登录'
      }
      clearToken()
      user.value = null
    } finally {
      loading.value = false
    }
  }
}

// Login function
async function login(username: string, password: string): Promise<boolean> {
  loading.value = true
  error.value = null

  try {
    const result = await apiLogin(username, password)
    setToken(result.token)
    user.value = await getCurrentUser()
    return true
  } catch (e) {
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '登录失败，请稍后重试'
    }
    return false
  } finally {
    loading.value = false
  }
}

// Register function
async function register(username: string, password: string): Promise<boolean> {
  loading.value = true
  error.value = null

  try {
    await apiRegister(username, password)
    const result = await apiLogin(username, password)
    setToken(result.token)
    user.value = await getCurrentUser()
    return true
  } catch (e) {
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '注册失败，请稍后重试'
    }
    return false
  } finally {
    loading.value = false
  }
}

// Logout function
function logout() {
  clearToken()
  user.value = null
  error.value = null
}

async function refreshCurrentUser(): Promise<boolean> {
  if (!isAuthenticated()) {
    user.value = null
    return false
  }
  loading.value = true
  error.value = null
  try {
    user.value = await getCurrentUser()
    return true
  } catch (e) {
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '刷新用户信息失败'
    }
    clearToken()
    user.value = null
    return false
  } finally {
    loading.value = false
  }
}

async function updateProfile(input: UserProfileUpdateRequest): Promise<boolean> {
  loading.value = true
  error.value = null
  try {
    user.value = await updateCurrentUserProfile(input)
    return true
  } catch (e) {
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '保存资料失败'
    }
    return false
  } finally {
    loading.value = false
  }
}

// Computed properties
const isLoggedIn = computed(() => user.value !== null)
const isAdmin = computed(() => user.value?.role === 'admin')

// Export composable
export function useAuth() {
  // Initialize on first use
  if (isAuthenticated() && !user.value && !loading.value) {
    initAuth()
  }
  
  return {
    user,
    loading,
    error,
    isLoggedIn,
    isAdmin,
    login,
    register,
    logout,
    initAuth,
    refreshCurrentUser,
    updateProfile,
  }
}