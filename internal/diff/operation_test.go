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
