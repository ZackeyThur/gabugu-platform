package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type upstreamContextTestKey string

func TestGatewayService_StreamingLargeContentPreservesFramingModelUsageAndDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	first := strings.Repeat("a", 2<<20) + "cc_sess_list"
	second := strings.Repeat("b", 2<<20)
	stream := `event: message_start` + "\n" + `data: {"type":"message_start","message":{"model":"mapped","usage":{"input_tokens":13,"cached_tokens":4}}}` + "\n\n" +
		`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"` + first + `"}}` + "\n\n" +
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"` + second + `"}}` + "\n\n" +
		`event: custom` + "\n" + `data: {"type":"custom","model":"provider-custom"}` + "\n\n" +
		`data: {"type":"message_delta","usage":{"output_tokens":19}}` + "\n\n" + `data: {"type":"message_stop"}` + "\n\n"
	for _, disconnect := range []bool{false, true} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
		if disconnect {
			c.Writer = &failWriteResponseWriter{ResponseWriter: c.Writer}
		}
		svc := newStreamingResponseTestGatewayService()
		resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(stream))}
		result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "original", "mapped", false)
		require.NoError(t, err)
		require.Equal(t, disconnect, result.clientDisconnect)
		require.Equal(t, 13, result.usage.InputTokens)
		require.Equal(t, 4, result.usage.CacheReadInputTokens)
		require.Equal(t, 19, result.usage.OutputTokens)
		require.True(t, upstreamResponseModelObserverFromContext(c).conflict)
		if !disconnect {
			body := rec.Body.String()
			require.True(t, strings.Contains(body, strings.ReplaceAll(first, "cc_sess_list", "sessions_list")+`"}}`+"\n\n"))
			require.True(t, strings.Contains(body, second+`"}}`+"\n\n"))
			require.Contains(t, body, `"model":"original"`)
			require.Contains(t, body, `"cache_read_input_tokens":4`)
		}
	}
}

func TestGatewayService_StreamingLargeContentCancellationDoesNotDeadlock(t *testing.T) {
	gin.SetMode(gin.TestMode)
	line := `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"` + strings.Repeat("x", 2<<20) + `"}}` + "\n\n"
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Writer = &failWriteResponseWriter{ResponseWriter: c.Writer}
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(io.MultiReader(strings.NewReader(line), &streamReadCloser{err: context.Canceled}))}
	finished := make(chan struct{})
	var result *streamingResult
	var err error
	go func() {
		defer close(finished)
		result, err = newStreamingResponseTestGatewayService().handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
	}()
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("large content buffer acknowledgement blocked cancellation")
	}
	require.ErrorIs(t, err, context.Canceled)
	require.True(t, result.clientDisconnect)
}

func TestGatewayService_StreamingLargeContentWaitsForCompleteEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	data := `{"type":"content_block_delta","delta":{"text":"` + strings.Repeat("x", 2<<20) + `"}}`
	t.Run("event field after data is terminal", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
		resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("data: " + data + "\nevent: message_stop\n\n"))}
		_, err := newStreamingResponseTestGatewayService().handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(rec.Body.String(), "event: message_stop\ndata: "))
		require.True(t, strings.HasSuffix(rec.Body.String(), "\n\n"))
	})
	t.Run("event field after data is error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
		resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("data: " + data + "\nevent: error\n\n"))}
		_, err := newStreamingResponseTestGatewayService().handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
		var streamErr *sseStreamErrorEventError
		require.ErrorAs(t, err, &streamErr)
		require.True(t, streamErr.RawData == data)
		require.Zero(t, rec.Body.Len())
	})
	t.Run("incomplete event read failure still permits failover", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
		resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(io.MultiReader(strings.NewReader("data: "+data+"\n"), &streamReadCloser{err: io.ErrUnexpectedEOF}))}
		_, err := newStreamingResponseTestGatewayService().handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
		var failover *UpstreamFailoverError
		require.ErrorAs(t, err, &failover)
		require.Zero(t, rec.Body.Len())
		require.False(t, c.Writer.Written())
	})
}

func TestCanonicalLargeAnthropicContentDeltaRejectsAmbiguousKeys(t *testing.T) {
	canonical := `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"x"}}`
	require.True(t, isCanonicalLargeAnthropicContentDelta([]byte(canonical)))
	for _, data := range []string{
		`{"type":"content_block_delta","type":"message_delta","usage":{"output_tokens":19},"delta":{"text":"x"}}`,
		`{"type":"content_block_delta","type":null,"delta":{"text":"x"}}`,
		`{"type":"content_block_delta","TYPE":"message_delta","delta":{"text":"x"}}`,
		`{"type":"content_block_delta","message":null,"delta":{"text":"x"}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","type":"input_json_delta"}}`,
		`{"t\u0079pe":"content_block_delta","delta":{"text":"x"}}`,
	} {
		require.False(t, isCanonicalLargeAnthropicContentDelta([]byte(data)), data)
	}
	keys := gjson.GetBytes([]byte(`{"type":"a","type":"b"}`), "@keys")
	require.Equal(t, `["type","type"]`, keys.Raw, "@keys must retain duplicate names for the fast-path guard")
}

func TestGatewayService_StreamingLargeDuplicateTypeRetainsLegacyUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	text := strings.Repeat("x", 2<<20)
	for _, suffix := range []string{`"type":"message_delta","usage":{"output_tokens":19}`, `"type":null`, `"TYPE":"message_delta"`} {
		data := `{"type":"content_block_delta","delta":{"text":"` + text + `"},` + suffix + `}`
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
		resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("data: " + data + "\n\ndata: [DONE]\n\n"))}
		result, err := newStreamingResponseTestGatewayService().handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
		require.NoError(t, err)
		expectedOutput := 0
		if strings.Contains(suffix, `"usage"`) {
			expectedOutput = 19
		}
		require.Equal(t, expectedOutput, result.usage.OutputTokens)
	}
}

func BenchmarkCanonicalLargeAnthropicContentDelta(b *testing.B) {
	data := []byte(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"` + strings.Repeat("x", 4<<20) + `"}}`)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		if !isCanonicalLargeAnthropicContentDelta(data) {
			b.Fatal("canonical delta rejected")
		}
	}
}

func newStreamingResponseTestGatewayService() *GatewayService {
	return &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				StreamDataIntervalTimeout: 0,
				MaxLineSize:               defaultMaxLineSize,
			},
		},
		rateLimitService: &RateLimitService{},
	}
}

func TestGatewayService_StreamingReusesScannerBufferAndStillParsesUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newStreamingResponseTestGatewayService()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: pr}

	go func() {
		defer func() { _ = pw.Close() }()
		// Minimal SSE event to trigger parseSSEUsage
		_, _ = pw.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":3}}}\n\n"))
		_, _ = pw.Write([]byte("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":7}}\n\n"))
		_, _ = pw.Write([]byte("data: [DONE]\n\n"))
	}()

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
	_ = pr.Close()
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Equal(t, 3, result.usage.InputTokens)
	require.Equal(t, 7, result.usage.OutputTokens)
}

func TestGatewayService_StreamingKeepaliveUsesIdleTimer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newStreamingResponseTestGatewayService()
	svc.cfg.Gateway.StreamKeepaliveInterval = 1

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: pr}

	go func() {
		defer func() { _ = pw.Close() }()
		_, _ = pw.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1}}}\n\n"))
		time.Sleep(1100 * time.Millisecond)
		_, _ = pw.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}()

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
	_ = pr.Close()
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, rec.Body.String(), "event: ping")
}

func TestGatewayService_StreamingKeepaliveUsesNoopDeltaForAffectedClaudeCodeVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newStreamingResponseTestGatewayService()
	svc.cfg.Gateway.StreamKeepaliveInterval = 1

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request.Header.Set("User-Agent", "claude-cli/2.1.198 (external, cli)")

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: pr}

	go func() {
		defer func() { _ = pw.Close() }()
		_, _ = pw.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1}}}\n\n"))
		_, _ = pw.Write([]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"))
		time.Sleep(1100 * time.Millisecond)
		_, _ = pw.Write([]byte("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n"))
		_, _ = pw.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}()

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
	_ = pr.Close()
	require.NoError(t, err)
	require.NotNil(t, result)
	body := rec.Body.String()
	require.Contains(t, body, "event: content_block_delta")
	require.Contains(t, body, `"delta":{"type":"text_delta","text":""}`)
}

func TestGatewayService_StreamingKeepaliveUsesNoopDeltaDuringToolUseForAffectedClaudeCodeVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newStreamingResponseTestGatewayService()
	svc.cfg.Gateway.StreamKeepaliveInterval = 1

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request.Header.Set("User-Agent", "claude-cli/2.1.198 (external, cli)")

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: pr}

	go func() {
		defer func() { _ = pw.Close() }()
		_, _ = pw.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1}}}\n\n"))
		_, _ = pw.Write([]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_1\",\"name\":\"Edit\",\"input\":{}}}\n\n"))
		time.Sleep(1100 * time.Millisecond)
		_, _ = pw.Write([]byte("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":1}\n\n"))
		_, _ = pw.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}()

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
	_ = pr.Close()
	require.NoError(t, err)
	require.NotNil(t, result)
	body := rec.Body.String()
	require.Contains(t, body, "event: content_block_delta")
	require.Contains(t, body, `"index":1`)
	require.Contains(t, body, `"delta":{"type":"input_json_delta","partial_json":""}`)
}

func TestGatewayService_StreamingKeepaliveKeepsPingForOlderClaudeCodeVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newStreamingResponseTestGatewayService()
	svc.cfg.Gateway.StreamKeepaliveInterval = 1

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request.Header.Set("User-Agent", "claude-cli/2.1.187 (external, cli)")

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: pr}

	go func() {
		defer func() { _ = pw.Close() }()
		_, _ = pw.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1}}}\n\n"))
		_, _ = pw.Write([]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"))
		time.Sleep(1100 * time.Millisecond)
		_, _ = pw.Write([]byte("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n"))
		_, _ = pw.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}()

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
	_ = pr.Close()
	require.NoError(t, err)
	require.NotNil(t, result)
	body := rec.Body.String()
	require.Contains(t, body, "event: ping")
	require.NotContains(t, body, `"delta":{"type":"text_delta","text":""}`)
}

func TestDetachUpstreamContextIgnoresClientCancel(t *testing.T) {
	parent, cancel := context.WithCancel(context.WithValue(context.Background(), upstreamContextTestKey("test-key"), "test-value"))
	upstreamCtx, release := detachUpstreamContext(parent)
	defer release()

	cancel()

	require.NoError(t, upstreamCtx.Err())
	require.Equal(t, "test-value", upstreamCtx.Value(upstreamContextTestKey("test-key")))
}
