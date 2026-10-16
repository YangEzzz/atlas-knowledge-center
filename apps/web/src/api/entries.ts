import { api } from "@/request";
export interface PersonalEntry {
  id: number; source: "private" | "project" | "tool" | "document";
  targetId: number; kind: "project" | "tool" | "document"; name: string; url: string;
  groupName: string; pinned: boolean; note: string; revision: number;
  available: boolean; accountCount: number; updatedAt: string; deletedAt?: string;
}
export type EntryInput = Pick<PersonalEntry, "source" | "targetId" | "kind" | "name" | "url" | "groupName" | "pinned" | "note" | "revision">;
export const listEntries = (params: Record<string, unknown>) => api.get<{list:PersonalEntry[];total:number}>({url:"/my/entries",params});
export const getEntry = (id:number) => api.get<PersonalEntry>({url:`/my/entries/${id}`});
export const saveEntry = (id:number|null,data:EntryInput) => id ? api.put<PersonalEntry>({url:`/my/entries/${id}`,data}) : api.post<PersonalEntry>({url:"/my/entries",data});
export const importEntries = (items:Partial<EntryInput>[]) => api.post({url:"/my/entries/import",data:{items}});
export const deleteEntry = (entry:PersonalEntry) => api.delete({url:`/my/entries/${entry.id}`,data:{revision:entry.revision}});
export const restoreEntry = (id:number) => api.post({url:`/my/entries/${id}/restore`});
