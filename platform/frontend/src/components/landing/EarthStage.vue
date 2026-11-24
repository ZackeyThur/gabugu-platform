<template>
  <div
    ref="box"
    class="earth-stage"
    :class="{ live }"
    aria-hidden="true"
  >
    <!-- Over a slow connection the still picture stands in until the live one is ready. Its small version
         is fetched at once, ahead of the maps, so that it is there when it is needed. -->
    <div
      v-if="posterKept"
      class="earth-stage-poster"
      :class="{ on: posterShown }"
    >
      <HeroPoster
        small
        :anchor="posterAnchor"
      />
    </div>
    <div
      ref="mount"
      class="earth-stage-canvas"
    ></div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import HeroPoster from './HeroPoster.vue'
import { earthPixelRatio, loadEarthImages, type EarthJourney, type EarthMaps, type EarthQuality, type EarthSurface } from './earth/context'
import type { EarthScene } from './earth/scene'
import type { JourneyState } from './earth/story'
import day4k from '@/assets/landing/earth-day-4k.webp'
import lights4k from '@/assets/landing/earth-lights-4k.webp'
import clouds4k from '@/assets/landing/earth-clouds-4k.webp'
import water4k from '@/assets/landing/earth-water-4k.webp'
import day2k from '@/assets/landing/earth-day-2k.webp'
import lights2k from '@/assets/landing/earth-lights-2k.webp'
import clouds2k from '@/assets/landing/earth-clouds-2k.webp'
import water2k from '@/assets/landing/earth-water-2k.webp'

/**
 * The live Earth behind the whole page. The page writes where it is into `journey`; this component
 * downloads the scene, draws it, follows the journey smoothly, and gives up cleanly (`failed`) where the
 * machine cannot keep up, so that the page can show its still picture instead.
 */
const MAPS: Record<EarthQuality, EarthMaps<string>> = {
  high: { day: day4k, lights: lights4k, clouds: clouds4k, water: water4k },
  low: { day: day2k, lights: lights2k, clouds: clouds2k, water: water2k }
}
/** The sunrise of the opening, in milliseconds: in full on a first visit, briefly on a return. */
const OPENING_MS = { full: 3400, short: 1500, none: 0 }
/**
 * After this long without the live picture the still one is shown in its place: early where not one of
 * the maps has arrived by then, which means a slow connection, and otherwise only after a longer wait,
 * so that an ordinary connection never sees it.
 */
const POSTER_EARLY_MS = 1400
const POSTER_AFTER_MS = 3200
/** A frame that costs the graphics card more than this is drawn less sharply; more than the second, not at all. */
const HEAVY_FRAME_MS = 20
const HOPELESS_FRAME_MS = 34
/**
 * While nothing moves but the clouds, the grain and the ground sliding by, frames are this far apart; and
 * once the visitor has left the page alone for a while, further apart still, to spare the battery.
 */
const IDLE_FRAME_MS = 33
const RESTING_FRAME_MS = 83
const RESTING_AFTER_MS = 20000

const props = defineProps<{
  surface: EarthSurface
  journey: EarthJourney
  opening: 'full' | 'short' | 'none'
  /** performance.now() at which the first screen's words began to appear; 0 until then. */
  after: number
}>()
const emit = defineEmits<{ ready: [], failed: [] }>()

const box = ref<HTMLElement | null>(null)
const mount = ref<HTMLElement | null>(null)
const live = ref(false)
const posterShown = ref(false)
const posterAnchor = ref(0)
/** False once the live picture has taken over and the still one is no longer needed. */
const posterKept = ref(true)

const clamp = (value: number) => Math.min(1, Math.max(0, value))

const state: JourneyState = { time: 0, intro: props.opening === 'none' ? 1 : 0, progress: 0, pointer: { x: 0, y: 0 } }
let scene: EarthScene | undefined
let ratio = 1
let stopped = false
let frame = 0
let last = 0
let drawnAt = 0
let startedAt = 0
let introFrom = 0
let slowFrames = 0
/** Set once the picture has been made less sharp because this machine could not keep up. */
let lowered = false
/** The screen's pixel density the picture was last sized for. */
let density = 0
let posterTimer: ReturnType<typeof setTimeout> | undefined
let posterGone: ReturnType<typeof setTimeout> | undefined
let idleTimer: ReturnType<typeof setTimeout> | undefined
let mapsArrived = 0
let observer: ResizeObserver | undefined
const abort = new AbortController()

function closeEarthImages(images: EarthMaps<unknown> | undefined) {
  if (!images) return
  for (const image of Object.values(images)) {
    const closable = image as { close?: () => void } | null
    closable?.close?.()
  }
}

function follow(dt: number, now: number) {
  const { journey } = props
  state.progress += (journey.progress - state.progress) * (1 - Math.exp(-dt / Math.max(1, journey.lag)))
  if (Math.abs(journey.progress - state.progress) < 0.0002) state.progress = journey.progress
  const ease = 1 - Math.exp(-dt / 420)
  state.pointer.x += (journey.pointerX - state.pointer.x) * ease
  state.pointer.y += (journey.pointerY - state.pointer.y) * ease
  state.tall = journey.tall
  state.marks = journey.marks
  state.clear = journey.clear
  state.time = (now - startedAt) / 1000
  if (state.intro < 1) {
    // The sunrise begins with the words, or as soon after them as the scene is ready.
    if (!introFrom && props.after) introFrom = Math.max(props.after, now)
    if (introFrom) state.intro = clamp((now - introFrom) / OPENING_MS[props.opening])
  }
}

function draw(now: number) {
  scene?.render(state)
  drawnAt = now
}

function tick(now: number) {
  frame = 0
  if (!scene || stopped) return
  const dt = Math.min(120, now - last)
  last = now
  // A zoom of the page, or a move to another screen, changes the screen's pixel density, and not every
  // browser sends an event for it; so it is simply looked at.
  if (window.devicePixelRatio !== density) resize()
  follow(dt, now)
  const { journey } = props
  const untouched = now - Math.max(journey.touched, startedAt)
  const moving = state.progress !== journey.progress || state.intro < 1 || untouched < 900
  const frameInterval = untouched > RESTING_AFTER_MS ? RESTING_FRAME_MS : IDLE_FRAME_MS
  if (moving || now - drawnAt >= frameInterval) draw(now)
  if (moving) {
    watchPace(dt)
    schedule()
    return
  }
  scheduleIdle(Math.max(16, frameInterval - (now - drawnAt)))
}

function scheduleIdle(delayMs: number) {
  if (frame || idleTimer || stopped || !scene || document.hidden) return
  idleTimer = setTimeout(() => {
    idleTimer = undefined
    schedule()
  }, delayMs)
}

function schedule() {
  if (idleTimer) {
    clearTimeout(idleTimer)
    idleTimer = undefined
  }
  if (frame || stopped || !scene || document.hidden) return
  if (!last || performance.now() - last > 250) last = performance.now()
  frame = requestAnimationFrame(tick)
}

/** A machine that cannot hold twenty-five frames a second is given a lighter picture, then the still one. */
function watchPace(dt: number) {
  slowFrames = dt > 40 ? slowFrames + 1 : Math.max(0, slowFrames - 2)
  if (slowFrames < 70) return
  slowFrames = 0
  if (ratio > 1) {
    ratio = 1
    lowered = true
    resize()
  } else {
    fail()
  }
}

function resize() {
  const element = box.value
  if (!element || !scene) return
  density = window.devicePixelRatio
  if (!lowered) ratio = earthPixelRatio(props.surface.quality)
  scene.resize(element.clientWidth, element.clientHeight, ratio)
  draw(performance.now())
}

/** Resolves once the page is in front of the visitor. */
function pageShown() {
  if (!document.hidden) return Promise.resolve()
  return new Promise<void>(resolve => {
    const check = () => {
      if (document.hidden && !stopped) return
      document.removeEventListener('visibilitychange', check)
      resolve()
    }
    document.addEventListener('visibilitychange', check)
  })
}

/** What a frame costs here. A hidden tab holds timers and frames back, so nothing measured there counts. */
async function frameCost(made: EarthScene, sample: JourneyState) {
  for (;;) {
    await pageShown()
    if (stopped) return 0
    const cost = await made.cost(sample)
    if (!document.hidden || stopped) return cost
  }
}

function fail() {
  if (stopped) return
  stop()
  emit('failed')
}

function stop() {
  stopped = true
  abort.abort()
  clearTimeout(posterTimer)
  clearTimeout(posterGone)
  clearTimeout(idleTimer)
  idleTimer = undefined
  cancelAnimationFrame(frame)
  observer?.disconnect()
  document.removeEventListener('visibilitychange', schedule)
  document.removeEventListener('visibilitychange', armPoster)
  props.surface.canvas.removeEventListener('webglcontextlost', fail)
  if (scene) scene.dispose()
  else props.surface.context.getExtension('WEBGL_lose_context')?.loseContext()
  scene = undefined
}

/**
 * The still picture stands in when the live one has not come after a while of the page being looked at.
 * A page opened in a background tab is not held to that: there the scene can only finish starting once
 * the tab is shown, so the wait is counted from then and the visitor still sees the sunrise.
 */
function armPoster() {
  clearTimeout(posterTimer)
  const element = box.value
  if (!element || document.hidden || live.value || stopped || posterShown.value) return
  const show = () => {
    if (live.value || stopped) return
    posterAnchor.value = (props.journey.marks[0] ?? 0.6) * element.clientHeight
    posterShown.value = true
  }
  posterTimer = setTimeout(() => {
    if (!mapsArrived) return show()
    posterTimer = setTimeout(show, POSTER_AFTER_MS - POSTER_EARLY_MS)
  }, POSTER_EARLY_MS)
}

async function start() {
  const { surface } = props
  const element = box.value
  if (!element || !mount.value) return
  mount.value.append(surface.canvas)
  surface.canvas.addEventListener('webglcontextlost', fail)
  armPoster()
  document.addEventListener('visibilitychange', armPoster)
  ratio = earthPixelRatio(surface.quality)
  density = window.devicePixelRatio
  try {
    const [images, module] = await Promise.all([
      loadEarthImages(MAPS[surface.quality], abort.signal, () => { mapsArrived++ }),
      import('./earth/scene')
    ])
    if (stopped) {
      closeEarthImages(images)
      return
    }
    const made = await module.createEarthScene({
      canvas: surface.canvas, context: surface.context, images, quality: surface.quality,
      pixelRatio: ratio, width: element.clientWidth, height: element.clientHeight
    })
    if (stopped) return made.dispose()
    scene = made
    // One measurement before the picture is shown: a heavy frame is drawn less sharply, a hopeless one not at all.
    const sample: JourneyState = { time: 0, intro: 1, progress: 0.3, pointer: { x: 0, y: 0 } }
    let cost = await frameCost(made, sample)
    if (cost > HEAVY_FRAME_MS && ratio > 1) {
      ratio = 1
      lowered = true
      made.resize(element.clientWidth, element.clientHeight, ratio)
      cost = await frameCost(made, sample)
    }
    if (stopped) return
    if (cost > HOPELESS_FRAME_MS) throw new Error('the scene is too slow here')
  } catch {
    return fail()
  }
  clearTimeout(posterTimer)
  // Where the still picture has already taken its place, the live one comes in finished, not from black.
  if (posterShown.value) state.intro = 1
  startedAt = performance.now()
  state.progress = props.journey.progress
  follow(0, startedAt)
  draw(startedAt)
  live.value = true
  // The still picture leaves the page once the live one has faded in over it.
  posterGone = setTimeout(() => { posterKept.value = false }, 1800)
  observer = typeof ResizeObserver === 'function' ? new ResizeObserver(resize) : undefined
  observer?.observe(element)
  document.addEventListener('visibilitychange', schedule)
  emit('ready')
  schedule()
}

/** The page calls this when something changed that the loop may have been asleep for. */
defineExpose({ wake: schedule })

onMounted(start)
onBeforeUnmount(stop)
</script>

<style scoped>
/* As tall as the window at its largest: a phone's toolbar comes and goes while scrolling, and the picture
   must not change size with it. */
.earth-stage { position: fixed; top: 0; left: 0; width: 100%; height: 100vh; z-index: 0; overflow: hidden; pointer-events: none; }
@supports (height: 100lvh) {
  .earth-stage { height: 100lvh; }
}
.earth-stage-canvas { position: absolute; inset: 0; opacity: 0; transition: opacity 1.1s cubic-bezier(.16, 1, .3, 1); }
.earth-stage.live .earth-stage-canvas { opacity: 1; }
.earth-stage-canvas :deep(canvas) { display: block; width: 100%; height: 100%; }
.earth-stage-poster { position: absolute; inset: 0; opacity: 0; transition: opacity .9s cubic-bezier(.16, 1, .3, 1); }
.earth-stage-poster.on { opacity: 1; }
.earth-stage.live .earth-stage-poster { opacity: 0; transition: opacity 1.1s ease .3s; }
@media (prefers-reduced-motion: reduce) {
  .earth-stage-canvas, .earth-stage-poster { transition: none; }
}
</style>
