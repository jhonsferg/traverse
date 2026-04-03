# Contributing to Traverse

Thank you for interest in contributing to Traverse! We welcome all contributions, from bug reports to feature implementations.

## Code of Conduct

This project adheres to the Contributor Covenant [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- Go 1.19 or higher
- Git
- Basic understanding of OData and HTTP concepts

### Setting Up Development Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/traverse.git
   cd traverse
   ```

3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/jhonsferg/traverse.git
   ```

4. Create a development branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

5. **Install development tools and setup git hooks**:
   ```bash
   make setup
   ```
   This will:
   - Install golangci-lint, goimports, gosec, and lefthook
   - Setup git hooks for automatic code quality checks

   *Alternatively, install tools manually:*
   ```bash
   make dev-setup  # Install tools without git hooks
   ```

### Development Workflow

1. **Make changes** in your feature branch

2. **Test your changes** (several options):
   ```bash
   make test              # Run tests with race detector
   make test-verbose      # Verbose test output
   make test-coverage     # Generate coverage report
   make watch-test        # TDD: auto-run tests on changes
   ```

3. **Check code quality**:
   ```bash
   make lint              # Run golangci-lint
   make vet               # Run go vet
   make fmt               # Auto-format code
   make fmt-check         # Check if code is formatted
   make security          # Run security checks
   ```

4. **Run benchmarks** (if performance-related):
   ```bash
   make bench             # Run benchmarks
   make bench-save        # Save benchmark results
   ```

5. **Build examples**:
   ```bash
   make examples          # Build all example programs
   ```

6. **Comprehensive quality check** (recommended before pushing):
   ```bash
   make all               # Runs: clean, fmt, lint, vet, test, bench
   ```

### Git Hooks

After running `make setup`, the following git hooks are installed:

- **pre-commit**: Automatically formats code and runs linting
- **commit-msg**: Enforces Conventional Commits format
- **pre-push**: Runs tests before allowing a push

To bypass hooks temporarily (not recommended):
```bash
git commit --no-verify  # Skip pre-commit hook
git push --no-verify    # Skip pre-push hook
```

## Submitting Contributions

### Bug Reports

File issues with:
- Description of the bug
- Steps to reproduce
- Expected vs. actual behavior
- Go version and OS
- Relevant code snippet or error message

### Feature Requests

Include:
- Clear description of the feature
- Use case and motivation
- Proposed API (if applicable)
- Examples of usage

### Pull Requests

1. **Create a feature branch** from the latest `main`
2. **Make focused commits**:
   ```bash
   git commit -m "feat: add query timeout support"
   ```
3. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```
4. **Open a pull request** with:
   - Clear title and description
   - Reference to any related issues
   - Evidence that tests pass
   - Updated documentation

#### Commit Message Guidelines

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`

Examples:
- `feat(stream): implement backpressure handling`
- `fix(client): resolve context cancellation race`
- `docs(readme): update SAP integration example`

## Code Style

### Guidelines

- Follow Go conventions and idioms
- Use meaningful variable names
- Keep functions focused and small
- Document exported types and functions
- Write tests for new functionality

### Commenting

- Exported items must have a comment
- Comments should explain *why*, not *what*
- Keep comments concise

```go
// Good: Explains why we use json.Decoder
// Instead of Unmarshal to avoid buffering large arrays
func (q *QueryBuilder) Stream(ctx context.Context) <-chan Result[interface{}] {
```

## Testing

### Test Requirements

- All public functions must have tests
- Aim for ≥85% code coverage
- Test both happy path and error cases
- Use table-driven tests for multiple scenarios

### Example Test Structure

```go
func TestQueryBuilderSelect(t *testing.T) {
	tests := []struct {
		name    string
		fields  []string
		want    []string
		wantErr bool
	}{
		{
			name:   "single field",
			fields: []string{"ID"},
			want:   []string{"ID"},
		},
		{
			name:   "multiple fields",
			fields: []string{"ID", "Name", "Email"},
			want:   []string{"ID", "Name", "Email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := &QueryBuilder{}
			qb.Select(tt.fields...)
			
			if !equal(qb.selectFields, tt.want) {
				t.Errorf("got %v, want %v", qb.selectFields, tt.want)
			}
		})
	}
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Verbose output
go test ./... -v

# With race detector
go test ./... -race

# With coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Documentation

### API Documentation

All exported functions and types must include proper doc comments:

```go
// Stream returns a channel that yields entities one at a time.
// It uses json.Decoder for memory-efficient streaming of large datasets.
// The channel is closed when all results are consumed or an error occurs.
func (q *QueryBuilder) Stream(ctx context.Context) <-chan Result[interface{}] {
```

### Examples

Add runnable examples in `example_*_test.go` files:

```go
func ExampleQueryBuilder_Select() {
	client, _ := traverse.New(traverse.WithBaseURL("http://localhost"))
	qb := client.From("Products").Select("ID", "Name")
	// Output: Products query ready
}
```

### README Updates

Update README.md if you:
- Add new features
- Change public API
- Add new packages or modules
- Fix documentation issues

## Performance Considerations

### Key Principles

1. **Memory Efficiency**: Never buffer entire result sets
2. **Streaming**: Prefer channels and streaming over slices
3. **Pagination**: Handle large datasets with $top and $skip
4. **Concurrency**: Make use of goroutines where appropriate

### Benchmarking

Add benchmarks for performance-critical code:

```bash
go test -bench=. -benchmem ./...
```

## Release Process

Releases are handled by maintainers following semantic versioning:

1. Update `CHANGELOG.md`
2. Update version in relevant files
3. Create git tag: `git tag v0.2.0`
4. Push tag to trigger release workflows

## Getting Help

- 📖 Read the [documentation](https://pkg.go.dev/github.com/jhonsferg/traverse)
- 💬 Join [discussions](https://github.com/jhonsferg/traverse/discussions)
- 🐛 Search [existing issues](https://github.com/jhonsferg/traverse/issues)
- ✉️ Ask maintainers in PR comments

## License

By contributing, you agree that your contributions will be licensed under the MIT License (see LICENSE file).

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- Release notes for significant contributions
- GitHub contributions graph

---

Thank you for contributing to Traverse! 🎉
