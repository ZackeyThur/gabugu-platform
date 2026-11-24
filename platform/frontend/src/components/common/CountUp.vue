<template>
  <span class="g-countup tabular-nums">
    <template v-if="decimalParts">
      <span class="g-countup-major">{{ decimalParts.major }}</span><span class="g-countup-cents">{{ decimalParts.cents }}</span><span v-if="decimalParts.suffix" class="g-countup-suffix">{{ decimalParts.suffix }}</span>
    </template>
    <template v-else>{{ formattedValue }}</template>
  </span>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    value: number | null | undefined
    /** Optional custom formatter (e.g. overviewMoney) for byte-identical display formatting */
    formatter?: (val: number) => string
    decimals?: number
    /** Optional string prepended to formatted value (avoiding DOM Element.prefix collision) */
    valuePrefix?: string
    /** Optional string appended to formatted value */
    valueSuffix?: string
    /** Initial mount animation duration in ms */
    duration?: number
    /** Update transition duration in ms when value changes after mount */
    updateDuration?: number
    locale?: boolean
    /** Render fractional cents/decimals in a recessed typographic span while preserving exact textContent */
    splitDecimals?: boolean
  }>(),
  {
    formatter: undefined,
    decimals: 0,
    valuePrefix: '',
    valueSuffix: '',
    duration: 640,
    updateDuration: 420,
    locale: true,
    splitDecimals: false,
  }
)

function shouldSkipAnimation(): boolean {
  if (import.meta.env.MODE === 'test') return true
  if (typeof window === 'undefined' || typeof window.requestAnimationFrame !== 'function') return true
  if (
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  ) {
    return true
  }
  return false
}

function normalizeTarget(raw: number | null | undefined): number {
  return typeof raw === 'number' && Number.isFinite(raw) ? raw : 0
}

const current = ref<number>(normalizeTarget(props.value))
let rafId = 0
let mounted = false

function cancelAnimation() {
  if (rafId && typeof window !== 'undefined' && typeof window.cancelAnimationFrame === 'function') {
    window.cancelAnimationFrame(rafId)
  }
  rafId = 0
}

function animateTo(from: number, to: number, durationMs: number) {
  cancelAnimation()

  if (
    shouldSkipAnimation() ||
    durationMs <= 0 ||
    Math.abs(to - from) < 1e-6 ||
    (to !== 0 && Math.abs(to) < 0.01)
  ) {
    current.value = to
    return
  }

  let startTime: number | null = null
  const delta = to - from

  const step = (now: number) => {
    if (startTime === null) startTime = now
    const elapsed = Math.max(0, now - startTime)
    const progress = Math.max(0, Math.min(1, elapsed / durationMs))
    // Ease-out quintic/expo curve: brisk takeoff, velvety glide to exact landing
    const eased = progress >= 1 ? 1 : 1 - Math.pow(2, -10 * progress)
    current.value = progress >= 1 ? to : from + delta * eased
    if (progress < 1) {
      rafId = window.requestAnimationFrame(step)
    } else {
      rafId = 0
    }
  }

  current.value = from
  rafId = window.requestAnimationFrame(step)
}

onMounted(() => {
  mounted = true
  const target = normalizeTarget(props.value)
  if (!shouldSkipAnimation() && target !== 0) {
    animateTo(0, target, props.duration)
  } else {
    current.value = target
  }
})

watch(
  () => props.value,
  (nextRaw) => {
    const target = normalizeTarget(nextRaw)
    if (!mounted || shouldSkipAnimation()) {
      current.value = target
      return
    }
    animateTo(current.value, target, props.updateDuration)
  }
)

onBeforeUnmount(() => {
  cancelAnimation()
})

const formattedValue = computed(() => {
  const raw = props.value
  if (raw == null || !Number.isFinite(raw)) {
    return props.formatter ? props.formatter(NaN) : '—'
  }
  const val = current.value
  if (props.formatter) {
    return props.formatter(val)
  }
  const numStr = props.locale
    ? val.toLocaleString('en-US', {
        minimumFractionDigits: props.decimals,
        maximumFractionDigits: props.decimals,
      })
    : val.toFixed(props.decimals)
  return `${props.valuePrefix}${numStr}${props.valueSuffix}`
})

const decimalParts = computed(() => {
  if (!props.splitDecimals) return null
  const text = formattedValue.value
  if (!text || text.includes('<')) return null
  const match = /^([^\d]*\d[\d,]*)(\.\d+)(.*)$/.exec(text)
  if (!match) return null
  return { major: match[1], cents: match[2], suffix: match[3] || '' }
})
</script>
