//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type handlerTestArenaClient struct {
	rows map[string][]service.ArenaRawRow
}

func (c *handlerTestArenaClient) FetchLatestCategory(_ context.Context, subset, _ string, _ int) ([]service.ArenaRawRow, int, error) {
	r := c.rows[subset]
	return r, len(r), nil
}

func (c *handlerTestArenaClient) FetchPreviousSnapshot(_ context.Context, _, _, _ string, _ int) (string, []service.ArenaRawRow, error) {
	return "", nil, nil
}

func TestArenaLeaderboardHandler_GetAndAdminSync(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		ArenaLeaderboard: config.ArenaLeaderboardConfig{
			Enabled:               true,
			DataDir:               t.TempDir(),
			RequestTimeoutSeconds: 10,
			TopLimit:              20,
			BootstrapHistory:      false,
		},
	}
	client := &handlerTestArenaClient{
		rows: map[string][]service.ArenaRawRow{
			"text_style_control":   {{ModelName: "Gemini 4 Carbon", Organization: "google", License: "Proprietary", Rating: 1527, Rank: 1, LeaderboardPublishDate: "2026-11-19"}},
			"webdev":               {{ModelName: "Claude Fable 5.2", Organization: "anthropic", License: "Proprietary", Rating: 1823.6, Rank: 1, LeaderboardPublishDate: "2026-11-20"}},
			"vision_style_control": {{ModelName: "Claude Fable 5", Organization: "anthropic", License: "Proprietary", Rating: 1308.8, Rank: 1, LeaderboardPublishDate: "2026-11-19"}},
			"text_to_image":        {{ModelName: "GPT-Image-2.5", Organization: "openai", License: "Proprietary", Rating: 1429.0, Rank: 1, LeaderboardPublishDate: "2026-11-20"}},
		},
	}
	svc := service.NewArenaLeaderboardService(cfg, client)
	h := NewArenaLeaderboardHandler(svc)

	r := gin.New()
	r.GET("/api/v1/arena/leaderboards", h.Get)
	r.POST("/api/v1/admin/arena/sync", h.AdminSync)

	// Before first sync -> 503 Service Unavailable
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/arena/leaderboards", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	// Invalid category -> 400 Bad Request
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/arena/leaderboards?category=unknown", nil))
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Admin manual sync -> 200 OK
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/admin/arena/sync", nil))
	require.Equal(t, http.StatusOK, w.Code)

	// Public GET all categories -> 200 OK + Cache-Control header
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/arena/leaderboards", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Cache-Control"), "max-age=300")

	var body struct {
		Code int                              `json:"code"`
		Data service.ArenaLeaderboardSnapshot `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Len(t, body.Data.Categories, 4)

	// Public GET filtered single category -> 1 category
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/arena/leaderboards?category=code", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var singleBody struct {
		Code int                              `json:"code"`
		Data service.ArenaLeaderboardSnapshot `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &singleBody))
	require.Len(t, singleBody.Data.Categories, 1)
	require.Equal(t, "code", singleBody.Data.Categories[0].ID)
}
