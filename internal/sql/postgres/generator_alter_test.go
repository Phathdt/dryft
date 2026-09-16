package postgres

import (
	"testing"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAlterColumn(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name        string
		op          diff.AlterColumn
		expectedSQL string
	}{
		{
			name: "change type compatible",
			op: diff.AlterColumn{
				Table:      "users",
				Column:     "age",
				OldType:    schema.DataType{Kind: schema.TypeInt32},
				NewType:    schema.DataType{Kind: schema.TypeInt64},
				OldNullable: true,
				NewNullable: true,
			},
			expectedSQL: "ALTER TABLE users\nALTER COLUMN age TYPE BIGINT;",
		},
		{
			name: "set not null",
			op: diff.AlterColumn{
				Table:       "users",
				Column:      "email",
				OldType:     schema.DataType{Kind: schema.TypeText},
				NewType:     schema.DataType{Kind: schema.TypeText},
				OldNullable: true,
				NewNullable: false,
			},
			expectedSQL: "ALTER TABLE users\nALTER COLUMN email SET NOT NULL;",
		},
		{
			name: "drop not null",
			op: diff.AlterColumn{
				Table:       "users",
				Column:      "phone",
				OldType:     schema.DataType{Kind: schema.TypeText},
				NewType:     schema.DataType{Kind: schema.TypeText},
				OldNullable: false,
				NewNullable: true,
			},
			expectedSQL: "ALTER TABLE users\nALTER COLUMN phone DROP NOT NULL;",
		},
		{
			name: "set default literal",
			op: diff.AlterColumn{
				Table:       "users",
				Column:      "status",
				OldType:     schema.DataType{Kind: schema.TypeText},
				NewType:     schema.DataType{Kind: schema.TypeText},
				OldNullable: true,
				NewNullable: true,
				OldDefault:  nil,
				NewDefault: &schema.DefaultValue{
					Kind:    schema.DefaultLiteral,
					Literal: "'active'",
				},
			},
			expectedSQL: "ALTER TABLE users\nALTER COLUMN status SET DEFAULT 'active';",
		},
		{
			name: "set default expression",
			op: diff.AlterColumn{
				Table:       "events",
				Column:      "created_at",
				OldType:     schema.DataType{Kind: schema.TypeTimestampTZ},
				NewType:     schema.DataType{Kind: schema.TypeTimestampTZ},
				OldNullable: false,
				NewNullable: false,
				OldDefault:  nil,
				NewDefault: &schema.DefaultValue{
					Kind:       schema.DefaultExpression,
					Expression: "now()",
				},
			},
			expectedSQL: "ALTER TABLE events\nALTER COLUMN created_at SET DEFAULT now();",
		},
		{
			name: "drop default",
			op: diff.AlterColumn{
				Table:       "users",
				Column:      "status",
				OldType:     schema.DataType{Kind: schema.TypeText},
				NewType:     schema.DataType{Kind: schema.TypeText},
				OldNullable: true,
				NewNullable: true,
				OldDefault: &schema.DefaultValue{
					Kind:    schema.DefaultLiteral,
					Literal: "'active'",
				},
				NewDefault: nil,
			},
			expectedSQL: "ALTER TABLE users\nALTER COLUMN status DROP DEFAULT;",
		},
		{
			name: "multiple changes: type and not null",
			op: diff.AlterColumn{
				Table:       "users",
				Column:      "age",
				OldType:     schema.DataType{Kind: schema.TypeInt32},
				NewType:     schema.DataType{Kind: schema.TypeInt64},
				OldNullable: true,
				NewNullable: false,
			},
			expectedSQL: "ALTER TABLE users\nALTER COLUMN age TYPE BIGINT;\n\nALTER TABLE users\nALTER COLUMN age SET NOT NULL;",
		},
		{
			name: "multiple changes: type, not null, and default",
			op: diff.AlterColumn{
				Table:       "products",
				Column:      "quantity",
				OldType:     schema.DataType{Kind: schema.TypeInt32},
				NewType:     schema.DataType{Kind: schema.TypeInt64},
				OldNullable: true,
				NewNullable: false,
				OldDefault:  nil,
				NewDefault: &schema.DefaultValue{
					Kind:    schema.DefaultLiteral,
					Literal: "0",
				},
			},
			expectedSQL: "ALTER TABLE products\nALTER COLUMN quantity TYPE BIGINT;\n\nALTER TABLE products\nALTER COLUMN quantity SET NOT NULL;\n\nALTER TABLE products\nALTER COLUMN quantity SET DEFAULT 0;",
		},
		{
			name: "reserved keyword table and column",
			op: diff.AlterColumn{
				Table:       "user",
				Column:      "select",
				OldType:     schema.DataType{Kind: schema.TypeText},
				NewType:     schema.DataType{Kind: schema.TypeVarChar, Precision: 100},
				OldNullable: true,
				NewNullable: true,
			},
			expectedSQL: "ALTER TABLE \"user\"\nALTER COLUMN \"select\" TYPE VARCHAR(100);",
		},
		{
			name: "change from text to int with using",
			op: diff.AlterColumn{
				Table:       "inventory",
				Column:      "code",
				OldType:     schema.DataType{Kind: schema.TypeText},
				NewType:     schema.DataType{Kind: schema.TypeInt32},
				OldNullable: true,
				NewNullable: true,
			},
			expectedSQL: "ALTER TABLE inventory\nALTER COLUMN code TYPE INTEGER;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := gen.generateAlterColumn(tt.op)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestGenerateRenameColumn(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name        string
		op          diff.RenameColumn
		expectedSQL string
	}{
		{
			name:        "simple rename",
			op:          diff.RenameColumn{Table: "users", From: "phone", To: "phone_number"},
			expectedSQL: "ALTER TABLE users\nRENAME COLUMN phone TO phone_number;",
		},
		{
			name:        "rename with reserved keyword source",
			op:          diff.RenameColumn{Table: "posts", From: "select", To: "selection"},
			expectedSQL: "ALTER TABLE posts\nRENAME COLUMN \"select\" TO selection;",
		},
		{
			name:        "rename to reserved keyword target",
			op:          diff.RenameColumn{Table: "posts", From: "name", To: "order"},
			expectedSQL: "ALTER TABLE posts\nRENAME COLUMN name TO \"order\";",
		},
		{
			name:        "rename both reserved",
			op:          diff.RenameColumn{Table: "user", From: "table", To: "select"},
			expectedSQL: "ALTER TABLE \"user\"\nRENAME COLUMN \"table\" TO \"select\";",
		},
		{
			name:        "rename uppercase columns",
			op:          diff.RenameColumn{Table: "Users", From: "PhoneNumber", To: "CellPhone"},
			expectedSQL: "ALTER TABLE \"Users\"\nRENAME COLUMN \"PhoneNumber\" TO \"CellPhone\";",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := gen.generateRenameColumn(tt.op)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestGenerateRenameTable(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name        string
		op          diff.RenameTable
		expectedSQL string
	}{
		{
			name:        "simple rename",
			op:          diff.RenameTable{From: "users", To: "accounts"},
			expectedSQL: "ALTER TABLE users RENAME TO accounts;",
		},
		{
			name:        "rename reserved keyword",
			op:          diff.RenameTable{From: "user", To: "members"},
			expectedSQL: "ALTER TABLE \"user\" RENAME TO members;",
		},
		{
			name:        "rename to reserved keyword",
			op:          diff.RenameTable{From: "members", To: "order"},
			expectedSQL: "ALTER TABLE members RENAME TO \"order\";",
		},
		{
			name:        "rename uppercase",
			op:          diff.RenameTable{From: "UserProfiles", To: "AccountProfiles"},
			expectedSQL: "ALTER TABLE \"UserProfiles\" RENAME TO \"AccountProfiles\";",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := gen.generateRenameTable(tt.op)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}
