// Everything above the ground: the atmosphere's edge glow, the sun, and the stars.

// ---- atmosphere halo, drawn on the inside of a slightly larger sphere (the globe hides the part behind it).
export const haloVertex = /* glsl */`
varying vec3 vPos;
void main() {
  vPos = position;
  gl_Position = projectionMatrix * viewMatrix * vec4(position, 1.0);
}`

export const haloFragment = /* glsl */`
precision highp float;
uniform vec3 uSunDisc;
uniform float uCore;   // scale height of the thin bright line
uniform float uWide;   // scale height of the soft outer falloff
uniform float uGain;
uniform float uWarm;  // 0 while the sun is well below the limb, 1 once it is about to rise
varying vec3 vPos;
void main() {
  vec3 rd = normalize(vPos - cameraPosition);
  vec3 pc = cameraPosition - rd * dot(cameraPosition, rd);   // point on the view ray closest to the Earth's centre
  float b = length(pc);
  float h = max(b - 1.0, 0.0);
  float mu = dot(pc / b, uSunDisc);
  float lit = pow(smoothstep(-0.3, 0.2, mu), 1.4);
  float core = exp(-h / uCore);
  float wide = exp(-h / uWide);
  float inner = exp(-h / (uCore * 0.4));
  vec3 col = vec3(0.1, 0.33, 1.0) * 1.2 * core + vec3(0.015, 0.07, 0.4) * 0.24 * wide;
  // Toward the sun the blue gives way to orange, and a warm glow hugs the horizon.
  float sa = max(dot(rd, uSunDisc), 0.0);
  col *= 1.0 - 0.8 * pow(sa, 90.0) * uWarm;
  // The innermost line stays cyan-white, and whitens further beside the sun.
  col += mix(vec3(0.5, 0.8, 1.0), vec3(1.0, 0.95, 0.86), pow(sa, 60.0) * uWarm) * (1.1 + 2.2 * pow(sa, 200.0) * uWarm) * inner;
  float low = exp(-h / (uCore * 2.2));
  col += vec3(1.0, 0.5, 0.14) * (pow(sa, 700.0) * 3.0 + pow(sa, 90.0) * 0.6 + pow(sa, 14.0) * 0.22) * low * uWarm;
  col += vec3(1.0, 0.42, 0.1) * pow(sa, 220.0) * 0.1 * exp(-h / (uWide * 1.1)) * uWarm;
  col *= lit * uGain;
  col += vec3(0.0035, 0.009, 0.024) * core;
  gl_FragColor = vec4(col, 1.0);
}`

// ---- sun: a camera-facing quad at infinity; the globe's depth hides whatever is below the horizon.
export const sunVertex = /* glsl */`
uniform vec3 uSunDisc;
uniform float uSize;   // tangent of the quad's half angle
varying vec2 vQ;
void main() {
  vQ = position.xy;
  vec3 centre = mat3(viewMatrix) * uSunDisc * 50.0;
  gl_Position = projectionMatrix * vec4(centre + vec3(position.xy * uSize * 50.0, 0.0), 1.0);
}`

export const sunFragment = /* glsl */`
precision highp float;
uniform float uSize;
uniform vec3 uCoreColor;
uniform vec3 uGlowColor;
uniform float uPower;
uniform float uGlow;
varying vec2 vQ;
void main() {
  float q = length(vQ);
  float r = q * uSize;
  const float R = 0.0052;
  float disc = smoothstep(R * 1.2, R * 0.75, r);
  float corona = exp(-r / (R * 1.5));
  float glow = exp(-r / (R * 2.4)) * 0.2 + 0.0035 / (1.0 + pow(r / (R * 9.0), 2.0));
  float fade = smoothstep(1.0, 0.3, q);
  // White-hot centre; the tint from the air only shows in the corona and the glow.
  vec3 hot = mix(uCoreColor, vec3(1.0, 0.97, 0.92), 0.7);
  vec3 col = hot * disc * 22.0 + mix(uCoreColor, hot, 0.4) * corona * 1.8 + mix(uGlowColor, vec3(1.0, 0.78, 0.55), 0.35) * glow * uGlow * fade;
  gl_FragColor = vec4(col * uPower, 1.0);
}`

// ---- stars: fixed-seed points at infinity.
export const starVertex = /* glsl */`
attribute float aSize;
attribute float aGain;
attribute float aPhase;
uniform float uTime;
uniform float uPixel;
uniform float uGain;
uniform vec3 uSunDisc;
uniform float uSunShow;
varying float vGain;
varying vec3 vTint;
void main() {
  vec3 dir = normalize(position);
  float twinkle = 0.86 + 0.14 * sin(uTime * (0.7 + aPhase * 1.3) + aPhase * 40.0);
  float nearSun = smoothstep(0.55, 0.985, dot(dir, uSunDisc)) * uSunShow;
  float k = fract(aPhase * 13.7);
  float t = fract(aPhase * 29.3);
  vGain = aGain * twinkle * uGain * (1.0 - 0.94 * nearSun) * (0.45 + 1.5 * k * k);
  // Mostly blue-white, a few warm giants.
  vTint = mix(mix(vec3(0.72, 0.83, 1.0), vec3(0.95, 0.96, 1.0), smoothstep(0.0, 0.6, t)), vec3(1.0, 0.8, 0.6), smoothstep(0.82, 1.0, t));
  gl_Position = projectionMatrix * vec4(mat3(viewMatrix) * position, 1.0);
  gl_PointSize = aSize * uPixel * (0.8 + 0.75 * k * k);
}`

export const starFragment = /* glsl */`
precision highp float;
varying float vGain;
varying vec3 vTint;
void main() {
  float d = length(gl_PointCoord - 0.5) * 2.0;
  float a = smoothstep(1.0, 0.0, d);
  gl_FragColor = vec4(vTint * vGain * a * a, 1.0);
}`
