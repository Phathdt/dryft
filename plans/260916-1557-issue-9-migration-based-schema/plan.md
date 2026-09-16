# Issue #9: Migration-Based Schema Detection

**Status:** Planning  
**Issue:** https://github.com/Phathdt/dryft/issues/9  
**Created:** 2026-09-16  
**Type:** Enhancement - Architecture Change

## Problem Statement

Hiện tại `dryft migration create` dùng DB introspection để detect schema drift:
- Kết nối database → introspect schema → so sánh với Prisma schema → generate migration
- **Vấn đề:** Khi DB connection fail → fallback về empty schema → tạo full CREATE TABLE thay vì incremental ALTER
- Migration history không được dùng làm source of truth

## Solution Overview

Parse existing migration files để build virtual schema, dùng làm baseline (100% offline):

```
Current:  DB introspection → schema → diff → migration (requires DB)
New:      migrations/ → parse SQL → virtual schema → diff → migration (offline)
```

**Key Changes:**
- `migration create`: 100% offline, no DB connection needed
- Empty migrations/ → empty schema (first migration scenario)
- No fallback to DB introspection
- `db pull`: Still uses DB introspection (unchanged)

## Architecture Components

```
internal/migration/
├── parser.go       # SQL statement parser
├── loader.go       # Migration file loader & orchestrator  
├── builder.go      # Schema builder from parsed statements
├── goose.go        # Existing Goose formatter (unchanged)
└── *_test.go       # Tests
```

## Implementation Phases

### Phase 1: SQL Parser Foundation
**File:** `internal/migration/parser.go`

Parse CREATE/ALTER/DROP statements cho tables, enums, indexes:
- CREATE TABLE với columns, constraints, types
- ALTER TABLE ADD/DROP/RENAME COLUMN, ALTER COLUMN
- DROP TABLE
- CREATE TYPE (enum), ALTER TYPE ADD VALUE, DROP TYPE
- CREATE INDEX (BTree, GIN, GiST, Hash, partial), DROP INDEX

**Dependencies:** None  
**Risk:** SQL syntax complexity - cần handle quoted identifiers, case sensitivity

### Phase 2: Migration Loader
**File:** `internal/migration/loader.go`

Load và sort migration files chronologically:
- Read .sql files từ migrations directory
- Parse filename format: `YYYYMMDDHHMMSS_name.sql`
- Sort by timestamp
- Extract statements (split by `;`, handle Goose directives)
- Return ordered list of statements

**Dependencies:** Phase 1  
**Risk:** File I/O errors, malformed filenames

### Phase 3: Schema Builder
**File:** `internal/migration/builder.go`

Build `schema.Schema` từ SQL statements:
- Apply statements chronologically
- Maintain state (tables, columns, types, indexes)
- Handle incremental operations (ALTER adds/modifies state)
- Handle dependencies (CREATE TYPE before CREATE TABLE)

**Dependencies:** Phase 1, 2  
**Risk:** Statement order dependencies, circular references

### Phase 4: CLI Integration
**Files:** `internal/cli/migration.go`

Replace `introspectDatabase()` logic:
- Try migration-based schema first (via loader + builder)
- Empty migrations/ → empty schema (first migration)
- **No DB fallback** - 100% offline
- Remove DB connection từ `migration create`
- Preserve `db pull` command (still uses DB)

**Dependencies:** Phase 1, 2, 3  
**Risk:** Breaking existing workflows (medium)

## Acceptance Criteria

- [ ] Parse CREATE/ALTER/DROP TABLE statements
- [ ] Parse CREATE/ALTER/DROP TYPE (enum) statements  
- [ ] Parse CREATE/DROP INDEX statements
- [ ] Load migrations chronologically và build schema
- [ ] `migration create` works **100% offline** without DB connection
- [ ] Empty migrations/ → empty schema (first migration)
- [ ] Incremental migrations work correctly
- [ ] Tests: parser, loader, builder, integration
- [ ] Documentation updated (README.md, CLAUDE.md)
- [ ] `db pull` still works (unchanged)

## Testing Strategy

**Unit Tests:**
- Parser: valid/invalid SQL statements, edge cases (quotes, keywords, types)
- Loader: filename parsing, file sorting, statement extraction
- Builder: incremental state building, dependency handling

**Integration Tests:**
- End-to-end: migrations/ → schema → diff → migration generation
- Fallback behavior: empty migrations/ → DB introspection
- `--from-db` flag behavior

**Regression Tests:**
- Existing test suite must pass
- DB introspection path still works

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| SQL parser incomplete | High | Start with common DDL subset, expand iteratively |
| Statement order bugs | High | Comprehensive unit tests for builder state |
| Breaking existing workflows | Medium | Clear error messages, empty migrations = first migration |
| Performance with many migrations | Low | Lazy loading, caching if needed |

## Open Questions

1. **Naming conventions:** Migration history dùng snake_case hay respect config?
   - **Decision:** Parse as-is từ SQL, không apply naming convention transforms
   
2. **Invalid migration files:** Skip hay fail hard?
   - **Decision:** Fail hard với descriptive error - integrity critical

3. **Partial migrations:** Handle Goose Up/Down sections riêng biệt?
   - **Decision:** Only parse Up section để build current state

## Related Work

- Config naming convention (snake_case vs camelCase) - out of scope
- Quoted/unquoted identifier handling - in scope Phase 1

## Next Steps

1. Review plan với maintainer
2. Create branch `feature/issue-9-migration-based-schema`
3. Implement Phase 1 (parser foundation)
4. Iterate through phases with tests
5. Update documentation
6. Create PR

---
**Plan Phase Files:**
- [Phase 1: SQL Parser](./phase-01-parser.md)
- [Phase 2: Migration Loader](./phase-02-loader.md)
- [Phase 3: Schema Builder](./phase-03-builder.md)
- [Phase 4: CLI Integration](./phase-04-integration.md)
