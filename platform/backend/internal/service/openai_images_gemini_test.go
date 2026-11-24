package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGeminiImagesClosedModelListAndURL(t *testing.T) {
	for _, model := range []string{"gemini-3-pro-image", "monkey-image-pro", "monkey-image-flash 2", "nano-banana-2.1"} {
		require.True(t, isGeminiNativeImageModel(model))
	}
	for _, model := range []string{"gemini-unknown-image", "gpt-image-2", "../monkey-image-pro", "monkey-image-pro?key=secret", "monkey-image-pro/extra"} {
		require.False(t, isGeminiNativeImageModel(model))
		_, err := buildGeminiImagesURL("https://relay.example/v1", model)
		require.Error(t, err)
	}
	for _, base := range []string{"https://relay.example/prefix/v1/", "https://relay.example/prefix/v1beta", "https://relay.example/prefix"} {
		got, err := buildGeminiImagesURL(base, "monkey-image-flash 2")
		require.NoError(t, err)
		require.Equal(t, "https://relay.example/prefix/v1beta/models/monkey-image-flash%202:generateContent", got)
	}
	for _, base := range []string{"https://relay.example/v1?key=secret", "https://user:pass@relay.example/v1", "file:///tmp/image"} {
		_, err := buildGeminiImagesURL(base, "monkey-image-pro")
		require.Error(t, err)
	}
}

func TestBuildGeminiImagesBodyPreservesNative4KAndUploadRatio(t *testing.T) {
	encoded := encodeOpenAIImageTestJPEG(t, 216, 384)
	raw, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	parsed := &OpenAIImagesRequest{Endpoint: openAIImagesEditsEndpoint, Model: "gemini-3-pro-image", Prompt: "保留所有中文，仅降噪", N: 1, Size: "4K", Uploads: []OpenAIImagesUpload{{Data: raw, ContentType: "application/octet-stream"}}}
	body, err := buildGeminiImagesBody(parsed)
	require.NoError(t, err)
	require.Equal(t, "4K", gjson.GetBytes(body, "generationConfig.imageConfig.imageSize").String())
	require.Equal(t, "9:16", gjson.GetBytes(body, "generationConfig.imageConfig.aspectRatio").String())
	require.Equal(t, "IMAGE", gjson.GetBytes(body, "generationConfig.responseModalities.0").String())
	require.Equal(t, parsed.Prompt, gjson.GetBytes(body, "contents.0.parts.0.text").String())
	require.Equal(t, "image/jpeg", gjson.GetBytes(body, "contents.0.parts.1.inlineData.mimeType").String())
	require.Equal(t, encoded, gjson.GetBytes(body, "contents.0.parts.1.inlineData.data").String())
	parsed.Size = "3840x2160"
	body, err = buildGeminiImagesBody(parsed)
	require.NoError(t, err)
	require.Equal(t, "16:9", gjson.GetBytes(body, "generationConfig.imageConfig.aspectRatio").String())
	require.Equal(t, "4K", gjson.GetBytes(body, "generationConfig.imageConfig.imageSize").String())
}

func TestBuildGeminiImagesBodyGenerationAndRejectsUnsupportedOptions(t *testing.T) {
	base := OpenAIImagesRequest{Endpoint: openAIImagesGenerationsEndpoint, Model: "gemini-3-pro-image", Prompt: "poster", N: 1, Size: "2K"}
	body, err := buildGeminiImagesBody(&base)
	require.NoError(t, err)
	require.Equal(t, 1, int(gjson.GetBytes(body, "contents.0.parts.#").Int()))
	require.False(t, gjson.GetBytes(body, "generationConfig.imageConfig.aspectRatio").Exists())
	zero := 0
	for _, tc := range []struct {
		name   string
		change func(*OpenAIImagesRequest)
	}{
		{"n", func(r *OpenAIImagesRequest) { r.N = 2 }},
		{"stream", func(r *OpenAIImagesRequest) { r.Stream = true }},
		{"mask", func(r *OpenAIImagesRequest) { r.HasMask = true }},
		{"url", func(r *OpenAIImagesRequest) { r.InputImageURLs = []string{"http://169.254.169.254/latest/meta-data"} }},
		{"response_format", func(r *OpenAIImagesRequest) { r.ResponseFormat = "url" }},
		{"quality", func(r *OpenAIImagesRequest) { r.Quality = "hd" }},
		{"background", func(r *OpenAIImagesRequest) { r.Background = "transparent" }},
		{"output_format", func(r *OpenAIImagesRequest) { r.OutputFormat = "png" }},
		{"compression", func(r *OpenAIImagesRequest) { r.OutputCompression = &zero }},
		{"partial", func(r *OpenAIImagesRequest) { r.PartialImages = &zero }},
		{"moderation", func(r *OpenAIImagesRequest) { r.Moderation = "low" }},
		{"fidelity", func(r *OpenAIImagesRequest) { r.InputFidelity = "high" }},
		{"style", func(r *OpenAIImagesRequest) { r.Style = "vivid" }},
		{"bad_size", func(r *OpenAIImagesRequest) { r.Size = "8K" }},
		{"empty_prompt", func(r *OpenAIImagesRequest) { r.Prompt = "" }},
		{"missing_image", func(r *OpenAIImagesRequest) { r.Endpoint = openAIImagesEditsEndpoint }},
		{"fake_image", func(r *OpenAIImagesRequest) {
			r.Uploads = []OpenAIImagesUpload{{Data: []byte("not image"), ContentType: "image/png"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := base
			tc.change(&r)
			_, err := buildGeminiImagesBody(&r)
			var invalid *OpenAIImagesUpstreamError
			require.ErrorAs(t, err, &invalid)
			require.Equal(t, http.StatusBadRequest, invalid.StatusCode)
		})
	}
}

func TestConvertGeminiImagesResponseUsesRealBytesAndUsage(t *testing.T) {
	encoded := encodeOpenAIImageTestJPEG(t, 216, 384)
	response := fmt.Sprintf(`{"candidates":[{"content":{"parts":[{"text":"done"},{"inlineData":{"mimeType":"image/jpeg","data":%q}},{"inlineData":{"mimeType":"image/jpeg","data":%q}}]}}],"usageMetadata":{"promptTokenCount":120,"candidatesTokenCount":500,"cachedContentTokenCount":20,"promptTokensDetails":[{"modality":"IMAGE","tokenCount":90}],"candidatesTokensDetails":[{"modality":"IMAGE","tokenCount":480}]}}`, encoded, encoded)
	body, usage, sizes, err := convertGeminiImagesResponse([]byte(response), &OpenAIImagesRequest{Model: "gemini-3-pro-image"})
	require.NoError(t, err)
	require.Equal(t, []string{"216x384"}, sizes)
	require.Equal(t, int64(1), gjson.GetBytes(body, "data.#").Int(), "duplicate image parts must not produce duplicate billable outputs")
	require.Equal(t, encoded, gjson.GetBytes(body, "data.0.b64_json").String())
	require.Equal(t, "jpeg", gjson.GetBytes(body, "output_format").String())
	require.Equal(t, "gemini-3-pro-image", gjson.GetBytes(body, "model").String())
	require.Equal(t, OpenAIUsage{InputTokens: 120, OutputTokens: 500, CacheReadInputTokens: 20, ImageInputTokens: 90, ImageOutputTokens: 480}, usage)
	for _, invalid := range []string{`not json`, `{"error":{"message":"failed"}}`, `{"candidates":[{"content":{"parts":[{"text":"I cannot do this"}]}}]}`, `{"candidates":[{"content":{"parts":[{"fileData":{"mimeType":"image/png","fileUri":"http://127.0.0.1/private"}}]}}]}`, `{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"garbage"}}]}}]}`} {
		_, _, _, err := convertGeminiImagesResponse([]byte(invalid), &OpenAIImagesRequest{})
		require.Error(t, err)
	}
}

func TestConvertGeminiImagesResponseRejectsTruncatedAndOversizedImages(t *testing.T) {
	valid, err := base64.StdEncoding.DecodeString(encodeOpenAIImageTestPNG(t, 1, 1))
	require.NoError(t, err)
	// Signature + complete, CRC-valid IHDR is enough for DecodeConfig but has
	// no IDAT or IEND. Dimension tests mutate only this tiny header; they never
	// allocate or encode a large raster.
	require.GreaterOrEqual(t, len(valid), 33)
	for _, tc := range []struct {
		name          string
		width, height uint32
		wantError     string
	}{
		{"33_byte_truncated_png", 1, 1, "failed full decode"},
		{"over_32_megapixels", 5700, 5700, "dimension or pixel limit"},
		{"over_8192_edge", 8193, 1, "dimension or pixel limit"},
		{"native_4k_header_needs_complete_pixels", 3072, 5504, "failed full decode"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			header := append([]byte(nil), valid[:33]...)
			binary.BigEndian.PutUint32(header[16:20], tc.width)
			binary.BigEndian.PutUint32(header[20:24], tc.height)
			binary.BigEndian.PutUint32(header[29:33], crc32.ChecksumIEEE(header[12:29]))
			encoded := base64.StdEncoding.EncodeToString(header)
			_, _, err := validateGeminiImageOutput(encoded)
			require.ErrorContains(t, err, tc.wantError)
			response := fmt.Sprintf(`{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":%q}}]}}]}`, encoded)
			body, _, sizes, err := convertGeminiImagesResponse([]byte(response), &OpenAIImagesRequest{N: 1})
			var imageErr *OpenAIImagesUpstreamError
			require.ErrorAs(t, err, &imageErr)
			require.Equal(t, http.StatusBadGateway, imageErr.StatusCode)
			require.Nil(t, body)
			require.Nil(t, sizes, "invalid bytes must never produce billable outputs")
		})
	}
}

func TestConvertGeminiImagesResponseRejectsMultipleDistinctImages(t *testing.T) {
	first := encodeOpenAIImageTestJPEG(t, 1, 1)
	second := encodeOpenAIImageTestPNG(t, 1, 1)
	response := fmt.Sprintf(`{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/jpeg","data":%q}},{"inlineData":{"mimeType":"image/png","data":%q}}]}}]}`, first, second)
	body, _, sizes, err := convertGeminiImagesResponse([]byte(response), &OpenAIImagesRequest{N: 1})
	require.ErrorContains(t, err, "multiple distinct images for n=1")
	require.Nil(t, body)
	require.Nil(t, sizes)
}

func TestForwardOpenAIImagesGeminiPreservesProxyAuthenticationAndBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	encoded := encodeOpenAIImageTestJPEG(t, 216, 384)
	response, err := json.Marshal(map[string]any{"modelVersion": "gemini-3.0-pro-image-portrait-4k", "candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"inlineData": map[string]string{"mimeType": "image/jpeg", "data": encoded}}}}}}, "usageMetadata": map[string]int{"promptTokenCount": 30, "candidatesTokenCount": 500}})
	require.NoError(t, err)
	upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"X-Request-Id": []string{"gemini-img-1"}}, Body: io.NopCloser(bytes.NewReader(response))}}
	svc := withTestGatewayPricing(&OpenAIGatewayService{httpUpstream: upstream})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, openAIImagesGenerationsEndpoint, nil)
	c.Set("api_key", &APIKey{ID: 42, Group: &Group{ImageCountBilling: true}})
	proxyID := int64(2)
	account := &Account{ID: 65, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, ProxyID: &proxyID, Proxy: &Proxy{Protocol: "http", Host: "127.0.0.1", Port: 19099}, Credentials: map[string]any{"api_key": "test-upstream-secret", "base_url": "https://relay.example/v1"}}
	parsed := &OpenAIImagesRequest{Endpoint: openAIImagesGenerationsEndpoint, Model: "gemini-3-pro-image", Prompt: "poster", N: 1, Size: "4K"}
	result, err := svc.forwardOpenAIImagesGemini(context.Background(), c, account, parsed, "monkey-image-pro")
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:19099", upstream.lastProxyURL)
	require.Equal(t, "test-upstream-secret", upstream.lastReq.Header.Get("x-goog-api-key"))
	require.Empty(t, upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "https://relay.example/v1beta/models/monkey-image-pro:generateContent", upstream.lastReq.URL.String())
	require.Equal(t, "4K", gjson.GetBytes(upstream.lastBody, "generationConfig.imageConfig.imageSize").String())
	require.Equal(t, "gemini-3-pro-image", result.Model)
	require.Equal(t, "monkey-image-pro", result.UpstreamModel)
	require.Equal(t, "gemini-3.0-pro-image-portrait-4k", result.UpstreamResponseModel)
	require.Equal(t, 1, result.ImageCount)
	require.True(t, result.ImageEndpoint)
	require.Equal(t, "4K", result.ImageSize)
	require.Equal(t, "4K", result.ImageInputSize)
	require.Equal(t, []string{"216x384"}, result.ImageOutputSizes)
	require.Equal(t, 30, result.Usage.InputTokens)
	require.Equal(t, 500, result.Usage.OutputTokens)
	require.Equal(t, "gemini-img-1", result.RequestID)
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestForwardOpenAIImagesGeminiRejectsBeforeRequestAndMalformedSuccess(t *testing.T) {
	for _, tc := range []struct {
		name         string
		stream       bool
		upstreamBody string
		want         int
		calls        int
	}{
		{"unsupported_stream", true, `{}`, http.StatusBadRequest, 0},
		{"empty_image", false, `{"candidates":[]}`, http.StatusBadGateway, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(tc.upstreamBody))}}
			svc := withTestGatewayPricing(&OpenAIGatewayService{httpUpstream: upstream})
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, openAIImagesGenerationsEndpoint, nil)
			account := &Account{ID: 65, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "test-secret", "base_url": "https://relay.example/v1"}}
			_, err := svc.forwardOpenAIImagesGemini(context.Background(), c, account, &OpenAIImagesRequest{Endpoint: openAIImagesGenerationsEndpoint, Model: "gemini-3-pro-image", Prompt: "test", N: 1, Stream: tc.stream}, "monkey-image-pro")
			var imageErr *OpenAIImagesUpstreamError
			require.True(t, errors.As(err, &imageErr))
			require.Equal(t, tc.want, rec.Code)
			require.True(t, c.Writer.Written())
			require.Len(t, upstream.requests, tc.calls)
		})
	}
}

func TestForwardOpenAIImagesGeminiMultipartDisconnectStillReturnsBillableResult(t *testing.T) {
	encoded := encodeOpenAIImageTestJPEG(t, 90, 160)
	imageBytes, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	for name, value := range map[string]string{"model": "gemini-3-pro-image", "prompt": "降噪，保留文字", "size": "4K", "response_format": "b64_json"} {
		require.NoError(t, writer.WriteField(name, value))
	}
	part, err := writer.CreateFormFile("image", "poster.jpg")
	require.NoError(t, err)
	_, err = part.Write(imageBytes)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	c, _ := gin.CreateTestContext(&geminiImagesDisconnectedWriter{header: http.Header{}})
	c.Request = httptest.NewRequest(http.MethodPost, openAIImagesEditsEndpoint, bytes.NewReader(requestBody.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("api_key", &APIKey{ID: 42, Group: &Group{ImageCountBilling: true}})
	upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/jpeg","data":%q}}]}}]}`, encoded)))}}
	svc := withTestGatewayPricing(&OpenAIGatewayService{httpUpstream: upstream})
	parsed, err := svc.ParseOpenAIImagesRequest(c, requestBody.Bytes())
	require.NoError(t, err)
	account := &Account{ID: 65, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "test-secret", "base_url": "https://relay.example/v1"}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := svc.forwardOpenAIImagesGemini(ctx, c, account, parsed, "monkey-image-pro")
	require.ErrorIs(t, err, io.ErrClosedPipe)
	require.NotNil(t, result)
	require.Equal(t, 1, result.ImageCount)
	require.True(t, result.ClientDisconnect)
	require.NoError(t, upstream.lastReq.Context().Err())
	require.Equal(t, "9:16", gjson.GetBytes(upstream.lastBody, "generationConfig.imageConfig.aspectRatio").String())
	require.Equal(t, encoded, gjson.GetBytes(upstream.lastBody, "contents.0.parts.1.inlineData.data").String())
}

type geminiImagesDisconnectedWriter struct{ header http.Header }

func (w *geminiImagesDisconnectedWriter) Header() http.Header       { return w.header }
func (w *geminiImagesDisconnectedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func (w *geminiImagesDisconnectedWriter) WriteHeader(int)           {}
func (w *geminiImagesDisconnectedWriter) Flush()                    {}
