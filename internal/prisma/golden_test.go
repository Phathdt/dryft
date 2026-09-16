package prisma

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGolden_RoundTrip tests that Prisma → Internal → Prisma round-trip produces consistent output.
// Note: This uses snake_case naming to preserve model names without transformation.
func TestGolden_RoundTrip(t *testing.T) {
	goldenFiles := []string{
		"basic.prisma",
		"enums.prisma",
		"composite_pk.prisma",
		"indexes.prisma",
		"defaults.prisma",
		"complete.prisma",
	}

	for _, filename := range goldenFiles {
		t.Run(filename, func(t *testing.T) {
			path := filepath.Join("testdata", "golden", filename)

			// Read golden file
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read golden file: %v", err)
			}

			input := string(content)

			// Parse Prisma → Internal Schema
			result, err := Parse(input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			// Internal Schema → Prisma (use snake_case to preserve names)
			naming := NamingConvention{
				Fields: "snake_case",
				Models: "snake_case",
			}
			writer := NewWriter(naming)
			output, err := writer.Write(result.Schema)
			if err != nil {
				t.Fatalf("write error: %v", err)
			}

			// Parse the output to verify it's valid Prisma
			_, err = Parse(output)
			if err != nil {
				t.Errorf("round-trip produced invalid schema: %v\nOutput:\n%s", err, output)
			}

			// Verify schema equivalence (not string equality, but semantic equality)
			result2, err := Parse(output)
			if err != nil {
				t.Fatalf("failed to parse output: %v", err)
			}

			// Compare table count
			if len(result.Schema.Tables) != len(result2.Schema.Tables) {
				t.Errorf("table count mismatch: expected %d, got %d",
					len(result.Schema.Tables), len(result2.Schema.Tables))
			}

			// Compare enum count
			if len(result.Schema.Enums) != len(result2.Schema.Enums) {
				t.Errorf("enum count mismatch: expected %d, got %d",
					len(result.Schema.Enums), len(result2.Schema.Enums))
			}

			// Log warnings if any
			if len(result.Warnings) > 0 {
				t.Logf("Warnings: %v", result.Warnings)
			}
		})
	}
}

// normalizeSchema removes generator/datasource blocks and normalizes whitespace for comparison.
func normalizeSchema(input string) string {
	lines := strings.Split(input, "\n")
	var normalized []string
	inBlock := false
	skipBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip generator and datasource blocks
		if strings.HasPrefix(trimmed, "generator ") || strings.HasPrefix(trimmed, "datasource ") {
			skipBlock = true
			inBlock = true
			continue
		}

		if inBlock {
			if trimmed == "}" {
				inBlock = false
				skipBlock = false
			}
			if skipBlock {
				continue
			}
		}

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		normalized = append(normalized, trimmed)
	}

	return strings.Join(normalized, "\n")
}

// TestGolden_ParseErrors tests that invalid schemas produce clear error messages.
func TestGolden_ParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "missing model name",
			input: `model {
  id Int @id
}`,
		},
		{
			name: "invalid syntax",
			input: `model User
  id Int @id
}`,
		},
		{
			name: "missing closing brace",
			input: `model User {
  id Int @id`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Error("expected parse error, got nil")
			}
		})
	}
}

// TestGolden_ComplexTypes tests various type combinations.
func TestGolden_ComplexTypes(t *testing.T) {
	input := `model Test {
  id         String   @id @default(uuid()) @db.Uuid
  name       String
  age        Int?
  tags       String[]
  metadata   Json     @db.JsonB
  balance    Decimal
  avatar     Bytes?
  createdAt  DateTime @default(now()) @db.Timestamptz
  lastLogin  DateTime?

  @@map("tests")
}`

	result, err := Parse(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(result.Schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(result.Schema.Tables))
	}

	table := result.Schema.Tables[0]
	if table.Name != "tests" {
		t.Errorf("expected table name 'tests', got %q", table.Name)
	}

	expectedColumns := 9
	if len(table.Columns) != expectedColumns {
		t.Errorf("expected %d columns, got %d", expectedColumns, len(table.Columns))
	}

	// Verify round-trip
	writer := NewWriter(DefaultNamingConvention())
	output, err := writer.Write(result.Schema)
	if err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Parse output again to verify it's valid
	_, err = Parse(output)
	if err != nil {
		t.Errorf("round-trip produced invalid schema: %v\nOutput:\n%s", err, output)
	}
}

// TestGolden_AllAttributes tests all supported attributes.
func TestGolden_AllAttributes(t *testing.T) {
	input := `enum Status {
  ACTIVE
  INACTIVE
}

model User {
  id        Int      @id @default(autoincrement())
  uuid      String   @unique @db.Uuid
  email     String   @unique
  name      String?
  status    Status   @default(ACTIVE)
  age       Int      @default(0)
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt

  @@map("users")
  @@index([email])
  @@index([status, createdAt])
  @@unique([email, name])
}`

	result, err := Parse(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Verify enum
	if len(result.Schema.Enums) != 1 {
		t.Errorf("expected 1 enum, got %d", len(result.Schema.Enums))
	}

	// Verify table
	table := result.Schema.Tables[0]
	if table.Name != "users" {
		t.Errorf("expected table name 'users', got %q", table.Name)
	}

	// Verify primary key
	if table.PrimaryKey == nil {
		t.Error("expected primary key")
	}

	// Verify constraints (unique constraints)
	uniqueCount := 0
	for _, c := range table.Constraints {
		if c.Type == 1 { // ConstraintUnique
			uniqueCount++
		}
	}
	if uniqueCount < 2 {
		t.Errorf("expected at least 2 unique constraints, got %d", uniqueCount)
	}

	// Verify indexes
	if len(table.Indexes) < 2 {
		t.Errorf("expected at least 2 indexes, got %d", len(table.Indexes))
	}

	// Verify round-trip
	writer := NewWriter(DefaultNamingConvention())
	output, err := writer.Write(result.Schema)
	if err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Verify output is parseable
	_, err = Parse(output)
	if err != nil {
		t.Errorf("round-trip produced invalid schema: %v", err)
	}
}
