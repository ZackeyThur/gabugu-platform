package service

import (
	"context"
	"fmt"
	"html"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

const (
	opsScheduledReportJobName = "ops_scheduled_reports"

	opsScheduledReportLeaderLockKeyDefault = "ops:scheduled_reports:leader"
	opsScheduledReportLeaderLockTTLDefault = 5 * time.Minute

	opsScheduledReportLastRunKeyPrefix = "ops:scheduled_reports:last_run:"

	opsScheduledReportTickInterval = 1 * time.Minute
)

var opsScheduledReportCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

var opsScheduledReportReleaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

type OpsScheduledReportService struct {
	opsService   *OpsService
	userService  *UserService
	emailService *EmailService
	redisClient  *redis.Client
	cfg          *config.Config

	instanceID string
	loc        *time.Location

	distributedLockOn bool
	warnNoRedisOnce   sync.Once

	startOnce sync.Once
	stopOnce  sync.Once
	stopCtx   context.Context
	stop      context.CancelFunc
	wg        sync.WaitGroup
}

func NewOpsScheduledReportService(
	opsService *OpsService,
	userService *UserService,
	emailService *EmailService,
	redisClient *redis.Client,
	cfg *config.Config,
) *OpsScheduledReportService {
	lockOn := cfg == nil || strings.TrimSpace(cfg.RunMode) != config.RunModeSimple

	loc := time.Local
	if cfg != nil && strings.TrimSpace(cfg.Timezone) != "" {
		if parsed, err := time.LoadLocation(strings.TrimSpace(cfg.Timezone)); err == nil && parsed != nil {
			loc = parsed
		}
	}
	return &OpsScheduledReportService{
		opsService:   opsService,
		userService:  userService,
		emailService: emailService,
		redisClient:  redisClient,
		cfg:          cfg,

		instanceID:        uuid.NewString(),
		loc:               loc,
		distributedLockOn: lockOn,
		warnNoRedisOnce:   sync.Once{},
		startOnce:         sync.Once{},
		stopOnce:          sync.Once{},
		stopCtx:           nil,
		stop:              nil,
		wg:                sync.WaitGroup{},
	}
}

func (s *OpsScheduledReportService) Start() {
	s.StartWithContext(context.Background())
}

func (s *OpsScheduledReportService) StartWithContext(ctx context.Context) {
	if s == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if s.cfg != nil && !s.cfg.Ops.Enabled {
		return
	}
	if s.opsService == nil || s.emailService == nil {
		return
	}

	s.startOnce.Do(func() {
		s.stopCtx, s.stop = context.WithCancel(ctx)
		s.wg.Add(1)
		go s.run()
	})
}

func (s *OpsScheduledReportService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.stop != nil {
			s.stop()
		}
	})
	s.wg.Wait()
}

func (s *OpsScheduledReportService) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(opsScheduledReportTickInterval)
	defer ticker.Stop()

	s.runOnce()
	for {
		select {
		case <-ticker.C:
			s.runOnce()
		case <-s.stopCtx.Done():
			return
		}
	}
}

func (s *OpsScheduledReportService) runOnce() {
	if s == nil || s.opsService == nil || s.emailService == nil {
		return
	}

	startedAt := time.Now().UTC()
	runAt := startedAt

	ctx, cancel := context.WithTimeout(s.stopCtx, 60*time.Second)
	defer cancel()

	// Respect ops monitoring enabled switch.
	if !s.opsService.IsMonitoringEnabled(ctx) {
		return
	}

	release, ok := s.tryAcquireLeaderLock(ctx)
	if !ok {
		return
	}
	if release != nil {
		defer release()
	}

	now := time.Now()
	if s.loc != nil {
		now = now.In(s.loc)
	}

	reports := s.listScheduledReports(ctx, now)
	if len(reports) == 0 {
		return
	}

	reportsTotal := len(reports)
	reportsDue := 0
	sentAttempts := 0

	for _, report := range reports {
		if report == nil || !report.Enabled {
			continue
		}
		if report.NextRunAt.After(now) {
			continue
		}
		reportsDue++

		attempts, err := s.runReport(ctx, report, now)
		if err != nil {
			s.recordHeartbeatError(runAt, time.Since(startedAt), err)
			return
		}
		sentAttempts += attempts
	}

	result := truncateString(fmt.Sprintf("reports=%d due=%d send_attempts=%d", reportsTotal, reportsDue, sentAttempts), 2048)
	s.recordHeartbeatSuccess(runAt, time.Since(startedAt), result)
}

type opsScheduledReport struct {
	Name       string
	ReportType string
	Schedule   string
	Enabled    bool

	TimeRange time.Duration

	Recipients []string

	ErrorDigestMinCount             int
	AccountHealthErrorRateThreshold float64

	LastRunAt *time.Time
	NextRunAt time.Time
}

type opsScheduledReportContent struct {
	html      string
	overview  *OpsDashboardOverview
	errorLogs []*OpsErrorLog
	avail     *OpsAccountAvailability
	start     time.Time
	end       time.Time
}

func (s *OpsScheduledReportService) listScheduledReports(ctx context.Context, now time.Time) []*opsScheduledReport {
	if s == nil || s.opsService == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	emailCfg, err := s.opsService.GetEmailNotificationConfig(ctx)
	if err != nil || emailCfg == nil {
		return nil
	}
	if !emailCfg.Report.Enabled {
		return nil
	}

	recipients := normalizeEmails(emailCfg.Report.Recipients)

	type reportDef struct {
		enabled   bool
		name      string
		kind      string
		timeRange time.Duration
		schedule  string
	}

	defs := []reportDef{
		{enabled: emailCfg.Report.DailySummaryEnabled, name: "日报", kind: "daily_summary", timeRange: 24 * time.Hour, schedule: emailCfg.Report.DailySummarySchedule},
		{enabled: emailCfg.Report.WeeklySummaryEnabled, name: "周报", kind: "weekly_summary", timeRange: 7 * 24 * time.Hour, schedule: emailCfg.Report.WeeklySummarySchedule},
		{enabled: emailCfg.Report.ErrorDigestEnabled, name: "错误摘要", kind: "error_digest", timeRange: 24 * time.Hour, schedule: emailCfg.Report.ErrorDigestSchedule},
		{enabled: emailCfg.Report.AccountHealthEnabled, name: "账号健康", kind: "account_health", timeRange: 24 * time.Hour, schedule: emailCfg.Report.AccountHealthSchedule},
	}

	out := make([]*opsScheduledReport, 0, len(defs))
	for _, d := range defs {
		if !d.enabled {
			continue
		}
		spec := strings.TrimSpace(d.schedule)
		if spec == "" {
			continue
		}
		sched, err := opsScheduledReportCronParser.Parse(spec)
		if err != nil {
			log.Printf("[OpsScheduledReport] invalid cron spec=%q for report=%s: %v", spec, d.kind, err)
			continue
		}

		lastRun := s.getLastRunAt(ctx, d.kind)
		base := lastRun
		if base.IsZero() {
			// Allow a schedule matching the current minute to trigger right after startup.
			base = now.Add(-1 * time.Minute)
		}
		next := sched.Next(base)
		if next.IsZero() {
			continue
		}

		var lastRunPtr *time.Time
		if !lastRun.IsZero() {
			lastCopy := lastRun
			lastRunPtr = &lastCopy
		}

		out = append(out, &opsScheduledReport{
			Name:       d.name,
			ReportType: d.kind,
			Schedule:   spec,
			Enabled:    true,

			TimeRange: d.timeRange,

			Recipients: recipients,

			ErrorDigestMinCount:             emailCfg.Report.ErrorDigestMinCount,
			AccountHealthErrorRateThreshold: emailCfg.Report.AccountHealthErrorRateThreshold,

			LastRunAt: lastRunPtr,
			NextRunAt: next,
		})
	}

	return out
}

func (s *OpsScheduledReportService) runReport(ctx context.Context, report *opsScheduledReport, now time.Time) (int, error) {
	if s == nil || s.opsService == nil || s.emailService == nil || report == nil {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Mark as "run" up-front so a broken SMTP config doesn't spam retries every minute.
	s.setLastRunAt(ctx, report.ReportType, now)

	content, err := s.generateReportContent(ctx, report, now)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(content.html) == "" {
		// Skip sending when the report decides not to emit content (e.g., digest below min count).
		return 0, nil
	}

	recipients := report.Recipients
	if len(recipients) == 0 && s.userService != nil {
		admin, err := s.userService.GetFirstAdmin(ctx)
		if err == nil && admin != nil && strings.TrimSpace(admin.Email) != "" {
			recipients = []string{strings.TrimSpace(admin.Email)}
		}
	}

	var emailCfg *OpsEmailNotificationConfig
	if s.opsService.settingRepo != nil {
		if cfgLoaded, cfgErr := s.opsService.GetEmailNotificationConfig(ctx); cfgErr == nil {
			emailCfg = cfgLoaded
		}
	}
	var aiCfg OpsAIDiagnosisConfig
	var aiAPIKey string
	var aiFallbackAPIKey string
	if emailCfg != nil {
		aiCfg = emailCfg.AIDiagnosis
		if aiCfg.Enabled {
			aiAPIKey, _ = s.opsService.GetOpsAIDiagnosisAPIKey(ctx)
			aiFallbackAPIKey, _ = s.opsService.GetOpsAIDiagnosisFallbackAPIKey(ctx)
		}
	}

	dashURL := ""
	if candidate := s.opsService.ResolveAdminOpsURL(ctx); candidate != "" && isSafeNotificationEmailURL(candidate) {
		dashURL = candidate
	}

	type reportLocaleBundle struct {
		periodDisplay string
		diagBadge     string
		diagSummary   string
		poolSummary   string
		clusters      []opsAlertErrorCluster
		insightsHTML  string
	}
	localeBundles := make(map[string]*reportLocaleBundle, 2)
	prepareReportBundle := func(locale string) *reportLocaleBundle {
		normLocale := normalizeNotificationLocale(locale)
		if existing, ok := localeBundles[normLocale]; ok {
			return existing
		}
		periodDisplay := formatOpsLocalPeriod(content.start, content.end, s.cfg, normLocale)
		clusters, _ := clusterOpsErrorLogs(content.errorLogs, normLocale, 5)
		poolSummary := formatOpsAvailabilitySnapshotSummary(content.avail, normLocale)

		var b reportLocaleBundle
		b.periodDisplay = periodDisplay
		b.clusters = clusters
		b.poolSummary = poolSummary

		if isOpsSummaryReport(report) {
			diagInput := buildOpsReportDiagnosisInput(report, content.overview, periodDisplay, poolSummary, clusters, normLocale)
			diagSummary := buildRuleBasedReportSummary(diagInput, normLocale)
			aiRoute := ""
			if aiCfg.Enabled {
				if aiCtx, aiCancel, ok := boundedOpsAIDiagnosisContext(ctx, opsDiagnosisMaxTotalBudget, opsDiagnosisDeliveryHeadroom); ok {
					diagSummary, aiRoute, _ = DiagnoseOpsScheduledReportWithRoutes(aiCtx, aiCfg, aiAPIKey, aiFallbackAPIKey, diagInput, normLocale)
					aiCancel()
				}
			}
			b.diagBadge = opsDiagnosisBadgeForRoute(aiRoute, aiCfg.Enabled, normLocale)
			b.diagSummary = diagSummary
			b.insightsHTML = buildOpsReportInsightsHTML(b.diagBadge, diagSummary, clusters, poolSummary, normLocale)
		}
		localeBundles[normLocale] = &b
		return &b
	}

	attempts := 0
	for _, to := range recipients {
		addr := strings.TrimSpace(to)
		if addr == "" {
			continue
		}
		attempts++
		locale := ""
		if s.emailService.notificationEmailService != nil {
			locale = s.emailService.notificationEmailService.ResolveRecipientLocale(ctx, 0, addr)
			bundle := prepareReportBundle(locale)
			templateVariables := opsScheduledReportLocalizedEmailVariables(report, now, locale)
			rawHTMLVariables := map[string]string{"report_html": content.html}
			if isOpsSummaryReport(report) {
				templateVariables = opsSummaryReportEmailVariables(report, now, content.overview, locale)
				if bundle.insightsHTML != "" {
					templateVariables["report_insights_display"] = "block"
					rawHTMLVariables["report_insights_html"] = bundle.insightsHTML
				}
			}
			if bundle.periodDisplay != "" && bundle.periodDisplay != "-" {
				templateVariables["report_period_display"] = bundle.periodDisplay
			}
			if dashURL != "" {
				templateVariables["dashboard_url"] = dashURL
				templateVariables["dashboard_cta_display"] = "block"
			}
			if err := s.emailService.notificationEmailService.Send(ctx, NotificationEmailSendInput{
				Event:            NotificationEmailEventOpsScheduledReport,
				Locale:           locale,
				RecipientEmail:   addr,
				RecipientName:    emailRecipientName(addr),
				SourceType:       "ops_scheduled_report",
				SourceID:         opsScheduledReportDeliverySourceID(report),
				ReminderKey:      now.UTC().Format("2006-01-02T15:04"),
				Variables:        templateVariables,
				RawHTMLVariables: rawHTMLVariables,
			}); err == nil {
				continue
			} else if !shouldFallbackNotificationEmail(err) {
				continue
			}
		}
		subjectName := strings.TrimSpace(report.Name)
		if locale != "" {
			subjectName = opsScheduledReportLocalizedName(report, locale)
		}
		subjectPrefix := "[Ops Report]"
		if strings.HasPrefix(strings.ToLower(locale), "zh") {
			subjectPrefix = "[运维报表]"
		}
		subject := fmt.Sprintf("%s %s", subjectPrefix, subjectName)
		if err := s.emailService.SendEmail(ctx, addr, subject, content.html); err != nil {
			// Ignore per-recipient failures; continue best-effort.
			continue
		}
	}

	if emailCfg != nil && emailCfg.PrivateWebhook.Enabled && emailCfg.PrivateWebhook.IncludeReports {
		bundle := prepareReportBundle("zh")
		tok, _ := s.opsService.GetOpsPrivateWebhookToken(ctx)
		reportTitle := fmt.Sprintf("[GABUGU %s] %s", opsScheduledReportLocalizedName(report, "zh"), bundle.periodDisplay)
		text := buildOpsReportPrivateText(report, content.overview, bundle.periodDisplay, bundle.diagBadge, bundle.diagSummary, bundle.poolSummary, bundle.clusters, dashURL)
		_ = sendOpsPrivateWebhook(ctx, emailCfg.PrivateWebhook, tok, opsPrivateWebhookPayload{
			EventType: "ops_scheduled_report",
			Target:    emailCfg.PrivateWebhook.Target,
			Title:     reportTitle,
			Text:      text,
		})
	}
	return attempts, nil
}

func opsScheduledReportDeliverySourceID(report *opsScheduledReport) string {
	if report == nil {
		return "scheduled_report"
	}
	parts := []string{
		strings.TrimSpace(report.ReportType),
		strings.TrimSpace(report.Name),
		strings.TrimSpace(report.Schedule),
	}
	joined := strings.Trim(strings.Join(parts, ":"), ":")
	if joined == "" {
		return "scheduled_report"
	}
	return joined
}

func opsScheduledReportEmailVariables(report *opsScheduledReport, now time.Time) map[string]string {
	end := now.UTC()
	start := end
	name := "Ops report"
	reportType := "scheduled_report"
	if report != nil {
		if strings.TrimSpace(report.Name) != "" {
			name = strings.TrimSpace(report.Name)
		}
		if strings.TrimSpace(report.ReportType) != "" {
			reportType = strings.TrimSpace(report.ReportType)
		}
		if report.TimeRange > 0 {
			start = end.Add(-report.TimeRange)
		}
	}
	return map[string]string{
		"report_name":       name,
		"report_type":       reportType,
		"report_start_time": start.Format(time.RFC3339),
		"report_end_time":   end.Format(time.RFC3339),
	}
}

func opsScheduledReportLocalizedEmailVariables(report *opsScheduledReport, now time.Time, locale string) map[string]string {
	variables := opsScheduledReportEmailVariables(report, now)
	variables["report_html"] = ""
	variables["report_detail_display"] = "block"
	for _, placeholder := range notificationEmailOpsSummaryPlaceholders {
		variables[placeholder] = "-"
	}
	variables["report_summary_display"] = "none"
	if name := opsScheduledReportLocalizedName(report, locale); name != "" {
		variables["report_name"] = name
	}
	return variables
}

func opsScheduledReportLocalizedName(report *opsScheduledReport, locale string) string {
	if report == nil {
		return "Ops report"
	}
	chinese := strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "zh")
	switch strings.TrimSpace(report.ReportType) {
	case "daily_summary":
		if chinese {
			return "日报"
		}
		return "Daily summary"
	case "weekly_summary":
		if chinese {
			return "周报"
		}
		return "Weekly summary"
	case "error_digest":
		if chinese {
			return "错误摘要"
		}
		return "Error digest"
	case "account_health":
		if chinese {
			return "账号健康"
		}
		return "Account health"
	default:
		return strings.TrimSpace(report.Name)
	}
}

func isOpsSummaryReport(report *opsScheduledReport) bool {
	if report == nil {
		return false
	}
	switch strings.TrimSpace(report.ReportType) {
	case "daily_summary", "weekly_summary":
		return true
	default:
		return false
	}
}

func opsSummaryReportEmailVariables(report *opsScheduledReport, now time.Time, overview *OpsDashboardOverview, locale string) map[string]string {
	variables := opsScheduledReportLocalizedEmailVariables(report, now, locale)
	variables["report_detail_display"] = "none"
	if overview == nil {
		for _, placeholder := range notificationEmailOpsSummaryPlaceholders {
			if placeholder == "report_summary_display" {
				continue
			}
			variables[placeholder] = "-"
		}
		variables["report_summary_display"] = "block"
		return variables
	}
	variables["report_summary_display"] = "block"

	variables["report_total_requests"] = formatOpsReportInteger(overview.RequestCountTotal)
	variables["report_success_count"] = formatOpsReportInteger(overview.SuccessCount)
	variables["report_sla_error_count"] = formatOpsReportInteger(overview.ErrorCountSLA)
	variables["report_business_limited_count"] = formatOpsReportInteger(overview.BusinessLimitedCount)
	variables["report_sla"] = fmt.Sprintf("%.2f%%", overview.SLA*100)
	variables["report_error_rate"] = fmt.Sprintf("%.2f%%", overview.ErrorRate*100)
	variables["report_upstream_error_rate"] = fmt.Sprintf("%.2f%%", overview.UpstreamErrorRate*100)
	variables["report_upstream_error_count_excl_429_529"] = formatOpsReportInteger(overview.UpstreamErrorCountExcl429529)
	variables["report_upstream_429_count"] = formatOpsReportInteger(overview.Upstream429Count)
	variables["report_upstream_529_count"] = formatOpsReportInteger(overview.Upstream529Count)
	variables["report_latency_p50"] = formatOpsReportMilliseconds(overview.Duration.P50)
	variables["report_latency_p99"] = formatOpsReportMilliseconds(overview.Duration.P99)
	variables["report_ttft_p50"] = formatOpsReportMilliseconds(overview.TTFT.P50)
	variables["report_ttft_p99"] = formatOpsReportMilliseconds(overview.TTFT.P99)
	variables["report_tokens"] = formatOpsReportInteger(overview.TokenConsumed)
	variables["report_qps_current"] = fmt.Sprintf("%.1f", overview.QPS.Current)
	variables["report_qps_peak"] = fmt.Sprintf("%.1f", overview.QPS.Peak)
	variables["report_qps_avg"] = fmt.Sprintf("%.1f", overview.QPS.Avg)
	variables["report_tps_current"] = fmt.Sprintf("%.1f", overview.TPS.Current)
	variables["report_tps_peak"] = fmt.Sprintf("%.1f", overview.TPS.Peak)
	variables["report_tps_avg"] = fmt.Sprintf("%.1f", overview.TPS.Avg)
	return variables
}

func formatOpsReportInteger(value int64) string {
	raw := strconv.FormatInt(value, 10)
	start := 0
	if strings.HasPrefix(raw, "-") {
		start = 1
	}
	if len(raw)-start <= 3 {
		return raw
	}

	var builder strings.Builder
	builder.Grow(len(raw) + (len(raw)-start-1)/3)
	_, _ = builder.WriteString(raw[:start])
	digitLen := len(raw) - start
	for offset := 0; offset < digitLen; offset++ {
		if offset > 0 && (digitLen-offset)%3 == 0 {
			_ = builder.WriteByte(',')
		}
		_ = builder.WriteByte(raw[start+offset])
	}
	return builder.String()
}

func formatOpsReportMilliseconds(value *int) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%s ms", formatOpsReportInteger(int64(*value)))
}

func (s *OpsScheduledReportService) generateReportContent(ctx context.Context, report *opsScheduledReport, now time.Time) (opsScheduledReportContent, error) {
	if s == nil || s.opsService == nil || report == nil {
		return opsScheduledReportContent{}, fmt.Errorf("service not initialized")
	}
	if report.TimeRange <= 0 {
		return opsScheduledReportContent{}, fmt.Errorf("invalid time range")
	}

	end := now.UTC()
	start := end.Add(-report.TimeRange)

	switch strings.TrimSpace(report.ReportType) {
	case "daily_summary", "weekly_summary":
		overview, err := s.opsService.GetDashboardOverview(ctx, &OpsDashboardFilter{
			StartTime: start,
			EndTime:   end,
			Platform:  "",
			GroupID:   nil,
			QueryMode: OpsQueryModeAuto,
		})
		if err != nil {
			// If pre-aggregation isn't ready but the report is requested, fall back to raw.
			if strings.TrimSpace(report.ReportType) == "daily_summary" || strings.TrimSpace(report.ReportType) == "weekly_summary" {
				overview, err = s.opsService.GetDashboardOverview(ctx, &OpsDashboardFilter{
					StartTime: start,
					EndTime:   end,
					Platform:  "",
					GroupID:   nil,
					QueryMode: OpsQueryModeRaw,
				})
			}
			if err != nil {
				return opsScheduledReportContent{}, err
			}
		}
		var errorLogs []*OpsErrorLog
		var avail *OpsAccountAvailability
		subCtx, subCancel := context.WithTimeout(ctx, 3*time.Second)
		defer subCancel()

		if s.opsService.opsRepo != nil {
			st := start
			et := end
			if errList, errErr := s.opsService.opsRepo.ListErrorLogs(subCtx, &OpsErrorLogFilter{
				StartTime:          &st,
				EndTime:            &et,
				ExcludeCountTokens: true,
				View:               "errors",
				Page:               1,
				PageSize:           200,
				SortBy:             "created_at",
				SortOrder:          "desc",
			}); errErr == nil && errList != nil {
				errorLogs = errList.Errors
			}
		}
		if accAvail, accErr := s.opsService.GetAccountAvailability(subCtx, "", nil); accErr == nil && accAvail != nil {
			avail = accAvail
		}
		enrichOpsErrorLogNames(subCtx, s.opsService, avail, errorLogs)

		return opsScheduledReportContent{
			html:      buildOpsSummaryEmailHTML(report.Name, start, end, overview),
			overview:  overview,
			errorLogs: errorLogs,
			avail:     avail,
			start:     start,
			end:       end,
		}, nil
	case "error_digest":
		// Lightweight digest: list recent errors (status>=400) and breakdown by type.
		startTime := start
		endTime := end
		filter := &OpsErrorLogFilter{
			StartTime:          &startTime,
			EndTime:            &endTime,
			ExcludeCountTokens: true,
			View:               "errors",
			Page:               1,
			PageSize:           100,
		}
		out, err := s.opsService.GetErrorLogs(ctx, filter)
		if err != nil {
			return opsScheduledReportContent{}, err
		}
		if report.ErrorDigestMinCount > 0 && out != nil && out.Total < report.ErrorDigestMinCount {
			return opsScheduledReportContent{}, nil
		}
		var digestLogs []*OpsErrorLog
		if out != nil && len(out.Errors) > 0 {
			enrichOpsErrorLogNames(ctx, s.opsService, nil, out.Errors)
			digestLogs = out.Errors
		}
		return opsScheduledReportContent{
			html:      buildOpsErrorDigestEmailHTML(report.Name, start, end, out, s.cfg),
			errorLogs: digestLogs,
			start:     start,
			end:       end,
		}, nil
	case "account_health":
		// Best-effort: use account availability (not error rate yet).
		avail, err := s.opsService.GetAccountAvailability(ctx, "", nil)
		if err != nil {
			return opsScheduledReportContent{}, err
		}
		_ = report.AccountHealthErrorRateThreshold // reserved for future per-account error rate report
		return opsScheduledReportContent{
			html:  buildOpsAccountHealthEmailHTML(report.Name, start, end, avail, s.cfg),
			avail: avail,
			start: start,
			end:   end,
		}, nil
	default:
		return opsScheduledReportContent{}, fmt.Errorf("unknown report type: %s", report.ReportType)
	}
}

func buildOpsSummaryEmailHTML(title string, start, end time.Time, overview *OpsDashboardOverview) string {
	if overview == nil {
		return fmt.Sprintf("<h2>%s</h2><p>No data.</p>", htmlEscape(title))
	}

	latP50 := "-"
	latP99 := "-"
	if overview.Duration.P50 != nil {
		latP50 = fmt.Sprintf("%dms", *overview.Duration.P50)
	}
	if overview.Duration.P99 != nil {
		latP99 = fmt.Sprintf("%dms", *overview.Duration.P99)
	}

	ttftP50 := "-"
	ttftP99 := "-"
	if overview.TTFT.P50 != nil {
		ttftP50 = fmt.Sprintf("%dms", *overview.TTFT.P50)
	}
	if overview.TTFT.P99 != nil {
		ttftP99 = fmt.Sprintf("%dms", *overview.TTFT.P99)
	}

	return fmt.Sprintf(`
<h2>%s</h2>
<p><b>Period</b>: %s ~ %s (UTC)</p>
<ul>
  <li><b>Total Requests</b>: %d</li>
  <li><b>Success</b>: %d</li>
  <li><b>Errors (SLA)</b>: %d</li>
  <li><b>Business Limited</b>: %d</li>
  <li><b>SLA</b>: %.2f%%</li>
  <li><b>Error Rate</b>: %.2f%%</li>
  <li><b>Upstream Error Rate (excl 429/529)</b>: %.2f%%</li>
  <li><b>Upstream Errors</b>: excl429/529=%d, 429=%d, 529=%d</li>
  <li><b>Latency</b>: p50=%s, p99=%s</li>
  <li><b>TTFT</b>: p50=%s, p99=%s</li>
  <li><b>Tokens</b>: %d</li>
  <li><b>QPS</b>: current=%.1f, peak=%.1f, avg=%.1f</li>
  <li><b>TPS</b>: current=%.1f, peak=%.1f, avg=%.1f</li>
</ul>
`,
		htmlEscape(strings.TrimSpace(title)),
		htmlEscape(start.UTC().Format(time.RFC3339)),
		htmlEscape(end.UTC().Format(time.RFC3339)),
		overview.RequestCountTotal,
		overview.SuccessCount,
		overview.ErrorCountSLA,
		overview.BusinessLimitedCount,
		overview.SLA*100,
		overview.ErrorRate*100,
		overview.UpstreamErrorRate*100,
		overview.UpstreamErrorCountExcl429529,
		overview.Upstream429Count,
		overview.Upstream529Count,
		htmlEscape(latP50),
		htmlEscape(latP99),
		htmlEscape(ttftP50),
		htmlEscape(ttftP99),
		overview.TokenConsumed,
		overview.QPS.Current,
		overview.QPS.Peak,
		overview.QPS.Avg,
		overview.TPS.Current,
		overview.TPS.Peak,
		overview.TPS.Avg,
	)
}

func buildOpsReportDiagnosisInput(
	report *opsScheduledReport,
	overview *OpsDashboardOverview,
	periodDisplay string,
	poolSummary string,
	clusters []opsAlertErrorCluster,
	locale string,
) *opsReportDiagnosisInput {
	input := &opsReportDiagnosisInput{
		ReportName:         opsScheduledReportLocalizedName(report, locale),
		PeriodDisplay:      periodDisplay,
		AccountPoolSummary: poolSummary,
		TopClusters:        clusters,
	}
	if report != nil {
		input.ReportType = strings.TrimSpace(report.ReportType)
	}
	if overview != nil {
		input.TotalRequests = overview.RequestCountTotal
		input.SLARequests = overview.RequestCountSLA
		input.SuccessCount = overview.SuccessCount
		input.ErrorCountSLA = overview.ErrorCountSLA
		input.BusinessLimitedCount = overview.BusinessLimitedCount
		input.SLAPercent = overview.SLA * 100
		input.ErrorRatePercent = overview.ErrorRate * 100
		input.UpstreamErrorRatePercent = overview.UpstreamErrorRate * 100
		input.UpstreamErrorCountExcl429529 = overview.UpstreamErrorCountExcl429529
		input.Upstream429Count = overview.Upstream429Count
		input.Upstream529Count = overview.Upstream529Count
		input.LatencyP50Ms = overview.Duration.P50
		input.LatencyP99Ms = overview.Duration.P99
		input.TTFTP50Ms = overview.TTFT.P50
		input.TTFTP99Ms = overview.TTFT.P99
		input.TokenConsumed = overview.TokenConsumed
	}
	return input
}

func buildOpsReportInsightsHTML(badge, summary string, clusters []opsAlertErrorCluster, accountPoolSummary, locale string) string {
	zh := isChineseOpsLocale(locale)
	var b strings.Builder

	if strings.TrimSpace(summary) != "" {
		b.WriteString(`<div style="margin:0 0 16px;padding:14px 16px;border-radius:8px;background:#f0fdfa;border:1px solid #ccfbf1;border-left:4px solid #0f766e;">`)
		b.WriteString(fmt.Sprintf(`<div style="font-size:12px;font-weight:700;color:#0f766e;margin-bottom:4px;">%s</div>`, html.EscapeString(badge)))
		b.WriteString(fmt.Sprintf(`<div style="font-size:14px;color:#0f172a;line-height:1.65;">%s</div>`, html.EscapeString(summary)))
		b.WriteString(`</div>`)
	}

	if strings.TrimSpace(accountPoolSummary) != "" {
		title := "Account Pool Health"
		if zh {
			title = "号池账号健康概览"
		}
		b.WriteString(`<div style="margin:0 0 16px;padding:12px 14px;border-radius:8px;background:#f8fafc;border:1px solid #e2e8f0;font-size:13px;line-height:1.6;">`)
		b.WriteString(fmt.Sprintf(`<div style="font-weight:700;color:#0f172a;margin-bottom:4px;">%s</div>`, html.EscapeString(title)))
		b.WriteString(fmt.Sprintf(`<div style="color:#334155;">%s</div>`, html.EscapeString(accountPoolSummary)))
		b.WriteString(`</div>`)
	}

	if len(clusters) > 0 {
		sectionTitle := "Top Translated Error Clusters"
		if zh {
			sectionTitle = "主要异常归类翻译（按次数排序）"
		}
		b.WriteString(fmt.Sprintf(`<div style="margin:18px 0 8px;font-size:14px;font-weight:700;color:#0f172a;">%s</div>`, html.EscapeString(sectionTitle)))
		for idx, c := range clusters {
			countUnit := fmt.Sprintf("%d times", c.Count)
			channelPrefix := "Account / Group: "
			reasonPrefix := "Reason: "
			actionPrefix := "Next step: "
			if zh {
				countUnit = fmt.Sprintf("%d 次", c.Count)
				channelPrefix = "账号与分组："
				reasonPrefix = "原因翻译："
				actionPrefix = "建议操作："
			}
			channelText := formatOpsClusterChannelLabel(c.GroupLabel, c.AccountLabel, locale)
			b.WriteString(`<div style="margin:0 0 10px;padding:12px 14px;border:1px solid #e2e8f0;border-radius:8px;background:#ffffff;">`)
			b.WriteString(fmt.Sprintf(`<div style="font-size:13px;font-weight:700;color:#0f172a;margin-bottom:4px;">%d. %s · %s <span style="display:inline-block;margin-left:6px;padding:1px 8px;border-radius:999px;background:#fee2e2;color:#b91c1c;font-size:12px;font-weight:600;">%s</span></div>`,
				idx+1, html.EscapeString(c.Platform), html.EscapeString(c.Model), html.EscapeString(countUnit)))
			b.WriteString(fmt.Sprintf(`<div style="font-size:12px;color:#475569;line-height:1.55;"><strong>%s</strong>%s</div>`,
				html.EscapeString(channelPrefix), html.EscapeString(channelText)))
			b.WriteString(fmt.Sprintf(`<div style="font-size:13px;color:#1e293b;line-height:1.55;margin-top:4px;"><strong>%s</strong>%s</div>`,
				html.EscapeString(reasonPrefix), html.EscapeString(c.Reason)))
			b.WriteString(fmt.Sprintf(`<div style="font-size:12px;color:#0f766e;line-height:1.55;margin-top:3px;"><strong>%s</strong>%s</div>`,
				html.EscapeString(actionPrefix), html.EscapeString(c.Suggestion)))
			b.WriteString(`</div>`)
		}
	}

	return b.String()
}

func buildOpsReportPrivateText(
	report *opsScheduledReport,
	overview *OpsDashboardOverview,
	periodDisplay string,
	badge string,
	summary string,
	poolSummary string,
	clusters []opsAlertErrorCluster,
	dashboardURL string,
) string {
	name := opsScheduledReportLocalizedName(report, "zh")
	var b strings.Builder
	b.WriteString(fmt.Sprintf("【GABUGU 运维%s】%s\n", name, periodDisplay))
	if strings.TrimSpace(summary) != "" {
		b.WriteString(fmt.Sprintf("结论（%s）：%s\n", badge, summary))
	}
	if overview != nil {
		b.WriteString(fmt.Sprintf("概览：总请求 %s 次（成功 %s / SLA 失败 %s），可用率 %.2f%%\n",
			formatOpsReportInteger(overview.RequestCountTotal),
			formatOpsReportInteger(overview.SuccessCount),
			formatOpsReportInteger(overview.ErrorCountSLA),
			overview.SLA*100))
	}
	if strings.TrimSpace(poolSummary) != "" {
		b.WriteString(fmt.Sprintf("号池：%s\n", poolSummary))
	}
	if len(clusters) > 0 {
		top := clusters[0]
		channelText := formatOpsClusterChannelLabel(top.GroupLabel, top.AccountLabel, "zh")
		b.WriteString(fmt.Sprintf("最多报错：%s · %s｜%s｜%d 次 — %s\n", top.Platform, top.Model, channelText, top.Count, top.Reason))
	}
	if dashboardURL != "" {
		b.WriteString(fmt.Sprintf("后台：%s", dashboardURL))
	}
	return strings.TrimSpace(b.String())
}

func buildOpsErrorDigestEmailHTML(title string, start, end time.Time, list *OpsErrorLogList, cfg *config.Config) string {
	total := 0
	recent := []*OpsErrorLog{}
	if list != nil {
		total = list.Total
		recent = list.Errors
	}
	clusters, affectedUsers := clusterOpsErrorLogs(recent, "zh", 5)
	if len(recent) > 10 {
		recent = recent[:10]
	}

	rows := ""
	for _, item := range recent {
		if item == nil {
			continue
		}
		_, reason, _ := translateOpsErrorReason(item.StatusCode, item.Phase, item.Type, item.Message, "zh")
		snippet := redactOpsErrorSnippet(item.Message, 100)
		modelName := strings.TrimSpace(item.RequestedModel)
		if modelName == "" {
			modelName = strings.TrimSpace(item.Model)
		}
		groupLabel, accountLabel := opsErrorLogGroupAndAccountLabels(item, "zh")
		channelText := formatOpsClusterChannelLabel(groupLabel, accountLabel, "zh")
		detailCell := htmlEscape(reason)
		if snippet != "" {
			detailCell += fmt.Sprintf(`<div style="color:#64748b;font-size:11px;margin-top:2px;">%s</div>`, htmlEscape(snippet))
		}
		rows += fmt.Sprintf(
			"<tr><td style=\"white-space:nowrap;\">%s</td><td>%s</td><td>%s · %s</td><td>%d</td><td>%s</td></tr>",
			htmlEscape(formatOpsLocalTime(item.CreatedAt, cfg, "zh")),
			htmlEscape(channelText),
			htmlEscape(item.Platform),
			htmlEscape(modelName),
			item.StatusCode,
			detailCell,
		)
	}
	if rows == "" {
		rows = "<tr><td colspan=\"5\">本周期内暂无异常错误记录。</td></tr>"
	}

	insights := buildOpsReportInsightsHTML("异常摘要归类", "", clusters, "", "zh")

	return fmt.Sprintf(`
<h2>%s</h2>
<p><b>统计周期</b>：%s</p>
<p><b>异常总次数</b>：%d（受影响用户 %d 位）</p>
%s
<h3>最近 10 条报错（已转为北京时间并翻译原因）</h3>
<table border="1" cellpadding="6" cellspacing="0" style="border-collapse:collapse;width:100%%;font-size:13px;">
  <thead><tr><th>北京时间</th><th>账号与分组</th><th>平台 / 模型</th><th>状态码</th><th>通俗原因与摘要</th></tr></thead>
  <tbody>%s</tbody>
</table>
`,
		htmlEscape(strings.TrimSpace(title)),
		htmlEscape(formatOpsLocalPeriod(start, end, cfg, "zh")),
		total,
		affectedUsers,
		insights,
		rows,
	)
}

func buildOpsAccountHealthEmailHTML(title string, start, end time.Time, avail *OpsAccountAvailability, cfg *config.Config) string {
	total := 0
	available := 0
	rateLimited := 0
	overloaded := 0
	hasError := 0
	issueItems := make([]string, 0, 8)
	limitedItems := make([]string, 0, 8)

	if avail != nil && len(avail.Accounts) > 0 {
		accounts := make([]*AccountAvailability, 0, len(avail.Accounts))
		for _, a := range avail.Accounts {
			if a != nil {
				accounts = append(accounts, a)
			}
		}
		sort.Slice(accounts, func(i, j int) bool {
			ni, nj := strings.TrimSpace(accounts[i].AccountName), strings.TrimSpace(accounts[j].AccountName)
			if ni != nj {
				return ni < nj
			}
			return accounts[i].AccountID < accounts[j].AccountID
		})
		for _, a := range accounts {
			total++
			if a.IsAvailable {
				available++
			}
			if a.IsRateLimited {
				rateLimited++
			}
			if a.IsOverloaded {
				overloaded++
			}
			name := strings.TrimSpace(a.AccountName)
			if name == "" {
				name = fmt.Sprintf("未命名账号（编号 %d）", a.AccountID)
			}
			metaParts := make([]string, 0, 2)
			if strings.TrimSpace(a.Platform) != "" {
				metaParts = append(metaParts, strings.TrimSpace(a.Platform))
			}
			if strings.TrimSpace(a.GroupName) != "" {
				metaParts = append(metaParts, "分组："+strings.TrimSpace(a.GroupName))
			}
			metaText := ""
			if len(metaParts) > 0 {
				metaText = "（" + strings.Join(metaParts, " · ") + "）"
			}
			if a.HasError {
				hasError++
				if len(issueItems) < 8 {
					msg := redactOpsErrorSnippet(a.ErrorMessage, 100)
					issueItems = append(issueItems, fmt.Sprintf("<li><strong>%s</strong>%s：%s</li>", htmlEscape(name), htmlEscape(metaText), htmlEscape(fallbackIfNonEmpty(msg, "账号处于异常状态"))))
				}
			} else if (a.IsRateLimited || a.IsOverloaded) && len(limitedItems) < 8 {
				stateText := "限流中（429）"
				if a.IsOverloaded && !a.IsRateLimited {
					stateText = "过载保护中（529）"
				} else if a.IsOverloaded && a.IsRateLimited {
					stateText = "限流且过载保护中"
				}
				limitedItems = append(limitedItems, fmt.Sprintf("<li><strong>%s</strong>%s：%s</li>", htmlEscape(name), htmlEscape(metaText), htmlEscape(stateText)))
			}
		}
	}

	issuesHTML := ""
	if len(issueItems) > 0 {
		issuesHTML += "<h3>异常账号清单</h3><ul>" + strings.Join(issueItems, "") + "</ul>"
	}
	if len(limitedItems) > 0 {
		issuesHTML += "<h3>限流 / 过载账号清单</h3><ul>" + strings.Join(limitedItems, "") + "</ul>"
	}

	return fmt.Sprintf(`
<h2>%s</h2>
<p><b>统计周期</b>：%s</p>
<ul>
  <li><b>账号总数</b>：%d</li>
  <li><b>正常可调度</b>：%d</li>
  <li><b>限流中（429）</b>：%d</li>
  <li><b>过载保护（529）</b>：%d</li>
  <li><b>异常报错</b>：%d</li>
</ul>
%s
<p style="color:#64748b;font-size:12px;">说明：以上数据反映生成报表时刻的账号池实时可用状态。</p>
`,
		htmlEscape(strings.TrimSpace(title)),
		htmlEscape(formatOpsLocalPeriod(start, end, cfg, "zh")),
		total,
		available,
		rateLimited,
		overloaded,
		hasError,
		issuesHTML,
	)
}

func (s *OpsScheduledReportService) tryAcquireLeaderLock(ctx context.Context) (func(), bool) {
	if s == nil || !s.distributedLockOn {
		return nil, true
	}
	if s.redisClient == nil {
		s.warnNoRedisOnce.Do(func() {
			log.Printf("[OpsScheduledReport] redis not configured; running without distributed lock")
		})
		return nil, true
	}
	if ctx == nil {
		ctx = context.Background()
	}

	key := opsScheduledReportLeaderLockKeyDefault
	ttl := opsScheduledReportLeaderLockTTLDefault
	if strings.TrimSpace(key) == "" {
		key = "ops:scheduled_reports:leader"
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	ok, err := s.redisClient.SetNX(ctx, key, s.instanceID, ttl).Result()
	if err != nil {
		// Prefer fail-closed to avoid duplicate report sends when Redis is flaky.
		log.Printf("[OpsScheduledReport] leader lock SetNX failed; skipping this cycle: %v", err)
		return nil, false
	}
	if !ok {
		return nil, false
	}
	return func() {
		_, _ = opsScheduledReportReleaseScript.Run(ctx, s.redisClient, []string{key}, s.instanceID).Result()
	}, true
}

func (s *OpsScheduledReportService) getLastRunAt(ctx context.Context, reportType string) time.Time {
	if s == nil || s.redisClient == nil {
		return time.Time{}
	}
	kind := strings.TrimSpace(reportType)
	if kind == "" {
		return time.Time{}
	}
	key := opsScheduledReportLastRunKeyPrefix + kind

	raw, err := s.redisClient.Get(ctx, key).Result()
	if err != nil || strings.TrimSpace(raw) == "" {
		return time.Time{}
	}
	sec, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || sec <= 0 {
		return time.Time{}
	}
	last := time.Unix(sec, 0)
	// Cron schedules are interpreted in the configured timezone (s.loc). Ensure the base time
	// passed into cron.Next() uses the same location; otherwise the job will drift by timezone
	// offset (e.g. Asia/Shanghai default would run 8h later after the first execution).
	if s.loc != nil {
		return last.In(s.loc)
	}
	return last.UTC()
}

func (s *OpsScheduledReportService) setLastRunAt(ctx context.Context, reportType string, t time.Time) {
	if s == nil || s.redisClient == nil {
		return
	}
	kind := strings.TrimSpace(reportType)
	if kind == "" {
		return
	}
	if t.IsZero() {
		t = time.Now().UTC()
	}
	key := opsScheduledReportLastRunKeyPrefix + kind
	_ = s.redisClient.Set(ctx, key, strconv.FormatInt(t.UTC().Unix(), 10), 14*24*time.Hour).Err()
}

func (s *OpsScheduledReportService) recordHeartbeatSuccess(runAt time.Time, duration time.Duration, result string) {
	if s == nil || s.opsService == nil || s.opsService.opsRepo == nil {
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
	_ = s.opsService.opsRepo.UpsertJobHeartbeat(ctx, &OpsUpsertJobHeartbeatInput{
		JobName:        opsScheduledReportJobName,
		LastRunAt:      &runAt,
		LastSuccessAt:  &now,
		LastDurationMs: &durMs,
		LastResult:     &msg,
	})
}

func (s *OpsScheduledReportService) recordHeartbeatError(runAt time.Time, duration time.Duration, err error) {
	if s == nil || s.opsService == nil || s.opsService.opsRepo == nil || err == nil {
		return
	}
	now := time.Now().UTC()
	durMs := duration.Milliseconds()
	msg := truncateString(err.Error(), 2048)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.opsService.opsRepo.UpsertJobHeartbeat(ctx, &OpsUpsertJobHeartbeatInput{
		JobName:        opsScheduledReportJobName,
		LastRunAt:      &runAt,
		LastErrorAt:    &now,
		LastError:      &msg,
		LastDurationMs: &durMs,
	})
}

func normalizeEmails(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		addr := strings.ToLower(strings.TrimSpace(raw))
		if addr == "" {
			continue
		}
		if _, ok := seen[addr]; ok {
			continue
		}
		seen[addr] = struct{}{}
		out = append(out, addr)
	}
	return out
}
