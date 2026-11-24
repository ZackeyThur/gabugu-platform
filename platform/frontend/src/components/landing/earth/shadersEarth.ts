// Globe surface: day/night blend, city lights, clouds with relief and cast shadows, sun glint, aerial perspective.
// World space equals object space (the globe never rotates; the storyline moves the camera and the sun instead).
// The maps are uploaded as they are stored, first row at the north edge, and are read through MAP(), which
// turns v over. All other arithmetic keeps v pointing north.
export const earthVertex = /* glsl */`
varying vec3 vPos;
varying vec2 vUv;
void main() {
  vPos = position;
  vUv = uv;
  gl_Position = projectionMatrix * viewMatrix * vec4(position, 1.0);
}`

export const earthFragment = /* glsl */`
precision highp float;
uniform sampler2D tDay;
uniform sampler2D tLights;
uniform sampler2D tClouds;
uniform sampler2D tWater;
uniform sampler2D tNoise;
uniform vec3 uSun;       // light direction on the ground (runs slightly ahead of the visible disc at sunrise)
uniform vec3 uSunDisc;   // direction of the visible sun
uniform float uCloudShift;
uniform float uSunPower;
uniform float uCity;
uniform float uTwilight;
uniform float uLow;      // 1 in low orbit at the limb (blue hour styling), 0 when the whole globe is in view
varying vec3 vPos;
varying vec2 vUv;

#define MAP(uv) vec2((uv).x, 1.0 - (uv).y)

// Sunlight colour after its path through the air; m is the cosine of the sun's zenith angle.
vec3 sunTint(float m) {
  float air = 1.0 / (max(m, 0.0) + 0.06);
  return exp(-vec3(0.03, 0.076, 0.22) * air) * vec3(1.03, 1.075, 1.24);
}

void main() {
  vec3 N = normalize(vPos);
  vec3 V = normalize(cameraPosition - vPos);
  float NdV = clamp(dot(N, V), 0.0, 1.0);
  float mu = dot(N, uSun);
  float muDisc = dot(N, uSunDisc);

  // Local east / north, and the sun's ground direction expressed as a UV step per radian of travel.
  vec3 E = normalize(vec3(N.z, 0.0, -N.x) + vec3(1e-6, 0.0, 0.0));
  vec3 Nn = cross(N, E);
  float cosLat = max(sqrt(max(1.0 - N.y * N.y, 0.0)), 0.08);
  vec3 Lt = normalize(uSun - N * mu + vec3(1e-6));
  vec2 sunUv = vec2(dot(Lt, E) / (cosLat * 6.28318), dot(Lt, Nn) / 3.14159);

  // Noise at three scales (about 2600, 520 and 120 km per tile), each warped by the one before.
  vec2 nUv = vUv * vec2(2.0, 1.0);
  vec4 n0 = texture2D(tNoise, nUv * 7.7);
  const mat2 ROT1 = mat2(0.8, 0.6, -0.6, 0.8);
  const mat2 ROT2 = mat2(0.6, -0.8, 0.8, 0.6);
  vec2 uv1 = ROT1 * nUv * 38.0 + n0.ra * 0.16;
  vec4 n1 = texture2D(tNoise, uv1);
  vec2 uv2 = ROT2 * nUv * 163.0 + n1.ar * 0.14;
  vec4 n2 = texture2D(tNoise, uv2);

  // ---- clouds: the map gives coverage; fractal noise erodes it into wisps and billows that survive magnification
  vec2 cuv = vUv + vec2(uCloudShift, 0.0);
  float cRaw = texture2D(tClouds, MAP(cuv)).r;
  float detail = n1.r * 0.34 + n1.g * 0.2 + n2.g * 0.28 + n2.a * 0.18;
#ifdef HIGH
  // A fourth, 30 km octave that only matters in low orbit, where the maps run out of pixels.
  vec4 n3 = texture2D(tNoise, ROT1 * nUv * 640.0 + n2.ra * 0.12);
  float fine = (n3.g * 0.6 + n3.a * 0.4 - 0.5) * uLow;
  detail = clamp(detail + fine * 0.34, 0.0, 1.0);
#else
  float fine = 0.0;
#endif
  float erodeK = mix(0.2, 0.3, uLow);
  float erode = (1.0 - detail) * erodeK;
  float dens = clamp((cRaw * mix(1.1, 1.4, uLow) - erode) / (1.0 - erode), 0.0, 1.0);
  float c = smoothstep(mix(0.0, 0.05, uLow), mix(0.45, 0.68, uLow), dens);
  c *= c * (3.0 - 2.0 * c) * 0.35 + 0.65;
  // Thin veil around the dense cores, so cloud banks have soft translucent margins instead of cut-out edges.
  c = max(c, 0.42 * smoothstep(0.05, 0.7, cRaw + (detail - 0.5) * 0.3) * uLow);

  float slant = 1.0 / max(mu + 0.02, 0.085);
  float cShadow = texture2D(tClouds, MAP(cuv + sunUv * mix(0.0017 * slant, 0.003, uLow))).r;
  float cSun = texture2D(tClouds, MAP(cuv + sunUv * mix(0.0007 * min(slant, 5.0), 0.0009, uLow))).r;
#ifdef HIGH
  // The same noise a step toward the sun: slopes that face it light up, the far sides fall into shade.
  vec4 n1s = texture2D(tNoise, uv1 + ROT1 * (sunUv * vec2(2.0, 1.0)) * 0.05);
  vec4 n2s = texture2D(tNoise, uv2 + ROT2 * (sunUv * vec2(2.0, 1.0)) * 0.05);
  float detailSun = n1s.r * 0.34 + n1s.g * 0.2 + n2s.g * 0.28 + n2s.a * 0.18;
  float relief = clamp((cRaw - cSun) * mix(2.6, 4.0, uLow) + (detail - detailSun) * mix(3.4, 2.6, uLow), -1.0, 1.0);
#else
  float relief = clamp((cRaw - cSun) * 3.2 + (n2.g - 0.5) * 0.5, -1.0, 1.0);
#endif
  float shadowMask = smoothstep(0.1, 0.7, cShadow);
  float selfShadow = 1.0 - mix(0.66, 0.5, uLow) * shadowMask * (1.0 - dens * 0.7);
  float groundShadow = 1.0 - mix(0.9, 0.78, uLow) * shadowMask;
  // Thick cores and billow tops are bright, thin veils and creases are dim.
  float body = (0.45 + 0.55 * smoothstep(0.05, 0.85, dens)) * mix(0.9 + 0.2 * detail, 0.55 + 0.8 * detail, uLow);

  // ---- light
  float toSun = max(dot(-V, uSunDisc), 0.0);
  // With the sun on the horizon most of its light reaches the camera by forward scattering.
  float phase = mix(1.0, 0.22 + 2.2 * pow(toSun, 5.0), uLow);
  float dif = pow(clamp(mu * 0.96 + 0.04, 0.0, 1.0), 0.8);
  // Cloud tops stand above the ground and catch the sun earlier, and redder.
  float muCloud = mu + 0.15 * uLow;
  float difCloud = pow(clamp(muCloud * 0.9 + 0.1, 0.0, 1.0), 0.6);
  vec3 sunC = sunTint(mu) * uSunPower * phase;
  float phaseCloud = mix(1.0, 0.24 + 0.85 * pow(toSun, 4.0), uLow);
  vec3 sunCloud = sunTint(muCloud - 0.1 * uLow + 0.04) * uSunPower * phaseCloud;
  // Blue hour: sky light that lingers on the night side of the terminator.
  float zone = smoothstep(mix(-0.15, -0.62, uLow), -0.02, mu) * (1.0 - 0.8 * smoothstep(0.0, 0.3, mu));
  vec3 fill = mix(vec3(0.045, 0.12, 0.24), vec3(0.075, 0.125, 0.25), uLow) * zone * mix(0.16, 1.0, uLow) * uTwilight;
  vec3 moon = vec3(0.0042, 0.0068, 0.0135) * mix(1.0, 1.3, uLow);

  // ---- ground and sea
  vec3 day = texture2D(tDay, MAP(vUv)).rgb;
  float water = texture2D(tWater, MAP(vUv)).r;
  vec3 albedo = mix(day * 1.12, vec3(0.006, 0.024, 0.064) + day * 0.1, water);
  albedo *= 1.0 + fine * mix(0.7, 0.35, water);
  vec3 dusk = mix(albedo, vec3(dot(albedo, vec3(0.3, 0.5, 0.2))), 0.55 * (1.0 - water));
  vec3 surf = albedo * sunC * dif * groundShadow + dusk * (fill * mix(2.2, 1.5, water) + moon);
  // The sea mirrors the sky at grazing angles; wind roughens it in patches.
  float graze = pow(1.0 - NdV, 4.0);
  float wind = smoothstep(0.25, 0.75, n0.a) * (0.45 + 1.1 * n1.r) + 0.25;
  surf += water * (fill * (0.35 + 0.5 * wind) + moon) * (0.1 + 1.5 * graze) * groundShadow;

  // Sun glint: a broad rough-sea lobe with a tight core.
  vec3 Lg = normalize(uSun + N * (0.03 + 0.15 * uLow));
  vec3 Hv = normalize(Lg + V);
  float nh = max(dot(N, Hv), 0.0);
  float fres = 0.02 + 0.98 * pow(1.0 - max(dot(V, Hv), 0.0), 5.0);
  float lobe = pow(nh, 9.0) * 0.07 * (1.0 - uLow) + pow(nh, 36.0) * mix(0.34, 0.2, uLow) + pow(nh, 170.0) * 0.9 + pow(nh, 800.0) * 3.0;
  float sparkle = mix(0.55 + 0.9 * n2.b + 0.5 * n2.r, 0.25 + 1.5 * n2.r * n2.r + 1.4 * n2.b, uLow);
  vec3 glintC = mix(sunTint(mu + 0.02), vec3(1.0, 0.47, 0.15), 0.8 * uLow) * uSunPower;
  float glintOn = smoothstep(-0.16, 0.02, dot(N, Lg) - 0.16 * uLow);
  surf += glintC * lobe * (min(fres, 0.5) * 5.0 + 0.14) * wind * sparkle * water * (1.0 - 0.85 * shadowMask) * glintOn;

  // ---- city lights: the map gives the glow, sparse speckle turns magnified blobs into separate lamps
  float night = 1.0 - smoothstep(-0.13, 0.03, mu);
  float li = texture2D(tLights, MAP(vUv)).r;
  float lamps = mix(0.16 + 2.6 * n2.b * n2.b + 1.1 * n2.a * n1.b, 0.07 + 9.0 * pow(n1.b, 1.6) * (0.5 + 1.5 * n2.a) + 2.2 * n2.b * n2.b, uLow);
  vec3 cityTone = mix(vec3(1.0, 0.46, 0.15), mix(vec3(1.0, 0.8, 0.56), vec3(1.0, 0.62, 0.3), uLow), smoothstep(0.25, 1.0, li));
  float liHigh = pow(li, 2.0) * 6.5 + li * 0.16;
  liHigh = liHigh / (1.0 + 0.3 * liHigh) * 2.1;   // big cities roll off instead of clipping to white
  vec3 cityGlow = cityTone * mix(liHigh, (pow(li, 1.8) * 2.6 + li * 0.2) * 0.5, uLow) * uCity * night;
  vec3 city = cityGlow * lamps;
  surf += city * (1.0 - 0.35 * graze);

  // ---- clouds over everything
  // Thin edges between the camera and the sun glow from behind (silver lining).
  float lining = c * (1.0 - c) * 4.0 * pow(toSun, 3.0) * uLow;
  vec3 cloudLight = sunCloud * difCloud * (max(mix(0.92, 0.34, uLow) + mix(1.25, 1.5, uLow) * relief, 0.03) * selfShadow * body + lining * 0.5)
    + fill * (mix(2.3, 1.7, uLow) + 1.0 * relief) * body + moon * mix(1.1, 1.3, uLow) * body;
  vec3 cloudCol = mix(vec3(1.02, 1.01, 0.98), vec3(0.94), uLow) * max(cloudLight, vec3(0.0));
  vec3 col = mix(surf, cloudCol, c) + mix(city, cityGlow * 0.3, uLow) * c * 0.3;

  // ---- air between the camera and the ground
  float path = pow(1.0 - NdV, 5.0);
  float airLit = smoothstep(mix(-0.06, -0.22, uLow), mix(0.3, 0.22, uLow), muDisc);
  vec3 airTint = mix(vec3(1.0), sunTint(muDisc + 0.1), 0.6);
  col *= 1.0 - 0.45 * path;
  col += vec3(0.1, 0.27, 0.78) * (0.07 * (1.0 - uLow * 0.6) + 1.15 * path) * airLit * airTint * (1.0 - 0.75 * pow(toSun, 90.0));
  // Warm belt along the terminator and toward the sun.
  col += vec3(1.0, 0.52, 0.17) * exp(-pow((mu - 0.03) / 0.045, 2.0)) * (0.012 + 0.2 * path) * (1.0 - uLow);
  col += vec3(1.0, 0.42, 0.11) * pow(toSun, 60.0) * path * 0.9 * smoothstep(-0.25, 0.02, muDisc);
  col += vec3(0.003, 0.0075, 0.02) * path * mix(1.0, 1.4, uLow);

  gl_FragColor = vec4(col, 1.0);
}`
