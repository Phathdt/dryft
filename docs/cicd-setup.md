# CI/CD Pipeline Setup

This document describes the continuous integration and deployment pipeline for dryft.

## Overview

The CI/CD pipeline ensures code quality and reliability through automated testing, linting, and build verification on every pull request and push to the main branch.

## Pipeline Components

### 1. PR Validation Workflow

**File**: `.github/workflows/pr-validation.yml`  
**Triggers**: Pull requests and pushes to `main`

The PR validation workflow runs six parallel jobs:

#### Format Check
- Verifies code is formatted with `gofmt`
- Fails if any file needs formatting
- **Local command**: `make fmt`

#### Lint
- Runs `golangci-lint` with configured rules
- Checks for code quality issues, potential bugs, and style violations
- **Version**: v2.12.2
- **Timeout**: 5 minutes
- **Local command**: `make lint` or `golangci-lint run ./...`

#### Go Vet
- Runs Go's built-in static analysis tool
- Catches common programming errors
- **Local command**: `make vet` or `go vet ./...`

#### Build
- Compiles the project to ensure it builds successfully
- Uploads the binary as an artifact (retained for 7 days)
- **Local command**: `make build`

#### Unit Tests
- Runs all unit tests with race detection
- Skips integration tests (using `-short` flag)
- Generates coverage report
- Uploads coverage to Codecov
- **Local command**: `make test-short`

#### Integration Tests
- Runs all tests including integration tests
- Uses PostgreSQL 16 service container
- Requires Docker (uses testcontainers in local environment)
- **Local command**: `make test`

### 2. Main Branch CI Workflow

**File**: `.github/workflows/main.yml`  
**Triggers**: Push to `main` branch

This workflow runs after code is merged to main:

#### Quality Check Job
- Format check
- Go vet
- Lint (golangci-lint v2.12.2)
- Uses Go module cache for faster builds

#### Test & Coverage Job
- Runs all tests (unit + integration) with race detection
- Uses PostgreSQL 16 service container
- Generates coverage report in atomic mode
- **Coverage threshold**: 85% minimum
- Uploads coverage to Codecov
- Fails if coverage drops below threshold

#### Build Job
- Compiles the binary
- Verifies the binary can execute
- Runs after tests pass

## Linter Configuration

**File**: `.golangci.yml`

Enabled linters:
- `errcheck` - Check for unchecked errors
- `govet` - Go vet examines Go source code
- `ineffassign` - Detect ineffectual assignments
- `staticcheck` - Static analysis checks
- `unused` - Check for unused code

Key settings:
- Timeout: 5 minutes
- Tests excluded from linting
- Type assertion checking enabled
- Shadow variable checking enabled

## Coverage Requirements

- **Minimum coverage**: 85%
- **Enforcement**: CI fails if coverage drops below threshold
- **Reporting**: Codecov integration for trend tracking

## Local Development Workflow

Before pushing code or creating a PR, run these commands:

```bash
# 1. Format code
make fmt

# 2. Run static analysis
make vet

# 3. Run linter
make lint

# 4. Run unit tests (fast)
make test-short

# 5. Run all tests including integration
make test

# 6. Build binary
make build

# Or run the complete development workflow:
make dev  # Runs fmt + vet + test-short + build
```

## Branch Protection Rules

To enable full CI/CD protection, configure these branch protection rules for `main`:

### Required Settings

1. **Require pull request before merging**
   - Require approvals: 1
   - Dismiss stale reviews when new commits are pushed

2. **Require status checks to pass**
   - Require branches to be up to date before merging
   - Required status checks:
     - `Format Check`
     - `Lint`
     - `Go Vet`
     - `Build`
     - `Test (Unit)`
     - `Test (Integration)`
     - `All Checks Passed`

3. **Additional protections**
   - Do not allow bypassing the above settings
   - Restrict who can push to matching branches (optional)
   - Require linear history (optional, recommended)

### Configuring on GitHub

1. Go to repository Settings → Branches
2. Click "Add branch protection rule"
3. Branch name pattern: `main`
4. Enable the settings listed above
5. Save changes

## Dependabot Configuration

**File**: `.github/dependabot.yml`

Automated dependency updates:
- **GitHub Actions**: Weekly updates
- **Go modules**: Weekly updates, grouped by type
- Auto-labels PRs with `dependencies`
- Uses conventional commit format

## CI/CD Badges

Add these badges to README.md (already added):

```markdown
[![CI](https://github.com/phathdt/dryft/actions/workflows/pr-validation.yml/badge.svg)](https://github.com/phathdt/dryft/actions/workflows/pr-validation.yml)
[![codecov](https://codecov.io/gh/phathdt/dryft/branch/main/graph/badge.svg)](https://codecov.io/gh/phathdt/dryft)
[![Go Report Card](https://goreportcard.com/badge/github.com/phathdt/dryft)](https://goreportcard.com/report/github.com/phathdt/dryft)
```

## Troubleshooting

### Lint Failures

If linting fails locally but you believe the code is correct:

1. Check golangci-lint version: `golangci-lint --version`
2. Update to v2.12.2 if needed
3. Run with same timeout: `golangci-lint run ./... --timeout=5m`
4. Check `.golangci.yml` for enabled linters

### Test Failures

**Unit tests fail locally**:
```bash
# Run specific test
go test -v -short -run TestName ./path/to/package

# Run with race detection
go test -v -short -race ./...
```

**Integration tests fail locally**:
- Ensure Docker is running (testcontainers requires Docker)
- Check Docker daemon: `docker ps`
- Verify PostgreSQL image can be pulled: `docker pull postgres:16-alpine`

### Coverage Below Threshold

If coverage drops below 85%:

1. Check which packages have low coverage:
   ```bash
   go test -cover ./...
   ```

2. Generate detailed coverage report:
   ```bash
   go test -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out
   ```

3. Add tests for uncovered code paths

## Performance Optimization

The CI pipeline is optimized for speed:

- **Parallel jobs**: Format, lint, vet, build, and test run concurrently
- **Go module caching**: Reduces dependency download time
- **Shared PostgreSQL container**: Integration tests reuse a single container
- **Short test mode**: Unit tests skip integration tests for fast feedback

Typical CI run time: **2-3 minutes**

## Future Enhancements

Planned improvements:

- [ ] Release automation (GitHub Releases)
- [ ] Multi-architecture builds (Linux, macOS, Windows)
- [ ] Security scanning (gosec, trivy)
- [ ] Performance benchmarking
- [ ] Nightly builds with extended test suite
- [ ] Docker image publishing

## Summary

**CI/CD Status**: ✅ Fully configured and operational

**Pipeline Checks**:
- ✅ Code formatting (gofmt)
- ✅ Linting (golangci-lint)
- ✅ Static analysis (go vet)
- ✅ Build verification
- ✅ Unit tests with race detection
- ✅ Integration tests with PostgreSQL
- ✅ 85% coverage requirement

**Next Steps**:
1. Enable branch protection rules on GitHub
2. Configure Codecov token (if private repo)
3. Monitor CI runs and adjust timeout/resources as needed
