package schema

// Schema represents the complete database schema with all its components.
type Schema struct {
	Tables []Table
	Enums  []Enum
}

// Table represents a database table with its structure and constraints.
type Table struct {
	// Name is the table name.
	Name string

	// Columns lists all columns in the table.
	Columns []Column

	// PrimaryKey defines the primary key constraint.
	PrimaryKey *PrimaryKey

	// ForeignKeys lists all foreign key constraints.
	ForeignKeys []ForeignKey

	// Indexes lists all indexes (excluding primary key).
	Indexes []Index

	// Constraints lists check constraints and other table-level constraints.
	Constraints []Constraint

	// Comment is the table-level comment.
	Comment string
}

// Enum represents a database enum type.
type Enum struct {
	// Name is the enum type name.
	Name string

	// Values lists all enum values in order.
	Values []EnumValue
}

// EnumValue represents a single value in an enum type.
type EnumValue struct {
	// Label is the enum value label.
	Label string

	// Order specifies the position in the enum (for ordering).
	Order int
}
