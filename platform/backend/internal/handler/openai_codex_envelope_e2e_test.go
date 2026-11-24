//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponsesCodexRawTextEnvelopesReachMockUpstream(t *testing.T) {
	hb1 := "<heartbeat>\r\n  <automation_id>gabugu</automation_id>\r\n  <current_time_iso>2026-11-22T00:59:57.398Z</current_time_iso>\r\n  <instructions>\r\nSave report to .agent-logs/<YYYYMMDD-HHmm>-daily-inspection/ && verify.\r\n  </instructions>\r\n</heartbeat>\r\n"
	hb2 := "<heartbeat>\r\n  <automation_id>gabugu-3</automation_id>\r\n  <current_time_iso>2026-11-23T06:04:26.356Z</current_time_iso>\r\n  <instructions>\r\nSave report to .agent-logs/<YYYYMMDD-HHmm>-product-inspection/ & check <br/>.\r\n  </instructions>\r\n</heartbeat>\r\n"
	deleg := "<codex_delegation>\r\n  <source_thread_id>01a1c116-aca6-7d71-8373-cb2aa473987b</source_thread_id>\r\n  <input>Check .agent-logs/<YYYYMMDD-HHmm>-task/ && <div>layout</div></input>\r\n</codex_delegation>\r\n"

	tests := []struct {
		name   string
		stream bool
		input  string
	}{
		{
			name:   "post-compaction accumulated automation heartbeats non-stream",
			stream: false,
			input: `[` +
				`{"type":"additional_tools","role":"developer","tools":[]},` +
				`{"type":"message","role":"user","content":"compacted summary"},` +
				`{"type":"compaction","encrypted_content":"opaque"},` +
				`{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` + mustJSON(t, hb1) + `},` +
				`{"type":"configuration_update","reasoning":{"effort":"high"}},` +
				`{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` + mustJSON(t, hb2) + `}` +
				`]`,
		},
		{
			name:   "post-compaction accumulated automation heartbeats stream",
			stream: true,
			input: `[` +
				`{"type":"compaction","encrypted_content":"opaque"},` +
				`{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` + mustJSON(t, hb2) + `}` +
				`]`,
		},
		{
			name:   "delegation envelope with raw placeholders and html",
			stream: false,
			input: `[` +
				`{"type":"function_call_output","namespace":"codex_app","name":"create_thread","output":` + mustJSON(t, deleg) + `}` +
				`]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, upstream, router, cleanup := newGrokCredentialFailoverHandler(t, "healthy")
			defer cleanup()

			streamLit := "false"
			if tt.stream {
				streamLit = "true"
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/openai/v1/responses", strings.NewReader(`{"model":"grok","stream":`+streamLit+`,"input":`+tt.input+`}`))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			require.NotEmpty(t, upstream.accountHits())
			require.Contains(t, recorder.Body.String(), "resp_healthy")
		})
	}
}

func TestResponsesMalformedCodexEnvelopesStillRejectedWith400(t *testing.T) {
	tests := []struct {
		name   string
		tool   string
		output string
	}{
		{
			name:   "heartbeat with root attribute",
			tool:   "automation_update",
			output: `<heartbeat status="ok"><automation_id>gabugu</automation_id><current_time_iso>2026-11-23T06:04:26.356Z</current_time_iso><instructions>hi</instructions></heartbeat>`,
		},
		{
			name:   "delegation with nested tag in source_thread_id",
			tool:   "create_thread",
			output: `<codex_delegation><source_thread_id><nested>thread-1</nested></source_thread_id><input>work</input></codex_delegation>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, upstream, router, cleanup := newGrokCredentialFailoverHandler(t, "healthy")
			defer cleanup()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/openai/v1/responses", strings.NewReader(
				`{"model":"grok","stream":false,"input":[{"type":"function_call_output","namespace":"codex_app","name":"`+tt.tool+`","output":`+mustJSON(t, tt.output)+`}]}`,
			))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
			require.Contains(t, recorder.Body.String(), "function_call_output requires call_id on HTTP requests")
			require.Empty(t, upstream.accountHits())
		})
	}
}
