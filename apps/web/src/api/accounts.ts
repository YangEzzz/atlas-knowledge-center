import { api } from "@/request";
export interface PersonalAccount {
  storage?: "local" | "server";
  id: number;
  projectId: number;
  toolId?: number;
  entryId?: number;
  projectName?: string;
  toolName?: string;
  entryName?: string;
  name: string;
  username: string;
  environment: "production" | "staging" | "other";
  resourceUrl: string;
  hasPassword: boolean;
  revision: number;
  updatedAt: string;
  deletedAt?: string;
}
export interface AccountInput {
  projectId: number;
  toolId?: number;
  entryId?: number;
  name: string;
  username: string;
  environment: string;
  resourceUrl: string;
  note: string;
  passwordAction: "keep" | "replace" | "clear";
  password: string;
  revision: number;
}
export const listAccounts = (params: Record<string, unknown>) =>
  api.get<{ list: PersonalAccount[]; total: number }>({
    url: "/my/accounts",
    params,
  });
export const getAccount = (id: number) =>
  api.get<{ account: PersonalAccount; note: string }>({
    url: `/my/accounts/${id}`,
  });
export const saveAccount = (id: number | null, data: AccountInput) =>
  id
    ? api.put<PersonalAccount>({ url: `/my/accounts/${id}`, data })
    : api.post<PersonalAccount>({ url: "/my/accounts", data });
export const readPassword = (id: number) =>
  api.post<{ password: string }>({ url: `/my/accounts/${id}/password` });
export const deleteAccount = (item: PersonalAccount) =>
  api.delete({
    url: `/my/accounts/${item.id}`,
    data: { revision: item.revision },
  });
export const restoreAccount = (id: number) =>
  api.post({ url: `/my/accounts/${id}/restore` });
