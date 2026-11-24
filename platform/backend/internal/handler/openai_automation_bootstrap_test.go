package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const automationBootstrapPrompt = "Review the project and report any important changes."

func codexAutomationBootstrap(automationID, lastRun, prompt string) string {
	return "Automation: Scheduled project review\n" +
		"Automation ID: " + automationID + "\n" +
		"Automation memory: $CODEX_HOME/automations/" + automationID + "/memory.md\n" +
		"Last run: " + lastRun + "\n\n" + prompt
}

func codexAutomationBootstrapBody(t *testing.T, output, callID string) []byte {
	t.Helper()
	return []byte(`{"model":"gpt-5","input":[{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` +
		mustJSON(t, output) + callID + `}]}`)
}

func TestNormalizeCodexAutomationBootstrapSupportedLastRunValues(t *testing.T) {
	tests := []struct {
		name    string
		lastRun string
		crlf    bool
	}{
		{name: "never", lastRun: "never"},
		{name: "timestamp", lastRun: "2026-09-01T02:06:34.536Z (1788228394536)"},
		{name: "crlf", lastRun: "never", crlf: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := codexAutomationBootstrap("wiki-maintenance", tt.lastRun, automationBootstrapPrompt)
			if tt.crlf {
				output = strings.ReplaceAll(output, "\n", "\r\n")
			}
			got, changed := normalizeCodexAutomationBootstrap(codexAutomationBootstrapBody(t, output, ""))
			require.True(t, changed)
			require.Equal(t, "message", gjson.GetBytes(got, "input.0.type").String())
			require.Equal(t, "user", gjson.GetBytes(got, "input.0.role").String())
			require.Equal(t, output, gjson.GetBytes(got, "input.0.content.0.text").String())
			require.False(t, gjson.GetBytes(got, "input.0.call_id").Exists())
		})
	}
}

func TestNormalizeCodexAutomationBootstrapHeartbeat(t *testing.T) {
	output := `<heartbeat><automation_id>wiki</automation_id></heartbeat>`
	got, changed := normalizeCodexAutomationBootstrap(codexAutomationBootstrapBody(t, output, ""))
	require.True(t, changed)
	require.Equal(t, "message", gjson.GetBytes(got, "input.0.type").String())
	require.Equal(t, "user", gjson.GetBytes(got, "input.0.role").String())
	require.Equal(t, output, gjson.GetBytes(got, "input.0.content.0.text").String())
	require.False(t, gjson.GetBytes(got, "input.0.call_id").Exists())

	again, changedAgain := normalizeCodexAutomationBootstrap(got)
	require.False(t, changedAgain)
	require.Equal(t, got, again)
}

func TestNormalizeCodexAutomationBootstrapRejectsUnsafeShapes(t *testing.T) {
	validOutput := codexAutomationBootstrap("wiki", "never", automationBootstrapPrompt)
	tests := []struct {
		name string
		body []byte
	}{
		{
			name: "ordinary missing call id",
			body: []byte(`{"model":"gpt-5","input":[{"type":"function_call_output","namespace":"codex_app","name":"other","output":` + mustJSON(t, validOutput) + `}]}`),
		},
		{
			name: "tui namespace",
			body: []byte(`{"model":"gpt-5","input":[{"type":"function_call_output","namespace":"codex_tui","name":"automation_update","output":` + mustJSON(t, validOutput) + `}]}`),
		},
		{
			name: "valid call id",
			body: codexAutomationBootstrapBody(t, validOutput, `,"call_id":"call-1"`),
		},
		{
			name: "previous response",
			body: []byte(`{"model":"gpt-5","previous_response_id":"resp-1","input":[{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` + mustJSON(t, validOutput) + `}]}`),
		},
		{
			name: "real call context",
			body: []byte(`{"model":"gpt-5","input":[{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` + mustJSON(t, validOutput) + `},{"type":"function_call","call_id":"call-1"}]}`),
		},
		{
			name: "mismatched memory id",
			body: codexAutomationBootstrapBody(t, strings.Replace(validOutput, "/wiki/memory.md", "/other/memory.md", 1), ""),
		},
		{
			name: "unsafe automation id",
			body: codexAutomationBootstrapBody(t, codexAutomationBootstrap("../wiki", "never", automationBootstrapPrompt), ""),
		},
		{
			name: "invalid timestamp",
			body: codexAutomationBootstrapBody(t, codexAutomationBootstrap("wiki", "yesterday", automationBootstrapPrompt), ""),
		},
		{
			name: "mismatched timestamp epoch",
			body: codexAutomationBootstrapBody(t, codexAutomationBootstrap("wiki", "2026-09-01T02:06:34.536Z (1)", automationBootstrapPrompt), ""),
		},
		{
			name: "missing separator",
			body: codexAutomationBootstrapBody(t, strings.Replace(validOutput, "\n\n"+automationBootstrapPrompt, "\n"+automationBootstrapPrompt, 1), ""),
		},
		{
			name: "empty prompt",
			body: codexAutomationBootstrapBody(t, codexAutomationBootstrap("wiki", "never", " \n"), ""),
		},
		{
			name: "mixed missing call id output",
			body: []byte(`{"model":"gpt-5","input":[{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` + mustJSON(t, validOutput) + `},{"type":"computer_call_output","output":"done"}]}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := normalizeCodexAutomationBootstrap(tt.body)
			require.False(t, changed)
			require.Equal(t, tt.body, got)
		})
	}
}

func TestNormalizeCodexAutomationBootstrapRejectsUnsafeHeartbeatShapes(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{name: "arbitrary tool output", output: `<result><automation_id>wiki</automation_id></result>`},
		{name: "root attribute", output: `<heartbeat status="ok"><automation_id>wiki</automation_id></heartbeat>`},
		{name: "namespaced root", output: `<heartbeat xmlns="urn:codex"><automation_id>wiki</automation_id></heartbeat>`},
		{name: "extra child", output: `<heartbeat><automation_id>wiki</automation_id><status>ok</status></heartbeat>`},
		{name: "nested id content", output: `<heartbeat><automation_id><value>wiki</value></automation_id></heartbeat>`},
		{name: "padded id", output: `<heartbeat><automation_id> wiki </automation_id></heartbeat>`},
		{name: "unsafe id", output: `<heartbeat><automation_id>../wiki</automation_id></heartbeat>`},
		{name: "comment", output: `<heartbeat><!-- ok --><automation_id>wiki</automation_id></heartbeat>`},
		{name: "trailing content", output: `<heartbeat><automation_id>wiki</automation_id></heartbeat>extra`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := codexAutomationBootstrapBody(t, tt.output, "")
			got, changed := normalizeCodexAutomationBootstrap(body)
			require.False(t, changed)
			require.Equal(t, body, got)
		})
	}
}

func TestNormalizeCodexAutomationBootstrapPreservesOrderAndIsIdempotent(t *testing.T) {
	output := codexAutomationBootstrap("wiki", "never", automationBootstrapPrompt)
	body := []byte(`{"model":"gpt-5","input":[{"type":"message","role":"user","content":"before"},{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` + mustJSON(t, output) + `},{"type":"message","role":"user","content":"after"}]}`)

	got, changed := normalizeCodexAutomationBootstrap(body)
	require.True(t, changed)
	require.Equal(t, "before", gjson.GetBytes(got, "input.0.content").String())
	require.Equal(t, output, gjson.GetBytes(got, "input.1.content.0.text").String())
	require.Equal(t, "after", gjson.GetBytes(got, "input.2.content").String())

	again, changedAgain := normalizeCodexAutomationBootstrap(got)
	require.False(t, changedAgain)
	require.Equal(t, got, again)
}

func TestNormalizeCodexAutomationBootstrapFullHeartbeat(t *testing.T) {
	const output = `<heartbeat>
  <automation_id>flutter</automation_id>
  <current_time_iso>2026-09-09T00:33:34.775Z</current_time_iso>
  <instructions>
Read the reference index and report changes. Preserve A &amp; B and &lt;tags&gt;.
  </instructions>
</heartbeat>`
	body := codexAutomationBootstrapBody(t, output, "")
	got, changed := normalizeCodexAutomationBootstrap(body)
	require.True(t, changed)
	require.Equal(t, "user", gjson.GetBytes(got, "input.0.role").String())
	require.Equal(t, output, gjson.GetBytes(got, "input.0.content.0.text").String())
	again, changedAgain := normalizeCodexAutomationBootstrap(got)
	require.False(t, changedAgain)
	require.Equal(t, got, again)

	for _, field := range []string{"automation_id", "current_time_iso", "instructions"} {
		duplicate := strings.Replace(output, "</heartbeat>", "<"+field+">duplicate</"+field+"></heartbeat>", 1)
		require.False(t, validCodexAutomationHeartbeat(duplicate), field)
	}
	for _, invalid := range []string{
		strings.Replace(output, "2026-09-09T00:33:34.775Z", "yesterday", 1),
		strings.Replace(output, "<current_time_iso>2026-09-09T00:33:34.775Z</current_time_iso>", "", 1),
		strings.Replace(output, "Read the reference index and report changes. Preserve A &amp; B and &lt;tags&gt;.", " ", 1),
		strings.Replace(output, "<instructions>", `<instructions source="other">`, 1),
		strings.Replace(output, "<automation_id>", `<automation_id kind="cron">`, 1),
		strings.Replace(output, "<current_time_iso>", `<current_time_iso tz="utc">`, 1),
		strings.Replace(output, "<current_time_iso>2026-09-09T00:33:34.775Z</current_time_iso>", "<current_time_iso><nested>2026-09-09T00:33:34.775Z</nested></current_time_iso>", 1),
		strings.Replace(output, "<instructions>", "<!-- comment --><instructions>", 1),
		strings.Replace(output, "</instructions>", "", 1),
		`<?xml version="1.0"?>` + output,
		`<!DOCTYPE heartbeat>` + output,
		output + output,
	} {
		require.False(t, validCodexAutomationHeartbeat(invalid), invalid)
	}
	for _, guarded := range [][]byte{
		codexAutomationBootstrapBody(t, output, `,"call_id":"call-1"`),
		[]byte(strings.Replace(string(body), `"model":"gpt-5"`, `"model":"gpt-5","previous_response_id":"resp-1"`, 1)),
		[]byte(strings.Replace(string(body), `"namespace":"codex_app"`, `"namespace":"other"`, 1)),
	} {
		unchanged, normalized := normalizeCodexAutomationBootstrap(guarded)
		require.False(t, normalized)
		require.Equal(t, guarded, unchanged)
	}
}

func TestNormalizeCodexAutomationBootstrapHeartbeatRawUnescapedInstructions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name         string
		instructions string
	}{
		{
			name:         "unclosed angle placeholder",
			instructions: "Save full report to .agent-logs/<YYYYMMDD-HHmm>-daily-inspection/ without committing.",
		},
		{
			name:         "raw ampersand and comparisons",
			instructions: "Run cmd1 && cmd2 & verify x < 10 and y > 0.",
		},
		{
			name:         "html and self-closing tags",
			instructions: "Check <nested/>, <br/>, and <div class=\"card\">content</div> rendering.",
		},
		{
			name:         "balanced nested instructions tag",
			instructions: "Verify <instructions>inner example</instructions> handling.",
		},
		{
			name:         "unclosed literal instructions tag",
			instructions: "Check raw <instructions> marker in prompt text.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := "<heartbeat>\r\n" +
				"  <automation_id>gabugu-3</automation_id>\r\n" +
				"  <current_time_iso>2026-11-23T06:04:26.356Z</current_time_iso>\r\n" +
				"  <instructions>\r\n" +
				tt.instructions + "\r\n" +
				"  </instructions>\r\n" +
				"</heartbeat>\r\n"
			body := codexAutomationBootstrapBody(t, output, "")

			// Raw request without normalization would fail tool-output validation.
			rawW := httptest.NewRecorder()
			rawC, _ := gin.CreateTestContext(rawW)
			h := &OpenAIGatewayHandler{}
			require.False(t, h.validateFunctionCallOutputRequest(rawC, body, zap.NewNop()))
			require.Equal(t, http.StatusBadRequest, rawW.Code)

			got, changed := normalizeCodexAutomationBootstrap(body)
			require.True(t, changed)
			require.Equal(t, "message", gjson.GetBytes(got, "input.0.type").String())
			require.Equal(t, "user", gjson.GetBytes(got, "input.0.role").String())
			require.Equal(t, output, gjson.GetBytes(got, "input.0.content.0.text").String())
			require.False(t, gjson.GetBytes(got, "input.0.call_id").Exists())

			normW := httptest.NewRecorder()
			normC, _ := gin.CreateTestContext(normW)
			require.True(t, h.validateFunctionCallOutputRequest(normC, got, zap.NewNop()))

			again, changedAgain := normalizeCodexAutomationBootstrap(got)
			require.False(t, changedAgain)
			require.Equal(t, got, again)
		})
	}
}

func TestNormalizeCodexAutomationBootstrapPostCompactionAccumulatedHeartbeats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hb1 := "<heartbeat>\r\n  <automation_id>gabugu</automation_id>\r\n  <current_time_iso>2026-11-22T00:59:57.398Z</current_time_iso>\r\n  <instructions>\r\nSave to .agent-logs/<YYYYMMDD-HHmm>-daily-inspection/\r\n  </instructions>\r\n</heartbeat>\r\n"
	hb2 := "<heartbeat>\r\n  <automation_id>gabugu-3</automation_id>\r\n  <current_time_iso>2026-11-23T06:04:26.356Z</current_time_iso>\r\n  <instructions>\r\nSave to .agent-logs/<YYYYMMDD-HHmm>-product-inspection/\r\n  </instructions>\r\n</heartbeat>\r\n"
	body := []byte(`{"model":"gpt-6.1-terra","input":[` +
		`{"type":"additional_tools","role":"developer","tools":[]},` +
		`{"type":"message","role":"user","content":"earlier summary"},` +
		`{"type":"compaction","encrypted_content":"opaque"},` +
		`{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` + mustJSON(t, hb1) + `},` +
		`{"type":"configuration_update","reasoning":{"effort":"high"}},` +
		`{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` + mustJSON(t, hb2) + `}` +
		`]}`)

	got, changed := normalizeCodexAutomationBootstrap(body)
	require.True(t, changed)
	require.Equal(t, "compaction", gjson.GetBytes(got, "input.2.type").String())
	require.Equal(t, "message", gjson.GetBytes(got, "input.3.type").String())
	require.Equal(t, "user", gjson.GetBytes(got, "input.3.role").String())
	require.Equal(t, hb1, gjson.GetBytes(got, "input.3.content.0.text").String())
	require.Equal(t, "configuration_update", gjson.GetBytes(got, "input.4.type").String())
	require.Equal(t, "message", gjson.GetBytes(got, "input.5.type").String())
	require.Equal(t, "user", gjson.GetBytes(got, "input.5.role").String())
	require.Equal(t, hb2, gjson.GetBytes(got, "input.5.content.0.text").String())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	require.True(t, (&OpenAIGatewayHandler{}).validateFunctionCallOutputRequest(c, got, zap.NewNop()))
}

// The byte precheck must stay a necessary condition: an escaped type or tool
// name is still a candidate, and a body without either is returned untouched
// without the token scan and decode.
func TestNormalizeCodexBootstrapPrecheck(t *testing.T) {
	output := codexAutomationBootstrap("wiki-maintenance", "never", automationBootstrapPrompt)
	escaped := []byte(strings.Replace(string(codexAutomationBootstrapBody(t, output, "")), `"automation_update"`, `"\u0061utomation_update"`, 1))
	got, changed := normalizeCodexAutomationBootstrap(escaped)
	require.True(t, changed)
	require.Equal(t, output, gjson.GetBytes(got, "input.0.content.0.text").String())

	plain := []byte(`{"model":"gpt-5","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"` + strings.Repeat("x", 4096) + `"}]},{"type":"function_call_output","call_id":"call_1","output":"ok"}]}`)
	for name, normalize := range map[string]func([]byte) ([]byte, bool){
		"automation": normalizeCodexAutomationBootstrap,
		"delegation": normalizeCodexDelegationBootstrap,
	} {
		t.Run(name, func(t *testing.T) {
			allocs := testing.AllocsPerRun(10, func() {
				got, changed := normalize(plain)
				if changed || &got[0] != &plain[0] {
					t.Fatal("non-candidate body must be returned unchanged")
				}
			})
			require.Zero(t, allocs)
		})
	}
}
