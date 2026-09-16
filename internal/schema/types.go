package schema

// TypeKind represents the data type category for a column.
type TypeKind int

const (
	// TypeUnknown represents an unknown or unsupported type.
	TypeUnknown TypeKind = iota
	// TypeUUID represents a UUID type.
	TypeUUID
	// TypeText represents unlimited text.
	TypeText
	// TypeVarChar represents variable-length character type.
	TypeVarChar
	// TypeChar represents fixed-length character type.
	TypeChar
	// TypeInt32 represents a 32-bit integer.
	TypeInt32
	// TypeInt64 represents a 64-bit integer.
	TypeInt64
	// TypeBool represents a boolean type.
	TypeBool
	// TypeFloat32 represents a 32-bit floating point.
	TypeFloat32
	// TypeFloat64 represents a 64-bit floating point.
	TypeFloat64
	// TypeNumeric represents arbitrary precision numeric.
	TypeNumeric
	// TypeTimestamp represents a timestamp without timezone.
	TypeTimestamp
	// TypeTimestampTZ represents a timestamp with timezone.
	TypeTimestampTZ
	// TypeDate represents a date type.
	TypeDate
	// TypeJSON represents JSON data.
	TypeJSON
	// TypeJSONB represents binary JSON data.
	TypeJSONB
	// TypeBytes represents binary data.
	TypeBytes
	// TypeEnum represents an enum type.
	TypeEnum
)

// String returns the string representation of TypeKind.
func (t TypeKind) String() string {
	switch t {
	case TypeUUID:
		return "UUID"
	case TypeText:
		return "Text"
	case TypeVarChar:
		return "VarChar"
	case TypeChar:
		return "Char"
	case TypeInt32:
		return "Int32"
	case TypeInt64:
		return "Int64"
	case TypeBool:
		return "Bool"
	case TypeFloat32:
		return "Float32"
	case TypeFloat64:
		return "Float64"
	case TypeNumeric:
		return "Numeric"
	case TypeTimestamp:
		return "Timestamp"
	case TypeTimestampTZ:
		return "TimestampTZ"
	case TypeDate:
		return "Date"
	case TypeJSON:
		return "JSON"
	case TypeJSONB:
		return "JSONB"
	case TypeBytes:
		return "Bytes"
	case TypeEnum:
		return "Enum"
	default:
		return "Unknown"
	}
}

// DefaultKind represents how a default value is specified.
type DefaultKind int

const (
	// DefaultLiteral represents a literal default value.
	DefaultLiteral DefaultKind = iota
	// DefaultExpression represents an expression for default value.
	DefaultExpression
	// DefaultSequence represents a sequence-generated default.
	DefaultSequence
)

// String returns the string representation of DefaultKind.
func (d DefaultKind) String() string {
	switch d {
	case DefaultLiteral:
		return "Literal"
	case DefaultExpression:
		return "Expression"
	case DefaultSequence:
		return "Sequence"
	default:
		return "Unknown"
	}
}

// SortOrder represents the sort direction for indexes.
type SortOrder int

const (
	// SortAsc represents ascending sort order.
	SortAsc SortOrder = iota
	// SortDesc represents descending sort order.
	SortDesc
)

// String returns the string representation of SortOrder.
func (s SortOrder) String() string {
	switch s {
	case SortAsc:
		return "ASC"
	case SortDesc:
		return "DESC"
	default:
		return "ASC"
	}
}

// NullsPosition represents the positioning of null values in sorted results.
type NullsPosition int

const (
	// NullsDefault uses database default null positioning.
	NullsDefault NullsPosition = iota
	// NullsFirst positions nulls before non-null values.
	NullsFirst
	// NullsLast positions nulls after non-null values.
	NullsLast
)

// String returns the string representation of NullsPosition.
func (n NullsPosition) String() string {
	switch n {
	case NullsFirst:
		return "NULLS FIRST"
	case NullsLast:
		return "NULLS LAST"
	default:
		return ""
	}
}
