package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	opsAlertEvaluatorLeaderLockKeyDefault = "ops:alert:evaluator:leader"
	opsAlertEvaluatorLeaderLockTTLDefault = 30 * time.Second
)

// =========================
// Email notification config
// =========================

func (s *OpsService) GetEmailNotificationConfig(ctx context.Context) (*OpsEmailNotificationConfig, error) {
	defaultCfg := defaultOpsEmailNotificationConfig()
	if s == nil || s.settingRepo == nil {
		return defaultCfg, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	raw, err := s.settingRepo.GetValue(ctx, SettingKeyOpsEmailNotificationConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			// Initialize defaults on first read (best-effort).
			if b, mErr := json.Marshal(defaultCfg); mErr == nil {
				_ = s.settingRepo.Set(ctx, SettingKeyOpsEmailNotificationConfig, string(b))
			}
			s.hydrateOpsNotificationSecrets(ctx, defaultCfg)
			return defaultCfg, nil
		}
		return nil, err
	}

	cfg := &OpsEmailNotificationConfig{}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		// Corrupted JSON should not break ops UI; fall back to defaults.
		s.hydrateOpsNotificationSecrets(ctx, defaultCfg)
		return defaultCfg, nil
	}
	normalizeOpsEmailNotificationConfig(cfg)
	s.hydrateOpsNotificationSecrets(ctx, cfg)
	return cfg, nil
}

func (s *OpsService) UpdateEmailNotificationConfig(ctx context.Context, req *OpsEmailNotificationConfigUpdateRequest) (*OpsEmailNotificationConfig, error) {
	if s == nil || s.settingRepo == nil {
		return nil, errors.New("setting repository not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil {
		return nil, errors.New("invalid request")
	}

	cfg, err := s.GetEmailNotificationConfig(ctx)
	if err != nil {
		return nil, err
	}

	if req.Alert != nil {
		cfg.Alert.Enabled = req.Alert.Enabled
		if req.Alert.Recipients != nil {
			cfg.Alert.Recipients = req.Alert.Recipients
		}
		cfg.Alert.MinSeverity = strings.TrimSpace(req.Alert.MinSeverity)
		cfg.Alert.RateLimitPerHour = req.Alert.RateLimitPerHour
		cfg.Alert.BatchingWindowSeconds = req.Alert.BatchingWindowSeconds
		cfg.Alert.IncludeResolvedAlerts = req.Alert.IncludeResolvedAlerts
	}

	if req.Report != nil {
		cfg.Report.Enabled = req.Report.Enabled
		if req.Report.Recipients != nil {
			cfg.Report.Recipients = req.Report.Recipients
		}
		cfg.Report.DailySummaryEnabled = req.Report.DailySummaryEnabled
		cfg.Report.DailySummarySchedule = strings.TrimSpace(req.Report.DailySummarySchedule)
		cfg.Report.WeeklySummaryEnabled = req.Report.WeeklySummaryEnabled
		cfg.Report.WeeklySummarySchedule = strings.TrimSpace(req.Report.WeeklySummarySchedule)
		cfg.Report.ErrorDigestEnabled = req.Report.ErrorDigestEnabled
		cfg.Report.ErrorDigestSchedule = strings.TrimSpace(req.Report.ErrorDigestSchedule)
		cfg.Report.ErrorDigestMinCount = req.Report.ErrorDigestMinCount
		cfg.Report.AccountHealthEnabled = req.Report.AccountHealthEnabled
		cfg.Report.AccountHealthSchedule = strings.TrimSpace(req.Report.AccountHealthSchedule)
		cfg.Report.AccountHealthErrorRateThreshold = req.Report.AccountHealthErrorRateThreshold
	}

	hasAIKey := cfg.AIDiagnosis.APIKeyConfigured
	var newAIKey string
	clearAIKey := false
	var newAIFallbackKey string
	clearAIFallbackKey := false
	if req.AIDiagnosis != nil {
		cfg.AIDiagnosis.Enabled = req.AIDiagnosis.Enabled
		cfg.AIDiagnosis.BaseURL = strings.TrimSpace(req.AIDiagnosis.BaseURL)
		cfg.AIDiagnosis.Model = strings.TrimSpace(req.AIDiagnosis.Model)
		cfg.AIDiagnosis.FallbackBaseURL = strings.TrimSpace(req.AIDiagnosis.FallbackBaseURL)
		cfg.AIDiagnosis.FallbackModel = strings.TrimSpace(req.AIDiagnosis.FallbackModel)
		cfg.AIDiagnosis.TimeoutSeconds = req.AIDiagnosis.TimeoutSeconds
		if req.AIDiagnosis.ClearAPIKey {
			clearAIKey = true
			hasAIKey = false
		} else if trimmedKey := strings.TrimSpace(req.AIDiagnosis.APIKey); trimmedKey != "" {
			newAIKey = trimmedKey
			hasAIKey = true
		}
		if req.AIDiagnosis.ClearFallbackAPIKey {
			clearAIFallbackKey = true
		} else if trimmedKey := strings.TrimSpace(req.AIDiagnosis.FallbackAPIKey); trimmedKey != "" {
			newAIFallbackKey = trimmedKey
		}
	}

	var newWebhookToken string
	clearWebhookToken := false
	if req.PrivateWebhook != nil {
		cfg.PrivateWebhook.Enabled = req.PrivateWebhook.Enabled
		cfg.PrivateWebhook.URL = strings.TrimSpace(req.PrivateWebhook.URL)
		cfg.PrivateWebhook.Target = strings.TrimSpace(req.PrivateWebhook.Target)
		cfg.PrivateWebhook.IncludeReports = req.PrivateWebhook.IncludeReports
		if req.PrivateWebhook.ClearToken {
			clearWebhookToken = true
		} else if trimmedTok := strings.TrimSpace(req.PrivateWebhook.BearerToken); trimmedTok != "" {
			newWebhookToken = trimmedTok
		}
	}

	if err := validateOpsEmailNotificationConfig(cfg, hasAIKey); err != nil {
		return nil, err
	}

	normalizeOpsEmailNotificationConfig(cfg)

	if clearAIKey {
		if err := s.deleteOpsSecret(ctx, SettingKeyOpsAIDiagnosisAPIKey); err != nil {
			return nil, err
		}
	} else if newAIKey != "" {
		if err := s.storeOpsSecret(ctx, SettingKeyOpsAIDiagnosisAPIKey, newAIKey); err != nil {
			return nil, err
		}
	}

	if clearAIFallbackKey {
		if err := s.deleteOpsSecret(ctx, SettingKeyOpsAIDiagnosisFallbackAPIKey); err != nil {
			return nil, err
		}
	} else if newAIFallbackKey != "" {
		if err := s.storeOpsSecret(ctx, SettingKeyOpsAIDiagnosisFallbackAPIKey, newAIFallbackKey); err != nil {
			return nil, err
		}
	}

	if clearWebhookToken {
		if err := s.deleteOpsSecret(ctx, SettingKeyOpsPrivateWebhookToken); err != nil {
			return nil, err
		}
	} else if newWebhookToken != "" {
		if err := s.storeOpsSecret(ctx, SettingKeyOpsPrivateWebhookToken, newWebhookToken); err != nil {
			return nil, err
		}
	}

	persistCfg := sanitizeOpsEmailNotificationConfigForPersist(cfg)
	raw, err := json.Marshal(persistCfg)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyOpsEmailNotificationConfig, string(raw)); err != nil {
		return nil, err
	}

	s.hydrateOpsNotificationSecrets(ctx, cfg)
	return cfg, nil
}

func defaultOpsEmailNotificationConfig() *OpsEmailNotificationConfig {
	return &OpsEmailNotificationConfig{
		Alert: OpsEmailAlertConfig{
			Enabled:               true,
			Recipients:            []string{},
			MinSeverity:           "",
			RateLimitPerHour:      0,
			BatchingWindowSeconds: 0,
			IncludeResolvedAlerts: false,
		},
		Report: OpsEmailReportConfig{
			Enabled:                         false,
			Recipients:                      []string{},
			DailySummaryEnabled:             false,
			DailySummarySchedule:            "0 9 * * *",
			WeeklySummaryEnabled:            false,
			WeeklySummarySchedule:           "0 9 * * 1",
			ErrorDigestEnabled:              false,
			ErrorDigestSchedule:             "0 9 * * *",
			ErrorDigestMinCount:             10,
			AccountHealthEnabled:            false,
			AccountHealthSchedule:           "0 9 * * *",
			AccountHealthErrorRateThreshold: 10.0,
		},
		AIDiagnosis: OpsAIDiagnosisConfig{
			Enabled:        false,
			BaseURL:        "",
			Model:          "",
			TimeoutSeconds: 8,
		},
		PrivateWebhook: OpsPrivateWebhookConfig{
			Enabled:        false,
			URL:            "",
			Target:         "",
			IncludeReports: false,
		},
	}
}

func normalizeOpsEmailNotificationConfig(cfg *OpsEmailNotificationConfig) {
	if cfg == nil {
		return
	}
	if cfg.Alert.Recipients == nil {
		cfg.Alert.Recipients = []string{}
	}
	if cfg.Report.Recipients == nil {
		cfg.Report.Recipients = []string{}
	}

	cfg.Alert.MinSeverity = strings.TrimSpace(cfg.Alert.MinSeverity)
	cfg.Report.DailySummarySchedule = strings.TrimSpace(cfg.Report.DailySummarySchedule)
	cfg.Report.WeeklySummarySchedule = strings.TrimSpace(cfg.Report.WeeklySummarySchedule)
	cfg.Report.ErrorDigestSchedule = strings.TrimSpace(cfg.Report.ErrorDigestSchedule)
	cfg.Report.AccountHealthSchedule = strings.TrimSpace(cfg.Report.AccountHealthSchedule)

	// Fill missing schedules with defaults to avoid breaking cron logic if clients send empty strings.
	if cfg.Report.DailySummarySchedule == "" {
		cfg.Report.DailySummarySchedule = "0 9 * * *"
	}
	if cfg.Report.WeeklySummarySchedule == "" {
		cfg.Report.WeeklySummarySchedule = "0 9 * * 1"
	}
	if cfg.Report.ErrorDigestSchedule == "" {
		cfg.Report.ErrorDigestSchedule = "0 9 * * *"
	}
	if cfg.Report.AccountHealthSchedule == "" {
		cfg.Report.AccountHealthSchedule = "0 9 * * *"
	}

	cfg.AIDiagnosis.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.AIDiagnosis.BaseURL), "/")
	cfg.AIDiagnosis.Model = strings.TrimSpace(cfg.AIDiagnosis.Model)
	cfg.AIDiagnosis.FallbackBaseURL = strings.TrimRight(strings.TrimSpace(cfg.AIDiagnosis.FallbackBaseURL), "/")
	cfg.AIDiagnosis.FallbackModel = strings.TrimSpace(cfg.AIDiagnosis.FallbackModel)
	if cfg.AIDiagnosis.TimeoutSeconds <= 0 {
		cfg.AIDiagnosis.TimeoutSeconds = 8
	}
	cfg.AIDiagnosis.APIKey = ""
	cfg.AIDiagnosis.ClearAPIKey = false
	cfg.AIDiagnosis.FallbackAPIKey = ""
	cfg.AIDiagnosis.ClearFallbackAPIKey = false

	cfg.PrivateWebhook.URL = strings.TrimSpace(cfg.PrivateWebhook.URL)
	cfg.PrivateWebhook.Target = strings.TrimSpace(cfg.PrivateWebhook.Target)
	cfg.PrivateWebhook.BearerToken = ""
	cfg.PrivateWebhook.ClearToken = false
}

func sanitizeOpsEmailNotificationConfigForPersist(cfg *OpsEmailNotificationConfig) *OpsEmailNotificationConfig {
	if cfg == nil {
		return nil
	}
	copyCfg := *cfg
	copyCfg.AIDiagnosis.APIKey = ""
	copyCfg.AIDiagnosis.ClearAPIKey = false
	copyCfg.AIDiagnosis.APIKeyConfigured = false
	copyCfg.AIDiagnosis.APIKeyMasked = ""
	copyCfg.AIDiagnosis.FallbackAPIKey = ""
	copyCfg.AIDiagnosis.ClearFallbackAPIKey = false
	copyCfg.AIDiagnosis.FallbackAPIKeyConfigured = false
	copyCfg.AIDiagnosis.FallbackAPIKeyMasked = ""
	copyCfg.PrivateWebhook.BearerToken = ""
	copyCfg.PrivateWebhook.ClearToken = false
	copyCfg.PrivateWebhook.TokenConfigured = false
	copyCfg.PrivateWebhook.TokenMasked = ""
	return &copyCfg
}

func isValidOpsHTTPURL(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}
	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return (scheme == "http" || scheme == "https") && strings.TrimSpace(parsed.Host) != ""
}

func isDisallowedOpsGroupTarget(target string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(target)), "@chatroom")
}

func validateOpsEmailNotificationConfig(cfg *OpsEmailNotificationConfig, hasAIKey bool) error {
	if cfg == nil {
		return errors.New("invalid config")
	}

	if cfg.Alert.RateLimitPerHour < 0 {
		return errors.New("alert.rate_limit_per_hour must be >= 0")
	}
	if cfg.Alert.BatchingWindowSeconds < 0 {
		return errors.New("alert.batching_window_seconds must be >= 0")
	}
	switch strings.TrimSpace(cfg.Alert.MinSeverity) {
	case "", "critical", "warning", "info":
	default:
		return errors.New("alert.min_severity must be one of: critical, warning, info, or empty")
	}

	if cfg.Report.ErrorDigestMinCount < 0 {
		return errors.New("report.error_digest_min_count must be >= 0")
	}
	if cfg.Report.AccountHealthErrorRateThreshold < 0 || cfg.Report.AccountHealthErrorRateThreshold > 100 {
		return errors.New("report.account_health_error_rate_threshold must be between 0 and 100")
	}

	if cfg.AIDiagnosis.TimeoutSeconds < 0 || (cfg.AIDiagnosis.TimeoutSeconds > 0 && (cfg.AIDiagnosis.TimeoutSeconds < 3 || cfg.AIDiagnosis.TimeoutSeconds > 30)) {
		return errors.New("ai_diagnosis.timeout_seconds must be between 3 and 30")
	}
	if cfg.AIDiagnosis.Enabled {
		if !isValidOpsHTTPURL(cfg.AIDiagnosis.BaseURL) {
			return errors.New("ai_diagnosis.base_url must be a valid http/https URL when enabled")
		}
		if strings.TrimSpace(cfg.AIDiagnosis.Model) == "" {
			return errors.New("ai_diagnosis.model is required when enabled")
		}
		if !hasAIKey {
			return errors.New("ai_diagnosis.api_key is required when enabled")
		}
	}
	fallbackURL := strings.TrimSpace(cfg.AIDiagnosis.FallbackBaseURL)
	fallbackModel := strings.TrimSpace(cfg.AIDiagnosis.FallbackModel)
	if fallbackURL != "" {
		if !isValidOpsHTTPURL(fallbackURL) {
			return errors.New("ai_diagnosis.fallback_base_url must be a valid http/https URL")
		}
		if fallbackModel == "" {
			return errors.New("ai_diagnosis.fallback_model is required when fallback_base_url is set")
		}
	} else if fallbackModel != "" {
		return errors.New("ai_diagnosis.fallback_base_url is required when fallback_model is set")
	}

	if isDisallowedOpsGroupTarget(cfg.PrivateWebhook.Target) {
		return errors.New("private_webhook.target cannot target a group chat (@chatroom)")
	}
	if cfg.PrivateWebhook.Enabled {
		if !isValidOpsHTTPURL(cfg.PrivateWebhook.URL) {
			return errors.New("private_webhook.url must be a valid http/https URL when enabled")
		}
		if strings.TrimSpace(cfg.PrivateWebhook.Target) == "" {
			return errors.New("private_webhook.target is required when enabled")
		}
	}

	return nil
}

func (s *OpsService) storeOpsSecret(ctx context.Context, key, plaintext string) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting repository not initialized")
	}
	trimmed := strings.TrimSpace(plaintext)
	if trimmed == "" {
		return nil
	}
	toStore := trimmed
	if s.encryptor != nil {
		encrypted, err := s.encryptor.Encrypt(trimmed)
		if err != nil {
			return fmt.Errorf("encrypt %s: %w", key, err)
		}
		toStore = encrypted
	}
	return s.settingRepo.Set(ctx, key, toStore)
}

func (s *OpsService) deleteOpsSecret(ctx context.Context, key string) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting repository not initialized")
	}
	if err := s.settingRepo.Delete(ctx, key); err != nil && !errors.Is(err, ErrSettingNotFound) {
		return err
	}
	return nil
}

func (s *OpsService) loadOpsSecret(ctx context.Context, key string) (string, bool, error) {
	if s == nil || s.settingRepo == nil {
		return "", false, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, key)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false, nil
	}
	if s.encryptor != nil {
		decrypted, decErr := s.encryptor.Decrypt(trimmed)
		if decErr != nil {
			return "", true, decErr
		}
		return strings.TrimSpace(decrypted), true, nil
	}
	return trimmed, true, nil
}

func (s *OpsService) hydrateOpsNotificationSecrets(ctx context.Context, cfg *OpsEmailNotificationConfig) {
	if cfg == nil {
		return
	}
	cfg.AIDiagnosis.APIKey = ""
	cfg.AIDiagnosis.ClearAPIKey = false
	if keyVal, exists, err := s.loadOpsSecret(ctx, SettingKeyOpsAIDiagnosisAPIKey); exists {
		cfg.AIDiagnosis.APIKeyConfigured = true
		if err == nil && keyVal != "" {
			cfg.AIDiagnosis.APIKeyMasked = maskOpsSecret(keyVal)
		} else {
			cfg.AIDiagnosis.APIKeyMasked = "****"
		}
	} else {
		cfg.AIDiagnosis.APIKeyConfigured = false
		cfg.AIDiagnosis.APIKeyMasked = ""
	}

	cfg.AIDiagnosis.FallbackAPIKey = ""
	cfg.AIDiagnosis.ClearFallbackAPIKey = false
	if keyVal, exists, err := s.loadOpsSecret(ctx, SettingKeyOpsAIDiagnosisFallbackAPIKey); exists {
		cfg.AIDiagnosis.FallbackAPIKeyConfigured = true
		if err == nil && keyVal != "" {
			cfg.AIDiagnosis.FallbackAPIKeyMasked = maskOpsSecret(keyVal)
		} else {
			cfg.AIDiagnosis.FallbackAPIKeyMasked = "****"
		}
	} else {
		cfg.AIDiagnosis.FallbackAPIKeyConfigured = false
		cfg.AIDiagnosis.FallbackAPIKeyMasked = ""
	}

	cfg.PrivateWebhook.BearerToken = ""
	cfg.PrivateWebhook.ClearToken = false
	if tokVal, exists, err := s.loadOpsSecret(ctx, SettingKeyOpsPrivateWebhookToken); exists {
		cfg.PrivateWebhook.TokenConfigured = true
		if err == nil && tokVal != "" {
			cfg.PrivateWebhook.TokenMasked = maskOpsSecret(tokVal)
		} else {
			cfg.PrivateWebhook.TokenMasked = "****"
		}
	} else {
		cfg.PrivateWebhook.TokenConfigured = false
		cfg.PrivateWebhook.TokenMasked = ""
	}
}

func maskOpsSecret(secret string) string {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= 6 {
		return "****"
	}
	if len(runes) <= 10 {
		return string(runes[:2]) + "***" + string(runes[len(runes)-2:])
	}
	return string(runes[:4]) + "***" + string(runes[len(runes)-4:])
}

// GetOpsAIDiagnosisAPIKey returns the decrypted API key for ops AI diagnosis.
func (s *OpsService) GetOpsAIDiagnosisAPIKey(ctx context.Context) (string, error) {
	if s == nil {
		return "", nil
	}
	val, _, err := s.loadOpsSecret(ctx, SettingKeyOpsAIDiagnosisAPIKey)
	return val, err
}

// GetOpsAIDiagnosisFallbackAPIKey returns the decrypted API key for ops AI backup diagnosis route.
func (s *OpsService) GetOpsAIDiagnosisFallbackAPIKey(ctx context.Context) (string, error) {
	if s == nil {
		return "", nil
	}
	val, _, err := s.loadOpsSecret(ctx, SettingKeyOpsAIDiagnosisFallbackAPIKey)
	return val, err
}

// GetOpsPrivateWebhookToken returns the decrypted bearer token for ops private DM webhook.
func (s *OpsService) GetOpsPrivateWebhookToken(ctx context.Context) (string, error) {
	if s == nil {
		return "", nil
	}
	val, _, err := s.loadOpsSecret(ctx, SettingKeyOpsPrivateWebhookToken)
	return val, err
}

// =========================
// Alert runtime settings
// =========================

func defaultOpsAlertRuntimeSettings() *OpsAlertRuntimeSettings {
	return &OpsAlertRuntimeSettings{
		EvaluationIntervalSeconds: 60,
		DistributedLock: OpsDistributedLockSettings{
			Enabled:    true,
			Key:        opsAlertEvaluatorLeaderLockKeyDefault,
			TTLSeconds: int(opsAlertEvaluatorLeaderLockTTLDefault.Seconds()),
		},
		Silencing: OpsAlertSilencingSettings{
			Enabled:            false,
			GlobalUntilRFC3339: "",
			GlobalReason:       "",
			Entries:            []OpsAlertSilenceEntry{},
		},
	}
}

func normalizeOpsDistributedLockSettings(s *OpsDistributedLockSettings, defaultKey string, defaultTTLSeconds int) {
	if s == nil {
		return
	}
	s.Key = strings.TrimSpace(s.Key)
	if s.Key == "" {
		s.Key = defaultKey
	}
	if s.TTLSeconds <= 0 {
		s.TTLSeconds = defaultTTLSeconds
	}
}

func normalizeOpsAlertSilencingSettings(s *OpsAlertSilencingSettings) {
	if s == nil {
		return
	}
	s.GlobalUntilRFC3339 = strings.TrimSpace(s.GlobalUntilRFC3339)
	s.GlobalReason = strings.TrimSpace(s.GlobalReason)
	if s.Entries == nil {
		s.Entries = []OpsAlertSilenceEntry{}
	}
	for i := range s.Entries {
		s.Entries[i].UntilRFC3339 = strings.TrimSpace(s.Entries[i].UntilRFC3339)
		s.Entries[i].Reason = strings.TrimSpace(s.Entries[i].Reason)
	}
}

func validateOpsDistributedLockSettings(s OpsDistributedLockSettings) error {
	if strings.TrimSpace(s.Key) == "" {
		return errors.New("distributed_lock.key is required")
	}
	if s.TTLSeconds <= 0 || s.TTLSeconds > int((24*time.Hour).Seconds()) {
		return errors.New("distributed_lock.ttl_seconds must be between 1 and 86400")
	}
	return nil
}

func validateOpsAlertSilencingSettings(s OpsAlertSilencingSettings) error {
	parse := func(raw string) error {
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		if _, err := time.Parse(time.RFC3339, raw); err != nil {
			return errors.New("silencing time must be RFC3339")
		}
		return nil
	}

	if err := parse(s.GlobalUntilRFC3339); err != nil {
		return err
	}
	for _, entry := range s.Entries {
		if strings.TrimSpace(entry.UntilRFC3339) == "" {
			return errors.New("silencing.entries.until_rfc3339 is required")
		}
		if _, err := time.Parse(time.RFC3339, entry.UntilRFC3339); err != nil {
			return errors.New("silencing.entries.until_rfc3339 must be RFC3339")
		}
	}
	return nil
}

func (s *OpsService) GetOpsAlertRuntimeSettings(ctx context.Context) (*OpsAlertRuntimeSettings, error) {
	defaultCfg := defaultOpsAlertRuntimeSettings()
	if s == nil || s.settingRepo == nil {
		return defaultCfg, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	raw, err := s.settingRepo.GetValue(ctx, SettingKeyOpsAlertRuntimeSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			if b, mErr := json.Marshal(defaultCfg); mErr == nil {
				_ = s.settingRepo.Set(ctx, SettingKeyOpsAlertRuntimeSettings, string(b))
			}
			return defaultCfg, nil
		}
		return nil, err
	}

	cfg := &OpsAlertRuntimeSettings{}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return defaultCfg, nil
	}

	if cfg.EvaluationIntervalSeconds <= 0 {
		cfg.EvaluationIntervalSeconds = defaultCfg.EvaluationIntervalSeconds
	}
	if cfg.MinSampleRequests < 0 {
		cfg.MinSampleRequests = 0
	}
	if cfg.MinErrorCount < 0 {
		cfg.MinErrorCount = 0
	}
	normalizeOpsDistributedLockSettings(&cfg.DistributedLock, opsAlertEvaluatorLeaderLockKeyDefault, defaultCfg.DistributedLock.TTLSeconds)
	normalizeOpsAlertSilencingSettings(&cfg.Silencing)

	return cfg, nil
}

func (s *OpsService) UpdateOpsAlertRuntimeSettings(ctx context.Context, cfg *OpsAlertRuntimeSettings) (*OpsAlertRuntimeSettings, error) {
	if s == nil || s.settingRepo == nil {
		return nil, errors.New("setting repository not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg == nil {
		return nil, errors.New("invalid config")
	}

	if cfg.EvaluationIntervalSeconds < 1 || cfg.EvaluationIntervalSeconds > int((24*time.Hour).Seconds()) {
		return nil, errors.New("evaluation_interval_seconds must be between 1 and 86400")
	}
	if cfg.MinSampleRequests < 0 {
		return nil, errors.New("min_sample_requests must be >= 0")
	}
	if cfg.MinErrorCount < 0 {
		return nil, errors.New("min_error_count must be >= 0")
	}
	if cfg.DistributedLock.Enabled {
		if err := validateOpsDistributedLockSettings(cfg.DistributedLock); err != nil {
			return nil, err
		}
	}
	if cfg.Silencing.Enabled {
		if err := validateOpsAlertSilencingSettings(cfg.Silencing); err != nil {
			return nil, err
		}
	}

	defaultCfg := defaultOpsAlertRuntimeSettings()
	normalizeOpsDistributedLockSettings(&cfg.DistributedLock, opsAlertEvaluatorLeaderLockKeyDefault, defaultCfg.DistributedLock.TTLSeconds)
	normalizeOpsAlertSilencingSettings(&cfg.Silencing)

	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyOpsAlertRuntimeSettings, string(raw)); err != nil {
		return nil, err
	}

	// Return a fresh copy (avoid callers holding pointers into internal slices that may be mutated).
	updated := &OpsAlertRuntimeSettings{}
	_ = json.Unmarshal(raw, updated)
	return updated, nil
}

// =========================
// Advanced settings
// =========================

func defaultOpsAdvancedSettings() *OpsAdvancedSettings {
	return &OpsAdvancedSettings{
		DataRetention: OpsDataRetentionSettings{
			// Keep the DB-backed default aligned with ops.cleanup.enabled so a
			// first settings read cannot silently disable configured retention.
			CleanupEnabled:             true,
			CleanupSchedule:            opsCleanupDefaultSchedule,
			ErrorLogRetentionDays:      30,
			MinuteMetricsRetentionDays: 30,
			HourlyMetricsRetentionDays: 30,
		},
		Aggregation: OpsAggregationSettings{
			AggregationEnabled: false,
		},
		OpenAIAccountQuotaAutoPause:     OpsOpenAIAccountQuotaAutoPauseSettings{},
		IgnoreCountTokensErrors:         true,  // count_tokens 404 是预期行为，默认忽略
		IgnoreContextCanceled:           true,  // Default to true - client disconnects are not errors
		IgnoreNoAvailableAccounts:       false, // Default to false - this is a real routing issue
		IgnoreInvalidApiKeyErrors:       true,  // Legacy compatibility field; admission rejects are always excluded.
		IgnoreInsufficientBalanceErrors: false, // 默认不忽略，余额不足可能需要关注
		DisplayOpenAITokenStats:         false,
		DisplayAlertEvents:              true,
		AutoRefreshEnabled:              false,
		AutoRefreshIntervalSec:          30,
	}
}

func defaultOpsAdvancedSettingsForConfig(cfg *config.Config) *OpsAdvancedSettings {
	defaults := defaultOpsAdvancedSettings()
	if cfg != nil {
		defaults.DataRetention.CleanupEnabled = cfg.Ops.Cleanup.Enabled
	}
	return defaults
}

func normalizeOpsAdvancedSettings(cfg *OpsAdvancedSettings) {
	if cfg == nil {
		return
	}
	// Admission rejects are a security/traffic concern, not an operational
	// request-error category. Keep the legacy field true for old clients but do
	// not allow it to re-enable those rows.
	cfg.IgnoreInvalidApiKeyErrors = true
	cfg.OpenAIAccountQuotaAutoPause.DefaultThreshold5h = clampOpsQuotaAutoPauseThreshold(cfg.OpenAIAccountQuotaAutoPause.DefaultThreshold5h)
	cfg.OpenAIAccountQuotaAutoPause.DefaultThreshold7d = clampOpsQuotaAutoPauseThreshold(cfg.OpenAIAccountQuotaAutoPause.DefaultThreshold7d)
	cfg.DataRetention.CleanupSchedule = strings.TrimSpace(cfg.DataRetention.CleanupSchedule)
	if cfg.DataRetention.CleanupSchedule == "" {
		cfg.DataRetention.CleanupSchedule = opsCleanupDefaultSchedule
	}
	// 保留天数：0 表示每次定时清理全部（清空所有），> 0 表示按天数保留；
	// 仅在拿到非法的负数时回填默认值，避免覆盖用户主动设的 0。
	if cfg.DataRetention.ErrorLogRetentionDays < 0 {
		cfg.DataRetention.ErrorLogRetentionDays = 30
	}
	if cfg.DataRetention.MinuteMetricsRetentionDays < 0 {
		cfg.DataRetention.MinuteMetricsRetentionDays = 30
	}
	if cfg.DataRetention.HourlyMetricsRetentionDays < 0 {
		cfg.DataRetention.HourlyMetricsRetentionDays = 30
	}
	// Normalize auto refresh interval (default 30 seconds)
	if cfg.AutoRefreshIntervalSec <= 0 {
		cfg.AutoRefreshIntervalSec = 30
	}
}

func clampOpsQuotaAutoPauseThreshold(value float64) float64 {
	if value <= 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func validateOpsAdvancedSettings(cfg *OpsAdvancedSettings) error {
	if cfg == nil {
		return errors.New("invalid config")
	}
	// 保留天数：0 表示每次清理全部，1-365 表示按天数保留。
	if cfg.DataRetention.ErrorLogRetentionDays < 0 || cfg.DataRetention.ErrorLogRetentionDays > 365 {
		return errors.New("error_log_retention_days must be between 0 and 365")
	}
	if cfg.DataRetention.MinuteMetricsRetentionDays < 0 || cfg.DataRetention.MinuteMetricsRetentionDays > 365 {
		return errors.New("minute_metrics_retention_days must be between 0 and 365")
	}
	if cfg.DataRetention.HourlyMetricsRetentionDays < 0 || cfg.DataRetention.HourlyMetricsRetentionDays > 365 {
		return errors.New("hourly_metrics_retention_days must be between 0 and 365")
	}
	if cfg.AutoRefreshIntervalSec < 15 || cfg.AutoRefreshIntervalSec > 300 {
		return errors.New("auto_refresh_interval_seconds must be between 15 and 300")
	}
	return nil
}

func (s *OpsService) GetOpsAdvancedSettings(ctx context.Context) (*OpsAdvancedSettings, error) {
	_ = ctx
	cfg := s.OpsAdvancedSettingsSnapshot()
	return &cfg, nil
}

// OpsAdvancedSettingsSnapshot returns a value copy for request hot paths. It
// avoids both repository I/O and pointer escape/allocation.
func (s *OpsService) OpsAdvancedSettingsSnapshot() OpsAdvancedSettings {
	if s != nil {
		if snapshot := s.runtimeSettings.Load(); snapshot != nil {
			return snapshot.advanced
		}
		return *defaultOpsAdvancedSettingsForConfig(s.cfg)
	}
	return *defaultOpsAdvancedSettings()
}

func (s *OpsService) UpdateOpsAdvancedSettings(ctx context.Context, cfg *OpsAdvancedSettings) (*OpsAdvancedSettings, error) {
	if s == nil || s.settingRepo == nil {
		return nil, errors.New("setting repository not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg == nil {
		return nil, errors.New("invalid config")
	}

	if err := validateOpsAdvancedSettings(cfg); err != nil {
		return nil, err
	}

	normalizeOpsAdvancedSettings(cfg)
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyOpsAdvancedSettings, string(raw)); err != nil {
		return nil, err
	}
	s.storeAdvancedSettingsSnapshot(cfg)
	// Push the new quota auto-pause settings straight into the in-memory cache that
	// the OpenAI scheduling hot path reads, so the next request observes the new value
	// without waiting for the background refresher's TTL.
	if s.quotaAutoPauseSink != nil {
		s.quotaAutoPauseSink(cfg.OpenAIAccountQuotaAutoPause)
	}

	// notify cleanup service to reload schedule/enabled.
	if s.cleanupReloader != nil {
		if rerr := s.cleanupReloader.Reload(ctx); rerr != nil {
			logger.LegacyPrintf("service.ops_settings",
				"[OpsSettings] cleanup reload after advanced-settings update failed: %v", rerr)
		}
	}

	updated := &OpsAdvancedSettings{}
	_ = json.Unmarshal(raw, updated)
	return updated, nil
}

// =========================
// Metric thresholds
// =========================

const SettingKeyOpsMetricThresholds = "ops_metric_thresholds"

func defaultOpsMetricThresholds() *OpsMetricThresholds {
	slaMin := 99.5
	ttftMax := 500.0
	reqErrMax := 5.0
	upstreamErrMax := 5.0
	return &OpsMetricThresholds{
		SLAPercentMin:               &slaMin,
		TTFTp99MsMax:                &ttftMax,
		RequestErrorRatePercentMax:  &reqErrMax,
		UpstreamErrorRatePercentMax: &upstreamErrMax,
	}
}

func (s *OpsService) GetMetricThresholds(ctx context.Context) (*OpsMetricThresholds, error) {
	defaultCfg := defaultOpsMetricThresholds()
	if s == nil || s.settingRepo == nil {
		return defaultCfg, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	raw, err := s.settingRepo.GetValue(ctx, SettingKeyOpsMetricThresholds)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			if b, mErr := json.Marshal(defaultCfg); mErr == nil {
				_ = s.settingRepo.Set(ctx, SettingKeyOpsMetricThresholds, string(b))
			}
			return defaultCfg, nil
		}
		return nil, err
	}

	cfg := &OpsMetricThresholds{}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return defaultCfg, nil
	}

	return cfg, nil
}

func (s *OpsService) UpdateMetricThresholds(ctx context.Context, cfg *OpsMetricThresholds) (*OpsMetricThresholds, error) {
	if s == nil || s.settingRepo == nil {
		return nil, errors.New("setting repository not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg == nil {
		return nil, errors.New("invalid config")
	}

	// Validate thresholds
	if cfg.SLAPercentMin != nil && (*cfg.SLAPercentMin < 0 || *cfg.SLAPercentMin > 100) {
		return nil, errors.New("sla_percent_min must be between 0 and 100")
	}
	if cfg.TTFTp99MsMax != nil && *cfg.TTFTp99MsMax < 0 {
		return nil, errors.New("ttft_p99_ms_max must be >= 0")
	}
	if cfg.RequestErrorRatePercentMax != nil && (*cfg.RequestErrorRatePercentMax < 0 || *cfg.RequestErrorRatePercentMax > 100) {
		return nil, errors.New("request_error_rate_percent_max must be between 0 and 100")
	}
	if cfg.UpstreamErrorRatePercentMax != nil && (*cfg.UpstreamErrorRatePercentMax < 0 || *cfg.UpstreamErrorRatePercentMax > 100) {
		return nil, errors.New("upstream_error_rate_percent_max must be between 0 and 100")
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyOpsMetricThresholds, string(raw)); err != nil {
		return nil, err
	}

	updated := &OpsMetricThresholds{}
	_ = json.Unmarshal(raw, updated)
	return updated, nil
}
