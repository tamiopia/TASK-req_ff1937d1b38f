package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eagle-point/service-portal/internal/config"
)

// setBaseEnv sets the DB env vars every test needs and unsets the rest so
// cases start from a clean slate.
func setBaseEnv(t *testing.T, appEnv string) {
	t.Helper()
	os.Setenv("APP_ENV", appEnv)
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Unsetenv("FIELD_ENCRYPTION_KEY")
	t.Cleanup(func() {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("FIELD_ENCRYPTION_KEY")
	})
}

func TestLoad_RequiredFields(t *testing.T) {
	// Clear env
	os.Unsetenv("DB_NAME")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("FIELD_ENCRYPTION_KEY")
	os.Setenv("APP_ENV", "test")
	t.Cleanup(func() { os.Unsetenv("APP_ENV") })

	_, err := config.Load()
	// In test mode, encryption key is not required but DB fields are
	assert.Error(t, err)
}

func TestLoad_ValidConfig(t *testing.T) {
	setBaseEnv(t, "test")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "testdb", cfg.DBName)
	assert.Equal(t, "testuser", cfg.DBUser)
}

func TestConfig_DSN(t *testing.T) {
	setBaseEnv(t, "test")
	os.Setenv("DB_NAME", "mydb")
	os.Setenv("DB_USER", "user")
	os.Setenv("DB_PASSWORD", "pass")

	cfg, err := config.Load()
	require.NoError(t, err)

	dsn := cfg.DSN()
	assert.Contains(t, dsn, "user:pass@tcp(")
	assert.Contains(t, dsn, "/mydb")
}

// ─── FIELD_ENCRYPTION_KEY hardening ──────────────────────────────────────────

const (
	placeholderKey = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	zerosKey       = "0000000000000000000000000000000000000000000000000000000000000000"
	validRealKey   = "a1b2c3d4e5f67890a1b2c3d4e5f67890a1b2c3d4e5f67890a1b2c3d4e5f67890"
)

func TestLoad_Production_RequiresKey(t *testing.T) {
	setBaseEnv(t, "production")
	// No FIELD_ENCRYPTION_KEY set

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FIELD_ENCRYPTION_KEY is required")
}

func TestLoad_Production_RejectsEnvExamplePlaceholder(t *testing.T) {
	setBaseEnv(t, "production")
	os.Setenv("FIELD_ENCRYPTION_KEY", placeholderKey)

	_, err := config.Load()
	require.Error(t, err, "production must reject the .env.example placeholder")
	assert.Contains(t, err.Error(), "placeholder")
	assert.Contains(t, err.Error(), "refusing to start in production")
}

func TestLoad_Production_RejectsZeroKey(t *testing.T) {
	setBaseEnv(t, "production")
	os.Setenv("FIELD_ENCRYPTION_KEY", zerosKey)

	_, err := config.Load()
	require.Error(t, err, "production must reject the all-zero dev key")
	assert.Contains(t, strings.ToLower(err.Error()), "refusing to start in production")
}

func TestLoad_Production_RejectsPlaceholderCaseInsensitive(t *testing.T) {
	setBaseEnv(t, "production")
	// Same bytes, uppercased — hex is case-insensitive so this is the same key.
	os.Setenv("FIELD_ENCRYPTION_KEY", strings.ToUpper(placeholderKey))

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refusing to start in production")
}

func TestLoad_Production_RejectsNonHexKey(t *testing.T) {
	setBaseEnv(t, "production")
	os.Setenv("FIELD_ENCRYPTION_KEY", "not-hex-at-all-not-hex-at-all-not-hex-at-all-not-hex-at-all-1234")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hex-encoded")
}

func TestLoad_Production_RejectsWrongLengthKey(t *testing.T) {
	setBaseEnv(t, "production")
	// 16-byte key (AES-128) — not allowed, we require AES-256.
	os.Setenv("FIELD_ENCRYPTION_KEY", "a1b2c3d4e5f67890a1b2c3d4e5f67890")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestLoad_Production_AcceptsFreshKey(t *testing.T) {
	setBaseEnv(t, "production")
	os.Setenv("FIELD_ENCRYPTION_KEY", validRealKey)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, validRealKey, cfg.FieldEncryptionKey)
}

func TestLoad_Development_AcceptsPlaceholderWithWarning(t *testing.T) {
	setBaseEnv(t, "development")
	os.Setenv("FIELD_ENCRYPTION_KEY", placeholderKey)

	// Dev tolerates the placeholder (so `docker-compose up` still works out of
	// the box) but the validator emits a log warning. We only assert no error.
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, placeholderKey, cfg.FieldEncryptionKey)
}

func TestLoad_Test_AllowsEmptyKey(t *testing.T) {
	setBaseEnv(t, "test")
	// FIELD_ENCRYPTION_KEY intentionally unset

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.FieldEncryptionKey)
}
