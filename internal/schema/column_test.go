package schema

import (
	"strings"
	"testing"
)

func TestColumn_Validate_ValidColumns(t *testing.T) {
	tests := []struct {
		name string
		col  Column
	}{
		{
			name: "UUID column",
			col:  Column{Name: "id", Type: DataType{Kind: TypeUUID}},
		},
		{
			name: "Text column",
			col:  Column{Name: "description", Type: DataType{Kind: TypeText}},
		},
		{
			name: "VarChar with precision",
			col:  Column{Name: "email", Type: DataType{Kind: TypeVarChar, Precision: 255}},
		},
		{
			name: "Int32 column",
			col:  Column{Name: "age", Type: DataType{Kind: TypeInt32}},
		},
		{
			name: "Int64 column",
			col:  Column{Name: "count", Type: DataType{Kind: TypeInt64}},
		},
		{
			name: "Bool column",
			col:  Column{Name: "active", Type: DataType{Kind: TypeBool}},
		},
		{
			name: "Float32 column",
			col:  Column{Name: "rating", Type: DataType{Kind: TypeFloat32}},
		},
		{
			name: "Float64 column",
			col:  Column{Name: "price", Type: DataType{Kind: TypeFloat64}},
		},
		{
			name: "Numeric with precision and scale",
			col:  Column{Name: "amount", Type: DataType{Kind: TypeNumeric, Precision: 10, Scale: 2}},
		},
		{
			name: "Timestamp column",
			col:  Column{Name: "created_at", Type: DataType{Kind: TypeTimestamp}},
		},
		{
			name: "TimestampTZ column",
			col:  Column{Name: "updated_at", Type: DataType{Kind: TypeTimestampTZ}},
		},
		{
			name: "Date column",
			col:  Column{Name: "birth_date", Type: DataType{Kind: TypeDate}},
		},
		{
			name: "JSON column",
			col:  Column{Name: "metadata", Type: DataType{Kind: TypeJSON}},
		},
		{
			name: "JSONB column",
			col:  Column{Name: "data", Type: DataType{Kind: TypeJSONB}},
		},
		{
			name: "Bytes column",
			col:  Column{Name: "file_data", Type: DataType{Kind: TypeBytes}},
		},
		{
			name: "Enum column",
			col:  Column{Name: "status", Type: DataType{Kind: TypeEnum}},
		},
		{
			name: "Nullable column",
			col:  Column{Name: "optional_field", Type: DataType{Kind: TypeText}, Nullable: true},
		},
		{
			name: "Column with comment",
			col:  Column{Name: "id", Type: DataType{Kind: TypeUUID}, Comment: "Primary identifier"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.col.Validate(); err != nil {
				t.Errorf("expected valid column, got error: %v", err)
			}
		})
	}
}

func TestColumn_Validate_EmptyName(t *testing.T) {
	col := Column{
		Name: "",
		Type: DataType{Kind: TypeText},
	}

	err := col.Validate()
	if err == nil {
		t.Fatal("expected error for empty column name, got nil")
	}

	if !strings.Contains(err.Error(), "name cannot be empty") {
		t.Errorf("expected 'name cannot be empty' in error, got: %v", err)
	}
}

func TestColumn_Validate_WhitespaceOnlyName(t *testing.T) {
	col := Column{
		Name: "   ",
		Type: DataType{Kind: TypeText},
	}

	err := col.Validate()
	if err == nil {
		t.Fatal("expected error for whitespace-only column name, got nil")
	}

	if !strings.Contains(err.Error(), "name cannot be empty") {
		t.Errorf("expected 'name cannot be empty' in error, got: %v", err)
	}
}

func TestColumn_Validate_UnknownType(t *testing.T) {
	col := Column{
		Name: "broken_column",
		Type: DataType{Kind: TypeUnknown},
	}

	err := col.Validate()
	if err == nil {
		t.Fatal("expected error for TypeUnknown, got nil")
	}

	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("expected 'unknown type' in error, got: %v", err)
	}
}

func TestColumn_WithDefaultLiteral(t *testing.T) {
	col := Column{
		Name: "status",
		Type: DataType{Kind: TypeText},
		Default: &DefaultValue{
			Kind:    DefaultLiteral,
			Literal: "'active'",
		},
	}

	if err := col.Validate(); err != nil {
		t.Errorf("expected valid column with default, got error: %v", err)
	}

	if col.Default.Literal != "'active'" {
		t.Errorf("expected default literal 'active', got: %s", col.Default.Literal)
	}
}

func TestColumn_WithDefaultExpression(t *testing.T) {
	col := Column{
		Name: "created_at",
		Type: DataType{Kind: TypeTimestampTZ},
		Default: &DefaultValue{
			Kind:       DefaultExpression,
			Expression: "now()",
		},
	}

	if err := col.Validate(); err != nil {
		t.Errorf("expected valid column with default expression, got error: %v", err)
	}

	if col.Default.Expression != "now()" {
		t.Errorf("expected default expression 'now()', got: %s", col.Default.Expression)
	}
}

func TestColumn_WithDefaultSequence(t *testing.T) {
	col := Column{
		Name: "id",
		Type: DataType{Kind: TypeInt32},
		Default: &DefaultValue{
			Kind: DefaultSequence,
			Sequence: &SequenceRef{
				Name:  "users_id_seq",
				Owned: true,
			},
		},
	}

	if err := col.Validate(); err != nil {
		t.Errorf("expected valid column with sequence, got error: %v", err)
	}

	if col.Default.Sequence.Name != "users_id_seq" {
		t.Errorf("expected sequence name 'users_id_seq', got: %s", col.Default.Sequence.Name)
	}
	if !col.Default.Sequence.Owned {
		t.Error("expected sequence to be owned")
	}
}

func TestDataType_WithPrecision(t *testing.T) {
	dt := DataType{
		Kind:      TypeVarChar,
		Precision: 100,
	}

	if dt.Kind != TypeVarChar {
		t.Errorf("expected TypeVarChar, got: %v", dt.Kind)
	}
	if dt.Precision != 100 {
		t.Errorf("expected precision 100, got: %v", dt.Precision)
	}
}

func TestDataType_WithPrecisionAndScale(t *testing.T) {
	dt := DataType{
		Kind:      TypeNumeric,
		Precision: 10,
		Scale:     2,
	}

	if dt.Kind != TypeNumeric {
		t.Errorf("expected TypeNumeric, got: %v", dt.Kind)
	}
	if dt.Precision != 10 {
		t.Errorf("expected precision 10, got: %v", dt.Precision)
	}
	if dt.Scale != 2 {
		t.Errorf("expected scale 2, got: %v", dt.Scale)
	}
}

func TestDataType_ArrayTypes(t *testing.T) {
	tests := []struct {
		name       string
		dt         DataType
		arrayDepth int
	}{
		{
			name:       "scalar text",
			dt:         DataType{Kind: TypeText, ArrayDepth: 0},
			arrayDepth: 0,
		},
		{
			name:       "text array",
			dt:         DataType{Kind: TypeText, ArrayDepth: 1},
			arrayDepth: 1,
		},
		{
			name:       "text array of arrays",
			dt:         DataType{Kind: TypeText, ArrayDepth: 2},
			arrayDepth: 2,
		},
		{
			name:       "int array",
			dt:         DataType{Kind: TypeInt32, ArrayDepth: 1},
			arrayDepth: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dt.ArrayDepth != tt.arrayDepth {
				t.Errorf("expected ArrayDepth %d, got: %d", tt.arrayDepth, tt.dt.ArrayDepth)
			}
		})
	}
}

func TestTypeKind_String(t *testing.T) {
	tests := []struct {
		kind     TypeKind
		expected string
	}{
		{TypeUUID, "UUID"},
		{TypeText, "Text"},
		{TypeVarChar, "VarChar"},
		{TypeChar, "Char"},
		{TypeInt32, "Int32"},
		{TypeInt64, "Int64"},
		{TypeBool, "Bool"},
		{TypeFloat32, "Float32"},
		{TypeFloat64, "Float64"},
		{TypeNumeric, "Numeric"},
		{TypeTimestamp, "Timestamp"},
		{TypeTimestampTZ, "TimestampTZ"},
		{TypeDate, "Date"},
		{TypeJSON, "JSON"},
		{TypeJSONB, "JSONB"},
		{TypeBytes, "Bytes"},
		{TypeEnum, "Enum"},
		{TypeUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.expected {
				t.Errorf("expected %s, got: %s", tt.expected, got)
			}
		})
	}
}

func TestDefaultKind_String(t *testing.T) {
	tests := []struct {
		kind     DefaultKind
		expected string
	}{
		{DefaultLiteral, "Literal"},
		{DefaultExpression, "Expression"},
		{DefaultSequence, "Sequence"},
		{DefaultKind(999), "Unknown"}, // unknown kind
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.expected {
				t.Errorf("expected %s, got: %s", tt.expected, got)
			}
		})
	}
}

func TestSortOrder_String(t *testing.T) {
	tests := []struct {
		order    SortOrder
		expected string
	}{
		{SortAsc, "ASC"},
		{SortDesc, "DESC"},
		{SortOrder(999), "ASC"}, // unknown defaults to ASC
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.order.String(); got != tt.expected {
				t.Errorf("expected %s, got: %s", tt.expected, got)
			}
		})
	}
}

func TestNullsPosition_String(t *testing.T) {
	tests := []struct {
		pos      NullsPosition
		expected string
	}{
		{NullsDefault, ""},
		{NullsFirst, "NULLS FIRST"},
		{NullsLast, "NULLS LAST"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.pos.String(); got != tt.expected {
				t.Errorf("expected %q, got: %q", tt.expected, got)
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
