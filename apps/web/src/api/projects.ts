import { api } from '@/request'

export interface ProjectOwner {
  id: number
  userId?: number | null
  ownerName: string
  roleType: 'frontend' | 'backend' | 'tester'
}

export interface ProjectOwners {
  frontend: ProjectOwner[] | null
  backend: ProjectOwner[] | null
  tester: ProjectOwner[] | null
}

export interface ProjectEntry {
  id: number
  type: 'production' | 'staging' | 'gitlab' | string
  title: string
  url: string
  sortOrder: number
}

export interface Project {
  id: number
  name: string
  type: string
  typeLabel: string
  description: string
  keywords: string[]
  owners: ProjectOwners
  entries: ProjectEntry[]
  sortOrder: number
  createdAt: string
  updatedAt: string
}

export interface ProjectListParams {
  keyword?: string
  type?: string
  page?: number
  pageSize?: number
}

export interface ProjectListResponse {
  list: Project[]
  page: number
  pageSize: number
  total: number
}

export interface ProjectOwnerPayload {
  ownerName: string
}

export interface ProjectSavePayload {
  name: string
  type: string
  description: string
  keywords: string[]
  entries: ProjectEntryPayload[]
  frontendOwners: ProjectOwnerPayload[]
  backendOwners: ProjectOwnerPayload[]
  testerOwners: ProjectOwnerPayload[]
  sortOrder: number
}

export interface ProjectEntryPayload {
  type: string
  title: string
  url: string
}

export function getProjects(params: ProjectListParams) {
  return api.get<ProjectListResponse>({
    url: '/projects',
    params,
  })
}

export function getProject(id: number) {
  return api.get<Project>({
    url: `/projects/${id}`,
  })
}

export function createProject(data: ProjectSavePayload) {
  return api.post<Project>({
    url: '/projects',
    data,
  })
}

export function updateProject(id: number, data: ProjectSavePayload) {
  return api.put<Project>({
    url: `/projects/${id}`,
    data,
  })
}
