package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type surveyRepository struct {
	db *sql.DB
}

func NewSurveyRepository(db *sql.DB) service.SurveyRepository {
	return &surveyRepository{db: db}
}

const surveySelectColumns = `
	s.id, s.title, s.description, s.status, s.display_mode,
	s.questions, s.targeting, s.delivery_policy,
	s.reward_type, s.reward_value, s.reward_budget_limit, s.reward_issued_count, s.reward_code_valid_days,
	s.starts_at, s.ends_at, s.created_by, s.updated_by, s.created_at, s.updated_at,
	(SELECT COUNT(*) FROM survey_responses sr WHERE sr.survey_id = s.id) AS response_count
`

func scanSurveyRow(row interface{ Scan(...any) error }) (*service.Survey, error) {
	var (
		item              service.Survey
		questionsRaw      []byte
		targetingRaw      []byte
		policyRaw         []byte
		rewardBudgetLimit sql.NullInt32
		startsAt          sql.NullTime
		endsAt            sql.NullTime
		createdBy         sql.NullInt64
		updatedBy         sql.NullInt64
	)

	err := row.Scan(
		&item.ID, &item.Title, &item.Description, &item.Status, &item.DisplayMode,
		&questionsRaw, &targetingRaw, &policyRaw,
		&item.RewardType, &item.RewardValue, &rewardBudgetLimit, &item.RewardIssuedCount, &item.RewardCodeValidDays,
		&startsAt, &endsAt, &createdBy, &updatedBy, &item.CreatedAt, &item.UpdatedAt,
		&item.ResponseCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSurveyNotFound
	}
	if err != nil {
		return nil, err
	}

	if len(questionsRaw) > 0 {
		if err := json.Unmarshal(questionsRaw, &item.Questions); err != nil {
			return nil, fmt.Errorf("unmarshal survey questions: %w", err)
		}
	}
	if len(targetingRaw) > 0 {
		if err := json.Unmarshal(targetingRaw, &item.Targeting); err != nil {
			return nil, fmt.Errorf("unmarshal survey targeting: %w", err)
		}
	}
	item.DeliveryPolicy = domain.DefaultSurveyDeliveryPolicy()
	if len(policyRaw) > 0 {
		if err := json.Unmarshal(policyRaw, &item.DeliveryPolicy); err != nil {
			return nil, fmt.Errorf("unmarshal survey delivery_policy: %w", err)
		}
	}
	if rewardBudgetLimit.Valid {
		v := int(rewardBudgetLimit.Int32)
		item.RewardBudgetLimit = &v
	}
	if startsAt.Valid {
		t := startsAt.Time
		item.StartsAt = &t
	}
	if endsAt.Valid {
		t := endsAt.Time
		item.EndsAt = &t
	}
	if createdBy.Valid {
		v := createdBy.Int64
		item.CreatedBy = &v
	}
	if updatedBy.Valid {
		v := updatedBy.Int64
		item.UpdatedBy = &v
	}

	return &item, nil
}

func (r *surveyRepository) Create(ctx context.Context, s *service.Survey) error {
	questionsJSON, err := json.Marshal(s.Questions)
	if err != nil {
		return err
	}
	targetingJSON, err := json.Marshal(s.Targeting)
	if err != nil {
		return err
	}
	policyJSON, err := json.Marshal(s.DeliveryPolicy)
	if err != nil {
		return err
	}

	err = r.db.QueryRowContext(ctx, `
		INSERT INTO surveys (
			title, description, status, display_mode,
			questions, targeting, delivery_policy,
			reward_type, reward_value, reward_budget_limit, reward_issued_count, reward_code_valid_days,
			starts_at, ends_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4,
			$5::jsonb, $6::jsonb, $7::jsonb,
			$8, $9, $10, 0, $11,
			$12, $13, $14, $15
		)
		RETURNING id, created_at, updated_at
	`,
		s.Title, s.Description, s.Status, s.DisplayMode,
		string(questionsJSON), string(targetingJSON), string(policyJSON),
		s.RewardType, s.RewardValue, s.RewardBudgetLimit, s.RewardCodeValidDays,
		s.StartsAt, s.EndsAt, s.CreatedBy, s.UpdatedBy,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	return err
}

func (r *surveyRepository) GetByID(ctx context.Context, id int64) (*service.Survey, error) {
	return scanSurveyRow(r.db.QueryRowContext(ctx, `SELECT `+surveySelectColumns+` FROM surveys s WHERE s.id = $1`, id))
}

func (r *surveyRepository) Update(ctx context.Context, s *service.Survey) error {
	questionsJSON, err := json.Marshal(s.Questions)
	if err != nil {
		return err
	}
	targetingJSON, err := json.Marshal(s.Targeting)
	if err != nil {
		return err
	}
	policyJSON, err := json.Marshal(s.DeliveryPolicy)
	if err != nil {
		return err
	}

	err = r.db.QueryRowContext(ctx, `
		UPDATE surveys
		SET title = $1,
		    description = $2,
		    status = $3,
		    display_mode = $4,
		    questions = $5::jsonb,
		    targeting = $6::jsonb,
		    delivery_policy = $7::jsonb,
		    reward_type = $8,
		    reward_value = $9,
		    reward_budget_limit = $10,
		    reward_code_valid_days = $11,
		    starts_at = $12,
		    ends_at = $13,
		    updated_by = $14,
		    updated_at = NOW()
		WHERE id = $15
		RETURNING updated_at
	`,
		s.Title, s.Description, s.Status, s.DisplayMode,
		string(questionsJSON), string(targetingJSON), string(policyJSON),
		s.RewardType, s.RewardValue, s.RewardBudgetLimit, s.RewardCodeValidDays,
		s.StartsAt, s.EndsAt, s.UpdatedBy, s.ID,
	).Scan(&s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrSurveyNotFound
	}
	return err
}

func (r *surveyRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM surveys WHERE id = $1`, id)
	return err
}

func (r *surveyRepository) List(
	ctx context.Context,
	params pagination.PaginationParams,
	filters service.SurveyListFilters,
) ([]service.Survey, *pagination.PaginationResult, error) {
	whereParts := make([]string, 0, 2)
	args := make([]any, 0, 4)

	if filters.Status != "" {
		args = append(args, filters.Status)
		whereParts = append(whereParts, fmt.Sprintf("s.status = $%d", len(args)))
	}
	if filters.Search != "" {
		args = append(args, "%"+filters.Search+"%")
		whereParts = append(whereParts, fmt.Sprintf("(s.title ILIKE $%d OR s.description ILIKE $%d)", len(args), len(args)))
	}

	whereSQL := ""
	if len(whereParts) > 0 {
		whereSQL = " WHERE " + strings.Join(whereParts, " AND ")
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM surveys s`+whereSQL, args...).Scan(&total); err != nil {
		return nil, nil, err
	}

	orderCol := surveyListOrderColumn(params.SortBy)
	sortDir := "DESC"
	if params.NormalizedSortOrder(pagination.SortOrderDesc) == pagination.SortOrderAsc {
		sortDir = "ASC"
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	listArgs := append(args, params.Limit(), params.Offset())

	query := fmt.Sprintf(
		`SELECT %s FROM surveys s%s ORDER BY %s %s, s.id %s LIMIT $%d OFFSET $%d`,
		surveySelectColumns, whereSQL, orderCol, sortDir, sortDir, limitIdx, offsetIdx,
	)

	rows, err := r.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.Survey, 0)
	for rows.Next() {
		item, err := scanSurveyRow(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return items, paginationResultFromTotal(total, params), nil
}

func surveyListOrderColumn(sortBy string) string {
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "title":
		return "s.title"
	case "status":
		return "s.status"
	case "display_mode":
		return "s.display_mode"
	case "starts_at":
		return "s.starts_at"
	case "ends_at":
		return "s.ends_at"
	case "id":
		return "s.id"
	default:
		return "s.created_at"
	}
}

func (r *surveyRepository) ListActive(ctx context.Context, now time.Time) ([]service.Survey, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+surveySelectColumns+`
		FROM surveys s
		WHERE s.status = $1
		  AND (s.starts_at IS NULL OR s.starts_at <= $2)
		  AND (s.ends_at IS NULL OR s.ends_at > $2)
		ORDER BY s.id DESC
		LIMIT 50
	`, service.SurveyStatusActive, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.Survey, 0)
	for rows.Next() {
		item, err := scanSurveyRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *surveyRepository) GetUserStates(ctx context.Context, userID int64, surveyIDs []int64) (map[int64]service.SurveyUserState, error) {
	out := make(map[int64]service.SurveyUserState, len(surveyIDs))
	if len(surveyIDs) == 0 {
		return out, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT survey_id, user_id, status, snooze_until, updated_at
		FROM survey_user_states
		WHERE user_id = $1 AND survey_id = ANY($2)
	`, userID, pq.Array(surveyIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			st          service.SurveyUserState
			snoozeUntil sql.NullTime
		)
		if err := rows.Scan(&st.SurveyID, &st.UserID, &st.Status, &snoozeUntil, &st.UpdatedAt); err != nil {
			return nil, err
		}
		if snoozeUntil.Valid {
			t := snoozeUntil.Time
			st.SnoozeUntil = &t
		}
		out[st.SurveyID] = st
	}
	return out, rows.Err()
}

func (r *surveyRepository) UpsertUserState(ctx context.Context, surveyID, userID int64, status string, snoozeUntil *time.Time, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO survey_user_states (survey_id, user_id, status, snooze_until, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (survey_id, user_id)
		DO UPDATE SET
			status = CASE WHEN survey_user_states.status = 'submitted' THEN 'submitted' ELSE EXCLUDED.status END,
			snooze_until = CASE WHEN survey_user_states.status = 'submitted' THEN NULL ELSE EXCLUDED.snooze_until END,
			updated_at = EXCLUDED.updated_at
	`, surveyID, userID, status, snoozeUntil, now)
	return err
}

func (r *surveyRepository) HasUserSubmitted(ctx context.Context, surveyID, userID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM survey_responses WHERE survey_id = $1 AND user_id = $2)
	`, surveyID, userID).Scan(&exists)
	return exists, err
}

func (r *surveyRepository) HasRecentUsage(ctx context.Context, userID int64, since time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM usage_logs WHERE user_id = $1 AND created_at >= $2 LIMIT 1)
	`, userID, since).Scan(&exists)
	return exists, err
}

func (r *surveyRepository) SubmitAndReserveReward(ctx context.Context, record *service.SubmitSurveyRecord) (*service.SubmitSurveyRecordResult, error) {
	answersJSON, err := json.Marshal(record.Answers)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	initialRewardStatus := service.SurveyRewardStatusNone
	initialRewardType := service.SurveyRewardTypeNone
	initialRewardValue := 0.0
	if record.AttemptReward {
		initialRewardStatus = service.SurveyRewardStatusPending
		initialRewardType = record.RewardType
		initialRewardValue = record.RewardValue
	}

	resp := &service.SurveyResponse{
		SurveyID:        record.SurveyID,
		UserID:          record.UserID,
		Answers:         record.Answers,
		DurationSeconds: record.DurationSeconds,
		RewardStatus:    initialRewardStatus,
		RewardType:      initialRewardType,
		RewardValue:     initialRewardValue,
		SubmittedAt:     record.SubmittedAt,
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO survey_responses (
			survey_id, user_id, answers, duration_seconds,
			reward_status, reward_type, reward_value, reward_redeem_code, submitted_at
		) VALUES (
			$1, $2, $3::jsonb, $4,
			$5, $6, $7, '', $8
		)
		RETURNING id, submitted_at
	`,
		record.SurveyID, record.UserID, string(answersJSON), record.DurationSeconds,
		initialRewardStatus, initialRewardType, initialRewardValue, record.SubmittedAt,
	).Scan(&resp.ID, &resp.SubmittedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, service.ErrSurveyAlreadySubmitted
		}
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO survey_user_states (survey_id, user_id, status, snooze_until, updated_at)
		VALUES ($1, $2, $3, NULL, $4)
		ON CONFLICT (survey_id, user_id)
		DO UPDATE SET status = $3, snooze_until = NULL, updated_at = $4
	`, record.SurveyID, record.UserID, service.SurveyUserStateSubmitted, record.SubmittedAt); err != nil {
		return nil, err
	}

	rewardReserved := false
	if record.AttemptReward {
		res, err := tx.ExecContext(ctx, `
			UPDATE surveys
			SET reward_issued_count = reward_issued_count + 1
			WHERE id = $1
			  AND (reward_budget_limit IS NULL OR reward_issued_count < reward_budget_limit)
		`, record.SurveyID)
		if err != nil {
			return nil, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if affected > 0 {
			rewardReserved = true
		} else {
			resp.RewardStatus = service.SurveyRewardStatusBudgetExhausted
			resp.RewardValue = 0
			if _, err := tx.ExecContext(ctx, `
				UPDATE survey_responses
				SET reward_status = $1, reward_value = 0
				WHERE id = $2
			`, service.SurveyRewardStatusBudgetExhausted, resp.ID); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &service.SubmitSurveyRecordResult{
		Response:       resp,
		RewardReserved: rewardReserved,
	}, nil
}

func (r *surveyRepository) UpdateResponseReward(ctx context.Context, responseID int64, rewardStatus string, rewardValue float64, redeemCode string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE survey_responses
		SET reward_status = $1, reward_value = $2, reward_redeem_code = $3
		WHERE id = $4
	`, rewardStatus, rewardValue, redeemCode, responseID)
	return err
}

func (r *surveyRepository) ReleaseRewardBudget(ctx context.Context, surveyID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE surveys
		SET reward_issued_count = GREATEST(reward_issued_count - 1, 0)
		WHERE id = $1
	`, surveyID)
	return err
}

func (r *surveyRepository) GetMicroOptionCounts(ctx context.Context, surveyID int64, questionID string) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT answers -> $2 ->> 'choice_id' AS opt_id, COUNT(*)
		FROM survey_responses
		WHERE survey_id = $1 AND COALESCE(answers -> $2 ->> 'choice_id', '') <> ''
		GROUP BY 1
	`, surveyID, questionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	counts := make(map[string]int64)
	for rows.Next() {
		var optID string
		var cnt int64
		if err := rows.Scan(&optID, &cnt); err != nil {
			return nil, err
		}
		counts[optID] = cnt
	}
	return counts, rows.Err()
}

func (r *surveyRepository) ListAllResponsesForStats(ctx context.Context, surveyID int64) ([]service.SurveyResponse, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			sr.id, sr.survey_id, sr.user_id,
			COALESCE(u.email, ''), COALESCE(u.username, ''),
			sr.answers, sr.duration_seconds,
			sr.reward_status, sr.reward_type, sr.reward_value, sr.reward_redeem_code, sr.submitted_at
		FROM survey_responses sr
		LEFT JOIN users u ON u.id = sr.user_id
		WHERE sr.survey_id = $1
		ORDER BY sr.submitted_at DESC
	`, surveyID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.SurveyResponse, 0)
	for rows.Next() {
		var (
			item       service.SurveyResponse
			answersRaw []byte
		)
		if err := rows.Scan(
			&item.ID, &item.SurveyID, &item.UserID,
			&item.UserEmail, &item.Username,
			&answersRaw, &item.DurationSeconds,
			&item.RewardStatus, &item.RewardType, &item.RewardValue, &item.RewardRedeemCode, &item.SubmittedAt,
		); err != nil {
			return nil, err
		}
		if len(answersRaw) > 0 {
			if err := json.Unmarshal(answersRaw, &item.Answers); err != nil {
				return nil, fmt.Errorf("unmarshal response answers: %w", err)
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *surveyRepository) CountUserStatesByStatus(ctx context.Context, surveyID int64) (snoozed int64, dismissed int64, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'snoozed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'dismissed' THEN 1 ELSE 0 END), 0)
		FROM survey_user_states
		WHERE survey_id = $1
	`, surveyID).Scan(&snoozed, &dismissed)
	return snoozed, dismissed, err
}

func (r *surveyRepository) ListResponses(
	ctx context.Context,
	surveyID int64,
	params pagination.PaginationParams,
	search string,
) ([]service.SurveyResponse, *pagination.PaginationResult, error) {
	args := []any{surveyID}
	whereSQL := "WHERE sr.survey_id = $1"
	if search != "" {
		args = append(args, "%"+search+"%")
		whereSQL += fmt.Sprintf(" AND (u.email ILIKE $%d OR u.username ILIKE $%d OR sr.answers::text ILIKE $%d)", len(args), len(args), len(args))
	}

	var total int64
	countQuery := `
		SELECT COUNT(*)
		FROM survey_responses sr
		LEFT JOIN users u ON u.id = sr.user_id
	` + whereSQL
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, err
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	listArgs := append(args, params.Limit(), params.Offset())

	query := fmt.Sprintf(`
		SELECT
			sr.id, sr.survey_id, sr.user_id,
			COALESCE(u.email, ''), COALESCE(u.username, ''),
			sr.answers, sr.duration_seconds,
			sr.reward_status, sr.reward_type, sr.reward_value, sr.reward_redeem_code, sr.submitted_at
		FROM survey_responses sr
		LEFT JOIN users u ON u.id = sr.user_id
		%s
		ORDER BY sr.submitted_at DESC, sr.id DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, limitIdx, offsetIdx)

	rows, err := r.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.SurveyResponse, 0)
	for rows.Next() {
		var (
			item       service.SurveyResponse
			answersRaw []byte
		)
		if err := rows.Scan(
			&item.ID, &item.SurveyID, &item.UserID,
			&item.UserEmail, &item.Username,
			&answersRaw, &item.DurationSeconds,
			&item.RewardStatus, &item.RewardType, &item.RewardValue, &item.RewardRedeemCode, &item.SubmittedAt,
		); err != nil {
			return nil, nil, err
		}
		if len(answersRaw) > 0 {
			if err := json.Unmarshal(answersRaw, &item.Answers); err != nil {
				return nil, nil, fmt.Errorf("unmarshal response answers: %w", err)
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return items, paginationResultFromTotal(total, params), nil
}
