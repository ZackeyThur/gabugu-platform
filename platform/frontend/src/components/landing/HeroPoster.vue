<template>
  <div
    ref="box"
    class="hero-poster"
    :class="{ shown }"
    aria-hidden="true"
  >
    <img
      :src="picture.src"
      :srcset="picture.srcset"
      :sizes="picture.sizes"
      :style="placement"
      alt=""
      decoding="async"
      fetchpriority="high"
      @load="shown = true"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import wide2560 from '@/assets/landing/hero-poster-2560.webp'
import wide1280 from '@/assets/landing/hero-poster-1280.webp'
import tall1440 from '@/assets/landing/hero-poster-portrait-1440.webp'
import tall720 from '@/assets/landing/hero-poster-portrait-720.webp'

/**
 * The still picture of the first screen: a sunrise over the planet's edge. It stands in for the live Earth
 * where that cannot be drawn, and while it is still on its way over a slow connection.
 * Measured on the pictures: `aspect` is width / height; `apex` is the highest point of the planet's edge
 * and `sun` the sun's centre, both as shares of the width and of the height.
 */
const WIDE = { aspect: 16 / 9, apex: 0.6182, sun: 0.6694, srcset: `${wide1280} 1280w, ${wide2560} 2560w`, src: wide1280 }
const TALL = { aspect: 9 / 16, apex: 0.5829, sun: 0.692, srcset: `${tall720} 720w, ${tall1440} 1440w`, src: tall720 }

const props = defineProps<{
  /** Pixels from the top of this box to where the planet's edge should stand; unset = where the picture has it. */
  anchor?: number
  /** Only the small version of the picture: for a stand-in that is fetched ahead of need. */
  small?: boolean
}>()

const box = ref<HTMLElement | null>(null)
// Starts from the window's shape, which the box nearly always has, so that the right one of the two
// pictures is asked for from the first render on.
const size = ref({ width: window.innerWidth, height: window.innerHeight })
const shown = ref(false)
let observer: ResizeObserver | undefined

const source = computed(() => (size.value.width && size.value.width / size.value.height < 0.8 ? TALL : WIDE))
/** The picture is as wide as the box at least, and reaches from the planet's edge down to the box's bottom. */
const frame = computed(() => {
  const { width, height } = size.value
  const { aspect, apex, sun } = source.value
  if (!width || !height) return null
  const anchor = Math.min(height * 0.8, Math.max(height * 0.3, props.anchor || height * apex))
  const tall = Math.max(width / aspect, (height - anchor) / (1 - apex))
  const wide = tall * aspect
  // Sideways the sun keeps its place in the picture; the planet's edge stands on the anchor.
  return { width: wide, height: tall, left: sun * (width - wide), top: anchor - apex * tall }
})
const placement = computed(() => {
  const value = frame.value
  if (!value) return { width: '100%', height: '100%', objectFit: 'cover' as const }
  return { width: `${value.width}px`, height: `${value.height}px`, left: `${value.left}px`, top: `${value.top}px` }
})
const picture = computed(() => props.small
  ? { src: source.value.src, srcset: undefined, sizes: undefined }
  : {
    src: source.value.src,
    srcset: source.value.srcset,
    // The picture is drawn wider than the screen where the screen is tall, and the browser should know.
    sizes: frame.value ? `${Math.ceil(frame.value.width)}px` : '100vw'
  })

function measure() {
  const element = box.value
  if (element) size.value = { width: element.clientWidth, height: element.clientHeight }
}

onMounted(() => {
  measure()
  if (typeof ResizeObserver !== 'function' || !box.value) return
  observer = new ResizeObserver(measure)
  observer.observe(box.value)
})
onBeforeUnmount(() => observer?.disconnect())
</script>

<style scoped>
/* Space in the picture is black and the page is not quite. The box carries the page colour and the
   picture is laid over it with 'lighten', so the page colour stands wherever the picture is darker and the
   picture has no visible edge. The box is isolated so that the blend is with its own ground and with
   nothing else. */
.hero-poster { position: absolute; inset: 0; overflow: hidden; isolation: isolate; background: #08080d; opacity: 0; transition: opacity .9s cubic-bezier(.16, 1, .3, 1); }
.hero-poster.shown { opacity: 1; }
.hero-poster img { position: absolute; max-width: none; mix-blend-mode: lighten; }
/* The lower edge runs into the page colour. */
.hero-poster::after { content: ''; position: absolute; inset: auto 0 0; height: 22%; background: linear-gradient(0deg, #08080d 0, rgba(8, 8, 13, 0) 100%); }
@media (prefers-reduced-motion: reduce) {
  .hero-poster { transition: none; }
}
</style>
