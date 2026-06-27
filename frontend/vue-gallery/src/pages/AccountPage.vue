<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useTabs } from '@/composables/useTabs'
import { useToast } from '@/composables/useToast'

const { activeTab, isActive, switchTab } = useTabs(['login', 'register'], 'login')
const { show } = useToast()

// Motion setup
onMounted(() => {
  document.documentElement.classList.add('motion-ready')
})

// Login form
const loginEmail = ref('mio@example.com')
const loginPassword = ref('password')

// Register form
const registerNickname = ref('新来的收藏家')
const registerEmail = ref('')
const registerPassword = ref('')

// Profile form
const profileNickname = ref('澪音收藏家')
const profileTags = ref('雨景, 制服, 场景, 暖光')
const profileBio = ref('喜欢整理可复用的 ACG 场景与角色参考，优先收藏高评分作品。')

// Preferences
const prefPublic = ref(true)
const prefEmail = ref(true)
const prefSync = ref(true)

// Form handlers
const handleLogin = () => show('登录状态已更新')
const handleRegister = () => show('注册信息已提交')
const handleSaveProfile = () => show('资料已保存')

// Keyboard navigation for tabs
const handleTabKeydown = (event: KeyboardEvent) => {
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
        <aside class="panel panel-raised stack">
          <div class="avatar-xl" aria-label="用户头像">澪</div>
          <div>
            <p class="eyebrow">个人资料</p>
            <h2>澪音收藏家</h2>
            <p class="meta">偏好：雨景、头像、角色设定。已创建 6 个相册，收藏 128 张作品。</p>
          </div>
          <div class="grid-2">
            <div class="panel">
              <strong class="num">128</strong>
              <p class="meta">收藏</p>
            </div>
            <div class="panel">
              <strong class="num">42</strong>
              <p class="meta">标签</p>
            </div>
          </div>
          <div class="status-row">
            <span class="status-badge status-synced">
              <span class="status-dot" aria-hidden="true"></span>
              同步正常
            </span>
            <span class="status-badge status-secure">
              <span class="status-dot" aria-hidden="true"></span>
              账户安全
            </span>
          </div>
        </aside>

        <section class="stack">
          <!-- Auth Tabs -->
          <div class="panel panel-raised">
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
                <label for="login-email">
                  <span class="label-text">邮箱 <span class="required" aria-hidden="true">*</span></span>
                </label>
                <input id="login-email" class="input" type="email" v-model="loginEmail" required autocomplete="email" />
              </div>
              <div class="field">
                <label for="login-password">
                  <span class="label-text">密码 <span class="required" aria-hidden="true">*</span></span>
                </label>
                <input id="login-password" class="input" type="password" v-model="loginPassword" required autocomplete="current-password" />
              </div>
              <div class="form-actions">
                <button class="btn btn-primary" type="button" @click="handleLogin">登录并同步收藏</button>
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
                <label for="register-nickname">
                  <span class="label-text">昵称 <span class="required" aria-hidden="true">*</span></span>
                </label>
                <input id="register-nickname" class="input" v-model="registerNickname" required maxlength="20" autocomplete="off" />
              </div>
              <div class="field">
                <label for="register-email">
                  <span class="label-text">邮箱 <span class="required" aria-hidden="true">*</span></span>
                </label>
                <input id="register-email" class="input" type="email" v-model="registerEmail" required autocomplete="email" />
              </div>
              <div class="field">
                <label for="register-password">
                  <span class="label-text">密码 <span class="required" aria-hidden="true">*</span></span>
                </label>
                <input id="register-password" class="input" type="password" v-model="registerPassword" required minlength="8" autocomplete="new-password" />
              </div>
              <div class="form-actions">
                <button class="btn btn-primary" type="button" @click="handleRegister">创建账户</button>
              </div>
            </div>
          </div>

          <!-- Profile Panel -->
          <div class="panel">
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
          <div class="panel">
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
          <div class="panel">
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
                  <p class="security-row__hint">上次修改于 2024 年 12 月</p>
                </div>
                <button class="btn btn-secondary btn-small" type="button">修改</button>
              </div>
              <div class="security-row">
                <div>
                  <p class="security-row__label">双重验证</p>
                  <p class="security-row__hint">通过邮箱验证码保护账户登录</p>
                </div>
                <span class="tag">已开启</span>
              </div>
              <div class="security-row">
                <div>
                  <p class="security-row__label">活跃会话</p>
                  <p class="security-row__hint">当前 2 个设备已登录</p>
                </div>
                <button class="btn btn-ghost btn-small" type="button">管理</button>
              </div>
            </div>
          </div>

          <!-- Activity Panel -->
          <div class="panel">
            <div class="panel-head">
              <div>
                <p class="eyebrow">最近活动</p>
                <h3>账户动态</h3>
              </div>
              <span class="tag">仅自己可见</span>
            </div>
            <div class="activity-empty" role="status">
              <div class="activity-empty__icon" aria-hidden="true">
                <svg viewBox="0 0 48 48" width="48" height="48">
                  <rect x="12" y="14" width="24" height="20" fill="none" stroke="currentColor" stroke-width="3" rx="4" />
                  <polyline points="16,30 22,23 27,28 31,24 36,30" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
              </div>
              <p class="activity-empty__title">暂无新的账户动态</p>
              <p class="activity-empty__desc">收藏、打分或批量标记图片后，这里会显示最近的同步记录。</p>
            </div>
          </div>
        </section>
      </div>
    </section>
  </main>
</template>