package migrations

import (
	"strings"
	"testing"
)

func TestMigration248NormalizesFactoryModelConfigurationBeforeUpdating(t *testing.T) {
	sql, err := FS.ReadFile("248_channel_monitor_v2_apply_popular_models.sql")
	if err != nil {
		t.Fatalf("read migration 248: %v", err)
	}

	content := string(sql)
	for _, needle := range []string{
		"jsonb_array_elements_text",
		"ORDER BY item->>'platform'",
		"SELECT value FROM current_normalized",
		"SELECT value FROM factory",
		"gpt-6.1-sol",
		"claude-opus-5-5",
	} {
		if !strings.Contains(content, needle) {
			t.Errorf("migration 248 missing %q", needle)
		}
	}
	if strings.Contains(content, "UPDATE schema_migrations") {
		t.Fatal("migration 248 must not mutate the migration ledger")
	}
}
