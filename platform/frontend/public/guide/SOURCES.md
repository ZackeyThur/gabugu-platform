# Beginner guide media sources

## `ccs-apps.webp`

- Source: [CC Switch release notes v4.0.0 screenshot](https://raw.githubusercontent.com/farion1231/cc-switch/main/assets/release-notes/v4.0.0/apps-zh.png)
- Repository: [farion1231/cc-switch](https://github.com/farion1231/cc-switch)
- Usage: official product screenshot showing the supported application list, compressed to WebP for the guide.
- Version note: the screenshot path is explicitly v4.0.0; this does not claim that v4.0.0 is the latest CC Switch release.
- License: CC Switch is distributed under the repository's MIT license (see the repository `LICENSE` file). This guide keeps the screenshot attribution and original UI context.

## Gabugu demo screenshots

- `keys-create.webp`, `keys-list.webp`, and `usage.webp` are captured locally from the Gabugu frontend with Playwright using synthetic demo fixtures.
- All identities, keys, groups, usage rows, IP addresses, and costs are synthetic demo data. No production credentials, real user configuration, or live model calls are used.
- Output dimensions: `keys-create.webp` 512x768, `keys-list.webp` 1184x1036, `usage.webp` 1184x1570.
- The screenshots show synthetic Anthropic and OpenAI demo groups, masked demo key fragments, synthetic usage, and the import/manual action controls. None contains a usable credential.

## Codex desktop guide refresh

- `keys-create-codex-detail.webp` (472x343) and `keys-create-codex.webp` (512x772) show the key creation form with the OpenAI vendor and synthetic Codex group selected.
- `keys-actions-focused.webp` (300x136; 150x68 CSS pixels at device scale 2) captures the **Use key** and **Import to CCS** buttons on a synthetic Codex key row; `keys-actions-codex.webp` (316x83) is the wider action-row reference.
- `GuideDesktopPreview.vue` is a localized HTML/CSS schematic illustration rather than a live desktop screenshot.
- Official desktop setup references: [Codex app documentation](https://learn.chatgpt.com/docs/app), [desktop quickstart](https://learn.chatgpt.com/docs/quickstart?setup=app), [gateway configuration](https://learn.chatgpt.com/docs/enterprise/connect-to-a-gateway), and [ChatGPT desktop](https://chatgpt.com/download/).
