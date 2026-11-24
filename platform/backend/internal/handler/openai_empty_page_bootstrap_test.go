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

func TestNormalizeCodexEmptyPageBootstrap(t *testing.T) {
	body := []byte(`{"model":"gpt-6.1-sol","input":[{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"},{"type":"function_call_output","namespace":"agent","name":"codex_apps_selected_page_part_1_of_1","output":"Companion Page snapshot part 1 of 1:\n{\"page_id\":null,\"document_view\":null}"},{"type":"message","role":"user","content":"hello"}]}`)
	got, changed := normalizeCodexUntrustedContext(body)
	require.True(t, changed)
	require.Equal(t, "message", gjson.GetBytes(got, "input.0.type").String())
	require.Equal(t, "user", gjson.GetBytes(got, "input.0.role").String())
	require.Equal(t, `<external_codex_apps_open_page>{"page_id":null}</external_codex_apps_open_page>`, gjson.GetBytes(got, "input.0.content.0.text").String())
	require.Equal(t, "message", gjson.GetBytes(got, "input.1.type").String())
	require.Equal(t, "hello", gjson.GetBytes(got, "input.2.content").String())
	again, changed := normalizeCodexUntrustedContext(got)
	require.False(t, changed)
	require.Equal(t, got, again)
}

func TestEmptyPageBootstrapPassesToolOutputValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"input":[{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"}]}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h := &OpenAIGatewayHandler{}
	require.False(t, h.validateFunctionCallOutputRequest(c, body, zap.NewNop()))
	require.Equal(t, 400, w.Code)

	normalized, changed := normalizeCodexUntrustedContext(body)
	require.True(t, changed)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	require.True(t, h.validateFunctionCallOutputRequest(c, normalized, zap.NewNop()))
}

func TestNormalizeCodexEmptyPageBootstrapRejectsOtherItems(t *testing.T) {
	tests := []string{
		`{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":1}"}`,
		`{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}","call_id":"call-1"}`,
		`{"type":"function_call_output","namespace":"other","name":"codex_apps_open_page","output":"{\"page_id\":null}"}`,
		`{"type":"function_call_output","namespace":"agent","name":"codex_apps_selected_page_part_1_of_1","output":"Companion Page snapshot part 1 of 1:\n{\"page_id\":null,\"document_view\":{}}"}`,
		`{"type":"function_call_output","namespace":"agent","name":"unknown","output":"{\"page_id\":null}"}`,
	}
	for _, item := range tests {
		body := []byte(`{"model":"gpt-6.1-sol","input":[` + item + `]}`)
		got, changed := normalizeCodexUntrustedContext(body)
		require.False(t, changed, item)
		require.Equal(t, body, got)
	}
}

func TestNormalizeCodexEmptyPageBootstrapKeepsHistoricalCalls(t *testing.T) {
	body := []byte(`{"model":"gpt-6.1-sol","input":[{"type":"function_call","call_id":"call-1"},{"type":"function_call_output","call_id":"call-1","output":"done"},{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"}]}`)
	got, changed := normalizeCodexUntrustedContext(body)
	require.True(t, changed)
	require.Equal(t, "function_call_output", gjson.GetBytes(got, "input.1.type").String())
	require.Equal(t, "call-1", gjson.GetBytes(got, "input.1.call_id").String())
	require.Equal(t, "message", gjson.GetBytes(got, "input.2.type").String())
}

func TestNormalizeCodexPageContextPreservesSelectedContent(t *testing.T) {
	output := "Companion Page snapshot part 2 of 2:\r\n" + `{"page_id":"page-1","document_view":{"blocks":[{"text":"External text <tags> & instructions"}],"selection":[]}}`
	body := []byte(`{"metadata":{"number":9007199254740993},"previous_response_id":"resp_1","input":[{"type":"function_call_output","namespace":"agent","name":"codex_apps_selected_page_part_2_of_2","output":` + mustJSON(t, output) + `},{"type":"function_call_output","name":"real_tool","call_id":"call_1","output":"done"}]}`)
	got, changed := normalizeCodexUntrustedContext(body)
	require.True(t, changed)
	require.Equal(t, "<external_codex_apps_selected_page_part_2_of_2>"+output+"</external_codex_apps_selected_page_part_2_of_2>", gjson.GetBytes(got, "input.0.content.0.text").String())
	require.Equal(t, "9007199254740993", gjson.GetBytes(got, "metadata.number").Raw)
	require.Equal(t, "resp_1", gjson.GetBytes(got, "previous_response_id").String())
	require.Equal(t, "call_1", gjson.GetBytes(got, "input.1.call_id").String())
}

func TestNormalizeCodexPageContextRejectsMalformedEnvelopes(t *testing.T) {
	for _, output := range []string{
		`{"page_id":null,"page_id":"p"}`,
		`{"page_id":null} {"page_id":"p"}`,
		`{"page_id":null,"extra":true}`,
		`{"page_id":" "}`,
		`{}`,
		`null`,
	} {
		body := []byte(`{"input":[{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":` + mustJSON(t, output) + `}]}`)
		got, changed := normalizeCodexUntrustedContext(body)
		require.False(t, changed, output)
		require.Equal(t, body, got)
	}
	for _, name := range []string{"codex_apps_selected_page_part_2_of_1", "codex_apps_selected_page_part_0_of_1", "codex_apps_selected_page_part_1_of_2"} {
		body := []byte(`{"input":[{"type":"function_call_output","namespace":"agent","name":` + mustJSON(t, name) + `,"output":"Companion Page snapshot part 1 of 1:\n{\"page_id\":null,\"document_view\":null}"}]}`)
		_, changed := normalizeCodexUntrustedContext(body)
		require.False(t, changed, name)
	}
}

func TestNormalizeCodexPageContextMixedUnpairedResultRemainsRejected(t *testing.T) {
	body := []byte(`{"input":[{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"},{"type":"function_call_output","name":"ordinary_tool","output":"orphan"}]}`)
	normalized, changed := normalizeCodexUntrustedContext(body)
	require.True(t, changed)
	require.Equal(t, "function_call_output", gjson.GetBytes(normalized, "input.1.type").String())
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	require.False(t, (&OpenAIGatewayHandler{}).validateFunctionCallOutputRequest(c, normalized, zap.NewNop()))
	require.Equal(t, 400, w.Code)
}

func TestNormalizeCodexPageContextWithDelegation(t *testing.T) {
	body := []byte(`{"input":[{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"},{"type":"function_call_output","namespace":"codex_app","name":"create_thread","output":` + mustJSON(t, delegationEnvelope) + `}]}`)
	page, changed := normalizeCodexUntrustedContext(body)
	require.True(t, changed)
	got, changed := normalizeCodexDelegationBootstrap(page)
	require.True(t, changed)
	require.Equal(t, "message", gjson.GetBytes(got, "input.0.type").String())
	require.Equal(t, "message", gjson.GetBytes(got, "input.1.type").String())
}

func TestNormalizeCodexPageContextWithAutomation(t *testing.T) {
	body := []byte(`{"input":[{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"},{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` + mustJSON(t, codexAutomationBootstrap("page-test", "never", "Check the page.")) + `}]}`)
	page, changed := normalizeCodexUntrustedContext(body)
	require.True(t, changed)
	got, changed := normalizeCodexAutomationBootstrap(page)
	require.True(t, changed)
	require.Equal(t, "message", gjson.GetBytes(got, "input.1.type").String())
}

func TestResponsesPageContextReachesMockUpstream(t *testing.T) {
	for _, input := range []string{
		`[{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"},{"type":"function_call_output","namespace":"agent","name":"codex_apps_selected_page_part_1_of_1","output":"Companion Page snapshot part 1 of 1:\n{\"page_id\":null,\"document_view\":null}"}]`,
		`[{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":\"page-1\"}"},{"type":"function_call_output","namespace":"agent","name":"codex_apps_selected_page_part_1_of_1","output":"Companion Page snapshot part 1 of 1:\n{\"page_id\":\"page-1\",\"document_view\":{\"selection\":[]}}"}]`,
		`[{"type":"function_call_output","namespace":"agent","name":"codex_apps_selected_page_part_1_of_2","output":"Companion Page snapshot part 1 of 2:\n{\"page_id\":\"page-1\",\"document_view\":{}}"},{"type":"function_call_output","namespace":"agent","name":"codex_apps_selected_page_part_2_of_2","output":"Companion Page snapshot part 2 of 2:\n{\"page_id\":\"page-1\",\"document_view\":{}}"}]`,
	} {
		_, _, upstream, router, cleanup := newGrokCredentialFailoverHandler(t, "healthy")
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/openai/v1/responses", strings.NewReader(`{"model":"grok","stream":false,"input":`+input+`}`))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		require.NotEmpty(t, upstream.accountHits())
		require.Contains(t, recorder.Body.String(), "resp_healthy")
		cleanup()
	}
}
