package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func pageContextItem(name, output string) map[string]any {
	return map[string]any{"type": "function_call_output", "id": "fco_page", "namespace": "agent", "name": name, "output": output}
}

func pageContextRequest(t *testing.T, items ...any) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{"model": "gpt-5", "input": items})
	require.NoError(t, err)
	return body
}

func TestNormalizeCodexAdditionalUntrustedContext(t *testing.T) {
	// AdditionalContextToolOutput derives names from caller-provided context keys.
	// Codex 0.169.0 enables this representation by default (openai/codex#60487).
	for _, name := range []string{"browser_info", "goal_objective", "untrusted_context"} {
		t.Run(name, func(t *testing.T) {
			output := "external context\nline two"
			body := pageContextRequest(t, pageContextItem(name, output),
				map[string]any{"role": "user", "content": "continue"})
			h := &OpenAIGatewayHandler{}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			require.False(t, h.validateFunctionCallOutputRequest(c, body, zap.NewNop()))

			got, changed := normalizeCodexUntrustedContext(body)
			require.True(t, changed)
			w = httptest.NewRecorder()
			c, _ = gin.CreateTestContext(w)
			require.True(t, h.validateFunctionCallOutputRequest(c, got, zap.NewNop()))
			require.False(t, c.Writer.Written())
			item := gjson.GetBytes(got, "input.0")
			require.Equal(t, "message", item.Get("type").String())
			require.Equal(t, "user", item.Get("role").String())
			require.Equal(t, "<external_"+name+">"+output+"</external_"+name+">", item.Get("content.0.text").String())
			require.False(t, item.Get("call_id").Exists())
			require.Equal(t, "continue", gjson.GetBytes(got, "input.1.content").String())
			again, changed := normalizeCodexUntrustedContext(got)
			require.False(t, changed)
			require.Equal(t, got, again)
		})
	}
}

func TestNormalizeCodexAdditionalUntrustedContextPreservesRealAndMalformedOutputs(t *testing.T) {
	for _, name := range []string{"browser_info", "goal_objective", "untrusted_context"} {
		for _, override := range []map[string]any{
			{"call_id": "call_real"},
			{"call_id": nil},
			{"call_id": 42},
			{"namespace": "other"},
			{"namespace": nil},
			{"namespace": 42},
			{"output": []any{map[string]any{"type": "encrypted_content", "encrypted_content": "sealed"}}},
			{"output": " "},
			{"output": nil},
		} {
			item := pageContextItem(name, "context")
			for key, value := range override {
				item[key] = value
			}
			body := pageContextRequest(t, item)
			got, changed := normalizeCodexUntrustedContext(body)
			require.False(t, changed, "name=%s override=%v", name, override)
			require.Equal(t, body, got)
		}
		item := pageContextItem(name, "context")
		delete(item, "namespace")
		body := pageContextRequest(t, item)
		got, changed := normalizeCodexUntrustedContext(body)
		require.False(t, changed)
		require.Equal(t, body, got)
	}
	// Recognized external context cannot make an unrelated tool result valid.
	body := pageContextRequest(t, pageContextItem("browser_info", "tab"),
		pageContextItem("read_file", "result"))
	got, changed := normalizeCodexUntrustedContext(body)
	require.True(t, changed)
	h := &OpenAIGatewayHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	require.False(t, h.validateFunctionCallOutputRequest(c, got, zap.NewNop()))
	require.Equal(t, 400, w.Code)

	// Keep historical calls, prior response and exact numeric metadata intact.
	body = []byte(`{"model":"gpt-5","previous_response_id":"resp_history","metadata":{"large":9007199254740993},"input":[{"type":"function_call","name":"read_file","call_id":"call_real","arguments":"{}"},{"type":"function_call_output","call_id":"call_real","output":"result"},{"type":"function_call_output","namespace":"agent","name":"goal_objective","output":"finish"}]}`)
	got, changed = normalizeCodexUntrustedContext(body)
	require.True(t, changed)
	require.Equal(t, "resp_history", gjson.GetBytes(got, "previous_response_id").String())
	require.Equal(t, "9007199254740993", gjson.GetBytes(got, "metadata.large").Raw)
	for _, index := range []string{"input.0", "input.1"} {
		require.JSONEq(t, gjson.GetBytes(body, index).Raw, gjson.GetBytes(got, index).Raw)
	}
}

func TestNormalizeCodexAdditionalUntrustedContextBlankCallIDsAndMixedInput(t *testing.T) {
	names := []string{"browser_info", "goal_objective", "untrusted_context"}
	for _, callID := range []string{"", " \t\n"} {
		items := make([]any, 0, len(names))
		for _, name := range names {
			item := pageContextItem(name, "context for "+name)
			item["call_id"] = callID
			delete(item, "id")
			items = append(items, item)
		}
		got, changed := normalizeCodexUntrustedContext(pageContextRequest(t, items...))
		require.True(t, changed)
		require.Len(t, gjson.GetBytes(got, "input").Array(), len(names))
		for i, item := range gjson.GetBytes(got, "input").Array() {
			require.Equal(t, "user", item.Get("role").String())
			require.Equal(t, "<external_"+names[i]+">context for "+names[i]+"</external_"+names[i]+">", item.Get("content.0.text").String())
		}
	}
}

func TestNormalizeCodexAdditionalUntrustedContextRejectsDuplicateMembers(t *testing.T) {
	for _, name := range []string{"browser_info", "goal_objective", "untrusted_context"} {
		for _, body := range []string{
			`{"input":[{"type":"function_call_output","namespace":"agent","name":"` + name + `","call_id":"real","call_id":"","output":"context"}]}`,
			`{"input":[{"type":"function_call_output","namespace":"agent","name":"` + name + `","output":"first","output":"second"}]}`,
		} {
			got, changed := normalizeCodexUntrustedContext([]byte(body))
			require.False(t, changed)
			require.Equal(t, body, string(got))
		}
	}
}
