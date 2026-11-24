// A small tileable noise texture, generated once on the CPU from a fixed seed (same pixels on every machine).
//   R  fractal gradient noise (wisps, wind patches)
//   G  fractal cellular noise, bright at cell centres (cloud billows)
//   B  sparse bright speckle (city light grain, sea sparkle)
//   A  a second, finer fractal gradient noise
import * as THREE from 'three'

function mulberry32(seed: number) {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = a
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

// Perlin gradient noise on a wrapping lattice of `period` cells.
function addGradient(target: Float32Array, size: number, period: number, gain: number, random: () => number) {
  const gx = new Float32Array(period * period)
  const gy = new Float32Array(period * period)
  for (let i = 0; i < gx.length; i++) {
    const a = random() * Math.PI * 2
    gx[i] = Math.cos(a)
    gy[i] = Math.sin(a)
  }
  const scale = period / size
  const fade = (t: number) => t * t * t * (t * (t * 6 - 15) + 10)
  for (let y = 0; y < size; y++) {
    const fy = y * scale
    const y0 = Math.floor(fy)
    const ty = fy - y0
    const ya = (y0 % period) * period
    const yb = ((y0 + 1) % period) * period
    const sy = fade(ty)
    for (let x = 0; x < size; x++) {
      const fx = x * scale
      const x0 = Math.floor(fx)
      const tx = fx - x0
      const xa = x0 % period
      const xb = (x0 + 1) % period
      const sx = fade(tx)
      const a = gx[ya + xa] * tx + gy[ya + xa] * ty
      const b = gx[ya + xb] * (tx - 1) + gy[ya + xb] * ty
      const c = gx[yb + xa] * tx + gy[yb + xa] * (ty - 1)
      const d = gx[yb + xb] * (tx - 1) + gy[yb + xb] * (ty - 1)
      target[y * size + x] += (a + (b - a) * sx + ((c + (d - c) * sx) - (a + (b - a) * sx)) * sy) * gain
    }
  }
}

// Cellular noise: 1 at a feature point, falling to 0 at the cell borders. `power` shapes the falloff.
function addCells(target: Float32Array, size: number, period: number, gain: number, random: () => number, power = 1, weights: Float32Array | null = null) {
  const px = new Float32Array(period * period)
  const py = new Float32Array(period * period)
  for (let i = 0; i < px.length; i++) { px[i] = random(); py[i] = random() }
  const scale = period / size
  for (let y = 0; y < size; y++) {
    const fy = y * scale
    const cy = Math.floor(fy)
    for (let x = 0; x < size; x++) {
      const fx = x * scale
      const cx = Math.floor(fx)
      let best = 9
      let bestCell = 0
      for (let j = -1; j <= 1; j++) {
        for (let i = -1; i <= 1; i++) {
          const cell = ((cy + j + period) % period) * period + ((cx + i + period) % period)
          const dx = cx + i + px[cell] - fx
          const dy = cy + j + py[cell] - fy
          const dist = dx * dx + dy * dy
          if (dist < best) { best = dist; bestCell = cell }
        }
      }
      const value = Math.pow(Math.max(0, 1 - Math.sqrt(best)), power)
      target[y * size + x] += value * gain * (weights ? weights[bestCell % weights.length] : 1)
    }
  }
}

function normalise(field: Float32Array) {
  let low = Infinity
  let high = -Infinity
  for (let i = 0; i < field.length; i++) { low = Math.min(low, field[i]); high = Math.max(high, field[i]) }
  for (let i = 0; i < field.length; i++) field[i] = (field[i] - low) / (high - low)
  return field
}

export function createNoiseTexture(size = 256) {
  const random = mulberry32(20261111)
  const r = new Float32Array(size * size)
  const g = new Float32Array(size * size)
  const b = new Float32Array(size * size)
  const a = new Float32Array(size * size)
  let gain = 1
  for (const period of [4, 8, 16, 32, 64]) { addGradient(r, size, period, gain, random); gain *= 0.55 }
  gain = 1
  for (const period of [5, 10, 20, 40, 80]) { addCells(g, size, period, gain, random, 1.2); gain *= 0.5 }
  gain = 1
  for (const period of [8, 16, 32, 64]) { addGradient(a, size, period, gain, random); gain *= 0.6 }
  const weights = new Float32Array(977)
  for (let i = 0; i < weights.length; i++) weights[i] = Math.pow(random(), 3)
  addCells(b, size, 48, 1, random, 5, weights)
  addCells(b, size, 90, 0.5, random, 4, weights)
  normalise(r); normalise(g); normalise(a); normalise(b)

  const data = new Uint8Array(size * size * 4)
  for (let i = 0; i < size * size; i++) {
    data[i * 4] = Math.round(r[i] * 255)
    data[i * 4 + 1] = Math.round(g[i] * 255)
    data[i * 4 + 2] = Math.round(b[i] * 255)
    data[i * 4 + 3] = Math.round(a[i] * 255)
  }
  const texture = new THREE.DataTexture(data, size, size, THREE.RGBAFormat, THREE.UnsignedByteType)
  texture.wrapS = texture.wrapT = THREE.RepeatWrapping
  texture.minFilter = THREE.LinearMipmapLinearFilter
  texture.magFilter = THREE.LinearFilter
  texture.generateMipmaps = true
  texture.colorSpace = THREE.NoColorSpace
  texture.needsUpdate = true
  return texture
}
