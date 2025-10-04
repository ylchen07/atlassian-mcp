# Repository Guidelines

## Project Structure & Module Organization
The stdio server entrypoint lives in `cmd/server`, with handler wiring and CLI tests alongside `main.go`. Core logic is grouped under `internal` (e.g., `jira`, `confluence`, `auth`, `state`) while cross-cutting helpers sit in `internal/mcp` and `pkg/logging`. Shared fixtures and HTTP transcripts are under `testdata`. Build output lands in `bin/`, and integration smoke checks reside in `integration/`.

## Build, Test, and Development Commands
- `make deps` – tidy Go modules using the repo-local cache.
- `make lint` – run `golangci-lint` v2 with project rules.
- `make test` – execute unit tests (`CGO_ENABLED=0`) across all packages.
- `make build` – compile the stdio server to `bin/atlassian-mcp`.
- `make run -e CONFIG=...` – launch the MCP server; pass config path or rely on env overrides.

## Coding Style & Naming Conventions
Go files must be formatted with `gofmt`; `goimports` is recommended before committing. Follow standard Go naming—exported types use PascalCase, locals use mixedCase, constants are SCREAMING_SNAKE when truly constant. Limit package exports to what callers need and keep MCP tools in cohesive subpackages. Lint fixes should satisfy `.golangci.yml` without suppressions.

## Testing Guidelines
Unit tests live next to implementations (`*_test.go`). Favour table-driven tests with descriptive `name` fields. When stubbing Atlassian APIs, store fixtures in `testdata/<service>/`. Run `go test ./...` or `make test` before every push. Integration tests in `integration/` are skipped unless credentials are provided—gate them behind environment checks and avoid hitting live APIs by default.

## Commit & Pull Request Guidelines
Use imperative, descriptive commit subjects (e.g., `feat: add Jira transition cache`). Group related changes per commit and include context in the body when behaviour shifts. PRs should link Jira tickets when available, outline configuration changes, and note required follow-up actions. Attach screenshots or sample CLI transcripts when the change affects user interactions.

## Security & Configuration Tips
Never commit secrets; rely on `config.yaml` only for local testing and prefer environment variables such as `ATLASSIAN_JIRA_API_TOKEN`. Treat cached Atlassian responses in `internal/state` as ephemeral; clear them if sharing logs. When developing against live tenants, enable rate-limit guards and confirm scopes before merging.
