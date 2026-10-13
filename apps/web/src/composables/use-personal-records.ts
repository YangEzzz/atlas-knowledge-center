import { computed, ref } from 'vue'
import { useAuthStore } from '@/store/auth'

export type PersonalRecordKind = 'project' | 'document' | 'public_link'

export interface PersonalRecord {
  id: string
  kind: PersonalRecordKind
  targetId: number
  title: string
  projectName?: string
  description?: string
  path: string
  badge?: string
  recordedAt: string
}

interface PersonalRecordStorage {
  favorites: PersonalRecord[]
  recentVisits: PersonalRecord[]
}

const STORAGE_PREFIX = 'ty-notebook:personal-records'
const RECENT_LIMIT = 30

function emptyStorage(): PersonalRecordStorage {
  return {
    favorites: [],
    recentVisits: [],
  }
}

function recordId(kind: PersonalRecordKind, targetId: number) {
  return `${kind}:${targetId}`
}

function normalizeRecord(input: Omit<PersonalRecord, 'id' | 'recordedAt'>): PersonalRecord {
  return {
    ...input,
    id: recordId(input.kind, input.targetId),
    recordedAt: new Date().toISOString(),
  }
}

export function usePersonalRecords() {
  const authStore = useAuthStore()
  const storageKey = computed(() => {
    const userId = authStore.user?.id || 'guest'
    return `${STORAGE_PREFIX}:${userId}`
  })

  const records = ref<PersonalRecordStorage>(readStorage(storageKey.value))

  function readLatest() {
    records.value = readStorage(storageKey.value)
  }

  function persist(nextRecords: PersonalRecordStorage) {
    records.value = nextRecords
    localStorage.setItem(storageKey.value, JSON.stringify(nextRecords))
  }

  function addRecentVisit(input: Omit<PersonalRecord, 'id' | 'recordedAt'>) {
    readLatest()
    const nextRecord = normalizeRecord(input)
    const recentVisits = [
      nextRecord,
      ...records.value.recentVisits.filter((item) => item.id !== nextRecord.id),
    ].slice(0, RECENT_LIMIT)

    persist({
      ...records.value,
      recentVisits,
    })
  }

  function clearRecentVisits() {
    readLatest()
    persist({
      ...records.value,
      recentVisits: [],
    })
  }

  function toggleFavorite(input: Omit<PersonalRecord, 'id' | 'recordedAt'>) {
    readLatest()
    const nextRecord = normalizeRecord(input)
    const exists = records.value.favorites.some((item) => item.id === nextRecord.id)
    const favorites = exists
      ? records.value.favorites.filter((item) => item.id !== nextRecord.id)
      : [nextRecord, ...records.value.favorites]

    persist({
      ...records.value,
      favorites,
    })
  }

  function removeFavorite(id: string) {
    readLatest()
    persist({
      ...records.value,
      favorites: records.value.favorites.filter((item) => item.id !== id),
    })
  }

  function clearFavorites() {
    readLatest()
    persist({
      ...records.value,
      favorites: [],
    })
  }

  function isFavorite(kind: PersonalRecordKind, targetId: number) {
    return records.value.favorites.some((item) => item.id === recordId(kind, targetId))
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
  }
}

function readStorage(key: string): PersonalRecordStorage {
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return emptyStorage()
    const parsed = JSON.parse(raw) as Partial<PersonalRecordStorage>
    return {
      favorites: Array.isArray(parsed.favorites) ? parsed.favorites : [],
      recentVisits: Array.isArray(parsed.recentVisits) ? parsed.recentVisits : [],
    }
  } catch {
    return emptyStorage()
  }
}
