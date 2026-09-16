package diff

import "github.com/phathdt/dryft/internal/schema"

// OperationKind represents the type of schema operation.
type OperationKind int

const (
	// OpCreateTable creates a new table.
	OpCreateTable OperationKind = iota
	// OpDropTable removes an existing table.
	OpDropTable
	// OpRenameTable renames an existing table.
	OpRenameTable
	// OpAddColumn adds a new column to a table.
	OpAddColumn
	// OpDropColumn removes a column from a table.
	OpDropColumn
	// OpRenameColumn renames a column in a table.
	OpRenameColumn
	// OpAlterColumn modifies a column's type or constraints.
	OpAlterColumn
	// OpCreateIndex creates a new index.
	OpCreateIndex
	// OpDropIndex removes an existing index.
	OpDropIndex
	// OpCreateForeignKey adds a foreign key constraint.
	OpCreateForeignKey
	// OpDropForeignKey removes a foreign key constraint.
	OpDropForeignKey
	// OpCreateEnum creates a new enum type.
	OpCreateEnum
	// OpAlterEnum modifies an enum type.
	OpAlterEnum
)

func (k OperationKind) String() string {
	switch k {
	case OpCreateTable:
		return "CREATE_TABLE"
	case OpDropTable:
		return "DROP_TABLE"
	case OpRenameTable:
		return "RENAME_TABLE"
	case OpAddColumn:
		return "ADD_COLUMN"
	case OpDropColumn:
		return "DROP_COLUMN"
	case OpRenameColumn:
		return "RENAME_COLUMN"
	case OpAlterColumn:
		return "ALTER_COLUMN"
	case OpCreateIndex:
		return "CREATE_INDEX"
	case OpDropIndex:
		return "DROP_INDEX"
	case OpCreateForeignKey:
		return "CREATE_FOREIGN_KEY"
	case OpDropForeignKey:
		return "DROP_FOREIGN_KEY"
	case OpCreateEnum:
		return "CREATE_ENUM"
	case OpAlterEnum:
		return "ALTER_ENUM"
	default:
		return "UNKNOWN"
	}
}

// DestructiveLevel represents how destructive an operation is.
type DestructiveLevel int

const (
	// Safe operations don't risk data loss.
	Safe DestructiveLevel = iota
	// Risky operations might cause issues but are generally safe.
	Risky
	// PotentiallyDestructive operations may lose data under certain conditions.
	PotentiallyDestructive
	// Destructive operations will cause data loss.
	Destructive
)

func (d DestructiveLevel) String() string {
	switch d {
	case Safe:
		return "SAFE"
	case Risky:
		return "RISKY"
	case PotentiallyDestructive:
		return "POTENTIALLY_DESTRUCTIVE"
	case Destructive:
		return "DESTRUCTIVE"
	default:
		return "UNKNOWN"
	}
}

// Operation represents a schema change operation.
type Operation interface {
	Kind() OperationKind
	IsDestructive() DestructiveLevel
	Description() string
}

// CreateTable creates a new table.
type CreateTable struct {
	Table schema.Table
}

func (op CreateTable) Kind() OperationKind             { return OpCreateTable }
func (op CreateTable) IsDestructive() DestructiveLevel { return Safe }
func (op CreateTable) Description() string {
	return "CREATE TABLE " + op.Table.Name
}

// DropTable drops an existing table.
type DropTable struct {
	Name string
}

func (op DropTable) Kind() OperationKind             { return OpDropTable }
func (op DropTable) IsDestructive() DestructiveLevel { return Destructive }
func (op DropTable) Description() string {
	return "DROP TABLE " + op.Name
}

// RenameTable renames a table.
type RenameTable struct {
	From string
	To   string
}

func (op RenameTable) Kind() OperationKind             { return OpRenameTable }
func (op RenameTable) IsDestructive() DestructiveLevel { return PotentiallyDestructive }
func (op RenameTable) Description() string {
	return "RENAME TABLE " + op.From + " TO " + op.To
}

// AddColumn adds a column to a table.
type AddColumn struct {
	Table  string
	Column schema.Column
}

func (op AddColumn) Kind() OperationKind { return OpAddColumn }
func (op AddColumn) IsDestructive() DestructiveLevel {
	// Adding NOT NULL column without default is potentially destructive
	if !op.Column.Nullable && op.Column.Default == nil {
		return PotentiallyDestructive
	}
	return Safe
}
func (op AddColumn) Description() string {
	return "ADD COLUMN " + op.Table + "." + op.Column.Name
}

// DropColumn drops a column from a table.
type DropColumn struct {
	Table  string
	Column string
}

func (op DropColumn) Kind() OperationKind             { return OpDropColumn }
func (op DropColumn) IsDestructive() DestructiveLevel { return Destructive }
func (op DropColumn) Description() string {
	return "DROP COLUMN " + op.Table + "." + op.Column
}

// RenameColumn renames a column.
type RenameColumn struct {
	Table string
	From  string
	To    string
}

func (op RenameColumn) Kind() OperationKind             { return OpRenameColumn }
func (op RenameColumn) IsDestructive() DestructiveLevel { return PotentiallyDestructive }
func (op RenameColumn) Description() string {
	return "RENAME COLUMN " + op.Table + "." + op.From + " TO " + op.To
}

// AlterColumn modifies column properties.
type AlterColumn struct {
	Table       string
	Column      string
	OldType     schema.DataType
	NewType     schema.DataType
	OldNullable bool
	NewNullable bool
	OldDefault  *schema.DefaultValue
	NewDefault  *schema.DefaultValue
}

func (op AlterColumn) Kind() OperationKind { return OpAlterColumn }
func (op AlterColumn) IsDestructive() DestructiveLevel {
	// Type change is destructive if incompatible
	if op.OldType.Kind != op.NewType.Kind {
		return Destructive
	}
	// Setting NOT NULL is potentially destructive
	if op.OldNullable && !op.NewNullable {
		return PotentiallyDestructive
	}
	return Safe
}
func (op AlterColumn) Description() string {
	return "ALTER COLUMN " + op.Table + "." + op.Column
}

// CreateIndex creates an index.
type CreateIndex struct {
	Table string
	Index schema.Index
}

func (op CreateIndex) Kind() OperationKind { return OpCreateIndex }
func (op CreateIndex) IsDestructive() DestructiveLevel {
	// CREATE INDEX without CONCURRENTLY is risky
	return Risky
}
func (op CreateIndex) Description() string {
	return "CREATE INDEX ON " + op.Table
}

// DropIndex drops an index.
type DropIndex struct {
	Table string
	Name  string
}

func (op DropIndex) Kind() OperationKind             { return OpDropIndex }
func (op DropIndex) IsDestructive() DestructiveLevel { return PotentiallyDestructive }
func (op DropIndex) Description() string {
	return "DROP INDEX " + op.Name
}

// CreateForeignKey adds a foreign key constraint.
type CreateForeignKey struct {
	Table      string
	Constraint schema.ForeignKey
}

func (op CreateForeignKey) Kind() OperationKind { return OpCreateForeignKey }
func (op CreateForeignKey) IsDestructive() DestructiveLevel {
	// Adding FK without NOT VALID is risky
	return Risky
}
func (op CreateForeignKey) Description() string {
	return "ADD FOREIGN KEY " + op.Table + " -> " + op.Constraint.RefTable
}

// DropForeignKey drops a foreign key constraint.
type DropForeignKey struct {
	Table string
	Name  string
}

func (op DropForeignKey) Kind() OperationKind             { return OpDropForeignKey }
func (op DropForeignKey) IsDestructive() DestructiveLevel { return Safe }
func (op DropForeignKey) Description() string {
	return "DROP FOREIGN KEY " + op.Name
}

// CreateEnum creates a new enum type.
type CreateEnum struct {
	Enum schema.Enum
}

func (op CreateEnum) Kind() OperationKind             { return OpCreateEnum }
func (op CreateEnum) IsDestructive() DestructiveLevel { return Safe }
func (op CreateEnum) Description() string {
	return "CREATE ENUM " + op.Enum.Name
}

// AlterEnum modifies an enum type.
type AlterEnum struct {
	Name       string
	AddValues  []string
	DropValues []string
}

func (op AlterEnum) Kind() OperationKind { return OpAlterEnum }
func (op AlterEnum) IsDestructive() DestructiveLevel {
	// Dropping enum values is destructive
	if len(op.DropValues) > 0 {
		return Destructive
	}
	return Safe
}
func (op AlterEnum) Description() string {
	return "ALTER ENUM " + op.Name
}
