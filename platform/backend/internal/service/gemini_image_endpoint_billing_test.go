//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Exercise the forwarding callers, rather than constructing a correctly marked
// ForwardResult by hand: image output (or the legacy model-name fallback) does
// not turn a conversation endpoint into the dedicated images API.
func TestGeminiConversationEndpoints_ImageCountBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const model = "gemini-3-pro-image"
	for _, endpoint := range []string{"chat", "messages", "native"} {
		for _, imageOutput := range []bool{false, true} {
			for _, stream := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/images=%v/stream=%v", endpoint, imageOutput, stream), func(t *testing.T) {
					svc, upstream := newGeminiVertexAPIKeyTestService()
					response := geminiVertexAPIKeyTestResponse
					if imageOutput {
						response = strings.Replace(response, `{"text":"hello"}`, `{"inlineData":{"mimeType":"image/png","data":"aGVsbG8="}}`, 1)
					}
					if stream {
						upstream.response.Header.Set("Content-Type", "text/event-stream")
						response = "data: " + response + "\n\n"
					}
					upstream.response.Body = io.NopCloser(strings.NewReader(response))
					c, _ := gin.CreateTestContext(httptest.NewRecorder())
					c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
					body := []byte(fmt.Sprintf(`{"model":%q,"stream":%v,"max_tokens":16,"messages":[{"role":"user","content":"draw"}]}`, model, stream))
					var result *ForwardResult
					var err error
					switch endpoint {
					case "chat":
						result, err = svc.ForwardAsChatCompletions(context.Background(), c, geminiVertexAPIKeyTestAccount(""), body)
					case "messages":
						result, err = svc.Forward(context.Background(), c, geminiVertexAPIKeyTestAccount(""), body)
					case "native":
						action := "generateContent"
						if stream {
							action = "streamGenerateContent"
						}
						result, err = svc.ForwardNative(context.Background(), c, geminiVertexAPIKeyTestAccount(""), model, action, stream, []byte(`{"contents":[{"role":"user","parts":[{"text":"draw"}]}]}`))
					}
					require.NoError(t, err)
					require.NotNil(t, result)
					require.Equal(t, 1, result.ImageCount, "preserve actual output and legacy model-name fallback")

					groupID := int64(906)
					resolver := newOpenAITokenImageChannelPricingResolverForTest(t, groupID, model)
					billing := &GatewayService{billingService: resolver.billingService, resolver: resolver}
					group := &Group{ID: groupID, Platform: PlatformComposite, Hydrated: true, AllowImageGeneration: true, ImagePrice1K: f64p(0.4)}
					key := &APIKey{GroupID: &groupID, Group: group}
					before := billing.calculateRecordUsageCost(context.Background(), result, key, model, 1, 1, time.Time{})
					group.ImageCountBilling = true
					after := billing.calculateRecordUsageCost(context.Background(), result, key, model, 1, 1, time.Time{})
					require.Equal(t, string(BillingModeToken), after.BillingMode)
					require.Greater(t, after.TotalCost, 0.0)
					require.Equal(t, before.TotalCost, after.TotalCost, "images-only switch must not change conversation pricing")
					require.False(t, result.ImageEndpoint, "conversation output must not enable dedicated-image billing")

					// Explicit channel image pricing remains valid for native and
					// compatibility requests, independently of endpoint identity.
					billing.resolver = newOpenAIImageChannelPricingResolverForTest(t, groupID, model, 0.25)
					group.ImagePrice1K = nil
					for _, enabled := range []bool{false, true} {
						group.ImageCountBilling = enabled
						cost := billing.calculateRecordUsageCost(context.Background(), result, key, model, 1, 1, time.Time{})
						require.Equal(t, string(BillingModeImage), cost.BillingMode)
						require.InDelta(t, 0.25, cost.TotalCost, 1e-12)
					}
				})
			}
		}
	}
}
