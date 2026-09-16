# Phase 4: CLI Integration

**Status:** Complete  
**Dependencies:** Phase 1 (Parser), Phase 2 (Loader), Phase 3 (Builder)  
**Estimated Effort:** 1 day  
**Risk Level:** Medium (workflow changes)

## Objective

Integrate migration-based schema detection vào `dryft migration create` command. Replace DB introspection với migration loader khi có migration files.

## Scope

### In Scope
- Replace `introspectDatabase()` call với migration-based loader
- **100% offline** - không cần DB connection
- Empty migrations/ → empty schema (first migration scenario)
- Remove DB introspection fallback từ `migration create`
- Preserve error handling và messaging
- Update help text

### Out of Scope
- Changes to diff/plan/SQL generation logic
- Migration execution
- Schema validation beyond parsing
- DB introspection (only for `db pull` command)

## Current Flow Analysis

**File:** `internal/cli/migration.go`

Current `migrationCreateAction()` flow:
```
1. Get migration name from args
2. Load config (.dryft.yaml)
3. Check schema.prisma exists
4. Parse schema.prisma → currentSchema
5. introspectDatabase() → previousSchema  ← REPLACE THIS
6. Diff(previousSchema, currentSchema)
7. Plan operations
8. Generate SQL (up/down)
9. Format as Goose migration
10. Write to migrations/ directory
```

## Changes Required

### 1. New Schema Source Function

```go
// internal/cli/migration.go

// loadPreviousSchemaFromMigrations loads the previous schema state from migration history.
// This function is 100% offline and does not require database connection.
//
// Returns:
//   - Non-empty schema if migrations exist
//   - Empty schema if migrations directory is empty (first migration scenario)
//   - Error only if migration parsing fails
func loadPreviousSchemaFromMigrations(cfg *config.Config) (*schema.Schema, error) {
	migrationDir := cfg.Migration.Directory

	if migrationDir == "" {
		return nil, fmt.Errorf("migration directory not configured")
	}

	// Load schema from migration files
	s, err := migration.LoadSchemaFromMigrations(migrationDir)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to load schema from migrations in %s: %w\n\n"+
				"Fix the migration file syntax error and try again.",
			migrationDir, err,
		)
	}

	if s == nil {
		// No migrations found - return empty schema for first migration
		fmt.Println("No migrations found, treating as empty schema (first migration)")
		return &schema.Schema{
			Tables: []schema.Table{},
			Enums:  []schema.Enum{},
		}, nil
	}

	// Successfully loaded from migrations
	count := countMigrationFiles(migrationDir)
	fmt.Printf("✓ Loaded schema from %d migration(s)\n", count)
	return s, nil
}

// countMigrationFiles returns count of .sql files in directory
func countMigrationFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			count++
		}
	}
	return count
}
```

### 2. Update migrationCreateAction

```go
func migrationCreateAction(_ context.Context, cmd *cli.Command) error {
	// ... (steps 1-4 unchanged)

	// 5. Load previous schema state from migration history (100% offline)
	previousSchema, err := loadPreviousSchemaFromMigrations(cfg)
	if err != nil {
		return fmt.Errorf("failed to load previous schema state: %w", err)
	}

	// ... (steps 6-12 unchanged)
}
```

### 3. Update CLI Command (Remove --from-db Flag)

```go
func MigrationCommand() *cli.Command {
	return &cli.Command{
		Name:  "migration",
		Usage: "Migration operations",
		Commands: []*cli.Command{
			{
				Name:      "create",
				Usage:     "Generate Goose migration from schema diff (100% offline)",
				ArgsUsage: "<name>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "allow-destructive",
						Usage: "Allow destructive operations (DROP TABLE, DROP COLUMN, etc.)",
					},
				},
				Action: migrationCreateAction,
			},
			// ... other subcommands
		},
	}
}
```

### 4. Enhanced Error Messages

```go
func loadPreviousSchemaFromMigrations(cfg *config.Config) (*schema.Schema, error) {
	migrationDir := cfg.Migration.Directory

	if migrationDir == "" {
		return nil, fmt.Errorf("migration directory not configured in .dryft.yaml")
	}

	s, err := migration.LoadSchemaFromMigrations(migrationDir)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to load schema from migrations in %s: %w\n\n"+
				"Fix the migration file syntax error and try again.",
			migrationDir, err,
		)
	}

	if s == nil {
		// No migrations found - first migration scenario
		fmt.Println("No migrations found, treating as empty schema (first migration)")
		return &schema.Schema{
			Tables: []schema.Table{},
			Enums:  []schema.Enum{},
		}, nil
	}

	count := countMigrationFiles(migrationDir)
	fmt.Printf("✓ Loaded schema from %d migration(s)\n", count)
	return s, nil
}
```

### 5. Remove introspectDatabase() from migration create

`introspectDatabase()` function (line 183-221) sẽ **bị xóa** hoặc chỉ còn dùng cho `db pull` command.

`migration create` command **không cần** DB connection.

## User Experience Changes

### Before (Current)
```bash
$ dryft migration create add_bio
warning: failed to connect to database, treating as empty schema: connection refused
✓ Created migration: 20240916000000_add_bio.sql

Operations:
  - CREATE TABLE users (id SERIAL, email TEXT, bio TEXT)  # Wrong! Should be ALTER
```

### After (100% Offline, First Migration)
```bash
$ dryft migration create initial
No migrations found, treating as empty schema (first migration)
✓ Created migration: 20240916000000_initial.sql

Operations:
  - CREATE TABLE users (id SERIAL, email TEXT)
```

### After (100% Offline, Incremental Migration)
```bash
$ dryft migration create add_bio
✓ Loaded schema from 2 migration(s)
✓ Created migration: 20240916000000_add_bio.sql

Operations:
  - ALTER TABLE users ADD COLUMN bio TEXT  # Correct!
```

### After (Parse Error)
```bash
$ dryft migration create add_bio
Error: failed to load previous schema state: failed to load schema from migrations in ./migrations: parse error at position 45: unexpected token 'CONSTRAINT'
SQL: ALTER TABLE users ADD CONSTRAINT ...

Fix the migration file syntax error and try again.
```

## Testing

### Integration Tests

```go
// internal/cli/migration_integration_test.go

func TestMigrationCreate_WithMigrationHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup test environment
	tmpDir := t.TempDir()
	
	// Create existing migrations
	migrationsDir := filepath.Join(tmpDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)
	
	os.WriteFile(filepath.Join(migrationsDir, "20240101000000_create_users.sql"), []byte(`
-- +goose Up
CREATE TABLE users (id SERIAL PRIMARY KEY, email TEXT NOT NULL);

-- +goose Down
DROP TABLE users;
`), 0644)

	// Create schema.prisma with new column
	schemaFile := filepath.Join(tmpDir, "schema.prisma")
	os.WriteFile(schemaFile, []byte(`
model User {
  id    Int    @id @default(autoincrement())
  email String
  bio   String?
}
`), 0644)

	// Create config (NO DATABASE_URL)
	configFile := filepath.Join(tmpDir, ".dryft.yaml")
	os.WriteFile(configFile, []byte(fmt.Sprintf(`
database:
  provider: postgresql
  url: ""

schema:
  file: %s

migration:
  directory: %s
  format: goose
`, schemaFile, migrationsDir)), 0644)

	// Run command (100% offline)
	os.Chdir(tmpDir)
	
	app := createTestApp()
	err := app.Run(context.Background(), []string{"dryft", "migration", "create", "add_bio"})
	require.NoError(t, err)

	// Verify migration created
	files, _ := os.ReadDir(migrationsDir)
	assert.Len(t, files, 2) // Initial + new

	// Verify migration content contains ALTER, not CREATE
	newMigration := files[1]
	content, _ := os.ReadFile(filepath.Join(migrationsDir, newMigration.Name()))
	assert.Contains(t, string(content), "ALTER TABLE users ADD COLUMN bio")
	assert.NotContains(t, string(content), "CREATE TABLE users")
}

func TestMigrationCreate_EmptyMigrations_FirstMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup with empty migrations directory
	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	schemaFile := filepath.Join(tmpDir, "schema.prisma")
	os.WriteFile(schemaFile, []byte(`
model User {
  id    Int    @id @default(autoincrement())
  email String
}
`), 0644)

	configFile := filepath.Join(tmpDir, ".dryft.yaml")
	os.WriteFile(configFile, []byte(fmt.Sprintf(`
database:
  provider: postgresql
  url: ""

schema:
  file: %s

migration:
  directory: %s
  format: goose
`, schemaFile, migrationsDir)), 0644)

	os.Chdir(tmpDir)
	
	app := createTestApp()
	err := app.Run(context.Background(), []string{"dryft", "migration", "create", "initial"})
	require.NoError(t, err)

	// Verify migration created with CREATE TABLE
	files, _ := os.ReadDir(migrationsDir)
	assert.Len(t, files, 1)

	content, _ := os.ReadFile(filepath.Join(migrationsDir, files[0].Name()))
	assert.Contains(t, string(content), "CREATE TABLE users")
}

func TestMigrationCreate_MalformedMigration(t *testing.T) {
	// Test error handling when migration file has parse errors
	// Should fail with descriptive message
}

func TestMigrationCreate_NoDBConnection(t *testing.T) {
	// Verify migration create works 100% offline
	// No DB connection attempted
}
```

### Manual Testing Checklist

- [ ] Create migration với existing migrations → incremental ALTER
- [ ] Create migration với empty migrations/ → CREATE TABLE (first migration)
- [ ] Create migration với parse error → descriptive error
- [ ] Create migration **without DB connection** → works 100% offline
- [ ] Verify `db pull` still uses DB introspection (unchanged)

## Documentation Updates

### README.md

Add section explaining 100% offline migration generation:

```markdown
## How It Works

dryft generates incremental migrations by comparing your Prisma schema against **migration history** (100% offline):

1. **Parse existing migrations** in `migrations/` directory
2. Build virtual schema from migration history
3. Compare with current `schema.prisma`
4. Generate incremental SQL (ALTER instead of CREATE)

### Offline Migration Generation

dryft migration generation is **100% offline** and does NOT require database connection:

```bash
# First migration (empty migrations/ directory)
dryft migration create initial
# → Creates full schema (CREATE TABLE)

# Incremental migrations (existing migrations/ files)
dryft migration create add_user_bio
# → Creates incremental changes (ALTER TABLE)
```

### When to Use Database Connection

Database connection is ONLY needed for `db pull`:

```bash
# Introspect existing database → generate schema.prisma
dryft db pull
```

After `db pull`, all future migrations are generated offline from migration history.
```

### CLAUDE.md

Update command documentation:

```markdown
## Commands

```bash
dryft db pull                      # Introspect PostgreSQL → generate schema.prisma (requires DB)
dryft migration create <name>      # Generate migration from schema diff (100% offline)
  --allow-destructive              # Allow DROP operations
dryft schema diff                  # Preview changes between schemas
dryft validate                     # Validate Prisma schema
```

**Migration Generation (100% Offline):**
- Uses migration history from `migrations/` directory
- Empty migrations → first migration (CREATE TABLE all)
- Existing migrations → incremental migration (ALTER TABLE)
- No database connection required
```

## Files to Modify

```
internal/cli/migration.go       # Replace introspectDatabase with loadPreviousSchemaFromMigrations
                                # Remove DB connection logic from migration create
internal/cli/migration_test.go  # Add integration tests for offline workflow
README.md                       # Add "How It Works" section (100% offline)
CLAUDE.md                       # Update command docs
```

## Validation

**Done When:**
- [x] `loadPreviousSchemaFromMigrations()` implemented (no DB fallback)
- [x] DB connection logic removed from `migration create`
- [x] Empty migrations/ returns empty schema (first migration scenario)
- [x] Error messages actionable
- [x] Integration tests pass (100% offline tests)
- [x] Manual testing checklist complete
- [x] Documentation updated (README, CLAUDE.md)
- [x] Existing tests still pass (regression check)
- [x] Verify `db pull` still works (unchanged)

## Rollback Plan

Changes isolated to `migration.go`. Can revert by:
1. Restore `introspectDatabase()` call
2. Remove `loadPreviousSchemaFromMigrations()` function

No breaking changes to config, APIs, or `db pull` command.

## Notes

- `migration create` là **100% offline** - không cần DB connection
- `db pull` vẫn dùng DB introspection (unchanged)
- Empty migrations/ = first migration = empty schema
- Migration parsing error → fail hard với clear message
- No fallback to DB - migration history is source of truth
