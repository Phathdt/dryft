package migration

// Statement represents a parsed SQL statement.
type Statement interface {
	Type() StatementType
}

// StatementType identifies the kind of SQL statement.
type StatementType string

const (
	CreateTableStmt StatementType = "CREATE_TABLE"
	DropTableStmt   StatementType = "DROP_TABLE"
	AlterTableStmt  StatementType = "ALTER_TABLE"
	CreateTypeStmt  StatementType = "CREATE_TYPE"
	DropTypeStmt    StatementType = "DROP_TYPE"
	AlterTypeStmt   StatementType = "ALTER_TYPE"
	CreateIndexStmt StatementType = "CREATE_INDEX"
	DropIndexStmt   StatementType = "DROP_INDEX"
)

// CreateTable represents a CREATE TABLE statement.
type CreateTable struct {
	Name        string
	Columns     []ColumnDef
	Constraints []TableConstraint
	IfNotExists bool
}

func (c *CreateTable) Type() StatementType { return CreateTableStmt }

// DropTable represents a DROP TABLE statement.
type DropTable struct {
	Name     string
	IfExists bool
	Cascade  bool
}

func (d *DropTable) Type() StatementType { return DropTableStmt }

// AlterTable represents an ALTER TABLE statement.
type AlterTable struct {
	Table  string
	Action AlterAction
}

func (a *AlterTable) Type() StatementType { return AlterTableStmt }

// CreateType represents a CREATE TYPE statement (for enums).
type CreateType struct {
	Name   string
	Values []string
}

func (c *CreateType) Type() StatementType { return CreateTypeStmt }

// DropType represents a DROP TYPE statement.
type DropType struct {
	Name     string
	IfExists bool
	Cascade  bool
}

func (d *DropType) Type() StatementType { return DropTypeStmt }

// AlterType represents an ALTER TYPE statement.
type AlterType struct {
	Name   string
	Action AlterTypeAction
}

func (a *AlterType) Type() StatementType { return AlterTypeStmt }

// CreateIndex represents a CREATE INDEX statement.
type CreateIndex struct {
	Name         string
	Table        string
	Columns      []IndexColumnDef
	Unique       bool
	Method       string // btree, gin, gist, hash, etc.
	Where        string // Partial index predicate
	Concurrently bool
}

func (c *CreateIndex) Type() StatementType { return CreateIndexStmt }

// DropIndex represents a DROP INDEX statement.
type DropIndex struct {
	Name         string
	IfExists     bool
	Cascade      bool
	Concurrently bool
}

func (d *DropIndex) Type() StatementType { return DropIndexStmt }

// ColumnDef represents a column definition in CREATE TABLE.
type ColumnDef struct {
	Name        string
	Type        string // Raw SQL type: "TEXT", "INTEGER", "VARCHAR(255)"
	Nullable    bool
	Default     *string
	Constraints []ColumnConstraint
}

// ColumnConstraint represents column-level constraints.
type ColumnConstraint struct {
	Type       ConstraintType
	Name       string
	RefTable   string
	RefColumns []string
	OnDelete   string
	OnUpdate   string
	CheckExpr  string
}

// TableConstraint represents table-level constraints.
type TableConstraint struct {
	Type       ConstraintType
	Name       string
	Columns    []string
	RefTable   string
	RefColumns []string
	OnDelete   string
	OnUpdate   string
	CheckExpr  string
}

// ConstraintType identifies constraint kinds.
type ConstraintType string

const (
	PrimaryKeyConstraint ConstraintType = "PRIMARY_KEY"
	ForeignKeyConstraint ConstraintType = "FOREIGN_KEY"
	UniqueConstraint     ConstraintType = "UNIQUE"
	CheckConstraint      ConstraintType = "CHECK"
)

// IndexColumnDef represents a column in an index definition.
type IndexColumnDef struct {
	Name  string
	Order string // ASC, DESC, or empty
}

// AlterAction represents an action in ALTER TABLE.
type AlterAction interface {
	ActionType() AlterActionType
}

// AlterActionType identifies the kind of ALTER TABLE action.
type AlterActionType string

const (
	AddColumnAction    AlterActionType = "ADD_COLUMN"
	DropColumnAction   AlterActionType = "DROP_COLUMN"
	RenameColumnAction AlterActionType = "RENAME_COLUMN"
	AlterColumnAction  AlterActionType = "ALTER_COLUMN"
)

// AddColumn represents ADD COLUMN action.
type AddColumn struct {
	Column ColumnDef
}

func (a *AddColumn) ActionType() AlterActionType { return AddColumnAction }

// DropColumn represents DROP COLUMN action.
type DropColumn struct {
	Name     string
	IfExists bool
	Cascade  bool
}

func (d *DropColumn) ActionType() AlterActionType { return DropColumnAction }

// RenameColumn represents RENAME COLUMN action.
type RenameColumn struct {
	OldName string
	NewName string
}

func (r *RenameColumn) ActionType() AlterActionType { return RenameColumnAction }

// AlterColumn represents ALTER COLUMN action.
type AlterColumn struct {
	Name         string
	SetNotNull   bool
	DropNotNull  bool
	SetDefault   *string
	DropDefault  bool
	SetType      string
	SetCollation string
}

func (a *AlterColumn) ActionType() AlterActionType { return AlterColumnAction }

// AlterTypeAction represents an action in ALTER TYPE.
type AlterTypeAction interface {
	AlterTypeActionType() string
}

// AddEnumValue represents ADD VALUE action for enum types.
type AddEnumValue struct {
	Value  string
	Before string
	After  string
}

func (a *AddEnumValue) AlterTypeActionType() string { return "ADD_VALUE" }
