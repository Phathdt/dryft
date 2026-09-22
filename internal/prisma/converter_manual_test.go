package prisma

import (
	"os"
	"strings"
	"testing"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
	"github.com/phathdt/dryft/internal/sql/postgres"
)

// TestManual_RealWorldSchema tests a realistic Prisma schema with relations
func TestManual_RealWorldSchema(t *testing.T) {
	schemaInput := `
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  published Boolean  @default(false)
  authorId  Int      @map("author_id")
  author    User     @relation(fields: [authorId], references: [id], onDelete: Cascade)
  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  @@map("posts")
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  posts     Post[]
  createdAt DateTime @default(now()) @map("created_at")

  @@map("users")
}
`

	parser := NewParser(schemaInput)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	afterSchema, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Verify FK was created
	var postsTable *schema.Table
	for i := range afterSchema.Tables {
		if afterSchema.Tables[i].Name == "posts" {
			postsTable = &afterSchema.Tables[i]
			break
		}
	}

	if postsTable == nil {
		t.Fatal("posts table not found")
	}

	if len(postsTable.ForeignKeys) != 1 {
		t.Fatalf("expected 1 FK, got %d", len(postsTable.ForeignKeys))
	}

	fk := postsTable.ForeignKeys[0]
	t.Logf("FK: %v -> %s(%v), OnDelete=%v", fk.Columns, fk.RefTable, fk.RefColumns, fk.OnDelete)

	// Generate SQL
	beforeSchema := &schema.Schema{Tables: []schema.Table{}, Enums: []schema.Enum{}}
	differ := diff.NewDiffer(nil)
	ops, err := differ.Diff(beforeSchema, afterSchema)
	if err != nil {
		t.Fatalf("diff error: %v", err)
	}

	sqlGen := postgres.NewGenerator(sql.GeneratorOptions{})
	statements, err := sqlGen.Generate(ops)
	if err != nil {
		t.Fatalf("SQL generation error: %v", err)
	}

	// Find FK statement
	var fkSQL string
	for _, stmt := range statements {
		if strings.Contains(stmt, "FOREIGN KEY") && strings.Contains(stmt, "author_id") {
			fkSQL = stmt
			break
		}
	}

	if fkSQL == "" {
		t.Fatalf("FK SQL not found in:\n%s", strings.Join(statements, "\n\n"))
	}

	// Verify FK SQL content
	if !strings.Contains(fkSQL, "REFERENCES") {
		t.Error("FK should contain REFERENCES")
	}

	if !strings.Contains(fkSQL, "ON DELETE CASCADE") {
		t.Error("FK should contain ON DELETE CASCADE")
	}

	// Save to file for manual inspection
	os.WriteFile("/tmp/generated-migration.sql", []byte(strings.Join(statements, ";\n\n")+";\n"), 0644)

	t.Logf("\n=== Generated FK SQL ===\n%s", fkSQL)
	t.Log("\nFull migration saved to: /tmp/generated-migration.sql")
}
