import { computed, getCurrentScope, onScopeDispose, readonly, ref, type ComputedRef, type Ref } from 'vue'

export const CHART_FONT_FAMILY = "Inter, 'Gabugu Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif"

export interface ChartThemeTokens {
  isDark: boolean
  surface: string
  tooltipBg: string
  tooltipBorder: string
  tooltipTitle: string
  tooltipBody: string
  text: string
  tick: string
  grid: string
  border: string
  crosshair: string
  mutedBar: string
  primary: string
  secondary: string
  emerald: string
  violet: string
  amber: string
  danger: string
  other: string
  series: readonly string[]
}

const LIGHT_TOKENS: ChartThemeTokens = {
  isDark: false,
  surface: '#FBF7F0',
  tooltipBg: '#FBF7F0',
  tooltipBorder: '#DACBB4',
  tooltipTitle: '#221D17',
  tooltipBody: '#574E43',
  text: '#221D17',
  tick: '#6A6052',
  grid: 'rgba(168, 146, 118, 0.22)',
  border: '#DACBB4',
  crosshair: 'rgba(45, 88, 144, 0.28)',
  mutedBar: '#A9BFDB',
  primary: '#2D5890',
  secondary: '#A5532F',
  emerald: '#1F7050',
  violet: '#6D5CE7',
  amber: '#8A6209',
  danger: '#C33441',
  other: '#94a3b8',
  series: [
    '#2D5890',
    '#A5532F',
    '#1F7050',
    '#6D5CE7',
    '#8A6209',
    '#4F7AA8',
    '#A04E16',
    '#7A6A58',
  ],
}

const DARK_TOKENS: ChartThemeTokens = {
  isDark: true,
  surface: '#151D30',
  tooltipBg: '#151D30',
  tooltipBorder: 'rgba(176, 204, 255, 0.16)',
  tooltipTitle: '#EEF2FB',
  tooltipBody: '#C4D2EA',
  text: '#EEF2FB',
  tick: '#8D9EBC',
  grid: 'rgba(176, 204, 255, 0.08)',
  border: 'rgba(176, 204, 255, 0.14)',
  crosshair: 'rgba(124, 162, 255, 0.34)',
  mutedBar: '#2B4A7A',
  primary: '#7CA2FF',
  secondary: '#A78BFA',
  emerald: '#38D3B4',
  violet: '#A78BFA',
  amber: '#F6D494',
  danger: '#F87171',
  other: '#64789C',
  series: [
    '#7CA2FF',
    '#A78BFA',
    '#38D3B4',
    '#F6D494',
    '#F29664',
    '#60A5FA',
    '#C084FC',
    '#64789C',
  ],
}

function readDomDark(): boolean {
  if (typeof document === 'undefined' || !document.documentElement) return false
  return document.documentElement.classList.contains('dark')
}

const sharedIsDark = ref<boolean>(readDomDark())
let observerRefCount = 0
let sharedObserver: MutationObserver | null = null

function syncSharedTheme() {
  sharedIsDark.value = readDomDark()
}

function attachObserver() {
  syncSharedTheme()
  observerRefCount += 1
  if (observerRefCount === 1 && typeof document !== 'undefined' && typeof MutationObserver !== 'undefined') {
    sharedObserver = new MutationObserver(() => {
      syncSharedTheme()
    })
    sharedObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
  }
}

function detachObserver() {
  observerRefCount = Math.max(0, observerRefCount - 1)
  if (observerRefCount === 0 && sharedObserver) {
    sharedObserver.disconnect()
    sharedObserver = null
  }
}

export function __getChartThemeObserverStateForTests() {
  return {
    refCount: observerRefCount,
    active: sharedObserver !== null,
  }
}

export interface UseChartThemeReturn {
  isDark: Readonly<Ref<boolean>>
  tokens: ComputedRef<ChartThemeTokens>
  seriesColor: (index: number) => string
  seriesColors: (count: number) => string[]
  tooltipTheme: (overrides?: Record<string, any>) => Record<string, any>
  axisTheme: (overrides?: {
    showGrid?: boolean
    dashedGrid?: boolean
    ticks?: Record<string, any>
    grid?: Record<string, any>
    border?: Record<string, any>
    title?: Record<string, any>
  }) => Record<string, any>
  legendTheme: (overrides?: Record<string, any>) => Record<string, any>
}

export function useChartTheme(): UseChartThemeReturn {
  if (getCurrentScope()) {
    attachObserver()
    onScopeDispose(() => {
      detachObserver()
    })
  } else {
    syncSharedTheme()
  }

  const tokens = computed<ChartThemeTokens>(() => (sharedIsDark.value ? DARK_TOKENS : LIGHT_TOKENS))

  const seriesColor = (index: number): string => {
    const palette = tokens.value.series
    const safeIdx = ((index % palette.length) + palette.length) % palette.length
    return palette[safeIdx]
  }

  const seriesColors = (count: number): string[] => {
    const out: string[] = []
    for (let i = 0; i < count; i += 1) {
      out.push(seriesColor(i))
    }
    return out
  }

  const tooltipTheme = (overrides: Record<string, any> = {}) => {
    const t = tokens.value
    return {
      backgroundColor: t.tooltipBg,
      titleColor: t.tooltipTitle,
      bodyColor: t.tooltipBody,
      borderColor: t.tooltipBorder,
      borderWidth: 1,
      cornerRadius: 12,
      padding: 12,
      displayColors: true,
      boxPadding: 5,
      titleFont: { family: CHART_FONT_FAMILY, size: 13, weight: 600 },
      bodyFont: { family: CHART_FONT_FAMILY, size: 13, weight: 500 },
      ...overrides,
    }
  }

  const axisTheme = (
    overrides: {
      showGrid?: boolean
      dashedGrid?: boolean
      ticks?: Record<string, any>
      grid?: Record<string, any>
      border?: Record<string, any>
      title?: Record<string, any>
    } = {}
  ) => {
    const t = tokens.value
    const showGrid = overrides.showGrid ?? true
    const dashedGrid = overrides.dashedGrid ?? true
    return {
      border: {
        display: false,
        ...(dashedGrid ? { dash: [4, 4] } : {}),
        ...(overrides.border ?? {}),
      },
      grid: {
        display: showGrid,
        color: t.grid,
        ...(overrides.grid ?? {}),
      },
      ticks: {
        color: t.tick,
        font: { family: CHART_FONT_FAMILY, size: 13, weight: 500 },
        ...(overrides.ticks ?? {}),
      },
      ...(overrides.title ? { title: { color: t.tick, font: { family: CHART_FONT_FAMILY, size: 13 }, ...overrides.title } } : {}),
    }
  }

  const legendTheme = (overrides: Record<string, any> = {}) => {
    const t = tokens.value
    const { labels: labelOverrides, ...rest } = overrides
    return {
      position: 'top' as const,
      ...rest,
      labels: {
        color: t.tick,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 14,
        font: { family: CHART_FONT_FAMILY, size: 13, weight: 500 },
        ...(labelOverrides ?? {}),
      },
    }
  }

  return {
    isDark: readonly(sharedIsDark),
    tokens,
    seriesColor,
    seriesColors,
    tooltipTheme,
    axisTheme,
    legendTheme,
  }
}

export { LIGHT_TOKENS as LIGHT_CHART_TOKENS, DARK_TOKENS as DARK_CHART_TOKENS }
