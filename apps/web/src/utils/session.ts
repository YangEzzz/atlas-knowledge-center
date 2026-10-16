export const apiBaseURL = (import.meta.env.VITE_APP_API_BASE_URL || '/api/v1').replace(/\/$/, '')
let csrfToken = ''
let csrfRequest: Promise<string> | null = null

export function clearLegacyAuth() {
  localStorage.removeItem('ty_notebook_token')
  localStorage.removeItem('user')
}

export async function authRequest<T>(path: string, data?: unknown): Promise<T> {
  const response = await fetch(`${apiBaseURL}/auth${path}`, {
    method: data === undefined ? 'GET' : 'POST',
    credentials: 'include',
    headers: data === undefined ? {} : { 'Content-Type': 'application/json', 'X-CSRF-Token': await getCSRFToken() },
    ...(data === undefined ? {} : { body: JSON.stringify(data) }),
    signal: AbortSignal.timeout(15000),
  })
  const result = await response.json()
  if (!response.ok || result.code !== 200)
    throw new Error(result.message || '请求失败，请稍后重试')
  return result.data as T
}

export function getCSRFToken(): Promise<string> {
  if (csrfToken)
    return Promise.resolve(csrfToken)
  if (!csrfRequest) {
    csrfRequest = authRequest<{ csrf_token: string }>('/csrf')
      .then((data) => {
        csrfToken = data.csrf_token
        return csrfToken
      })
      .finally(() => { csrfRequest = null })
  }
  return csrfRequest
}

export function safeReturnPath(value: unknown): string {
  if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//')
    || /[\\\r\n]/.test(value) || value.startsWith('/auth/')) {
    return '/'
  }
  return value
}

