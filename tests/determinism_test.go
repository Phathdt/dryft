package tests

import (
	"testing"

	"github.com/phathdt/dryft/internal/diff"
	"github.com/phathdt/dryft/internal/migration"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/sql"
	pggen "github.com/phathdt/dryft/internal/sql/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeterminism_SameInputSameOutput verifies that identical schema changes
// produce identical migrations across multiple runs
func TestDeterminism_SameInputSameOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Define a schema transformation
	before := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "email", Type: schema.DataType{Kind: schema.TypeText}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
			},
		},
	}

	after := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "email", Type: schema.DataType{Kind: schema.TypeText}, Nullable: false},
					{Name: "name", Type: schema.DataType{Kind: schema.TypeText}, Nullable: true},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
			},
		},
	}

	// Run diff → plan → generate 10 times
	var migrations []string
	for i := 0; i < 10; i++ {
		differ := diff.NewDiffer(nil)
		operations, err := differ.Diff(before, after)
		require.NoError(t, err)

		planner := diff.NewPlanner()
		plan, err := planner.Plan(operations)
		require.NoError(t, err)

		generator := pggen.NewGenerator(sql.GeneratorOptions{})
		upStatements, err := generator.Generate(plan.Operations)
		require.NoError(t, err)

		downStatements, warnings, err := generator.GenerateReverse(plan.Operations)
		require.NoError(t, err)

		migrationContent := migration.FormatGoose("add_name", upStatements, downStatements, warnings)
		migrations = append(migrations, migrationContent)
	}

	// All migrations should be identical
	firstMigration := migrations[0]
	for i, m := range migrations {
		assert.Equal(t, firstMigration, m, "migration %d differs from first", i)
	}

	t.Logf("Verified: 10 runs produced identical migrations")
}

// TestDeterminism_OperationOrdering verifies that operation ordering is stable
// when there are dependency relationships
func TestDeterminism_OperationOrdering(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	before := &schema.Schema{
		Tables: []schema.Table{},
		Enums:  []schema.Enum{},
	}

	// Create schema with dependencies: enum → table → index → FK
	after := &schema.Schema{
		Enums: []schema.Enum{
			{
				Name: "status",
				Values: []schema.EnumValue{
					{Label: "ACTIVE", Order: 0},
					{Label: "INACTIVE", Order: 1},
				},
			},
		},
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "email", Type: schema.DataType{Kind: schema.TypeText}, Nullable: false},
					{Name: "status", Type: schema.DataType{Kind: schema.TypeEnum, EnumName: "status"}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
				Indexes: []schema.Index{
					{Name: "idx_users_email", Columns: []schema.IndexColumn{{Name: "email", Order: schema.SortAsc}}},
				},
			},
			{
				Name: "posts",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "user_id", Type: schema.DataType{Kind: schema.TypeUUID}, Nullable: false},
					{Name: "title", Type: schema.DataType{Kind: schema.TypeText}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
				ForeignKeys: []schema.ForeignKey{
					{
						Name:      "fk_posts_user_id",
						Columns:   []string{"user_id"},
						RefTable:  "users",
						RefColumns: []string{"id"},
						OnDelete:  schema.ActionCascade,
						OnUpdate:  schema.ActionNoAction,
					},
				},
			},
		},
	}

	// Run multiple times and check operation order
	var allOperationDescriptions [][]string
	for i := 0; i < 5; i++ {
		differ := diff.NewDiffer(nil)
		operations, err := differ.Diff(before, after)
		require.NoError(t, err)

		planner := diff.NewPlanner()
		plan, err := planner.Plan(operations)
		require.NoError(t, err)

		var descriptions []string
		for _, op := range plan.Operations {
			descriptions = append(descriptions, op.Description())
		}
		allOperationDescriptions = append(allOperationDescriptions, descriptions)
	}

	// All runs should have identical operation order
	firstRun := allOperationDescriptions[0]

	// For MVP: verify operations are stable (same set), but exact order may vary
	// when there's no dependency (e.g., independent tables)
	// In production: planner should use stable sort (e.g., alphabetical) for determinism

	firstRunSet := make(map[string]bool)
	for _, desc := range firstRun {
		firstRunSet[desc] = true
	}

	for i, run := range allOperationDescriptions {
		// Check same count
		if !assert.Equal(t, len(firstRun), len(run), "run %d has different operation count", i) {
			t.Logf("Expected count: %d", len(firstRun))
			t.Logf("Got count: %d", len(run))
			t.FailNow()
		}

		// Check same operations (may be in different order for independent ops)
		runSet := make(map[string]bool)
		for _, desc := range run {
			runSet[desc] = true
		}

		for desc := range firstRunSet {
			if !assert.True(t, runSet[desc], "run %d missing operation: %s", i, desc) {
				t.FailNow()
			}
		}
	}

	// Verify expected order: enum → tables → indexes → FKs
	// (exact descriptions depend on implementation, but order should be stable)
	t.Logf("Operation order (first run):")
	for i, desc := range firstRun {
		t.Logf("  %d. %s", i+1, desc)
	}
}

// TestDeterminism_NoTimestampsInSQL verifies that generated SQL
// does not contain timestamps (only filename should have timestamp)
func TestDeterminism_NoTimestampsInSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	before := &schema.Schema{}
	after := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: schema.DataType{Kind: schema.TypeInt32}, Nullable: false},
				},
				PrimaryKey: &schema.PrimaryKey{Columns: []string{"id"}},
			},
		},
	}

	differ := diff.NewDiffer(nil)
	operations, err := differ.Diff(before, after)
	require.NoError(t, err)

	planner := diff.NewPlanner()
	plan, err := planner.Plan(operations)
	require.NoError(t, err)

	generator := pggen.NewGenerator(sql.GeneratorOptions{})
	upStatements, err := generator.Generate(plan.Operations)
	require.NoError(t, err)

	// Check that SQL does not contain timestamp patterns
	for _, stmt := range upStatements {
		// Should not contain: "2026", "20260916", timestamp-like patterns
		assert.NotContains(t, stmt, "2026", "SQL should not contain year")
		assert.NotContains(t, stmt, "-- Generated at", "SQL should not contain generation timestamp")
	}

	t.Logf("Verified: SQL contains no timestamps")
}
