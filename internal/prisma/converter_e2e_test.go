package prisma

import (
	"strings"
	"testing"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
	"github.com/phathdt/dryft/internal/sql/postgres"
)

// TestE2E_PrismaRelationToSQL verifies the full pipeline:
// Prisma schema → Parse → Convert → Diff → SQL generation
func TestE2E_PrismaRelationToSQL(t *testing.T) {
	input := `
model Post {
  id     Int  @id @default(autoincrement())
  title  String
  userId Int  @map("user_id")
  user   User @relation(fields: [userId], references: [id], onDelete: Cascade)
}

model User {
  id    Int    @id @default(autoincrement())
  name  String
  posts Post[]
}
`

	// Parse Prisma schema
	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Convert to internal schema
	converter := NewConverter()
	afterSchema, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Diff against empty schema (initial migration)
	beforeSchema := &schema.Schema{Tables: []schema.Table{}, Enums: []schema.Enum{}}
	differ := diff.NewDiffer(nil)
	ops, err := differ.Diff(beforeSchema, afterSchema)
	if err != nil {
		t.Fatalf("diff error: %v", err)
	}

	// Generate SQL
	sqlGen := postgres.NewGenerator(sql.GeneratorOptions{})
	sqlStatements, err := sqlGen.Generate(ops)
	if err != nil {
		t.Fatalf("SQL generation error: %v", err)
	}

	// Verify FK constraint in generated SQL
	foundFK := false
	var fkSQL string
	for _, stmt := range sqlStatements {
		if strings.Contains(stmt, "FOREIGN KEY") && strings.Contains(stmt, "user_id") {
			foundFK = true
			fkSQL = stmt
			break
		}
	}

	if !foundFK {
		t.Errorf("Expected FOREIGN KEY constraint in SQL, statements:\n%s",
			strings.Join(sqlStatements, "\n"))
	}

	// Verify FK details
	if !strings.Contains(fkSQL, "REFERENCES") {
		t.Error("FK SQL should contain REFERENCES clause")
	}

	if !strings.Contains(fkSQL, "ON DELETE CASCADE") {
		t.Error("FK SQL should contain ON DELETE CASCADE")
	}

	t.Logf("Generated FK SQL:\n%s", fkSQL)
}

// TestE2E_CompositeFK_ToSQL verifies composite FK generation
func TestE2E_CompositeFK_ToSQL(t *testing.T) {
	input := `
model OrderItem {
  tenantId Int @map("tenant_id")
  orderId  Int @map("order_id")
  itemId   Int @map("item_id")
  order    Order @relation(fields: [tenantId, orderId], references: [tenantId, id])

  @@id([tenantId, orderId, itemId])
}

model Order {
  tenantId Int @map("tenant_id")
  id       Int
  items    OrderItem[]

  @@id([tenantId, id])
}
`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	afterSchema, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	beforeSchema := &schema.Schema{Tables: []schema.Table{}, Enums: []schema.Enum{}}
	differ := diff.NewDiffer(nil)
	ops, err := differ.Diff(beforeSchema, afterSchema)
	if err != nil {
		t.Fatalf("diff error: %v", err)
	}

	sqlGen := postgres.NewGenerator(sql.GeneratorOptions{})
	sqlStatements, err := sqlGen.Generate(ops)
	if err != nil {
		t.Fatalf("SQL generation error: %v", err)
	}

	// Verify composite FK
	foundFK := false
	var fkSQL string
	for _, stmt := range sqlStatements {
		if strings.Contains(stmt, "FOREIGN KEY") &&
			strings.Contains(stmt, "tenant_id") &&
			strings.Contains(stmt, "order_id") {
			foundFK = true
			fkSQL = stmt
			break
		}
	}

	if !foundFK {
		t.Errorf("Expected composite FOREIGN KEY in SQL, statements:\n%s",
			strings.Join(sqlStatements, "\n"))
	}

	// Should reference both columns
	if !strings.Contains(fkSQL, "tenant_id") || !strings.Contains(fkSQL, "order_id") {
		t.Errorf("FK should reference both columns, got: %s", fkSQL)
	}

	t.Logf("Generated composite FK SQL:\n%s", fkSQL)
}

// TestE2E_MultipleRelations_IndependentFKs verifies multiple FKs in same table
func TestE2E_MultipleRelations_IndependentFKs(t *testing.T) {
	input := `
model Post {
  id       Int  @id
  authorId Int  @map("author_id")
  editorId Int? @map("editor_id")
  author   User @relation("PostAuthor", fields: [authorId], references: [id])
  editor   User? @relation("PostEditor", fields: [editorId], references: [id])
}

model User {
  id            Int    @id
  authoredPosts Post[] @relation("PostAuthor")
  editedPosts   Post[] @relation("PostEditor")
}
`

	parser := NewParser(input)
	ast, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	converter := NewConverter()
	afterSchema, err := converter.Convert(ast)
	if err != nil {
		t.Fatalf("convert error: %v", err)
	}

	// Find Post table
	var postTable *schema.Table
	for i := range afterSchema.Tables {
		if afterSchema.Tables[i].Name == "Post" {
			postTable = &afterSchema.Tables[i]
			break
		}
	}

	if postTable == nil {
		t.Fatal("Post table not found")
	}

	// Should have 2 FKs
	if len(postTable.ForeignKeys) != 2 {
		t.Fatalf("expected 2 foreign keys, got %d", len(postTable.ForeignKeys))
	}

	// Verify both FKs point to User
	for _, fk := range postTable.ForeignKeys {
		if fk.RefTable != "User" {
			t.Errorf("expected FK to reference User, got %s", fk.RefTable)
		}
	}

	t.Logf("Successfully generated %d independent FKs", len(postTable.ForeignKeys))
}
