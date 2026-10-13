import { api } from '@/request'

export interface PingResponse {
  message: string
  time: string
}

export function ping() {
  return api.get<PingResponse>({
    url: '/ping',
  })
}
