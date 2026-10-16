import type { CurrentUser } from '@/store/auth'
import { authRequest } from '@/utils/session'

export interface LoginStatus { status: string, message: string, return_path: string, mode: string }
export const getLoginStatus = (transaction: string) => authRequest<LoginStatus>(`/sso/status?transaction=${encodeURIComponent(transaction)}`)
export const startSSOLogin = (data: { mode: string, return_path: string }) => authRequest<{ authorize_url: string, transaction_id: string }>('/sso/start', data)
export const cancelSSOLogin = (transaction: string) => authRequest<unknown>('/sso/cancel', { transaction })
export const logoutSession = () => authRequest<unknown>('/logout', {})
export const getCurrentUser = () => authRequest<CurrentUser>('/me')

