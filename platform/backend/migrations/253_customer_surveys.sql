-- Customer in-app interactive surveys: definitions, responses, and user delivery states.
-- Additive-only migration: creates three standalone tables and leaves all existing tables untouched.

CREATE TABLE IF NOT EXISTS surveys (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    display_mode VARCHAR(20) NOT NULL DEFAULT 'corner',
    questions JSONB NOT NULL DEFAULT '[]'::jsonb,
    targeting JSONB NOT NULL DEFAULT '{}'::jsonb,
    delivery_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    reward_type VARCHAR(20) NOT NULL DEFAULT 'none',
    reward_value NUMERIC(20, 8) NOT NULL DEFAULT 0,
    reward_budget_limit INTEGER NULL,
    reward_issued_count INTEGER NOT NULL DEFAULT 0,
    reward_code_valid_days INTEGER NOT NULL DEFAULT 30,
    starts_at TIMESTAMPTZ NULL,
    ends_at TIMESTAMPTZ NULL,
    created_by BIGINT NULL,
    updated_by BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT surveys_status_check CHECK (status IN ('draft', 'active', 'archived')),
    CONSTRAINT surveys_display_mode_check CHECK (display_mode IN ('corner', 'modal', 'micro')),
    CONSTRAINT surveys_reward_type_check CHECK (reward_type IN ('none', 'balance', 'redeem_code')),
    CONSTRAINT surveys_reward_value_nonnegative CHECK (reward_value >= 0),
    CONSTRAINT surveys_reward_budget_valid CHECK (reward_budget_limit IS NULL OR reward_budget_limit > 0),
    CONSTRAINT surveys_reward_issued_nonnegative CHECK (reward_issued_count >= 0),
    CONSTRAINT surveys_reward_valid_days_positive CHECK (reward_code_valid_days > 0)
);

CREATE INDEX IF NOT EXISTS idx_surveys_status ON surveys (status);
CREATE INDEX IF NOT EXISTS idx_surveys_created_at ON surveys (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_surveys_schedule ON surveys (status, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS survey_responses (
    id BIGSERIAL PRIMARY KEY,
    survey_id BIGINT NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    answers JSONB NOT NULL DEFAULT '{}'::jsonb,
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    reward_status VARCHAR(20) NOT NULL DEFAULT 'none',
    reward_type VARCHAR(20) NOT NULL DEFAULT 'none',
    reward_value NUMERIC(20, 8) NOT NULL DEFAULT 0,
    reward_redeem_code VARCHAR(64) NOT NULL DEFAULT '',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_survey_responses_survey_user UNIQUE (survey_id, user_id),
    CONSTRAINT survey_responses_duration_nonnegative CHECK (duration_seconds >= 0),
    CONSTRAINT survey_responses_reward_status_check CHECK (
        reward_status IN ('none', 'pending', 'granted', 'code_issued', 'budget_exhausted', 'failed')
    )
);

CREATE INDEX IF NOT EXISTS idx_survey_responses_survey_submitted
    ON survey_responses (survey_id, submitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_survey_responses_user_id
    ON survey_responses (user_id);

CREATE TABLE IF NOT EXISTS survey_user_states (
    id BIGSERIAL PRIMARY KEY,
    survey_id BIGINT NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    snooze_until TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_survey_user_states_survey_user UNIQUE (survey_id, user_id),
    CONSTRAINT survey_user_states_status_check CHECK (status IN ('snoozed', 'dismissed', 'submitted'))
);

CREATE INDEX IF NOT EXISTS idx_survey_user_states_user_id
    ON survey_user_states (user_id);
CREATE INDEX IF NOT EXISTS idx_survey_user_states_survey_status
    ON survey_user_states (survey_id, status);
