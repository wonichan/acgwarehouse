<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useTabs } from '@/composables/useTabs'
import { useToast } from '@/composables/useToast'
import { useAuth } from '@/composables/useAuth'

const { activeTab, isActive, switchTab } = useTabs(['login', 'register'], 'login')
const { show } = useToast()
const { user, loading, error, isLoggedIn, login, register, logout, initAuth } = useAuth()

// Motion setup
onMounted(() => {
  document.documentElement.classList.add('motion-ready')
  initAuth()
})

// Login form
const loginUsername = ref('')
const loginPassword = ref('')

// Register form
const registerUsername = ref('')
const registerPassword = ref('')

// Profile form (placeholder - no backend API yet)
const profileNickname = ref('')
const profileTags = ref('雨景, 制服, 场景, 暖光')
const profileBio = ref('喜欢整理可复用的 ACG 场景与角色参考，优先收藏高评分作品。')

// Preferences (placeholder)
const prefPublic = ref(true)
const prefEmail = ref(true)
const prefSync = ref(true)

// Form handlers
async function handleLogin() {
  const success = await login(loginUsername.value, loginPassword.value)
  if (success) {
    show('登录成功')
    loginUsername.value = ''
    loginPassword.value = ''
  } else if (error.value) {
    show(error.value)
  }
}

async function handleRegister() {
  const success = await register(registerUsername.value, registerPassword.value)
  if (success) {
    show('注册成功')
    registerUsername.value = ''
    registerPassword.value = ''
  } else if (error.value) {
    show(error.value)
  }
}

function handleLogout() {
  logout()
  show('已退出登录')
}

function handleSaveProfile() {
  show('资料已保存（功能开发中）')
}

// Keyboard navigation for tabs
function handleTabKeydown(event: KeyboardEvent) {
  const allTabs: ('login' | 'register')[] = ['login', 'register']
  const currentIndex = allTabs.indexOf(activeTab.value)
  let nextIndex = currentIndex
  if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
    nextIndex = (currentIndex + 1) % allTabs.length
    event.preventDefault()
  } else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
    nextIndex = (currentIndex - 1 + allTabs.length) % allTabs.length
    event.preventDefault()
  }
  if (nextIndex !== currentIndex) {
    switchTab(allTabs[nextIndex])
  }
}
</script>

<template>
  <main>
    <section class="section" data-od-id="account-page" aria-labelledby="account-title">
      <div class="container account-intro">
        <p class="eyebrow">账户中心</p>
        <h1 id="account-title">管理你的社区身份与收藏同步</h1>
        <p class="lead">登录、资料、偏好与安全集中在一个页面，确保收藏夹、标签和评分在 Web 端持续同步。</p>
      </div>

      <div class="container profile-grid">
        <!-- User Profile Sidebar -->
        <aside class="panel panel-raised stack">
          <div class="avatar-xl" aria-label="用户头像">{{ user?.username?.charAt(0) || '?' }}</div>
          <div>
            <p class="eyebrow">个人资料</p>
            <h2>{{ user?.username || '未登录' }}</h2>
            <p class="meta">
              {{ isLoggedIn ? `角色：${user?.role} · 注册于 ${user?.created_at}` : '登录后可查看收藏和同步数据' }}
            </p>
          </div>
          <div v-if="isLoggedIn" class="grid-2">
            <div class="panel">
              <strong class="num">0</strong>
              <p class="meta">收藏</p>
            </div>
            <div class="panel">
              <strong class="num">0</strong>
              <p class="meta">标签</p>
            </div>
          </div>
          <div v-if="isLoggedIn" class="status-row">
            <span class="status-badge status-synced">
              <span class="status-dot" aria-hidden="true"></span>
              已登录
            </span>
            <span class="status-badge status-secure">
              <span class="status-dot" aria-hidden="true"></span>
              账户安全
            </span>
          </div>
          <button v-if="isLoggedIn" class="btn btn-secondary btn-small" @click="handleLogout">退出登录</button>
        </aside>

        <section class="stack">
          <!-- Auth Tabs (show when not logged in) -->
          <div v-if="!isLoggedIn" class="panel panel-raised">
            <div class="auth-tabs" role="tablist" aria-label="账户登录与注册">
              <button
                type="button"
                role="tab"
                :class="{ 'is-active': isActive('login') }"
                :aria-selected="isActive('login')"
                aria-controls="pane-login"
                :tabindex="isActive('login') ? 0 : -1"
                @click="switchTab('login')"
                @keydown="handleTabKeydown"
              >
                登录
              </button>
              <button
                type="button"
                role="tab"
                :class="{ 'is-active': isActive('register') }"
                :aria-selected="isActive('register')"
                aria-controls="pane-register"
                :tabindex="isActive('register') ? 0 : -1"
                @click="switchTab('register')"
                @keydown="handleTabKeydown"
              >
                注册
              </button>
            </div>

            <!-- Login Pane -->
            <div
              id="pane-login"
              class="auth-pane form-grid"
              :class="{ 'is-active': isActive('login') }"
              role="tabpanel"
              aria-labelledby="tab-login"
              :aria-hidden="!isActive('login')"
            >
              <div class="field">
                <label for="login-username">
                  <span class="label-text">用户名 <span class="required" aria-hidden="true">*</span></span>
                </label>
                <input 
                  id="login-username" 
                  class="input" 
                  type="text" 
                  v-model="loginUsername" 
                  required 
                  autocomplete="username"
                  placeholder="请输入用户名"
                />
              </div>
              <div class="field">
                <label for="login-password">
                  <span class="label-text">密码 <span class="required" aria-hidden="true">*</span></span>
                </label>
                <input 
                  id="login-password" 
                  class="input" 
                  type="password" 
                  v-model="loginPassword" 
                  required 
                  autocomplete="current-password"
                  placeholder="请输入密码"
                />
              </div>
              <div class="form-actions">
                <button 
                  class="btn btn-primary" 
                  type="button" 
                  @click="handleLogin"
                  :disabled="loading"
                >
                  {{ loading ? '登录中...' : '登录' }}
                </button>
              </div>
            </div>

            <!-- Register Pane -->
            <div
              id="pane-register"
              class="auth-pane form-grid"
              :class="{ 'is-active': isActive('register') }"
              role="tabpanel"
              aria-labelledby="tab-register"
              :aria-hidden="!isActive('register')"
              v-show="isActive('register')"
            >
              <div class="field">
                <label for="register-username">
                  <span class="label-text">用户名 <span class="required" aria-hidden="true">*</span></span>
                </label>
                <input 
                  id="register-username" 
                  class="input" 
                  v-model="registerUsername" 
                  required 
                  minlength="3"
                  maxlength="32"
                  autocomplete="username"
                  placeholder="3-32个字符"
                />
              </div>
              <div class="field">
                <label for="register-password">
                  <span class="label-text">密码 <span class="required" aria-hidden="true">*</span></span>
                </label>
                <input 
                  id="register-password" 
                  class="input" 
                  type="password" 
                  v-model="registerPassword" 
                  required 
                  minlength="6"
                  autocomplete="new-password"
                  placeholder="至少6个字符"
                />
              </div>
              <div class="form-actions">
                <button 
                  class="btn btn-primary" 
                  type="button" 
                  @click="handleRegister"
                  :disabled="loading"
                >
                  {{ loading ? '注册中...' : '创建账户' }}
                </button>
              </div>
            </div>
          </div>

          <!-- Logged in message -->
          <div v-if="isLoggedIn" class="panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">登录状态</p>
                <h3>已登录</h3>
              </div>
            </div>
            <p class="lead">你已成功登录，可以浏览收藏夹、给作品打标签和评分。</p>
            <RouterLink class="btn btn-primary" to="/collections">查看我的收藏</RouterLink>
          </div>

          <!-- Profile Panel (placeholder for logged in users) -->
          <div v-if="isLoggedIn" class="panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">资料编辑</p>
                <h3>社区展示信息</h3>
              </div>
              <span class="tag">公开资料</span>
            </div>
            <div class="form-grid">
              <div class="field">
                <label for="profile-nickname"><span class="label-text">显示昵称</span></label>
                <input id="profile-nickname" class="input" v-model="profileNickname" maxlength="20" autocomplete="off" />
              </div>
              <div class="field">
                <label for="profile-tags"><span class="label-text">常用标签</span></label>
                <input id="profile-tags" class="input" v-model="profileTags" autocomplete="off" />
              </div>
              <div class="field">
                <label for="profile-bio"><span class="label-text">个人简介</span></label>
                <textarea id="profile-bio" class="textarea" v-model="profileBio" maxlength="200"></textarea>
              </div>
              <div class="form-actions">
                <button class="btn btn-primary" type="button" @click="handleSaveProfile">保存个人资料</button>
              </div>
            </div>
          </div>

          <!-- Preferences Panel -->
          <div v-if="isLoggedIn" class="panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">偏好设置</p>
                <h3>个性化与通知</h3>
              </div>
            </div>
            <div class="preference-list">
              <div class="preference-row">
                <div>
                  <p class="preference-row__label">公开个人资料</p>
                  <p class="preference-row__hint">其他用户可在社区中查看你的收藏与标签</p>
                </div>
                <label class="toggle-control">
                  <span class="sr-only">公开个人资料</span>
                  <input class="toggle" type="checkbox" v-model="prefPublic" />
                </label>
              </div>
              <div class="preference-row">
                <div>
                  <p class="preference-row__label">收藏更新邮件通知</p>
                  <p class="preference-row__hint">当关注的标签有新作品时发送邮件</p>
                </div>
                <label class="toggle-control">
                  <span class="sr-only">收藏更新邮件通知</span>
                  <input class="toggle" type="checkbox" v-model="prefEmail" />
                </label>
              </div>
              <div class="preference-row">
                <div>
                  <p class="preference-row__label">同步收藏到云端</p>
                  <p class="preference-row__hint">在多台设备间保持收藏与标签同步</p>
                </div>
                <label class="toggle-control">
                  <span class="sr-only">同步收藏到云端</span>
                  <input class="toggle" type="checkbox" v-model="prefSync" />
                </label>
              </div>
            </div>
          </div>

          <!-- Security Panel -->
          <div v-if="isLoggedIn" class="panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">账户安全</p>
                <h3>安全状态</h3>
              </div>
            </div>
            <div class="account-security">
              <div class="security-row">
                <div>
                  <p class="security-row__label">登录密码</p>
                  <p class="security-row__hint">定期修改密码保护账户安全</p>
                </div>
                <button class="btn btn-secondary btn-small" type="button">修改（开发中）</button>
              </div>
              <div class="security-row">
                <div>
                  <p class="security-row__label">JWT令牌</p>
                  <p class="security-row__hint">当前会话已通过JWT验证</p>
                </div>
                <span class="tag">已开启</span>
              </div>
            </div>
          </div>
        </section>
      </div>
    </section>
  </main>
</template>