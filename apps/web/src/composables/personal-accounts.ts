import * as server from "@/api/accounts";
import type { AccountInput, PersonalAccount } from "@/api/accounts";
import * as local from "@/utils/local-accounts";
import { useAuthStore } from "@/store/auth";

export type { AccountInput, PersonalAccount } from "@/api/accounts";
export type AccountStorage = "local" | "server";
function owner() {
  const id = useAuthStore().user?.id;
  if (id === undefined || id === null) throw new Error("请先登录。");
  return String(id);
}
function stillOwner(id: string) {
  if (owner() !== id) throw new Error("登录用户已变化，请重新操作。");
}
const response = <T>(data: T) => ({ code: 200, message: "", data });

export async function listAccounts(params: Record<string, unknown>) {
  const user = owner();
  const storage = params.storage || "all";
  const warnings: string[] = [];
  const tasks = await Promise.allSettled([
    storage === "server" ? Promise.resolve([]) : local.listLocalAccounts(user),
    (async () => {
      if (storage === "local") return [];
      const rows: PersonalAccount[] = [];
      // Search privately in memory, so a local username entered in search is never sent to the server.
      for (let page = 1; ; page++) {
        stillOwner(user);
        const result = await server.listAccounts({ projectId: params.projectId, toolId: params.toolId,
          entryId: params.entryId, trash: params.trash, environment: params.environment, page, pageSize: 50 });
        rows.push(...result.data.list.map(row => ({ ...row, storage: "server" as const })));
        if (rows.length >= result.data.total || !result.data.list.length) return rows;
      }
    })(),
  ]);
  stillOwner(user);
  const sources = ["本地", "服务器"];
  const all: PersonalAccount[] = [];
  tasks.forEach((task, i) => {
    if (task.status === "fulfilled") all.push(...task.value);
    else warnings.push(`${sources[i]}账号加载失败，请重试。${i === 0 ? "请检查网站存储权限和剩余空间。" : "可切换到仅此浏览器查看本地账号。"}`);
  });
  if (tasks.every(task => task.status === "rejected") || (storage === "local" && tasks[0]?.status === "rejected") ||
    (storage === "server" && tasks[1]?.status === "rejected")) throw new Error(warnings.join(" "));
  const query = String(params.keyword || "").trim().toLocaleLowerCase();
  const filtered = all.filter(row => !!row.deletedAt === !!params.trash &&
    (!params.projectId || row.projectId === params.projectId) && (!params.toolId || row.toolId === params.toolId) &&
    (!params.entryId || row.entryId === params.entryId) && (!params.environment || row.environment === params.environment) &&
    (!query || [row.name, row.username, row.projectName, row.toolName, row.entryName].some(value => value?.toLocaleLowerCase().includes(query))))
    .sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt) || b.id - a.id);
  const size = Number(params.pageSize) || 12, page = Number(params.page) || 1;
  return response({ list: filtered.slice((page - 1) * size, page * size), total: filtered.length, warning: warnings.join(" ") });
}
export async function getAccount(id: number) {
  const user = owner();
  const result = id < 0 ? response(await local.getLocalAccount(user, id)) : await server.getAccount(id);
  stillOwner(user);
  return result;
}
export async function readPassword(id: number) {
  const user = owner();
  const result = id < 0 ? response(await local.readLocalPassword(user, id)) : await server.readPassword(id);
  stillOwner(user);
  return result;
}
export async function saveAccount(id: number | null, input: AccountInput, storage: AccountStorage, names: local.AccountNames) {
  const user = owner();
  if (id !== null && (id < 0) !== (storage === "local")) throw new Error("编辑时不能更改保存位置。");
  const result = storage === "local" ? response(await local.saveLocalAccount(user, id, input, names)) : await server.saveAccount(id, input);
  stillOwner(user);
  return result;
}
export async function deleteAccount(item: PersonalAccount) {
  const user = owner();
  if (item.id < 0) await local.deleteLocalAccount(user, item); else await server.deleteAccount(item);
  stillOwner(user);
}
export async function restoreAccount(id: number) {
  const user = owner();
  if (id < 0) await local.restoreLocalAccount(user, id); else await server.restoreAccount(id);
  stillOwner(user);
}
