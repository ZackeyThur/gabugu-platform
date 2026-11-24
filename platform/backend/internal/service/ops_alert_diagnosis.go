package service

import (
	"context"
	"fmt"
	"html"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

var (
	opsRedactBearerPattern = regexp.MustCompile(`(?i)(authorization\s*[:=]\s*bearer\s+|bearer\s+)[A-Za-z0-9._\-~+/=]+`)
	opsRedactKeyPattern    = regexp.MustCompile(`\b(sk-[A-Za-z0-9_\-]{8,}|AIza[0-9A-Za-z\-_]{16,}|ghp_[A-Za-z0-9]{16,}|xox[baprs]-[A-Za-z0-9\-]{10,}|admin-[A-Za-z0-9_\-]{8,})\b`)
	opsRedactQueryPattern  = regexp.MustCompile(`(?i)([?&](?:api_?key|key|token|secret|password|access_token)=)[^&\s"'<>]+`)
	opsRedactEmailPattern  = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	opsSpaceCollapse       = regexp.MustCompile(`\s+`)
)

type opsAlertErrorCluster struct {
	Platform      string `json:"platform"`
	Model         string `json:"model"`
	GroupLabel    string `json:"group_label"`
	AccountLabel  string `json:"account_label"`
	StatusCode    int    `json:"status_code"`
	Phase         string `json:"phase"`
	Count         int    `json:"count"`
	ReasonKey     string `json:"reason_key"`
	Reason        string `json:"reason"`
	Suggestion    string `json:"suggestion"`
	SampleMessage string `json:"sample_message,omitempty"`
}

type opsAlertEvidence struct {
	Locale               string
	RuleID               int64
	RuleName             string
	Severity             string
	SeverityLabel        string
	Status               string
	StatusLabel          string
	MetricType           string
	MetricLabel          string
	MetricUnit           string
	Operator             string
	MetricValue          *float64
	ThresholdValue       *float64
	MetricSummary        string
	ScopeLabel           string
	WindowMinutes        int
	WindowLabel          string
	FiredAt              time.Time
	FiredAtLocal         string
	ResolvedAt           *time.Time
	ResolvedAtLocal      string
	DurationHuman        string
	TotalRequests        int64
	SLARequests          int64
	SuccessCount         int64
	ErrorCount           int64
	BusinessLimitedCount int64
	AffectedUsers        int
	LowSampleWarning     bool
	ShowRequestOverview  bool
	TopClusters          []opsAlertErrorCluster
	SystemSummary        string
	AccountPoolSummary   string
	FallbackSummary      string
	Suggestions          []string
	DashboardURL         string
}

func isChineseOpsLocale(locale string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(locale))
	return strings.HasPrefix(trimmed, "zh") || trimmed == "cn"
}

func loadBeijingOpsLocation() *time.Location {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil && loc != nil {
		return loc
	}
	return time.FixedZone("CST", 8*3600)
}

func resolveOpsDisplayLocation(cfg *config.Config, locale string) (*time.Location, string) {
	if cfg != nil && strings.TrimSpace(cfg.Timezone) != "" {
		tz := strings.TrimSpace(cfg.Timezone)
		if tz == "Asia/Shanghai" {
			loc := loadBeijingOpsLocation()
			if isChineseOpsLocale(locale) {
				return loc, "北京时间"
			}
			return loc, "Asia/Shanghai"
		}
		if loc, err := time.LoadLocation(tz); err == nil && loc != nil {
			return loc, tz
		}
	}
	if isChineseOpsLocale(locale) {
		return loadBeijingOpsLocation(), "北京时间"
	}
	return time.UTC, "UTC"
}

func formatOpsLocalTime(t time.Time, cfg *config.Config, locale string) string {
	if t.IsZero() {
		return "-"
	}
	loc, label := resolveOpsDisplayLocation(cfg, locale)
	local := t.In(loc)
	if isChineseOpsLocale(locale) {
		return fmt.Sprintf("%s（%s）", local.Format("2006-01-02 15:04:05"), label)
	}
	return fmt.Sprintf("%s (%s)", local.Format("2006-01-02 15:04:05"), label)
}

func formatOpsLocalPeriod(start, end time.Time, cfg *config.Config, locale string) string {
	if start.IsZero() || end.IsZero() {
		return "-"
	}
	loc, label := resolveOpsDisplayLocation(cfg, locale)
	s := start.In(loc).Format("2006-01-02 15:04")
	e := end.In(loc).Format("2006-01-02 15:04")
	if isChineseOpsLocale(locale) {
		return fmt.Sprintf("%s ~ %s（%s）", s, e, label)
	}
	return fmt.Sprintf("%s to %s (%s)", s, e, label)
}

func formatOpsDurationHuman(d time.Duration, locale string) string {
	if d <= 0 {
		if isChineseOpsLocale(locale) {
			return "不足 1 分钟"
		}
		return "< 1m"
	}
	totalSec := int64(d.Round(time.Second).Seconds())
	if totalSec < 60 {
		if isChineseOpsLocale(locale) {
			return fmt.Sprintf("%d 秒", totalSec)
		}
		return fmt.Sprintf("%ds", totalSec)
	}
	hours := totalSec / 3600
	mins := (totalSec % 3600) / 60
	secs := totalSec % 60
	if hours > 0 {
		if isChineseOpsLocale(locale) {
			if mins > 0 {
				return fmt.Sprintf("%d 小时 %d 分", hours, mins)
			}
			return fmt.Sprintf("%d 小时", hours)
		}
		if mins > 0 {
			return fmt.Sprintf("%dh %dm", hours, mins)
		}
		return fmt.Sprintf("%dh", hours)
	}
	if isChineseOpsLocale(locale) {
		if secs > 0 {
			return fmt.Sprintf("%d 分 %d 秒", mins, secs)
		}
		return fmt.Sprintf("%d 分钟", mins)
	}
	if secs > 0 {
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	return fmt.Sprintf("%dm", mins)
}

func opsSeverityLabel(severity, locale string) string {
	sev := strings.ToUpper(strings.TrimSpace(severity))
	if isChineseOpsLocale(locale) {
		switch sev {
		case "P0":
			return "紧急（P0）"
		case "P1":
			return "重要（P1）"
		case "P2":
			return "关注（P2）"
		case "P3":
			return "提示（P3）"
		default:
			if sev == "" {
				return "普通"
			}
			return sev
		}
	}
	switch sev {
	case "P0":
		return "Critical (P0)"
	case "P1":
		return "Warning (P1)"
	case "P2":
		return "Notice (P2)"
	case "P3":
		return "Info (P3)"
	default:
		if sev == "" {
			return "Info"
		}
		return sev
	}
}

func opsSeverityShortTag(severity, status, locale string) string {
	if status == OpsAlertStatusResolved || status == OpsAlertStatusManualResolved {
		if isChineseOpsLocale(locale) {
			return "已恢复"
		}
		return "Recovered"
	}
	sev := strings.ToUpper(strings.TrimSpace(severity))
	if isChineseOpsLocale(locale) {
		switch sev {
		case "P0":
			return "紧急告警"
		case "P1":
			return "重要告警"
		case "P2":
			return "运维提醒"
		case "P3":
			return "状态提示"
		default:
			return "运维告警"
		}
	}
	switch sev {
	case "P0":
		return "Critical"
	case "P1":
		return "Warning"
	case "P2":
		return "Notice"
	default:
		return "Alert"
	}
}

func opsSeverityAccentColor(severity, status string) string {
	if status == OpsAlertStatusResolved || status == OpsAlertStatusManualResolved {
		return "#16a34a"
	}
	switch strings.ToUpper(strings.TrimSpace(severity)) {
	case "P0":
		return "#dc2626"
	case "P1":
		return "#ea580c"
	case "P2":
		return "#d97706"
	case "P3":
		return "#2563eb"
	default:
		return "#ea580c"
	}
}

func opsStatusLabel(status, locale string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	if isChineseOpsLocale(locale) {
		switch s {
		case OpsAlertStatusFiring:
			return "异常触发中"
		case OpsAlertStatusResolved:
			return "已自动恢复"
		case OpsAlertStatusManualResolved:
			return "已手动解除"
		default:
			if s == "" {
				return "-"
			}
			return s
		}
	}
	switch s {
	case OpsAlertStatusFiring:
		return "Firing"
	case OpsAlertStatusResolved:
		return "Auto-Resolved"
	case OpsAlertStatusManualResolved:
		return "Manually Resolved"
	default:
		if s == "" {
			return "-"
		}
		return s
	}
}

func opsMetricMeta(metricType, locale string) (label string, unit string) {
	zh := isChineseOpsLocale(locale)
	switch strings.TrimSpace(metricType) {
	case "success_rate":
		if zh {
			return "请求成功率", "%"
		}
		return "Success rate", "%"
	case "error_rate":
		if zh {
			return "请求错误率", "%"
		}
		return "Error rate", "%"
	case "upstream_error_rate":
		if zh {
			return "上游错误率（不含限流）", "%"
		}
		return "Upstream error rate (excl. 429/529)", "%"
	case "cpu_usage_percent":
		if zh {
			return "服务器 CPU 使用率", "%"
		}
		return "CPU usage", "%"
	case "memory_usage_percent":
		if zh {
			return "服务器内存使用率", "%"
		}
		return "Memory usage", "%"
	case "concurrency_queue_depth":
		if zh {
			return "并发请求排队深度", "个"
		}
		return "Concurrency queue depth", "reqs"
	case "group_available_accounts":
		if zh {
			return "分组可用账号数", "个"
		}
		return "Group available accounts", "accounts"
	case "group_available_ratio":
		if zh {
			return "分组可用账号比例", "%"
		}
		return "Group available ratio", "%"
	case "group_rate_limit_ratio":
		if zh {
			return "分组限流账号比例", "%"
		}
		return "Group rate-limit ratio", "%"
	case "account_rate_limited_count":
		if zh {
			return "被限流账号数", "个"
		}
		return "Rate-limited accounts", "accounts"
	case "account_error_count":
		if zh {
			return "异常报错账号数", "个"
		}
		return "Errored accounts", "accounts"
	case "account_error_ratio":
		if zh {
			return "异常报错账号比例", "%"
		}
		return "Errored account ratio", "%"
	case "account_temp_unscheduled_count":
		if zh {
			return "临时冷却（不可调度）账号数", "个"
		}
		return "Temp-unscheduled accounts", "accounts"
	case "overload_account_count":
		if zh {
			return "过载账号数", "个"
		}
		return "Overloaded accounts", "accounts"
	case "proxy_expired_count":
		if zh {
			return "已过期代理数", "个"
		}
		return "Expired proxies", "proxies"
	case "proxy_expiring_soon_count":
		if zh {
			return "即将过期代理数", "个"
		}
		return "Expiring-soon proxies", "proxies"
	case "p95_latency_ms":
		if zh {
			return "P95 请求延迟", "ms"
		}
		return "P95 latency", "ms"
	case "p99_latency_ms":
		if zh {
			return "P99 请求延迟", "ms"
		}
		return "P99 latency", "ms"
	default:
		return strings.TrimSpace(metricType), ""
	}
}

func isRequestRateOpsMetric(metricType string) bool {
	switch strings.TrimSpace(metricType) {
	case "success_rate", "error_rate", "upstream_error_rate":
		return true
	default:
		return false
	}
}

func formatOpsMetricValue(val float64, unit string) string {
	switch unit {
	case "%":
		return fmt.Sprintf("%.2f%%", val)
	case "ms":
		if math.Abs(val-math.Trunc(val)) < 0.01 {
			return fmt.Sprintf("%.0f ms", val)
		}
		return fmt.Sprintf("%.1f ms", val)
	case "":
		if math.Abs(val-math.Trunc(val)) < 0.01 {
			return fmt.Sprintf("%.0f", val)
		}
		return fmt.Sprintf("%.2f", val)
	default:
		if math.Abs(val-math.Trunc(val)) < 0.01 {
			return fmt.Sprintf("%.0f %s", val, unit)
		}
		return fmt.Sprintf("%.1f %s", val, unit)
	}
}

func formatOpsMetricSummary(metricType, operator string, val, threshold *float64, locale string) string {
	label, unit := opsMetricMeta(metricType, locale)
	valStr := "-"
	if val != nil {
		valStr = formatOpsMetricValue(*val, unit)
	}
	thrStr := "-"
	if threshold != nil {
		thrStr = formatOpsMetricValue(*threshold, unit)
	}
	op := strings.TrimSpace(operator)
	if isChineseOpsLocale(locale) {
		return fmt.Sprintf("%s 当前 %s（告警条件：%s %s）", label, valStr, op, thrStr)
	}
	return fmt.Sprintf("%s is %s (rule: %s %s)", label, valStr, op, thrStr)
}

func formatOpsScopeLabel(platform string, groupID *int64, locale string) string {
	return formatOpsScopeLabelWithGroupName(platform, groupID, "", locale)
}

func formatOpsScopeLabelWithGroupName(platform string, groupID *int64, groupName, locale string) string {
	p := strings.TrimSpace(platform)
	gName := strings.TrimSpace(groupName)
	hasGroup := (groupID != nil && *groupID > 0) || gName != ""
	if p == "" && !hasGroup {
		if isChineseOpsLocale(locale) {
			return "全站全部渠道"
		}
		return "All platforms & groups"
	}
	parts := make([]string, 0, 2)
	if p != "" {
		if isChineseOpsLocale(locale) {
			parts = append(parts, fmt.Sprintf("平台 %s", p))
		} else {
			parts = append(parts, fmt.Sprintf("platform %s", p))
		}
	}
	if hasGroup {
		if gName != "" {
			if isChineseOpsLocale(locale) {
				parts = append(parts, fmt.Sprintf("分组「%s」", gName))
			} else {
				parts = append(parts, fmt.Sprintf("group \"%s\"", gName))
			}
		} else if groupID != nil && *groupID > 0 {
			if isChineseOpsLocale(locale) {
				parts = append(parts, fmt.Sprintf("未命名分组（编号 %d）", *groupID))
			} else {
				parts = append(parts, fmt.Sprintf("unnamed group (ID %d)", *groupID))
			}
		}
	}
	return strings.Join(parts, " · ")
}

func opsErrorLogGroupAndAccountLabels(item *OpsErrorLog, locale string) (groupLabel string, accountLabel string) {
	groupLabel = "-"
	accountLabel = "-"
	if item == nil {
		return groupLabel, accountLabel
	}
	zh := isChineseOpsLocale(locale)
	if name := strings.TrimSpace(item.GroupName); name != "" {
		groupLabel = name
	} else if item.GroupID != nil && *item.GroupID > 0 {
		if zh {
			groupLabel = fmt.Sprintf("未命名分组（编号 %d）", *item.GroupID)
		} else {
			groupLabel = fmt.Sprintf("Unnamed group (ID %d)", *item.GroupID)
		}
	}
	if name := strings.TrimSpace(item.AccountName); name != "" {
		accountLabel = name
	} else if item.AccountID != nil && *item.AccountID > 0 {
		if zh {
			accountLabel = fmt.Sprintf("未命名账号（编号 %d）", *item.AccountID)
		} else {
			accountLabel = fmt.Sprintf("Unnamed account (ID %d)", *item.AccountID)
		}
	}
	return groupLabel, accountLabel
}

func formatOpsClusterChannelLabel(groupLabel, accountLabel, locale string) string {
	group := strings.TrimSpace(groupLabel)
	if group == "-" {
		group = ""
	}
	account := strings.TrimSpace(accountLabel)
	if account == "-" {
		account = ""
	}
	zh := isChineseOpsLocale(locale)
	switch {
	case account != "" && group != "":
		if zh {
			return fmt.Sprintf("%s（分组：%s）", account, group)
		}
		return fmt.Sprintf("%s (Group: %s)", account, group)
	case account != "":
		return account
	case group != "":
		if zh {
			return fmt.Sprintf("未分配具体账号（分组：%s）", group)
		}
		return fmt.Sprintf("No account assigned (Group: %s)", group)
	default:
		if zh {
			return "未分配账号"
		}
		return "Unassigned"
	}
}

func enrichOpsErrorLogNames(ctx context.Context, opsSvc *OpsService, avail *OpsAccountAvailability, logs []*OpsErrorLog) {
	if len(logs) == 0 {
		return
	}
	accountNames := make(map[int64]string)
	groupNames := make(map[int64]string)

	absorbAvail := func(a *OpsAccountAvailability) {
		if a == nil {
			return
		}
		if a.Group != nil && a.Group.GroupID > 0 && strings.TrimSpace(a.Group.GroupName) != "" {
			groupNames[a.Group.GroupID] = strings.TrimSpace(a.Group.GroupName)
		}
		for _, acc := range a.Accounts {
			if acc == nil {
				continue
			}
			if acc.AccountID > 0 && strings.TrimSpace(acc.AccountName) != "" {
				accountNames[acc.AccountID] = strings.TrimSpace(acc.AccountName)
			}
			if acc.GroupID > 0 && strings.TrimSpace(acc.GroupName) != "" {
				groupNames[acc.GroupID] = strings.TrimSpace(acc.GroupName)
			}
		}
	}
	absorbAvail(avail)

	for _, item := range logs {
		if item == nil {
			continue
		}
		if item.AccountID != nil && *item.AccountID > 0 && strings.TrimSpace(item.AccountName) != "" {
			accountNames[*item.AccountID] = strings.TrimSpace(item.AccountName)
		}
		if item.GroupID != nil && *item.GroupID > 0 && strings.TrimSpace(item.GroupName) != "" {
			groupNames[*item.GroupID] = strings.TrimSpace(item.GroupName)
		}
	}

	hasMissing := func() bool {
		for _, item := range logs {
			if item == nil {
				continue
			}
			if item.AccountID != nil && *item.AccountID > 0 && strings.TrimSpace(item.AccountName) == "" && accountNames[*item.AccountID] == "" {
				return true
			}
			if item.GroupID != nil && *item.GroupID > 0 && strings.TrimSpace(item.GroupName) == "" && groupNames[*item.GroupID] == "" {
				return true
			}
		}
		return false
	}

	if hasMissing() && opsSvc != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		subCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if avail == nil {
			if fetchedAvail, err := opsSvc.GetAccountAvailability(subCtx, "", nil); err == nil && fetchedAvail != nil {
				absorbAvail(fetchedAvail)
			}
		}

		if opsSvc.accountRepo != nil {
			const maxAccountLookups = 10
			lookedUp := make(map[int64]struct{})
			for _, item := range logs {
				if item == nil || item.AccountID == nil || *item.AccountID <= 0 {
					continue
				}
				id := *item.AccountID
				if accountNames[id] != "" {
					continue
				}
				if _, seen := lookedUp[id]; seen {
					continue
				}
				if len(lookedUp) >= maxAccountLookups {
					break
				}
				lookedUp[id] = struct{}{}
				acc, err := opsSvc.accountRepo.GetByID(subCtx, id)
				if err != nil || acc == nil {
					continue
				}
				if strings.TrimSpace(acc.Name) != "" {
					accountNames[id] = strings.TrimSpace(acc.Name)
				}
				for _, grp := range acc.Groups {
					if grp != nil && grp.ID > 0 && strings.TrimSpace(grp.Name) != "" {
						groupNames[grp.ID] = strings.TrimSpace(grp.Name)
					}
				}
			}
		}
	}

	for _, item := range logs {
		if item == nil {
			continue
		}
		if item.AccountID != nil && *item.AccountID > 0 && strings.TrimSpace(item.AccountName) == "" {
			if name := accountNames[*item.AccountID]; name != "" {
				item.AccountName = name
			}
		}
		if item.GroupID != nil && *item.GroupID > 0 && strings.TrimSpace(item.GroupName) == "" {
			if name := groupNames[*item.GroupID]; name != "" {
				item.GroupName = name
			}
		}
	}
}

func formatOpsWindowLabel(windowMinutes int, locale string) string {
	if windowMinutes <= 0 {
		windowMinutes = 1
	}
	if isChineseOpsLocale(locale) {
		return fmt.Sprintf("最近 %d 分钟", windowMinutes)
	}
	return fmt.Sprintf("Last %d min", windowMinutes)
}

func opsPhaseLabel(phase, locale string) string {
	p := strings.ToLower(strings.TrimSpace(phase))
	if isChineseOpsLocale(locale) {
		switch p {
		case "request":
			return "请求校验"
		case "auth":
			return "用户鉴权"
		case "account_auth":
			return "账号授权"
		case "routing":
			return "路由选号"
		case "upstream":
			return "上游调用"
		case "network":
			return "网络连接"
		case "internal":
			return "内部处理"
		default:
			if p == "" {
				return "未知"
			}
			return p
		}
	}
	if p == "" {
		return "unknown"
	}
	return p
}

func redactOpsErrorSnippet(raw string, maxRunes int) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = opsRedactBearerPattern.ReplaceAllString(s, "${1}[REDACTED]")
	s = opsRedactKeyPattern.ReplaceAllString(s, "[REDACTED_KEY]")
	s = opsRedactQueryPattern.ReplaceAllString(s, "${1}[REDACTED]")
	s = opsRedactEmailPattern.ReplaceAllString(s, "[REDACTED_EMAIL]")
	s = opsSpaceCollapse.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	if maxRunes > 0 {
		runes := []rune(s)
		if len(runes) > maxRunes {
			return string(runes[:maxRunes]) + "..."
		}
	}
	return s
}

func translateOpsErrorReason(statusCode int, phase, errType, msg, locale string) (reasonKey, reason, suggestion string) {
	zh := isChineseOpsLocale(locale)
	lowerMsg := strings.ToLower(strings.TrimSpace(msg))
	lowerPhase := strings.ToLower(strings.TrimSpace(phase))
	lowerType := strings.ToLower(strings.TrimSpace(errType))

	switch {
	case statusCode == 524 || statusCode == 504 || strings.Contains(lowerMsg, "524") || strings.Contains(lowerMsg, "timeout") || strings.Contains(lowerMsg, "deadline exceeded"):
		if zh {
			return "upstream_timeout",
				fmt.Sprintf("上游服务响应超时（%d），长时间未返回结果", positiveIntOrDefault(statusCode, 524)),
				"检查对应账号的线路或代理延迟；若反复出现，可暂时停用该账号或切换备用线路"
		}
		return "upstream_timeout",
			fmt.Sprintf("Upstream response timed out (%d)", positiveIntOrDefault(statusCode, 524)),
			"Check upstream/proxy latency for the affected account and fail over if it persists"

	case statusCode == 499:
		if zh {
			return "client_canceled",
				"请求等待期间客户端提前断开连接（499）",
				"可能是用户主动取消或上游响应偏慢导致客户端超时放弃，少量属正常现象"
		}
		return "client_canceled",
			"Client closed connection before receiving response (499)",
			"Usually client cancellation or slow upstream causing client timeout"

	case statusCode == 503 || strings.Contains(lowerMsg, "no available compatible accounts") || strings.Contains(lowerMsg, "all available accounts exhausted"):
		if zh {
			return "no_available_account",
				"对应分组/模型下暂无可调度的健康账号（503）",
				"立即到后台查看该模型所属分组的账号池，检查是否全部限流、停用或额度耗尽"
		}
		return "no_available_account",
			"No healthy compatible upstream accounts available (503)",
			"Inspect the target group's account pool for rate limits, errors, or exhausted quotas"

	case statusCode == 502 || strings.Contains(lowerMsg, "bad gateway") || strings.Contains(lowerMsg, "service is busy"):
		if zh {
			return "upstream_502",
				"上游服务商接口繁忙或网关异常（502）",
				"多为上游瞬时波动；若连续出现超过 5 分钟，建议检查账号线路或临时切流"
		}
		return "upstream_502",
			"Upstream gateway error or service busy (502)",
			"Often transient upstream jitter; switch accounts or routes if sustained"

	case statusCode == 529 || strings.Contains(lowerMsg, "overloaded"):
		if zh {
			return "upstream_overloaded",
				"上游官网当前过载限流（529）",
				"属于上游高峰期官方限流，系统会自动故障转移，可补充备用渠道分散压力"
		}
		return "upstream_overloaded",
			"Upstream provider overloaded (529)",
			"Provider peak-load throttling; ensure backup accounts/channels are active"

	case statusCode == 429 || strings.Contains(lowerType, "rate_limit") || strings.Contains(lowerMsg, "daily_limit") || strings.Contains(lowerMsg, "rate limit"):
		if strings.Contains(lowerMsg, "daily") || strings.Contains(lowerMsg, "quota") {
			if zh {
				return "daily_quota_limit",
					"触发每日使用额度或配额上限（429）",
					"核实是上游账号日限额已满，还是平台侧并发/日消费策略限制"
			}
			return "daily_quota_limit",
				"Daily usage quota/limit exceeded (429)",
				"Check whether the upstream account quota or group daily limit is exhausted"
		}
		if zh {
			return "rate_limit",
				"触发上游请求频率/并发限制（429）",
				"可增加同分组可用账号数量，或适当降低单号并发上限"
		}
		return "rate_limit",
			"Upstream rate limit triggered (429)",
			"Add more accounts to the pool or lower per-account concurrency"

	case statusCode == 401 || strings.Contains(lowerMsg, "invalid token") || strings.Contains(lowerMsg, "authentication failed") || strings.Contains(lowerMsg, "unauthorized"):
		if zh {
			return "upstream_auth",
				"上游账号凭据失效或鉴权失败（401）",
				"尽快到后台核对对应账号的 API Key / OAuth 授权是否过期、被封禁或需要重新登录"
		}
		return "upstream_auth",
			"Upstream account credential invalid or expired (401)",
			"Refresh OAuth credentials or replace the API key for the affected account"

	case statusCode == 403:
		if strings.Contains(lowerMsg, "balance") || strings.Contains(lowerMsg, "insufficient") || strings.Contains(lowerMsg, "credit") {
			if zh {
				return "insufficient_balance",
					"余额或上游账号额度不足（403）",
					"检查对应上游账号余额是否耗尽（或用户自身余额不足）"
			}
			return "insufficient_balance",
				"Insufficient upstream account or user balance (403)",
				"Check upstream account credits and top up or pause depleted accounts"
		}
		if zh {
			return "upstream_forbidden",
				"上游拒绝访问或分组/权限配置异常（403）",
				"核查对应账号状态、API Key 所属分组及上游访问权限是否正常"
		}
		return "upstream_forbidden",
			"Upstream access forbidden or permission error (403)",
			"Verify account permissions, group binding, and IP/key access policy"

	case statusCode == 404 || strings.Contains(lowerMsg, "not supported") || strings.Contains(lowerMsg, "unknown provider"):
		if strings.Contains(lowerMsg, "model") || strings.Contains(lowerMsg, "provider") || strings.Contains(lowerMsg, "not supported") {
			if zh {
				return "model_route_mismatch",
					fmt.Sprintf("分组/账号未支持或未映射所请求的模型（%d）", positiveIntOrDefault(statusCode, 404)),
					"到后台检查分组的模型白名单以及账号的 model_mapping 别名配置"
			}
			return "model_route_mismatch",
				fmt.Sprintf("Model unsupported or unmapped on selected account/group (%d)", positiveIntOrDefault(statusCode, 404)),
				"Check group allowlist and account model_mapping configuration"
		}
		if zh {
			return "not_found",
				"请求的接口地址或资源不存在（404）",
				"核对上游 Base URL 与协议路径配置是否正确"
		}
		return "not_found",
			"Requested endpoint or resource not found (404)",
			"Verify upstream base URL and endpoint path configuration"

	case statusCode == 400 || lowerType == "invalid_request_error":
		if strings.Contains(lowerMsg, "policy") || strings.Contains(lowerMsg, "violate") || lowerType == "cyber_policy" {
			if zh {
				return "policy_blocked",
					"请求内容被上游安全/合规策略拦截（400）",
					"属于单次内容合规拦截，一般不是服务本身故障"
			}
			return "policy_blocked",
				"Request blocked by upstream safety/usage policy (400)",
				"Content policy rejection on specific prompt; usually not an outage"
		}
		if zh {
			return "invalid_request",
				"客户端请求参数或协议格式不兼容（400）",
				"多为调用方传参问题，可在运维日志查看具体模型和客户端参数"
		}
		return "invalid_request",
			"Invalid client request parameters (400)",
			"Check client payload compatibility for the target model"

	case lowerPhase == "network" || strings.Contains(lowerMsg, "socks") || strings.Contains(lowerMsg, "dial tcp") || strings.Contains(lowerMsg, "connection reset") || strings.Contains(lowerMsg, "eof"):
		if zh {
			return "network_proxy",
				"代理节点或上游网络连接中断",
				"检查绑定代理节点是否在线，或更换更稳定的代理线路"
		}
		return "network_proxy",
			"Proxy or upstream network connection failure",
			"Check bound proxy connectivity or switch to a healthier proxy"

	case statusCode >= 500 || lowerPhase == "internal":
		if zh {
			return "internal_error",
				fmt.Sprintf("服务内部处理异常（%d）", positiveIntOrDefault(statusCode, 500)),
				"前往后台运维监控查看系统日志与错误堆栈排查原因"
		}
		return "internal_error",
			fmt.Sprintf("Internal gateway processing error (%d)", positiveIntOrDefault(statusCode, 500)),
			"Inspect ops system logs and error details in admin dashboard"

	default:
		if zh {
			return "other",
				fmt.Sprintf("请求处理异常（状态码 %d，阶段：%s）", statusCode, opsPhaseLabel(phase, locale)),
				"点击下方按钮前往后台运维监控查看详细报错上下文"
		}
		return "other",
			fmt.Sprintf("Request error (status %d, phase %s)", statusCode, opsPhaseLabel(phase, locale)),
			"Open the Ops Dashboard to inspect full error context"
	}
}

func positiveIntOrDefault(a, fallback int) int {
	if a > 0 {
		return a
	}
	return fallback
}

func (s *OpsService) ResolveAdminOpsURL(ctx context.Context) string {
	base := ""
	if s != nil && s.settingRepo != nil {
		for _, key := range []string{SettingKeyFrontendURL, SettingKeyAPIBaseURL} {
			val, err := s.settingRepo.GetValue(ctx, key)
			if err == nil && strings.TrimSpace(val) != "" {
				candidate := strings.TrimRight(strings.TrimSpace(val), "/")
				if isValidOpsHTTPURL(candidate) {
					base = candidate
					break
				}
			}
		}
	}
	if base == "" {
		return ""
	}
	return base + "/admin/ops"
}

func (s *OpsAlertEvaluatorService) collectOpsAlertEvidence(
	ctx context.Context,
	rule *OpsAlertRule,
	event *OpsAlertEvent,
	snap opsAlertMetricSnapshot,
	windowStart time.Time,
	windowEnd time.Time,
	platform string,
	groupID *int64,
	locale string,
) *opsAlertEvidence {
	if ctx == nil {
		ctx = context.Background()
	}
	ev := &opsAlertEvidence{
		Locale:         locale,
		ScopeLabel:     formatOpsScopeLabel(platform, groupID, locale),
		WindowMinutes:  1,
		WindowLabel:    formatOpsWindowLabel(1, locale),
		DashboardURL:   "",
		TopClusters:    []opsAlertErrorCluster{},
		Suggestions:    []string{},
	}
	if s != nil && s.opsService != nil {
		ev.DashboardURL = s.opsService.ResolveAdminOpsURL(ctx)
	}
	var cfg *config.Config
	if s != nil {
		cfg = s.cfg
	}
	if rule != nil {
		ev.RuleID = rule.ID
		ev.RuleName = strings.TrimSpace(rule.Name)
		ev.Severity = strings.TrimSpace(rule.Severity)
		ev.MetricType = strings.TrimSpace(rule.MetricType)
		ev.Operator = strings.TrimSpace(rule.Operator)
		if rule.WindowMinutes > 0 {
			ev.WindowMinutes = rule.WindowMinutes
			ev.WindowLabel = formatOpsWindowLabel(rule.WindowMinutes, locale)
		}
		thr := rule.Threshold
		ev.ThresholdValue = &thr
	}
	if event != nil {
		if strings.TrimSpace(event.Severity) != "" {
			ev.Severity = strings.TrimSpace(event.Severity)
		}
		ev.Status = strings.TrimSpace(event.Status)
		ev.MetricValue = event.MetricValue
		if event.ThresholdValue != nil {
			ev.ThresholdValue = event.ThresholdValue
		}
		ev.FiredAt = event.FiredAt
		ev.FiredAtLocal = formatOpsLocalTime(event.FiredAt, cfg, locale)
		ev.ResolvedAt = event.ResolvedAt
		if event.ResolvedAt != nil && !event.ResolvedAt.IsZero() {
			ev.ResolvedAtLocal = formatOpsLocalTime(*event.ResolvedAt, cfg, locale)
			if !event.FiredAt.IsZero() && event.ResolvedAt.After(event.FiredAt) {
				ev.DurationHuman = formatOpsDurationHuman(event.ResolvedAt.Sub(event.FiredAt), locale)
			}
		} else if ev.WindowMinutes > 0 {
			ev.DurationHuman = formatOpsDurationHuman(time.Duration(ev.WindowMinutes)*time.Minute, locale)
		}
	}
	if ev.DurationHuman == "" && ev.WindowMinutes > 0 {
		ev.DurationHuman = formatOpsDurationHuman(time.Duration(ev.WindowMinutes)*time.Minute, locale)
	}
	ev.SeverityLabel = opsSeverityLabel(ev.Severity, locale)
	ev.StatusLabel = opsStatusLabel(ev.Status, locale)
	ev.MetricLabel, ev.MetricUnit = opsMetricMeta(ev.MetricType, locale)
	ev.MetricSummary = formatOpsMetricSummary(ev.MetricType, ev.Operator, ev.MetricValue, ev.ThresholdValue, locale)

	if snap.Overview != nil {
		ev.ShowRequestOverview = true
		ev.TotalRequests = snap.Overview.RequestCountTotal
		ev.SLARequests = snap.Overview.RequestCountSLA
		ev.SuccessCount = snap.Overview.SuccessCount
		ev.ErrorCount = snap.Overview.ErrorCountSLA
		ev.BusinessLimitedCount = snap.Overview.BusinessLimitedCount
		if snap.Overview.RequestCountSLA > 0 && snap.Overview.RequestCountSLA < 10 {
			ev.LowSampleWarning = true
		}
	}

	if snap.SystemMetrics != nil {
		ev.SystemSummary = formatOpsSystemSnapshotSummary(snap.SystemMetrics, locale)
	}
	if snap.Availability != nil {
		ev.AccountPoolSummary = formatOpsAvailabilitySnapshotSummary(snap.Availability, locale)
	}

	scopeGroupName := ""
	if groupID != nil && *groupID > 0 && snap.Availability != nil {
		if snap.Availability.Group != nil && snap.Availability.Group.GroupID == *groupID && strings.TrimSpace(snap.Availability.Group.GroupName) != "" {
			scopeGroupName = strings.TrimSpace(snap.Availability.Group.GroupName)
		} else {
			for _, acc := range snap.Availability.Accounts {
				if acc != nil && acc.GroupID == *groupID && strings.TrimSpace(acc.GroupName) != "" {
					scopeGroupName = strings.TrimSpace(acc.GroupName)
					break
				}
			}
		}
	}

	// Best-effort query recent error logs in the triggering window (or incident span when resolved) to build concrete incident context.
	if s != nil && s.opsRepo != nil {
		st := windowStart
		et := windowEnd
		if (ev.Status == OpsAlertStatusResolved || ev.Status == OpsAlertStatusManualResolved) && event != nil && event.ResolvedAt != nil && !event.ResolvedAt.IsZero() {
			et = *event.ResolvedAt
			winDur := time.Duration(ev.WindowMinutes) * time.Minute
			if winDur <= 0 {
				winDur = time.Minute
			}
			earliest := et.Add(-60 * time.Minute)
			if !event.FiredAt.IsZero() {
				st = event.FiredAt.Add(-winDur)
				if st.Before(earliest) {
					st = earliest
				}
			} else {
				st = earliest
			}
		}
		if !st.IsZero() && !et.IsZero() {
			subCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			if !et.After(st) {
				et = st.Add(time.Duration(ev.WindowMinutes) * time.Minute)
			}
			// Add a tiny 1-second cushion on end time so errors written right before safeEnd truncation are captured.
			if ev.Status != OpsAlertStatusResolved && ev.Status != OpsAlertStatusManualResolved && event != nil && !event.FiredAt.IsZero() && event.FiredAt.After(et) {
				et = event.FiredAt.Add(time.Second)
			}
			list, err := s.opsRepo.ListErrorLogs(subCtx, &OpsErrorLogFilter{
				StartTime:          &st,
				EndTime:            &et,
				Platform:           strings.TrimSpace(platform),
				GroupID:            groupID,
				ExcludeCountTokens: true,
				View:               "errors",
				Page:               1,
				PageSize:           200,
				SortBy:             "created_at",
				SortOrder:          "desc",
			})
			if err == nil && list != nil && len(list.Errors) > 0 {
				enrichOpsErrorLogNames(subCtx, s.opsService, snap.Availability, list.Errors)
				if scopeGroupName == "" && groupID != nil && *groupID > 0 {
					for _, item := range list.Errors {
						if item != nil && item.GroupID != nil && *item.GroupID == *groupID && strings.TrimSpace(item.GroupName) != "" {
							scopeGroupName = strings.TrimSpace(item.GroupName)
							break
						}
					}
				}
				clusters, usersCount := clusterOpsErrorLogs(list.Errors, locale, 5)
				ev.TopClusters = clusters
				ev.AffectedUsers = usersCount
			}
		}
	}

	if scopeGroupName == "" && groupID != nil && *groupID > 0 && s != nil && s.opsService != nil {
		subCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if avail, err := s.opsService.GetAccountAvailability(subCtx, platform, groupID); err == nil && avail != nil {
			if avail.Group != nil && strings.TrimSpace(avail.Group.GroupName) != "" {
				scopeGroupName = strings.TrimSpace(avail.Group.GroupName)
			}
		}
	}
	ev.ScopeLabel = formatOpsScopeLabelWithGroupName(platform, groupID, scopeGroupName, locale)

	ev.FallbackSummary, ev.Suggestions = buildRuleBasedAlertDiagnosis(ev, locale)
	return ev
}

func clusterOpsErrorLogs(errorsList []*OpsErrorLog, locale string, topN int) ([]opsAlertErrorCluster, int) {
	if len(errorsList) == 0 {
		return nil, 0
	}
	if topN <= 0 {
		topN = 5
	}
	userSet := make(map[int64]struct{})
	type clusterKey struct {
		platform   string
		model      string
		group      string
		account    string
		statusCode int
		reasonKey  string
	}
	grouped := make(map[clusterKey]*opsAlertErrorCluster)
	for _, item := range errorsList {
		if item == nil {
			continue
		}
		if item.UserID != nil && *item.UserID > 0 {
			userSet[*item.UserID] = struct{}{}
		}
		platform := strings.TrimSpace(item.Platform)
		if platform == "" {
			platform = "-"
		}
		model := strings.TrimSpace(item.RequestedModel)
		if model == "" {
			model = strings.TrimSpace(item.Model)
		}
		if model == "" {
			model = "-"
		}
		groupLabel, accountLabel := opsErrorLogGroupAndAccountLabels(item, locale)
		reasonKey, reason, suggestion := translateOpsErrorReason(item.StatusCode, item.Phase, item.Type, item.Message, locale)
		k := clusterKey{
			platform:   platform,
			model:      model,
			group:      groupLabel,
			account:    accountLabel,
			statusCode: item.StatusCode,
			reasonKey:  reasonKey,
		}
		existing, ok := grouped[k]
		if !ok {
			grouped[k] = &opsAlertErrorCluster{
				Platform:      platform,
				Model:         model,
				GroupLabel:    groupLabel,
				AccountLabel:  accountLabel,
				StatusCode:    item.StatusCode,
				Phase:         item.Phase,
				Count:         1,
				ReasonKey:     reasonKey,
				Reason:        reason,
				Suggestion:    suggestion,
				SampleMessage: redactOpsErrorSnippet(item.Message, 120),
			}
			continue
		}
		existing.Count++
		if existing.SampleMessage == "" {
			existing.SampleMessage = redactOpsErrorSnippet(item.Message, 120)
		}
	}

	out := make([]opsAlertErrorCluster, 0, len(grouped))
	for _, c := range grouped {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		if out[i].Platform != out[j].Platform {
			return out[i].Platform < out[j].Platform
		}
		return out[i].Model < out[j].Model
	})
	if len(out) > topN {
		out = out[:topN]
	}
	return out, len(userSet)
}

func formatOpsSystemSnapshotSummary(m *OpsSystemMetricsSnapshot, locale string) string {
	if m == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	zh := isChineseOpsLocale(locale)
	if m.CPUUsagePercent != nil {
		if zh {
			parts = append(parts, fmt.Sprintf("CPU %.1f%%", *m.CPUUsagePercent))
		} else {
			parts = append(parts, fmt.Sprintf("CPU %.1f%%", *m.CPUUsagePercent))
		}
	}
	if m.MemoryUsagePercent != nil {
		if zh {
			parts = append(parts, fmt.Sprintf("内存 %.1f%%", *m.MemoryUsagePercent))
		} else {
			parts = append(parts, fmt.Sprintf("Memory %.1f%%", *m.MemoryUsagePercent))
		}
	}
	if m.ConcurrencyQueueDepth != nil {
		if zh {
			parts = append(parts, fmt.Sprintf("排队 %d", *m.ConcurrencyQueueDepth))
		} else {
			parts = append(parts, fmt.Sprintf("Queue %d", *m.ConcurrencyQueueDepth))
		}
	}
	if m.DBOK != nil && !*m.DBOK {
		if zh {
			parts = append(parts, "数据库异常")
		} else {
			parts = append(parts, "DB unhealthy")
		}
	}
	if m.RedisOK != nil && !*m.RedisOK {
		if zh {
			parts = append(parts, "Redis 异常")
		} else {
			parts = append(parts, "Redis unhealthy")
		}
	}
	return strings.Join(parts, " · ")
}

func formatOpsNamedAccountList(names []string, locale string, maxShow int) string {
	if len(names) == 0 {
		return ""
	}
	if maxShow <= 0 {
		maxShow = 5
	}
	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Strings(sorted)
	zh := isChineseOpsLocale(locale)
	sep := ", "
	if zh {
		sep = "、"
	}
	if len(sorted) <= maxShow {
		return strings.Join(sorted, sep)
	}
	shown := strings.Join(sorted[:maxShow], sep)
	if zh {
		return fmt.Sprintf("%s 等 %d 个", shown, len(sorted))
	}
	return fmt.Sprintf("%s (+%d more)", shown, len(sorted)-maxShow)
}

func formatOpsAvailabilitySnapshotSummary(avail *OpsAccountAvailability, locale string) string {
	if avail == nil {
		return ""
	}
	zh := isChineseOpsLocale(locale)
	total, healthy, rateLimited, errored, overloaded := 0, 0, 0, 0, 0
	erroredNames := make([]string, 0, 4)
	limitedNames := make([]string, 0, 4)
	for _, acc := range avail.Accounts {
		if acc == nil {
			continue
		}
		total++
		accName := strings.TrimSpace(acc.AccountName)
		if accName == "" && acc.AccountID > 0 {
			if zh {
				accName = fmt.Sprintf("未命名账号（编号 %d）", acc.AccountID)
			} else {
				accName = fmt.Sprintf("Unnamed account (ID %d)", acc.AccountID)
			}
		}
		if acc.IsAvailable {
			healthy++
		}
		if acc.IsRateLimited {
			rateLimited++
		}
		if acc.HasError {
			errored++
			if accName != "" {
				erroredNames = append(erroredNames, accName)
			}
		} else if acc.IsRateLimited || acc.IsOverloaded {
			if accName != "" {
				limitedNames = append(limitedNames, accName)
			}
		}
		if acc.IsOverloaded {
			overloaded++
		}
	}
	if total == 0 {
		return ""
	}
	if zh {
		base := fmt.Sprintf("账号总计 %d 个 · 正常可用 %d 个 · 限流 %d 个 · 过载 %d 个 · 异常 %d 个", total, healthy, rateLimited, overloaded, errored)
		if len(erroredNames) > 0 {
			base += "｜异常账号：" + formatOpsNamedAccountList(erroredNames, locale, 5)
		}
		if len(limitedNames) > 0 {
			base += "｜限流/过载账号：" + formatOpsNamedAccountList(limitedNames, locale, 5)
		}
		return base
	}
	base := fmt.Sprintf("Total %d accounts · Available %d · Rate-limited %d · Overloaded %d · Error %d", total, healthy, rateLimited, overloaded, errored)
	if len(erroredNames) > 0 {
		base += " | Errored: " + formatOpsNamedAccountList(erroredNames, locale, 5)
	}
	if len(limitedNames) > 0 {
		base += " | Rate-limited/Overloaded: " + formatOpsNamedAccountList(limitedNames, locale, 5)
	}
	return base
}

func buildRuleBasedAlertDiagnosis(ev *opsAlertEvidence, locale string) (string, []string) {
	if ev == nil {
		return "", nil
	}
	zh := isChineseOpsLocale(locale)
	suggestions := make([]string, 0, 3)
	seenSugg := make(map[string]struct{})
	addSugg := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seenSugg[s]; ok {
			return
		}
		seenSugg[s] = struct{}{}
		suggestions = append(suggestions, s)
	}

	if ev.Status == OpsAlertStatusResolved || ev.Status == OpsAlertStatusManualResolved {
		for _, c := range ev.TopClusters {
			addSugg(c.Suggestion)
			if len(suggestions) >= 2 {
				break
			}
		}
		if zh {
			durText := ev.DurationHuman
			if durText == "" {
				durText = "刚才"
			}
			summary := fmt.Sprintf("此前「%s」告警现已恢复正常（本次持续 %s）。%s。",
				ev.RuleName, durText, ev.MetricSummary)
			if len(ev.TopClusters) > 0 {
				top := ev.TopClusters[0]
				if top.AccountLabel != "" && top.AccountLabel != "-" {
					summary += fmt.Sprintf(" 故障期间最集中的报错来自账号「%s」的 %s（%s）。", top.AccountLabel, top.Model, top.Reason)
				} else {
					summary += fmt.Sprintf(" 故障期间最集中的报错为 %s/%s（%s）。", top.Platform, top.Model, top.Reason)
				}
			}
			return summary, suggestions
		}
		durText := ev.DurationHuman
		if durText == "" {
			durText = "a short period"
		}
		return fmt.Sprintf("Alert \"%s\" has recovered after %s. %s.", ev.RuleName, durText, ev.MetricSummary), suggestions
	}

	if len(ev.TopClusters) > 0 {
		top := ev.TopClusters[0]
		addSugg(top.Suggestion)
		for _, c := range ev.TopClusters[1:] {
			addSugg(c.Suggestion)
			if len(suggestions) >= 3 {
				break
			}
		}
		if zh {
			var b strings.Builder
			if ev.ShowRequestOverview && ev.SLARequests > 0 {
				b.WriteString(fmt.Sprintf("%s内共处理 %d 次有效请求，其中成功 %d 次、失败 %d 次",
					ev.WindowLabel, ev.SLARequests, ev.SuccessCount, ev.ErrorCount))
				if ev.AffectedUsers > 0 {
					b.WriteString(fmt.Sprintf("（影响 %d 位用户）", ev.AffectedUsers))
				}
				b.WriteString("。")
			} else {
				b.WriteString(fmt.Sprintf("%s触发「%s」告警，%s。", ev.WindowLabel, ev.RuleName, ev.MetricSummary))
			}
			if ev.LowSampleWarning {
				b.WriteString(" 注意：当前统计窗口请求量很少，可能属于单次偶发波动，建议先观察。")
			}
			b.WriteString(fmt.Sprintf(" 最主要原因为 %s 的「%s」出现 %d 次报错：%s。",
				top.Platform, top.Model, top.Count, top.Reason))
			if top.AccountLabel != "" && top.AccountLabel != "-" {
				if top.GroupLabel != "" && top.GroupLabel != "-" {
					b.WriteString(fmt.Sprintf(" 主要集中在账号「%s」（所属分组：%s）。", top.AccountLabel, top.GroupLabel))
				} else {
					b.WriteString(fmt.Sprintf(" 主要集中在账号「%s」。", top.AccountLabel))
				}
			} else if top.GroupLabel != "" && top.GroupLabel != "-" {
				b.WriteString(fmt.Sprintf(" 主要集中在分组「%s」。", top.GroupLabel))
			}
			return b.String(), suggestions
		}

		var b strings.Builder
		if ev.ShowRequestOverview && ev.SLARequests > 0 {
			b.WriteString(fmt.Sprintf("Over the %s, %d of %d requests failed (%d succeeded",
				strings.ToLower(ev.WindowLabel), ev.ErrorCount, ev.SLARequests, ev.SuccessCount))
			if ev.AffectedUsers > 0 {
				b.WriteString(fmt.Sprintf(", affecting %d user(s)", ev.AffectedUsers))
			}
			b.WriteString(").")
		} else {
			b.WriteString(fmt.Sprintf("Rule \"%s\" fired: %s.", ev.RuleName, ev.MetricSummary))
		}
		if ev.LowSampleWarning {
			b.WriteString(" Note: request volume is very small, so this may be transient jitter.")
		}
		b.WriteString(fmt.Sprintf(" Top issue: %s/%s (%d errors) — %s.", top.Platform, top.Model, top.Count, top.Reason))
		if top.AccountLabel != "" && top.AccountLabel != "-" {
			if top.GroupLabel != "" && top.GroupLabel != "-" {
				b.WriteString(fmt.Sprintf(" Concentrated on account \"%s\" (group \"%s\").", top.AccountLabel, top.GroupLabel))
			} else {
				b.WriteString(fmt.Sprintf(" Concentrated on account \"%s\".", top.AccountLabel))
			}
		} else if top.GroupLabel != "" && top.GroupLabel != "-" {
			b.WriteString(fmt.Sprintf(" Concentrated in group \"%s\".", top.GroupLabel))
		}
		return b.String(), suggestions
	}

	if ev.AccountPoolSummary != "" {
		if zh {
			addSugg("前往后台号池/账号管理检查异常或被限流的账号")
			return fmt.Sprintf("当前检测到「%s」异常：%s。目前账号池状态：%s。", ev.RuleName, ev.MetricSummary, ev.AccountPoolSummary), suggestions
		}
		addSugg("Inspect rate-limited or errored accounts in Admin > Accounts")
		return fmt.Sprintf("Alert \"%s\" triggered (%s). Pool status: %s.", ev.RuleName, ev.MetricSummary, ev.AccountPoolSummary), suggestions
	}

	if ev.SystemSummary != "" {
		if zh {
			addSugg("登录服务器或查看后台系统指标，确认资源占用及进程负载")
			return fmt.Sprintf("系统资源告警「%s」：%s。当前实例状态：%s。", ev.RuleName, ev.MetricSummary, ev.SystemSummary), suggestions
		}
		addSugg("Check server CPU, memory, and queue depth on the Ops Dashboard")
		return fmt.Sprintf("System alert \"%s\" triggered (%s). Instance state: %s.", ev.RuleName, ev.MetricSummary, ev.SystemSummary), suggestions
	}

	if zh {
		addSugg("前往后台运维监控查看对应时间段的请求与错误日志明细")
		return fmt.Sprintf("%s触发「%s」告警（%s）。范围内：%s。", ev.WindowLabel, ev.RuleName, ev.MetricSummary, ev.ScopeLabel), suggestions
	}
	addSugg("Open the Ops Dashboard to check recent logs for this time window")
	return fmt.Sprintf("\"%s\" triggered (%s) for %s.", ev.RuleName, ev.MetricSummary, ev.ScopeLabel), suggestions
}

func buildOpsAlertHeadline(ev *opsAlertEvidence, locale string) string {
	if ev == nil {
		return ""
	}
	zh := isChineseOpsLocale(locale)
	if ev.Status == OpsAlertStatusResolved || ev.Status == OpsAlertStatusManualResolved {
		if zh {
			return fmt.Sprintf("「%s」已恢复正常", ev.RuleName)
		}
		return fmt.Sprintf("\"%s\" recovered", ev.RuleName)
	}
	if ev.ShowRequestOverview && ev.SLARequests > 0 {
		if zh {
			base := fmt.Sprintf("%s %d 次请求失败 %d 次", ev.WindowLabel, ev.SLARequests, ev.ErrorCount)
			if len(ev.TopClusters) > 0 {
				top := ev.TopClusters[0]
				if top.AccountLabel != "" && top.AccountLabel != "-" {
					return fmt.Sprintf("%s（账号「%s」· %s）", base, top.AccountLabel, top.Model)
				}
				return fmt.Sprintf("%s（%s · %s）", base, top.Platform, top.Model)
			}
			return base
		}
		if len(ev.TopClusters) > 0 {
			top := ev.TopClusters[0]
			if top.AccountLabel != "" && top.AccountLabel != "-" {
				return fmt.Sprintf("%d/%d requests failed (%s · %s)", ev.ErrorCount, ev.SLARequests, top.AccountLabel, top.Model)
			}
		}
		return fmt.Sprintf("%d/%d requests failed (%s)", ev.ErrorCount, ev.SLARequests, ev.WindowLabel)
	}
	if zh {
		return fmt.Sprintf("%s：%s", ev.RuleName, ev.MetricSummary)
	}
	return fmt.Sprintf("%s: %s", ev.RuleName, ev.MetricSummary)
}

func buildOpsAlertEmailSubject(ev *opsAlertEvidence, locale string) string {
	if ev == nil {
		return "[Ops Alert]"
	}
	tag := opsSeverityShortTag(ev.Severity, ev.Status, locale)
	headline := buildOpsAlertHeadline(ev, locale)
	if isChineseOpsLocale(locale) {
		return sanitizeEmailHeader(fmt.Sprintf("[GABUGU %s] %s", tag, headline))
	}
	return sanitizeEmailHeader(fmt.Sprintf("[Ops %s] %s", tag, headline))
}

func buildOpsAlertDetailsHTML(ev *opsAlertEvidence, locale string) string {
	if ev == nil {
		return ""
	}
	zh := isChineseOpsLocale(locale)
	var b strings.Builder

	if (ev.Status == OpsAlertStatusResolved || ev.Status == OpsAlertStatusManualResolved) && ev.ResolvedAtLocal != "" {
		durPart := ""
		if ev.DurationHuman != "" {
			if zh {
				durPart = fmt.Sprintf("，本次共持续 %s", html.EscapeString(ev.DurationHuman))
			} else {
				durPart = fmt.Sprintf(", total duration %s", html.EscapeString(ev.DurationHuman))
			}
		}
		b.WriteString(`<div style="margin:0 0 16px;padding:12px 14px;border-radius:8px;background:#f0fdf4;border:1px solid #bbf7d0;color:#166534;font-size:13px;line-height:1.6;">`)
		if zh {
			b.WriteString(fmt.Sprintf(`<strong>恢复确认：</strong>指标已恢复正常，恢复时间：%s%s。`, html.EscapeString(ev.ResolvedAtLocal), durPart))
		} else {
			b.WriteString(fmt.Sprintf(`<strong>Recovery Confirmed:</strong> Metric returned to normal at %s%s.`, html.EscapeString(ev.ResolvedAtLocal), durPart))
		}
		b.WriteString(`</div>`)
	}

	if ev.LowSampleWarning {
		b.WriteString(`<div style="margin:16px 0;padding:12px 14px;border-radius:8px;background:#fffbeb;border:1px solid #fde68a;color:#92400e;font-size:13px;line-height:1.6;">`)
		if zh {
			b.WriteString(`<strong>低流量提示：</strong>统计窗口内有效请求数较少（仅 ` + fmt.Sprintf("%d", ev.SLARequests) + ` 次），少量报错就会拉高百分比，可能是单次上游抖动，建议结合下方明细判断是否需要介入。`)
		} else {
			b.WriteString(`<strong>Low-sample notice:</strong> Only ` + fmt.Sprintf("%d", ev.SLARequests) + ` SLA requests occurred in this window. Small sample sizes amplify percentages.`)
		}
		b.WriteString(`</div>`)
	}

	if ev.ShowRequestOverview {
		b.WriteString(`<table role="presentation" style="width:100%;border-collapse:separate;border-spacing:8px;margin:12px -8px 16px;"><tr>`)
		cells := []struct {
			label string
			val   string
			color string
		}{}
		if zh {
			cells = []struct {
				label string
				val   string
				color string
			}{
				{"有效请求数", formatOpsReportInteger(ev.SLARequests), "#0f172a"},
				{"成功请求", formatOpsReportInteger(ev.SuccessCount), "#15803d"},
				{"失败请求", formatOpsReportInteger(ev.ErrorCount), "#b91c1c"},
				{"受影响用户", fmt.Sprintf("%d 位", ev.AffectedUsers), "#0f172a"},
			}
		} else {
			cells = []struct {
				label string
				val   string
				color string
			}{
				{"SLA Requests", formatOpsReportInteger(ev.SLARequests), "#0f172a"},
				{"Succeeded", formatOpsReportInteger(ev.SuccessCount), "#15803d"},
				{"Failed", formatOpsReportInteger(ev.ErrorCount), "#b91c1c"},
				{"Affected Users", fmt.Sprintf("%d", ev.AffectedUsers), "#0f172a"},
			}
		}
		for i, cell := range cells {
			if i == 2 {
				b.WriteString(`</tr><tr>`)
			}
			b.WriteString(fmt.Sprintf(`<td style="width:50%%;padding:12px 14px;border:1px solid #e2e8f0;border-radius:8px;background:#f8fafc;vertical-align:top;"><div style="font-size:12px;color:#64748b;margin-bottom:4px;">%s</div><div style="font-size:18px;font-weight:700;color:%s;">%s</div></td>`,
				html.EscapeString(cell.label), cell.color, html.EscapeString(cell.val)))
		}
		b.WriteString(`</tr></table>`)
	}

	if ev.AccountPoolSummary != "" {
		title := "Account Pool Snapshot"
		if zh {
			title = "账号池状态"
		}
		b.WriteString(fmt.Sprintf(`<div style="margin:14px 0;padding:12px 14px;border-radius:8px;background:#f8fafc;border:1px solid #e2e8f0;font-size:13px;line-height:1.6;"><div style="font-weight:700;color:#0f172a;margin-bottom:4px;">%s</div><div style="color:#334155;">%s</div></div>`,
			html.EscapeString(title), html.EscapeString(ev.AccountPoolSummary)))
	}

	if ev.SystemSummary != "" {
		title := "Instance Resources"
		if zh {
			title = "实例资源状态"
		}
		b.WriteString(fmt.Sprintf(`<div style="margin:14px 0;padding:12px 14px;border-radius:8px;background:#f8fafc;border:1px solid #e2e8f0;font-size:13px;line-height:1.6;"><div style="font-weight:700;color:#0f172a;margin-bottom:4px;">%s</div><div style="color:#334155;">%s</div></div>`,
			html.EscapeString(title), html.EscapeString(ev.SystemSummary)))
	}

	if len(ev.TopClusters) > 0 {
		sectionTitle := "Error Details (Plain-Language Breakdown)"
		if ev.Status == OpsAlertStatusResolved || ev.Status == OpsAlertStatusManualResolved {
			sectionTitle = "Incident Error Breakdown (For Postmortem)"
			if zh {
				sectionTitle = "故障期间报错明细（已归类翻译，供复盘参考）"
			}
		} else if zh {
			sectionTitle = "出错明细（已自动归类翻译）"
		}
		b.WriteString(fmt.Sprintf(`<div style="margin:18px 0 8px;font-size:14px;font-weight:700;color:#0f172a;">%s</div>`, html.EscapeString(sectionTitle)))
		for idx, c := range ev.TopClusters {
			countUnit := fmt.Sprintf("%d times", c.Count)
			channelPrefix := "Account / Group: "
			reasonPrefix := "Reason: "
			actionPrefix := "Next step: "
			rawPrefix := "Raw error: "
			if zh {
				countUnit = fmt.Sprintf("%d 次", c.Count)
				channelPrefix = "账号与分组："
				reasonPrefix = "原因翻译："
				actionPrefix = "建议操作："
				rawPrefix = "原始摘要："
			}
			channelText := formatOpsClusterChannelLabel(c.GroupLabel, c.AccountLabel, locale)
			b.WriteString(`<div style="margin:0 0 10px;padding:12px 14px;border:1px solid #e2e8f0;border-radius:8px;background:#ffffff;">`)
			b.WriteString(fmt.Sprintf(`<div style="font-size:13px;font-weight:700;color:#0f172a;margin-bottom:6px;">%d. %s · %s <span style="display:inline-block;margin-left:6px;padding:1px 8px;border-radius:999px;background:#fee2e2;color:#b91c1c;font-size:12px;font-weight:600;">%s</span></div>`,
				idx+1, html.EscapeString(c.Platform), html.EscapeString(c.Model), html.EscapeString(countUnit)))
			b.WriteString(fmt.Sprintf(`<div style="font-size:12px;color:#475569;line-height:1.6;"><strong>%s</strong>%s</div>`,
				html.EscapeString(channelPrefix), html.EscapeString(channelText)))
			b.WriteString(fmt.Sprintf(`<div style="font-size:13px;color:#1e293b;line-height:1.6;margin-top:4px;"><strong>%s</strong>%s</div>`,
				html.EscapeString(reasonPrefix), html.EscapeString(c.Reason)))
			b.WriteString(fmt.Sprintf(`<div style="font-size:12px;color:#0f766e;line-height:1.6;margin-top:3px;"><strong>%s</strong>%s</div>`,
				html.EscapeString(actionPrefix), html.EscapeString(c.Suggestion)))
			if c.SampleMessage != "" {
				b.WriteString(fmt.Sprintf(`<div style="font-size:11px;color:#64748b;line-height:1.5;margin-top:4px;word-break:break-word;overflow-wrap:anywhere;"><strong>%s</strong><code>%s</code></div>`,
					html.EscapeString(rawPrefix), html.EscapeString(c.SampleMessage)))
			}
			b.WriteString(`</div>`)
		}
	}

	if len(ev.Suggestions) > 0 && len(ev.TopClusters) == 0 {
		suggTitle := "Recommended Next Steps"
		if zh {
			suggTitle = "建议处理动作"
		}
		b.WriteString(fmt.Sprintf(`<div style="margin:16px 0 6px;font-size:14px;font-weight:700;color:#0f172a;">%s</div><ul style="margin:0;padding-left:18px;font-size:13px;color:#334155;line-height:1.7;">`,
			html.EscapeString(suggTitle)))
		for _, item := range ev.Suggestions {
			b.WriteString(fmt.Sprintf(`<li>%s</li>`, html.EscapeString(item)))
		}
		b.WriteString(`</ul>`)
	}

	return b.String()
}

func buildOpsAlertPrivateText(ev *opsAlertEvidence, diagnosisBadge, diagnosisSummary, locale string) string {
	if ev == nil {
		return ""
	}
	zh := isChineseOpsLocale(locale)
	tag := opsSeverityShortTag(ev.Severity, ev.Status, locale)
	var b strings.Builder
	if zh {
		b.WriteString(fmt.Sprintf("【GABUGU %s】%s\n", tag, buildOpsAlertHeadline(ev, locale)))
		b.WriteString(fmt.Sprintf("结论（%s）：%s\n", diagnosisBadge, diagnosisSummary))
		b.WriteString(fmt.Sprintf("指标：%s\n", ev.MetricSummary))
		if ev.ShowRequestOverview {
			b.WriteString(fmt.Sprintf("影响：有效请求 %d 次（成功 %d / 失败 %d），影响 %d 位用户\n",
				ev.SLARequests, ev.SuccessCount, ev.ErrorCount, ev.AffectedUsers))
		}
		if len(ev.TopClusters) > 0 {
			top := ev.TopClusters[0]
			channelText := formatOpsClusterChannelLabel(top.GroupLabel, top.AccountLabel, locale)
			b.WriteString(fmt.Sprintf("主要异常：%s · %s｜%s｜%d 次 — %s\n",
				top.Platform, top.Model, channelText, top.Count, top.Reason))
		}
		if ev.Status == OpsAlertStatusResolved || ev.Status == OpsAlertStatusManualResolved {
			b.WriteString(fmt.Sprintf("恢复时间：%s（持续 %s）\n", ev.ResolvedAtLocal, ev.DurationHuman))
		} else {
			b.WriteString(fmt.Sprintf("触发时间：%s\n", ev.FiredAtLocal))
		}
		if ev.DashboardURL != "" {
			b.WriteString(fmt.Sprintf("后台：%s", ev.DashboardURL))
		}
		return strings.TrimSpace(b.String())
	}

	b.WriteString(fmt.Sprintf("[GABUGU %s] %s\n", tag, buildOpsAlertHeadline(ev, locale)))
	b.WriteString(fmt.Sprintf("Summary (%s): %s\n", diagnosisBadge, diagnosisSummary))
	b.WriteString(fmt.Sprintf("Metric: %s\n", ev.MetricSummary))
	if ev.DashboardURL != "" {
		b.WriteString(fmt.Sprintf("Dashboard: %s", ev.DashboardURL))
	}
	return strings.TrimSpace(b.String())
}
