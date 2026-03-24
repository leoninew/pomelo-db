package db

import (
	"database/sql"
	"fmt"
)

// scanRows scans all rows from a result set and returns columns order
// Handles duplicate and anonymous column names similar to Python version
func scanRows(rows *sql.Rows) ([]string, []map[string]interface{}, error) {
	rawColumns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Process column names to handle duplicates and anonymous columns
	// Similar to Python's SessionExtension.fetch_all()
	columnNames := processColumnNames(rawColumns)

	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(rawColumns))
		valuePtrs := make([]interface{}, len(rawColumns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]interface{})
		for i, colName := range columnNames {
			val := values[i]
			// Convert []byte to string for better display
			if b, ok := val.([]byte); ok {
				row[colName] = string(b)
			} else {
				row[colName] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("row iteration error: %w", err)
	}

	return columnNames, results, nil
}

// processColumnNames handles duplicate and anonymous column names
// Matching Python's logic in SessionExtension.fetch_all()
func processColumnNames(rawColumns []string) []string {
	columnNames := make([]string, len(rawColumns))
	seenNames := make(map[string]bool)
	anonymousCount := 0

	for i, col := range rawColumns {
		// Check if this is an anonymous column
		// PostgreSQL: "?column?", SQL Server: empty string ""
		isAnonymous := col == "" || col == "?column?"

		if isAnonymous {
			// Generate unique name for anonymous column
			columnNames[i] = fmt.Sprintf("col_%d", anonymousCount)
			anonymousCount++
		} else {
			// Handle duplicate column names
			if seenNames[col] {
				// Column name already used, generate unique name
				columnNames[i] = fmt.Sprintf("col_%d", anonymousCount)
				anonymousCount++
			} else {
				// First occurrence, use original name
				columnNames[i] = col
				seenNames[col] = true
			}
		}
	}

	return columnNames
}
