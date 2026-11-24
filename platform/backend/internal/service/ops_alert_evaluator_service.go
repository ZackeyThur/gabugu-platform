package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	opsAlertEvaluatorJobName = "ops_alert_evaluator"

	opsAlertEvaluatorTimeout         = 45 * time.Second
	opsAlertEvaluatorLeaderLockKey   = "ops:alert:evaluator:leader"
	opsAlertEvaluatorLeaderLockTTL   = 90 * time.Second
	opsAlertEvaluatorSkipLogInterval = 1 * time.Minute
)

var opsAlertEvaluatorReleaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

type OpsAlertEvaluatorService struct {
	opsService   *OpsService
	opsRepo      OpsRepository
	emailService *EmailService
	proxyRepo    ProxyRepository

	redisClient *redis.Client
	cfg         *config.Config
	instanceID  string

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	mu         sync.Mutex
	ruleStates map[int64]*opsAlertRuleState

	emailLimiter *slidingWindowLimiter

	skipLogMu sync.Mutex
	skipLogAt time.Time

	warnNoRedisOnce sync.Once
}

type opsAlertRuleState struct {
	LastEvaluatedAt     time.Time
	ConsecutiveBreaches int
}

func NewOpsAlertEvaluatorService(
	opsService *OpsService,
	opsRepo OpsRepository,
	emailService *EmailService,
	redisClient *redis.Client,
	cfg *config.Config,
	proxyRepo ProxyRepository,
) *OpsAlertEvaluatorService {
	return &OpsAlertEvaluatorService{
		opsService:   opsService,
		opsRepo:      opsRepo,
		emailService: emailService,
		proxyRepo:    proxyRepo,
		redisClient:  redisClient,
		cfg:          cfg,
		instanceID:   uuid.NewString(),
		ruleStates:   map[int64]*opsAlertRuleState{},
		emailLimiter: newSlidingWindowLimiter(0, time.Hour),
	}
}

func (s *OpsAlertEvaluatorService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		if s.stopCh == nil {
			s.stopCh = make(chan struct{})
		}
		s.wg.Add(1)
		go s.run()
	})
}

func (s *OpsAlertEvaluatorService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}
	})
	s.wg.Wait()
}

func (s *OpsAlertEvaluatorService) run() {
	defer s.wg.Done()

	// Start immediately to produce early feedback in ops dashboard.
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			interval := s.getInterval()
			s.evaluateOnce(interval)
			timer.Reset(interval)
		case <-s.stopCh:
			return
		}
	}
}

func (s *OpsAlertEvaluatorService) getInterval() time.Duration {
	// Default.
	interval := 60 * time.Second

	if s == nil || s.opsService == nil {
		return interval
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg, err := s.opsService.GetOpsAlertRuntimeSettings(ctx)
	if err != nil || cfg == nil {
		return interval
	}
	if cfg.EvaluationIntervalSeconds <= 0 {
		return interval
	}
	if cfg.EvaluationIntervalSeconds < 1 {
		return interval
	}
	if cfg.EvaluationIntervalSeconds > int((24 * time.Hour).Seconds()) {
		return interval
	}
	return time.Duration(cfg.EvaluationIntervalSeconds) * time.Second
}

func (s *OpsAlertEvaluatorService) evaluateOnce(interval time.Duration) {
	if s == nil || s.opsRepo == nil {
		return
	}
	if s.cfg != nil && !s.cfg.Ops.Enabled {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), opsAlertEvaluatorTimeout)
	defer cancel()

	if s.opsService != nil && !s.opsService.IsMonitoringEnabled(ctx) {
		return
	}

	runtimeCfg := defaultOpsAlertRuntimeSettings()
	if s.opsService != nil {
		if loaded, err := s.opsService.GetOpsAlertRuntimeSettings(ctx); err == nil && loaded != nil {
			runtimeCfg = loaded
		}
	}

	release, ok := s.tryAcquireLeaderLock(ctx, runtimeCfg.DistributedLock)
	if !ok {
		return
	}
	if release != nil {
		defer release()
	}

	startedAt := time.Now().UTC()
	runAt := startedAt

	rules, err := s.opsRepo.ListAlertRules(ctx)
	if err != nil {
		s.recordHeartbeatError(runAt, time.Since(startedAt), err)
		logger.LegacyPrintf("service.ops_alert_evaluator", "[OpsAlertEvaluator] list rules failed: %v", err)
		return
	}

	rulesTotal := len(rules)
	rulesEnabled := 0
	rulesEvaluated := 0
	eventsCreated := 0
	eventsResolved := 0
	emailsSent := 0

	now := time.Now().UTC()
	safeEnd := now.Truncate(time.Minute)
	if safeEnd.IsZero() {
		safeEnd = now
	}

	systemMetrics, _ := s.opsRepo.GetLatestSystemMetrics(ctx, 1)

	// Cleanup stale state for removed rules.
	s.pruneRuleStates(rules)

	for _, rule := range rules {
		if rule == nil || !rule.Enabled || rule.ID <= 0 {
			continue
		}
		rulesEnabled++

		scopePlatform, scopeGroupID, scopeRegion := parseOpsAlertRuleScope(rule.Filters)

		windowMinutes := rule.WindowMinutes
		if windowMinutes <= 0 {
			windowMinutes = 1
		}
		windowStart := safeEnd.Add(-time.Duration(windowMinutes) * time.Minute)
		windowEnd := safeEnd

		metricSnap := s.evaluateRuleMetric(ctx, rule, systemMetrics, windowStart, windowEnd, scopePlatform, scopeGroupID)
		if !metricSnap.Valid {
			s.resetRuleState(rule.ID, now)
			continue
		}
		metricValue := metricSnap.Value
		rulesEvaluated++

		rawBreached := compareMetric(metricValue, rule.Operator, rule.Threshold)
		breachedNow := rawBreached
		if breachedNow && shouldSuppressLowSampleBreach(rule, metricSnap, runtimeCfg) {
			breachedNow = false
		}
		required := requiredSustainedBreaches(rule.SustainedMinutes, interval)
		consecutive := s.updateRuleBreaches(rule.ID, now, interval, breachedNow)

		activeEvent, err := s.opsRepo.GetActiveAlertEvent(ctx, rule.ID)
		if err != nil {
			logger.LegacyPrintf("service.ops_alert_evaluator", "[OpsAlertEvaluator] get active event failed (rule=%d): %v", rule.ID, err)
			continue
		}

		if breachedNow && consecutive >= required {
			if activeEvent != nil {
				continue
			}

			// Scoped silencing: if a matching silence exists, skip creating a firing event.
			if s.opsService != nil {
				platform := strings.TrimSpace(scopePlatform)
				region := scopeRegion
				if platform != "" {
					if ok, err := s.opsService.IsAlertSilenced(ctx, rule.ID, platform, scopeGroupID, region, now); err == nil && ok {
						continue
					}
				}
			}

			latestEvent, err := s.opsRepo.GetLatestAlertEvent(ctx, rule.ID)
			if err != nil {
				logger.LegacyPrintf("service.ops_alert_evaluator", "[OpsAlertEvaluator] get latest event failed (rule=%d): %v", rule.ID, err)
				continue
			}
			if latestEvent != nil && rule.CooldownMinutes > 0 {
				cooldown := time.Duration(rule.CooldownMinutes) * time.Minute
				if now.Sub(latestEvent.FiredAt) < cooldown {
					continue
				}
			}

			scopeGroupName := ""
			if scopeGroupID != nil && *scopeGroupID > 0 {
				if metricSnap.Availability != nil && metricSnap.Availability.Group != nil && strings.TrimSpace(metricSnap.Availability.Group.GroupName) != "" {
					scopeGroupName = strings.TrimSpace(metricSnap.Availability.Group.GroupName)
				} else if s.opsService != nil {
					subCtx, subCancel := context.WithTimeout(ctx, 2*time.Second)
					if avail, aErr := s.opsService.GetAccountAvailability(subCtx, scopePlatform, scopeGroupID); aErr == nil && avail != nil && avail.Group != nil {
						scopeGroupName = strings.TrimSpace(avail.Group.GroupName)
					}
					subCancel()
				}
			}

			firedEvent := &OpsAlertEvent{
				RuleID:         rule.ID,
				Severity:       strings.TrimSpace(rule.Severity),
				Status:         OpsAlertStatusFiring,
				Title:          fmt.Sprintf("%s: %s", strings.TrimSpace(rule.Severity), strings.TrimSpace(rule.Name)),
				Description:    buildOpsAlertDescriptionWithGroupName(rule, metricValue, windowMinutes, scopePlatform, scopeGroupID, scopeGroupName),
				MetricValue:    float64Ptr(metricValue),
				ThresholdValue: float64Ptr(rule.Threshold),
				Dimensions:     buildOpsAlertDimensions(scopePlatform, scopeGroupID),
				FiredAt:        now,
				CreatedAt:      now,
			}

			created, err := s.opsRepo.CreateAlertEvent(ctx, firedEvent)
			if err != nil {
				logger.LegacyPrintf("service.ops_alert_evaluator", "[OpsAlertEvaluator] create event failed (rule=%d): %v", rule.ID, err)
				continue
			}

			eventsCreated++
			if created != nil && created.ID > 0 {
				if s.deliverOpsAlertNotification(ctx, runtimeCfg, rule, created, metricSnap, windowStart, windowEnd, scopePlatform, scopeGroupID, false) {
					emailsSent++
				}
			}
			continue
		}

		// Not breached: resolve active event only when the raw metric actually recovered
		// (never auto-resolve an active outage merely because traffic dipped below min_sample_requests).
		if activeEvent != nil && !rawBreached {
			resolvedAt := now
			if err := s.opsRepo.UpdateAlertEventStatus(ctx, activeEvent.ID, OpsAlertStatusResolved, &resolvedAt); err != nil {
				logger.LegacyPrintf("service.ops_alert_evaluator", "[OpsAlertEvaluator] resolve event failed (event=%d): %v", activeEvent.ID, err)
			} else {
				eventsResolved++
				resolvedEvent := *activeEvent
				resolvedEvent.Status = OpsAlertStatusResolved
				resolvedEvent.ResolvedAt = &resolvedAt
				resolvedEvent.MetricValue = float64Ptr(metricValue)
				if s.deliverOpsAlertNotification(ctx, runtimeCfg, rule, &resolvedEvent, metricSnap, windowStart, windowEnd, scopePlatform, scopeGroupID, true) {
					emailsSent++
				}
			}
		}
	}

	result := truncateString(fmt.Sprintf("rules=%d enabled=%d evaluated=%d created=%d resolved=%d emails_sent=%d", rulesTotal, rulesEnabled, rulesEvaluated, eventsCreated, eventsResolved, emailsSent), 2048)
	s.recordHeartbeatSuccess(runAt, time.Since(startedAt), result)
}

func (s *OpsAlertEvaluatorService) pruneRuleStates(rules []*OpsAlertRule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	live := map[int64]struct{}{}
	for _, r := range rules {
		if r != nil && r.ID > 0 {
			live[r.ID] = struct{}{}
		}
	}
	for id := range s.ruleStates {
		if _, ok := live[id]; !ok {
			delete(s.ruleStates, id)
		}
	}
}

func (s *OpsAlertEvaluatorService) resetRuleState(ruleID int64, now time.Time) {
	if ruleID <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.ruleStates[ruleID]
	if !ok {
		state = &opsAlertRuleState{}
		s.ruleStates[ruleID] = state
	}
	state.LastEvaluatedAt = now
	state.ConsecutiveBreaches = 0
}

func (s *OpsAlertEvaluatorService) updateRuleBreaches(ruleID int64, now time.Time, interval time.Duration, breached bool) int {
	if ruleID <= 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.ruleStates[ruleID]
	if !ok {
		state = &opsAlertRuleState{}
		s.ruleStates[ruleID] = state
	}

	if !state.LastEvaluatedAt.IsZero() && interval > 0 {
		if now.Sub(state.LastEvaluatedAt) > interval*2 {
			state.ConsecutiveBreaches = 0
		}
	}

	state.LastEvaluatedAt = now
	if breached {
		state.ConsecutiveBreaches++
	} else {
		state.ConsecutiveBreaches = 0
	}
	return state.ConsecutiveBreaches
}

func requiredSustainedBreaches(sustainedMinutes int, interval time.Duration) int {
	if sustainedMinutes <= 0 {
		return 1
	}
	if interval <= 0 {
		return sustainedMinutes
	}
	required := int(math.Ceil(float64(sustainedMinutes*60) / interval.Seconds()))
	if required < 1 {
		return 1
	}
	return required
}

func parseOpsAlertRuleScope(filters map[string]any) (platform string, groupID *int64, region *string) {
	if filters == nil {
		return "", nil, nil
	}
	if v, ok := filters["platform"]; ok {
		if s, ok := v.(string); ok {
			platform = strings.TrimSpace(s)
		}
	}
	if v, ok := filters["group_id"]; ok {
		switch t := v.(type) {
		case float64:
			if t > 0 {
				id := int64(t)
				groupID = &id
			}
		case int64:
			if t > 0 {
				id := t
				groupID = &id
			}
		case int:
			if t > 0 {
				id := int64(t)
				groupID = &id
			}
		case string:
			n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
			if err == nil && n > 0 {
				groupID = &n
			}
		}
	}
	if v, ok := filters["region"]; ok {
		if s, ok := v.(string); ok {
			vv := strings.TrimSpace(s)
			if vv != "" {
				region = &vv
			}
		}
	}
	return platform, groupID, region
}

type opsAlertMetricSnapshot struct {
	Value         float64
	Valid         bool
	Overview      *OpsDashboardOverview
	Availability  *OpsAccountAvailability
	SystemMetrics *OpsSystemMetricsSnapshot
}

func (s *OpsAlertEvaluatorService) computeRuleMetric(
	ctx context.Context,
	rule *OpsAlertRule,
	systemMetrics *OpsSystemMetricsSnapshot,
	start time.Time,
	end time.Time,
	platform string,
	groupID *int64,
) (float64, bool) {
	snap := s.evaluateRuleMetric(ctx, rule, systemMetrics, start, end, platform, groupID)
	return snap.Value, snap.Valid
}

func (s *OpsAlertEvaluatorService) evaluateRuleMetric(
	ctx context.Context,
	rule *OpsAlertRule,
	systemMetrics *OpsSystemMetricsSnapshot,
	start time.Time,
	end time.Time,
	platform string,
	groupID *int64,
) opsAlertMetricSnapshot {
	if rule == nil {
		return opsAlertMetricSnapshot{}
	}
	switch strings.TrimSpace(rule.MetricType) {
	case "cpu_usage_percent":
		if systemMetrics != nil && systemMetrics.CPUUsagePercent != nil {
			return opsAlertMetricSnapshot{Value: *systemMetrics.CPUUsagePercent, Valid: true, SystemMetrics: systemMetrics}
		}
		return opsAlertMetricSnapshot{}
	case "memory_usage_percent":
		if systemMetrics != nil && systemMetrics.MemoryUsagePercent != nil {
			return opsAlertMetricSnapshot{Value: *systemMetrics.MemoryUsagePercent, Valid: true, SystemMetrics: systemMetrics}
		}
		return opsAlertMetricSnapshot{}
	case "concurrency_queue_depth":
		if systemMetrics != nil && systemMetrics.ConcurrencyQueueDepth != nil {
			return opsAlertMetricSnapshot{Value: float64(*systemMetrics.ConcurrencyQueueDepth), Valid: true, SystemMetrics: systemMetrics}
		}
		return opsAlertMetricSnapshot{}
	case "group_available_accounts":
		if groupID == nil || *groupID <= 0 {
			return opsAlertMetricSnapshot{}
		}
		if s == nil || s.opsService == nil {
			return opsAlertMetricSnapshot{}
		}
		availability, err := s.opsService.GetAccountAvailability(ctx, platform, groupID)
		if err != nil || availability == nil {
			return opsAlertMetricSnapshot{}
		}
		if availability.Group == nil {
			return opsAlertMetricSnapshot{Value: 0, Valid: true, Availability: availability}
		}
		return opsAlertMetricSnapshot{Value: float64(availability.Group.AvailableCount), Valid: true, Availability: availability}
	case "group_available_ratio":
		if groupID == nil || *groupID <= 0 {
			return opsAlertMetricSnapshot{}
		}
		if s == nil || s.opsService == nil {
			return opsAlertMetricSnapshot{}
		}
		availability, err := s.opsService.GetAccountAvailability(ctx, platform, groupID)
		if err != nil || availability == nil {
			return opsAlertMetricSnapshot{}
		}
		return opsAlertMetricSnapshot{Value: computeGroupAvailableRatio(availability.Group), Valid: true, Availability: availability}
	case "account_rate_limited_count":
		if s == nil || s.opsService == nil {
			return opsAlertMetricSnapshot{}
		}
		availability, err := s.opsService.GetAccountAvailability(ctx, platform, groupID)
		if err != nil || availability == nil {
			return opsAlertMetricSnapshot{}
		}
		val := float64(countAccountsByCondition(availability.Accounts, func(acc *AccountAvailability) bool {
			return acc.IsRateLimited
		}))
		return opsAlertMetricSnapshot{Value: val, Valid: true, Availability: availability}
	case "account_error_count":
		if s == nil || s.opsService == nil {
			return opsAlertMetricSnapshot{}
		}
		availability, err := s.opsService.GetAccountAvailability(ctx, platform, groupID)
		if err != nil || availability == nil {
			return opsAlertMetricSnapshot{}
		}
		val := float64(countAccountsByCondition(availability.Accounts, func(acc *AccountAvailability) bool {
			return acc.HasError && acc.TempUnschedulableUntil == nil
		}))
		return opsAlertMetricSnapshot{Value: val, Valid: true, Availability: availability}
	case "account_temp_unscheduled_count":
		if s == nil || s.opsService == nil {
			return opsAlertMetricSnapshot{}
		}
		availability, err := s.opsService.GetAccountAvailability(ctx, platform, groupID)
		if err != nil || availability == nil {
			return opsAlertMetricSnapshot{}
		}
		now := time.Now().UTC()
		val := float64(countAccountsByCondition(availability.Accounts, func(acc *AccountAvailability) bool {
			return acc.TempUnschedulableUntil != nil && now.Before(*acc.TempUnschedulableUntil)
		}))
		return opsAlertMetricSnapshot{Value: val, Valid: true, Availability: availability}
	case "group_rate_limit_ratio":
		if groupID == nil || *groupID <= 0 {
			return opsAlertMetricSnapshot{}
		}
		if s == nil || s.opsService == nil {
			return opsAlertMetricSnapshot{}
		}
		availability, err := s.opsService.GetAccountAvailability(ctx, platform, groupID)
		if err != nil || availability == nil {
			return opsAlertMetricSnapshot{}
		}
		if availability.Group == nil || availability.Group.TotalAccounts <= 0 {
			return opsAlertMetricSnapshot{Value: 0, Valid: true, Availability: availability}
		}
		val := (float64(availability.Group.RateLimitCount) / float64(availability.Group.TotalAccounts)) * 100
		return opsAlertMetricSnapshot{Value: val, Valid: true, Availability: availability}
	case "account_error_ratio":
		if s == nil || s.opsService == nil {
			return opsAlertMetricSnapshot{}
		}
		availability, err := s.opsService.GetAccountAvailability(ctx, platform, groupID)
		if err != nil || availability == nil {
			return opsAlertMetricSnapshot{}
		}
		total := int64(len(availability.Accounts))
		if total <= 0 {
			return opsAlertMetricSnapshot{Value: 0, Valid: true, Availability: availability}
		}
		errorCount := countAccountsByCondition(availability.Accounts, func(acc *AccountAvailability) bool {
			return acc.HasError && acc.TempUnschedulableUntil == nil
		})
		val := (float64(errorCount) / float64(total)) * 100
		return opsAlertMetricSnapshot{Value: val, Valid: true, Availability: availability}
	case "overload_account_count":
		if s == nil || s.opsService == nil {
			return opsAlertMetricSnapshot{}
		}
		availability, err := s.opsService.GetAccountAvailability(ctx, platform, groupID)
		if err != nil || availability == nil {
			return opsAlertMetricSnapshot{}
		}
		val := float64(countAccountsByCondition(availability.Accounts, func(acc *AccountAvailability) bool {
			return acc.IsOverloaded
		}))
		return opsAlertMetricSnapshot{Value: val, Valid: true, Availability: availability}
	case "proxy_expired_count":
		if s == nil || s.proxyRepo == nil {
			return opsAlertMetricSnapshot{}
		}
		n, err := s.proxyRepo.CountExpired(ctx)
		if err != nil {
			return opsAlertMetricSnapshot{}
		}
		return opsAlertMetricSnapshot{Value: float64(n), Valid: true}
	case "proxy_expiring_soon_count":
		if s == nil || s.proxyRepo == nil {
			return opsAlertMetricSnapshot{}
		}
		n, err := s.proxyRepo.CountExpiringSoon(ctx, time.Now())
		if err != nil {
			return opsAlertMetricSnapshot{}
		}
		return opsAlertMetricSnapshot{Value: float64(n), Valid: true}
	}

	overview, err := s.opsRepo.GetDashboardOverview(ctx, &OpsDashboardFilter{
		StartTime: start,
		EndTime:   end,
		Platform:  platform,
		GroupID:   groupID,
		QueryMode: OpsQueryModeRaw,
	})
	if err != nil || overview == nil {
		return opsAlertMetricSnapshot{}
	}

	switch strings.TrimSpace(rule.MetricType) {
	case "success_rate":
		if overview.RequestCountSLA <= 0 {
			return opsAlertMetricSnapshot{}
		}
		return opsAlertMetricSnapshot{Value: overview.SLA * 100, Valid: true, Overview: overview}
	case "error_rate":
		if overview.RequestCountSLA <= 0 {
			return opsAlertMetricSnapshot{}
		}
		return opsAlertMetricSnapshot{Value: overview.ErrorRate * 100, Valid: true, Overview: overview}
	case "upstream_error_rate":
		if overview.RequestCountSLA <= 0 {
			return opsAlertMetricSnapshot{}
		}
		return opsAlertMetricSnapshot{Value: overview.UpstreamErrorRate * 100, Valid: true, Overview: overview}
	default:
		return opsAlertMetricSnapshot{}
	}
}

func parseOpsPositiveIntFilterValue(v any) int {
	switch t := v.(type) {
	case float64:
		if t > 0 {
			return int(t)
		}
	case int64:
		if t > 0 {
			return int(t)
		}
	case int:
		if t > 0 {
			return t
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

func parseOpsAlertRuleMinSamples(filters map[string]any, runtimeCfg *OpsAlertRuntimeSettings) (minRequests, minErrors int) {
	if runtimeCfg != nil {
		if runtimeCfg.MinSampleRequests > 0 {
			minRequests = runtimeCfg.MinSampleRequests
		}
		if runtimeCfg.MinErrorCount > 0 {
			minErrors = runtimeCfg.MinErrorCount
		}
	}
	if filters == nil {
		return minRequests, minErrors
	}
	if v, ok := filters["min_requests"]; ok {
		if parsed := parseOpsPositiveIntFilterValue(v); parsed > 0 {
			minRequests = parsed
		}
	}
	if v, ok := filters["min_errors"]; ok {
		if parsed := parseOpsPositiveIntFilterValue(v); parsed > 0 {
			minErrors = parsed
		}
	}
	return minRequests, minErrors
}

func shouldSuppressLowSampleBreach(rule *OpsAlertRule, snap opsAlertMetricSnapshot, runtimeCfg *OpsAlertRuntimeSettings) bool {
	if rule == nil || snap.Overview == nil || !isRequestRateOpsMetric(rule.MetricType) {
		return false
	}
	minRequests, minErrors := parseOpsAlertRuleMinSamples(rule.Filters, runtimeCfg)
	if minRequests > 0 && snap.Overview.RequestCountSLA < int64(minRequests) {
		return true
	}
	if minErrors > 0 {
		errCount := snap.Overview.ErrorCountSLA
		if strings.TrimSpace(rule.MetricType) == "upstream_error_rate" {
			errCount = snap.Overview.UpstreamErrorCountExcl429529
		}
		if errCount < int64(minErrors) {
			return true
		}
	}
	return false
}

func compareMetric(value float64, operator string, threshold float64) bool {
	switch strings.TrimSpace(operator) {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}

func buildOpsAlertDimensions(platform string, groupID *int64) map[string]any {
	dims := map[string]any{}
	if strings.TrimSpace(platform) != "" {
		dims["platform"] = strings.TrimSpace(platform)
	}
	if groupID != nil && *groupID > 0 {
		dims["group_id"] = *groupID
	}
	if len(dims) == 0 {
		return nil
	}
	return dims
}

func buildOpsAlertDescription(rule *OpsAlertRule, value float64, windowMinutes int, platform string, groupID *int64) string {
	return buildOpsAlertDescriptionWithGroupName(rule, value, windowMinutes, platform, groupID, "")
}

func buildOpsAlertDescriptionWithGroupName(rule *OpsAlertRule, value float64, windowMinutes int, platform string, groupID *int64, groupName string) string {
	if rule == nil {
		return ""
	}
	summary := formatOpsMetricSummary(rule.MetricType, rule.Operator, float64Ptr(value), float64Ptr(rule.Threshold), "zh")
	return fmt.Sprintf("%s｜窗口：%s｜范围：%s",
		summary,
		formatOpsWindowLabel(windowMinutes, "zh"),
		formatOpsScopeLabelWithGroupName(platform, groupID, groupName, "zh"),
	)
}

func (s *OpsAlertEvaluatorService) maybeSendAlertEmail(ctx context.Context, runtimeCfg *OpsAlertRuntimeSettings, rule *OpsAlertRule, event *OpsAlertEvent) bool {
	return s.deliverOpsAlertNotification(ctx, runtimeCfg, rule, event, opsAlertMetricSnapshot{}, time.Time{}, time.Time{}, "", nil, false)
}

func (s *OpsAlertEvaluatorService) deliverOpsAlertNotification(
	ctx context.Context,
	runtimeCfg *OpsAlertRuntimeSettings,
	rule *OpsAlertRule,
	event *OpsAlertEvent,
	snap opsAlertMetricSnapshot,
	windowStart time.Time,
	windowEnd time.Time,
	scopePlatform string,
	scopeGroupID *int64,
	isResolved bool,
) bool {
	if s == nil || s.opsService == nil || event == nil || rule == nil {
		return false
	}
	if !isResolved && event.EmailSent {
		return false
	}
	if !rule.NotifyEmail {
		return false
	}

	emailCfg, err := s.opsService.GetEmailNotificationConfig(ctx)
	if err != nil || emailCfg == nil {
		return false
	}
	if !shouldSendOpsAlertEmailByMinSeverity(strings.TrimSpace(emailCfg.Alert.MinSeverity), strings.TrimSpace(rule.Severity)) {
		return false
	}
	if runtimeCfg != nil && runtimeCfg.Silencing.Enabled {
		if isOpsAlertSilenced(time.Now().UTC(), rule, event, runtimeCfg.Silencing) {
			return false
		}
	}

	emailAllowed := s.emailService != nil &&
		emailCfg.Alert.Enabled &&
		len(emailCfg.Alert.Recipients) > 0 &&
		(!isResolved || (event.EmailSent && emailCfg.Alert.IncludeResolvedAlerts))

	webhookAllowed := emailCfg.PrivateWebhook.Enabled &&
		(!isResolved || emailCfg.Alert.IncludeResolvedAlerts)

	if !emailAllowed && !webhookAllowed {
		return false
	}

	var aiAPIKey string
	var aiFallbackAPIKey string
	if emailCfg.AIDiagnosis.Enabled {
		if key, kErr := s.opsService.GetOpsAIDiagnosisAPIKey(ctx); kErr == nil {
			aiAPIKey = key
		}
		if fKey, fErr := s.opsService.GetOpsAIDiagnosisFallbackAPIKey(ctx); fErr == nil {
			aiFallbackAPIKey = fKey
		}
	}

	type preparedLocaleBundle struct {
		ev               *opsAlertEvidence
		diagnosisBadge   string
		diagnosisSummary string
		detailsHTML      string
		variables        map[string]string
		subject          string
		fallbackHTML     string
	}
	bundles := make(map[string]*preparedLocaleBundle, 2)
	prepareLocale := func(locale string) *preparedLocaleBundle {
		normLocale := normalizeNotificationLocale(locale)
		if b, ok := bundles[normLocale]; ok {
			return b
		}
		ev := s.collectOpsAlertEvidence(ctx, rule, event, snap, windowStart, windowEnd, scopePlatform, scopeGroupID, normLocale)
		diagSummary := ev.FallbackSummary
		aiRoute := ""
		if emailCfg.AIDiagnosis.Enabled {
			if aiCtx, aiCancel, ok := boundedOpsAIDiagnosisContext(ctx, opsDiagnosisMaxTotalBudget, opsDiagnosisDeliveryHeadroom); ok {
				diagSummary, aiRoute, _ = DiagnoseOpsAlertWithRoutes(aiCtx, emailCfg.AIDiagnosis, aiAPIKey, aiFallbackAPIKey, ev, normLocale)
				aiCancel()
			}
		}
		badge := opsDiagnosisBadgeForRoute(aiRoute, emailCfg.AIDiagnosis.Enabled, normLocale)
		detailsHTML := buildOpsAlertDetailsHTML(ev, normLocale)
		vars := opsAlertEmailVariablesWithEvidence(rule, event, ev, badge, diagSummary, normLocale)
		subj := buildOpsAlertEmailSubject(ev, normLocale)
		fallbackBody := buildOpsAlertFallbackEmailHTML(ev, badge, diagSummary, detailsHTML, normLocale)
		b := &preparedLocaleBundle{
			ev:               ev,
			diagnosisBadge:   badge,
			diagnosisSummary: diagSummary,
			detailsHTML:      detailsHTML,
			variables:        vars,
			subject:          subj,
			fallbackHTML:     fallbackBody,
		}
		bundles[normLocale] = b
		return b
	}

	anyEmailSent := false
	if emailAllowed {
		s.emailLimiter.SetLimit(emailCfg.Alert.RateLimitPerHour)
		reminderKey := ""
		if isResolved {
			reminderKey = "resolved"
		}
		for _, to := range emailCfg.Alert.Recipients {
			addr := strings.TrimSpace(to)
			if addr == "" {
				continue
			}
			if !s.emailLimiter.Allow(time.Now().UTC()) {
				continue
			}
			locale := "zh"
			if s.emailService.notificationEmailService != nil {
				locale = s.emailService.notificationEmailService.ResolveRecipientLocale(ctx, 0, addr)
			}
			bundle := prepareLocale(locale)
			if s.emailService.notificationEmailService != nil {
				if err := s.emailService.notificationEmailService.Send(ctx, NotificationEmailSendInput{
					Event:            NotificationEmailEventOpsAlert,
					Locale:           locale,
					RecipientEmail:   addr,
					RecipientName:    emailRecipientName(addr),
					SourceType:       "ops_alert",
					SourceID:         fmt.Sprintf("%d", event.ID),
					ReminderKey:      reminderKey,
					Variables:        bundle.variables,
					RawHTMLVariables: map[string]string{"alert_details_html": bundle.detailsHTML},
				}); err == nil {
					anyEmailSent = true
					continue
				} else if !shouldFallbackNotificationEmail(err) {
					continue
				}
			}
			if err := s.emailService.SendEmail(ctx, addr, bundle.subject, bundle.fallbackHTML); err != nil {
				continue
			}
			anyEmailSent = true
		}

		if anyEmailSent && !isResolved && s.opsRepo != nil {
			_ = s.opsRepo.UpdateAlertEventEmailSent(context.Background(), event.ID, true)
		}
	}

	if webhookAllowed {
		bundle := prepareLocale("zh")
		tok, _ := s.opsService.GetOpsPrivateWebhookToken(ctx)
		eventType := "ops_alert"
		if isResolved {
			eventType = "ops_alert_resolved"
		}
		_ = sendOpsPrivateWebhook(ctx, emailCfg.PrivateWebhook, tok, opsPrivateWebhookPayload{
			EventType: eventType,
			Target:    emailCfg.PrivateWebhook.Target,
			Title:     bundle.subject,
			Severity:  strings.TrimSpace(rule.Severity),
			Status:    strings.TrimSpace(event.Status),
			Text:      buildOpsAlertPrivateText(bundle.ev, bundle.diagnosisBadge, bundle.diagnosisSummary, "zh"),
		})
	}

	return anyEmailSent
}

func opsAlertEmailVariables(rule *OpsAlertRule, event *OpsAlertEvent) map[string]string {
	return opsAlertEmailVariablesWithEvidence(rule, event, nil, "", "", "zh")
}

func opsAlertEmailVariablesWithEvidence(
	rule *OpsAlertRule,
	event *OpsAlertEvent,
	ev *opsAlertEvidence,
	diagnosisBadge string,
	diagnosisSummary string,
	locale string,
) map[string]string {
	zh := isChineseOpsLocale(locale)
	variables := map[string]string{
		"rule_name":             "-",
		"severity":              "-",
		"severity_label":        "-",
		"alert_status":          "-",
		"status_label":          "-",
		"metric_type":           "-",
		"metric_label":          "-",
		"operator":              "-",
		"metric_value":          "-",
		"threshold_value":       "-",
		"metric_summary":        "-",
		"triggered_at":          time.Now().UTC().Format(time.RFC3339),
		"triggered_at_local":    formatOpsLocalTime(time.Now(), nil, locale),
		"window_label":          "-",
		"alert_description":     "-",
		"diagnosis_badge":       fallbackIfNonEmpty(diagnosisBadge, opsDiagnosisBadge(false, false, locale)),
		"diagnosis_summary":     "-",
		"alert_details_html":    "",
		"dashboard_url":         "",
		"dashboard_cta_display": "none",
		"header_color":          "#ea580c",
		"email_title":           "运维告警",
		"alert_headline":        "-",
		"duration_human":        "-",
	}
	if !zh {
		variables["email_title"] = "Ops Alert"
	}
	if rule != nil {
		variables["rule_name"] = strings.TrimSpace(rule.Name)
		variables["severity"] = strings.TrimSpace(rule.Severity)
		variables["severity_label"] = opsSeverityLabel(rule.Severity, locale)
		variables["metric_type"] = strings.TrimSpace(rule.MetricType)
		lbl, _ := opsMetricMeta(rule.MetricType, locale)
		variables["metric_label"] = lbl
		variables["operator"] = strings.TrimSpace(rule.Operator)
		variables["threshold_value"] = fmt.Sprintf("%.2f", rule.Threshold)
		variables["window_label"] = formatOpsWindowLabel(rule.WindowMinutes, locale)
		if strings.TrimSpace(rule.Description) != "" {
			variables["alert_description"] = strings.TrimSpace(rule.Description)
		}
	}
	if event != nil {
		variables["alert_status"] = strings.TrimSpace(event.Status)
		variables["status_label"] = opsStatusLabel(event.Status, locale)
		variables["header_color"] = opsSeverityAccentColor(event.Severity, event.Status)
		if event.MetricValue != nil {
			variables["metric_value"] = fmt.Sprintf("%.2f", *event.MetricValue)
		}
		if event.ThresholdValue != nil {
			variables["threshold_value"] = fmt.Sprintf("%.2f", *event.ThresholdValue)
		}
		if !event.FiredAt.IsZero() {
			variables["triggered_at"] = event.FiredAt.UTC().Format(time.RFC3339)
			variables["triggered_at_local"] = formatOpsLocalTime(event.FiredAt, nil, locale)
		}
		if strings.TrimSpace(event.Description) != "" {
			variables["alert_description"] = strings.TrimSpace(event.Description)
		}
		if event.Status == OpsAlertStatusResolved || event.Status == OpsAlertStatusManualResolved {
			if zh {
				variables["email_title"] = "告警已恢复正常"
			} else {
				variables["email_title"] = "Ops Alert Recovered"
			}
		} else {
			if zh {
				variables["email_title"] = opsSeverityShortTag(event.Severity, event.Status, locale)
			} else {
				variables["email_title"] = "Ops Alert"
			}
		}
	}
	if ev != nil {
		variables["severity_label"] = ev.SeverityLabel
		variables["status_label"] = ev.StatusLabel
		variables["metric_label"] = ev.MetricLabel
		variables["metric_summary"] = ev.MetricSummary
		variables["window_label"] = ev.WindowLabel
		if ev.FiredAtLocal != "" {
			variables["triggered_at_local"] = ev.FiredAtLocal
		}
		if (ev.Status == OpsAlertStatusResolved || ev.Status == OpsAlertStatusManualResolved) && ev.ResolvedAtLocal != "" {
			if zh {
				variables["duration_human"] = fmt.Sprintf("%s · 恢复于 %s", fallbackIfNonEmpty(ev.DurationHuman, "较短时间"), ev.ResolvedAtLocal)
			} else {
				variables["duration_human"] = fmt.Sprintf("%s · recovered %s", fallbackIfNonEmpty(ev.DurationHuman, "briefly"), ev.ResolvedAtLocal)
			}
		} else if ev.DurationHuman != "" {
			variables["duration_human"] = ev.DurationHuman
		} else if ev.WindowLabel != "" {
			variables["duration_human"] = ev.WindowLabel
		}
		variables["alert_headline"] = buildOpsAlertHeadline(ev, locale)
		variables["header_color"] = opsSeverityAccentColor(ev.Severity, ev.Status)
		if ev.DashboardURL != "" && isSafeNotificationEmailURL(ev.DashboardURL) {
			variables["dashboard_url"] = ev.DashboardURL
			variables["dashboard_cta_display"] = "block"
		}
		if strings.TrimSpace(diagnosisSummary) != "" {
			variables["diagnosis_summary"] = strings.TrimSpace(diagnosisSummary)
		} else if strings.TrimSpace(ev.FallbackSummary) != "" {
			variables["diagnosis_summary"] = strings.TrimSpace(ev.FallbackSummary)
		}
	} else {
		var mVal, tVal *float64
		if event != nil {
			mVal = event.MetricValue
			tVal = event.ThresholdValue
		}
		if rule != nil {
			if tVal == nil {
				thr := rule.Threshold
				tVal = &thr
			}
			variables["metric_summary"] = formatOpsMetricSummary(rule.MetricType, rule.Operator, mVal, tVal, locale)
			variables["alert_headline"] = fmt.Sprintf("%s：%s", strings.TrimSpace(rule.Name), variables["metric_summary"])
		}
		if strings.TrimSpace(diagnosisSummary) != "" {
			variables["diagnosis_summary"] = strings.TrimSpace(diagnosisSummary)
		} else {
			variables["diagnosis_summary"] = variables["alert_description"]
		}
	}
	return variables
}

func buildOpsAlertEmailBody(rule *OpsAlertRule, event *OpsAlertEvent) string {
	if rule == nil || event == nil {
		return ""
	}
	ev := &opsAlertEvidence{
		Locale:         "zh",
		RuleID:         rule.ID,
		RuleName:       strings.TrimSpace(rule.Name),
		Severity:       strings.TrimSpace(rule.Severity),
		SeverityLabel:  opsSeverityLabel(rule.Severity, "zh"),
		Status:         strings.TrimSpace(event.Status),
		StatusLabel:    opsStatusLabel(event.Status, "zh"),
		MetricType:     strings.TrimSpace(rule.MetricType),
		Operator:       strings.TrimSpace(rule.Operator),
		MetricValue:    event.MetricValue,
		ThresholdValue: event.ThresholdValue,
		WindowMinutes:  rule.WindowMinutes,
		WindowLabel:    formatOpsWindowLabel(rule.WindowMinutes, "zh"),
		FiredAt:        event.FiredAt,
		FiredAtLocal:   formatOpsLocalTime(event.FiredAt, nil, "zh"),
	}
	ev.MetricLabel, ev.MetricUnit = opsMetricMeta(ev.MetricType, "zh")
	if ev.ThresholdValue == nil {
		thr := rule.Threshold
		ev.ThresholdValue = &thr
	}
	ev.MetricSummary = formatOpsMetricSummary(ev.MetricType, ev.Operator, ev.MetricValue, ev.ThresholdValue, "zh")
	summary, _ := buildRuleBasedAlertDiagnosis(ev, "zh")
	return buildOpsAlertFallbackEmailHTML(ev, opsDiagnosisBadge(false, false, "zh"), summary, "", "zh")
}

func buildOpsAlertFallbackEmailHTML(ev *opsAlertEvidence, badge, summary, detailsHTML, locale string) string {
	if ev == nil {
		return ""
	}
	zh := isChineseOpsLocale(locale)
	accent := opsSeverityAccentColor(ev.Severity, ev.Status)
	title := opsSeverityShortTag(ev.Severity, ev.Status, locale)
	headline := buildOpsAlertHeadline(ev, locale)
	btnLabel := "Open Ops Dashboard"
	ruleLbl, sevLbl, statusLbl, metricLbl, timeLbl, windowLbl := "Rule", "Severity", "Status", "Metric", "Triggered at", "Window"
	if zh {
		btnLabel = "打开后台运维监控"
		ruleLbl, sevLbl, statusLbl, metricLbl, timeLbl, windowLbl = "告警规则", "严重程度", "当前状态", "触发指标", "发生时间", "统计窗口"
	}
	cta := ""
	if ev.DashboardURL != "" && isSafeNotificationEmailURL(ev.DashboardURL) {
		cta = fmt.Sprintf(`<div style="margin-top:20px;"><a href="%s" style="display:inline-block;padding:11px 20px;border-radius:8px;background:%s;color:#ffffff;text-decoration:none;font-weight:600;font-size:14px;">%s</a></div>`,
			htmlEscape(ev.DashboardURL), accent, htmlEscape(btnLabel))
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"></head>
<body style="margin:0;padding:24px;background:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','PingFang SC','Microsoft YaHei',sans-serif;color:#18181b;">
  <div style="max-width:640px;margin:0 auto;background:#ffffff;border-radius:12px;overflow:hidden;border:1px solid #e4e4e7;">
    <div style="background:%s;color:#ffffff;padding:24px 28px;">
      <div style="font-size:12px;font-weight:700;opacity:0.9;margin-bottom:6px;">%s</div>
      <h1 style="margin:0;font-size:21px;line-height:1.35;">%s</h1>
    </div>
    <div style="padding:24px 28px;font-size:14px;line-height:1.7;">
      <div style="margin-bottom:18px;padding:14px 16px;border-radius:8px;background:#f8fafc;border-left:4px solid %s;">
        <div style="font-size:12px;font-weight:700;color:%s;margin-bottom:4px;">%s</div>
        <div style="font-size:14px;color:#0f172a;">%s</div>
      </div>
      <table style="width:100%%;border-collapse:collapse;font-size:13px;margin-bottom:16px;">
        <tr><td style="padding:8px 0;color:#64748b;width:108px;border-bottom:1px solid #f1f5f9;">%s</td><td style="padding:8px 0;color:#0f172a;font-weight:600;border-bottom:1px solid #f1f5f9;">%s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b;border-bottom:1px solid #f1f5f9;">%s</td><td style="padding:8px 0;color:#0f172a;border-bottom:1px solid #f1f5f9;">%s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b;border-bottom:1px solid #f1f5f9;">%s</td><td style="padding:8px 0;color:#0f172a;border-bottom:1px solid #f1f5f9;">%s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b;border-bottom:1px solid #f1f5f9;">%s</td><td style="padding:8px 0;color:#0f172a;border-bottom:1px solid #f1f5f9;">%s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b;border-bottom:1px solid #f1f5f9;">%s</td><td style="padding:8px 0;color:#0f172a;border-bottom:1px solid #f1f5f9;">%s</td></tr>
        <tr><td style="padding:8px 0;color:#64748b;">%s</td><td style="padding:8px 0;color:#0f172a;">%s</td></tr>
      </table>
      %s
      %s
    </div>
  </div>
</body>
</html>`,
		accent,
		htmlEscape(title),
		htmlEscape(headline),
		accent,
		accent,
		htmlEscape(badge),
		htmlEscape(summary),
		htmlEscape(ruleLbl), htmlEscape(ev.RuleName),
		htmlEscape(sevLbl), htmlEscape(ev.SeverityLabel),
		htmlEscape(statusLbl), htmlEscape(ev.StatusLabel),
		htmlEscape(metricLbl), htmlEscape(ev.MetricSummary),
		htmlEscape(timeLbl), htmlEscape(ev.FiredAtLocal),
		htmlEscape(windowLbl), htmlEscape(ev.WindowLabel),
		detailsHTML,
		cta,
	)
}

type OpsTestAlertEmailInput struct {
	Recipient          string `json:"recipient"`
	Locale             string `json:"locale"`
	SimulateAIFallback bool   `json:"simulate_ai_fallback"`
}

type OpsTestAlertEmailResult struct {
	Sent             bool   `json:"sent"`
	Recipient        string `json:"recipient"`
	UsedAI           bool   `json:"used_ai"`
	UsedBackupRoute  bool   `json:"used_backup_route,omitempty"`
	FallbackReason   string `json:"fallback_reason,omitempty"`
	DiagnosisBadge   string `json:"diagnosis_badge"`
	DiagnosisSummary string `json:"diagnosis_summary"`
	Subject          string `json:"subject"`
}

func pickSampleOpsAccountsForTestAlert(avail *OpsAccountAvailability, maxCount int) []*AccountAvailability {
	if avail == nil || len(avail.Accounts) == 0 || maxCount <= 0 {
		return nil
	}
	named := make([]*AccountAvailability, 0, len(avail.Accounts))
	for _, acc := range avail.Accounts {
		if acc == nil || strings.TrimSpace(acc.AccountName) == "" {
			continue
		}
		named = append(named, acc)
	}
	if len(named) == 0 {
		return nil
	}
	sort.Slice(named, func(i, j int) bool {
		// Prefer accounts with active errors/rate limits first, then deterministic by AccountID.
		score := func(a *AccountAvailability) int {
			switch {
			case a.HasError:
				return 0
			case a.IsRateLimited || a.IsOverloaded:
				return 1
			case a.IsAvailable:
				return 2
			default:
				return 3
			}
		}
		si, sj := score(named[i]), score(named[j])
		if si != sj {
			return si < sj
		}
		return named[i].AccountID < named[j].AccountID
	})

	picked := make([]*AccountAvailability, 0, maxCount)
	seenPlatform := make(map[string]struct{}, maxCount)
	for _, acc := range named {
		p := strings.ToLower(strings.TrimSpace(acc.Platform))
		if _, ok := seenPlatform[p]; ok {
			continue
		}
		picked = append(picked, acc)
		seenPlatform[p] = struct{}{}
		if len(picked) >= maxCount {
			return picked
		}
	}
	for _, acc := range named {
		already := false
		for _, chosen := range picked {
			if chosen.AccountID == acc.AccountID {
				already = true
				break
			}
		}
		if already {
			continue
		}
		picked = append(picked, acc)
		if len(picked) >= maxCount {
			break
		}
	}
	return picked
}

func defaultOpsTestModelForPlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "anthropic":
		return "claude-opus-5-5"
	case "gemini", "antigravity":
		return "gemini-2.5-pro"
	default:
		return "gpt-6.2-sol"
	}
}

func buildOpsTestAlertClusters(avail *OpsAccountAvailability, locale string) []opsAlertErrorCluster {
	zh := isChineseOpsLocale(locale)
	reason1, sugg1 := "上游服务响应超时（524），长时间未返回结果", "检查对应账号的线路或代理延迟；若反复出现，可暂时停用该账号或切换备用线路"
	reason2, sugg2 := "上游服务商接口繁忙或网关异常（502）", "多为上游瞬时波动；若持续超过 5 分钟，建议检查账号线路或临时切流"
	group1, acc1 := "主力 GPT 分组", "主力 GPT 账号"
	group2, acc2 := "Claude 专线", "主力 Claude 账号"
	if !zh {
		reason1, sugg1 = "Upstream response timed out (524)", "Check proxy/upstream latency for this account or pause it temporarily"
		reason2, sugg2 = "Upstream gateway error (502)", "Usually transient upstream jitter; switch traffic if it persists >5 min"
		group1, acc1 = "Primary GPT Group", "Primary GPT Account"
		group2, acc2 = "Claude Group", "Primary Claude Account"
	}

	c1 := opsAlertErrorCluster{
		Platform:      "openai",
		Model:         "gpt-6.2-sol",
		GroupLabel:    group1,
		AccountLabel:  acc1,
		StatusCode:    524,
		Phase:         "upstream",
		Count:         3,
		ReasonKey:     "upstream_timeout",
		Reason:        reason1,
		Suggestion:    sugg1,
		SampleMessage: "upstream gateway timeout (524) before client disconnect",
	}
	c2 := opsAlertErrorCluster{
		Platform:      "anthropic",
		Model:         "claude-opus-5-5",
		GroupLabel:    group2,
		AccountLabel:  acc2,
		StatusCode:    502,
		Phase:         "upstream",
		Count:         1,
		ReasonKey:     "upstream_502",
		Reason:        reason2,
		Suggestion:    sugg2,
		SampleMessage: "The service is busy. Please retry later.",
	}

	picked := pickSampleOpsAccountsForTestAlert(avail, 2)
	if len(picked) >= 1 {
		p := fallbackIfNonEmpty(picked[0].Platform, "openai")
		c1.Platform = p
		c1.Model = defaultOpsTestModelForPlatform(p)
		c1.AccountLabel = strings.TrimSpace(picked[0].AccountName)
		c1.GroupLabel = fallbackIfNonEmpty(picked[0].GroupName, "-")
	}
	if len(picked) >= 2 {
		p := fallbackIfNonEmpty(picked[1].Platform, "anthropic")
		c2.Platform = p
		c2.Model = defaultOpsTestModelForPlatform(p)
		c2.AccountLabel = strings.TrimSpace(picked[1].AccountName)
		c2.GroupLabel = fallbackIfNonEmpty(picked[1].GroupName, "-")
		return []opsAlertErrorCluster{c1, c2}
	}
	if len(picked) == 1 {
		return []opsAlertErrorCluster{c1}
	}
	return []opsAlertErrorCluster{c1, c2}
}

func (s *OpsService) SendTestAlertEmail(ctx context.Context, input OpsTestAlertEmailInput) (*OpsTestAlertEmailResult, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	if s.emailService == nil {
		return nil, infraerrors.ServiceUnavailable("EMAIL_SERVICE_UNAVAILABLE", "邮件服务未初始化")
	}
	emailCfg, err := s.GetEmailNotificationConfig(ctx)
	if err != nil {
		return nil, err
	}
	recipient := strings.TrimSpace(input.Recipient)
	if recipient == "" && emailCfg != nil {
		for _, r := range emailCfg.Alert.Recipients {
			if strings.TrimSpace(r) != "" {
				recipient = strings.TrimSpace(r)
				break
			}
		}
	}
	if recipient == "" {
		return nil, infraerrors.BadRequest("ALERT_RECIPIENT_REQUIRED", "请先配置至少一个告警接收邮箱")
	}

	locale := ""
	if strings.TrimSpace(input.Locale) != "" {
		locale = normalizeNotificationLocale(input.Locale)
	} else {
		locale = "zh"
		if s.emailService.notificationEmailService != nil {
			resolved := s.emailService.notificationEmailService.ResolveRecipientLocale(ctx, 0, recipient)
			if strings.TrimSpace(resolved) != "" && resolved != notificationEmailDefaultLocale {
				locale = resolved
			}
		}
	}

	var liveAvail *OpsAccountAvailability
	availCtx, availCancel := context.WithTimeout(ctx, 2*time.Second)
	if avail, aErr := s.GetAccountAvailability(availCtx, "", nil); aErr == nil && avail != nil {
		liveAvail = avail
	}
	availCancel()

	zh := isChineseOpsLocale(locale)
	ruleName := "请求错误率过高（测试演练）"
	ruleDesc := "用于验证告警邮件可读性与 AI / 规则诊断降级效果的演练通知"
	metricLabel := "请求错误率"
	if !zh {
		ruleName = "High error rate (test drill)"
		ruleDesc = "Drill alert to verify email readability and AI / built-in diagnosis fallback"
		metricLabel = "Error rate"
	}

	now := time.Now().UTC()
	metricVal := 28.57
	thresholdVal := 15.0
	rule := &OpsAlertRule{
		ID:            999,
		Name:          ruleName,
		Description:   ruleDesc,
		Severity:      "P1",
		MetricType:    "error_rate",
		Operator:      ">",
		Threshold:     thresholdVal,
		WindowMinutes: 5,
		NotifyEmail:   true,
	}
	event := &OpsAlertEvent{
		ID:             0,
		RuleID:         rule.ID,
		Severity:       rule.Severity,
		Status:         OpsAlertStatusFiring,
		MetricValue:    &metricVal,
		ThresholdValue: &thresholdVal,
		FiredAt:        now,
	}
	ev := &opsAlertEvidence{
		Locale:              locale,
		RuleID:              rule.ID,
		RuleName:            rule.Name,
		Severity:            rule.Severity,
		SeverityLabel:       opsSeverityLabel(rule.Severity, locale),
		Status:              OpsAlertStatusFiring,
		StatusLabel:         opsStatusLabel(OpsAlertStatusFiring, locale),
		MetricType:          rule.MetricType,
		MetricLabel:         metricLabel,
		MetricUnit:          "%",
		Operator:            rule.Operator,
		MetricValue:         &metricVal,
		ThresholdValue:      &thresholdVal,
		MetricSummary:       formatOpsMetricSummary(rule.MetricType, rule.Operator, &metricVal, &thresholdVal, locale),
		ScopeLabel:          formatOpsScopeLabel("", nil, locale),
		WindowMinutes:       5,
		WindowLabel:         formatOpsWindowLabel(5, locale),
		FiredAt:             now,
		FiredAtLocal:        formatOpsLocalTime(now, s.cfg, locale),
		DurationHuman:       formatOpsDurationHuman(5*time.Minute, locale),
		TotalRequests:       14,
		SLARequests:         14,
		SuccessCount:        10,
		ErrorCount:          4,
		AffectedUsers:       2,
		ShowRequestOverview: true,
		AccountPoolSummary:  formatOpsAvailabilitySnapshotSummary(liveAvail, locale),
		DashboardURL:        s.ResolveAdminOpsURL(ctx),
		TopClusters:         buildOpsTestAlertClusters(liveAvail, locale),
	}
	ev.FallbackSummary, ev.Suggestions = buildRuleBasedAlertDiagnosis(ev, locale)

	diagSummary := ev.FallbackSummary
	aiRoute := ""
	fallbackReason := "simulated_fallback"
	aiEnabled := (emailCfg != nil && emailCfg.AIDiagnosis.Enabled) || input.SimulateAIFallback
	if !input.SimulateAIFallback && emailCfg != nil && emailCfg.AIDiagnosis.Enabled {
		aiKey, _ := s.GetOpsAIDiagnosisAPIKey(ctx)
		aiFallbackKey, _ := s.GetOpsAIDiagnosisFallbackAPIKey(ctx)
		if aiCtx, aiCancel, ok := boundedOpsAIDiagnosisContext(ctx, opsDiagnosisMaxTotalBudget, 5*time.Second); ok {
			diagSummary, aiRoute, fallbackReason = DiagnoseOpsAlertWithRoutes(aiCtx, emailCfg.AIDiagnosis, aiKey, aiFallbackKey, ev, locale)
			aiCancel()
		} else {
			fallbackReason = "insufficient_budget"
		}
	} else if !input.SimulateAIFallback {
		fallbackReason = "not_configured"
	}
	usedAI := aiRoute != ""
	usedBackupRoute := aiRoute == opsDiagnosisRouteBackup
	badge := opsDiagnosisBadgeForRoute(aiRoute, aiEnabled, locale)
	detailsHTML := buildOpsAlertDetailsHTML(ev, locale)
	vars := opsAlertEmailVariablesWithEvidence(rule, event, ev, badge, diagSummary, locale)
	subject := buildOpsAlertEmailSubject(ev, locale)

	sent := false
	if s.emailService.notificationEmailService != nil {
		err = s.emailService.notificationEmailService.Send(ctx, NotificationEmailSendInput{
			Event:            NotificationEmailEventOpsAlert,
			Locale:           locale,
			RecipientEmail:   recipient,
			RecipientName:    emailRecipientName(recipient),
			SourceType:       "ops_alert_test",
			SourceID:         uuid.NewString(),
			Variables:        vars,
			RawHTMLVariables: map[string]string{"alert_details_html": detailsHTML},
		})
		if err == nil {
			sent = true
		} else if !shouldFallbackNotificationEmail(err) {
			return nil, err
		}
	}
	if !sent {
		fallbackHTML := buildOpsAlertFallbackEmailHTML(ev, badge, diagSummary, detailsHTML, locale)
		if err := s.emailService.SendEmail(ctx, recipient, subject, fallbackHTML); err != nil {
			return nil, err
		}
		sent = true
	}

	return &OpsTestAlertEmailResult{
		Sent:             sent,
		Recipient:        recipient,
		UsedAI:           usedAI,
		UsedBackupRoute:  usedBackupRoute,
		FallbackReason:   fallbackReason,
		DiagnosisBadge:   badge,
		DiagnosisSummary: diagSummary,
		Subject:          subject,
	}, nil
}

func shouldSendOpsAlertEmailByMinSeverity(minSeverity string, ruleSeverity string) bool {
	minSeverity = strings.ToLower(strings.TrimSpace(minSeverity))
	if minSeverity == "" {
		return true
	}

	eventLevel := opsEmailSeverityForOps(ruleSeverity)
	minLevel := strings.ToLower(minSeverity)

	rank := func(level string) int {
		switch level {
		case "critical":
			return 3
		case "warning":
			return 2
		case "info":
			return 1
		default:
			return 0
		}
	}
	return rank(eventLevel) >= rank(minLevel)
}

func opsEmailSeverityForOps(severity string) string {
	switch strings.ToUpper(strings.TrimSpace(severity)) {
	case "P0":
		return "critical"
	case "P1":
		return "warning"
	default:
		return "info"
	}
}

func isOpsAlertSilenced(now time.Time, rule *OpsAlertRule, event *OpsAlertEvent, silencing OpsAlertSilencingSettings) bool {
	if !silencing.Enabled {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if strings.TrimSpace(silencing.GlobalUntilRFC3339) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(silencing.GlobalUntilRFC3339)); err == nil {
			if now.Before(t) {
				return true
			}
		}
	}

	for _, entry := range silencing.Entries {
		untilRaw := strings.TrimSpace(entry.UntilRFC3339)
		if untilRaw == "" {
			continue
		}
		until, err := time.Parse(time.RFC3339, untilRaw)
		if err != nil {
			continue
		}
		if now.After(until) {
			continue
		}
		if entry.RuleID != nil && rule != nil && rule.ID > 0 && *entry.RuleID != rule.ID {
			continue
		}
		if len(entry.Severities) > 0 {
			match := false
			for _, s := range entry.Severities {
				if strings.EqualFold(strings.TrimSpace(s), strings.TrimSpace(event.Severity)) || strings.EqualFold(strings.TrimSpace(s), strings.TrimSpace(rule.Severity)) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		return true
	}

	return false
}

func (s *OpsAlertEvaluatorService) tryAcquireLeaderLock(ctx context.Context, lock OpsDistributedLockSettings) (func(), bool) {
	if !lock.Enabled {
		return nil, true
	}
	if s.redisClient == nil {
		s.warnNoRedisOnce.Do(func() {
			logger.LegacyPrintf("service.ops_alert_evaluator", "[OpsAlertEvaluator] redis not configured; running without distributed lock")
		})
		return nil, true
	}
	key := strings.TrimSpace(lock.Key)
	if key == "" {
		key = opsAlertEvaluatorLeaderLockKey
	}
	ttl := time.Duration(lock.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = opsAlertEvaluatorLeaderLockTTL
	}

	ok, err := s.redisClient.SetNX(ctx, key, s.instanceID, ttl).Result()
	if err != nil {
		// Prefer fail-closed to avoid duplicate evaluators stampeding the DB when Redis is flaky.
		// Single-node deployments can disable the distributed lock via runtime settings.
		s.warnNoRedisOnce.Do(func() {
			logger.LegacyPrintf("service.ops_alert_evaluator", "[OpsAlertEvaluator] leader lock SetNX failed; skipping this cycle: %v", err)
		})
		return nil, false
	}
	if !ok {
		s.maybeLogSkip(key)
		return nil, false
	}
	return func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer releaseCancel()
		_, _ = opsAlertEvaluatorReleaseScript.Run(releaseCtx, s.redisClient, []string{key}, s.instanceID).Result()
	}, true
}

func (s *OpsAlertEvaluatorService) maybeLogSkip(key string) {
	s.skipLogMu.Lock()
	defer s.skipLogMu.Unlock()

	now := time.Now()
	if !s.skipLogAt.IsZero() && now.Sub(s.skipLogAt) < opsAlertEvaluatorSkipLogInterval {
		return
	}
	s.skipLogAt = now
	logger.LegacyPrintf("service.ops_alert_evaluator", "[OpsAlertEvaluator] leader lock held by another instance; skipping (key=%q)", key)
}

func (s *OpsAlertEvaluatorService) recordHeartbeatSuccess(runAt time.Time, duration time.Duration, result string) {
	if s == nil || s.opsRepo == nil {
		return
	}
	now := time.Now().UTC()
	durMs := duration.Milliseconds()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	msg := strings.TrimSpace(result)
	if msg == "" {
		msg = "ok"
	}
	msg = truncateString(msg, 2048)
	_ = s.opsRepo.UpsertJobHeartbeat(ctx, &OpsUpsertJobHeartbeatInput{
		JobName:        opsAlertEvaluatorJobName,
		LastRunAt:      &runAt,
		LastSuccessAt:  &now,
		LastDurationMs: &durMs,
		LastResult:     &msg,
	})
}

func (s *OpsAlertEvaluatorService) recordHeartbeatError(runAt time.Time, duration time.Duration, err error) {
	if s == nil || s.opsRepo == nil || err == nil {
		return
	}
	now := time.Now().UTC()
	durMs := duration.Milliseconds()
	msg := truncateString(err.Error(), 2048)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.opsRepo.UpsertJobHeartbeat(ctx, &OpsUpsertJobHeartbeatInput{
		JobName:        opsAlertEvaluatorJobName,
		LastRunAt:      &runAt,
		LastErrorAt:    &now,
		LastError:      &msg,
		LastDurationMs: &durMs,
	})
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

type slidingWindowLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	sent   []time.Time
}

func newSlidingWindowLimiter(limit int, window time.Duration) *slidingWindowLimiter {
	if window <= 0 {
		window = time.Hour
	}
	return &slidingWindowLimiter{
		limit:  limit,
		window: window,
		sent:   []time.Time{},
	}
}

func (l *slidingWindowLimiter) SetLimit(limit int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.limit = limit
}

func (l *slidingWindowLimiter) Allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.limit <= 0 {
		return true
	}
	cutoff := now.Add(-l.window)
	keep := l.sent[:0]
	for _, t := range l.sent {
		if t.After(cutoff) {
			keep = append(keep, t)
		}
	}
	l.sent = keep
	if len(l.sent) >= l.limit {
		return false
	}
	l.sent = append(l.sent, now)
	return true
}

// computeGroupAvailableRatio returns the available percentage for a group.
// Formula: (AvailableCount / TotalAccounts) * 100.
// Returns 0 when TotalAccounts is 0.
func computeGroupAvailableRatio(group *GroupAvailability) float64 {
	if group == nil || group.TotalAccounts <= 0 {
		return 0
	}
	return (float64(group.AvailableCount) / float64(group.TotalAccounts)) * 100
}

// countAccountsByCondition counts accounts that satisfy the given condition.
func countAccountsByCondition(accounts map[int64]*AccountAvailability, condition func(*AccountAvailability) bool) int64 {
	if len(accounts) == 0 || condition == nil {
		return 0
	}
	var count int64
	for _, account := range accounts {
		if account != nil && condition(account) {
			count++
		}
	}
	return count
}
