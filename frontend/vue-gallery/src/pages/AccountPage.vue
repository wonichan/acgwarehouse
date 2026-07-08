<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { ApiError, changeCurrentUserPassword } from '@/api/client'
import { useTabs } from '@/composables/useTabs'
import { useToast } from '@/composables/useToast'
import { useAuth } from '@/composables/useAuth'
import CheckInCalendar from '@/components/CheckInCalendar.vue'
import {
  setStatus,
  statusClass,
  validateCredential,
  validatePasswordFields,
  validateProfileFields,
} from './accountForms'
import type { FieldErrors, FormStatus } from './accountForms'
import type { UserProfileUpdateRequest } from '@/api/client'

type AuthTab = 'login' | 'register'

const authTabs: AuthTab[] = ['login', 'register']
const { activeTab, isActive, switchTab } = useTabs<AuthTab>(authTabs, 'login')
const { show } = useToast()
const { user, loading, error, isLoggedIn, login, register, logout, initAuth, updateProfile } = useAuth()

const loginUsername = ref('')
const loginPassword = ref('')
const registerUsername = ref('')
const registerPassword = ref('')
const profileNickname = ref('')
const profileTags = ref('')
const profileBio = ref('')
const prefPublic = ref(true)
const prefEmail = ref(true)
const prefSync = ref(true)
const oldPassword = ref('')
const newPassword = ref('')
const passwordBusy = ref(false)

const loginErrors = reactive<FieldErrors>({ username: '', password: '' })
const registerErrors = reactive<FieldErrors>({ username: '', password: '' })
const profileErrors = reactive<FieldErrors>({ nickname: '', tags: '', bio: '' })
const passwordErrors = reactive<FieldErrors>({ old: '', next: '' })
const authStatus = reactive<FormStatus>({ message: '', tone: 'idle' })
const profileStatus = reactive<FormStatus>({ message: '', tone: 'idle' })
const preferenceStatus = reactive<FormStatus>({ message: '', tone: 'idle' })
const passwordStatus = reactive<FormStatus>({ message: '', tone: 'idle' })

const displayName = computed(() => user.value?.nickname || user.value?.username || '未登录')
const accountInitial = computed(() => displayName.value.trim().charAt(0) || '?')
const accountRole = computed(() => (user.value?.role === 'admin' ? '管理员' : '普通用户'))
const formattedCreatedAt = computed(() => {
  const raw = user.value?.created_at
  if (!raw) return '登录后显示注册时间'
  const date = new Date(raw)
  return Number.isNaN(date.getTime()) ? raw : date.toLocaleDateString('zh-CN')
})

onMounted(() => {
  initAuth()
})

watch(user, (nextUser) => {
  if (!nextUser) return
  profileNickname.value = nextUser.nickname || nextUser.username
  profileTags.value = nextUser.favorite_tags
  profileBio.value = nextUser.bio
  prefPublic.value = nextUser.public_profile
  prefEmail.value = nextUser.email_notifications
  prefSync.value = nextUser.sync_collections
}, { immediate: true })

function validateProfile(): boolean {
  return validateProfileFields(profileNickname.value, profileTags.value, profileBio.value, profileErrors)
}

function validatePassword(): boolean {
  return validatePasswordFields(oldPassword.value, newPassword.value, passwordErrors)
}

function profilePayload(): UserProfileUpdateRequest {
  return {
    nickname: profileNickname.value.trim(),
    favorite_tags: profileTags.value.trim(),
    bio: profileBio.value.trim(),
    public_profile: prefPublic.value,
    email_notifications: prefEmail.value,
    sync_collections: prefSync.value,
  }
}

async function handleLogin(): Promise<void> {
  if (!validateCredential(loginUsername.value, loginPassword.value, loginErrors)) {
    setStatus(authStatus, '请先修正登录表单中的错误。', 'error')
    return
  }
  setStatus(authStatus, '正在登录并同步账户资料。', 'loading')
  const success = await login(loginUsername.value.trim(), loginPassword.value)
  if (success) {
    loginUsername.value = ''
    loginPassword.value = ''
    setStatus(authStatus, '登录成功，账户资料已同步。', 'success')
    show('登录成功')
    return
  }
  const message = error.value || '登录失败，请稍后重试'
  setStatus(authStatus, message, 'error')
  show(message)
}

async function handleRegister(): Promise<void> {
  if (!validateCredential(registerUsername.value, registerPassword.value, registerErrors)) {
    setStatus(authStatus, '请先修正注册表单中的错误。', 'error')
    return
  }
  setStatus(authStatus, '正在创建账户并自动登录。', 'loading')
  const success = await register(registerUsername.value.trim(), registerPassword.value)
  if (success) {
    registerUsername.value = ''
    registerPassword.value = ''
    setStatus(authStatus, '注册成功，已自动进入登录状态。', 'success')
    show('注册成功')
    return
  }
  const message = error.value || '注册失败，请稍后重试'
  setStatus(authStatus, message, 'error')
  show(message)
}

async function submitProfile(status: FormStatus): Promise<void> {
  if (!validateProfile()) {
    setStatus(status, '请先修正资料字段后再保存。', 'error')
    return
  }
  setStatus(status, '正在保存到后端。', 'loading')
  const success = await updateProfile(profilePayload())
  if (success) {
    setStatus(status, '保存成功，刷新后仍会保留。', 'success')
    show('保存成功')
    return
  }
  const message = error.value || '保存失败'
  setStatus(status, message, 'error')
  show(message)
}

async function submitPassword(): Promise<void> {
  if (!validatePassword()) {
    setStatus(passwordStatus, '请先修正密码表单中的错误。', 'error')
    return
  }
  passwordBusy.value = true
  setStatus(passwordStatus, '正在修改登录密码。', 'loading')
  try {
    await changeCurrentUserPassword({ old_password: oldPassword.value, new_password: newPassword.value })
    oldPassword.value = ''
    newPassword.value = ''
    setStatus(passwordStatus, '密码已修改，可继续使用当前会话。', 'success')
    show('密码已修改')
  } catch (e) {
    const message = e instanceof ApiError ? e.message : '密码修改失败'
    setStatus(passwordStatus, message, 'error')
    show(message)
  } finally {
    passwordBusy.value = false
  }
}

function handleLogout(): void {
  logout()
  setStatus(authStatus, '已退出登录。', 'success')
  show('已退出登录')
}

async function focusTab(tab: AuthTab): Promise<void> {
  await nextTick()
  document.getElementById(`tab-${tab}`)?.focus()
}

function handleTabKeydown(event: KeyboardEvent): void {
  const currentIndex = authTabs.indexOf(activeTab.value)
  let nextIndex = currentIndex
  if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
    nextIndex = (currentIndex + 1) % authTabs.length
    event.preventDefault()
  } else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
    nextIndex = (currentIndex - 1 + authTabs.length) % authTabs.length
    event.preventDefault()
  } else if (event.key === 'Home') {
    nextIndex = 0
    event.preventDefault()
  } else if (event.key === 'End') {
    nextIndex = authTabs.length - 1
    event.preventDefault()
  }
  const nextTab = authTabs[nextIndex]
  if (nextTab && nextIndex !== currentIndex) {
    switchTab(nextTab)
    focusTab(nextTab)
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
        <aside class="panel panel-raised stack" aria-labelledby="account-profile-title">
          <div class="avatar-xl" aria-label="用户头像">{{ accountInitial }}</div>
          <div>
            <p class="eyebrow">个人资料</p>
            <h2 id="account-profile-title">{{ displayName }}</h2>
            <p class="meta">{{ isLoggedIn ? `${accountRole} · 注册于 ${formattedCreatedAt}` : '登录后可查看资料、偏好和收藏同步状态' }}</p>
          </div>
          <div v-if="isLoggedIn" class="grid-3" aria-label="账户摘要">
            <div class="panel"><strong class="num">{{ user?.public_profile ? '公开' : '私密' }}</strong><p class="meta">资料</p></div>
            <div class="panel"><strong class="num">{{ user?.sync_collections ? '开启' : '关闭' }}</strong><p class="meta">同步</p></div>
            <div class="panel"><strong class="num">{{ user?.points ?? 0 }}</strong><p class="meta">积分</p></div>
          </div>
          <div v-if="isLoggedIn" class="status-row">
            <span class="status-badge status-synced"><span class="status-dot" aria-hidden="true"></span>{{ user?.sync_collections ? '同步正常' : '同步关闭' }}</span>
            <span class="status-badge status-secure"><span class="status-dot" aria-hidden="true"></span>账户安全</span>
          </div>
          <p v-else class="meta">未登录用户仍可浏览公开图库；保存收藏、评分和个人资料需要先登录。</p>
          <button v-if="isLoggedIn" class="btn btn-secondary btn-small" type="button" @click="handleLogout">退出登录</button>
        </aside>

        <section class="stack" aria-label="账户操作">
          <div v-if="!isLoggedIn" class="panel panel-raised">
            <div class="auth-tabs" role="tablist" aria-label="账户登录与注册">
              <button id="tab-login" type="button" role="tab" :class="{ 'is-active': isActive('login') }" :aria-selected="isActive('login')" aria-controls="pane-login" :tabindex="isActive('login') ? 0 : -1" @click="switchTab('login')" @keydown="handleTabKeydown">登录</button>
              <button id="tab-register" type="button" role="tab" :class="{ 'is-active': isActive('register') }" :aria-selected="isActive('register')" aria-controls="pane-register" :tabindex="isActive('register') ? 0 : -1" @click="switchTab('register')" @keydown="handleTabKeydown">注册</button>
            </div>

            <form id="pane-login" class="auth-pane form-grid" :class="{ 'is-active': isActive('login') }" role="tabpanel" aria-labelledby="tab-login" :aria-hidden="!isActive('login')" v-show="isActive('login')" @submit.prevent="handleLogin">
              <div class="field" :class="{ 'is-error': Boolean(loginErrors['username']) }"><label for="login-username"><span class="label-text">用户名 <span class="required" aria-hidden="true">*</span></span></label><input id="login-username" v-model="loginUsername" class="input" autocomplete="username" aria-describedby="login-username-hint login-username-error" :aria-invalid="Boolean(loginErrors['username'])" /><p id="login-username-hint" class="helper-text">使用注册时填写的用户名登录。</p><p v-if="loginErrors['username']" id="login-username-error" class="error-text">{{ loginErrors['username'] }}</p></div>
              <div class="field" :class="{ 'is-error': Boolean(loginErrors['password']) }"><label for="login-password"><span class="label-text">密码 <span class="required" aria-hidden="true">*</span></span></label><input id="login-password" v-model="loginPassword" class="input" type="password" autocomplete="current-password" aria-describedby="login-password-hint login-password-error" :aria-invalid="Boolean(loginErrors['password'])" /><p id="login-password-hint" class="helper-text">至少 6 个字符；登录成功后会同步当前资料。</p><p v-if="loginErrors['password']" id="login-password-error" class="error-text">{{ loginErrors['password'] }}</p></div>
              <div class="form-actions"><button class="btn btn-primary" type="submit" :disabled="loading">{{ loading ? '登录中...' : '登录并同步收藏' }}</button></div>
              <div class="status" :class="statusClass(authStatus)" role="alert" aria-live="assertive">{{ authStatus.message }}</div>
            </form>

            <form id="pane-register" class="auth-pane form-grid" :class="{ 'is-active': isActive('register') }" role="tabpanel" aria-labelledby="tab-register" :aria-hidden="!isActive('register')" v-show="isActive('register')" @submit.prevent="handleRegister">
              <div class="field" :class="{ 'is-error': Boolean(registerErrors['username']) }"><label for="register-username"><span class="label-text">用户名 <span class="required" aria-hidden="true">*</span></span></label><input id="register-username" v-model="registerUsername" class="input" autocomplete="username" minlength="3" maxlength="32" aria-describedby="register-username-hint register-username-error" :aria-invalid="Boolean(registerErrors['username'])" /><p id="register-username-hint" class="helper-text">3-32 个字符，注册成功后自动登录。</p><p v-if="registerErrors['username']" id="register-username-error" class="error-text">{{ registerErrors['username'] }}</p></div>
              <div class="field" :class="{ 'is-error': Boolean(registerErrors['password']) }"><label for="register-password"><span class="label-text">密码 <span class="required" aria-hidden="true">*</span></span></label><input id="register-password" v-model="registerPassword" class="input" type="password" autocomplete="new-password" minlength="6" aria-describedby="register-password-hint register-password-error" :aria-invalid="Boolean(registerErrors['password'])" /><p id="register-password-hint" class="helper-text">至少 6 个字符，建议包含字母与数字组合。</p><p v-if="registerErrors['password']" id="register-password-error" class="error-text">{{ registerErrors['password'] }}</p></div>
              <div class="form-actions"><button class="btn btn-primary" type="submit" :disabled="loading">{{ loading ? '注册中...' : '创建账户' }}</button></div>
              <div class="status" :class="statusClass(authStatus)" role="alert" aria-live="assertive">{{ authStatus.message }}</div>
            </form>
          </div>

          <div v-if="isLoggedIn" class="panel"><div class="panel-head"><div><p class="eyebrow">登录状态</p><h3>已登录</h3></div><span class="tag">{{ displayName }}</span></div><p class="lead">账户资料来自后端 /users/me，刷新页面后会通过已保存 token 自动恢复。</p></div>

          <CheckInCalendar v-if="isLoggedIn" />

          <div v-if="isLoggedIn" class="panel">
            <div class="panel-head"><div><p class="eyebrow">资料编辑</p><h3>社区展示信息</h3></div><span class="tag">公开资料</span></div>
            <form class="form-grid" @submit.prevent="submitProfile(profileStatus)">
              <div class="field" :class="{ 'is-error': Boolean(profileErrors['nickname']) }"><label for="profile-nickname"><span class="label-text">显示昵称 <span class="required" aria-hidden="true">*</span></span></label><input id="profile-nickname" v-model="profileNickname" class="input" maxlength="20" autocomplete="off" aria-describedby="profile-nickname-hint profile-nickname-error" :aria-invalid="Boolean(profileErrors['nickname'])" /><p id="profile-nickname-hint" class="helper-text">其他用户在图库与收藏夹中看到的名字。</p><p v-if="profileErrors['nickname']" id="profile-nickname-error" class="error-text">{{ profileErrors['nickname'] }}</p></div>
              <div class="field" :class="{ 'is-error': Boolean(profileErrors['tags']) }"><label for="profile-tags"><span class="label-text">常用标签</span></label><input id="profile-tags" v-model="profileTags" class="input" maxlength="120" autocomplete="off" aria-describedby="profile-tags-hint profile-tags-error" :aria-invalid="Boolean(profileErrors['tags'])" /><p id="profile-tags-hint" class="helper-text">用逗号分隔，最多 120 个字符。</p><p v-if="profileErrors['tags']" id="profile-tags-error" class="error-text">{{ profileErrors['tags'] }}</p></div>
              <div class="field" :class="{ 'is-error': Boolean(profileErrors['bio']) }"><label for="profile-bio"><span class="label-text">个人简介</span></label><textarea id="profile-bio" v-model="profileBio" class="textarea" maxlength="200" aria-describedby="profile-bio-hint profile-bio-error" :aria-invalid="Boolean(profileErrors['bio'])"></textarea><p id="profile-bio-hint" class="helper-text">最多 200 字，简述你的收藏偏好与创作方向。</p><p v-if="profileErrors['bio']" id="profile-bio-error" class="error-text">{{ profileErrors['bio'] }}</p></div>
              <div class="form-actions"><button class="btn btn-primary" type="submit" :disabled="loading">{{ loading ? '保存中...' : '保存个人资料' }}</button></div><div class="status" :class="statusClass(profileStatus)" role="status" aria-live="polite">{{ profileStatus.message }}</div>
            </form>
          </div>

          <div v-if="isLoggedIn" class="panel"><div class="panel-head"><div><p class="eyebrow">偏好设置</p><h3>个性化与通知</h3></div></div><form class="form-grid" @submit.prevent="submitProfile(preferenceStatus)"><div class="preference-list"><div class="preference-row"><div><p class="preference-row__label">公开个人资料</p><p class="preference-row__hint">其他用户可在社区中查看你的公开资料。</p></div><label class="toggle-control" for="pref-public"><span class="sr-only">公开个人资料</span><input id="pref-public" v-model="prefPublic" class="toggle" type="checkbox" /></label></div><div class="preference-row"><div><p class="preference-row__label">收藏更新邮件通知</p><p class="preference-row__hint">当关注标签有新作品时发送通知。</p></div><label class="toggle-control" for="pref-email"><span class="sr-only">收藏更新邮件通知</span><input id="pref-email" v-model="prefEmail" class="toggle" type="checkbox" /></label></div><div class="preference-row"><div><p class="preference-row__label">同步收藏到云端</p><p class="preference-row__hint">在多台设备间保持收藏与标签同步。</p></div><label class="toggle-control" for="pref-sync"><span class="sr-only">同步收藏到云端</span><input id="pref-sync" v-model="prefSync" class="toggle" type="checkbox" /></label></div></div><div class="form-actions"><button class="btn btn-primary" type="submit" :disabled="loading">{{ loading ? '保存中...' : '保存偏好设置' }}</button></div><div class="status" :class="statusClass(preferenceStatus)" role="status" aria-live="polite">{{ preferenceStatus.message }}</div></form></div>

          <div v-if="isLoggedIn" class="panel"><div class="panel-head"><div><p class="eyebrow">账户安全</p><h3>修改登录密码</h3></div><span class="tag">JWT 会话保持</span></div><form class="form-grid" @submit.prevent="submitPassword"><div class="field" :class="{ 'is-error': Boolean(passwordErrors['old']) }"><label for="old-password"><span class="label-text">旧密码 <span class="required" aria-hidden="true">*</span></span></label><input id="old-password" v-model="oldPassword" class="input" type="password" autocomplete="current-password" aria-describedby="old-password-hint old-password-error" :aria-invalid="Boolean(passwordErrors['old'])" /><p id="old-password-hint" class="helper-text">需要验证当前密码后才能修改。</p><p v-if="passwordErrors['old']" id="old-password-error" class="error-text">{{ passwordErrors['old'] }}</p></div><div class="field" :class="{ 'is-error': Boolean(passwordErrors['next']) }"><label for="new-password"><span class="label-text">新密码 <span class="required" aria-hidden="true">*</span></span></label><input id="new-password" v-model="newPassword" class="input" type="password" autocomplete="new-password" aria-describedby="new-password-hint new-password-error" :aria-invalid="Boolean(passwordErrors['next'])" /><p id="new-password-hint" class="helper-text">至少 6 个字符，成功后可用新密码登录。</p><p v-if="passwordErrors['next']" id="new-password-error" class="error-text">{{ passwordErrors['next'] }}</p></div><div class="form-actions"><button class="btn btn-primary" type="submit" :disabled="passwordBusy">{{ passwordBusy ? '修改中...' : '修改密码' }}</button></div><div class="status" :class="statusClass(passwordStatus)" role="alert" aria-live="assertive">{{ passwordStatus.message }}</div></form></div>

          <div v-if="isLoggedIn" class="panel"><div class="panel-head"><div><p class="eyebrow">最近活动</p><h3>账户动态</h3></div><span class="tag">仅自己可见</span></div><div class="activity-empty" role="status" aria-live="polite"><div class="activity-empty__icon" aria-hidden="true"><svg viewBox="0 0 48 48" width="48" height="48" role="img" focusable="false"><path d="M12 14h24v20H12z" fill="none" stroke="currentColor" stroke-width="3" /><path d="M16 30l6-7 5 5 4-4 5 6" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" /></svg></div><p class="activity-empty__title">暂无新的账户动态</p><p class="activity-empty__desc">收藏、打分或批量标记图片后，这里会显示最近的同步记录。</p></div></div>
        </section>
      </div>
    </section>
  </main>
</template>
