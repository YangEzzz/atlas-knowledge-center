interface PagePosition {
  top: number
  left: number
}

let stopPendingRestore: (() => void) | undefined

export function cancelPageScrollRestoration() {
  stopPendingRestore?.()
}

// Route data can arrive after Vue Router's nextTick. Keep the requested history
// position until the page is tall enough, unless the user takes over scrolling.
export function restorePageScroll(position: PagePosition) {
  cancelPageScrollRestoration()
  const viewport = document.getElementById('app')
  if (!viewport)
    return
  const apply = () => {
    viewport.scrollTo({ ...position, behavior: 'instant' })
    return viewport.scrollHeight - viewport.clientHeight >= position.top
      && viewport.scrollWidth - viewport.clientWidth >= position.left
  }
  if (apply())
    return

  let frame = 0
  let mutations: MutationObserver | undefined
  let resize: ResizeObserver | undefined
  let timeout = 0
  const events = ['wheel', 'touchstart', 'pointerdown', 'keydown'] as const
  const stop = () => {
    mutations?.disconnect()
    resize?.disconnect()
    cancelAnimationFrame(frame)
    clearTimeout(timeout)
    events.forEach(event => viewport.removeEventListener(event, stop, true))
    if (stopPendingRestore === stop)
      stopPendingRestore = undefined
  }
  const schedule = () => {
    cancelAnimationFrame(frame)
    frame = requestAnimationFrame(() => {
      Array.from(viewport.children).forEach(child => resize?.observe(child))
      if (apply())
        stop()
    })
  }
  mutations = new MutationObserver(schedule)
  resize = new ResizeObserver(schedule)
  timeout = window.setTimeout(stop, 15000)
  mutations.observe(viewport, { childList: true, subtree: true, characterData: true })
  Array.from(viewport.children).forEach(child => resize?.observe(child))
  events.forEach(event => viewport.addEventListener(event, stop, { capture: true, passive: true }))
  stopPendingRestore = stop
}
