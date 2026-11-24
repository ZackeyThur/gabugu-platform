-- Gabugu keeps the operator's existing image-generation settings when adopting
-- the upstream Gemini gate. Upstream v0.2.19 enables every existing Gemini and
-- Antigravity group here, including deliberately disabled and deleted groups.
-- This fork has not applied this migration in production; retain its filename
-- for the upgrade ledger without granting new image-generation permissions.
SELECT 1;
