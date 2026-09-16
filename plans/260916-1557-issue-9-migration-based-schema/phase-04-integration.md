# Phase 4: CLI Integration

**Status:** Not Started  
**Dependencies:** Phase 1 (Parser), Phase 2 (Loader), Phase 3 (Builder)  
**Estimated Effort:** 1 day  
**Risk Level:** Medium (workflow changes)

## Objective

Integrate migration-based schema detection vào `dryft migration create` command. Replace DB introspection với migration loader khi có migration files.

## Scope

### In Scope
- Replace `introspectDatabase()` call với migration-based loader
- Fallback to DB introspection khi migrations directory empty
- Add `--from-db` flag để force DB introspection
- Preserve error handling và messaging
- Update help text

### Out of Scope
- Changes to diff/plan/SQL generation logic
- Migration execution
- Schema validation beyond parsing

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

// loadPreviousSchema loads the previous schema state from migrations or database.
// Priority:
//   1. Migration-based schema (if migrations exist and --from-db not set)
//   2. Database introspection (fallback or when --from-db is set)
func loadPreviousSchema(ctx context.Context, cfg *config.Config, forceDB bool) (*schema.Schema, error) {
	migrationDir := cfg.Migration.Directory

	// Force DB introspection if flag set
	if forceDB {
		fmt.Println("Using database introspection (--from-db)")
		return introspectDatabase(ctx, cfg)
	}

	// Try migration-based schema first
	if migrationDir != "" {
		s, err := migration.LoadSchemaFromMigrations(migrationDir)
		if err != nil {
			// Migration parsing failed - this is an error, not a fallback case
			return nil, fmt.Errorf("failed to load schema from migrations: %w", err)
		}

		if s != nil {
			// Successfully loaded from migrations
			fmt.Printf("Loaded previous schema from %d migration files\n", len(getMigrationCount(migrationDir)))
			return s, nil
		}

		// s == nil means no migrations found - fallback to DB
		fmt.Println("No migrations found, using database introspection")
	}

	// Fallback: DB introspection
	return introspectDatabase(ctx, cfg)
}

// getMigrationCount returns count of .sql files in directory (for logging)
func getMigrationCount(dir string) int {
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

	// 5. Load previous schema state from migrations or database
	ctx := context.Background()
	forceDB := cmd.Bool("from-db")
	previousSchema, err := loadPreviousSchema(ctx, cfg, forceDB)
	if err != nil {
		return fmt.Errorf("failed to load previous schema state: %w", err)
	}

	// ... (steps 6-12 unchanged)
}
```

### 3. Add CLI Flag

```go
func MigrationCommand() *cli.Command {
	return &cli.Command{
		Name:  "migration",
		Usage: "Migration operations",
		Commands: []*cli.Command{
			{
				Name:      "create",
				Usage:     "Generate Goose migration from schema diff",
				ArgsUsage: "<name>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "allow-destructive",
						Usage: "Allow destructive operations (DROP TABLE, DROP COLUMN, etc.)",
					},
					&cli.BoolFlag{
						Name:  "from-db",
						Usage: "Force database introspection instead of using migration history",
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
func loadPreviousSchema(ctx context.Context, cfg *config.Config, forceDB bool) (*schema.Schema, error) {
	migrationDir := cfg.Migration.Directory

	if forceDB {
		fmt.Println("Using database introspection (--from-db)")
		return introspectDatabase(ctx, cfg)
	}

	if migrationDir != "" {
		s, err := migration.LoadSchemaFromMigrations(migrationDir)
		if err != nil {
			// Provide actionable error message
			return nil, fmt.Errorf(
				"failed to load schema from migrations in %s: %w\n\n"+
					"Possible solutions:\n"+
					"  1. Fix the migration file syntax error\n"+
					"  2. Use --from-db to skip migration parsing and use database instead\n"+
					"  3. Remove invalid migration files",
				migrationDir, err,
			)
		}

		if s != nil {
			count := getMigrationCount(migrationDir)
			if count > 0 {
				fmt.Printf("✓ Loaded schema from %d migration(s)\n", count)
			}
			return s, nil
		}

		fmt.Println("No migrations found, using database introspection")
	}

	return introspectDatabase(ctx, cfg)
}
```

### 5. Keep introspectDatabase() Unchanged

Preserve existing `introspectDatabase()` function (line 183-221) - no changes needed. It remains as fallback path.

## User Experience Changes

### Before (Current)
```bash
$ dryft migration create add_bio
warning: failed to connect to database, treating as empty schema: connection refused
✓ Created migration: 20240916000000_add_bio.sql

Operations:
  - CREATE TABLE users (id SERIAL, email TEXT, bio TEXT)  # Wrong! Should be ALTER
```

### After (With Migrations)
```bash
$ dryft migration create add_bio
✓ Loaded schema from 2 migration(s)
✓ Created migration: 20240916000000_add_bio.sql

Operations:
  - ALTER TABLE users ADD COLUMN bio TEXT  # Correct!
```

### After (Force DB)
```bash
$ dryft migration create add_bio --from-db
Using database introspection (--from-db)
✓ Created migration: 20240916000000_add_bio.sql

Operations:
  - ALTER TABLE users ADD COLUMN bio TEXT
```

### After (Parse Error)
```bash
$ dryft migration create add_bio
Error: failed to load previous schema state: failed to load schema from migrations in ./migrations: parse error at position 45: unexpected token 'CONSTRAINT'
SQL: ALTER TABLE users ADD CONSTRAINT ...

Possible solutions:
  1. Fix the migration file syntax error
  2. Use --from-db to skip migration parsing and use database instead
  3. Remove invalid migration files
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

	// Create config
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

	// Run command
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

func TestMigrationCreate_ForceDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	container := testutil.SetupPostgresContainer(t)
	defer container.Terminate(context.Background())

	// Setup with both migrations and database
	// ... setup code ...

	// Run with --from-db flag
	app := createTestApp()
	err := app.Run(context.Background(), []string{
		"dryft", "migration", "create", "add_bio", "--from-db",
	})
	require.NoError(t, err)

	// Verify it used DB, not migrations (check output/logs)
}

func TestMigrationCreate_EmptyMigrations(t *testing.T) {
	// Test fallback behavior when migrations/ is empty
	// Should use DB introspection automatically
}

func TestMigrationCreate_MalformedMigration(t *testing.T) {
	// Test error handling when migration file has parse errors
	// Should fail with actionable message
}
```

### Manual Testing Checklist

- [ ] Create migration với existing migrations → incremental ALTER
- [ ] Create migration với empty migrations/ → CREATE TABLE
- [ ] Create migration với `--from-db` flag → uses DB
- [ ] Create migration với parse error → descriptive error
- [ ] Create migration với no DB connection, có migrations → works offline
- [ ] Create migration với no DB connection, no migrations → falls back gracefully

## Documentation Updates

### README.md

Add section explaining migration-based schema:

```markdown
## How It Works

dryft generates incremental migrations by comparing your Prisma schema against the **migration history**:

1. **Parse existing migrations** in `migrations/` directory
2. Build virtual schema from migration history
3. Compare with current `schema.prisma`
4. Generate incremental SQL (ALTER instead of CREATE)

### Offline Migration Generation

Because dryft uses migration history, you can generate migrations **without a database connection**:

```bash
# Works offline if you have migration files
dryft migration create add_user_bio
```

### Force Database Introspection

To use database introspection instead of migration history:

```bash
dryft migration create add_user_bio --from-db
```

This is useful when:
- Migration history is incomplete or corrupted
- You want to sync with actual database state
- Debugging migration parsing issues
```

### CLAUDE.md

Update command documentation:

```markdown
## Commands

```bash
dryft migration create <name>  # Generate migration from schema diff
  --allow-destructive         # Allow DROP operations
  --from-db                   # Force database introspection (skip migration history)
```

**Migration Detection:**
- Default: Uses migration history from `migrations/` directory
- Fallback: Database introspection if no migrations exist
- Force DB: Use `--from-db` flag to skip migration parsing
```

## Files to Modify

```
internal/cli/migration.go       # Add loadPreviousSchema(), update flags
internal/cli/migration_test.go  # Add integration tests
README.md                       # Add "How It Works" section
CLAUDE.md                       # Update command docs
```

## Validation

**Done When:**
- [ ] `loadPreviousSchema()` implemented with migration priority
- [ ] `--from-db` flag added and functional
- [ ] Error messages actionable
- [ ] Integration tests pass
- [ ] Manual testing checklist complete
- [ ] Documentation updated (README, CLAUDE.md)
- [ ] Existing tests still pass (regression check)

## Rollback Plan

Changes isolated to `migration.go`. Can revert by:
1. Remove `loadPreviousSchema()` function
2. Restore direct `introspectDatabase()` call
3. Remove `--from-db` flag

No breaking changes to config or APIs.

## Notes

- Preserve backward compatibility - existing workflows unaffected
- Migration-based approach is **additive**, not replacement
- DB introspection remains supported and available
- Error messages guide users to workarounds (`--from-db`)
