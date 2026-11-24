/**
 * GABUGU 画室 — 全站主题继承与双向同步工具。
 *
 * 与 `src/main.ts` (`initThemeClass`)、`AppSidebar.vue`、`KeyUsageView.vue` 共用
 * `document.documentElement.classList('dark')` 与 `localStorage['theme']`，
 * 确保用户进入画室时 100% 继承进入前的浅色/深色主题，在画室内切换时也与全站保持一致。
 */

export type StudioTheme = 'light' | 'dark'

export const SITE_THEME_STORAGE_KEY = 'theme'
const LEGACY_STUDIO_THEME_STORAGE_KEY = 'gabugu-studio-theme-v2'

export function readInheritedStudioTheme(): StudioTheme {
  if (typeof document !== 'undefined' && document.documentElement.classList.contains('dark')) {
    return 'dark'
  }
  try {
    const saved = localStorage.getItem(SITE_THEME_STORAGE_KEY)
    if (saved === 'dark') return 'dark'
    if (saved === 'light') return 'light'
  } catch {
    // 忽略无痕模式存储限制
  }
  if (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-color-scheme: dark)').matches
  ) {
    return 'dark'
  }
  return 'light'
}

export function applyStudioTheme(theme: StudioTheme): void {
  if (typeof document !== 'undefined') {
    document.documentElement.classList.toggle('dark', theme === 'dark')
  }
  try {
    localStorage.setItem(SITE_THEME_STORAGE_KEY, theme)
    localStorage.removeItem(LEGACY_STUDIO_THEME_STORAGE_KEY)
  } catch {
    // 忽略无痕模式存储限制
  }
}
