package prisma

import (
	"strings"
	"testing"

	"github.com/phathdt/dryft/internal/schema"
)

// TestIntegration_CompleteWorkflow tests the complete flow from Internal Schema to Prisma output.
func TestIntegration_CompleteWorkflow(t *testing.T) {
	// Create a realistic schema similar to what PostgreSQL introspector would produce
	s := &schema.Schema{
		Enums: []schema.Enum{
			{
				Name: "user_status",
				Values: []schema.EnumValue{
					{Label: "ACTIVE", Order: 0},
					{Label: "INACTIVE", Order: 1},
					{Label: "SUSPENDED", Order: 2},
				},
			},
		},
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{
						Name:     "id",
						Type:     schema.DataType{Kind: schema.TypeUUID},
						Nullable: false,
					},
					{
						Name:     "email",
						Type:     schema.DataType{Kind: schema.TypeText},
						Nullable: false,
					},
					{
						Name:     "username",
						Type:     schema.DataType{Kind: schema.TypeVarChar, Precision: 50},
						Nullable: false,
					},
					{
						Name:     "status",
						Type:     schema.DataType{Kind: schema.TypeEnum, EnumName: "user_status"},
						Nullable: false,
						Default: &schema.DefaultValue{
							Kind:    schema.DefaultLiteral,
							Literal: "'ACTIVE'",
						},
					},
					{
						Name:     "created_at",
						Type:     schema.DataType{Kind: schema.TypeTimestampTZ},
						Nullable: false,
						Default: &schema.DefaultValue{
							Kind:       schema.DefaultExpression,
							Expression: "now()",
						},
					},
					{
						Name:     "updated_at",
						Type:     schema.DataType{Kind: schema.TypeTimestampTZ},
						Nullable: false,
					},
				},
				PrimaryKey: &schema.PrimaryKey{
					Columns: []string{"id"},
				},
				Constraints: []schema.Constraint{
					{
						Type:    schema.ConstraintUnique,
						Columns: []string{"email"},
					},
					{
						Type:    schema.ConstraintUnique,
						Columns: []string{"username"},
					},
				},
				Indexes: []schema.Index{
					{
						Name:   "idx_users_status",
						Unique: false,
						Columns: []schema.IndexColumn{
							{Name: "status"},
						},
					},
				},
			},
			{
				Name: "posts",
				Columns: []schema.Column{
					{
						Name:     "id",
						Type:     schema.DataType{Kind: schema.TypeInt64},
						Nullable: false,
						Default: &schema.DefaultValue{
							Kind: schema.DefaultSequence,
							Sequence: &schema.SequenceRef{
								Name:  "posts_id_seq",
								Owned: true,
							},
						},
					},
					{
						Name:     "user_id",
						Type:     schema.DataType{Kind: schema.TypeUUID},
						Nullable: false,
					},
					{
						Name:     "title",
						Type:     schema.DataType{Kind: schema.TypeText},
						Nullable: false,
					},
					{
						Name:     "content",
						Type:     schema.DataType{Kind: schema.TypeText},
						Nullable: true,
					},
					{
						Name:     "published",
						Type:     schema.DataType{Kind: schema.TypeBool},
						Nullable: false,
						Default: &schema.DefaultValue{
							Kind:    schema.DefaultLiteral,
							Literal: "false",
						},
					},
				},
				PrimaryKey: &schema.PrimaryKey{
					Columns: []string{"id"},
				},
				Indexes: []schema.Index{
					{
						Name:   "idx_posts_user_id",
						Unique: false,
						Columns: []schema.IndexColumn{
							{Name: "user_id"},
						},
					},
				},
			},
		},
	}

	// Generate Prisma schema
	writer := NewWriter(DefaultNamingConvention())
	result, err := writer.Write(s)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Verify the output contains expected components
	expectedComponents := []string{
		// Generator and datasource
		"generator client",
		`provider = "prisma-client-js"`,
		"datasource db",
		`provider = "postgresql"`,
		`url      = env("DATABASE_URL")`,

		// Enum
		"enum UserStatus",
		"ACTIVE",
		"INACTIVE",
		"SUSPENDED",

		// Users model
		"model Users {",
		"id",
		"email",
		"username",
		"status",
		"createdAt",
		"updatedAt",
		"@id",
		"@unique",
		"@default(now())",
		"@updatedAt",
		"@db.Uuid",
		"@db.VarChar(50)",
		"@db.Timestamptz",
		`@map("created_at")`,
		`@map("updated_at")`,
		`@@map("users")`,
		"@@index([status])",

		// Posts model
		"model Posts {",
		"userId",
		"title",
		"content",
		"published",
		"@default(autoincrement())",
		"@default(false)",
		`@@map("posts")`,
		"@@index([userId])",
	}

	for _, component := range expectedComponents {
		if !strings.Contains(result, component) {
			t.Errorf("Expected output to contain %q\nFull output:\n%s", component, result)
		}
	}

	// Verify structure: should have proper blocks separated by blank lines
	lines := strings.Split(result, "\n")
	if len(lines) < 10 {
		t.Errorf("Expected output to have multiple lines, got %d", len(lines))
	}

	// Verify no duplicate empty lines (good formatting)
	consecutiveEmpty := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			consecutiveEmpty++
			if consecutiveEmpty > 2 {
				t.Error("Found more than 2 consecutive empty lines (bad formatting)")
				break
			}
		} else {
			consecutiveEmpty = 0
		}
	}
}

// TestIntegration_TypeMapping tests all type mappings work correctly.
func TestIntegration_TypeMapping(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "type_test",
				Columns: []schema.Column{
					{Name: "col_uuid", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "col_text", Type: schema.DataType{Kind: schema.TypeText}, Nullable: false},
					{Name: "col_varchar", Type: schema.DataType{Kind: schema.TypeVarChar, Precision: 100}, Nullable: false},
					{Name: "col_int32", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
					{Name: "col_int64", Type: schema.DataType{Kind: schema.TypeInt64}, Nullable: false},
					{Name: "col_bool", Type: schema.DataType{Kind: schema.TypeBool}, Nullable: false},
					{Name: "col_float32", Type: schema.DataType{Kind: schema.TypeFloat32}, Nullable: false},
					{Name: "col_float64", Type: schema.DataType{Kind: schema.TypeFloat64}, Nullable: false},
					{Name: "col_numeric", Type: schema.DataType{Kind: schema.TypeNumeric, Precision: 10, Scale: 2}, Nullable: false},
					{Name: "col_timestamp", Type: schema.DataType{Kind: schema.TypeTimestamp}, Nullable: false},
					{Name: "col_timestamptz", Type: schema.DataType{Kind: schema.TypeTimestampTZ}, Nullable: false},
					{Name: "col_date", Type: schema.DataType{Kind: schema.TypeDate}, Nullable: false},
					{Name: "col_json", Type: schema.DataType{Kind: schema.TypeJSON}, Nullable: false},
					{Name: "col_jsonb", Type: schema.DataType{Kind: schema.TypeJSONB}, Nullable: false},
					{Name: "col_bytes", Type: schema.DataType{Kind: schema.TypeBytes}, Nullable: false},
					{Name: "col_array", Type: schema.DataType{Kind: schema.TypeText, ArrayDepth: 1}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"col_uuid"}},
			},
		},
	}

	writer := NewWriter(DefaultNamingConvention())
	result, err := writer.Write(s)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Verify all type mappings
	typeMappings := map[string]string{
		"colUuid":        "String @id @db.Uuid",
		"colText":        "String",
		"colVarchar":     "String @db.VarChar(100)",
		"colInt32":       "Int",
		"colInt64":       "BigInt",
		"colBool":        "Boolean",
		"colFloat32":     "Float @db.Real",
		"colFloat64":     "Float",
		"colNumeric":     "Decimal @db.Numeric(10,2)",
		"colTimestamp":   "DateTime",
		"colTimestamptz": "DateTime @db.Timestamptz",
		"colDate":        "DateTime @db.Date",
		"colJson":        "Json",
		"colJsonb":       "Json @db.JsonB",
		"colBytes":       "Bytes",
		"colArray":       "String[]",
	}

	for field, expectedType := range typeMappings {
		if !strings.Contains(result, field) {
			t.Errorf("Expected field %q in output", field)
		}
		// Check that the type appears near the field name
		fieldIndex := strings.Index(result, field)
		if fieldIndex != -1 {
			// Get the line containing the field
			lineStart := strings.LastIndex(result[:fieldIndex], "\n")
			lineEnd := strings.Index(result[fieldIndex:], "\n")
			if lineEnd != -1 {
				line := result[lineStart+1 : fieldIndex+lineEnd]
				if !strings.Contains(line, field) || !strings.Contains(line, strings.Split(expectedType, " ")[0]) {
					t.Errorf("Field %q does not have expected type pattern %q\nLine: %s", field, expectedType, line)
				}
			}
		}
	}
}

// TestIntegration_CompositePK_RoundTrip tests full round-trip integrity for composite primary keys.
func TestIntegration_CompositePK_RoundTrip(t *testing.T) {
	// Start with internal schema (as DB introspector produces)
	originalSchema := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "user_roles",
				Columns: []schema.Column{
					{Name: "user_id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
					{Name: "role_id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
					{Name: "granted_at", Type: schema.DataType{Kind: schema.TypeTimestampTZ}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{
					Columns: []string{"user_id", "role_id"},
				},
			},
		},
	}

	// Step 1: Write to Prisma schema
	writer := NewWriter(DefaultNamingConvention())
	prismaOutput, err := writer.Write(originalSchema)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// Verify output contains @@id
	if !strings.Contains(prismaOutput, "@@id([userId, roleId])") {
		t.Errorf("Expected @@id([userId, roleId]) in output:\n%s", prismaOutput)
	}

	// Step 2: Parse back
	parser := NewParser(prismaOutput)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// Step 3: Convert back to internal schema
	converter := NewConverter()
	roundTripSchema, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	// Step 4: Verify round-trip integrity
	if len(roundTripSchema.Tables) != 1 {
		t.Fatalf("expected 1 table after round-trip, got %d", len(roundTripSchema.Tables))
	}

	table := roundTripSchema.Tables[0]
	if table.PrimaryKey == nil {
		t.Fatal("expected primary key after round-trip")
	}

	// Check columns match (order matters)
	expected := []string{"user_id", "role_id"}
	if len(table.PrimaryKey.Columns) != len(expected) {
		t.Errorf("PK columns count: expected %d, got %d", len(expected), len(table.PrimaryKey.Columns))
	}
	for i, col := range expected {
		if i >= len(table.PrimaryKey.Columns) || table.PrimaryKey.Columns[i] != col {
			t.Errorf("PK column[%d]: expected %q, got %q", i, col, table.PrimaryKey.Columns[i])
		}
	}
}

// TestIntegration_ThreeColumnPK_RoundTrip tests round-trip with 3-column composite PK.
func TestIntegration_ThreeColumnPK_RoundTrip(t *testing.T) {
	originalSchema := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "order_items",
				Columns: []schema.Column{
					{Name: "order_id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
					{Name: "item_id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
					{Name: "revision", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{
					Columns: []string{"order_id", "item_id", "revision"},
				},
			},
		},
	}

	// Write → Parse → Convert
	writer := NewWriter(DefaultNamingConvention())
	prismaOutput, err := writer.Write(originalSchema)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	parser := NewParser(prismaOutput)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	converter := NewConverter()
	roundTripSchema, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	// Verify
	table := roundTripSchema.Tables[0]
	expected := []string{"order_id", "item_id", "revision"}
	if len(table.PrimaryKey.Columns) != len(expected) {
		t.Errorf("expected %d PK columns, got %d", len(expected), len(table.PrimaryKey.Columns))
	}
	for i, col := range expected {
		if i >= len(table.PrimaryKey.Columns) || table.PrimaryKey.Columns[i] != col {
			t.Errorf("PK column[%d]: expected %q, got %q", i, col, table.PrimaryKey.Columns[i])
		}
	}
}

// TestIntegration_CompositePK_WithOtherConstraints tests composite PK with UNIQUE and INDEX.
func TestIntegration_CompositePK_WithOtherConstraints(t *testing.T) {
	originalSchema := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "user_roles",
				Columns: []schema.Column{
					{Name: "user_id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
					{Name: "role_id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
					{Name: "granted_at", Type: schema.DataType{Kind: schema.TypeTimestampTZ}, Nullable: false},
					{Name: "granted_by", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: true},
				},
				PrimaryKey: &schema.PrimaryKey{
					Columns: []string{"user_id", "role_id"},
				},
				Constraints: []schema.Constraint{
					{
						Type:    schema.ConstraintUnique,
						Columns: []string{"user_id", "granted_at"},
					},
				},
				Indexes: []schema.Index{
					{
						Columns: []schema.IndexColumn{
							{Name: "role_id"},
							{Name: "granted_at"},
						},
					},
				},
			},
		},
	}

	// Round-trip
	writer := NewWriter(DefaultNamingConvention())
	prismaOutput, err := writer.Write(originalSchema)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	parser := NewParser(prismaOutput)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	converter := NewConverter()
	roundTripSchema, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	table := roundTripSchema.Tables[0]

	// Verify PK
	expectedPK := []string{"user_id", "role_id"}
	if len(table.PrimaryKey.Columns) != len(expectedPK) {
		t.Errorf("PK count: expected %d, got %d", len(expectedPK), len(table.PrimaryKey.Columns))
	}
	for i, col := range expectedPK {
		if i >= len(table.PrimaryKey.Columns) || table.PrimaryKey.Columns[i] != col {
			t.Errorf("PK column[%d]: expected %q, got %q", i, col, table.PrimaryKey.Columns[i])
		}
	}

	// Verify UNIQUE constraint
	uniqueCount := 0
	for _, c := range table.Constraints {
		if c.Type == schema.ConstraintUnique {
			uniqueCount++
			expectedUnique := []string{"user_id", "granted_at"}
			if len(c.Columns) != len(expectedUnique) {
				t.Errorf("UNIQUE count: expected %d columns, got %d", len(expectedUnique), len(c.Columns))
			}
			for i, col := range expectedUnique {
				if i >= len(c.Columns) || c.Columns[i] != col {
					t.Errorf("UNIQUE column[%d]: expected %q, got %q", i, col, c.Columns[i])
				}
			}
		}
	}
	if uniqueCount != 1 {
		t.Errorf("expected 1 unique constraint, got %d", uniqueCount)
	}

	// Verify INDEX
	if len(table.Indexes) != 1 {
		t.Fatalf("expected 1 index, got %d", len(table.Indexes))
	}
	if table.Indexes[0].Columns[0].Name != "role_id" {
		t.Errorf("index col 0: expected 'role_id', got %q", table.Indexes[0].Columns[0].Name)
	}
	if table.Indexes[0].Columns[1].Name != "granted_at" {
		t.Errorf("index col 1: expected 'granted_at', got %q", table.Indexes[0].Columns[1].Name)
	}
}
