# Landing

The default home page (`views/HomeView.vue` when neither custom home content nor compact mode is set).
Brand rules for this page are in `docs/BRAND_FRONTEND.md`: black, silver and blue; sans type only.

The page is eight screens, each as tall as the window: the first screen, models, integration, tools,
pricing, control, questions, and what to do next — then the footer. It lies over a live picture of the
Earth, drawn with Three.js and fixed behind it. Scrolling moves a camera through one day, one stop on
each screen: a sunrise seen from orbit, the whole planet by day, close by day, the line between day and
night, dusk along the horizon, the night side with its city lights, close over those lights, and a
sunrise again. Where the picture cannot be drawn, or the visitor asked for less motion or less data, the
page shows a still picture of the same sunrise.

## How the pieces fit

- `views/HomeView.vue` decides, before the first paint, whether the live picture is possible
  (`createEarthSurface`), because that decides the layout (class `staged`). It measures the page after
  every change of size or content, and keeps a plain object, the *journey*, up to date: how far the page
  has been scrolled through the story, what each screen's words take of the picture (see `marks` in
  `earth/context.ts`), and how much of the picture the header takes. Nothing in it is reactive; the stage
  reads it on every frame. It also decides how the page turns (see *Paging* below).
- `journey.ts` turns a scroll position into the story's progress. Each screen has a fixed place in the
  story (`earthBeat`): screen `i` of `n` stands at `i / (n - 1)`, reached when its top stands under the
  header; in between the progress moves evenly with the scroll.
- `EarthStage.vue` downloads the maps and the scene in parallel, measures what one frame costs on this
  machine, and draws. It follows the journey smoothly, draws at full rate only while something moves,
  at about 30 frames a second otherwise and at about 12 once the page has been left alone for twenty
  seconds, and not at all while the tab is hidden. The cost of a frame is measured only while the page is
  in view, so a tab opened in the background is not mistaken for a slow machine.
  It gives up (`failed`) when the context is lost or the machine cannot keep up; the page then tries the
  light version once and after that shows the still picture.
- `earth/` is the scene. Only `earth/scene.ts` imports Three.js, and it is loaded on demand, so the rest
  of the application never downloads it (chunk `vendor-three`; see `manualChunks` in `vite.config.ts`).
  - `context.ts`: everything the page needs before Three.js arrives: the two versions (`high`: 4K maps
    and an HDR pipeline with bloom; `low`: 2K maps drawn straight to the screen, for phones and small or
    weak machines), the canvas and its context, the maps, and `earthBeat`.
  - `story.ts`: the camera, the sun and the exposure for any moment, as a pure function of the journey.
    The same state always gives the same picture, so the page can be scrolled back and forth. One key
    per screen (`JOURNEY_STOPS`): the camera stays over East Asia and the sun goes round, so the move
    from screen to screen changes the light and the distance and never spins the globe. Where the whole
    globe is in view it is placed by what the page measured: its edge stands just beyond the words — to
    their right on a wide page, in the band between the header and the words on an upright one. Where
    the camera looks along the horizon it lifts until the planet's edge stands below the words.
  - `scene.ts`: the globe, its atmosphere, the sun and the stars, and the final grade.
    `shadersEarth.ts`, `shadersSky.ts` and `post.ts` hold the shaders, `noise.ts` a small generated
    noise texture. The look is tuned with the constants in `TUNE`.
- `HeroPoster.vue`: the still picture, in a wide and an upright version. It is placed so that the
  planet's edge stands where the page wants it.
- Entrances fade with opacity, never with visibility, and a part that receives focus is shown at once:
  everything on the page can be reached with the keyboard and by a screen reader before it has come in.
- `useLandingMotion.ts`: motion with GSAP, also loaded on demand (chunk `vendor-gsap`): the opening of
  the first screen, the fade of the words as the page leaves a screen, the single entrance of each part
  of a screen when the page first reaches it, the pricing formula read term by term, the sample key's
  meter, and the fade when the protocol example changes. Without GSAP, or with reduced motion,
  everything is simply visible.
- `useLandingCopy.ts`: all page text in both languages.
- `LandingIcon.vue`, `LandingBrandMark.vue`: icons and the brand mark, shared with the login page.

## Paging

`paging.ts` holds the rules as pure functions; `useLandingPager.ts` applies them. The page stays an
ordinary scrolling document, so links to a screen, the Back button, find-in-page and a screen reader work
as anywhere; paging only decides where scrolling comes to rest.

- `choosePagingMode`: `paged` where every screen fits in the window and the visitor has not asked for
  less motion; `snap` on a touch screen (the browser's own scroll snapping, `scroll-snap-stop: always`,
  set on the document element); `free` otherwise, which is ordinary scrolling.
- In `paged` mode one wheel gesture or one key turns exactly one screen (`stopBeyond`, an rAF tween of
  `turnSeconds` with `easeTurn`). A wheel gesture is read across its events (`readWheel`): a mouse
  notch turns once; a trackpad that goes on sending smaller events for a second after the fingers have
  left it belongs to the gesture that has already turned; a second swipe, a change of direction, a pause
  of more than 140 ms, or a wheel held on at full size for more than 1.1 s begins a new one. A finger
  brushing the trackpad (under 10 px) turns nothing.
- Keys: PageDown, ArrowDown, Space (Shift for up), PageUp, ArrowUp, Home, End. Keys are left alone inside
  forms, tab lists and menus, and on links and buttons.
- Anything else that scrolls — the scrollbar, find-in-page, the browser bringing what has the focus into
  view — is left to do so; 170 ms after it comes to rest the page is set on the nearest screen. A move
  in progress gives way when the page is found away from where the move last put it.
- Whether every screen fits is measured (`measure()`), not assumed: one that does not fit makes the whole
  page scroll freely, so that nothing is ever cut off to keep the screens even.
- The page is always told where the scroll is in the same frame it is put there (`moved`), so the words
  and the picture never trail the scroll by a frame.
- The screen the visitor is on is always read from the scroll position (`nearestStop`), never counted.

## The opening

A first visit sees the sunrise in full (about 3.4 seconds); a later visit in the same session, or a
reload, sees a brief version; a visitor who arrives at a screen or comes back with the Back button finds
everything in place. The headline never waits for the picture: it waits at most 0.6 seconds for its
typeface and 1.5 seconds for GSAP. Over a slow connection the still picture stands in: after 1.4 seconds
where not one map has arrived by then, otherwise after 3.2 seconds, so that an ordinary connection never
sees it. Its small version (about 43 KB) is fetched at once, ahead of the maps, so that it is there when
it is needed. The live picture takes over when it is ready. A tab opened in the background counts these
waits from the moment it is shown.

## Layout

- Wide page (`min-width: 1021px` and wider than tall): each screen is a column of at most 560px on the
  left and the planet stands to its right. The first screen, pricing and the closing use the full width.
- Narrow or upright page: each screen's words stand at the bottom, over a gradient that darkens the
  picture below them, and the planet has the band between the header and the words.
- Every length that matters is in `--landing-vh` (one hundredth of a screen), so the page keeps its
  proportions on a short window. `--if-roomy` turns into a max-height where the window is too short for
  a description, and the description is dropped rather than pushed out of its screen.
- The screen count (`05 / 08`) and the dots at the right say where the visitor is; on a phone only the
  count is shown, under the header, so that it never stands over the words.
- When the live picture goes away while the visitor is reading (the scene failed, or reduced motion was
  switched on), the page becomes shorter; it is scrolled so that what was being read stays in place.
- The media query in `HomeView.vue` (`TALL_MEDIA`) and the one in `landing.css` must stay the same.

## What the page claims

Every statement on the page was checked against the product; change one only together with that check.

- Model families: GPT, Claude, Gemini, Grok and DeepSeek, each with the protocol its groups are called
  by. Grok and DeepSeek groups are served by the OpenAI-compatible endpoints
  (`backend/internal/.../domain/platforms.go`), which is why their rows open the OpenAI example.
- Integration: the three official Python SDKs, pointed at this site. OpenAI's `base_url` takes `/v1`,
  Anthropic's does not, and Google's needs `"api_version": "v1beta"` — its default is `v1`, a route the
  gateway does not serve. All three were run against a local stand-in for the gateway.
- Tools: the tools a key's "Use Key" dialog has configurations for, which of them depending on the
  key's group; the beginner guide after sign-in covers the Codex desktop app and Claude Code.
- Pricing: `費用 = 基础单价 × 分组倍率 × 用量`; models billed per request and peak hours follow their own
  rules, and the model catalog and the usage records are what counts.
- Control: every key can take its own limits; every call is recorded; a key's quota and usage summary
  can be read without signing in (`/key-usage`). The panel on that screen is labelled sample data.
- Settings: the model catalog link, the ways to pay, the documentation link, the agreements in the
  footer and the contact line all follow what the site has switched on; the FAQ drops the question that
  compares balance and subscription where the site does not have both.

## Pictures and their sources

`src/assets/landing/` (see the README there). The maps are equirectangular, 2:1; they are uploaded as they
are stored and the shader turns them over (`MAP()`), so nothing depends on a browser flipping pictures
while decoding. The footer credits NASA and, for the clouds, Solar System Scope (CC BY 4.0, attribution
required). Vite gives the files content-hashed names, so the backend serves them with a long cache.

## Third-party code

- three 0.187.0, MIT.
- gsap 3.15.1, under GSAP's own "Standard 'no charge' licence" (https://gsap.com/standard-license). It is
  free of charge, also on commercial sites, but it is not an open-source licence: it forbids using GSAP
  in tools that compete with Webflow's visual animation builder, and the copyright notices must stay in
  the built files (they do). ScrollTrigger is no longer used.

## Fonts

The page uses the project's sans pair and nothing else: Inter for Latin and digits, and a self-hosted
Noto Sans SC subset for Chinese (`public/brand/noto-sans-sc-landing-<hash>.woff2`, SIL Open Font License,
text in `public/brand/noto-sans-sc-OFL.txt`). No serif face is used. Sizes, weights and tracking follow
`docs/BRAND_FRONTEND.md`.

The subset holds only the characters that appear in `useLandingCopy.ts` and `views/HomeView.vue`. After
changing the Chinese text, rebuild it from `platform/frontend`:

```bash
node scripts/update-landing-font.mjs
```

The script writes the subset under a new name and points `landing.css` at it; published static files keep
their names, so earlier subsets stay where they are. `src/views/__tests__/HomeView.landing.spec.ts` fails
when the page shows a Chinese character the subset lacks.

## Tests

- `src/views/__tests__/HomeView.landing.spec.ts`: the eight screens in order, the header and its links,
  the model rows with their protocols, the SDK examples and the copy button, the pricing formula, the
  ways to pay and the FAQ that follows them, the settings-driven catalog and agreement links, the
  tools, the headline's accessible name, both languages, the credits, the font subset, and the page with
  and without the live picture together with every fallback.
- `src/components/landing/__tests__/paging.spec.ts`: the choice of paging mode, the stop a page rests on,
  the step down and up, the timer of a turn, and the wheel read across mouse notches, trackpad coasting
  and brushes, second swipes, changes of direction, pauses and a held wheel.
- `src/components/landing/__tests__/earthShaders.spec.ts`: the light version's grade is written into
  every material.
- `src/components/landing/__tests__/journey.spec.ts`: scroll position to progress, the eight stops, and
  the camera: the same picture for the same state, each screen's distance and light, the globe beside or
  above the words as measured, the planet's edge below the words along the horizon, and the opening.
- WebGL itself is not exercised by these tests (jsdom has none); it is checked in a real browser, on the
  development server and on the built files, at nine window sizes, in both languages, without WebGL and
  with reduced motion.
