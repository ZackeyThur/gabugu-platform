/**
 * Rebuilds the Landing's Chinese font subset (Noto Sans SC) after the page text changes.
 *
 *   node scripts/update-landing-font.mjs                        (from platform/frontend)
 *   NODE_USE_ENV_PROXY=1 node scripts/update-landing-font.mjs   (behind an HTTP proxy)
 *
 * Published static files keep their names for good, so each rebuild is written under a new name, the
 * first eight hex digits of its SHA-256, and landing.css is pointed at it. Earlier subsets stay in place.
 */
import { createHash } from 'node:crypto'
import { access, readFile, writeFile } from 'node:fs/promises'

const root = new URL('../', import.meta.url)
const sources = ['src/components/landing/useLandingCopy.ts', 'src/views/HomeView.vue']
const stylesheet = new URL('src/components/landing/landing.css', root)
const subsetUrl = /\/brand\/noto-sans-sc-landing[\w.-]*\.woff2/

// '中文' is the language switcher's label; it comes from src/i18n, not from the Landing copy.
const text = (await Promise.all(sources.map(file => readFile(new URL(file, root), 'utf8')))).join('') + '中文'
const characters = [...new Set([...text].filter(character => character.codePointAt(0) > 127))].sort().join('')

async function download(url, headers) {
  const response = await fetch(url, { headers })
  if (!response.ok) throw new Error(`Font asset request failed: ${response.status} ${new URL(url).hostname}`)
  return response
}

// Google Fonts returns one variable WOFF2 holding exactly these characters.
const cssUrl = 'https://fonts.googleapis.com/css2?family=Noto+Sans+SC:wght@400;500;600&display=swap&text=' + encodeURIComponent(characters)
const fontCss = await (await download(cssUrl, {
  'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36'
})).text()
const fontUrls = [...new Set([...fontCss.matchAll(/url\(([^)]+)\)/g)].map(match => match[1]))]
if (fontUrls.length !== 1) throw new Error('Expected a single variable WOFF2 subset; inspect the returned font CSS before updating.')
const font = Buffer.from(await (await download(fontUrls[0])).arrayBuffer())
if (font.subarray(0, 4).toString() !== 'wOF2') throw new Error('Expected a WOFF2 font')

// The licence published beside the first subset covers every subset of the same family.
await access(new URL('public/brand/noto-sans-sc-OFL.txt', root))

const styles = await readFile(stylesheet, 'utf8')
if (!subsetUrl.test(styles)) throw new Error('landing.css no longer points at a Landing font subset')
const name = `noto-sans-sc-landing-${createHash('sha256').update(font).digest('hex').slice(0, 8)}.woff2`
await writeFile(new URL(`public/brand/${name}`, root), font)
await writeFile(stylesheet, styles.replace(subsetUrl, `/brand/${name}`))
console.log(`Wrote public/brand/${name}: ${characters.length} characters, ${font.length} bytes; landing.css now uses it.`)
