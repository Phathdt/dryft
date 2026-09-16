package prisma

import (
	"testing"
)

func TestParser_ParseModel(t *testing.T) {
	input := `model User {
  id    Int     @id
  email String  @unique
  name  String?
}`

	parser := NewParser(input)
	schema, err := parser.ParseSchema()

	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	if len(schema.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(schema.Declarations))
	}

	model, ok := schema.Declarations[0].(*ModelDeclaration)
	if !ok {
		t.Fatalf("expected ModelDeclaration, got %T", schema.Declarations[0])
	}

	if model.Name != "User" {
		t.Errorf("expected model name 'User', got %q", model.Name)
	}

	if len(model.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(model.Fields))
	}

	// Check first field
	if model.Fields[0].Name != "id" {
		t.Errorf("expected field name 'id', got %q", model.Fields[0].Name)
	}
	if model.Fields[0].Type.Name != "Int" {
		t.Errorf("expected type 'Int', got %q", model.Fields[0].Type.Name)
	}
	if len(model.Fields[0].Attributes) != 1 {
		t.Errorf("expected 1 attribute, got %d", len(model.Fields[0].Attributes))
	}
	if model.Fields[0].Attributes[0].Name != "id" {
		t.Errorf("expected attribute 'id', got %q", model.Fields[0].Attributes[0].Name)
	}

	// Check third field (optional)
	if model.Fields[2].Name != "name" {
		t.Errorf("expected field name 'name', got %q", model.Fields[2].Name)
	}
	if !model.Fields[2].Type.Optional {
		t.Errorf("expected field 'name' to be optional")
	}
}

func TestParser_ParseEnum(t *testing.T) {
	input := `enum Role {
  USER
  ADMIN
  MODERATOR
}`

	parser := NewParser(input)
	schema, err := parser.ParseSchema()

	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	if len(schema.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(schema.Declarations))
	}

	enum, ok := schema.Declarations[0].(*EnumDeclaration)
	if !ok {
		t.Fatalf("expected EnumDeclaration, got %T", schema.Declarations[0])
	}

	if enum.Name != "Role" {
		t.Errorf("expected enum name 'Role', got %q", enum.Name)
	}

	if len(enum.Values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(enum.Values))
	}

	expectedValues := []string{"USER", "ADMIN", "MODERATOR"}
	for i, expected := range expectedValues {
		if enum.Values[i].Name != expected {
			t.Errorf("expected value %q, got %q", expected, enum.Values[i].Name)
		}
	}
}

func TestParser_ParseModelWithAttributes(t *testing.T) {
	input := `model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  createdAt DateTime @default(now())

  @@map("users")
  @@index([email])
}`

	parser := NewParser(input)
	schema, err := parser.ParseSchema()

	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	model := schema.Declarations[0].(*ModelDeclaration)

	// Check field attributes
	if len(model.Fields[0].Attributes) != 2 {
		t.Fatalf("expected 2 attributes on 'id' field, got %d", len(model.Fields[0].Attributes))
	}
	if model.Fields[0].Attributes[0].Name != "id" {
		t.Errorf("expected first attribute 'id', got %q", model.Fields[0].Attributes[0].Name)
	}
	if model.Fields[0].Attributes[1].Name != "default" {
		t.Errorf("expected second attribute 'default', got %q", model.Fields[0].Attributes[1].Name)
	}

	// Check model attributes
	if len(model.Attributes) != 2 {
		t.Fatalf("expected 2 model attributes, got %d", len(model.Attributes))
	}
	if model.Attributes[0].Name != "map" {
		t.Errorf("expected first model attribute 'map', got %q", model.Attributes[0].Name)
	}
	if model.Attributes[1].Name != "index" {
		t.Errorf("expected second model attribute 'index', got %q", model.Attributes[1].Name)
	}
}

func TestParser_ParseDatasource(t *testing.T) {
	input := `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}`

	parser := NewParser(input)
	schema, err := parser.ParseSchema()

	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	if len(schema.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(schema.Declarations))
	}

	ds, ok := schema.Declarations[0].(*DatasourceDeclaration)
	if !ok {
		t.Fatalf("expected DatasourceDeclaration, got %T", schema.Declarations[0])
	}

	if ds.Name != "db" {
		t.Errorf("expected datasource name 'db', got %q", ds.Name)
	}

	if len(ds.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(ds.Properties))
	}

	if ds.Properties[0].Key != "provider" {
		t.Errorf("expected property key 'provider', got %q", ds.Properties[0].Key)
	}
	if ds.Properties[0].Value != "postgresql" {
		t.Errorf("expected property value 'postgresql', got %v", ds.Properties[0].Value)
	}
}

func TestParser_ParseGenerator(t *testing.T) {
	input := `generator client {
  provider = "prisma-client-js"
}`

	parser := NewParser(input)
	schema, err := parser.ParseSchema()

	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	if len(schema.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(schema.Declarations))
	}

	gen, ok := schema.Declarations[0].(*GeneratorDeclaration)
	if !ok {
		t.Fatalf("expected GeneratorDeclaration, got %T", schema.Declarations[0])
	}

	if gen.Name != "client" {
		t.Errorf("expected generator name 'client', got %q", gen.Name)
	}
}

func TestParser_ParseCompleteSchema(t *testing.T) {
	input := `generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id        String   @id @default(uuid()) @db.Uuid
  email     String   @unique
  name      String?
  createdAt DateTime @default(now())
  posts     Post[]

  @@map("users")
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  published Boolean  @default(false)
  authorId  String   @db.Uuid

  @@index([authorId])
}

enum Role {
  USER
  ADMIN
}`

	parser := NewParser(input)
	schema, err := parser.ParseSchema()

	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	if len(schema.Declarations) != 5 {
		t.Fatalf("expected 5 declarations, got %d", len(schema.Declarations))
	}

	// Verify declaration types
	_, ok := schema.Declarations[0].(*GeneratorDeclaration)
	if !ok {
		t.Errorf("expected first declaration to be GeneratorDeclaration")
	}

	_, ok = schema.Declarations[1].(*DatasourceDeclaration)
	if !ok {
		t.Errorf("expected second declaration to be DatasourceDeclaration")
	}

	_, ok = schema.Declarations[2].(*ModelDeclaration)
	if !ok {
		t.Errorf("expected third declaration to be ModelDeclaration")
	}

	_, ok = schema.Declarations[3].(*ModelDeclaration)
	if !ok {
		t.Errorf("expected fourth declaration to be ModelDeclaration")
	}

	_, ok = schema.Declarations[4].(*EnumDeclaration)
	if !ok {
		t.Errorf("expected fifth declaration to be EnumDeclaration")
	}
}

func TestParser_ParseFieldTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected FieldType
	}{
		{
			name:  "simple type",
			input: "model Test { field String }",
			expected: FieldType{
				Name:     "String",
				Optional: false,
				List:     false,
			},
		},
		{
			name:  "optional type",
			input: "model Test { field String? }",
			expected: FieldType{
				Name:     "String",
				Optional: true,
				List:     false,
			},
		},
		{
			name:  "list type",
			input: "model Test { field String[] }",
			expected: FieldType{
				Name:     "String",
				Optional: false,
				List:     true,
			},
		},
		{
			name:  "relation type",
			input: "model Test { field User }",
			expected: FieldType{
				Name:     "User",
				Optional: false,
				List:     false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			schema, err := parser.ParseSchema()

			if err != nil {
				t.Fatalf("parser error: %v", err)
			}

			model := schema.Declarations[0].(*ModelDeclaration)
			field := model.Fields[0]

			if field.Type.Name != tt.expected.Name {
				t.Errorf("expected type name %q, got %q", tt.expected.Name, field.Type.Name)
			}
			if field.Type.Optional != tt.expected.Optional {
				t.Errorf("expected optional %v, got %v", tt.expected.Optional, field.Type.Optional)
			}
			if field.Type.List != tt.expected.List {
				t.Errorf("expected list %v, got %v", tt.expected.List, field.Type.List)
			}
		})
	}
}

func TestParser_ErrorHandling(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "missing model name",
			input: "model { id Int }",
		},
		{
			name:  "missing brace",
			input: "model User id Int }",
		},
		{
			name:  "invalid token",
			input: "invalid User { id Int }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			_, err := parser.ParseSchema()

			if err == nil {
				t.Errorf("expected parser error, got nil")
			}
		})
	}
}
