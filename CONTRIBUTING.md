# Contributing to wetwire-aws-go

Thank you for your interest in contributing to wetwire-aws-go!

## Getting Started

See the [Developer Guide](docs/DEVELOPERS.md) for:
- Development environment setup
- Project structure
- Running tests

## Code Style

- **Formatting**: Use `gofmt` (automatic in most editors)
- **Linting**: Use `go vet`
- **Imports**: Use `goimports` for automatic import management

```bash
# Format code
gofmt -w .

# Lint
go vet ./...

# Check for common issues
go build ./...
```

## Commit Messages

Follow conventional commits:

```
feat: Add support for EC2 resources
fix: Correct S3 bucket serialization
docs: Update installation instructions
test: Add tests for linter rules
chore: Update dependencies
```

## Pull Request Process

1. Create feature branch: `git checkout -b feature/my-feature`
2. Make changes with tests
3. Run tests: `go test ./...`
4. Run CI: `./scripts/ci.sh`
5. Commit with clear messages
6. Push and open Pull Request
7. Address review comments

## Adding a New Lint Rule

1. Add rule to `internal/lint/rules.go` or `rules_extra.go`
2. Implement the check in `checkFile()` or appropriate function
3. Add test case in `rules_test.go`
4. Update docs/FAQ.md with the new rule
5. Update CLAUDE.md if it affects syntax guidance

## Adding a New CLI Command

1. Create `cmd/wetwire-aws/<command>.go`
2. Implement `new<Command>Cmd()` function returning `*cobra.Command`
3. Register in `main.go` with `rootCmd.AddCommand()`
4. Add tests in `<command>_test.go`
5. Update docs/CLI.md documentation

## Reporting Issues

- Use GitHub Issues for bug reports and feature requests
- Include reproduction steps for bugs
- Check existing issues before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
