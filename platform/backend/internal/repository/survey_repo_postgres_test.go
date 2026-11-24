//go:build survey_pg_integration

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func openSurveyPGTestDB(t *testing.T) (*sql.DB, *dbent.Client) {
	t.Helper()
	dsn := os.Getenv("SURVEY_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set SURVEY_TEST_POSTGRES_DSN to run real PostgreSQL survey integration tests")
	}
	require.NoError(t, timezone.Init("UTC"))

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.PingContext(context.Background()))

	entClient := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = entClient.Close() })
	return db, entClient
}

func TestSurveyPostgresMigrationsAndEndToEndFlow(t *testing.T) {
	ctx := context.Background()
	db, entClient := openSurveyPGTestDB(t)

	// 1. 执行完整生产迁移链（含 253_customer_surveys.sql），并再次执行验证幂等性
	require.NoError(t, ApplyMigrations(ctx, db))
	require.NoError(t, ApplyMigrations(ctx, db))

	var migrationCount int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE filename = '253_customer_surveys.sql'`).Scan(&migrationCount)
	require.NoError(t, err)
	require.Equal(t, 1, migrationCount)

	var tablesOK bool
	err = db.QueryRowContext(ctx, `
		SELECT
			to_regclass('public.surveys') IS NOT NULL
			AND to_regclass('public.survey_responses') IS NOT NULL
			AND to_regclass('public.survey_user_states') IS NOT NULL
	`).Scan(&tablesOK)
	require.NoError(t, err)
	require.True(t, tablesOK)

	// 2. 开启 GABUGU_LEGACY_WALLET_COMPAT=true 并安装真实 Postgres 余额触发器审计表，
	// 验证自动发奖与手动兑换码核销均在 UPDATE users.balance 同一事务内写入 gift 赠送 GUC。
	t.Setenv("GABUGU_LEGACY_WALLET_COMPAT", "true")
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS survey_wallet_guc_audit (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			delta NUMERIC(20, 8) NOT NULL,
			wallet_credit TEXT NOT NULL
		);
		TRUNCATE survey_wallet_guc_audit;
		CREATE OR REPLACE FUNCTION survey_capture_wallet_credit() RETURNS trigger AS $$
		BEGIN
			IF NEW.balance IS DISTINCT FROM OLD.balance THEN
				INSERT INTO survey_wallet_guc_audit (user_id, delta, wallet_credit)
				VALUES (NEW.id, NEW.balance - OLD.balance, COALESCE(current_setting('gabugu.wallet_credit', true), ''));
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
		DROP TRIGGER IF EXISTS trg_survey_capture_wallet_credit ON users;
		CREATE TRIGGER trg_survey_capture_wallet_credit
			BEFORE UPDATE OF balance ON users
			FOR EACH ROW
			EXECUTE FUNCTION survey_capture_wallet_credit();
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `
			DROP TRIGGER IF EXISTS trg_survey_capture_wallet_credit ON users;
			DROP FUNCTION IF EXISTS survey_capture_wallet_credit();
			DROP TABLE IF EXISTS survey_wallet_guc_audit;
		`)
	})

	// 3. 启用真实邀请返利配置（正对照：普通充值码应返利，问卷奖励必须屏蔽返利）
	settingRepo := NewSettingRepository(entClient)
	require.NoError(t, settingRepo.SetMultiple(ctx, map[string]string{
		service.SettingKeyAffiliateEnabled:           "true",
		service.SettingKeyAffiliateRebateRate:        "10",
		service.SettingKeyAffiliateRebateFreezeHours: "0",
	}))
	settingSvc := service.NewSettingService(settingRepo, &config.Config{})
	affiliateRepo := NewAffiliateRepository(entClient, db)
	affiliateSvc := service.NewAffiliateService(affiliateRepo, settingSvc, nil, nil)

	userRepo := NewUserRepository(entClient, db)
	subRepo := NewUserSubscriptionRepository(entClient)
	redeemRepo := NewRedeemCodeRepository(entClient)
	surveyRepo := NewSurveyRepository(db)

	redeemSvc := service.NewRedeemService(
		redeemRepo,
		userRepo,
		nil, // subscriptionService not needed for balance codes
		nil, // cache
		nil, // billingCache
		entClient,
		nil, // authCacheInvalidator
		affiliateSvc,
	)
	surveySvc := service.NewSurveyService(surveyRepo, userRepo, subRepo, redeemSvc)

	// 创建邀请人 + 多名真实测试用户（全部通过真实 Ent 写入 users 表）
	inviter, err := entClient.User.Create().
		SetEmail(fmt.Sprintf("inviter-%d@example.invalid", time.Now().UnixNano())).
		SetUsername("inviter").
		SetPasswordHash("synthetic").
		SetBalance(0).
		Save(ctx)
	require.NoError(t, err)
	inviterSummary, err := affiliateSvc.EnsureUserAffiliate(ctx, inviter.ID)
	require.NoError(t, err)
	require.NotEmpty(t, inviterSummary.AffCode)

	createUser := func(tag string, balance float64, createdAt time.Time) *dbent.User {
		u, createErr := entClient.User.Create().
			SetEmail(fmt.Sprintf("%s-%d@example.invalid", tag, time.Now().UnixNano())).
			SetUsername(tag).
			SetPasswordHash("synthetic").
			SetBalance(balance).
			SetCreatedAt(createdAt).
			Save(ctx)
		require.NoError(t, createErr)
		return u
	}

	oldTime := time.Now().Add(-20 * 24 * time.Hour)
	u1 := createUser("survey_u1", 10.0, oldTime)
	require.NoError(t, affiliateSvc.BindInviterByCode(ctx, u1.ID, inviterSummary.AffCode))

	// 正对照：普通余额兑换码应正常触发邀请返利
	normalCode := fmt.Sprintf("NORM-%d", time.Now().UnixNano()%1000000)
	require.NoError(t, redeemSvc.CreateCode(ctx, &service.RedeemCode{
		Code:   normalCode,
		Type:   service.RedeemTypeBalance,
		Value:  10.0,
		Status: service.StatusUnused,
		Notes:  "regular topup",
	}))
	_, err = redeemSvc.Redeem(ctx, u1.ID, normalCode)
	require.NoError(t, err)
	inviterAfterNormal, err := affiliateSvc.EnsureUserAffiliate(ctx, inviter.ID)
	require.NoError(t, err)
	require.InDelta(t, 1.0, inviterAfterNormal.AffHistoryQuota, 0.001, "positive control: normal balance redeem should accrue 10%% affiliate rebate")

	// 准备近期真实 usage_logs 用于校验 require_recent_usage_days
	apiKey, err := entClient.APIKey.Create().
		SetUserID(u1.ID).
		SetName("survey-test-key").
		SetKey(fmt.Sprintf("sk-survey-%d", time.Now().UnixNano())).
		Save(ctx)
	require.NoError(t, err)
	account, err := entClient.Account.Create().
		SetName(fmt.Sprintf("survey-acc-%d", time.Now().UnixNano())).
		SetPlatform("anthropic").
		SetType("apikey").
		SetCredentials(map[string]any{"api_key": "synthetic"}).
		Save(ctx)
	require.NoError(t, err)

	insertRecentUsage := func(userID int64) {
		_, uErr := db.ExecContext(ctx, `
			INSERT INTO usage_logs (user_id, api_key_id, account_id, model, input_tokens, output_tokens, total_cost, actual_cost, created_at)
			VALUES ($1, $2, $3, 'claude-sonnet-4', 100, 50, 0.01, 0.01, NOW())
		`, userID, apiKey.ID, account.ID)
		require.NoError(t, uErr)
	}
	insertRecentUsage(u1.ID)
	uNoUsage := createUser("survey_nousage", 10.0, oldTime)

	// 4. 场景 A：自动赠送余额（balance）+ 近期活跃过滤 + 防重提交 + legacy gift GUC + 屏蔽返利
	budget := 2
	policy := service.SurveyDeliveryPolicy{
		SnoozeDays:               3,
		MinAccountAgeDays:        7,
		RequireRecentUsageDays:   7,
		SkipIfAnnouncementUnread: true,
		DelaySeconds:             4,
		ShowMicroResults:         true,
	}
	autoSurvey, err := surveySvc.Create(ctx, &service.CreateSurveyInput{
		Title:             "真实库自动入账体验调研",
		Description:       "验证 Postgres 迁移、并发预算与账务标记",
		Status:            service.SurveyStatusActive,
		DisplayMode:       service.SurveyDisplayModeCorner,
		DeliveryPolicy:    &policy,
		RewardType:        service.SurveyRewardTypeBalance,
		RewardValue:       0.50,
		RewardBudgetLimit: &budget,
		Questions: []service.SurveyQuestion{
			{
				ID:          "q1",
				Type:        service.SurveyQuestionTypeSingle,
				Title:       "主要用途？",
				Required:    true,
				AutoAdvance: true,
				Options: []service.SurveyQuestionOption{
					{ID: "coding", Label: "AI 编程"},
					{ID: "app", Label: "API 集成"},
				},
			},
			{
				ID:       "q2",
				Type:     service.SurveyQuestionTypeRating,
				Title:    "稳定性评分",
				Required: true,
				ScaleMin: 1,
				ScaleMax: 5,
			},
			{
				ID:                     "q3",
				Type:                   service.SurveyQuestionTypeTagsText,
				Title:                  "建议",
				Required:               false,
				LinkedRatingQuestionID: "q2",
				LowScoreThreshold:      3,
				LowChips:               []string{"晚高峰变慢"},
				HighChips:              []string{"希望上线新模型"},
			},
		},
	})
	require.NoError(t, err)

	// 无近 7 天调用记录的用户不可见，有调用记录的用户可见
	visibleNoUsage, err := surveySvc.ListActiveForUser(ctx, uNoUsage.ID)
	require.NoError(t, err)
	require.Empty(t, visibleNoUsage)
	visibleU1, err := surveySvc.ListActiveForUser(ctx, u1.ID)
	require.NoError(t, err)
	require.Len(t, visibleU1, 1)
	require.Equal(t, autoSurvey.ID, visibleU1[0].ID)

	score5 := 5
	outU1, err := surveySvc.Submit(ctx, u1.ID, autoSurvey.ID, &service.SubmitSurveyInput{
		DurationSeconds: 22,
		Answers: service.SurveyAnswers{
			"q1": {ChoiceID: "coding"},
			"q2": {Rating: &score5},
			"q3": {Tags: []string{"希望上线新模型"}, Text: "Postgres 实库验证通过"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, service.SurveyRewardStatusGranted, outU1.Response.RewardStatus)
	require.InDelta(t, 0.50, outU1.Response.RewardValue, 0.0001)
	require.LessOrEqual(t, len(outU1.Response.RewardRedeemCode), 32)

	// 核对真实 users.balance 精确增加 +0.50（初始 10 + 普通码 10 + 问卷 0.50 = 20.50）
	u1Reload, err := userRepo.GetByID(ctx, u1.ID)
	require.NoError(t, err)
	require.InDelta(t, 20.50, u1Reload.Balance, 0.0001)

	// 核对真实 Postgres 触发器收到的 gabugu.wallet_credit GUC（cash=0, gift=0.5, source=survey:<id>）
	var rawGUC string
	var gucDelta float64
	err = db.QueryRowContext(ctx, `
		SELECT delta, wallet_credit
		FROM survey_wallet_guc_audit
		WHERE user_id = $1
		ORDER BY id DESC
		LIMIT 1
	`, u1.ID).Scan(&gucDelta, &rawGUC)
	require.NoError(t, err)
	require.InDelta(t, 0.50, gucDelta, 0.0001)
	var parsedGUC map[string]any
	require.NoError(t, json.Unmarshal([]byte(rawGUC), &parsedGUC))
	require.Equal(t, float64(0), parsedGUC["cash"])
	require.Equal(t, 0.5, parsedGUC["gift"])
	require.Equal(t, fmt.Sprintf("survey:%d", autoSurvey.ID), parsedGUC["source"])

	// 核对邀请返利未被问卷奖励触发（仍是 1.0）
	inviterAfterAuto, err := affiliateSvc.EnsureUserAffiliate(ctx, inviter.ID)
	require.NoError(t, err)
	require.InDelta(t, 1.0, inviterAfterAuto.AffHistoryQuota, 0.001, "survey auto balance grant must suppress affiliate rebate")

	// 核对重复提交被 UNIQUE (survey_id, user_id) 拒绝，且不重复发奖
	_, err = surveySvc.Submit(ctx, u1.ID, autoSurvey.ID, &service.SubmitSurveyInput{
		Answers: service.SurveyAnswers{"q1": {ChoiceID: "coding"}, "q2": {Rating: &score5}},
	})
	require.ErrorIs(t, err, service.ErrSurveyAlreadySubmitted)

	// 5. 场景 B：5 名并发用户抢 1 个剩余预算名额，验证数据库行锁原子预算防超发
	racers := make([]*dbent.User, 5)
	for i := range racers {
		racers[i] = createUser(fmt.Sprintf("racer_%d", i), 5.0, oldTime)
		insertRecentUsage(racers[i].ID)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	grantedCount := 0
	exhaustedCount := 0
	for i := range racers {
		u := racers[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			s4 := 4
			res, subErr := surveySvc.Submit(ctx, u.ID, autoSurvey.ID, &service.SubmitSurveyInput{
				DurationSeconds: 15,
				Answers:         service.SurveyAnswers{"q1": {ChoiceID: "app"}, "q2": {Rating: &s4}},
			})
			require.NoError(t, subErr)
			mu.Lock()
			defer mu.Unlock()
			switch res.Response.RewardStatus {
			case service.SurveyRewardStatusGranted:
				grantedCount++
			case service.SurveyRewardStatusBudgetExhausted:
				exhaustedCount++
			default:
				t.Errorf("unexpected reward status in budget contention: %s", res.Response.RewardStatus)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 1, grantedCount, "only 1 remaining budget slot should be granted across 5 concurrent submitters")
	require.Equal(t, 4, exhaustedCount, "remaining 4 submitters should save response with budget_exhausted")

	reloadedAuto, err := surveyRepo.GetByID(ctx, autoSurvey.ID)
	require.NoError(t, err)
	require.Equal(t, 2, reloadedAuto.RewardIssuedCount)

	// 6. 场景 C：专属兑换码（redeem_code）+ 他人核销 403 拦截 + 本人公开核销 gift GUC + 屏蔽返利
	codeSurvey, err := surveySvc.Create(ctx, &service.CreateSurveyInput{
		Title:               "真实库专属兑换码调研",
		Status:              service.SurveyStatusActive,
		DisplayMode:         service.SurveyDisplayModeModal,
		RewardType:          service.SurveyRewardTypeRedeemCode,
		RewardValue:         1.25,
		RewardCodeValidDays: 14,
		Questions: []service.SurveyQuestion{
			{
				ID:       "q1",
				Type:     service.SurveyQuestionTypeSingle,
				Title:    "是否满意？",
				Required: true,
				Options: []service.SurveyQuestionOption{
					{ID: "yes", Label: "满意"},
					{ID: "no", Label: "待改进"},
				},
			},
		},
	})
	require.NoError(t, err)

	codeOut, err := surveySvc.Submit(ctx, u1.ID, codeSurvey.ID, &service.SubmitSurveyInput{
		DurationSeconds: 10,
		Answers:         service.SurveyAnswers{"q1": {ChoiceID: "yes"}},
	})
	require.NoError(t, err)
	require.Equal(t, service.SurveyRewardStatusCodeIssued, codeOut.Response.RewardStatus)
	issuedCode := codeOut.Response.RewardRedeemCode
	require.NotEmpty(t, issuedCode)
	require.LessOrEqual(t, len(issuedCode), 32)

	// 他人尝试通过公开 Redeem 接口核销 -> 403 拒绝，码仍 unused
	_, err = redeemSvc.Redeem(ctx, uNoUsage.ID, issuedCode)
	require.Error(t, err)
	var appErr *infraerrors.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, int32(http.StatusForbidden), appErr.Code)
	require.Equal(t, "SURVEY_REWARD_CODE_OWNER_MISMATCH", appErr.Reason)

	codeRowBefore, err := redeemRepo.GetByCode(ctx, issuedCode)
	require.NoError(t, err)
	require.Equal(t, service.StatusUnused, codeRowBefore.Status)
	require.NotNil(t, codeRowBefore.ExpiresAt)

	// 本人公开核销 -> 到账 +1.25、legacy gift GUC 写入、推广返利屏蔽
	_, err = redeemSvc.Redeem(ctx, u1.ID, issuedCode)
	require.NoError(t, err)

	u1AfterCode, err := userRepo.GetByID(ctx, u1.ID)
	require.NoError(t, err)
	require.InDelta(t, 21.75, u1AfterCode.Balance, 0.0001)

	err = db.QueryRowContext(ctx, `
		SELECT delta, wallet_credit
		FROM survey_wallet_guc_audit
		WHERE user_id = $1
		ORDER BY id DESC
		LIMIT 1
	`, u1.ID).Scan(&gucDelta, &rawGUC)
	require.NoError(t, err)
	require.InDelta(t, 1.25, gucDelta, 0.0001)
	require.NoError(t, json.Unmarshal([]byte(rawGUC), &parsedGUC))
	require.Equal(t, float64(0), parsedGUC["cash"])
	require.Equal(t, 1.25, parsedGUC["gift"])
	require.Equal(t, fmt.Sprintf("survey:%d", codeSurvey.ID), parsedGUC["source"])

	inviterAfterCode, err := affiliateSvc.EnsureUserAffiliate(ctx, inviter.ID)
	require.NoError(t, err)
	require.InDelta(t, 1.0, inviterAfterCode.AffHistoryQuota, 0.001, "manual redeem of survey reward code must also suppress affiliate rebate")

	// 7. 场景 D：1 题快问 SQL JSONB GROUP BY 实时聚合 + Snooze/Dismiss 状态 + 管理端统计/搜索分页/级联删除
	microSurvey, err := surveySvc.Create(ctx, &service.CreateSurveyInput{
		Title:       "真实库快问投票",
		Status:      service.SurveyStatusActive,
		DisplayMode: service.SurveyDisplayModeMicro,
		Questions: []service.SurveyQuestion{
			{
				ID:       "poll",
				Type:     service.SurveyQuestionTypeSingle,
				Title:    "下周优先做什么？",
				Required: true,
				Options: []service.SurveyQuestionOption{
					{ID: "a", Label: "延迟看板"},
					{ID: "b", Label: "Key 日限额"},
				},
			},
		},
	})
	require.NoError(t, err)

	m1 := createUser("micro_1", 0, oldTime)
	m2 := createUser("micro_2", 0, oldTime)
	m3 := createUser("micro_3", 0, oldTime)
	mSnooze := createUser("micro_snooze", 0, oldTime)
	mDismiss := createUser("micro_dismiss", 0, oldTime)

	_, err = surveySvc.Submit(ctx, m1.ID, microSurvey.ID, &service.SubmitSurveyInput{Answers: service.SurveyAnswers{"poll": {ChoiceID: "a"}}})
	require.NoError(t, err)
	_, err = surveySvc.Submit(ctx, m2.ID, microSurvey.ID, &service.SubmitSurveyInput{Answers: service.SurveyAnswers{"poll": {ChoiceID: "a"}}})
	require.NoError(t, err)
	outM3, err := surveySvc.Submit(ctx, m3.ID, microSurvey.ID, &service.SubmitSurveyInput{Answers: service.SurveyAnswers{"poll": {ChoiceID: "b"}}})
	require.NoError(t, err)
	require.Len(t, outM3.MicroResults, 2)
	require.Equal(t, int64(2), outM3.MicroResults[0].Count)
	require.Equal(t, 66.7, outM3.MicroResults[0].Percent)
	require.Equal(t, int64(1), outM3.MicroResults[1].Count)
	require.Equal(t, 33.3, outM3.MicroResults[1].Percent)

	snoozeUntil, err := surveySvc.RecordUserAction(ctx, mSnooze.ID, microSurvey.ID, service.SurveyActionSnooze)
	require.NoError(t, err)
	require.NotNil(t, snoozeUntil)
	_, err = surveySvc.RecordUserAction(ctx, mDismiss.ID, microSurvey.ID, service.SurveyActionDismiss)
	require.NoError(t, err)

	stats, err := surveySvc.GetStats(ctx, microSurvey.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), stats.TotalResponses)
	require.Equal(t, int64(1), stats.SnoozedCount)
	require.Equal(t, int64(1), stats.DismissedCount)

	// 搜索与分页
	found, pageRes, err := surveySvc.ListResponses(ctx, autoSurvey.ID, pagination.PaginationParams{Page: 1, PageSize: 20}, "Postgres 实库验证通过")
	require.NoError(t, err)
	require.Equal(t, int64(1), pageRes.Total)
	require.Len(t, found, 1)
	require.Equal(t, u1.ID, found[0].UserID)

	// 删除问卷后级联清理 survey_responses 和 survey_user_states
	require.NoError(t, surveySvc.Delete(ctx, microSurvey.ID))
	var leftoverResp, leftoverStates int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM survey_responses WHERE survey_id = $1`, microSurvey.ID).Scan(&leftoverResp))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM survey_user_states WHERE survey_id = $1`, microSurvey.ID).Scan(&leftoverStates))
	require.Zero(t, leftoverResp)
	require.Zero(t, leftoverStates)
}
