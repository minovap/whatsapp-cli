package api

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"API_KEY", "PORT", "STORE_DIR", "MAX_MESSAGES",
		"MAX_HOURS", "PHONE_WHITELIST", "PHONE_BLACKLIST", "LOG_LEVEL",
	} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

func TestParseConfig_RequiresAPIKey(t *testing.T) {
	clearEnv(t)
	_, err := ParseConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API_KEY")
}

func TestParseConfig_Defaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("API_KEY", "test-key")

	cfg, err := ParseConfig()
	require.NoError(t, err)
	assert.Equal(t, "test-key", cfg.APIKey)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "/data/store", cfg.StoreDir)
	assert.Equal(t, 100, cfg.MaxMessages)
	assert.Equal(t, 48, cfg.MaxHours)
	assert.Empty(t, cfg.PhoneWhitelist)
	assert.Empty(t, cfg.PhoneBlacklist)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestParseConfig_AllEnvVars(t *testing.T) {
	clearEnv(t)
	t.Setenv("API_KEY", "my-secret")
	t.Setenv("PORT", "9090")
	t.Setenv("STORE_DIR", "/tmp/store")
	t.Setenv("MAX_MESSAGES", "50")
	t.Setenv("MAX_HOURS", "24")
	t.Setenv("PHONE_WHITELIST", "123456,789012")
	t.Setenv("PHONE_BLACKLIST", "111111")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := ParseConfig()
	require.NoError(t, err)
	assert.Equal(t, "my-secret", cfg.APIKey)
	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, "/tmp/store", cfg.StoreDir)
	assert.Equal(t, 50, cfg.MaxMessages)
	assert.Equal(t, 24, cfg.MaxHours)
	assert.Equal(t, []string{"123456", "789012"}, cfg.PhoneWhitelist)
	assert.Equal(t, []string{"111111"}, cfg.PhoneBlacklist)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestParseConfig_InvalidPort(t *testing.T) {
	clearEnv(t)
	t.Setenv("API_KEY", "test-key")
	t.Setenv("PORT", "not-a-number")

	_, err := ParseConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PORT")
}

func TestParseConfig_InvalidMaxMessages(t *testing.T) {
	clearEnv(t)
	t.Setenv("API_KEY", "test-key")
	t.Setenv("MAX_MESSAGES", "abc")

	_, err := ParseConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MAX_MESSAGES")
}

func TestParseConfig_InvalidMaxHours(t *testing.T) {
	clearEnv(t)
	t.Setenv("API_KEY", "test-key")
	t.Setenv("MAX_HOURS", "xyz")

	_, err := ParseConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MAX_HOURS")
}

func TestParseConfig_WhitelistTrimming(t *testing.T) {
	clearEnv(t)
	t.Setenv("API_KEY", "test-key")
	t.Setenv("PHONE_WHITELIST", " 123 , 456 , , 789 ")

	cfg, err := ParseConfig()
	require.NoError(t, err)
	assert.Equal(t, []string{"123", "456", "789"}, cfg.PhoneWhitelist)
}
