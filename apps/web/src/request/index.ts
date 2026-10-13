import type { AxiosError, AxiosRequestConfig, AxiosRequestHeaders, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import axios from 'axios'
import router from '@/router'
import { getToken, removeToken } from '@/utils/auth'

export interface ResponseData<T = unknown> {
  code: number
  message: string
  data: T
}

export type RequestError = AxiosError<{
  message?: string
  errorMessage?: string
}>

const request = axios.create({
  baseURL: import.meta.env.VITE_APP_API_BASE_URL || '/api/v1',
  timeout: 30000,
})

function requestHandler(config: InternalAxiosRequestConfig): InternalAxiosRequestConfig {
  const token = getToken()
  if (token) {
    ;(config.headers as AxiosRequestHeaders).Authorization = `Bearer ${token}`
  }
  return config
}

function errorHandler(error: RequestError): Promise<never> {
  const status = error.response?.status
  if (status === 401) {
    removeToken()
    router.push('/auth/sign-in')
  }
  return Promise.reject(error)
}

function responseHandler(response: AxiosResponse<ResponseData>) {
  const body = response.data
  if (!body) {
    throw new Error('请求没有返回值')
  }
  if (body.code !== 200) {
    throw new Error(body.message || '请求失败')
  }
  return body
}

request.interceptors.request.use(requestHandler, errorHandler)
request.interceptors.response.use(responseHandler as never, errorHandler)

interface UploadRequestConfig extends AxiosRequestConfig {
  data?: FormData | Record<string, unknown>
}

export const api = {
  get: <T>(option: AxiosRequestConfig): Promise<ResponseData<T>> => {
    return request({ method: 'GET', ...option })
  },
  post: <T>(option: AxiosRequestConfig): Promise<ResponseData<T>> => {
    return request({ method: 'POST', ...option })
  },
  put: <T>(option: AxiosRequestConfig): Promise<ResponseData<T>> => {
    return request({ method: 'PUT', ...option })
  },
  delete: <T>(option: AxiosRequestConfig): Promise<ResponseData<T>> => {
    return request({ method: 'DELETE', ...option })
  },
  upload: <T>(option: UploadRequestConfig): Promise<ResponseData<T>> => {
    return request({
      method: 'POST',
      headers: { 'Content-Type': 'multipart/form-data' },
      ...option,
    })
  },
}

export default request
