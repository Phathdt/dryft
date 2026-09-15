# dryft

> **Schema in. Migration out.**

dryft is a Go-based CLI developer tool for managing PostgreSQL schema changes through Prisma schema definitions and generating Goose migrations.

The core purpose is to bridge the gap between:

- an existing/real PostgreSQL database
- a human-readable `schema.prisma`
- versioned Goose SQL migrations

dryft does **not** execute migrations. Goose remains responsible for migration execution and migration state.

---

## 1. Product Vision

dryft should provide a simple and predictable workflow:

```text
                 PostgreSQL
                     │
                  db pull
                     │
                     ▼
              schema.prisma
                     │
                developer
                  changes
                     │
                     ▼
              schema.prisma
                     │
               schema diff
                     │
                     ▼
             Goose migration
                     │
                  goose up
                     │
                     ▼
                 PostgreSQL
```

The long-term goal is to make database schema drift visible and migration generation safe, deterministic, and developer-friendly.

---

# 2. Goals

## P0 — MVP

### 2.1 PostgreSQL introspection

Support generating a Prisma schema from an existing PostgreSQL database.

```bash
 dryft db pull
```

The command should inspect the real database and generate/update:

```text
prisma/
└── schema.prisma
```

Supported metadata:

- tables
- columns
- PostgreSQL data types
- nullable / NOT NULL
- default values
- primary keys
- foreign keys
- indexes
- unique constraints
- PostgreSQL enums

---

## 2.2 Prisma schema parsing

dryft must parse the required subset of Prisma Schema Language without requiring the Prisma CLI or Prisma migration engine.

Example:

```prisma
model User {
  id        String   @id @db.Uuid
  email     String   @unique
  createdAt DateTime @default(now()) @map("created_at")

  @@map("users")
}
```

The parser converts the Prisma schema into an internal normalized schema representation.

---

## 2.3 Schema diff engine

Compare two schema states and produce a list of database operations.

Example:

```diff
model User {
  id       String @id
- name     String
+ username String
}
```

Expected operation:

```text
RENAME COLUMN users.name → users.username
```

The diff engine must be independent from the migration format.

---

## 2.4 Goose migration generation

Generate a Goose-compatible SQL migration.

```bash
 dryft migration create add_username
```

Output:

```text
migrations/
└── 20260915180000_add_username.sql
```

Example:

```sql
-- +goose Up

ALTER TABLE users
RENAME COLUMN name TO username;

-- +goose Down

ALTER TABLE users
RENAME COLUMN username TO name;
```

dryft generates migration files only.

It does not execute them.

The user runs:

```bash
goose up
```

---

## 2.5 Existing database bootstrap

An existing PostgreSQL database must be a first-class use case.

Example:

```bash
 dryft db pull
```

This should produce a schema representation of the existing database without requiring the user to recreate the database from migrations.

The user can then establish the current database state as a baseline:

```bash
 dryft db baseline
```

After that, future changes should generate only deltas from the baseline.

---

## 2.6 Destructive change protection

dryft must detect potentially destructive operations.

Examples:

```text
DROP TABLE
DROP COLUMN
ALTER COLUMN TYPE
DROP CONSTRAINT
DROP INDEX
```

Default behavior:

```text
⚠ Destructive change detected

  DROP COLUMN users.email

This operation may cause data loss.

Migration generation aborted.
```

The user must explicitly opt in:

```bash
 dryft migration create remove_email --allow-destructive
```

---

# 3. Non-Goals

The MVP will NOT:

- execute migrations
- replace Goose
- manage migration locking
- manage migration history
- implement an ORM
- generate Go database code
- support MySQL
- support SQLite
- support MongoDB
- implement Prisma Client
- depend on Prisma Migrate
- execute Prisma CLI commands
- support `go-migrate` output

Go-migrate support is a future feature.

---

# 4. Core Concepts

## 4.1 Desired Schema

The schema represented by:

```text
schema.prisma
```

This represents the developer's intended database structure.

---

## 4.2 Actual Schema

The schema obtained from the real PostgreSQL database through introspection.

```text
PostgreSQL
    │
    ▼
Introspector
    │
    ▼
Internal Schema
```

---

## 4.3 Schema State

dryft may maintain metadata under:

```text
./dryft
```

For example:

```text
./dryft
└── state.json
```

This state is used to preserve information that cannot always be represented completely by Prisma schema syntax.

The state format is an implementation detail and should not be considered a stable public API in the MVP.

---

## 4.4 Migration Operation

The diff engine produces database-independent operations.

Examples:

```text
CreateTable
DropTable
AddColumn
DropColumn
RenameColumn
AlterColumn
CreateIndex
DropIndex
CreateForeignKey
DropForeignKey
CreateEnum
AlterEnum
```

These operations are then translated into PostgreSQL SQL.

---

# 5. Architecture

```text
                     schema.prisma
                          │
                          ▼
                   Prisma Parser
                          │
                          ▼
                   Internal Schema
                          │
                          │
PostgreSQL ──► DB Introspector
                          │
                          ▼
                   Internal Schema
                          │
                          └────────────┐
                                       ▼
                                  Diff Engine
                                       │
                                       ▼
                              Migration Operations
                                       │
                                       ▼
                              PostgreSQL SQL
                                       │
                                       ▼
                                Goose Formatter
                                       │
                                       ▼
                              migration/*.sql
```

The core diff engine must not depend on Goose.

---

# 6. Project Structure

Proposed Go project structure:

```text
/dryft
├── cmd/
│   └── /dryft
│       └── main.go
│
├── internal/
│   ├── cli/
│   │
│   ├── prisma/
│   │   ├── parser.go
│   │   ├── lexer.go
│   │   ├── ast.go
│   │   └── writer.go
│   │
│   ├── schema/
│   │   ├── schema.go
│   │   ├── table.go
│   │   ├── column.go
│   │   ├── index.go
│   │   ├── constraint.go
│   │   └── normalize.go
│   │
│   ├── introspect/
│   │   ├── introspector.go
│   │   └── postgres/
│   │       ├── postgres.go
│   │       ├── tables.go
│   │       ├── columns.go
│   │       ├── indexes.go
│   │       ├── constraints.go
│   │       └── enums.go
│   │
│   ├── diff/
│   │   ├── diff.go
│   │   ├── operation.go
│   │   ├── planner.go
│   │   └── rename.go
│   │
│   ├── sql/
│   │   └── postgres/
│   │       ├── create_table.go
│   │       ├── alter_table.go
│   │       ├── indexes.go
│   │       ├── constraints.go
│   │       └── enums.go
│   │
│   ├── migration/
│   │   ├── migration.go
│   │   └── goose.go
│   │
│   └── state/
│       └── state.go
│
├── prisma/
│   └── schema.prisma
│
├── migrations/
│
├── schemaforge.yaml
├── go.mod
└── README.md
```

Rename `schemaforge.yaml` to:

```text
.dryft yaml
```

---

# 7. Internal Schema

Example:

```go
type Schema struct {
    Tables []Table
    Enums  []Enum
}

type Table struct {
    Name        string
    Columns     []Column
    PrimaryKey  *PrimaryKey
    ForeignKeys []ForeignKey
    Indexes     []Index
    Constraints []Constraint
}

type Column struct {
    Name       string
    Type       DataType
    Nullable   bool
    Default    *DefaultValue
    PrimaryKey bool
}
```

Diff operations:

```go
type Operation interface {
    Kind() OperationKind
}

type CreateTable struct {
    Table Table
}

type DropTable struct {
    Name string
}

type AddColumn struct {
    Table  string
    Column Column
}

type DropColumn struct {
    Table string
    Name  string
}

type RenameColumn struct {
    Table string
    From  string
    To    string
}

type AlterColumn struct {
    Table string
    From  Column
    To    Column
}
```

---

# 8. Database Introspection

## Command

```bash
 dryft db pull
```

Optional:

```bash
 dryft db pull --url "$DATABASE_URL"
```

Default configuration:

```yaml
database:
  provider: postgresql
  url: ${DATABASE_URL}
```

The introspector queries PostgreSQL system catalogs rather than parsing SQL dumps.

Primary catalog sources may include:

```text
pg_class
pg_attribute
pg_type
pg_constraint
pg_index
pg_indexes
pg_namespace
pg_enum
pg_attrdef
```

The exact implementation is internal.

---

# 9. Prisma Generation

Example database:

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    age INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Expected generated schema:

```prisma
model User {
  id        String   @id @db.Uuid
  email     String
  age       Int?
  createdAt DateTime @default(now()) @db.Timestamptz

  @@map("users")
}
```

Naming conversion:

```text
snake_case → camelCase
```

should be configurable.

Example:

```yaml
schema:
  naming:
    fields: camelCase
    tables: PascalCase
```

---

# 10. PostgreSQL Type Mapping

MVP support:

| PostgreSQL       | Prisma                   |
| ---------------- | ------------------------ |
| uuid             | String @db.Uuid          |
| text             | String                   |
| varchar          | String @db.VarChar       |
| char             | String @db.Char          |
| integer          | Int                      |
| bigint           | BigInt                   |
| smallint         | Int                      |
| boolean          | Boolean                  |
| real             | Float @db.Real           |
| double precision | Float                    |
| numeric          | Decimal                  |
| timestamp        | DateTime                 |
| timestamptz      | DateTime @db.Timestamptz |
| date             | DateTime @db.Date        |
| json             | Json                     |
| jsonb            | Json @db.JsonB           |
| bytea            | Bytes                    |

Additional PostgreSQL-specific types can be added later.

---

# 11. Migration Generation

Command:

```bash
 dryft migration create add_reconciliation_events
```

Process:

```text
schema.prisma
      │
      ▼
Current Schema
      │
      ▼
Previous Schema / State
      │
      ▼
Diff
      │
      ▼
Operations
      │
      ▼
PostgreSQL SQL
      │
      ▼
Goose Migration
```

Output:

```text
migrations/
└── 20260915180000_add_reconciliation_events.sql
```

---

# 12. Goose Format

Every migration must follow Goose's SQL migration format.

```sql
-- +goose Up

CREATE TABLE reconciliation_events (
    id UUID PRIMARY KEY
);

-- +goose Down

DROP TABLE reconciliation_events;
```

Multiple statements are supported.

Operations must be ordered according to dependency.

Example:

```text
CREATE TYPE
    ↓
CREATE TABLE
    ↓
ADD COLUMN
    ↓
CREATE FOREIGN KEY
    ↓
CREATE INDEX
```

Drops should use the reverse dependency order.

---

# 13. Migration Reversibility

Every generated migration should attempt to provide a `Down` migration.

Example:

```sql
-- +goose Up

ALTER TABLE users
ADD COLUMN username TEXT;

-- +goose Down

ALTER TABLE users
DROP COLUMN username;
```

If an operation cannot safely be reversed automatically, dryft must mark it clearly.

Example:

```text
⚠ Cannot automatically generate a safe Down migration.

Operation:
  ALTER COLUMN users.balance TYPE bigint

Reason:
  Existing data may not be safely convertible back.

Manual intervention required.
```

---

# 14. Rename Detection

Rename detection is important because:

```diff
- name
+ username
```

could mean either:

```sql
RENAME COLUMN
```

or:

```sql
DROP COLUMN
ADD COLUMN
```

dryft should eventually support explicit rename declarations.

Example:

```bash
 dryft migration create rename_user_name \
  --rename users.name:users.username
```

Automatic rename detection may be introduced after MVP.

The diff engine must support `RenameColumn` independently from the beginning.

---

# 15. Destructive Operations

The following operations are classified as destructive:

```text
DROP TABLE
DROP COLUMN
DROP INDEX
DROP CONSTRAINT
ALTER COLUMN TYPE
ALTER ENUM
```

Potentially destructive:

```text
ADD NOT NULL column
CREATE UNIQUE constraint
SET NOT NULL
```

Default:

```bash
 dryft migration create remove_email
```

should fail when destructive changes are detected.

Explicit override:

```bash
 dryft migration create remove_email \
  --allow-destructive
```

---

# 16. Drift Detection

A major long-term feature is:

```bash
 dryft db diff
```

This compares:

```text
schema.prisma
      │
      ▼
Desired Schema
      │
      │ compare
      ▼
Actual PostgreSQL Schema
```

Example:

```text
Schema drift detected:

  + users.phone_number
  ~ users.email TEXT → VARCHAR(255)
  - users.created_at
```

This allows dryft to detect manual database changes that were not represented in migrations.

---

# 17. Validation

Command:

```bash
 dryft validate
```

Checks:

```text
✓ schema.prisma is valid
✓ PostgreSQL types are supported
✓ migration numbering is valid
✓ migration names are valid
✓ migration operations are consistent
✓ no duplicate migration versions
✓ destructive operations are explicitly acknowledged
```

Future checks:

```text
✓ database matches expected schema
✓ migrations can be reversed
✓ migration history is consistent
```

---

# 18. CLI Specification

## Initialize

```bash
 dryft init
```

Creates:

```text
.dryft yaml
prisma/schema.prisma
migrations/
./dryft
```

---

## Pull database

```bash
 dryft db pull
```

Database → Prisma schema.

---

## Inspect database

```bash
 dryft db inspect
```

Print internal database schema without modifying files.

Example:

```text
Tables:

  users
    id          uuid         PK
    email       text         NOT NULL
    created_at  timestamptz  NOT NULL

  orders
    id          uuid         PK
    user_id     uuid         FK → users.id
```

---

## Baseline

```bash
 dryft db baseline
```

Marks the current database schema as the initial state.

---

## Diff

```bash
 dryft diff
```

Shows schema changes without generating a migration.

---

## Create migration

```bash
 dryft migration create <name>
```

Example:

```bash
 dryft migration create add_bad_debts
```

---

## Migration status

```bash
 dryft migration status
```

Example:

```text
Migrations:

  20260910100000_init                    applied
  20260912120000_add_cashflow            applied
  20260915180000_add_bad_debts           pending
```

Note: actual migration execution/history remains Goose's responsibility.

---

## Validate

```bash
 dryft validate
```

---

# 19. Configuration

Example:

```yaml
database:
  provider: postgresql
  url: ${DATABASE_URL}

schema:
  file: prisma/schema.prisma

migration:
  directory: migrations
  format: goose
  naming: timestamp

goose:
  version_table: goose_db_version
```

Environment variables should be preferred for credentials.

dryft must not require credentials to be stored in `.dryft yaml`.

---

# 20. Determinism

Given:

```text
Schema A
Schema B
Configuration
```

the generated migration must be deterministic.

The same input should produce:

```text
same operations
same SQL
same ordering
```

Operations should be sorted using deterministic rules.

For example:

```text
1. Extensions / types
2. Tables
3. Columns
4. Primary keys
5. Foreign keys
6. Constraints
7. Indexes
```

Within each category, sort by object name unless dependency ordering requires otherwise.

---

# 21. Safety Principles

dryft should follow these principles:

### Never silently lose data

Destructive changes require explicit confirmation/flag.

### Never silently overwrite migrations

Existing migration files must never be modified automatically.

### Never execute migrations

dryft generates migrations.

Goose executes them.

### Never hide schema drift

Differences between the desired schema and real database should be visible.

### Prefer explicitness over magic

When automatic inference is ambiguous, dryft should ask the developer or require explicit configuration.

---

# 22. Testing Strategy

Testing is a critical part of the project.

## Unit tests

Test:

- Prisma parser
- PostgreSQL type mapping
- schema normalization
- diff engine
- operation ordering
- SQL generation
- Goose formatting
- destructive-change detection

---

## Golden tests

Example:

```text
testdata/
├── add_column/
│   ├── before.prisma
│   ├── after.prisma
│   └── expected.sql
│
├── create_table/
│   ├── before.prisma
│   ├── after.prisma
│   └── expected.sql
│
└── rename_column/
    ├── before.prisma
    ├── after.prisma
    └── expected.sql
```

---

## Integration tests

Use a real PostgreSQL instance, preferably Testcontainers.

Test:

```text
PostgreSQL
    ↓
db pull
    ↓
schema.prisma
    ↓
modify schema
    ↓
migration create
    ↓
goose up
    ↓
PostgreSQL
```

The final database schema should match the expected schema.

---

# 23. Round-Trip Requirement

One of the most important acceptance criteria:

```text
DB₀
 │
 │ db pull
 ▼
schema.prisma
 │
 │ modify
 ▼
schema.prisma'
 │
 │ migration create
 ▼
migration.sql
 │
 │ goose up
 ▼
DB₁
```

Then:

```text
Schema(DB₁) == Schema(schema.prisma')
```

for all metadata representable by the supported Prisma/PostgreSQL subset.

This should be covered by integration tests.

---

# 24. MVP Acceptance Criteria

dryft v0.1 is considered successful when it can:

1. Connect to a real PostgreSQL database.
2. Introspect tables, columns, types, indexes, constraints, FKs and enums.
3. Generate a valid `schema.prisma`.
4. Parse the generated Prisma schema again.
5. Detect schema changes.
6. Generate valid Goose migrations.
7. Generate reversible migrations for supported operations.
8. Reject destructive changes by default.
9. Establish an existing database as a baseline.
10. Successfully apply generated migrations using Goose.
11. Pass round-trip integration tests.
12. Produce deterministic migration output.

---

# 25. Roadmap

## v0.1 — Foundation

```text
PostgreSQL
     ↓
db pull
     ↓
Prisma schema
     ↓
diff
     ↓
Goose migration
```

Features:

- PostgreSQL introspection
- Prisma parser
- Internal schema model
- Schema diff
- Goose formatter
- baseline
- destructive-change protection
- integration tests

---

## v0.2 — Production Quality

- rename detection
- advanced PostgreSQL indexes
- partial indexes
- expression indexes
- advanced constraints
- enum alteration
- better default-value handling
- migration validation
- schema drift detection

---

## v0.3 — Advanced Workflow

- `db diff`
- CI mode
- migration planning
- migration linting
- migration conflict detection
- improved schema state management

---

## v0.4 — Additional Migration Formats

Add:

```text
Goose
go-migrate
```

Go-migrate output:

```text
000001_add_users.up.sql
000001_add_users.down.sql
```

The diff engine and PostgreSQL SQL generator remain unchanged.

Only the migration formatter changes.

---

# 26. Future Architecture

Long term:

```text
                     dryft Core
                          │
             ┌────────────┼────────────┐
             │            │            │
             ▼            ▼            ▼
        PostgreSQL      MySQL       SQLite
             │
             ▼
       Internal Schema
             │
             ▼
         Diff Engine
             │
             ▼
      Migration Operations
             │
      ┌──────┼────────┐
      │      │        │
      ▼      ▼        ▼
    Goose  Migrate   Atlas
```

The core should remain database-schema and migration-format agnostic wherever practical.

---

# 27. Product Positioning

**Name:**

```text
dryft
```

**CLI:**

```bash
dryft
```

**Tagline:**

> Schema in. Migration out.

Alternative tagline:

> Detect schema drift before it becomes database drift.

Short description:

> dryft is a Go CLI that introspects PostgreSQL into Prisma schemas and generates safe, versioned Goose migrations from schema changes.

---

# 28. Design Principle

dryft should behave more like a compiler than an ORM.

```text
Prisma Schema
      │
      ▼
     AST
      │
      ▼
Internal Schema
      │
      ▼
Diff / Plan
      │
      ▼
Migration Operations
      │
      ▼
PostgreSQL SQL
      │
      ▼
Goose Migration
```

This separation is the core architectural decision of the project.

The Prisma syntax, PostgreSQL introspection, diff engine, SQL generator and Goose output layer should remain independently testable and replaceable.
