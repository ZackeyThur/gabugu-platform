import { describe, expect, it } from 'vitest'

import { withGrade } from '../earth/post'
import { earthFragment } from '../earth/shadersEarth'
import { haloFragment, starFragment, sunFragment } from '../earth/shadersSky'

// The light version of the scene (phones, small or weak machines) has no off-screen pass: the final grade
// is written into each material's fragment shader as text. These tests keep a later edit of a shader from
// silently leaving a material ungraded, which would show as washed-out colour on phones only.
describe('the light version of the live Earth', () => {
  const materials = [
    ['globe', earthFragment, false],
    ['atmosphere', haloFragment, true],
    ['sun', sunFragment, true],
    ['stars', starFragment, true],
  ] as const

  it.each(materials)('writes the final grade into the %s material', (_name, fragment, additive) => {
    const graded = withGrade(fragment, additive)

    expect(graded).toContain('vec3 grade(')
    expect(graded.match(/gl_FragColor = /g)).toHaveLength(1)
    expect(graded).toMatch(/gl_FragColor = vec4\(grade\(.+, uExposure, uFade\)/)
    // What is drawn on top of the picture must not add the page colour a second time.
    expect(graded.includes('- vec3(0.0314, 0.0314, 0.051)')).toBe(additive)
  })

  it('refuses a shader that does not end in the form the grade is written into', () => {
    expect(() => withGrade('void main() {\n  gl_FragColor = vec4(1.0);\n}', false)).toThrow()
  })
})
