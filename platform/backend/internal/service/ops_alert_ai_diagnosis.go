package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	opsDiagnosisMaxPromptRunes   = 600
	opsDiagnosisMaxResponseBytes = 128 * 1024
)

type opsReportDiagnosisInput struct {
	ReportName                   string                 `json:"report_name"`
	ReportType                   string                 `json:"report_type"`
	PeriodDisplay                string                 `json:"period_display"`
	TotalRequests                int64                  `json:"total_requests"`
	SLARequests                  int64                  `json:"sla_requests"`
	SuccessCount                 int64                  `json:"success_count"`
	ErrorCountSLA                int64                  `json:"error_count_sla"`
	BusinessLimitedCount         int64                  `json:"business_limited_count"`
	SLAPercent                   float64                `json:"sla_percent"`
	ErrorRatePercent             float64                `json:"error_rate_percent"`
	UpstreamErrorRatePercent     float64                `json:"upstream_error_rate_percent"`
	UpstreamErrorCountExcl429529 int64                  `json:"upstream_error_count_excl_429_529"`
	Upstream429Count             int64                  `json:"upstream_429_count"`
	Upstream529Count             int64                  `json:"upstream_529_count"`
	LatencyP50Ms                 *int                   `json:"latency_p50_ms,omitempty"`
	LatencyP99Ms                 *int                   `json:"latency_p99_ms,omitempty"`
	TTFTP50Ms                    *int                   `json:"ttft_p50_ms,omitempty"`
	TTFTP99Ms                    *int                   `json:"ttft_p99_ms,omitempty"`
	TokenConsumed                int64                  `json:"token_consumed"`
	AccountPoolSummary           string                 `json:"account_pool_summary,omitempty"`
	TopClusters                  []opsAlertErrorCluster `json:"top_error_clusters,omitempty"`
}

type opsAIDiagnosisRequest struct {
	Model       string                   `json:"model"`
	Temperature float64                  `json:"temperature"`
	MaxTokens   int                      `json:"max_tokens"`
	Messages    []opsAIDiagnosisMessage  `json:"messages"`
}

type opsAIDiagnosisMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type opsAIDiagnosisResponse struct {
	Choices []struct {
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

const (
	opsDiagnosisRoutePrimary       = "primary"
	opsDiagnosisRouteBackup        = "backup"
	opsDiagnosisMaxTotalBudget     = 15 * time.Second
	opsDiagnosisDeliveryHeadroom   = 15 * time.Second
	opsDiagnosisMinUsableBudget    = 2 * time.Second
)

func boundedOpsAIDiagnosisContext(parent context.Context, maxBudget, reserveHeadroom time.Duration) (context.Context, context.CancelFunc, bool) {
	if parent == nil {
		parent = context.Background()
	}
	if parent.Err() != nil {
		return parent, func() {}, false
	}
	if maxBudget <= 0 {
		maxBudget = opsDiagnosisMaxTotalBudget
	}
	budget := maxBudget
	if deadline, ok := parent.Deadline(); ok {
		avail := time.Until(deadline) - reserveHeadroom
		if avail < opsDiagnosisMinUsableBudget {
			return parent, func() {}, false
		}
		if avail < budget {
			budget = avail
		}
	}
	ctx, cancel := context.WithTimeout(parent, budget)
	return ctx, cancel, true
}

func opsDiagnosisBadge(usedAI, aiEnabled bool, locale string) string {
	route := ""
	if usedAI {
		route = opsDiagnosisRoutePrimary
	}
	return opsDiagnosisBadgeForRoute(route, aiEnabled, locale)
}

func opsDiagnosisBadgeForRoute(route string, aiEnabled bool, locale string) string {
	zh := isChineseOpsLocale(locale)
	switch route {
	case opsDiagnosisRoutePrimary:
		if zh {
			return "AI 智能诊断"
		}
		return "AI Diagnosis"
	case opsDiagnosisRouteBackup:
		if zh {
			return "AI 智能诊断（备用线路）"
		}
		return "AI Diagnosis (Backup Route)"
	}
	if aiEnabled {
		if zh {
			return "内置规则诊断（AI 暂不可用，已自动兜底）"
		}
		return "Built-in Rule Summary (AI fallback)"
	}
	if zh {
		return "内置规则自动诊断"
	}
	return "Built-in Rule Summary"
}

func DiagnoseOpsAlert(
	ctx context.Context,
	cfg OpsAIDiagnosisConfig,
	apiKey string,
	ev *opsAlertEvidence,
	locale string,
) (summary string, usedAI bool, fallbackReason string) {
	text, route, reason := DiagnoseOpsAlertWithRoutes(ctx, cfg, apiKey, "", ev, locale)
	return text, route != "", reason
}

func DiagnoseOpsAlertWithRoutes(
	ctx context.Context,
	cfg OpsAIDiagnosisConfig,
	primaryAPIKey string,
	fallbackAPIKey string,
	ev *opsAlertEvidence,
	locale string,
) (summary string, route string, fallbackReason string) {
	if ev == nil {
		return "", "", "empty_evidence"
	}
	fallback := strings.TrimSpace(ev.FallbackSummary)
	if !cfg.Enabled {
		return fallback, "", "not_configured"
	}

	type aiPromptCluster struct {
		Platform     string `json:"platform"`
		Model        string `json:"model"`
		Group        string `json:"group"`
		Account      string `json:"account"`
		StatusCode   int    `json:"status_code"`
		Count        int    `json:"count"`
		Reason       string `json:"reason"`
		SampleNotice string `json:"sample_message,omitempty"`
	}
	clusters := make([]aiPromptCluster, 0, len(ev.TopClusters))
	for _, c := range ev.TopClusters {
		clusters = append(clusters, aiPromptCluster{
			Platform:     c.Platform,
			Model:        c.Model,
			Group:        c.GroupLabel,
			Account:      c.AccountLabel,
			StatusCode:   c.StatusCode,
			Count:        c.Count,
			Reason:       c.Reason,
			SampleNotice: c.SampleMessage,
		})
	}
	payload := map[string]any{
		"rule_name":            ev.RuleName,
		"severity":             ev.SeverityLabel,
		"status":               ev.StatusLabel,
		"metric_summary":       ev.MetricSummary,
		"scope":                ev.ScopeLabel,
		"window":               ev.WindowLabel,
		"fired_at":             ev.FiredAtLocal,
		"resolved_at":          ev.ResolvedAtLocal,
		"duration":             ev.DurationHuman,
		"total_requests":       ev.TotalRequests,
		"sla_requests":         ev.SLARequests,
		"success_count":        ev.SuccessCount,
		"error_count":          ev.ErrorCount,
		"affected_users":       ev.AffectedUsers,
		"low_sample_warning":   ev.LowSampleWarning,
		"system_summary":       ev.SystemSummary,
		"account_pool_summary": ev.AccountPoolSummary,
		"top_error_clusters":   clusters,
	}
	userBytes, err := json.Marshal(payload)
	if err != nil {
		return fallback, "", "marshal_failed"
	}

	systemPrompt := buildOpsAlertAISystemPrompt(locale)
	aiText, usedRoute, reason := callOpsDiagnosisWithFailover(ctx, cfg, primaryAPIKey, fallbackAPIKey, systemPrompt, string(userBytes))
	if aiText == "" {
		return fallback, "", reason
	}
	return aiText, usedRoute, reason
}

func DiagnoseOpsScheduledReport(
	ctx context.Context,
	cfg OpsAIDiagnosisConfig,
	apiKey string,
	input *opsReportDiagnosisInput,
	locale string,
) (summary string, usedAI bool, fallbackReason string) {
	text, route, reason := DiagnoseOpsScheduledReportWithRoutes(ctx, cfg, apiKey, "", input, locale)
	return text, route != "", reason
}

func DiagnoseOpsScheduledReportWithRoutes(
	ctx context.Context,
	cfg OpsAIDiagnosisConfig,
	primaryAPIKey string,
	fallbackAPIKey string,
	input *opsReportDiagnosisInput,
	locale string,
) (summary string, route string, fallbackReason string) {
	fallback := buildRuleBasedReportSummary(input, locale)
	if input == nil {
		return fallback, "", "empty_input"
	}
	if !cfg.Enabled {
		return fallback, "", "not_configured"
	}
	userBytes, err := json.Marshal(input)
	if err != nil {
		return fallback, "", "marshal_failed"
	}
	systemPrompt := buildOpsReportAISystemPrompt(locale)
	aiText, usedRoute, reason := callOpsDiagnosisWithFailover(ctx, cfg, primaryAPIKey, fallbackAPIKey, systemPrompt, string(userBytes))
	if aiText == "" {
		return fallback, "", reason
	}
	return aiText, usedRoute, reason
}

func callOpsDiagnosisWithFailover(
	ctx context.Context,
	cfg OpsAIDiagnosisConfig,
	primaryAPIKey string,
	fallbackAPIKey string,
	systemPrompt string,
	userPrompt string,
) (text string, route string, reason string) {
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", "", "timeout"
		}
		return "", "", "context_canceled"
	}

	primaryReady := strings.TrimSpace(cfg.BaseURL) != "" && strings.TrimSpace(cfg.Model) != "" && strings.TrimSpace(primaryAPIKey) != ""
	backupKey := strings.TrimSpace(fallbackAPIKey)
	if backupKey == "" {
		backupKey = strings.TrimSpace(primaryAPIKey)
	}
	backupReady := strings.TrimSpace(cfg.FallbackBaseURL) != "" && strings.TrimSpace(cfg.FallbackModel) != "" && backupKey != ""

	if !primaryReady && !backupReady {
		return "", "", "not_configured"
	}

	primaryReason := "primary_not_configured"
	if primaryReady {
		primaryCtx := ctx
		primaryCancel := func() {}
		if backupReady {
			// Reserve half of remaining deadline (when bounded) so a slow primary still leaves room for backup failover.
			if deadline, ok := ctx.Deadline(); ok {
				remaining := time.Until(deadline)
				if remaining >= 4*time.Second {
					half := remaining / 2
					if cfg.TimeoutSeconds > 0 && time.Duration(cfg.TimeoutSeconds)*time.Second < half {
						half = time.Duration(cfg.TimeoutSeconds) * time.Second
					}
					primaryCtx, primaryCancel = context.WithTimeout(ctx, half)
				}
			}
		}
		aiText, errReason := callOpsDiagnosisChatCompletion(primaryCtx, cfg, primaryAPIKey, systemPrompt, userPrompt)
		primaryCancel()
		if aiText != "" {
			return aiText, opsDiagnosisRoutePrimary, ""
		}
		primaryReason = errReason
	}

	if backupReady {
		if ctx.Err() != nil {
			return "", "", fmt.Sprintf("primary:%s;backup:context_expired", primaryReason)
		}
		backupCfg := cfg
		backupCfg.BaseURL = strings.TrimSpace(cfg.FallbackBaseURL)
		backupCfg.Model = strings.TrimSpace(cfg.FallbackModel)
		logger.LegacyPrintf("service.ops_alert_ai_diagnosis", "[OpsAIDiagnosis] primary failed (%s), switching to backup model=%q", primaryReason, backupCfg.Model)
		aiText, backupReason := callOpsDiagnosisChatCompletion(ctx, backupCfg, backupKey, systemPrompt, userPrompt)
		if aiText != "" {
			return aiText, opsDiagnosisRouteBackup, primaryReason
		}
		return "", "", fmt.Sprintf("primary:%s;backup:%s", primaryReason, backupReason)
	}

	return "", "", primaryReason
}

func buildOpsAlertAISystemPrompt(locale string) string {
	if isChineseOpsLocale(locale) {
		return "你是 AI API 中转站（GABUGU）的运维值班助手。站点老板不熟悉代码与网络术语。请根据提供的脱敏聚合数据，用平实、冷静、直白的中文写 2 到 3 句话（不超过 180 字）：1）说明发生了什么、严重程度或是否只是低流量偶发波动；2）指出最可能的原因及具体平台/模型/账号；3）给出最直接的一步处理建议。提及账号或分组时，必须原样使用数据中提供的后台账号名称（account）和分组名称（group），不要使用“#数字”之类的编号或代号。禁止使用 Markdown 标题、代码块或复杂黑话。"
	}
	return "You are an operations assistant for an AI API gateway. Write 2-3 plain, calm sentences (under 140 words) for a non-technical owner: 1) what happened and whether volume is large enough to worry, 2) the most likely cause with affected platform/model/account, 3) the single clearest next action. Always refer to accounts and groups by their exact names from the provided data (account/group fields); never use numeric IDs or codes like #88. Do not use markdown headings or code blocks."
}

func buildOpsReportAISystemPrompt(locale string) string {
	if isChineseOpsLocale(locale) {
		return "你是 AI API 中转站（GABUGU）的运维值班助手。请根据过去一个周期的运行统计，用亲切平实的中文写 2 到 3 句早报结论（不超过 180 字）：概括整体稳不稳定、哪类报错或哪个账号最值得关注、账号池是否健康，以及今天需不需要处理。提及账号或分组时，必须原样使用数据里的真实后台名称，不要出现“#数字”之类的代号。禁止使用 Markdown 标题或代码块。"
	}
	return "You are an operations assistant for an AI API gateway. Write a 2-3 sentence plain-language summary (under 140 words) covering overall stability, notable error clusters or latency issues, account pool health, and whether any action is needed today. When mentioning accounts or groups, use their exact names from the data rather than numeric IDs. Do not use markdown headings or code blocks."
}

func callOpsDiagnosisChatCompletion(
	ctx context.Context,
	cfg OpsAIDiagnosisConfig,
	apiKey string,
	systemPrompt string,
	userPrompt string,
) (string, string) {
	endpoint := normalizeOpsChatCompletionsURL(cfg.BaseURL)
	if !isValidOpsHTTPURL(endpoint) {
		return "", "invalid_url"
	}
	timeoutSec := cfg.TimeoutSeconds
	if timeoutSec <= 0 {
		timeoutSec = 8
	} else if timeoutSec < 3 {
		timeoutSec = 3
	} else if timeoutSec > 30 {
		timeoutSec = 30
	}
	if ctx == nil {
		ctx = context.Background()
	}
	callCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	reqBody := opsAIDiagnosisRequest{
		Model:       strings.TrimSpace(cfg.Model),
		Temperature: 0.2,
		MaxTokens:   320,
		Messages: []opsAIDiagnosisMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	rawReq, err := json.Marshal(reqBody)
	if err != nil {
		return "", "marshal_error"
	}

	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint, bytes.NewReader(rawReq))
	if err != nil {
		return "", "request_build_error"
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))

	client := &http.Client{Timeout: time.Duration(timeoutSec+1) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.LegacyPrintf("service.ops_alert_ai_diagnosis", "[OpsAIDiagnosis] request failed (model=%q): %s", cfg.Model, redactOpsErrorSnippet(err.Error(), 240))
		if callCtx.Err() == context.DeadlineExceeded {
			return "", "timeout"
		}
		return "", "http_error"
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		logger.LegacyPrintf("service.ops_alert_ai_diagnosis", "[OpsAIDiagnosis] non-2xx status=%d (model=%q)", resp.StatusCode, cfg.Model)
		return "", fmt.Sprintf("http_status_%d", resp.StatusCode)
	}

	rawResp, err := io.ReadAll(io.LimitReader(resp.Body, opsDiagnosisMaxResponseBytes))
	if err != nil {
		logger.LegacyPrintf("service.ops_alert_ai_diagnosis", "[OpsAIDiagnosis] read body failed: %v", err)
		return "", "read_error"
	}

	var decoded opsAIDiagnosisResponse
	if err := json.Unmarshal(rawResp, &decoded); err != nil {
		logger.LegacyPrintf("service.ops_alert_ai_diagnosis", "[OpsAIDiagnosis] decode response failed: %v", err)
		return "", "decode_error"
	}
	if len(decoded.Choices) == 0 {
		return "", "empty_choices"
	}

	text := extractOpsAIChoiceText(decoded.Choices[0].Message.Content)
	text = sanitizeOpsAIDiagnosisText(text)
	if text == "" {
		return "", "empty_content"
	}
	return text, ""
}

func normalizeOpsChatCompletionsURL(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(trimmed), "/chat/completions") {
		return trimmed
	}
	if parsed, err := url.Parse(trimmed); err == nil && (parsed.Path == "" || parsed.Path == "/") {
		return trimmed + "/v1/chat/completions"
	}
	return trimmed + "/chat/completions"
}

func extractOpsAIChoiceText(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return str
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var out []string
		for _, p := range parts {
			if strings.TrimSpace(p.Text) != "" {
				out = append(out, strings.TrimSpace(p.Text))
			}
		}
		return strings.Join(out, " ")
	}
	return ""
}

func sanitizeOpsAIDiagnosisText(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = strings.TrimPrefix(s, "```markdown")
	s = strings.TrimPrefix(s, "```text")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.ReplaceAll(s, "**", "")
	s = opsSpaceCollapse.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) > opsDiagnosisMaxPromptRunes {
		s = string(runes[:opsDiagnosisMaxPromptRunes]) + "..."
	}
	return s
}

func buildRuleBasedReportSummary(input *opsReportDiagnosisInput, locale string) string {
	if input == nil {
		if isChineseOpsLocale(locale) {
			return "暂无统计周期内的运行数据。"
		}
		return "No operational data available for this reporting period."
	}
	zh := isChineseOpsLocale(locale)
	if zh {
		if input.TotalRequests == 0 {
			return fmt.Sprintf("本周期（%s）暂无用户调用请求。%s",
				input.PeriodDisplay,
				fallbackIfNonEmpty(input.AccountPoolSummary, "系统保持待命运行中。"))
		}
		healthDesc := "运行整体平稳"
		if input.SLAPercent < 95 {
			healthDesc = "可用率偏低，建议关注失败明细"
		} else if input.SLAPercent < 99 {
			healthDesc = "整体基本稳定，存在少量异常波动"
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("本周期共处理 %s 次请求（其中有效计入可用率 %s 次，成功 %s 次，SLA 失败 %s 次，真实可用率 %.2f%%），%s。",
			formatOpsReportInteger(input.TotalRequests),
			formatOpsReportInteger(input.SLARequests),
			formatOpsReportInteger(input.SuccessCount),
			formatOpsReportInteger(input.ErrorCountSLA),
			input.SLAPercent,
			healthDesc))
		if len(input.TopClusters) > 0 {
			top := input.TopClusters[0]
			if top.AccountLabel != "" && top.AccountLabel != "-" {
				b.WriteString(fmt.Sprintf(" 报错最多的是账号「%s」的 %s（%d 次）：%s。", top.AccountLabel, top.Model, top.Count, top.Reason))
			} else {
				b.WriteString(fmt.Sprintf(" 报错最多的是 %s · %s（%d 次）：%s。", top.Platform, top.Model, top.Count, top.Reason))
			}
		}
		if input.AccountPoolSummary != "" {
			b.WriteString(" " + input.AccountPoolSummary + "。")
		}
		return b.String()
	}

	if input.TotalRequests == 0 {
		return fmt.Sprintf("No requests were recorded during %s.", input.PeriodDisplay)
	}
	healthDesc := "Service stability looks healthy"
	if input.SLAPercent < 95 {
		healthDesc = "Availability dipped below 95% and warrants review"
	} else if input.SLAPercent < 99 {
		healthDesc = "Service remained mostly stable with minor error spikes"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s: %s total requests (%s SLA requests, %s SLA errors, %.2f%% SLA).",
		healthDesc,
		formatOpsReportInteger(input.TotalRequests),
		formatOpsReportInteger(input.SLARequests),
		formatOpsReportInteger(input.ErrorCountSLA),
		input.SLAPercent))
	if len(input.TopClusters) > 0 {
		top := input.TopClusters[0]
		if top.AccountLabel != "" && top.AccountLabel != "-" {
			b.WriteString(fmt.Sprintf(" Top error cluster: account \"%s\" · %s (%d times) — %s.", top.AccountLabel, top.Model, top.Count, top.Reason))
		} else {
			b.WriteString(fmt.Sprintf(" Top error cluster: %s/%s (%d times) — %s.", top.Platform, top.Model, top.Count, top.Reason))
		}
	}
	return b.String()
}

func fallbackIfNonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	return fallback
}
