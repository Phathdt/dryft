package postgres

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phathdt/dryft/internal/schema"
)

// PostgresIntrospector implements the Introspector interface for PostgreSQL databases.
type PostgresIntrospector struct {
	pool   *pgxpool.Pool
	schema string // Default schema name (usually "public")
}

// NewPostgresIntrospector creates a new PostgreSQL introspector.
// The DSN should be in the format: postgres://user:password@host:port/database
func NewPostgresIntrospector(ctx context.Context, dsn string) (*PostgresIntrospector, error) {
	pool, err := createPool(ctx, dsn)
	if err != nil {
		return nil, err
	}

	introspector := &PostgresIntrospector{
		pool:   pool,
		schema: "public",
	}

	// Check PostgreSQL version
	if err := introspector.checkVersion(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	// Warn about multi-schema setup
	if err := introspector.checkMultiSchema(ctx); err != nil {
		// Non-fatal, just log the warning
		log.Printf("warning: %v", err)
	}

	return introspector, nil
}

// Close releases all resources held by the introspector.
func (p *PostgresIntrospector) Close() error {
	if p.pool != nil {
		p.pool.Close()
	}
	return nil
}

// Introspect retrieves the complete database schema.
func (p *PostgresIntrospector) Introspect(ctx context.Context) (*schema.Schema, error) {
	// 1. List all tables
	tables, err := p.ListTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}

	// 2. List all enums
	enums, err := p.listEnums(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enums: %w", err)
	}

	// 3. Introspect each table fully
	var tableDefs []schema.Table
	for _, tableInfo := range tables {
		tableDef, err := p.IntrospectTable(ctx, tableInfo.Name)
		if err != nil {
			return nil, fmt.Errorf("introspect table %q: %w", tableInfo.Name, err)
		}
		tableDef.Comment = tableInfo.Comment
		tableDefs = append(tableDefs, *tableDef)
	}

	// 4. Build schema
	s := &schema.Schema{
		Tables: tableDefs,
		Enums:  enums,
	}

	// 5. Validate
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("introspected schema invalid: %w", err)
	}

	return s, nil
}

// IntrospectTable retrieves the complete schema for a specific table including all constraints and indexes.
func (p *PostgresIntrospector) IntrospectTable(ctx context.Context, tableName string) (*schema.Table, error) {
	schemaName := p.schema

	// Introspect columns
	columns, err := p.IntrospectColumns(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("introspect columns: %w", err)
	}

	// Introspect primary key
	pk, err := p.IntrospectPrimaryKey(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("introspect primary key: %w", err)
	}

	// Introspect foreign keys
	fks, err := p.IntrospectForeignKeys(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("introspect foreign keys: %w", err)
	}

	// Introspect indexes
	indexes, err := p.IntrospectIndexes(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("introspect indexes: %w", err)
	}

	// Introspect CHECK constraints
	checkConstraints, err := p.IntrospectCheckConstraints(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("introspect check constraints: %w", err)
	}

	// Introspect UNIQUE constraints
	uniqueConstraints, err := p.IntrospectUniqueConstraints(ctx, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("introspect unique constraints: %w", err)
	}

	// Combine all constraints
	constraints := append(checkConstraints, uniqueConstraints...)

	return &schema.Table{
		Name:        tableName,
		Columns:     columns,
		PrimaryKey:  pk,
		ForeignKeys: fks,
		Indexes:     indexes,
		Constraints: constraints,
	}, nil
}

// checkVersion verifies that PostgreSQL version is 12 or higher.
func (p *PostgresIntrospector) checkVersion(ctx context.Context) error {
	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	var versionStr string
	err := p.pool.QueryRow(queryCtx, "SELECT version()").Scan(&versionStr)
	if err != nil {
		return fmt.Errorf("failed to query PostgreSQL version: %w", err)
	}

	version, err := parsePostgresVersion(versionStr)
	if err != nil {
		return fmt.Errorf("failed to parse PostgreSQL version: %w", err)
	}

	if version < 12 {
		return fmt.Errorf("PostgreSQL 12+ required, found version %d", version)
	}

	return nil
}

// checkMultiSchema checks if there are schemas beyond 'public' and logs a warning.
func (p *PostgresIntrospector) checkMultiSchema(ctx context.Context) error {
	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	query := `
		SELECT COUNT(*)
		FROM pg_namespace
		WHERE nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast', 'public')
		  AND nspname NOT LIKE 'pg_temp_%'
		  AND nspname NOT LIKE 'pg_toast_temp_%'
	`

	var count int
	err := p.pool.QueryRow(queryCtx, query).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for multiple schemas: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("detected %d schema(s) beyond 'public' - only 'public' schema will be introspected", count)
	}

	return nil
}

// parsePostgresVersion extracts the major version number from PostgreSQL version string.
// Example: "PostgreSQL 14.5 (Ubuntu 14.5-1.pgdg20.04+1)" -> 14
func parsePostgresVersion(versionStr string) (int, error) {
	// Find "PostgreSQL" prefix
	parts := strings.Fields(versionStr)
	if len(parts) < 2 {
		return 0, fmt.Errorf("unexpected version format: %s", versionStr)
	}

	// Extract version number (second field)
	versionNum := parts[1]

	// Split by dot to get major version
	versionParts := strings.Split(versionNum, ".")
	if len(versionParts) == 0 {
		return 0, fmt.Errorf("unexpected version format: %s", versionStr)
	}

	majorVersion, err := strconv.Atoi(versionParts[0])
	if err != nil {
		return 0, fmt.Errorf("failed to parse major version: %w", err)
	}

	return majorVersion, nil
}
