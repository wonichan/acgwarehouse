import { ref, reactive } from 'vue'
import type { FormStatus } from '@/types'

export function useForm<T extends Record<string, string>>(initialValues: T) {
  const values = reactive<T>({ ...initialValues } as T)
  const errors = reactive<Record<string, string>>({})
  const status = ref<FormStatus>('idle')
  const statusMessage = ref('')

  const validateEmail = (email: string): boolean => {
    const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    return emailPattern.test(email.trim())
  }

  const validate = (): boolean => {
    // Clear previous errors
    Object.keys(errors).forEach(key => delete errors[key])
    
    let isValid = true

    // Check required fields
    Object.entries(values).forEach(([key, value]) => {
      if (!value.trim()) {
        errors[key] = '此项为必填内容，请检查后重新输入'
        isValid = false
      }
    })

    // Check email format
    if (values.email && !validateEmail(values.email)) {
      errors.email = '邮箱格式不正确'
      isValid = false
    }

    return isValid
  }

  const setStatus = (newStatus: FormStatus, message = '') => {
    status.value = newStatus
    statusMessage.value = message
  }

  const submit = async (onSuccess: () => Promise<void> | void, successMessage = '已保存') => {
    if (!validate()) {
      setStatus('error', '表单存在错误，请修正后重试')
      return false
    }

    setStatus('loading', '处理中…')

    try {
      await onSuccess()
      setStatus('success', successMessage)
      
      // Reset status after delay
      setTimeout(() => {
        setStatus('idle', '')
      }, 1800)
      
      return true
    } catch {
      setStatus('error', '操作失败，请重试')
      return false
    }
  }

  const reset = () => {
    Object.assign(values, initialValues)
    Object.keys(errors).forEach(key => delete errors[key])
    setStatus('idle', '')
  }

  return {
    values,
    errors,
    status,
    statusMessage,
    validate,
    setStatus,
    submit,
    reset
  }
}