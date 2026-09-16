package prisma

import (
	"strings"
	"testing"

	"github.com/phathdt/dryft/internal/schema"
)

func TestWriter_Write_EmptySchema(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())
	s := &schema.Schema{
		Tables: []schema.Table{},
		Enums:  []schema.Enum{},
	}

	result, err := writer.Write(s)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Should contain generator and datasource blocks
	if !strings.Contains(result, "generator client") {
		t.Error("Expected generator block in output")
	}
	if !strings.Contains(result, "datasource db") {
		t.Error("Expected datasource block in output")
	}
}

func TestWriter_Write_NilSchema(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())
	_, err := writer.Write(nil)
	if err == nil {
		t.Error("Expected error for nil schema")
	}
}

func TestWriter_WriteTable_BasicTable(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	table := &schema.Table{
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
		},
		PrimaryKey: &schema.PrimaryKey{
			Columns: []string{"id"},
		},
	}

	result, err := writer.writeTable(table, nil)
	if err != nil {
		t.Fatalf("writeTable() failed: %v", err)
	}

	// Check model name transformation (users -> Users in PascalCase)
	if !strings.Contains(result, "model Users {") {
		t.Errorf("Expected PascalCase model name 'Users'\nGot:\n%s", result)
	}

	// Check field name transformation
	if !strings.Contains(result, "id") && !strings.Contains(result, "email") {
		t.Error("Expected fields 'id' and 'email'")
	}

	// Check @id attribute
	if !strings.Contains(result, "@id") {
		t.Error("Expected @id attribute on primary key")
	}

	// Check @@map attribute
	if !strings.Contains(result, `@@map("users")`) {
		t.Error("Expected @@map attribute for table name")
	}
}

func TestWriter_WriteTable_SnakeCaseColumns(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	table := &schema.Table{
		Name: "user_profiles",
		Columns: []schema.Column{
			{
				Name:     "user_id",
				Type:     schema.DataType{Kind: schema.TypeUUID},
				Nullable: false,
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
			Columns: []string{"user_id"},
		},
	}

	result, err := writer.writeTable(table, nil)
	if err != nil {
		t.Fatalf("writeTable() failed: %v", err)
	}

	// Check field name transformations
	if !strings.Contains(result, "userId") {
		t.Error("Expected camelCase field 'userId'")
	}
	if !strings.Contains(result, "createdAt") {
		t.Error("Expected camelCase field 'createdAt'")
	}
	if !strings.Contains(result, "updatedAt") {
		t.Error("Expected camelCase field 'updatedAt'")
	}

	// Check @map attributes
	if !strings.Contains(result, `@map("user_id")`) {
		t.Error("Expected @map attribute for user_id")
	}
	if !strings.Contains(result, `@map("created_at")`) {
		t.Error("Expected @map attribute for created_at")
	}

	// Check @updatedAt attribute
	if !strings.Contains(result, "@updatedAt") {
		t.Error("Expected @updatedAt attribute on updated_at field")
	}

	// Check @default(now())
	if !strings.Contains(result, "@default(now())") {
		t.Error("Expected @default(now()) on created_at field")
	}
}

func TestWriter_WriteTable_NullableFields(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	table := &schema.Table{
		Name: "posts",
		Columns: []schema.Column{
			{
				Name:     "id",
				Type:     schema.DataType{Kind: schema.TypeInt32},
				Nullable: false,
			},
			{
				Name:     "title",
				Type:     schema.DataType{Kind: schema.TypeText},
				Nullable: false,
			},
			{
				Name:     "description",
				Type:     schema.DataType{Kind: schema.TypeText},
				Nullable: true,
			},
		},
		PrimaryKey: &schema.PrimaryKey{
			Columns: []string{"id"},
		},
	}

	result, err := writer.writeTable(table, nil)
	if err != nil {
		t.Fatalf("writeTable() failed: %v", err)
	}

	// Check nullable field has '?' marker
	lines := strings.Split(result, "\n")
	var descriptionLine string
	for _, line := range lines {
		if strings.Contains(line, "description") {
			descriptionLine = line
			break
		}
	}

	if !strings.Contains(descriptionLine, "String?") {
		t.Errorf("Expected nullable field 'description' to have 'String?' type, got: %s", descriptionLine)
	}
}

func TestWriter_WriteTable_UniqueConstraints(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	table := &schema.Table{
		Name: "users",
		Columns: []schema.Column{
			{
				Name:     "id",
				Type:     schema.DataType{Kind: schema.TypeInt32},
				Nullable: false,
			},
			{
				Name:     "email",
				Type:     schema.DataType{Kind: schema.TypeText},
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
		},
	}

	result, err := writer.writeTable(table, nil)
	if err != nil {
		t.Fatalf("writeTable() failed: %v", err)
	}

	// Check @unique attribute
	if !strings.Contains(result, "@unique") {
		t.Error("Expected @unique attribute on email field")
	}
}

func TestWriter_WriteTable_CompositeUniqueConstraint(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	table := &schema.Table{
		Name: "user_roles",
		Columns: []schema.Column{
			{
				Name:     "user_id",
				Type:     schema.DataType{Kind: schema.TypeInt32},
				Nullable: false,
			},
			{
				Name:     "role_id",
				Type:     schema.DataType{Kind: schema.TypeInt32},
				Nullable: false,
			},
		},
		Constraints: []schema.Constraint{
			{
				Type:    schema.ConstraintUnique,
				Columns: []string{"user_id", "role_id"},
			},
		},
	}

	result, err := writer.writeTable(table, nil)
	if err != nil {
		t.Fatalf("writeTable() failed: %v", err)
	}

	// Check @@unique with composite columns
	if !strings.Contains(result, "@@unique([userId, roleId])") {
		t.Error("Expected @@unique([userId, roleId]) for composite constraint")
	}
}

func TestWriter_WriteTable_Indexes(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	table := &schema.Table{
		Name: "posts",
		Columns: []schema.Column{
			{
				Name:     "id",
				Type:     schema.DataType{Kind: schema.TypeInt32},
				Nullable: false,
			},
			{
				Name:     "user_id",
				Type:     schema.DataType{Kind: schema.TypeInt32},
				Nullable: false,
			},
		},
		PrimaryKey: &schema.PrimaryKey{
			Columns: []string{"id"},
		},
		Indexes: []schema.Index{
			{
				Name: "idx_posts_user_id",
				Columns: []schema.IndexColumn{
					{Name: "user_id"},
				},
				Unique: false,
			},
		},
	}

	result, err := writer.writeTable(table, nil)
	if err != nil {
		t.Fatalf("writeTable() failed: %v", err)
	}

	// Check @@index attribute
	if !strings.Contains(result, "@@index([userId])") {
		t.Error("Expected @@index([userId]) for index")
	}
}

func TestWriter_WriteEnum(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	enum := &schema.Enum{
		Name: "user_role",
		Values: []schema.EnumValue{
			{Label: "ADMIN", Order: 0},
			{Label: "USER", Order: 1},
			{Label: "GUEST", Order: 2},
		},
	}

	result := writer.writeEnum(enum)

	// Check enum name transformation
	if !strings.Contains(result, "enum UserRole {") {
		t.Error("Expected PascalCase enum name 'UserRole'")
	}

	// Check enum values
	if !strings.Contains(result, "ADMIN") {
		t.Error("Expected enum value 'ADMIN'")
	}
	if !strings.Contains(result, "USER") {
		t.Error("Expected enum value 'USER'")
	}
	if !strings.Contains(result, "GUEST") {
		t.Error("Expected enum value 'GUEST'")
	}
}

func TestWriter_FormatDefault_Literals(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	tests := []struct {
		name     string
		def      *schema.DefaultValue
		expected string
	}{
		{
			name: "boolean true",
			def: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "true",
			},
			expected: "@default(true)",
		},
		{
			name: "boolean false",
			def: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "false",
			},
			expected: "@default(false)",
		},
		{
			name: "numeric literal",
			def: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "42",
			},
			expected: "@default(42)",
		},
		{
			name: "string literal",
			def: &schema.DefaultValue{
				Kind:    schema.DefaultLiteral,
				Literal: "'active'",
			},
			expected: `@default("active")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := writer.formatDefault(tt.def)
			if result != tt.expected {
				t.Errorf("formatDefault() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestWriter_FormatDefault_Expressions(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	tests := []struct {
		name     string
		def      *schema.DefaultValue
		expected string
	}{
		{
			name: "now() function",
			def: &schema.DefaultValue{
				Kind:       schema.DefaultExpression,
				Expression: "now()",
			},
			expected: "@default(now())",
		},
		{
			name: "uuid_generate_v4()",
			def: &schema.DefaultValue{
				Kind:       schema.DefaultExpression,
				Expression: "uuid_generate_v4()",
			},
			expected: "@default(uuid())",
		},
		{
			name: "gen_random_uuid()",
			def: &schema.DefaultValue{
				Kind:       schema.DefaultExpression,
				Expression: "gen_random_uuid()",
			},
			expected: "@default(uuid())",
		},
		{
			name: "current_timestamp",
			def: &schema.DefaultValue{
				Kind:       schema.DefaultExpression,
				Expression: "current_timestamp",
			},
			expected: "@default(now())",
		},
		{
			name: "custom expression",
			def: &schema.DefaultValue{
				Kind:       schema.DefaultExpression,
				Expression: "ARRAY[]::text[]",
			},
			expected: `@default(dbgenerated("ARRAY[]::text[]"))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := writer.formatDefault(tt.def)
			if result != tt.expected {
				t.Errorf("formatDefault() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestWriter_FormatDefault_Sequence(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	def := &schema.DefaultValue{
		Kind: schema.DefaultSequence,
		Sequence: &schema.SequenceRef{
			Name:  "users_id_seq",
			Owned: true,
		},
	}

	result := writer.formatDefault(def)
	expected := "@default(autoincrement())"

	if result != expected {
		t.Errorf("formatDefault() = %q, want %q", result, expected)
	}
}

func TestWriter_Write_CompleteSchema(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	s := &schema.Schema{
		Enums: []schema.Enum{
			{
				Name: "user_role",
				Values: []schema.EnumValue{
					{Label: "ADMIN", Order: 0},
					{Label: "USER", Order: 1},
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
						Name:     "role",
						Type:     schema.DataType{Kind: schema.TypeEnum, EnumName: "user_role"},
						Nullable: false,
						Default: &schema.DefaultValue{
							Kind:    schema.DefaultLiteral,
							Literal: "'USER'",
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
				},
				PrimaryKey: &schema.PrimaryKey{
					Columns: []string{"id"},
				},
				Constraints: []schema.Constraint{
					{
						Type:    schema.ConstraintUnique,
						Columns: []string{"email"},
					},
				},
			},
		},
	}

	result, err := writer.Write(s)
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Verify key components are present
	requiredElements := []string{
		"generator client",
		"datasource db",
		"enum UserRole",
		"ADMIN",
		"USER",
		"model Users {",
		"id",
		"email",
		"role",
		"createdAt",
		"@id",
		"@unique",
		"@default(now())",
		`@@map("users")`,
		`@map("created_at")`,
	}

	for _, elem := range requiredElements {
		if !strings.Contains(result, elem) {
			t.Errorf("Expected output to contain %q\nGot:\n%s", elem, result)
		}
	}
}

func TestWriter_ParseTypeAnnotation(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	tests := []struct {
		name          string
		prismaType    string
		expectedBase  string
		expectedAnnot string
	}{
		{
			name:          "simple type",
			prismaType:    "String",
			expectedBase:  "String",
			expectedAnnot: "",
		},
		{
			name:          "type with annotation",
			prismaType:    "String @db.Uuid",
			expectedBase:  "String",
			expectedAnnot: "@db.Uuid",
		},
		{
			name:          "type with complex annotation",
			prismaType:    "Decimal @db.Numeric(10,2)",
			expectedBase:  "Decimal",
			expectedAnnot: "@db.Numeric(10,2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, annot := writer.parseTypeAnnotation(tt.prismaType)
			if base != tt.expectedBase {
				t.Errorf("parseTypeAnnotation() base = %q, want %q", base, tt.expectedBase)
			}
			if annot != tt.expectedAnnot {
				t.Errorf("parseTypeAnnotation() annotation = %q, want %q", annot, tt.expectedAnnot)
			}
		})
	}
}

func TestWriter_ReservedModelNames(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	tests := []struct {
		name          string
		tableName     string
		expectError   bool
	}{
		{
			name:        "reserved word model",
			tableName:   "model",
			expectError: true,
		},
		{
			name:        "reserved word enum",
			tableName:   "enum",
			expectError: true,
		},
		{
			name:        "reserved word type",
			tableName:   "type",
			expectError: true,
		},
		{
			name:        "normal table name",
			tableName:   "users",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &schema.Schema{
				Tables: []schema.Table{
					{
						Name: tt.tableName,
						Columns: []schema.Column{
							{Name: "id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
						},
						PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
					},
				},
			}

			_, err := writer.Write(schema)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error=%v, got error=%v", tt.expectError, err != nil)
			}
		})
	}
}

func TestWriter_ReservedFieldNames(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	schema := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
					{Name: "type", Type: schema.DataType{Kind: schema.TypeText}, Nullable: true},
					{Name: "model", Type: schema.DataType{Kind: schema.TypeText}, Nullable: true},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
			},
		},
	}

	result, err := writer.Write(schema)
	if err != nil {
		t.Errorf("Write() should not error for field names: %v", err)
	}

	// Prisma doesn't reserve field names - they should be generated normally
	if !strings.Contains(result, "type") {
		t.Error("Expected 'type' field in output")
	}
	if !strings.Contains(result, "model") {
		t.Error("Expected 'model' field in output")
	}
}

func TestWriter_CollisionDetection(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	// Two distinct column names that transform to the same field name
	schema := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
					{Name: "user_id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: true},
					{Name: "user__id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: true},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
			},
		},
	}

	_, err := writer.Write(schema)
	if err == nil {
		t.Error("Expected error for field name collision, got nil")
	}
}

func TestWriter_WriteTableWithDefaultValues(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	table := &schema.Table{
		Name: "posts",
		Columns: []schema.Column{
			{
				Name:     "id",
				Type:     schema.DataType{Kind: schema.TypeInt32},
				Nullable: false,
			},
			{
				Name:     "published",
				Type:     schema.DataType{Kind: schema.TypeBool},
				Nullable: false,
				Default:  &schema.DefaultValue{Kind: schema.DefaultLiteral, Literal: "false"},
			},
			{
				Name:     "created_at",
				Type:     schema.DataType{Kind: schema.TypeTimestampTZ},
				Nullable: false,
				Default:  &schema.DefaultValue{Kind: schema.DefaultExpression, Expression: "now()"},
			},
			{
				Name:     "views",
				Type:     schema.DataType{Kind: schema.TypeInt32},
				Nullable: false,
				Default:  &schema.DefaultValue{Kind: schema.DefaultLiteral, Literal: "0"},
			},
		},
		PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
	}

	result, err := writer.writeTable(table, nil)
	if err != nil {
		t.Fatalf("writeTable() failed: %v", err)
	}

	if !strings.Contains(result, "@default(false)") {
		t.Error("Expected @default(false) for published field")
	}
	if !strings.Contains(result, "@default(now())") {
		t.Error("Expected @default(now()) for created_at field")
	}
	if !strings.Contains(result, "@default(0)") {
		t.Error("Expected @default(0) for views field")
	}
}

func TestWriter_UpdatedAtField(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	tests := []struct {
		name        string
		columnName  string
		expectUpdAt bool
	}{
		{
			name:        "updated_at",
			columnName:  "updated_at",
			expectUpdAt: true,
		},
		{
			name:        "updatedat",
			columnName:  "updatedat",
			expectUpdAt: true,
		},
		{
			name:        "modified_at",
			columnName:  "modified_at",
			expectUpdAt: true,
		},
		{
			name:        "created_at",
			columnName:  "created_at",
			expectUpdAt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := writer.isUpdatedAtField(tt.columnName)
			if result != tt.expectUpdAt {
				t.Errorf("expected %v, got %v", tt.expectUpdAt, result)
			}
		})
	}
}

func TestWriter_IsUniqueColumn(t *testing.T) {
	writer := NewWriter(DefaultNamingConvention())

	table := &schema.Table{
		Name: "users",
		Columns: []schema.Column{
			{Name: "id", Type: schema.DataType{Kind: schema.TypeInt32}},
			{Name: "email", Type: schema.DataType{Kind: schema.TypeText}},
		},
		Constraints: []schema.Constraint{
			{Type: schema.ConstraintUnique, Columns: []string{"email"}},
		},
	}

	tests := []struct {
		name       string
		columnName string
		expect     bool
	}{
		{"email is unique", "email", true},
		{"id is not unique", "id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := writer.isUniqueColumn(tt.columnName, table)
			if result != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, result)
			}
		})
	}
}
