// What the page needs for the live Earth before Three.js has been downloaded: whether it can be drawn at
// all, the canvas it will be drawn on, and its maps. Nothing here imports Three.js.

/** 'high': 4K maps and an HDR pipeline with bloom. 'low': 2K maps drawn straight to the screen. */
export type EarthQuality = 'high' | 'low'

/**
 * The journey has one stop for each screen of the page, evenly spaced (the keys in story.ts are composed
 * for them): screen `index` of `count` stands here, between 0 and 1.
 */
export const earthBeat = (index: number, count: number) => count > 1 ? index / (count - 1) : 0

/** Equirectangular maps, 2:1, longitude -180° at the left edge, first row at the north edge. */
export interface EarthMaps<T> {
  /** Colour of the ground by day (sRGB). */
  day: T
  /** City lights; one channel. */
  lights: T
  /** Cloud cover; one channel. */
  clouds: T
  /** White where there is water; one channel. */
  water: T
}
export type EarthImage = ImageBitmap | HTMLImageElement

/**
 * What the page tells the stage, written by the page and read by the stage on every frame. A plain object
 * on purpose: nothing here should make Vue render.
 */
export interface EarthJourney {
  /** 0..1, where scrolling has brought the journey; the stage follows it smoothly. */
  progress: number
  /** The words stand above the planet (true) or in a column to its left (false). */
  tall: boolean
  /**
   * What the page measured on each of its screens, in order: where the globe stands beside the words, the
   * share of the picture's width they take; where it stands above them, the share of its height at which
   * they begin; along a horizon, the share of its height at which they end. Replaced, never changed in place.
   */
  marks: readonly number[]
  /** The share of the picture's height that the page's header takes at the top. */
  clear: number
  /** Milliseconds by which the picture trails the page: little where the page itself moves smoothly. */
  lag: number
  /** The pointer, each -1..1. */
  pointerX: number
  pointerY: number
  /** performance.now() of the visitor's last scroll or pointer movement. */
  touched: number
}

export function createEarthJourney(): EarthJourney {
  return { progress: 0, tall: false, marks: [], clear: 0, lag: 170, pointerX: 0, pointerY: 0, touched: 0 }
}

/** A canvas that can draw the scene, with the context the scene will use. */
export interface EarthSurface {
  canvas: HTMLCanvasElement
  context: WebGL2RenderingContext
  quality: EarthQuality
}

/** Phones and small or weak machines take the light version. */
function pickQuality(): EarthQuality {
  const small = window.matchMedia('(max-width: 820px), (pointer: coarse) and (max-height: 820px)').matches
  const memory = (navigator as Navigator & { deviceMemory?: number }).deviceMemory ?? 8
  const weak = (navigator.hardwareConcurrency ?? 8) <= 4 || memory <= 4
  return small || weak ? 'low' : 'high'
}

/** The drawing context of each version. The scene creates its renderer with the same attributes. */
export function earthContextAttributes(quality: EarthQuality): WebGLContextAttributes {
  const high = quality === 'high'
  return {
    alpha: false,
    antialias: !high,
    depth: !high,
    stencil: false,
    powerPreference: 'high-performance',
    // A browser that could only draw in software gets no context, and the page shows its still image.
    failIfMajorPerformanceCaveat: true
  }
}

function surfaceOf(quality: EarthQuality): EarthSurface | null {
  const canvas = document.createElement('canvas')
  let context: WebGL2RenderingContext | null = null
  try {
    context = canvas.getContext('webgl2', earthContextAttributes(quality))
  } catch {
    context = null
  }
  return context ? { canvas, context, quality } : null
}

/**
 * Finds out, when the page is set up, whether the scene can be drawn here. The answer decides the page's
 * layout, so it must be known before the first paint; the scene itself arrives later and draws on this
 * canvas. Returns null where there is no WebGL 2 or only a software renderer. `only` asks for the light
 * version whatever the machine, for one that proved too slow for the full one.
 */
export function createEarthSurface(only?: EarthQuality): EarthSurface | null {
  if (typeof WebGL2RenderingContext === 'undefined') return null
  const quality = only ?? pickQuality()
  const surface = surfaceOf(quality)
  if (!surface || quality === 'low') return surface
  // The full version draws into a half-float target and reads 4K maps.
  const { context } = surface
  const capable = context.getExtension('EXT_color_buffer_float') !== null &&
    context.getParameter(context.MAX_TEXTURE_SIZE) >= 4096
  if (capable) return surface
  context.getExtension('WEBGL_lose_context')?.loseContext()
  return surfaceOf('low')
}

/** Sharper than this the picture is not drawn: beyond it the cost grows and the eye gains nothing. */
export function earthPixelRatio(quality: EarthQuality) {
  return Math.min(window.devicePixelRatio || 1, quality === 'high' ? 1.75 : 2)
}

function imageElement(url: string) {
  return new Promise<HTMLImageElement>((resolve, reject) => {
    const image = new Image()
    image.decoding = 'async'
    image.onload = () => resolve(image)
    image.onerror = () => reject(new Error(`map ${url} did not load`))
    image.src = url
  })
}

async function loadImage(url: string, signal: AbortSignal): Promise<EarthImage> {
  try {
    const response = await fetch(url, { signal })
    if (!response.ok) throw new Error(`map ${url}: HTTP ${response.status}`)
    // Decoded off the main thread, and left as stored: the shader turns the picture over.
    return await createImageBitmap(await response.blob(), { premultiplyAlpha: 'none', colorSpaceConversion: 'none' })
  } catch (error) {
    if (signal.aborted) throw error
    // A browser that cannot decode this way takes the same picture through an image element.
    return imageElement(url)
  }
}

/**
 * Downloads and decodes the maps. The page starts this at once, while Three.js is still on its way;
 * `arrived` is called as each map is ready.
 */
export async function loadEarthImages(urls: EarthMaps<string>, signal: AbortSignal, arrived?: () => void): Promise<EarthMaps<EarthImage>> {
  const one = async (url: string) => {
    const image = await loadImage(url, signal)
    arrived?.()
    return image
  }
  const [day, lights, clouds, water] = await Promise.all([one(urls.day), one(urls.lights), one(urls.clouds), one(urls.water)])
  return { day, lights, clouds, water }
}
