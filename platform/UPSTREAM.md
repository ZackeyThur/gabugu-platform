# Gabugu Upstream Attribution & Fork Summary

## Upstream Source

Gabugu Platform is forked and customized from [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api).

- **Pinned upstream release**: `Wei-Shaw/sub2api` `v0.2.23`
- **Upstream tag object**: `ce19a640eab5c4b5ef1827c7c1fcfce49569411a`
- **Upstream tag commit**: `5af9afb588c4b014b3df9db7d4073af1f8ae01b6`
- **Upgrade workflow**: Because upstream and this repository maintain separate Git histories, upstream release deltas are applied cleanly against the pinned upstream tag (`git apply --3way --directory=platform`) rather than merging floating upstream branches.

## Key Fork Customizations

- **Branding & UI**: Gabugu brand visual system (dark orbital landing & auth stage, warm beige light / Starlight Indigo dark workspace theme), simplified cost/usage views, dashboard overview, beginner onboarding guide, studio, arena leaderboard, customer survey, and custom account display ordering.
- **Billing & Accounting**: 1:1 USD/CNY display parity convention, optional `GABUGU_LEGACY_WALLET_COMPAT=true` legacy wallet transaction compatibility, pre-forward pricing completeness checks, dedicated image-count billing, and tiered/scheduled channel pricing support.
- **Gateway & Reliability**: Dual-source scheduling (upstream API lines and managed account pools with independent per-account proxies), reasoning replay normalization, stream error classification, Codex companion page handling, and large Anthropic SSE event handling.
- **Migration Preservation**: All applied additive SQL migrations under `platform/backend/migrations/` are preserved verbatim without post-hoc modification.

## License

The upstream `LGPL-3.0` license ([LICENSE](LICENSE)) and upstream attribution are retained in full.
