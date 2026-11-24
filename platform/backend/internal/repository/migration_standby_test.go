package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func newStandbyTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func standbyMigrationChecksum(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:])
}

func TestStandbyMigrationsOnlySelects(t *testing.T) {
	db, mock := newStandbyTestDB(t)
	fsys := fstest.MapFS{
		"002_second_notx.sql": {Data: []byte("CREATE INDEX CONCURRENTLY idx ON test (id);")},
		"001_first.sql":       {Data: []byte(" \nCREATE TABLE test (id int);\n ")},
		"003_empty.sql":       {Data: []byte(" \n\t ")},
		"ignored.txt":         {Data: []byte("not a migration")},
	}
	rows := sqlmock.NewRows([]string{"filename", "checksum"}).
		AddRow("older_historical.sql", "tolerated").
		AddRow("002_second_notx.sql", standbyMigrationChecksum(string(fsys["002_second_notx.sql"].Data))).
		AddRow("001_first.sql", standbyMigrationChecksum(string(fsys["001_first.sql"].Data)))
	mock.ExpectQuery(`^SELECT filename, checksum FROM schema_migrations$`).WillReturnRows(rows).RowsWillBeClosed()
	mock.ExpectQuery(`^SELECT COUNT\(\*\) FROM atlas_schema_revisions$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1)).RowsWillBeClosed()
	// The migration body is intentionally mutating. The standby must only read
	// the ledger: no lock, transaction, DDL, DML or ledger repair is expected.
	require.NoError(t, initializeMigrationsForStartup(context.Background(), db, fsys, true))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStandbyMigrationsFailClosed(t *testing.T) {
	const content = "CREATE TABLE test (id int);"
	fsys := fstest.MapFS{"001_first.sql": {Data: []byte(content)}}
	for _, tt := range []struct {
		name      string
		rows      *sqlmock.Rows
		ledgerErr error
		atlasRows *sqlmock.Rows
		atlasErr  error
		want      string
	}{
		{name: "missing ledger", ledgerErr: errors.New("relation schema_migrations does not exist"), want: "read standby migration ledger"},
		{name: "missing migration", rows: sqlmock.NewRows([]string{"filename", "checksum"}), want: "001_first.sql is not applied"},
		{name: "checksum mismatch", rows: sqlmock.NewRows([]string{"filename", "checksum"}).AddRow("001_first.sql", "wrong"), want: "001_first.sql checksum mismatch"},
		{name: "scan failure", rows: sqlmock.NewRows([]string{"filename", "checksum"}).AddRow("001_first.sql", nil), want: "scan standby migration ledger"},
		{name: "row failure", rows: sqlmock.NewRows([]string{"filename", "checksum"}).AddRow("001_first.sql", standbyMigrationChecksum(content)).RowError(0, errors.New("read interrupted")), want: "read standby migration ledger"},
		{name: "missing atlas table", atlasErr: errors.New("relation atlas_schema_revisions does not exist"), want: "read standby atlas baseline"},
		{name: "empty atlas baseline", atlasRows: sqlmock.NewRows([]string{"count"}).AddRow(0), want: "standby atlas baseline is missing"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newStandbyTestDB(t)
			query := mock.ExpectQuery(`^SELECT filename, checksum FROM schema_migrations$`)
			if tt.ledgerErr != nil {
				query.WillReturnError(tt.ledgerErr)
			} else {
				rows := tt.rows
				if rows == nil {
					rows = sqlmock.NewRows([]string{"filename", "checksum"}).AddRow("001_first.sql", standbyMigrationChecksum(content))
				}
				query.WillReturnRows(rows).RowsWillBeClosed()
			}
			if tt.atlasErr != nil || tt.atlasRows != nil {
				query := mock.ExpectQuery(`^SELECT COUNT\(\*\) FROM atlas_schema_revisions$`)
				if tt.atlasErr != nil {
					query.WillReturnError(tt.atlasErr)
				} else {
					query.WillReturnRows(tt.atlasRows).RowsWillBeClosed()
				}
			}
			err := verifyStandbyMigrationsFS(context.Background(), db, fsys)
			require.ErrorContains(t, err, tt.want)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStandbyMigrationsUsesExistingChecksumCompatibility(t *testing.T) {
	const name = "054_drop_legacy_cache_columns.sql"
	content, err := migrations.FS.ReadFile(name)
	require.NoError(t, err)
	db, mock := newStandbyTestDB(t)
	mock.ExpectQuery(`^SELECT filename, checksum FROM schema_migrations$`).
		WillReturnRows(sqlmock.NewRows([]string{"filename", "checksum"}).
			AddRow(name, "182c193f3359946cf094090cd9e57d5c3fd9abaffbc1e8fc378646b8a6fa12b4"))
	mock.ExpectQuery(`^SELECT COUNT\(\*\) FROM atlas_schema_revisions$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	require.NoError(t, verifyStandbyMigrationsFS(context.Background(), db, fstest.MapFS{name: {Data: content}}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStandbyMigrationsRejectsMissingInputs(t *testing.T) {
	require.ErrorContains(t, verifyStandbyMigrationsFS(context.Background(), nil, fstest.MapFS{}), "nil sql db")
	db, mock := newStandbyTestDB(t)
	require.ErrorContains(t, verifyStandbyMigrationsFS(context.Background(), db, nil), "nil migrations filesystem")
	require.ErrorContains(t, verifyStandbyMigrationsFS(context.Background(), db, fstest.MapFS{}), "non-empty embedded migration set")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStandbyBootstrapKeepsConfiguredSecretWithoutDatabase(t *testing.T) {
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "  " + strings.Repeat("x", 32) + "  "}}
	require.NoError(t, initializeBootstrapSecretsForStartup(context.Background(), nil, cfg, true))
	require.Equal(t, strings.Repeat("x", 32), cfg.JWT.Secret)
	// Even an unavailable database is irrelevant to a supplied valid secret.
	db, mock := newStandbyTestDB(t)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, initializeBootstrapSecretsForStartup(context.Background(), client, cfg, true))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStandbyBootstrapOnlyLoadsMissingSecret(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value string
		found bool
		want  string
	}{
		{name: "existing", value: "  " + strings.Repeat("s", 32) + "  ", found: true},
		{name: "missing", want: "existing persisted jwt secret"},
		{name: "invalid stored", value: "short", found: true, want: "stored jwt secret must be at least 32 bytes"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newStandbyTestDB(t)
			client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			t.Cleanup(func() { _ = client.Close() })
			rows := sqlmock.NewRows([]string{"key", "value"})
			if tt.found {
				rows.AddRow(securitySecretKeyJWT, tt.value)
			}
			mock.ExpectQuery(`^SELECT .* FROM "security_secrets" WHERE "security_secrets"\."key" = \$1 LIMIT 2$`).
				WithArgs(securitySecretKeyJWT).WillReturnRows(rows).RowsWillBeClosed()
			cfg := &config.Config{}
			err := initializeBootstrapSecretsForStartup(context.Background(), client, cfg, true)
			if tt.want == "" {
				require.NoError(t, err)
				require.Equal(t, strings.TrimSpace(tt.value), cfg.JWT.Secret)
			} else {
				require.ErrorContains(t, err, tt.want)
				require.Empty(t, cfg.JWT.Secret)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStandbyBootstrapRejectsInvalidInputs(t *testing.T) {
	require.ErrorContains(t, initializeBootstrapSecretsForStartup(context.Background(), nil, nil, true), "nil config")
	require.ErrorContains(t, initializeBootstrapSecretsForStartup(context.Background(), nil, &config.Config{}, true), "nil ent client")
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "short"}}
	require.ErrorContains(t, initializeBootstrapSecretsForStartup(context.Background(), nil, cfg, true), "configured jwt secret must be at least 32 bytes")
}

func TestStandbySimpleModeSkipsSeedingAndSettingsMigration(t *testing.T) {
	for _, mode := range []string{config.RunModeSimple, config.RunModeStandard} {
		for _, seed := range []bool{true, false} {
			db, mock := newStandbyTestDB(t)
			client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			t.Cleanup(func() { _ = client.Close() })
			cfg := &config.Config{RunMode: mode, SimpleMode: config.SimpleModeConfig{AutoCreateDefaultGroups: seed}}
			require.NoError(t, initializeSimpleModeForStartup(context.Background(), client, cfg, true))
			require.NoError(t, mock.ExpectationsWereMet())
		}
	}
	require.ErrorContains(t, initializeSimpleModeForStartup(context.Background(), nil, nil, true), "nil config")
}

func TestUngatedStartupKeepsOriginalInitialization(t *testing.T) {
	t.Run("migration runner", func(t *testing.T) {
		db, mock := newStandbyTestDB(t)
		want := errors.New("normal runner lock reached")
		mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).WithArgs(migrationsAdvisoryLockID).WillReturnError(want)
		require.ErrorIs(t, initializeMigrationsForStartup(context.Background(), db, fstest.MapFS{}, false), want)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("secret persistence", func(t *testing.T) {
		db, mock := newStandbyTestDB(t)
		client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
		t.Cleanup(func() { _ = client.Close() })
		want := errors.New("normal secret persistence reached")
		mock.ExpectQuery(`INSERT INTO "security_secrets"`).WillReturnError(want)
		cfg := &config.Config{JWT: config.JWTConfig{Secret: strings.Repeat("x", 32)}}
		require.ErrorIs(t, initializeBootstrapSecretsForStartup(context.Background(), client, cfg, false), want)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("simple seeding", func(t *testing.T) {
		db, mock := newStandbyTestDB(t)
		client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
		t.Cleanup(func() { _ = client.Close() })
		want := errors.New("normal group seeding reached")
		mock.ExpectExec(`UPDATE "groups"`).WillReturnError(want)
		cfg := &config.Config{RunMode: config.RunModeSimple, SimpleMode: config.SimpleModeConfig{AutoCreateDefaultGroups: true}}
		require.ErrorIs(t, initializeSimpleModeForStartup(context.Background(), client, cfg, false), want)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
