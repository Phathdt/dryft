package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/phathdt/dryft/internal/schema"
)

// indexRow represents a row from the index query.
type indexRow struct {
	IndexName   string
	IsUnique    bool
	IndexType   string
	Columns     []string
	WhereClause sql.NullString
}

// IntrospectIndexes retrieves all indexes for a table (excluding primary key).
func (p *PostgresIntrospector) IntrospectIndexes(ctx context.Context, schemaName, tableName string) ([]schema.Index, error) {
	query := `
		SELECT
			ic.relname AS index_name,
			i.indisunique AS is_unique,
			am.amname AS index_type,
			ARRAY(
				SELECT a.attname
				FROM pg_attribute a
				WHERE a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
				ORDER BY array_position(i.indkey, a.attnum)
			) AS columns,
			pg_get_expr(i.indpred, i.indrelid) AS where_clause
		FROM pg_index i
		JOIN pg_class ic ON ic.oid = i.indexrelid
		JOIN pg_am am ON am.oid = ic.relam
		WHERE i.indrelid = ($1 || '.' || $2)::regclass
		  AND NOT i.indisprimary
		ORDER BY ic.relname`

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	rows, err := p.pool.Query(queryCtx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("query indexes: %w", err)
	}
	defer rows.Close()

	var indexes []schema.Index
	for rows.Next() {
		var row indexRow
		if err := rows.Scan(
			&row.IndexName,
			&row.IsUnique,
			&row.IndexType,
			&row.Columns,
			&row.WhereClause,
		); err != nil {
			return nil, fmt.Errorf("scan index row: %w", err)
		}

		idx := schema.Index{
			Name:   row.IndexName,
			Unique: row.IsUnique,
			Type:   mapIndexType(row.IndexType),
			Where:  row.WhereClause.String,
		}

		// Convert column names to IndexColumn structs
		for _, colName := range row.Columns {
			idx.Columns = append(idx.Columns, schema.IndexColumn{
				Name:     colName,
				Order:    schema.SortAsc,
				NullsPos: schema.NullsDefault,
			})
		}

		indexes = append(indexes, idx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate indexes: %w", err)
	}

	return indexes, nil
}

// mapIndexType converts PostgreSQL index type name to schema.IndexType.
func mapIndexType(indexType string) schema.IndexType {
	switch indexType {
	case "btree":
		return schema.IndexBTree
	case "hash":
		return schema.IndexHash
	case "gin":
		return schema.IndexGIN
	case "gist":
		return schema.IndexGiST
	case "spgist":
		return schema.IndexSPGiST
	case "brin":
		return schema.IndexBRIN
	default:
		return schema.IndexBTree
	}
}
