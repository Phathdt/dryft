package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/phathdt/dryft/internal/schema"
)

// columnRow represents a row from the pg_attribute query.
type columnRow struct {
	ColumnName  string
	ColumnOrder int
	TypeName    string
	TypeMod     sql.NullInt32
	NotNull     bool
	DefaultExpr sql.NullString
	Identity    string
	Comment     sql.NullString
}

// IntrospectColumns retrieves all columns for a table.
func (p *PostgresIntrospector) IntrospectColumns(ctx context.Context, schemaName, tableName string) ([]schema.Column, error) {
	query := `
		SELECT
			a.attname AS column_name,
			a.attnum AS column_order,
			t.typname AS type_name,
			CASE
				WHEN t.typname = 'varchar' OR t.typname = 'numeric' OR t.typname = 'bpchar' THEN a.atttypmod
				ELSE NULL
			END AS type_mod,
			a.attnotnull AS not_null,
			pg_get_expr(d.adbin, d.adrelid) AS default_expr,
			a.attidentity AS identity,
			col_description(a.attrelid, a.attnum) AS comment
		FROM pg_attribute a
		JOIN pg_type t ON t.oid = a.atttypid
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE a.attrelid = ($1 || '.' || $2)::regclass
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		ORDER BY a.attnum`

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	rows, err := p.pool.Query(queryCtx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("query columns: %w", err)
	}
	defer rows.Close()

	var columns []schema.Column
	for rows.Next() {
		var row columnRow
		if err := rows.Scan(
			&row.ColumnName,
			&row.ColumnOrder,
			&row.TypeName,
			&row.TypeMod,
			&row.NotNull,
			&row.DefaultExpr,
			&row.Identity,
			&row.Comment,
		); err != nil {
			return nil, fmt.Errorf("scan column row: %w", err)
		}

		col, err := p.buildColumn(ctx, schemaName, tableName, row)
		if err != nil {
			return nil, fmt.Errorf("build column %q: %w", row.ColumnName, err)
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate columns: %w", err)
	}

	return columns, nil
}

// buildColumn constructs a schema.Column from a columnRow.
func (p *PostgresIntrospector) buildColumn(ctx context.Context, schemaName, tableName string, row columnRow) (schema.Column, error) {
	dataType, err := p.mapType(row.TypeName, row.TypeMod)
	if err != nil {
		return schema.Column{}, err
	}

	col := schema.Column{
		Name:     row.ColumnName,
		Type:     dataType,
		Nullable: !row.NotNull,
		Comment:  row.Comment.String,
	}

	// Handle IDENTITY columns (a = ALWAYS, d = BY DEFAULT)
	if row.Identity == "a" || row.Identity == "d" {
		// Identity columns have implicit sequence
		col.Default = &schema.DefaultValue{
			Kind: schema.DefaultSequence,
			Sequence: &schema.SequenceRef{
				Name:  fmt.Sprintf("%s_%s_seq", tableName, row.ColumnName),
				Owned: true,
			},
		}
	} else if row.DefaultExpr.Valid {
		defaultVal, err := p.parseDefault(ctx, schemaName, tableName, row.ColumnName, row.DefaultExpr.String)
		if err != nil {
			return schema.Column{}, fmt.Errorf("parse default: %w", err)
		}
		col.Default = defaultVal
	}

	return col, nil
}

// mapType converts PostgreSQL type information into internal DataType.
func (p *PostgresIntrospector) mapType(typeName string, typeMod sql.NullInt32) (schema.DataType, error) {
	arrayDepth := 0
	baseTypeName := typeName

	if strings.HasPrefix(typeName, "_") {
		arrayDepth = 1
		baseTypeName = strings.TrimPrefix(typeName, "_")
	}

	kind, ok := schema.PostgreSQLToInternal[baseTypeName]
	if !ok {
		// Check if it's a user-defined enum type
		if p.isEnumType(baseTypeName) {
			kind = schema.TypeEnum
			dt := schema.DataType{
				Kind:       kind,
				ArrayDepth: arrayDepth,
				EnumName:   baseTypeName,
			}
			return dt, nil
		}
		return schema.DataType{}, fmt.Errorf("unsupported PostgreSQL type: %q", typeName)
	}

	dt := schema.DataType{
		Kind:       kind,
		ArrayDepth: arrayDepth,
	}

	if typeMod.Valid && typeMod.Int32 != -1 {
		p.extractTypeMod(&dt, baseTypeName, typeMod.Int32)
	}

	return dt, nil
}

// isEnumType checks if the given type name is a user-defined enum type.
func (p *PostgresIntrospector) isEnumType(typeName string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
	defer cancel()

	query := `
		SELECT EXISTS(
			SELECT 1
			FROM pg_type t
			JOIN pg_namespace n ON n.oid = t.typnamespace
			WHERE t.typname = $1
			  AND t.typcategory = 'E'
			  AND n.nspname = $2
		)`

	var exists bool
	err := p.pool.QueryRow(ctx, query, typeName, p.schema).Scan(&exists)
	if err != nil {
		return false
	}

	return exists
}

// extractTypeMod extracts precision and scale from PostgreSQL atttypmod.
func (p *PostgresIntrospector) extractTypeMod(dt *schema.DataType, typeName string, typeMod int32) {
	switch typeName {
	case "varchar", "bpchar":
		dt.Precision = int(typeMod - 4)

	case "numeric", "decimal":
		adjustedMod := typeMod - 4
		precision := (adjustedMod >> 16) & 0xFFFF
		scale := adjustedMod & 0xFFFF
		dt.Precision = int(precision)
		dt.Scale = int(scale)
	}
}
