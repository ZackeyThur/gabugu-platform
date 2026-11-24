-- Refresh the channel-status V2 model dimensions with the current high-traffic
-- model IDs.  The monitor stores real request models; this list only controls
-- which names get their own dimension instead of being grouped as "other".
--
-- Keep operator edits intact.  The exact JSON predicate matches the factory
-- list seeded by migration 197, so a customized configuration is not replaced.

UPDATE channel_monitor_v2_config
SET
    platforms = $platforms$
[
  {
    "platform": "anthropic",
    "enabled": true,
    "models": [
      "claude-opus-5-5",
      "claude-opus-5",
      "claude-opus-4-8",
      "claude-opus-4-7",
      "claude-opus-4-6",
      "claude-sonnet-5-5",
      "claude-sonnet-5",
      "claude-sonnet-4-6",
      "claude-sonnet-4-5-20250929",
      "claude-haiku-5-5",
      "claude-haiku-4-5-20251001",
      "claude-fable-5-1"
    ]
  },
  {
    "platform": "openai",
    "enabled": true,
    "models": [
      "gpt-6.1-sol",
      "gpt-6-sol",
      "gpt-6-luna",
      "gpt-6-astra",
      "gpt-5.6-sol",
      "gpt-5.6-terra",
      "gpt-5.6-luna",
      "gpt-5.5",
      "gpt-5.4",
      "gpt-5.4-mini",
      "gpt-5.3-codex-spark",
      "gpt-5.2",
      "codex-auto-review",
      "gpt-image-2.5-flare",
      "gpt-image-2.5-sunburst"
    ]
  },
  {
    "platform": "grok",
    "enabled": true,
    "models": [
      "grok-4.7",
      "grok-4.6",
      "grok-4.5",
      "grok-4.3",
      "grok-4.20-0309-reasoning",
      "grok-4.20-0309-non-reasoning",
      "grok-4.20-multi-agent-0309",
      "grok-build-0.1",
      "grok-composer-2.5-fast"
    ]
  },
  {
    "platform": "gemini",
    "enabled": true,
    "models": [
      "gemini-4-argon",
      "gemini-3.5-flash",
      "gemini-3-flash-preview",
      "gemini-3-pro-preview",
      "gemini-3.1-pro-preview",
      "gemini-3.1-flash-image",
      "gemini-2.5-pro",
      "gemini-2.5-flash",
      "gemini-2.5-flash-image"
    ]
  },
  {
    "platform": "kiro",
    "enabled": true,
    "models": [
      "claude-opus-5-5",
      "claude-opus-5",
      "claude-opus-4-8",
      "claude-opus-4-6",
      "claude-sonnet-5-5",
      "claude-sonnet-5",
      "claude-sonnet-4-6",
      "claude-haiku-5-5"
    ]
  },
  {
    "platform": "antigravity",
    "enabled": true,
    "models": [
      "claude-opus-5-5",
      "claude-opus-5",
      "claude-opus-4-8",
      "claude-opus-4-6",
      "claude-sonnet-5-5",
      "claude-sonnet-5",
      "claude-sonnet-4-6",
      "claude-haiku-5-5",
      "gemini-4-argon",
      "gemini-3.1-pro-high",
      "gemini-3.1-pro-low",
      "gemini-3.1-flash-image"
    ]
  }
]
$platforms$::jsonb,
    version = version + 1,
    updated_at = NOW()
WHERE id = 1
  AND platforms = '[{"platform":"anthropic","enabled":true,"models":["claude-opus-5","claude-opus-4-8","claude-opus-4-7","claude-opus-4-6","claude-opus-4.6","claude-opus-4-5","claude-opus-4-1","claude-opus-4","claude-fable-5","claude-sonnet-5","claude-sonnet-4-6","claude-sonnet-4.6","claude-sonnet-4-5","claude-sonnet-4.5","claude-sonnet-4","claude-haiku-4-5","claude-haiku-4.5","claude-3-7-sonnet","claude-3-5-sonnet","claude-3-5-haiku"]},{"platform":"openai","enabled":true,"models":["gpt-5.6-sol","gpt-5.6-terra","gpt-5.6-luna","gpt-5.6","gpt-5.5","gpt-5.4","gpt-5.4-mini","gpt-5.3-codex-spark","gpt-5.2","gpt-5.2-pro","gpt-5","gpt-4.1","gpt-4.1-mini","gpt-4.1-nano","gpt-4o","gpt-4o-mini","gpt-4-turbo","gpt-4","o3","o3-mini","o4-mini","codex-auto-review","gpt-image-2","gpt-image-1"]},{"platform":"grok","enabled":true,"models":["grok-4.5","grok-4.5-latest","grok-4.3","grok-4.20-multi-agent","grok-4.20-0309-reasoning","grok-4.20-0309-non-reasoning","grok-build-0.1","grok-build","grok-3-mini","grok-3-mini-fast","grok-code-fast-1","grok-code-fast"]},{"platform":"gemini","enabled":true,"models":["gemini-2.5-pro","gemini-2.5-flash","gemini-2.5-flash-lite","gemini-2.5-flash-image","gemini-2.0-flash","gemini-2.0-flash-lite","gemini-3-flash","gemini-3-pro-image","gemini-3.1-pro-high","gemini-3.1-pro-low","gemini-3.1-flash-lite","gemini-3.5-flash","gemini-3.5-flash-lite"]},{"platform":"kiro","enabled":true,"models":["claude-opus-4-8","claude-opus-4-6","claude-opus-4-5","claude-sonnet-4-5","claude-sonnet-4-6","claude-haiku-4-5"]},{"platform":"antigravity","enabled":true,"models":["claude-opus-4-6","claude-sonnet-4-5","claude-sonnet-4-6","gemini-2.5-pro","gemini-2.5-flash"]}]'::jsonb;
