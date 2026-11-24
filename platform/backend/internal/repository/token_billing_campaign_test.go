//go:build unit

package repository

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type tokenCampaignPreparedScanner []any

func (values tokenCampaignPreparedScanner) Scan(dest ...any) error {
	if len(dest) != len(values) {
		return fmt.Errorf("unexpected scan count %d/%d", len(dest), len(values))
	}
	for i, value := range values {
		if scanner, ok := dest[i].(sql.Scanner); ok {
			if v := reflect.ValueOf(value); v.IsValid() && v.Kind() == reflect.Pointer {
				if v.IsNil() {
					value = nil
				} else {
					value = v.Elem().Interface()
				}
			}
			if valuer, ok := value.(driver.Valuer); ok {
				var err error
				value, err = valuer.Value()
				if err != nil {
					return err
				}
			}
			if err := scanner.Scan(value); err != nil {
				return err
			}
			continue
		}
		v := reflect.ValueOf(value)
		target := reflect.ValueOf(dest[i]).Elem()
		if !v.IsValid() || !v.Type().ConvertibleTo(target.Type()) {
			return fmt.Errorf("unexpected scan type at %d", i)
		}
		target.Set(v.Convert(target.Type()))
	}
	return nil
}

func TestTokenCampaignUsagePersistence(t *testing.T) {
	at := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	snapshot := &service.TokenCampaignUsage{ID: "blg-20261115", Revision: 1, Phase: "championship-day", Discount: .2, PricingAt: at, OriginalActualCost: .37, DiscountAmount: .296}
	log := &service.UsageLog{UserID: 1, APIKeyID: 2, AccountID: 3, RequestID: "campaign-persist", Model: "gpt-5.1", ActualCost: .074, TotalCost: 1, CreatedAt: at, TokenCampaign: snapshot}
	prepared := prepareUsageLogInsert(log)
	require.Len(t, prepared.args, len(usageLogInsertArgTypes))
	require.Equal(t, "jsonb", usageLogInsertArgTypes[len(usageLogInsertArgTypes)-1])
	var decoded service.TokenCampaignUsage
	require.NoError(t, json.Unmarshal([]byte(prepared.args[len(prepared.args)-1].(string)), &decoded))
	require.Equal(t, *snapshot, decoded)
	// Build a database-style row from the same complete insert sequence. This
	// exercises scan alignment, nullable historical rows, and persisted snapshots.
	values := append([]any{int64(9)}, prepared.args...)
	values[4] = sql.NullString{String: log.RequestID, Valid: true}
	values[30] = int16(log.BillingType)
	values[len(values)-1] = sql.NullString{String: prepared.args[len(prepared.args)-1].(string), Valid: true}
	got, err := scanUsageLog(tokenCampaignPreparedScanner(values))
	require.NoError(t, err)
	require.Equal(t, snapshot, got.TokenCampaign)
	require.Equal(t, .074, got.ActualCost)
	require.Equal(t, 1.0, got.TotalCost)
	values[len(values)-1] = sql.NullString{}
	got, err = scanUsageLog(tokenCampaignPreparedScanner(values))
	require.NoError(t, err)
	require.Nil(t, got.TokenCampaign)
	query, args := buildUsageLogBestEffortInsertQuery([]usageLogInsertPrepared{prepared})
	require.Contains(t, query, "token_campaign")
	require.Len(t, args, len(prepared.args))
	require.Equal(t, len(strings.Split(usageLogSelectColumns, ",")), len(prepared.args)+1)
}
