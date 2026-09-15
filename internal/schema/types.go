package schema

// TypeKind represents the data type category for a column.
type TypeKind int

const (
	TypeUnknown TypeKind = iota
	TypeUUID
	TypeText
	TypeVarChar
	TypeChar
	TypeInt32
	TypeInt64
	TypeBool
	TypeFloat32
	TypeFloat64
	TypeNumeric
	TypeTimestamp
	TypeTimestampTZ
	TypeDate
	TypeJSON
	TypeJSONB
	TypeBytes
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
	DefaultLiteral DefaultKind = iota
	DefaultExpression
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
	SortAsc SortOrder = iota
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
	NullsDefault NullsPosition = iota
	NullsFirst
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
