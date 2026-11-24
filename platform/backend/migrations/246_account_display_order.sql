-- Admin presentation preference only. Existing/new accounts remain unarranged.
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS display_order INTEGER NOT NULL DEFAULT 0;
