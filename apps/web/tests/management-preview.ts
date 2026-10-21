// Disposable management fixtures for the existing development-only harness.
import type {
  AuditLog,
  DocumentTemplate,
  TrashItem,
} from "../src/api/management";
export function createManagementPreview(scenario: string | null, now: string) {
  let rows: DocumentTemplate[] =
    scenario === "empty"
      ? []
      : Array.from({ length: 23 }, (_, i) => ({
          id: 600 + i,
          name: i === 0 ? "团队环境说明模板" : `项目资料模板 ${i}`,
          type: i === 0 ? "environment" : "intro",
          content: "<h2>团队模板正文</h2><p>请填写访问方式和维护人。</p>",
          isEnabled: i === 0,
          updatedBy: 1,
          createdAt: now,
          updatedAt: now,
        }));
  let trash: TrashItem[] =
    scenario === "empty"
      ? []
      : ["project", "entry", "document", "tool"].map((kind, i) => ({
          id: 701 + i,
          name: ["误删项目", "测试发布入口", "部署说明", "监控工具"][i]!,
          objectType: kind,
          projectId: i === 3 ? null : 101,
          projectName: i === 3 ? "" : "Nexus 多语言平台",
          deletedBy: 1,
          deletedByName: "杨林",
          deletedAt: now,
          parentDeleted: i === 1,
        }));
  const audits: AuditLog[] =
    scenario === "empty"
      ? []
      : [
          {
            id: 800,
            actorId: 1,
            actorName: "杨林",
            objectType: "document",
            objectId: 201,
            objectName: "生产环境部署指南",
            projectId: 101,
            action: "update",
            summary: "发布文档",
            fields: "正文、状态",
            createdAt: now,
          },
        ];
  let failed = false;
  return async (config: any) => {
    const url = config.url as string;
    const method = config.method || "get";
    const params = config.params || {};
    const input =
      typeof config.data === "string"
        ? JSON.parse(config.data)
        : config.data || {};
    if (url.startsWith("/document-templates/active/")) {
      if (scenario === "template-error" && !failed) {
        failed = true;
        throw new Error("模板读取失败，请重试");
      }
      if (scenario === "template-slow")
        await new Promise((resolve) =>
          setTimeout(resolve, url.endsWith("environment") ? 1500 : 100),
        );
      const active = rows.find(
        (r) => r.type === url.split("/").pop() && r.isEnabled,
      );
      return active || null;
    }
    if (scenario === "management-failure" && method !== "get" && !failed) {
      failed = true;
      throw new Error("保存失败，请稍后重试");
    }
    if (scenario === "management-slow")
      await new Promise((resolve) => setTimeout(resolve, 1200));
    if (url.startsWith("/manage/templates")) {
      const id = Number(url.split("/").pop());
      const row = rows.find((r) => r.id === id);
      if (method === "delete") {
        rows = rows.filter((r) => r.id !== id);
        return {};
      }
      if (method !== "get") {
        if (scenario === "management-conflict")
          throw new Error("模板已被其他人修改，请关闭表单并刷新列表后重试");
        if (input.isEnabled)
          rows
            .filter((r) => r.type === input.type)
            .forEach((r) => {
              r.isEnabled = false;
            });
        const result = {
          ...input,
          id: row?.id || 1000 + rows.length,
          createdAt: row?.createdAt || now,
          updatedAt: new Date().toISOString(),
        };
        if (row) Object.assign(row, result);
        else rows.unshift(result);
        return result;
      }
      let filtered = rows.filter(
        (r) =>
          (!params.keyword || r.name.includes(params.keyword)) &&
          (!params.type || r.type === params.type) &&
          (!params.enabled || String(r.isEnabled) === params.enabled),
      );
      return paginate(filtered, params);
    }
    if (url.startsWith("/manage/trash")) {
      if (method !== "get") {
        const id = Number(url.split("/").at(-2));
        trash = trash.filter((r) => r.id !== id);
        return {};
      }
      return paginate(
        trash.filter(
          (r) =>
            (!params.keyword || r.name.includes(params.keyword)) &&
            (!params.type || r.objectType === params.type),
        ),
        params,
      );
    }
    if (/^\/manage\/audit\/\d+$/.test(url))
      return audits.find((r) => r.id === Number(url.split("/").pop()));
    return paginate(
      audits.filter(
        (r) =>
          (!params.actor ||
            r.actorName.includes(params.actor) ||
            String(r.actorId) === params.actor) &&
          (!params.type || r.objectType === params.type) &&
          (!params.action || r.action === params.action),
      ),
      params,
    );
  };
}
function paginate<T>(rows: T[], params: any) {
  const page = Number(params.page) || 1;
  const size = Number(params.pageSize) || 20;
  return {
    list: rows.slice((page - 1) * size, page * size),
    total: rows.length,
    page,
    pageSize: size,
  };
}
