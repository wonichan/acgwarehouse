/**
 * Auth state management composable
 * Provides reactive auth state and login/logout methods
 */

import { ref, computed } from 'vue'
import { 
  login as apiLogin,
  register as apiRegister,
  getCurrentUser,
  setToken,
  clearToken,
  isAuthenticated,
  ApiError
} from '@/api/client'
import type { UserResponse } from '@/api/client'

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
      // Token invalid, clear it
      clearToken()
      user.value = null
    }
    loading.value = false
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
    loading.value = false
    return true
  } catch (e) {
    loading.value = false
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '登录失败，请稍后重试'
    }
    return false
  }
}

// Register function
async function register(username: string, password: string): Promise<boolean> {
  loading.value = true
  error.value = null
  
  try {
    const newUser = await apiRegister(username, password)
    // Auto login after register
    const result = await apiLogin(username, password)
    setToken(result.token)
    user.value = newUser
    loading.value = false
    return true
  } catch (e) {
    loading.value = false
    if (e instanceof ApiError) {
      error.value = e.message
    } else {
      error.value = '注册失败，请稍后重试'
    }
    return false
  }
}

// Logout function
function logout() {
  clearToken()
  user.value = null
  error.value = null
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
  }
}