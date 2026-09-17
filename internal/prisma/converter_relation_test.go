package prisma

import (
	"strings"
	"testing"
)

func TestConverter_CompositeID_WithRelationField(t *testing.T) {
	input := `
model User {
  id    Int    @id
  email String
  posts Post[]

  @@id([email, posts])
}

model Post {
  id     Int    @id
  userId Int
  user   User   @relation(fields: [userId], references: [id])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	_, err = converter.Convert(ast)

	// Should return error because posts is a relation field
	if err == nil {
		t.Fatal("expected error for relation field in @@id, got nil")
	}

	expectedMsg := "relation field"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestConverter_CompositeUnique_WithRelationField(t *testing.T) {
	input := `
model User {
  id    Int    @id
  email String
  posts Post[]

  @@unique([email, posts])
}

model Post {
  id     Int    @id
  userId Int
  user   User   @relation(fields: [userId], references: [id])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	_, err = converter.Convert(ast)

	// Should return error because posts is a relation field
	if err == nil {
		t.Fatal("expected error for relation field in @@unique, got nil")
	}

	expectedMsg := "relation field"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestConverter_CompositeIndex_WithRelationField(t *testing.T) {
	input := `
model User {
  id    Int    @id
  email String
  posts Post[]

  @@index([email, posts])
}

model Post {
  id     Int    @id
  userId Int
  user   User   @relation(fields: [userId], references: [id])
}
`
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	_, err = converter.Convert(ast)

	// Should return error because posts is a relation field
	if err == nil {
		t.Fatal("expected error for relation field in @@index, got nil")
	}

	expectedMsg := "relation field"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}
