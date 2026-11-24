package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type SurveyService struct {
	surveyRepo   SurveyRepository
	userRepo     UserRepository
	userSubRepo  UserSubscriptionRepository
	rewardIssuer SurveyRewardIssuer
}

func NewSurveyService(
	surveyRepo SurveyRepository,
	userRepo UserRepository,
	userSubRepo UserSubscriptionRepository,
	redeemService *RedeemService,
) *SurveyService {
	var issuer SurveyRewardIssuer
	if redeemService != nil {
		issuer = redeemService
	}
	return NewSurveyServiceWithIssuer(surveyRepo, userRepo, userSubRepo, issuer)
}

func NewSurveyServiceWithIssuer(
	surveyRepo SurveyRepository,
	userRepo UserRepository,
	userSubRepo UserSubscriptionRepository,
	rewardIssuer SurveyRewardIssuer,
) *SurveyService {
	return &SurveyService{
		surveyRepo:   surveyRepo,
		userRepo:     userRepo,
		userSubRepo:  userSubRepo,
		rewardIssuer: rewardIssuer,
	}
}

type CreateSurveyInput struct {
	Title               string
	Description         string
	Status              string
	DisplayMode         string
	Questions           []SurveyQuestion
	Targeting           AnnouncementTargeting
	DeliveryPolicy      *SurveyDeliveryPolicy
	RewardType          string
	RewardValue         float64
	RewardBudgetLimit   *int
	RewardCodeValidDays int
	StartsAt            *time.Time
	EndsAt              *time.Time
	ActorID             *int64
}

type UpdateSurveyInput struct {
	Title               *string
	Description         *string
	Status              *string
	DisplayMode         *string
	Questions           *[]SurveyQuestion
	Targeting           *AnnouncementTargeting
	DeliveryPolicy      *SurveyDeliveryPolicy
	RewardType          *string
	RewardValue         *float64
	RewardBudgetLimit   **int
	RewardCodeValidDays *int
	StartsAt            **time.Time
	EndsAt              **time.Time
	ActorID             *int64
}

type SubmitSurveyInput struct {
	Answers         SurveyAnswers
	DurationSeconds int
}

type SubmitSurveyOutput struct {
	Response     *SurveyResponse
	MicroResults []MicroPollOptionStat
}

func (s *SurveyService) Create(ctx context.Context, input *CreateSurveyInput) (*Survey, error) {
	if input == nil {
		return nil, ErrSurveyNilInput
	}
	if !isJSONTimeInRange(input.StartsAt) || !isJSONTimeInRange(input.EndsAt) {
		return nil, ErrSurveyInvalidSchedule
	}

	title := strings.TrimSpace(input.Title)
	if title == "" || len([]rune(title)) > 200 {
		return nil, ErrSurveyInvalidTitle
	}
	description := strings.TrimSpace(input.Description)
	if len([]rune(description)) > 1000 {
		description = string([]rune(description)[:1000])
	}

	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = SurveyStatusDraft
	}
	if !isValidSurveyStatus(status) {
		return nil, ErrSurveyInvalidStatus
	}

	displayMode := strings.TrimSpace(input.DisplayMode)
	if displayMode == "" {
		displayMode = SurveyDisplayModeCorner
	}
	if !isValidSurveyDisplayMode(displayMode) {
		return nil, ErrSurveyInvalidDisplayMode
	}

	questions, err := domain.NormalizeAndValidateSurveyQuestions(displayMode, input.Questions)
	if err != nil {
		return nil, err
	}

	targeting, err := domain.AnnouncementTargeting(input.Targeting).NormalizeAndValidate()
	if err != nil {
		return nil, err
	}

	rawPolicy := domain.DefaultSurveyDeliveryPolicy()
	if input.DeliveryPolicy != nil {
		rawPolicy = *input.DeliveryPolicy
	}
	policy, err := rawPolicy.NormalizeAndValidate()
	if err != nil {
		return nil, err
	}

	rewardType, rewardValue, validDays, err := domain.ValidateSurveyReward(
		input.RewardType,
		input.RewardValue,
		input.RewardBudgetLimit,
		input.RewardCodeValidDays,
	)
	if err != nil {
		return nil, err
	}
	var budgetLimit *int
	if rewardType != SurveyRewardTypeNone && input.RewardBudgetLimit != nil {
		v := *input.RewardBudgetLimit
		budgetLimit = &v
	}

	if input.StartsAt != nil && input.EndsAt != nil && !input.StartsAt.Before(*input.EndsAt) {
		return nil, ErrSurveyInvalidSchedule
	}

	item := &Survey{
		Title:               title,
		Description:         description,
		Status:              status,
		DisplayMode:         displayMode,
		Questions:           questions,
		Targeting:           targeting,
		DeliveryPolicy:      policy,
		RewardType:          rewardType,
		RewardValue:         rewardValue,
		RewardBudgetLimit:   budgetLimit,
		RewardCodeValidDays: validDays,
		StartsAt:            input.StartsAt,
		EndsAt:              input.EndsAt,
	}
	if input.ActorID != nil && *input.ActorID > 0 {
		item.CreatedBy = input.ActorID
		item.UpdatedBy = input.ActorID
	}

	if err := s.surveyRepo.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("create survey: %w", err)
	}
	return item, nil
}

func (s *SurveyService) Update(ctx context.Context, id int64, input *UpdateSurveyInput) (*Survey, error) {
	if input == nil {
		return nil, ErrSurveyNilInput
	}
	if (input.StartsAt != nil && !isJSONTimeInRange(*input.StartsAt)) ||
		(input.EndsAt != nil && !isJSONTimeInRange(*input.EndsAt)) {
		return nil, ErrSurveyInvalidSchedule
	}

	item, err := s.surveyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" || len([]rune(title)) > 200 {
			return nil, ErrSurveyInvalidTitle
		}
		item.Title = title
	}
	if input.Description != nil {
		desc := strings.TrimSpace(*input.Description)
		if len([]rune(desc)) > 1000 {
			desc = string([]rune(desc)[:1000])
		}
		item.Description = desc
	}
	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		if !isValidSurveyStatus(status) {
			return nil, ErrSurveyInvalidStatus
		}
		item.Status = status
	}
	if input.DisplayMode != nil {
		mode := strings.TrimSpace(*input.DisplayMode)
		if !isValidSurveyDisplayMode(mode) {
			return nil, ErrSurveyInvalidDisplayMode
		}
		item.DisplayMode = mode
	}

	questionsToValidate := item.Questions
	if input.Questions != nil {
		questionsToValidate = *input.Questions
	}
	normalizedQuestions, err := domain.NormalizeAndValidateSurveyQuestions(item.DisplayMode, questionsToValidate)
	if err != nil {
		return nil, err
	}
	item.Questions = normalizedQuestions

	if input.Targeting != nil {
		targeting, err := domain.AnnouncementTargeting(*input.Targeting).NormalizeAndValidate()
		if err != nil {
			return nil, err
		}
		item.Targeting = targeting
	}
	if input.DeliveryPolicy != nil {
		policy, err := input.DeliveryPolicy.NormalizeAndValidate()
		if err != nil {
			return nil, err
		}
		item.DeliveryPolicy = policy
	}

	rewardType := item.RewardType
	if input.RewardType != nil {
		rewardType = *input.RewardType
	}
	rewardValue := item.RewardValue
	if input.RewardValue != nil {
		rewardValue = *input.RewardValue
	}
	budgetLimit := item.RewardBudgetLimit
	if input.RewardBudgetLimit != nil {
		budgetLimit = *input.RewardBudgetLimit
	}
	validDays := item.RewardCodeValidDays
	if input.RewardCodeValidDays != nil {
		validDays = *input.RewardCodeValidDays
	}

	normRewardType, normRewardValue, normValidDays, err := domain.ValidateSurveyReward(
		rewardType,
		rewardValue,
		budgetLimit,
		validDays,
	)
	if err != nil {
		return nil, err
	}
	item.RewardType = normRewardType
	item.RewardValue = normRewardValue
	item.RewardCodeValidDays = normValidDays
	if normRewardType == SurveyRewardTypeNone {
		item.RewardBudgetLimit = nil
	} else {
		item.RewardBudgetLimit = budgetLimit
	}

	if input.StartsAt != nil {
		item.StartsAt = *input.StartsAt
	}
	if input.EndsAt != nil {
		item.EndsAt = *input.EndsAt
	}
	if item.StartsAt != nil && item.EndsAt != nil && !item.StartsAt.Before(*item.EndsAt) {
		return nil, ErrSurveyInvalidSchedule
	}
	if input.ActorID != nil && *input.ActorID > 0 {
		item.UpdatedBy = input.ActorID
	}

	if err := s.surveyRepo.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("update survey: %w", err)
	}
	return item, nil
}

func (s *SurveyService) Delete(ctx context.Context, id int64) error {
	if err := s.surveyRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete survey: %w", err)
	}
	return nil
}

func (s *SurveyService) GetByID(ctx context.Context, id int64) (*Survey, error) {
	return s.surveyRepo.GetByID(ctx, id)
}

func (s *SurveyService) List(ctx context.Context, params pagination.PaginationParams, filters SurveyListFilters) ([]Survey, *pagination.PaginationResult, error) {
	return s.surveyRepo.List(ctx, params, filters)
}

func (s *SurveyService) ListActiveForUser(ctx context.Context, userID int64) ([]Survey, error) {
	user, activeGroupIDs, err := s.loadUserAudienceContext(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	activeSurveys, err := s.surveyRepo.ListActive(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("list active surveys: %w", err)
	}
	if len(activeSurveys) == 0 {
		return []Survey{}, nil
	}

	ids := make([]int64, 0, len(activeSurveys))
	for i := range activeSurveys {
		ids = append(ids, activeSurveys[i].ID)
	}
	stateMap, err := s.surveyRepo.GetUserStates(ctx, userID, ids)
	if err != nil {
		return nil, fmt.Errorf("get user survey states: %w", err)
	}

	usageCache := make(map[int]bool)
	visible := make([]Survey, 0, len(activeSurveys))
	for i := range activeSurveys {
		item := activeSurveys[i]
		if !item.IsActiveAt(now) {
			continue
		}
		if st, ok := stateMap[item.ID]; ok {
			if st.Status == SurveyUserStateSubmitted || st.Status == SurveyUserStateDismissed {
				continue
			}
			if st.Status == SurveyUserStateSnoozed && st.SnoozeUntil != nil && now.Before(*st.SnoozeUntil) {
				continue
			}
		}

		eligible, err := s.matchesUserPolicy(ctx, &item, user, activeGroupIDs, now, usageCache)
		if err != nil {
			return nil, err
		}
		if !eligible {
			continue
		}
		visible = append(visible, item)
	}

	// 排序：优先带奖励，其次新创建的问卷靠前
	sort.Slice(visible, func(i, j int) bool {
		ri := visible[i].HasAvailableRewardBudget()
		rj := visible[j].HasAvailableRewardBudget()
		if ri != rj {
			return ri
		}
		return visible[i].ID > visible[j].ID
	})

	return visible, nil
}

func (s *SurveyService) RecordUserAction(ctx context.Context, userID, surveyID int64, action string) (*time.Time, error) {
	act := strings.TrimSpace(action)
	if act != SurveyActionSnooze && act != SurveyActionDismiss {
		return nil, ErrSurveyInvalidAction
	}

	item, err := s.ensureSurveyVisibleToUser(ctx, userID, surveyID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var status string
	var snoozeUntil *time.Time
	if act == SurveyActionSnooze {
		status = SurveyUserStateSnoozed
		days := item.DeliveryPolicy.SnoozeDays
		if days <= 0 {
			days = 3
		}
		until := now.Add(time.Duration(days) * 24 * time.Hour)
		snoozeUntil = &until
	} else {
		status = SurveyUserStateDismissed
	}

	if err := s.surveyRepo.UpsertUserState(ctx, surveyID, userID, status, snoozeUntil, now); err != nil {
		return nil, fmt.Errorf("record survey user state: %w", err)
	}
	return snoozeUntil, nil
}

func (s *SurveyService) Submit(ctx context.Context, userID, surveyID int64, input *SubmitSurveyInput) (*SubmitSurveyOutput, error) {
	if input == nil {
		return nil, ErrSurveyNilInput
	}

	item, err := s.ensureSurveyVisibleToUser(ctx, userID, surveyID)
	if err != nil {
		return nil, err
	}

	already, err := s.surveyRepo.HasUserSubmitted(ctx, surveyID, userID)
	if err != nil {
		return nil, fmt.Errorf("check submitted: %w", err)
	}
	if already {
		return nil, ErrSurveyAlreadySubmitted
	}

	normalizedAnswers, err := domain.SurveyAnswers(input.Answers).NormalizeAndValidate(item.Questions)
	if err != nil {
		return nil, err
	}

	duration := input.DurationSeconds
	if duration < 0 {
		duration = 0
	}
	if duration > 3600 {
		duration = 3600
	}

	now := time.Now()
	attemptReward := item.RewardType != SurveyRewardTypeNone && item.RewardValue > 0

	submitRes, err := s.surveyRepo.SubmitAndReserveReward(ctx, &SubmitSurveyRecord{
		SurveyID:        surveyID,
		UserID:          userID,
		Answers:         normalizedAnswers,
		DurationSeconds: duration,
		AttemptReward:   attemptReward,
		RewardType:      item.RewardType,
		RewardValue:     item.RewardValue,
		SubmittedAt:     now,
	})
	if err != nil {
		if errors.Is(err, ErrSurveyAlreadySubmitted) {
			return nil, ErrSurveyAlreadySubmitted
		}
		return nil, fmt.Errorf("submit survey response: %w", err)
	}

	resp := submitRes.Response
	if attemptReward && submitRes.RewardReserved {
		s.fulfillSurveyReward(ctx, item, userID, resp)
	}

	var microResults []MicroPollOptionStat
	if item.DisplayMode == SurveyDisplayModeMicro && item.DeliveryPolicy.ShowMicroResults && len(item.Questions) == 1 {
		counts, err := s.surveyRepo.GetMicroOptionCounts(ctx, surveyID, item.Questions[0].ID)
		if err == nil {
			microResults = BuildMicroPollStatsFromCounts(item.Questions[0], counts)
		}
	}

	return &SubmitSurveyOutput{
		Response:     resp,
		MicroResults: microResults,
	}, nil
}

func (s *SurveyService) fulfillSurveyReward(ctx context.Context, survey *Survey, userID int64, resp *SurveyResponse) {
	// 答卷与名额预算已落库，发奖阶段脱离客户端请求取消信号，防止断连导致名额泄露或状态悬挂
	rewardCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()

	if s.rewardIssuer == nil {
		_ = s.surveyRepo.ReleaseRewardBudget(rewardCtx, survey.ID)
		_ = s.surveyRepo.UpdateResponseReward(rewardCtx, resp.ID, SurveyRewardStatusFailed, 0, "")
		resp.RewardStatus = SurveyRewardStatusFailed
		resp.RewardValue = 0
		return
	}

	code, err := generateSurveyRewardCode(survey.ID, userID)
	if err != nil {
		logger.LegacyPrintf("service.survey", "generate reward code failed: survey=%d user=%d err=%v", survey.ID, userID, err)
		_ = s.surveyRepo.ReleaseRewardBudget(rewardCtx, survey.ID)
		_ = s.surveyRepo.UpdateResponseReward(rewardCtx, resp.ID, SurveyRewardStatusFailed, 0, "")
		resp.RewardStatus = SurveyRewardStatusFailed
		resp.RewardValue = 0
		return
	}

	notes := fmt.Sprintf("问卷调查奖励 #%d: %s", survey.ID, truncateRunes(survey.Title, 80))
	redeemCode := &RedeemCode{
		Code:   code,
		Type:   RedeemTypeBalance,
		Value:  survey.RewardValue,
		Status: StatusUnused,
		Notes:  notes,
	}
	if survey.RewardType == SurveyRewardTypeRedeemCode {
		validDays := survey.RewardCodeValidDays
		if validDays <= 0 {
			validDays = 30
		}
		exp := time.Now().Add(time.Duration(validDays) * 24 * time.Hour)
		redeemCode.ExpiresAt = &exp
	}

	if err := s.rewardIssuer.CreateCode(rewardCtx, redeemCode); err != nil {
		logger.LegacyPrintf("service.survey", "create survey reward code failed: survey=%d user=%d err=%v", survey.ID, userID, err)
		_ = s.surveyRepo.ReleaseRewardBudget(rewardCtx, survey.ID)
		_ = s.surveyRepo.UpdateResponseReward(rewardCtx, resp.ID, SurveyRewardStatusFailed, 0, "")
		resp.RewardStatus = SurveyRewardStatusFailed
		resp.RewardValue = 0
		return
	}

	if survey.RewardType == SurveyRewardTypeRedeemCode {
		if err := s.surveyRepo.UpdateResponseReward(rewardCtx, resp.ID, SurveyRewardStatusCodeIssued, survey.RewardValue, code); err != nil {
			logger.LegacyPrintf("service.survey", "update response reward status failed: response=%d err=%v", resp.ID, err)
		}
		resp.RewardStatus = SurveyRewardStatusCodeIssued
		resp.RewardValue = survey.RewardValue
		resp.RewardRedeemCode = code
		return
	}

	// SurveyRewardTypeBalance: 立即原子入账（不触发推广返利、标为 gift 赠送额度）
	source := fmt.Sprintf("survey:%d", survey.ID)
	if _, err := s.rewardIssuer.RedeemForSurveyReward(rewardCtx, userID, code, survey.RewardValue, source); err != nil && !errors.Is(err, ErrRedeemCodeUsed) {
		logger.LegacyPrintf("service.survey", "auto redeem survey balance failed: survey=%d user=%d code=%s err=%v", survey.ID, userID, code, err)
		// 码已成功生成但自动入账失败：降级为 code_issued，用户仍可手动兑换，避免吞掉或重复发放
		_ = s.surveyRepo.UpdateResponseReward(rewardCtx, resp.ID, SurveyRewardStatusCodeIssued, survey.RewardValue, code)
		resp.RewardStatus = SurveyRewardStatusCodeIssued
		resp.RewardValue = survey.RewardValue
		resp.RewardRedeemCode = code
		return
	}

	if err := s.surveyRepo.UpdateResponseReward(rewardCtx, resp.ID, SurveyRewardStatusGranted, survey.RewardValue, code); err != nil {
		logger.LegacyPrintf("service.survey", "update response granted status failed: response=%d err=%v", resp.ID, err)
	}
	resp.RewardStatus = SurveyRewardStatusGranted
	resp.RewardValue = survey.RewardValue
	resp.RewardRedeemCode = code
}

func (s *SurveyService) GetStats(ctx context.Context, surveyID int64) (*SurveyStats, error) {
	item, err := s.surveyRepo.GetByID(ctx, surveyID)
	if err != nil {
		return nil, err
	}
	responses, err := s.surveyRepo.ListAllResponsesForStats(ctx, surveyID)
	if err != nil {
		return nil, fmt.Errorf("list responses for stats: %w", err)
	}
	snoozed, dismissed, err := s.surveyRepo.CountUserStatesByStatus(ctx, surveyID)
	if err != nil {
		return nil, fmt.Errorf("count user states: %w", err)
	}
	return BuildSurveyStats(item, responses, snoozed, dismissed), nil
}

func (s *SurveyService) ListResponses(ctx context.Context, surveyID int64, params pagination.PaginationParams, search string) ([]SurveyResponse, *pagination.PaginationResult, error) {
	if _, err := s.surveyRepo.GetByID(ctx, surveyID); err != nil {
		return nil, nil, err
	}
	return s.surveyRepo.ListResponses(ctx, surveyID, params, strings.TrimSpace(search))
}

func (s *SurveyService) loadUserAudienceContext(ctx context.Context, userID int64) (*User, map[int64]struct{}, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("get user: %w", err)
	}
	activeSubs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("list active subscriptions: %w", err)
	}
	activeGroupIDs := make(map[int64]struct{}, len(activeSubs))
	for i := range activeSubs {
		activeGroupIDs[activeSubs[i].GroupID] = struct{}{}
	}
	return user, activeGroupIDs, nil
}

func (s *SurveyService) ensureSurveyVisibleToUser(ctx context.Context, userID, surveyID int64) (*Survey, error) {
	user, activeGroupIDs, err := s.loadUserAudienceContext(ctx, userID)
	if err != nil {
		return nil, err
	}
	item, err := s.surveyRepo.GetByID(ctx, surveyID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if !item.IsActiveAt(now) {
		return nil, ErrSurveyNotFound
	}
	eligible, err := s.matchesUserPolicy(ctx, item, user, activeGroupIDs, now, nil)
	if err != nil {
		return nil, err
	}
	if !eligible {
		return nil, ErrSurveyNotFound
	}
	return item, nil
}

func (s *SurveyService) matchesUserPolicy(
	ctx context.Context,
	item *Survey,
	user *User,
	activeGroupIDs map[int64]struct{},
	now time.Time,
	usageCache map[int]bool,
) (bool, error) {
	if !item.Targeting.Matches(user.Balance, activeGroupIDs) {
		return false, nil
	}
	if item.DeliveryPolicy.MinAccountAgeDays > 0 {
		minAge := time.Duration(item.DeliveryPolicy.MinAccountAgeDays) * 24 * time.Hour
		if !user.CreatedAt.IsZero() && now.Sub(user.CreatedAt) < minAge {
			return false, nil
		}
	}
	if item.DeliveryPolicy.RequireRecentUsageDays > 0 {
		days := item.DeliveryPolicy.RequireRecentUsageDays
		if usageCache != nil {
			if has, cached := usageCache[days]; cached {
				if !has {
					return false, nil
				}
				return true, nil
			}
		}
		since := now.Add(-time.Duration(days) * 24 * time.Hour)
		hasUsage, err := s.surveyRepo.HasRecentUsage(ctx, user.ID, since)
		if err != nil {
			return false, fmt.Errorf("check recent usage: %w", err)
		}
		if usageCache != nil {
			usageCache[days] = hasUsage
		}
		if !hasUsage {
			return false, nil
		}
	}
	return true, nil
}

func BuildMicroPollStats(q SurveyQuestion, responses []SurveyResponse) []MicroPollOptionStat {
	counts := make(map[string]int64, len(q.Options))
	for i := range responses {
		ans, ok := responses[i].Answers[q.ID]
		if !ok || ans.ChoiceID == "" {
			continue
		}
		counts[ans.ChoiceID]++
	}
	return BuildMicroPollStatsFromCounts(q, counts)
}

func BuildMicroPollStatsFromCounts(q SurveyQuestion, counts map[string]int64) []MicroPollOptionStat {
	var total int64
	for _, opt := range q.Options {
		total += counts[opt.ID]
	}
	out := make([]MicroPollOptionStat, 0, len(q.Options))
	for _, opt := range q.Options {
		c := counts[opt.ID]
		pct := 0.0
		if total > 0 {
			pct = math.Round(float64(c)*1000/float64(total)) / 10
		}
		out = append(out, MicroPollOptionStat{
			OptionID: opt.ID,
			Label:    opt.Label,
			Count:    c,
			Percent:  pct,
		})
	}
	return out
}

func BuildSurveyStats(survey *Survey, responses []SurveyResponse, snoozed, dismissed int64) *SurveyStats {
	stats := &SurveyStats{
		SurveyID:       survey.ID,
		TotalResponses: int64(len(responses)),
		SnoozedCount:   snoozed,
		DismissedCount: dismissed,
		Questions:      make([]SurveyQuestionStats, 0, len(survey.Questions)),
	}

	var totalDuration int64
	for i := range responses {
		r := responses[i]
		totalDuration += int64(r.DurationSeconds)
		if r.RewardStatus == SurveyRewardStatusGranted || r.RewardStatus == SurveyRewardStatusCodeIssued {
			stats.RewardIssuedCount++
			stats.RewardTotalValue += r.RewardValue
		}
	}
	stats.RewardTotalValue = math.Round(stats.RewardTotalValue*100) / 100
	if len(responses) > 0 {
		stats.AverageDurationSec = math.Round(float64(totalDuration)*10/float64(len(responses))) / 10
	}

	for _, q := range survey.Questions {
		qs := SurveyQuestionStats{
			QuestionID: q.ID,
			Type:       q.Type,
			Title:      q.Title,
		}

		switch q.Type {
		case SurveyQuestionTypeSingle, SurveyQuestionTypeMulti:
			counts := make(map[string]int64, len(q.Options))
			for i := range responses {
				ans, ok := responses[i].Answers[q.ID]
				if !ok {
					continue
				}
				if q.Type == SurveyQuestionTypeSingle && ans.ChoiceID != "" {
					qs.AnswerCount++
					counts[ans.ChoiceID]++
				} else if q.Type == SurveyQuestionTypeMulti && len(ans.ChoiceIDs) > 0 {
					qs.AnswerCount++
					for _, id := range ans.ChoiceIDs {
						counts[id]++
					}
				}
			}
			qs.Options = make([]SurveyOptionStat, 0, len(q.Options))
			for _, opt := range q.Options {
				c := counts[opt.ID]
				pct := 0.0
				if qs.AnswerCount > 0 {
					pct = math.Round(float64(c)*1000/float64(qs.AnswerCount)) / 10
				}
				qs.Options = append(qs.Options, SurveyOptionStat{
					OptionID: opt.ID,
					Label:    opt.Label,
					Count:    c,
					Percent:  pct,
				})
			}

		case SurveyQuestionTypeRating:
			buckets := make(map[int]int64, 5)
			var sum int64
			for i := range responses {
				ans, ok := responses[i].Answers[q.ID]
				if !ok || ans.Rating == nil {
					continue
				}
				qs.AnswerCount++
				score := *ans.Rating
				sum += int64(score)
				buckets[score]++
			}
			if qs.AnswerCount > 0 {
				qs.AverageRating = math.Round(float64(sum)*100/float64(qs.AnswerCount)) / 100
			}
			qs.RatingBuckets = make([]SurveyRatingBucket, 0, 5)
			for score := 1; score <= 5; score++ {
				c := buckets[score]
				pct := 0.0
				if qs.AnswerCount > 0 {
					pct = math.Round(float64(c)*1000/float64(qs.AnswerCount)) / 10
				}
				qs.RatingBuckets = append(qs.RatingBuckets, SurveyRatingBucket{
					Score:   score,
					Count:   c,
					Percent: pct,
				})
			}

		case SurveyQuestionTypeTagsText:
			tagCounts := make(map[string]int64)
			texts := make([]SurveyTextFeedback, 0)
			for i := range responses {
				r := responses[i]
				ans, ok := r.Answers[q.ID]
				if !ok {
					continue
				}
				if len(ans.Tags) > 0 || ans.Text != "" {
					qs.AnswerCount++
				}
				for _, tag := range ans.Tags {
					tagCounts[tag]++
				}
				if ans.Text != "" {
					texts = append(texts, SurveyTextFeedback{
						UserEmail:   r.UserEmail,
						Text:        ans.Text,
						SubmittedAt: r.SubmittedAt,
					})
				}
			}
			topTags := make([]SurveyTagStat, 0, len(tagCounts))
			for tag, c := range tagCounts {
				topTags = append(topTags, SurveyTagStat{Tag: tag, Count: c})
			}
			sort.Slice(topTags, func(i, j int) bool {
				if topTags[i].Count != topTags[j].Count {
					return topTags[i].Count > topTags[j].Count
				}
				return topTags[i].Tag < topTags[j].Tag
			})
			if len(topTags) > 20 {
				topTags = topTags[:20]
			}
			qs.TopTags = topTags

			sort.Slice(texts, func(i, j int) bool {
				return texts[i].SubmittedAt.After(texts[j].SubmittedAt)
			})
			if len(texts) > 50 {
				texts = texts[:50]
			}
			qs.RecentTexts = texts
		}

		stats.Questions = append(stats.Questions, qs)
	}

	return stats
}

func generateSurveyRewardCode(surveyID, userID int64) (string, error) {
	buf := make([]byte, 3)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	sid := strings.ToUpper(strconv.FormatInt(surveyID, 36))
	uid := strings.ToUpper(strconv.FormatInt(userID, 36))
	prefix := fmt.Sprintf("SRV-%s-%s-", sid, uid)
	suffix := strings.ToUpper(hex.EncodeToString(buf))
	if maxSuffix := 32 - len(prefix); maxSuffix < len(suffix) {
		if maxSuffix < 2 {
			maxSuffix = 2
		}
		suffix = suffix[:maxSuffix]
	}
	return prefix + suffix, nil
}

func truncateRunes(s string, maxLen int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= maxLen {
		return string(runes)
	}
	return string(runes[:maxLen])
}

func isValidSurveyStatus(status string) bool {
	switch status {
	case SurveyStatusDraft, SurveyStatusActive, SurveyStatusArchived:
		return true
	default:
		return false
	}
}

func isValidSurveyDisplayMode(mode string) bool {
	switch mode {
	case SurveyDisplayModeCorner, SurveyDisplayModeModal, SurveyDisplayModeMicro:
		return true
	default:
		return false
	}
}
