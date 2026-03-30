package config

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Log     LogConfig   `mapstructure:"log"`
	Query   QueryConfig `mapstructure:"query"`
	Sources []string    `mapstructure:"-"` // Configuration sources (not from yaml)
}

// LogConfig represents the log configuration section
type LogConfig struct {
	Level string `mapstructure:"level"`
}

// QueryConfig represents the query configuration section
type QueryConfig struct {
	AllowedOperators []string          `mapstructure:"allowed_operators"`
	Datasources     map[string]string `mapstructure:"datasources"` // DSN format only
}

// DatasourceConfig represents a single datasource configuration
type DatasourceConfig struct {
	Type     string            `mapstructure:"type"`
	Host     string            `mapstructure:"host"`
	Port     int               `mapstructure:"port"`
	Database string            `mapstructure:"database"` // Used for all database types (matches Python)
	User     string            `mapstructure:"user"`
	Password string            `mapstructure:"password"`
	Options  map[string]string `mapstructure:"options"` // Optional parameters from query string (schema, charset, etc.)
}

// Load loads configuration.
// defaults: embedded config.defaults.yaml content.
//
// Configuration priority (highest to lowest):
// 1. .env file (current directory) ← HIGHEST for datasources
// 2. config.yaml (current directory) ← project-level config
// 3. config.yaml (user home directory ~/.pomelo-db/) ← global config
// 4. Embedded defaults
func Load(defaults []byte) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	// Set defaults before loading any config
	var sources []string

	// 1. Load embedded defaults
	if err := v.ReadConfig(bytes.NewReader(defaults)); err != nil {
		return nil, fmt.Errorf("failed to parse embedded defaults: %w", err)
	}
	sources = append(sources, "embedded defaults")
	slog.Debug("loaded embedded default configuration")

	// 2. Merge user config on top of defaults
	userConfigPath := resolveUserConfigPath()
	if userConfigPath != "" {
		userData, err := os.ReadFile(userConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", userConfigPath, err)
		}
		if err := v.MergeConfig(bytes.NewReader(userData)); err != nil {
			return nil, fmt.Errorf("failed to parse config from %s: %w", userConfigPath, err)
		}
		sources = append(sources, userConfigPath)
		slog.Debug("merged user configuration", "path", userConfigPath)
	}

	// 3. Unmarshal into struct (before .env to get base config)
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Normalize nil slices to empty slices so callers only need to check len()
	if cfg.Query.AllowedOperators == nil {
		cfg.Query.AllowedOperators = []string{}
	}

	// 4. Load .env file if exists (project-level, HIGHEST priority for datasources)
	envDatasources, envPath, err := loadEnvDatasources()
	if err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}
	if len(envDatasources) > 0 {
		// Merge .env datasources (overwrite existing)
		if cfg.Query.Datasources == nil {
			cfg.Query.Datasources = make(map[string]string)
		}
		for name, dsn := range envDatasources {
			cfg.Query.Datasources[name] = dsn
		}
		sources = append(sources, envPath)
		slog.Debug("merged .env datasources", "count", len(envDatasources))
	}

	cfg.Sources = sources

	// 5. Validate configuration
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	datasourceCount := len(cfg.Query.Datasources)
	slog.Debug("configuration loaded successfully", "datasources", datasourceCount, "log_level", cfg.Log.Level)
	return &cfg, nil
}

// resolveUserConfigPath determines the user config file path.
// Returns empty string if no user config file is found.
// Priority: ./config.yaml > ~/.pomelo-db/config.yaml
func resolveUserConfigPath() string {
	// 1. Try current directory (project-level, highest priority)
	if _, err := os.Stat("config.yaml"); err == nil {
		slog.Debug("config file found in current directory", "path", "config.yaml")
		return "config.yaml"
	}

	// 2. Try user home directory (global)
	home, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(home, ".pomelo-db", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			slog.Debug("config file found in home directory", "path", configPath)
			return configPath
		}
	}

	slog.Debug("no user config file found, using defaults only")
	return ""
}

// loadEnvDatasources loads datasources from .env file in current directory.
// Format: POMELO_DB_<NAME>=<DSN>
// Example:
//   POMELO_DB_MYDB=sqlite:///./data.db
//   POMELO_DB_PROD=mysql://user:pass@host:3306/db
//
// Returns map of datasource name to DSN string, and the env file path if loaded.
func loadEnvDatasources() (map[string]string, string, error) {
	envPath := ".env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return nil, "", nil // No .env file, not an error
	}

	// Read .env file
	data, err := os.ReadFile(envPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read .env file: %w", err)
	}

	datasources := make(map[string]string)
	lines := strings.Split(string(data), "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue // Skip malformed lines
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Only process POMELO_DB_* variables
		if !strings.HasPrefix(key, "POMELO_DB_") {
			continue
		}

		// Extract datasource name (lowercase)
		name := strings.ToLower(strings.TrimPrefix(key, "POMELO_DB_"))
		if name == "" {
			slog.Warn("invalid datasource name in .env", "line", i+1, "key", key)
			continue
		}

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		if value == "" {
			slog.Warn("empty datasource value in .env", "line", i+1, "key", key)
			continue
		}

		datasources[name] = value
		slog.Debug("loaded datasource from .env", "name", name, "line", i+1)
	}

	if len(datasources) > 0 {
		slog.Info("loaded datasources from .env", "path", envPath, "count", len(datasources))
	}

	return datasources, envPath, nil
}

// GetDatasource returns a datasource configuration by name
func (c *Config) GetDatasource(name string) (*DatasourceConfig, error) {
	dsn, ok := c.Query.Datasources[name]
	if !ok {
		available := make([]string, 0, len(c.Query.Datasources))
		for k := range c.Query.Datasources {
			available = append(available, k)
		}
		return nil, fmt.Errorf("datasource '%s' not found. Available: %v", name, available)
	}

	// Parse DSN string
	ds, err := ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("datasource '%s': %w", name, err)
	}

	// Validate datasource configuration before returning
	if err := ds.Validate(); err != nil {
		return nil, fmt.Errorf("datasource '%s': %w", name, err)
	}

	return ds, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Query.Datasources) == 0 {
		return fmt.Errorf("no datasources configured")
	}

	for name, dsn := range c.Query.Datasources {
		ds, err := ParseDSN(dsn)
		if err != nil {
			return fmt.Errorf("datasource '%s': %w", name, err)
		}
		if err := ds.Validate(); err != nil {
			return fmt.Errorf("datasource '%s': %w", name, err)
		}
	}

	return nil
}

// Validate validates a datasource configuration
func (d *DatasourceConfig) Validate() error {
	if d.Type == "" {
		return fmt.Errorf("type is required")
	}

	// Validate supported database types (matching Python)
	validTypes := map[string]bool{
		"mysql":     true,
		"sqlserver": true,
		"dm":        true,
		"opengauss": true,
		"vastbase":  true,
		"sqlite":    true,
	}
	if !validTypes[d.Type] {
		return fmt.Errorf("unsupported type: %s (supported: mysql, sqlserver, dm, opengauss, vastbase, sqlite)", d.Type)
	}

	// SQLite only requires database (file path)
	if d.Type == "sqlite" {
		if d.Database == "" {
			return fmt.Errorf("database (file path) is required for sqlite")
		}
		return nil
	}

	if d.Host == "" {
		return fmt.Errorf("host is required")
	}
	if d.Port == 0 {
		return fmt.Errorf("port is required")
	}
	if d.Database == "" {
		return fmt.Errorf("database is required")
	}
	if d.User == "" {
		return fmt.Errorf("user is required")
	}
	return nil
}

// ParseDSN parses a DSN string into DatasourceConfig
// Format: <db-type>://<user>:<password>@<host>:<port>/<database>[?key=value&...]
// Special cases:
//   - sqlite:///path/to/file.db
//   - vastbase://user:pass@host:port/db?schema=public&charset=utf8
//
// All query parameters are stored in Options map for extensibility
func ParseDSN(dsn string) (*DatasourceConfig, error) {
	// Handle sqlite special case: sqlite://<file_path>
	// - sqlite://C:/data/db.sqlite (Windows absolute)
	// - sqlite:///var/data/db.sqlite (Linux absolute, triple slash)
	// - sqlite://./data/db.sqlite (relative)
	if strings.HasPrefix(dsn, "sqlite://") {
		dbPath := strings.TrimPrefix(dsn, "sqlite://")
		return &DatasourceConfig{
			Type:     "sqlite",
			Database: dbPath,
		}, nil
	}

	// Parse standard URL format
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN format: %w", err)
	}

	// Validate that scheme exists (must be a proper URL)
	if u.Scheme == "" {
		return nil, fmt.Errorf("invalid DSN format: missing scheme (e.g., mysql://)")
	}

	cfg := &DatasourceConfig{
		Type: u.Scheme,
	}

	// SQLite doesn't need host/port/user
	if cfg.Type == "sqlite" {
		dbPath := u.Path
		// Convert relative path to absolute path
		if !filepath.IsAbs(dbPath) {
			absPath, err := filepath.Abs(dbPath)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve sqlite path: %w", err)
			}
			dbPath = absPath
		}
		cfg.Database = dbPath
		return cfg, nil
	}

	// Extract user and password
	if u.User != nil {
		cfg.User = u.User.Username()
		if password, ok := u.User.Password(); ok {
			cfg.Password = password
		}
	}

	// Extract host and port
	cfg.Host = u.Hostname()
	if portStr := u.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port: %s", portStr)
		}
		cfg.Port = port
	}

	// Extract database (path without leading slash)
	cfg.Database = strings.TrimPrefix(u.Path, "/")

	// Extract all query parameters as options
	cfg.Options = make(map[string]string)
	for key, values := range u.Query() {
		if len(values) > 0 {
			cfg.Options[key] = values[0] // Take first value if multiple
		}
	}

	return cfg, nil
}
