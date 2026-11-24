import { describe, expect, it } from 'vitest'

import {
  choosePagingMode,
  easeTurn,
  nearestStop,
  readWheel,
  stopBeyond,
  turnSeconds,
  type WheelTrace
} from '../paging'

describe('how the page turns', () => {
  it('turns screen by screen only where every screen fits and the visitor has not asked for less motion', () => {
    expect(choosePagingMode({ fits: true, reducedMotion: false, touch: false })).toBe('paged')
    // A finger scrolls as on any other page; the browser snaps the screens.
    expect(choosePagingMode({ fits: true, reducedMotion: false, touch: true })).toBe('snap')
    expect(choosePagingMode({ fits: false, reducedMotion: false, touch: false })).toBe('free')
    expect(choosePagingMode({ fits: false, reducedMotion: false, touch: true })).toBe('free')
    // Less motion wins over everything: the page turns at once, or not at all.
    expect(choosePagingMode({ fits: true, reducedMotion: true, touch: false })).toBe('free')
    expect(choosePagingMode({ fits: true, reducedMotion: true, touch: true })).toBe('free')
    expect(choosePagingMode({ fits: false, reducedMotion: true, touch: false })).toBe('free')
  })

  it('knows which stop the page is at, and which one lies a step down or up', () => {
    const stops = [0, 800, 1600, 2400, 2800]
    expect(nearestStop(0, stops)).toBe(0)
    expect(nearestStop(390, stops)).toBe(0)
    expect(nearestStop(420, stops)).toBe(1)
    expect(nearestStop(2790, stops)).toBe(4)
    // Halfway between two stops counts as lying at the first of them.
    expect(nearestStop(400, stops)).toBe(0)
    expect(nearestStop(2600, stops)).toBe(3)
    expect(nearestStop(2610, stops)).toBe(4)
    expect(nearestStop(999, [])).toBe(0)

    expect(stopBeyond(0, stops, 1)).toBe(1)
    // A page between two stops is on its way already: a step goes on to the next stop in that direction.
    expect(stopBeyond(800, stops, 1)).toBe(2)
    expect(stopBeyond(790, stops, 1)).toBe(1)
    // At either end it stays there.
    expect(stopBeyond(2800, stops, 1)).toBe(4)
    expect(stopBeyond(2790, stops, 1)).toBe(4)
    expect(stopBeyond(0, stops, -1)).toBe(0)
    expect(stopBeyond(800, stops, -1)).toBe(0)
    expect(stopBeyond(810, stops, -1)).toBe(1)
    // A position a little above a stop is at that stop, not already past it.
    expect(stopBeyond(801, stops, -1)).toBe(0)
  })

  it('takes a little longer the further it goes, but not in proportion', () => {
    expect(turnSeconds(0, 800)).toBe(0)
    expect(turnSeconds(-5, 800)).toBe(0)
    expect(turnSeconds(300, 0)).toBe(0)
    // One screen: well under a second.
    expect(turnSeconds(800, 800)).toBeCloseTo(0.75)
    expect(turnSeconds(200, 800)).toBeCloseTo(0.525)
    expect(turnSeconds(3200, 800)).toBeCloseTo(1.2)
    // However far, never a long wait.
    expect(turnSeconds(12800, 800)).toBe(1.6)
  })

  it('eases a turn in and out, evenly, and clamps what is outside it', () => {
    expect(easeTurn(0)).toBe(0)
    expect(easeTurn(1)).toBe(1)
    expect(easeTurn(0.5)).toBeCloseTo(0.5)
    expect(easeTurn(0.25)).toBeCloseTo(0.0625)
    expect(easeTurn(0.75)).toBeCloseTo(0.9375)
    expect(easeTurn(-1)).toBe(0)
    expect(easeTurn(2)).toBe(1)
    let before = -1
    for (let step = 0; step <= 100; step++) {
      const value = easeTurn(step / 100)
      expect(value).toBeGreaterThanOrEqual(before)
      before = value
    }
  })
})

/** One gesture: its events, each with the time it came and its size in pixels. */
function gesture(events: [number, number][], direction: 1 | -1 = 1) {
  const seen: { trace: WheelTrace | undefined } = { trace: undefined }
  const turns: number[] = []
  events.forEach(([at, size], index) => {
    const read = readWheel(seen.trace, at, size, direction)
    seen.trace = read.trace
    if (read.turn) turns.push(index)
  })
  return turns
}

describe('reading the wheel', () => {
  it('turns the page on one notch of a mouse wheel, and not again for the rest of that gesture', () => {
    expect(gesture([[1000, 100]])).toEqual([0])
    expect(gesture([[1000, 100], [1016, 120], [1032, 120], [1048, 90]])).toEqual([0])
    expect(gesture([[1000, 100]], -1)).toEqual([0])
  })

  it('leaves a finger brushing the trackpad alone', () => {
    // Below what a turn is worth, however many events it comes in.
    expect(gesture([[1000, 3], [1016, 4], [1032, 2]])).toEqual([])
    // The sum of the gesture is what counts.
    expect(gesture([[1000, 9], [1016, 1]])).toEqual([1])
  })

  it('turns once however long a trackpad goes on coasting after the fingers have left it', () => {
    expect(gesture([[1000, 60], [1016, 40], [1032, 25], [1048, 15], [1064, 9], [1080, 5]])).toEqual([0])
  })

  it('turns again on a second swipe, once the events had died down and grow again', () => {
    // The second swipe arrives out of the coasting of the first, 300 ms after it began.
    expect(gesture([[1000, 60], [1016, 30], [1032, 18], [1048, 10], [1300, 55]])).toEqual([0, 4])
    // Growth alone is not a second swipe: a coasting tail grows for a moment now and then.
    expect(gesture([[1000, 60], [1016, 30], [1032, 18], [1048, 12], [1064, 16]])).toEqual([0])
    // Too late to belong to the first, so it is simply a new gesture.
    expect(gesture([[1000, 60], [1200, 60]])).toEqual([0, 1])
  })

  it('turns on the way back at once', () => {
    const first = readWheel(undefined, 1000, 100, 1)
    expect(first.turn).toBe(true)
    const back = readWheel(first.trace, 1016, 100, -1)
    expect(back.turn).toBe(true)
    expect(back.trace.direction).toBe(-1)
  })

  it('turns again after a wheel that is held on and on at full size', () => {
    const held: [number, number][] = Array.from({ length: 30 }, (_, index) => [1000 + index * 100, 100])
    // A second turn a little over a second into the gesture, and the next a second later.
    expect(gesture(held)).toEqual([0, 12, 24])
  })

  it('keeps a gesture to its own direction and pause, and remembers the turn it spent', () => {
    const first = readWheel(undefined, 1000, 40, 1)
    expect(first.turn).toBe(true)
    expect(first.trace.spent).toBe(true)
    expect(first.trace.sum).toBe(40)
    const next = readWheel(first.trace, 1016, 40, 1)
    expect(next.turn).toBe(false)
    expect(next.trace.sum).toBe(80)
    expect(next.trace.peak).toBe(40)
    // A pause of more than a moment begins a new gesture, spent again from zero.
    const later = readWheel(next.trace, 1200, 40, 1)
    expect(later.turn).toBe(true)
    expect(later.trace.sum).toBe(40)
  })
})
