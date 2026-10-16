import type { AccountInput, PersonalAccount } from "../api/accounts";

// Plain local storage is intentional: no uploaded secrets and no claim of encryption.
// Positive IndexedDB keys become negative UI IDs, keeping server IDs unambiguous.
interface StoredAccount extends Omit<PersonalAccount, "id"> {
  id?: number;
  owner: string;
  note: string;
  password: string;
}
export type AccountNames = Pick<PersonalAccount, "projectName" | "toolName" | "entryName">;
const RETENTION = 7 * 24 * 60 * 60 * 1000;
const unavailable = () => new Error("无法使用浏览器本地存储，请检查网站存储权限或剩余空间；账号未上传服务器。");

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    if (!globalThis.indexedDB) return reject(unavailable());
    const request = indexedDB.open("atlas-local-accounts", 1);
    request.onupgradeneeded = () => {
      const store = request.result.createObjectStore("accounts", { keyPath: "id", autoIncrement: true });
      store.createIndex("owner", "owner");
    };
    request.onerror = () => reject(unavailable());
    let blocked = false;
    request.onblocked = () => { blocked = true; reject(new Error("请关闭其他 Atlas 页面后重试本地保存。")); };
    request.onsuccess = () => {
      const db = request.result;
      if (blocked) { db.close(); return; }
      db.onversionchange = () => db.close();
      resolve(db);
    };
  });
}

async function transaction<T>(owner: string, mode: IDBTransactionMode,
  run: (store: IDBObjectStore, finish: (value: T) => void, fail: (error: Error) => void) => void): Promise<T> {
  if (!owner) throw new Error("请先登录后使用本地账号。");
  let db: IDBDatabase;
  try { db = await openDB(); } catch { throw unavailable(); }
  return new Promise<T>((resolve, reject) => {
    const tx = db.transaction("accounts", mode);
    let value: T, failure: Error | undefined;
    tx.oncomplete = () => { db.close(); resolve(value); };
    tx.onabort = tx.onerror = () => { db.close(); reject(failure || unavailable()); };
    try {
      run(tx.objectStore("accounts"), result => { value = result; }, error => {
        failure = error;
        tx.abort();
      });
    } catch { tx.abort(); }
  });
}

function expired(row: StoredAccount) {
  return !!row.deletedAt && Date.now() - Date.parse(row.deletedAt) >= RETENTION;
}
function publicRow(row: StoredAccount): PersonalAccount {
  const { owner: _owner, password: _password, note: _note, id, ...account } = row;
  return { ...account, id: -id!, storage: "local" };
}
export function listLocalAccounts(owner: string) {
  return transaction<PersonalAccount[]>(owner, "readwrite", (store, done) => {
    const request = store.index("owner").getAll(owner);
    request.onsuccess = () => {
      const rows = request.result as StoredAccount[];
      for (const row of rows) if (expired(row)) store.delete(row.id!);
      done(rows.filter(row => !expired(row)).map(publicRow));
    };
  });
}

function withRow<T>(owner: string, id: number, mode: IDBTransactionMode,
  action: (row: StoredAccount, store: IDBObjectStore, done: (value: T) => void, fail: (error: Error) => void) => void) {
  if (!Number.isSafeInteger(id) || id >= 0) return Promise.reject(new Error("无效的本地账号。"));
  return transaction<T>(owner, mode, (store, done, fail) => {
    const request = store.get(-id);
    request.onsuccess = () => {
      const row = request.result as StoredAccount | undefined;
      if (!row || row.owner !== owner || expired(row)) return fail(new Error("本地账号不存在或已过期。"));
      action(row, store, done, fail);
    };
  });
}
export function getLocalAccount(owner: string, id: number) {
  return withRow<{ account: PersonalAccount; note: string }>(owner, id, "readonly", (row, _store, done, fail) => {
    if (row.deletedAt) return fail(new Error("请先恢复本地账号。"));
    done({ account: publicRow(row), note: row.note });
  });
}
export function readLocalPassword(owner: string, id: number) {
  return withRow<{ password: string }>(owner, id, "readonly", (row, _store, done, fail) => {
    if (row.deletedAt) return fail(new Error("请先恢复本地账号。"));
    done({ password: row.password });
  });
}
export function saveLocalAccount(owner: string, id: number | null, input: AccountInput, names: AccountNames) {
  const targetCount = [input.projectId, input.toolId, input.entryId].filter(Boolean).length;
  if (targetCount !== 1 || !input.username.trim() ||
    !["production", "staging", "other"].includes(input.environment) ||
    !["keep", "replace", "clear"].includes(input.passwordAction) ||
    (input.passwordAction === "replace" && !input.password) ||
    input.name.length > 200 || input.username.length > 300 || input.note.length > 10000 || input.password.length > 4096) {
    return Promise.reject(new Error("请检查账号字段及关联入口。"));
  }
  if (input.resourceUrl) {
    try { const url = new URL(input.resourceUrl); if (!/^https?:$/.test(url.protocol) || url.username || url.password) throw 0; }
    catch { return Promise.reject(new Error("入口网址必须是有效的 HTTP 或 HTTPS 地址。")); }
  }
  const write = (previous: StoredAccount | undefined, store: IDBObjectStore, done: (value: PersonalAccount) => void) => {
    const password = input.passwordAction === "replace" ? input.password : input.passwordAction === "clear" ? "" : previous?.password || "";
    const row: StoredAccount = {
      ...names, ...(previous ? { id: previous.id } : {}), owner,
      projectId: input.projectId, toolId: input.toolId || 0, entryId: input.entryId || 0,
      name: input.name.trim(), username: input.username.trim(),
      environment: input.environment as PersonalAccount["environment"], resourceUrl: input.resourceUrl,
      note: input.note, password, hasPassword: !!password,
      updatedAt: new Date().toISOString(), revision: (previous?.revision || 0) + 1,
    };
    const request = store.put(row);
    request.onsuccess = () => done(publicRow({ ...row, id: Number(request.result) }));
  };
  if (id === null) return transaction<PersonalAccount>(owner, "readwrite", (store, done) => write(undefined, store, done));
  return withRow<PersonalAccount>(owner, id, "readwrite", (row, store, done, fail) => {
    if (row.deletedAt || row.revision !== input.revision) return fail(new Error("账号已更新，请关闭编辑并刷新后重试。"));
    if (row.projectId !== input.projectId || (row.toolId || 0) !== (input.toolId || 0) || (row.entryId || 0) !== (input.entryId || 0))
      return fail(new Error("编辑时不能更改账号归属。"));
    write(row, store, done);
  });
}
export function deleteLocalAccount(owner: string, item: PersonalAccount) {
  return withRow<void>(owner, item.id, "readwrite", (row, store, done, fail) => {
    if (row.deletedAt || row.revision !== item.revision) return fail(new Error("账号已更新，请刷新后重试。"));
    store.put({ ...row, deletedAt: new Date().toISOString(), revision: row.revision + 1 });
    done();
  });
}
export function restoreLocalAccount(owner: string, id: number) {
  return withRow<void>(owner, id, "readwrite", (row, store, done, fail) => {
    if (!row.deletedAt) return fail(new Error("账号已恢复，请刷新。"));
    const { deletedAt: _deletedAt, ...restored } = row;
    store.put({ ...restored, revision: row.revision + 1, updatedAt: new Date().toISOString() });
    done();
  });
}
