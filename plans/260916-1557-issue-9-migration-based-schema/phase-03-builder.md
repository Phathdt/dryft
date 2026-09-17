# Phase 3: Schema Builder

**Status:** ✅ Completed (2026-09-16)  
**Dependencies:** Phase 1 (Parser), Phase 2 (Loader)  
**Estimated Effort:** 2 days  
**Risk Level:** High (stateful complexity)

## Objective

Build `schema.Schema` từ parsed SQL statements bằng cách apply statements chronologically và maintain state.

## Scope

### In Scope
- Apply CREATE TABLE → add table to schema
- Apply ALTER TABLE ADD COLUMN → add column to existing table
- Apply ALTER TABLE DROP COLUMN → remove column from existing table
- Apply ALTER TABLE RENAME COLUMN → rename column
- Apply DROP TABLE → remove table from schema
- Apply CREATE TYPE (enum), DROP TYPE
- Apply CREATE INDEX, DROP INDEX
- Handle dependencies (CREATE TYPE before CREATE TABLE using that type)
- Validate operations (cannot DROP non-existent table)

### Out of Scope
- Complex constraint validation
- Circular dependency detection (assume migrations are valid)
- Schema versioning/snapshotting
- Rollback/undo operations

## Data Structures

```go
// internal/migration/builder.go

package migration

import (
	"fmt"
	"github.com/phathdt/dryft/internal/schema"
)

// SchemaBuilder builds a schema from SQL statements.
type SchemaBuilder struct {
	tables map[string]*schema.Table  // Keyed by table name
	enums  map[string]*schema.Enum   // Keyed by type name
}

// NewSchemaBuilder creates a new schema builder.
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		tables: make(map[string]*schema.Table),
		enums:  make(map[string]*schema.Enum),
	}
}

// Apply applies a parsed statement to the schema state.
func (b *SchemaBuilder) Apply(stmt Statement) error {
	switch s := stmt.(type) {
	case *CreateTable:
		return b.applyCreateTable(s)
	case *DropTable:
		return b.applyDropTable(s)
	case *AlterTable:
		return b.applyAlterTable(s)
	case *CreateType:
		return b.applyCreateType(s)
	case *DropType:
		return b.applyDropType(s)
	case *CreateIndex:
		return b.applyCreateIndex(s)
	case *DropIndex:
		return b.applyDropIndex(s)
	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// Build returns the final schema.
func (b *SchemaBuilder) Build() *schema.Schema {
	// Convert maps to slices
	tables := make([]schema.Table, 0, len(b.tables))
	for _, t := range b.tables {
		tables = append(tables, *t)
	}

	enums := make([]schema.Enum, 0, len(b.enums))
	for _, e := range b.enums {
		enums = append(enums, *e)
	}

	return &schema.Schema{
		Tables: tables,
		Enums:  enums,
	}
}
```

## Implementation Steps

### 1. CREATE TABLE

```go
func (b *SchemaBuilder) applyCreateTable(stmt *CreateTable) error {
	if _, exists := b.tables[stmt.Name]; exists {
		if stmt.IfNotExists {
			return nil // Skip silently
		}
		return fmt.Errorf("table %s already exists", stmt.Name)
	}

	table := &schema.Table{
		Name:        stmt.Name,
		Columns:     []schema.Column{},
		Indexes:     []schema.Index{},
		ForeignKeys: []schema.ForeignKey{},
		Constraints: []schema.Constraint{},
	}

	// Convert ColumnDef → schema.Column
	for _, colDef := range stmt.Columns {
		col, err := b.convertColumn(colDef)
		if err != nil {
			return fmt.Errorf("failed to convert column %s: %w", colDef.Name, err)
		}
		table.Columns = append(table.Columns, col)
	}

	// Convert TableConstraint → schema constraints
	for _, constraint := range stmt.Constraints {
		if err := b.applyTableConstraint(table, constraint); err != nil {
			return err
		}
	}

	b.tables[stmt.Name] = table
	return nil
}

func (b *SchemaBuilder) convertColumn(def ColumnDef) (schema.Column, error) {
	typeKind, err := mapSQLType(def.Type)
	if err != nil {
		return schema.Column{}, err
	}

	col := schema.Column{
		Name:     def.Name,
		Type:     typeKind,
		Nullable: def.Nullable,
	}

	if def.Default != nil {
		col.Default = def.Default
	}

	return col, nil
}

func mapSQLType(sqlType string) (schema.TypeKind, error) {
	// Normalize: "VARCHAR(255)" → "VARCHAR", "TIMESTAMP WITH TIME ZONE" → "TIMESTAMPTZ"
	normalized := strings.ToUpper(strings.TrimSpace(sqlType))
	
	// Remove parentheses content
	if idx := strings.Index(normalized, "("); idx != -1 {
		normalized = normalized[:idx]
	}

	// Handle special cases
	normalized = strings.ReplaceAll(normalized, " WITH TIME ZONE", "TZ")
	normalized = strings.ReplaceAll(normalized, " WITHOUT TIME ZONE", "")

	switch normalized {
	case "INTEGER", "INT", "INT4":
		return schema.TypeInt32, nil
	case "BIGINT", "INT8":
		return schema.TypeInt64, nil
	case "SERIAL":
		return schema.TypeInt32, nil
	case "BIGSERIAL":
		return schema.TypeInt64, nil
	case "TEXT":
		return schema.TypeText, nil
	case "VARCHAR", "CHARACTER VARYING":
		return schema.TypeText, nil
	case "UUID":
		return schema.TypeUUID, nil
	case "BOOLEAN", "BOOL":
		return schema.TypeBoolean, nil
	case "TIMESTAMP", "TIMESTAMPTZ":
		return schema.TypeTimestamp, nil
	case "DATE":
		return schema.TypeDate, nil
	case "JSONB":
		return schema.TypeJSONB, nil
	case "JSON":
		return schema.TypeJSON, nil
	default:
		// Check if it's an enum (registered type)
		if _, exists := b.enums[sqlType]; exists {
			return schema.TypeEnum, nil
		}
		return "", fmt.Errorf("unsupported SQL type: %s", sqlType)
	}
}
```

### 2. ALTER TABLE

```go
func (b *SchemaBuilder) applyAlterTable(stmt *AlterTable) error {
	table, exists := b.tables[stmt.Table]
	if !exists {
		return fmt.Errorf("table %s does not exist", stmt.Table)
	}

	switch action := stmt.Action.(type) {
	case *AddColumn:
		return b.applyAddColumn(table, action)
	case *DropColumn:
		return b.applyDropColumn(table, action)
	case *RenameColumn:
		return b.applyRenameColumn(table, action)
	case *AlterColumn:
		return b.applyAlterColumn(table, action)
	default:
		return fmt.Errorf("unsupported alter action: %T", action)
	}
}

func (b *SchemaBuilder) applyAddColumn(table *schema.Table, action *AddColumn) error {
	// Check if column already exists
	for _, col := range table.Columns {
		if col.Name == action.Column.Name {
			if action.IfNotExists {
				return nil
			}
			return fmt.Errorf("column %s already exists in table %s", action.Column.Name, table.Name)
		}
	}

	col, err := b.convertColumn(action.Column)
	if err != nil {
		return err
	}

	table.Columns = append(table.Columns, col)
	return nil
}

func (b *SchemaBuilder) applyDropColumn(table *schema.Table, action *DropColumn) error {
	for i, col := range table.Columns {
		if col.Name == action.ColumnName {
			// Remove column
			table.Columns = append(table.Columns[:i], table.Columns[i+1:]...)
			return nil
		}
	}

	if action.IfExists {
		return nil
	}
	return fmt.Errorf("column %s does not exist in table %s", action.ColumnName, table.Name)
}

func (b *SchemaBuilder) applyRenameColumn(table *schema.Table, action *RenameColumn) error {
	for i := range table.Columns {
		if table.Columns[i].Name == action.OldName {
			table.Columns[i].Name = action.NewName
			return nil
		}
	}
	return fmt.Errorf("column %s does not exist in table %s", action.OldName, table.Name)
}
```

### 3. DROP TABLE

```go
func (b *SchemaBuilder) applyDropTable(stmt *DropTable) error {
	if _, exists := b.tables[stmt.Name]; !exists {
		if stmt.IfExists {
			return nil
		}
		return fmt.Errorf("table %s does not exist", stmt.Name)
	}

	delete(b.tables, stmt.Name)
	return nil
}
```

### 4. Enum Operations

```go
func (b *SchemaBuilder) applyCreateType(stmt *CreateType) error {
	if _, exists := b.enums[stmt.Name]; exists {
		return fmt.Errorf("type %s already exists", stmt.Name)
	}

	enum := &schema.Enum{
		Name:   stmt.Name,
		Values: []schema.EnumValue{},
	}

	for i, val := range stmt.Values {
		enum.Values = append(enum.Values, schema.EnumValue{
			Label: val,
			Order: i,
		})
	}

	b.enums[stmt.Name] = enum
	return nil
}

func (b *SchemaBuilder) applyDropType(stmt *DropType) error {
	if _, exists := b.enums[stmt.Name]; !exists {
		if stmt.IfExists {
			return nil
		}
		return fmt.Errorf("type %s does not exist", stmt.Name)
	}

	delete(b.enums, stmt.Name)
	return nil
}
```

### 5. Index Operations

```go
func (b *SchemaBuilder) applyCreateIndex(stmt *CreateIndex) error {
	table, exists := b.tables[stmt.Table]
	if !exists {
		return fmt.Errorf("table %s does not exist", stmt.Table)
	}

	// Check if index already exists
	for _, idx := range table.Indexes {
		if idx.Name == stmt.Name {
			return fmt.Errorf("index %s already exists on table %s", stmt.Name, stmt.Table)
		}
	}

	index := schema.Index{
		Name:    stmt.Name,
		Columns: stmt.Columns,
		Unique:  stmt.Unique,
		Method:  stmt.Method, // btree, gin, gist, hash
		Where:   stmt.Where,  // Partial index condition
	}

	table.Indexes = append(table.Indexes, index)
	return nil
}

func (b *SchemaBuilder) applyDropIndex(stmt *DropIndex) error {
	// Index can be on any table - search all
	for _, table := range b.tables {
		for i, idx := range table.Indexes {
			if idx.Name == stmt.Name {
				table.Indexes = append(table.Indexes[:i], table.Indexes[i+1:]...)
				return nil
			}
		}
	}

	if stmt.IfExists {
		return nil
	}
	return fmt.Errorf("index %s does not exist", stmt.Name)
}
```

## Testing

### Unit Tests (`builder_test.go`)

```go
func TestSchemaBuilder_CreateTable(t *testing.T) {
	builder := NewSchemaBuilder()

	stmt := &CreateTable{
		Name: "users",
		Columns: []ColumnDef{
			{Name: "id", Type: "SERIAL", Nullable: false},
			{Name: "email", Type: "TEXT", Nullable: false},
		},
	}

	err := builder.Apply(stmt)
	require.NoError(t, err)

	schema := builder.Build()
	require.Len(t, schema.Tables, 1)
	assert.Equal(t, "users", schema.Tables[0].Name)
	assert.Len(t, schema.Tables[0].Columns, 2)
}

func TestSchemaBuilder_AlterTableAddColumn(t *testing.T) {
	builder := NewSchemaBuilder()

	// Create table first
	builder.Apply(&CreateTable{
		Name: "users",
		Columns: []ColumnDef{
			{Name: "id", Type: "INTEGER", Nullable: false},
		},
	})

	// Add column
	err := builder.Apply(&AlterTable{
		Table: "users",
		Action: &AddColumn{
			Column: ColumnDef{Name: "email", Type: "TEXT", Nullable: true},
		},
	})
	require.NoError(t, err)

	schema := builder.Build()
	table := schema.Tables[0]
	assert.Len(t, table.Columns, 2)
	assert.Equal(t, "email", table.Columns[1].Name)
}

func TestSchemaBuilder_DropColumn(t *testing.T) {
	builder := NewSchemaBuilder()

	builder.Apply(&CreateTable{
		Name: "users",
		Columns: []ColumnDef{
			{Name: "id", Type: "INTEGER"},
			{Name: "legacy", Type: "TEXT"},
		},
	})

	err := builder.Apply(&AlterTable{
		Table: "users",
		Action: &DropColumn{ColumnName: "legacy"},
	})
	require.NoError(t, err)

	schema := builder.Build()
	assert.Len(t, schema.Tables[0].Columns, 1)
	assert.Equal(t, "id", schema.Tables[0].Columns[0].Name)
}

func TestSchemaBuilder_DropTable(t *testing.T) {
	builder := NewSchemaBuilder()

	builder.Apply(&CreateTable{Name: "users"})
	builder.Apply(&DropTable{Name: "users"})

	schema := builder.Build()
	assert.Len(t, schema.Tables, 0)
}

func TestSchemaBuilder_CreateEnum(t *testing.T) {
	builder := NewSchemaBuilder()

	stmt := &CreateType{
		Name:   "user_role",
		Values: []string{"admin", "user", "guest"},
	}

	err := builder.Apply(stmt)
	require.NoError(t, err)

	schema := builder.Build()
	require.Len(t, schema.Enums, 1)
	assert.Equal(t, "user_role", schema.Enums[0].Name)
	assert.Len(t, schema.Enums[0].Values, 3)
}

func TestSchemaBuilder_ErrorHandling(t *testing.T) {
	t.Run("drop non-existent table", func(t *testing.T) {
		builder := NewSchemaBuilder()
		err := builder.Apply(&DropTable{Name: "nonexistent"})
		assert.Error(t, err)
	})

	t.Run("add column to non-existent table", func(t *testing.T) {
		builder := NewSchemaBuilder()
		err := builder.Apply(&AlterTable{
			Table:  "nonexistent",
			Action: &AddColumn{Column: ColumnDef{Name: "col", Type: "TEXT"}},
		})
		assert.Error(t, err)
	})

	t.Run("duplicate table", func(t *testing.T) {
		builder := NewSchemaBuilder()
		builder.Apply(&CreateTable{Name: "users"})
		err := builder.Apply(&CreateTable{Name: "users"})
		assert.Error(t, err)
	})
}
```

### Integration Test

```go
func TestSchemaBuilder_IncrementalMigrations(t *testing.T) {
	builder := NewSchemaBuilder()
	parser := NewParser()

	// Sequence of migrations
	migrations := []string{
		"CREATE TABLE users (id SERIAL PRIMARY KEY, email TEXT NOT NULL)",
		"ALTER TABLE users ADD COLUMN name TEXT",
		"CREATE TYPE user_role AS ENUM ('admin', 'user')",
		"ALTER TABLE users ADD COLUMN role user_role",
		"CREATE INDEX idx_users_email ON users(email)",
	}

	for i, sql := range migrations {
		stmt, err := parser.Parse(sql)
		require.NoError(t, err, "migration %d failed to parse", i)

		err = builder.Apply(stmt)
		require.NoError(t, err, "migration %d failed to apply", i)
	}

	schema := builder.Build()

	// Verify final state
	require.Len(t, schema.Tables, 1)
	table := schema.Tables[0]
	assert.Equal(t, "users", table.Name)
	assert.Len(t, table.Columns, 4) // id, email, name, role
	assert.Len(t, table.Indexes, 1)
	
	require.Len(t, schema.Enums, 1)
	assert.Equal(t, "user_role", schema.Enums[0].Name)
}
```

## Error Handling

```go
type BuildError struct {
	Statement string
	Cause     error
}

func (e *BuildError) Error() string {
	return fmt.Sprintf("failed to apply statement: %v\nStatement: %s", e.Cause, e.Statement)
}
```

## Files to Create

```
internal/migration/
├── builder.go
├── builder_test.go
└── type_mapping.go  # SQL type → schema.TypeKind mapping
```

## Validation

**Done When:**
- [x] All statement types handled
- [x] Type mapping complete
- [x] Incremental operations work correctly
- [x] Error handling for invalid operations
- [x] Unit tests pass (81.4% coverage for migration package)
- [x] Integration test with realistic migration sequence passes
- [x] State consistency verified

## Implementation Summary

**Files Created:**
- `internal/migration/builder.go` (12KB) - SchemaBuilder implementation
- `internal/migration/builder_test.go` (16KB) - Unit tests
- `internal/migration/builder_integration_test.go` (10KB) - Integration tests
- `internal/migration/type_mapping.go` (3.1KB) - SQL type mapping
- `internal/migration/type_mapping_test.go` (8.4KB) - Type mapping tests

**Test Results:**
- All unit tests pass (47 tests)
- All integration tests pass (4 scenarios)
- Package coverage: 81.4%
- builder.go coverage: 88.9% avg
- type_mapping.go coverage: 100%

**Features Implemented:**
- CREATE TABLE with columns, constraints, indexes
- ALTER TABLE (ADD/DROP/RENAME/ALTER COLUMN)
- DROP TABLE with IF EXISTS support
- CREATE/DROP TYPE (enum support)
- CREATE/DROP INDEX with all index types (btree, gin, gist, hash)
- Primary key (column-level and table-level)
- Foreign keys with referential actions
- Unique and CHECK constraints
- Comprehensive type mapping (all PostgreSQL types)
- Array types support
- Precision/scale handling for numeric types
- IF NOT EXISTS / IF EXISTS support

## Notes

- Builder là stateful - mỗi Apply() call mutates internal state
- Assume migrations are valid và well-ordered (dryft generates them)
- Performance: O(n) statements, O(1) lookup per table/enum - acceptable
- Type mapping cần handle PostgreSQL type variations (INT vs INTEGER, VARCHAR vs TEXT)
