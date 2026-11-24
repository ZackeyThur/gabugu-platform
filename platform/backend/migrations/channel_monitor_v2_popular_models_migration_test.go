package migrations

import (
	"strings"
	"testing"
)

func TestMigration247RefreshesFactoryChannelMonitorModels(t *testing.T) {
	sql, err := FS.ReadFile("247_channel_monitor_v2_refresh_popular_models.sql")
	if err != nil {
		t.Fatalf("read migration 247: %v", err)
	}

	content := string(sql)
	for _, model := range []string{
		"claude-opus-5-5",
		"claude-sonnet-5-5",
		"gpt-6.1-sol",
		"gpt-5.6-sol",
		"grok-4.7",
		"gemini-4-argon",
		"gemini-3.1-pro-preview",
	} {
		if !strings.Contains(content, `"`+model+`"`) {
			t.Errorf("migration 247 missing current model %q", model)
		}
	}

	if !strings.Contains(content, "AND platforms = '") {
		t.Fatal("migration 247 must only rewrite the exact factory configuration")
	}
	if !strings.Contains(content, "version = version + 1") {
		t.Fatal("migration 247 must bump the configuration version")
	}
	setClause, _, ok := strings.Cut(content, "WHERE id = 1")
	if !ok {
		t.Fatal("migration 247 missing update predicate")
	}
	for _, retired := range []string{"claude-3-5-sonnet", "gpt-4o", "gemini-2.0-flash", "grok-3-mini"} {
		if strings.Contains(setClause, `"`+retired+`"`) {
			t.Errorf("migration 247 still seeds retired factory model %q", retired)
		}
	}
}
