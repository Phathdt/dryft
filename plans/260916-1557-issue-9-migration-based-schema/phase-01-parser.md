# Phase 1: SQL Parser Foundation

**Status:** Not Started  
**Dependencies:** None  
**Estimated Effort:** 2-3 days  
**Risk Level:** High (SQL syntax complexity)

## Objective

Build SQL parser để extract schema definitions từ migration statements. Parse CREATE/ALTER/DROP cho tables, enums, indexes.

## Scope

### In Scope
- CREATE TABLE với columns, data types, constraints (PK, FK, CHECK, UNIQUE, NOT NULL)
- ALTER TABLE ADD COLUMN, DROP COLUMN, RENAME COLUMN, ALTER COLUMN
- DROP TABLE
- CREATE TYPE (enum), ALTER TYPE ADD VALUE, DROP TYPE  
- CREATE INDEX (BTree, GIN, GiST, Hash, partial WHERE clauses), DROP INDEX
- Quoted identifiers (`"table_name"`) và unquoted (table_name)
- PostgreSQL data types: INT, BIGINT, TEXT, VARCHAR, TIMESTAMP, UUID, BOOLEAN, JSONB, ARRAY

### Out of Scope
- Complex constraints (EXCLUDE, multi-column CHECK expressions)
- Views, triggers, functions, procedures
- Advanced ALTER TABLE (ADD CONSTRAINT, DROP CONSTRAINT - handle in later iteration)
- COMMENT ON statements (nice-to-have)
- Transaction control (BEGIN, COMMIT, ROLLBACK)

## Data Structures

```go
// internal/migration/parser.go

package migration

import (
	"github.com/phathdt/dryft/internal/schema"
)

// Statement represents a parsed SQL statement.
type Statement interface {
	Type() StatementType
}

type StatementType string

const (
	CreateTableStmt    StatementType = "CREATE_TABLE"
	DropTableStmt      StatementType = "DROP_TABLE"
	AlterTableStmt     StatementType = "ALTER_TABLE"
	CreateTypeStmt     StatementType = "CREATE_TYPE"
	DropTypeStmt       StatementType = "DROP_TYPE"
	AlterTypeStmt      StatementType = "ALTER_TYPE"
	CreateIndexStmt    StatementType = "CREATE_INDEX"
	DropIndexStmt      StatementType = "DROP_INDEX"
)

// CreateTable represents CREATE TABLE statement.
type CreateTable struct {
	Name        string
	Columns     []ColumnDef
	Constraints []TableConstraint
	IfNotExists bool
}

// ColumnDef represents column definition.
type ColumnDef struct {
	Name       string
	Type       string           // Raw SQL type: "TEXT", "INTEGER", "VARCHAR(255)"
	Nullable   bool
	Default    *string
	Constraints []ColumnConstraint
}

// TableConstraint represents table-level constraints.
type TableConstraint struct {
	Type       ConstraintType
	Name       string
	Columns    []string
	RefTable   string   // For FK
	RefColumns []string // For FK
	OnDelete   string   // CASCADE, SET NULL, etc.
	OnUpdate   string
	CheckExpr  string   // For CHECK
}

type ConstraintType string

const (
	PrimaryKeyConstraint ConstraintType = "PRIMARY_KEY"
	ForeignKeyConstraint ConstraintType = "FOREIGN_KEY"
	UniqueConstraint     ConstraintType = "UNIQUE"
	CheckConstraint      ConstraintType = "CHECK"
)

// AlterTable represents ALTER TABLE statement.
type AlterTable struct {
	Table  string
	Action AlterAction
}

type AlterAction interface {
	ActionType() AlterActionType
}

type AlterActionType string

const (
	AddColumnAction    AlterActionType = "ADD_COLUMN"
	DropColumnAction   AlterActionType = "DROP_COLUMN"
	RenameColumnAction AlterActionType = "RENAME_COLUMN"
	AlterColumnAction  AlterActionType = "ALTER_COLUMN"
)

// Parser parses SQL statements.
type Parser struct{}

// NewParser creates a new SQL parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a SQL statement.
func (p *Parser) Parse(sql string) (Statement, error) {
	// Implementation
}
```

## Implementation Steps

### 1. Setup Parser Structure
- Create `internal/migration/parser.go`
- Define AST types (Statement interfaces, concrete types)
- Define error types

### 2. Tokenizer/Lexer
Simple token-based approach (không cần full lexer):
- Split by whitespace, preserve quoted strings
- Handle SQL keywords (case-insensitive)
- Track parentheses nesting

### 3. CREATE TABLE Parser
```
CREATE TABLE [IF NOT EXISTS] table_name (
  column_name type [constraints],
  ...
  [table_constraints]
)
```

**Test cases:**
```sql
CREATE TABLE users (id SERIAL PRIMARY KEY, email TEXT NOT NULL);
CREATE TABLE "user_profiles" (user_id INTEGER REFERENCES users(id));
CREATE TABLE posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);
```

### 4. ALTER TABLE Parser
```
ALTER TABLE table_name
  ADD COLUMN column_name type [constraints]
  | DROP COLUMN column_name
  | RENAME COLUMN old_name TO new_name
  | ALTER COLUMN column_name ...
```

**Test cases:**
```sql
ALTER TABLE users ADD COLUMN bio TEXT;
ALTER TABLE users DROP COLUMN legacy_field;
ALTER TABLE users RENAME COLUMN user_name TO username;
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
```

### 5. DROP TABLE Parser
```
DROP TABLE [IF EXISTS] table_name [CASCADE]
```

### 6. Enum Parsers
```
CREATE TYPE type_name AS ENUM ('value1', 'value2');
ALTER TYPE type_name ADD VALUE 'value3';
DROP TYPE type_name;
```

### 7. Index Parsers
```
CREATE [UNIQUE] INDEX [CONCURRENTLY] index_name 
  ON table_name USING method (columns) [WHERE condition];
DROP INDEX [CONCURRENTLY] [IF EXISTS] index_name;
```

**Test cases:**
```sql
CREATE INDEX idx_users_email ON users(email);
CREATE UNIQUE INDEX idx_users_username ON users(username);
CREATE INDEX idx_posts_created ON posts USING btree(created_at);
CREATE INDEX idx_posts_content_gin ON posts USING gin(content);
CREATE INDEX idx_active_users ON users(email) WHERE active = true;
```

## Type Mapping

Map SQL types → `schema.TypeKind`:

```go
var typeMapping = map[string]schema.TypeKind{
	"INTEGER":   schema.TypeInt32,
	"INT":       schema.TypeInt32,
	"BIGINT":    schema.TypeInt64,
	"SERIAL":    schema.TypeInt32,
	"BIGSERIAL": schema.TypeInt64,
	"TEXT":      schema.TypeText,
	"VARCHAR":   schema.TypeText,
	"UUID":      schema.TypeUUID,
	"BOOLEAN":   schema.TypeBoolean,
	"BOOL":      schema.TypeBoolean,
	"TIMESTAMP": schema.TypeTimestamp,
	"TIMESTAMPTZ": schema.TypeTimestampTZ,
	"DATE":      schema.TypeDate,
	"JSONB":     schema.TypeJSONB,
	"JSON":      schema.TypeJSON,
	// Add more as needed
}
```

## Testing

### Unit Tests (`parser_test.go`)

```go
func TestParser_CreateTable(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    *CreateTable
		wantErr bool
	}{
		{
			name: "simple table",
			sql:  "CREATE TABLE users (id SERIAL PRIMARY KEY, email TEXT NOT NULL)",
			want: &CreateTable{
				Name: "users",
				Columns: []ColumnDef{
					{Name: "id", Type: "SERIAL", Nullable: false},
					{Name: "email", Type: "TEXT", Nullable: false},
				},
			},
		},
		{
			name: "quoted identifiers",
			sql:  `CREATE TABLE "user_profiles" ("user_id" INTEGER)`,
			want: &CreateTable{
				Name: "user_profiles",
				Columns: []ColumnDef{
					{Name: "user_id", Type: "INTEGER", Nullable: true},
				},
			},
		},
		// More test cases...
	}
}

func TestParser_AlterTable(t *testing.T) {
	// Test ALTER TABLE variants
}

func TestParser_CreateEnum(t *testing.T) {
	// Test CREATE TYPE enum
}

func TestParser_CreateIndex(t *testing.T) {
	// Test CREATE INDEX variants
}
```

### Edge Cases
- Case sensitivity: `CREATE table`, `create TABLE`, `Create Table`
- Quoted vs unquoted identifiers
- Complex types: `VARCHAR(255)`, `DECIMAL(10,2)`, `TEXT[]`
- Multi-line statements with comments
- Trailing semicolons, whitespace
- Reserved keywords as identifiers (quoted)

## Error Handling

```go
type ParseError struct {
	SQL     string
	Message string
	Pos     int // Position in SQL string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at position %d: %s\nSQL: %s", e.Pos, e.Message, e.SQL)
}
```

## Files to Create

```
internal/migration/
├── parser.go          # Parser implementation
├── parser_test.go     # Unit tests
├── types.go           # AST type definitions (optional split)
└── testdata/          # Test SQL files
    ├── create_table.sql
    ├── alter_table.sql
    └── indexes.sql
```

## Validation

**Done When:**
- [ ] All parser functions implemented
- [ ] Type mapping complete for common PostgreSQL types
- [ ] Unit tests pass (>90% coverage)
- [ ] Edge cases handled (quoted identifiers, case insensitivity)
- [ ] Error messages descriptive
- [ ] Code reviewed

## Rollback Plan

Parser isolated trong `internal/migration/` package - không affect existing code. Có thể revert toàn bộ phase nếu approach không khả thi.

## Notes

- Không cần perfect SQL parser - chỉ cần parse DDL statements dryft generates
- Có thể dùng string parsing thay vì formal lexer/parser (simpler, sufficient)
- Consider library như `pingcap/parser` nếu complexity tăng, nhưng prefer zero-dependency initially
