package migration

import (
	"testing"

	"github.com/phathdt/dryft/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapSQLType_IntegerTypes(t *testing.T) {
	enums := make(map[string]*schema.Enum)

	tests := []struct {
		sqlType  string
		expected schema.TypeKind
	}{
		{"INTEGER", schema.TypeInt32},
		{"INT", schema.TypeInt32},
		{"INT4", schema.TypeInt32},
		{"BIGINT", schema.TypeInt64},
		{"INT8", schema.TypeInt64},
		{"SMALLINT", schema.TypeInt32},
		{"INT2", schema.TypeInt32},
		{"SERIAL", schema.TypeInt32},
		{"SERIAL4", schema.TypeInt32},
		{"BIGSERIAL", schema.TypeInt64},
		{"SERIAL8", schema.TypeInt64},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			result, err := mapSQLType(tt.sqlType, enums)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Kind)
			assert.Equal(t, 0, result.ArrayDepth)
		})
	}
}

func TestMapSQLType_TextTypes(t *testing.T) {
	enums := make(map[string]*schema.Enum)

	tests := []struct {
		sqlType  string
		expected schema.TypeKind
	}{
		{"TEXT", schema.TypeText},
		{"VARCHAR", schema.TypeVarChar},
		{"CHARACTER VARYING", schema.TypeVarChar},
		{"CHAR", schema.TypeChar},
		{"CHARACTER", schema.TypeChar},
		{"BPCHAR", schema.TypeChar},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			result, err := mapSQLType(tt.sqlType, enums)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Kind)
		})
	}
}

func TestMapSQLType_WithPrecision(t *testing.T) {
	enums := make(map[string]*schema.Enum)

	tests := []struct {
		sqlType       string
		expectedKind  schema.TypeKind
		expectedPrec  int
		expectedScale int
	}{
		{"VARCHAR(255)", schema.TypeVarChar, 255, 0},
		{"CHAR(10)", schema.TypeChar, 10, 0},
		{"NUMERIC(10,2)", schema.TypeNumeric, 10, 2},
		{"DECIMAL(8, 3)", schema.TypeNumeric, 8, 3},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			result, err := mapSQLType(tt.sqlType, enums)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedKind, result.Kind)
			assert.Equal(t, tt.expectedPrec, result.Precision)
			assert.Equal(t, tt.expectedScale, result.Scale)
		})
	}
}

func TestMapSQLType_UUID(t *testing.T) {
	enums := make(map[string]*schema.Enum)
	result, err := mapSQLType("UUID", enums)
	require.NoError(t, err)
	assert.Equal(t, schema.TypeUUID, result.Kind)
}

func TestMapSQLType_Boolean(t *testing.T) {
	enums := make(map[string]*schema.Enum)

	tests := []string{"BOOLEAN", "BOOL"}
	for _, sqlType := range tests {
		t.Run(sqlType, func(t *testing.T) {
			result, err := mapSQLType(sqlType, enums)
			require.NoError(t, err)
			assert.Equal(t, schema.TypeBool, result.Kind)
		})
	}
}

func TestMapSQLType_FloatingPoint(t *testing.T) {
	enums := make(map[string]*schema.Enum)

	tests := []struct {
		sqlType  string
		expected schema.TypeKind
	}{
		{"REAL", schema.TypeFloat32},
		{"FLOAT4", schema.TypeFloat32},
		{"DOUBLE PRECISION", schema.TypeFloat64},
		{"FLOAT8", schema.TypeFloat64},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			result, err := mapSQLType(tt.sqlType, enums)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Kind)
		})
	}
}

func TestMapSQLType_Numeric(t *testing.T) {
	enums := make(map[string]*schema.Enum)

	tests := []string{"NUMERIC", "DECIMAL"}
	for _, sqlType := range tests {
		t.Run(sqlType, func(t *testing.T) {
			result, err := mapSQLType(sqlType, enums)
			require.NoError(t, err)
			assert.Equal(t, schema.TypeNumeric, result.Kind)
		})
	}
}

func TestMapSQLType_TimestampTypes(t *testing.T) {
	enums := make(map[string]*schema.Enum)

	tests := []struct {
		sqlType  string
		expected schema.TypeKind
	}{
		{"TIMESTAMP", schema.TypeTimestamp},
		{"TIMESTAMP WITHOUT TIME ZONE", schema.TypeTimestamp},
		{"TIMESTAMP WITH TIME ZONE", schema.TypeTimestampTZ},
		{"TIMESTAMPTZ", schema.TypeTimestampTZ},
		{"DATE", schema.TypeDate},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			result, err := mapSQLType(tt.sqlType, enums)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Kind)
		})
	}
}

func TestMapSQLType_JSONTypes(t *testing.T) {
	enums := make(map[string]*schema.Enum)

	tests := []struct {
		sqlType  string
		expected schema.TypeKind
	}{
		{"JSON", schema.TypeJSON},
		{"JSONB", schema.TypeJSONB},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			result, err := mapSQLType(tt.sqlType, enums)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Kind)
		})
	}
}

func TestMapSQLType_Binary(t *testing.T) {
	enums := make(map[string]*schema.Enum)
	result, err := mapSQLType("BYTEA", enums)
	require.NoError(t, err)
	assert.Equal(t, schema.TypeBytes, result.Kind)
}

func TestMapSQLType_EnumTypes(t *testing.T) {
	enums := map[string]*schema.Enum{
		"user_role": {
			Name: "user_role",
			Values: []schema.EnumValue{
				{Label: "admin", Order: 0},
				{Label: "user", Order: 1},
			},
		},
	}

	result, err := mapSQLType("user_role", enums)
	require.NoError(t, err)
	assert.Equal(t, schema.TypeEnum, result.Kind)
	assert.Equal(t, "user_role", result.EnumName)
}

func TestMapSQLType_ArrayTypes(t *testing.T) {
	enums := make(map[string]*schema.Enum)

	tests := []struct {
		sqlType       string
		expectedKind  schema.TypeKind
		expectedDepth int
	}{
		{"TEXT[]", schema.TypeText, 1},
		{"INTEGER[]", schema.TypeInt32, 1},
		{"TEXT[][]", schema.TypeText, 2},
		{"UUID[]", schema.TypeUUID, 1},
		{"JSONB[]", schema.TypeJSONB, 1},
	}

	for _, tt := range tests {
		t.Run(tt.sqlType, func(t *testing.T) {
			result, err := mapSQLType(tt.sqlType, enums)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedKind, result.Kind)
			assert.Equal(t, tt.expectedDepth, result.ArrayDepth)
		})
	}
}

func TestMapSQLType_CaseInsensitive(t *testing.T) {
	enums := make(map[string]*schema.Enum)

	tests := []string{
		"text",
		"TEXT",
		"Text",
		"TeXt",
	}

	for _, sqlType := range tests {
		t.Run(sqlType, func(t *testing.T) {
			result, err := mapSQLType(sqlType, enums)
			require.NoError(t, err)
			assert.Equal(t, schema.TypeText, result.Kind)
		})
	}
}

func TestMapSQLType_UnsupportedType(t *testing.T) {
	enums := make(map[string]*schema.Enum)
	_, err := mapSQLType("UNSUPPORTED_TYPE", enums)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported SQL type")
}

func TestMapSQLType_ComplexCases(t *testing.T) {
	enums := map[string]*schema.Enum{
		"status": {Name: "status"},
	}

	tests := []struct {
		name              string
		sqlType           string
		expectedKind      schema.TypeKind
		expectedPrecision int
		expectedScale     int
		expectedDepth     int
		expectedEnum      string
	}{
		{
			name:              "varchar with precision",
			sqlType:           "VARCHAR(100)",
			expectedKind:      schema.TypeVarChar,
			expectedPrecision: 100,
			expectedScale:     0,
			expectedDepth:     0,
		},
		{
			name:              "numeric with precision and scale",
			sqlType:           "NUMERIC(12, 4)",
			expectedKind:      schema.TypeNumeric,
			expectedPrecision: 12,
			expectedScale:     4,
			expectedDepth:     0,
		},
		{
			name:              "array of varchar",
			sqlType:           "VARCHAR(50)[]",
			expectedKind:      schema.TypeVarChar,
			expectedPrecision: 50,
			expectedScale:     0,
			expectedDepth:     1,
		},
		{
			name:              "enum type",
			sqlType:           "status",
			expectedKind:      schema.TypeEnum,
			expectedPrecision: 0,
			expectedScale:     0,
			expectedDepth:     0,
			expectedEnum:      "status",
		},
		{
			name:              "timestamp with time zone",
			sqlType:           "TIMESTAMP WITH TIME ZONE",
			expectedKind:      schema.TypeTimestampTZ,
			expectedPrecision: 0,
			expectedScale:     0,
			expectedDepth:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mapSQLType(tt.sqlType, enums)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedKind, result.Kind, "kind mismatch")
			assert.Equal(t, tt.expectedPrecision, result.Precision, "precision mismatch")
			assert.Equal(t, tt.expectedScale, result.Scale, "scale mismatch")
			assert.Equal(t, tt.expectedDepth, result.ArrayDepth, "array depth mismatch")
			if tt.expectedEnum != "" {
				assert.Equal(t, tt.expectedEnum, result.EnumName, "enum name mismatch")
			}
		})
	}
}
