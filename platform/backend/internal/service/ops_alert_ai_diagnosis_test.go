package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubSecretEncryptor struct{}

func (stubSecretEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (stubSecretEncryptor) Decrypt(ciphertext string) (string, error) {
	if !strings.HasPrefix(ciphertext, "enc:") {
		return ciphertext, nil
	}
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}

func TestDiagnoseOpsAlertAndFallback(t *testing.T) {
	t.Parallel()

	metricVal := 25.0
	thrVal := 10.0
	ev := &opsAlertEvidence{
		Locale:         "zh",
		RuleName:       "请求错误率过高",
		Severity:       "P1",
		Status:         OpsAlertStatusFiring,
		MetricType:     "error_rate",
		Operator:       ">",
		MetricValue:    &metricVal,
		ThresholdValue: &thrVal,
		MetricSummary:  formatOpsMetricSummary("error_rate", ">", &metricVal, &thrVal, "zh"),
		WindowLabel:    "最近 5 分钟",
		SLARequests:    20,
		SuccessCount:   15,
		ErrorCount:     5,
		ShowRequestOverview: true,
		FallbackSummary: "兜底中文结论：最近 5 分钟 20 次请求失败 5 次。",
	}

	t.Run("happy path returns sanitized AI summary", func(t *testing.T) {
		t.Parallel()
		var gotAuth string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			require.Equal(t, "/v1/chat/completions", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"**核心原因**：账号「主力 GPT 账号」上游响应超时（524），导致错误率升至 25%。建议先查看该账号代理延迟。"}}]}`))
		}))
		defer srv.Close()

		summary, usedAI, reason := DiagnoseOpsAlert(context.Background(), OpsAIDiagnosisConfig{
			Enabled:        true,
			BaseURL:        srv.URL + "/v1",
			Model:          "gpt-4o-mini",
			TimeoutSeconds: 5,
		}, "test-ai-key", ev, "zh")

		require.True(t, usedAI)
		require.Empty(t, reason)
		require.Equal(t, "Bearer test-ai-key", gotAuth)
		require.Contains(t, summary, "核心原因：账号「主力 GPT 账号」")
		require.NotContains(t, summary, "**")
	})

	t.Run("HTTP 500 falls back cleanly", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "upstream down", http.StatusInternalServerError)
		}))
		defer srv.Close()

		summary, usedAI, reason := DiagnoseOpsAlert(context.Background(), OpsAIDiagnosisConfig{
			Enabled:        true,
			BaseURL:        srv.URL + "/v1",
			Model:          "gpt-4o-mini",
			TimeoutSeconds: 3,
		}, "test-ai-key", ev, "zh")

		require.False(t, usedAI)
		require.Equal(t, "http_status_500", reason)
		require.Equal(t, ev.FallbackSummary, summary)
	})

	t.Run("timeout falls back cleanly", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(150 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
		defer cancel()

		summary, usedAI, reason := DiagnoseOpsAlert(ctx, OpsAIDiagnosisConfig{
			Enabled:        true,
			BaseURL:        srv.URL + "/v1",
			Model:          "gpt-4o-mini",
			TimeoutSeconds: 3,
		}, "test-ai-key", ev, "zh")

		require.False(t, usedAI)
		require.Equal(t, "timeout", reason)
		require.Equal(t, ev.FallbackSummary, summary)
	})

	t.Run("malformed JSON falls back cleanly", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not json`))
		}))
		defer srv.Close()

		summary, usedAI, reason := DiagnoseOpsAlert(context.Background(), OpsAIDiagnosisConfig{
			Enabled:        true,
			BaseURL:        srv.URL + "/v1",
			Model:          "gpt-4o-mini",
			TimeoutSeconds: 3,
		}, "test-ai-key", ev, "zh")

		require.False(t, usedAI)
		require.Equal(t, "decode_error", reason)
		require.Equal(t, ev.FallbackSummary, summary)
	})

	t.Run("empty choices falls back cleanly", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"choices":[]}`))
		}))
		defer srv.Close()

		summary, usedAI, reason := DiagnoseOpsAlert(context.Background(), OpsAIDiagnosisConfig{
			Enabled:        true,
			BaseURL:        srv.URL + "/v1",
			Model:          "gpt-4o-mini",
			TimeoutSeconds: 3,
		}, "test-ai-key", ev, "zh")

		require.False(t, usedAI)
		require.Equal(t, "empty_choices", reason)
		require.Equal(t, ev.FallbackSummary, summary)
	})

	t.Run("unconfigured falls back cleanly", func(t *testing.T) {
		t.Parallel()
		summary, usedAI, reason := DiagnoseOpsAlert(context.Background(), OpsAIDiagnosisConfig{
			Enabled: false,
		}, "", ev, "zh")

		require.False(t, usedAI)
		require.Equal(t, "not_configured", reason)
		require.Equal(t, ev.FallbackSummary, summary)
	})

	t.Run("primary failure automatically switches to backup AI route", func(t *testing.T) {
		t.Parallel()
		primarySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "primary unavailable", http.StatusBadGateway)
		}))
		defer primarySrv.Close()

		var gotBackupAuth string
		var gotBackupPath string
		backupSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotBackupAuth = r.Header.Get("Authorization")
			gotBackupPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"备用线路诊断：账号「主力 GPT 账号」出现 524 上游超时，建议临时切流。"}}]}`))
		}))
		defer backupSrv.Close()

		summary, route, reason := DiagnoseOpsAlertWithRoutes(context.Background(), OpsAIDiagnosisConfig{
			Enabled:         true,
			BaseURL:         primarySrv.URL + "/v1",
			Model:           "primary-model",
			FallbackBaseURL: backupSrv.URL,
			FallbackModel:   "backup-model",
			TimeoutSeconds:  5,
		}, "primary-key", "backup-key", ev, "zh")

		require.Equal(t, opsDiagnosisRouteBackup, route)
		require.Equal(t, "http_status_502", reason)
		require.Equal(t, "Bearer backup-key", gotBackupAuth)
		require.Equal(t, "/v1/chat/completions", gotBackupPath)
		require.Contains(t, summary, "备用线路诊断")
		require.Equal(t, "AI 智能诊断（备用线路）", opsDiagnosisBadgeForRoute(route, true, "zh"))
	})

	t.Run("both primary and backup failures fall back to built-in rules", func(t *testing.T) {
		t.Parallel()
		primarySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "primary down", http.StatusInternalServerError)
		}))
		defer primarySrv.Close()
		backupSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "backup down", http.StatusServiceUnavailable)
		}))
		defer backupSrv.Close()

		summary, route, reason := DiagnoseOpsAlertWithRoutes(context.Background(), OpsAIDiagnosisConfig{
			Enabled:         true,
			BaseURL:         primarySrv.URL + "/v1",
			Model:           "primary-model",
			FallbackBaseURL: backupSrv.URL + "/v1",
			FallbackModel:   "backup-model",
			TimeoutSeconds:  5,
		}, "primary-key", "backup-key", ev, "zh")

		require.Empty(t, route)
		require.Equal(t, "primary:http_status_500;backup:http_status_503", reason)
		require.Equal(t, ev.FallbackSummary, summary)
	})
}

func TestOpsPrivateWebhookDMOnlyEnforcement(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	var receivedAuth string
	var receivedPayload opsPrivateWebhookPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		receivedAuth = r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&receivedPayload))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Group targets MUST be strictly blocked before hitting HTTP client.
	for _, target := range []string{"123456@chatroom", "  room_999@CHATROOM  "} {
		err := sendOpsPrivateWebhook(context.Background(), OpsPrivateWebhookConfig{
			Enabled: true,
			URL:     srv.URL,
			Target:  target,
		}, "bot-tok", opsPrivateWebhookPayload{
			EventType: "ops_alert",
			Text:      "test message",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "@chatroom")
	}
	require.Equal(t, int64(0), calls.Load())

	// Private DM target succeeds with bearer token.
	err := sendOpsPrivateWebhook(context.Background(), OpsPrivateWebhookConfig{
		Enabled: true,
		URL:     srv.URL,
		Target:  "wxid_owner_001",
	}, "bot-tok", opsPrivateWebhookPayload{
		EventType: "ops_alert",
		Title:     "[GABUGU 紧急告警] 测试",
		Severity:  "P0",
		Status:    "firing",
		Text:      "这是发给老板私聊的告警摘要",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), calls.Load())
	require.Equal(t, "Bearer bot-tok", receivedAuth)
	require.Equal(t, "wxid_owner_001", receivedPayload.Target)
}

func TestOpsEmailNotificationConfigSecretsAndChatroomBlock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newNotificationEmailMemorySettingRepo()
	svc := &OpsService{
		settingRepo: repo,
		encryptor:   stubSecretEncryptor{},
	}

	// Reject @chatroom even if webhook is disabled when user enters it.
	_, err := svc.UpdateEmailNotificationConfig(ctx, &OpsEmailNotificationConfigUpdateRequest{
		PrivateWebhook: &OpsPrivateWebhookConfig{
			Enabled: false,
			URL:     "https://bot.example.com/send",
			Target:  "987654@chatroom",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "@chatroom")

	// Valid save encrypts secrets in separate settings keys and never exposes plaintext.
	updated, err := svc.UpdateEmailNotificationConfig(ctx, &OpsEmailNotificationConfigUpdateRequest{
		Alert: &OpsEmailAlertConfig{
			Enabled:               true,
			Recipients:            []string{"owner@example.com"},
			MinSeverity:           "critical",
			IncludeResolvedAlerts: true,
		},
		AIDiagnosis: &OpsAIDiagnosisConfig{
			Enabled:         true,
			BaseURL:         "https://api.example.com/v1/",
			Model:           "gpt-4o-mini",
			APIKey:          "sk-super-secret-value-1234",
			FallbackBaseURL: "https://backup.example.com/v1/",
			FallbackModel:   "deepseek-v4.1-flash",
			FallbackAPIKey:  "sk-backup-secret-value-5678",
			TimeoutSeconds:  10,
		},
		PrivateWebhook: &OpsPrivateWebhookConfig{
			Enabled:        true,
			URL:            "https://bot.example.com/dm",
			Target:         "wxid_owner_only",
			BearerToken:    "tok-very-secret-9999",
			IncludeReports: true,
		},
	})
	require.NoError(t, err)
	require.True(t, updated.AIDiagnosis.APIKeyConfigured)
	require.Equal(t, "sk-s***1234", updated.AIDiagnosis.APIKeyMasked)
	require.Empty(t, updated.AIDiagnosis.APIKey)
	require.True(t, updated.AIDiagnosis.FallbackAPIKeyConfigured)
	require.Equal(t, "sk-b***5678", updated.AIDiagnosis.FallbackAPIKeyMasked)
	require.Empty(t, updated.AIDiagnosis.FallbackAPIKey)
	require.Equal(t, "https://backup.example.com/v1", updated.AIDiagnosis.FallbackBaseURL)
	require.True(t, updated.PrivateWebhook.TokenConfigured)
	require.Equal(t, "tok-***9999", updated.PrivateWebhook.TokenMasked)
	require.Empty(t, updated.PrivateWebhook.BearerToken)

	// Verify persisted JSON does NOT contain plaintext secrets.
	rawJSON, err := repo.GetValue(ctx, SettingKeyOpsEmailNotificationConfig)
	require.NoError(t, err)
	require.NotContains(t, rawJSON, "sk-super-secret-value-1234")
	require.NotContains(t, rawJSON, "sk-backup-secret-value-5678")
	require.NotContains(t, rawJSON, "tok-very-secret-9999")

	// Verify stored secrets are encrypted and decrypted on read.
	encAIKey, err := repo.GetValue(ctx, SettingKeyOpsAIDiagnosisAPIKey)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(encAIKey, "enc:"))
	plainAIKey, err := svc.GetOpsAIDiagnosisAPIKey(ctx)
	require.NoError(t, err)
	require.Equal(t, "sk-super-secret-value-1234", plainAIKey)

	encBackupKey, err := repo.GetValue(ctx, SettingKeyOpsAIDiagnosisFallbackAPIKey)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(encBackupKey, "enc:"))
	plainBackupKey, err := svc.GetOpsAIDiagnosisFallbackAPIKey(ctx)
	require.NoError(t, err)
	require.Equal(t, "sk-backup-secret-value-5678", plainBackupKey)

	encTok, err := repo.GetValue(ctx, SettingKeyOpsPrivateWebhookToken)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(encTok, "enc:"))
	plainTok, err := svc.GetOpsPrivateWebhookToken(ctx)
	require.NoError(t, err)
	require.Equal(t, "tok-very-secret-9999", plainTok)

	// Clearing API keys removes stored secrets cleanly.
	_, err = svc.UpdateEmailNotificationConfig(ctx, &OpsEmailNotificationConfigUpdateRequest{
		AIDiagnosis: &OpsAIDiagnosisConfig{
			Enabled:             false,
			BaseURL:             "https://api.example.com/v1",
			Model:               "gpt-4o-mini",
			ClearAPIKey:         true,
			ClearFallbackAPIKey: true,
			TimeoutSeconds:      10,
		},
	})
	require.NoError(t, err)
	reloaded, err := svc.GetEmailNotificationConfig(ctx)
	require.NoError(t, err)
	require.False(t, reloaded.AIDiagnosis.APIKeyConfigured)
	require.Empty(t, reloaded.AIDiagnosis.APIKeyMasked)
	require.False(t, reloaded.AIDiagnosis.FallbackAPIKeyConfigured)
	require.Empty(t, reloaded.AIDiagnosis.FallbackAPIKeyMasked)
	_, err = repo.GetValue(ctx, SettingKeyOpsAIDiagnosisAPIKey)
	require.ErrorIs(t, err, ErrSettingNotFound)
	_, err = repo.GetValue(ctx, SettingKeyOpsAIDiagnosisFallbackAPIKey)
	require.ErrorIs(t, err, ErrSettingNotFound)
}

func TestBoundedOpsAIDiagnosisContextPreservesDeliveryHeadroom(t *testing.T) {
	t.Parallel()

	// Insufficient headroom (< reserve + minUsable) skips AI immediately.
	tightParent, tightCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer tightCancel()
	_, c1, ok1 := boundedOpsAIDiagnosisContext(tightParent, 15*time.Second, 15*time.Second)
	c1()
	require.False(t, ok1)

	// Sufficient parent budget (45s) caps AI to <= 15s, leaving >= 30s for SMTP/DB delivery.
	roomyParent, roomyCancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer roomyCancel()
	aiCtx, c2, ok2 := boundedOpsAIDiagnosisContext(roomyParent, 15*time.Second, 15*time.Second)
	defer c2()
	require.True(t, ok2)
	dl, hasDeadline := aiCtx.Deadline()
	require.True(t, hasDeadline)
	require.LessOrEqual(t, time.Until(dl), 15*time.Second+200*time.Millisecond)
}
