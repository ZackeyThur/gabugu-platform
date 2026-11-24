import { describe, expect, it } from 'vitest'

import { journeyAt } from '../journey'
import { earthBeat } from '../earth/context'
import { evaluate, JOURNEY_STOPS, limbAngle, type JourneyState, type JourneyView } from '../earth/story'

describe('journeyAt', () => {
  const stops = [
    { at: 0, progress: 0 },
    { at: 800, progress: 0.34 },
    { at: 1800, progress: 0.6 },
    { at: 3000, progress: 1 },
  ]

  it('reaches each part of the journey where that part of the page begins', () => {
    expect(stops.map((stop) => journeyAt(stop.at, stops))).toEqual([0, 0.34, 0.6, 1])
  })

  it('moves evenly with the scroll between two parts and stays within the journey outside them', () => {
    expect(journeyAt(400, stops)).toBeCloseTo(0.17)
    expect(journeyAt(1300, stops)).toBeCloseTo(0.47)
    expect(journeyAt(-50, stops)).toBe(0)
    expect(journeyAt(9000, stops)).toBe(1)
  })

  it('never runs backwards when a part of the page has no height', () => {
    // On a short page the last sections all end where the page ends.
    const short = [{ at: 0, progress: 0 }, { at: 500, progress: 0.34 }, { at: 400, progress: 0.6 }, { at: 500, progress: 1 }]
    let before = -1
    for (let scroll = 0; scroll <= 600; scroll += 25) {
      const now = journeyAt(scroll, short)
      expect(now).toBeGreaterThanOrEqual(before)
      before = now
    }
    expect(journeyAt(500, short)).toBe(1)
    expect(journeyAt(0, [])).toBe(0)
  })
})

describe('the stops of the journey', () => {
  it('stands one stop on each screen of the page, from 0 to 1', () => {
    // The keys in story.ts are composed for one screen each; adding a screen means adding a key.
    expect(JOURNEY_STOPS).toBe(8)
    const beats = Array.from({ length: JOURNEY_STOPS }, (_, index) => earthBeat(index, JOURNEY_STOPS))
    expect(beats[0]).toBe(0)
    expect(beats[JOURNEY_STOPS - 1]).toBe(1)
    expect(beats[3]).toBeCloseTo(3 / 7)
    for (let index = 1; index < beats.length; index++) expect(beats[index]).toBeGreaterThan(beats[index - 1])
    // A page of a single screen has nowhere to go.
    expect(earthBeat(0, 1)).toBe(0)
  })
})

describe('the camera of the live Earth', () => {
  const state = (progress: number, extra: Partial<JourneyState> = {}): JourneyState =>
    ({ time: 0, intro: 1, progress, pointer: { x: 0, y: 0 }, ...extra })
  const view = (progress: number, aspect: number, extra: Partial<JourneyState> = {}) =>
    evaluate(state(progress, extra), aspect, {} as JourneyView)
  /** The globe's radius on screen, in half-heights of the picture. */
  const radius = (camera: JourneyView) => Math.tan(Math.asin(1 / camera.distance)) / Math.tan(camera.fov * Math.PI / 360)
  /** Where the globe's left edge stands, as a share of the picture's width. */
  const leftEdge = (camera: JourneyView, aspect: number) => (camera.shiftX - radius(camera) / aspect + 1) / 2
  /** Where the edge of the planet crosses the middle of the picture, as a share of its height. */
  const groundEdge = (camera: JourneyView) => {
    const up = camera.position.map((value) => value / camera.distance)
    const fromNadir = Math.acos(-(camera.direction[0] * up[0] + camera.direction[1] * up[1] + camera.direction[2] * up[2]))
    return (1 + Math.tan(fromNadir - Math.asin(1 / camera.distance)) / Math.tan(camera.fov * Math.PI / 360)) / 2
  }
  const stop = (index: number) => index / (JOURNEY_STOPS - 1)

  it('gives the same picture for the same state, whatever was drawn before', () => {
    const first = JSON.stringify(view(0.42, 1.6, { tall: false, marks: [0.5], clear: 0.08, time: 12 }))
    view(0.9, 0.5, { tall: true, marks: [0.3], clear: 0.1 })
    expect(JSON.stringify(view(0.42, 1.6, { tall: false, marks: [0.5], clear: 0.08, time: 12 }))).toBe(first)
  })

  it('reaches each screen of the page at the distance and the light composed for it', () => {
    expect(view(stop(0), 1.6).distance).toBeCloseTo(1.36, 2)
    expect(view(stop(1), 1.6).distance).toBeCloseTo(2.62, 2)
    expect(view(stop(4), 1.6).distance).toBeCloseTo(1.62, 2)
    expect(view(1, 1.6).distance).toBeCloseTo(1.3, 2)
    // The first and the last screen look along the horizon: the planet's edge is in the picture.
    expect(limbAngle(view(stop(0), 1.6).distance)).toBeCloseTo(47.3, 1)
    // The sun sits just above the limb at a sunrise and a sunset, and well up at midday.
    expect(view(stop(0), 1.6).sunAltitude).toBeCloseTo(0.75, 1)
    expect(view(stop(4), 1.6).sunAltitude).toBeCloseTo(0.5, 1)
    expect(view(1, 1.6).sunAltitude).toBeCloseTo(1.0, 1)
    expect(view(stop(1), 1.6).sunAltitude).toBeGreaterThan(30)
  })

  /** One entry for each screen, as the page hands its measures over. */
  const every = (share: number) => Array.from({ length: JOURNEY_STOPS }, () => share)

  it('keeps the globe beside the words on a wide page, just beyond their column', () => {
    for (const aspect of [1.25, 1.6, 2.4]) {
      for (const words of [0.38, 0.46, 0.55]) {
        for (const index of [1, 2, 3, 5, 6]) {
          const camera = view(stop(index), aspect, { tall: false, marks: every(words), clear: 0.08 })
          // The globe begins where the words end, a little clear of them.
          expect(leftEdge(camera, aspect)).toBeCloseTo(Math.min(0.62, words + 0.03), 6)
        }
      }
    }
  })

  it('keeps the globe on an upright page within the band between the header and the words', () => {
    for (const aspect of [0.46, 0.62, 1]) {
      for (const words of [0.3, 0.42]) {
        for (const index of [1, 2, 3, 5]) {
          const camera = view(stop(index), aspect, { tall: true, marks: every(words), clear: 0.1 })
          const top = (1 - camera.shiftY - radius(camera)) / 2
          const bottom = (1 - camera.shiftY + radius(camera)) / 2
          // Centred: a phone's words take the full page, and the globe stands in the middle of it.
          expect(camera.shiftX).toBe(0)
          // Below the header, and beginning above where the words begin...
          expect(top).toBeGreaterThanOrEqual(0.1)
          expect(top).toBeLessThanOrEqual(Math.min(0.8, words) + 1e-6)
          // ...and no wider than the page where the lens can make it so; a close key on a narrow page
          // stands at the widest lens instead, and is cropped by the sides.
          if (radius(camera) / aspect > 1.04 + 1e-9) expect(camera.fov).toBeCloseTo(70, 6)
          expect(bottom).toBeLessThanOrEqual(1)
        }
      }
    }
  })

  it('keeps the edge of the planet below the words of the screens that look along the horizon', () => {
    for (const [aspect, tall] of [[1.6, false], [0.46, true]] as const) {
      for (const index of [0, 4, 7]) {
        const at = stop(index)
        // The camera lifts until the planet's edge stands where the words end.
        expect(groundEdge(view(at, aspect, { tall, marks: every(0.72), clear: 0.08 }))).toBeCloseTo(0.72, 6)
        // Words that end high on the screen leave the composed picture as it is.
        const composed = groundEdge(view(at, aspect, { tall, clear: 0.08 }))
        expect(groundEdge(view(at, aspect, { tall, marks: every(0.2), clear: 0.08 }))).toBeCloseTo(composed, 6)
      }
    }
  })

  it('composes each page shape again when the page has measured again', () => {
    const first = view(stop(3), 1.6, { tall: false, marks: every(0.3), clear: 0.08 })
    const second = view(stop(3), 1.6, { tall: false, marks: every(0.5), clear: 0.08 })
    expect(leftEdge(second, 1.6)).toBeGreaterThan(leftEdge(first, 1.6) + 0.15)
    // The same measures, given again, give the same picture.
    const again = view(stop(3), 1.6, { tall: false, marks: every(0.5), clear: 0.08 })
    expect(JSON.stringify(again)).toBe(JSON.stringify(second))
  })

  it('opens from black and is fully lit once the opening is over or the page has moved on', () => {
    expect(view(0, 1.6, { intro: 0 }).fade).toBe(0)
    expect(view(0, 1.6, { intro: 1 }).fade).toBe(1)
    // A visitor who scrolls on during the opening does not look at a dark picture.
    expect(view(0.2, 1.6, { intro: 0 }).fade).toBe(1)
  })
})
