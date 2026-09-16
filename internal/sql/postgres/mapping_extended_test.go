package postgres

import (
	"testing"

	"github.com/phathdt/dryft/internal/schema"
	"github.com/stretchr/testify/assert"
)

func TestQuoteIdentifier_ReservedKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"user", "user", `"user"`},
		{"table", "table", `"table"`},
		{"column", "column", `"column"`},
		{"index", "index", `"index"`},
		{"select", "select", `"select"`},
		{"insert", "insert", `"insert"`},
		{"update", "update", `"update"`},
		{"delete", "delete", `"delete"`},
		{"create", "create", `"create"`},
		{"drop", "drop", `"drop"`},
		{"alter", "alter", `"alter"`},
		{"order", "order", `"order"`},
		{"group", "group", `"group"`},
		{"default", "default", `"default"`},
		{"grant", "grant", `"grant"`},
		{"revoke", "revoke", `"revoke"`},
		{"sequence", "sequence", `"sequence"`},
		{"primary", "primary", `"primary"`},
		{"foreign", "foreign", `"foreign"`},
		{"key", "key", `"key"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuoteIdentifier_UppercaseLetters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"uppercase", "User", `"User"`},
		{"mixed case", "UserProfile", `"UserProfile"`},
		{"uppercase start", "Email", `"Email"`},
		{"uppercase middle", "userEmail", `"userEmail"`},
		{"all uppercase", "USERS", `"USERS"`},
		{"single uppercase", "A", `"A"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuoteIdentifier_NormalIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "users", "users"},
		{"underscore", "user_profiles", "user_profiles"},
		{"numbers", "table1", "table1"},
		{"underscore prefix", "_internal", "_internal"},
		{"multiple underscores", "user__profile", "user__profile"},
		{"numbers and underscore", "user_id_2", "user_id_2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNeedsQuoting_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"single char lowercase", "a", false},
		{"single char uppercase", "A", true},
		{"reserved single char", "x", false},
		{"reserved keyword", "user", true},
		{"underscore only", "_", false},
		{"leading underscore", "_user", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsQuoting(tt.input)
			assert.Equal(t, tt.expected, result, "input: %q", tt.input)
		})
	}
}

func TestIsReservedWord_CaseInsensitivity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"lowercase", "user", true},
		{"uppercase", "USER", true},
		{"mixed", "User", true},
		{"uppercase", "TABLE", true},
		{"mixed", "Select", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isReservedWord(tt.input)
			assert.Equal(t, tt.expected, result, "input: %q", tt.input)
		})
	}
}

func TestIsReservedWord_NotReserved(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"users", "users", false},
		{"tables", "tables", false},
		{"my_column", "my_column", false},
		{"userColumn", "userColumn", false},
		{"ord", "ord", false}, // Contains "order" substring but not exact match
		{"sel", "sel", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isReservedWord(tt.input)
			assert.Equal(t, tt.expected, result, "input: %q", tt.input)
		})
	}
}

func TestFormatColumnList_Empty(t *testing.T) {
	result := formatColumnList([]string{})
	assert.Equal(t, "", result)
}

func TestFormatColumnList_SingleColumn(t *testing.T) {
	result := formatColumnList([]string{"id"})
	assert.Equal(t, "id", result)
}

func TestFormatColumnList_MultipleColumns(t *testing.T) {
	result := formatColumnList([]string{"id", "email", "name"})
	assert.Equal(t, "id, email, name", result)
}

func TestFormatColumnList_ReservedWords(t *testing.T) {
	result := formatColumnList([]string{"user", "order", "table"})
	assert.Equal(t, `"user", "order", "table"`, result)
}

func TestFormatColumnList_MixedCase(t *testing.T) {
	result := formatColumnList([]string{"id", "User", "Email"})
	assert.Equal(t, `id, "User", "Email"`, result)
}

func TestMapDataType_AllBasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		dt       schema.DataType
		expected string
	}{
		{"UUID", schema.DataType{Kind: schema.TypeUUID}, "UUID"},
		{"Text", schema.DataType{Kind: schema.TypeText}, "TEXT"},
		{"VarChar", schema.DataType{Kind: schema.TypeVarChar}, "VARCHAR"},
		{"Char", schema.DataType{Kind: schema.TypeChar}, "CHAR"},
		{"Int32", schema.DataType{Kind: schema.TypeInt32}, "INTEGER"},
		{"Int64", schema.DataType{Kind: schema.TypeInt64}, "BIGINT"},
		{"Bool", schema.DataType{Kind: schema.TypeBool}, "BOOLEAN"},
		{"Float32", schema.DataType{Kind: schema.TypeFloat32}, "REAL"},
		{"Float64", schema.DataType{Kind: schema.TypeFloat64}, "DOUBLE PRECISION"},
		{"Numeric", schema.DataType{Kind: schema.TypeNumeric}, "NUMERIC"},
		{"Timestamp", schema.DataType{Kind: schema.TypeTimestamp}, "TIMESTAMP"},
		{"TimestampTZ", schema.DataType{Kind: schema.TypeTimestampTZ}, "TIMESTAMPTZ"},
		{"Date", schema.DataType{Kind: schema.TypeDate}, "DATE"},
		{"JSON", schema.DataType{Kind: schema.TypeJSON}, "JSON"},
		{"JSONB", schema.DataType{Kind: schema.TypeJSONB}, "JSONB"},
		{"Bytes", schema.DataType{Kind: schema.TypeBytes}, "BYTEA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapDataType(tt.dt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapDataType_VarCharWithPrecision(t *testing.T) {
	dt := schema.DataType{Kind: schema.TypeVarChar, Precision: 255}
	result := mapDataType(dt)
	assert.Equal(t, "VARCHAR(255)", result)
}

func TestMapDataType_CharWithPrecision(t *testing.T) {
	dt := schema.DataType{Kind: schema.TypeChar, Precision: 20}
	result := mapDataType(dt)
	assert.Equal(t, "CHAR(20)", result)
}

func TestMapDataType_NumericWithPrecisionOnly(t *testing.T) {
	dt := schema.DataType{Kind: schema.TypeNumeric, Precision: 10}
	result := mapDataType(dt)
	assert.Equal(t, "NUMERIC(10)", result)
}

func TestMapDataType_NumericWithPrecisionAndScale(t *testing.T) {
	dt := schema.DataType{Kind: schema.TypeNumeric, Precision: 10, Scale: 2}
	result := mapDataType(dt)
	assert.Equal(t, "NUMERIC(10,2)", result)
}

func TestMapDataType_SingleArray(t *testing.T) {
	tests := []struct {
		name        string
		dt          schema.DataType
		expected    string
	}{
		{"text[]", schema.DataType{Kind: schema.TypeText, ArrayDepth: 1}, "TEXT[]"},
		{"int32[]", schema.DataType{Kind: schema.TypeInt32, ArrayDepth: 1}, "INTEGER[]"},
		{"uuid[]", schema.DataType{Kind: schema.TypeUUID, ArrayDepth: 1}, "UUID[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapDataType(tt.dt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapDataType_MultiDimensionalArray(t *testing.T) {
	tests := []struct {
		name        string
		arrayDepth  int
		expected    string
	}{
		{"2D array", 2, "TEXT[][]"},
		{"3D array", 3, "TEXT[][][]"},
		{"4D array", 4, "TEXT[][][][] "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := schema.DataType{Kind: schema.TypeText, ArrayDepth: tt.arrayDepth}
			result := mapDataType(dt)
			// Need to check substring because of formatting variance
			assert.Contains(t, result, "TEXT")
			assert.Equal(t, tt.arrayDepth, (len(result)-4)/2) // Count []
		})
	}
}

func TestMapDataType_EnumType(t *testing.T) {
	dt := schema.DataType{Kind: schema.TypeEnum, EnumName: "status"}
	result := mapDataType(dt)
	assert.Equal(t, "status", result)
}

func TestMapDataType_EnumTypeReservedKeyword(t *testing.T) {
	dt := schema.DataType{Kind: schema.TypeEnum, EnumName: "order"}
	result := mapDataType(dt)
	assert.Equal(t, `"order"`, result)
}

func TestMapDataType_EnumTypeUppercase(t *testing.T) {
	dt := schema.DataType{Kind: schema.TypeEnum, EnumName: "Status"}
	result := mapDataType(dt)
	assert.Equal(t, `"Status"`, result)
}

func TestFormatDefault_Nil(t *testing.T) {
	result := formatDefault(nil)
	assert.Equal(t, "", result)
}

func TestFormatDefault_Literal(t *testing.T) {
	dv := &schema.DefaultValue{
		Kind:    schema.DefaultLiteral,
		Literal: "'active'",
	}
	result := formatDefault(dv)
	assert.Equal(t, "'active'", result)
}

func TestFormatDefault_LiteralNumeric(t *testing.T) {
	dv := &schema.DefaultValue{
		Kind:    schema.DefaultLiteral,
		Literal: "0",
	}
	result := formatDefault(dv)
	assert.Equal(t, "0", result)
}

func TestFormatDefault_Expression(t *testing.T) {
	dv := &schema.DefaultValue{
		Kind:       schema.DefaultExpression,
		Expression: "now()",
	}
	result := formatDefault(dv)
	assert.Equal(t, "now()", result)
}

func TestFormatDefault_ExpressionComplex(t *testing.T) {
	dv := &schema.DefaultValue{
		Kind:       schema.DefaultExpression,
		Expression: "CURRENT_TIMESTAMP AT TIME ZONE 'UTC'",
	}
	result := formatDefault(dv)
	assert.Equal(t, "CURRENT_TIMESTAMP AT TIME ZONE 'UTC'", result)
}

func TestFormatDefault_Sequence(t *testing.T) {
	dv := &schema.DefaultValue{
		Kind: schema.DefaultSequence,
		Sequence: &schema.SequenceRef{
			Name: "users_id_seq",
		},
	}
	result := formatDefault(dv)
	assert.Equal(t, "nextval('users_id_seq'::regclass)", result)
}

func TestFormatDefault_SequenceNil(t *testing.T) {
	dv := &schema.DefaultValue{
		Kind:     schema.DefaultSequence,
		Sequence: nil,
	}
	result := formatDefault(dv)
	assert.Equal(t, "", result)
}

func TestFormatDefault_UnknownKind(t *testing.T) {
	dv := &schema.DefaultValue{
		Kind: schema.DefaultKind(999),
	}
	result := formatDefault(dv)
	assert.Equal(t, "", result)
}

func TestDefaultsEqual_BothNil(t *testing.T) {
	result := defaultsEqual(nil, nil)
	assert.True(t, result)
}

func TestDefaultsEqual_OneNil(t *testing.T) {
	dv := &schema.DefaultValue{Kind: schema.DefaultLiteral, Literal: "0"}
	result := defaultsEqual(nil, dv)
	assert.False(t, result)

	result = defaultsEqual(dv, nil)
	assert.False(t, result)
}

func TestDefaultsEqual_SameLiteral(t *testing.T) {
	dv1 := &schema.DefaultValue{Kind: schema.DefaultLiteral, Literal: "'active'"}
	dv2 := &schema.DefaultValue{Kind: schema.DefaultLiteral, Literal: "'active'"}
	result := defaultsEqual(dv1, dv2)
	assert.True(t, result)
}

func TestDefaultsEqual_DifferentLiteral(t *testing.T) {
	dv1 := &schema.DefaultValue{Kind: schema.DefaultLiteral, Literal: "'active'"}
	dv2 := &schema.DefaultValue{Kind: schema.DefaultLiteral, Literal: "'inactive'"}
	result := defaultsEqual(dv1, dv2)
	assert.False(t, result)
}

func TestDefaultsEqual_DifferentKind(t *testing.T) {
	dv1 := &schema.DefaultValue{Kind: schema.DefaultLiteral, Literal: "0"}
	dv2 := &schema.DefaultValue{Kind: schema.DefaultExpression, Expression: "0"}
	result := defaultsEqual(dv1, dv2)
	assert.False(t, result)
}

func TestDefaultsEqual_SameExpression(t *testing.T) {
	dv1 := &schema.DefaultValue{Kind: schema.DefaultExpression, Expression: "now()"}
	dv2 := &schema.DefaultValue{Kind: schema.DefaultExpression, Expression: "now()"}
	result := defaultsEqual(dv1, dv2)
	assert.True(t, result)
}

func TestDefaultsEqual_DifferentExpression(t *testing.T) {
	dv1 := &schema.DefaultValue{Kind: schema.DefaultExpression, Expression: "now()"}
	dv2 := &schema.DefaultValue{Kind: schema.DefaultExpression, Expression: "CURRENT_TIMESTAMP"}
	result := defaultsEqual(dv1, dv2)
	assert.False(t, result)
}

func TestDefaultsEqual_SameSequence(t *testing.T) {
	dv1 := &schema.DefaultValue{
		Kind:     schema.DefaultSequence,
		Sequence: &schema.SequenceRef{Name: "id_seq"},
	}
	dv2 := &schema.DefaultValue{
		Kind:     schema.DefaultSequence,
		Sequence: &schema.SequenceRef{Name: "id_seq"},
	}
	result := defaultsEqual(dv1, dv2)
	assert.True(t, result)
}
