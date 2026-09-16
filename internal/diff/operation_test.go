package diff

import (
	"testing"

	"github.com/phathdt/dryft/internal/schema"
)

func TestOperationKind_String(t *testing.T) {
	tests := []struct {
		kind     OperationKind
		expected string
	}{
		{OpCreateTable, "CREATE_TABLE"},
		{OpDropTable, "DROP_TABLE"},
		{OpRenameTable, "RENAME_TABLE"},
		{OpAddColumn, "ADD_COLUMN"},
		{OpDropColumn, "DROP_COLUMN"},
		{OpRenameColumn, "RENAME_COLUMN"},
		{OpAlterColumn, "ALTER_COLUMN"},
		{OpCreateIndex, "CREATE_INDEX"},
		{OpDropIndex, "DROP_INDEX"},
		{OpCreateForeignKey, "CREATE_FOREIGN_KEY"},
		{OpDropForeignKey, "DROP_FOREIGN_KEY"},
		{OpCreateEnum, "CREATE_ENUM"},
		{OpAlterEnum, "ALTER_ENUM"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.expected {
				t.Errorf("OperationKind.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDestructiveLevel_String(t *testing.T) {
	tests := []struct {
		level    DestructiveLevel
		expected string
	}{
		{Safe, "SAFE"},
		{Risky, "RISKY"},
		{PotentiallyDestructive, "POTENTIALLY_DESTRUCTIVE"},
		{Destructive, "DESTRUCTIVE"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("DestructiveLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCreateTable_IsDestructive(t *testing.T) {
	op := CreateTable{Table: schema.Table{Name: "users"}}
	if op.IsDestructive() != Safe {
		t.Errorf("CreateTable.IsDestructive() = %v, want Safe", op.IsDestructive())
	}
}

func TestDropTable_IsDestructive(t *testing.T) {
	op := DropTable{Name: "users"}
	if op.IsDestructive() != Destructive {
		t.Errorf("DropTable.IsDestructive() = %v, want Destructive", op.IsDestructive())
	}
}

func TestAddColumn_IsDestructive(t *testing.T) {
	tests := []struct {
		name     string
		op       AddColumn
		expected DestructiveLevel
	}{
		{
			name: "nullable column",
			op: AddColumn{
				Table: "users",
				Column: schema.Column{
					Name:     "email",
					Nullable: true,
				},
			},
			expected: Safe,
		},
		{
			name: "not null with default",
			op: AddColumn{
				Table: "users",
				Column: schema.Column{
					Name:     "email",
					Nullable: false,
					Default:  &schema.DefaultValue{Kind: schema.DefaultLiteral, Literal: "''"},
				},
			},
			expected: Safe,
		},
		{
			name: "not null without default",
			op: AddColumn{
				Table: "users",
				Column: schema.Column{
					Name:     "email",
					Nullable: false,
					Default:  nil,
				},
			},
			expected: PotentiallyDestructive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.IsDestructive(); got != tt.expected {
				t.Errorf("AddColumn.IsDestructive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDropColumn_IsDestructive(t *testing.T) {
	op := DropColumn{Table: "users", Column: "email"}
	if op.IsDestructive() != Destructive {
		t.Errorf("DropColumn.IsDestructive() = %v, want Destructive", op.IsDestructive())
	}
}

func TestAlterColumn_IsDestructive(t *testing.T) {
	tests := []struct {
		name     string
		op       AlterColumn
		expected DestructiveLevel
	}{
		{
			name: "type change",
			op: AlterColumn{
				Table:   "users",
				Column:  "id",
				OldType: schema.DataType{Kind: schema.TypeInt32},
				NewType: schema.DataType{Kind: schema.TypeInt64},
			},
			expected: Destructive,
		},
		{
			name: "set not null",
			op: AlterColumn{
				Table:       "users",
				Column:      "email",
				OldType:     schema.DataType{Kind: schema.TypeText},
				NewType:     schema.DataType{Kind: schema.TypeText},
				OldNullable: true,
				NewNullable: false,
			},
			expected: PotentiallyDestructive,
		},
		{
			name: "remove not null",
			op: AlterColumn{
				Table:       "users",
				Column:      "email",
				OldType:     schema.DataType{Kind: schema.TypeText},
				NewType:     schema.DataType{Kind: schema.TypeText},
				OldNullable: false,
				NewNullable: true,
			},
			expected: Safe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.IsDestructive(); got != tt.expected {
				t.Errorf("AlterColumn.IsDestructive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCreateIndex_IsDestructive(t *testing.T) {
	op := CreateIndex{
		Table: "users",
		Index: schema.Index{
			Columns: []schema.IndexColumn{{Name: "email"}},
		},
	}
	if op.IsDestructive() != Risky {
		t.Errorf("CreateIndex.IsDestructive() = %v, want Risky", op.IsDestructive())
	}
}

func TestDropIndex_IsDestructive(t *testing.T) {
	op := DropIndex{Table: "users", Name: "idx_email"}
	if op.IsDestructive() != PotentiallyDestructive {
		t.Errorf("DropIndex.IsDestructive() = %v, want PotentiallyDestructive", op.IsDestructive())
	}
}

func TestCreateForeignKey_IsDestructive(t *testing.T) {
	op := CreateForeignKey{
		Table: "posts",
		Constraint: schema.ForeignKey{
			Columns:  []string{"user_id"},
			RefTable: "users",
		},
	}
	if op.IsDestructive() != Risky {
		t.Errorf("CreateForeignKey.IsDestructive() = %v, want Risky", op.IsDestructive())
	}
}

func TestAlterEnum_IsDestructive(t *testing.T) {
	tests := []struct {
		name     string
		op       AlterEnum
		expected DestructiveLevel
	}{
		{
			name: "add values only",
			op: AlterEnum{
				Name:      "status",
				AddValues: []string{"pending"},
			},
			expected: Safe,
		},
		{
			name: "drop values",
			op: AlterEnum{
				Name:       "status",
				DropValues: []string{"archived"},
			},
			expected: Destructive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.IsDestructive(); got != tt.expected {
				t.Errorf("AlterEnum.IsDestructive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCreateTable_Description(t *testing.T) {
	op := CreateTable{Table: schema.Table{Name: "users"}}
	desc := op.Description()
	if desc != "CREATE TABLE users" {
		t.Errorf("CreateTable.Description() = %v, want 'CREATE TABLE users'", desc)
	}
}

func TestDropTable_Description(t *testing.T) {
	op := DropTable{Name: "users"}
	desc := op.Description()
	if desc != "DROP TABLE users" {
		t.Errorf("DropTable.Description() = %v, want 'DROP TABLE users'", desc)
	}
}

func TestRenameTable_Kind(t *testing.T) {
	op := RenameTable{From: "old_users", To: "users"}
	if op.Kind() != OpRenameTable {
		t.Errorf("RenameTable.Kind() = %v, want OpRenameTable", op.Kind())
	}
}

func TestRenameTable_IsDestructive(t *testing.T) {
	op := RenameTable{From: "old_users", To: "users"}
	if op.IsDestructive() != PotentiallyDestructive {
		t.Errorf("RenameTable.IsDestructive() = %v, want PotentiallyDestructive", op.IsDestructive())
	}
}

func TestRenameTable_Description(t *testing.T) {
	op := RenameTable{From: "old_users", To: "users"}
	desc := op.Description()
	if desc != "RENAME TABLE old_users TO users" {
		t.Errorf("RenameTable.Description() = %v, want 'RENAME TABLE old_users TO users'", desc)
	}
}

func TestAddColumn_Kind(t *testing.T) {
	op := AddColumn{
		Table: "users",
		Column: schema.Column{Name: "email"},
	}
	if op.Kind() != OpAddColumn {
		t.Errorf("AddColumn.Kind() = %v, want OpAddColumn", op.Kind())
	}
}

func TestAddColumn_Description(t *testing.T) {
	op := AddColumn{
		Table: "users",
		Column: schema.Column{Name: "email"},
	}
	desc := op.Description()
	if desc != "ADD COLUMN users.email" {
		t.Errorf("AddColumn.Description() = %v, want 'ADD COLUMN users.email'", desc)
	}
}

func TestDropColumn_Kind(t *testing.T) {
	op := DropColumn{Table: "users", Column: "email"}
	if op.Kind() != OpDropColumn {
		t.Errorf("DropColumn.Kind() = %v, want OpDropColumn", op.Kind())
	}
}

func TestDropColumn_Description(t *testing.T) {
	op := DropColumn{Table: "users", Column: "email"}
	desc := op.Description()
	if desc != "DROP COLUMN users.email" {
		t.Errorf("DropColumn.Description() = %v, want 'DROP COLUMN users.email'", desc)
	}
}

func TestRenameColumn_Kind(t *testing.T) {
	op := RenameColumn{Table: "users", From: "name", To: "username"}
	if op.Kind() != OpRenameColumn {
		t.Errorf("RenameColumn.Kind() = %v, want OpRenameColumn", op.Kind())
	}
}

func TestRenameColumn_IsDestructive(t *testing.T) {
	op := RenameColumn{Table: "users", From: "name", To: "username"}
	if op.IsDestructive() != PotentiallyDestructive {
		t.Errorf("RenameColumn.IsDestructive() = %v, want PotentiallyDestructive", op.IsDestructive())
	}
}

func TestRenameColumn_Description(t *testing.T) {
	op := RenameColumn{Table: "users", From: "name", To: "username"}
	desc := op.Description()
	if desc != "RENAME COLUMN users.name TO username" {
		t.Errorf("RenameColumn.Description() = %v, want 'RENAME COLUMN users.name TO username'", desc)
	}
}

func TestAlterColumn_Kind(t *testing.T) {
	op := AlterColumn{
		Table:   "users",
		Column:  "id",
		OldType: schema.DataType{Kind: schema.TypeInt32},
		NewType: schema.DataType{Kind: schema.TypeInt64},
	}
	if op.Kind() != OpAlterColumn {
		t.Errorf("AlterColumn.Kind() = %v, want OpAlterColumn", op.Kind())
	}
}

func TestAlterColumn_Description(t *testing.T) {
	op := AlterColumn{
		Table:   "users",
		Column:  "id",
		OldType: schema.DataType{Kind: schema.TypeInt32},
		NewType: schema.DataType{Kind: schema.TypeInt64},
	}
	desc := op.Description()
	if desc != "ALTER COLUMN users.id" {
		t.Errorf("AlterColumn.Description() = %v, want 'ALTER COLUMN users.id'", desc)
	}
}

func TestCreateIndex_Kind(t *testing.T) {
	op := CreateIndex{
		Table: "users",
		Index: schema.Index{Columns: []schema.IndexColumn{{Name: "email"}}},
	}
	if op.Kind() != OpCreateIndex {
		t.Errorf("CreateIndex.Kind() = %v, want OpCreateIndex", op.Kind())
	}
}

func TestCreateIndex_Description(t *testing.T) {
	op := CreateIndex{
		Table: "users",
		Index: schema.Index{Columns: []schema.IndexColumn{{Name: "email"}}},
	}
	desc := op.Description()
	if desc != "CREATE INDEX ON users" {
		t.Errorf("CreateIndex.Description() = %v, want 'CREATE INDEX ON users'", desc)
	}
}

func TestDropIndex_Kind(t *testing.T) {
	op := DropIndex{Table: "users", Name: "idx_email"}
	if op.Kind() != OpDropIndex {
		t.Errorf("DropIndex.Kind() = %v, want OpDropIndex", op.Kind())
	}
}

func TestDropIndex_Description(t *testing.T) {
	op := DropIndex{Table: "users", Name: "idx_email"}
	desc := op.Description()
	if desc != "DROP INDEX idx_email" {
		t.Errorf("DropIndex.Description() = %v, want 'DROP INDEX idx_email'", desc)
	}
}

func TestCreateForeignKey_Kind(t *testing.T) {
	op := CreateForeignKey{
		Table: "posts",
		Constraint: schema.ForeignKey{
			Columns:  []string{"user_id"},
			RefTable: "users",
		},
	}
	if op.Kind() != OpCreateForeignKey {
		t.Errorf("CreateForeignKey.Kind() = %v, want OpCreateForeignKey", op.Kind())
	}
}

func TestCreateForeignKey_Description(t *testing.T) {
	op := CreateForeignKey{
		Table: "posts",
		Constraint: schema.ForeignKey{
			Columns:  []string{"user_id"},
			RefTable: "users",
		},
	}
	desc := op.Description()
	if desc != "ADD FOREIGN KEY posts -> users" {
		t.Errorf("CreateForeignKey.Description() = %v, want 'ADD FOREIGN KEY posts -> users'", desc)
	}
}

func TestDropForeignKey_Kind(t *testing.T) {
	op := DropForeignKey{Table: "posts", Name: "fk_user"}
	if op.Kind() != OpDropForeignKey {
		t.Errorf("DropForeignKey.Kind() = %v, want OpDropForeignKey", op.Kind())
	}
}

func TestDropForeignKey_IsDestructive(t *testing.T) {
	op := DropForeignKey{Table: "posts", Name: "fk_user"}
	if op.IsDestructive() != Safe {
		t.Errorf("DropForeignKey.IsDestructive() = %v, want Safe", op.IsDestructive())
	}
}

func TestDropForeignKey_Description(t *testing.T) {
	op := DropForeignKey{Table: "posts", Name: "fk_user"}
	desc := op.Description()
	if desc != "DROP FOREIGN KEY fk_user" {
		t.Errorf("DropForeignKey.Description() = %v, want 'DROP FOREIGN KEY fk_user'", desc)
	}
}

func TestCreateEnum_Kind(t *testing.T) {
	op := CreateEnum{Enum: schema.Enum{Name: "status"}}
	if op.Kind() != OpCreateEnum {
		t.Errorf("CreateEnum.Kind() = %v, want OpCreateEnum", op.Kind())
	}
}

func TestCreateEnum_IsDestructive(t *testing.T) {
	op := CreateEnum{Enum: schema.Enum{Name: "status"}}
	if op.IsDestructive() != Safe {
		t.Errorf("CreateEnum.IsDestructive() = %v, want Safe", op.IsDestructive())
	}
}

func TestCreateEnum_Description(t *testing.T) {
	op := CreateEnum{Enum: schema.Enum{Name: "status"}}
	desc := op.Description()
	if desc != "CREATE ENUM status" {
		t.Errorf("CreateEnum.Description() = %v, want 'CREATE ENUM status'", desc)
	}
}

func TestAlterEnum_Kind(t *testing.T) {
	op := AlterEnum{Name: "status", AddValues: []string{"pending"}}
	if op.Kind() != OpAlterEnum {
		t.Errorf("AlterEnum.Kind() = %v, want OpAlterEnum", op.Kind())
	}
}

func TestAlterEnum_Description(t *testing.T) {
	op := AlterEnum{Name: "status", AddValues: []string{"pending"}}
	desc := op.Description()
	if desc != "ALTER ENUM status" {
		t.Errorf("AlterEnum.Description() = %v, want 'ALTER ENUM status'", desc)
	}
}
