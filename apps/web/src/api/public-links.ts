import { api } from '@/request'

export interface PublicLink {
  id: number
  name: string
  type: string
  typeLabel: string
  url: string
  description: string
  isEnabled: boolean
  sortOrder: number
  createdAt: string
  updatedAt: string
}

export interface PublicLinkListParams {
  keyword?: string
  type?: string
  enabled?: boolean
  page?: number
  pageSize?: number
}

export interface PublicLinkListResponse {
  list: PublicLink[]
  page: number
  pageSize: number
  total: number
}

export interface PublicLinkSavePayload {
  name: string
  type: string
  url: string
  description: string
  isEnabled: boolean
  sortOrder: number
}

export function getPublicLinks(params: PublicLinkListParams) {
  return api.get<PublicLinkListResponse>({
    url: '/public-links',
    params,
  })
}

export function getPublicLink(id: number) {
  return api.get<PublicLink>({ url: `/public-links/${id}` })
}

export function createPublicLink(data: PublicLinkSavePayload) {
  return api.post<PublicLink>({
    url: '/public-links',
    data,
  })
}

export function updatePublicLink(id: number, data: PublicLinkSavePayload) {
  return api.put<PublicLink>({
    url: `/public-links/${id}`,
    data,
  })
}

export function deletePublicLink(id: number) {
  return api.delete<{ id: number }>({
    url: `/public-links/${id}`,
  })
}
