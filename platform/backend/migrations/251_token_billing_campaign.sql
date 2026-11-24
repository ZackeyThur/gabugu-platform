-- Additive and nullable: historical logs and older binaries remain valid.
-- Rollback disables the setting and restores the prior binary; preserve this
-- column and all transactions instead of dropping campaign audit evidence.
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS token_campaign JSONB;
