-- Add optional dashboard poster showcase fields to announcements.
-- Safe, additive, idempotent migration (no destructive table rewrite).
ALTER TABLE announcements
    ADD COLUMN IF NOT EXISTS show_on_dashboard BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS dashboard_link VARCHAR(500) NOT NULL DEFAULT '';
