# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**dryft** is a Go CLI tool for PostgreSQL schema management via Prisma schemas and Goose migrations. It bridges PostgreSQL databases, human-readable Prisma schemas, and versioned SQL migrations.

Core workflow: `PostgreSQL → db pull → schema.prisma → developer edits → schema diff → Goose migration`

dryft generates migrations but does NOT execute them — Goose remains responsible for migration execution and state tracking.

## Commands

```bash
# Development
make dev              # fmt + vet + test-short + build (fastest workflow)
make build            # Build to ./bin/dryft
make install          # Install to $GOPATH/bin

# Testing
make test-short       # Skip integration tests (fast, use during development)
make test             # All tests including integration (uses testcontainers)
make test-integration # Integration tests only

# Quality
make fmt              # gofmt
make vet              # go vet
make lint             # golangci-lint (if installed)

# CLI usage
dryft init                     # Initialize config and state directory
dryft db pull                  # Introspect PostgreSQL → generate schema.prisma
dryft migration create <name>  # Generate Goose migration from schema diff
dryft schema diff              # Preview changes between schemas
dryft validate                 # Validate Prisma schema
```

**Integration tests require Docker** — testcontainers spins up PostgreSQL. Use `make test-short` during development to skip them.

## Architecture

### Package Structure

```
internal/
├── cli/          CLI commands and app entry (urfave/cli/v3)
├── config/       .dryft.yaml parsing and validation
├── introspect/   PostgreSQL introspection (pgx/v5)
│   └── postgres/ PostgreSQL-specific introspector implementation
├── prisma/       Prisma schema writer (Internal Schema → .prisma)
├── schema/       Internal schema model (normalized representation)
├── diff/         Schema diff engine (not yet implemented)
├── migration/    Goose migration generation (not yet implemented)
├── sql/          SQL generation utilities (not yet implemented)
└── testutil/     Test helpers and testcontainers setup
```

### Core Data Flow

1. **Introspection** (`introspect` package):
   - Queries PostgreSQL system catalogs (`pg_catalog`)
   - Extracts tables, columns, constraints, indexes, enums
   - Produces `*schema.Schema` (internal normalized model)

2. **Internal Schema** (`schema` package):
   - Central normalized representation
   - Used by all transformations (introspection → schema, schema → Prisma, schema → diff)
   - Key types: `Schema`, `Table`, `Column`, `Constraint`, `Index`, `Enum`
   - Type system: `TypeKind` enum (UUID, Text, Int32, Timestamp, etc.)

3. **Prisma Writer** (`prisma` package):
   - Transforms `*schema.Schema` → Prisma Schema Language
   - Handles naming conventions (camelCase/PascalCase transforms)
   - Generates relation fields (forward and back-references)
   - Reserved keyword validation (Prisma model/field name collisions)
   - Output: formatted `.prisma` file

4. **Configuration** (`config` package):
   - Loads `.dryft.yaml` with env var expansion (`${DATABASE_URL}`)
   - Schema file path, naming conventions, migration directory

### Key Design Patterns

**Naming Convention Transform**: Database snake_case → Prisma camelCase/PascalCase happens in `prisma.NamingConvention`. Two modes:
- `PascalCase`: table names (e.g., `user_profiles` → `UserProfile`)
- `camelCase`: field names (e.g., `created_at` → `createdAt`)

**Relation Field Generation**: `prisma.Writer.buildRelationPlan()` analyzes all foreign keys across the schema to emit both forward references (`Post.author User`) and back-references (`User.posts Post[]`).

**Reserved Keyword Validation**: `prisma.IsReservedModelName()` and `prisma.IsReservedFieldName()` prevent collisions with Prisma keywords like `model`, `enum`, `type`.

**Test Isolation**: Integration tests use `testutil.SetupPostgresContainer()` for isolated PostgreSQL instances per test. Always use `testing.Short()` gate: `if testing.Short() { t.Skip() }`.

## Testing

- Unit tests: `*_test.go` files alongside source
- Integration tests: `*_integration_test.go` or `*_regression_test.go`
- Test helpers: `internal/testutil/` (container setup, schema fixtures)
- Coverage target: Run `make test` to see coverage report

**Writing integration tests**:
```go
func TestSomething_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    container := testutil.SetupPostgresContainer(t)
    defer container.Terminate(context.Background())
    
    // ... test logic
}
```

## Current Implementation Status

**Phase 1-4 Complete** (as of 2026-09-16):
- ✅ CLI framework and config loading
- ✅ Internal schema model
- ✅ PostgreSQL introspection (tables, columns, constraints, indexes, enums)
- ✅ Prisma schema writer with relation field generation

**Not Yet Implemented**:
- Schema diff engine (`diff` package)
- Migration generation (`migration` package)
- SQL DDL generation (`sql` package)

## Configuration

`.dryft.yaml` schema:
```yaml
database:
  provider: postgresql
  url: ${DATABASE_URL}  # Env var expansion supported

schema:
  file: prisma/schema.prisma
  naming:
    fields: camelCase    # or snake_case
    tables: PascalCase   # or snake_case

migration:
  directory: migrations
  format: goose
  naming: timestamp

goose:
  version_table: goose_db_version
```

## Code Conventions

- **Error handling**: Wrap with context (`fmt.Errorf("failed to X: %w", err)`)
- **Naming**: Go standard (snake_case for files, PascalCase for exports)
- **Testing**: Table-driven tests with `t.Run()` subtests
- **Package imports**: Standard library → external → internal (grouped by blank lines)
- **Comments**: Godoc format for exported symbols

## Dependencies

- **CLI**: `github.com/urfave/cli/v3`
- **Database**: `github.com/jackc/pgx/v5` (PostgreSQL driver)
- **Testing**: `github.com/testcontainers/testcontainers-go`
- **Config**: `gopkg.in/yaml.v3`

No external Prisma CLI or migration tools required at runtime.
