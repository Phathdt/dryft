package migration

import (
	"testing"

	"github.com/phathdt/dryft/internal/schema"
)

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
					{
						Name:     "id",
						Type:     "SERIAL",
						Nullable: false,
						Constraints: []ColumnConstraint{
							{Type: PrimaryKeyConstraint},
						},
					},
					{
						Name:     "email",
						Type:     "TEXT",
						Nullable: false,
					},
				},
			},
		},
		{
			name: "quoted identifiers",
			sql:  `CREATE TABLE "user_profiles" ("user_id" INTEGER, "full_name" TEXT)`,
			want: &CreateTable{
				Name: "user_profiles",
				Columns: []ColumnDef{
					{Name: "user_id", Type: "INTEGER", Nullable: true},
					{Name: "full_name", Type: "TEXT", Nullable: true},
				},
			},
		},
		{
			name: "if not exists",
			sql:  "CREATE TABLE IF NOT EXISTS posts (id UUID PRIMARY KEY)",
			want: &CreateTable{
				Name:        "posts",
				IfNotExists: true,
				Columns: []ColumnDef{
					{
						Name:     "id",
						Type:     "UUID",
						Nullable: false,
						Constraints: []ColumnConstraint{
							{Type: PrimaryKeyConstraint},
						},
					},
				},
			},
		},
		{
			name: "with default values",
			sql:  "CREATE TABLE logs (id SERIAL, created_at TIMESTAMP DEFAULT NOW(), active BOOLEAN DEFAULT true)",
			want: &CreateTable{
				Name: "logs",
				Columns: []ColumnDef{
					{Name: "id", Type: "SERIAL", Nullable: true},
					{Name: "created_at", Type: "TIMESTAMP", Nullable: true, Default: stringPtr("NOW()")},
					{Name: "active", Type: "BOOLEAN", Nullable: true, Default: stringPtr("true")},
				},
			},
		},
		{
			name: "foreign key column constraint",
			sql:  "CREATE TABLE posts (user_id INTEGER REFERENCES users(id))",
			want: &CreateTable{
				Name: "posts",
				Columns: []ColumnDef{
					{
						Name:     "user_id",
						Type:     "INTEGER",
						Nullable: true,
						Constraints: []ColumnConstraint{
							{
								Type:       ForeignKeyConstraint,
								RefTable:   "users",
								RefColumns: []string{"id"},
							},
						},
					},
				},
			},
		},
		{
			name: "foreign key with on delete cascade",
			sql:  "CREATE TABLE posts (user_id INTEGER REFERENCES users(id) ON DELETE CASCADE)",
			want: &CreateTable{
				Name: "posts",
				Columns: []ColumnDef{
					{
						Name:     "user_id",
						Type:     "INTEGER",
						Nullable: true,
						Constraints: []ColumnConstraint{
							{
								Type:       ForeignKeyConstraint,
								RefTable:   "users",
								RefColumns: []string{"id"},
								OnDelete:   "CASCADE",
							},
						},
					},
				},
			},
		},
		{
			name: "table-level primary key",
			sql:  "CREATE TABLE users (id INTEGER, email TEXT, PRIMARY KEY (id))",
			want: &CreateTable{
				Name: "users",
				Columns: []ColumnDef{
					{Name: "id", Type: "INTEGER", Nullable: true},
					{Name: "email", Type: "TEXT", Nullable: true},
				},
				Constraints: []TableConstraint{
					{
						Type:    PrimaryKeyConstraint,
						Columns: []string{"id"},
					},
				},
			},
		},
		{
			name: "composite primary key",
			sql:  "CREATE TABLE user_roles (user_id INTEGER, role_id INTEGER, PRIMARY KEY (user_id, role_id))",
			want: &CreateTable{
				Name: "user_roles",
				Columns: []ColumnDef{
					{Name: "user_id", Type: "INTEGER", Nullable: true},
					{Name: "role_id", Type: "INTEGER", Nullable: true},
				},
				Constraints: []TableConstraint{
					{
						Type:    PrimaryKeyConstraint,
						Columns: []string{"user_id", "role_id"},
					},
				},
			},
		},
		{
			name: "table-level foreign key",
			sql:  "CREATE TABLE posts (id INTEGER, user_id INTEGER, FOREIGN KEY (user_id) REFERENCES users(id))",
			want: &CreateTable{
				Name: "posts",
				Columns: []ColumnDef{
					{Name: "id", Type: "INTEGER", Nullable: true},
					{Name: "user_id", Type: "INTEGER", Nullable: true},
				},
				Constraints: []TableConstraint{
					{
						Type:       ForeignKeyConstraint,
						Columns:    []string{"user_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					},
				},
			},
		},
		{
			name: "unique constraint",
			sql:  "CREATE TABLE users (id INTEGER, email TEXT UNIQUE)",
			want: &CreateTable{
				Name: "users",
				Columns: []ColumnDef{
					{Name: "id", Type: "INTEGER", Nullable: true},
					{
						Name:     "email",
						Type:     "TEXT",
						Nullable: true,
						Constraints: []ColumnConstraint{
							{Type: UniqueConstraint},
						},
					},
				},
			},
		},
		{
			name: "check constraint",
			sql:  "CREATE TABLE products (id INTEGER, price NUMERIC CHECK (price > 0))",
			want: &CreateTable{
				Name: "products",
				Columns: []ColumnDef{
					{Name: "id", Type: "INTEGER", Nullable: true},
					{
						Name:     "price",
						Type:     "NUMERIC",
						Nullable: true,
						Constraints: []ColumnConstraint{
							{Type: CheckConstraint, CheckExpr: "price > 0"},
						},
					},
				},
			},
		},
		{
			name: "named constraint",
			sql:  "CREATE TABLE users (id INTEGER, email TEXT, CONSTRAINT users_pk PRIMARY KEY (id))",
			want: &CreateTable{
				Name: "users",
				Columns: []ColumnDef{
					{Name: "id", Type: "INTEGER", Nullable: true},
					{Name: "email", Type: "TEXT", Nullable: true},
				},
				Constraints: []TableConstraint{
					{
						Type:    PrimaryKeyConstraint,
						Name:    "users_pk",
						Columns: []string{"id"},
					},
				},
			},
		},
		{
			name: "varchar with length",
			sql:  "CREATE TABLE users (username VARCHAR(50))",
			want: &CreateTable{
				Name: "users",
				Columns: []ColumnDef{
					{Name: "username", Type: "VARCHAR(50)", Nullable: true},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got, ok := stmt.(*CreateTable)
			if !ok {
				t.Fatalf("expected *CreateTable, got %T", stmt)
			}

			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}

			if got.IfNotExists != tt.want.IfNotExists {
				t.Errorf("IfNotExists = %v, want %v", got.IfNotExists, tt.want.IfNotExists)
			}

			if len(got.Columns) != len(tt.want.Columns) {
				t.Errorf("Columns count = %d, want %d", len(got.Columns), len(tt.want.Columns))
			}

			for i := range tt.want.Columns {
				if i >= len(got.Columns) {
					break
				}
				compareColumnDef(t, got.Columns[i], tt.want.Columns[i])
			}

			if len(got.Constraints) != len(tt.want.Constraints) {
				t.Errorf("Constraints count = %d, want %d", len(got.Constraints), len(tt.want.Constraints))
			}

			for i := range tt.want.Constraints {
				if i >= len(got.Constraints) {
					break
				}
				compareTableConstraint(t, got.Constraints[i], tt.want.Constraints[i])
			}
		})
	}
}

func TestParser_DropTable(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    *DropTable
		wantErr bool
	}{
		{
			name: "simple drop",
			sql:  "DROP TABLE users",
			want: &DropTable{Name: "users"},
		},
		{
			name: "if exists",
			sql:  "DROP TABLE IF EXISTS users",
			want: &DropTable{Name: "users", IfExists: true},
		},
		{
			name: "cascade",
			sql:  "DROP TABLE users CASCADE",
			want: &DropTable{Name: "users", Cascade: true},
		},
		{
			name: "if exists cascade",
			sql:  "DROP TABLE IF EXISTS users CASCADE",
			want: &DropTable{Name: "users", IfExists: true, Cascade: true},
		},
		{
			name: "quoted table name",
			sql:  `DROP TABLE "user_profiles"`,
			want: &DropTable{Name: "user_profiles"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got, ok := stmt.(*DropTable)
			if !ok {
				t.Fatalf("expected *DropTable, got %T", stmt)
			}

			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.IfExists != tt.want.IfExists {
				t.Errorf("IfExists = %v, want %v", got.IfExists, tt.want.IfExists)
			}
			if got.Cascade != tt.want.Cascade {
				t.Errorf("Cascade = %v, want %v", got.Cascade, tt.want.Cascade)
			}
		})
	}
}

func TestParser_AlterTable(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    *AlterTable
		wantErr bool
	}{
		{
			name: "add column",
			sql:  "ALTER TABLE users ADD COLUMN bio TEXT",
			want: &AlterTable{
				Table: "users",
				Action: &AddColumn{
					Column: ColumnDef{Name: "bio", Type: "TEXT", Nullable: true},
				},
			},
		},
		{
			name: "add column with default",
			sql:  "ALTER TABLE users ADD COLUMN active BOOLEAN DEFAULT true",
			want: &AlterTable{
				Table: "users",
				Action: &AddColumn{
					Column: ColumnDef{Name: "active", Type: "BOOLEAN", Nullable: true, Default: stringPtr("true")},
				},
			},
		},
		{
			name: "drop column",
			sql:  "ALTER TABLE users DROP COLUMN bio",
			want: &AlterTable{
				Table:  "users",
				Action: &DropColumn{Name: "bio"},
			},
		},
		{
			name: "drop column if exists",
			sql:  "ALTER TABLE users DROP COLUMN IF EXISTS bio",
			want: &AlterTable{
				Table:  "users",
				Action: &DropColumn{Name: "bio", IfExists: true},
			},
		},
		{
			name: "drop column cascade",
			sql:  "ALTER TABLE users DROP COLUMN bio CASCADE",
			want: &AlterTable{
				Table:  "users",
				Action: &DropColumn{Name: "bio", Cascade: true},
			},
		},
		{
			name: "rename column",
			sql:  "ALTER TABLE users RENAME COLUMN user_name TO username",
			want: &AlterTable{
				Table:  "users",
				Action: &RenameColumn{OldName: "user_name", NewName: "username"},
			},
		},
		{
			name: "alter column set not null",
			sql:  "ALTER TABLE users ALTER COLUMN email SET NOT NULL",
			want: &AlterTable{
				Table:  "users",
				Action: &AlterColumn{Name: "email", SetNotNull: true},
			},
		},
		{
			name: "alter column drop not null",
			sql:  "ALTER TABLE users ALTER COLUMN bio DROP NOT NULL",
			want: &AlterTable{
				Table:  "users",
				Action: &AlterColumn{Name: "bio", DropNotNull: true},
			},
		},
		{
			name: "alter column set default",
			sql:  "ALTER TABLE users ALTER COLUMN active SET DEFAULT true",
			want: &AlterTable{
				Table:  "users",
				Action: &AlterColumn{Name: "active", SetDefault: stringPtr("true")},
			},
		},
		{
			name: "alter column drop default",
			sql:  "ALTER TABLE users ALTER COLUMN active DROP DEFAULT",
			want: &AlterTable{
				Table:  "users",
				Action: &AlterColumn{Name: "active", DropDefault: true},
			},
		},
		{
			name: "alter column type",
			sql:  "ALTER TABLE users ALTER COLUMN age TYPE INTEGER",
			want: &AlterTable{
				Table:  "users",
				Action: &AlterColumn{Name: "age", SetType: "INTEGER"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got, ok := stmt.(*AlterTable)
			if !ok {
				t.Fatalf("expected *AlterTable, got %T", stmt)
			}

			if got.Table != tt.want.Table {
				t.Errorf("Table = %v, want %v", got.Table, tt.want.Table)
			}

			compareAlterAction(t, got.Action, tt.want.Action)
		})
	}
}

func TestParser_CreateType(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    *CreateType
		wantErr bool
	}{
		{
			name: "simple enum",
			sql:  "CREATE TYPE status AS ENUM ('active', 'inactive', 'pending')",
			want: &CreateType{
				Name:   "status",
				Values: []string{"active", "inactive", "pending"},
			},
		},
		{
			name: "enum with double quotes",
			sql:  `CREATE TYPE status AS ENUM ("active", "inactive")`,
			want: &CreateType{
				Name:   "status",
				Values: []string{"active", "inactive"},
			},
		},
		{
			name: "quoted type name",
			sql:  `CREATE TYPE "user_status" AS ENUM ('active', 'inactive')`,
			want: &CreateType{
				Name:   "user_status",
				Values: []string{"active", "inactive"},
			},
		},
		{
			name: "single value",
			sql:  "CREATE TYPE role AS ENUM ('admin')",
			want: &CreateType{
				Name:   "role",
				Values: []string{"admin"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got, ok := stmt.(*CreateType)
			if !ok {
				t.Fatalf("expected *CreateType, got %T", stmt)
			}

			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}

			if len(got.Values) != len(tt.want.Values) {
				t.Errorf("Values count = %d, want %d", len(got.Values), len(tt.want.Values))
			}

			for i := range tt.want.Values {
				if i >= len(got.Values) {
					break
				}
				if got.Values[i] != tt.want.Values[i] {
					t.Errorf("Values[%d] = %v, want %v", i, got.Values[i], tt.want.Values[i])
				}
			}
		})
	}
}

func TestParser_DropType(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    *DropType
		wantErr bool
	}{
		{
			name: "simple drop",
			sql:  "DROP TYPE status",
			want: &DropType{Name: "status"},
		},
		{
			name: "if exists",
			sql:  "DROP TYPE IF EXISTS status",
			want: &DropType{Name: "status", IfExists: true},
		},
		{
			name: "cascade",
			sql:  "DROP TYPE status CASCADE",
			want: &DropType{Name: "status", Cascade: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got, ok := stmt.(*DropType)
			if !ok {
				t.Fatalf("expected *DropType, got %T", stmt)
			}

			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.IfExists != tt.want.IfExists {
				t.Errorf("IfExists = %v, want %v", got.IfExists, tt.want.IfExists)
			}
			if got.Cascade != tt.want.Cascade {
				t.Errorf("Cascade = %v, want %v", got.Cascade, tt.want.Cascade)
			}
		})
	}
}

func TestParser_AlterType(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    *AlterType
		wantErr bool
	}{
		{
			name: "add value",
			sql:  "ALTER TYPE status ADD VALUE 'archived'",
			want: &AlterType{
				Name:   "status",
				Action: &AddEnumValue{Value: "archived"},
			},
		},
		{
			name: "add value before",
			sql:  "ALTER TYPE status ADD VALUE 'new_status' BEFORE 'active'",
			want: &AlterType{
				Name: "status",
				Action: &AddEnumValue{
					Value:  "new_status",
					Before: "active",
				},
			},
		},
		{
			name: "add value after",
			sql:  "ALTER TYPE status ADD VALUE 'new_status' AFTER 'active'",
			want: &AlterType{
				Name: "status",
				Action: &AddEnumValue{
					Value: "new_status",
					After: "active",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got, ok := stmt.(*AlterType)
			if !ok {
				t.Fatalf("expected *AlterType, got %T", stmt)
			}

			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}

			wantAction, ok := tt.want.Action.(*AddEnumValue)
			if !ok {
				t.Fatalf("expected *AddEnumValue in want, got %T", tt.want.Action)
			}

			gotAction, ok := got.Action.(*AddEnumValue)
			if !ok {
				t.Fatalf("expected *AddEnumValue in got, got %T", got.Action)
			}

			if gotAction.Value != wantAction.Value {
				t.Errorf("Value = %v, want %v", gotAction.Value, wantAction.Value)
			}
			if gotAction.Before != wantAction.Before {
				t.Errorf("Before = %v, want %v", gotAction.Before, wantAction.Before)
			}
			if gotAction.After != wantAction.After {
				t.Errorf("After = %v, want %v", gotAction.After, wantAction.After)
			}
		})
	}
}

func TestParser_CreateIndex(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    *CreateIndex
		wantErr bool
	}{
		{
			name: "simple index",
			sql:  "CREATE INDEX idx_users_email ON users(email)",
			want: &CreateIndex{
				Name:   "idx_users_email",
				Table:  "users",
				Method: "btree",
				Columns: []IndexColumnDef{
					{Name: "email"},
				},
			},
		},
		{
			name: "unique index",
			sql:  "CREATE UNIQUE INDEX idx_users_username ON users(username)",
			want: &CreateIndex{
				Name:   "idx_users_username",
				Table:  "users",
				Unique: true,
				Method: "btree",
				Columns: []IndexColumnDef{
					{Name: "username"},
				},
			},
		},
		{
			name: "with method",
			sql:  "CREATE INDEX idx_posts_content ON posts USING gin(content)",
			want: &CreateIndex{
				Name:   "idx_posts_content",
				Table:  "posts",
				Method: "gin",
				Columns: []IndexColumnDef{
					{Name: "content"},
				},
			},
		},
		{
			name: "composite index",
			sql:  "CREATE INDEX idx_users_name ON users(first_name, last_name)",
			want: &CreateIndex{
				Name:   "idx_users_name",
				Table:  "users",
				Method: "btree",
				Columns: []IndexColumnDef{
					{Name: "first_name"},
					{Name: "last_name"},
				},
			},
		},
		{
			name: "with order",
			sql:  "CREATE INDEX idx_posts_created ON posts(created_at DESC)",
			want: &CreateIndex{
				Name:   "idx_posts_created",
				Table:  "posts",
				Method: "btree",
				Columns: []IndexColumnDef{
					{Name: "created_at", Order: "DESC"},
				},
			},
		},
		{
			name: "partial index",
			sql:  "CREATE INDEX idx_active_users ON users(email) WHERE active = true",
			want: &CreateIndex{
				Name:   "idx_active_users",
				Table:  "users",
				Method: "btree",
				Columns: []IndexColumnDef{
					{Name: "email"},
				},
				Where: "active = true",
			},
		},
		{
			name: "concurrently",
			sql:  "CREATE INDEX CONCURRENTLY idx_users_email ON users(email)",
			want: &CreateIndex{
				Name:         "idx_users_email",
				Table:        "users",
				Method:       "btree",
				Concurrently: true,
				Columns: []IndexColumnDef{
					{Name: "email"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got, ok := stmt.(*CreateIndex)
			if !ok {
				t.Fatalf("expected *CreateIndex, got %T", stmt)
			}

			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Table != tt.want.Table {
				t.Errorf("Table = %v, want %v", got.Table, tt.want.Table)
			}
			if got.Unique != tt.want.Unique {
				t.Errorf("Unique = %v, want %v", got.Unique, tt.want.Unique)
			}
			if got.Method != tt.want.Method {
				t.Errorf("Method = %v, want %v", got.Method, tt.want.Method)
			}
			if got.Concurrently != tt.want.Concurrently {
				t.Errorf("Concurrently = %v, want %v", got.Concurrently, tt.want.Concurrently)
			}
			if got.Where != tt.want.Where {
				t.Errorf("Where = %v, want %v", got.Where, tt.want.Where)
			}

			if len(got.Columns) != len(tt.want.Columns) {
				t.Errorf("Columns count = %d, want %d", len(got.Columns), len(tt.want.Columns))
			}

			for i := range tt.want.Columns {
				if i >= len(got.Columns) {
					break
				}
				if got.Columns[i].Name != tt.want.Columns[i].Name {
					t.Errorf("Columns[%d].Name = %v, want %v", i, got.Columns[i].Name, tt.want.Columns[i].Name)
				}
				if got.Columns[i].Order != tt.want.Columns[i].Order {
					t.Errorf("Columns[%d].Order = %v, want %v", i, got.Columns[i].Order, tt.want.Columns[i].Order)
				}
			}
		})
	}
}

func TestParser_DropIndex(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    *DropIndex
		wantErr bool
	}{
		{
			name: "simple drop",
			sql:  "DROP INDEX idx_users_email",
			want: &DropIndex{Name: "idx_users_email"},
		},
		{
			name: "if exists",
			sql:  "DROP INDEX IF EXISTS idx_users_email",
			want: &DropIndex{Name: "idx_users_email", IfExists: true},
		},
		{
			name: "cascade",
			sql:  "DROP INDEX idx_users_email CASCADE",
			want: &DropIndex{Name: "idx_users_email", Cascade: true},
		},
		{
			name: "concurrently",
			sql:  "DROP INDEX CONCURRENTLY idx_users_email",
			want: &DropIndex{Name: "idx_users_email", Concurrently: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got, ok := stmt.(*DropIndex)
			if !ok {
				t.Fatalf("expected *DropIndex, got %T", stmt)
			}

			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.IfExists != tt.want.IfExists {
				t.Errorf("IfExists = %v, want %v", got.IfExists, tt.want.IfExists)
			}
			if got.Cascade != tt.want.Cascade {
				t.Errorf("Cascade = %v, want %v", got.Cascade, tt.want.Cascade)
			}
			if got.Concurrently != tt.want.Concurrently {
				t.Errorf("Concurrently = %v, want %v", got.Concurrently, tt.want.Concurrently)
			}
		})
	}
}

func TestSQLTypeToSchemaType(t *testing.T) {
	tests := []struct {
		sqlType string
		want    schema.TypeKind
	}{
		{"INTEGER", schema.TypeInt32},
		{"INT", schema.TypeInt32},
		{"BIGINT", schema.TypeInt64},
		{"SERIAL", schema.TypeInt32},
		{"BIGSERIAL", schema.TypeInt64},
		{"TEXT", schema.TypeText},
		{"VARCHAR", schema.TypeVarChar},
		{"VARCHAR(255)", schema.TypeVarChar},
		{"CHAR", schema.TypeChar},
		{"UUID", schema.TypeUUID},
		{"BOOLEAN", schema.TypeBool},
		{"BOOL", schema.TypeBool},
		{"REAL", schema.TypeFloat32},
		{"DOUBLE PRECISION", schema.TypeFloat64},
		{"NUMERIC", schema.TypeNumeric},
		{"DECIMAL", schema.TypeNumeric},
		{"TIMESTAMP", schema.TypeTimestamp},
		{"TIMESTAMPTZ", schema.TypeTimestampTZ},
		{"DATE", schema.TypeDate},
		{"JSON", schema.TypeJSON},
		{"JSONB", schema.TypeJSONB},
		{"BYTEA", schema.TypeBytes},
		{"UNKNOWN_TYPE", schema.TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			got := SQLTypeToSchemaType(tt.sqlType)
			if got != tt.want {
				t.Errorf("SQLTypeToSchemaType(%q) = %v, want %v", tt.sqlType, got, tt.want)
			}
		})
	}
}

func TestParser_CaseInsensitivity(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{"uppercase", "CREATE TABLE USERS (ID INTEGER)"},
		{"lowercase", "create table users (id integer)"},
		{"mixed case", "Create Table Users (Id Integer)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			stmt, err := p.Parse(tt.sql)
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}

			ct, ok := stmt.(*CreateTable)
			if !ok {
				t.Fatalf("expected *CreateTable, got %T", stmt)
			}

			// Table name should preserve original case (or be normalized)
			if len(ct.Columns) != 1 {
				t.Errorf("expected 1 column, got %d", len(ct.Columns))
			}
		})
	}
}

func TestParser_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{"trailing semicolon", "CREATE TABLE users (id INTEGER);", false},
		{"extra whitespace", "CREATE   TABLE   users   (id   INTEGER)", false},
		{"empty SQL", "", true},
		{"incomplete statement", "CREATE TABLE", true},
		{"missing parentheses", "CREATE TABLE users", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			_, err := p.Parse(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper functions for comparisons

func stringPtr(s string) *string {
	return &s
}

func compareColumnDef(t *testing.T, got, want ColumnDef) {
	t.Helper()
	if got.Name != want.Name {
		t.Errorf("Column.Name = %v, want %v", got.Name, want.Name)
	}
	if got.Type != want.Type {
		t.Errorf("Column.Type = %v, want %v", got.Type, want.Type)
	}
	if got.Nullable != want.Nullable {
		t.Errorf("Column.Nullable = %v, want %v", got.Nullable, want.Nullable)
	}

	if (got.Default == nil) != (want.Default == nil) {
		t.Errorf("Column.Default presence mismatch: got %v, want %v", got.Default, want.Default)
	} else if got.Default != nil && *got.Default != *want.Default {
		t.Errorf("Column.Default = %v, want %v", *got.Default, *want.Default)
	}

	if len(got.Constraints) != len(want.Constraints) {
		t.Errorf("Column.Constraints count = %d, want %d", len(got.Constraints), len(want.Constraints))
	}

	for i := range want.Constraints {
		if i >= len(got.Constraints) {
			break
		}
		compareColumnConstraint(t, got.Constraints[i], want.Constraints[i])
	}
}

func compareColumnConstraint(t *testing.T, got, want ColumnConstraint) {
	t.Helper()
	if got.Type != want.Type {
		t.Errorf("ColumnConstraint.Type = %v, want %v", got.Type, want.Type)
	}
	if got.RefTable != want.RefTable {
		t.Errorf("ColumnConstraint.RefTable = %v, want %v", got.RefTable, want.RefTable)
	}
	if len(got.RefColumns) != len(want.RefColumns) {
		t.Errorf("ColumnConstraint.RefColumns count = %d, want %d", len(got.RefColumns), len(want.RefColumns))
	}
	if got.OnDelete != want.OnDelete {
		t.Errorf("ColumnConstraint.OnDelete = %v, want %v", got.OnDelete, want.OnDelete)
	}
}

func compareTableConstraint(t *testing.T, got, want TableConstraint) {
	t.Helper()
	if got.Type != want.Type {
		t.Errorf("TableConstraint.Type = %v, want %v", got.Type, want.Type)
	}
	if got.Name != want.Name {
		t.Errorf("TableConstraint.Name = %v, want %v", got.Name, want.Name)
	}
	if len(got.Columns) != len(want.Columns) {
		t.Errorf("TableConstraint.Columns count = %d, want %d", len(got.Columns), len(want.Columns))
	}
	if got.RefTable != want.RefTable {
		t.Errorf("TableConstraint.RefTable = %v, want %v", got.RefTable, want.RefTable)
	}
}

func compareAlterAction(t *testing.T, got, want AlterAction) {
	t.Helper()
	if got.ActionType() != want.ActionType() {
		t.Errorf("AlterAction.ActionType = %v, want %v", got.ActionType(), want.ActionType())
		return
	}

	switch w := want.(type) {
	case *AddColumn:
		g, ok := got.(*AddColumn)
		if !ok {
			t.Errorf("expected *AddColumn, got %T", got)
			return
		}
		compareColumnDef(t, g.Column, w.Column)
	case *DropColumn:
		g, ok := got.(*DropColumn)
		if !ok {
			t.Errorf("expected *DropColumn, got %T", got)
			return
		}
		if g.Name != w.Name {
			t.Errorf("DropColumn.Name = %v, want %v", g.Name, w.Name)
		}
		if g.IfExists != w.IfExists {
			t.Errorf("DropColumn.IfExists = %v, want %v", g.IfExists, w.IfExists)
		}
		if g.Cascade != w.Cascade {
			t.Errorf("DropColumn.Cascade = %v, want %v", g.Cascade, w.Cascade)
		}
	case *RenameColumn:
		g, ok := got.(*RenameColumn)
		if !ok {
			t.Errorf("expected *RenameColumn, got %T", got)
			return
		}
		if g.OldName != w.OldName {
			t.Errorf("RenameColumn.OldName = %v, want %v", g.OldName, w.OldName)
		}
		if g.NewName != w.NewName {
			t.Errorf("RenameColumn.NewName = %v, want %v", g.NewName, w.NewName)
		}
	case *AlterColumn:
		g, ok := got.(*AlterColumn)
		if !ok {
			t.Errorf("expected *AlterColumn, got %T", got)
			return
		}
		if g.Name != w.Name {
			t.Errorf("AlterColumn.Name = %v, want %v", g.Name, w.Name)
		}
		if g.SetNotNull != w.SetNotNull {
			t.Errorf("AlterColumn.SetNotNull = %v, want %v", g.SetNotNull, w.SetNotNull)
		}
		if g.DropNotNull != w.DropNotNull {
			t.Errorf("AlterColumn.DropNotNull = %v, want %v", g.DropNotNull, w.DropNotNull)
		}
		if g.DropDefault != w.DropDefault {
			t.Errorf("AlterColumn.DropDefault = %v, want %v", g.DropDefault, w.DropDefault)
		}
		if g.SetType != w.SetType {
			t.Errorf("AlterColumn.SetType = %v, want %v", g.SetType, w.SetType)
		}
	}
}
