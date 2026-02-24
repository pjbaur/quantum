## Description

<!-- Brief description of changes -->

## Type of Change

- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to change)
- [ ] Documentation update
- [ ] Refactoring (no functional changes)

## Checklist

### General

- [ ] Code compiles correctly (`go build ./...`)
- [ ] Tests pass (`go test ./...`)
- [ ] Linter passes (`go vet ./...`)
- [ ] Code is formatted (`gofmt -s -w .`)

### CLI and Documentation Synchronization

If this PR changes CLI behavior, flags, demos, or help text:

- [ ] Updated `usage()` function in `cmd/quantum/main.go`
- [ ] Updated `-demo` flag description to include all available demos
- [ ] Updated example commands in help text
- [ ] Verified help text matches actual implementation

If this PR changes public APIs:

- [ ] Updated package documentation comments
- [ ] Updated `docs/` if behavior changes
- [ ] Created or updated ADR if architectural decision changed

### Tests

- [ ] Added tests for new functionality
- [ ] Added negative-path tests for error cases
- [ ] All new and existing tests pass

## Related Issues

<!-- Link to any related issues -->

## Additional Notes

<!-- Any additional information for reviewers -->
