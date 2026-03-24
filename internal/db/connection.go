package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "gitee.com/chunanyong/dm"
	_ "gitee.com/opengauss/openGauss-connector-go-pq"
	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mingyuan/pomelo-db/internal/config"
	_ "modernc.org/sqlite"
)

// Connection represents a database connection
type Connection struct {
	db     *sql.DB
	dbType string
}

// NewConnection creates a new database connection
func NewConnection(cfg *config.DatasourceConfig) (*Connection, error) {
	slog.Debug("connecting to database", "type", cfg.Type, "host", cfg.Host, "database", cfg.Database)

	dsn, err := buildDSN(cfg)
	if err != nil {
		return nil, err
	}

	var driverName string
	switch cfg.Type {
	case "mysql":
		driverName = "mysql"
	case "sqlserver":
		driverName = "sqlserver"
	case "vastbase", "opengauss":
		driverName = "opengauss"
	case "dm":
		driverName = "dm"
	case "sqlite":
		driverName = "sqlite"
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		slog.Error("failed to open database", "type", cfg.Type, "error", err)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		slog.Error("failed to connect to database", "type", cfg.Type, "error", err)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	slog.Debug("database connection established", "type", cfg.Type)
	return &Connection{
		db:     db,
		dbType: cfg.Type,
	}, nil
}

// buildDSN builds database connection string
func buildDSN(cfg *config.DatasourceConfig) (string, error) {
	switch cfg.Type {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database), nil
	case "sqlserver":
		return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database), nil
	case "vastbase", "opengauss":
		// For PostgreSQL-based databases, use 'database' as dbname (matches Python)
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database)
		if schema := cfg.Options["schema"]; schema != "" {
			dsn += fmt.Sprintf(" search_path=%s", schema)
		}
		dsn += " TimeZone=Asia/Shanghai sslmode=disable"
		return dsn, nil
	case "dm":
		return fmt.Sprintf("dm://%s:%s@%s:%d/%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database), nil
	case "sqlite":
		// Extract the actual file path (remove sqlite:// prefix if present)
		dbPath := cfg.Database
		// Check if file exists for better error message
		// Skip for :memory: databases, empty paths, or absolute paths (sqlite:///path)
		if dbPath != ":memory:" && dbPath != "" && dbPath[0] != '?' && dbPath[0] != '/' {
			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				return "", fmt.Errorf("database file does not exist: %s", dbPath)
			}
		}
		return cfg.Database, nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

// Query executes a query and returns results with column order
func (c *Connection) Query(ctx context.Context, query string) ([]string, []map[string]interface{}, error) {
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	columns, results, err := scanRows(rows)
	if err != nil {
		return nil, nil, err
	}

	return columns, results, nil
}

// Exec executes a statement and returns affected rows
func (c *Connection) Exec(ctx context.Context, stmt string) (int64, error) {
	result, err := c.db.ExecContext(ctx, stmt)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return affected, nil
}

// Close closes the database connection
func (c *Connection) Close() error {
	return c.db.Close()
}

// Ping tests the database connection
func (c *Connection) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}
