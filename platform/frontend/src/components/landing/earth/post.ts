// Final grade: exposure, ACES tone mapping, a little vignette and film grain, then sRGB encoding.
// GRADE is also inlined into every material on the low tier, where there is no off-screen pass at all.
// BLACK in it is the page colour of the Landing (--landing-bg, #08080d).
export const GRADE = /* glsl */`
vec3 acesFilm(vec3 color) {
  // ACES fitted (Stephen Hill): sRGB -> ACEScg-ish -> RRT+ODT fit -> sRGB.
  const mat3 inputMat = mat3(0.59719, 0.07600, 0.02840, 0.35458, 0.90834, 0.13383, 0.04823, 0.01566, 0.83777);
  const mat3 outputMat = mat3(1.60475, -0.10208, -0.00327, -0.53108, 1.10813, -0.07276, -0.07367, -0.00605, 1.07602);
  color = inputMat * color;
  vec3 a = color * (color + 0.0245786) - 0.000090537;
  vec3 b = color * (0.983729 * color + 0.4329510) + 0.238081;
  color = outputMat * (a / b);
  return clamp(color, 0.0, 1.0);
}
vec3 encodeSrgb(vec3 c) {
  return mix(c * 12.92, 1.055 * pow(max(c, vec3(0.0)), vec3(1.0 / 2.4)) - 0.055, step(vec3(0.0031308), c));
}
// Scene-linear radiance -> display. BLACK is the page background the picture has to sink into.
vec3 grade(vec3 hdr, float exposure, float fade) {
  vec3 c = acesFilm(hdr * exposure * 1.5 * fade);
  c = encodeSrgb(c);
  const vec3 BLACK = vec3(0.0314, 0.0314, 0.051);
  return BLACK * (1.0 - c) + c;
}`

/**
 * The light version has no off-screen pass, so each material grades its own output: the grade is written
 * into its fragment shader. `additive` is for what is drawn on top of the picture, which must not add the
 * page colour a second time. Throws when a shader does not end as expected, so that a later edit cannot
 * leave a material ungraded without anyone noticing.
 */
export function withGrade(fragment: string, additive: boolean) {
  const graded = fragment
    .replace('void main() {', `uniform float uExposure;\nuniform float uFade;\n${GRADE}\nvoid main() {`)
    .replace(/gl_FragColor = vec4\((.+), 1\.0\);\s*}\s*$/, additive
      ? 'gl_FragColor = vec4(grade($1, uExposure, uFade) - vec3(0.0314, 0.0314, 0.051), 1.0);\n}'
      : 'gl_FragColor = vec4(grade($1, uExposure, uFade), 1.0);\n}')
  if (!graded.includes('vec3 grade(') || !/gl_FragColor = vec4\(grade\(/.test(graded)) {
    throw new Error('the fragment shader does not end in the form the grade is written into')
  }
  return graded
}

export const FinalShader = {
  name: 'EarthFinalShader',
  uniforms: {
    tDiffuse: { value: null },
    uExposure: { value: 1 },
    uFade: { value: 1 },
    uTime: { value: 0 },
    uAspect: { value: 1.6 },
    uVignette: { value: 0.3 },
    uGrain: { value: 0.022 }
  },
  vertexShader: /* glsl */`
    varying vec2 vUv;
    void main() {
      vUv = uv;
      gl_Position = vec4(position.xy, 0.0, 1.0);
    }`,
  fragmentShader: /* glsl */`
    precision highp float;
    uniform sampler2D tDiffuse;
    uniform float uExposure;
    uniform float uFade;
    uniform float uTime;
    uniform float uAspect;
    uniform float uVignette;
    uniform float uGrain;
    varying vec2 vUv;
    ${GRADE}
    float hash(vec2 p) {
      vec3 q = fract(vec3(p.xyx) * vec3(0.1031, 0.1030, 0.0973));
      q += dot(q, q.yzx + 33.33);
      return fract((q.x + q.y) * q.z);
    }
    void main() {
      vec3 hdr = texture2D(tDiffuse, vUv).rgb;
      vec2 q = (vUv - 0.5) * vec2(min(uAspect, 1.7), 1.0) * 1.25;
      float vig = 1.0 - uVignette * smoothstep(0.35, 1.25, length(q));
      vec3 c = grade(hdr * vig, uExposure, uFade);
      // Grain: strongest in the mid tones, quiet in pure black and in highlights. Also dithers away banding.
      float frame = floor(uTime * 24.0);
      float n = hash(gl_FragCoord.xy + frame * vec2(17.0, 59.0)) + hash(gl_FragCoord.yx + frame * vec2(31.0, 7.0)) - 1.0;
      float luma = dot(c, vec3(0.299, 0.587, 0.114));
      c += n * (uGrain * (0.35 + 1.4 * luma * (1.0 - luma)) + 0.004);
      gl_FragColor = vec4(c, 1.0);
    }`
}
