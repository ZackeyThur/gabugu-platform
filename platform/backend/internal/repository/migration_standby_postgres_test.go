package repository

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/backgroundgate"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/lib/pq"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestStandbyPostgresReadOnly is opt-in and restricted to the dedicated local
// verification database. It must never inherit a production DATABASE_URL.
func TestStandbyPostgresReadOnly(t *testing.T) {
	dsn := os.Getenv("GABUGU_STANDBY_VERIFY_DSN")
	if dsn == "" {
		t.Skip("set GABUGU_STANDBY_VERIFY_DSN for the dedicated local PostgreSQL verification database")
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", u.Hostname(), "verification database must be local")
	require.Equal(t, "15433", u.Port(), "verification database must use its isolated port")
	require.Equal(t, "/rolling", u.Path)
	require.Equal(t, "rolling", u.User.Username())
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	admin, err := db.Conn(ctx)
	require.NoError(t, err)
	defer func() { _ = admin.Close() }()

	// The restored schema and ledger contain no production data. Preparation
	// exercises the unchanged writer runner and creates only the Atlas baseline.
	require.NoError(t, initializeMigrationsForStartup(ctx, db, migrations.FS, false))
	_, err = admin.ExecContext(ctx, `DELETE FROM security_secrets WHERE key = $1`, securitySecretKeyJWT)
	require.NoError(t, err)
	_, err = admin.ExecContext(ctx, `ALTER ROLE `+pq.QuoteIdentifier(u.User.Username())+` SET default_transaction_read_only = on`)
	require.NoError(t, err)
	defer func() {
		_, err := admin.ExecContext(context.Background(), `ALTER ROLE `+pq.QuoteIdentifier(u.User.Username())+` RESET default_transaction_read_only`)
		require.NoError(t, err)
	}()

	viper.Reset()
	t.Cleanup(viper.Reset)
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("{}\n"), 0600))
	t.Setenv("CONFIG_FILE", configPath)
	t.Setenv("JWT_SECRET", strings.Repeat("j", 32))
	t.Setenv("TOTP_ENCRYPTION_KEY", strings.Repeat("a", 64))
	t.Setenv(backgroundgate.ActivationFileEnv, filepath.Join(t.TempDir(), "activate.json"))
	t.Setenv(backgroundgate.ReleaseIDEnv, "standby-postgres-verification")
	require.NoError(t, backgroundgate.InitializeFromEnv())
	defer func() {
		backgroundgate.Close()
		t.Setenv(backgroundgate.ActivationFileEnv, "")
		t.Setenv(backgroundgate.ReleaseIDEnv, "")
		require.NoError(t, backgroundgate.InitializeFromEnv())
	}()
	cfg, err := config.LoadForBootstrap()
	require.NoError(t, err)
	password, _ := u.User.Password()
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)
	cfg.Database.Host = u.Hostname()
	cfg.Database.Port = port
	cfg.Database.User = u.User.Username()
	cfg.Database.Password = password
	cfg.Database.DBName = strings.TrimPrefix(u.Path, "/")
	cfg.Database.SSLMode = "disable"
	cfg.RunMode = config.RunModeSimple
	cfg.SimpleMode.AutoCreateDefaultGroups = true

	// A successful public startup in a server-enforced read-only session proves
	// that no schema, seed, settings migration or secret persistence is attempted.
	client, standbyDB, err := InitEnt(cfg)
	require.NoError(t, err)
	var readOnly string
	require.NoError(t, standbyDB.QueryRowContext(ctx, `SHOW transaction_read_only`).Scan(&readOnly))
	require.Equal(t, "on", readOnly)
	require.Equal(t, strings.Repeat("j", 32), cfg.JWT.Secret)
	var secretCount, groupCount, settingCount int
	require.NoError(t, standbyDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM security_secrets`).Scan(&secretCount))
	require.NoError(t, standbyDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM groups`).Scan(&groupCount))
	require.NoError(t, standbyDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM settings`).Scan(&settingCount))
	require.Zero(t, secretCount)
	require.Zero(t, groupCount)
	require.Zero(t, settingCount)
	require.NoError(t, client.Close())
	t.Log("InitEnt succeeded with transaction_read_only=on; security_secrets/groups/settings remain empty")

	// The pinned preparation connection remains writable. Seed a synthetic key
	// and prove a missing configuration can be filled by SELECT alone.
	_, err = admin.ExecContext(ctx, `INSERT INTO security_secrets (key, value, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())`, securitySecretKeyJWT, strings.Repeat("s", 32))
	require.NoError(t, err)
	cfg.JWT.Secret = ""
	client, _, err = InitEnt(cfg)
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("s", 32), cfg.JWT.Secret)
	require.NoError(t, client.Close())
	_, err = admin.ExecContext(ctx, `DELETE FROM security_secrets WHERE key = $1`, securitySecretKeyJWT)
	require.NoError(t, err)
	cfg.JWT.Secret = ""
	_, _, err = InitEnt(cfg)
	require.ErrorContains(t, err, "existing persisted jwt secret")

	// Independent temporary schemas keep negative ledger cases separate from
	// the complete restored ledger and verify that they cannot self-repair.
	for _, tt := range []struct{ name, ledgerDDL, want string }{
		{"missing ledger", "", "read standby migration ledger"},
		{"missing migration", `CREATE TABLE schema_migrations (filename text PRIMARY KEY, checksum text NOT NULL)`, "is not applied"},
		{"checksum mismatch", `CREATE TABLE schema_migrations (filename text PRIMARY KEY, checksum text NOT NULL); INSERT INTO schema_migrations VALUES ('001_test.sql', 'wrong')`, "checksum mismatch"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			schema := "standby_" + strings.ReplaceAll(tt.name, " ", "_")
			_, err := admin.ExecContext(ctx, `CREATE SCHEMA `+pq.QuoteIdentifier(schema))
			require.NoError(t, err)
			defer func() {
				_, err := admin.ExecContext(context.Background(), `DROP SCHEMA `+pq.QuoteIdentifier(schema)+` CASCADE`)
				require.NoError(t, err)
			}()
			if tt.ledgerDDL != "" {
				_, err = admin.ExecContext(ctx, `SET search_path = `+pq.QuoteIdentifier(schema)+`; `+tt.ledgerDDL+`; RESET search_path`)
				require.NoError(t, err)
			}
			isolatedURL := *u
			query := isolatedURL.Query()
			query.Set("search_path", schema)
			isolatedURL.RawQuery = query.Encode()
			readDB, err := sql.Open("postgres", isolatedURL.String())
			require.NoError(t, err)
			defer func() { _ = readDB.Close() }()
			err = verifyStandbyMigrationsFS(ctx, readDB, fstest.MapFS{"001_test.sql": {Data: []byte("CREATE TABLE must_not_exist (id int);")}})
			require.ErrorContains(t, err, tt.want)
			var created bool
			require.NoError(t, admin.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = 'must_not_exist')`, schema).Scan(&created))
			require.False(t, created)
		})
	}

	// A direct missing-key read under the same read-only role stays empty.
	readDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = readDB.Close() }()
	readClient := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, readDB)))
	defer func() { _ = readClient.Close() }()
	require.ErrorContains(t, initializeBootstrapSecretsForStartup(ctx, readClient, &config.Config{}, true), "existing persisted jwt secret")
}
