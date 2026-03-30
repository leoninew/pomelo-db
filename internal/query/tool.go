package query

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mingyuan/pomelo-db/internal/config"
	"github.com/mingyuan/pomelo-db/internal/db"
)

// validateQuery validates SQL is not empty or whitespace-only
func validateQuery(sql string) error {
	if strings.TrimSpace(sql) == "" {
		return fmt.Errorf("SQL cannot be empty")
	}
	return nil
}

// Tool represents a query execution tool
type Tool struct {
	conn             *db.Connection
	allowedOperators []string
}

// NewTool creates a new query tool.
// allowedOperators: list of allowed SQL operator prefixes (e.g., SELECT, SHOW); empty slice means unrestricted.
func NewTool(cfg *config.DatasourceConfig, allowedOperators []string) (*Tool, error) {
	conn, err := db.NewConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &Tool{
		conn:             conn,
		allowedOperators: allowedOperators,
	}, nil
}

// ExecuteQuery executes a query and returns columns and results
func (t *Tool) ExecuteQuery(sql string, timeout time.Duration) ([]string, []map[string]interface{}, error) {
	if err := validateQuery(sql); err != nil {
		slog.Warn("query validation failed", "error", err)
		return nil, nil, err
	}

	// Validate allowed operators when restriction is configured (nil or empty = unrestricted)
	if len(t.allowedOperators) > 0 {
		sqlUpper := strings.TrimSpace(strings.ToUpper(sql))
		allowed := false
		for _, op := range t.allowedOperators {
			if strings.HasPrefix(sqlUpper, op) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, nil, fmt.Errorf("only %v queries are allowed; use --write to allow write operations", t.allowedOperators)
		}
	}

	slog.Debug("executing query", "timeout", timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	columns, results, err := t.conn.Query(ctx, sql)
	if err != nil {
		slog.Error("query failed", "error", err)
		return nil, nil, err
	}

	slog.Debug("query executed successfully", "rows", len(results))
	return columns, results, nil
}

// ExecuteStatement executes a DML statement (INSERT/UPDATE/DELETE) and returns affected rows
func (t *Tool) ExecuteStatement(sql string, timeout time.Duration) (int64, error) {
	if err := validateQuery(sql); err != nil {
		slog.Warn("statement validation failed", "error", err)
		return 0, err
	}

	slog.Debug("executing statement", "timeout", timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	affected, err := t.conn.Exec(ctx, sql)
	if err != nil {
		slog.Error("statement failed", "error", err)
		return 0, err
	}

	slog.Debug("statement executed successfully", "affected", affected)
	return affected, nil
}

// TestConnection tests the database connection
func (t *Tool) TestConnection(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return t.conn.Ping(ctx)
}

// Close closes the tool and releases resources
func (t *Tool) Close() error {
	return t.conn.Close()
}
