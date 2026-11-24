package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	arenaRemoteMaxBodyBytes = 8 * 1024 * 1024 // 8 MB safety cap per response
	arenaRemoteMaxRetries   = 2
	arenaRemoteUserAgent    = "Gabugu-ArenaSync/1.0"
)

type arenaLeaderboardRemoteClient struct {
	httpClient *http.Client
	baseURL    string
	dataset    string
}

type arenaLeaderboardRemoteClientError struct {
	err error
}

func (c *arenaLeaderboardRemoteClientError) FetchLatestCategory(_ context.Context, _, _ string, _ int) ([]service.ArenaRawRow, int, error) {
	return nil, 0, c.err
}

func (c *arenaLeaderboardRemoteClientError) FetchPreviousSnapshot(_ context.Context, _, _, _ string, _ int) (string, []service.ArenaRawRow, error) {
	return "", nil, c.err
}

// ProvideArenaLeaderboardRemoteClient constructs the Hugging Face Datasets Server client for Arena sync.
func ProvideArenaLeaderboardRemoteClient(cfg *config.Config) service.ArenaLeaderboardRemoteClient {
	baseURL := "https://datasets-server.huggingface.co"
	dataset := "lmarena-ai/leaderboard-dataset"
	timeout := 20 * time.Second
	proxyURL := ""
	allowDirect := false

	if cfg != nil {
		if strings.TrimSpace(cfg.ArenaLeaderboard.BaseURL) != "" {
			baseURL = strings.TrimSpace(cfg.ArenaLeaderboard.BaseURL)
		}
		if strings.TrimSpace(cfg.ArenaLeaderboard.Dataset) != "" {
			dataset = strings.TrimSpace(cfg.ArenaLeaderboard.Dataset)
		}
		if cfg.ArenaLeaderboard.RequestTimeoutSeconds >= 5 {
			timeout = time.Duration(cfg.ArenaLeaderboard.RequestTimeoutSeconds) * time.Second
		}
		proxyURL = cfg.Update.ProxyURL
		allowDirect = cfg.Security.ProxyFallback.AllowDirectOnError
	}

	return NewArenaLeaderboardRemoteClient(baseURL, dataset, timeout, proxyURL, allowDirect)
}

// NewArenaLeaderboardRemoteClient creates a remote client for Hugging Face Datasets Server.
func NewArenaLeaderboardRemoteClient(
	baseURL string,
	dataset string,
	timeout time.Duration,
	proxyURL string,
	allowDirectOnProxyError bool,
) service.ArenaLeaderboardRemoteClient {
	if timeout < 5*time.Second {
		timeout = 20 * time.Second
	}
	// Keep http.Client.Timeout at least 45s so longer per-call contexts (e.g. 40s history bootstrap)
	// are controlled by their context deadline rather than prematurely truncated by the transport client.
	clientTimeout := timeout
	if clientTimeout < 45*time.Second {
		clientTimeout = 45 * time.Second
	}
	sharedClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:  clientTimeout,
		ProxyURL: proxyURL,
	})
	if err != nil {
		if strings.TrimSpace(proxyURL) != "" && !allowDirectOnProxyError {
			slog.Warn("proxy client init failed, arena leaderboard sync will fail", "service", "arena_leaderboard", "error", err)
			return &arenaLeaderboardRemoteClientError{
				err: fmt.Errorf("proxy client init failed and direct fallback is disabled: %w", err),
			}
		}
		sharedClient = &http.Client{Timeout: clientTimeout}
	}
	return &arenaLeaderboardRemoteClient{
		httpClient: sharedClient,
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		dataset:    strings.TrimSpace(dataset),
	}
}

type hfFilterResponse struct {
	Rows         []hfFilterRowWrapper `json:"rows"`
	NumRowsTotal int                  `json:"num_rows_total"`
}

type hfFilterRowWrapper struct {
	Row hfLeaderboardRow `json:"row"`
}

// hfLeaderboardRow accepts both integer and floating-point JSON numbers for rank/vote_count,
// as observed across LMArena subsets (e.g. text emits 1.0, webdev emits 1).
type hfLeaderboardRow struct {
	ModelName              string   `json:"model_name"`
	Organization           *string  `json:"organization"`
	License                *string  `json:"license"`
	Rating                 *float64 `json:"rating"`
	RatingLower            *float64 `json:"rating_lower"`
	RatingUpper            *float64 `json:"rating_upper"`
	VoteCount              *float64 `json:"vote_count"`
	Rank                   *float64 `json:"rank"`
	Category               *string  `json:"category"`
	LeaderboardPublishDate *string  `json:"leaderboard_publish_date"`
}

func (c *arenaLeaderboardRemoteClient) FetchLatestCategory(
	ctx context.Context,
	subset string,
	filterCategory string,
	limit int,
) ([]service.ArenaRawRow, int, error) {
	if limit <= 0 {
		limit = 100
	}
	where := fmt.Sprintf(`"category"='%s'`, escapeHFSQLValue(filterCategory))
	resp, err := c.queryFilter(ctx, subset, "latest", where, `"rank" ASC`, 0, limit)
	if err != nil {
		return nil, 0, err
	}
	return decodeHFRows(resp.Rows), resp.NumRowsTotal, nil
}

func (c *arenaLeaderboardRemoteClient) FetchPreviousSnapshot(
	ctx context.Context,
	subset string,
	filterCategory string,
	currentPublishDate string,
	limit int,
) (string, []service.ArenaRawRow, error) {
	if limit <= 0 {
		limit = 100
	}
	catEscaped := escapeHFSQLValue(filterCategory)
	dateWhere := fmt.Sprintf(`"category"='%s' AND "rank"=1`, catEscaped)
	dateResp, err := c.queryFilter(ctx, subset, "full", dateWhere, `"leaderboard_publish_date" DESC`, 0, 10)
	if err != nil {
		return "", nil, err
	}

	prevDate := ""
	for _, r := range decodeHFRows(dateResp.Rows) {
		d := strings.TrimSpace(r.LeaderboardPublishDate)
		if d != "" && d < currentPublishDate {
			if prevDate == "" || d > prevDate {
				prevDate = d
			}
		}
	}
	if prevDate == "" {
		return "", nil, nil
	}

	rowsWhere := fmt.Sprintf(`"category"='%s' AND "leaderboard_publish_date"='%s'`, catEscaped, escapeHFSQLValue(prevDate))
	rowsResp, err := c.queryFilter(ctx, subset, "full", rowsWhere, `"rank" ASC`, 0, limit)
	if err != nil {
		return "", nil, err
	}
	return prevDate, decodeHFRows(rowsResp.Rows), nil
}

func (c *arenaLeaderboardRemoteClient) queryFilter(
	ctx context.Context,
	subset string,
	split string,
	where string,
	orderBy string,
	offset int,
	length int,
) (*hfFilterResponse, error) {
	q := url.Values{}
	q.Set("dataset", c.dataset)
	q.Set("config", subset)
	q.Set("split", split)
	q.Set("where", where)
	if orderBy != "" {
		q.Set("orderby", orderBy)
	}
	q.Set("offset", strconv.Itoa(offset))
	q.Set("length", strconv.Itoa(length))

	endpoint := c.baseURL + "/filter?" + q.Encode()

	var lastErr error
	for attempt := 0; attempt <= arenaRemoteMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * 800 * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", arenaRemoteUserAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, arenaRemoteMaxBodyBytes+1))
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if int64(len(body)) > arenaRemoteMaxBodyBytes {
			return nil, fmt.Errorf("arena upstream response exceeded %d bytes", arenaRemoteMaxBodyBytes)
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("HTTP %d from arena upstream", resp.StatusCode)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d from arena upstream", resp.StatusCode)
		}

		var parsed hfFilterResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("decode arena upstream JSON: %w", err)
		}
		return &parsed, nil
	}

	return nil, lastErr
}

func decodeHFRows(wrappers []hfFilterRowWrapper) []service.ArenaRawRow {
	out := make([]service.ArenaRawRow, 0, len(wrappers))
	for _, w := range wrappers {
		r := w.Row
		if strings.TrimSpace(r.ModelName) == "" {
			continue
		}
		raw := service.ArenaRawRow{
			ModelName: strings.TrimSpace(r.ModelName),
		}
		if r.Organization != nil {
			raw.Organization = strings.TrimSpace(*r.Organization)
		}
		if r.License != nil {
			raw.License = strings.TrimSpace(*r.License)
		}
		if r.Rating != nil {
			raw.Rating = *r.Rating
		}
		if r.RatingLower != nil {
			raw.RatingLower = *r.RatingLower
		}
		if r.RatingUpper != nil {
			raw.RatingUpper = *r.RatingUpper
		}
		if r.VoteCount != nil {
			raw.VoteCount = int(math.Round(*r.VoteCount))
		}
		if r.Rank != nil {
			raw.Rank = int(math.Round(*r.Rank))
		}
		if r.Category != nil {
			raw.Category = strings.TrimSpace(*r.Category)
		}
		if r.LeaderboardPublishDate != nil {
			raw.LeaderboardPublishDate = strings.TrimSpace(*r.LeaderboardPublishDate)
		}
		out = append(out, raw)
	}
	return out
}

func escapeHFSQLValue(val string) string {
	return strings.ReplaceAll(val, "'", "''")
}
