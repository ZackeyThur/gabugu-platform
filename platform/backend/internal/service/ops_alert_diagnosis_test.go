package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestOpsAlertDiagnosisTimeAndMetricFormatting(t *testing.T) {
	t.Parallel()

	utcTime := time.Date(2026, time.November, 24, 1, 54, 59, 0, time.UTC)
	zhTime := formatOpsLocalTime(utcTime, &config.Config{Timezone: "Asia/Shanghai"}, "zh")
	require.Contains(t, zhTime, "2026-11-24 09:54:59")
	require.Contains(t, zhTime, "北京时间")

	period := formatOpsLocalPeriod(
		time.Date(2026, time.November, 23, 1, 0, 0, 0, time.UTC),
		time.Date(2026, time.November, 24, 1, 0, 0, 0, time.UTC),
		nil,
		"zh",
	)
	require.Equal(t, "2026-11-23 09:00 ~ 2026-11-24 09:00（北京时间）", period)

	require.Equal(t, "不足 1 分钟", formatOpsDurationHuman(0, "zh"))
	require.Equal(t, "45 秒", formatOpsDurationHuman(45*time.Second, "zh"))
	require.Equal(t, "4 分钟", formatOpsDurationHuman(4*time.Minute, "zh"))
	require.Equal(t, "1 小时 5 分", formatOpsDurationHuman(65*time.Minute, "zh"))

	val := 50.0
	thr := 20.0
	summary := formatOpsMetricSummary("error_rate", ">", &val, &thr, "zh")
	require.Equal(t, "请求错误率 当前 50.00%（告警条件：> 20.00%）", summary)
}

func TestOpsAlertDiagnosisRedactionAndTranslation(t *testing.T) {
	t.Parallel()

	raw := "Authorization: Bearer secret_token_12345 failed with sk-abcdef1234567890 for ops@example.com?api_key=verysecret&foo=1"
	redacted := redactOpsErrorSnippet(raw, 200)
	require.NotContains(t, redacted, "secret_token_12345")
	require.NotContains(t, redacted, "sk-abcdef1234567890")
	require.NotContains(t, redacted, "ops@example.com")
	require.NotContains(t, redacted, "verysecret")
	require.Contains(t, redacted, "[REDACTED]")
	require.Contains(t, redacted, "[REDACTED_KEY]")
	require.Contains(t, redacted, "[REDACTED_EMAIL]")

	cases := []struct {
		statusCode int
		phase      string
		errType    string
		msg        string
		wantKey    string
		wantSubstr string
	}{
		{524, "upstream", "", "context deadline exceeded", "upstream_timeout", "超时"},
		{499, "upstream", "", "client canceled", "client_canceled", "提前断开"},
		{503, "routing", "", "no available compatible accounts", "no_available_account", "暂无可调度"},
		{502, "upstream", "", "bad gateway", "upstream_502", "网关异常"},
		{529, "upstream", "", "server overloaded", "upstream_overloaded", "过载"},
		{429, "upstream", "rate_limit_error", "daily_limit reached", "daily_quota_limit", "额度或配额上限"},
		{429, "upstream", "rate_limit_error", "too many requests", "rate_limit", "频率/并发限制"},
		{401, "upstream", "", "unauthorized invalid token", "upstream_auth", "凭据失效"},
		{403, "upstream", "", "insufficient balance", "insufficient_balance", "额度不足"},
		{404, "routing", "", "model not supported", "model_route_mismatch", "未映射"},
		{400, "request", "invalid_request_error", "invalid json schema", "invalid_request", "请求参数"},
		{0, "network", "", "dial tcp: connection reset by peer", "network_proxy", "网络连接中断"},
		{500, "internal", "", "unexpected panic", "internal_error", "内部处理异常"},
	}
	for _, tc := range cases {
		key, reason, sugg := translateOpsErrorReason(tc.statusCode, tc.phase, tc.errType, tc.msg, "zh")
		require.Equal(t, tc.wantKey, key)
		require.Contains(t, reason, tc.wantSubstr)
		require.NotEmpty(t, sugg)
	}
}

func TestClusterOpsErrorLogsAndDetailsHTMLEscaping(t *testing.T) {
	t.Parallel()

	u1 := int64(10)
	u2 := int64(20)
	g4 := int64(4)
	g9 := int64(9)
	a88 := int64(88)
	a12 := int64(12)

	logs := []*OpsErrorLog{
		{
			Platform:       "openai",
			RequestedModel: "gpt-6.2-sol",
			GroupID:        &g4,
			GroupName:      `主力分组<script>alert("g")</script>`,
			AccountID:      &a88,
			AccountName:    "主力 GPT 账号",
			StatusCode:     524,
			Phase:          "upstream",
			Message:        "upstream timeout with sk-secretkey1234567890 <img src=x onerror=alert(1)>",
			UserID:         &u1,
		},
		{
			Platform:       "openai",
			RequestedModel: "gpt-6.2-sol",
			GroupID:        &g4,
			GroupName:      `主力分组<script>alert("g")</script>`,
			AccountID:      &a88,
			AccountName:    "主力 GPT 账号",
			StatusCode:     524,
			Phase:          "upstream",
			Message:        "upstream timeout again",
			UserID:         &u2,
		},
		{
			Platform:       "anthropic",
			RequestedModel: "claude-opus-5-5",
			AccountID:      &a12,
			AccountName:    "主力 Claude 账号",
			StatusCode:     502,
			Phase:          "upstream",
			Message:        "bad gateway",
			UserID:         &u1,
		},
		{
			Platform:       "anthropic",
			RequestedModel: "claude-opus-5-5",
			GroupID:        &g9,
			GroupName:      "低价 Claude 分组",
			StatusCode:     503,
			Phase:          "routing",
			Message:        "no available compatible accounts",
			UserID:         &u2,
		},
	}

	clusters, users := clusterOpsErrorLogs(logs, "zh", 5)
	require.Equal(t, 2, users)
	require.Len(t, clusters, 3)
	require.Equal(t, 2, clusters[0].Count)
	require.Equal(t, "openai", clusters[0].Platform)
	require.Equal(t, "主力 GPT 账号", clusters[0].AccountLabel)
	require.NotContains(t, clusters[0].AccountLabel, "#88")
	require.NotContains(t, clusters[0].GroupLabel, "#4")
	require.NotContains(t, clusters[0].SampleMessage, "sk-secretkey1234567890")

	require.Equal(t, "主力 GPT 账号（分组：主力 GPT 分组）", formatOpsClusterChannelLabel("主力 GPT 分组", "主力 GPT 账号", "zh"))
	require.Equal(t, "主力 Claude 账号", formatOpsClusterChannelLabel("-", "主力 Claude 账号", "zh"))
	require.Equal(t, "未分配具体账号（分组：低价 Claude 分组）", formatOpsClusterChannelLabel("低价 Claude 分组", "-", "zh"))
	require.Equal(t, "未分配账号", formatOpsClusterChannelLabel("-", "-", "zh"))

	// enrichOpsErrorLogNames fills blank names from availability snapshot.
	a99 := int64(99)
	g5 := int64(5)
	missingNameLogs := []*OpsErrorLog{
		{AccountID: &a99, GroupID: &g5},
	}
	avail := &OpsAccountAvailability{
		Accounts: map[int64]*AccountAvailability{
			99: {AccountID: 99, AccountName: "备用稳定号", GroupID: 5, GroupName: "主力 GPT 分组", HasError: true},
			12: {AccountID: 12, AccountName: "主力 Claude 账号", GroupID: 6, GroupName: "Claude 专线", IsRateLimited: true},
			10: {AccountID: 10, AccountName: "主力 GPT 账号", GroupID: 5, GroupName: "主力 GPT 分组", IsAvailable: true},
		},
	}
	enrichOpsErrorLogNames(context.Background(), nil, avail, missingNameLogs)
	require.Equal(t, "备用稳定号", missingNameLogs[0].AccountName)
	require.Equal(t, "主力 GPT 分组", missingNameLogs[0].GroupName)

	poolSummary := formatOpsAvailabilitySnapshotSummary(avail, "zh")
	require.Contains(t, poolSummary, "异常账号：备用稳定号")
	require.Contains(t, poolSummary, "限流/过载账号：主力 Claude 账号")
	require.NotContains(t, poolSummary, "#99")

	resolvedAt := time.Date(2026, time.November, 24, 2, 4, 0, 0, time.UTC)
	ev := &opsAlertEvidence{
		Locale:              "zh",
		RuleName:            "错误率极高",
		Severity:            "P0",
		Status:              OpsAlertStatusResolved,
		ResolvedAt:          &resolvedAt,
		ResolvedAtLocal:     "2026-11-24 10:04:00（北京时间）",
		DurationHuman:       "4 分钟",
		ShowRequestOverview: true,
		SLARequests:         12,
		SuccessCount:        10,
		ErrorCount:          2,
		AffectedUsers:       users,
		TopClusters:         clusters,
	}

	detailsHTML := buildOpsAlertDetailsHTML(ev, "zh")
	require.Contains(t, detailsHTML, "恢复确认：")
	require.Contains(t, detailsHTML, "2026-11-24 10:04:00（北京时间）")
	require.Contains(t, detailsHTML, "账号与分组：")
	require.Contains(t, detailsHTML, "主力 GPT 账号")
	require.Contains(t, detailsHTML, "未分配具体账号（分组：低价 Claude 分组）")
	require.NotContains(t, detailsHTML, "#88")
	require.NotContains(t, detailsHTML, "(#4)")
	require.NotContains(t, detailsHTML, "分组 #4")
	require.NotContains(t, detailsHTML, `<script>alert("g")</script>`)
	require.NotContains(t, detailsHTML, `<img src=x onerror=alert(1)>`)
	require.Contains(t, detailsHTML, "&lt;script&gt;")
	require.Contains(t, detailsHTML, "&lt;img src=x onerror=alert(1)&gt;")
}

// TestGenerateOpsEmailVisualPreviews writes the 4 redesigned Chinese emails to OPS_PREVIEW_OUT_DIR
// when that environment variable is explicitly set, enabling local Playwright visual verification.
func TestGenerateOpsEmailVisualPreviews(t *testing.T) {
	outDir := strings.TrimSpace(os.Getenv("OPS_PREVIEW_OUT_DIR"))
	if outDir == "" {
		t.Skip("OPS_PREVIEW_OUT_DIR not set")
	}
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	ctx := context.Background()
	repo := newNotificationEmailMemorySettingRepo()
	require.NoError(t, repo.Set(ctx, SettingKeySiteName, "GABUGU"))
	require.NoError(t, repo.Set(ctx, SettingKeyFrontendURL, "https://gabugu.com"))
	svc := NewNotificationEmailService(repo, nil)

	clusters := []opsAlertErrorCluster{
		{
			Platform:      "openai",
			Model:         "gpt-6.2-sol",
			GroupLabel:    "主力 GPT 分组",
			AccountLabel:  "主力 GPT 账号",
			StatusCode:    524,
			Phase:         "upstream",
			Count:         5,
			ReasonKey:     "upstream_timeout",
			Reason:        "上游服务响应超时（524），长时间未返回结果",
			Suggestion:    "检查对应账号的线路或代理延迟；若反复出现，可暂时停用该账号或切换备用线路",
			SampleMessage: "upstream gateway timeout (524) before client disconnect",
		},
		{
			Platform:      "anthropic",
			Model:         "claude-opus-5-5",
			GroupLabel:    "Claude 专线",
			AccountLabel:  "主力 Claude 账号",
			StatusCode:    502,
			Phase:         "upstream",
			Count:         1,
			ReasonKey:     "upstream_502",
			Reason:        "上游服务商接口繁忙或网关异常（502）",
			Suggestion:    "多为上游瞬时波动；若持续超过 5 分钟，建议检查账号线路或临时切流",
			SampleMessage: "The service is busy. Please retry later.",
		},
	}

	writePreview := func(filename, event string, vars, raw map[string]string) {
		tmpl, err := svc.GetTemplate(ctx, event, "zh")
		require.NoError(t, err)
		rendered, err := renderNotificationEmail(event, tmpl.Subject, tmpl.HTML, vars, raw)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(outDir, filename), []byte(rendered.HTML), 0o644))
	}

	firedAt := time.Date(2026, time.November, 24, 1, 55, 0, 0, time.UTC)
	valAI := 25.00
	thrAI := 20.00
	ruleAI := &OpsAlertRule{ID: 8, Name: "错误率极高", Severity: "P0", MetricType: "error_rate", Operator: ">", Threshold: thrAI, WindowMinutes: 5}
	evAI := &opsAlertEvidence{
		Locale:              "zh",
		RuleID:              8,
		RuleName:            "错误率极高",
		Severity:            "P0",
		SeverityLabel:       opsSeverityLabel("P0", "zh"),
		Status:              OpsAlertStatusFiring,
		StatusLabel:         opsStatusLabel(OpsAlertStatusFiring, "zh"),
		MetricType:          "error_rate",
		MetricLabel:         "请求错误率",
		Operator:            ">",
		MetricValue:         &valAI,
		ThresholdValue:      &thrAI,
		MetricSummary:       formatOpsMetricSummary("error_rate", ">", &valAI, &thrAI, "zh"),
		ScopeLabel:          "全站全部渠道",
		WindowMinutes:       5,
		WindowLabel:         "最近 5 分钟",
		FiredAt:             firedAt,
		FiredAtLocal:        formatOpsLocalTime(firedAt, nil, "zh"),
		DurationHuman:       "5 分钟",
		TotalRequests:       24,
		SLARequests:         24,
		SuccessCount:        18,
		ErrorCount:          6,
		AffectedUsers:       2,
		ShowRequestOverview: true,
		AccountPoolSummary:  "账号总计 18 个 · 正常可用 16 个 · 限流 1 个 · 过载 0 个 · 异常 1 个｜异常账号：主力 GPT 账号｜限流/过载账号：主力 Claude 账号",
		DashboardURL:        "https://gabugu.com/admin/ops",
		TopClusters:         clusters,
	}
	aiSummary := "最近 5 分钟全站 24 次有效请求中失败 6 次，影响 2 位用户。主要集中在账号「主力 GPT 账号」（所属分组：主力 GPT 分组）调用 gpt-6.2-sol 时连续出现 524 上游超时。建议立即在后台检查账号「主力 GPT 账号」的代理线路延迟，必要时临时切换至备用账号。"
	varsAI := opsAlertEmailVariablesWithEvidence(ruleAI, &OpsAlertEvent{ID: 101, RuleID: 8, Severity: "P0", Status: OpsAlertStatusFiring, FiredAt: firedAt, MetricValue: &valAI, ThresholdValue: &thrAI}, evAI, opsDiagnosisBadge(true, true, "zh"), aiSummary, "zh")
	varsAI["site_name"] = "GABUGU"
	writePreview("01-alert-ai-zh.html", NotificationEmailEventOpsAlert, varsAI, map[string]string{"alert_details_html": buildOpsAlertDetailsHTML(evAI, "zh")})

	// 2. Fallback alert (AI timeout/unavailable)
	valFB := 28.57
	thrFB := 15.00
	ruleFB := &OpsAlertRule{ID: 12, Name: "请求错误率过高", Severity: "P1", MetricType: "error_rate", Operator: ">", Threshold: thrFB, WindowMinutes: 5}
	fbClusters := []opsAlertErrorCluster{clusters[0], clusters[1]}
	fbClusters[0].Count = 3
	fbClusters[1].Count = 1
	evFB := *evAI
	evFB.Severity = "P1"
	evFB.SeverityLabel = opsSeverityLabel("P1", "zh")
	evFB.RuleName = ruleFB.Name
	evFB.MetricValue = &valFB
	evFB.ThresholdValue = &thrFB
	evFB.MetricSummary = formatOpsMetricSummary("error_rate", ">", &valFB, &thrFB, "zh")
	evFB.TotalRequests = 14
	evFB.SLARequests = 14
	evFB.SuccessCount = 10
	evFB.ErrorCount = 4
	evFB.LowSampleWarning = false
	evFB.TopClusters = fbClusters
	evFB.FallbackSummary, evFB.Suggestions = buildRuleBasedAlertDiagnosis(&evFB, "zh")
	varsFB := opsAlertEmailVariablesWithEvidence(ruleFB, &OpsAlertEvent{ID: 102, RuleID: 12, Severity: "P1", Status: OpsAlertStatusFiring, FiredAt: firedAt, MetricValue: &valFB, ThresholdValue: &thrFB}, &evFB, opsDiagnosisBadge(false, true, "zh"), evFB.FallbackSummary, "zh")
	varsFB["site_name"] = "GABUGU"
	writePreview("02-alert-fallback-zh.html", NotificationEmailEventOpsAlert, varsFB, map[string]string{"alert_details_html": buildOpsAlertDetailsHTML(&evFB, "zh")})

	// 3. Auto-recovery alert
	resolvedAt := firedAt.Add(4 * time.Minute)
	valRec := 1.20
	evRec := *evAI
	evRec.Status = OpsAlertStatusResolved
	evRec.StatusLabel = opsStatusLabel(OpsAlertStatusResolved, "zh")
	evRec.MetricValue = &valRec
	evRec.MetricSummary = formatOpsMetricSummary("error_rate", ">", &valRec, &thrAI, "zh")
	evRec.ResolvedAt = &resolvedAt
	evRec.ResolvedAtLocal = formatOpsLocalTime(resolvedAt, nil, "zh")
	evRec.DurationHuman = "4 分钟"
	evRec.SuccessCount = 41
	evRec.ErrorCount = 1
	evRec.SLARequests = 42
	evRec.TotalRequests = 42
	evRec.TopClusters = clusters[:1]
	evRec.AccountPoolSummary = "账号总计 18 个 · 正常可用 18 个 · 限流 0 个 · 过载 0 个 · 异常 0 个"
	evRec.FallbackSummary, _ = buildRuleBasedAlertDiagnosis(&evRec, "zh")
	varsRec := opsAlertEmailVariablesWithEvidence(ruleAI, &OpsAlertEvent{ID: 101, RuleID: 8, Severity: "P0", Status: OpsAlertStatusResolved, FiredAt: firedAt, ResolvedAt: &resolvedAt, MetricValue: &valRec, ThresholdValue: &thrAI}, &evRec, opsDiagnosisBadge(false, false, "zh"), evRec.FallbackSummary, "zh")
	varsRec["site_name"] = "GABUGU"
	writePreview("03-alert-recovered-zh.html", NotificationEmailEventOpsAlert, varsRec, map[string]string{"alert_details_html": buildOpsAlertDetailsHTML(&evRec, "zh")})

	// 4. Daily Chinese health digest
	reportNow := time.Date(2026, time.November, 24, 1, 0, 0, 0, time.UTC)
	reportStart := reportNow.Add(-24 * time.Hour)
	latP50, latP99, ttftP50, ttftP99 := 8231, 151260, 1674, 11222
	overview := &OpsDashboardOverview{
		RequestCountTotal:            2374,
		SuccessCount:                 1451,
		ErrorCountSLA:                2,
		BusinessLimitedCount:         921,
		SLA:                          0.9986,
		ErrorRate:                    0.0014,
		UpstreamErrorRate:            0.0028,
		UpstreamErrorCountExcl429529: 4,
		Upstream429Count:             0,
		Upstream529Count:             0,
		TokenConsumed:                121550190,
		Duration:                     OpsPercentiles{P50: &latP50, P99: &latP99},
		TTFT:                         OpsPercentiles{P50: &ttftP50, P99: &ttftP99},
		QPS:                          OpsRateSummary{Current: 0.4, Peak: 1.8, Avg: 0.3},
		TPS:                          OpsRateSummary{Current: 312.5, Peak: 2450.2, Avg: 418.6},
	}
	report := &opsScheduledReport{Name: "日报", ReportType: "daily_summary", TimeRange: 24 * time.Hour}
	reportVars := opsSummaryReportEmailVariables(report, reportNow, overview, "zh")
	reportVars["site_name"] = "GABUGU"
	reportVars["report_period_display"] = formatOpsLocalPeriod(reportStart, reportNow, nil, "zh")
	reportVars["report_insights_display"] = "block"
	reportVars["dashboard_url"] = "https://gabugu.com/admin/ops"
	reportVars["dashboard_cta_display"] = "block"
	reportSummary := "过去 24 小时全站共处理 2,374 次请求，真实可用率达到 99.86%，运行整体平稳。仅出现 2 次影响用户的报错（集中在账号「主力 GPT 账号」偶发 524 超时），当前账号池绝大多数账号健康可用，无需紧急干预。"
	reportClusters := []opsAlertErrorCluster{clusters[0]}
	reportClusters[0].Count = 2
	insightsHTML := buildOpsReportInsightsHTML(
		opsDiagnosisBadge(true, true, "zh"),
		reportSummary,
		reportClusters,
		"账号总计 18 个 · 正常可用 16 个 · 限流 1 个 · 过载 0 个 · 异常 1 个｜异常账号：主力 GPT 账号｜限流/过载账号：主力 Claude 账号",
		"zh",
	)
	writePreview("04-daily-report-zh.html", NotificationEmailEventOpsScheduledReport, reportVars, map[string]string{"report_insights_html": insightsHTML})
}
