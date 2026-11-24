import type { Chart, Plugin, TooltipModel } from 'chart.js'

export interface GlassTooltipRow {
  label: string
  value: string
  color?: string
  muted?: boolean
}

export interface GlassTooltipContent {
  title: string
  subtitle?: string
  rows: GlassTooltipRow[]
  footer?: string
  footerTone?: 'up' | 'down' | 'neutral'
  hint?: string
}

export function hexToRgba(hex: string, alpha: number): string {
  const clean = hex.replace('#', '').trim()
  if (clean.length === 3) {
    const r = parseInt(clean[0] + clean[0], 16)
    const g = parseInt(clean[1] + clean[1], 16)
    const b = parseInt(clean[2] + clean[2], 16)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }
  if (clean.length === 6) {
    const r = parseInt(clean.slice(0, 2), 16)
    const g = parseInt(clean.slice(2, 4), 16)
    const b = parseInt(clean.slice(4, 6), 16)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }
  return hex
}

export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return false
  }
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

export function chartAnimationConfig() {
  if (prefersReducedMotion()) {
    return false as const
  }
  return {
    duration: 420,
    easing: 'easeOutQuart' as const,
  }
}

/**
 * Formats currency Y-axis ticks adaptively so micro-spends (< $0.01) never collapse to repeated $0.00 ticks.
 */
export function formatAdaptiveCurrency(value: number, maxInSeries: number): string {
  if (!Number.isFinite(value)) return '$0'
  if (Math.abs(value) < 1e-9) return '$0'
  const abs = Math.abs(value)
  const sign = value < 0 ? '-' : ''
  if (abs >= 1000) {
    return `${sign}$${(abs / 1000).toFixed(1)}K`
  }
  if (maxInSeries > 0 && maxInSeries < 0.01) {
    return `${sign}$${abs.toFixed(4)}`
  }
  if (maxInSeries > 0 && maxInSeries < 0.1) {
    return `${sign}$${abs.toFixed(3)}`
  }
  return `${sign}$${abs.toFixed(2)}`
}

/**
 * Chart.js v4 plugin that draws an ambient vertical spotlight column & dashed vertical crosshair
 * snapped to the hovered/active bucket.
 */
export function createCrosshairPlugin(
  getColor: () => string,
  getColumnWash?: () => string
): Plugin {
  return {
    id: 'gabuguCrosshair',
    beforeDatasetsDraw(chart: Chart) {
      if (!getColumnWash) return
      const tipActive = chart.tooltip?.getActiveElements?.()
      const active = tipActive && tipActive.length > 0 ? tipActive : (chart.getActiveElements?.() ?? [])
      if (!active.length) return
      const x = active[0]?.element?.x
      if (typeof x !== 'number' || !Number.isFinite(x)) return
      const { ctx, chartArea, scales } = chart
      if (!chartArea) return
      const count = Math.max(1, scales?.x?.ticks?.length ?? 7)
      const colWidth = Math.min(64, Math.max(26, (chartArea.right - chartArea.left) / count * 0.84))
      ctx.save()
      ctx.fillStyle = getColumnWash()
      const rx = x - colWidth / 2
      const ry = chartArea.top
      const rw = colWidth
      const rh = chartArea.bottom - chartArea.top
      if (typeof ctx.roundRect === 'function') {
        ctx.beginPath()
        ctx.roundRect(rx, ry, rw, rh, 10)
        ctx.fill()
      } else {
        ctx.fillRect(rx, ry, rw, rh)
      }
      ctx.restore()
    },
    afterDatasetsDraw(chart: Chart) {
      const tipActive = chart.tooltip?.getActiveElements?.()
      const active = tipActive && tipActive.length > 0 ? tipActive : (chart.getActiveElements?.() ?? [])
      if (!active.length) return
      const x = active[0]?.element?.x
      if (typeof x !== 'number' || !Number.isFinite(x)) return
      const { ctx, chartArea } = chart
      if (!chartArea) return
      ctx.save()
      ctx.beginPath()
      ctx.setLineDash([3, 3])
      ctx.lineWidth = 1
      ctx.strokeStyle = getColor()
      ctx.moveTo(x, chartArea.top)
      ctx.lineTo(x, chartArea.bottom)
      ctx.stroke()
      ctx.restore()
    },
  }
}

export interface MeanLineConfig {
  visible: boolean
  value: number
  label: string
  prefix?: string
  valueText?: string
  lineColor: string
  pillBg: string
  pillBorder: string
  pillText: string
  mutedText?: string
  fontFamily: string
}

/**
 * Draws a whisper-fine horizontal dashed mean line ending in a crisp borderless typographic readout
 * docked cleanly in the right gutter outside chartArea (zero pill capsule, zero bar overlap).
 */
export function createMeanLinePlugin(getConfig: () => MeanLineConfig | null): Plugin {
  return {
    id: 'gabuguMeanLine',
    afterDatasetsDraw(chart: Chart) {
      const cfg = getConfig()
      if (!cfg || !cfg.visible || !Number.isFinite(cfg.value) || cfg.value <= 0) return
      const { ctx, chartArea, scales, width } = chart
      const yScale = scales?.y
      if (!chartArea || !yScale) return
      const y = yScale.getPixelForValue(cfg.value)
      if (!Number.isFinite(y) || y < chartArea.top - 2 || y > chartArea.bottom + 2) return

      ctx.save()
      ctx.font = `500 12.5px ${cfg.fontFamily}`
      const totalWidth = ctx.measureText(cfg.label).width
      const rightGutter = width - chartArea.right
      const useRightGutter = rightGutter >= totalWidth + 14

      // Whisper-fine horizontal dashed reference line
      ctx.beginPath()
      ctx.setLineDash([3, 4])
      ctx.lineWidth = 1
      ctx.strokeStyle = cfg.lineColor
      ctx.moveTo(chartArea.left, y)
      ctx.lineTo(useRightGutter ? chartArea.right + 4 : chartArea.right, y)
      ctx.stroke()
      ctx.setLineDash([])

      // Only draw gutter label when dedicated right gutter has clean room (zero collision on mobile)
      if (!useRightGutter) {
        ctx.restore()
        return
      }

      // Precision anchor dot at line terminus
      ctx.beginPath()
      ctx.arc(chartArea.right + 5, y, 2, 0, Math.PI * 2)
      ctx.fillStyle = cfg.lineColor
      ctx.fill()

      // Borderless split typographic readout (muted prefix + crisp tabular value)
      const textX = chartArea.right + 12
      const clampedY = Math.max(chartArea.top + 8, Math.min(chartArea.bottom - 8, y))
      ctx.textBaseline = 'middle'
      ctx.textAlign = 'left'

      if (cfg.prefix && cfg.valueText) {
        ctx.font = `500 12.5px ${cfg.fontFamily}`
        ctx.fillStyle = cfg.mutedText || cfg.pillText
        ctx.fillText(cfg.prefix, textX, clampedY + 0.5)
        const prefixW = ctx.measureText(cfg.prefix + ' ').width
        ctx.font = `600 12.5px ${cfg.fontFamily}`
        ctx.fillStyle = cfg.pillText
        ctx.fillText(cfg.valueText, textX + prefixW, clampedY + 0.5)
      } else {
        ctx.font = `500 12.5px ${cfg.fontFamily}`
        ctx.fillStyle = cfg.pillText
        ctx.fillText(cfg.label, textX, clampedY + 0.5)
      }
      ctx.restore()
    },
  }
}

export interface BarCraftConfig {
  enabled: boolean
  todayIndex: number | null
  hoveredIndex: number | null
  specularColor: string
  hatchColor: string
}

/**
 * Adds subtle flagship craftsmanship to bar charts:
 * 1) A crisp 2px luminous top specular highlight inside each bar's top rim.
 * 2) A delicate clipped 45deg diagonal hatch overlay on today's in-progress bar.
 */
export function createBarCraftPlugin(getConfig: () => BarCraftConfig): Plugin {
  return {
    id: 'gabuguBarCraft',
    afterDatasetsDraw(chart: Chart) {
      const cfg = getConfig()
      if (!cfg.enabled) return
      const meta = chart.getDatasetMeta(0)
      if (!meta || meta.type !== 'bar' || meta.hidden) return
      const { ctx, chartArea } = chart
      if (!chartArea) return

      ctx.save()
      meta.data.forEach((el: any, idx: number) => {
        if (!el || typeof el.getProps !== 'function') return
        const { x, y, base, width: barW } = el.getProps(['x', 'y', 'base', 'width'], true)
        if (!Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(base) || !Number.isFinite(barW)) return
        const barH = base - y
        if (barH < 6 || barW < 6) return

        const left = x - barW / 2
        const isDimmed = cfg.hoveredIndex !== null && cfg.hoveredIndex !== idx

        // Subtle diagonal hatch inside today's in-progress bar
        if (cfg.todayIndex === idx) {
          ctx.save()
          ctx.beginPath()
          if (typeof ctx.roundRect === 'function') {
            ctx.roundRect(left, y, barW, barH, [8, 8, 2, 2])
          } else {
            ctx.rect(left, y, barW, barH)
          }
          ctx.clip()
          ctx.strokeStyle = cfg.hatchColor
          ctx.lineWidth = 1.25
          ctx.globalAlpha = isDimmed ? 0.45 : 1
          const step = 9
          for (let offset = -barH; offset < barW + barH; offset += step) {
            ctx.beginPath()
            ctx.moveTo(left + offset, base)
            ctx.lineTo(left + offset + barH, y)
            ctx.stroke()
          }
          ctx.restore()
        }

        // Crisp luminous top specular rim
        const inset = Math.min(6, barW * 0.2)
        ctx.save()
        ctx.globalAlpha = isDimmed ? 0.25 : 0.9
        ctx.strokeStyle = cfg.specularColor
        ctx.lineWidth = 1.6
        ctx.lineCap = 'round'
        ctx.beginPath()
        ctx.moveTo(left + inset, y + 1.5)
        ctx.lineTo(left + barW - inset, y + 1.5)
        ctx.stroke()
        ctx.restore()
      })
      ctx.restore()
    },
  }
}

export interface PeakCalloutConfig {
  enabled: boolean
  peakIndex: number | null
  label: string
  hoveredIndex: number | null
  textColor: string
  dotColor: string
  fontFamily: string
}

/**
 * Draws a whisper-clean floating peak value anchor above the highest bar or point
 * when the series has concise cardinality (<= 14 buckets), giving instant scannability.
 */
export function createPeakCalloutPlugin(getConfig: () => PeakCalloutConfig): Plugin {
  return {
    id: 'gabuguPeakCallout',
    afterDatasetsDraw(chart: Chart) {
      const cfg = getConfig()
      if (!cfg.enabled || cfg.peakIndex === null || !cfg.label) return
      const meta = chart.getDatasetMeta(0)
      if (!meta || meta.hidden) return
      const el: any = meta.data[cfg.peakIndex]
      if (!el || typeof el.getProps !== 'function') return
      const { x, y } = el.getProps(['x', 'y'], true)
      if (!Number.isFinite(x) || !Number.isFinite(y)) return
      const { ctx, chartArea } = chart
      if (!chartArea) return

      // Hide callout if hovered elsewhere so tooltip focus stays pure
      const isDimmed = cfg.hoveredIndex !== null && cfg.hoveredIndex !== cfg.peakIndex

      ctx.save()
      ctx.globalAlpha = isDimmed ? 0.28 : 0.92
      ctx.font = `600 12px ${cfg.fontFamily}`
      ctx.fillStyle = cfg.textColor
      ctx.textAlign = 'center'
      ctx.textBaseline = 'bottom'
      const textY = Math.max(chartArea.top + 12, y - 8)
      ctx.fillText(cfg.label, x, textY)

      // Subtle glowing crown dot right under label on line charts
      if (meta.type === 'line') {
        ctx.beginPath()
        ctx.arc(x, y, 5.5, 0, Math.PI * 2)
        ctx.fillStyle = hexToRgba(cfg.dotColor, 0.22)
        ctx.fill()
      }
      ctx.restore()
    },
  }
}

export function hideGlassTooltip(container: HTMLElement | null | undefined) {
  const el = container?.querySelector<HTMLDivElement>(':scope > .gabugu-chart-tooltip')
  if (el) {
    el.style.opacity = '0'
    el.style.visibility = 'hidden'
    el.style.transform = 'translate3d(0, 0, 0)'
  }
}

/**
 * Creates a Chart.js external HTML tooltip renderer styled with GABUGU frosted surface tokens.
 * Uses pure DOM textContent construction (zero innerHTML) for strict XSS safety.
 */
export function createGlassTooltipHandler(
  format: (context: { chart: Chart; tooltip: TooltipModel<any> }) => GlassTooltipContent | null
) {
  return (context: { chart: Chart; tooltip: TooltipModel<any> }) => {
    const { chart, tooltip } = context
    const parent = chart.canvas?.parentNode as HTMLElement | null
    if (!parent || typeof document === 'undefined') return

    let el = parent.querySelector<HTMLDivElement>(':scope > .gabugu-chart-tooltip')
    if (!el) {
      el = document.createElement('div')
      el.className = 'gabugu-chart-tooltip'
      el.setAttribute('role', 'status')
      el.style.opacity = '0'
      el.style.visibility = 'hidden'
      el.style.pointerEvents = 'none'
      parent.appendChild(el)
    }

    if (tooltip.opacity === 0) {
      el.style.opacity = '0'
      el.style.visibility = 'hidden'
      el.style.transform = 'translate3d(0, 0, 0)'
      return
    }

    const content = format(context)
    if (!content) {
      el.style.opacity = '0'
      el.style.visibility = 'hidden'
      el.style.transform = 'translate3d(0, 0, 0)'
      return
    }

    el.replaceChildren()

    const header = document.createElement('div')
    header.className = 'gabugu-chart-tooltip-header'
    const titleEl = document.createElement('div')
    titleEl.className = 'gabugu-chart-tooltip-title'
    titleEl.textContent = content.title
    header.appendChild(titleEl)
    if (content.subtitle) {
      const subEl = document.createElement('div')
      subEl.className = 'gabugu-chart-tooltip-sub'
      subEl.textContent = content.subtitle
      header.appendChild(subEl)
    }
    el.appendChild(header)

    if (content.rows.length > 0) {
      const list = document.createElement('div')
      list.className = 'gabugu-chart-tooltip-rows'
      for (const row of content.rows) {
        const rowEl = document.createElement('div')
        rowEl.className = row.muted
          ? 'gabugu-chart-tooltip-row is-muted'
          : 'gabugu-chart-tooltip-row'

        const labelWrap = document.createElement('span')
        labelWrap.className = 'gabugu-chart-tooltip-label'
        if (row.color) {
          const dot = document.createElement('span')
          dot.className = 'gabugu-chart-tooltip-dot'
          dot.style.backgroundColor = row.color
          labelWrap.appendChild(dot)
        }
        const labelText = document.createElement('span')
        labelText.textContent = row.label
        labelWrap.appendChild(labelText)

        const valueEl = document.createElement('strong')
        valueEl.className = 'gabugu-chart-tooltip-value'
        valueEl.textContent = row.value

        rowEl.appendChild(labelWrap)
        rowEl.appendChild(valueEl)
        list.appendChild(rowEl)
      }
      el.appendChild(list)
    }

    if (content.footer) {
      const footerEl = document.createElement('div')
      footerEl.className = content.footerTone
        ? `gabugu-chart-tooltip-footer is-${content.footerTone}`
        : 'gabugu-chart-tooltip-footer'
      footerEl.textContent = content.footer
      el.appendChild(footerEl)
    }

    if (content.hint) {
      const hintEl = document.createElement('div')
      hintEl.className = 'gabugu-chart-tooltip-hint'
      hintEl.textContent = content.hint
      el.appendChild(hintEl)
    }

    el.style.visibility = 'visible'
    el.style.opacity = '1'

    const { offsetLeft, offsetTop } = chart.canvas
    const parentWidth = parent.clientWidth || 320
    const parentHeight = parent.clientHeight || 240
    const tipWidth = el.offsetWidth || 190
    const tipHeight = el.offsetHeight || 96

    let left = offsetLeft + tooltip.caretX + 14
    if (left + tipWidth > parentWidth - 8) {
      left = offsetLeft + tooltip.caretX - tipWidth - 14
    }
    left = Math.max(8, Math.min(left, Math.max(8, parentWidth - tipWidth - 8)))

    let top = offsetTop + tooltip.caretY - tipHeight / 2
    top = Math.max(8, Math.min(top, Math.max(8, parentHeight - tipHeight - 8)))

    el.style.transform = `translate3d(${Math.round(left)}px, ${Math.round(top)}px, 0)`
  }
}
