package postgres

import (
	"testing"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/sql"
	"github.com/stretchr/testify/assert"
)

func TestGenerateDropTable(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name        string
		op          diff.DropTable
		expectedSQL string
	}{
		{
			name:        "simple drop table",
			op:          diff.DropTable{Name: "users"},
			expectedSQL: "DROP TABLE users;",
		},
		{
			name:        "drop table reserved keyword",
			op:          diff.DropTable{Name: "order"},
			expectedSQL: "DROP TABLE \"order\";",
		},
		{
			name:        "drop table uppercase",
			op:          diff.DropTable{Name: "Users"},
			expectedSQL: "DROP TABLE \"Users\";",
		},
		{
			name:        "drop table with underscore",
			op:          diff.DropTable{Name: "user_profiles"},
			expectedSQL: "DROP TABLE user_profiles;",
		},
		{
			name:        "drop table multiple reserved keywords",
			op:          diff.DropTable{Name: "select"},
			expectedSQL: "DROP TABLE \"select\";",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := gen.generateDropTable(tt.op)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestGenerateDropColumn(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name        string
		op          diff.DropColumn
		expectedSQL string
	}{
		{
			name:        "simple drop column",
			op:          diff.DropColumn{Table: "users", Column: "phone"},
			expectedSQL: "ALTER TABLE users\nDROP COLUMN phone;",
		},
		{
			name:        "drop column reserved keyword",
			op:          diff.DropColumn{Table: "posts", Column: "order"},
			expectedSQL: "ALTER TABLE posts\nDROP COLUMN \"order\";",
		},
		{
			name:        "drop column from table with uppercase",
			op:          diff.DropColumn{Table: "User", Column: "email"},
			expectedSQL: "ALTER TABLE \"User\"\nDROP COLUMN email;",
		},
		{
			name:        "drop column with underscore",
			op:          diff.DropColumn{Table: "user_profiles", Column: "created_at"},
			expectedSQL: "ALTER TABLE user_profiles\nDROP COLUMN created_at;",
		},
		{
			name:        "drop column both reserved",
			op:          diff.DropColumn{Table: "user", Column: "select"},
			expectedSQL: "ALTER TABLE \"user\"\nDROP COLUMN \"select\";",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := gen.generateDropColumn(tt.op)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestGenerateDropIndex(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name        string
		op          diff.DropIndex
		expectedSQL string
	}{
		{
			name:        "simple drop index",
			op:          diff.DropIndex{Table: "users", Name: "idx_users_email"},
			expectedSQL: "DROP INDEX idx_users_email;",
		},
		{
			name:        "drop index reserved keyword",
			op:          diff.DropIndex{Table: "posts", Name: "order"},
			expectedSQL: "DROP INDEX \"order\";",
		},
		{
			name:        "drop index with uppercase",
			op:          diff.DropIndex{Table: "users", Name: "IDX_Users"},
			expectedSQL: "DROP INDEX \"IDX_Users\";",
		},
		{
			name:        "drop index with drop type",
			op:          diff.DropIndex{Table: "users", Name: "idx_unique_email"},
			expectedSQL: "DROP INDEX idx_unique_email;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := gen.generateDropIndex(tt.op)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestGenerateDropForeignKey(t *testing.T) {
	gen := NewGenerator(sql.GeneratorOptions{})

	tests := []struct {
		name        string
		op          diff.DropForeignKey
		expectedSQL string
	}{
		{
			name:        "simple drop foreign key",
			op:          diff.DropForeignKey{Table: "posts", Name: "fk_posts_user_id"},
			expectedSQL: "ALTER TABLE posts\nDROP CONSTRAINT fk_posts_user_id;",
		},
		{
			name:        "drop foreign key reserved keyword table",
			op:          diff.DropForeignKey{Table: "order", Name: "fk_order_user_id"},
			expectedSQL: "ALTER TABLE \"order\"\nDROP CONSTRAINT fk_order_user_id;",
		},
		{
			name:        "drop foreign key uppercase constraint name",
			op:          diff.DropForeignKey{Table: "posts", Name: "FK_Posts_User"},
			expectedSQL: "ALTER TABLE posts\nDROP CONSTRAINT \"FK_Posts_User\";",
		},
		{
			name:        "drop fk both parts reserved",
			op:          diff.DropForeignKey{Table: "user", Name: "fk_user_table"},
			expectedSQL: "ALTER TABLE \"user\"\nDROP CONSTRAINT fk_user_table;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := gen.generateDropForeignKey(tt.op)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}
