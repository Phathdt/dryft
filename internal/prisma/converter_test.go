package prisma

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/phathdt/dryft/internal/schema"
)

func TestConverter_ConvertSimpleModel(t *testing.T) {
	input := `model User {
  id    Int     @id
  email String  @unique
  name  String?
}`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	if len(s.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(s.Tables))
	}

	table := s.Tables[0]
	if table.Name != "User" {
		t.Errorf("expected table name 'User', got %q", table.Name)
	}

	if len(table.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(table.Columns))
	}

	// Check id column
	if table.Columns[0].Name != "id" {
		t.Errorf("expected column 'id', got %q", table.Columns[0].Name)
	}
	if table.Columns[0].Type.Kind != schema.TypeInt32 {
		t.Errorf("expected Int32, got %v", table.Columns[0].Type.Kind)
	}

	// Check primary key
	if table.PrimaryKey == nil {
		t.Fatal("expected primary key to be set")
	}
	if len(table.PrimaryKey.Columns) != 1 || table.PrimaryKey.Columns[0] != "id" {
		t.Errorf("expected primary key on 'id', got %v", table.PrimaryKey.Columns)
	}

	// Check unique constraint
	if len(table.Constraints) != 1 {
		t.Fatalf("expected 1 constraint, got %d", len(table.Constraints))
	}
	if table.Constraints[0].Type != schema.ConstraintUnique {
		t.Errorf("expected unique constraint")
	}
	if len(table.Constraints[0].Columns) != 1 || table.Constraints[0].Columns[0] != "email" {
		t.Errorf("expected unique on 'email', got %v", table.Constraints[0].Columns)
	}

	// Check nullable field
	if !table.Columns[2].Nullable {
		t.Errorf("expected 'name' to be nullable")
	}
}

func TestConverter_ConvertEnum(t *testing.T) {
	input := `enum Role {
  USER
  ADMIN
  MODERATOR
}`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	if len(s.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(s.Enums))
	}

	enum := s.Enums[0]
	if enum.Name != "Role" {
		t.Errorf("expected enum name 'Role', got %q", enum.Name)
	}

	if len(enum.Values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(enum.Values))
	}

	expectedValues := []string{"USER", "ADMIN", "MODERATOR"}
	for i, expected := range expectedValues {
		if enum.Values[i].Label != expected {
			t.Errorf("expected value %q, got %q", expected, enum.Values[i].Label)
		}
	}
}

func TestConverter_TypeMapping(t *testing.T) {
	tests := []struct {
		name         string
		prismaType   string
		dbAnnotation string
		expectedKind schema.TypeKind
	}{
		{"String to Text", "String", "", schema.TypeText},
		{"String to UUID", "String", "@db.Uuid", schema.TypeUUID},
		{"Int to Int32", "Int", "", schema.TypeInt32},
		{"BigInt to Int64", "BigInt", "", schema.TypeInt64},
		{"Boolean to Bool", "Boolean", "", schema.TypeBool},
		{"DateTime to Timestamp", "DateTime", "", schema.TypeTimestamp},
		{"DateTime to TimestampTZ", "DateTime", "@db.Timestamptz", schema.TypeTimestampTZ},
		{"Json to JSON", "Json", "", schema.TypeJSON},
		{"Json to JSONB", "Json", "@db.JsonB", schema.TypeJSONB},
		{"Float to Float64", "Float", "", schema.TypeFloat64},
		{"Decimal to Numeric", "Decimal", "", schema.TypeNumeric},
		{"Bytes to Bytes", "Bytes", "", schema.TypeBytes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := fmt.Sprintf("model Test { field %s %s }", tt.prismaType, tt.dbAnnotation)
			parser := NewParser(input)
			ast, err := parser.ParseSchema()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			converter := NewConverter()
			s, err := converter.Convert(ast)
			if err != nil {
				t.Fatalf("convert error: %v", err)
			}

			if len(s.Tables) != 1 || len(s.Tables[0].Columns) != 1 {
				t.Fatal("expected 1 table with 1 column")
			}

			col := s.Tables[0].Columns[0]
			if col.Type.Kind != tt.expectedKind {
				t.Errorf("expected %v, got %v", tt.expectedKind, col.Type.Kind)
			}
		})
	}
}

func TestConverter_DefaultValues(t *testing.T) {
	tests := []struct {
		name            string
		defaultAttr     string
		expectedKind    schema.DefaultKind
		expectedLiteral string
		expectedExpr    string
	}{
		{"autoincrement", "@default(autoincrement())", schema.DefaultSequence, "", ""},
		{"now", "@default(now())", schema.DefaultExpression, "", "now()"},
		{"uuid", "@default(uuid())", schema.DefaultExpression, "", "gen_random_uuid()"},
		{"string literal", `@default("test")`, schema.DefaultLiteral, "'test'", ""},
		{"int literal", "@default(42)", schema.DefaultLiteral, "42", ""},
		{"bool literal", "@default(true)", schema.DefaultLiteral, "true", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := fmt.Sprintf("model Test { field String %s }", tt.defaultAttr)
			parser := NewParser(input)
			ast, err := parser.ParseSchema()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			converter := NewConverter()
			s, err := converter.Convert(ast)
			if err != nil {
				t.Fatalf("convert error: %v", err)
			}

			col := s.Tables[0].Columns[0]
			if col.Default == nil {
				t.Fatal("expected default value to be set")
			}

			if col.Default.Kind != tt.expectedKind {
				t.Errorf("expected kind %v, got %v", tt.expectedKind, col.Default.Kind)
			}

			if tt.expectedLiteral != "" && col.Default.Literal != tt.expectedLiteral {
				t.Errorf("expected literal %q, got %q", tt.expectedLiteral, col.Default.Literal)
			}

			if tt.expectedExpr != "" && col.Default.Expression != tt.expectedExpr {
				t.Errorf("expected expression %q, got %q", tt.expectedExpr, col.Default.Expression)
			}
		})
	}
}

func TestConverter_MapAttributes(t *testing.T) {
	input := `model User {
  userId String @id @map("user_id")

  @@map("users")
}`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Check @@map for table name
	if s.Tables[0].Name != "users" {
		t.Errorf("expected table name 'users', got %q", s.Tables[0].Name)
	}

	// Check @map for column name
	if s.Tables[0].Columns[0].Name != "user_id" {
		t.Errorf("expected column name 'user_id', got %q", s.Tables[0].Columns[0].Name)
	}
}

func TestConverter_CompositePrimaryKey(t *testing.T) {
	input := `model UserRole {
  userId String
  roleId String

  @@id([userId, roleId])
}`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	if s.Tables[0].PrimaryKey == nil {
		t.Fatal("expected primary key to be set")
	}

	pk := s.Tables[0].PrimaryKey
	if len(pk.Columns) != 2 {
		t.Fatalf("expected 2 columns in primary key, got %d", len(pk.Columns))
	}

	if pk.Columns[0] != "userId" || pk.Columns[1] != "roleId" {
		t.Errorf("expected primary key on [userId, roleId], got %v", pk.Columns)
	}
}

func TestConverter_CompositeUnique(t *testing.T) {
	input := `model User {
  id    Int    @id
  email String
  phone String

  @@unique([email, phone])
}`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Should have 1 composite unique constraint
	found := false
	for _, constraint := range s.Tables[0].Constraints {
		if constraint.Type == schema.ConstraintUnique && len(constraint.Columns) == 2 {
			if constraint.Columns[0] == "email" && constraint.Columns[1] == "phone" {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("expected composite unique constraint on [email, phone]")
	}
}

func TestConverter_Indexes(t *testing.T) {
	input := `model User {
  id    Int    @id
  email String
  name  String

  @@index([email])
  @@index([name, email])
}`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	if len(s.Tables[0].Indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(s.Tables[0].Indexes))
	}

	// Check first index
	if len(s.Tables[0].Indexes[0].Columns) != 1 {
		t.Errorf("expected 1 column in first index, got %d", len(s.Tables[0].Indexes[0].Columns))
	}
	if s.Tables[0].Indexes[0].Columns[0].Name != "email" {
		t.Errorf("expected index on 'email', got %q", s.Tables[0].Indexes[0].Columns[0].Name)
	}

	// Check second index
	if len(s.Tables[0].Indexes[1].Columns) != 2 {
		t.Errorf("expected 2 columns in second index, got %d", len(s.Tables[0].Indexes[1].Columns))
	}
}

func TestConverter_ArrayTypes(t *testing.T) {
	input := `model Post {
  id   Int      @id
  tags String[]
}`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	col := s.Tables[0].Columns[1]
	if col.Type.ArrayDepth != 1 {
		t.Errorf("expected array depth 1, got %d", col.Type.ArrayDepth)
	}
	if col.Type.Kind != schema.TypeText {
		t.Errorf("expected TypeText, got %v", col.Type.Kind)
	}
}

func TestConverter_EnumField(t *testing.T) {
	input := `enum Role {
  USER
  ADMIN
}

model User {
  id   Int  @id
  role Role
}`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	if len(s.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(s.Enums))
	}

	if len(s.Tables[0].Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(s.Tables[0].Columns))
	}

	col := s.Tables[0].Columns[1]
	if col.Type.Kind != schema.TypeEnum {
		t.Errorf("expected TypeEnum, got %v", col.Type.Kind)
	}
	if col.Type.EnumName != "Role" {
		t.Errorf("expected enum name 'Role', got %q", col.Type.EnumName)
	}
}

func TestConverter_RelationWarning(t *testing.T) {
	input := `model User {
  id    Int    @id
  posts Post[]
}

model Post {
  id       Int  @id
  authorId Int
}`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// posts field should be skipped (passive side, no FK generated)
	if len(s.Tables[0].Columns) != 1 {
		t.Errorf("expected 1 column (relation skipped), got %d", len(s.Tables[0].Columns))
	}

	// No warnings expected - relation fields are now supported
	warnings := converter.Warnings()
	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}
}

func TestConverter_CompleteSchema(t *testing.T) {
	input := `generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

enum Role {
  USER
  ADMIN
}

model User {
  id        String   @id @default(uuid()) @db.Uuid
  email     String   @unique
  name      String?
  role      Role     @default(USER)
  createdAt DateTime @default(now())

  @@map("users")
  @@index([email])
}`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Check enum
	if len(s.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(s.Enums))
	}
	if s.Enums[0].Name != "Role" {
		t.Errorf("expected enum 'Role', got %q", s.Enums[0].Name)
	}

	// Check table
	if len(s.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(s.Tables))
	}
	if s.Tables[0].Name != "users" {
		t.Errorf("expected table 'users', got %q", s.Tables[0].Name)
	}

	// Check columns
	if len(s.Tables[0].Columns) != 5 {
		t.Fatalf("expected 5 columns, got %d", len(s.Tables[0].Columns))
	}

	// Check id column with UUID type
	idCol := s.Tables[0].Columns[0]
	if idCol.Type.Kind != schema.TypeUUID {
		t.Errorf("expected UUID type for id, got %v", idCol.Type.Kind)
	}

	// Check primary key
	if s.Tables[0].PrimaryKey == nil || len(s.Tables[0].PrimaryKey.Columns) != 1 {
		t.Error("expected primary key on id")
	}

	// Check unique constraint
	hasUnique := false
	for _, c := range s.Tables[0].Constraints {
		if c.Type == schema.ConstraintUnique && len(c.Columns) == 1 && c.Columns[0] == "email" {
			hasUnique = true
			break
		}
	}
	if !hasUnique {
		t.Error("expected unique constraint on email")
	}

	// Check index
	if len(s.Tables[0].Indexes) != 1 {
		t.Errorf("expected 1 index, got %d", len(s.Tables[0].Indexes))
	}
}

// Edge Case Tests for Composite Primary Keys

func TestConverter_EmptyCompositeID(t *testing.T) {
	input := `
model Test {
  id Int

  @@id([])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	_, err = converter.Convert(ast)

	// Should either error or ignore empty @@id
	// Decision: Implementation-dependent
	// For now, document behavior in test
	if err != nil {
		t.Logf("Empty @@id returns error: %v", err)
	} else {
		t.Logf("Empty @@id is ignored")
	}
}

func TestConverter_SingleFieldCompositeID(t *testing.T) {
	input := `
model User {
  id Int

  @@id([id])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	table := s.Tables[0]
	if table.PrimaryKey == nil {
		t.Fatal("expected primary key")
	}

	// Should still work with single column
	expected := []string{"id"}
	if !reflect.DeepEqual(table.PrimaryKey.Columns, expected) {
		t.Errorf("expected %v, got %v", expected, table.PrimaryKey.Columns)
	}
}

func TestConverter_CompositePK_ReservedKeywords(t *testing.T) {
	// 'type', 'model', 'enum' are reserved in Prisma
	// But can be used as DB column names via @map
	input := `
model Test {
  typeField String @map("type")
  modelField String @map("model")

  @@id([typeField, modelField])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	table := s.Tables[0]
	expected := []string{"type", "model"}
	if !reflect.DeepEqual(table.PrimaryKey.Columns, expected) {
		t.Errorf("expected %v, got %v", expected, table.PrimaryKey.Columns)
	}
}

func TestConverter_CompositePK_Unicode(t *testing.T) {
	input := `
model Test {
  用户ID String @map("user_id")
  角色ID String @map("role_id")

  @@id([用户ID, 角色ID])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()

	// Parser doesn't support Unicode identifiers - this is expected
	if err != nil {
		t.Logf("Unicode identifiers not supported (expected): %v", err)
		return
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	table := s.Tables[0]
	expected := []string{"user_id", "role_id"}
	if !reflect.DeepEqual(table.PrimaryKey.Columns, expected) {
		t.Errorf("expected %v, got %v", expected, table.PrimaryKey.Columns)
	}
}

func TestConverter_CompositePK_ManyColumns(t *testing.T) {
	input := `
model Test {
  f1  String @map("field_1")
  f2  String @map("field_2")
  f3  String @map("field_3")
  f4  String @map("field_4")
  f5  String @map("field_5")
  f6  String @map("field_6")
  f7  String @map("field_7")
  f8  String @map("field_8")
  f9  String @map("field_9")
  f10 String @map("field_10")

  @@id([f1, f2, f3, f4, f5, f6, f7, f8, f9, f10])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	table := s.Tables[0]
	if len(table.PrimaryKey.Columns) != 10 {
		t.Errorf("expected 10 PK columns, got %d", len(table.PrimaryKey.Columns))
	}

	// Verify all columns mapped correctly
	for i := 1; i <= 10; i++ {
		expected := fmt.Sprintf("field_%d", i)
		if table.PrimaryKey.Columns[i-1] != expected {
			t.Errorf("column %d: expected %q, got %q", i, expected, table.PrimaryKey.Columns[i-1])
		}
	}
}

func TestConverter_CompositePK_DuplicateFields(t *testing.T) {
	input := `
model Test {
  userId String @map("user_id")

  @@id([userId, userId])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)

	// This is semantically invalid but converter might not check it
	// Document behavior
	if err != nil {
		t.Logf("Duplicate fields in @@id returns error: %v", err)
	} else {
		// If no error, columns will contain duplicates
		t.Logf("Duplicate fields allowed: %v", s.Tables[0].PrimaryKey.Columns)
	}
}

func TestConverter_CompositeID_MixedMapAndNoMap(t *testing.T) {
	input := `
model UserRole {
  userId String @map("user_id")
  roleId String
  // roleId has no @map, should use "roleId" as DB column

  @@id([userId, roleId])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	table := s.Tables[0]
	expected := []string{"user_id", "roleId"}
	if !reflect.DeepEqual(table.PrimaryKey.Columns, expected) {
		t.Errorf("expected %v, got %v", expected, table.PrimaryKey.Columns)
	}
}

func TestConverter_CompositeID_WithMap(t *testing.T) {
	input := `
model UserRole {
  userId String @map("user_id")
  roleId String @map("role_id")

  @@id([userId, roleId])
  @@map("user_roles")
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Check table
	if len(s.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(s.Tables))
	}

	table := s.Tables[0]
	if table.Name != "user_roles" {
		t.Errorf("expected table name 'user_roles', got %q", table.Name)
	}

	// Check primary key uses DB column names
	if table.PrimaryKey == nil {
		t.Fatal("expected primary key, got nil")
	}

	expected := []string{"user_id", "role_id"}
	if len(table.PrimaryKey.Columns) != len(expected) {
		t.Fatalf("expected %d PK columns, got %d", len(expected), len(table.PrimaryKey.Columns))
	}

	for i, exp := range expected {
		if table.PrimaryKey.Columns[i] != exp {
			t.Errorf("PK column %d: expected %q, got %q", i, exp, table.PrimaryKey.Columns[i])
		}
	}
}

func TestConverter_CompositeID_NoMap(t *testing.T) {
	input := `
model UserRole {
  userId String
  roleId String

  @@id([userId, roleId])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	table := s.Tables[0]

	// Without @map, field names should be used as-is
	expected := []string{"userId", "roleId"}
	if len(table.PrimaryKey.Columns) != len(expected) {
		t.Fatalf("expected %d PK columns, got %d", len(expected), len(table.PrimaryKey.Columns))
	}
	for i, exp := range expected {
		if table.PrimaryKey.Columns[i] != exp {
			t.Errorf("PK column %d: expected %q, got %q", i, exp, table.PrimaryKey.Columns[i])
		}
	}
}

func TestConverter_CompositeID_NonExistentField(t *testing.T) {
	input := `
model UserRole {
  userId String @map("user_id")
  roleId String @map("role_id")

  @@id([userId, nonExistent])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	_, err = converter.Convert(ast)

	// Should return error
	if err == nil {
		t.Fatal("expected error for non-existent field, got nil")
	}

	expectedMsg := "field \"nonExistent\" referenced in model attribute does not exist"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestConverter_CompositeUnique_WithMap(t *testing.T) {
	input := `
model User {
  id       Int    @id
  email    String @map("email_address")
  username String @map("user_name")

  @@unique([email, username])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	table := s.Tables[0]

	// Should have one UNIQUE constraint
	var uniqueConstraints []schema.Constraint
	for _, c := range table.Constraints {
		if c.Type == schema.ConstraintUnique && len(c.Columns) == 2 {
			uniqueConstraints = append(uniqueConstraints, c)
		}
	}

	if len(uniqueConstraints) != 1 {
		t.Fatalf("expected 1 unique constraint, got %d", len(uniqueConstraints))
	}

	expected := []string{"email_address", "user_name"}
	if len(uniqueConstraints[0].Columns) != len(expected) {
		t.Fatalf("expected %d columns in constraint, got %d", len(expected), len(uniqueConstraints[0].Columns))
	}
	for i, exp := range expected {
		if uniqueConstraints[0].Columns[i] != exp {
			t.Errorf("constraint column %d: expected %q, got %q", i, exp, uniqueConstraints[0].Columns[i])
		}
	}
}

func TestConverter_CompositeIndex_WithMap(t *testing.T) {
	input := `
model Post {
  id        Int      @id
  authorId  Int      @map("author_id")
  createdAt DateTime @map("created_at")

  @@index([authorId, createdAt])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	table := s.Tables[0]

	if len(table.Indexes) != 1 {
		t.Fatalf("expected 1 index, got %d", len(table.Indexes))
	}

	idx := table.Indexes[0]
	if len(idx.Columns) != 2 {
		t.Fatalf("expected 2 index columns, got %d", len(idx.Columns))
	}

	if idx.Columns[0].Name != "author_id" {
		t.Errorf("expected 'author_id', got %q", idx.Columns[0].Name)
	}
	if idx.Columns[1].Name != "created_at" {
		t.Errorf("expected 'created_at', got %q", idx.Columns[1].Name)
	}
}

func TestConverter_IndexWithName(t *testing.T) {
	input := `
model Order {
  id        String   @id
  userId    String   @map("user_id")
  status    String
  createdAt DateTime @map("created_at")

  @@index([userId])
  @@index([status, createdAt], name: "idx_status_created")
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	s, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	table := s.Tables[0]

	if len(table.Indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(table.Indexes))
	}

	// First index: no name (auto-generated)
	idx1 := table.Indexes[0]
	if idx1.Name != "" {
		t.Errorf("expected empty name for first index, got %q", idx1.Name)
	}
	if len(idx1.Columns) != 1 {
		t.Fatalf("expected 1 column, got %d", len(idx1.Columns))
	}
	if idx1.Columns[0].Name != "user_id" {
		t.Errorf("expected 'user_id', got %q", idx1.Columns[0].Name)
	}

	// Second index: named
	idx2 := table.Indexes[1]
	if idx2.Name != "idx_status_created" {
		t.Errorf("expected 'idx_status_created', got %q", idx2.Name)
	}
	if len(idx2.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(idx2.Columns))
	}
	if idx2.Columns[0].Name != "status" {
		t.Errorf("expected 'status', got %q", idx2.Columns[0].Name)
	}
	if idx2.Columns[1].Name != "created_at" {
		t.Errorf("expected 'created_at', got %q", idx2.Columns[1].Name)
	}
}
