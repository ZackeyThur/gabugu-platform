// The journey the page scrolls through: scroll progress -> camera, sun and exposure. Pure functions of the
// state: no clocks, no history, so the page can be scrolled back and forth and a frame can be reproduced.
// Everything is expressed in the Earth-fixed frame (the globe never rotates; the camera and the sun move):
//   lat, lon      where the camera hovers (degrees; lon keeps growing eastward past 360)
//   d             distance from the Earth's centre in Earth radii
//   head          compass bearing of the gaze (clockwise from north); with aim 0 it is the direction of "screen up"
//   aim, lift     view direction measured from straight down: aim * (angle to the limb) + lift degrees
//   sx, sy        lens shift (moves the picture without distorting the globe)
//   zen, saz      sun: angle from the camera's zenith, and compass bearing
//   rise          degrees by which the light on the ground runs ahead of the visible sun disc (longer golden hour)
//   exp           exposure multiplier
//   roll          degrees the camera is tipped to its right: the horizon rises towards the right of the picture
//   m             overrides for a page whose words stand above the planet (a phone held upright)
//   ground        a view along the horizon: the planet's edge keeps below the words
//   shown, size   of a fitted globe on a wide page: the share of its width that stays in the picture,
//                 and its largest radius in half-heights of the picture
const DEG = Math.PI / 180
export const limbAngle = (d: number) => Math.asin(1 / d) / DEG
// Sun zenith angle that puts the sun `alt` degrees above the limb as seen from distance d.
const atLimb = (d: number, alt: number) => 180 - (limbAngle(d) + alt)

const FIELDS = ['lat', 'lon', 'd', 'head', 'aim', 'lift', 'fov', 'sx', 'sy', 'zen', 'saz', 'rise', 'exp', 'roll'] as const
type Field = typeof FIELDS[number]
type Key = Record<Exclude<Field, 'roll'>, number> & { roll?: number, ground?: boolean, shown?: number, size?: number, m?: Partial<Record<Field, number>> }

// One key for each screen of the page, in order: screen i of n stands at p = i / (n - 1). The camera keeps
// to one side of the globe and the sun goes round it, from one sunrise to the next: a move across several
// screens changes the light and the distance, and does not spin the globe.
export const JOURNEY_STOPS = 8
const KEYS: Key[] = [
  // 1 first screen: sunrise from low orbit, the horizon tipped across the picture
  { ground: true, lat: 28, lon: 109, d: 1.36, head: 72, aim: 1, lift: 4.6, fov: 36, sx: 0, sy: 0, zen: atLimb(1.36, 0.75), saz: 80.5, rise: 11, exp: 1.0, roll: 17,
    m: { fov: 58, lift: 5.5, head: 77, d: 1.3, zen: atLimb(1.3, 0.75), roll: 5 } },
  // 2 models: the whole globe by day
  { shown: 0.94, size: 0.84, lat: 14, lon: 132, d: 2.62, head: 16, aim: 0, lift: 0, fov: 36, sx: 0.5, sy: -0.08, zen: 50, saz: 118, rise: 0, exp: 1.1,
    m: { fov: 50, d: 2.7 } },
  // 3 integration: close by day
  { shown: 0.5, lat: 4, lon: 118, d: 2.0, head: 24, aim: 0, lift: 0, fov: 36, sx: 0.6, sy: -0.3, zen: 42, saz: 135, rise: 0, exp: 1.08,
    m: { fov: 50, d: 2.2 } },
  // 4 tools: the line between day and night
  { shown: 0.72, lat: 24, lon: 100, d: 2.7, head: 14, aim: 0, lift: 0, fov: 36, sx: 0.66, sy: -0.46, zen: 84, saz: 215, rise: 0, exp: 1.12,
    m: { fov: 50 } },
  // 5 pricing: dusk along the horizon, seen towards the west (a bearing west of north is written as a
  // negative one, so that the camera turns the short way from the screens before and after)
  { ground: true, lat: 24, lon: 112, d: 1.62, head: -78, aim: 1, lift: 5, fov: 36, sx: 0, sy: 0, zen: atLimb(1.62, 0.5), saz: 300, rise: 5, exp: 1.06, roll: -9,
    m: { fov: 58, lift: 7, roll: -3 } },
  // 6 control: the night side and its city lights
  { shown: 0.8, lat: 30, lon: 102, d: 2.5, head: 12, aim: 0, lift: 0, fov: 36, sx: 0.56, sy: -0.24, zen: 132, saz: 262, rise: 0, exp: 1.4,
    m: { fov: 50 } },
  // 7 questions: close over the city lights, deep in the night
  { shown: 0.55, lat: 33, lon: 124, d: 1.5, head: 8, aim: 0, lift: 0, fov: 36, sx: 0.6, sy: 0.1, zen: 144, saz: 96, rise: 0, exp: 1.5,
    m: { fov: 50, d: 1.9 } },
  // 8 last screen: sunrise again
  { ground: true, lat: 31, lon: 128, d: 1.3, head: 72, aim: 1, lift: 7.5, fov: 36, sx: 0, sy: 0, zen: atLimb(1.3, 1.0), saz: 72, rise: 10, exp: 1.0,
    m: { fov: 58, lift: 10 } }
]
/** Where each key stands in the journey. */
const AT = KEYS.map((_, index) => index / (KEYS.length - 1))

type Track = { values: number[], tangent: number[] }

// Monotone cubic Hermite through the keys: smooth velocity across keys, no overshoot between them.
function buildTrack(values: number[]): Track {
  const n = KEYS.length
  const slope: number[] = []
  for (let i = 0; i < n - 1; i++) slope.push((values[i + 1] - values[i]) / (AT[i + 1] - AT[i]))
  const tangent = new Array<number>(n).fill(0)
  for (let i = 1; i < n - 1; i++) {
    const a = slope[i - 1]
    const b = slope[i]
    if (a * b <= 0) continue
    const mid = (values[i + 1] - values[i - 1]) / (AT[i + 1] - AT[i - 1])
    tangent[i] = Math.sign(mid) * Math.min(Math.abs(mid), 3 * Math.abs(a), 3 * Math.abs(b))
  }
  return { values, tangent }
}
function sample(track: Track, p: number) {
  const n = KEYS.length
  let i = 0
  while (i < n - 2 && p > AT[i + 1]) i++
  const span = AT[i + 1] - AT[i]
  const t = Math.min(1, Math.max(0, (p - AT[i]) / span))
  const t2 = t * t
  const t3 = t2 * t
  return (2 * t3 - 3 * t2 + 1) * track.values[i] + (t3 - 2 * t2 + t) * span * track.tangent[i] +
    (-2 * t3 + 3 * t2) * track.values[i + 1] + (t3 - t2) * span * track.tangent[i + 1]
}

/** Clear space kept between the words and the globe, as a share of the picture. */
const CLEARANCE = 0.03
/** Beside the words at least this much of the globe's width stays in the picture, unless the key says otherwise. */
const GLOBE_SHOWN = 0.72
const WIDEST_LENS = 70
/** The lift that puts the planet's edge at this share of the picture's height (with aim 1 and no lens shift). */
const liftFor = (share: number, fov: number) => Math.atan((2 * Math.min(0.86, share) - 1) * Math.tan(fov * DEG / 2)) / DEG

const tracks = {} as Record<Field, Track>
/** What the curves in `tracks` were built for. */
const built: { portrait: number, aspect: number, clear: number, marks: readonly number[] | undefined | null } = { portrait: -1, aspect: -1, clear: -1, marks: null }
/**
 * The curves for one shape of page. Every stop is placed by what the page measured on its screen
 * (`marks`): a whole globe stands just beyond the words, to their right or below them, and along a
 * horizon the planet's edge keeps below them, whatever the window's proportions.
 */
function tracksFor(portrait: number, aspect: number, marks: readonly number[] | undefined, clear: number) {
  // The page hands over a new list whenever it has measured again, so the list itself tells a change.
  if (built.marks === marks && built.portrait === portrait && built.aspect === aspect && built.clear === clear) return tracks
  const at = (key: Key, field: Field) => {
    const wide = key[field] ?? 0
    return wide + ((key.m?.[field] ?? wide) - wide) * portrait
  }
  /** Lens and lens shift that stand the globe beside the words, or above them. */
  const fitted = (key: Key, edge: number): Partial<Record<Field, number>> => {
    const reach = Math.tan(Math.asin(1 / at(key, 'd')))
    let fov = at(key, 'fov')
    // the globe's radius on screen, in half-heights of the picture
    let radius = reach / Math.tan(fov * DEG / 2)
    const widen = (most: number) => {
      if (radius <= most) return
      fov = Math.min(WIDEST_LENS, 2 * Math.atan(reach / most) / DEG)
      radius = reach / Math.tan(fov * DEG / 2)
    }
    if (portrait >= 0.5) {
      // The words stand at the bottom of an upright page. The globe has the band between the header and
      // them, rises a little from behind them, and is no wider than the page.
      const band = Math.max(0.1, edge - clear)
      widen(Math.min(1.3 * band, 1.04 * aspect))
      return { fov, sx: 0, sy: 1 - 2 * (clear + Math.max(0.015, (band - radius) / 2)) - radius }
    }
    // Beside the words there is only so much room; the lens widens until the globe is small enough.
    const room = (1 - edge) * 2 * aspect
    widen(Math.min(room / (2 * (key.shown ?? GLOBE_SHOWN)), key.size ?? Infinity))
    return { fov, sx: 2 * edge - 1 + radius / aspect }
  }
  const placed = KEYS.map((key, index): Partial<Record<Field, number>> => {
    const mark = marks?.[index]
    if (mark === undefined || !Number.isFinite(mark)) return {}
    if (key.ground) return { lift: Math.max(at(key, 'lift'), liftFor(mark, at(key, 'fov'))) }
    return fitted(key, portrait >= 0.5 ? Math.min(0.8, mark) : Math.min(0.62, mark + CLEARANCE))
  })
  for (const field of FIELDS) {
    tracks[field] = buildTrack(KEYS.map((key, index) => placed[index][field] ?? at(key, field)))
  }
  built.portrait = portrait
  built.aspect = aspect
  built.clear = clear
  built.marks = marks
  return tracks
}

type Vec3 = [number, number, number]
const clamp01 = (v: number) => Math.min(1, Math.max(0, v))
const smooth = (a: number, b: number, v: number) => { const t = clamp01((v - a) / (b - a)); return t * t * (3 - 2 * t) }
const cross = (a: Vec3, b: Vec3): Vec3 => [a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2], a[0] * b[1] - a[1] * b[0]]
const mix2 = (a: Vec3, ka: number, b: Vec3, kb: number): Vec3 => [a[0] * ka + b[0] * kb, a[1] * ka + b[1] * kb, a[2] * ka + b[2] * kb]

/** What the page hands the scene for one frame. */
export interface JourneyState {
  /** Seconds; only for slow ambient motion (cloud drift, the ground sliding by). */
  time: number
  /** 0..1, the opening: black, then sunrise. */
  intro: number
  /** 0..1, how far the page has been scrolled through the journey. */
  progress: number
  /** Pointer position, each -1..1; a slight parallax. */
  pointer: { x: number, y: number }
  /** The page's layout: its words stand above the planet (true) or in a column to its left (false). */
  tall?: boolean
  /**
   * What the page measured on each screen, in the screens' order. Where the globe stands beside the words:
   * the share of the picture's width they take. Where it stands above them: the share of its height at
   * which they begin. Along a horizon: the share of its height at which they end.
   */
  marks?: readonly number[]
  /** The share of the picture's height that the page's header takes at the top. */
  clear?: number
}

export interface JourneyView {
  position: Vec3
  direction: Vec3
  up: Vec3
  sun: Vec3
  sunLight: Vec3
  distance: number
  fov: number
  shiftX: number
  shiftY: number
  exposure: number
  fade: number
  portrait: number
  low: number
  sunAltitude: number
}

export function evaluate(state: JourneyState, aspect: number, out: JourneyView) {
  const portrait = state.tall === undefined ? smooth(1.15, 0.7, aspect) : state.tall ? 1 : 0
  const set = tracksFor(portrait, aspect, state.marks, state.clear ?? 0)
  const p = clamp01(state.progress || 0)
  const v = {} as Record<Field, number>
  for (const field of FIELDS) v[field] = sample(set[field], p)

  const nearTop = 1 - smooth(0, 0.12, p)

  // Opening: only acts near the top of the page. The sun starts below the limb; the camera comes in from
  // further out, tipped further, with the planet lower in the picture, and settles as the sun breaks.
  // The picture is up from black early, so that most of that move is seen.
  const intro = clamp01(state.intro === undefined ? 1 : state.intro)
  const settle = 1 - Math.pow(1 - intro, 2.2)
  const before = (1 - settle) * nearTop
  v.zen += before * 10
  v.rise -= before * 8
  v.d += before * 0.12
  v.lift += before * 2.6
  v.roll += before * 7
  const fade = 1 - nearTop * (1 - smooth(0.0, 0.3, intro))

  // Slow forward drift that settles after a few minutes, so a visitor who lingers sees the ground slide by.
  const time = state.time || 0
  v.lon += 6 * (1 - Math.exp(-time / 240))
  const px = state.pointer ? state.pointer.x || 0 : 0
  const py = state.pointer ? state.pointer.y || 0 : 0
  const near = smooth(2.2, 1.25, v.d)
  v.head += px * 0.6 * near
  v.lift -= py * 0.5 * near
  v.sx += px * 0.014 * (1 - near)
  v.sy -= py * 0.014 * (1 - near)

  const lat = v.lat * DEG
  const lon = v.lon * DEG
  const up: Vec3 = [Math.cos(lat) * Math.cos(lon), Math.sin(lat), -Math.cos(lat) * Math.sin(lon)]
  const east: Vec3 = [-Math.sin(lon), 0, -Math.cos(lon)]
  const north = cross(up, east)
  const bearing = (angle: number) => mix2(north, Math.cos(angle * DEG), east, Math.sin(angle * DEG))

  const gaze = bearing(v.head)
  const limb = Math.asin(1 / v.d)
  const tilt = v.aim * limb + v.lift * DEG
  out.position = [up[0] * v.d, up[1] * v.d, up[2] * v.d]
  out.direction = mix2(up, -Math.cos(tilt), gaze, Math.sin(tilt))
  out.up = mix2(up, Math.sin(tilt), gaze, Math.cos(tilt))
  if (v.roll) {
    const side = cross(out.direction, out.up)
    out.up = mix2(out.up, Math.cos(v.roll * DEG), side, Math.sin(v.roll * DEG))
  }

  const sunGround = bearing(v.saz)
  out.sun = mix2(up, Math.cos(v.zen * DEG), sunGround, Math.sin(v.zen * DEG))
  const lightZen = (v.zen - v.rise) * DEG
  out.sunLight = mix2(up, Math.cos(lightZen), sunGround, Math.sin(lightZen))

  out.distance = v.d
  out.fov = v.fov
  out.shiftX = v.sx
  out.shiftY = v.sy
  out.exposure = v.exp
  out.fade = fade
  out.portrait = portrait
  out.low = smooth(1.75, 1.3, v.d)
  // Degrees of the sun's centre above the Earth's limb as the camera sees it (negative: hidden behind the planet).
  out.sunAltitude = 180 - v.zen - limb / DEG
  return out
}
