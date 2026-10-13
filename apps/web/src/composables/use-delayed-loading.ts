import { onScopeDispose, ref, watch, type Ref } from 'vue'

export function useDelayedLoading(source: Ref<boolean>, delay = 220) {
  const visible = ref(false)
  let timer: ReturnType<typeof setTimeout> | null = null

  function clearDelayTimer() {
    if (!timer) return
    clearTimeout(timer)
    timer = null
  }

  watch(
    source,
    (loading) => {
      clearDelayTimer()

      if (!loading) {
        visible.value = false
        return
      }

      timer = setTimeout(() => {
        visible.value = true
        timer = null
      }, delay)
    },
    { immediate: true },
  )

  onScopeDispose(clearDelayTimer)

  return visible
}
