package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	NotificationEmailEventAuthVerifyCode              = "auth.verify_code"
	NotificationEmailEventAuthPasswordReset           = "auth.password_reset"
	NotificationEmailEventNotificationEmailVerifyCode = "notification_email.verify_code"
	NotificationEmailEventSubscriptionPurchaseSuccess = "subscription.purchase_success"
	NotificationEmailEventSubscriptionExpiryReminder  = "subscription.expiry_reminder"
	NotificationEmailEventBalanceLow                  = "balance.low"
	NotificationEmailEventBalanceRechargeSuccess      = "balance.recharge_success"
	NotificationEmailEventAccountQuotaAlert           = "account.quota_alert"
	NotificationEmailEventContentModerationViolation  = "content_moderation.violation_notice"
	NotificationEmailEventContentModerationDisabled   = "content_moderation.account_disabled"
	NotificationEmailEventCyberPolicyNotice           = "content_moderation.cyber_policy_notice"
	NotificationEmailEventOpsAlert                    = "ops.alert"
	NotificationEmailEventOpsScheduledReport          = "ops.scheduled_report"

	notificationEmailTemplateKeyPrefix    = "notification_email_template:"
	notificationEmailPreferenceKeyPrefix  = "notification_email_preference:"
	notificationEmailDeliveryKeyPrefix    = "notification_email_delivery:"
	notificationEmailLocaleUserKeyPrefix  = "notification_email_locale:user:"
	notificationEmailLocaleEmailKeyPrefix = "notification_email_locale:email:"
	notificationEmailUnsubscribeSecretKey = "notification_email_unsubscribe_secret"
	notificationEmailDefaultLocale        = "en"
	notificationEmailLocaleChinese        = "zh"
	notificationEmailMaxSubjectLength     = 200
	notificationEmailMaxHTMLLength        = 30000
	notificationEmailUnsubscribeTTL       = 365 * 24 * time.Hour
)

var (
	notificationEmailPlaceholderPattern = regexp.MustCompile(`{{\s*([a-zA-Z][a-zA-Z0-9_]*)\s*}}`)
	notificationEmailHexColorPattern    = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	notificationEmailLocales            = []string{notificationEmailDefaultLocale, notificationEmailLocaleChinese}
	notificationEmailCommonPlaceholders = []string{"site_name", "recipient_name", "recipient_email"}
	// Keep summary values separate so admins can rearrange or omit individual metrics in the template.
	notificationEmailOpsSummaryPlaceholders = []string{
		"report_summary_display",
		"report_total_requests",
		"report_success_count",
		"report_sla_error_count",
		"report_business_limited_count",
		"report_sla",
		"report_error_rate",
		"report_upstream_error_rate",
		"report_upstream_error_count_excl_429_529",
		"report_upstream_429_count",
		"report_upstream_529_count",
		"report_latency_p50",
		"report_latency_p99",
		"report_ttft_p50",
		"report_ttft_p99",
		"report_tokens",
		"report_qps_current",
		"report_qps_peak",
		"report_qps_avg",
		"report_tps_current",
		"report_tps_peak",
		"report_tps_avg",
	}
)

type NotificationEmailService struct {
	settingRepo  SettingRepository
	emailService *EmailService
}

type NotificationEmailEventInfo struct {
	Event        string   `json:"event"`
	Label        string   `json:"label"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	Optional     bool     `json:"optional"`
	Placeholders []string `json:"placeholders"`
}

type NotificationEmailTemplate struct {
	Event        string     `json:"event"`
	Locale       string     `json:"locale"`
	Subject      string     `json:"subject"`
	HTML         string     `json:"html"`
	IsCustom     bool       `json:"is_custom"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
	Placeholders []string   `json:"placeholders"`
}

type NotificationEmailPreview struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type NotificationEmailPreviewInput struct {
	Event     string            `json:"event"`
	Locale    string            `json:"locale"`
	Subject   string            `json:"subject"`
	HTML      string            `json:"html"`
	Variables map[string]string `json:"variables,omitempty"`
}

type NotificationEmailSendInput struct {
	Event            string
	Locale           string
	RecipientEmail   string
	RecipientName    string
	UserID           int64
	SourceType       string
	SourceID         string
	ReminderKey      string
	Variables        map[string]string
	RawHTMLVariables map[string]string
}

type NotificationEmailUnsubscribeResult struct {
	Event string `json:"event"`
	Email string `json:"email"`
	Done  bool   `json:"done"`
}

type notificationEmailStoredTemplate struct {
	Subject   string    `json:"subject"`
	HTML      string    `json:"html"`
	UpdatedAt time.Time `json:"updated_at"`
}

type notificationEmailOfficialTemplate struct {
	Subject string
	HTML    string
}

type notificationEmailTemplateError struct {
	Err error
}

func (e notificationEmailTemplateError) Error() string {
	return e.Err.Error()
}

func (e notificationEmailTemplateError) Unwrap() error {
	return e.Err
}

type notificationEmailConfigError struct {
	Err error
}

func (e notificationEmailConfigError) Error() string {
	return e.Err.Error()
}

func (e notificationEmailConfigError) Unwrap() error {
	return e.Err
}

type notificationEmailDeliveryError struct {
	Err error
}

func (e notificationEmailDeliveryError) Error() string {
	return e.Err.Error()
}

func (e notificationEmailDeliveryError) Unwrap() error {
	return e.Err
}

type notificationEmailUnsubscribeClaims struct {
	Email string `json:"email"`
	Event string `json:"event"`
	Exp   int64  `json:"exp"`
}

func NewNotificationEmailService(settingRepo SettingRepository, emailService *EmailService) *NotificationEmailService {
	svc := &NotificationEmailService{settingRepo: settingRepo, emailService: emailService}
	if emailService != nil {
		emailService.SetNotificationEmailService(svc)
	}
	return svc
}

func notificationEmailTemplateErr(err error) error {
	if err == nil {
		return nil
	}
	return notificationEmailTemplateError{Err: err}
}

func notificationEmailConfigErr(err error) error {
	if err == nil {
		return nil
	}
	return notificationEmailConfigError{Err: err}
}

func notificationEmailDeliveryErr(err error) error {
	if err == nil {
		return nil
	}
	return notificationEmailDeliveryError{Err: err}
}

func shouldFallbackNotificationEmail(err error) bool {
	if err == nil {
		return false
	}
	var templateErr notificationEmailTemplateError
	if errors.As(err, &templateErr) {
		return true
	}
	var configErr notificationEmailConfigError
	return errors.As(err, &configErr)
}

func isNotificationEmailDeliveryError(err error) bool {
	var deliveryErr notificationEmailDeliveryError
	return errors.As(err, &deliveryErr)
}

func (s *NotificationEmailService) ListEventInfos() []NotificationEmailEventInfo {
	infos := make([]NotificationEmailEventInfo, 0, len(notificationEmailEventDefinitions))
	for _, event := range notificationEmailEventOrder {
		info := notificationEmailEventDefinitions[event]
		info.Placeholders = append([]string(nil), info.Placeholders...)
		infos = append(infos, info)
	}
	return infos
}

func (s *NotificationEmailService) SupportedLocales() []string {
	return append([]string(nil), notificationEmailLocales...)
}

func (s *NotificationEmailService) ListTemplates(ctx context.Context) ([]NotificationEmailTemplate, error) {
	items := make([]NotificationEmailTemplate, 0, len(notificationEmailEventOrder)*len(notificationEmailLocales))
	for _, event := range notificationEmailEventOrder {
		for _, locale := range notificationEmailLocales {
			tmpl, err := s.GetTemplate(ctx, event, locale)
			if err != nil {
				return nil, err
			}
			items = append(items, tmpl)
		}
	}
	return items, nil
}

func (s *NotificationEmailService) GetTemplate(ctx context.Context, event, locale string) (NotificationEmailTemplate, error) {
	info, normalizedEvent, err := s.eventInfo(event)
	if err != nil {
		return NotificationEmailTemplate{}, err
	}
	normalizedLocale := normalizeNotificationLocale(locale)
	official, ok := notificationEmailOfficialTemplates[normalizedEvent][normalizedLocale]
	if !ok {
		return NotificationEmailTemplate{}, fmt.Errorf("official template not found for %s/%s", normalizedEvent, normalizedLocale)
	}

	tmpl := NotificationEmailTemplate{
		Event:        normalizedEvent,
		Locale:       normalizedLocale,
		Subject:      official.Subject,
		HTML:         official.HTML,
		Placeholders: append([]string(nil), info.Placeholders...),
	}

	raw, err := s.settingRepo.GetValue(ctx, notificationEmailTemplateKey(normalizedEvent, normalizedLocale))
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return tmpl, nil
		}
		return NotificationEmailTemplate{}, err
	}
	if strings.TrimSpace(raw) == "" {
		return tmpl, nil
	}

	var stored notificationEmailStoredTemplate
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return NotificationEmailTemplate{}, fmt.Errorf("decode email template override: %w", err)
	}
	if err := validateNotificationEmailTemplate(normalizedEvent, stored.Subject, stored.HTML); err != nil {
		return NotificationEmailTemplate{}, err
	}
	tmpl.Subject = stored.Subject
	tmpl.HTML = stored.HTML
	tmpl.IsCustom = true
	updatedAt := stored.UpdatedAt
	tmpl.UpdatedAt = &updatedAt
	return tmpl, nil
}

func (s *NotificationEmailService) UpdateTemplate(ctx context.Context, event, locale, subject, htmlBody string) (NotificationEmailTemplate, error) {
	_, normalizedEvent, err := s.eventInfo(event)
	if err != nil {
		return NotificationEmailTemplate{}, err
	}
	normalizedLocale := normalizeNotificationLocale(locale)
	if err := validateNotificationEmailTemplate(normalizedEvent, subject, htmlBody); err != nil {
		return NotificationEmailTemplate{}, err
	}
	stored := notificationEmailStoredTemplate{
		Subject:   strings.TrimSpace(subject),
		HTML:      htmlBody,
		UpdatedAt: time.Now().UTC(),
	}
	payload, err := json.Marshal(stored)
	if err != nil {
		return NotificationEmailTemplate{}, err
	}
	if err := s.settingRepo.Set(ctx, notificationEmailTemplateKey(normalizedEvent, normalizedLocale), string(payload)); err != nil {
		return NotificationEmailTemplate{}, err
	}
	return s.GetTemplate(ctx, normalizedEvent, normalizedLocale)
}

func (s *NotificationEmailService) RestoreOfficialTemplate(ctx context.Context, event, locale string) (NotificationEmailTemplate, error) {
	_, normalizedEvent, err := s.eventInfo(event)
	if err != nil {
		return NotificationEmailTemplate{}, err
	}
	normalizedLocale := normalizeNotificationLocale(locale)
	if err := s.settingRepo.Delete(ctx, notificationEmailTemplateKey(normalizedEvent, normalizedLocale)); err != nil && !errors.Is(err, ErrSettingNotFound) {
		return NotificationEmailTemplate{}, err
	}
	return s.GetTemplate(ctx, normalizedEvent, normalizedLocale)
}

func (s *NotificationEmailService) PreviewTemplate(ctx context.Context, input NotificationEmailPreviewInput) (NotificationEmailPreview, error) {
	_, normalizedEvent, err := s.eventInfo(input.Event)
	if err != nil {
		return NotificationEmailPreview{}, err
	}
	normalizedLocale := normalizeNotificationLocale(input.Locale)
	subject := input.Subject
	htmlBody := input.HTML
	if strings.TrimSpace(subject) == "" || strings.TrimSpace(htmlBody) == "" {
		tmpl, err := s.GetTemplate(ctx, normalizedEvent, normalizedLocale)
		if err != nil {
			return NotificationEmailPreview{}, err
		}
		if strings.TrimSpace(subject) == "" {
			subject = tmpl.Subject
		}
		if strings.TrimSpace(htmlBody) == "" {
			htmlBody = tmpl.HTML
		}
	}
	if err := validateNotificationEmailTemplate(normalizedEvent, subject, htmlBody); err != nil {
		return NotificationEmailPreview{}, err
	}
	variables := s.sampleVariables(ctx, normalizedEvent, normalizedLocale)
	rawHTMLVariables := notificationEmailSampleRawHTML(normalizedEvent, normalizedLocale)
	for key, value := range input.Variables {
		variables[key] = value
		delete(rawHTMLVariables, key)
	}
	if normalizedEvent == NotificationEmailEventOpsAlert {
		variables["header_color"] = sanitizeNotificationEmailHexColor(variables["header_color"], "#ea580c")
	}
	return renderNotificationEmail(normalizedEvent, subject, htmlBody, variables, rawHTMLVariables)
}

func (s *NotificationEmailService) Send(ctx context.Context, input NotificationEmailSendInput) error {
	info, normalizedEvent, err := s.eventInfo(input.Event)
	if err != nil {
		return notificationEmailTemplateErr(err)
	}
	recipient := strings.TrimSpace(input.RecipientEmail)
	if recipient == "" {
		return nil
	}
	if info.Optional {
		unsubscribed, err := s.IsUnsubscribed(ctx, recipient, normalizedEvent)
		if err != nil {
			return err
		}
		if unsubscribed {
			slog.Info("notification email suppressed by unsubscribe preference", "event", normalizedEvent, "recipient_hash", notificationEmailHash(recipient))
			return nil
		}
	}

	locale := normalizeNotificationLocale(input.Locale)
	if strings.TrimSpace(input.Locale) == "" {
		locale = s.ResolveRecipientLocale(ctx, input.UserID, recipient)
	}
	tmpl, err := s.GetTemplate(ctx, normalizedEvent, locale)
	if err != nil {
		return notificationEmailTemplateErr(err)
	}
	variables := s.runtimeVariables(ctx, normalizedEvent, locale, input)
	rendered, err := renderNotificationEmail(normalizedEvent, tmpl.Subject, tmpl.HTML, variables, input.RawHTMLVariables)
	if err != nil {
		return notificationEmailTemplateErr(err)
	}

	deliveryKey := notificationEmailDeliveryKey(normalizedEvent, input.SourceType, input.SourceID, recipient, input.ReminderKey)
	if deliveryKey != "" {
		sent, err := s.deliveryExists(ctx, deliveryKey, legacyNotificationEmailDeliveryKey(normalizedEvent, input.SourceType, input.SourceID, recipient, input.ReminderKey))
		if err != nil {
			return err
		}
		if sent {
			return nil
		}
	}

	if s.emailService == nil {
		return notificationEmailConfigErr(errors.New("email service is not configured"))
	}
	if err := s.emailService.SendEmail(ctx, recipient, rendered.Subject, rendered.HTML); err != nil {
		return notificationEmailDeliveryErr(err)
	}
	if deliveryKey != "" {
		if err := s.settingRepo.Set(ctx, deliveryKey, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	return nil
}

func (s *NotificationEmailService) RememberRecipientLocale(ctx context.Context, userID int64, email, acceptLanguage string) {
	locale := normalizeNotificationLocale(acceptLanguage)
	if strings.TrimSpace(acceptLanguage) == "" || s == nil || s.settingRepo == nil {
		return
	}
	if userID > 0 {
		_ = s.settingRepo.Set(ctx, notificationEmailLocaleUserKeyPrefix+strconv.FormatInt(userID, 10), locale)
	}
	if emailHash := notificationEmailHash(email); emailHash != "" {
		_ = s.settingRepo.Set(ctx, notificationEmailLocaleEmailKeyPrefix+emailHash, locale)
	}
}

func (s *NotificationEmailService) ResolveRecipientLocale(ctx context.Context, userID int64, email string) string {
	if s == nil || s.settingRepo == nil {
		return notificationEmailDefaultLocale
	}
	if userID > 0 {
		if locale, err := s.settingRepo.GetValue(ctx, notificationEmailLocaleUserKeyPrefix+strconv.FormatInt(userID, 10)); err == nil && strings.TrimSpace(locale) != "" {
			return normalizeNotificationLocale(locale)
		}
	}
	if emailHash := notificationEmailHash(email); emailHash != "" {
		if locale, err := s.settingRepo.GetValue(ctx, notificationEmailLocaleEmailKeyPrefix+emailHash); err == nil && strings.TrimSpace(locale) != "" {
			return normalizeNotificationLocale(locale)
		}
	}
	return notificationEmailDefaultLocale
}

func (s *NotificationEmailService) IsUnsubscribed(ctx context.Context, email, event string) (bool, error) {
	info, normalizedEvent, err := s.eventInfo(event)
	if err != nil {
		return false, err
	}
	if !info.Optional {
		return false, nil
	}
	for _, key := range []string{notificationEmailPreferenceKey(normalizedEvent, email), legacyNotificationEmailPreferenceKey(normalizedEvent, email)} {
		if strings.TrimSpace(key) == "" {
			continue
		}
		value, err := s.settingRepo.GetValue(ctx, key)
		if err == nil {
			return strings.EqualFold(strings.TrimSpace(value), "unsubscribed"), nil
		}
		if !errors.Is(err, ErrSettingNotFound) {
			return false, err
		}
	}
	return false, nil
}

func (s *NotificationEmailService) Unsubscribe(ctx context.Context, token string) (NotificationEmailUnsubscribeResult, error) {
	claims, err := s.parseUnsubscribeToken(ctx, token)
	if err != nil {
		return NotificationEmailUnsubscribeResult{}, err
	}
	info, normalizedEvent, err := s.eventInfo(claims.Event)
	if err != nil {
		return NotificationEmailUnsubscribeResult{}, err
	}
	if !info.Optional {
		return NotificationEmailUnsubscribeResult{}, fmt.Errorf("%s is transactional and cannot be unsubscribed", normalizedEvent)
	}
	if err := s.settingRepo.Set(ctx, notificationEmailPreferenceKey(normalizedEvent, claims.Email), "unsubscribed"); err != nil {
		return NotificationEmailUnsubscribeResult{}, err
	}
	return NotificationEmailUnsubscribeResult{Event: normalizedEvent, Email: claims.Email, Done: true}, nil
}

func (s *NotificationEmailService) eventInfo(event string) (NotificationEmailEventInfo, string, error) {
	normalized := strings.ToLower(strings.TrimSpace(event))
	info, ok := notificationEmailEventDefinitions[normalized]
	if !ok {
		return NotificationEmailEventInfo{}, "", fmt.Errorf("unsupported email template event: %s", event)
	}
	return info, normalized, nil
}

func (s *NotificationEmailService) sampleVariables(ctx context.Context, event, locale string) map[string]string {
	info := notificationEmailEventDefinitions[event]
	variables := make(map[string]string, len(info.Placeholders))
	for key, value := range notificationEmailSampleVariables(locale) {
		variables[key] = value
	}
	variables["site_name"] = s.siteName(ctx)
	if variables["unsubscribe_url"] == "" && info.Optional {
		variables["unsubscribe_url"] = "https://example.com/unsubscribe"
	}
	return variables
}

func (s *NotificationEmailService) runtimeVariables(ctx context.Context, event, locale string, input NotificationEmailSendInput) map[string]string {
	variables := s.sampleVariables(ctx, event, locale)
	for key, value := range input.Variables {
		variables[key] = value
	}
	if event == NotificationEmailEventOpsAlert {
		if _, ok := input.Variables["alert_details_html"]; !ok {
			variables["alert_details_html"] = ""
		}
		if _, ok := input.Variables["dashboard_cta_display"]; !ok {
			variables["dashboard_cta_display"] = "none"
			variables["dashboard_url"] = ""
		}
		if _, ok := input.Variables["diagnosis_badge"]; !ok {
			variables["diagnosis_badge"] = opsDiagnosisBadge(false, false, locale)
		}
		if _, ok := input.Variables["diagnosis_summary"]; !ok {
			variables["diagnosis_summary"] = variables["alert_description"]
		}
		variables["header_color"] = sanitizeNotificationEmailHexColor(variables["header_color"], "#ea580c")
	}
	if event == NotificationEmailEventOpsScheduledReport {
		// Scheduled reports may be sent by integrations that only provide report_html.
		// Do not let preview sample values appear in a live email in that case.
		if _, ok := input.Variables["report_html"]; !ok {
			variables["report_html"] = ""
		}
		if _, ok := input.Variables["report_insights_html"]; !ok {
			variables["report_insights_html"] = ""
		}
		if _, ok := input.Variables["report_insights_display"]; !ok {
			variables["report_insights_display"] = "none"
		}
		if _, ok := input.Variables["dashboard_cta_display"]; !ok {
			variables["dashboard_cta_display"] = "none"
			variables["dashboard_url"] = ""
		}
		if _, ok := input.Variables["report_period_display"]; !ok {
			variables["report_period_display"] = variables["report_start_time"] + " ~ " + variables["report_end_time"]
		}
		if _, ok := input.Variables["report_detail_display"]; !ok {
			// Keep legacy/custom templates useful when they only render report_html.
			variables["report_detail_display"] = "block"
		}
		hasSummaryValues := false
		for _, placeholder := range notificationEmailOpsSummaryPlaceholders {
			if _, ok := input.Variables[placeholder]; ok {
				if placeholder != "report_summary_display" {
					hasSummaryValues = true
				}
				continue
			}
			variables[placeholder] = "-"
		}
		if _, ok := input.Variables["report_summary_display"]; !ok {
			if hasSummaryValues {
				variables["report_summary_display"] = "block"
			} else {
				variables["report_summary_display"] = "none"
			}
		}
	}
	variables["site_name"] = s.siteName(ctx)
	variables["recipient_email"] = input.RecipientEmail
	if strings.TrimSpace(input.RecipientName) != "" {
		variables["recipient_name"] = input.RecipientName
	}
	if notificationEmailEventDefinitions[event].Optional {
		if unsubscribeURL, err := s.buildUnsubscribeURL(ctx, input.RecipientEmail, event); err == nil {
			variables["unsubscribe_url"] = unsubscribeURL
		}
	}
	return variables
}

func (s *NotificationEmailService) siteName(ctx context.Context) string {
	if s == nil || s.settingRepo == nil {
		return defaultSiteName
	}
	name, err := s.settingRepo.GetValue(ctx, SettingKeySiteName)
	if err != nil || strings.TrimSpace(name) == "" {
		return defaultSiteName
	}
	return strings.TrimSpace(name)
}

func (s *NotificationEmailService) baseURL(ctx context.Context) string {
	if s == nil || s.settingRepo == nil {
		return ""
	}
	for _, key := range []string{SettingKeyAPIBaseURL, SettingKeyFrontendURL} {
		value, err := s.settingRepo.GetValue(ctx, key)
		if err == nil && strings.TrimSpace(value) != "" {
			return strings.TrimRight(strings.TrimSpace(value), "/")
		}
	}
	return ""
}

func (s *NotificationEmailService) buildUnsubscribeURL(ctx context.Context, email, event string) (string, error) {
	token, err := s.createUnsubscribeToken(ctx, email, event)
	if err != nil {
		return "", err
	}
	path := "/api/v1/settings/email-unsubscribe?token=" + url.QueryEscape(token)
	baseURL := s.baseURL(ctx)
	if baseURL == "" {
		return path, nil
	}
	return baseURL + path, nil
}

func (s *NotificationEmailService) createUnsubscribeToken(ctx context.Context, email, event string) (string, error) {
	secret, err := s.unsubscribeSecret(ctx)
	if err != nil {
		return "", err
	}
	claims := notificationEmailUnsubscribeClaims{Email: strings.TrimSpace(email), Event: event, Exp: time.Now().Add(notificationEmailUnsubscribeTTL).Unix()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := signNotificationEmailToken(secret, encodedPayload)
	return encodedPayload + "." + signature, nil
}

func (s *NotificationEmailService) parseUnsubscribeToken(ctx context.Context, token string) (notificationEmailUnsubscribeClaims, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return notificationEmailUnsubscribeClaims{}, errors.New("invalid unsubscribe token")
	}
	secret, err := s.unsubscribeSecret(ctx)
	if err != nil {
		return notificationEmailUnsubscribeClaims{}, err
	}
	expected := signNotificationEmailToken(secret, parts[0])
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return notificationEmailUnsubscribeClaims{}, errors.New("invalid unsubscribe token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return notificationEmailUnsubscribeClaims{}, errors.New("invalid unsubscribe token payload")
	}
	var claims notificationEmailUnsubscribeClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return notificationEmailUnsubscribeClaims{}, errors.New("invalid unsubscribe token payload")
	}
	if strings.TrimSpace(claims.Email) == "" || strings.TrimSpace(claims.Event) == "" {
		return notificationEmailUnsubscribeClaims{}, errors.New("invalid unsubscribe token claims")
	}
	if claims.Exp <= time.Now().Unix() {
		return notificationEmailUnsubscribeClaims{}, errors.New("unsubscribe token expired")
	}
	return claims, nil
}

func (s *NotificationEmailService) unsubscribeSecret(ctx context.Context) (string, error) {
	secret, err := s.settingRepo.GetValue(ctx, notificationEmailUnsubscribeSecretKey)
	if err == nil && strings.TrimSpace(secret) != "" {
		return strings.TrimSpace(secret), nil
	}
	if err != nil && !errors.Is(err, ErrSettingNotFound) {
		return "", err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	secret = base64.RawURLEncoding.EncodeToString(buf)
	if err := s.settingRepo.Set(ctx, notificationEmailUnsubscribeSecretKey, secret); err != nil {
		return "", err
	}
	return secret, nil
}

func (s *NotificationEmailService) deliveryExists(ctx context.Context, keys ...string) (bool, error) {
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		_, err := s.settingRepo.GetValue(ctx, key)
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, ErrSettingNotFound) {
			return false, err
		}
	}
	return false, nil
}

func validateNotificationEmailTemplate(event, subject, htmlBody string) error {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return errors.New("email subject cannot be empty")
	}
	if len([]rune(subject)) > notificationEmailMaxSubjectLength {
		return fmt.Errorf("email subject cannot exceed %d characters", notificationEmailMaxSubjectLength)
	}
	if strings.TrimSpace(htmlBody) == "" {
		return errors.New("email html cannot be empty")
	}
	if len([]byte(htmlBody)) > notificationEmailMaxHTMLLength {
		return fmt.Errorf("email html cannot exceed %d bytes", notificationEmailMaxHTMLLength)
	}
	allowed := notificationEmailAllowedPlaceholderSet(event)
	for _, placeholder := range notificationEmailPlaceholdersIn(subject + "\n" + htmlBody) {
		if _, ok := allowed[placeholder]; !ok {
			return fmt.Errorf("unsupported placeholder {{%s}} for event %s", placeholder, event)
		}
	}
	return nil
}

func renderNotificationEmail(event, subject, htmlBody string, variables map[string]string, rawHTMLVariables map[string]string) (NotificationEmailPreview, error) {
	if err := validateNotificationEmailTemplate(event, subject, htmlBody); err != nil {
		return NotificationEmailPreview{}, err
	}
	renderedSubject, err := renderNotificationEmailString(event, subject, variables, nil, false)
	if err != nil {
		return NotificationEmailPreview{}, err
	}
	renderedHTML, err := renderNotificationEmailString(event, htmlBody, variables, rawHTMLVariables, true)
	if err != nil {
		return NotificationEmailPreview{}, err
	}
	return NotificationEmailPreview{Subject: sanitizeEmailHeader(renderedSubject), HTML: renderedHTML}, nil
}

func renderNotificationEmailString(event, raw string, variables map[string]string, rawHTMLVariables map[string]string, escapeHTML bool) (string, error) {
	allowed := notificationEmailAllowedPlaceholderSet(event)
	var renderErr error
	rendered := notificationEmailPlaceholderPattern.ReplaceAllStringFunc(raw, func(match string) string {
		if renderErr != nil {
			return ""
		}
		parts := notificationEmailPlaceholderPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return ""
		}
		name := parts[1]
		if _, ok := allowed[name]; !ok {
			renderErr = fmt.Errorf("unsupported placeholder {{%s}} for event %s", name, event)
			return ""
		}
		value := variables[name]
		if escapeHTML && notificationEmailRawHTMLAllowed(event, name) {
			if rawHTMLVariables != nil {
				if rawValue, ok := rawHTMLVariables[name]; ok {
					return rawValue
				}
			}
		}
		if strings.HasSuffix(name, "_url") && !isSafeNotificationEmailURL(value) {
			value = ""
		}
		if escapeHTML {
			return html.EscapeString(value)
		}
		return sanitizeEmailHeader(value)
	})
	if renderErr != nil {
		return "", renderErr
	}
	return rendered, nil
}

func notificationEmailRawHTMLAllowed(event, placeholder string) bool {
	if event == NotificationEmailEventOpsScheduledReport {
		return placeholder == "report_html" || placeholder == "report_insights_html"
	}
	if event == NotificationEmailEventOpsAlert {
		return placeholder == "alert_details_html"
	}
	return false
}

func notificationEmailAllowedPlaceholderSet(event string) map[string]struct{} {
	info := notificationEmailEventDefinitions[event]
	allowed := make(map[string]struct{}, len(info.Placeholders))
	for _, placeholder := range info.Placeholders {
		allowed[placeholder] = struct{}{}
	}
	return allowed
}

func notificationEmailPlaceholdersIn(raw string) []string {
	matches := notificationEmailPlaceholderPattern.FindAllStringSubmatch(raw, -1)
	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		if _, exists := seen[match[1]]; exists {
			continue
		}
		seen[match[1]] = struct{}{}
		out = append(out, match[1])
	}
	return out
}

func normalizeNotificationLocale(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return notificationEmailDefaultLocale
	}
	for _, part := range strings.Split(trimmed, ",") {
		tag := strings.TrimSpace(strings.Split(part, ";")[0])
		if strings.HasPrefix(tag, "zh") || tag == "cn" {
			return notificationEmailLocaleChinese
		}
		if strings.HasPrefix(tag, "en") {
			return notificationEmailDefaultLocale
		}
	}
	return notificationEmailDefaultLocale
}

func notificationEmailTemplateKey(event, locale string) string {
	return notificationEmailTemplateKeyPrefix + event + ":" + locale
}

func notificationEmailPreferenceKey(event, email string) string {
	if strings.TrimSpace(event) == "" || strings.TrimSpace(email) == "" {
		return ""
	}
	identity := strings.TrimSpace(event) + "\x00" + strings.ToLower(strings.TrimSpace(email))
	return notificationEmailPreferenceKeyPrefix + "v2:" + notificationEmailHash(identity)
}

func legacyNotificationEmailPreferenceKey(event, email string) string {
	return notificationEmailPreferenceKeyPrefix + event + ":" + notificationEmailHash(email)
}

func notificationEmailDeliveryKey(event, sourceType, sourceID, recipient, reminderKey string) string {
	if strings.TrimSpace(sourceType) == "" || strings.TrimSpace(sourceID) == "" || strings.TrimSpace(recipient) == "" {
		return ""
	}
	identity := strings.Join([]string{
		strings.ToLower(strings.TrimSpace(event)),
		safeNotificationEmailKeyPart(sourceType),
		safeNotificationEmailKeyPart(sourceID),
		strings.ToLower(strings.TrimSpace(recipient)),
		safeNotificationEmailKeyPart(reminderKey),
	}, "\x00")
	return notificationEmailDeliveryKeyPrefix + "v2:" + notificationEmailHash(identity)
}

func legacyNotificationEmailDeliveryKey(event, sourceType, sourceID, recipient, reminderKey string) string {
	if strings.TrimSpace(sourceType) == "" || strings.TrimSpace(sourceID) == "" || strings.TrimSpace(recipient) == "" {
		return ""
	}
	parts := []string{notificationEmailDeliveryKeyPrefix, event, ":", safeNotificationEmailKeyPart(sourceType), ":", safeNotificationEmailKeyPart(sourceID), ":", notificationEmailHash(recipient)}
	if strings.TrimSpace(reminderKey) != "" {
		parts = append(parts, ":", safeNotificationEmailKeyPart(reminderKey))
	}
	return strings.Join(parts, "")
}

func notificationEmailHash(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func safeNotificationEmailKeyPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			_, _ = builder.WriteRune(r)
		} else {
			_, _ = builder.WriteRune('_')
		}
	}
	return builder.String()
}

func signNotificationEmailToken(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func isSafeNotificationEmailURL(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return true
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return false
	}
	if parsed.IsAbs() {
		scheme := strings.ToLower(parsed.Scheme)
		return scheme == "http" || scheme == "https" || scheme == "mailto"
	}
	return strings.HasPrefix(trimmed, "/")
}

func sanitizeNotificationEmailHexColor(raw, fallback string) string {
	trimmed := strings.TrimSpace(raw)
	if notificationEmailHexColorPattern.MatchString(trimmed) {
		return strings.ToLower(trimmed)
	}
	return fallback
}

func notificationEmailSampleVariables(locale string) map[string]string {
	if normalizeNotificationLocale(locale) == notificationEmailLocaleChinese {
		variables := map[string]string{
			"site_name":           defaultSiteName,
			"recipient_name":      "张三",
			"recipient_email":     "user@example.com",
			"verification_code":   "123456",
			"expires_in_minutes":  "15",
			"reset_url":           "https://example.com/reset-password?token=preview",
			"subscription_group":  "Claude Pro",
			"subscription_days":   "30",
			"expiry_time":         "2026-06-18 12:00",
			"days_remaining":      "3",
			"current_balance":     "12.34",
			"threshold":           "20.00",
			"recharge_url":        "https://example.com/recharge",
			"recharge_amount":     "50.00",
			"order_id":            "1024",
			"unsubscribe_url":     "https://example.com/unsubscribe",
			"account_id":          "1001",
			"account_name":        "openai-main",
			"platform":            "openai",
			"quota_dimension":     "每日额度",
			"quota_used":          "80.00",
			"quota_limit":         "100.00",
			"quota_remaining":     "20.00",
			"quota_threshold":     "20%",
			"triggered_at":        "2026-05-20 12:00:00",
			"group_name":          "默认分组",
			"moderation_category": "violence",
			"moderation_score":    "0.982",
			"violation_count":     "2",
			"ban_threshold":       "3",
			"rule_name":             "错误率过高",
			"severity":              "P1",
			"severity_label":        "重要（P1）",
			"alert_status":          "firing",
			"status_label":          "异常触发中",
			"metric_type":           "error_rate",
			"metric_label":          "请求错误率",
			"operator":              ">",
			"metric_value":          "12.50",
			"threshold_value":       "10.00",
			"metric_summary":        "请求错误率 当前 12.50%（告警条件：> 10.00%）",
			"triggered_at_local":    "2026-05-20 20:00:00（北京时间）",
			"window_label":          "最近 5 分钟",
			"duration_human":        "5 分钟",
			"alert_description":     "请求错误率 当前 12.50%（告警条件：> 10.00%）｜窗口：最近 5 分钟｜范围：全站全部渠道",
			"diagnosis_badge":       "AI 智能诊断",
			"diagnosis_summary":     "最近 5 分钟共处理 48 次请求，失败 6 次（影响 2 位用户）。主要为账号「主力 GPT 账号」调用 gpt-6.2-sol 出现上游超时（524），建议检查该账号代理延迟或切备用号。",
			"alert_details_html":    "最近 5 分钟共 48 次请求，失败 6 次（主要：账号「主力 GPT 账号」上游超时）",
			"dashboard_url":         "https://example.com/admin/ops",
			"dashboard_cta_display": "block",
			"header_color":          "#ea580c",
			"email_title":           "重要告警",
			"alert_headline":        "最近 5 分钟 48 次请求失败 6 次（账号「主力 GPT 账号」· gpt-6.2-sol）",
			"report_name":           "日报",
			"report_type":           "daily_summary",
			"report_start_time":     "2026-07-18T01:00:26Z",
			"report_end_time":       "2026-07-19T01:00:26Z",
			"report_period_display": "2026-07-18 09:00 ~ 2026-07-19 09:00（北京时间）",
			"report_html":           "<h2>日报</h2><p>请求量：2,374</p>",
		}
		addNotificationEmailOpsSummarySampleVariables(variables, notificationEmailLocaleChinese)
		return variables
	}
	variables := map[string]string{
		"site_name":           defaultSiteName,
		"recipient_name":      "Alex",
		"recipient_email":     "user@example.com",
		"verification_code":   "123456",
		"expires_in_minutes":  "15",
		"reset_url":           "https://example.com/reset-password?token=preview",
		"subscription_group":  "Claude Pro",
		"subscription_days":   "30",
		"expiry_time":         "2026-06-18 12:00",
		"days_remaining":      "3",
		"current_balance":     "12.34",
		"threshold":           "20.00",
		"recharge_url":        "https://example.com/recharge",
		"recharge_amount":     "50.00",
		"order_id":            "1024",
		"unsubscribe_url":     "https://example.com/unsubscribe",
		"account_id":          "1001",
		"account_name":        "openai-main",
		"platform":            "openai",
		"quota_dimension":     "Daily quota",
		"quota_used":          "80.00",
		"quota_limit":         "100.00",
		"quota_remaining":     "20.00",
		"quota_threshold":     "20%",
		"triggered_at":        "2026-05-20 12:00:00",
		"group_name":          "Default group",
		"moderation_category": "violence",
		"moderation_score":    "0.982",
		"violation_count":     "2",
		"ban_threshold":       "3",
		"rule_name":           "High error rate",
		"severity":            "P1",
		"severity_label":      "Warning (P1)",
		"alert_status":        "firing",
		"status_label":        "Firing",
		"metric_type":         "error_rate",
		"metric_label":        "Error rate",
		"operator":            ">",
		"metric_value":        "12.50",
		"threshold_value":     "10.00",
		"metric_summary":      "Error rate is 12.50% (rule: > 10.00%)",
		"triggered_at_local":  "2026-05-20 12:00:00 (UTC)",
		"window_label":        "Last 5 min",
		"duration_human":      "5m",
		"alert_description":   "Error rate is 12.50% (rule: > 10.00%).",
		"diagnosis_badge":     "AI Diagnosis",
		"diagnosis_summary":   "6 of 48 requests failed in the last 5 min, mostly gpt-6.2-sol timing out (524) on account \"Primary GPT Account\". Check its proxy latency.",
		"alert_details_html":  "48 SLA requests, 6 failed (top: Primary GPT Account · gpt-6.2-sol upstream timeout)",
		"dashboard_url":         "https://example.com/admin/ops",
		"dashboard_cta_display": "block",
		"header_color":          "#ea580c",
		"email_title":           "Ops Alert",
		"alert_headline":        "6/48 requests failed (Primary GPT Account · gpt-6.2-sol)",
		"report_name":           "Daily summary",
		"report_type":           "daily_summary",
		"report_start_time":     "2026-07-18T01:00:26Z",
		"report_end_time":       "2026-07-19T01:00:26Z",
		"report_period_display": "2026-07-18 01:00 to 2026-07-19 01:00 (UTC)",
		"report_html":           "<h2>Daily summary</h2><p>Requests: 2,374</p>",
	}
	addNotificationEmailOpsSummarySampleVariables(variables, notificationEmailDefaultLocale)
	return variables
}

func addNotificationEmailOpsSummarySampleVariables(variables map[string]string, locale string) {
	variables["report_summary_display"] = "block"
	variables["report_detail_display"] = "none"
	variables["report_insights_display"] = "block"
	if isChineseOpsLocale(locale) {
		variables["report_insights_html"] = "本周期共处理 2,374 次请求，真实可用率 99.86%，整体运行平稳。"
	} else {
		variables["report_insights_html"] = "Service ran smoothly with 99.86% SLA across 2,374 total requests."
	}
	variables["dashboard_url"] = "https://example.com/admin/ops"
	variables["dashboard_cta_display"] = "block"
	variables["report_total_requests"] = "2,374"
	variables["report_success_count"] = "1,451"
	variables["report_sla_error_count"] = "2"
	variables["report_business_limited_count"] = "921"
	variables["report_sla"] = "99.86%"
	variables["report_error_rate"] = "0.14%"
	variables["report_upstream_error_rate"] = "0.28%"
	variables["report_upstream_error_count_excl_429_529"] = "4"
	variables["report_upstream_429_count"] = "0"
	variables["report_upstream_529_count"] = "0"
	variables["report_latency_p50"] = "8,231 ms"
	variables["report_latency_p99"] = "151,260 ms"
	variables["report_ttft_p50"] = "1,674 ms"
	variables["report_ttft_p99"] = "11,222 ms"
	variables["report_tokens"] = "121,550,190"
	variables["report_qps_current"] = "0.0"
	variables["report_qps_peak"] = "1.2"
	variables["report_qps_avg"] = "0.0"
	variables["report_tps_current"] = "0.0"
	variables["report_tps_peak"] = "133421.2"
	variables["report_tps_avg"] = "1406.8"
}

func notificationEmailSampleRawHTML(event, locale string) map[string]string {
	zh := isChineseOpsLocale(locale)
	switch event {
	case NotificationEmailEventOpsAlert:
		metricVal := 12.50
		thresholdVal := 10.00
		sampleEv := &opsAlertEvidence{
			Locale:              locale,
			RuleName:            "错误率过高",
			Severity:            "P1",
			Status:              OpsAlertStatusFiring,
			MetricType:          "error_rate",
			Operator:            ">",
			MetricValue:         &metricVal,
			ThresholdValue:      &thresholdVal,
			WindowMinutes:       5,
			WindowLabel:         formatOpsWindowLabel(5, locale),
			TotalRequests:       48,
			SLARequests:         48,
			SuccessCount:        42,
			ErrorCount:          6,
			AffectedUsers:       2,
			ShowRequestOverview: true,
			TopClusters: []opsAlertErrorCluster{
				{
					Platform:      "openai",
					Model:         "gpt-6.2-sol",
					GroupLabel:    "主力 GPT 分组",
					AccountLabel:  "主力 GPT 账号",
					StatusCode:    524,
					Count:         5,
					Reason:        "上游服务响应超时（524），长时间未返回结果",
					Suggestion:    "检查对应账号的线路或代理延迟；若反复出现，可暂时停用该账号或切换备用线路",
					SampleMessage: "upstream gateway timeout (524)",
				},
			},
		}
		if !zh {
			sampleEv.RuleName = "High error rate"
			sampleEv.TopClusters[0].GroupLabel = "Primary GPT Group"
			sampleEv.TopClusters[0].AccountLabel = "Primary GPT Account"
			sampleEv.TopClusters[0].Reason = "Upstream response timed out (524)"
			sampleEv.TopClusters[0].Suggestion = "Check upstream/proxy latency for Primary GPT Account"
		}
		return map[string]string{
			"alert_details_html": buildOpsAlertDetailsHTML(sampleEv, locale),
		}
	case NotificationEmailEventOpsScheduledReport:
		summary := "过去 24 小时全站共处理 2,374 次请求，真实可用率达到 99.86%，整体表现很稳定。仅发生 2 次可影响用户的失败请求，当前账号池绝大多数账号健康可用。"
		pool := "账号总计 18 个 · 正常可用 17 个 · 限流 1 个 · 过载 0 个 · 异常 0 个｜限流/过载账号：主力 Claude 账号"
		if !zh {
			summary = "Over the last 24h, 2,374 requests were served with 99.86% SLA and only 2 SLA errors. Account pool is healthy."
			pool = "Total 18 accounts · Available 17 · Rate-limited 1 · Overloaded 0 · Error 0 | Rate-limited/Overloaded: Primary Claude Account"
		}
		return map[string]string{
			"report_insights_html": buildOpsReportInsightsHTML(opsDiagnosisBadge(true, true, locale), summary, nil, pool, locale),
		}
	default:
		return map[string]string{}
	}
}

var notificationEmailEventOrder = []string{
	NotificationEmailEventAuthVerifyCode,
	NotificationEmailEventAuthPasswordReset,
	NotificationEmailEventNotificationEmailVerifyCode,
	NotificationEmailEventSubscriptionPurchaseSuccess,
	NotificationEmailEventSubscriptionExpiryReminder,
	NotificationEmailEventBalanceLow,
	NotificationEmailEventBalanceRechargeSuccess,
	NotificationEmailEventAccountQuotaAlert,
	NotificationEmailEventContentModerationViolation,
	NotificationEmailEventContentModerationDisabled,
	NotificationEmailEventCyberPolicyNotice,
	NotificationEmailEventOpsAlert,
	NotificationEmailEventOpsScheduledReport,
}

var notificationEmailEventDefinitions = map[string]NotificationEmailEventInfo{
	NotificationEmailEventAuthVerifyCode: {
		Event:        NotificationEmailEventAuthVerifyCode,
		Label:        "Email verification code",
		Description:  "Sent for registration, email binding, OAuth pending email, and TOTP verification flows.",
		Category:     "auth",
		Optional:     false,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...), "verification_code", "expires_in_minutes"),
	},
	NotificationEmailEventAuthPasswordReset: {
		Event:        NotificationEmailEventAuthPasswordReset,
		Label:        "Password reset",
		Description:  "Sent when a user requests a password reset link.",
		Category:     "auth",
		Optional:     false,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...), "reset_url", "expires_in_minutes"),
	},
	NotificationEmailEventNotificationEmailVerifyCode: {
		Event:        NotificationEmailEventNotificationEmailVerifyCode,
		Label:        "Notification email verification code",
		Description:  "Sent when a user verifies an extra notification email address.",
		Category:     "auth",
		Optional:     false,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...), "verification_code", "expires_in_minutes"),
	},
	NotificationEmailEventSubscriptionPurchaseSuccess: {
		Event:        NotificationEmailEventSubscriptionPurchaseSuccess,
		Label:        "Subscription purchase success",
		Description:  "Sent after a subscription purchase is fulfilled.",
		Category:     "subscription",
		Optional:     false,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...), "subscription_group", "subscription_days", "expiry_time", "order_id"),
	},
	NotificationEmailEventSubscriptionExpiryReminder: {
		Event:        NotificationEmailEventSubscriptionExpiryReminder,
		Label:        "Subscription expiry reminder",
		Description:  "Optional reminder sent before an active subscription expires.",
		Category:     "subscription",
		Optional:     true,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...), "subscription_group", "expiry_time", "days_remaining", "unsubscribe_url"),
	},
	NotificationEmailEventBalanceLow: {
		Event:        NotificationEmailEventBalanceLow,
		Label:        "Low balance alert",
		Description:  "Optional alert sent when balance crosses the configured low-balance threshold.",
		Category:     "billing",
		Optional:     true,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...), "current_balance", "threshold", "recharge_url", "unsubscribe_url"),
	},
	NotificationEmailEventBalanceRechargeSuccess: {
		Event:        NotificationEmailEventBalanceRechargeSuccess,
		Label:        "Balance recharge success",
		Description:  "Sent after a balance recharge order is fulfilled.",
		Category:     "billing",
		Optional:     false,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...), "recharge_amount", "current_balance", "order_id"),
	},
	NotificationEmailEventAccountQuotaAlert: {
		Event:       NotificationEmailEventAccountQuotaAlert,
		Label:       "Account quota alert",
		Description: "Sent to configured admin notification emails when an upstream account quota threshold is crossed.",
		Category:    "admin",
		Optional:    false,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...),
			"account_id", "account_name", "platform", "quota_dimension", "quota_used", "quota_limit", "quota_remaining", "quota_threshold"),
	},
	NotificationEmailEventContentModerationViolation: {
		Event:       NotificationEmailEventContentModerationViolation,
		Label:       "Risk control violation notice",
		Description: "Sent to users when a request triggers content moderation/risk control rules.",
		Category:    "risk_control",
		Optional:    false,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...),
			"triggered_at", "group_name", "moderation_category", "moderation_score", "violation_count", "ban_threshold"),
	},
	NotificationEmailEventContentModerationDisabled: {
		Event:       NotificationEmailEventContentModerationDisabled,
		Label:       "Risk control account disabled",
		Description: "Sent to users when content moderation automatically disables their account.",
		Category:    "risk_control",
		Optional:    false,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...),
			"triggered_at", "group_name", "moderation_category", "moderation_score", "violation_count", "ban_threshold"),
	},
	NotificationEmailEventCyberPolicyNotice: {
		Event:       NotificationEmailEventCyberPolicyNotice,
		Label:       "Cyber policy notice",
		Description: "Sent to users when an upstream request is blocked by cyber-security policy (cyber_policy).",
		Category:    "risk_control",
		Optional:    false,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...),
			"triggered_at", "model", "group_name", "upstream_message"),
	},
	NotificationEmailEventOpsAlert: {
		Event:       NotificationEmailEventOpsAlert,
		Label:       "Ops alert",
		Description: "Sent to configured operations recipients when an ops alert rule fires or recovers.",
		Category:    "ops",
		Optional:    false,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...),
			"rule_name", "severity", "severity_label", "alert_status", "status_label",
			"metric_type", "metric_label", "operator", "metric_value", "threshold_value", "metric_summary",
			"triggered_at", "triggered_at_local", "window_label", "duration_human",
			"alert_description", "diagnosis_badge", "diagnosis_summary", "alert_details_html",
			"dashboard_url", "dashboard_cta_display", "header_color", "email_title", "alert_headline"),
	},
	NotificationEmailEventOpsScheduledReport: {
		Event:       NotificationEmailEventOpsScheduledReport,
		Label:       "Ops scheduled report",
		Description: "Sent to configured operations recipients for scheduled daily/weekly/error/account-health reports.",
		Category:    "ops",
		Optional:    false,
		Placeholders: append(
			append(
				append([]string{}, notificationEmailCommonPlaceholders...),
				"report_name", "report_type", "report_start_time", "report_end_time", "report_period_display",
			),
			append(append([]string{}, notificationEmailOpsSummaryPlaceholders...), "report_detail_display", "report_html", "report_insights_display", "report_insights_html", "dashboard_url", "dashboard_cta_display")...,
		),
	},
}

var notificationEmailOfficialTemplates = map[string]map[string]notificationEmailOfficialTemplate{
	NotificationEmailEventAuthVerifyCode: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] Email verification code",
			HTML: notificationEmailCard("#4f46e5", "Email verification code", `
<p>Hello {{recipient_name}},</p>
<p>Your verification code is:</p>
<p style="font-size: 32px; font-weight: 700; letter-spacing: 8px; text-align: center;">{{verification_code}}</p>
<p>This code expires in <strong>{{expires_in_minutes}}</strong> minutes.</p>
<p>If you did not request this code, please ignore this email.</p>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] 邮箱验证码",
			HTML: notificationEmailCard("#4f46e5", "邮箱验证码", `
<p>{{recipient_name}}，您好：</p>
<p>您的验证码是：</p>
<p style="font-size: 32px; font-weight: 700; letter-spacing: 8px; text-align: center;">{{verification_code}}</p>
<p>验证码将在 <strong>{{expires_in_minutes}}</strong> 分钟后失效。</p>
<p>如果不是您本人操作，请忽略此邮件。</p>`),
		},
	},
	NotificationEmailEventAuthPasswordReset: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] Password reset request",
			HTML: notificationEmailCard("#7c3aed", "Password reset", `
<p>Hello {{recipient_name}},</p>
<p>We received a request to reset your password. Click the button below to set a new password.</p>
<p><a class="button" href="{{reset_url}}">Reset password</a></p>
<p>This link expires in <strong>{{expires_in_minutes}}</strong> minutes.</p>
<p class="muted">If the button does not work, copy this link into your browser:<br>{{reset_url}}</p>
<p>If you did not request this, you can safely ignore this email.</p>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] 密码重置请求",
			HTML: notificationEmailCard("#7c3aed", "密码重置", `
<p>{{recipient_name}}，您好：</p>
<p>我们收到了您的密码重置请求，请点击下方按钮设置新密码。</p>
<p><a class="button" href="{{reset_url}}">重置密码</a></p>
<p>此链接将在 <strong>{{expires_in_minutes}}</strong> 分钟后失效。</p>
<p class="muted">如果按钮无法点击，请复制以下链接到浏览器中打开：<br>{{reset_url}}</p>
<p>如果不是您本人操作，请忽略此邮件。</p>`),
		},
	},
	NotificationEmailEventNotificationEmailVerifyCode: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] Notification email verification code",
			HTML: notificationEmailCard("#0ea5e9", "Notification email verification", `
<p>Hello {{recipient_name}},</p>
<p>You are adding this address as an extra notification email.</p>
<p>Your verification code is:</p>
<p style="font-size: 32px; font-weight: 700; letter-spacing: 8px; text-align: center;">{{verification_code}}</p>
<p>This code expires in <strong>{{expires_in_minutes}}</strong> minutes.</p>
<p>If you did not request this code, please ignore this email.</p>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] 通知邮箱验证码",
			HTML: notificationEmailCard("#0ea5e9", "通知邮箱验证", `
<p>{{recipient_name}}，您好：</p>
<p>您正在添加额外的通知邮箱，请输入以下验证码完成验证。</p>
<p style="font-size: 32px; font-weight: 700; letter-spacing: 8px; text-align: center;">{{verification_code}}</p>
<p>验证码将在 <strong>{{expires_in_minutes}}</strong> 分钟后失效。</p>
<p>如果不是您本人操作，请忽略此邮件。</p>`),
		},
	},
	NotificationEmailEventSubscriptionPurchaseSuccess: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] Subscription purchase successful",
			HTML: notificationEmailCard("#2563eb", "Subscription activated", `
<p>Hello {{recipient_name}},</p>
<p>Your subscription for <strong>{{subscription_group}}</strong> has been activated for <strong>{{subscription_days}}</strong> days.</p>
<p>Expiry time: <strong>{{expiry_time}}</strong></p>
<p>Order ID: {{order_id}}</p>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] 订阅购买成功",
			HTML: notificationEmailCard("#2563eb", "订阅已开通", `
<p>{{recipient_name}}，您好：</p>
<p>您的 <strong>{{subscription_group}}</strong> 订阅已成功开通，有效期 <strong>{{subscription_days}}</strong> 天。</p>
<p>到期时间：<strong>{{expiry_time}}</strong></p>
<p>订单号：{{order_id}}</p>`),
		},
	},
	NotificationEmailEventSubscriptionExpiryReminder: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] Subscription expires in {{days_remaining}} day(s)",
			HTML: notificationEmailCard("#f97316", "Subscription expiry reminder", `
<p>Hello {{recipient_name}},</p>
<p>Your <strong>{{subscription_group}}</strong> subscription will expire in <strong>{{days_remaining}}</strong> day(s).</p>
<p>Expiry time: <strong>{{expiry_time}}</strong></p>
<p class="muted"><a href="{{unsubscribe_url}}">Unsubscribe from optional subscription reminders</a></p>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] 订阅将在 {{days_remaining}} 天后到期",
			HTML: notificationEmailCard("#f97316", "订阅到期提醒", `
<p>{{recipient_name}}，您好：</p>
<p>您的 <strong>{{subscription_group}}</strong> 订阅将在 <strong>{{days_remaining}}</strong> 天后到期。</p>
<p>到期时间：<strong>{{expiry_time}}</strong></p>
<p class="muted"><a href="{{unsubscribe_url}}">退订此类订阅提醒</a></p>`),
		},
	},
	NotificationEmailEventBalanceLow: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] Low balance alert",
			HTML: notificationEmailCard("#d97706", "Low balance alert", `
<p>Hello {{recipient_name}},</p>
<p>Your current balance is <strong>${{current_balance}}</strong>, below the configured alert threshold of <strong>${{threshold}}</strong>.</p>
<p>Please recharge in time to avoid service interruption.</p>
<p><a class="button" href="{{recharge_url}}">Recharge now</a></p>
<p class="muted"><a href="{{unsubscribe_url}}">Unsubscribe from optional balance alerts</a></p>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] 余额不足提醒",
			HTML: notificationEmailCard("#d97706", "余额不足提醒", `
<p>{{recipient_name}}，您好：</p>
<p>您当前余额为 <strong>${{current_balance}}</strong>，已低于提醒阈值 <strong>${{threshold}}</strong>。</p>
<p>请及时充值以免服务中断。</p>
<p><a class="button" href="{{recharge_url}}">立即充值</a></p>
<p class="muted"><a href="{{unsubscribe_url}}">退订此类余额提醒</a></p>`),
		},
	},
	NotificationEmailEventBalanceRechargeSuccess: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] Balance recharge successful",
			HTML: notificationEmailCard("#16a34a", "Recharge successful", `
<p>Hello {{recipient_name}},</p>
<p>Your balance recharge of <strong>${{recharge_amount}}</strong> has been completed.</p>
<p>Current balance: <strong>${{current_balance}}</strong></p>
<p>Order ID: {{order_id}}</p>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] 余额充值成功",
			HTML: notificationEmailCard("#16a34a", "余额充值成功", `
<p>{{recipient_name}}，您好：</p>
<p>您的余额充值 <strong>${{recharge_amount}}</strong> 已完成。</p>
<p>当前余额：<strong>${{current_balance}}</strong></p>
			<p>订单号：{{order_id}}</p>`),
		},
	},
	NotificationEmailEventAccountQuotaAlert: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] Account quota alert - {{account_name}}",
			HTML: notificationEmailCard("#dc2626", "Account quota alert", `
<p>The upstream account <strong>{{account_name}}</strong> has crossed its configured quota alert threshold.</p>
<table style="width:100%;border-collapse:collapse;">
  <tr><td>Account ID</td><td>{{account_id}}</td></tr>
  <tr><td>Platform</td><td>{{platform}}</td></tr>
  <tr><td>Dimension</td><td>{{quota_dimension}}</td></tr>
  <tr><td>Used / Limit</td><td>{{quota_used}} / {{quota_limit}}</td></tr>
  <tr><td>Remaining</td><td>{{quota_remaining}}</td></tr>
  <tr><td>Threshold</td><td>{{quota_threshold}}</td></tr>
</table>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] 账号限额告警 - {{account_name}}",
			HTML: notificationEmailCard("#dc2626", "账号限额告警", `
<p>上游账号 <strong>{{account_name}}</strong> 已触发配置的额度告警阈值。</p>
<table style="width:100%;border-collapse:collapse;">
  <tr><td>账号 ID</td><td>{{account_id}}</td></tr>
  <tr><td>平台</td><td>{{platform}}</td></tr>
  <tr><td>维度</td><td>{{quota_dimension}}</td></tr>
  <tr><td>已用 / 限额</td><td>{{quota_used}} / {{quota_limit}}</td></tr>
  <tr><td>剩余额度</td><td>{{quota_remaining}}</td></tr>
  <tr><td>告警阈值</td><td>{{quota_threshold}}</td></tr>
</table>`),
		},
	},
	NotificationEmailEventContentModerationViolation: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] Risk control notice",
			HTML: notificationEmailCard("#ef4444", "Risk control notice", `
<p>Hello {{recipient_name}},</p>
<p>Your API request triggered the platform content moderation/risk-control policy.</p>
<table style="width:100%;border-collapse:collapse;">
  <tr><td>Triggered at</td><td>{{triggered_at}}</td></tr>
  <tr><td>Group</td><td>{{group_name}}</td></tr>
  <tr><td>Category / Score</td><td>{{moderation_category}} / {{moderation_score}}</td></tr>
  <tr><td>Violation count</td><td>{{violation_count}} / {{ban_threshold}}</td></tr>
</table>
<p>Please review your request content to avoid future service interruptions.</p>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] 账户风控提醒",
			HTML: notificationEmailCard("#ef4444", "账户风控提醒", `
<p>{{recipient_name}}，您好：</p>
<p>您的 API 请求触发了平台内容审核/风控策略。</p>
<table style="width:100%;border-collapse:collapse;">
  <tr><td>触发时间</td><td>{{triggered_at}}</td></tr>
  <tr><td>所属分组</td><td>{{group_name}}</td></tr>
  <tr><td>命中类别 / 分数</td><td>{{moderation_category}} / {{moderation_score}}</td></tr>
  <tr><td>累计触发次数</td><td>{{violation_count}} / {{ban_threshold}}</td></tr>
</table>
<p>请检查请求内容，避免后续服务受到影响。</p>`),
		},
	},
	NotificationEmailEventContentModerationDisabled: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] Account disabled by risk control",
			HTML: notificationEmailCard("#b91c1c", "Account disabled", `
<p>Hello {{recipient_name}},</p>
<p>Your account has repeatedly triggered platform content moderation/risk-control rules and has been automatically disabled.</p>
<table style="width:100%;border-collapse:collapse;">
  <tr><td>Disabled at</td><td>{{triggered_at}}</td></tr>
  <tr><td>Group</td><td>{{group_name}}</td></tr>
  <tr><td>Category / Score</td><td>{{moderation_category}} / {{moderation_score}}</td></tr>
  <tr><td>Violation count</td><td>{{violation_count}} / {{ban_threshold}}</td></tr>
</table>
<p>Please contact the administrator if you need to appeal or restore access.</p>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] 账户已被禁用",
			HTML: notificationEmailCard("#b91c1c", "账户已被禁用", `
<p>{{recipient_name}}，您好：</p>
<p>您的账户在统计周期内多次触发平台内容审核/风控规则，系统已自动禁用该账户。</p>
<table style="width:100%;border-collapse:collapse;">
  <tr><td>禁用时间</td><td>{{triggered_at}}</td></tr>
  <tr><td>所属分组</td><td>{{group_name}}</td></tr>
  <tr><td>命中类别 / 分数</td><td>{{moderation_category}} / {{moderation_score}}</td></tr>
  <tr><td>累计触发次数</td><td>{{violation_count}} / {{ban_threshold}}</td></tr>
</table>
<p>如需申诉或恢复账号，请联系平台管理员处理。</p>`),
		},
	},
	NotificationEmailEventCyberPolicyNotice: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] Cyber-security policy notice",
			HTML: notificationEmailCard("#ef4444", "Cyber-security policy notice", `
<p>Hello {{recipient_name}},</p>
<p>Your request was blocked by the upstream provider's cyber-security policy.</p>
<table style="width:100%;border-collapse:collapse;table-layout:fixed;">
  <tr><td style="width:128px;vertical-align:top;">Triggered at</td><td style="overflow-wrap:anywhere;word-break:break-word;">{{triggered_at}}</td></tr>
  <tr><td style="width:128px;vertical-align:top;">Model</td><td style="overflow-wrap:anywhere;word-break:break-word;">{{model}}</td></tr>
  <tr><td style="width:128px;vertical-align:top;">Group</td><td style="overflow-wrap:anywhere;word-break:break-word;">{{group_name}}</td></tr>
  <tr><td style="width:128px;vertical-align:top;">Upstream message</td><td style="overflow-wrap:anywhere;word-break:break-all;white-space:pre-wrap;">{{upstream_message}}</td></tr>
</table>
<p>If you believe this is a mistake, try rephrasing your request, or apply for authorized security access.</p>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] 网络安全策略拦截提醒",
			HTML: notificationEmailCard("#ef4444", "网络安全策略拦截提醒", `
<p>{{recipient_name}}，您好：</p>
<p>您的请求被上游服务商的网络安全策略（cyber policy）拦截。</p>
<table style="width:100%;border-collapse:collapse;table-layout:fixed;">
  <tr><td style="width:128px;vertical-align:top;">触发时间</td><td style="overflow-wrap:anywhere;word-break:break-word;">{{triggered_at}}</td></tr>
  <tr><td style="width:128px;vertical-align:top;">模型</td><td style="overflow-wrap:anywhere;word-break:break-word;">{{model}}</td></tr>
  <tr><td style="width:128px;vertical-align:top;">所属分组</td><td style="overflow-wrap:anywhere;word-break:break-word;">{{group_name}}</td></tr>
  <tr><td style="width:128px;vertical-align:top;">上游说明</td><td style="overflow-wrap:anywhere;word-break:break-all;white-space:pre-wrap;">{{upstream_message}}</td></tr>
</table>
<p>如认为系误判，可调整请求措辞后重试，或申请获得授权的安全访问权限。</p>`),
		},
	},
	NotificationEmailEventOpsAlert: {
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}} {{email_title}}] {{alert_headline}}",
			HTML:    notificationEmailOpsAlertTemplate(notificationEmailDefaultLocale),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}} {{email_title}}] {{alert_headline}}",
			HTML:    notificationEmailOpsAlertTemplate(notificationEmailLocaleChinese),
		},
	},
	NotificationEmailEventOpsScheduledReport: {
		notificationEmailDefaultLocale: {
			Subject: "[Ops Report] {{report_name}}",
			HTML:    notificationEmailOpsScheduledReportTemplate(notificationEmailDefaultLocale),
		},
		notificationEmailLocaleChinese: {
			Subject: "[运维报表] {{report_name}}",
			HTML:    notificationEmailOpsScheduledReportTemplate(notificationEmailLocaleChinese),
		},
	},
}

func notificationEmailOpsAlertTemplate(locale string) string {
	if normalizeNotificationLocale(locale) == notificationEmailLocaleChinese {
		return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { margin: 0; padding: 20px 12px; background: #f4f6f8; color: #1f2937; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif; }
    .container { width: 100%; max-width: 660px; margin: 0 auto; background: #ffffff; border: 1px solid #e2e8f0; border-radius: 10px; overflow: hidden; }
    .header { padding: 24px 28px; background: {{header_color}}; color: #ffffff; }
    .eyebrow { margin: 0 0 6px; font-size: 12px; font-weight: 700; opacity: 0.92; }
    h1 { margin: 0; font-size: 21px; line-height: 1.35; }
    .content { padding: 24px 28px 28px; font-size: 14px; line-height: 1.7; }
    .diag-box { margin: 0 0 18px; padding: 14px 16px; border-radius: 8px; background: #f8fafc; border-left: 4px solid {{header_color}}; }
    .diag-badge { font-size: 12px; font-weight: 700; color: {{header_color}}; margin-bottom: 4px; }
    .diag-text { font-size: 14px; color: #0f172a; line-height: 1.65; }
    .kv { width: 100%; border-collapse: collapse; margin: 0 0 16px; font-size: 13px; }
    .kv td { padding: 9px 4px; border-bottom: 1px solid #f1f5f9; vertical-align: top; }
    .kv tr:last-child td { border-bottom: 0; }
    .kv-label { width: 108px; color: #64748b; font-weight: 600; }
    .kv-val { color: #0f172a; }
    .cta-wrap { margin-top: 20px; }
    .cta-btn { display: inline-block; padding: 11px 20px; border-radius: 8px; background: {{header_color}}; color: #ffffff !important; text-decoration: none; font-weight: 600; font-size: 14px; }
    .footer { padding: 16px 28px; background: #f8fafc; border-top: 1px solid #e2e8f0; color: #64748b; font-size: 12px; line-height: 1.6; }
    @media only screen and (max-width: 620px) {
      body { padding: 0; }
      .container { border: 0; border-radius: 0; }
      .header, .content, .footer { padding-left: 18px; padding-right: 18px; }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <p class="eyebrow">{{site_name}} 运维监控 · {{email_title}}</p>
      <h1>{{alert_headline}}</h1>
    </div>
    <div class="content">
      <div class="diag-box">
        <div class="diag-badge">{{diagnosis_badge}}</div>
        <div class="diag-text">{{diagnosis_summary}}</div>
      </div>
      <table class="kv" role="presentation">
        <tr><td class="kv-label">告警规则</td><td class="kv-val"><strong>{{rule_name}}</strong></td></tr>
        <tr><td class="kv-label">严重程度</td><td class="kv-val">{{severity_label}}</td></tr>
        <tr><td class="kv-label">当前状态</td><td class="kv-val">{{status_label}}</td></tr>
        <tr><td class="kv-label">触发指标</td><td class="kv-val">{{metric_summary}}</td></tr>
        <tr><td class="kv-label">触发时间</td><td class="kv-val">{{triggered_at_local}}</td></tr>
        <tr><td class="kv-label">窗口 / 持续</td><td class="kv-val">{{window_label}} · 持续 {{duration_human}}</td></tr>
      </table>
      <div>{{alert_details_html}}</div>
      <div class="cta-wrap" style="display: {{dashboard_cta_display}};">
        <a class="cta-btn" href="{{dashboard_url}}">打开后台运维监控</a>
      </div>
    </div>
    <div class="footer">此邮件由 {{site_name}} 自动发送；若对噪音规则或恢复通知有调整需求，可在后台「运维监控 → 邮件通知 / 告警规则」修改。</div>
  </div>
</body>
</html>`
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { margin: 0; padding: 20px 12px; background: #f4f6f8; color: #1f2937; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    .container { width: 100%; max-width: 660px; margin: 0 auto; background: #ffffff; border: 1px solid #e2e8f0; border-radius: 10px; overflow: hidden; }
    .header { padding: 24px 28px; background: {{header_color}}; color: #ffffff; }
    .eyebrow { margin: 0 0 6px; font-size: 12px; font-weight: 700; opacity: 0.92; }
    h1 { margin: 0; font-size: 21px; line-height: 1.35; }
    .content { padding: 24px 28px 28px; font-size: 14px; line-height: 1.7; }
    .diag-box { margin: 0 0 18px; padding: 14px 16px; border-radius: 8px; background: #f8fafc; border-left: 4px solid {{header_color}}; }
    .diag-badge { font-size: 12px; font-weight: 700; color: {{header_color}}; margin-bottom: 4px; }
    .diag-text { font-size: 14px; color: #0f172a; line-height: 1.65; }
    .kv { width: 100%; border-collapse: collapse; margin: 0 0 16px; font-size: 13px; }
    .kv td { padding: 9px 4px; border-bottom: 1px solid #f1f5f9; vertical-align: top; }
    .kv tr:last-child td { border-bottom: 0; }
    .kv-label { width: 120px; color: #64748b; font-weight: 600; }
    .kv-val { color: #0f172a; }
    .cta-wrap { margin-top: 20px; }
    .cta-btn { display: inline-block; padding: 11px 20px; border-radius: 8px; background: {{header_color}}; color: #ffffff !important; text-decoration: none; font-weight: 600; font-size: 14px; }
    .footer { padding: 16px 28px; background: #f8fafc; border-top: 1px solid #e2e8f0; color: #64748b; font-size: 12px; line-height: 1.6; }
    @media only screen and (max-width: 620px) {
      body { padding: 0; }
      .container { border: 0; border-radius: 0; }
      .header, .content, .footer { padding-left: 18px; padding-right: 18px; }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <p class="eyebrow">{{site_name}} Ops Monitor · {{email_title}}</p>
      <h1>{{alert_headline}}</h1>
    </div>
    <div class="content">
      <div class="diag-box">
        <div class="diag-badge">{{diagnosis_badge}}</div>
        <div class="diag-text">{{diagnosis_summary}}</div>
      </div>
      <table class="kv" role="presentation">
        <tr><td class="kv-label">Rule</td><td class="kv-val"><strong>{{rule_name}}</strong></td></tr>
        <tr><td class="kv-label">Severity</td><td class="kv-val">{{severity_label}}</td></tr>
        <tr><td class="kv-label">Status</td><td class="kv-val">{{status_label}}</td></tr>
        <tr><td class="kv-label">Metric</td><td class="kv-val">{{metric_summary}}</td></tr>
        <tr><td class="kv-label">Triggered at</td><td class="kv-val">{{triggered_at_local}}</td></tr>
        <tr><td class="kv-label">Window / Duration</td><td class="kv-val">{{window_label}} · {{duration_human}}</td></tr>
      </table>
      <div>{{alert_details_html}}</div>
      <div class="cta-wrap" style="display: {{dashboard_cta_display}};">
        <a class="cta-btn" href="{{dashboard_url}}">Open Ops Dashboard</a>
      </div>
    </div>
    <div class="footer">Sent automatically by {{site_name}}. Please do not reply directly.</div>
  </div>
</body>
</html>`
}

func notificationEmailOpsScheduledReportTemplate(locale string) string {
	if normalizeNotificationLocale(locale) == notificationEmailLocaleChinese {
		return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { margin: 0; padding: 24px 12px; background: #f4f6f8; color: #1f2937; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif; }
    .container { width: 100%; max-width: 680px; margin: 0 auto; background: #ffffff; border: 1px solid #dfe7ea; border-radius: 8px; overflow: hidden; }
    .header { padding: 28px 32px 24px; background: #0f766e; color: #ffffff; }
    .eyebrow { margin: 0 0 8px; color: #ccfbf1; font-size: 12px; font-weight: 700; letter-spacing: 0; text-transform: uppercase; }
    h1 { margin: 0; font-size: 26px; line-height: 1.3; }
    .header p { margin: 8px 0 0; color: #e6fffb; font-size: 14px; }
    .content { padding: 28px 32px 32px; }
    .meta { width: 100%; margin: 0 0 20px; border-collapse: collapse; background: #f8fafc; border: 1px solid #e2e8f0; }
    .meta td { padding: 10px 12px; border-bottom: 1px solid #e2e8f0; font-size: 13px; vertical-align: top; }
    .meta tr:last-child td { border-bottom: 0; }
    .meta-label { width: 112px; color: #64748b; font-weight: 600; }
    .section-title { margin: 28px 0 12px; color: #0f172a; font-size: 16px; line-height: 1.4; }
    .metric-grid { width: 100%; border-collapse: separate; border-spacing: 8px; margin: -8px; }
    .metric-cell { width: 50%; padding: 14px 16px; border: 1px solid #e2e8f0; background: #ffffff; vertical-align: top; }
    .metric-label { display: block; color: #64748b; font-size: 12px; line-height: 1.4; }
    .metric-value { display: block; margin-top: 6px; color: #0f172a; font-size: 20px; font-weight: 700; line-height: 1.2; }
    .metric-value.good { color: #15803d; }
    .metric-value.alert { color: #b91c1c; }
    .detail { width: 100%; border-collapse: collapse; }
    .detail td { padding: 10px 12px; border-bottom: 1px solid #e2e8f0; font-size: 13px; }
    .detail td:first-child { width: 56%; color: #475569; }
    .detail td:last-child { color: #0f172a; font-weight: 600; text-align: right; }
    .report-detail { margin-top: 28px; }
    .report-detail:empty { display: none; }
    .footer { padding: 18px 32px; background: #f8fafc; border-top: 1px solid #e2e8f0; color: #64748b; font-size: 12px; line-height: 1.6; }
    @media only screen and (max-width: 620px) {
      body { padding: 0; }
      .container { border: 0; border-radius: 0; }
      .header, .content, .footer { padding-left: 20px; padding-right: 20px; }
      .metric-grid, .metric-grid tbody, .metric-grid tr, .metric-cell { display: block; width: 100% !important; box-sizing: border-box; }
      .metric-cell { margin: 8px 0; }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <p class="eyebrow">运维报表</p>
      <h1>{{report_name}}</h1>
      <p>{{site_name}} 的运行概览</p>
    </div>
    <div class="content">
      <table class="meta" role="presentation">
        <tr><td class="meta-label">报表名称</td><td>{{report_name}}</td></tr>
        <tr><td class="meta-label">统计周期</td><td>{{report_period_display}}</td></tr>
      </table>

      <div style="display: {{report_insights_display}}; margin-bottom: 22px;">{{report_insights_html}}</div>

      <div style="display: {{report_summary_display}};">
      <h2 class="section-title">请求概览</h2>
      <table class="metric-grid" role="presentation"><tr>
        <td class="metric-cell"><span class="metric-label">总请求数</span><span class="metric-value">{{report_total_requests}}</span></td>
        <td class="metric-cell"><span class="metric-label">成功请求</span><span class="metric-value good">{{report_success_count}}</span></td>
      </tr><tr>
        <td class="metric-cell"><span class="metric-label">影响用户失败（SLA）</span><span class="metric-value alert">{{report_sla_error_count}}</span></td>
        <td class="metric-cell"><span class="metric-label">业务限流（额度/限频拦截）</span><span class="metric-value">{{report_business_limited_count}}</span></td>
      </tr></table>

      <h2 class="section-title">可靠性</h2>
      <table class="detail" role="presentation">
        <tr><td>SLA 真实可用率</td><td>{{report_sla}}</td></tr>
        <tr><td>整体错误率</td><td>{{report_error_rate}}</td></tr>
        <tr><td>上游错误率（不含 429 / 529）</td><td>{{report_upstream_error_rate}}</td></tr>
        <tr><td>上游错误次数（不含 429 / 529）</td><td>{{report_upstream_error_count_excl_429_529}}</td></tr>
        <tr><td>上游 429 限流 / 529 过载</td><td>{{report_upstream_429_count}} / {{report_upstream_529_count}}</td></tr>
      </table>

      <h2 class="section-title">响应速度</h2>
      <table class="detail" role="presentation">
        <tr><td>请求总耗时（中位 p50 / 慢请求 p99）</td><td>{{report_latency_p50}} / {{report_latency_p99}}</td></tr>
        <tr><td>首字响应延迟（p50 / p99）</td><td>{{report_ttft_p50}} / {{report_ttft_p99}}</td></tr>
      </table>

      <h2 class="section-title">吞吐量</h2>
      <table class="detail" role="presentation">
        <tr><td>Token 消耗总量</td><td>{{report_tokens}}</td></tr>
        <tr><td>每秒请求数 QPS（当前 / 峰值 / 平均）</td><td>{{report_qps_current}} / {{report_qps_peak}} / {{report_qps_avg}}</td></tr>
        <tr><td>每秒 Token 数 TPS（当前 / 峰值 / 平均）</td><td>{{report_tps_current}} / {{report_tps_peak}} / {{report_tps_avg}}</td></tr>
      </table>

      </div>
      <div class="report-detail" style="display: {{report_detail_display}};">{{report_html}}</div>
      <div style="display: {{dashboard_cta_display}}; margin-top: 24px;">
        <a href="{{dashboard_url}}" style="display: inline-block; padding: 11px 20px; border-radius: 8px; background: #0f766e; color: #ffffff; text-decoration: none; font-weight: 600; font-size: 14px;">打开后台运维监控</a>
      </div>
    </div>
    <div class="footer">此邮件由 {{site_name}} 自动发送，请勿直接回复。</div>
  </div>
</body>
</html>`
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { margin: 0; padding: 24px 12px; background: #f4f6f8; color: #1f2937; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    .container { width: 100%; max-width: 680px; margin: 0 auto; background: #ffffff; border: 1px solid #dfe7ea; border-radius: 8px; overflow: hidden; }
    .header { padding: 28px 32px 24px; background: #0f766e; color: #ffffff; }
    .eyebrow { margin: 0 0 8px; color: #ccfbf1; font-size: 12px; font-weight: 700; letter-spacing: 0; text-transform: uppercase; }
    h1 { margin: 0; font-size: 26px; line-height: 1.3; }
    .header p { margin: 8px 0 0; color: #e6fffb; font-size: 14px; }
    .content { padding: 28px 32px 32px; }
    .meta { width: 100%; margin: 0 0 20px; border-collapse: collapse; background: #f8fafc; border: 1px solid #e2e8f0; }
    .meta td { padding: 10px 12px; border-bottom: 1px solid #e2e8f0; font-size: 13px; vertical-align: top; }
    .meta tr:last-child td { border-bottom: 0; }
    .meta-label { width: 112px; color: #64748b; font-weight: 600; }
    .section-title { margin: 28px 0 12px; color: #0f172a; font-size: 16px; line-height: 1.4; }
    .metric-grid { width: 100%; border-collapse: separate; border-spacing: 8px; margin: -8px; }
    .metric-cell { width: 50%; padding: 14px 16px; border: 1px solid #e2e8f0; background: #ffffff; vertical-align: top; }
    .metric-label { display: block; color: #64748b; font-size: 12px; line-height: 1.4; }
    .metric-value { display: block; margin-top: 6px; color: #0f172a; font-size: 20px; font-weight: 700; line-height: 1.2; }
    .metric-value.good { color: #15803d; }
    .metric-value.alert { color: #b91c1c; }
    .detail { width: 100%; border-collapse: collapse; }
    .detail td { padding: 10px 12px; border-bottom: 1px solid #e2e8f0; font-size: 13px; }
    .detail td:first-child { width: 56%; color: #475569; }
    .detail td:last-child { color: #0f172a; font-weight: 600; text-align: right; }
    .report-detail { margin-top: 28px; }
    .report-detail:empty { display: none; }
    .footer { padding: 18px 32px; background: #f8fafc; border-top: 1px solid #e2e8f0; color: #64748b; font-size: 12px; line-height: 1.6; }
    @media only screen and (max-width: 620px) {
      body { padding: 0; }
      .container { border: 0; border-radius: 0; }
      .header, .content, .footer { padding-left: 20px; padding-right: 20px; }
      .metric-grid, .metric-grid tbody, .metric-grid tr, .metric-cell { display: block; width: 100% !important; box-sizing: border-box; }
      .metric-cell { margin: 8px 0; }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <p class="eyebrow">Operations report</p>
      <h1>{{report_name}}</h1>
      <p>{{site_name}} runtime overview</p>
    </div>
    <div class="content">
      <table class="meta" role="presentation">
        <tr><td class="meta-label">Report</td><td>{{report_name}}</td></tr>
        <tr><td class="meta-label">Reporting period</td><td>{{report_period_display}}</td></tr>
      </table>

      <div style="display: {{report_insights_display}}; margin-bottom: 22px;">{{report_insights_html}}</div>

      <div style="display: {{report_summary_display}};">
      <h2 class="section-title">Request Overview</h2>
      <table class="metric-grid" role="presentation"><tr>
        <td class="metric-cell"><span class="metric-label">Total Requests</span><span class="metric-value">{{report_total_requests}}</span></td>
        <td class="metric-cell"><span class="metric-label">Successful Requests</span><span class="metric-value good">{{report_success_count}}</span></td>
      </tr><tr>
        <td class="metric-cell"><span class="metric-label">SLA Errors</span><span class="metric-value alert">{{report_sla_error_count}}</span></td>
        <td class="metric-cell"><span class="metric-label">Business Limited</span><span class="metric-value">{{report_business_limited_count}}</span></td>
      </tr></table>

      <h2 class="section-title">Reliability</h2>
      <table class="detail" role="presentation">
        <tr><td>SLA</td><td>{{report_sla}}</td></tr>
        <tr><td>Error Rate</td><td>{{report_error_rate}}</td></tr>
        <tr><td>Upstream Error Rate (excluding 429 / 529)</td><td>{{report_upstream_error_rate}}</td></tr>
        <tr><td>Upstream Errors (excluding 429 / 529)</td><td>{{report_upstream_error_count_excl_429_529}}</td></tr>
        <tr><td>Upstream 429 / 529</td><td>{{report_upstream_429_count}} / {{report_upstream_529_count}}</td></tr>
      </table>

      <h2 class="section-title">Latency</h2>
      <table class="detail" role="presentation">
        <tr><td>Request Latency p50 / p99</td><td>{{report_latency_p50}} / {{report_latency_p99}}</td></tr>
        <tr><td>Time to First Token p50 / p99</td><td>{{report_ttft_p50}} / {{report_ttft_p99}}</td></tr>
      </table>

      <h2 class="section-title">Throughput</h2>
      <table class="detail" role="presentation">
        <tr><td>Tokens Consumed</td><td>{{report_tokens}}</td></tr>
        <tr><td>QPS (current / peak / average)</td><td>{{report_qps_current}} / {{report_qps_peak}} / {{report_qps_avg}}</td></tr>
        <tr><td>TPS (current / peak / average)</td><td>{{report_tps_current}} / {{report_tps_peak}} / {{report_tps_avg}}</td></tr>
      </table>

      </div>
      <div class="report-detail" style="display: {{report_detail_display}};">{{report_html}}</div>
      <div style="display: {{dashboard_cta_display}}; margin-top: 24px;">
        <a href="{{dashboard_url}}" style="display: inline-block; padding: 11px 20px; border-radius: 8px; background: #0f766e; color: #ffffff; text-decoration: none; font-weight: 600; font-size: 14px;">Open Ops Dashboard</a>
      </div>
    </div>
    <div class="footer">This email was sent automatically by {{site_name}}. Please do not reply directly.</div>
  </div>
</body>
</html>`
}

func notificationEmailCard(accent, title, content string) string {
	return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { margin: 0; padding: 24px; background: #f4f4f5; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color: #18181b; }
    .container { max-width: 640px; margin: 0 auto; background: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 8px 30px rgba(15, 23, 42, 0.10); }
    .header { background: ` + accent + `; color: #ffffff; padding: 28px 32px; }
    .header h1 { margin: 0; font-size: 24px; line-height: 1.25; }
    .content { padding: 32px; font-size: 15px; line-height: 1.7; }
    .button { display: inline-block; margin-top: 12px; padding: 11px 18px; border-radius: 8px; background: ` + accent + `; color: #ffffff; text-decoration: none; font-weight: 600; }
    .muted { color: #71717a; font-size: 13px; }
    .footer { padding: 18px 32px; background: #fafafa; color: #a1a1aa; font-size: 12px; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header"><h1>` + title + `</h1></div>
    <div class="content">` + content + `</div>
    <div class="footer">This email was sent by {{site_name}}. Please do not reply directly.</div>
  </div>
</body>
</html>`
}
