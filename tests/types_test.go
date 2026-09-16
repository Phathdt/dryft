package tests

import (
	"context"
	"testing"

	"github.com/phathdt/dryft/internal/introspect/postgres"
	"github.com/phathdt/dryft/internal/prisma"
	"github.com/phathdt/dryft/internal/schema"
	"github.com/phathdt/dryft/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTypes_UUID tests UUID type: DB → Prisma → SQL roundtrip
func TestTypes_UUID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value UUID
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]
	assert.Len(t, table.Columns, 2)

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeUUID, valueCol.Type.Kind)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "@db.Uuid")
}

// TestTypes_Text tests TEXT type
func TestTypes_Text(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value TEXT NOT NULL
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeText, valueCol.Type.Kind)
	assert.False(t, valueCol.Nullable)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "value")
	assert.Contains(t, prismaText, "String")
}

// TestTypes_VarChar tests VARCHAR(n) with length
func TestTypes_VarChar(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value VARCHAR(255)
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeVarChar, valueCol.Type.Kind)
	assert.Equal(t, 255, valueCol.Type.Precision)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "@db.VarChar(255)")
}

// TestTypes_Char tests CHAR(n) with length
func TestTypes_Char(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value CHAR(10)
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeChar, valueCol.Type.Kind)
	assert.Equal(t, 10, valueCol.Type.Precision)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "@db.Char(10)")
}

// TestTypes_Int32 tests INTEGER type
func TestTypes_Int32(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value INTEGER
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeInt32, valueCol.Type.Kind)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "Int")
}

// TestTypes_Int64 tests BIGINT type
func TestTypes_Int64(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value BIGINT
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeInt64, valueCol.Type.Kind)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "BigInt")
}

// TestTypes_Boolean tests BOOLEAN type
func TestTypes_Boolean(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value BOOLEAN NOT NULL DEFAULT false
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeBool, valueCol.Type.Kind)
	assert.False(t, valueCol.Nullable)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "Boolean")
}

// TestTypes_Float32 tests REAL type
func TestTypes_Float32(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value REAL
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeFloat32, valueCol.Type.Kind)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "@db.Real")
}

// TestTypes_Float64 tests DOUBLE PRECISION type
func TestTypes_Float64(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value DOUBLE PRECISION
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeFloat64, valueCol.Type.Kind)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "Float")
}

// TestTypes_Numeric tests NUMERIC(p,s) with precision/scale
func TestTypes_Numeric(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value NUMERIC(10,2)
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeNumeric, valueCol.Type.Kind)
	assert.Equal(t, 10, valueCol.Type.Precision)
	assert.Equal(t, 2, valueCol.Type.Scale)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "@db.Numeric(10,2)")
	assert.Contains(t, prismaText, "Decimal")
}

// TestTypes_Timestamp tests TIMESTAMP WITHOUT TIME ZONE
func TestTypes_Timestamp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value TIMESTAMP WITHOUT TIME ZONE
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeTimestamp, valueCol.Type.Kind)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "DateTime")
}

// TestTypes_TimestampTZ tests TIMESTAMP WITH TIME ZONE
func TestTypes_TimestampTZ(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeTimestampTZ, valueCol.Type.Kind)
	assert.False(t, valueCol.Nullable)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "@db.Timestamptz")
}

// TestTypes_Date tests DATE type
func TestTypes_Date(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value DATE
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeDate, valueCol.Type.Kind)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "@db.Date")
}

// TestTypes_JSON tests JSON type
func TestTypes_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value JSON
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeJSON, valueCol.Type.Kind)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "Json")
}

// TestTypes_JSONB tests JSONB type
func TestTypes_JSONB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	container, err := testutil.StartPostgres(ctx, t)
	require.NoError(t, err)

	sql := `CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		value JSONB
	);`
	err = container.ExecuteSQL(ctx, sql)
	require.NoError(t, err)

	introspector, err := postgres.NewPostgresIntrospector(ctx, container.ConnString)
	require.NoError(t, err)
	schema1, err := introspector.Introspect(ctx)
	require.NoError(t, err)

	assert.Len(t, schema1.Tables, 1)
	table := schema1.Tables[0]

	var valueCol *schema.Column
	for i := range table.Columns {
		if table.Columns[i].Name == "value" {
			valueCol = &table.Columns[i]
			break
		}
	}
	require.NotNil(t, valueCol)
	assert.Equal(t, schema.TypeJSONB, valueCol.Type.Kind)

	writer := prisma.NewWriter(prisma.DefaultNamingConvention())
	prismaText, err := writer.Write(schema1)
	require.NoError(t, err)
	assert.Contains(t, prismaText, "@db.JsonB")
}
