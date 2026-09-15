package postgres

import (
	"context"
	"fmt"

	"github.com/phathdt/dryft/internal/schema"
)

// enumRow represents a row from the pg_enum query.
type enumRow struct {
	EnumName   string
	EnumValues []string
}

// listEnums retrieves all enum types in the specified schema.
func (p *PostgresIntrospector) listEnums(ctx context.Context) ([]schema.Enum, error) {
	query := `
		SELECT
			t.typname AS enum_name,
			array_agg(e.enumlabel ORDER BY e.enumsortorder) AS enum_values
		FROM pg_type t
		JOIN pg_enum e ON e.enumtypid = t.oid
		JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE t.typcategory = 'E'
		  AND n.nspname = $1
		GROUP BY t.typname
		ORDER BY t.typname`

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	rows, err := p.pool.Query(queryCtx, query, p.schema)
	if err != nil {
		return nil, fmt.Errorf("query enums: %w", err)
	}
	defer rows.Close()

	var enums []schema.Enum
	for rows.Next() {
		var row enumRow
		if err := rows.Scan(&row.EnumName, &row.EnumValues); err != nil {
			return nil, fmt.Errorf("scan enum row: %w", err)
		}

		enumDef := schema.Enum{
			Name:   row.EnumName,
			Values: make([]schema.EnumValue, len(row.EnumValues)),
		}

		for i, label := range row.EnumValues {
			enumDef.Values[i] = schema.EnumValue{
				Label: label,
				Order: i,
			}
		}

		enums = append(enums, enumDef)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enums: %w", err)
	}

	return enums, nil
}
