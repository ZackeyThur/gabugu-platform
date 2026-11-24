package handler

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGeminiImagesKeepaliveProtectsLongNativeRequests(t *testing.T) {
	for _, cfg := range []*config.Config{nil, {}} {
		h := &OpenAIGatewayHandler{cfg: cfg}
		for _, model := range []string{"gemini-3-pro-image", "monkey-image-pro", "monkey-image-flash 2", "nano-banana-2.1"} {
			require.Equal(t, 15*time.Second, h.openAIImagesJSONKeepaliveInterval(model), model)
		}
		require.Zero(t, h.openAIImagesJSONKeepaliveInterval("gpt-image-2"))
		require.Zero(t, h.openAIImagesJSONKeepaliveInterval())
	}
	cfg := &config.Config{}
	cfg.Gateway.ImageNonstreamKeepaliveInterval = 10
	h := &OpenAIGatewayHandler{cfg: cfg}
	require.Equal(t, 10*time.Second, h.openAIImagesJSONKeepaliveInterval("monkey-image-pro"))
}
