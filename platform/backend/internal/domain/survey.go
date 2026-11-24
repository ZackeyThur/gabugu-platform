package domain

import (
	"fmt"
	"math"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SurveyStatusDraft    = "draft"
	SurveyStatusActive   = "active"
	SurveyStatusArchived = "archived"
)

const (
	SurveyDisplayModeCorner = "corner"
	SurveyDisplayModeModal  = "modal"
	SurveyDisplayModeMicro  = "micro"
)

const (
	SurveyQuestionTypeSingle   = "single"
	SurveyQuestionTypeMulti    = "multi"
	SurveyQuestionTypeRating   = "rating"
	SurveyQuestionTypeTagsText = "tags_text"
)

const (
	SurveyRewardTypeNone       = "none"
	SurveyRewardTypeBalance    = "balance"
	SurveyRewardTypeRedeemCode = "redeem_code"
)

const (
	SurveyRewardStatusNone            = "none"
	SurveyRewardStatusPending         = "pending"
	SurveyRewardStatusGranted         = "granted"
	SurveyRewardStatusCodeIssued      = "code_issued"
	SurveyRewardStatusBudgetExhausted = "budget_exhausted"
	SurveyRewardStatusFailed          = "failed"
)

const (
	SurveyUserStateSnoozed   = "snoozed"
	SurveyUserStateDismissed = "dismissed"
	SurveyUserStateSubmitted = "submitted"
)

const (
	SurveyActionSnooze  = "snooze"
	SurveyActionDismiss = "dismiss"
)

const (
	maxSurveyQuestions          = 10
	maxSurveyOptionsPerQuestion = 12
	maxSurveyChipsPerGroup      = 12
	maxSurveyTextLength         = 1000
	maxSurveyRewardValue        = 1000.0
)

var (
	ErrSurveyNotFound           = infraerrors.NotFound("SURVEY_NOT_FOUND", "survey not found")
	ErrSurveyNilInput           = infraerrors.BadRequest("SURVEY_NIL_INPUT", "survey input is required")
	ErrSurveyInvalidTitle       = infraerrors.BadRequest("SURVEY_INVALID_TITLE", "survey title is required and must be at most 200 characters")
	ErrSurveyInvalidStatus      = infraerrors.BadRequest("SURVEY_INVALID_STATUS", "invalid survey status")
	ErrSurveyInvalidDisplayMode = infraerrors.BadRequest("SURVEY_INVALID_DISPLAY_MODE", "invalid survey display mode")
	ErrSurveyInvalidQuestions   = infraerrors.BadRequest("SURVEY_INVALID_QUESTIONS", "invalid survey questions configuration")
	ErrSurveyInvalidPolicy      = infraerrors.BadRequest("SURVEY_INVALID_DELIVERY_POLICY", "invalid survey delivery policy")
	ErrSurveyInvalidReward      = infraerrors.BadRequest("SURVEY_INVALID_REWARD", "invalid survey reward configuration")
	ErrSurveyInvalidSchedule    = infraerrors.BadRequest("SURVEY_INVALID_SCHEDULE", "starts_at must be before ends_at")
	ErrSurveyInvalidAnswers     = infraerrors.BadRequest("SURVEY_INVALID_ANSWERS", "invalid survey answers")
	ErrSurveyInvalidAction      = infraerrors.BadRequest("SURVEY_INVALID_ACTION", "action must be snooze or dismiss")
	ErrSurveyAlreadySubmitted   = infraerrors.Conflict("SURVEY_ALREADY_SUBMITTED", "you have already submitted this survey")
)

type SurveyQuestionOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

type SurveyQuestion struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // single | multi | rating | tags_text
	Title       string                 `json:"title"`
	Hint        string                 `json:"hint,omitempty"`
	Required    bool                   `json:"required"`
	AutoAdvance bool                   `json:"auto_advance,omitempty"`
	Options     []SurveyQuestionOption `json:"options,omitempty"`

	// rating 题配置（默认 1..5）
	ScaleMin int `json:"scale_min,omitempty"`
	ScaleMax int `json:"scale_max,omitempty"`

	// tags_text 题配置：可选关联前置 rating 题，实现低分/高分动态快捷标签
	LinkedRatingQuestionID string   `json:"linked_rating_question_id,omitempty"`
	LowScoreThreshold      int      `json:"low_score_threshold,omitempty"`
	LowTitle               string   `json:"low_title,omitempty"`
	HighTitle              string   `json:"high_title,omitempty"`
	LowChips               []string `json:"low_chips,omitempty"`
	HighChips              []string `json:"high_chips,omitempty"`
	Placeholder            string   `json:"placeholder,omitempty"`
}

type SurveyDeliveryPolicy struct {
	SnoozeDays               int  `json:"snooze_days"`                 // 「稍后再说」冷却天数，默认 3
	MinAccountAgeDays        int  `json:"min_account_age_days"`        // 注册满 N 天才展示，0 表示不限
	RequireRecentUsageDays   int  `json:"require_recent_usage_days"`   // 近 N 天有 API 使用才展示，0 表示不限
	SkipIfAnnouncementUnread bool `json:"skip_if_announcement_unread"` // 有未读弹窗公告时自动错峰顺延
	DelaySeconds             int  `json:"delay_seconds"`               // 登录后延迟弹窗秒数，默认 4
	ShowMicroResults         bool `json:"show_micro_results"`          // 1 题快问投票后是否立即展示实时票数占比
}

func DefaultSurveyDeliveryPolicy() SurveyDeliveryPolicy {
	return SurveyDeliveryPolicy{
		SnoozeDays:               3,
		MinAccountAgeDays:        0,
		RequireRecentUsageDays:   0,
		SkipIfAnnouncementUnread: true,
		DelaySeconds:             4,
		ShowMicroResults:         true,
	}
}

func (p SurveyDeliveryPolicy) NormalizeAndValidate() (SurveyDeliveryPolicy, error) {
	out := p
	if out.SnoozeDays == 0 {
		out.SnoozeDays = 3
	}
	if out.SnoozeDays < 1 || out.SnoozeDays > 90 {
		return SurveyDeliveryPolicy{}, ErrSurveyInvalidPolicy
	}
	if out.MinAccountAgeDays < 0 || out.MinAccountAgeDays > 3650 {
		return SurveyDeliveryPolicy{}, ErrSurveyInvalidPolicy
	}
	if out.RequireRecentUsageDays < 0 || out.RequireRecentUsageDays > 365 {
		return SurveyDeliveryPolicy{}, ErrSurveyInvalidPolicy
	}
	if out.DelaySeconds < 0 || out.DelaySeconds > 60 {
		return SurveyDeliveryPolicy{}, ErrSurveyInvalidPolicy
	}
	return out, nil
}

type Survey struct {
	ID                  int64
	Title               string
	Description         string
	Status              string
	DisplayMode         string
	Questions           []SurveyQuestion
	Targeting           AnnouncementTargeting
	DeliveryPolicy      SurveyDeliveryPolicy
	RewardType          string
	RewardValue         float64
	RewardBudgetLimit   *int
	RewardIssuedCount   int
	RewardCodeValidDays int
	ResponseCount       int64
	StartsAt            *time.Time
	EndsAt              *time.Time
	CreatedBy           *int64
	UpdatedBy           *int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (s *Survey) IsActiveAt(now time.Time) bool {
	if s == nil {
		return false
	}
	if s.Status != SurveyStatusActive {
		return false
	}
	if s.StartsAt != nil && now.Before(*s.StartsAt) {
		return false
	}
	if s.EndsAt != nil && !now.Before(*s.EndsAt) {
		return false
	}
	return true
}

func (s *Survey) HasAvailableRewardBudget() bool {
	if s == nil || s.RewardType == SurveyRewardTypeNone || s.RewardValue <= 0 {
		return false
	}
	if s.RewardBudgetLimit == nil {
		return true
	}
	return s.RewardIssuedCount < *s.RewardBudgetLimit
}

func NormalizeAndValidateSurveyQuestions(displayMode string, questions []SurveyQuestion) ([]SurveyQuestion, error) {
	if len(questions) == 0 || len(questions) > maxSurveyQuestions {
		return nil, ErrSurveyInvalidQuestions
	}
	if displayMode == SurveyDisplayModeMicro && len(questions) != 1 {
		return nil, infraerrors.BadRequest("SURVEY_MICRO_REQUIRES_SINGLE_QUESTION", "micro poll mode requires exactly 1 single-choice question")
	}

	seenQIDs := make(map[string]struct{}, len(questions))
	out := make([]SurveyQuestion, 0, len(questions))

	for i, raw := range questions {
		q := SurveyQuestion{
			ID:                     strings.TrimSpace(raw.ID),
			Type:                   strings.TrimSpace(raw.Type),
			Title:                  strings.TrimSpace(raw.Title),
			Hint:                   strings.TrimSpace(raw.Hint),
			Required:               raw.Required,
			AutoAdvance:            raw.AutoAdvance,
			ScaleMin:               raw.ScaleMin,
			ScaleMax:               raw.ScaleMax,
			LinkedRatingQuestionID: strings.TrimSpace(raw.LinkedRatingQuestionID),
			LowScoreThreshold:      raw.LowScoreThreshold,
			LowTitle:               strings.TrimSpace(raw.LowTitle),
			HighTitle:              strings.TrimSpace(raw.HighTitle),
			Placeholder:            strings.TrimSpace(raw.Placeholder),
		}
		if q.ID == "" {
			q.ID = fmt.Sprintf("q%d", i+1)
		}
		if len(q.ID) > 64 {
			return nil, ErrSurveyInvalidQuestions
		}
		if _, dup := seenQIDs[q.ID]; dup {
			return nil, ErrSurveyInvalidQuestions
		}
		seenQIDs[q.ID] = struct{}{}

		if q.Title == "" || len([]rune(q.Title)) > 200 {
			return nil, ErrSurveyInvalidQuestions
		}
		if len([]rune(q.Hint)) > 240 {
			return nil, ErrSurveyInvalidQuestions
		}

		if displayMode == SurveyDisplayModeMicro && q.Type != SurveyQuestionTypeSingle {
			return nil, infraerrors.BadRequest("SURVEY_MICRO_REQUIRES_SINGLE_QUESTION", "micro poll mode only supports single-choice question")
		}

		switch q.Type {
		case SurveyQuestionTypeSingle, SurveyQuestionTypeMulti:
			if len(raw.Options) < 2 || len(raw.Options) > maxSurveyOptionsPerQuestion {
				return nil, ErrSurveyInvalidQuestions
			}
			seenOptIDs := make(map[string]struct{}, len(raw.Options))
			q.Options = make([]SurveyQuestionOption, 0, len(raw.Options))
			for j, rawOpt := range raw.Options {
				opt := SurveyQuestionOption{
					ID:          strings.TrimSpace(rawOpt.ID),
					Label:       strings.TrimSpace(rawOpt.Label),
					Description: strings.TrimSpace(rawOpt.Description),
					Tag:         strings.TrimSpace(rawOpt.Tag),
					Icon:        strings.TrimSpace(rawOpt.Icon),
				}
				if opt.ID == "" {
					opt.ID = fmt.Sprintf("opt%d", j+1)
				}
				if len(opt.ID) > 64 || opt.Label == "" || len([]rune(opt.Label)) > 100 || len([]rune(opt.Description)) > 200 || len([]rune(opt.Tag)) > 40 {
					return nil, ErrSurveyInvalidQuestions
				}
				if _, dup := seenOptIDs[opt.ID]; dup {
					return nil, ErrSurveyInvalidQuestions
				}
				seenOptIDs[opt.ID] = struct{}{}
				q.Options = append(q.Options, opt)
			}

		case SurveyQuestionTypeRating:
			if q.ScaleMin == 0 && q.ScaleMax == 0 {
				q.ScaleMin = 1
				q.ScaleMax = 5
			}
			if q.ScaleMin != 1 || q.ScaleMax != 5 {
				return nil, ErrSurveyInvalidQuestions
			}

		case SurveyQuestionTypeTagsText:
			if q.LowScoreThreshold == 0 {
				q.LowScoreThreshold = 3
			}
			if q.LowScoreThreshold < 1 || q.LowScoreThreshold > 5 {
				return nil, ErrSurveyInvalidQuestions
			}
			q.LowChips = normalizeChips(raw.LowChips)
			q.HighChips = normalizeChips(raw.HighChips)
			if len(q.LowChips) > maxSurveyChipsPerGroup || len(q.HighChips) > maxSurveyChipsPerGroup {
				return nil, ErrSurveyInvalidQuestions
			}
			if len([]rune(q.LowTitle)) > 200 || len([]rune(q.HighTitle)) > 200 || len([]rune(q.Placeholder)) > 200 {
				return nil, ErrSurveyInvalidQuestions
			}

		default:
			return nil, ErrSurveyInvalidQuestions
		}

		out = append(out, q)
	}

	return out, nil
}

func normalizeChips(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		v := strings.TrimSpace(item)
		if v == "" || len([]rune(v)) > 60 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func ValidateSurveyReward(rewardType string, rewardValue float64, budgetLimit *int, validDays int) (string, float64, int, error) {
	rt := strings.TrimSpace(rewardType)
	if rt == "" {
		rt = SurveyRewardTypeNone
	}
	switch rt {
	case SurveyRewardTypeNone:
		if validDays <= 0 {
			validDays = 30
		}
		return rt, 0, validDays, nil
	case SurveyRewardTypeBalance, SurveyRewardTypeRedeemCode:
		if math.IsNaN(rewardValue) || math.IsInf(rewardValue, 0) || rewardValue <= 0 || rewardValue > maxSurveyRewardValue {
			return "", 0, 0, ErrSurveyInvalidReward
		}
		quantized := math.Round(rewardValue*100) / 100
		if quantized <= 0 {
			return "", 0, 0, ErrSurveyInvalidReward
		}
		if budgetLimit != nil && *budgetLimit <= 0 {
			return "", 0, 0, ErrSurveyInvalidReward
		}
		if validDays == 0 {
			validDays = 30
		}
		if validDays < 1 || validDays > 365 {
			return "", 0, 0, ErrSurveyInvalidReward
		}
		return rt, quantized, validDays, nil
	default:
		return "", 0, 0, ErrSurveyInvalidReward
	}
}

type SurveyAnswerValue struct {
	ChoiceID  string   `json:"choice_id,omitempty"`
	ChoiceIDs []string `json:"choice_ids,omitempty"`
	Rating    *int     `json:"rating,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Text      string   `json:"text,omitempty"`
}

type SurveyAnswers map[string]SurveyAnswerValue

func (a SurveyAnswers) NormalizeAndValidate(questions []SurveyQuestion) (SurveyAnswers, error) {
	normalized := make(SurveyAnswers, len(questions))

	for _, q := range questions {
		ans, exists := a[q.ID]
		if !exists {
			if q.Required {
				return nil, infraerrors.BadRequest("SURVEY_REQUIRED_QUESTION_MISSING", fmt.Sprintf("question %s is required", q.ID))
			}
			continue
		}

		switch q.Type {
		case SurveyQuestionTypeSingle:
			choiceID := strings.TrimSpace(ans.ChoiceID)
			if choiceID == "" {
				if q.Required {
					return nil, ErrSurveyInvalidAnswers
				}
				continue
			}
			if !hasSurveyOption(q.Options, choiceID) {
				return nil, ErrSurveyInvalidAnswers
			}
			normalized[q.ID] = SurveyAnswerValue{ChoiceID: choiceID}

		case SurveyQuestionTypeMulti:
			seen := make(map[string]struct{}, len(ans.ChoiceIDs))
			picked := make([]string, 0, len(ans.ChoiceIDs))
			for _, rawID := range ans.ChoiceIDs {
				id := strings.TrimSpace(rawID)
				if id == "" {
					continue
				}
				if !hasSurveyOption(q.Options, id) {
					return nil, ErrSurveyInvalidAnswers
				}
				if _, dup := seen[id]; dup {
					continue
				}
				seen[id] = struct{}{}
				picked = append(picked, id)
			}
			if len(picked) == 0 {
				if q.Required {
					return nil, ErrSurveyInvalidAnswers
				}
				continue
			}
			normalized[q.ID] = SurveyAnswerValue{ChoiceIDs: picked}

		case SurveyQuestionTypeRating:
			if ans.Rating == nil {
				if q.Required {
					return nil, ErrSurveyInvalidAnswers
				}
				continue
			}
			minVal := q.ScaleMin
			maxVal := q.ScaleMax
			if minVal == 0 {
				minVal = 1
			}
			if maxVal == 0 {
				maxVal = 5
			}
			if *ans.Rating < minVal || *ans.Rating > maxVal {
				return nil, ErrSurveyInvalidAnswers
			}
			score := *ans.Rating
			normalized[q.ID] = SurveyAnswerValue{Rating: &score}

		case SurveyQuestionTypeTagsText:
			allowedTags := make(map[string]struct{}, len(q.LowChips)+len(q.HighChips))
			for _, t := range q.LowChips {
				allowedTags[t] = struct{}{}
			}
			for _, t := range q.HighChips {
				allowedTags[t] = struct{}{}
			}
			seenTag := make(map[string]struct{}, len(ans.Tags))
			pickedTags := make([]string, 0, len(ans.Tags))
			for _, rawTag := range ans.Tags {
				tag := strings.TrimSpace(rawTag)
				if tag == "" {
					continue
				}
				if len(allowedTags) > 0 {
					if _, ok := allowedTags[tag]; !ok {
						return nil, ErrSurveyInvalidAnswers
					}
				} else if len([]rune(tag)) > 60 {
					return nil, ErrSurveyInvalidAnswers
				}
				if _, dup := seenTag[tag]; dup {
					continue
				}
				seenTag[tag] = struct{}{}
				pickedTags = append(pickedTags, tag)
			}
			text := strings.TrimSpace(ans.Text)
			if len([]rune(text)) > maxSurveyTextLength {
				return nil, ErrSurveyInvalidAnswers
			}
			if len(pickedTags) == 0 && text == "" {
				if q.Required {
					return nil, ErrSurveyInvalidAnswers
				}
				continue
			}
			normalized[q.ID] = SurveyAnswerValue{
				Tags: pickedTags,
				Text: text,
			}
		}
	}

	return normalized, nil
}

func hasSurveyOption(options []SurveyQuestionOption, id string) bool {
	for _, opt := range options {
		if opt.ID == id {
			return true
		}
	}
	return false
}

type SurveyResponse struct {
	ID               int64
	SurveyID         int64
	UserID           int64
	UserEmail        string
	Username         string
	Answers          SurveyAnswers
	DurationSeconds  int
	RewardStatus     string
	RewardType       string
	RewardValue      float64
	RewardRedeemCode string
	SubmittedAt      time.Time
}

type SurveyUserState struct {
	SurveyID    int64
	UserID      int64
	Status      string
	SnoozeUntil *time.Time
	UpdatedAt   time.Time
}

type MicroPollOptionStat struct {
	OptionID string  `json:"option_id"`
	Label    string  `json:"label"`
	Count    int64   `json:"count"`
	Percent  float64 `json:"percent"`
}

type SurveyOptionStat struct {
	OptionID string  `json:"option_id"`
	Label    string  `json:"label"`
	Count    int64   `json:"count"`
	Percent  float64 `json:"percent"`
}

type SurveyRatingBucket struct {
	Score   int     `json:"score"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

type SurveyTagStat struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

type SurveyTextFeedback struct {
	UserEmail   string    `json:"user_email"`
	Text        string    `json:"text"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type SurveyQuestionStats struct {
	QuestionID    string               `json:"question_id"`
	Type          string               `json:"type"`
	Title         string               `json:"title"`
	AnswerCount   int64                `json:"answer_count"`
	Options       []SurveyOptionStat   `json:"options,omitempty"`
	AverageRating float64              `json:"average_rating,omitempty"`
	RatingBuckets []SurveyRatingBucket `json:"rating_buckets,omitempty"`
	TopTags       []SurveyTagStat      `json:"top_tags,omitempty"`
	RecentTexts   []SurveyTextFeedback `json:"recent_texts,omitempty"`
}

type SurveyStats struct {
	SurveyID           int64                 `json:"survey_id"`
	TotalResponses     int64                 `json:"total_responses"`
	SnoozedCount       int64                 `json:"snoozed_count"`
	DismissedCount     int64                 `json:"dismissed_count"`
	AverageDurationSec float64               `json:"average_duration_seconds"`
	RewardIssuedCount  int64                 `json:"reward_issued_count"`
	RewardTotalValue   float64               `json:"reward_total_value"`
	Questions          []SurveyQuestionStats `json:"questions"`
}
