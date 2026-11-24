# Pictures of the Landing

Imported by `components/landing/EarthStage.vue` and `HeroPoster.vue`. Vite publishes them under
content-hashed names in `assets/`.

## Maps of the live Earth

Equirectangular, 2:1, longitude -180° at the left edge, first row at the north edge. The `4k` files
(4096×2048) are for the full version of the scene, the `2k` files (2048×1024) for the light one.

| File | What | Source | Licence |
| --- | --- | --- | --- |
| `earth-day-*.webp` | Colour of the ground by day | NASA Blue Marble Next Generation with topography, July 2004 | Public domain (NASA) |
| `earth-water-*.webp` | White where there is water | Derived from the same Blue Marble image | Public domain (NASA) |
| `earth-lights-*.webp` | City lights, one channel | NASA Earth Observatory, Black Marble 2016 | Public domain (NASA) |
| `earth-clouds-*.webp` | Cloud cover, one channel | Solar System Scope, "Earth Clouds" texture, https://www.solarsystemscope.com/textures/ | CC BY 4.0 (https://creativecommons.org/licenses/by/4.0/): attribution required. Adapted: resized from 8K, reduced to one channel, re-encoded as WebP |

The page footer names both sources and links to them; for the cloud map it names the author, links the
source and the licence, and says that the map was adapted, as CC BY 4.0 asks. NASA imagery is used without
any suggestion of endorsement by NASA. The cloud map must keep its attribution wherever it is shown.

## Still pictures of the first screen

`hero-poster-2560.webp`, `hero-poster-1280.webp` (16:9) and `hero-poster-portrait-1440.webp`,
`hero-poster-portrait-720.webp` (9:16): a sunrise over the planet's edge. They stand in for the live Earth
where it cannot be drawn and while it is still on its way.

These two pictures are generated images, made on 2026-11-11 for this page with an image model
(`gpt-image-2.6-sunburst`) from a written description of an orbital sunrise; they are not photographs
and carry no third-party rights. `HeroPoster.vue` holds measurements of them (where the planet's edge
and the sun stand); if a picture is replaced, measure again.

## Size

The twelve pictures (eight maps, four stills) add about 2.9 MB to the repository (the 4K cloud map
alone 1.16 MB). The built
front end under `backend/internal/web/dist` is tracked in Git too, so a release adds them there once more.
A visit downloads one set of maps only: about 1.9 MB for the full version, 0.7 MB for the light one, and
none of them where the still picture is shown.

## Rebuilding

The maps were resized and encoded as WebP from the sources above (day 80, lights 84, water 86, clouds
2K 76 and 4K 52). The originals are not in the repository.
