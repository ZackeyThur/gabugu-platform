package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type Survey struct {
	ID                  int64                         `json:"id"`
	Title               string                        `json:"title"`
	Description         string                        `json:"description"`
	Status              string                        `json:"status"`
	DisplayMode         string                        `json:"display_mode"`
	Questions           []service.SurveyQuestion      `json:"questions"`
	Targeting           service.AnnouncementTargeting `json:"targeting"`
	DeliveryPolicy      service.SurveyDeliveryPolicy  `json:"delivery_policy"`
	RewardType          string                        `json:"reward_type"`
	RewardValue         float64                       `json:"reward_value"`
	RewardBudgetLimit   *int                          `json:"reward_budget_limit,omitempty"`
	RewardIssuedCount   int                           `json:"reward_issued_count"`
	RewardCodeValidDays int                           `json:"reward_code_valid_days"`
	ResponseCount       int64                         `json:"response_count"`
	StartsAt            *time.Time                    `json:"starts_at,omitempty"`
	EndsAt              *time.Time                    `json:"ends_at,omitempty"`
	CreatedBy           *int64                        `json:"created_by,omitempty"`
	UpdatedBy           *int64                        `json:"updated_by,omitempty"`
	CreatedAt           time.Time                     `json:"created_at"`
	UpdatedAt           time.Time                     `json:"updated_at"`
}

type UserSurvey struct {
	ID                       int64                    `json:"id"`
	Title                    string                   `json:"title"`
	Description              string                   `json:"description"`
	DisplayMode              string                   `json:"display_mode"`
	Questions                []service.SurveyQuestion `json:"questions"`
	HasReward                bool                     `json:"has_reward"`
	RewardType               string                   `json:"reward_type"`
	RewardValue              float64                  `json:"reward_value"`
	SnoozeDays               int                      `json:"snooze_days"`
	SkipIfAnnouncementUnread bool                     `json:"skip_if_announcement_unread"`
	DelaySeconds             int                      `json:"delay_seconds"`
	ShowMicroResults         bool                     `json:"show_micro_results"`
	CreatedAt                time.Time                `json:"created_at"`
}

type SubmitSurveyResult struct {
	ResponseID       int64                         `json:"response_id"`
	SurveyID         int64                         `json:"survey_id"`
	RewardStatus     string                        `json:"reward_status"`
	RewardType       string                        `json:"reward_type"`
	RewardValue      float64                       `json:"reward_value"`
	RewardRedeemCode string                        `json:"reward_redeem_code,omitempty"`
	MicroResults     []service.MicroPollOptionStat `json:"micro_results,omitempty"`
	SubmittedAt      time.Time                     `json:"submitted_at"`
}

type SurveyResponseItem struct {
	ID               int64                 `json:"id"`
	SurveyID         int64                 `json:"survey_id"`
	UserID           int64                 `json:"user_id"`
	UserEmail        string                `json:"user_email"`
	Username         string                `json:"username"`
	Answers          service.SurveyAnswers `json:"answers"`
	DurationSeconds  int                   `json:"duration_seconds"`
	RewardStatus     string                `json:"reward_status"`
	RewardType       string                `json:"reward_type"`
	RewardValue      float64               `json:"reward_value"`
	RewardRedeemCode string                `json:"reward_redeem_code,omitempty"`
	SubmittedAt      time.Time             `json:"submitted_at"`
}

func SurveyFromService(s *service.Survey) *Survey {
	if s == nil {
		return nil
	}
	return &Survey{
		ID:                  s.ID,
		Title:               s.Title,
		Description:         s.Description,
		Status:              s.Status,
		DisplayMode:         s.DisplayMode,
		Questions:           s.Questions,
		Targeting:           s.Targeting,
		DeliveryPolicy:      s.DeliveryPolicy,
		RewardType:          s.RewardType,
		RewardValue:         s.RewardValue,
		RewardBudgetLimit:   s.RewardBudgetLimit,
		RewardIssuedCount:   s.RewardIssuedCount,
		RewardCodeValidDays: s.RewardCodeValidDays,
		ResponseCount:       s.ResponseCount,
		StartsAt:            s.StartsAt,
		EndsAt:              s.EndsAt,
		CreatedBy:           s.CreatedBy,
		UpdatedBy:           s.UpdatedBy,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}
}

func UserSurveyFromService(s *service.Survey) *UserSurvey {
	if s == nil {
		return nil
	}
	hasReward := s.HasAvailableRewardBudget()
	rewardType := service.SurveyRewardTypeNone
	rewardValue := 0.0
	if hasReward {
		rewardType = s.RewardType
		rewardValue = s.RewardValue
	}
	snoozeDays := s.DeliveryPolicy.SnoozeDays
	if snoozeDays <= 0 {
		snoozeDays = 3
	}
	return &UserSurvey{
		ID:                       s.ID,
		Title:                    s.Title,
		Description:              s.Description,
		DisplayMode:              s.DisplayMode,
		Questions:                s.Questions,
		HasReward:                hasReward,
		RewardType:               rewardType,
		RewardValue:              rewardValue,
		SnoozeDays:               snoozeDays,
		SkipIfAnnouncementUnread: s.DeliveryPolicy.SkipIfAnnouncementUnread,
		DelaySeconds:             s.DeliveryPolicy.DelaySeconds,
		ShowMicroResults:         s.DeliveryPolicy.ShowMicroResults,
		CreatedAt:                s.CreatedAt,
	}
}

func SubmitSurveyResultFromService(out *service.SubmitSurveyOutput) *SubmitSurveyResult {
	if out == nil || out.Response == nil {
		return nil
	}
	return &SubmitSurveyResult{
		ResponseID:       out.Response.ID,
		SurveyID:         out.Response.SurveyID,
		RewardStatus:     out.Response.RewardStatus,
		RewardType:       out.Response.RewardType,
		RewardValue:      out.Response.RewardValue,
		RewardRedeemCode: out.Response.RewardRedeemCode,
		MicroResults:     out.MicroResults,
		SubmittedAt:      out.Response.SubmittedAt,
	}
}

func SurveyResponseItemFromService(r *service.SurveyResponse) *SurveyResponseItem {
	if r == nil {
		return nil
	}
	return &SurveyResponseItem{
		ID:               r.ID,
		SurveyID:         r.SurveyID,
		UserID:           r.UserID,
		UserEmail:        r.UserEmail,
		Username:         r.Username,
		Answers:          r.Answers,
		DurationSeconds:  r.DurationSeconds,
		RewardStatus:     r.RewardStatus,
		RewardType:       r.RewardType,
		RewardValue:      r.RewardValue,
		RewardRedeemCode: r.RewardRedeemCode,
		SubmittedAt:      r.SubmittedAt,
	}
}
