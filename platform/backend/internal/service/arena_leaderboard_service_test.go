//go:build unit

package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type stubArenaRemoteClient struct {
	latestBySubset map[string][]ArenaRawRow
	totalBySubset  map[string]int
	errBySubset    map[string]error

	prevDateBySubset map[string]string
	prevBySubset     map[string][]ArenaRawRow
	prevErrBySubset  map[string]error
	prevCalls        int
}

func (s *stubArenaRemoteClient) FetchLatestCategory(_ context.Context, subset, _ string, _ int) ([]ArenaRawRow, int, error) {
	if err, ok := s.errBySubset[subset]; ok && err != nil {
		return nil, 0, err
	}
	rows := s.latestBySubset[subset]
	total := s.totalBySubset[subset]
	if total == 0 {
		total = len(rows)
	}
	return rows, total, nil
}

func (s *stubArenaRemoteClient) FetchPreviousSnapshot(_ context.Context, subset, _, _ string, _ int) (string, []ArenaRawRow, error) {
	s.prevCalls++
	if err, ok := s.prevErrBySubset[subset]; ok && err != nil {
		return "", nil, err
	}
	return s.prevDateBySubset[subset], s.prevBySubset[subset], nil
}

func newTestArenaConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		ArenaLeaderboard: config.ArenaLeaderboardConfig{
			Enabled:               true,
			BaseURL:               "https://datasets-server.huggingface.co",
			Dataset:               "lmarena-ai/leaderboard-dataset",
			DataDir:               t.TempDir(),
			SyncHour:              8,
			CheckIntervalMinutes:  360,
			RequestTimeoutSeconds: 10,
			TopLimit:              50,
			BootstrapHistory:      true,
		},
	}
}

func TestArenaLeaderboardService_SyncDeltasPersistenceAndReload(t *testing.T) {
	cfg := newTestArenaConfig(t)
	stub := &stubArenaRemoteClient{
		latestBySubset: map[string][]ArenaRawRow{
			"text_style_control": {
				{ModelName: "Model A", Organization: "google", License: "Proprietary", Rating: 1530.4, RatingLower: 1522.4, RatingUpper: 1538.4, VoteCount: 12000, Rank: 1, LeaderboardPublishDate: "2026-11-19"},
				{ModelName: "Model B", Organization: "deepseek", License: "MIT", Rating: 1521.0, RatingLower: 1515.0, RatingUpper: 1527.0, VoteCount: 9500, Rank: 2, LeaderboardPublishDate: "2026-11-19"},
				{ModelName: "Model New", Organization: "anthropic", License: "Proprietary", Rating: 1510.0, RatingLower: 1500.0, RatingUpper: 1520.0, VoteCount: 3000, Rank: 3, LeaderboardPublishDate: "2026-11-19"},
			},
			"webdev": {
				{ModelName: "Code Pro", Organization: "anthropic", License: "Proprietary", Rating: 1823.6, RatingLower: 1810.0, RatingUpper: 1837.2, VoteCount: 2042, Rank: 1, LeaderboardPublishDate: "2026-11-20"},
			},
			"vision_style_control": {
				{ModelName: "Vision Prime", Organization: "anthropic", License: "Proprietary", Rating: 1308.8, RatingLower: 1301.0, RatingUpper: 1316.6, VoteCount: 15449, Rank: 1, LeaderboardPublishDate: "2026-11-19"},
			},
			"text_to_image": {
				{ModelName: "Img Sunburst", Organization: "openai", License: "Proprietary", Rating: 1429.0, RatingLower: 1424.0, RatingUpper: 1434.0, VoteCount: 73040, Rank: 1, LeaderboardPublishDate: "2026-11-20"},
			},
		},
		totalBySubset: map[string]int{
			"text_style_control": 426,
		},
		prevDateBySubset: map[string]string{
			"text_style_control": "2026-11-18",
		},
		prevBySubset: map[string][]ArenaRawRow{
			"text_style_control": {
				{ModelName: "Model A", Organization: "google", Rating: 1528.0, Rank: 1, LeaderboardPublishDate: "2026-11-18"},
				{ModelName: "Old Second", Organization: "openai", Rating: 1522.0, Rank: 2, LeaderboardPublishDate: "2026-11-18"},
				{ModelName: "Model B", Organization: "deepseek", Rating: 1516.5, Rank: 4, LeaderboardPublishDate: "2026-11-18"},
			},
		},
		prevErrBySubset: map[string]error{
			"text_to_image": errors.New("bootstrap timeout"),
		},
	}

	svc := NewArenaLeaderboardService(cfg, stub)
	res, err := svc.SyncNow(context.Background())
	require.NoError(t, err)
	require.Equal(t, "ok", res.Status)

	snap, err := svc.GetSnapshot("")
	require.NoError(t, err)
	require.Len(t, snap.Categories, 4)

	textBoard := snap.Categories[0]
	require.Equal(t, ArenaCategoryText, textBoard.ID)
	require.Equal(t, "2026-11-19", textBoard.PublishDate)
	require.Equal(t, "2026-11-18", textBoard.PreviousPublishDate)
	require.Equal(t, 426, textBoard.TotalModels)
	require.Len(t, textBoard.Entries, 3)

	// Model A stayed #1 (+2.4)
	require.NotNil(t, textBoard.Entries[0].RankDelta)
	require.Equal(t, 0, *textBoard.Entries[0].RankDelta)
	require.NotNil(t, textBoard.Entries[0].ScoreDelta)
	require.InDelta(t, 2.4, *textBoard.Entries[0].ScoreDelta, 0.01)
	require.Equal(t, 8, textBoard.Entries[0].CI)
	require.False(t, textBoard.Entries[0].IsOpen)

	// Model B rose from #4 -> #2 (+2) and is open-source (MIT)
	require.NotNil(t, textBoard.Entries[1].RankDelta)
	require.Equal(t, 2, *textBoard.Entries[1].RankDelta)
	require.True(t, textBoard.Entries[1].IsOpen)

	// Model New was absent previously
	require.True(t, textBoard.Entries[2].IsNew)
	require.Nil(t, textBoard.Entries[2].RankDelta)

	// Highlights check
	require.Equal(t, "Model A", textBoard.Highlights.Leader)
	require.InDelta(t, 9.4, textBoard.Highlights.RunnerUpGap, 0.01)
	require.Equal(t, 1, textBoard.Highlights.NewEntries)
	require.NotNil(t, textBoard.Highlights.BiggestRiser)
	require.Equal(t, "Model B", textBoard.Highlights.BiggestRiser.ModelName)
	require.Equal(t, 2, textBoard.Highlights.BiggestRiser.RankDelta)
	require.Equal(t, "Model B", textBoard.Highlights.TopOpenModel)
	require.Equal(t, 1, textBoard.Highlights.OpenModelCount)

	// text_to_image bootstrap failed gracefully but board still populated
	imageBoard := snap.Categories[3]
	require.Equal(t, ArenaCategoryImage, imageBoard.ID)
	require.Empty(t, imageBoard.PreviousPublishDate)
	require.Len(t, imageBoard.Entries, 1)

	// Re-create service pointing to the same DataDir and verify disk reload works without network.
	svc2 := NewArenaLeaderboardService(cfg, nil)
	loaded, err := svc2.loadSnapshotFromDisk()
	require.NoError(t, err)
	require.Len(t, loaded.Categories, 4)
	require.Equal(t, "Model A", loaded.Categories[0].Entries[0].ModelName)
}

func TestArenaLeaderboardService_StalePreservedOnUpstreamFailure(t *testing.T) {
	cfg := newTestArenaConfig(t)
	cfg.ArenaLeaderboard.BootstrapHistory = false

	stub := &stubArenaRemoteClient{
		latestBySubset: map[string][]ArenaRawRow{
			"text_style_control":   {{ModelName: "Text1", Rating: 1500, Rank: 1, LeaderboardPublishDate: "2026-11-19"}},
			"webdev":               {{ModelName: "Code1", Rating: 1800, Rank: 1, LeaderboardPublishDate: "2026-11-19"}},
			"vision_style_control": {{ModelName: "Vision1", Rating: 1300, Rank: 1, LeaderboardPublishDate: "2026-11-19"}},
			"text_to_image":        {{ModelName: "Image1", Rating: 1400, Rank: 1, LeaderboardPublishDate: "2026-11-19"}},
		},
	}

	svc := NewArenaLeaderboardService(cfg, stub)
	_, err := svc.SyncNow(context.Background())
	require.NoError(t, err)

	// Now fail 2 categories and ensure old boards stay intact.
	stub.errBySubset = map[string]error{
		"webdev":        errors.New("network timeout"),
		"text_to_image": errors.New("503 upstream"),
	}
	res, err := svc.SyncNow(context.Background())
	require.NoError(t, err)
	require.Equal(t, "partial", res.Status)

	snap, err := svc.GetSnapshot("")
	require.NoError(t, err)
	require.Equal(t, "partial", snap.SyncStatus)
	require.Len(t, snap.Categories, 4)
	require.Equal(t, "Code1", snap.Categories[1].Entries[0].ModelName)

	// Fail all 4 categories: SyncNow returns error, yet GetSnapshot still serves valid cached data.
	stub.errBySubset = map[string]error{
		"text_style_control":   errors.New("down"),
		"webdev":               errors.New("down"),
		"vision_style_control": errors.New("down"),
		"text_to_image":        errors.New("down"),
	}
	_, err = svc.SyncNow(context.Background())
	require.Error(t, err)

	snapAfterTotalOutage, err := svc.GetSnapshot("code")
	require.NoError(t, err)
	require.Len(t, snapAfterTotalOutage.Categories, 1)
	require.Equal(t, "Code1", snapAfterTotalOutage.Categories[0].Entries[0].ModelName)
}

func TestArenaLeaderboardService_HistoryPrunesToMaxLimit(t *testing.T) {
	cfg := newTestArenaConfig(t)
	svc := NewArenaLeaderboardService(cfg, nil)

	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 35; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		err := svc.saveHistorySnapshot(ArenaCategoryText, d, []ArenaRawRow{{ModelName: "M1", Rank: 1, Rating: 1500, LeaderboardPublishDate: d}})
		require.NoError(t, err)
	}

	entries, err := os.ReadDir(filepath.Join(cfg.ArenaLeaderboard.DataDir, "history", ArenaCategoryText))
	require.NoError(t, err)
	require.Len(t, entries, arenaMaxHistoryFiles)
}

func TestNextArenaSyncDelayAndOpenLicense(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 11, 23, 7, 30, 0, 0, loc)
	delay := nextArenaSyncDelay(now, 8, 6*time.Hour)
	// Next 08:00 in server timezone should be <= 6h.
	require.LessOrEqual(t, delay, 6*time.Hour)
	require.GreaterOrEqual(t, delay, time.Minute)

	require.False(t, IsOpenArenaLicense("Proprietary"))
	require.False(t, IsOpenArenaLicense(""))
	require.True(t, IsOpenArenaLicense("MIT"))
	require.True(t, IsOpenArenaLicense("Apache 2.0"))
	require.True(t, IsOpenArenaLicense("Kimi K3 license"))
}

type blockingArenaRemoteClient struct {
	started chan struct{}
}

func (b *blockingArenaRemoteClient) FetchLatestCategory(ctx context.Context, _, _ string, _ int) ([]ArenaRawRow, int, error) {
	select {
	case b.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, 0, ctx.Err()
}

func (b *blockingArenaRemoteClient) FetchPreviousSnapshot(ctx context.Context, _, _, _ string, _ int) (string, []ArenaRawRow, error) {
	<-ctx.Done()
	return "", nil, ctx.Err()
}

func TestArenaLeaderboardService_StopCancelsInFlightSync(t *testing.T) {
	cfg := newTestArenaConfig(t)
	client := &blockingArenaRemoteClient{started: make(chan struct{}, 1)}
	svc := NewArenaLeaderboardService(cfg, client)
	svc.startScheduler()

	select {
	case <-client.started:
	case <-time.After(2 * time.Second):
		t.Fatal("expected initial background sync to start")
	}

	done := make(chan struct{})
	go func() {
		svc.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() stalled while remote fetch was blocked")
	}
}
