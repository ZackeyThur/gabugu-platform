package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration251AddsOnlyNullableTokenCampaignAuditColumn(t *testing.T) {
	raw, err := FS.ReadFile("251_token_billing_campaign.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))
	require.Contains(t, sql, "alter table usage_logs add column if not exists token_campaign jsonb")
	require.NotContains(t, sql, "not null")
	require.NotContains(t, sql, "default")
	require.NotContains(t, sql, "update usage_logs")
}
