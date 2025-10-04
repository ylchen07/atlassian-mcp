# Atlassian MCP Server

This project provides a Model Context Protocol (MCP) server implemented in Go using the [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) framework. The server exposes tooling for interacting with Atlassian Jira and Confluence over stdio.

## Features

- Jira support: list projects, search issues (JQL), create issues, update fields, manage comments.
- Confluence support: list spaces, search pages, create and update pages.
- Configurable authentication via Atlassian API tokens or OAuth client credentials.
- Structured error handling with rate limit awareness.
- Extensible MCP tool registry built on `mcp-go`.

## Getting Started

1. Install Go 1.25+.
2. Clone the repository and run `make deps` to tidy dependencies.
3. Copy `config.example.yaml` to `config.yaml` and populate Jira/Confluence credentials.
4. Build the binary: `make build` (outputs to `bin/atlassian-mcp`).
5. Run the server: `make run` (starts stdio MCP server).

## Configuration

Configuration is resolved from `config.yaml` and environment variables. See `config.example.yaml` for the full schema.

### Environment variables

- `ATLASSIAN_JIRA_SITE` – Jira base URL (e.g. `https://jira.internal.example`).
- `ATLASSIAN_JIRA_API_BASE` – Optional Jira REST API base override (defaults to `<ATLASSIAN_JIRA_SITE>/rest/api/3`).
- `ATLASSIAN_JIRA_EMAIL` / `ATLASSIAN_JIRA_API_TOKEN` – Jira API credentials when not using OAuth.
- `ATLASSIAN_JIRA_OAUTH_TOKEN` – Jira OAuth 2.0 token (set instead of email/api token).
- `ATLASSIAN_CONFLUENCE_SITE` – Confluence base URL (e.g. `https://confluence.internal.example`).
- `ATLASSIAN_CONFLUENCE_API_BASE` – Optional Confluence REST API base (defaults to `<ATLASSIAN_CONFLUENCE_SITE>/wiki/rest/api`).
- `ATLASSIAN_CONFLUENCE_EMAIL` / `ATLASSIAN_CONFLUENCE_API_TOKEN` – Confluence API credentials when not using OAuth.
- `ATLASSIAN_CONFLUENCE_OAUTH_TOKEN` – Confluence OAuth 2.0 token (set instead of email/api token).
- `ATLASSIAN_SITE` – Legacy shared hostname fallback when per-service sites are unset.

## Development

- `make lint` – run `golangci-lint` (ensure v2.5.x or newer is installed locally; the config uses the v2 schema).
- `make test` – execute unit tests with a local build cache.
- `make build` – compile the MCP server binary into `bin/`.
- `make run` – launch the stdio MCP server.

You can also run the linters directly:

```bash
CGO_ENABLED=0 XDG_CACHE_HOME=$(pwd)/.cache GOLANGCI_LINT_CACHE=$(pwd)/.cache/golangci golangci-lint run ./...
```

### Configuration tips

- Set `ATLASSIAN_SITE`, `ATLASSIAN_JIRA_EMAIL`, and `ATLASSIAN_CONFLUENCE_EMAIL` (with their corresponding tokens) to avoid committing secrets.
- Use `config.yaml` only for local development; CI loads credentials from environment variables.

## CI/CD

A `.gitlab-ci.yml` file is included for linting, testing, and building in GitLab pipelines.

## License

MIT
