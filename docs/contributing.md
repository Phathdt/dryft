# Contributing to dryft

Thank you for your interest in contributing to dryft!

## Development Setup

### Prerequisites

- Go 1.23 or later
- Docker (for integration tests)
- golangci-lint (for linting)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/phathdt/dryft.git
cd dryft

# Install dependencies
go mod download

# Build the project
make build

# Run tests
make test

# Run development checks (fast)
make dev
```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/bug-description
```

### 2. Make Changes

Write your code following our coding standards:

- Follow Go conventions and idioms
- Write descriptive variable and function names
- Add comments for complex logic
- Keep functions focused and small

### 3. Format and Lint

Before committing, ensure your code is properly formatted and passes lint checks:

```bash
# Format code
make fmt

# Run vet
make vet

# Run linter
make lint
```

### 4. Write Tests

- Add unit tests for new functionality
- Add integration tests for database interactions
- Ensure test coverage remains ≥85%

```bash
# Run tests
make test

# Run only unit tests (fast)
make test-short

# Check coverage
go test -cover ./...
```

### 5. Commit Changes

Use conventional commit messages:

```bash
# Format: <type>(<scope>): <description>

git commit -m "feat(cli): add db push command"
git commit -m "fix(introspect): handle null values in CHECK constraints"
git commit -m "test: improve coverage for diff package"
git commit -m "docs: update README with installation instructions"
```

**Commit types:**
- `feat`: New feature
- `fix`: Bug fix
- `test`: Test additions or modifications
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build process or auxiliary tool changes

### 6. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

## Pull Request Guidelines

### PR Requirements

All PRs must pass these checks before merging:

✅ **Format Check** - Code must be properly formatted (`gofmt`)  
✅ **Lint Check** - No linting errors (`golangci-lint`)  
✅ **Vet Check** - Pass `go vet` analysis  
✅ **Build Check** - Code must compile successfully  
✅ **Unit Tests** - All unit tests must pass  
✅ **Integration Tests** - All integration tests must pass  
✅ **Coverage** - Maintain ≥85% test coverage

### PR Review Process

1. **Automated Checks**: GitHub Actions will automatically run all checks
2. **Code Review**: At least one maintainer review required
3. **Address Feedback**: Make requested changes
4. **Merge**: Once approved and all checks pass, PR will be merged

### PR Best Practices

- Keep PRs focused and small
- Write clear PR descriptions
- Reference related issues
- Respond to review comments promptly
- Squash commits before merging (if requested)

## Coding Standards

### Go Style

Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines:

- Use `gofmt` for formatting
- Use meaningful variable names
- Avoid deep nesting (max 4 levels)
- Keep functions under 50 lines when possible
- Write self-documenting code

### Error Handling

```go
// ✅ Good: wrap errors with context
if err != nil {
    return fmt.Errorf("failed to introspect database: %w", err)
}

// ❌ Bad: lose error context
if err != nil {
    return err
}
```

### Testing Standards

```go
// Use table-driven tests
func TestFunction(t *testing.T) {
    tests := []struct{
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {name: "valid input", input: "test", expected: "TEST"},
        {name: "empty input", input: "", wantErr: true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    container := testutil.SetupPostgresContainer(t)
    defer container.Terminate(context.Background())
    
    // Test logic...
}
```

## Project Structure

```
dryft/
├── cmd/dryft/           # CLI entry point
├── internal/
│   ├── cli/            # CLI commands
│   ├── config/         # Configuration handling
│   ├── diff/           # Schema diff engine
│   ├── introspect/     # Database introspection
│   ├── migration/      # Migration generation
│   ├── prisma/         # Prisma schema parsing/writing
│   ├── schema/         # Internal schema model
│   ├── sql/            # SQL generation
│   └── testutil/       # Test utilities
├── docs/               # Documentation
├── plans/              # Implementation plans
└── tests/              # Integration tests
```

## Branch Protection

The `main` branch is protected with these rules:

- Require PR before merging
- Require status checks to pass:
  - format
  - lint
  - vet
  - build
  - test
  - test-integration
- Require up-to-date branches
- No direct pushes to main

## Getting Help

- Read the [README](../README.md)
- Check [CLAUDE.md](../CLAUDE.md) for project context
- Open an issue for questions
- Join discussions in existing issues

## License

By contributing to dryft, you agree that your contributions will be licensed under the same license as the project.
