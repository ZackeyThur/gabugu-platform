//go:build unit

package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestForwardImages_APIKeyUpstreamHAliases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, model := range []string{"gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst", "gpt-image-2.6-flare", "gpt-image-2.6-sunburst"} {
		upstreamModel := "all" + model[3:] + "-H"
		for _, edits := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/edits=%t", model, edits), func(t *testing.T) {
				endpoint, contentType := openAIImagesGenerationsEndpoint, "application/json"
				body := []byte(fmt.Sprintf(`{"model":%q,"prompt":"test","quality":"medium","size":"1024x1024","n":1}`, model))
				if edits {
					endpoint = openAIImagesEditsEndpoint
					var buffer bytes.Buffer
					writer := multipart.NewWriter(&buffer)
					for field, value := range map[string]string{"model": model, "prompt": "test", "quality": "medium", "size": "1024x1024", "n": "1"} {
						require.NoError(t, writer.WriteField(field, value))
					}
					part, err := writer.CreateFormFile("image", "reference.png")
					require.NoError(t, err)
					_, err = part.Write([]byte("fixture-reference"))
					require.NoError(t, err)
					require.NoError(t, writer.Close())
					body, contentType = buffer.Bytes(), writer.FormDataContentType()
				}
				upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: http.StatusOK,
					Header: http.Header{"Content-Type": {"application/json"}},
					Body:   io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, encodeOpenAIImageTestPNG(t, 1, 1))))}}
				svc := withTestGatewayPricing(&OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream})
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
				c.Request.Header.Set("Content-Type", contentType)
				c.Set("api_key", &APIKey{ID: 1, Group: &Group{ImageCountBilling: true}})
				parsed, err := svc.ParseOpenAIImagesRequest(c, body)
				require.NoError(t, err)
				proxyID := int64(5)
				account := &Account{ID: 80, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
					ProxyID: &proxyID, Proxy: &Proxy{Protocol: "http", Host: "127.0.0.1", Port: 19099},
					Credentials: map[string]any{"api_key": "fixture-key", "base_url": "https://image-upstream.example/v1", "model_mapping": map[string]any{model: upstreamModel}}}
				result, err := svc.ForwardImages(context.Background(), c, account, body, parsed, "")
				require.NoError(t, err)
				require.Equal(t, model, result.Model)
				require.Equal(t, upstreamModel, result.UpstreamModel)
				require.Equal(t, 1, result.ImageCount)
				require.True(t, result.ImageEndpoint)
				require.Equal(t, "https://image-upstream.example"+endpoint, upstream.lastReq.URL.String())
				require.Equal(t, "http://127.0.0.1:19099", upstream.lastProxyURL)
				require.Equal(t, "Bearer fixture-key", upstream.lastReq.Header.Get("Authorization"))
				if edits {
					req := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(upstream.lastBody))
					req.Header.Set("Content-Type", upstream.lastReq.Header.Get("Content-Type"))
					require.NoError(t, req.ParseMultipartForm(1<<20))
					t.Cleanup(func() { _ = req.MultipartForm.RemoveAll() })
					require.Equal(t, upstreamModel, req.FormValue("model"))
					require.Equal(t, "medium", req.FormValue("quality"))
					file, _, err := req.FormFile("image")
					require.NoError(t, err)
					defer file.Close()
					image, err := io.ReadAll(file)
					require.NoError(t, err)
					require.Equal(t, "fixture-reference", string(image))
				} else {
					require.Equal(t, upstreamModel, gjson.GetBytes(upstream.lastBody, "model").String())
					require.Equal(t, "medium", gjson.GetBytes(upstream.lastBody, "quality").String())
				}
			})
		}
	}
}

func TestForwardImages_APIKeyRejectsUnknownUpstreamImageAliases(t *testing.T) {
	for _, mapped := range []string{"all-image-unknown-H", "all-image-2-h", "all-image-2-H/extra", "gpt-6-sol"} {
		t.Run(mapped, func(t *testing.T) {
			upstream := &httpUpstreamRecorder{}
			svc := &OpenAIGatewayService{httpUpstream: upstream}
			account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-image-2": mapped}}}
			_, err := svc.ForwardImages(context.Background(), nil, account, nil, &OpenAIImagesRequest{Model: "gpt-image-2"}, "")
			require.ErrorContains(t, err, "images endpoint requires an image model")
			require.Nil(t, upstream.lastReq)
		})
	}
}
