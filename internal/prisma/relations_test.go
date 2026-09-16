package prisma

import (
	"strings"
	"testing"

	"github.com/phathdt/dryft/internal/schema"
)

func TestBuildRelationPlan_SimpleForeignKey(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
			},
			{
				Name: "posts",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "author_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
				ForeignKeys: []schema.ForeignKey{
					{
						Name:       "fk_posts_author",
						Columns:    []string{"author_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					},
				},
			},
		},
	}

	writer := NewWriter(DefaultNamingConvention())
	plan, err := writer.buildRelationPlan(s)
	if err != nil {
		t.Fatalf("buildRelationPlan failed: %v", err)
	}

	// Posts should have forward relation "author"
	if len(plan["posts"]) != 1 {
		t.Errorf("expected 1 relation in posts, got %d", len(plan["posts"]))
	}
	if plan["posts"][0].Name != "author" {
		t.Errorf("expected relation name 'author', got %q", plan["posts"][0].Name)
	}
	// FK is NOT nullable, so relation should be non-optional
	if plan["posts"][0].Type != "Users" {
		t.Errorf("expected type 'Users', got %q", plan["posts"][0].Type)
	}

	// Users should have back-reference "posts"
	if len(plan["users"]) != 1 {
		t.Errorf("expected 1 relation in users, got %d", len(plan["users"]))
	}
	if plan["users"][0].Name != "posts" {
		t.Errorf("expected relation name 'posts', got %q", plan["users"][0].Name)
	}
	if plan["users"][0].Type != "Posts[]" {
		t.Errorf("expected type 'Posts[]', got %q", plan["users"][0].Type)
	}
}

func TestBuildRelationPlan_SelfReferentialFK(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "manager_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: true},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
				ForeignKeys: []schema.ForeignKey{
					{
						Name:       "fk_users_manager",
						Columns:    []string{"manager_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					},
				},
			},
		},
	}

	writer := NewWriter(DefaultNamingConvention())
	plan, err := writer.buildRelationPlan(s)
	if err != nil {
		t.Fatalf("buildRelationPlan failed: %v", err)
	}

	userRelations := plan["users"]
	if len(userRelations) != 2 {
		t.Errorf("expected 2 relations for self-referential FK, got %d", len(userRelations))
	}

	// Should have forward (manager) and back-reference (subordinates or similar)
	hasManager := false
	hasBackRef := false

	for _, rel := range userRelations {
		if rel.Name == "manager" {
			hasManager = true
			if rel.Type != "Users?" {
				t.Errorf("expected manager type 'Users?', got %q", rel.Type)
			}
		}
		// Back-reference will have a disambiguated name due to self-relation
		if strings.Contains(rel.Type, "Users[]") && rel.Name != "manager" {
			hasBackRef = true
		}
	}

	if !hasManager {
		t.Error("expected manager relation not found")
	}
	if !hasBackRef {
		t.Error("expected back-reference relation not found")
	}
}

func TestBuildRelationPlan_MultipleFKsToSameTable(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
			},
			{
				Name: "posts",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "author_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "reviewer_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: true},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
				ForeignKeys: []schema.ForeignKey{
					{
						Name:       "fk_posts_author",
						Columns:    []string{"author_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					},
					{
						Name:       "fk_posts_reviewer",
						Columns:    []string{"reviewer_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					},
				},
			},
		},
	}

	writer := NewWriter(DefaultNamingConvention())
	plan, err := writer.buildRelationPlan(s)
	if err != nil {
		t.Fatalf("buildRelationPlan failed: %v", err)
	}

	postRelations := plan["posts"]
	if len(postRelations) != 2 {
		t.Errorf("expected 2 relations in posts, got %d", len(postRelations))
	}

	hasAuthor := false
	hasReviewer := false
	for _, rel := range postRelations {
		if rel.Name == "author" {
			hasAuthor = true
		}
		if rel.Name == "reviewer" {
			hasReviewer = true
		}
	}

	if !hasAuthor {
		t.Error("expected author relation not found")
	}
	if !hasReviewer {
		t.Error("expected reviewer relation not found")
	}

	// Users should have 2 back-references, disambiguated
	userRelations := plan["users"]
	if len(userRelations) != 2 {
		t.Errorf("expected 2 back-reference relations in users, got %d", len(userRelations))
	}
}

func TestBuildRelationPlan_CompositeFK(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "accounts",
				Columns: []schema.Column{
					{Name: "tenant_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "account_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"tenant_id", "account_id"}},
			},
			{
				Name: "posts",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "author_tenant_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "author_account_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
				ForeignKeys: []schema.ForeignKey{
					{
						Name:       "fk_posts_author",
						Columns:    []string{"author_tenant_id", "author_account_id"},
						RefTable:   "accounts",
						RefColumns: []string{"tenant_id", "account_id"},
					},
				},
			},
		},
	}

	writer := NewWriter(DefaultNamingConvention())
	plan, err := writer.buildRelationPlan(s)
	if err != nil {
		t.Fatalf("buildRelationPlan failed: %v", err)
	}

	postRelations := plan["posts"]
	if len(postRelations) != 1 {
		t.Errorf("expected 1 relation in posts, got %d", len(postRelations))
	}

	rel := postRelations[0]
	// Composite FKs fall back to table name since columns don't strip to single base
	if rel.Name != "accounts" {
		t.Errorf("expected relation name 'accounts', got %q", rel.Name)
	}

	// Check @relation attribute includes both fields
	if !strings.Contains(rel.Attributes[0], "authorTenantId") && !strings.Contains(rel.Attributes[0], "author_tenant_id") {
		t.Errorf("expected composite field reference in %q", rel.Attributes[0])
	}
}

func TestBuildRelationPlan_NoForeignKeys(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "standalone",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
			},
		},
	}

	writer := NewWriter(DefaultNamingConvention())
	plan, err := writer.buildRelationPlan(s)
	if err != nil {
		t.Fatalf("buildRelationPlan failed: %v", err)
	}

	if len(plan["standalone"]) != 0 {
		t.Errorf("expected no relations for standalone table, got %d", len(plan["standalone"]))
	}
}

func TestBuildRelationPlan_ExternalRefTable(t *testing.T) {
	// FK references a table not in the schema (e.g., external database)
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "posts",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "author_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
				ForeignKeys: []schema.ForeignKey{
					{
						Name:       "fk_posts_author",
						Columns:    []string{"author_id"},
						RefTable:   "users", // Not in schema
						RefColumns: []string{"id"},
					},
				},
			},
		},
	}

	writer := NewWriter(DefaultNamingConvention())
	plan, err := writer.buildRelationPlan(s)
	if err != nil {
		t.Fatalf("buildRelationPlan failed: %v", err)
	}

	// Should skip the external FK (no model to reference)
	if len(plan["posts"]) != 0 {
		t.Errorf("expected no relations when ref table is external, got %d", len(plan["posts"]))
	}
}

func TestForwardRelationFieldName_IDSuffix(t *testing.T) {
	tests := []struct {
		name       string
		fkColumns  []string
		refTable   string
		columnName string
		expected   string
	}{
		{
			name:       "standard user_id suffix",
			fkColumns:  []string{"user_id"},
			refTable:   "users",
			columnName: "user_id",
			expected:   "user",
		},
		{
			name:       "camelCase userId",
			fkColumns:  []string{"userId"},
			refTable:   "users",
			columnName: "userId",
			expected:   "user",
		},
		{
			name:       "multiple columns - fallback to table",
			fkColumns:  []string{"user_id", "tenant_id"},
			refTable:   "users",
			columnName: "user_id",
			expected:   "users",
		},
		{
			name:       "no _id suffix - fallback to table",
			fkColumns:  []string{"creator"},
			refTable:   "users",
			columnName: "creator",
			expected:   "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewWriter(DefaultNamingConvention())
			fk := schema.ForeignKey{
				Name:       "test_fk",
				Columns:    tt.fkColumns,
				RefTable:   tt.refTable,
				RefColumns: []string{"id"},
			}

			result := writer.forwardRelationFieldName(fk)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestAnyColumnNullable_Mixed(t *testing.T) {
	table := &schema.Table{
		Name: "posts",
		Columns: []schema.Column{
			{Name: "author_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
			{Name: "reviewer_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: true},
		},
	}

	// Check nullable column
	if !anyColumnNullable(table, []string{"reviewer_id"}) {
		t.Error("expected anyColumnNullable to return true for nullable column")
	}

	// Check non-nullable column
	if anyColumnNullable(table, []string{"author_id"}) {
		t.Error("expected anyColumnNullable to return false for non-nullable column")
	}

	// Check mixed - should return true if ANY is nullable
	if !anyColumnNullable(table, []string{"author_id", "reviewer_id"}) {
		t.Error("expected anyColumnNullable to return true when any column is nullable")
	}

	// Check non-existent column
	if anyColumnNullable(table, []string{"nonexistent"}) {
		t.Error("expected anyColumnNullable to return false for non-existent column")
	}
}

func TestHasUniqueOn_PrimaryKey(t *testing.T) {
	table := &schema.Table{
		Name: "users",
		Columns: []schema.Column{
			{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
		},
		PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
	}

	if !hasUniqueOn(table, []string{"id"}) {
		t.Error("expected hasUniqueOn to return true for primary key")
	}
}

func TestHasUniqueOn_UniqueConstraint(t *testing.T) {
	table := &schema.Table{
		Name: "users",
		Columns: []schema.Column{
			{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
			{Name: "email", Type: schema.DataType{Kind: schema.TypeText}, Nullable: false},
		},
		Constraints: []schema.Constraint{
			{
				Name:    "unique_email",
				Type:    schema.ConstraintUnique,
				Columns: []string{"email"},
			},
		},
	}

	if !hasUniqueOn(table, []string{"email"}) {
		t.Error("expected hasUniqueOn to return true for unique constraint")
	}

	if hasUniqueOn(table, []string{"id"}) {
		t.Error("expected hasUniqueOn to return false for non-unique column")
	}
}

func TestHasUniqueOn_UniqueIndex(t *testing.T) {
	table := &schema.Table{
		Name: "users",
		Columns: []schema.Column{
			{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
			{Name: "username", Type: schema.DataType{Kind: schema.TypeText}, Nullable: false},
		},
		Indexes: []schema.Index{
			{
				Name:   "unique_username",
				Unique: true,
				Columns: []schema.IndexColumn{
					{Name: "username"},
				},
			},
		},
	}

	if !hasUniqueOn(table, []string{"username"}) {
		t.Error("expected hasUniqueOn to return true for unique index")
	}
}

func TestSameColumnSet_Identical(t *testing.T) {
	a := []string{"id"}
	b := []string{"id"}

	if !sameColumnSet(a, b) {
		t.Error("expected sameColumnSet to return true for identical sets")
	}
}

func TestSameColumnSet_DifferentOrder(t *testing.T) {
	a := []string{"tenant_id", "user_id"}
	b := []string{"user_id", "tenant_id"}

	if !sameColumnSet(a, b) {
		t.Error("expected sameColumnSet to return true for same columns in different order")
	}
}

func TestSameColumnSet_Different(t *testing.T) {
	a := []string{"id"}
	b := []string{"author_id"}

	if sameColumnSet(a, b) {
		t.Error("expected sameColumnSet to return false for different sets")
	}
}

func TestSameColumnSet_Empty(t *testing.T) {
	a := []string{}
	b := []string{}

	if sameColumnSet(a, b) {
		t.Error("expected sameColumnSet to return false for empty sets")
	}
}

func TestSameColumnSet_LengthMismatch(t *testing.T) {
	a := []string{"id"}
	b := []string{"id", "tenant_id"}

	if sameColumnSet(a, b) {
		t.Error("expected sameColumnSet to return false for different lengths")
	}
}

func TestPrismaReferentialAction_Cascade(t *testing.T) {
	result := prismaReferentialAction(schema.ActionCascade)
	if result != "Cascade" {
		t.Errorf("expected 'Cascade', got %q", result)
	}
}

func TestPrismaReferentialAction_SetNull(t *testing.T) {
	result := prismaReferentialAction(schema.ActionSetNull)
	if result != "SetNull" {
		t.Errorf("expected 'SetNull', got %q", result)
	}
}

func TestPrismaReferentialAction_SetDefault(t *testing.T) {
	result := prismaReferentialAction(schema.ActionSetDefault)
	if result != "SetDefault" {
		t.Errorf("expected 'SetDefault', got %q", result)
	}
}

func TestPrismaReferentialAction_Restrict(t *testing.T) {
	result := prismaReferentialAction(schema.ActionRestrict)
	if result != "Restrict" {
		t.Errorf("expected 'Restrict', got %q", result)
	}
}

func TestPrismaReferentialAction_NoAction(t *testing.T) {
	result := prismaReferentialAction(schema.ActionNoAction)
	if result != "NoAction" {
		t.Errorf("expected 'NoAction', got %q", result)
	}
}

func TestPrismaReferentialAction_Default(t *testing.T) {
	// Test unknown/default action
	result := prismaReferentialAction(schema.ReferentialAction(999))
	if result != "" {
		t.Errorf("expected empty string for unknown action, got %q", result)
	}
}

func TestForwardRelationField_WithCascadeDelete(t *testing.T) {
	table := &schema.Table{
		Name: "posts",
		Columns: []schema.Column{
			{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
			{Name: "author_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
		},
	}

	fk := schema.ForeignKey{
		Name:       "fk_posts_author",
		Columns:    []string{"author_id"},
		RefTable:   "users",
		RefColumns: []string{"id"},
		OnDelete:   schema.ActionCascade,
		OnUpdate:   schema.ActionCascade,
	}

	writer := NewWriter(DefaultNamingConvention())
	rel, err := writer.forwardRelationField("author", "Users", "", table, fk)

	if err != nil {
		t.Fatalf("forwardRelationField failed: %v", err)
	}

	if rel.Name != "author" {
		t.Errorf("expected name 'author', got %q", rel.Name)
	}

	if rel.Type != "Users" {
		t.Errorf("expected type 'Users', got %q", rel.Type)
	}

	// Check onDelete and onUpdate in attributes
	attrStr := rel.Attributes[0]
	if !strings.Contains(attrStr, "onDelete: Cascade") {
		t.Errorf("expected onDelete: Cascade in %q", attrStr)
	}
	if !strings.Contains(attrStr, "onUpdate: Cascade") {
		t.Errorf("expected onUpdate: Cascade in %q", attrStr)
	}
}

func TestForwardRelationField_NullableFK(t *testing.T) {
	table := &schema.Table{
		Name: "posts",
		Columns: []schema.Column{
			{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
			{Name: "reviewer_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: true},
		},
	}

	fk := schema.ForeignKey{
		Name:       "fk_posts_reviewer",
		Columns:    []string{"reviewer_id"},
		RefTable:   "users",
		RefColumns: []string{"id"},
	}

	writer := NewWriter(DefaultNamingConvention())
	rel, err := writer.forwardRelationField("reviewer", "Users", "", table, fk)

	if err != nil {
		t.Fatalf("forwardRelationField failed: %v", err)
	}

	if rel.Type != "Users?" {
		t.Errorf("expected type 'Users?' for nullable FK, got %q", rel.Type)
	}
}

func TestBackRelationField_OneToMany(t *testing.T) {
	table := &schema.Table{
		Name: "posts",
		Columns: []schema.Column{
			{Name: "author_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
		},
	}

	fk := schema.ForeignKey{
		Name:       "fk_posts_author",
		Columns:    []string{"author_id"},
		RefTable:   "users",
		RefColumns: []string{"id"},
	}

	writer := NewWriter(DefaultNamingConvention())
	rel := writer.backRelationField("posts", "Posts", "", table, fk)

	if rel.Type != "Posts[]" {
		t.Errorf("expected type 'Posts[]', got %q", rel.Type)
	}
}

func TestBackRelationField_OneToOne(t *testing.T) {
	table := &schema.Table{
		Name: "profiles",
		Columns: []schema.Column{
			{Name: "user_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
		},
		Constraints: []schema.Constraint{
			{
				Name:    "unique_user_id",
				Type:    schema.ConstraintUnique,
				Columns: []string{"user_id"},
			},
		},
	}

	fk := schema.ForeignKey{
		Name:       "fk_profiles_user",
		Columns:    []string{"user_id"},
		RefTable:   "users",
		RefColumns: []string{"id"},
	}

	writer := NewWriter(DefaultNamingConvention())
	rel := writer.backRelationField("profile", "Profiles", "", table, fk)

	if rel.Type != "Profiles?" {
		t.Errorf("expected type 'Profiles?' for one-to-one, got %q", rel.Type)
	}
}

func TestBuildRelationPlan_RelationNameDisambiguation(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
			},
			{
				Name: "posts",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "author_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "editor_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
				ForeignKeys: []schema.ForeignKey{
					{
						Name:       "fk_posts_author",
						Columns:    []string{"author_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					},
					{
						Name:       "fk_posts_editor",
						Columns:    []string{"editor_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
					},
				},
			},
		},
	}

	writer := NewWriter(DefaultNamingConvention())
	plan, err := writer.buildRelationPlan(s)
	if err != nil {
		t.Fatalf("buildRelationPlan failed: %v", err)
	}

	// Users should have disambiguated back-reference names
	userRelations := plan["users"]
	if len(userRelations) != 2 {
		t.Errorf("expected 2 back-references in users, got %d", len(userRelations))
	}

	// Both should have @relation with explicit relation names
	for _, rel := range userRelations {
		if !strings.Contains(rel.Attributes[0], "@relation(") {
			t.Errorf("expected @relation in %q", rel.Attributes[0])
		}
	}
}
