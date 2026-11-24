package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	SurveyStatusDraft    = domain.SurveyStatusDraft
	SurveyStatusActive   = domain.SurveyStatusActive
	SurveyStatusArchived = domain.SurveyStatusArchived

	SurveyDisplayModeCorner = domain.SurveyDisplayModeCorner
	SurveyDisplayModeModal  = domain.SurveyDisplayModeModal
	SurveyDisplayModeMicro  = domain.SurveyDisplayModeMicro

	SurveyQuestionTypeSingle   = domain.SurveyQuestionTypeSingle
	SurveyQuestionTypeMulti    = domain.SurveyQuestionTypeMulti
	SurveyQuestionTypeRating   = domain.SurveyQuestionTypeRating
	SurveyQuestionTypeTagsText = domain.SurveyQuestionTypeTagsText

	SurveyRewardTypeNone       = domain.SurveyRewardTypeNone
	SurveyRewardTypeBalance    = domain.SurveyRewardTypeBalance
	SurveyRewardTypeRedeemCode = domain.SurveyRewardTypeRedeemCode

	SurveyRewardStatusNone            = domain.SurveyRewardStatusNone
	SurveyRewardStatusPending         = domain.SurveyRewardStatusPending
	SurveyRewardStatusGranted         = domain.SurveyRewardStatusGranted
	SurveyRewardStatusCodeIssued      = domain.SurveyRewardStatusCodeIssued
	SurveyRewardStatusBudgetExhausted = domain.SurveyRewardStatusBudgetExhausted
	SurveyRewardStatusFailed          = domain.SurveyRewardStatusFailed

	SurveyUserStateSnoozed   = domain.SurveyUserStateSnoozed
	SurveyUserStateDismissed = domain.SurveyUserStateDismissed
	SurveyUserStateSubmitted = domain.SurveyUserStateSubmitted

	SurveyActionSnooze  = domain.SurveyActionSnooze
	SurveyActionDismiss = domain.SurveyActionDismiss
)

var (
	ErrSurveyNotFound           = domain.ErrSurveyNotFound
	ErrSurveyNilInput           = domain.ErrSurveyNilInput
	ErrSurveyInvalidTitle       = domain.ErrSurveyInvalidTitle
	ErrSurveyInvalidStatus      = domain.ErrSurveyInvalidStatus
	ErrSurveyInvalidDisplayMode = domain.ErrSurveyInvalidDisplayMode
	ErrSurveyInvalidQuestions   = domain.ErrSurveyInvalidQuestions
	ErrSurveyInvalidPolicy      = domain.ErrSurveyInvalidPolicy
	ErrSurveyInvalidReward      = domain.ErrSurveyInvalidReward
	ErrSurveyInvalidSchedule    = domain.ErrSurveyInvalidSchedule
	ErrSurveyInvalidAnswers     = domain.ErrSurveyInvalidAnswers
	ErrSurveyInvalidAction      = domain.ErrSurveyInvalidAction
	ErrSurveyAlreadySubmitted   = domain.ErrSurveyAlreadySubmitted
)

type (
	Survey               = domain.Survey
	SurveyQuestion       = domain.SurveyQuestion
	SurveyQuestionOption = domain.SurveyQuestionOption
	SurveyDeliveryPolicy = domain.SurveyDeliveryPolicy
	SurveyAnswerValue    = domain.SurveyAnswerValue
	SurveyAnswers        = domain.SurveyAnswers
	SurveyResponse       = domain.SurveyResponse
	SurveyUserState      = domain.SurveyUserState
	MicroPollOptionStat  = domain.MicroPollOptionStat
	SurveyStats          = domain.SurveyStats
	SurveyQuestionStats  = domain.SurveyQuestionStats
	SurveyOptionStat     = domain.SurveyOptionStat
	SurveyRatingBucket   = domain.SurveyRatingBucket
	SurveyTagStat        = domain.SurveyTagStat
	SurveyTextFeedback   = domain.SurveyTextFeedback
)

type SurveyListFilters struct {
	Status string
	Search string
}

type SubmitSurveyRecord struct {
	SurveyID          int64
	UserID            int64
	Answers           SurveyAnswers
	DurationSeconds   int
	AttemptReward     bool
	RewardType        string
	RewardValue       float64
	SubmittedAt       time.Time
}

type SubmitSurveyRecordResult struct {
	Response       *SurveyResponse
	RewardReserved bool
}

type SurveyRepository interface {
	Create(ctx context.Context, s *Survey) error
	GetByID(ctx context.Context, id int64) (*Survey, error)
	Update(ctx context.Context, s *Survey) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params pagination.PaginationParams, filters SurveyListFilters) ([]Survey, *pagination.PaginationResult, error)
	ListActive(ctx context.Context, now time.Time) ([]Survey, error)

	GetUserStates(ctx context.Context, userID int64, surveyIDs []int64) (map[int64]SurveyUserState, error)
	UpsertUserState(ctx context.Context, surveyID, userID int64, status string, snoozeUntil *time.Time, now time.Time) error
	HasUserSubmitted(ctx context.Context, surveyID, userID int64) (bool, error)
	HasRecentUsage(ctx context.Context, userID int64, since time.Time) (bool, error)

	SubmitAndReserveReward(ctx context.Context, record *SubmitSurveyRecord) (*SubmitSurveyRecordResult, error)
	UpdateResponseReward(ctx context.Context, responseID int64, rewardStatus string, rewardValue float64, redeemCode string) error
	ReleaseRewardBudget(ctx context.Context, surveyID int64) error

	GetMicroOptionCounts(ctx context.Context, surveyID int64, questionID string) (map[string]int64, error)
	ListAllResponsesForStats(ctx context.Context, surveyID int64) ([]SurveyResponse, error)
	CountUserStatesByStatus(ctx context.Context, surveyID int64) (snoozed int64, dismissed int64, err error)
	ListResponses(ctx context.Context, surveyID int64, params pagination.PaginationParams, search string) ([]SurveyResponse, *pagination.PaginationResult, error)
}

// SurveyRewardIssuer abstracts RedeemService for survey reward issuance and unit tests.
type SurveyRewardIssuer interface {
	CreateCode(ctx context.Context, code *RedeemCode) error
	RedeemForSurveyReward(ctx context.Context, userID int64, code string, amount float64, source string) (*RedeemCode, error)
}
