package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	_ "golang.org/x/image/webp"
)

const (
	// The observed native 4K result is 3072x5504 (16.9MP). 32MP and an
	// 8192-pixel edge leave room for supported native aspect ratios while
	// bounding full decoder allocations (including 16-bit PNG pixels).
	geminiImagesMaxOutputPixels = 32_000_000
	geminiImagesMaxOutputEdge   = 8192
)

// This is intentionally a closed list. A provider-specific alias must never
// turn arbitrary chat models into paid image generation requests.
func isGeminiNativeImageModel(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "gemini-3-pro-image", "gemini-3-pro-image-preview",
		"gemini-3.1-flash-image", "gemini-3.1-flash-image-preview",
		"gemini-2.5-flash-image", "gemini-2.5-flash-image-preview",
		"gemini-nano-banana-2.1", "monkey-image-pro", "monkey-image-flash 2", "nano-banana-2.1":
		return true
	default:
		return false
	}
}

func geminiImagesInvalidRequest(param, message string) *OpenAIImagesUpstreamError {
	return &OpenAIImagesUpstreamError{StatusCode: http.StatusBadRequest, ErrorType: "invalid_request_error", Code: "unsupported_gemini_image_option", Param: param, Message: message}
}

// buildGeminiImagesURL preserves administrator-configured path prefixes. The
// closed model list also permits the provider's literal space-containing alias;
// URL.String escapes that path safely without permitting path injection.
func buildGeminiImagesURL(baseURL, model string) (string, error) {
	if !isGeminiNativeImageModel(model) {
		return "", geminiImagesInvalidRequest("model", "Unsupported Gemini image model")
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", fmt.Errorf("invalid Gemini image upstream base URL")
	}
	prefix := strings.TrimRight(u.Path, "/")
	for _, suffix := range []string{"/v1beta", "/v1"} {
		if strings.HasSuffix(prefix, suffix) {
			prefix = strings.TrimSuffix(prefix, suffix)
			break
		}
	}
	u.Path = prefix + "/v1beta/models/" + strings.TrimSpace(model) + ":generateContent"
	u.RawPath = ""
	return u.String(), nil
}

func buildGeminiImagesBody(parsed *OpenAIImagesRequest) ([]byte, error) {
	if parsed == nil {
		return nil, geminiImagesInvalidRequest("", "Image request is required")
	}
	if parsed.N != 1 {
		return nil, geminiImagesInvalidRequest("n", "Gemini images currently supports only n=1")
	}
	if parsed.Stream {
		return nil, geminiImagesInvalidRequest("stream", "Gemini images currently supports only non-streaming requests")
	}
	if parsed.HasMask || parsed.MaskUpload != nil || parsed.MaskImageURL != "" {
		return nil, geminiImagesInvalidRequest("mask", "Gemini images does not support masks")
	}
	if parsed.ResponseFormat != "" && parsed.ResponseFormat != "b64_json" {
		return nil, geminiImagesInvalidRequest("response_format", "Gemini images supports response_format=b64_json")
	}
	for _, option := range []struct{ name, value string }{
		{"quality", parsed.Quality}, {"background", parsed.Background}, {"output_format", parsed.OutputFormat}, {"moderation", parsed.Moderation},
	} {
		if option.value != "" && !strings.EqualFold(option.value, "auto") {
			return nil, geminiImagesInvalidRequest(option.name, "Gemini images does not support the requested "+option.name)
		}
	}
	for _, option := range []struct{ name, value string }{{"input_fidelity", parsed.InputFidelity}, {"style", parsed.Style}} {
		if option.value != "" {
			return nil, geminiImagesInvalidRequest(option.name, "Gemini images does not support "+option.name)
		}
	}
	if parsed.OutputCompression != nil {
		return nil, geminiImagesInvalidRequest("output_compression", "Gemini images does not support output_compression")
	}
	if parsed.PartialImages != nil {
		return nil, geminiImagesInvalidRequest("partial_images", "Gemini images does not support partial_images")
	}
	if len(parsed.InputImageURLs) > 0 {
		return nil, geminiImagesInvalidRequest("images", "Gemini image edits requires multipart image uploads; image URLs are not supported")
	}
	if strings.TrimSpace(parsed.Prompt) == "" {
		return nil, geminiImagesInvalidRequest("prompt", "A non-empty prompt is required")
	}
	if parsed.IsEdits() && len(parsed.Uploads) == 0 {
		return nil, geminiImagesInvalidRequest("image", "Gemini image edits requires an image upload")
	}
	parts := []any{map[string]any{"text": parsed.Prompt}}
	var imageWidth, imageHeight int
	for _, upload := range parsed.Uploads {
		if len(upload.Data) == 0 || len(upload.Data) > openAIImageMaxUploadPartSize {
			return nil, geminiImagesInvalidRequest("image", "Invalid image upload size")
		}
		// Detect the bytes, not the untrusted filename or Content-Type header.
		mimeType := http.DetectContentType(upload.Data)
		if mimeType != "image/png" && mimeType != "image/jpeg" && mimeType != "image/webp" {
			return nil, geminiImagesInvalidRequest("image", "Gemini image uploads must be PNG, JPEG or WebP")
		}
		width, height := 0, 0
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(upload.Data)); err == nil {
			width, height = cfg.Width, cfg.Height
		} else if w, h, ok := detectOpenAIWebPDimensions(upload.Data); ok {
			width, height = w, h
		}
		if width <= 0 || height <= 0 {
			return nil, geminiImagesInvalidRequest("image", "Invalid image upload")
		}
		if imageWidth == 0 {
			imageWidth, imageHeight = width, height
		}
		parts = append(parts, map[string]any{"inlineData": map[string]string{"mimeType": mimeType, "data": base64.StdEncoding.EncodeToString(upload.Data)}})
	}
	size := strings.TrimSpace(parsed.Size)
	tier := ImageBillingSize2K
	if size != "" && !strings.EqualFold(size, "auto") {
		var ok bool
		tier, ok = ClassifyImageBillingTier(size)
		if !ok {
			return nil, geminiImagesInvalidRequest("size", "Gemini image size must be 1K, 2K, 4K or WIDTHxHEIGHT")
		}
		if width, height, ok := parseImageBillingDimensions(size); ok {
			imageWidth, imageHeight = width, height
		}
	}
	imageConfig := map[string]string{"imageSize": tier}
	if imageWidth > 0 && imageHeight > 0 {
		imageConfig["aspectRatio"] = nearestGeminiImageAspectRatio(imageWidth, imageHeight)
	}
	return json.Marshal(map[string]any{
		"contents":         []any{map[string]any{"role": "user", "parts": parts}},
		"generationConfig": map[string]any{"responseModalities": []string{"IMAGE"}, "imageConfig": imageConfig},
	})
}

func nearestGeminiImageAspectRatio(width, height int) string {
	best, distance := "1:1", math.Inf(1)
	for _, ratio := range []struct {
		value         string
		width, height float64
	}{
		{"1:1", 1, 1}, {"2:3", 2, 3}, {"3:2", 3, 2}, {"3:4", 3, 4}, {"4:3", 4, 3},
		{"4:5", 4, 5}, {"5:4", 5, 4}, {"9:16", 9, 16}, {"16:9", 16, 9}, {"21:9", 21, 9},
	} {
		delta := math.Abs(math.Log((float64(width) / float64(height)) / (ratio.width / ratio.height)))
		if delta < distance {
			best, distance = ratio.value, delta
		}
	}
	return best
}

// validateGeminiImageOutput checks compressed bytes and dimensions before a full
// decode. DecodeConfig alone accepts a PNG containing only its IHDR, which must
// not be returned or billed as a generated image. Only one distinct image is
// decoded per n=1 response, and the existing 50MB per-image file limit applies.
func validateGeminiImageOutput(encoded string) (mimeType, size string, err error) {
	if len(encoded) == 0 || len(encoded) > base64.StdEncoding.EncodedLen(openAIImageMaxUploadPartSize) {
		return "", "", fmt.Errorf("image output exceeds encoded byte limit")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) == 0 || len(decoded) > openAIImageMaxUploadPartSize {
		return "", "", fmt.Errorf("invalid image output bytes")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(decoded))
	if err != nil || (format != "png" && format != "jpeg" && format != "webp") {
		return "", "", fmt.Errorf("invalid image output header")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > geminiImagesMaxOutputEdge || cfg.Height > geminiImagesMaxOutputEdge || int64(cfg.Width)*int64(cfg.Height) > geminiImagesMaxOutputPixels {
		return "", "", fmt.Errorf("image output exceeds dimension or pixel limit")
	}
	img, decodedFormat, err := image.Decode(bytes.NewReader(decoded))
	if err != nil || img == nil || decodedFormat != format || img.Bounds().Dx() != cfg.Width || img.Bounds().Dy() != cfg.Height {
		return "", "", fmt.Errorf("image output failed full decode")
	}
	return "image/" + format, fmt.Sprintf("%dx%d", cfg.Width, cfg.Height), nil
}

// convertGeminiImagesResponse never fetches fileData URLs. Only inline image
// bytes produce a successful OpenAI image response and a billable image count.
func convertGeminiImagesResponse(body []byte, parsed *OpenAIImagesRequest) ([]byte, OpenAIUsage, []string, error) {
	invalid := func(message string) ([]byte, OpenAIUsage, []string, error) {
		return nil, OpenAIUsage{}, nil, &OpenAIImagesUpstreamError{StatusCode: http.StatusBadGateway, ErrorType: "upstream_error", Code: "invalid_image_response", Message: message}
	}
	if !gjson.ValidBytes(body) || gjson.GetBytes(body, "error").Exists() {
		return invalid("Gemini upstream returned an invalid image response")
	}
	data := make([]map[string]any, 0, 1)
	sizes := make([]string, 0, 1)
	seen := map[string]bool{}
	malformed := false
	invalidReason := "Gemini upstream did not return a valid inline image"
	gjson.GetBytes(body, "candidates").ForEach(func(_, candidate gjson.Result) bool {
		candidate.Get("content.parts").ForEach(func(_, part gjson.Result) bool {
			inline := part.Get("inlineData")
			if !inline.Exists() {
				inline = part.Get("inline_data")
			}
			if !inline.Exists() {
				return true
			}
			mimeType := inline.Get("mimeType").String()
			if mimeType == "" {
				mimeType = inline.Get("mime_type").String()
			}
			encoded := strings.TrimSpace(inline.Get("data").String())
			if encoded == "" || !isGeminiInlineImageMIMEType(strings.ToLower(mimeType)) {
				malformed = true
				return false
			}
			if seen[encoded] {
				return true
			}
			// n=1 cannot authorize multiple distinct outputs or repeated full
			// decodes of supplier-controlled images. Reject before decoding a
			// second image; identical repeated parts remain deduplicated.
			if len(data) > 0 {
				malformed = true
				invalidReason = "Gemini upstream returned multiple distinct images for n=1"
				return false
			}
			actualMIME, size, err := validateGeminiImageOutput(encoded)
			if err != nil {
				malformed = true
				return false
			}
			seen[encoded] = true
			sizes = append(sizes, size)
			data = append(data, map[string]any{"b64_json": encoded, "size": size, "output_format": strings.TrimPrefix(actualMIME, "image/")})
			return true
		})
		return !malformed
	})
	if malformed || len(data) == 0 {
		return invalid(invalidReason)
	}
	metadata := gjson.GetBytes(body, "usageMetadata")
	// OpenAI input_tokens includes its cached subset. Gemini thinking tokens,
	// however, are reported separately from candidate output tokens.
	usage := OpenAIUsage{InputTokens: max(int(metadata.Get("promptTokenCount").Int()), 0), OutputTokens: max(int(metadata.Get("candidatesTokenCount").Int()), 0) + max(int(metadata.Get("thoughtsTokenCount").Int()), 0), CacheReadInputTokens: max(int(metadata.Get("cachedContentTokenCount").Int()), 0)}
	metadata.Get("candidatesTokensDetails").ForEach(func(_, detail gjson.Result) bool {
		if strings.EqualFold(detail.Get("modality").String(), "IMAGE") {
			usage.ImageOutputTokens += max(int(detail.Get("tokenCount").Int()), 0)
		}
		return true
	})
	metadata.Get("promptTokensDetails").ForEach(func(_, detail gjson.Result) bool {
		if strings.EqualFold(detail.Get("modality").String(), "IMAGE") {
			usage.ImageInputTokens += max(int(detail.Get("tokenCount").Int()), 0)
		}
		return true
	})
	converted, err := json.Marshal(map[string]any{"created": time.Now().Unix(), "model": parsed.Model, "data": data, "size": sizes[0], "output_format": data[0]["output_format"], "usage": usage})
	return converted, usage, sizes, err
}

func (s *OpenAIGatewayService) forwardOpenAIImagesGemini(ctx context.Context, c *gin.Context, account *Account, parsed *OpenAIImagesRequest, upstreamModel string) (*OpenAIForwardResult, error) {
	start := time.Now()
	forwardBody, err := buildGeminiImagesBody(parsed)
	if err != nil {
		if upErr, ok := err.(*OpenAIImagesUpstreamError); ok {
			writeOpenAIImagesUpstreamErrorResponse(c, upErr)
		}
		return nil, err
	}
	baseURL, err := s.validateUpstreamBaseURL(account.GetOpenAIBaseURL())
	if err != nil {
		return nil, err
	}
	targetURL, err := buildGeminiImagesURL(baseURL, upstreamModel)
	if err != nil {
		return nil, err
	}
	upstreamCtx, release := detachUpstreamContext(ctx)
	defer release()
	token, _, err := s.GetAccessToken(upstreamCtx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIImagesRequest(upstreamCtx, c, account, forwardBody, "application/json", token, parsed.Endpoint)
	if err != nil {
		return nil, err
	}
	upstreamReq.URL, err = url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	upstreamReq.Header.Del("Authorization")
	upstreamReq.Header.Set("x-goog-api-key", token)
	account.ApplyHeaderOverrides(upstreamReq.Header)
	SetOpsUpstreamModel(c, upstreamModel)
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	upstreamStart := time.Now()
	resp, err := s.doOpenAIUpstream(upstreamReq, proxyURL, account)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{ProxyID: opsUpstreamProxyID(account), ProxyName: opsUpstreamProxyName(account), Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamURL: safeUpstreamURL(targetURL), Kind: "request_error", Message: safeErr})
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		errorBody := s.readUpstreamErrorBody(resp)
		_ = resp.Body.Close()
		errorBody = s.redactAgentIdentitySensitiveBody(upstreamCtx, account, errorBody)
		resp.Body = io.NopCloser(bytes.NewReader(errorBody))
		message := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(errorBody)))
		setOpsUpstreamError(c, resp.StatusCode, message, "")
		failureEvent := OpsUpstreamErrorEvent{ProxyID: opsUpstreamProxyID(account), ProxyName: opsUpstreamProxyName(account), Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), UpstreamURL: safeUpstreamURL(targetURL), Kind: "failover", Message: message}
		if isOpenAIImagesInsufficientBalance(errorBody) {
			failureEvent.Message = OpenAIImagesInsufficientBalanceMessage
			appendOpsUpstreamError(c, failureEvent)
			s.coolOpenAIImagesInsufficientBalance(upstreamCtx, account)
			return nil, newOpenAIImagesInsufficientBalanceFailoverError(resp.StatusCode, resp.Header, errorBody)
		}
		if s.shouldFailoverOpenAIUpstreamResponse(account, resp.StatusCode, message, errorBody) {
			appendOpsUpstreamError(c, failureEvent)
			shouldDisable := s.handleFailoverSideEffects(upstreamCtx, resp, account, errorBody, upstreamModel)
			retryable := !shouldDisable && account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode)
			if isOpenAIHTTPUpstreamAccessStateError(resp.StatusCode, message, errorBody) {
				return nil, newOpenAIUpstreamFailoverError(resp.StatusCode, resp.Header, errorBody, message, retryable)
			}
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: errorBody, RetryableOnSameAccount: retryable}
		}
		return s.handleOpenAIImagesErrorResponse(upstreamCtx, resp, c, account, upstreamModel)
	}
	body, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}
	converted, usage, sizes, err := convertGeminiImagesResponse(body, parsed)
	if err != nil {
		if upErr, ok := err.(*OpenAIImagesUpstreamError); ok {
			setOpsUpstreamError(c, resp.StatusCode, upErr.Message, "")
			writeOpenAIImagesUpstreamErrorResponse(c, upErr)
		}
		return nil, err
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	// Keep the public request identity here. ChannelUsageFields records the
	// channel/account mapping chain separately; OriginalModel owns billing.
	result := &OpenAIForwardResult{RequestID: resp.Header.Get("x-request-id"), UpstreamHeaders: resp.Header, ResponseHeaders: resp.Header.Clone(), Usage: usage, Model: parsed.Model, UpstreamModel: upstreamModel, UpstreamResponseModel: gjson.GetBytes(body, "modelVersion").String(), UpstreamEndpoint: upstreamReq.URL.Path, Duration: time.Since(start), ImageEndpoint: true, ImageCount: resolveBillableImageCount(len(sizes), 0, imageCountBillingEnabled(getAPIKeyFromContext(c))), ImageSize: normalizeOpenAIImageSizeTier(parsed.Size), ImageInputSize: parsed.Size, ImageOutputSizes: sizes}
	c.Header("Content-Type", "application/json")
	c.Status(http.StatusOK)
	_, writeErr := c.Writer.Write(converted)
	// Generation has already incurred upstream cost even if the caller left.
	// Return the successful image metadata so the handler can still record usage.
	if writeErr != nil {
		result.ClientDisconnect = true
		return result, writeErr
	}
	return result, nil
}
