package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/mingyuan/pomelo-db/internal/config"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// getSQL gets SQL from command line or file
func getSQL() (string, error) {
	if execute != "" {
		return cleanSQL(execute), nil
	}

	if file != "" {
		slog.Debug("reading sql from file", "path", file)
		content, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		slog.Debug("sql file loaded", "path", file, "bytes", len(content))
		return cleanSQL(string(content)), nil
	}

	return "", fmt.Errorf("either --execute or --file must be specified")
}

// cleanSQL removes shell line-continuation sequences (backslash + newline)
// and collapses extra whitespace so multi-line SQL works naturally.
func cleanSQL(sql string) string {
	// Remove \<newline> (shell continuation) — handles \\\n, \\\r\n
	sql = strings.ReplaceAll(sql, "\\\r\n", " ")
	sql = strings.ReplaceAll(sql, "\\\n", " ")
	return strings.TrimSpace(sql)
}

// outputResults outputs query results as JSON (aligned with Python format)
func outputResults(columns []string, results []map[string]interface{}) error {
	// Convert time.Time values to strings for JSON serialization
	for i := range results {
		for k, v := range results[i] {
			if t, ok := v.(time.Time); ok {
				results[i][k] = t.Format(time.RFC3339Nano)
			}
		}
	}

	if len(results) == 0 {
		// Empty result set (aligned with Python format)
		output := map[string]interface{}{
			"row_affected": 0,
			"message":      "No results",
			"data":         []map[string]interface{}{},
		}
		return encodeJSON(output)
	}

	// Non-empty result (aligned with Python format)
	output := map[string]interface{}{
		"row_affected": len(results),
		"data":         results,
	}

	return encodeJSON(output)
}

// outputResultsAsTable outputs query results as a table
func outputResultsAsTable(columns []string, results []map[string]interface{}, elapsed time.Duration) error {
	if len(results) == 0 {
		fmt.Println("No results")
		return nil
	}

	// Create table with ASCII style and no borders
	table := tablewriter.NewWriter(os.Stdout)

	// Configure table style - use ASCII symbols and disable outer borders
	table.Options(
		tablewriter.WithSymbols(tw.NewSymbols(tw.StyleASCII)),
		tablewriter.WithRendition(tw.Rendition{
			Borders: tw.Border{
				Left:   tw.Off,
				Right:  tw.Off,
				Top:    tw.Off,
				Bottom: tw.Off,
			},
		}),
	)

	// Set header - convert []string to []any for variadic parameter
	headerAny := make([]any, len(columns))
	for i, col := range columns {
		headerAny[i] = col
	}
	table.Header(headerAny...)

	// Add rows using column order
	for _, row := range results {
		var rowData []any
		for _, col := range columns {
			val := row[col]
			// Format time.Time values using ISO 8601 format
			if t, ok := val.(time.Time); ok {
				rowData = append(rowData, t.Format(time.RFC3339Nano))
			} else {
				rowData = append(rowData, fmt.Sprintf("%v", val))
			}
		}
		if err := table.Append(rowData...); err != nil {
			return fmt.Errorf("failed to append row: %w", err)
		}
	}

	if err := table.Render(); err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	fmt.Printf("\n%d row(s) in %d ms\n", len(results), elapsed.Milliseconds())
	return nil
}

// outputStatementResult outputs statement execution result (aligned with Python format)
func outputStatementResult(rowcount int64, elapsed time.Duration) error {
	output := map[string]interface{}{
		"row_affected": rowcount,
		"message":      fmt.Sprintf("%d row(s) affected in %d ms", rowcount, elapsed.Milliseconds()),
	}

	return encodeJSON(output)
}

// outputStatementResultAsTable outputs statement execution result as a table
func outputStatementResultAsTable(rowcount int64, elapsed time.Duration) error {
	fmt.Printf("%d row(s) affected in %d ms\n", rowcount, elapsed.Milliseconds())
	return nil
}

// encodeJSON encodes output to stdout as indented JSON
func encodeJSON(output interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode results: %w", err)
	}
	return nil
}

// addDatasourceCommand adds a new datasource to .env file
// Format: name=dsn (e.g., mydb=mysql://user:pass@host:3306/db)
// Returns error if datasource already exists or DSN is invalid.
func addDatasourceCommand(ds string) error {
	name, dsn, ok := strings.Cut(ds, "=")
	if !ok || strings.TrimSpace(name) == "" {
		return fmt.Errorf("invalid format for -a: %q (expected name=dsn)", ds)
	}
	name = strings.TrimSpace(name)
	dsn = strings.TrimSpace(dsn)
	envKey := "POMELO_DB_" + strings.ToUpper(name)

	// Validate DSN format before adding
	if _, err := config.ParseDSN(dsn); err != nil {
		return fmt.Errorf("invalid DSN: %w", err)
	}

	envPath := ".env"

	// Read existing content to check for duplicates and trailing newline
	existingData, _ := os.ReadFile(envPath)
	for _, line := range strings.Split(string(existingData), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, envKey+"=") {
			return fmt.Errorf("datasource '%s' already exists in .env", name)
		}
	}

	// Prepend newline if file exists and doesn't end with one
	entry := formatEnvLine(envKey, dsn) + "\n"
	if len(existingData) > 0 && existingData[len(existingData)-1] != '\n' {
		entry = "\n" + entry
	}

	f, err := os.OpenFile(envPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .env file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write to .env file: %w", err)
	}

	fmt.Printf("Added datasource: %s\n", name)
	fmt.Printf("Saved to: %s\n", envPath)
	return nil
}

// removeDatasourceCommand removes a datasource from .env file
func removeDatasourceCommand(name string) error {
	envPath := ".env"
	name = strings.TrimSpace(name)
	envKey := "POMELO_DB_" + strings.ToUpper(name)

	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf(".env file not found")
	}

	found := false
	var newLines []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			newLines = append(newLines, line)
			continue
		}
		if strings.HasPrefix(trimmed, envKey+"=") {
			found = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !found {
		return fmt.Errorf("datasource '%s' not found in .env", name)
	}

	content := strings.Join(newLines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	fmt.Printf("Removed datasource: %s\n", name)
	fmt.Printf("Saved to: %s\n", envPath)
	return nil
}

// formatEnvLine formats a key=value line, quoting if needed
func formatEnvLine(key, value string) string {
	if strings.ContainsAny(value, " \"'#$!") {
		return fmt.Sprintf(`%s="%s"`, key, value)
	}
	return key + "=" + value
}
