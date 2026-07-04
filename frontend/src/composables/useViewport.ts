import { ref, onMounted, onUnmounted } from 'vue'

/**
 * 全站统一的视口断点判断。
 *
 * 背景:此前 TablePageLayout 用 innerWidth<1024、DataTable 用 matchMedia(min-width:768px),
 * 各处断点不一致,导致 768–1024px(大屏手机横屏 / 竖屏平板)时"页面框架"与"表格内容"
 * 对是否移动端判断打架、显示错位。此 composable 统一断点为 md(768px),并用 matchMedia
 * 事件(无需 resize 高频轮询)驱动,天然无抖动、无性能问题。
 *
 * isMobile === true 表示视口 < 768px。
 */

// Tailwind md 断点。桌面视口 = 宽度 >= 768px。
const DESKTOP_QUERY = '(min-width: 768px)'

export function useViewport() {
  const isDesktop = ref(
    typeof window === 'undefined' ? true : window.matchMedia(DESKTOP_QUERY).matches
  )
  // isMobile 是 isDesktop 的取反,方便旧调用点直接替换。
  const isMobile = ref(!isDesktop.value)

  let mql: MediaQueryList | null = null

  const update = (matches: boolean) => {
    isDesktop.value = matches
    isMobile.value = !matches
  }

  const onChange = (e: MediaQueryListEvent) => update(e.matches)

  onMounted(() => {
    if (typeof window === 'undefined') return
    mql = window.matchMedia(DESKTOP_QUERY)
    update(mql.matches)
    // 现代浏览器用 addEventListener;matchMedia 事件只在跨越断点时触发,无高频回调。
    if (typeof mql.addEventListener === 'function') {
      mql.addEventListener('change', onChange)
    } else {
      // 老浏览器兜底(已弃用的 addListener)
      mql.addListener(onChange)
    }
  })

  onUnmounted(() => {
    if (!mql) return
    if (typeof mql.removeEventListener === 'function') {
      mql.removeEventListener('change', onChange)
    } else {
      mql.removeListener(onChange)
    }
    mql = null
  })

  return { isMobile, isDesktop }
}
