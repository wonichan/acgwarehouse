export type StatusTone = 'idle' | 'loading' | 'success' | 'error'

export interface FormStatus {
  message: string
  tone: StatusTone
}

export type FieldErrors = Record<string, string>

export function setStatus(status: FormStatus, message: string, tone: StatusTone): void {
  status.message = message
  status.tone = tone
}

export function statusClass(status: FormStatus): string {
  return status.tone === 'idle' ? '' : `is-visible status--${status.tone}`
}

export function validateCredential(username: string, password: string, errors: FieldErrors): boolean {
  errors['username'] = ''
  errors['password'] = ''
  const trimmed = username.trim()
  if (trimmed.length < 3 || trimmed.length > 32) errors['username'] = '用户名需为 3-32 个字符。'
  if (password.length < 6) errors['password'] = '密码至少需要 6 个字符。'
  return !errors['username'] && !errors['password']
}

function characterCount(value: string): number {
  return Array.from(value).length
}

export function validateProfileFields(nickname: string, tags: string, bio: string, errors: FieldErrors): boolean {
  errors['nickname'] = ''
  errors['tags'] = ''
  errors['bio'] = ''
  const trimmedNickname = nickname.trim()
  const trimmedTags = tags.trim()
  const trimmedBio = bio.trim()
  if (characterCount(trimmedNickname) < 1 || characterCount(trimmedNickname) > 20) errors['nickname'] = '显示昵称需为 1-20 个字符。'
  if (characterCount(trimmedTags) > 120) errors['tags'] = '常用标签最多 120 个字符。'
  if (characterCount(trimmedBio) > 200) errors['bio'] = '个人简介最多 200 个字符。'
  return !errors['nickname'] && !errors['tags'] && !errors['bio']
}

export function validatePasswordFields(oldPassword: string, newPassword: string, errors: FieldErrors): boolean {
  errors['old'] = ''
  errors['next'] = ''
  if (oldPassword.length < 6) errors['old'] = '旧密码至少需要 6 个字符。'
  if (newPassword.length < 6) errors['next'] = '新密码至少需要 6 个字符。'
  return !errors['old'] && !errors['next']
}
