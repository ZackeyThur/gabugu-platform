//go:build unit

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

var _ OpsRepository = (*stubOpsRepo)(nil)

type stubOpsRepo struct {
	OpsRepository
	overview *OpsDashboardOverview
	err      error
}

func (s *stubOpsRepo) GetDashboardOverview(ctx context.Context, filter *OpsDashboardFilter) (*OpsDashboardOverview, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.overview != nil {
		return s.overview, nil
	}
	return &OpsDashboardOverview{}, nil
}

func TestComputeGroupAvailableRatio(t *testing.T) {
	t.Parallel()

	t.Run("正常情况: 10个账号, 8个可用 = 80%", func(t *testing.T) {
		t.Parallel()

		got := computeGroupAvailableRatio(&GroupAvailability{
			TotalAccounts:  10,
			AvailableCount: 8,
		})
		require.InDelta(t, 80.0, got, 0.0001)
	})

	t.Run("边界情况: TotalAccounts = 0 应返回 0", func(t *testing.T) {
		t.Parallel()

		got := computeGroupAvailableRatio(&GroupAvailability{
			TotalAccounts:  0,
			AvailableCount: 8,
		})
		require.Equal(t, 0.0, got)
	})

	t.Run("边界情况: AvailableCount = 0 应返回 0%", func(t *testing.T) {
		t.Parallel()

		got := computeGroupAvailableRatio(&GroupAvailability{
			TotalAccounts:  10,
			AvailableCount: 0,
		})
		require.Equal(t, 0.0, got)
	})
}

func TestCountAccountsByCondition(t *testing.T) {
	t.Parallel()

	t.Run("测试限流账号统计: acc.IsRateLimited", func(t *testing.T) {
		t.Parallel()

		accounts := map[int64]*AccountAvailability{
			1: {IsRateLimited: true},
			2: {IsRateLimited: false},
			3: {IsRateLimited: true},
		}

		got := countAccountsByCondition(accounts, func(acc *AccountAvailability) bool {
			return acc.IsRateLimited
		})
		require.Equal(t, int64(2), got)
	})

	t.Run("测试错误账号统计（排除临时不可调度）: acc.HasError && acc.TempUnschedulableUntil == nil", func(t *testing.T) {
		t.Parallel()

		until := time.Now().UTC().Add(5 * time.Minute)
		accounts := map[int64]*AccountAvailability{
			1: {HasError: true},
			2: {HasError: true, TempUnschedulableUntil: &until},
			3: {HasError: false},
		}

		got := countAccountsByCondition(accounts, func(acc *AccountAvailability) bool {
			return acc.HasError && acc.TempUnschedulableUntil == nil
		})
		require.Equal(t, int64(1), got)
	})

	t.Run("边界情况: 空 map 应返回 0", func(t *testing.T) {
		t.Parallel()

		got := countAccountsByCondition(map[int64]*AccountAvailability{}, func(acc *AccountAvailability) bool {
			return acc.IsRateLimited
		})
		require.Equal(t, int64(0), got)
	})
}

// TestComputeRuleMetric_AccountTempUnscheduledCount verifies the new
// account_temp_unscheduled_count metric counts accounts currently in the
// temp-unscheduled window and ignores those whose window has expired or
// were never temp-unscheduled.
func TestComputeRuleMetric_AccountTempUnscheduledCount(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	futureUntil := now.Add(5 * time.Minute)
	pastUntil := now.Add(-1 * time.Minute)

	availability := &OpsAccountAvailability{
		Accounts: map[int64]*AccountAvailability{
			// currently temp-unscheduled (window active)
			1: {TempUnschedulableUntil: &futureUntil},
			2: {TempUnschedulableUntil: &futureUntil},
			// temp-unsched window already expired → should NOT count
			3: {TempUnschedulableUntil: &pastUntil},
			// never temp-unscheduled
			4: {HasError: true},
			5: {IsRateLimited: true},
		},
	}

	opsService := &OpsService{
		getAccountAvailability: func(_ context.Context, _ string, _ *int64) (*OpsAccountAvailability, error) {
			return availability, nil
		},
	}
	svc := &OpsAlertEvaluatorService{
		opsService: opsService,
		opsRepo:    &stubOpsRepo{},
	}

	rule := &OpsAlertRule{MetricType: "account_temp_unscheduled_count"}
	val, ok := svc.computeRuleMetric(context.Background(), rule, nil,
		now.Add(-5*time.Minute), now, "", nil)

	require.True(t, ok)
	require.InDelta(t, 2.0, val, 0.0001, "only 2 accounts have an active temp-unsched window")
}

func TestComputeRuleMetricNewIndicators(t *testing.T) {
	t.Parallel()

	groupID := int64(101)
	platform := "openai"

	availability := &OpsAccountAvailability{
		Group: &GroupAvailability{
			GroupID:        groupID,
			TotalAccounts:  10,
			AvailableCount: 8,
		},
		Accounts: map[int64]*AccountAvailability{
			1: {IsRateLimited: true},
			2: {IsRateLimited: true},
			3: {HasError: true},
			4: {HasError: true, TempUnschedulableUntil: timePtr(time.Now().UTC().Add(2 * time.Minute))},
			5: {HasError: false, IsRateLimited: false},
		},
	}

	opsService := &OpsService{
		getAccountAvailability: func(_ context.Context, _ string, _ *int64) (*OpsAccountAvailability, error) {
			return availability, nil
		},
	}

	svc := &OpsAlertEvaluatorService{
		opsService: opsService,
		opsRepo:    &stubOpsRepo{overview: &OpsDashboardOverview{}},
	}

	start := time.Now().UTC().Add(-5 * time.Minute)
	end := time.Now().UTC()
	ctx := context.Background()

	tests := []struct {
		name       string
		metricType string
		groupID    *int64
		wantValue  float64
		wantOK     bool
	}{
		{
			name:       "group_available_accounts",
			metricType: "group_available_accounts",
			groupID:    &groupID,
			wantValue:  8,
			wantOK:     true,
		},
		{
			name:       "group_available_ratio",
			metricType: "group_available_ratio",
			groupID:    &groupID,
			wantValue:  80.0,
			wantOK:     true,
		},
		{
			name:       "account_rate_limited_count",
			metricType: "account_rate_limited_count",
			groupID:    nil,
			wantValue:  2,
			wantOK:     true,
		},
		{
			name:       "account_error_count",
			metricType: "account_error_count",
			groupID:    nil,
			wantValue:  1,
			wantOK:     true,
		},
		{
			name:       "group_available_accounts without group_id returns false",
			metricType: "group_available_accounts",
			groupID:    nil,
			wantValue:  0,
			wantOK:     false,
		},
		{
			name:       "group_available_ratio without group_id returns false",
			metricType: "group_available_ratio",
			groupID:    nil,
			wantValue:  0,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rule := &OpsAlertRule{
				MetricType: tt.metricType,
			}
			gotValue, gotOK := svc.computeRuleMetric(ctx, rule, nil, start, end, platform, tt.groupID)
			require.Equal(t, tt.wantOK, gotOK)
			if !tt.wantOK {
				return
			}
			require.InDelta(t, tt.wantValue, gotValue, 0.0001)
		})
	}
}

func TestShouldSuppressLowSampleBreach(t *testing.T) {
	t.Parallel()

	runtimeCfg := &OpsAlertRuntimeSettings{
		MinSampleRequests: 10,
		MinErrorCount:     3,
	}

	// 1 error out of 2 SLA requests (50% error rate) -> suppressed by min_sample_requests=10 & min_error_count=3.
	suppressed := shouldSuppressLowSampleBreach(
		&OpsAlertRule{MetricType: "error_rate"},
		opsAlertMetricSnapshot{
			Value: 50.0,
			Valid: true,
			Overview: &OpsDashboardOverview{
				RequestCountSLA: 2,
				ErrorCountSLA:   1,
			},
		},
		runtimeCfg,
	)
	require.True(t, suppressed)

	// 2 errors out of 20 requests -> enough samples, but error count (2) < min_error_count (3) -> suppressed.
	suppressedByErrors := shouldSuppressLowSampleBreach(
		&OpsAlertRule{MetricType: "success_rate"},
		opsAlertMetricSnapshot{
			Value: 90.0,
			Valid: true,
			Overview: &OpsDashboardOverview{
				RequestCountSLA: 20,
				ErrorCountSLA:   2,
			},
		},
		runtimeCfg,
	)
	require.True(t, suppressedByErrors)

	// 4 errors out of 12 requests -> NOT suppressed.
	notSuppressed := shouldSuppressLowSampleBreach(
		&OpsAlertRule{MetricType: "error_rate"},
		opsAlertMetricSnapshot{
			Value: 33.33,
			Valid: true,
			Overview: &OpsDashboardOverview{
				RequestCountSLA: 12,
				ErrorCountSLA:   4,
			},
		},
		runtimeCfg,
	)
	require.False(t, notSuppressed)

	// Per-rule filters override global runtime settings.
	overrideAllowed := shouldSuppressLowSampleBreach(
		&OpsAlertRule{
			MetricType: "error_rate",
			Filters: map[string]any{
				"min_requests": 5,
				"min_errors":   2,
			},
		},
		opsAlertMetricSnapshot{
			Value: 40.0,
			Valid: true,
			Overview: &OpsDashboardOverview{
				RequestCountSLA: 5,
				ErrorCountSLA:   2,
			},
		},
		runtimeCfg,
	)
	require.False(t, overrideAllowed)
}

func TestDeliverOpsAlertNotificationFiringAndRecoveryAndPrivateWebhook(t *testing.T) {
	ctx := context.Background()
	settingRepo := newNotificationEmailMemorySettingRepo()
	smtpServer := startNotificationEmailTestSMTPServer(t)
	require.NoError(t, settingRepo.SetMultiple(ctx, smtpServer.settings()))
	require.NoError(t, settingRepo.Set(ctx, SettingKeySiteName, "GABUGU"))

	var dmCount int
	var lastDM opsPrivateWebhookPayload
	dmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dmCount++
		require.NoError(t, json.NewDecoder(r.Body).Decode(&lastDM))
		w.WriteHeader(http.StatusOK)
	}))
	defer dmSrv.Close()

	emailService := NewEmailService(settingRepo, nil)
	notificationService := NewNotificationEmailService(settingRepo, emailService)
	notificationService.RememberRecipientLocale(ctx, 0, "owner@example.com", "zh-CN")
	opsService := &OpsService{
		settingRepo:  settingRepo,
		emailService: emailService,
	}
	_, err := opsService.UpdateEmailNotificationConfig(ctx, &OpsEmailNotificationConfigUpdateRequest{
		Alert: &OpsEmailAlertConfig{
			Enabled:               true,
			Recipients:            []string{"owner@example.com"},
			MinSeverity:           "warning",
			IncludeResolvedAlerts: true,
		},
		PrivateWebhook: &OpsPrivateWebhookConfig{
			Enabled: true,
			URL:     dmSrv.URL,
			Target:  "wxid_owner_dm",
		},
	})
	require.NoError(t, err)

	g4 := int64(4)
	a88 := int64(88)
	var emailSentUpdated bool
	opsRepo := &testAlertOpsRepo{
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
					Message:        "upstream gateway timeout",
				},
			},
		},
		onUpdateEmailSent: func(eventID int64, sent bool) {
			emailSentUpdated = sent
		},
	}

	evalSvc := NewOpsAlertEvaluatorService(opsService, opsRepo, emailService, nil, &config.Config{Timezone: "Asia/Shanghai"}, nil)

	rule := &OpsAlertRule{
		ID:            8,
		Name:          "错误率极高",
		Severity:      "P0",
		MetricType:    "error_rate",
		Operator:      ">",
		Threshold:     20,
		WindowMinutes: 5,
		NotifyEmail:   true,
	}
	valFire := 33.33
	firedAt := time.Date(2026, time.November, 24, 1, 55, 0, 0, time.UTC)
	event := &OpsAlertEvent{
		ID:          701,
		RuleID:      8,
		Severity:    "P0",
		Status:      OpsAlertStatusFiring,
		MetricValue: &valFire,
		FiredAt:     firedAt,
	}
	snapFire := opsAlertMetricSnapshot{
		Value: valFire,
		Valid: true,
		Overview: &OpsDashboardOverview{
			RequestCountTotal: 12,
			RequestCountSLA:   12,
			SuccessCount:      8,
			ErrorCountSLA:     4,
		},
	}

	okFire := evalSvc.deliverOpsAlertNotification(ctx, nil, rule, event, snapFire, firedAt.Add(-5*time.Minute), firedAt, "", nil, false)
	require.True(t, okFire)
	require.True(t, emailSentUpdated)
	require.Equal(t, int64(1), smtpServer.messageCount())
	fireBody := smtpServer.lastMessageBody(t)
	require.Contains(t, fireBody, "gpt-6.2-sol")
	require.Contains(t, fireBody, "上游服务响应超时")
	require.Contains(t, fireBody, "主力 GPT 账号")
	require.Contains(t, fireBody, "主力 GPT 分组")
	require.NotContains(t, fireBody, "(#88)")
	require.NotContains(t, fireBody, "(#4)")
	require.Equal(t, 1, dmCount)
	require.Equal(t, "ops_alert", lastDM.EventType)
	require.Contains(t, lastDM.Text, "主力 GPT 账号")
	require.NotContains(t, lastDM.Text, "(#88)")

	// Mark EmailSent = true, then recover and assert recovery email uses distinct delivery key & green banner.
	event.EmailSent = true
	resolvedAt := firedAt.Add(3 * time.Minute)
	valRec := 1.0
	resolvedEvent := *event
	resolvedEvent.Status = OpsAlertStatusResolved
	resolvedEvent.ResolvedAt = &resolvedAt
	resolvedEvent.MetricValue = &valRec

	okRec := evalSvc.deliverOpsAlertNotification(ctx, nil, rule, &resolvedEvent, opsAlertMetricSnapshot{Value: valRec, Valid: true}, resolvedAt.Add(-5*time.Minute), resolvedAt, "", nil, true)
	require.True(t, okRec)
	require.Equal(t, int64(2), smtpServer.messageCount())
	recBody := smtpServer.lastMessageBody(t)
	require.Contains(t, recBody, "恢复确认：")
	require.Contains(t, recBody, "已自动恢复")
	require.Equal(t, 2, dmCount)
	require.Equal(t, "ops_alert_resolved", lastDM.EventType)

	// Recovery email is skipped if initial firing email was never sent.
	unsentEvent := &OpsAlertEvent{
		ID:         702,
		RuleID:     8,
		Severity:   "P0",
		Status:     OpsAlertStatusResolved,
		FiredAt:    firedAt,
		ResolvedAt: &resolvedAt,
		EmailSent:  false,
	}
	okSkipped := evalSvc.deliverOpsAlertNotification(ctx, nil, rule, unsentEvent, opsAlertMetricSnapshot{Value: valRec, Valid: true}, resolvedAt.Add(-5*time.Minute), resolvedAt, "", nil, true)
	require.False(t, okSkipped)
	require.Equal(t, int64(2), smtpServer.messageCount())
}

func TestOpsServiceSendTestAlertEmailFallbackAndSuccess(t *testing.T) {
	ctx := context.Background()
	settingRepo := newNotificationEmailMemorySettingRepo()
	smtpServer := startNotificationEmailTestSMTPServer(t)
	require.NoError(t, settingRepo.SetMultiple(ctx, smtpServer.settings()))
	require.NoError(t, settingRepo.Set(ctx, SettingKeySiteName, "GABUGU"))

	emailService := NewEmailService(settingRepo, nil)
	notificationService := NewNotificationEmailService(settingRepo, emailService)
	notificationService.RememberRecipientLocale(ctx, 0, "owner@example.com", "zh-CN")
	opsService := &OpsService{
		settingRepo:  settingRepo,
		emailService: emailService,
	}

	// Fails cleanly without recipient.
	_, err := opsService.SendTestAlertEmail(ctx, OpsTestAlertEmailInput{})
	require.Error(t, err)

	_, err = opsService.UpdateEmailNotificationConfig(ctx, &OpsEmailNotificationConfigUpdateRequest{
		Alert: &OpsEmailAlertConfig{
			Enabled:    true,
			Recipients: []string{"admin@example.com"},
		},
	})
	require.NoError(t, err)

	res, err := opsService.SendTestAlertEmail(ctx, OpsTestAlertEmailInput{
		Locale:             "zh",
		SimulateAIFallback: true,
	})
	require.NoError(t, err)
	require.True(t, res.Sent)
	require.False(t, res.UsedAI)
	require.Equal(t, "admin@example.com", res.Recipient)
	require.Contains(t, res.DiagnosisBadge, "兜底")
	require.Contains(t, res.DiagnosisSummary, "主力 GPT 账号")
	require.NotContains(t, res.DiagnosisSummary, "#88")
	require.Equal(t, int64(1), smtpServer.messageCount())

	// When real backend accounts exist, test alert automatically displays their backend names.
	opsService.getAccountAvailability = func(_ context.Context, _ string, _ *int64) (*OpsAccountAvailability, error) {
		return &OpsAccountAvailability{
			Accounts: map[int64]*AccountAvailability{
				101: {AccountID: 101, AccountName: "测试后台主账号", GroupID: 4, GroupName: "测试后台分组", Platform: "openai", HasError: true},
				102: {AccountID: 102, AccountName: "测试后台备用号", GroupID: 5, GroupName: "测试 Claude 组", Platform: "anthropic", IsAvailable: true},
			},
		}, nil
	}
	resLive, err := opsService.SendTestAlertEmail(ctx, OpsTestAlertEmailInput{
		Locale:             "zh",
		SimulateAIFallback: true,
	})
	require.NoError(t, err)
	require.True(t, resLive.Sent)
	require.Contains(t, resLive.Subject, "测试后台主账号")
	require.Contains(t, resLive.DiagnosisSummary, "测试后台主账号")
	require.NotContains(t, resLive.DiagnosisSummary, "#101")
	liveBody := smtpServer.lastMessageBody(t)
	require.Contains(t, liveBody, "测试后台主账号")
	require.Contains(t, liveBody, "测试后台分组")
	require.Contains(t, liveBody, "异常账号：测试后台主账号")
	require.NotContains(t, liveBody, "(#101)")
}

type testAlertOpsRepo struct {
	opsRepoMock
	errorLogs         *OpsErrorLogList
	rules             []*OpsAlertRule
	activeEvent       *OpsAlertEvent
	overview          *OpsDashboardOverview
	resolvedCount     int
	onUpdateEmailSent func(eventID int64, sent bool)
}

func (m *testAlertOpsRepo) ListAlertRules(_ context.Context) ([]*OpsAlertRule, error) {
	return m.rules, nil
}

func (m *testAlertOpsRepo) GetActiveAlertEvent(_ context.Context, _ int64) (*OpsAlertEvent, error) {
	return m.activeEvent, nil
}

func (m *testAlertOpsRepo) GetDashboardOverview(_ context.Context, _ *OpsDashboardFilter) (*OpsDashboardOverview, error) {
	return m.overview, nil
}

func (m *testAlertOpsRepo) UpdateAlertEventStatus(_ context.Context, _ int64, status string, _ *time.Time) error {
	if status == OpsAlertStatusResolved {
		m.resolvedCount++
		m.activeEvent = nil
	}
	return nil
}

func (m *testAlertOpsRepo) ListErrorLogs(_ context.Context, _ *OpsErrorLogFilter) (*OpsErrorLogList, error) {
	return m.errorLogs, nil
}

func (m *testAlertOpsRepo) UpdateAlertEventEmailSent(_ context.Context, eventID int64, emailSent bool) error {
	if m.onUpdateEmailSent != nil {
		m.onUpdateEmailSent(eventID, emailSent)
	}
	return nil
}

func TestEvaluateOnceLowSampleDoesNotAutoResolveActiveOutage(t *testing.T) {
	ctx := context.Background()
	settingRepo := newNotificationEmailMemorySettingRepo()
	opsService := &OpsService{
		settingRepo: settingRepo,
		cfg:         &config.Config{Ops: config.OpsConfig{Enabled: true}},
	}
	_, err := opsService.UpdateOpsAlertRuntimeSettings(ctx, &OpsAlertRuntimeSettings{
		EvaluationIntervalSeconds: 60,
		MinSampleRequests:         10,
		MinErrorCount:             3,
	})
	require.NoError(t, err)

	activeEv := &OpsAlertEvent{
		ID:       901,
		RuleID:   8,
		Severity: "P0",
		Status:   OpsAlertStatusFiring,
		FiredAt:  time.Now().UTC().Add(-5 * time.Minute),
	}
	repo := &testAlertOpsRepo{
		rules: []*OpsAlertRule{
			{
				ID:               8,
				Name:             "错误率极高",
				Enabled:          true,
				Severity:         "P0",
				MetricType:       "error_rate",
				Operator:         ">",
				Threshold:        20,
				WindowMinutes:    5,
				SustainedMinutes: 1,
				NotifyEmail:      false,
			},
		},
		activeEvent: activeEv,
		// Traffic dipped to 5 requests (< min_sample_requests=10), but ALL 5 failed (100% error rate)!
		overview: &OpsDashboardOverview{
			RequestCountTotal: 5,
			RequestCountSLA:   5,
			SuccessCount:      0,
			ErrorCountSLA:     5,
			ErrorRate:         1.0,
		},
	}

	evalSvc := NewOpsAlertEvaluatorService(opsService, repo, nil, nil, &config.Config{Ops: config.OpsConfig{Enabled: true}}, nil)
	evalSvc.evaluateOnce(time.Minute)
	require.Equal(t, 0, repo.resolvedCount, "active outage must NOT auto-resolve when raw metric is still breaching")
	require.NotNil(t, repo.activeEvent)

	// Once raw error rate genuinely drops below threshold (0%), active outage resolves cleanly.
	repo.overview = &OpsDashboardOverview{
		RequestCountTotal: 5,
		RequestCountSLA:   5,
		SuccessCount:      5,
		ErrorCountSLA:     0,
		ErrorRate:         0.0,
	}
	evalSvc.evaluateOnce(time.Minute)
	require.Equal(t, 1, repo.resolvedCount, "active outage must resolve once raw metric actually recovers")
	require.Nil(t, repo.activeEvent)
}

