package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/securitysecret"
	"github.com/Wei-Shaw/sub2api/internal/config"
)

func initializeMigrationsForStartup(ctx context.Context, db *sql.DB, fsys fs.FS, standby bool) error {
	if standby {
		return verifyStandbyMigrationsFS(ctx, db, fsys)
	}
	return applyMigrationsFS(ctx, db, fsys)
}

// verifyStandbyMigrationsFS deliberately performs SELECTs only: a standby must
// fail closed instead of preparing a schema while the old writer is serving.
// Keep file selection, TrimSpace hashing and historical checksum compatibility
// identical to applyMigrationsFS. Extra historical ledger rows are tolerated by
// that runner too; they do not replace any required embedded migration.
func verifyStandbyMigrationsFS(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	if db == nil {
		return errors.New("nil sql db")
	}
	if fsys == nil {
		return errors.New("nil migrations filesystem")
	}
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return fmt.Errorf("list standby migrations: %w", err)
	}
	sort.Strings(files)
	checksums := make(map[string]string, len(files))
	for _, name := range files {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("read standby migration %s: %w", name, err)
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}
		sum := sha256.Sum256([]byte(content))
		checksums[name] = hex.EncodeToString(sum[:])
	}
	if len(checksums) == 0 {
		return errors.New("standby requires a non-empty embedded migration set")
	}
	rows, err := db.QueryContext(ctx, "SELECT filename, checksum FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("read standby migration ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()
	applied := make(map[string]string)
	for rows.Next() {
		var name, checksum string
		if err := rows.Scan(&name, &checksum); err != nil {
			return fmt.Errorf("scan standby migration ledger: %w", err)
		}
		applied[name] = checksum
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read standby migration ledger: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close standby migration ledger: %w", err)
	}
	for _, name := range files {
		checksum, required := checksums[name]
		if !required {
			continue
		}
		existing, exists := applied[name]
		if !exists {
			return fmt.Errorf("standby migration %s is not applied; run migrations before rolling handoff", name)
		}
		if existing != checksum && !isMigrationChecksumCompatible(name, existing, checksum) {
			return fmt.Errorf("standby migration %s checksum mismatch", name)
		}
	}
	// The normal runner creates/seeds this table if absent or empty. A standby
	// may only verify that this preparation already happened.
	var baselineCount int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM atlas_schema_revisions").Scan(&baselineCount); err != nil {
		return fmt.Errorf("read standby atlas baseline: %w", err)
	}
	if baselineCount == 0 {
		return errors.New("standby atlas baseline is missing; prepare database before rolling handoff")
	}
	return nil
}

func initializeBootstrapSecretsForStartup(ctx context.Context, client *ent.Client, cfg *config.Config, standby bool) error {
	if !standby {
		return ensureBootstrapSecrets(ctx, client, cfg)
	}
	if cfg == nil {
		return errors.New("nil config")
	}
	// An explicitly configured key is authoritative for this release. Only fill
	// missing configuration from the database; never persist or generate a key.
	cfg.JWT.Secret = strings.TrimSpace(cfg.JWT.Secret)
	if cfg.JWT.Secret != "" {
		if len([]byte(cfg.JWT.Secret)) < 32 {
			return errors.New("configured jwt secret must be at least 32 bytes")
		}
		return nil
	}
	if client == nil {
		return errors.New("nil ent client")
	}
	stored, err := client.SecuritySecret.Query().Where(securitysecret.KeyEQ(securitySecretKeyJWT)).Only(ctx)
	if ent.IsNotFound(err) {
		return errors.New("standby requires an existing persisted jwt secret")
	}
	if err != nil {
		return fmt.Errorf("read standby jwt secret: %w", err)
	}
	value := strings.TrimSpace(stored.Value)
	if len([]byte(value)) < 32 {
		return errors.New("stored jwt secret must be at least 32 bytes")
	}
	cfg.JWT.Secret = value
	return nil
}

func initializeSimpleModeForStartup(ctx context.Context, client *ent.Client, cfg *config.Config, standby bool) error {
	if !standby {
		return ensureSimpleModeStartup(ctx, client, cfg)
	}
	if cfg == nil {
		return errors.New("nil config")
	}
	// Both group seeding and the admin-concurrency settings migration belong to
	// the writer. A standby serves the existing state without mutating either.
	return nil
}
