package schema

// PrimaryKey represents a primary key constraint on a table.
type PrimaryKey struct {
	// Columns is the list of column names that form the primary key.
	Columns []string

	// Name is the constraint name. If empty, the database will generate one.
	Name string
}

// ForeignKey represents a foreign key constraint that establishes a relationship
// between tables by referencing columns in another table.
type ForeignKey struct {
	// Name is the constraint name. If empty, the database will generate one.
	Name string

	// Columns is the list of column names in the current table.
	Columns []string

	// RefTable is the name of the referenced table.
	RefTable string

	// RefColumns is the list of column names in the referenced table.
	RefColumns []string

	// OnDelete specifies the action to take when the referenced row is deleted.
	OnDelete ReferentialAction

	// OnUpdate specifies the action to take when the referenced row is updated.
	OnUpdate ReferentialAction

	// Deferrable indicates whether constraint checking can be deferred until
	// the end of the transaction.
	Deferrable bool

	// InitiallyDeferred indicates whether the constraint is initially deferred.
	// Only applies if Deferrable is true.
	InitiallyDeferred bool
}

// ReferentialAction defines the action to take when a referenced row is modified.
type ReferentialAction int

const (
	// ActionNone indicates no referential action is specified (use database default).
	ActionNone ReferentialAction = iota

	// ActionNoAction produces an error if the constraint is violated.
	ActionNoAction

	// ActionRestrict produces an immediate error if the constraint is violated.
	ActionRestrict

	// ActionCascade propagates the operation (DELETE or UPDATE) to dependent rows.
	ActionCascade

	// ActionSetNull sets the foreign key columns to NULL.
	ActionSetNull

	// ActionSetDefault sets the foreign key columns to their default values.
	ActionSetDefault
)

// String returns the SQL representation of the referential action.
func (r ReferentialAction) String() string {
	switch r {
	case ActionNone:
		return ""
	case ActionNoAction:
		return "NO ACTION"
	case ActionRestrict:
		return "RESTRICT"
	case ActionCascade:
		return "CASCADE"
	case ActionSetNull:
		return "SET NULL"
	case ActionSetDefault:
		return "SET DEFAULT"
	default:
		return "UNKNOWN"
	}
}

// Constraint represents a table-level constraint such as CHECK, UNIQUE, or EXCLUDE.
type Constraint struct {
	// Name is the constraint name. If empty, the database will generate one.
	Name string

	// Type specifies the kind of constraint (CHECK, UNIQUE, EXCLUDE).
	Type ConstraintType

	// Columns is the list of column names involved in the constraint.
	// Not used for CHECK constraints with expressions.
	Columns []string

	// Expression is the constraint expression (e.g., "age >= 0" for CHECK).
	Expression string

	// Comment is an optional description of the constraint's purpose.
	Comment string
}

// ConstraintType represents the kind of table constraint.
type ConstraintType int

const (
	// ConstraintCheck verifies that column values satisfy a boolean expression.
	ConstraintCheck ConstraintType = iota

	// ConstraintUnique ensures that values in specified columns are unique.
	ConstraintUnique

	// ConstraintExclude prevents rows from having overlapping values (PostgreSQL extension).
	ConstraintExclude
)

// String returns the string representation of the constraint type.
func (c ConstraintType) String() string {
	switch c {
	case ConstraintCheck:
		return "CHECK"
	case ConstraintUnique:
		return "UNIQUE"
	case ConstraintExclude:
		return "EXCLUDE"
	default:
		return "UNKNOWN"
	}
}
