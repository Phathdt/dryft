package prisma

import (
	"fmt"
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

	// posts field should be skipped with warning
	if len(s.Tables[0].Columns) != 1 {
		t.Errorf("expected 1 column (relation skipped), got %d", len(s.Tables[0].Columns))
	}

	warnings := converter.Warnings()
	if len(warnings) == 0 {
		t.Error("expected warning about relation field")
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
