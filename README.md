# Jira MCP Server

This project provides a Model Context Protocol (MCP) server implemented in Go using the [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) framework. The server exposes tooling for interacting with Atlassian Jira and Confluence over stdio.

## Features

- Jira support: list projects, search issues (JQL), create issues, update fields, manage comments.
- Confluence support: list spaces, search pages, create and update pages.
- Configurable authentication via Atlassian API tokens or OAuth client credentials.
- Structured error handling with rate limit awareness.
- Extensible MCP tool registry built on `mcp-go`.

## Getting Started

1. Install Go 1.22+.
2. Clone the repository and run `make deps` to tidy dependencies.
3. Copy `config.example.yaml` to `config.yaml` and populate Jira/Confluence credentials.
4. Run the server: `make run` (starts stdio MCP server).

## Configuration

Configuration is resolved from `config.yaml` and environment variables. See `config.example.yaml` for the full schema. Environment variables use the prefix `JIRA_MCP_`.

## Development

- `make lint` – run `golangci-lint` (ensure version 2.x is installed locally).
- `make test` – execute unit tests with a local build cache.
- `make run` – launch the stdio MCP server.

You can also run the linters directly:

```bash
CGO_ENABLED=0 XDG_CACHE_HOME=$(pwd)/.cache GOLANGCI_LINT_CACHE=$(pwd)/.cache/golangci golangci-lint run ./...
```

### Configuration tips

- Set `JIRA_MCP_ATLASSIAN_SITE`, `JIRA_MCP_ATLASSIAN_JIRA_EMAIL`, and related env vars to avoid committing secrets.
- Use `config.yaml` only for local development; CI loads credentials from environment variables.

## CI/CD

A `.gitlab-ci.yml` file is included for linting, testing, and building in GitLab pipelines.

## License

MIT
