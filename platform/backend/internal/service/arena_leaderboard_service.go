package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/backgroundgate"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

const (
	ArenaCategoryText   = "text"
	ArenaCategoryCode   = "code"
	ArenaCategoryVision = "vision"
	ArenaCategoryImage  = "image"

	arenaSourceName       = "LMArena (Chatbot Arena)"
	arenaSourceSiteURL    = "https://arena.ai/leaderboard"
	arenaDefaultDataset   = "lmarena-ai/leaderboard-dataset"
	arenaMaxHistoryFiles  = 30
	arenaBootstrapTimeout = 40 * time.Second
	// arenaSyncBudget bounds one full sync: four category fetches plus first-run history lookups.
	arenaSyncBudget       = 4 * time.Minute
	arenaSnapshotFilename = "snapshot.json"
)

var (
	// ErrArenaDisabled indicates the Arena leaderboard feature is disabled in configuration.
	ErrArenaDisabled = errors.New("arena leaderboard is disabled")
	// ErrArenaNotReady indicates the initial sync has not yet produced a snapshot.
	ErrArenaNotReady = errors.New("arena leaderboard data is initializing")
	// ErrArenaCategoryInvalid indicates an unsupported category query parameter.
	ErrArenaCategoryInvalid = errors.New("invalid arena leaderboard category")

	arenaDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// ArenaCategorySpec defines how a Gabugu leaderboard tab maps to an official LMArena subset.
type ArenaCategorySpec struct {
	ID              string
	Subset          string
	FilterCategory  string
	StyleControlled bool
}

// DefaultArenaCategorySpecs returns the 4 curated v1 tracks in display order.
func DefaultArenaCategorySpecs() []ArenaCategorySpec {
	return []ArenaCategorySpec{
		{ID: ArenaCategoryText, Subset: "text_style_control", FilterCategory: "overall", StyleControlled: true},
		{ID: ArenaCategoryCode, Subset: "webdev", FilterCategory: "overall", StyleControlled: false},
		{ID: ArenaCategoryVision, Subset: "vision_style_control", FilterCategory: "overall", StyleControlled: true},
		{ID: ArenaCategoryImage, Subset: "text_to_image", FilterCategory: "overall", StyleControlled: false},
	}
}

// ArenaRawRow represents a normalized upstream leaderboard row fetched from Hugging Face.
type ArenaRawRow struct {
	ModelName              string  `json:"model_name"`
	Organization           string  `json:"organization"`
	License                string  `json:"license"`
	Rating                 float64 `json:"rating"`
	RatingLower            float64 `json:"rating_lower"`
	RatingUpper            float64 `json:"rating_upper"`
	VoteCount              int     `json:"vote_count"`
	Rank                   int     `json:"rank"`
	Category               string  `json:"category"`
	LeaderboardPublishDate string  `json:"leaderboard_publish_date"`
}

// ArenaLeaderboardRemoteClient fetches leaderboard snapshots from the upstream dataset API.
type ArenaLeaderboardRemoteClient interface {
	FetchLatestCategory(ctx context.Context, subset, filterCategory string, limit int) (rows []ArenaRawRow, totalRows int, err error)
	FetchPreviousSnapshot(ctx context.Context, subset, filterCategory, currentPublishDate string, limit int) (prevDate string, rows []ArenaRawRow, err error)
}

// ArenaSourceInfo describes public attribution for the leaderboard data.
type ArenaSourceInfo struct {
	Name       string `json:"name"`
	SiteURL    string `json:"site_url"`
	DatasetURL string `json:"dataset_url"`
}

// ArenaBiggestRiser highlights the entry with the largest positive rank gain.
type ArenaBiggestRiser struct {
	ModelName    string   `json:"model_name"`
	Organization string   `json:"organization"`
	Rank         int      `json:"rank"`
	RankDelta    int      `json:"rank_delta"`
	ScoreDelta   *float64 `json:"score_delta,omitempty"`
}

// ArenaCategoryHighlights provides pre-computed summary metrics for a category card strip.
type ArenaCategoryHighlights struct {
	Leader              string             `json:"leader"`
	LeaderOrganization  string             `json:"leader_organization"`
	LeaderScore         float64            `json:"leader_score"`
	RunnerUp            string             `json:"runner_up,omitempty"`
	RunnerUpGap         float64            `json:"runner_up_gap"`
	NewEntries          int                `json:"new_entries"`
	BiggestRiser        *ArenaBiggestRiser `json:"biggest_riser,omitempty"`
	OpenModelCount      int                `json:"open_model_count"`
	TopOpenModel        string             `json:"top_open_model,omitempty"`
	TopOpenOrganization string             `json:"top_open_organization,omitempty"`
	TopOpenRank         int                `json:"top_open_rank,omitempty"`
	TopOpenScore        float64            `json:"top_open_score,omitempty"`
}

// ArenaModelEntry represents a single ranked model inside a category board.
type ArenaModelEntry struct {
	Rank         int      `json:"rank"`
	RankDelta    *int     `json:"rank_delta,omitempty"`
	IsNew        bool     `json:"is_new"`
	ModelName    string   `json:"model_name"`
	Organization string   `json:"organization"`
	License      string   `json:"license"`
	IsOpen       bool     `json:"is_open"`
	Score        float64  `json:"score"`
	ScoreDelta   *float64 `json:"score_delta,omitempty"`
	CI           int      `json:"ci"`
	RatingLower  float64  `json:"rating_lower"`
	RatingUpper  float64  `json:"rating_upper"`
	Votes        int      `json:"votes"`
}

// ArenaCategoryBoard holds the ranked entries and metadata for one track.
type ArenaCategoryBoard struct {
	ID                  string                  `json:"id"`
	Subset              string                  `json:"subset"`
	StyleControlled     bool                    `json:"style_controlled"`
	PublishDate         string                  `json:"publish_date"`
	PreviousPublishDate string                  `json:"previous_publish_date,omitempty"`
	TotalModels         int                     `json:"total_models"`
	Highlights          ArenaCategoryHighlights `json:"highlights"`
	Entries             []ArenaModelEntry       `json:"entries"`
}

// ArenaLeaderboardSnapshot is the cached public API payload persisted on disk and in memory.
type ArenaLeaderboardSnapshot struct {
	Source        ArenaSourceInfo      `json:"source"`
	LastCheckedAt time.Time            `json:"last_checked_at"`
	LastSyncedAt  time.Time            `json:"last_synced_at"`
	SyncStatus    string               `json:"sync_status"`
	Categories    []ArenaCategoryBoard `json:"categories"`
}

// ArenaCategorySyncStatus summarizes the outcome of syncing one category.
type ArenaCategorySyncStatus struct {
	ID          string `json:"id"`
	PublishDate string `json:"publish_date,omitempty"`
	ModelCount  int    `json:"model_count"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// ArenaSyncResult reports the outcome of a manual or scheduled sync run.
type ArenaSyncResult struct {
	CheckedAt  time.Time                 `json:"checked_at"`
	DurationMs int64                     `json:"duration_ms"`
	Status     string                    `json:"status"`
	Categories []ArenaCategorySyncStatus `json:"categories"`
}

type arenaHistoryFile struct {
	CategoryID  string        `json:"category_id"`
	PublishDate string        `json:"publish_date"`
	SavedAt     time.Time     `json:"saved_at"`
	Rows        []ArenaRawRow `json:"rows"`
}

// ArenaLeaderboardService manages scheduled fetching, local caching, history diffing, and serving of Arena rankings.
type ArenaLeaderboardService struct {
	cfg          *config.Config
	remoteClient ArenaLeaderboardRemoteClient
	specs        []ArenaCategorySpec
	nowFn        func() time.Time

	mu       sync.RWMutex
	snapshot *ArenaLeaderboardSnapshot

	syncMu     sync.Mutex
	stopOnce   sync.Once
	stopCh     chan struct{}
	stopCtx    context.Context
	stopCancel context.CancelFunc
	wg         sync.WaitGroup
}

// NewArenaLeaderboardService creates a new ArenaLeaderboardService.
func NewArenaLeaderboardService(cfg *config.Config, remoteClient ArenaLeaderboardRemoteClient) *ArenaLeaderboardService {
	stopCtx, stopCancel := context.WithCancel(context.Background())
	return &ArenaLeaderboardService{
		cfg:          cfg,
		remoteClient: remoteClient,
		specs:        DefaultArenaCategorySpecs(),
		nowFn:        timezone.Now,
		stopCh:       make(chan struct{}),
		stopCtx:      stopCtx,
		stopCancel:   stopCancel,
	}
}

// Enabled reports whether the Arena leaderboard feature is enabled.
func (s *ArenaLeaderboardService) Enabled() bool {
	if s == nil || s.cfg == nil {
		return false
	}
	return s.cfg.ArenaLeaderboard.Enabled
}

// Initialize loads any cached local snapshot into memory and registers the background sync scheduler.
func (s *ArenaLeaderboardService) Initialize() error {
	if !s.Enabled() {
		logger.LegacyPrintf("service.arena", "%s", "[ArenaLeaderboard] Service disabled by configuration")
		return nil
	}

	dataDir := s.dataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Failed to create data dir %s: %v", dataDir, err)
	}

	if snap, err := s.loadSnapshotFromDisk(); err == nil && snap != nil {
		s.mu.Lock()
		s.snapshot = snap
		s.mu.Unlock()
		logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Loaded cached snapshot from disk (%d categories, last_synced=%s)",
			len(snap.Categories), snap.LastSyncedAt.Format(time.RFC3339))
	} else if err != nil && !os.IsNotExist(err) {
		logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Failed to load cached snapshot: %v", err)
	}

	backgroundgate.Run("ArenaLeaderboardSync", s.startScheduler)
	return nil
}

// Stop stops the background scheduler cleanly and cancels any in-flight sync.
func (s *ArenaLeaderboardService) Stop() {
	if s == nil {
		return
	}
	backgroundgate.Stop("ArenaLeaderboardSync", nil)
	s.stopOnce.Do(func() {
		if s.stopCancel != nil {
			s.stopCancel()
		}
		close(s.stopCh)
	})
	s.wg.Wait()
}

// GetSnapshot returns the cached snapshot, optionally filtered to a single category ID.
func (s *ArenaLeaderboardService) GetSnapshot(category string) (*ArenaLeaderboardSnapshot, error) {
	if !s.Enabled() {
		return nil, ErrArenaDisabled
	}

	cat := strings.TrimSpace(strings.ToLower(category))
	if cat != "" && !s.isValidCategory(cat) {
		return nil, ErrArenaCategoryInvalid
	}

	s.mu.RLock()
	snap := s.snapshot
	s.mu.RUnlock()

	if snap == nil || len(snap.Categories) == 0 {
		return nil, ErrArenaNotReady
	}

	out := &ArenaLeaderboardSnapshot{
		Source:        snap.Source,
		LastCheckedAt: snap.LastCheckedAt,
		LastSyncedAt:  snap.LastSyncedAt,
		SyncStatus:    snap.SyncStatus,
		Categories:    make([]ArenaCategoryBoard, 0, len(snap.Categories)),
	}

	for _, b := range snap.Categories {
		if cat != "" && b.ID != cat {
			continue
		}
		out.Categories = append(out.Categories, cloneCategoryBoard(b))
	}

	if cat != "" && len(out.Categories) == 0 {
		return nil, ErrArenaNotReady
	}
	return out, nil
}

// SyncNow fetches all configured tracks from the upstream dataset, computes diffs against history,
// persists atomically to disk, and swaps the in-memory snapshot.
func (s *ArenaLeaderboardService) SyncNow(ctx context.Context) (*ArenaSyncResult, error) {
	if !s.Enabled() {
		return nil, ErrArenaDisabled
	}
	if s.remoteClient == nil {
		return nil, errors.New("arena leaderboard remote client is nil")
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	started := s.nowFn()
	existingByID := make(map[string]ArenaCategoryBoard)
	var prevSyncedAt time.Time

	s.mu.RLock()
	if s.snapshot != nil {
		prevSyncedAt = s.snapshot.LastSyncedAt
		for _, b := range s.snapshot.Categories {
			existingByID[b.ID] = cloneCategoryBoard(b)
		}
	}
	s.mu.RUnlock()

	topLimit := s.topLimit()
	boards := make([]ArenaCategoryBoard, 0, len(s.specs))
	statuses := make([]ArenaCategorySyncStatus, 0, len(s.specs))
	successCount := 0

	for _, spec := range s.specs {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		board, syncStatus := s.syncSingleCategory(ctx, spec, topLimit, existingByID[spec.ID])
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		statuses = append(statuses, syncStatus)
		if syncStatus.Success {
			successCount++
		}
		if board != nil {
			boards = append(boards, *board)
		}
	}

	checkedAt := s.nowFn()
	syncStatus := "ok"
	lastSyncedAt := checkedAt
	switch {
	case successCount == 0:
		syncStatus = "error"
		lastSyncedAt = prevSyncedAt
	case successCount < len(s.specs):
		syncStatus = "partial"
	}

	if len(boards) > 0 {
		newSnap := &ArenaLeaderboardSnapshot{
			Source:        s.sourceInfo(),
			LastCheckedAt: checkedAt,
			LastSyncedAt:  lastSyncedAt,
			SyncStatus:    syncStatus,
			Categories:    boards,
		}
		s.mu.Lock()
		s.snapshot = newSnap
		s.mu.Unlock()

		if err := s.saveSnapshotToDisk(newSnap); err != nil {
			logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Failed to persist snapshot: %v", err)
		}
	}

	res := &ArenaSyncResult{
		CheckedAt:  checkedAt,
		DurationMs: time.Since(started).Milliseconds(),
		Status:     syncStatus,
		Categories: statuses,
	}

	if successCount == 0 {
		return res, errors.New("failed to sync any arena leaderboard category")
	}
	return res, nil
}

func (s *ArenaLeaderboardService) syncSingleCategory(
	ctx context.Context,
	spec ArenaCategorySpec,
	topLimit int,
	fallback ArenaCategoryBoard,
) (*ArenaCategoryBoard, ArenaCategorySyncStatus) {
	reqCtx, cancel := context.WithTimeout(ctx, s.requestTimeout())
	rows, totalRows, err := s.remoteClient.FetchLatestCategory(reqCtx, spec.Subset, spec.FilterCategory, topLimit)
	cancel()

	if err != nil || len(rows) == 0 {
		errMsg := "upstream returned empty rows"
		if err != nil {
			errMsg = err.Error()
		}
		logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Sync category=%s failed: %s", spec.ID, errMsg)
		var preserved *ArenaCategoryBoard
		if len(fallback.Entries) > 0 {
			cp := cloneCategoryBoard(fallback)
			preserved = &cp
		}
		return preserved, ArenaCategorySyncStatus{
			ID:      spec.ID,
			Success: false,
			Error:   errMsg,
		}
	}

	sortedRows := normalizeAndSortRows(rows, topLimit)
	publishDate := extractLatestPublishDate(sortedRows)
	if totalRows < len(sortedRows) {
		totalRows = len(sortedRows)
	}

	// Save current publish_date snapshot into rolling local history.
	if isValidArenaDate(publishDate) {
		if saveErr := s.saveHistorySnapshot(spec.ID, publishDate, sortedRows); saveErr != nil {
			logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Save history failed (%s/%s): %v", spec.ID, publishDate, saveErr)
		}
	}

	// Locate most recent earlier publish_date in local history.
	prevDate, prevRows := s.findPreviousLocalHistory(spec.ID, publishDate)

	// Day-1 cold bootstrap: if no earlier local history exists yet, try one bounded lookup against split=full.
	if prevDate == "" && isValidArenaDate(publishDate) && s.bootstrapHistoryEnabled() {
		bootCtx, bootCancel := context.WithTimeout(ctx, arenaBootstrapTimeout)
		fetchedPrevDate, fetchedPrevRows, bootErr := s.remoteClient.FetchPreviousSnapshot(
			bootCtx, spec.Subset, spec.FilterCategory, publishDate, topLimit,
		)
		bootCancel()
		if bootErr != nil {
			logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Bootstrap history skipped for %s: %v", spec.ID, bootErr)
		} else if isValidArenaDate(fetchedPrevDate) && fetchedPrevDate < publishDate && len(fetchedPrevRows) > 0 {
			prevDate = fetchedPrevDate
			prevRows = normalizeAndSortRows(fetchedPrevRows, topLimit)
			if saveErr := s.saveHistorySnapshot(spec.ID, prevDate, prevRows); saveErr != nil {
				logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Save bootstrap history failed (%s/%s): %v", spec.ID, prevDate, saveErr)
			}
		}
	}

	board := buildCategoryBoard(spec, publishDate, prevDate, totalRows, sortedRows, prevRows)
	return &board, ArenaCategorySyncStatus{
		ID:          spec.ID,
		PublishDate: publishDate,
		ModelCount:  len(board.Entries),
		Success:     true,
	}
}

func (s *ArenaLeaderboardService) startScheduler() {
	if s == nil || !s.Enabled() {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		if s.shouldSyncOnStartup(s.nowFn()) {
			baseCtx := s.stopCtx
			if baseCtx == nil {
				baseCtx = context.Background()
			}
			ctx, cancel := context.WithTimeout(baseCtx, arenaSyncBudget)
			if res, err := s.SyncNow(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					cancel()
					return
				}
				logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Initial background sync failed: %v", err)
			} else {
				logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Initial background sync finished (status=%s, duration=%dms)",
					res.Status, res.DurationMs)
			}
			cancel()
		}

		for {
			delay := nextArenaSyncDelay(s.nowFn(), s.syncHour(), s.checkInterval())
			timer := time.NewTimer(delay)
			select {
			case <-s.stopCh:
				timer.Stop()
				return
			case <-timer.C:
				baseCtx := s.stopCtx
				if baseCtx == nil {
					baseCtx = context.Background()
				}
				ctx, cancel := context.WithTimeout(baseCtx, arenaSyncBudget)
				if res, err := s.SyncNow(ctx); err != nil {
					if errors.Is(err, context.Canceled) {
						cancel()
						return
					}
					logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Scheduled sync failed: %v", err)
				} else {
					logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Scheduled sync finished (status=%s, duration=%dms)",
						res.Status, res.DurationMs)
				}
				cancel()
			}
		}
	}()

	logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Scheduler started (daily_hour=%02d:00, check_interval=%v)",
		s.syncHour(), s.checkInterval())
}

func (s *ArenaLeaderboardService) shouldSyncOnStartup(now time.Time) bool {
	s.mu.RLock()
	snap := s.snapshot
	s.mu.RUnlock()

	if snap == nil || len(snap.Categories) < len(s.specs) || snap.LastSyncedAt.IsZero() {
		return true
	}
	if now.Sub(snap.LastSyncedAt) >= s.checkInterval() {
		return true
	}
	loc := timezone.Location()
	localNow := now.In(loc)
	todaySlot := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), s.syncHour(), 0, 0, 0, loc)
	return localNow.After(todaySlot) && snap.LastSyncedAt.In(loc).Before(todaySlot)
}

func nextArenaSyncDelay(now time.Time, syncHour int, checkInterval time.Duration) time.Duration {
	if checkInterval < time.Minute {
		checkInterval = 6 * time.Hour
	}
	loc := timezone.Location()
	localNow := now.In(loc)
	dailySlot := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), syncHour, 0, 0, 0, loc)
	if !dailySlot.After(localNow) {
		dailySlot = dailySlot.Add(24 * time.Hour)
	}
	dailyDelay := dailySlot.Sub(localNow)
	delay := checkInterval
	if dailyDelay < delay {
		delay = dailyDelay
	}
	if delay < time.Minute {
		return time.Minute
	}
	return delay
}

func buildCategoryBoard(
	spec ArenaCategorySpec,
	publishDate string,
	prevDate string,
	totalModels int,
	currentRows []ArenaRawRow,
	prevRows []ArenaRawRow,
) ArenaCategoryBoard {
	hasPrev := prevDate != "" && len(prevRows) > 0
	prevByName := make(map[string]ArenaRawRow, len(prevRows))
	for _, r := range prevRows {
		key := strings.ToLower(strings.TrimSpace(r.ModelName))
		if key != "" {
			prevByName[key] = r
		}
	}

	entries := make([]ArenaModelEntry, 0, len(currentRows))
	highlights := ArenaCategoryHighlights{}

	for i, row := range currentRows {
		rank := row.Rank
		if rank <= 0 {
			rank = i + 1
		}
		score := roundOneDecimal(row.Rating)
		ratingLower := roundOneDecimal(row.RatingLower)
		ratingUpper := roundOneDecimal(row.RatingUpper)
		ci := computeCI(row.Rating, row.RatingLower, row.RatingUpper)
		isOpen := IsOpenArenaLicense(row.License)

		entry := ArenaModelEntry{
			Rank:         rank,
			ModelName:    strings.TrimSpace(row.ModelName),
			Organization: normalizeOrg(row.Organization),
			License:      strings.TrimSpace(row.License),
			IsOpen:       isOpen,
			Score:        score,
			CI:           ci,
			RatingLower:  ratingLower,
			RatingUpper:  ratingUpper,
			Votes:        row.VoteCount,
		}

		if hasPrev {
			key := strings.ToLower(entry.ModelName)
			if prev, ok := prevByName[key]; ok {
				prevRank := prev.Rank
				if prevRank > 0 {
					rd := prevRank - rank
					entry.RankDelta = &rd
				}
				sd := roundOneDecimal(row.Rating - prev.Rating)
				entry.ScoreDelta = &sd
			} else {
				entry.IsNew = true
				highlights.NewEntries++
			}
		}

		if isOpen {
			highlights.OpenModelCount++
			if highlights.TopOpenModel == "" {
				highlights.TopOpenModel = entry.ModelName
				highlights.TopOpenOrganization = entry.Organization
				highlights.TopOpenRank = entry.Rank
				highlights.TopOpenScore = entry.Score
			}
		}

		if entry.RankDelta != nil && *entry.RankDelta > 0 {
			if highlights.BiggestRiser == nil ||
				*entry.RankDelta > highlights.BiggestRiser.RankDelta ||
				(*entry.RankDelta == highlights.BiggestRiser.RankDelta && entry.Rank < highlights.BiggestRiser.Rank) {
				var sdCopy *float64
				if entry.ScoreDelta != nil {
					v := *entry.ScoreDelta
					sdCopy = &v
				}
				highlights.BiggestRiser = &ArenaBiggestRiser{
					ModelName:    entry.ModelName,
					Organization: entry.Organization,
					Rank:         entry.Rank,
					RankDelta:    *entry.RankDelta,
					ScoreDelta:   sdCopy,
				}
			}
		}

		entries = append(entries, entry)
	}

	if len(entries) > 0 {
		highlights.Leader = entries[0].ModelName
		highlights.LeaderOrganization = entries[0].Organization
		highlights.LeaderScore = entries[0].Score
	}
	if len(entries) > 1 {
		highlights.RunnerUp = entries[1].ModelName
		highlights.RunnerUpGap = roundOneDecimal(entries[0].Score - entries[1].Score)
	}

	return ArenaCategoryBoard{
		ID:                  spec.ID,
		Subset:              spec.Subset,
		StyleControlled:     spec.StyleControlled,
		PublishDate:         publishDate,
		PreviousPublishDate: prevDate,
		TotalModels:         totalModels,
		Highlights:          highlights,
		Entries:             entries,
	}
}

func normalizeAndSortRows(rows []ArenaRawRow, limit int) []ArenaRawRow {
	seen := make(map[string]struct{}, len(rows))
	out := make([]ArenaRawRow, 0, len(rows))
	for _, r := range rows {
		name := strings.TrimSpace(r.ModelName)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		r.ModelName = name
		out = append(out, r)
	}

	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := out[i].Rank, out[j].Rank
		if ri > 0 && rj > 0 && ri != rj {
			return ri < rj
		}
		if out[i].Rating != out[j].Rating {
			return out[i].Rating > out[j].Rating
		}
		return out[i].ModelName < out[j].ModelName
	})

	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func extractLatestPublishDate(rows []ArenaRawRow) string {
	latest := ""
	for _, r := range rows {
		d := strings.TrimSpace(r.LeaderboardPublishDate)
		if isValidArenaDate(d) && d > latest {
			latest = d
		}
	}
	return latest
}

func computeCI(rating, lower, upper float64) int {
	if upper > lower {
		return int(math.Round((upper - lower) / 2.0))
	}
	if upper > rating {
		return int(math.Round(upper - rating))
	}
	if rating > lower && lower > 0 {
		return int(math.Round(rating - lower))
	}
	return 0
}

func roundOneDecimal(v float64) float64 {
	return math.Round(v*10) / 10
}

func normalizeOrg(org string) string {
	return strings.TrimSpace(org)
}

// IsOpenArenaLicense classifies whether an LMArena license label denotes an open-weight / open-source license.
func IsOpenArenaLicense(license string) bool {
	s := strings.ToLower(strings.TrimSpace(license))
	if s == "" {
		return false
	}
	if strings.Contains(s, "proprietary") || strings.Contains(s, "closed") || s == "api" {
		return false
	}
	return true
}

func isValidArenaDate(d string) bool {
	return arenaDatePattern.MatchString(strings.TrimSpace(d))
}

func (s *ArenaLeaderboardService) isValidCategory(id string) bool {
	for _, spec := range s.specs {
		if spec.ID == id {
			return true
		}
	}
	return false
}

func (s *ArenaLeaderboardService) sourceInfo() ArenaSourceInfo {
	dataset := arenaDefaultDataset
	if s.cfg != nil && strings.TrimSpace(s.cfg.ArenaLeaderboard.Dataset) != "" {
		dataset = strings.TrimSpace(s.cfg.ArenaLeaderboard.Dataset)
	}
	return ArenaSourceInfo{
		Name:       arenaSourceName,
		SiteURL:    arenaSourceSiteURL,
		DatasetURL: "https://huggingface.co/datasets/" + dataset,
	}
}

func (s *ArenaLeaderboardService) dataDir() string {
	if s.cfg != nil {
		if dir := strings.TrimSpace(s.cfg.ArenaLeaderboard.DataDir); dir != "" {
			return dir
		}
		if base := strings.TrimSpace(s.cfg.Pricing.DataDir); base != "" {
			return filepath.Join(base, "arena_leaderboard")
		}
	}
	return filepath.Join(".", "data", "arena_leaderboard")
}

func (s *ArenaLeaderboardService) syncHour() int {
	if s.cfg != nil && s.cfg.ArenaLeaderboard.SyncHour >= 0 && s.cfg.ArenaLeaderboard.SyncHour <= 23 {
		return s.cfg.ArenaLeaderboard.SyncHour
	}
	return 8
}

func (s *ArenaLeaderboardService) checkInterval() time.Duration {
	if s.cfg != nil && s.cfg.ArenaLeaderboard.CheckIntervalMinutes >= 5 {
		return time.Duration(s.cfg.ArenaLeaderboard.CheckIntervalMinutes) * time.Minute
	}
	return 6 * time.Hour
}

func (s *ArenaLeaderboardService) requestTimeout() time.Duration {
	if s.cfg != nil && s.cfg.ArenaLeaderboard.RequestTimeoutSeconds >= 5 {
		return time.Duration(s.cfg.ArenaLeaderboard.RequestTimeoutSeconds) * time.Second
	}
	return 20 * time.Second
}

func (s *ArenaLeaderboardService) topLimit() int {
	if s.cfg != nil && s.cfg.ArenaLeaderboard.TopLimit >= 10 && s.cfg.ArenaLeaderboard.TopLimit <= 200 {
		return s.cfg.ArenaLeaderboard.TopLimit
	}
	return 100
}

func (s *ArenaLeaderboardService) bootstrapHistoryEnabled() bool {
	if s.cfg == nil {
		return true
	}
	return s.cfg.ArenaLeaderboard.BootstrapHistory
}

func (s *ArenaLeaderboardService) snapshotFilePath() string {
	return filepath.Join(s.dataDir(), arenaSnapshotFilename)
}

func (s *ArenaLeaderboardService) historyDir(categoryID string) string {
	return filepath.Join(s.dataDir(), "history", categoryID)
}

func (s *ArenaLeaderboardService) loadSnapshotFromDisk() (*ArenaLeaderboardSnapshot, error) {
	raw, err := os.ReadFile(s.snapshotFilePath())
	if err != nil {
		return nil, err
	}
	var snap ArenaLeaderboardSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, err
	}
	if len(snap.Categories) == 0 {
		return nil, errors.New("empty cached arena snapshot")
	}
	return &snap, nil
}

func (s *ArenaLeaderboardService) saveSnapshotToDisk(snap *ArenaLeaderboardSnapshot) error {
	if err := os.MkdirAll(s.dataDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(s.snapshotFilePath(), data)
}

func (s *ArenaLeaderboardService) saveHistorySnapshot(categoryID, publishDate string, rows []ArenaRawRow) error {
	publishDate = strings.TrimSpace(publishDate)
	if !isValidArenaDate(publishDate) || !s.isValidCategory(categoryID) {
		return nil
	}
	dir := s.historyDir(categoryID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	payload := arenaHistoryFile{
		CategoryID:  categoryID,
		PublishDate: publishDate,
		SavedAt:     s.nowFn(),
		Rows:        rows,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := writeFileAtomic(filepath.Join(dir, publishDate+".json"), data); err != nil {
		return err
	}
	s.pruneHistoryDir(dir)
	return nil
}

func (s *ArenaLeaderboardService) pruneHistoryDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	dates := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		if isValidArenaDate(name) {
			dates = append(dates, name)
		}
	}
	if len(dates) <= arenaMaxHistoryFiles {
		return
	}
	sort.Strings(dates)
	toDelete := dates[:len(dates)-arenaMaxHistoryFiles]
	for _, d := range toDelete {
		_ = os.Remove(filepath.Join(dir, d+".json"))
	}
}

func (s *ArenaLeaderboardService) findPreviousLocalHistory(categoryID, currentPublishDate string) (string, []ArenaRawRow) {
	dir := s.historyDir(categoryID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil
	}
	dates := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		d := strings.TrimSuffix(e.Name(), ".json")
		if !isValidArenaDate(d) {
			continue
		}
		if currentPublishDate != "" && d >= currentPublishDate {
			continue
		}
		dates = append(dates, d)
	}
	if len(dates) == 0 {
		return "", nil
	}
	sort.Strings(dates)
	prevDate := dates[len(dates)-1]
	raw, err := os.ReadFile(filepath.Join(dir, prevDate+".json"))
	if err != nil {
		return "", nil
	}
	var hf arenaHistoryFile
	if err := json.Unmarshal(raw, &hf); err != nil || len(hf.Rows) == 0 {
		return "", nil
	}
	return prevDate, hf.Rows
}

func writeFileAtomic(targetPath string, data []byte) error {
	dir := filepath.Dir(targetPath)
	tmp, err := os.CreateTemp(dir, ".arena-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	success := false
	defer func() {
		if !success {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return err
	}
	success = true
	return nil
}

func cloneCategoryBoard(b ArenaCategoryBoard) ArenaCategoryBoard {
	out := b
	if b.Highlights.BiggestRiser != nil {
		br := *b.Highlights.BiggestRiser
		if br.ScoreDelta != nil {
			v := *br.ScoreDelta
			br.ScoreDelta = &v
		}
		out.Highlights.BiggestRiser = &br
	}
	if len(b.Entries) > 0 {
		out.Entries = make([]ArenaModelEntry, len(b.Entries))
		for i, e := range b.Entries {
			cp := e
			if e.RankDelta != nil {
				v := *e.RankDelta
				cp.RankDelta = &v
			}
			if e.ScoreDelta != nil {
				v := *e.ScoreDelta
				cp.ScoreDelta = &v
			}
			out.Entries[i] = cp
		}
	}
	return out
}

// ProvideArenaLeaderboardService constructs and initializes ArenaLeaderboardService for Wire DI.
func ProvideArenaLeaderboardService(cfg *config.Config, remoteClient ArenaLeaderboardRemoteClient) (*ArenaLeaderboardService, error) {
	svc := NewArenaLeaderboardService(cfg, remoteClient)
	if err := svc.Initialize(); err != nil {
		logger.LegacyPrintf("service.arena", "[ArenaLeaderboard] Initialization warning: %v", err)
	}
	return svc, nil
}
