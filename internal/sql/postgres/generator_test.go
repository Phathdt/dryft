package postgres

import (
	"strings"
	"testing"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
)

func TestGenerator_GenerateCreateTable(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.CreateTable{
		Table: schema.Table{
			Name: "users",
			Columns: []schema.Column{
				{
					Name:     "id",
					Type:     schema.DataType{Kind: schema.TypeUUID},
					Nullable: false,
				},
				{
					Name:     "email",
					Type:     schema.DataType{Kind: schema.TypeText},
					Nullable: false,
				},
				{
					Name:     "name",
					Type:     schema.DataType{Kind: schema.TypeText},
					Nullable: true,
				},
			},
			PrimaryKey: &schema.PrimaryKey{
				Columns: []string{"id"},
			},
		},
	}

	sql := gen.generateCreateTable(op)

	if !strings.Contains(sql, "CREATE TABLE users") {
		t.Errorf("expected CREATE TABLE users, got: %s", sql)
	}
	if !strings.Contains(sql, "id UUID NOT NULL") {
		t.Errorf("expected id UUID NOT NULL, got: %s", sql)
	}
	if !strings.Contains(sql, "PRIMARY KEY (id)") {
		t.Errorf("expected PRIMARY KEY (id), got: %s", sql)
	}
}

func TestGenerator_GenerateAddColumn(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name     string
		op       diff.AddColumn
		contains []string
	}{
		{
			name: "nullable column",
			op: diff.AddColumn{
				Table: "users",
				Column: schema.Column{
					Name:     "phone",
					Type:     schema.DataType{Kind: schema.TypeText},
					Nullable: true,
				},
			},
			contains: []string{
				"ALTER TABLE users",
				"ADD COLUMN phone TEXT",
			},
		},
		{
			name: "not null with default",
			op: diff.AddColumn{
				Table: "users",
				Column: schema.Column{
					Name:     "created_at",
					Type:     schema.DataType{Kind: schema.TypeTimestampTZ},
					Nullable: false,
					Default: &schema.DefaultValue{
						Kind:       schema.DefaultExpression,
						Expression: "now()",
					},
				},
			},
			contains: []string{
				"ALTER TABLE users",
				"ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now()",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := gen.generateAddColumn(tt.op)
			for _, substr := range tt.contains {
				if !strings.Contains(sql, substr) {
					t.Errorf("expected SQL to contain %q, got: %s", substr, sql)
				}
			}
		})
	}
}

func TestGenerator_GenerateCreateIndex(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name     string
		op       diff.CreateIndex
		contains []string
	}{
		{
			name: "simple index",
			op: diff.CreateIndex{
				Table: "users",
				Index: schema.Index{
					Name:    "idx_users_email",
					Columns: []schema.IndexColumn{{Name: "email"}},
					Unique:  false,
				},
			},
			contains: []string{
				"CREATE INDEX idx_users_email ON users (email)",
			},
		},
		{
			name: "unique index",
			op: diff.CreateIndex{
				Table: "users",
				Index: schema.Index{
					Name:    "idx_users_email",
					Columns: []schema.IndexColumn{{Name: "email"}},
					Unique:  true,
				},
			},
			contains: []string{
				"CREATE UNIQUE INDEX idx_users_email ON users (email)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := gen.generateCreateIndex(tt.op)
			for _, substr := range tt.contains {
				if !strings.Contains(sql, substr) {
					t.Errorf("expected SQL to contain %q, got: %s", substr, sql)
				}
			}
		})
	}
}

func TestGenerator_GenerateCreateForeignKey(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.CreateForeignKey{
		Table: "posts",
		Constraint: schema.ForeignKey{
			Name:       "fk_posts_user_id",
			Columns:    []string{"user_id"},
			RefTable:   "users",
			RefColumns: []string{"id"},
			OnDelete:   schema.ActionCascade,
		},
	}

	sql := gen.generateCreateForeignKey(op)

	expectedParts := []string{
		"ALTER TABLE posts",
		"ADD CONSTRAINT fk_posts_user_id",
		"FOREIGN KEY (user_id)",
		"REFERENCES users (id)",
		"ON DELETE CASCADE",
	}

	for _, part := range expectedParts {
		if !strings.Contains(sql, part) {
			t.Errorf("expected SQL to contain %q, got: %s", part, sql)
		}
	}
}

func TestGenerator_GenerateCreateEnum(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	op := diff.CreateEnum{
		Enum: schema.Enum{
			Name: "status",
			Values: []schema.EnumValue{
				{Label: "active"},
				{Label: "inactive"},
			},
		},
	}

	sql := gen.generateCreateEnum(op)

	if !strings.Contains(sql, "CREATE TYPE status AS ENUM") {
		t.Errorf("expected CREATE TYPE, got: %s", sql)
	}
	if !strings.Contains(sql, "'active'") {
		t.Errorf("expected 'active', got: %s", sql)
	}
	if !strings.Contains(sql, "'inactive'") {
		t.Errorf("expected 'inactive', got: %s", sql)
	}
}

func TestGenerator_Generate(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	ops := []diff.Operation{
		diff.CreateEnum{
			Enum: schema.Enum{
				Name: "role",
				Values: []schema.EnumValue{
					{Label: "USER"},
					{Label: "ADMIN"},
				},
			},
		},
		diff.CreateTable{
			Table: schema.Table{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}},
					{Name: "email", Type: schema.DataType{Kind: schema.TypeText}},
				},
			},
		},
	}

	statements, err := gen.Generate(ops)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(statements) != 2 {
		t.Errorf("expected 2 statements, got %d", len(statements))
	}
}

func TestGenerator_GenerateReverse(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	ops := []diff.Operation{
		diff.CreateTable{
			Table: schema.Table{Name: "users"},
		},
		diff.CreateIndex{
			Table: "users",
			Index: schema.Index{
				Name:    "idx_email",
				Columns: []schema.IndexColumn{{Name: "email"}},
			},
		},
	}

	statements, warnings, err := gen.GenerateReverse(ops)
	if err != nil {
		t.Fatalf("GenerateReverse() error = %v", err)
	}

	// Should be reversed: drop index, then drop table
	if len(statements) != 2 {
		t.Errorf("expected 2 statements, got %d", len(statements))
	}

	if !strings.Contains(statements[0], "DROP INDEX") {
		t.Errorf("expected first statement to DROP INDEX, got: %s", statements[0])
	}

	if !strings.Contains(statements[1], "DROP TABLE") {
		t.Errorf("expected second statement to DROP TABLE, got: %s", statements[1])
	}

	if len(warnings) > 0 {
		t.Logf("Warnings: %v", warnings)
	}
}

func TestMapDataType(t *testing.T) {
	tests := []struct {
		dt   schema.DataType
		want string
	}{
		{schema.DataType{Kind: schema.TypeUUID}, "UUID"},
		{schema.DataType{Kind: schema.TypeText}, "TEXT"},
		{schema.DataType{Kind: schema.TypeInt32}, "INTEGER"},
		{schema.DataType{Kind: schema.TypeInt64}, "BIGINT"},
		{schema.DataType{Kind: schema.TypeBool}, "BOOLEAN"},
		{schema.DataType{Kind: schema.TypeTimestampTZ}, "TIMESTAMPTZ"},
		{schema.DataType{Kind: schema.TypeJSONB}, "JSONB"},
		{schema.DataType{Kind: schema.TypeVarChar, Precision: 255}, "VARCHAR(255)"},
		{schema.DataType{Kind: schema.TypeNumeric, Precision: 10, Scale: 2}, "NUMERIC(10,2)"},
		{schema.DataType{Kind: schema.TypeText, ArrayDepth: 1}, "TEXT[]"},
		{schema.DataType{Kind: schema.TypeInt32, ArrayDepth: 2}, "INTEGER[][]"},
		{schema.DataType{Kind: schema.TypeEnum, EnumName: "status"}, "status"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := mapDataType(tt.dt)
			if got != tt.want {
				t.Errorf("mapDataType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"users", "users"},
		{"User", `"User"`},
		{"user_name", "user_name"},
		{"user", `"user"`},   // Reserved word
		{"table", `"table"`}, // Reserved word
		{"My Table", `"My Table"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteIdentifier(tt.name)
			if got != tt.want {
				t.Errorf("quoteIdentifier(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
