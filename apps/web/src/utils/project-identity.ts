import type { Project } from "@/api/projects";

export function projectIdentity(id: number | string, name: string) {
  const words = name.trim().match(/[A-Za-z0-9]+/g);
  const label = words?.length
    ? (words.length > 1
        ? words
            .slice(0, 2)
            .map((word) => word[0])
            .join("")
        : words[0].slice(0, 2)
      ).toUpperCase()
    : Array.from(name.trim()).slice(0, 2).join("") || "项";
  const hash = Array.from(String(id)).reduce(
    (value, char) => (value * 31 + char.charCodeAt(0)) >>> 0,
    0,
  );
  return { label, tone: ["blue", "green", "clay"][hash % 3] };
}

export function projectOwnerNames(project: Project) {
  return (
    [
      ...new Set(
        Object.values(project.owners || {})
          .flatMap((group) => group || [])
          .map((owner) => owner.ownerName),
      ),
    ].join("、") || "负责人待补充"
  );
}

export function entryLabel(type: string) {
  return (
    (
      {
        production: "线上",
        staging: "测试",
        gitlab: "GitLab",
        prototype: "原型",
        api_doc: "API 文档",
        admin: "后台",
        monitor: "监控",
        logs: "日志",
        deploy: "发布",
        access: "权限",
      } as Record<string, string>
    )[type] || "资源入口"
  );
}

export function formatDate(value: string) {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "暂无更新时间";
  return date.toLocaleString("zh-CN", {
    timeZone: "Asia/Shanghai",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function isStale(value: string) {
  const time = new Date(value).getTime();
  return Number.isFinite(time) && Date.now() - time > 90 * 86400000;
}
