// The live Earth of the Landing, in the manner of a film shot.
// The globe and its air are cheap analytic shading; the budget goes to an HDR pipeline: a half-float
// target, bloom, filmic tone mapping, a little vignette and grain. The light version draws straight to
// the canvas with the same grade written into each material.
// This module is the only one that imports Three.js, and the page loads it on demand.
import * as THREE from 'three'
import { EffectComposer } from 'three/addons/postprocessing/EffectComposer.js'
import { RenderPass } from 'three/addons/postprocessing/RenderPass.js'
import { UnrealBloomPass } from 'three/addons/postprocessing/UnrealBloomPass.js'
import { ShaderPass } from 'three/addons/postprocessing/ShaderPass.js'
import type { EarthImage, EarthMaps, EarthQuality } from './context'
import { evaluate, type JourneyState, type JourneyView } from './story'
import { createNoiseTexture } from './noise'
import { earthVertex, earthFragment } from './shadersEarth'
import { haloVertex, haloFragment, sunVertex, sunFragment, starVertex, starFragment } from './shadersSky'
import { FinalShader, withGrade } from './post'

const BACKGROUND = 0x08080d
/**
 * The picture is never larger than this many pixels: the full version, where 1440×900 at a ratio of 1.75
 * fits, and the light one, which is for machines with less to spare.
 */
const MAX_PIXELS = 4.2e6
const MAX_PIXELS_LIGHT = 2.6e6

/** The look, in one place. */
const TUNE = {
  exposure: 1.0,
  sunPower: 1.0,
  city: 1.0,
  twilight: 1.0,
  stars: 1.1,
  halo: 1.0,
  haloCore: 0.0018,
  haloWide: 0.011,
  bloomStrength: 0.22,
  bloomRadius: 0.3,
  bloomThreshold: 0.95,
  bloomKnee: 0.7,
  vignette: 0.3,
  grain: 0.022
}

export interface EarthSceneOptions {
  canvas: HTMLCanvasElement
  /** The context the page created for this canvas (createEarthSurface in context.ts). */
  context: WebGL2RenderingContext
  /** The maps, already downloaded and decoded (loadEarthImages). */
  images: EarthMaps<EarthImage>
  quality: EarthQuality
  pixelRatio: number
  width: number
  height: number
}

export interface EarthScene {
  /** Draws one frame. The same state always gives the same picture. */
  render(state: JourneyState): void
  /**
   * Milliseconds the graphics card needs for one frame of this state. Draws a few frames and waits for
   * each without blocking the page, so it is for one measurement before the scene is shown.
   */
  cost(state: JourneyState): Promise<number>
  resize(width: number, height: number, pixelRatio: number): void
  dispose(): void
}

const smooth = (a: number, b: number, v: number) => { const t = Math.min(1, Math.max(0, (v - a) / (b - a))); return t * t * (3 - 2 * t) }
const mix = (a: number, b: number, t: number) => a + (b - a) * t

interface TextureOptions { srgb?: boolean, single?: boolean, repeat?: boolean, anisotropy?: number }

function makeTexture(image: EarthImage, { srgb = false, single = false, repeat = false, anisotropy = 1 }: TextureOptions = {}) {
  const texture = new THREE.Texture(image)
  // The picture goes to the GPU as it is stored and the shader turns it over, so nothing depends on
  // whether a browser can flip a picture while decoding it.
  texture.flipY = false
  texture.colorSpace = srgb ? THREE.SRGBColorSpace : THREE.NoColorSpace
  if (single) texture.format = THREE.RedFormat
  texture.wrapS = repeat ? THREE.RepeatWrapping : THREE.ClampToEdgeWrapping
  texture.wrapT = THREE.ClampToEdgeWrapping
  texture.minFilter = THREE.LinearMipmapLinearFilter
  texture.magFilter = THREE.LinearFilter
  texture.generateMipmaps = true
  texture.anisotropy = anisotropy
  texture.needsUpdate = true
  return texture
}

/** Stars at fixed places: the same sky on every visit. */
function createStars(count: number) {
  let seed = 977
  const random = () => { seed = (seed * 1664525 + 1013904223) >>> 0; return seed / 4294967296 }
  const position = new Float32Array(count * 3)
  const size = new Float32Array(count)
  const gain = new Float32Array(count)
  const phase = new Float32Array(count)
  for (let i = 0; i < count; i++) {
    const y = random() * 2 - 1
    const a = random() * Math.PI * 2
    const r = Math.sqrt(1 - y * y)
    position.set([r * Math.cos(a) * 60, y * 60, r * Math.sin(a) * 60], i * 3)
    const bright = Math.pow(random(), 7)
    gain[i] = 0.05 + bright * 1.5
    size[i] = 1.25 + bright * 1.5
    phase[i] = random()
  }
  const geometry = new THREE.BufferGeometry()
  geometry.setAttribute('position', new THREE.BufferAttribute(position, 3))
  geometry.setAttribute('aSize', new THREE.BufferAttribute(size, 1))
  geometry.setAttribute('aGain', new THREE.BufferAttribute(gain, 1))
  geometry.setAttribute('aPhase', new THREE.BufferAttribute(phase, 1))
  return geometry
}

const nextFrame = () => new Promise<void>(resolve => requestAnimationFrame(() => resolve()))

export async function createEarthScene({ canvas, context, images, quality, pixelRatio, width, height }: EarthSceneOptions): Promise<EarthScene> {
  const high = quality === 'high'
  // The renderer is told what the context really has, which is what the page asked for where the browser
  // could give it. Nothing is imported from the page's own code here: this chunk depends on Three.js alone.
  const renderer = new THREE.WebGLRenderer({ canvas, context, ...context.getContextAttributes() })
  renderer.setClearColor(high ? 0x000000 : BACKGROUND, 1)
  renderer.toneMapping = THREE.NoToneMapping
  renderer.outputColorSpace = THREE.SRGBColorSpace
  const anisotropy = Math.min(high ? 8 : 4, renderer.capabilities.getMaxAnisotropy())

  const day = makeTexture(images.day, { srgb: true, anisotropy })
  const lights = makeTexture(images.lights, { single: true, anisotropy })
  const clouds = makeTexture(images.clouds, { single: true, repeat: true, anisotropy })
  const water = makeTexture(images.water, { single: true, anisotropy })
  const noise = createNoiseTexture(256)
  const owned = [day, lights, clouds, water, noise]

  // Uniforms shared by every material.
  const shared = {
    uSun: { value: new THREE.Vector3(1, 0, 0) },
    uSunDisc: { value: new THREE.Vector3(1, 0, 0) },
    uExposure: { value: 1 },
    uFade: { value: 1 }
  }
  const defines: Record<string, number> = high ? { HIGH: 1 } : { DIRECT: 1 }
  // The light version has no off-screen pass, so each material grades its own output.
  const direct = (fragment: string, additive: boolean) => high ? fragment : withGrade(fragment, additive)

  const scene = new THREE.Scene()
  const camera = new THREE.PerspectiveCamera(36, width / height, 0.01, 200)

  const earthMaterial = new THREE.ShaderMaterial({
    defines,
    uniforms: {
      ...shared,
      tDay: { value: day }, tLights: { value: lights }, tClouds: { value: clouds }, tWater: { value: water }, tNoise: { value: noise },
      uCloudShift: { value: 0 }, uSunPower: { value: TUNE.sunPower }, uCity: { value: TUNE.city }, uTwilight: { value: TUNE.twilight }, uLow: { value: 1 }
    },
    vertexShader: earthVertex,
    fragmentShader: direct(earthFragment, false)
  })
  const earth = new THREE.Mesh(new THREE.SphereGeometry(1, high ? 256 : 160, high ? 128 : 80), earthMaterial)

  const additive = { blending: THREE.AdditiveBlending, transparent: true, depthWrite: false, depthTest: true }
  const haloMaterial = new THREE.ShaderMaterial({
    defines, ...additive, side: THREE.BackSide,
    uniforms: { ...shared, uCore: { value: 0.004 }, uWide: { value: 0.03 }, uGain: { value: 1 }, uWarm: { value: 1 } },
    vertexShader: haloVertex,
    fragmentShader: direct(haloFragment, true)
  })
  const halo = new THREE.Mesh(new THREE.SphereGeometry(1.16, 96, 48), haloMaterial)

  const sunMaterial = new THREE.ShaderMaterial({
    defines, ...additive,
    uniforms: { ...shared, uSize: { value: 0.36 }, uCoreColor: { value: new THREE.Color() }, uGlowColor: { value: new THREE.Color() }, uPower: { value: 1 }, uGlow: { value: 1 } },
    vertexShader: sunVertex,
    fragmentShader: direct(sunFragment, true)
  })
  const sun = new THREE.Mesh(new THREE.PlaneGeometry(2, 2), sunMaterial)

  const starMaterial = new THREE.ShaderMaterial({
    defines, ...additive,
    uniforms: { ...shared, uTime: { value: 0 }, uPixel: { value: 1 }, uGain: { value: 1 }, uSunShow: { value: 1 } },
    vertexShader: starVertex,
    fragmentShader: direct(starFragment, true)
  })
  const stars = new THREE.Points(createStars(high ? 1500 : 900), starMaterial)

  const objects = [earth, stars, halo, sun]
  objects.forEach((object, order) => {
    object.renderOrder = order
    object.frustumCulled = false
    object.matrixAutoUpdate = false
    scene.add(object)
  })

  // The HDR pipeline, in the full version only. Edges are smoothed by multisampling, lightly where the
  // screen is dense enough to do most of that itself.
  let composer: EffectComposer | undefined
  let bloom: UnrealBloomPass | undefined
  let finalPass: ShaderPass | undefined
  let hdrTarget: THREE.WebGLRenderTarget | undefined
  if (high) {
    hdrTarget = new THREE.WebGLRenderTarget(width, height, { type: THREE.HalfFloatType, samples: pixelRatio >= 1.5 ? 2 : 4, depthBuffer: true, stencilBuffer: false })
    composer = new EffectComposer(renderer, hdrTarget)
    composer.addPass(new RenderPass(scene, camera))
    bloom = new UnrealBloomPass(new THREE.Vector2(width, height), TUNE.bloomStrength, TUNE.bloomRadius, TUNE.bloomThreshold)
    ;(bloom.highPassUniforms as Record<string, THREE.IUniform>).smoothWidth.value = TUNE.bloomKnee
    composer.addPass(bloom)
    finalPass = new ShaderPass(FinalShader)
    finalPass.uniforms.uVignette.value = TUNE.vignette
    finalPass.uniforms.uGrain.value = TUNE.grain
    composer.addPass(finalPass)
  }

  let cssWidth = width
  let cssHeight = height
  function resize(w: number, h: number, ratio: number) {
    cssWidth = Math.max(1, w)
    cssHeight = Math.max(1, h)
    const scale = Math.min(ratio, Math.sqrt((high ? MAX_PIXELS : MAX_PIXELS_LIGHT) / (cssWidth * cssHeight)))
    renderer.setPixelRatio(scale)
    renderer.setSize(cssWidth, cssHeight, false)
    camera.aspect = cssWidth / cssHeight
    starMaterial.uniforms.uPixel.value = Math.max(1, scale) * (cssHeight < 700 ? 0.9 : 1)
    if (composer && bloom && finalPass) {
      composer.setPixelRatio(scale)
      composer.setSize(cssWidth, cssHeight)
      // Bloom works at the size of the page, not of the screen's pixels: the same look at every pixel
      // ratio, and a fixed cost.
      const bloomScale = Math.min(1, 1500 / Math.max(cssWidth, cssHeight))
      bloom.setSize(Math.round(cssWidth * bloomScale), Math.round(cssHeight * bloomScale))
      finalPass.uniforms.uAspect.value = camera.aspect
    }
  }
  resize(width, height, pixelRatio)

  const view = {} as JourneyView
  const target = new THREE.Vector3()
  const warm = new THREE.Color(1.0, 0.5, 0.16)
  const white = new THREE.Color(1.0, 0.93, 0.82)
  const glowWarm = new THREE.Color(1.0, 0.4, 0.08)
  const glowWhite = new THREE.Color(1.0, 0.86, 0.7)

  function render(state: JourneyState) {
    evaluate(state, camera.aspect, view)
    camera.position.fromArray(view.position)
    camera.up.fromArray(view.up)
    camera.lookAt(target.set(view.position[0] + view.direction[0], view.position[1] + view.direction[1], view.position[2] + view.direction[2]))
    camera.fov = view.fov
    camera.updateProjectionMatrix()
    // A lens shift: the picture moves on the screen without the globe changing shape.
    camera.projectionMatrix.elements[8] = -view.shiftX
    camera.projectionMatrix.elements[9] = -view.shiftY
    camera.projectionMatrixInverse.copy(camera.projectionMatrix).invert()

    const time = state.time || 0
    shared.uSun.value.fromArray(view.sunLight)
    shared.uSunDisc.value.fromArray(view.sun)
    shared.uExposure.value = view.exposure * TUNE.exposure
    shared.uFade.value = view.fade
    earthMaterial.uniforms.uCloudShift.value = time * 0.00006
    earthMaterial.uniforms.uLow.value = view.low
    haloMaterial.uniforms.uGain.value = TUNE.halo * mix(0.8, 1, view.low)

    // The line of the atmosphere keeps a readable thickness when the camera pulls back.
    const toLimb = Math.sqrt(Math.max(view.distance * view.distance - 1, 0.01))
    haloMaterial.uniforms.uCore.value = TUNE.haloCore * Math.pow(toLimb / 0.66, 0.55)
    haloMaterial.uniforms.uWide.value = TUNE.haloWide * Math.pow(toLimb / 0.66, 0.35)

    // The sun reddens and dims as it sinks to the edge of the planet, and is gone a little below it.
    const altitude = view.sunAltitude
    haloMaterial.uniforms.uWarm.value = smooth(-7, -0.5, altitude)
    const rise = smooth(0.5, 9, altitude)
    sunMaterial.uniforms.uCoreColor.value.copy(warm).lerp(white, rise)
    sunMaterial.uniforms.uGlowColor.value.copy(glowWarm).lerp(glowWhite, smooth(2, 30, altitude))
    sunMaterial.uniforms.uPower.value = smooth(-2.2, 0.6, altitude) * mix(0.75, 1, rise)
    sunMaterial.uniforms.uGlow.value = high ? 1 : 2

    starMaterial.uniforms.uTime.value = time
    starMaterial.uniforms.uSunShow.value = smooth(-2, 1, altitude)
    starMaterial.uniforms.uGain.value = TUNE.stars / Math.max(0.6, view.exposure)

    if (composer && finalPass) {
      finalPass.uniforms.uExposure.value = shared.uExposure.value
      finalPass.uniforms.uFade.value = view.fade
      finalPass.uniforms.uTime.value = time
      composer.render(0)
    } else {
      renderer.render(scene, camera)
    }
  }

  function drawn(state: JourneyState) {
    return new Promise<number>(resolve => {
      const gl = renderer.getContext() as WebGL2RenderingContext
      const start = performance.now()
      render(state)
      const fence = gl.fenceSync(gl.SYNC_GPU_COMMANDS_COMPLETE, 0)
      gl.flush()
      const poll = () => {
        // A fence is signalled only after the browser has handed the frame over, so ask from a later task.
        const pending = fence !== null && !disposed && !gl.isContextLost() &&
          gl.clientWaitSync(fence, 0, 0) === gl.TIMEOUT_EXPIRED && performance.now() - start < 400
        if (pending) return void setTimeout(poll, 2)
        if (fence) gl.deleteSync(fence)
        resolve(performance.now() - start)
      }
      setTimeout(poll, 0)
    })
  }
  async function cost(state: JourneyState) {
    // The first frame after a change of size still sets things up; the better of the next two counts.
    await drawn(state)
    return Math.min(await drawn(state), await drawn(state))
  }

  let disposed = false
  function dispose() {
    if (disposed) return
    disposed = true
    for (const texture of owned) {
      const image = texture.image as { close?: () => void } | null
      image?.close?.()
      texture.dispose()
    }
    for (const object of objects) {
      object.geometry.dispose()
      object.material.dispose()
    }
    bloom?.dispose()
    finalPass?.dispose()
    composer?.dispose()
    hdrTarget?.dispose()
    renderer.dispose()
    // Give the graphics context back now rather than when the canvas is collected. A context that is
    // already lost has nothing left to give back.
    if (!renderer.getContext().isContextLost()) renderer.forceContextLoss()
  }

  // Everything slow happens here, before the page is told the scene is ready: programs are compiled
  // without blocking, maps are uploaded one per frame, and one black frame goes through the whole
  // pipeline. The first frame the visitor sees then costs no more than any other.
  try {
    // Compiled in the background where the browser can report when that is done (not in Firefox).
    if (renderer.extensions.has('KHR_parallel_shader_compile')) await renderer.compileAsync(scene, camera)
    else renderer.compile(scene, camera)
    for (const texture of owned) {
      renderer.initTexture(texture)
      // Once a map is on the graphics card its decoded picture has done its work; at 4K that is 32 MB each.
      ;(texture.image as { close?: () => void } | null)?.close?.()
      await nextFrame()
    }
    render({ time: 0, intro: 0, progress: 0, pointer: { x: 0, y: 0 } })
  } catch (error) {
    dispose()
    throw error
  }

  return { render, cost, resize, dispose }
}
