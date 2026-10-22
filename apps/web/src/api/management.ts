import { api } from "@/request";

export const documentTypes = [
  { value: "general", label: "通用文档" },
  { value: "intro", label: "项目介绍" },
  { value: "environment", label: "环境说明" },
  { value: "deploy", label: "部署说明" },
  { value: "api", label: "API 说明" },
  { value: "access", label: "账号/权限申请说明" },
  { value: "faq", label: "常见问题 FAQ" },
  { value: "standard", label: "开发规范" },
];
export const objectTypes = [
  { value: "project", label: "项目" },
  { value: "entry", label: "资源入口" },
  { value: "document", label: "文档" },
  { value: "tool", label: "工具导航" },
  { value: "template", label: "文档模板" },
];
export const auditActions = [
  { value: "create", label: "创建" },
  { value: "update", label: "修改" },
  { value: "delete", label: "删除" },
  { value: "restore", label: "恢复" },
  { value: "restore_version", label: "恢复版本" },
];
export interface DocumentTemplate {
  id: number;
  name: string;
  type: string;
  content: string;
  isEnabled: boolean;
  updatedBy: number | null;
  updatedAt: string;
  createdAt: string;
}
export interface TemplatePayload {
  name: string;
  type: string;
  content: string;
  isEnabled: boolean;
  updatedAt?: string;
}
export interface TrashItem {
  id: number;
  name: string;
  objectType: string;
  projectId: number | null;
  projectName: string;
  deletedBy: number | null;
  deletedByName: string;
  deletedAt: string;
  parentDeleted: boolean;
}
export interface AuditLog {
  id: number;
  actorId: number | null;
  actorName: string;
  objectType: string;
  objectId: number;
  objectName: string;
  projectId: number | null;
  action: string;
  summary: string;
  fields: string;
  createdAt: string;
}
export interface PageResult<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}
export interface ManagementQuery {
  page?: number;
  pageSize?: number;
  keyword?: string;
  type?: string;
  enabled?: string;
  actor?: string;
  action?: string;
  from?: string;
  to?: string;
}
export const getTemplates = (params: ManagementQuery, signal?: AbortSignal) =>
  api.get<PageResult<DocumentTemplate>>({
    url: "/manage/templates",
    params,
    signal,
  });
export const saveTemplate = (id: number | undefined, data: TemplatePayload) =>
  id
    ? api.put<DocumentTemplate>({ url: `/manage/templates/${id}`, data })
    : api.post<DocumentTemplate>({ url: "/manage/templates", data });
export const deleteTemplate = (row: DocumentTemplate) =>
  api.delete({
    url: `/manage/templates/${row.id}`,
    data: { updatedAt: row.updatedAt },
  });
export const getActiveTemplate = (type: string, signal?: AbortSignal) =>
  api.get<DocumentTemplate | null>({
    url: `/document-templates/active/${encodeURIComponent(type)}`,
    signal,
  });
export const getTrash = (params: ManagementQuery, signal?: AbortSignal) =>
  api.get<PageResult<TrashItem>>({ url: "/manage/trash", params, signal });
export const restoreTrash = (row: TrashItem) =>
  api.post({ url: `/manage/trash/${row.objectType}/${row.id}/restore` });
export const getAudit = (params: ManagementQuery, signal?: AbortSignal) =>
  api.get<PageResult<AuditLog>>({ url: "/manage/audit", params, signal });
export const getAuditDetail = (id: number) =>
  api.get<AuditLog>({ url: `/manage/audit/${id}` });
export const labelFor = (
  options: { value: string; label: string }[],
  value: string,
) => options.find((o) => o.value === value)?.label || value;
export const managementTime = (value: string) =>
  new Date(value).toLocaleString("zh-CN", {
    timeZone: "Asia/Shanghai",
    hour12: false,
  });
