package prisma

import (
	"strings"
	"testing"
)

func TestFormatter_FormatBlock(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name      string
		blockType string
		blockName string
		content   []string
		want      []string
	}{
		{
			name:      "generator block",
			blockType: "generator",
			blockName: "client",
			content:   []string{`provider = "prisma-client-js"`},
			want: []string{
				"generator client {",
				`  provider = "prisma-client-js"`,
				"}",
			},
		},
		{
			name:      "model block",
			blockType: "model",
			blockName: "User",
			content:   []string{"id String @id", "email String @unique"},
			want: []string{
				"model User {",
				"  id String @id",
				"  email String @unique",
				"}",
			},
		},
		{
			name:      "enum block",
			blockType: "enum",
			blockName: "Role",
			content:   []string{"ADMIN", "USER"},
			want: []string{
				"enum Role {",
				"  ADMIN",
				"  USER",
				"}",
			},
		},
		{
			name:      "empty content",
			blockType: "model",
			blockName: "Empty",
			content:   []string{},
			want: []string{
				"model Empty {",
				"}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.FormatBlock(tt.blockType, tt.blockName, tt.content)
			gotLines := strings.Split(got, "\n")

			if len(gotLines) != len(tt.want) {
				t.Fatalf("FormatBlock() got %d lines, want %d lines\nGot:\n%s", len(gotLines), len(tt.want), got)
			}

			for i, line := range gotLines {
				if line != tt.want[i] {
					t.Errorf("Line %d: got %q, want %q", i, line, tt.want[i])
				}
			}
		})
	}
}

func TestFormatter_FormatField(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name       string
		fieldName  string
		fieldType  string
		attributes []string
		expected   string
	}{
		{
			name:       "simple field",
			fieldName:  "id",
			fieldType:  "String",
			attributes: []string{"@id"},
			expected:   "id String @id",
		},
		{
			name:       "field with multiple attributes",
			fieldName:  "email",
			fieldType:  "String",
			attributes: []string{"@unique", "@db.VarChar(255)"},
			expected:   "email String @unique @db.VarChar(255)",
		},
		{
			name:       "field without attributes",
			fieldName:  "name",
			fieldType:  "String",
			attributes: []string{},
			expected:   "name String",
		},
		{
			name:       "nullable field",
			fieldName:  "bio",
			fieldType:  "String?",
			attributes: []string{},
			expected:   "bio String?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatField(tt.fieldName, tt.fieldType, tt.attributes)
			if result != tt.expected {
				t.Errorf("FormatField() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatter_AlignFields(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name   string
		fields []FieldLine
		want   []string
	}{
		{
			name: "aligned fields with varying lengths",
			fields: []FieldLine{
				{Name: "id", Type: "String", Attributes: []string{"@id", "@db.Uuid"}},
				{Name: "email", Type: "String", Attributes: []string{"@unique"}},
				{Name: "createdAt", Type: "DateTime", Attributes: []string{"@default(now())", "@map(\"created_at\")"}},
			},
			want: []string{
				"id        String   @id @db.Uuid",
				"email     String   @unique",
				"createdAt DateTime @default(now()) @map(\"created_at\")",
			},
		},
		{
			name: "fields with no attributes",
			fields: []FieldLine{
				{Name: "id", Type: "Int", Attributes: []string{}},
				{Name: "name", Type: "String", Attributes: []string{}},
			},
			want: []string{
				"id   Int",
				"name String",
			},
		},
		{
			name:   "empty fields",
			fields: []FieldLine{},
			want:   nil,
		},
		{
			name: "single field",
			fields: []FieldLine{
				{Name: "id", Type: "String", Attributes: []string{"@id"}},
			},
			want: []string{
				"id String @id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.AlignFields(tt.fields)

			if len(got) != len(tt.want) {
				t.Fatalf("AlignFields() returned %d lines, want %d lines", len(got), len(tt.want))
			}

			for i, line := range got {
				if line != tt.want[i] {
					t.Errorf("Line %d: got %q, want %q", i, line, tt.want[i])
				}
			}
		})
	}
}

func TestFormatter_AlignFields_Spacing(t *testing.T) {
	f := NewFormatter()

	fields := []FieldLine{
		{Name: "id", Type: "Int", Attributes: []string{"@id"}},
		{Name: "veryLongFieldName", Type: "String", Attributes: []string{"@unique"}},
		{Name: "x", Type: "Boolean", Attributes: []string{}},
	}

	result := f.AlignFields(fields)

	// All field names should be padded to the same width (veryLongFieldName = 17 chars)
	// All types should be padded to the same width (Boolean = 7 chars)
	expected := []string{
		"id                Int     @id",
		"veryLongFieldName String  @unique",
		"x                 Boolean",
	}

	if len(result) != len(expected) {
		t.Fatalf("AlignFields() returned %d lines, want %d", len(result), len(expected))
	}

	for i, line := range result {
		if line != expected[i] {
			t.Errorf("Line %d:\ngot:  %q\nwant: %q", i, line, expected[i])
		}
	}
}

func TestFormatter_FormatAttributes(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name       string
		attributes []string
		want       []string
	}{
		{
			name:       "multiple attributes",
			attributes: []string{"@@map(\"users\")", "@@index([email])"},
			want:       []string{"@@map(\"users\")", "@@index([email])"},
		},
		{
			name:       "single attribute",
			attributes: []string{"@@map(\"users\")"},
			want:       []string{"@@map(\"users\")"},
		},
		{
			name:       "empty attributes",
			attributes: []string{},
			want:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.FormatAttributes(tt.attributes)

			if len(got) != len(tt.want) {
				t.Fatalf("FormatAttributes() returned %d items, want %d", len(got), len(tt.want))
			}

			for i, attr := range got {
				if attr != tt.want[i] {
					t.Errorf("Attribute %d: got %q, want %q", i, attr, tt.want[i])
				}
			}
		})
	}
}

func TestFormatter_FormatEnumValue(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "uppercase value",
			value:    "ADMIN",
			expected: "ADMIN",
		},
		{
			name:     "mixed case value",
			value:    "SuperUser",
			expected: "SuperUser",
		},
		{
			name:     "lowercase value",
			value:    "guest",
			expected: "guest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatEnumValue(tt.value)
			if result != tt.expected {
				t.Errorf("FormatEnumValue(%q) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestFormatter_JoinBlocks(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name   string
		blocks []string
		want   string
	}{
		{
			name: "multiple blocks",
			blocks: []string{
				"generator client {\n  provider = \"prisma-client-js\"\n}",
				"datasource db {\n  provider = \"postgresql\"\n}",
				"model User {\n  id String @id\n}",
			},
			want: "generator client {\n  provider = \"prisma-client-js\"\n}\n\ndatasource db {\n  provider = \"postgresql\"\n}\n\nmodel User {\n  id String @id\n}",
		},
		{
			name:   "single block",
			blocks: []string{"model User {\n  id String @id\n}"},
			want:   "model User {\n  id String @id\n}",
		},
		{
			name:   "empty blocks",
			blocks: []string{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.JoinBlocks(tt.blocks)
			if got != tt.want {
				t.Errorf("JoinBlocks() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestNewFormatter(t *testing.T) {
	f := NewFormatter()

	if f.indent != "  " {
		t.Errorf("NewFormatter().indent = %q, want %q", f.indent, "  ")
	}
}
