import { api } from '@/request'

export interface DocumentItem {
  id: number
  projectId: number
  type: string
  typeLabel: string
  title: string
  summary: string
  content: string
  status: 'draft' | 'published' | string
  statusLabel: string
  currentVersionNo: number
  createdAt: string
  updatedAt: string
  deletedAt?: string
}

export interface DocumentVersionItem {
  id: number
  documentId: number
  versionNo: number
  title: string
  type: string
  typeLabel: string
  summary: string
  content: string
  changeSummary: string
  createdAt: string
}

export interface DocumentListParams {
  keyword?: string
  type?: string
  status?: string
  projectId?: number
  page?: number
  pageSize?: number
}

export interface DocumentListResponse {
  list: DocumentItem[]
  page: number
  pageSize: number
  total: number
}

export interface DocumentVersionListResponse {
  list: DocumentVersionItem[]
}

export interface DocumentSavePayload {
  projectId: number
  type: string
  title: string
  summary: string
  content: string
  status: 'draft' | 'published'
}

export interface DocumentVersionRestorePayload {
  changeSummary?: string
}

export function getDocuments(params: DocumentListParams) {
  return api.get<DocumentListResponse>({
    url: '/documents',
    params,
  })
}

export function getDeletedDocuments(params: DocumentListParams) {
  return api.get<DocumentListResponse>({
    url: '/documents/trash',
    params,
  })
}

export function getDocument(id: number) {
  return api.get<DocumentItem>({
    url: `/documents/${id}`,
  })
}

export function getProjectDocument(projectId: number, type: string) {
  return api.get<DocumentItem>({
    url: `/documents/project/${projectId}/type/${type}`,
  })
}

export function saveDocument(data: DocumentSavePayload) {
  return api.post<DocumentItem>({
    url: '/documents',
    data,
  })
}

export function deleteDocument(id: number) {
  return api.delete<{ deleted: boolean }>({
    url: `/documents/${id}`,
  })
}

export function restoreDocument(id: number) {
  return api.post<DocumentItem>({
    url: `/documents/${id}/restore`,
  })
}

export function getDocumentVersions(documentId: number) {
  return api.get<DocumentVersionListResponse>({
    url: `/documents/${documentId}/versions`,
  })
}

export function getDocumentVersion(documentId: number, versionId: number) {
  return api.get<DocumentVersionItem>({
    url: `/documents/${documentId}/versions/${versionId}`,
  })
}

export function restoreDocumentVersion(
  documentId: number,
  versionId: number,
  data: DocumentVersionRestorePayload,
) {
  return api.post<DocumentItem>({
    url: `/documents/${documentId}/versions/${versionId}/restore`,
    data,
  })
}
