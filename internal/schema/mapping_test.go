package schema

import (
	"strings"
	"testing"
)

func TestParsePostgreSQLType_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		pgType   string
		expected DataType
	}{
		{
			name:     "uuid",
			pgType:   "uuid",
			expected: DataType{Kind: TypeUUID},
		},
		{
			name:     "text",
			pgType:   "text",
			expected: DataType{Kind: TypeText},
		},
		{
			name:     "varchar",
			pgType:   "varchar",
			expected: DataType{Kind: TypeVarChar},
		},
		{
			name:     "character varying",
			pgType:   "character varying",
			expected: DataType{Kind: TypeVarChar},
		},
		{
			name:     "char",
			pgType:   "char",
			expected: DataType{Kind: TypeChar},
		},
		{
			name:     "integer",
			pgType:   "integer",
			expected: DataType{Kind: TypeInt32},
		},
		{
			name:     "int",
			pgType:   "int",
			expected: DataType{Kind: TypeInt32},
		},
		{
			name:     "int4",
			pgType:   "int4",
			expected: DataType{Kind: TypeInt32},
		},
		{
			name:     "bigint",
			pgType:   "bigint",
			expected: DataType{Kind: TypeInt64},
		},
		{
			name:     "int8",
			pgType:   "int8",
			expected: DataType{Kind: TypeInt64},
		},
		{
			name:     "boolean",
			pgType:   "boolean",
			expected: DataType{Kind: TypeBool},
		},
		{
			name:     "bool",
			pgType:   "bool",
			expected: DataType{Kind: TypeBool},
		},
		{
			name:     "real",
			pgType:   "real",
			expected: DataType{Kind: TypeFloat32},
		},
		{
			name:     "float4",
			pgType:   "float4",
			expected: DataType{Kind: TypeFloat32},
		},
		{
			name:     "double precision",
			pgType:   "double precision",
			expected: DataType{Kind: TypeFloat64},
		},
		{
			name:     "float8",
			pgType:   "float8",
			expected: DataType{Kind: TypeFloat64},
		},
		{
			name:     "numeric",
			pgType:   "numeric",
			expected: DataType{Kind: TypeNumeric},
		},
		{
			name:     "decimal",
			pgType:   "decimal",
			expected: DataType{Kind: TypeNumeric},
		},
		{
			name:     "timestamp",
			pgType:   "timestamp",
			expected: DataType{Kind: TypeTimestamp},
		},
		{
			name:     "timestamp without time zone",
			pgType:   "timestamp without time zone",
			expected: DataType{Kind: TypeTimestamp},
		},
		{
			name:     "timestamptz",
			pgType:   "timestamptz",
			expected: DataType{Kind: TypeTimestampTZ},
		},
		{
			name:     "timestamp with time zone",
			pgType:   "timestamp with time zone",
			expected: DataType{Kind: TypeTimestampTZ},
		},
		{
			name:     "date",
			pgType:   "date",
			expected: DataType{Kind: TypeDate},
		},
		{
			name:     "json",
			pgType:   "json",
			expected: DataType{Kind: TypeJSON},
		},
		{
			name:     "jsonb",
			pgType:   "jsonb",
			expected: DataType{Kind: TypeJSONB},
		},
		{
			name:     "bytea",
			pgType:   "bytea",
			expected: DataType{Kind: TypeBytes},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePostgreSQLType(tt.pgType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Kind != tt.expected.Kind {
				t.Errorf("expected Kind %v, got: %v", tt.expected.Kind, got.Kind)
			}
		})
	}
}

func TestParsePostgreSQLType_WithPrecision(t *testing.T) {
	tests := []struct {
		name      string
		pgType    string
		wantKind  TypeKind
		wantPrec  int
		wantScale int
	}{
		{
			name:     "varchar(100)",
			pgType:   "varchar(100)",
			wantKind: TypeVarChar,
			wantPrec: 100,
		},
		{
			name:     "char(10)",
			pgType:   "char(10)",
			wantKind: TypeChar,
			wantPrec: 10,
		},
		{
			name:      "numeric(10,2)",
			pgType:    "numeric(10,2)",
			wantKind:  TypeNumeric,
			wantPrec:  10,
			wantScale: 2,
		},
		{
			name:     "numeric(8)",
			pgType:   "numeric(8)",
			wantKind: TypeNumeric,
			wantPrec: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePostgreSQLType(tt.pgType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Kind != tt.wantKind {
				t.Errorf("expected Kind %v, got: %v", tt.wantKind, got.Kind)
			}
			if got.Precision != tt.wantPrec {
				t.Errorf("expected Precision %d, got: %d", tt.wantPrec, got.Precision)
			}
			if tt.wantScale > 0 && got.Scale != tt.wantScale {
				t.Errorf("expected Scale %d, got: %d", tt.wantScale, got.Scale)
			}
		})
	}
}

func TestParsePostgreSQLType_ArrayTypes(t *testing.T) {
	tests := []struct {
		name           string
		pgType         string
		wantKind       TypeKind
		wantArrayDepth int
	}{
		{
			name:           "text[]",
			pgType:         "text[]",
			wantKind:       TypeText,
			wantArrayDepth: 1,
		},
		{
			name:           "integer[]",
			pgType:         "integer[]",
			wantKind:       TypeInt32,
			wantArrayDepth: 1,
		},
		{
			name:           "text[][]",
			pgType:         "text[][]",
			wantKind:       TypeText,
			wantArrayDepth: 2,
		},
		{
			name:           "varchar(100)[]",
			pgType:         "varchar(100)[]",
			wantKind:       TypeVarChar,
			wantArrayDepth: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePostgreSQLType(tt.pgType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Kind != tt.wantKind {
				t.Errorf("expected Kind %v, got: %v", tt.wantKind, got.Kind)
			}
			if got.ArrayDepth != tt.wantArrayDepth {
				t.Errorf("expected ArrayDepth %d, got: %d", tt.wantArrayDepth, got.ArrayDepth)
			}
		})
	}
}

func TestParsePostgreSQLType_Errors(t *testing.T) {
	tests := []struct {
		name    string
		pgType  string
		wantErr string
	}{
		{
			name:    "empty string",
			pgType:  "",
			wantErr: "empty type string",
		},
		{
			name:    "unsupported type",
			pgType:  "geometry",
			wantErr: "unsupported PostgreSQL type",
		},
		{
			name:    "invalid precision format",
			pgType:  "varchar(abc)",
			wantErr: "invalid precision format",
		},
		{
			name:    "invalid precision/scale format",
			pgType:  "numeric(10,x)",
			wantErr: "invalid precision/scale format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePostgreSQLType(tt.pgType)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestDataType_ToPrisma(t *testing.T) {
	tests := []struct {
		name     string
		dt       DataType
		expected string
	}{
		{
			name:     "UUID",
			dt:       DataType{Kind: TypeUUID},
			expected: "String @db.Uuid",
		},
		{
			name:     "Text",
			dt:       DataType{Kind: TypeText},
			expected: "String",
		},
		{
			name:     "VarChar without precision",
			dt:       DataType{Kind: TypeVarChar},
			expected: "String @db.VarChar",
		},
		{
			name:     "VarChar with precision",
			dt:       DataType{Kind: TypeVarChar, Precision: 100},
			expected: "String @db.VarChar(100)",
		},
		{
			name:     "Int32",
			dt:       DataType{Kind: TypeInt32},
			expected: "Int",
		},
		{
			name:     "Int64",
			dt:       DataType{Kind: TypeInt64},
			expected: "BigInt",
		},
		{
			name:     "Bool",
			dt:       DataType{Kind: TypeBool},
			expected: "Boolean",
		},
		{
			name:     "Float32",
			dt:       DataType{Kind: TypeFloat32},
			expected: "Float @db.Real",
		},
		{
			name:     "Float64",
			dt:       DataType{Kind: TypeFloat64},
			expected: "Float",
		},
		{
			name:     "Numeric with precision and scale",
			dt:       DataType{Kind: TypeNumeric, Precision: 10, Scale: 2},
			expected: "Decimal @db.Numeric(10,2)",
		},
		{
			name:     "Timestamp",
			dt:       DataType{Kind: TypeTimestamp},
			expected: "DateTime",
		},
		{
			name:     "TimestampTZ",
			dt:       DataType{Kind: TypeTimestampTZ},
			expected: "DateTime @db.Timestamptz",
		},
		{
			name:     "Date",
			dt:       DataType{Kind: TypeDate},
			expected: "DateTime @db.Date",
		},
		{
			name:     "JSON",
			dt:       DataType{Kind: TypeJSON},
			expected: "Json",
		},
		{
			name:     "JSONB",
			dt:       DataType{Kind: TypeJSONB},
			expected: "Json @db.JsonB",
		},
		{
			name:     "Bytes",
			dt:       DataType{Kind: TypeBytes},
			expected: "Bytes",
		},
		{
			name:     "Text array",
			dt:       DataType{Kind: TypeText, ArrayDepth: 1},
			expected: "String[]",
		},
		{
			name:     "Int array",
			dt:       DataType{Kind: TypeInt32, ArrayDepth: 1},
			expected: "Int[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dt.ToPrisma()
			if got != tt.expected {
				t.Errorf("expected %q, got: %q", tt.expected, got)
			}
		})
	}
}

func TestDataType_ToPostgreSQLDDL(t *testing.T) {
	tests := []struct {
		name     string
		dt       DataType
		expected string
	}{
		{
			name:     "UUID",
			dt:       DataType{Kind: TypeUUID},
			expected: "UUID",
		},
		{
			name:     "TEXT",
			dt:       DataType{Kind: TypeText},
			expected: "TEXT",
		},
		{
			name:     "VARCHAR without precision",
			dt:       DataType{Kind: TypeVarChar},
			expected: "VARCHAR",
		},
		{
			name:     "VARCHAR with precision",
			dt:       DataType{Kind: TypeVarChar, Precision: 255},
			expected: "VARCHAR(255)",
		},
		{
			name:     "INTEGER",
			dt:       DataType{Kind: TypeInt32},
			expected: "INTEGER",
		},
		{
			name:     "BIGINT",
			dt:       DataType{Kind: TypeInt64},
			expected: "BIGINT",
		},
		{
			name:     "BOOLEAN",
			dt:       DataType{Kind: TypeBool},
			expected: "BOOLEAN",
		},
		{
			name:     "REAL",
			dt:       DataType{Kind: TypeFloat32},
			expected: "REAL",
		},
		{
			name:     "DOUBLE PRECISION",
			dt:       DataType{Kind: TypeFloat64},
			expected: "DOUBLE PRECISION",
		},
		{
			name:     "NUMERIC with precision and scale",
			dt:       DataType{Kind: TypeNumeric, Precision: 10, Scale: 2},
			expected: "NUMERIC(10,2)",
		},
		{
			name:     "TIMESTAMP",
			dt:       DataType{Kind: TypeTimestamp},
			expected: "TIMESTAMP",
		},
		{
			name:     "TIMESTAMPTZ",
			dt:       DataType{Kind: TypeTimestampTZ},
			expected: "TIMESTAMPTZ",
		},
		{
			name:     "DATE",
			dt:       DataType{Kind: TypeDate},
			expected: "DATE",
		},
		{
			name:     "JSON",
			dt:       DataType{Kind: TypeJSON},
			expected: "JSON",
		},
		{
			name:     "JSONB",
			dt:       DataType{Kind: TypeJSONB},
			expected: "JSONB",
		},
		{
			name:     "BYTEA",
			dt:       DataType{Kind: TypeBytes},
			expected: "BYTEA",
		},
		{
			name:     "TEXT array",
			dt:       DataType{Kind: TypeText, ArrayDepth: 1},
			expected: "TEXT[]",
		},
		{
			name:     "INTEGER array",
			dt:       DataType{Kind: TypeInt32, ArrayDepth: 1},
			expected: "INTEGER[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dt.ToPostgreSQLDDL()
			if got != tt.expected {
				t.Errorf("expected %q, got: %q", tt.expected, got)
			}
		})
	}
}

func TestTypeMapping_RoundTrip_PostgreSQLToInternal(t *testing.T) {
	tests := []struct {
		name   string
		pgType string
	}{
		{"uuid", "uuid"},
		{"text", "text"},
		{"varchar(100)", "varchar(100)"},
		{"integer", "integer"},
		{"bigint", "bigint"},
		{"boolean", "boolean"},
		{"real", "real"},
		{"double precision", "double precision"},
		{"numeric(10,2)", "numeric(10,2)"},
		{"timestamp", "timestamp"},
		{"timestamptz", "timestamptz"},
		{"date", "date"},
		{"json", "json"},
		{"jsonb", "jsonb"},
		{"bytea", "bytea"},
		{"text[]", "text[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse PostgreSQL type to internal
			dt, err := ParsePostgreSQLType(tt.pgType)
			if err != nil {
				t.Fatalf("failed to parse %q: %v", tt.pgType, err)
			}

			// Convert back to PostgreSQL DDL
			ddl := dt.ToPostgreSQLDDL()

			// Parse again
			dt2, err := ParsePostgreSQLType(strings.ToLower(ddl))
			if err != nil {
				t.Fatalf("failed to parse back %q: %v", ddl, err)
			}

			// Compare
			if dt.Kind != dt2.Kind {
				t.Errorf("Kind mismatch: original=%v, round-trip=%v", dt.Kind, dt2.Kind)
			}
			if dt.Precision != dt2.Precision {
				t.Errorf("Precision mismatch: original=%d, round-trip=%d", dt.Precision, dt2.Precision)
			}
			if dt.Scale != dt2.Scale {
				t.Errorf("Scale mismatch: original=%d, round-trip=%d", dt.Scale, dt2.Scale)
			}
			if dt.ArrayDepth != dt2.ArrayDepth {
				t.Errorf("ArrayDepth mismatch: original=%d, round-trip=%d", dt.ArrayDepth, dt2.ArrayDepth)
			}
		})
	}
}

func TestTypeMapping_PrecisionScalePreservation(t *testing.T) {
	dt := DataType{
		Kind:      TypeNumeric,
		Precision: 18,
		Scale:     4,
	}

	// To Prisma
	prisma := dt.ToPrisma()
	if !strings.Contains(prisma, "18,4") {
		t.Errorf("Prisma output should contain precision/scale, got: %s", prisma)
	}

	// To PostgreSQL
	ddl := dt.ToPostgreSQLDDL()
	if !strings.Contains(ddl, "18,4") {
		t.Errorf("DDL output should contain precision/scale, got: %s", ddl)
	}
}

func TestTypeMapping_ArrayHandling(t *testing.T) {
	dt := DataType{
		Kind:       TypeInt32,
		ArrayDepth: 2,
	}

	prisma := dt.ToPrisma()
	if prisma != "Int[][]" {
		t.Errorf("expected 'Int[][]', got: %s", prisma)
	}

	ddl := dt.ToPostgreSQLDDL()
	if ddl != "INTEGER[][]" {
		t.Errorf("expected 'INTEGER[][]', got: %s", ddl)
	}
}

func TestDataType_ToPrisma_Enum(t *testing.T) {
	dt := DataType{
		Kind: TypeEnum,
	}

	// Enum type currently maps to String in Prisma
	// EnumName field exists for future use but is not currently used in ToPrisma
	prisma := dt.ToPrisma()
	if prisma != "String" {
		t.Errorf("expected 'String' for enum type, got: %s", prisma)
	}
}

func TestDataType_ToPrisma_UnknownType(t *testing.T) {
	dt := DataType{
		Kind: TypeUnknown,
	}

	prisma := dt.ToPrisma()
	if prisma != "String" {
		t.Errorf("expected 'String' for unknown type, got: %s", prisma)
	}
}

func TestDataType_ToPostgreSQLDDL_Enum(t *testing.T) {
	dt := DataType{
		Kind: TypeEnum,
	}

	// Enum type currently maps to TEXT in PostgreSQL DDL
	// EnumName field exists for future use but is not currently used in ToPostgreSQLDDL
	ddl := dt.ToPostgreSQLDDL()
	if ddl != "TEXT" {
		t.Errorf("expected 'TEXT' for enum type, got: %s", ddl)
	}
}

func TestDataType_ToPostgreSQLDDL_UnknownType(t *testing.T) {
	dt := DataType{
		Kind: TypeUnknown,
	}

	ddl := dt.ToPostgreSQLDDL()
	if ddl != "TEXT" {
		t.Errorf("expected 'TEXT' for unknown type, got: %s", ddl)
	}
}

func TestDataType_ToPrisma_NumericWithPrecisionOnly(t *testing.T) {
	dt := DataType{
		Kind:      TypeNumeric,
		Precision: 10,
		Scale:     0,
	}

	prisma := dt.ToPrisma()
	if !strings.Contains(prisma, "Decimal") || !strings.Contains(prisma, "@db.Numeric(10)") {
		t.Errorf("expected Numeric with precision only, got: %s", prisma)
	}
}

func TestDataType_ToPostgreSQLDDL_NumericWithPrecisionOnly(t *testing.T) {
	dt := DataType{
		Kind:      TypeNumeric,
		Precision: 8,
		Scale:     0,
	}

	ddl := dt.ToPostgreSQLDDL()
	if ddl != "NUMERIC(8)" {
		t.Errorf("expected 'NUMERIC(8)', got: %s", ddl)
	}
}

func TestDataType_ToPostgreSQLDDL_CharWithoutPrecision(t *testing.T) {
	dt := DataType{
		Kind:      TypeChar,
		Precision: 0,
	}

	ddl := dt.ToPostgreSQLDDL()
	if ddl != "CHAR" {
		t.Errorf("expected 'CHAR', got: %s", ddl)
	}
}

func TestDataType_ToPrisma_CharWithoutPrecision(t *testing.T) {
	dt := DataType{
		Kind: TypeChar,
	}

	prisma := dt.ToPrisma()
	if !strings.Contains(prisma, "String") || !strings.Contains(prisma, "@db.Char") {
		t.Errorf("expected String with @db.Char, got: %s", prisma)
	}
}
