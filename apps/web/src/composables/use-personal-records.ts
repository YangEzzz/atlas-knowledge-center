import { computed, reactive, watch } from "vue";
import { useAuthStore } from "@/store/auth";
import { importEntries, listEntries, deleteEntry, type PersonalEntry } from "@/api/entries";
import { toast } from "vue-sonner";

export type PersonalRecordKind = "project" | "document" | "public_link";

export interface PersonalRecord {
  id: string;
  kind: PersonalRecordKind;
  targetId: number;
  title: string;
  projectName?: string;
  description?: string;
  path: string;
  badge?: string;
  recordedAt: string;
}

interface PersonalRecordStorage {
  favorites: PersonalRecord[];
  recentVisits: PersonalRecord[];
}

const STORAGE_PREFIX = "ty-notebook:personal-records";
const RECENT_LIMIT = 30;
const recordCache = reactive<Record<string, PersonalRecordStorage>>({});
const syncTasks = new Map<string, Promise<void>>();
const synced = new Set<string>();
const referenceCache = new Map<string, PersonalEntry[]>();
const favoriteWrites = new Set<string>();
window.addEventListener("storage", (event) => {
  if (event.key?.startsWith(STORAGE_PREFIX)) delete recordCache[event.key];
});

function emptyStorage(): PersonalRecordStorage {
  return {
    favorites: [],
    recentVisits: [],
  };
}

function recordId(kind: PersonalRecordKind, targetId: number) {
  return `${kind}:${targetId}`;
}

function normalizeRecord(
  input: Omit<PersonalRecord, "id" | "recordedAt">,
): PersonalRecord {
  return {
    ...input,
    id: recordId(input.kind, input.targetId),
    recordedAt: new Date().toISOString(),
  };
}

export function usePersonalRecords() {
  const authStore = useAuthStore();
  const storageKey = computed(() => {
    const userId = authStore.user?.id || "guest";
    return `${STORAGE_PREFIX}:${userId}`;
  });

  const records = computed({
    get: () =>
      (recordCache[storageKey.value] ||= readStorage(storageKey.value)),
    set: (value: PersonalRecordStorage) => {
      recordCache[storageKey.value] = value;
    },
  });

  function readLatest() {
    records.value = readStorage(storageKey.value);
  }

  function persist(nextRecords: PersonalRecordStorage) {
    records.value = nextRecords;
    localStorage.setItem(storageKey.value, JSON.stringify(nextRecords));
  }

  async function syncFavorites(force = false) {
    if (!authStore.user?.id) return;
    const user = authStore.user.id;
    const key = storageKey.value;
    if (syncTasks.has(key)) return syncTasks.get(key);
    if (synced.has(key) && !force) return;
    const task = (async () => {
      // Never delete the legacy browser copy before the server acknowledges import.
      if (!synced.has(key) && localStorage.getItem(`${key}:entries-imported`)!=="1") {
        const legacy = readStorage(key).favorites;
        if(!localStorage.getItem(`${key}:legacy-favorites-backup`))localStorage.setItem(`${key}:legacy-favorites-backup`,JSON.stringify(legacy));
        for (let offset = 0; offset < legacy.length; offset += 100) {
          await importEntries(legacy.slice(offset, offset + 100).filter(item => item.targetId > 0).map(item => ({
            source: item.kind === 'public_link' ? 'tool' : item.kind,
            kind: item.kind === 'public_link' ? 'tool' : item.kind,
            targetId: item.targetId, name: (item.title || '已收藏入口').slice(0, 200),
          })));
        }
      }
      const entries: PersonalEntry[] = [];
      for (let page = 1; ; page++) {
        const result = await listEntries({ source: 'public', page, pageSize: 100 });
        entries.push(...result.data.list);
        if (entries.length >= result.data.total || !result.data.list.length) break;
      }
      if (authStore.user?.id !== user) return;
      referenceCache.set(key, entries);
      const latest = readStorage(key);
      const next = { ...latest, favorites: entries.map(entry => {
        const kind = entry.source === 'tool' ? 'public_link' : entry.source as PersonalRecordKind;
        return {id:recordId(kind,entry.targetId),kind,targetId:entry.targetId,title:entry.name,path:entry.url || '/public-links',recordedAt:entry.updatedAt};
      }) };
      recordCache[key] = next;
      localStorage.setItem(key, JSON.stringify(next));
      localStorage.setItem(`${key}:entries-imported`,"1");
      synced.add(key);
    })();
    syncTasks.set(key, task);
    try { await task; } finally { syncTasks.delete(key); }
  }
  watch(() => authStore.user?.id, () => {
    if (authStore.user?.id) void syncFavorites().catch(() => { /* Explicit retry lives on My Entries; preserve legacy records on failure. */ });
  }, { immediate: true });

  function addRecentVisit(input: Omit<PersonalRecord, "id" | "recordedAt">) {
    readLatest();
    const nextRecord = normalizeRecord(input);
    const recentVisits = [
      nextRecord,
      ...records.value.recentVisits.filter((item) => item.id !== nextRecord.id),
    ].slice(0, RECENT_LIMIT);

    persist({
      ...records.value,
      recentVisits,
    });
  }

  function clearRecentVisits() {
    readLatest();
    persist({
      ...records.value,
      recentVisits: [],
    });
  }

  async function toggleFavorite(input: Omit<PersonalRecord, "id" | "recordedAt">) {
    const user = authStore.user?.id;
    const pendingKey = `${storageKey.value}:${input.kind}:${input.targetId}`;
    if (favoriteWrites.has(pendingKey)) return;
    favoriteWrites.add(pendingKey);
    try {
      await syncFavorites();
      if (!user || user !== authStore.user?.id) return;
      const source = input.kind === 'public_link' ? 'tool' : input.kind;
      const existing = referenceCache.get(storageKey.value)?.find(item => item.source === source && item.targetId === input.targetId);
      if (existing) await deleteEntry(existing);
      else await importEntries([{source,kind:source,targetId:input.targetId,name:input.title.slice(0,200)}]);
      await syncFavorites(true);
    } catch { toast.error('收藏未能保存，请重试'); }
    finally { favoriteWrites.delete(pendingKey); }
  }

  async function removeFavorite(id: string) {
    const record = records.value.favorites.find(item => item.id === id);
    if (record) await toggleFavorite(record);
  }

  async function clearFavorites() {
    for (const record of [...records.value.favorites]) await removeFavorite(record.id);
  }

  function isFavorite(kind: PersonalRecordKind, targetId: number) {
    return records.value.favorites.some(
      (item) => item.id === recordId(kind, targetId),
    );
  }

  return {
    favorites: computed(() => records.value.favorites),
    recentVisits: computed(() => records.value.recentVisits),
    addRecentVisit,
    clearFavorites,
    clearRecentVisits,
    isFavorite,
    removeFavorite,
    toggleFavorite,
    syncFavorites,
  };
}

function readStorage(key: string): PersonalRecordStorage {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return emptyStorage();
    const parsed = JSON.parse(raw) as Partial<PersonalRecordStorage>;
    return {
      favorites: Array.isArray(parsed.favorites) ? parsed.favorites : [],
      recentVisits: Array.isArray(parsed.recentVisits)
        ? parsed.recentVisits
        : [],
    };
  } catch {
    return emptyStorage();
  }
}
