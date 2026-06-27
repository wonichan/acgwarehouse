const API_BASE = '/api/v1'
const TOKEN_KEY = 'acgwarehouse_token'

type ApiCallOptions = Omit<RequestInit, 'headers'> & {
  readonly headers?: HeadersInit
  readonly skipAuth?: boolean
}

export interface ApiResponse<T> {
  readonly code: number
  readonly msg: string
  readonly data: T
}

export class ApiError extends Error {
  readonly status: number
  readonly code?: number

  constructor(message: string, status: number, code?: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    if (code !== undefined) this.code = code
  }
}

function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

export function isAuthenticated(): boolean {
  return getToken() !== null
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function readResponseMessage(payload: unknown, fallback: string): string {
  if (!isRecord(payload)) return fallback

  const msg = payload['msg']
  if (typeof msg === 'string' && msg.length > 0) return msg

  const message = payload['message']
  if (typeof message === 'string' && message.length > 0) return message

  return fallback
}

function readResponseCode(payload: unknown): number | undefined {
  if (!isRecord(payload)) return undefined

  const code = payload['code']
  return typeof code === 'number' ? code : undefined
}

async function readJsonPayload(response: Response): Promise<unknown> {
  try {
    return await response.json()
  } catch (error) {
    if (error instanceof SyntaxError) return null
    throw error
  }
}

export async function apiCall<T>(path: string, options?: ApiCallOptions): Promise<T> {
  const skipAuth = options?.skipAuth === true
  const headers = new Headers(options?.headers)
  headers.set('Content-Type', 'application/json')

  const token = getToken()
  if (token && !skipAuth) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method: options?.method,
    body: options?.body,
    cache: options?.cache,
    credentials: options?.credentials,
    integrity: options?.integrity,
    keepalive: options?.keepalive,
    mode: options?.mode,
    priority: options?.priority,
    redirect: options?.redirect,
    referrer: options?.referrer,
    referrerPolicy: options?.referrerPolicy,
    signal: options?.signal,
    window: options?.window,
    headers,
  })

  if (!response.ok) {
    const payload = await readJsonPayload(response)
    throw new ApiError(
      readResponseMessage(payload, `API Error: ${response.status}`),
      response.status,
      readResponseCode(payload)
    )
  }

  return response.json()
}

export async function unwrapResponse<T>(promise: Promise<ApiResponse<T>>): Promise<T> {
  const response = await promise
  if (response.code !== 0) {
    throw new ApiError(response.msg || '请求失败', 200, response.code)
  }
  return response.data
}
