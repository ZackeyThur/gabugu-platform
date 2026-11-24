//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type surveyRepoStub struct {
	surveys           map[int64]*Survey
	nextID            int64
	states            map[string]SurveyUserState
	responses         []SurveyResponse
	recentUsageByUser map[int64]bool
	budgetExhausted   bool
	releasedBudget    int
}

func newSurveyRepoStub() *surveyRepoStub {
	return &surveyRepoStub{
		surveys:           make(map[int64]*Survey),
		nextID:            1,
		states:            make(map[string]SurveyUserState),
		recentUsageByUser: make(map[int64]bool),
	}
}

func surveyStateKey(surveyID, userID int64) string {
	return time.Now().Format("") + string(rune(surveyID)) + ":" + string(rune(userID))
}

func (r *surveyRepoStub) Create(_ context.Context, s *Survey) error {
	s.ID = r.nextID
	r.nextID++
	s.CreatedAt = time.Now()
	s.UpdatedAt = s.CreatedAt
	cp := *s
	r.surveys[s.ID] = &cp
	return nil
}

func (r *surveyRepoStub) GetByID(_ context.Context, id int64) (*Survey, error) {
	s, ok := r.surveys[id]
	if !ok {
		return nil, ErrSurveyNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *surveyRepoStub) Update(_ context.Context, s *Survey) error {
	if _, ok := r.surveys[s.ID]; !ok {
		return ErrSurveyNotFound
	}
	s.UpdatedAt = time.Now()
	cp := *s
	r.surveys[s.ID] = &cp
	return nil
}

func (r *surveyRepoStub) Delete(_ context.Context, id int64) error {
	delete(r.surveys, id)
	return nil
}

func (r *surveyRepoStub) List(_ context.Context, params pagination.PaginationParams, _ SurveyListFilters) ([]Survey, *pagination.PaginationResult, error) {
	out := make([]Survey, 0, len(r.surveys))
	for _, s := range r.surveys {
		out = append(out, *s)
	}
	return out, &pagination.PaginationResult{Total: int64(len(out)), Page: params.Page, PageSize: params.PageSize}, nil
}

func (r *surveyRepoStub) ListActive(_ context.Context, now time.Time) ([]Survey, error) {
	out := make([]Survey, 0)
	for _, s := range r.surveys {
		if s.IsActiveAt(now) {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (r *surveyRepoStub) GetUserStates(_ context.Context, userID int64, surveyIDs []int64) (map[int64]SurveyUserState, error) {
	out := make(map[int64]SurveyUserState)
	for _, sid := range surveyIDs {
		if st, ok := r.states[surveyStateKey(sid, userID)]; ok {
			out[sid] = st
		}
	}
	return out, nil
}

func (r *surveyRepoStub) UpsertUserState(_ context.Context, surveyID, userID int64, status string, snoozeUntil *time.Time, now time.Time) error {
	r.states[surveyStateKey(surveyID, userID)] = SurveyUserState{
		SurveyID:    surveyID,
		UserID:      userID,
		Status:      status,
		SnoozeUntil: snoozeUntil,
		UpdatedAt:   now,
	}
	return nil
}

func (r *surveyRepoStub) HasUserSubmitted(_ context.Context, surveyID, userID int64) (bool, error) {
	for _, resp := range r.responses {
		if resp.SurveyID == surveyID && resp.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

func (r *surveyRepoStub) HasRecentUsage(_ context.Context, userID int64, _ time.Time) (bool, error) {
	return r.recentUsageByUser[userID], nil
}

func (r *surveyRepoStub) SubmitAndReserveReward(_ context.Context, record *SubmitSurveyRecord) (*SubmitSurveyRecordResult, error) {
	for _, existing := range r.responses {
		if existing.SurveyID == record.SurveyID && existing.UserID == record.UserID {
			return nil, ErrSurveyAlreadySubmitted
		}
	}
	resp := SurveyResponse{
		ID:              int64(len(r.responses) + 1),
		SurveyID:        record.SurveyID,
		UserID:          record.UserID,
		Answers:         record.Answers,
		DurationSeconds: record.DurationSeconds,
		RewardStatus:    SurveyRewardStatusNone,
		RewardType:      SurveyRewardTypeNone,
		SubmittedAt:     record.SubmittedAt,
	}
	reserved := false
	if record.AttemptReward {
		if r.budgetExhausted {
			resp.RewardStatus = SurveyRewardStatusBudgetExhausted
			resp.RewardType = record.RewardType
		} else {
			resp.RewardStatus = SurveyRewardStatusPending
			resp.RewardType = record.RewardType
			resp.RewardValue = record.RewardValue
			reserved = true
		}
	}
	r.responses = append(r.responses, resp)
	r.states[surveyStateKey(record.SurveyID, record.UserID)] = SurveyUserState{
		SurveyID: record.SurveyID,
		UserID:   record.UserID,
		Status:   SurveyUserStateSubmitted,
	}
	cp := resp
	return &SubmitSurveyRecordResult{Response: &cp, RewardReserved: reserved}, nil
}

func (r *surveyRepoStub) UpdateResponseReward(_ context.Context, responseID int64, rewardStatus string, rewardValue float64, redeemCode string) error {
	for i := range r.responses {
		if r.responses[i].ID == responseID {
			r.responses[i].RewardStatus = rewardStatus
			r.responses[i].RewardValue = rewardValue
			r.responses[i].RewardRedeemCode = redeemCode
		}
	}
	return nil
}

func (r *surveyRepoStub) ReleaseRewardBudget(_ context.Context, _ int64) error {
	r.releasedBudget++
	return nil
}

func (r *surveyRepoStub) GetMicroOptionCounts(_ context.Context, surveyID int64, questionID string) (map[string]int64, error) {
	counts := make(map[string]int64)
	for _, resp := range r.responses {
		if resp.SurveyID != surveyID {
			continue
		}
		if ans, ok := resp.Answers[questionID]; ok && ans.ChoiceID != "" {
			counts[ans.ChoiceID]++
		}
	}
	return counts, nil
}

func (r *surveyRepoStub) ListAllResponsesForStats(_ context.Context, surveyID int64) ([]SurveyResponse, error) {
	out := make([]SurveyResponse, 0)
	for _, resp := range r.responses {
		if resp.SurveyID == surveyID {
			out = append(out, resp)
		}
	}
	return out, nil
}

func (r *surveyRepoStub) CountUserStatesByStatus(_ context.Context, surveyID int64) (int64, int64, error) {
	var snoozed, dismissed int64
	for _, st := range r.states {
		if st.SurveyID != surveyID {
			continue
		}
		if st.Status == SurveyUserStateSnoozed {
			snoozed++
		} else if st.Status == SurveyUserStateDismissed {
			dismissed++
		}
	}
	return snoozed, dismissed, nil
}

func (r *surveyRepoStub) ListResponses(_ context.Context, surveyID int64, params pagination.PaginationParams, _ string) ([]SurveyResponse, *pagination.PaginationResult, error) {
	items, _ := r.ListAllResponsesForStats(context.Background(), surveyID)
	return items, &pagination.PaginationResult{Total: int64(len(items)), Page: params.Page, PageSize: params.PageSize}, nil
}

type surveyUserRepoStub struct {
	UserRepository
	users map[int64]*User
}

func (u *surveyUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	item, ok := u.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return item, nil
}

type surveyUserSubRepoStub struct {
	UserSubscriptionRepository
	subs map[int64][]UserSubscription
}

func (s *surveyUserSubRepoStub) ListActiveByUserID(_ context.Context, userID int64) ([]UserSubscription, error) {
	return s.subs[userID], nil
}

type surveyRewardIssuerStub struct {
	createdCodes []*RedeemCode
	redeemed     []struct {
		userID int64
		code   string
		amount float64
		source string
	}
	createErr      error
	redeemErr      error
	seenCtxErrs    []error
}

func (i *surveyRewardIssuerStub) CreateCode(ctx context.Context, code *RedeemCode) error {
	i.seenCtxErrs = append(i.seenCtxErrs, ctx.Err())
	if i.createErr != nil {
		return i.createErr
	}
	cp := *code
	i.createdCodes = append(i.createdCodes, &cp)
	return nil
}

func (i *surveyRewardIssuerStub) RedeemForSurveyReward(ctx context.Context, userID int64, code string, amount float64, source string) (*RedeemCode, error) {
	i.seenCtxErrs = append(i.seenCtxErrs, ctx.Err())
	if i.redeemErr != nil {
		return nil, i.redeemErr
	}
	i.redeemed = append(i.redeemed, struct {
		userID int64
		code   string
		amount float64
		source string
	}{userID: userID, code: code, amount: amount, source: source})
	return &RedeemCode{Code: code, Value: amount, Type: RedeemTypeBalance, Status: StatusUsed}, nil
}

func sampleSurveyQuestions() []SurveyQuestion {
	return []SurveyQuestion{
		{
			ID:          "q1",
			Type:        SurveyQuestionTypeSingle,
			Title:       "主要使用场景",
			Required:    true,
			AutoAdvance: true,
			Options: []SurveyQuestionOption{
				{ID: "coding", Label: "AI 编程助手"},
				{ID: "app", Label: "自建应用集成"},
			},
		},
		{
			ID:       "q2",
			Type:     SurveyQuestionTypeMulti,
			Title:    "常用模型",
			Required: false,
			Options: []SurveyQuestionOption{
				{ID: "claude", Label: "Claude"},
				{ID: "gpt", Label: "GPT"},
			},
		},
		{
			ID:       "q3",
			Type:     SurveyQuestionTypeRating,
			Title:    "稳定性评分",
			Required: true,
			ScaleMin: 1,
			ScaleMax: 5,
		},
		{
			ID:                     "q4",
			Type:                   SurveyQuestionTypeTagsText,
			Title:                  "建议补充",
			Required:               false,
			LinkedRatingQuestionID: "q3",
			LowScoreThreshold:      3,
			LowChips:               []string{"晚高峰变慢", "报错提示不清"},
			HighChips:              []string{"希望上线新模型", "保持稳定就好"},
		},
	}
}

func TestSurveyServiceCreateValidation(t *testing.T) {
	svc := NewSurveyServiceWithIssuer(newSurveyRepoStub(), nil, nil, nil)

	// micro 模式只能有一道单选题
	_, err := svc.Create(context.Background(), &CreateSurveyInput{
		Title:       "快问测试",
		DisplayMode: SurveyDisplayModeMicro,
		Questions:   sampleSurveyQuestions(),
	})
	require.Error(t, err)

	// 奖励金额超限被拒
	_, err = svc.Create(context.Background(), &CreateSurveyInput{
		Title:       "异常奖励",
		DisplayMode: SurveyDisplayModeCorner,
		Questions:   sampleSurveyQuestions(),
		RewardType:  SurveyRewardTypeBalance,
		RewardValue: 5000,
	})
	require.ErrorIs(t, err, ErrSurveyInvalidReward)

	// 正常创建
	budget := 100
	created, err := svc.Create(context.Background(), &CreateSurveyInput{
		Title:             "体验调研",
		Status:            SurveyStatusActive,
		DisplayMode:       SurveyDisplayModeCorner,
		Questions:         sampleSurveyQuestions(),
		RewardType:        SurveyRewardTypeBalance,
		RewardValue:       0.5,
		RewardBudgetLimit: &budget,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), created.ID)
	require.Equal(t, 3, created.DeliveryPolicy.SnoozeDays)
	require.Equal(t, 4, created.DeliveryPolicy.DelaySeconds)
}

func TestSurveyServiceAudienceAndDeliveryPolicyFiltering(t *testing.T) {
	repo := newSurveyRepoStub()
	now := time.Now()
	users := &surveyUserRepoStub{
		users: map[int64]*User{
			10: {ID: 10, Email: "active@example.com", Balance: 25, CreatedAt: now.Add(-15 * 24 * time.Hour)},
			11: {ID: 11, Email: "brandnew@example.com", Balance: 25, CreatedAt: now.Add(-2 * time.Hour)},
			12: {ID: 12, Email: "inactive@example.com", Balance: 25, CreatedAt: now.Add(-20 * 24 * time.Hour)},
		},
	}
	repo.recentUsageByUser[10] = true
	repo.recentUsageByUser[11] = true
	repo.recentUsageByUser[12] = false

	subs := &surveyUserSubRepoStub{subs: map[int64][]UserSubscription{}}
	svc := NewSurveyServiceWithIssuer(repo, users, subs, nil)

	policy := SurveyDeliveryPolicy{
		SnoozeDays:               3,
		MinAccountAgeDays:        7,
		RequireRecentUsageDays:   14,
		SkipIfAnnouncementUnread: true,
		DelaySeconds:             4,
	}
	survey, err := svc.Create(context.Background(), &CreateSurveyInput{
		Title:          "活跃老客户调研",
		Status:         SurveyStatusActive,
		DisplayMode:    SurveyDisplayModeCorner,
		Questions:      sampleSurveyQuestions(),
		DeliveryPolicy: &policy,
	})
	require.NoError(t, err)

	// 用户 10 满足账号年龄和近 14 天活跃调用
	list10, err := svc.ListActiveForUser(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, list10, 1)

	// 用户 11 注册不足 7 天 -> 过滤
	list11, err := svc.ListActiveForUser(context.Background(), 11)
	require.NoError(t, err)
	require.Empty(t, list11)

	// 用户 12 近 14 天无调用 -> 过滤
	list12, err := svc.ListActiveForUser(context.Background(), 12)
	require.NoError(t, err)
	require.Empty(t, list12)

	// 用户 10 点「稍后再说」-> 冷却期内隐藏
	until, err := svc.RecordUserAction(context.Background(), 10, survey.ID, SurveyActionSnooze)
	require.NoError(t, err)
	require.NotNil(t, until)
	list10AfterSnooze, err := svc.ListActiveForUser(context.Background(), 10)
	require.NoError(t, err)
	require.Empty(t, list10AfterSnooze)
}

func TestSurveyServiceSubmitRewardsAndIdempotency(t *testing.T) {
	t.Run("auto balance reward grants once and blocks duplicate submit", func(t *testing.T) {
		repo := newSurveyRepoStub()
		users := &surveyUserRepoStub{users: map[int64]*User{
			1: {ID: 1, Email: "u1@example.com", Balance: 10, CreatedAt: time.Now().Add(-10 * 24 * time.Hour)},
		}}
		subs := &surveyUserSubRepoStub{subs: map[int64][]UserSubscription{}}
		issuer := &surveyRewardIssuerStub{}
		svc := NewSurveyServiceWithIssuer(repo, users, subs, issuer)

		budget := 10
		created, err := svc.Create(context.Background(), &CreateSurveyInput{
			Title:             "有奖体验调研",
			Status:            SurveyStatusActive,
			DisplayMode:       SurveyDisplayModeModal,
			Questions:         sampleSurveyQuestions(),
			RewardType:        SurveyRewardTypeBalance,
			RewardValue:       0.5,
			RewardBudgetLimit: &budget,
		})
		require.NoError(t, err)

		score := 5
		out, err := svc.Submit(context.Background(), 1, created.ID, &SubmitSurveyInput{
			DurationSeconds: 28,
			Answers: SurveyAnswers{
				"q1": {ChoiceID: "coding"},
				"q2": {ChoiceIDs: []string{"claude", "gpt"}},
				"q3": {Rating: &score},
				"q4": {Tags: []string{"希望上线新模型"}, Text: "响应很快"},
			},
		})
		require.NoError(t, err)
		require.Equal(t, SurveyRewardStatusGranted, out.Response.RewardStatus)
		require.Equal(t, 0.5, out.Response.RewardValue)
		require.NotEmpty(t, out.Response.RewardRedeemCode)
		require.Len(t, issuer.createdCodes, 1)
		require.Len(t, issuer.redeemed, 1)
		require.Equal(t, "survey:1", issuer.redeemed[0].source)

		// 重复提交应被拒绝，且不会再发奖
		_, err = svc.Submit(context.Background(), 1, created.ID, &SubmitSurveyInput{
			Answers: SurveyAnswers{
				"q1": {ChoiceID: "coding"},
				"q3": {Rating: &score},
			},
		})
		require.ErrorIs(t, err, ErrSurveyAlreadySubmitted)
		require.Len(t, issuer.redeemed, 1)
	})

	t.Run("redeem_code reward creates code without auto redeeming", func(t *testing.T) {
		repo := newSurveyRepoStub()
		users := &surveyUserRepoStub{users: map[int64]*User{2: {ID: 2, Balance: 5}}}
		subs := &surveyUserSubRepoStub{subs: map[int64][]UserSubscription{}}
		issuer := &surveyRewardIssuerStub{}
		svc := NewSurveyServiceWithIssuer(repo, users, subs, issuer)

		created, err := svc.Create(context.Background(), &CreateSurveyInput{
			Title:               "兑换码调研",
			Status:              SurveyStatusActive,
			DisplayMode:         SurveyDisplayModeCorner,
			Questions:           sampleSurveyQuestions(),
			RewardType:          SurveyRewardTypeRedeemCode,
			RewardValue:         1.0,
			RewardCodeValidDays: 14,
		})
		require.NoError(t, err)

		score := 4
		out, err := svc.Submit(context.Background(), 2, created.ID, &SubmitSurveyInput{
			Answers: SurveyAnswers{"q1": {ChoiceID: "app"}, "q3": {Rating: &score}},
		})
		require.NoError(t, err)
		require.Equal(t, SurveyRewardStatusCodeIssued, out.Response.RewardStatus)
		require.NotEmpty(t, out.Response.RewardRedeemCode)
		require.Len(t, issuer.createdCodes, 1)
		require.NotNil(t, issuer.createdCodes[0].ExpiresAt)
		require.Empty(t, issuer.redeemed)
	})

	t.Run("budget exhausted saves response without issuing reward", func(t *testing.T) {
		repo := newSurveyRepoStub()
		repo.budgetExhausted = true
		users := &surveyUserRepoStub{users: map[int64]*User{3: {ID: 3, Balance: 5}}}
		subs := &surveyUserSubRepoStub{subs: map[int64][]UserSubscription{}}
		issuer := &surveyRewardIssuerStub{}
		svc := NewSurveyServiceWithIssuer(repo, users, subs, issuer)

		created, err := svc.Create(context.Background(), &CreateSurveyInput{
			Title:       "限量有奖",
			Status:      SurveyStatusActive,
			DisplayMode: SurveyDisplayModeModal,
			Questions:   sampleSurveyQuestions(),
			RewardType:  SurveyRewardTypeBalance,
			RewardValue: 0.5,
		})
		require.NoError(t, err)

		score := 3
		out, err := svc.Submit(context.Background(), 3, created.ID, &SubmitSurveyInput{
			Answers: SurveyAnswers{"q1": {ChoiceID: "coding"}, "q3": {Rating: &score}},
		})
		require.NoError(t, err)
		require.Equal(t, SurveyRewardStatusBudgetExhausted, out.Response.RewardStatus)
		require.Zero(t, out.Response.RewardValue)
		require.Empty(t, issuer.createdCodes)
	})

	t.Run("auto redeem failure degrades gracefully to code_issued", func(t *testing.T) {
		repo := newSurveyRepoStub()
		users := &surveyUserRepoStub{users: map[int64]*User{4: {ID: 4, Balance: 5}}}
		subs := &surveyUserSubRepoStub{subs: map[int64][]UserSubscription{}}
		issuer := &surveyRewardIssuerStub{redeemErr: errors.New("temporary lock")}
		svc := NewSurveyServiceWithIssuer(repo, users, subs, issuer)

		created, err := svc.Create(context.Background(), &CreateSurveyInput{
			Title:       "降级保护",
			Status:      SurveyStatusActive,
			DisplayMode: SurveyDisplayModeCorner,
			Questions:   sampleSurveyQuestions(),
			RewardType:  SurveyRewardTypeBalance,
			RewardValue: 0.5,
		})
		require.NoError(t, err)

		score := 4
		out, err := svc.Submit(context.Background(), 4, created.ID, &SubmitSurveyInput{
			Answers: SurveyAnswers{"q1": {ChoiceID: "coding"}, "q3": {Rating: &score}},
		})
		require.NoError(t, err)
		require.Equal(t, SurveyRewardStatusCodeIssued, out.Response.RewardStatus)
		require.NotEmpty(t, out.Response.RewardRedeemCode)
	})

	t.Run("micro poll returns live option percentages after submit", func(t *testing.T) {
		repo := newSurveyRepoStub()
		users := &surveyUserRepoStub{users: map[int64]*User{
			1: {ID: 1, Balance: 5},
			2: {ID: 2, Balance: 5},
		}}
		subs := &surveyUserSubRepoStub{subs: map[int64][]UserSubscription{}}
		svc := NewSurveyServiceWithIssuer(repo, users, subs, nil)

		created, err := svc.Create(context.Background(), &CreateSurveyInput{
			Title:       "下周优先做什么？",
			Status:      SurveyStatusActive,
			DisplayMode: SurveyDisplayModeMicro,
			Questions: []SurveyQuestion{
				{
					ID:       "poll",
					Type:     SurveyQuestionTypeSingle,
					Title:    "最想要的新能力？",
					Required: true,
					Options: []SurveyQuestionOption{
						{ID: "budget", Label: "Key 日限额"},
						{ID: "status", Label: "延迟看板"},
					},
				},
			},
		})
		require.NoError(t, err)

		_, err = svc.Submit(context.Background(), 1, created.ID, &SubmitSurveyInput{
			Answers: SurveyAnswers{"poll": {ChoiceID: "status"}},
		})
		require.NoError(t, err)

		out2, err := svc.Submit(context.Background(), 2, created.ID, &SubmitSurveyInput{
			Answers: SurveyAnswers{"poll": {ChoiceID: "budget"}},
		})
		require.NoError(t, err)
		require.Len(t, out2.MicroResults, 2)
		require.Equal(t, 50.0, out2.MicroResults[0].Percent)
		require.Equal(t, 50.0, out2.MicroResults[1].Percent)
	})
}

func TestBuildSurveyStats(t *testing.T) {
	survey := &Survey{
		ID:        9,
		Title:     "统计测试",
		Questions: sampleSurveyQuestions(),
	}
	s5, s2 := 5, 2
	responses := []SurveyResponse{
		{
			ID: 1, SurveyID: 9, UserID: 1, UserEmail: "a@test.com",
			DurationSeconds: 20, RewardStatus: SurveyRewardStatusGranted, RewardValue: 0.5,
			SubmittedAt: time.Now().Add(-time.Minute),
			Answers: SurveyAnswers{
				"q1": {ChoiceID: "coding"},
				"q2": {ChoiceIDs: []string{"claude", "gpt"}},
				"q3": {Rating: &s5},
				"q4": {Tags: []string{"希望上线新模型"}, Text: "很不错"},
			},
		},
		{
			ID: 2, SurveyID: 9, UserID: 2, UserEmail: "b@test.com",
			DurationSeconds: 40, RewardStatus: SurveyRewardStatusCodeIssued, RewardValue: 0.5,
			SubmittedAt: time.Now(),
			Answers: SurveyAnswers{
				"q1": {ChoiceID: "app"},
				"q2": {ChoiceIDs: []string{"claude"}},
				"q3": {Rating: &s2},
				"q4": {Tags: []string{"晚高峰变慢", "希望上线新模型"}, Text: "晚上偶尔卡"},
			},
		},
	}

	stats := BuildSurveyStats(survey, responses, 3, 1)
	require.Equal(t, int64(2), stats.TotalResponses)
	require.Equal(t, int64(3), stats.SnoozedCount)
	require.Equal(t, int64(1), stats.DismissedCount)
	require.Equal(t, 30.0, stats.AverageDurationSec)
	require.Equal(t, int64(2), stats.RewardIssuedCount)
	require.Equal(t, 1.0, stats.RewardTotalValue)
	require.Len(t, stats.Questions, 4)
	require.Equal(t, 3.5, stats.Questions[2].AverageRating)
	require.Equal(t, "希望上线新模型", stats.Questions[3].TopTags[0].Tag)
	require.Equal(t, int64(2), stats.Questions[3].TopTags[0].Count)
	require.Len(t, stats.Questions[3].RecentTexts, 2)
}

type surveyRedeemRepoStub struct {
	RedeemCodeRepository
	byCode map[string]*RedeemCode
}

func (r *surveyRedeemRepoStub) GetByCode(_ context.Context, code string) (*RedeemCode, error) {
	item, ok := r.byCode[code]
	if !ok {
		return nil, ErrRedeemCodeNotFound
	}
	cp := *item
	return &cp, nil
}

func TestSurveyRewardCodeFormatAndOwnerGuard(t *testing.T) {
	code, err := generateSurveyRewardCode(12, 101)
	require.NoError(t, err)
	require.LessOrEqual(t, len(code), 32)

	sid, uid, ok := parseSurveyRewardCodeMeta(code)
	require.True(t, ok)
	require.Equal(t, int64(12), sid)
	require.Equal(t, int64(101), uid)
	require.Equal(t, "survey:12", surveySourceFromRedeemCode(code))

	// 超大 ID 仍需满足 redeem_codes.code VARCHAR(32) 长度上限
	bigCode, err := generateSurveyRewardCode(987654321, 987654321012)
	require.NoError(t, err)
	require.LessOrEqual(t, len(bigCode), 32)
	bigSID, bigUID, ok := parseSurveyRewardCodeMeta(bigCode)
	require.True(t, ok)
	require.Equal(t, int64(987654321), bigSID)
	require.Equal(t, int64(987654321012), bigUID)

	rc := &RedeemCode{
		ID:     501,
		Code:   code,
		Type:   RedeemTypeBalance,
		Value:  0.5,
		Status: StatusUnused,
		Notes:  "问卷调查奖励 #12: 体验调研",
	}
	require.True(t, isSurveyRewardRedeemCode(rc))
	require.False(t, isSurveyRewardRedeemCode(&RedeemCode{
		Code:  "ABCD1234-EF567890-11223344-55667788",
		Notes: "normal purchase",
	}))
	require.False(t, isSurveyRewardRedeemCode(&RedeemCode{
		Code:  "SRV-FAKE",
		Notes: "admin manual note",
	}))

	// 他人尝试核销本人的问卷专属码应返回 403
	redeemSvc := NewRedeemService(
		&surveyRedeemRepoStub{byCode: map[string]*RedeemCode{code: rc}},
		&surveyUserRepoStub{users: map[int64]*User{
			101: {ID: 101, Balance: 5},
			202: {ID: 202, Balance: 5},
		}},
		nil, nil, nil, nil, nil, nil,
	)
	_, err = redeemSvc.Redeem(context.Background(), 202, code)
	require.Error(t, err)
	var appErr *infraerrors.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, int32(http.StatusForbidden), appErr.Code)
	require.Equal(t, "SURVEY_REWARD_CODE_OWNER_MISMATCH", appErr.Reason)
}

func TestSurveyFulfillRewardDetachesFromCancelledContextAndHandlesAlreadyUsed(t *testing.T) {
	repo := newSurveyRepoStub()
	users := &surveyUserRepoStub{users: map[int64]*User{5: {ID: 5, Balance: 10}}}
	subs := &surveyUserSubRepoStub{subs: map[int64][]UserSubscription{}}
	issuer := &surveyRewardIssuerStub{redeemErr: ErrRedeemCodeUsed}
	svc := NewSurveyServiceWithIssuer(repo, users, subs, issuer)

	created, err := svc.Create(context.Background(), &CreateSurveyInput{
		Title:       "断连与重入保护",
		Status:      SurveyStatusActive,
		DisplayMode: SurveyDisplayModeCorner,
		Questions:   sampleSurveyQuestions(),
		RewardType:  SurveyRewardTypeBalance,
		RewardValue: 0.5,
	})
	require.NoError(t, err)

	// 模拟答卷落库后客户端取消请求：发奖阶段应使用 detached ctx 正常完成，且 ErrRedeemCodeUsed 不降级
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	resp := &SurveyResponse{
		ID:           99,
		SurveyID:     created.ID,
		UserID:       5,
		RewardStatus: SurveyRewardStatusPending,
		RewardType:   SurveyRewardTypeBalance,
		RewardValue:  0.5,
	}
	svc.fulfillSurveyReward(cancelledCtx, created, 5, resp)

	require.Equal(t, SurveyRewardStatusGranted, resp.RewardStatus)
	require.NotEmpty(t, resp.RewardRedeemCode)
	require.Len(t, issuer.seenCtxErrs, 2)
	require.NoError(t, issuer.seenCtxErrs[0])
	require.NoError(t, issuer.seenCtxErrs[1])
}
