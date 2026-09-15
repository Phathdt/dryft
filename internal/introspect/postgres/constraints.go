package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/phathdt/dryft/internal/schema"
)

// primaryKeyRow represents a row from the primary key query.
type primaryKeyRow struct {
	ConstraintName string
	Columns        []string
}

// foreignKeyRow represents a row from the foreign key query.
type foreignKeyRow struct {
	ConstraintName string
	Columns        []string
	RefTable       string
	RefColumns     []string
	OnDelete       string
	OnUpdate       string
}

// checkConstraintRow represents a row from the CHECK constraint query.
type checkConstraintRow struct {
	ConstraintName string
	Definition     string
}

// IntrospectPrimaryKey retrieves the primary key constraint for a table.
func (p *PostgresIntrospector) IntrospectPrimaryKey(ctx context.Context, schemaName, tableName string) (*schema.PrimaryKey, error) {
	query := `
		SELECT
			con.conname AS constraint_name,
			ARRAY(
				SELECT a.attname
				FROM pg_attribute a
				WHERE a.attrelid = con.conrelid AND a.attnum = ANY(con.conkey)
				ORDER BY array_position(con.conkey, a.attnum)
			) AS columns
		FROM pg_constraint con
		WHERE con.conrelid = ($1 || '.' || $2)::regclass
		  AND con.contype = 'p'`

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	var row primaryKeyRow
	err := p.pool.QueryRow(queryCtx, query, schemaName, tableName).Scan(&row.ConstraintName, &row.Columns)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No primary key is valid
			return nil, nil
		}
		return nil, fmt.Errorf("query primary key: %w", err)
	}

	return &schema.PrimaryKey{
		Name:    row.ConstraintName,
		Columns: row.Columns,
	}, nil
}

// IntrospectForeignKeys retrieves all foreign key constraints for a table.
func (p *PostgresIntrospector) IntrospectForeignKeys(ctx context.Context, schemaName, tableName string) ([]schema.ForeignKey, error) {
	query := `
		SELECT
			con.conname AS constraint_name,
			ARRAY(
				SELECT a.attname
				FROM pg_attribute a
				WHERE a.attrelid = con.conrelid AND a.attnum = ANY(con.conkey)
				ORDER BY array_position(con.conkey, a.attnum)
			) AS columns,
			ref_class.relname AS ref_table,
			ARRAY(
				SELECT a.attname
				FROM pg_attribute a
				WHERE a.attrelid = con.confrelid AND a.attnum = ANY(con.confkey)
				ORDER BY array_position(con.confkey, a.attnum)
			) AS ref_columns,
			con.confdeltype AS on_delete,
			con.confupdtype AS on_update
		FROM pg_constraint con
		JOIN pg_class ref_class ON ref_class.oid = con.confrelid
		WHERE con.conrelid = ($1 || '.' || $2)::regclass
		  AND con.contype = 'f'
		ORDER BY con.conname`

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	rows, err := p.pool.Query(queryCtx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("query foreign keys: %w", err)
	}
	defer rows.Close()

	var foreignKeys []schema.ForeignKey
	for rows.Next() {
		var row foreignKeyRow
		if err := rows.Scan(
			&row.ConstraintName,
			&row.Columns,
			&row.RefTable,
			&row.RefColumns,
			&row.OnDelete,
			&row.OnUpdate,
		); err != nil {
			return nil, fmt.Errorf("scan foreign key row: %w", err)
		}

		fk := schema.ForeignKey{
			Name:       row.ConstraintName,
			Columns:    row.Columns,
			RefTable:   row.RefTable,
			RefColumns: row.RefColumns,
			OnDelete:   mapReferentialAction(row.OnDelete),
			OnUpdate:   mapReferentialAction(row.OnUpdate),
		}

		foreignKeys = append(foreignKeys, fk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate foreign keys: %w", err)
	}

	return foreignKeys, nil
}

// IntrospectCheckConstraints retrieves all CHECK constraints for a table.
func (p *PostgresIntrospector) IntrospectCheckConstraints(ctx context.Context, schemaName, tableName string) ([]schema.Constraint, error) {
	query := `
		SELECT
			con.conname AS constraint_name,
			pg_get_constraintdef(con.oid) AS definition
		FROM pg_constraint con
		WHERE con.conrelid = ($1 || '.' || $2)::regclass
		  AND con.contype = 'c'
		ORDER BY con.conname`

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	rows, err := p.pool.Query(queryCtx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("query check constraints: %w", err)
	}
	defer rows.Close()

	var constraints []schema.Constraint
	for rows.Next() {
		var row checkConstraintRow
		if err := rows.Scan(&row.ConstraintName, &row.Definition); err != nil {
			return nil, fmt.Errorf("scan check constraint row: %w", err)
		}

		constraint := schema.Constraint{
			Name:       row.ConstraintName,
			Type:       schema.ConstraintCheck,
			Expression: row.Definition,
		}

		constraints = append(constraints, constraint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate check constraints: %w", err)
	}

	return constraints, nil
}

// IntrospectUniqueConstraints retrieves all UNIQUE constraints for a table.
func (p *PostgresIntrospector) IntrospectUniqueConstraints(ctx context.Context, schemaName, tableName string) ([]schema.Constraint, error) {
	query := `
		SELECT
			con.conname AS constraint_name,
			ARRAY(
				SELECT a.attname
				FROM pg_attribute a
				WHERE a.attrelid = con.conrelid AND a.attnum = ANY(con.conkey)
				ORDER BY array_position(con.conkey, a.attnum)
			) AS columns
		FROM pg_constraint con
		WHERE con.conrelid = ($1 || '.' || $2)::regclass
		  AND con.contype = 'u'
		ORDER BY con.conname`

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	rows, err := p.pool.Query(queryCtx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("query unique constraints: %w", err)
	}
	defer rows.Close()

	var constraints []schema.Constraint
	for rows.Next() {
		var name string
		var columns []string
		if err := rows.Scan(&name, &columns); err != nil {
			return nil, fmt.Errorf("scan unique constraint row: %w", err)
		}

		constraint := schema.Constraint{
			Name:    name,
			Type:    schema.ConstraintUnique,
			Columns: columns,
		}

		constraints = append(constraints, constraint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unique constraints: %w", err)
	}

	return constraints, nil
}

// mapReferentialAction converts PostgreSQL confdeltype/confupdtype to schema.ReferentialAction.
func mapReferentialAction(action string) schema.ReferentialAction {
	switch action {
	case "a":
		return schema.ActionNoAction
	case "r":
		return schema.ActionRestrict
	case "c":
		return schema.ActionCascade
	case "n":
		return schema.ActionSetNull
	case "d":
		return schema.ActionSetDefault
	default:
		return schema.ActionNone
	}
}
