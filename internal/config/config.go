package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config holds application configuration
type Config struct {
	// Global settings
	Format  string `mapstructure:"format"`
	Level   string `mapstructure:"level"`
	Quiet   bool   `mapstructure:"quiet"`
	Verbose bool   `mapstructure:"verbose"`

	// Default values for commands
	Defaults DefaultsConfig `mapstructure:"defaults"`
}

// DefaultsConfig holds default values for various commands
type DefaultsConfig struct {
	// Tail command defaults
	Simulator       string   `mapstructure:"simulator"`
	App             string   `mapstructure:"app"`
	BufferSize      int      `mapstructure:"buffer_size"`
	SummaryInterval string   `mapstructure:"summary_interval"`
	Heartbeat       string   `mapstructure:"heartbeat"`
	Subsystems      []string `mapstructure:"subsystems"`
	Categories      []string `mapstructure:"categories"`

	// Query command defaults
	Since string `mapstructure:"since"`
	Limit int    `mapstructure:"limit"`

	// Exclusion filters
	ExcludeSubsystems []string `mapstructure:"exclude_subsystems"`
	ExcludePattern    string   `mapstructure:"exclude_pattern"`
}

// Default returns a Config with default values
func Default() *Config {
	return &Config{
		Format:  "ndjson",
		Level:   "default",
		Quiet:   false,
		Verbose: false,
		Defaults: DefaultsConfig{
			Simulator:  "booted",
			BufferSize: 100,
			Since:      "5m",
			Limit:      1000,
		},
	}
}

// Load loads configuration from files and environment
func Load() (*Config, error) {
	v := viper.New()

	// Set config name and type
	v.SetConfigName("xcw")
	v.SetConfigType("yaml")

	// Add config paths (in order of precedence, lowest first)
	// 1. System-wide config
	v.AddConfigPath("/etc/xcw/")
	// 2. User config directory
	if configDir, err := os.UserConfigDir(); err == nil {
		v.AddConfigPath(filepath.Join(configDir, "xcw"))
	}
	// 3. Home directory (as .xcwrc.yaml or .xcw.yaml)
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(home)
		v.SetConfigName(".xcw")
	}
	// 4. Current directory
	v.AddConfigPath(".")

	// Also check for .xcwrc file
	v.SetConfigName(".xcwrc")
	v.AddConfigPath(".")
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(home)
	}

	// Environment variables
	v.SetEnvPrefix("XCW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Bind specific environment variables
	v.BindEnv("format", "XCW_FORMAT")
	v.BindEnv("level", "XCW_LEVEL")
	v.BindEnv("quiet", "XCW_QUIET")
	v.BindEnv("verbose", "XCW_VERBOSE")
	v.BindEnv("defaults.app", "XCW_APP")
	v.BindEnv("defaults.simulator", "XCW_SIMULATOR")

	// Set defaults
	cfg := Default()
	v.SetDefault("format", cfg.Format)
	v.SetDefault("level", cfg.Level)
	v.SetDefault("quiet", cfg.Quiet)
	v.SetDefault("verbose", cfg.Verbose)
	v.SetDefault("defaults.simulator", cfg.Defaults.Simulator)
	v.SetDefault("defaults.buffer_size", cfg.Defaults.BufferSize)
	v.SetDefault("defaults.since", cfg.Defaults.Since)
	v.SetDefault("defaults.limit", cfg.Defaults.Limit)

	// Try to read config file (ignore if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error occurred
			return nil, err
		}
		// Config file not found; use defaults
	}

	// Unmarshal into struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromFile loads configuration from a specific file
func LoadFromFile(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := Default()
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ConfigFile returns the path to the config file that was loaded
func ConfigFile() string {
	v := viper.New()

	v.SetConfigName("xcw")
	v.SetConfigType("yaml")

	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(home)
	}
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err == nil {
		return v.ConfigFileUsed()
	}

	// Try .xcwrc
	v.SetConfigName(".xcwrc")
	if err := v.ReadInConfig(); err == nil {
		return v.ConfigFileUsed()
	}

	return ""
}
