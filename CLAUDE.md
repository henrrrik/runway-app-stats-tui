# Build & Test
- `go test -v -race ./...` — run tests (matches CI)
- `gofmt -s -w .` — run before committing
- `go vet ./...` — run before committing
- `golangci-lint run ./...` — run before committing (config in `.golangci.yml`, includes gocyclo with max complexity 10)

## Workflow

- Use Red/Green TDD
- Create a PR for all changes — do not push directly to main.
- Use interfaces for test doubles — keep this pattern
