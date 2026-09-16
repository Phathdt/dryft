// Package schema defines the internal schema representation.
package schema

// Column represents a table column with its properties and constraints.
type Column struct {
	Name     string
	Type     DataType
	Nullable bool
	Default  *DefaultValue
	Comment  string
}

// DataType represents the data type specification for a column.
type DataType struct {
	// Kind is the base type category (e.g., TypeText, TypeInt32).
	Kind TypeKind

	// Precision specifies the total number of digits for numeric types
	// or the maximum length for character types.
	// Zero means no precision specified.
	Precision int

	// Scale specifies the number of digits after the decimal point for numeric types.
	// Zero means no scale specified.
	Scale int

	// ArrayDepth indicates the level of array nesting.
	// 0 means not an array, 1 means array, 2 means array of arrays, etc.
	ArrayDepth int

	// EnumName is the name of the enum type when Kind is TypeEnum.
	EnumName string
}

// DefaultValue represents the default value specification for a column.
type DefaultValue struct {
	// Kind indicates how the default is specified.
	Kind DefaultKind

	// Literal holds the literal value as a string (e.g., "true", "42", "'text'").
	// Used when Kind is DefaultLiteral.
	Literal string

	// Expression holds a SQL expression (e.g., "now()", "uuid_generate_v4()").
	// Used when Kind is DefaultExpression.
	Expression string

	// Sequence references a database sequence for auto-increment behavior.
	// Used when Kind is DefaultSequence.
	Sequence *SequenceRef
}

// SequenceRef references a database sequence used for column defaults.
type SequenceRef struct {
	// Name is the fully qualified or simple sequence name.
	Name string

	// Owned indicates whether the sequence is owned by this column
	// and should be dropped when the column is dropped.
	Owned bool
}
