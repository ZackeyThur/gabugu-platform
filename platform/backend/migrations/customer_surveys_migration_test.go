package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration253CreatesIdempotentCustomerSurveyTables(t *testing.T) {
	raw, err := FS.ReadFile("253_customer_surveys.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))

	require.Contains(t, sql, "create table if not exists surveys")
	require.Contains(t, sql, "create table if not exists survey_responses")
	require.Contains(t, sql, "create table if not exists survey_user_states")
	require.Contains(t, sql, "unique (survey_id, user_id)")
	require.NotContains(t, sql, "alter table users")
	require.NotContains(t, sql, "alter table announcements")
	require.NotContains(t, sql, "drop table")
}
