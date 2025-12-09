package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	require.NotNil(t, cfg)
	assert.Equal(t, "ndjson", cfg.Format)
	assert.Equal(t, "default", cfg.Level)
	assert.False(t, cfg.Quiet)
	assert.False(t, cfg.Verbose)
	assert.Equal(t, "booted", cfg.Defaults.Simulator)
	assert.Equal(t, 100, cfg.Defaults.BufferSize)
	assert.Equal(t, "5m", cfg.Defaults.Since)
	assert.Equal(t, 1000, cfg.Defaults.Limit)
}

func TestLoad(t *testing.T) {
	t.Run("returns defaults when no config file exists", func(t *testing.T) {
		// Create temp dir with no config
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(origDir)

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Should have default values
		assert.Equal(t, "ndjson", cfg.Format)
	})

	t.Run("loads config from file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file
		configContent := `
format: text
level: error
quiet: true
defaults:
  simulator: "iPhone 15"
  buffer_size: 500
`
		configPath := filepath.Join(tmpDir, "xcw.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadFromFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, "text", cfg.Format)
		assert.Equal(t, "error", cfg.Level)
		assert.True(t, cfg.Quiet)
		assert.Equal(t, "iPhone 15", cfg.Defaults.Simulator)
		assert.Equal(t, 500, cfg.Defaults.BufferSize)
	})
}

func TestLoadFromFile(t *testing.T) {
	t.Run("returns error for non-existent file", func(t *testing.T) {
		cfg, err := LoadFromFile("/nonexistent/path/config.yaml")
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "bad.yaml")
		err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
		require.NoError(t, err)

		cfg, err := LoadFromFile(configPath)
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("parses all config fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		configContent := `
format: ndjson
level: debug
quiet: false
verbose: true
defaults:
  simulator: booted
  app: com.test.app
  buffer_size: 200
  summary_interval: 30s
  heartbeat: 10s
  since: 10m
  limit: 500
  subsystems:
    - com.test.app
  categories:
    - network
  exclude_subsystems:
    - com.apple.*
  exclude_pattern: "heartbeat|keepalive"
`
		configPath := filepath.Join(tmpDir, "xcw.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadFromFile(configPath)
		require.NoError(t, err)

		assert.Equal(t, "ndjson", cfg.Format)
		assert.Equal(t, "debug", cfg.Level)
		assert.False(t, cfg.Quiet)
		assert.True(t, cfg.Verbose)
		assert.Equal(t, "booted", cfg.Defaults.Simulator)
		assert.Equal(t, "com.test.app", cfg.Defaults.App)
		assert.Equal(t, 200, cfg.Defaults.BufferSize)
		assert.Equal(t, "30s", cfg.Defaults.SummaryInterval)
		assert.Equal(t, "10s", cfg.Defaults.Heartbeat)
		assert.Equal(t, "10m", cfg.Defaults.Since)
		assert.Equal(t, 500, cfg.Defaults.Limit)
		assert.Contains(t, cfg.Defaults.Subsystems, "com.test.app")
		assert.Contains(t, cfg.Defaults.Categories, "network")
		assert.Contains(t, cfg.Defaults.ExcludeSubsystems, "com.apple.*")
		assert.Equal(t, "heartbeat|keepalive", cfg.Defaults.ExcludePattern)
	})
}

func TestConfigEnvironmentVariables(t *testing.T) {
	// Save original env
	origFormat := os.Getenv("XCW_FORMAT")
	origApp := os.Getenv("XCW_APP")
	defer func() {
		os.Setenv("XCW_FORMAT", origFormat)
		os.Setenv("XCW_APP", origApp)
	}()

	// Set env variables
	os.Setenv("XCW_FORMAT", "text")
	os.Setenv("XCW_APP", "com.env.app")

	// Load config (should pick up env vars)
	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "text", cfg.Format)
	assert.Equal(t, "com.env.app", cfg.Defaults.App)
}

func TestDefaultsConfig(t *testing.T) {
	defaults := DefaultsConfig{
		Simulator:         "booted",
		App:               "com.test",
		BufferSize:        100,
		SummaryInterval:   "30s",
		Heartbeat:         "10s",
		Subsystems:        []string{"sub1", "sub2"},
		Categories:        []string{"cat1"},
		Since:             "5m",
		Limit:             1000,
		ExcludeSubsystems: []string{"exclude1"},
		ExcludePattern:    "pattern",
	}

	assert.Equal(t, "booted", defaults.Simulator)
	assert.Equal(t, "com.test", defaults.App)
	assert.Equal(t, 100, defaults.BufferSize)
	assert.Len(t, defaults.Subsystems, 2)
	assert.Len(t, defaults.Categories, 1)
}

func TestFindConfigFile(t *testing.T) {
	t.Run("finds .xcw.yaml in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(origDir)

		// Create config file
		configPath := filepath.Join(tmpDir, ".xcw.yaml")
		err := os.WriteFile(configPath, []byte("format: text"), 0644)
		require.NoError(t, err)

		found := findConfigFile()
		// Resolve symlinks for comparison (macOS /var -> /private/var)
		expectedPath, _ := filepath.EvalSymlinks(configPath)
		foundPath, _ := filepath.EvalSymlinks(found)
		assert.Equal(t, expectedPath, foundPath)
	})

	t.Run("finds .xcw.yml in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(origDir)

		configPath := filepath.Join(tmpDir, ".xcw.yml")
		err := os.WriteFile(configPath, []byte("format: text"), 0644)
		require.NoError(t, err)

		found := findConfigFile()
		expectedPath, _ := filepath.EvalSymlinks(configPath)
		foundPath, _ := filepath.EvalSymlinks(found)
		assert.Equal(t, expectedPath, foundPath)
	})

	t.Run("prefers .xcw.yaml over .xcw.yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(origDir)

		// Create both files
		yamlPath := filepath.Join(tmpDir, ".xcw.yaml")
		ymlPath := filepath.Join(tmpDir, ".xcw.yml")
		err := os.WriteFile(yamlPath, []byte("format: yaml"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(ymlPath, []byte("format: yml"), 0644)
		require.NoError(t, err)

		found := findConfigFile()
		expectedPath, _ := filepath.EvalSymlinks(yamlPath)
		foundPath, _ := filepath.EvalSymlinks(found)
		assert.Equal(t, expectedPath, foundPath)
	})

	t.Run("returns empty string when no config found", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(origDir)

		found := findConfigFile()
		assert.Empty(t, found)
	})
}

func TestApplyEnvOverrides(t *testing.T) {
	t.Run("overrides format from env", func(t *testing.T) {
		cfg := Default()
		os.Setenv("XCW_FORMAT", "text")
		defer os.Unsetenv("XCW_FORMAT")

		applyEnvOverrides(cfg)
		assert.Equal(t, "text", cfg.Format)
	})

	t.Run("overrides quiet from env with true", func(t *testing.T) {
		cfg := Default()
		os.Setenv("XCW_QUIET", "true")
		defer os.Unsetenv("XCW_QUIET")

		applyEnvOverrides(cfg)
		assert.True(t, cfg.Quiet)
	})

	t.Run("overrides quiet from env with 1", func(t *testing.T) {
		cfg := Default()
		os.Setenv("XCW_QUIET", "1")
		defer os.Unsetenv("XCW_QUIET")

		applyEnvOverrides(cfg)
		assert.True(t, cfg.Quiet)
	})

	t.Run("does not override quiet with other values", func(t *testing.T) {
		cfg := Default()
		os.Setenv("XCW_QUIET", "yes")
		defer os.Unsetenv("XCW_QUIET")

		applyEnvOverrides(cfg)
		assert.False(t, cfg.Quiet)
	})

	t.Run("overrides app from env", func(t *testing.T) {
		cfg := Default()
		os.Setenv("XCW_APP", "com.example.app")
		defer os.Unsetenv("XCW_APP")

		applyEnvOverrides(cfg)
		assert.Equal(t, "com.example.app", cfg.Defaults.App)
	})
}
