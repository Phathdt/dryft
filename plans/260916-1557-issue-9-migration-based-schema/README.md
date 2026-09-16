# Issue #9 Implementation Plan

**Migration-Based Schema Detection Instead of DB Introspection**

## Quick Reference

- **GitHub Issue:** https://github.com/Phathdt/dryft/issues/9
- **Status:** Planning Complete
- **Estimated Total Effort:** 6-7 days
- **Risk Level:** High (architectural change)

## Problem

`dryft migration create` dùng DB introspection → khi connection fail → fallback empty schema → tạo full CREATE TABLE thay vì incremental ALTER.

**New Requirement:** `migration create` phải 100% offline, không cần DB connection, chỉ dùng migration history.

## Solution

Parse existing migration files để build virtual schema làm baseline (100% offline, no DB connection needed).

**Key Points:**
- `migration create`: 100% offline, no DB fallback
- Empty migrations/ → empty schema → first migration (CREATE TABLE)
- Existing migrations → incremental migration (ALTER TABLE)
- `db pull`: Still uses DB introspection (unchanged)

## Plan Structure

```
plans/260916-1557-issue-9-migration-based-schema/
├── README.md              # This file - quick reference
├── plan.md                # Overview, phases, risks, decisions
├── phase-01-parser.md     # SQL statement parser (2-3 days, HIGH risk)
├── phase-02-loader.md     # Migration file loader (1 day, LOW risk)
├── phase-03-builder.md    # Schema builder (2 days, HIGH risk)
└── phase-04-integration.md # CLI integration (1 day, MEDIUM risk)
```

## Implementation Order

1. **Phase 1: Parser** → Foundation, highest complexity
2. **Phase 2: Loader** → File I/O và orchestration
3. **Phase 3: Builder** → Stateful schema building
4. **Phase 4: Integration** → Wire everything together

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Naming conventions | Parse as-is, no transforms | Preserve migration history integrity |
| Invalid migrations | Fail hard | Schema correctness critical |
| Goose sections | Only parse Up | Down không cần cho current state |
| Parser approach | Token-based, not formal lexer | Simpler, sufficient for DDL subset |
| DB connection | No fallback, 100% offline | Migration history is source of truth |
| Empty migrations | Return empty schema | First migration scenario |

## Success Criteria

- [ ] Parse CREATE/ALTER/DROP TABLE, TYPE, INDEX
- [ ] Load migrations chronologically
- [ ] Build incremental schema correctly
- [ ] `migration create` works **100% offline** without DB
- [ ] Empty migrations/ → empty schema → first migration
- [ ] All tests pass (unit + integration)
- [ ] Documentation updated
- [ ] `db pull` unchanged (still uses DB)

## New Files Created

```
internal/migration/
├── parser.go           # SQL parser
├── parser_test.go
├── loader.go           # Migration loader
├── loader_test.go
├── builder.go          # Schema builder
├── builder_test.go
├── type_mapping.go     # SQL → schema.TypeKind
└── testdata/
    └── migrations/     # Test fixtures
```

## Modified Files

```
internal/cli/migration.go       # Replace introspectDatabase with loadPreviousSchemaFromMigrations
                                # Remove DB connection logic, no fallback
internal/cli/migration_test.go  # Integration tests for offline workflow
README.md                       # Add "How It Works" section (100% offline)
CLAUDE.md                       # Update command docs
```

## Testing Strategy

- **Unit:** Parser (AST), Loader (file I/O), Builder (state)
- **Integration:** End-to-end migration sequence
- **Regression:** Existing tests must pass
- **Manual:** Offline workflow, error handling, fallbacks

## Rollback Plan

All changes isolated in `internal/migration/` package. Can revert Phase 4 integration to restore old behavior without touching new code.

## Next Steps

1. Review plan với maintainer
2. Create feature branch: `feature/issue-9-migration-based-schema`
3. Start Phase 1 implementation
4. Iterate with tests
5. Final review và PR

---

**Total Effort:** 6-7 days  
**Phases:** 4  
**New Packages:** 1 (migration)  
**Modified Packages:** 1 (cli)
