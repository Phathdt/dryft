package postgres

import (
	"context"
	"fmt"
)

// TableInfo holds basic information about a database table.
type TableInfo struct {
	Name    string
	Comment string
}

// ListTables retrieves all tables in the configured schema (default "public").
func (p *PostgresIntrospector) ListTables(ctx context.Context) ([]TableInfo, error) {
	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	query := `
		SELECT
			c.relname AS table_name,
			obj_description(c.oid) AS comment
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'r'
		  AND n.nspname = $1
		ORDER BY c.relname
	`

	rows, err := p.pool.Query(queryCtx, query, p.schema)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		var comment *string

		err := rows.Scan(&table.Name, &comment)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}

		if comment != nil {
			table.Comment = *comment
		}

		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	return tables, nil
}
