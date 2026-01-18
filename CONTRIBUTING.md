# Contributing to Astral

First off, thank you for considering contributing to Astral! 🎉

## Code of Conduct

Be respectful, inclusive, and constructive in all interactions.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce**
- **Expected vs actual behavior**
- **Environment** (OS, Go version, etc.)
- **Screenshots** if applicable

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. Provide:

- **Clear use case** - Why is this needed?
- **Proposed solution** - How should it work?
- **Alternatives considered**

### Pull Requests

1. **Fork the repo** and create your branch from `main`
2. **Follow Go conventions** - Use `gofmt`, `golint`
3. **Add tests** - Maintain coverage above 80%
4. **Update documentation** - Keep README and comments current
5. **Write clear commit messages**

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/astral.git
cd astral

# Install dependencies
go mod download

# Run tests
make test

# Build
make build
```

## Code Style

### Go Guidelines

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Maximum line length: 120 characters
- Add comments for exported functions
- Use meaningful variable names

### Error Handling

```go
// Good
if err != nil {
    return fmt.Errorf("failed to read file: %w", err)
}

// Bad - never panic in library code
if err != nil {
    panic(err)
}
```

### Testing

- Write table-driven tests where appropriate
- Use descriptive test names: `TestFunctionName_Scenario`
- Add benchmarks for performance-critical code
- Aim for >80% coverage

Example:

```go
func TestHashBytes(t *testing.T) {
    tests := []struct {
        name string
        data []byte
        want string
    }{
        {"empty", []byte{}, "expected_hash"},
        {"simple", []byte("hello"), "expected_hash"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := HashBytes(tt.data)
            if got.String() != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Commit Messages

Follow conventional commits:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `test`: Tests
- `refactor`: Code refactoring
- `perf`: Performance improvement

Examples:
```
feat(storage): add object caching layer
fix(cli): correct branch switching behavior
docs(readme): update installation instructions
```

## Testing Checklist

Before submitting a PR:

- [ ] All tests pass (`make test`)
- [ ] Code is formatted (`make fmt`)
- [ ] No lint errors (`make vet`)
- [ ] Benchmarks run without regression
- [ ] Documentation is updated
- [ ] Commit messages are clear

## Performance Considerations

- Use benchmarks for critical paths
- Profile before optimizing
- Consider memory allocations
- Use concurrent operations where safe
- Avoid premature optimization

## Questions?

Feel free to open an issue for discussion!

---

**Thank you for contributing to Astral!** 🚀
