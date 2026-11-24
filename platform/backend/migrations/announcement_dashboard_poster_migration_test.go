package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration254AddsIdempotentAnnouncementDashboardPosterColumns(t *testing.T) {
	raw, err := FS.ReadFile("254_add_announcement_dashboard_poster.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))

	require.Contains(t, sql, "alter table announcements")
	require.Contains(t, sql, "add column if not exists show_on_dashboard boolean not null default false")
	require.Contains(t, sql, "add column if not exists dashboard_link varchar(500) not null default ''")
	require.NotContains(t, sql, "drop column")
	require.NotContains(t, sql, "drop table")
	require.NotContains(t, sql, "usage_logs")
}
