//go:build unit

package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestArenaLeaderboardRemoteClient_FetchLatestCategory_FloatAndIntFields(t *testing.T) {
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		require.Equal(t, "/filter", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"num_rows_total": 426,
			"rows": [
				{
					"row": {
						"model_name": "Gemini 4 Carbon (High)",
						"organization": "google",
						"license": "Proprietary",
						"rating": 1527.04,
						"rating_lower": 1517.2,
						"rating_upper": 1536.8,
						"vote_count": 4262.0,
						"rank": 1.0,
						"category": "overall",
						"leaderboard_publish_date": "2026-11-19"
					}
				},
				{
					"row": {
						"model_name": "DeepSeek V4",
						"organization": "deepseek",
						"license": "MIT",
						"rating": 1490.5,
						"rating_lower": 1482.5,
						"rating_upper": 1498.5,
						"vote_count": 9800,
						"rank": 2,
						"category": "overall",
						"leaderboard_publish_date": "2026-11-19"
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	client := NewArenaLeaderboardRemoteClient(srv.URL, "lmarena-ai/leaderboard-dataset", 10*time.Second, "", false)
	rows, total, err := client.FetchLatestCategory(context.Background(), "text_style_control", "overall", 50)
	require.NoError(t, err)
	require.Equal(t, 426, total)
	require.Len(t, rows, 2)
	require.Contains(t, capturedQuery, "config=text_style_control")
	require.Contains(t, capturedQuery, "split=latest")
	require.Contains(t, capturedQuery, "length=50")

	require.Equal(t, "Gemini 4 Carbon (High)", rows[0].ModelName)
	require.Equal(t, 1, rows[0].Rank)
	require.Equal(t, 4262, rows[0].VoteCount)
	require.Equal(t, "2026-11-19", rows[0].LeaderboardPublishDate)
	require.Equal(t, "MIT", rows[1].License)
	require.Equal(t, 2, rows[1].Rank)
}

func TestArenaLeaderboardRemoteClient_FetchPreviousSnapshot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		where := r.URL.Query().Get("where")
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(where, `"rank"=1`) {
			_, _ = w.Write([]byte(`{
				"num_rows_total": 10,
				"rows": [
					{"row": {"model_name": "M1", "rank": 1, "leaderboard_publish_date": "2026-11-19"}},
					{"row": {"model_name": "M1", "rank": 1, "leaderboard_publish_date": "2026-11-18"}},
					{"row": {"model_name": "M2", "rank": 1, "leaderboard_publish_date": "2026-11-11"}}
				]
			}`))
			return
		}
		require.Contains(t, where, `'2026-11-18'`)
		_, _ = w.Write([]byte(`{
			"num_rows_total": 2,
			"rows": [
				{"row": {"model_name": "M1", "organization": "google", "rating": 1520.0, "rank": 1, "leaderboard_publish_date": "2026-11-18"}},
				{"row": {"model_name": "M2", "organization": "anthropic", "rating": 1515.0, "rank": 2, "leaderboard_publish_date": "2026-11-18"}}
			]
		}`))
	}))
	defer srv.Close()

	client := NewArenaLeaderboardRemoteClient(srv.URL, "lmarena-ai/leaderboard-dataset", 10*time.Second, "", false)
	prevDate, rows, err := client.FetchPreviousSnapshot(context.Background(), "text_style_control", "overall", "2026-11-19", 50)
	require.NoError(t, err)
	require.Equal(t, "2026-11-18", prevDate)
	require.Len(t, rows, 2)
	require.Equal(t, "M2", rows[1].ModelName)
}

func TestArenaLeaderboardRemoteClient_RetryOn5xx(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"num_rows_total": 1, "rows": [{"row": {"model_name": "Alpha", "rank": 1, "rating": 1500, "leaderboard_publish_date": "2026-11-20"}}]}`))
	}))
	defer srv.Close()

	client := NewArenaLeaderboardRemoteClient(srv.URL, "lmarena-ai/leaderboard-dataset", 10*time.Second, "", false)
	rows, total, err := client.FetchLatestCategory(context.Background(), "webdev", "overall", 10)
	require.NoError(t, err)
	require.Equal(t, int32(2), attempts.Load())
	require.Equal(t, 1, total)
	require.Len(t, rows, 1)
}

func TestNewArenaLeaderboardRemoteClient_InvalidProxy_FailClosed(t *testing.T) {
	client := NewArenaLeaderboardRemoteClient("https://example.com", "ds", 10*time.Second, "://bad-proxy", false)
	_, ok := client.(*arenaLeaderboardRemoteClientError)
	require.True(t, ok)

	_, _, err := client.FetchLatestCategory(context.Background(), "webdev", "overall", 10)
	require.Error(t, err)
}
