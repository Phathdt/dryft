package migration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParser_EnumDefaultWithTypeCast tests parsing enum defaults with type casts
// This was a regression case (commit cae4e27) where 'active'::user_role failed
func TestParser_EnumDefaultWithTypeCast(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string // expected default value
	}{
		{
			name: "enum default with type cast",
			sql:  "CREATE TABLE users (status user_status DEFAULT 'active'::user_status)",
			want: "'active'::user_status",
		},
		{
			name: "enum default without type cast",
			sql:  "CREATE TABLE users (status user_status DEFAULT 'active')",
			want: "'active'",
		},
		{
			name: "enum default with double quotes",
			sql:  `CREATE TABLE users (status user_status DEFAULT "active"::user_status)`,
			want: `"active"::user_status`,
		},
		{
			name: "enum default with quoted type name",
			sql:  `CREATE TABLE users (status "UserStatus" DEFAULT 'pending'::"UserStatus")`,
			want: `'pending'::"UserStatus"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			require.NoError(t, err)

			ct, ok := stmt.(*CreateTable)
			require.True(t, ok)
			require.Len(t, ct.Columns, 1)

			col := ct.Columns[0]
			require.NotNil(t, col.Default, "default should not be nil")
			assert.Equal(t, tt.want, *col.Default,
				"default value should preserve type cast syntax")
		})
	}
}

// TestParser_ComplexCheckExpressions tests CHECK constraints with complex expressions
func TestParser_ComplexCheckExpressions(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		checkExpr string
	}{
		{
			name:      "simple comparison",
			sql:       "CREATE TABLE products (price NUMERIC CHECK (price > 0))",
			checkExpr: "price > 0",
		},
		{
			name:      "compound expression with AND",
			sql:       "CREATE TABLE products (quantity INTEGER CHECK (quantity >= 0 AND quantity <= 1000))",
			checkExpr: "quantity >= 0 AND quantity <= 1000",
		},
		{
			name:      "expression with parentheses",
			sql:       "CREATE TABLE orders (total NUMERIC CHECK ((total - discount) >= 0))",
			checkExpr: "(total - discount) >= 0",
		},
		{
			name:      "expression with string comparison",
			sql:       "CREATE TABLE users (email TEXT CHECK (email LIKE '%@%'))",
			checkExpr: "email LIKE '%@%'",
		},
		{
			name:      "expression with IN clause",
			sql:       "CREATE TABLE items (status TEXT CHECK (status IN ('active', 'inactive', 'pending')))",
			checkExpr: "status IN ('active', 'inactive', 'pending')",
		},
		{
			name:      "expression with NOT",
			sql:       "CREATE TABLE data (value INTEGER CHECK (value IS NOT NULL))",
			checkExpr: "value IS NOT NULL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			require.NoError(t, err)

			ct, ok := stmt.(*CreateTable)
			require.True(t, ok)
			require.Len(t, ct.Columns, 1)

			col := ct.Columns[0]
			require.Len(t, col.Constraints, 1)
			require.Equal(t, CheckConstraint, col.Constraints[0].Type)
			assert.Equal(t, tt.checkExpr, col.Constraints[0].CheckExpr)
		})
	}
}

// TestParser_CompositeForeignKeys tests multi-column foreign keys
func TestParser_CompositeForeignKeys(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		columns    []string
		refColumns []string
	}{
		{
			name: "composite FK two columns",
			sql: `CREATE TABLE order_items (
				order_id INTEGER,
				item_id INTEGER,
				FOREIGN KEY (order_id, item_id) REFERENCES orders(id, item_id)
			)`,
			columns:    []string{"order_id", "item_id"},
			refColumns: []string{"id", "item_id"},
		},
		{
			name: "composite FK with ON DELETE CASCADE",
			sql: `CREATE TABLE user_permissions (
				user_id INTEGER,
				role_id INTEGER,
				FOREIGN KEY (user_id, role_id) REFERENCES user_roles(user_id, role_id) ON DELETE CASCADE
			)`,
			columns:    []string{"user_id", "role_id"},
			refColumns: []string{"user_id", "role_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			require.NoError(t, err)

			ct, ok := stmt.(*CreateTable)
			require.True(t, ok)
			require.Len(t, ct.Constraints, 1)

			fk := ct.Constraints[0]
			assert.Equal(t, ForeignKeyConstraint, fk.Type)
			assert.Equal(t, tt.columns, fk.Columns)
			assert.Equal(t, tt.refColumns, fk.RefColumns)
		})
	}
}

// TestParser_QuotedIdentifiersWithSpecialChars tests identifiers with special characters
func TestParser_QuotedIdentifiersWithSpecialChars(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		tableName string
		colName   string
	}{
		{
			name:      "table with underscore",
			sql:       `CREATE TABLE "user_profiles" ("user_id" INTEGER)`,
			tableName: "user_profiles",
			colName:   "user_id",
		},
		{
			name:      "table with hyphen",
			sql:       `CREATE TABLE "user-profiles" ("user-id" INTEGER)`,
			tableName: "user-profiles",
			colName:   "user-id",
		},
		{
			name:      "table with space",
			sql:       `CREATE TABLE "User Profiles" ("User ID" INTEGER)`,
			tableName: "User Profiles",
			colName:   "User ID",
		},
		{
			name:      "unicode characters",
			sql:       `CREATE TABLE "用户表" ("用户ID" INTEGER)`,
			tableName: "用户表",
			colName:   "用户ID",
		},
		{
			name:      "mixed case preserved",
			sql:       `CREATE TABLE "UserProfiles" ("UserId" INTEGER)`,
			tableName: "UserProfiles",
			colName:   "UserId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			require.NoError(t, err)

			ct, ok := stmt.(*CreateTable)
			require.True(t, ok)
			assert.Equal(t, tt.tableName, ct.Name)
			require.Len(t, ct.Columns, 1)
			assert.Equal(t, tt.colName, ct.Columns[0].Name)
		})
	}
}

// TestParser_DefaultValuesWithExpressions tests various default value expressions
func TestParser_DefaultValuesWithExpressions(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		colName string
		want    string
	}{
		{
			name:    "NOW() function",
			sql:     "CREATE TABLE logs (created_at TIMESTAMP DEFAULT NOW())",
			colName: "created_at",
			want:    "NOW()",
		},
		{
			name:    "CURRENT_TIMESTAMP",
			sql:     "CREATE TABLE logs (created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)",
			colName: "created_at",
			want:    "CURRENT_TIMESTAMP",
		},
		{
			name:    "uuid_generate_v4()",
			sql:     "CREATE TABLE users (id UUID DEFAULT uuid_generate_v4())",
			colName: "id",
			want:    "uuid_generate_v4()",
		},
		{
			name:    "gen_random_uuid()",
			sql:     "CREATE TABLE users (id UUID DEFAULT gen_random_uuid())",
			colName: "id",
			want:    "gen_random_uuid()",
		},
		{
			name:    "string literal",
			sql:     "CREATE TABLE users (country TEXT DEFAULT 'US')",
			colName: "country",
			want:    "'US'",
		},
		{
			name:    "boolean literal",
			sql:     "CREATE TABLE users (active BOOLEAN DEFAULT true)",
			colName: "active",
			want:    "true",
		},
		{
			name:    "numeric literal",
			sql:     "CREATE TABLE products (quantity INTEGER DEFAULT 0)",
			colName: "quantity",
			want:    "0",
		},
		{
			name:    "negative number",
			sql:     "CREATE TABLE accounts (balance NUMERIC DEFAULT -100.50)",
			colName: "balance",
			want:    "-100.50",
		},
		{
			name:    "NULL default",
			sql:     "CREATE TABLE users (bio TEXT DEFAULT NULL)",
			colName: "bio",
			want:    "NULL",
		},
		{
			name:    "expression with operator",
			sql:     "CREATE TABLE items (expires_at TIMESTAMP DEFAULT (NOW() + INTERVAL '7 days'))",
			colName: "expires_at",
			want:    "(NOW() + INTERVAL '7 days')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			require.NoError(t, err)

			ct, ok := stmt.(*CreateTable)
			require.True(t, ok)
			require.Len(t, ct.Columns, 1)

			col := ct.Columns[0]
			assert.Equal(t, tt.colName, col.Name)
			require.NotNil(t, col.Default)
			assert.Equal(t, tt.want, *col.Default)
		})
	}
}

// TestParser_PartialIndexes tests partial indexes with WHERE clauses
func TestParser_PartialIndexes(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		whereExpr string
	}{
		{
			name:      "simple WHERE clause",
			sql:       "CREATE INDEX idx_active_users ON users(email) WHERE active = true",
			whereExpr: "active = true",
		},
		{
			name:      "WHERE with IS NOT NULL",
			sql:       "CREATE INDEX idx_verified_emails ON users(email) WHERE email IS NOT NULL",
			whereExpr: "email IS NOT NULL",
		},
		{
			name:      "WHERE with compound condition",
			sql:       "CREATE INDEX idx_premium_active ON users(id) WHERE status = 'premium' AND active = true",
			whereExpr: "status = 'premium' AND active = true",
		},
		{
			name:      "WHERE with parentheses",
			sql:       "CREATE INDEX idx_filtered ON items(id) WHERE (deleted_at IS NULL)",
			whereExpr: "(deleted_at IS NULL)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			require.NoError(t, err)

			idx, ok := stmt.(*CreateIndex)
			require.True(t, ok)
			assert.Equal(t, tt.whereExpr, idx.Where)
		})
	}
}

// TestParser_MultipleConstraints tests columns with multiple constraints
func TestParser_MultipleConstraints(t *testing.T) {
	sql := "CREATE TABLE users (email TEXT NOT NULL UNIQUE)"

	p := NewParser()
	stmt, err := p.Parse(sql)
	require.NoError(t, err)

	ct, ok := stmt.(*CreateTable)
	require.True(t, ok)
	require.Len(t, ct.Columns, 1)

	col := ct.Columns[0]
	assert.Equal(t, "email", col.Name)
	assert.False(t, col.Nullable, "should be NOT NULL")
	require.Len(t, col.Constraints, 1)
	assert.Equal(t, UniqueConstraint, col.Constraints[0].Type)
}

// TestParser_ComplexDataTypes tests parsing of complex PostgreSQL data types
// Note: Parser may simplify some multi-word types (e.g., TIMESTAMP WITH TIME ZONE -> TIMESTAMP)
func TestParser_ComplexDataTypes(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		colType  string
		nullable bool
	}{
		{
			name:     "VARCHAR with length",
			sql:      "CREATE TABLE users (username VARCHAR(255))",
			colType:  "VARCHAR(255)",
			nullable: true,
		},
		{
			name:     "NUMERIC with precision",
			sql:      "CREATE TABLE products (price NUMERIC(10, 2))",
			colType:  "NUMERIC(10, 2)",
			nullable: true,
		},
		{
			name:     "TIMESTAMPTZ",
			sql:      "CREATE TABLE logs (created_at TIMESTAMPTZ)",
			colType:  "TIMESTAMPTZ",
			nullable: true,
		},
		{
			name:     "array type",
			sql:      "CREATE TABLE posts (tags TEXT[])",
			colType:  "TEXT[]",
			nullable: true,
		},
		{
			name:     "CHAR with length",
			sql:      "CREATE TABLE codes (code CHAR(10))",
			colType:  "CHAR(10)",
			nullable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			require.NoError(t, err)

			ct, ok := stmt.(*CreateTable)
			require.True(t, ok)
			require.Len(t, ct.Columns, 1)

			col := ct.Columns[0]
			assert.Equal(t, tt.colType, col.Type)
			assert.Equal(t, tt.nullable, col.Nullable)
		})
	}
}

// TestParser_AlterColumnWithTypeCast tests ALTER COLUMN with USING clause
func TestParser_AlterColumnWithTypeCast(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		setType string
	}{
		{
			name:    "simple type change",
			sql:     "ALTER TABLE users ALTER COLUMN age TYPE BIGINT",
			setType: "BIGINT",
		},
		{
			name:    "type change with USING",
			sql:     "ALTER TABLE users ALTER COLUMN age TYPE INTEGER USING age::INTEGER",
			setType: "INTEGER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			require.NoError(t, err)

			at, ok := stmt.(*AlterTable)
			require.True(t, ok)

			ac, ok := at.Action.(*AlterColumn)
			require.True(t, ok)
			assert.Equal(t, tt.setType, ac.SetType)
		})
	}
}

// TestParser_CaseSensitivityEdgeCases tests case handling edge cases
// Note: Current parser preserves case for unquoted identifiers (doesn't lowercase like PostgreSQL)
func TestParser_CaseSensitivityEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		tableName string
		colName   string
	}{
		{
			name:      "unquoted lowercase",
			sql:       "CREATE TABLE users (id INTEGER)",
			tableName: "users",
			colName:   "id",
		},
		{
			name:      "unquoted uppercase preserved by parser",
			sql:       "CREATE TABLE USERS (ID INTEGER)",
			tableName: "USERS", // Parser preserves as-is (PostgreSQL would lowercase)
			colName:   "ID",
		},
		{
			name:      "quoted preserves case",
			sql:       `CREATE TABLE "Users" ("Id" INTEGER)`,
			tableName: "Users",
			colName:   "Id",
		},
		{
			name:      "quoted all uppercase",
			sql:       `CREATE TABLE "USERS" ("ID" INTEGER)`,
			tableName: "USERS",
			colName:   "ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			require.NoError(t, err)

			ct, ok := stmt.(*CreateTable)
			require.True(t, ok)
			assert.Equal(t, tt.tableName, ct.Name)
			require.Len(t, ct.Columns, 1)
			assert.Equal(t, tt.colName, ct.Columns[0].Name)
		})
	}
}

// TestParser_EnumWithSpecialCharacters tests enum values with special characters
func TestParser_EnumWithSpecialCharacters(t *testing.T) {
	tests := []struct {
		name   string
		sql    string
		values []string
	}{
		{
			name:   "enum with spaces",
			sql:    "CREATE TYPE status AS ENUM ('in progress', 'not started', 'done')",
			values: []string{"in progress", "not started", "done"},
		},
		{
			name:   "enum with special chars",
			sql:    "CREATE TYPE grade AS ENUM ('A+', 'A-', 'B+', 'B-')",
			values: []string{"A+", "A-", "B+", "B-"},
		},
		{
			name:   "enum with numbers",
			sql:    "CREATE TYPE level AS ENUM ('level1', 'level2', 'level3')",
			values: []string{"level1", "level2", "level3"},
		},
		{
			name:   "enum with unicode",
			sql:    "CREATE TYPE mood AS ENUM ('😀', '😢', '😡')",
			values: []string{"😀", "😢", "😡"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			require.NoError(t, err)

			ct, ok := stmt.(*CreateType)
			require.True(t, ok)
			assert.Equal(t, tt.values, ct.Values)
		})
	}
}

// TestParser_IndexWithExpressions tests expression indexes
func TestParser_IndexWithExpressions(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{
			name:    "simple column index",
			sql:     "CREATE INDEX idx_email ON users(email)",
			wantErr: false,
		},
		{
			name:    "lowercase expression index",
			sql:     "CREATE INDEX idx_email_lower ON users(LOWER(email))",
			wantErr: false,
		},
		{
			name:    "composite with expression",
			sql:     "CREATE INDEX idx_name ON users(first_name, LOWER(last_name))",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.Parse(tt.sql)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
