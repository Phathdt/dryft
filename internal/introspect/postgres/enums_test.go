package postgres

import (
	"context"
	"testing"

	"github.com/phathdt/dryft/internal/testutil"
)

// TestIntrospect_EnumOrderPreservation tests that enum value ordering is preserved.
func TestIntrospect_EnumOrderPreservation(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TYPE status AS ENUM ('pending', 'active', 'inactive', 'deleted');
		CREATE TABLE users (id UUID PRIMARY KEY, status status);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(schema.Enums))
	}

	statusEnum := schema.Enums[0]
	if statusEnum.Name != "status" {
		t.Errorf("expected enum name 'status', got %q", statusEnum.Name)
	}

	if len(statusEnum.Values) != 4 {
		t.Fatalf("expected 4 enum values, got %d", len(statusEnum.Values))
	}

	tests := []struct {
		idx   int
		label string
		order int
	}{
		{0, "pending", 0},
		{1, "active", 1},
		{2, "inactive", 2},
		{3, "deleted", 3},
	}

	for _, tt := range tests {
		val := statusEnum.Values[tt.idx]
		if val.Label != tt.label {
			t.Errorf("enum value[%d]: expected label %q, got %q", tt.idx, tt.label, val.Label)
		}
		if val.Order != tt.order {
			t.Errorf("enum value[%d] (%s): expected order %d, got %d", tt.idx, tt.label, tt.order, val.Order)
		}
	}
}

// TestIntrospect_MultipleEnums tests introspection of multiple enums.
func TestIntrospect_MultipleEnums(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TYPE status AS ENUM ('pending', 'active', 'completed');
		CREATE TYPE priority AS ENUM ('low', 'medium', 'high', 'critical');
		CREATE TYPE role AS ENUM ('admin', 'user', 'guest');

		CREATE TABLE tasks (
			id UUID PRIMARY KEY,
			status status,
			priority priority,
			role role
		);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Enums) != 3 {
		t.Fatalf("expected 3 enums, got %d", len(schema.Enums))
	}

	enumNames := make(map[string]bool)
	for _, e := range schema.Enums {
		enumNames[e.Name] = true
	}

	if !enumNames["status"] {
		t.Error("expected enum 'status'")
	}
	if !enumNames["priority"] {
		t.Error("expected enum 'priority'")
	}
	if !enumNames["role"] {
		t.Error("expected enum 'role'")
	}
}

// TestIntrospect_EnumWithSpecialCharacters tests enums with special characters in values.
func TestIntrospect_EnumWithSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TYPE status_ext AS ENUM ('in-progress', 'on_hold', 'to-do', 'done');
		CREATE TABLE tasks (id UUID PRIMARY KEY, status status_ext);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(schema.Enums))
	}

	statusEnum := schema.Enums[0]
	if len(statusEnum.Values) != 4 {
		t.Fatalf("expected 4 enum values, got %d", len(statusEnum.Values))
	}

	expectedLabels := []string{"in-progress", "on_hold", "to-do", "done"}
	for i, expected := range expectedLabels {
		if statusEnum.Values[i].Label != expected {
			t.Errorf("enum value[%d]: expected %q, got %q", i, expected, statusEnum.Values[i].Label)
		}
	}
}

// TestIntrospect_EnumSingleValue tests enum with single value.
func TestIntrospect_EnumSingleValue(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TYPE singleton AS ENUM ('only');
		CREATE TABLE test_table (id UUID PRIMARY KEY, val singleton);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(schema.Enums))
	}

	singletonEnum := schema.Enums[0]
	if len(singletonEnum.Values) != 1 {
		t.Fatalf("expected 1 enum value, got %d", len(singletonEnum.Values))
	}

	if singletonEnum.Values[0].Label != "only" {
		t.Errorf("expected 'only', got %q", singletonEnum.Values[0].Label)
	}
}

// TestIntrospect_EnumCaseSensitivity tests that enum values are case-sensitive.
func TestIntrospect_EnumCaseSensitivity(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TYPE case_test AS ENUM ('Value', 'value', 'VALUE');
		CREATE TABLE test_table (id UUID PRIMARY KEY, val case_test);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(schema.Enums))
	}

	caseEnum := schema.Enums[0]
	if len(caseEnum.Values) != 3 {
		t.Fatalf("expected 3 enum values, got %d", len(caseEnum.Values))
	}

	expectedLabels := []string{"Value", "value", "VALUE"}
	for i, expected := range expectedLabels {
		if caseEnum.Values[i].Label != expected {
			t.Errorf("enum value[%d]: expected %q, got %q", i, expected, caseEnum.Values[i].Label)
		}
	}
}

// TestIntrospect_UnusedEnum tests enum that exists but is not used by any column.
func TestIntrospect_UnusedEnum(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TYPE status AS ENUM ('active', 'inactive');
		CREATE TYPE unused_type AS ENUM ('value1', 'value2');
		CREATE TABLE users (id UUID PRIMARY KEY, status status);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	// MVP may or may not include unused enums - just verify it doesn't crash
	t.Logf("Total enums found: %d", len(schema.Enums))

	statusFound := false
	for _, e := range schema.Enums {
		if e.Name == "status" {
			statusFound = true
		}
	}

	if !statusFound {
		t.Error("expected to find enum 'status'")
	}
}

// TestIntrospect_EnumWithNumericLike tests enum with numeric-looking values.
func TestIntrospect_EnumWithNumericLike(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	err = pg.ExecuteSQL(ctx, `
		CREATE TYPE level AS ENUM ('1_low', '2_medium', '3_high', '4_critical');
		CREATE TABLE alerts (id UUID PRIMARY KEY, level level);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	inspector, err := NewPostgresIntrospector(ctx, pg.ConnString)
	if err != nil {
		t.Fatalf("create inspector: %v", err)
	}
	defer inspector.Close()

	schema, err := inspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}

	if len(schema.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(schema.Enums))
	}

	levelEnum := schema.Enums[0]
	if len(levelEnum.Values) != 4 {
		t.Fatalf("expected 4 enum values, got %d", len(levelEnum.Values))
	}

	expectedLabels := []string{"1_low", "2_medium", "3_high", "4_critical"}
	for i, expected := range expectedLabels {
		if levelEnum.Values[i].Label != expected {
			t.Errorf("enum value[%d]: expected %q, got %q", i, expected, levelEnum.Values[i].Label)
		}
	}
}
