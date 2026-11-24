package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestOpsSummaryReportEmailVariables(t *testing.T) {
	latencyP50 := 8231
	latencyP99 := 151260
	ttftP50 := 1674
	ttftP99 := 11222
	now := time.Date(2026, time.July, 19, 1, 0, 26, 0, time.UTC)
	report := &opsScheduledReport{
		Name:       "日报",
		ReportType: "daily_summary",
		TimeRange:  24 * time.Hour,
	}
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
		Duration:                     OpsPercentiles{P50: &latencyP50, P99: &latencyP99},
		TTFT:                         OpsPercentiles{P50: &ttftP50, P99: &ttftP99},
		QPS:                          OpsRateSummary{Current: 0, Peak: 1.2, Avg: 0},
		TPS:                          OpsRateSummary{Current: 0, Peak: 133421.2, Avg: 1406.8},
	}

	variables := opsSummaryReportEmailVariables(report, now, overview, "en")
	require.Equal(t, "Daily summary", variables["report_name"])
	require.Equal(t, "daily_summary", variables["report_type"])
	require.Equal(t, "2026-07-18T01:00:26Z", variables["report_start_time"])
	require.Equal(t, "2026-07-19T01:00:26Z", variables["report_end_time"])
	require.Equal(t, "2,374", variables["report_total_requests"])
	require.Equal(t, "1,451", variables["report_success_count"])
	require.Equal(t, "2", variables["report_sla_error_count"])
	require.Equal(t, "921", variables["report_business_limited_count"])
	require.Equal(t, "99.86%", variables["report_sla"])
	require.Equal(t, "0.14%", variables["report_error_rate"])
	require.Equal(t, "0.28%", variables["report_upstream_error_rate"])
	require.Equal(t, "4", variables["report_upstream_error_count_excl_429_529"])
	require.Equal(t, "8,231 ms", variables["report_latency_p50"])
	require.Equal(t, "151,260 ms", variables["report_latency_p99"])
	require.Equal(t, "1,674 ms", variables["report_ttft_p50"])
	require.Equal(t, "11,222 ms", variables["report_ttft_p99"])
	require.Equal(t, "121,550,190", variables["report_tokens"])
	require.Equal(t, "1.2", variables["report_qps_peak"])
	require.Equal(t, "133421.2", variables["report_tps_peak"])
	require.Equal(t, "1406.8", variables["report_tps_avg"])
	require.Empty(t, variables["report_html"])
	require.Equal(t, "none", variables["report_detail_display"])

	zhVariables := opsSummaryReportEmailVariables(report, now, overview, "zh-CN")
	require.Equal(t, "日报", zhVariables["report_name"])
	emptyVariables := opsSummaryReportEmailVariables(report, now, nil, "en")
	require.Equal(t, "block", emptyVariables["report_summary_display"])
	require.Equal(t, "none", emptyVariables["report_detail_display"])

	legacyTemplate, err := renderNotificationEmail(
		NotificationEmailEventOpsScheduledReport,
		"Report",
		`<section>{{report_html}}</section>`,
		variables,
		map[string]string{"report_html": `<h2>generated summary</h2>`},
	)
	require.NoError(t, err)
	require.Contains(t, legacyTemplate.HTML, `<h2>generated summary</h2>`)
}

func TestOpsScheduledReportVariablesDoNotUsePreviewMetrics(t *testing.T) {
	now := time.Date(2026, time.July, 19, 1, 0, 26, 0, time.UTC)
	report := &opsScheduledReport{
		Name:       "错误摘要",
		ReportType: "error_digest",
		TimeRange:  24 * time.Hour,
	}

	variables := opsScheduledReportLocalizedEmailVariables(report, now, "en")
	require.Equal(t, "Error digest", variables["report_name"])
	require.Empty(t, variables["report_html"])
	require.Equal(t, "block", variables["report_detail_display"])
	for _, placeholder := range notificationEmailOpsSummaryPlaceholders {
		if placeholder == "report_summary_display" {
			require.Equal(t, "none", variables[placeholder])
			continue
		}
		require.Equal(t, "-", variables[placeholder])
	}
}

func TestOpsScheduledReportLegacyTemplateReceivesSummaryHTML(t *testing.T) {
	ctx := context.Background()
	repo := newNotificationEmailMemorySettingRepo()
	smtpServer := startNotificationEmailTestSMTPServer(t)
	require.NoError(t, repo.SetMultiple(ctx, smtpServer.settings()))

	emailService := NewEmailService(repo, nil)
	notificationService := NewNotificationEmailService(repo, emailService)
	_, err := notificationService.UpdateTemplate(
		ctx,
		NotificationEmailEventOpsScheduledReport,
		"en",
		"Legacy report {{report_name}}",
		`<section data-template="legacy">{{report_html}}</section>`,
	)
	require.NoError(t, err)

	svc := &OpsScheduledReportService{
		opsService:   &OpsService{opsRepo: &opsRepoMock{}},
		emailService: emailService,
	}
	report := &opsScheduledReport{
		Name:       "日报",
		ReportType: "daily_summary",
		TimeRange:  24 * time.Hour,
		Recipients: []string{"ops@example.com"},
	}

	attempts, err := svc.runReport(ctx, report, time.Date(2026, time.July, 19, 1, 0, 26, 0, time.UTC))
	require.NoError(t, err)
	require.Equal(t, 1, attempts)
	require.Equal(t, int64(1), smtpServer.messageCount())
	messageBody := smtpServer.lastMessageBody(t)
	require.Contains(t, messageBody, `<section data-template="legacy">`)
	require.Contains(t, messageBody, `<h2>日报</h2>`)
}

func TestFormatOpsReportIntegerGroupsDigits(t *testing.T) {
	require.Equal(t, "2,374", formatOpsReportInteger(2374))
	require.Equal(t, "-1,234", formatOpsReportInteger(-1234))
	require.Equal(t, "42", formatOpsReportInteger(42))
}

type testReportOpsRepo struct {
	opsRepoMock
	overview  *OpsDashboardOverview
	errorLogs *OpsErrorLogList
}

func (m *testReportOpsRepo) GetDashboardOverview(_ context.Context, _ *OpsDashboardFilter) (*OpsDashboardOverview, error) {
	if m.overview != nil {
		return m.overview, nil
	}
	return &OpsDashboardOverview{}, nil
}

func (m *testReportOpsRepo) ListErrorLogs(_ context.Context, _ *OpsErrorLogFilter) (*OpsErrorLogList, error) {
	return m.errorLogs, nil
}

func TestOpsScheduledReportChineseDailyDigestAndPrivateWebhook(t *testing.T) {
	ctx := context.Background()
	settingRepo := newNotificationEmailMemorySettingRepo()
	smtpServer := startNotificationEmailTestSMTPServer(t)
	require.NoError(t, settingRepo.SetMultiple(ctx, smtpServer.settings()))
	require.NoError(t, settingRepo.Set(ctx, SettingKeySiteName, "GABUGU"))
	require.NoError(t, settingRepo.Set(ctx, SettingKeyFrontendURL, "https://gabugu.com"))

	var dmReceived opsPrivateWebhookPayload
	var dmCalls int
	dmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dmCalls++
		require.NoError(t, json.NewDecoder(r.Body).Decode(&dmReceived))
		w.WriteHeader(http.StatusOK)
	}))
	defer dmSrv.Close()

	emailService := NewEmailService(settingRepo, nil)
	notificationService := NewNotificationEmailService(settingRepo, emailService)
	notificationService.RememberRecipientLocale(ctx, 0, "owner@example.com", "zh-CN")

	g4 := int64(4)
	a88 := int64(88)
	repo := &testReportOpsRepo{
		overview: &OpsDashboardOverview{
			RequestCountTotal:    2374,
			RequestCountSLA:      1453,
			SuccessCount:         1451,
			ErrorCountSLA:        2,
			BusinessLimitedCount: 921,
			SLA:                  0.9986,
			ErrorRate:            0.0014,
			TokenConsumed:        121550190,
		},
		errorLogs: &OpsErrorLogList{
			Total: 2,
			Errors: []*OpsErrorLog{
				{
					Platform:       "openai",
					RequestedModel: "gpt-6.2-sol",
					GroupID:        &g4,
					GroupName:      "主力 GPT 分组",
					AccountID:      &a88,
					AccountName:    "主力 GPT 账号",
					StatusCode:     524,
					Phase:          "upstream",
					Message:        "upstream timeout",
				},
			},
		},
	}

	opsCfg := &config.Config{
		Timezone: "Asia/Shanghai",
		Ops:      config.OpsConfig{Enabled: true},
	}
	opsService := &OpsService{
		opsRepo:     repo,
		settingRepo: settingRepo,
		cfg:         opsCfg,
		getAccountAvailability: func(_ context.Context, _ string, _ *int64) (*OpsAccountAvailability, error) {
			return &OpsAccountAvailability{
				Accounts: map[int64]*AccountAvailability{
					88: {AccountID: 88, AccountName: "主力 GPT 账号", GroupID: 4, GroupName: "主力 GPT 分组", IsAvailable: true},
					12: {AccountID: 12, AccountName: "主力 Claude 账号", GroupID: 5, GroupName: "Claude 专线", IsRateLimited: true},
				},
			}, nil
		},
	}
	_, err := opsService.UpdateEmailNotificationConfig(ctx, &OpsEmailNotificationConfigUpdateRequest{
		PrivateWebhook: &OpsPrivateWebhookConfig{
			Enabled:        true,
			URL:            dmSrv.URL,
			Target:         "wxid_owner_only",
			IncludeReports: true,
		},
	})
	require.NoError(t, err)

	svc := &OpsScheduledReportService{
		opsService:   opsService,
		emailService: emailService,
		cfg:          opsCfg,
	}
	report := &opsScheduledReport{
		Name:       "日报",
		ReportType: "daily_summary",
		Schedule:   "0 9 * * *",
		TimeRange:  24 * time.Hour,
		Recipients: []string{"owner@example.com"},
	}

	attempts, err := svc.runReport(ctx, report, time.Date(2026, time.November, 24, 1, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Equal(t, 1, attempts)
	require.Equal(t, int64(1), smtpServer.messageCount())

	body := smtpServer.lastMessageBody(t)
	require.Contains(t, body, "北京时间")
	require.Contains(t, body, "号池账号健康概览")
	require.Contains(t, body, "gpt-6.2-sol")
	require.Contains(t, body, "主力 GPT 账号")
	require.Contains(t, body, "主力 GPT 分组")
	require.Contains(t, body, "限流/过载账号：主力 Claude 账号")
	require.NotContains(t, body, "(#88)")
	require.NotContains(t, body, "(#4)")
	require.Contains(t, body, "https://gabugu.com/admin/ops")

	require.Equal(t, 1, dmCalls)
	require.Equal(t, "ops_scheduled_report", dmReceived.EventType)
	require.Contains(t, dmReceived.Text, "可用率 99.86%")
	require.Contains(t, dmReceived.Text, "主力 GPT 账号")
	require.NotContains(t, dmReceived.Text, "(#88)")
}
