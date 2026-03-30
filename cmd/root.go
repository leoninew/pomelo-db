package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mingyuan/pomelo-db/internal/config"
	"github.com/mingyuan/pomelo-db/internal/query"
	"github.com/spf13/cobra"
)

var (
	datasource       string
	execute          string
	file             string
	format           string
	timeout          int
	listDatasources  bool
	verbose          bool
	showConfig       string
	addDatasource    string
	removeDatasource string
	allowWrite       bool
	configDefaults   []byte
	rootCmd          = &cobra.Command{
		Use:   "pomelo-db",
		Short: "Pomelo DB - Database query tool",
		Long: `Database query tool with datasource management.

USAGE:
  pomelo-db -l                                # List all datasources
  pomelo-db -a mydb=mysql://...               # Add a datasource
  pomelo-db -r mydb                           # Remove a datasource
  pomelo-db -d mydb -e "SELECT * FROM t"      # Execute query (readonly, JSON output)
  pomelo-db -d mydb -e "SELECT" -o table        # Execute query (table output)
  pomelo-db -d mydb -e "INSERT..." -w         # Execute write operation

DSN FORMAT:
  mysql://user:pass@host:port/db
  sqlite://./path/to/db
  sqlserver://user:pass@host:port/db
  vastbase://user:pass@host:port/db?schema=public
  opengauss://user:pass@host:port/db
  dm://user:pass@host:port/db`,
		RunE:  runQuery,
	}
)

func init() {
	// Disable completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Disable usage printing on error
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true // Don't print error, we log it ourselves

	// Info flags
	rootCmd.Flags().BoolVarP(&listDatasources, "list", "l", false, "list all configured datasources")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output (show config sources)")
	rootCmd.Flags().StringVarP(&showConfig, "show-config", "s", "", "show config for a specific datasource")

	// Datasource management flags
	rootCmd.Flags().StringVarP(&addDatasource, "add", "a", "", "add a new datasource to .env (format: name=dsn, e.g., mydb=mysql://user:pass@host:3306/db)")
	rootCmd.Flags().StringVarP(&removeDatasource, "remove", "r", "", "remove a datasource from .env")

	// Query flags
	rootCmd.Flags().StringVarP(&datasource, "datasource", "d", "", "datasource name (required for queries)")
	rootCmd.Flags().StringVarP(&execute, "execute", "e", "", "SQL query to execute")
	rootCmd.Flags().StringVarP(&file, "file", "f", "", "SQL file to execute")
	rootCmd.Flags().StringVarP(&format, "output", "o", "json", "output format: json or table")
	rootCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "query timeout in seconds")
	rootCmd.Flags().BoolVarP(&allowWrite, "allow-write", "w", false, "allow write operations (INSERT/UPDATE/DELETE)")
}

// setupLogger configures the global logger based on log level string.
// Supported levels: debug, info, warn, error. Defaults to info if unrecognized.
func setupLogger(logLevel string) {
	var level slog.Level
	switch strings.ToLower(logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Log to stderr (like Python version)
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Simplify log format to match Python version: just the message
			if a.Key == slog.TimeKey || a.Key == slog.LevelKey {
				return slog.Attr{}
			}
			return a
		},
	})
	slog.SetDefault(slog.New(handler))
}

func runQuery(cmd *cobra.Command, args []string) error {
	// 1. Load configuration
	cfg, err := config.Load(configDefaults)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Setup logger (use info level if verbose, otherwise use config)
	logLevel := cfg.Log.Level
	if verbose {
		logLevel = "info"
	}
	setupLogger(logLevel)

	slog.Info("loading configuration")
	slog.Debug("effective log level", "level", logLevel)

	// Handle --list flag
	if listDatasources {
		return listDatasourcesCommand(cfg)
	}

	// Handle --show-config flag
	if showConfig != "" {
		return showConfigCommand(cfg, showConfig)
	}

	// Handle --add flag (add/update datasource to .env)
	if addDatasource != "" {
		return addDatasourceCommand(addDatasource)
	}

	// Handle --remove flag (remove datasource from .env)
	if removeDatasource != "" {
		return removeDatasourceCommand(removeDatasource)
	}

	// If no datasource and no SQL provided, show help
	if datasource == "" && execute == "" && file == "" {
		return cmd.Help()
	}

	// Require datasource for actual queries
	if datasource == "" {
		return fmt.Errorf("--datasource/-d is required for queries (use --list to see available datasources)")
	}

	// Get datasource config
	slog.Info("resolving datasource", "name", datasource)
	dsConfig, err := cfg.GetDatasource(datasource)
	if err != nil {
		return err
	}
	slog.Info("datasource resolved", "type", dsConfig.Type, "host", dsConfig.Host, "port", dsConfig.Port, "database", dsConfig.Database)

	// In write mode, bypass allowed_operators so any SQL can be executed;
	// the configured restriction list only applies to read queries.
	allowedOps := cfg.Query.AllowedOperators
	if allowWrite {
		allowedOps = []string{}
	}
	tool, err := query.NewTool(dsConfig, allowedOps)
	if err != nil {
		return fmt.Errorf("failed to create query tool: %w", err)
	}
	defer tool.Close()

	// Get SQL to execute
	sql, err := getSQL()
	if err != nil {
		return err
	}
	slog.Info("sql prepared", "source", sqlSource(), "length", len(sql))

	// Execute SQL with timeout
	timeoutDuration := time.Duration(timeout) * time.Second

	slog.Info("executing sql", "timeout", timeoutDuration, "format", format, "write", allowWrite)

	if allowWrite {
		start := time.Now()
		affected, err := tool.ExecuteStatement(sql, timeoutDuration)
		elapsed := time.Since(start)
		if err != nil {
			return err
		}
		slog.Info("statement completed", "affected", affected)
		if format == "table" {
			return outputStatementResultAsTable(affected, elapsed)
		}
		return outputStatementResult(affected, elapsed)
	}

	start := time.Now()
	columns, results, err := tool.ExecuteQuery(sql, timeoutDuration)
	elapsed := time.Since(start)
	if err != nil {
		return err
	}
	slog.Info("query completed", "columns", len(columns), "rows", len(results))

	if format == "table" {
		return outputResultsAsTable(columns, results, elapsed)
	}
	return outputResults(columns, results)
}

// sqlSource returns a label describing where the SQL comes from
func sqlSource() string {
	if execute != "" {
		return "cli"
	}
	if file != "" {
		return "file:" + file
	}
	return "unknown"
}

// listDatasourcesCommand lists all configured datasources
func listDatasourcesCommand(cfg *config.Config) error {
	// Show config sources in verbose mode
	if verbose && len(cfg.Sources) > 0 {
		slog.Info("config sources (priority: low -> high)")
		for i, src := range cfg.Sources {
			slog.Info("source", "order", i+1, "path", src)
		}
	}

	if len(cfg.Query.Datasources) == 0 {
		fmt.Println("No datasources configured.")
		return nil
	}
	// Sort names for consistent output
	names := make([]string, 0, len(cfg.Query.Datasources))
	for name := range cfg.Query.Datasources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}

// showConfigCommand shows config for a specific datasource
func showConfigCommand(cfg *config.Config, name string) error {
	dsn, ok := cfg.Query.Datasources[name]
	if !ok {
		return fmt.Errorf("datasource '%s' not found (use --list to see available datasources)", name)
	}
	fmt.Printf("%s: %s\n", name, maskPassword(dsn))
	return nil
}

// maskPassword masks password in DSN string for display
func maskPassword(dsn string) string {
	// Handle sqlite (no password)
	if strings.HasPrefix(dsn, "sqlite://") {
		return dsn
	}

	// Parse URL and mask password
	if idx := strings.Index(dsn, "://"); idx > 0 {
		scheme := dsn[:idx+3]
		rest := dsn[idx+3:]

		// Find @ symbol (separates user:pass from host)
		if atIdx := strings.Index(rest, "@"); atIdx > 0 {
			userPass := rest[:atIdx]
			hostPart := rest[atIdx:]

			// Mask password
			if colonIdx := strings.Index(userPass, ":"); colonIdx > 0 {
				user := userPass[:colonIdx]
				return scheme + user + ":****" + hostPart
			}
		}
	}

	return dsn
}

// Execute runs the root command with embedded default configuration and version.
func Execute(defaults []byte, version string) error {
	configDefaults = defaults
	rootCmd.Version = version
	return rootCmd.Execute()
}
