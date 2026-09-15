package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/phathdt/dryft/internal/schema"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB creates a PostgreSQL testcontainer and returns connection string.
func setupTestDB(t *testing.T) (string, func()) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	cleanup := func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return connStr, cleanup
}

// execSQL executes SQL statements on the test database.
func execSQL(t *testing.T, connStr string, sql string) {
	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	_, err = intr.pool.Exec(ctx, sql)
	if err != nil {
		t.Fatalf("failed to execute SQL: %v", err)
	}
}

func TestIntrospect_EmptyDatabase(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	s, err := intr.Introspect(ctx)
	if err != nil {
		t.Fatalf("failed to introspect: %v", err)
	}

	if len(s.Tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(s.Tables))
	}

	if len(s.Enums) != 0 {
		t.Errorf("expected 0 enums, got %d", len(s.Enums))
	}
}

func TestIntrospect_SingleTableWithPK(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()

	execSQL(t, connStr, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL
		);
	`)

	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	s, err := intr.Introspect(ctx)
	if err != nil {
		t.Fatalf("failed to introspect: %v", err)
	}

	if len(s.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(s.Tables))
	}

	usersTable := s.Tables[0]
	if usersTable.Name != "users" {
		t.Errorf("expected table name 'users', got %q", usersTable.Name)
	}

	if usersTable.PrimaryKey == nil {
		t.Fatal("expected primary key, got nil")
	}

	if len(usersTable.PrimaryKey.Columns) != 1 || usersTable.PrimaryKey.Columns[0] != "id" {
		t.Errorf("expected PK on 'id', got %v", usersTable.PrimaryKey.Columns)
	}
}

func TestIntrospect_ForeignKeyRelationship(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()

	execSQL(t, connStr, `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL
		);

		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL
		);
	`)

	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	s, err := intr.Introspect(ctx)
	if err != nil {
		t.Fatalf("failed to introspect: %v", err)
	}

	if len(s.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(s.Tables))
	}

	var postsTable *schema.Table
	for i := range s.Tables {
		if s.Tables[i].Name == "posts" {
			postsTable = &s.Tables[i]
			break
		}
	}

	if postsTable == nil {
		t.Fatal("posts table not found")
	}

	if len(postsTable.ForeignKeys) != 1 {
		t.Fatalf("expected 1 foreign key, got %d", len(postsTable.ForeignKeys))
	}

	fk := postsTable.ForeignKeys[0]
	if fk.RefTable != "users" {
		t.Errorf("expected FK to reference 'users', got %q", fk.RefTable)
	}

	if fk.OnDelete != schema.ActionCascade {
		t.Errorf("expected ON DELETE CASCADE, got %v", fk.OnDelete)
	}
}

func TestIntrospect_EnumType(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()

	execSQL(t, connStr, `
		CREATE TYPE status AS ENUM ('active', 'inactive', 'pending');

		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			status status DEFAULT 'active'
		);
	`)

	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	s, err := intr.Introspect(ctx)
	if err != nil {
		t.Fatalf("failed to introspect: %v", err)
	}

	if len(s.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(s.Enums))
	}

	statusEnum := s.Enums[0]
	if statusEnum.Name != "status" {
		t.Errorf("expected enum name 'status', got %q", statusEnum.Name)
	}

	expectedValues := []string{"active", "inactive", "pending"}
	if len(statusEnum.Values) != len(expectedValues) {
		t.Fatalf("expected %d enum values, got %d", len(expectedValues), len(statusEnum.Values))
	}

	for i, expected := range expectedValues {
		if statusEnum.Values[i].Label != expected {
			t.Errorf("expected enum value %d to be %q, got %q", i, expected, statusEnum.Values[i].Label)
		}
		if statusEnum.Values[i].Order != i {
			t.Errorf("expected enum value %d to have order %d, got %d", i, i, statusEnum.Values[i].Order)
		}
	}
}

func TestIntrospect_ArrayColumn(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()

	execSQL(t, connStr, `
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			tags TEXT[]
		);
	`)

	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	s, err := intr.Introspect(ctx)
	if err != nil {
		t.Fatalf("failed to introspect: %v", err)
	}

	postsTable := s.Tables[0]
	var tagsCol *schema.Column
	for i := range postsTable.Columns {
		if postsTable.Columns[i].Name == "tags" {
			tagsCol = &postsTable.Columns[i]
			break
		}
	}

	if tagsCol == nil {
		t.Fatal("tags column not found")
	}

	if tagsCol.Type.ArrayDepth != 1 {
		t.Errorf("expected array depth 1, got %d", tagsCol.Type.ArrayDepth)
	}

	if tagsCol.Type.Kind != schema.TypeText {
		t.Errorf("expected base type TEXT, got %v", tagsCol.Type.Kind)
	}
}

func TestIntrospect_PartialIndex(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()

	execSQL(t, connStr, `
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			user_id UUID NOT NULL,
			title TEXT NOT NULL
		);

		CREATE INDEX idx_posts_user ON posts(user_id) WHERE user_id IS NOT NULL;
	`)

	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	s, err := intr.Introspect(ctx)
	if err != nil {
		t.Fatalf("failed to introspect: %v", err)
	}

	postsTable := s.Tables[0]
	if len(postsTable.Indexes) != 1 {
		t.Fatalf("expected 1 index, got %d", len(postsTable.Indexes))
	}

	idx := postsTable.Indexes[0]
	if idx.Where == "" {
		t.Error("expected partial index with WHERE clause, got empty string")
	}
}

func TestIntrospect_CheckConstraint(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()

	execSQL(t, connStr, `
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			CHECK (length(title) > 0)
		);
	`)

	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	s, err := intr.Introspect(ctx)
	if err != nil {
		t.Fatalf("failed to introspect: %v", err)
	}

	postsTable := s.Tables[0]

	var checkConstraint *schema.Constraint
	for i := range postsTable.Constraints {
		if postsTable.Constraints[i].Type == schema.ConstraintCheck {
			checkConstraint = &postsTable.Constraints[i]
			break
		}
	}

	if checkConstraint == nil {
		t.Fatal("CHECK constraint not found")
	}

	if checkConstraint.Expression == "" {
		t.Error("expected CHECK constraint expression, got empty string")
	}
}

func TestIntrospect_SerialColumn(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()

	execSQL(t, connStr, `
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL
		);
	`)

	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	s, err := intr.Introspect(ctx)
	if err != nil {
		t.Fatalf("failed to introspect: %v", err)
	}

	postsTable := s.Tables[0]
	var idCol *schema.Column
	for i := range postsTable.Columns {
		if postsTable.Columns[i].Name == "id" {
			idCol = &postsTable.Columns[i]
			break
		}
	}

	if idCol == nil {
		t.Fatal("id column not found")
	}

	if idCol.Default == nil {
		t.Fatal("expected default value for SERIAL column, got nil")
	}

	if idCol.Default.Kind != schema.DefaultSequence {
		t.Errorf("expected default kind Sequence, got %v", idCol.Default.Kind)
	}
}

func TestIntrospect_MultiColumnPK(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()

	execSQL(t, connStr, `
		CREATE TABLE user_roles (
			user_id UUID NOT NULL,
			role_id UUID NOT NULL,
			PRIMARY KEY (user_id, role_id)
		);
	`)

	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	s, err := intr.Introspect(ctx)
	if err != nil {
		t.Fatalf("failed to introspect: %v", err)
	}

	table := s.Tables[0]
	if table.PrimaryKey == nil {
		t.Fatal("expected primary key, got nil")
	}

	if len(table.PrimaryKey.Columns) != 2 {
		t.Fatalf("expected 2 PK columns, got %d", len(table.PrimaryKey.Columns))
	}

	expectedCols := []string{"user_id", "role_id"}
	for i, expected := range expectedCols {
		if table.PrimaryKey.Columns[i] != expected {
			t.Errorf("expected PK column %d to be %q, got %q", i, expected, table.PrimaryKey.Columns[i])
		}
	}
}

func TestIntrospect_CompositeForeignKey(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()

	execSQL(t, connStr, `
		CREATE TABLE users (
			id UUID NOT NULL,
			tenant_id UUID NOT NULL,
			email VARCHAR(255) NOT NULL,
			PRIMARY KEY (tenant_id, id)
		);

		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			tenant_id UUID NOT NULL,
			user_id UUID NOT NULL,
			title TEXT NOT NULL,
			FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id)
		);
	`)

	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	s, err := intr.Introspect(ctx)
	if err != nil {
		t.Fatalf("failed to introspect: %v", err)
	}

	var postsTable *schema.Table
	for i := range s.Tables {
		if s.Tables[i].Name == "posts" {
			postsTable = &s.Tables[i]
			break
		}
	}

	if postsTable == nil {
		t.Fatal("posts table not found")
	}

	if len(postsTable.ForeignKeys) != 1 {
		t.Fatalf("expected 1 foreign key, got %d", len(postsTable.ForeignKeys))
	}

	fk := postsTable.ForeignKeys[0]
	if len(fk.Columns) != 2 {
		t.Fatalf("expected 2 FK columns, got %d", len(fk.Columns))
	}

	if len(fk.RefColumns) != 2 {
		t.Fatalf("expected 2 ref columns, got %d", len(fk.RefColumns))
	}
}

func TestIntrospect_ComplexSchema(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()

	execSQL(t, connStr, `
		CREATE TYPE status AS ENUM ('active', 'inactive');

		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			status status DEFAULT 'active',
			created_at TIMESTAMPTZ DEFAULT now()
		);

		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			tags TEXT[],
			CHECK (length(title) > 0)
		);

		CREATE INDEX idx_posts_user ON posts(user_id) WHERE user_id IS NOT NULL;
	`)

	ctx := context.Background()
	intr, err := NewPostgresIntrospector(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer intr.Close()

	s, err := intr.Introspect(ctx)
	if err != nil {
		t.Fatalf("failed to introspect: %v", err)
	}

	// Verify tables
	if len(s.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(s.Tables))
	}

	// Verify enums
	if len(s.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(s.Enums))
	}

	// Find users table
	var usersTable *schema.Table
	for i := range s.Tables {
		if s.Tables[i].Name == "users" {
			usersTable = &s.Tables[i]
			break
		}
	}

	if usersTable == nil {
		t.Fatal("users table not found")
	}

	// Verify PK
	if usersTable.PrimaryKey == nil {
		t.Fatal("expected primary key on users table")
	}

	if len(usersTable.PrimaryKey.Columns) != 1 || usersTable.PrimaryKey.Columns[0] != "id" {
		t.Errorf("expected PK on 'id', got %v", usersTable.PrimaryKey.Columns)
	}

	// Find posts table
	var postsTable *schema.Table
	for i := range s.Tables {
		if s.Tables[i].Name == "posts" {
			postsTable = &s.Tables[i]
			break
		}
	}

	if postsTable == nil {
		t.Fatal("posts table not found")
	}

	// Verify FK
	if len(postsTable.ForeignKeys) != 1 {
		t.Fatalf("expected 1 foreign key, got %d", len(postsTable.ForeignKeys))
	}

	// Verify indexes
	if len(postsTable.Indexes) == 0 {
		t.Fatal("expected at least 1 index")
	}

	// Verify CHECK constraint
	hasCheck := false
	for _, c := range postsTable.Constraints {
		if c.Type == schema.ConstraintCheck {
			hasCheck = true
			break
		}
	}
	if !hasCheck {
		t.Error("expected CHECK constraint")
	}
}
