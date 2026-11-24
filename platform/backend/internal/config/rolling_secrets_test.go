package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRollingRequiresInstalledTOTPKey(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("GABUGU_BACKGROUND_ACTIVATION_FILE", filepath.Join(t.TempDir(), "activate.json"))
	t.Setenv("GABUGU_RELEASE_ID", "test-rolling")
	t.Setenv("TOTP_ENCRYPTION_KEY", "")
	cfg, err := LoadForBootstrap()
	require.Nil(t, cfg)
	require.ErrorContains(t, err, "totp.encryption_key must be configured")
}

func TestRollingPreservesInstalledTOTPKey(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("GABUGU_BACKGROUND_ACTIVATION_FILE", filepath.Join(t.TempDir(), "activate.json"))
	t.Setenv("GABUGU_RELEASE_ID", "test-rolling")
	key := strings.Repeat("a1", 32)
	t.Setenv("TOTP_ENCRYPTION_KEY", key)
	cfg, err := LoadForBootstrap()
	require.NoError(t, err)
	require.Equal(t, key, cfg.Totp.EncryptionKey)
	require.True(t, cfg.Totp.EncryptionKeyConfigured)
}

func TestUngatedRetainsDevelopmentTOTPGeneration(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("GABUGU_BACKGROUND_ACTIVATION_FILE", "")
	t.Setenv("GABUGU_RELEASE_ID", "")
	t.Setenv("TOTP_ENCRYPTION_KEY", "")
	cfg, err := LoadForBootstrap()
	require.NoError(t, err)
	require.NotEmpty(t, cfg.Totp.EncryptionKey)
	require.False(t, cfg.Totp.EncryptionKeyConfigured)
}
