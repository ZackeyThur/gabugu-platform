package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestChatBufferedTerminalErrorPreservesFallbackMessage(t *testing.T) {
	for _, tc := range []struct{ name, payload, message string }{
		{"top_level", `{"type":"error","message":"input exceeds the context window"}`, "input exceeds the context window"},
		{"default", `{"type":"error","error":{"type":"invalid_request_error"}}`, "Upstream response failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var event apicompat.ResponsesStreamEvent
			require.NoError(t, json.Unmarshal([]byte(tc.payload), &event))
			response := openAICompatTerminalResponse(&event, []byte(tc.payload))
			require.NotNil(t, response)
			require.Equal(t, tc.message, response.Error.Message)
			c, rec, svc, account, resp := chatStreamErrorFixture(t, tc.payload)
			_, err := svc.handleChatBufferedStreamingResponse(resp, c, account, "model", "model", "model", time.Now())
			require.Error(t, err)
			var failover *UpstreamFailoverError
			require.False(t, errors.As(err, &failover))
			require.Contains(t, rec.Body.String(), tc.message)
		})
	}
}

func TestChatBufferedExplicitStatusOverridesInvalidRequestType(t *testing.T) {
	for _, status := range []int{401, 429, 529} {
		for _, field := range []string{"status", "status_code"} {
			for _, eventType := range []string{"error", "response.failed"} {
				t.Run(fmt.Sprintf("%s/%s/%d", eventType, field, status), func(t *testing.T) {
					upstreamError := fmt.Sprintf(`{"type":"invalid_request_error","%s":%d,"code":"provider_busy","message":"provider busy"}`, field, status)
					payload := `{"type":"error","error":` + upstreamError + `}`
					if eventType == "response.failed" {
						payload = `{"type":"response.failed","response":{"status":"failed","error":` + upstreamError + `}}`
					}
					c, _, svc, account, resp := chatStreamErrorFixture(t, payload)
					_, err := svc.handleChatBufferedStreamingResponse(resp, c, account, "model", "model", "model", time.Now())
					var failover *UpstreamFailoverError
					require.ErrorAs(t, err, &failover)
					require.Equal(t, status, failover.StatusCode)
					require.False(t, c.Writer.Written())
					require.Equal(t, status, c.GetInt(OpsUpstreamStatusCodeKey))
				})
			}
		}
	}
}

func TestChatStreamClientErrorClassification(t *testing.T) {
	const imageError = `{"type":"invalid_request_error","code":"invalid_image_format","param":"input[0].content[0].image_url","message":"input_image image_url unsupported image format: unknown"}`
	for _, buffered := range []bool{false, true} {
		for _, eventType := range []string{"error", "response.failed"} {
			for _, priorOutput := range []bool{false, true} {
				if buffered && priorOutput {
					continue
				}
				name := eventType
				if buffered {
					name += "/buffered"
				} else {
					name += "/stream"
				}
				if priorOutput {
					name += "/after_output"
				}
				t.Run(name, func(t *testing.T) {
					payload := `{"type":"error","error":` + imageError + `}`
					if eventType == "response.failed" {
						payload = `{"type":"response.failed","response":{"status":"failed","error":` + imageError + `}}`
					}
					c, rec, svc, account, resp := chatStreamErrorFixture(t, payload)
					if priorOutput {
						resp.Body = io.NopCloser(strings.NewReader("data: {\"type\":\"response.output_text.delta\",\"output_index\":0,\"content_index\":0,\"delta\":\"partial\"}\n\n" + "data: " + payload + "\n\n"))
					}
					var err error
					if buffered {
						_, err = svc.handleChatBufferedStreamingResponse(resp, c, account, "model", "model", "model", time.Now())
					} else {
						_, err = svc.handleChatStreamingResponse(resp, c, account, "model", "model", "model", time.Now(), 0)
					}
					require.Error(t, err)
					var failover *UpstreamFailoverError
					require.False(t, errors.As(err, &failover))
					wantStatus := http.StatusBadRequest
					body := rec.Body.String()
					if priorOutput {
						wantStatus = http.StatusOK
						require.Contains(t, body, "partial")
						for _, line := range strings.Split(body, "\n") {
							if strings.HasPrefix(line, "data: ") && gjson.Get(strings.TrimPrefix(line, "data: "), "error").Exists() {
								body = strings.TrimPrefix(line, "data: ")
							}
						}
					}
					require.Equal(t, wantStatus, rec.Code)
					require.Equal(t, "invalid_request_error", gjson.Get(body, "error.type").String())
					require.Equal(t, "invalid_image_format", gjson.Get(body, "error.code").String())
					require.Equal(t, "input[0].content[0].image_url", gjson.Get(body, "error.param").String())
					require.Equal(t, http.StatusBadRequest, c.GetInt(OpsUpstreamStatusCodeKey))
					streamErrors := GetOpsStreamErrors(c)
					require.Len(t, streamErrors, 1)
					require.Equal(t, buffered, streamErrors[0].NonStream)
					require.True(t, streamErrors[0].CountTowardsSLA)
					v, ok := c.Get(OpsUpstreamErrorsKey)
					require.True(t, ok)
					events := v.([]*OpsUpstreamErrorEvent)
					require.Len(t, events, 1)
					require.Equal(t, http.StatusBadRequest, events[0].UpstreamStatusCode)
				})
			}
		}
	}
}

func TestChatStreamOpsFailureStatusForCapacityAndUnknownErrors(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		status        int
	}{
		{"capacity", `{"type":"error","error":{"type":"invalid_request_error","code":"server_is_overloaded","message":"Selected model is at capacity. Please try again later."}}`, http.StatusServiceUnavailable},
		{"unknown", `{"type":"error","error":{"type":"server_error","code":"mystery_failure","message":"unclassified upstream failure"}}`, http.StatusBadGateway},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := tc.payload
			c, rec, svc, account, resp := chatStreamErrorFixture(t, payload)
			resp.Body = io.NopCloser(strings.NewReader("data: {\"type\":\"response.output_text.delta\",\"output_index\":0,\"content_index\":0,\"delta\":\"partial\"}\n\n" + "data: " + payload + "\n\n"))
			_, err := svc.handleChatStreamingResponse(resp, c, account, "model", "model", "model", time.Now(), 0)
			require.Error(t, err)
			require.Equal(t, http.StatusOK, rec.Code)
			streamErrors := GetOpsStreamErrors(c)
			require.Len(t, streamErrors, 1)
			require.True(t, streamErrors[0].CountTowardsSLA)
			require.Equal(t, tc.status, streamErrors[0].IntendedStatus)
			require.Equal(t, tc.status, c.GetInt(OpsUpstreamStatusCodeKey))
		})
	}
}

func chatStreamErrorFixture(t *testing.T, payload string) (*gin.Context, *httptest.ResponseRecorder, *OpenAIGatewayService, *Account, *http.Response) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	svc := openAIChatPricingTestService(nil, nil)
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: " + payload + "\n\n"))}
	return c, rec, svc, account, resp
}

func TestChatStreamClientErrorPassthroughPriority(t *testing.T) {
	for _, buffered := range []bool{false, true} {
		t.Run(map[bool]string{false: "stream", true: "buffered"}[buffered], func(t *testing.T) {
			c, rec, svc, account, resp := chatStreamErrorFixture(t, `{"type":"error","error":{"type":"invalid_request_error","message":"unsupported image format"}}`)
			rules := &ErrorPassthroughService{}
			rule := newNonFailoverPassthroughRule(400, "unsupported image format", 422, "custom validation message")
			rule.Platforms = []string{PlatformOpenAI}
			rules.setLocalCache([]*model.ErrorPassthroughRule{rule})
			BindErrorPassthroughService(c, rules)
			if buffered {
				_, _ = svc.handleChatBufferedStreamingResponse(resp, c, account, "model", "model", "model", time.Now())
			} else {
				_, _ = svc.handleChatStreamingResponse(resp, c, account, "model", "model", "model", time.Now(), 0)
			}
			require.Equal(t, 422, rec.Code)
			require.Equal(t, "custom validation message", gjson.Get(rec.Body.String(), "error.message").String())
		})
	}
}

func TestChatStreamPassthroughAfterOutputPreservesClientAndUpstreamStatuses(t *testing.T) {
	const payload = `{"type":"error","error":{"type":"invalid_request_error","message":"unsupported image format"}}`
	c, rec, svc, account, resp := chatStreamErrorFixture(t, payload)
	rules := &ErrorPassthroughService{}
	rule := newNonFailoverPassthroughRule(400, "unsupported image format", 422, "custom validation message")
	rule.Platforms = []string{PlatformOpenAI}
	rules.setLocalCache([]*model.ErrorPassthroughRule{rule})
	BindErrorPassthroughService(c, rules)
	resp.Body = io.NopCloser(strings.NewReader("data: {\"type\":\"response.output_text.delta\",\"output_index\":0,\"content_index\":0,\"delta\":\"partial\"}\n\ndata: " + payload + "\n\n"))
	_, err := svc.handleChatStreamingResponse(resp, c, account, "model", "model", "model", time.Now(), 0)
	require.Error(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "custom validation message")
	streamErrors := GetOpsStreamErrors(c)
	require.Len(t, streamErrors, 1)
	require.Equal(t, 422, streamErrors[0].IntendedStatus)
	require.Equal(t, 400, streamErrors[0].UpstreamStatus)
	require.Equal(t, 400, c.GetInt(OpsUpstreamStatusCodeKey))
}
