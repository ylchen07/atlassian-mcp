# Jira MCP Server Agent Notes

## Project Overview
- Implements an MCP stdio server using `github.com/mark3labs/mcp-go`.
- Provides tooling for Atlassian Jira (project listing, JQL search, issue lifecycle) and Confluence (spaces, search, page CRUD).
- Configuration resolved via `config.yaml` plus `JIRA_MCP_*` environment overrides.

## Tech Stack
- Go 1.25
- `mcp-go` v0.41.1 for MCP protocol helpers.
- `github.com/spf13/viper` for config management.
- `golangci-lint` v2 for linting (configured in `.golangci.yml`).

## Local Workflow
- `make deps` → tidy modules using workspace cache.
- `make test` → runs Go unit tests with CGO disabled.
- `make lint` → executes golangci-lint with cache directories bound to repo.
- `make build` → compiles the stdio server to `bin/jira-mcp`.
- `make run` → starts stdio server; supply config via `-config` flag or env vars.

Useful manual commands:
```bash
CGO_ENABLED=0 GOCACHE=$(pwd)/.cache/go-build go test ./...
CGO_ENABLED=0 XDG_CACHE_HOME=$(pwd)/.cache GOLANGCI_LINT_CACHE=$(pwd)/.cache/golangci golangci-lint run ./...
```

## Testing & Coverage
- Unit tests cover config validation, state cache, helper utilities, and HTTP service interactions via mock transports.
- Confluence tests validate list/search/create/update flows; Jira tests cover project search and issue search payloads.
- Integration tests against live Atlassian APIs are not included; add before production usage.

## CI/CD
- `.gitlab-ci.yml` defines lint and test stages using Go 1.25 image.
- Pipeline installs `golangci-lint` 1.60.3 explicitly before lint stage.

## Open Follow-ups
- Add Jira transition/comment retrieval handlers and Confluence attachment support.
- Provide integration test harness guarded by environment variables.
- Consider secrets management integration (e.g., Vault) for production deployments.
