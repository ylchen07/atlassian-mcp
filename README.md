# Atlassian MCP Server

The Atlassian MCP server is a Go implementation of the Model Context Protocol. It exposes Jira and Confluence tooling over stdio using [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go), enabling AI copilots and other MCP-aware clients to automate Atlassian workflows.

## Capabilities

- Idiomatic Jira and Confluence clients powered by the go-atlassian v2 SDK with shared authentication, caching, and structured error reporting.
- Ready-to-use MCP tools for common Jira and Confluence tasks (projects, issues, pages, search, transitions, attachments, and comments).
- Configuration by file or environment variables with `.env` support for local development.
- Make-based developer workflow for dependency tidying, linting, testing, and building a standalone stdio binary.

## Package Relationships

```mermaid
graph TD
  cmd["cmd/server"] --> cfg["internal/config"]
  cmd --> log["pkg/logging"]
  cmd --> jira["internal/jira"]
  cmd --> conf["internal/confluence"]
  cmd --> mcp["internal/mcp"]
  cmd --> state["internal/state"]

  mcp --> jira
  mcp --> conf
  mcp --> state

  jira --> goatl["github.com/ctreminiom/go-atlassian/v2"]
  conf --> cfgoatl["github.com/ctreminiom/go-atlassian/v2"]
```

### Jira tools

| Tool ID                 | Description                                                   |
| ----------------------- | ------------------------------------------------------------- |
| `jira.list_projects`    | Return the accessible Jira projects (cached for the session). |
| `jira.search_issues`    | Run a JQL query and return issue summaries.                   |
| `jira.create_issue`     | Create a new issue in the specified project.                  |
| `jira.update_issue`     | Update issue fields with partial payloads.                    |
| `jira.add_comment`      | Append a comment using Atlassian document format.             |
| `jira.list_transitions` | Retrieve available workflow transitions for an issue.         |
| `jira.transition_issue` | Apply a workflow transition, optionally updating fields.      |
| `jira.add_attachment`   | Upload binary attachments to an issue.                        |

### Confluence tools

| Tool ID                   | Description                                            |
| ------------------------- | ------------------------------------------------------ |
| `confluence.list_spaces`  | List spaces available to the authenticated account.    |
| `confluence.search_pages` | Execute CQL searches and return page summaries.        |
| `confluence.create_page`  | Create pages with optional parent relationships.       |
| `confluence.update_page`  | Update existing pages with optimistic version control. |

## Prerequisites

- Go 1.25+ (matching the module directive in `go.mod`). Install via `gotip` or a Go distribution that provides 1.25 if you are building before the official release.
- Atlassian Jira and/or Confluence site with an API token or OAuth access token (per service).

## Quick Start

1. Clone the repository and install dependencies:

   ```bash
   git clone https://github.com/ylchen07/atlassian-mcp.git
   cd atlassian-mcp
   make deps
   ```

2. Copy the sample configuration and fill in your tenant details:

   ```bash
   cp config.example.yaml config.yaml
   ```

   Alternatively, copy `.env.example` to `.env` and export credentials there. Environment variables always override file-based values.

3. Build the stdio binary (emits `bin/atlassian-mcp`):

   ```bash
   make build
   ```

4. Run the server:

   ```bash
   # Uses config.yaml in the repo root by default
   make run

   # or explicitly point to a config file/directory
   go run ./cmd/server --config /path/to/config.yaml
   ```

Connect the resulting stdio process to any MCP-compatible client (e.g. mark3labs tools, Automations, IDE agents).

## Configuration

Configuration can be supplied by:

- `config.yaml` or another file passed via `--config` (see `config.example.yaml` for the full schema).
- Environment variables (`viper` automatically maps uppercase underscore-separated keys).
- A local `.env` file loaded by tooling such as `direnv` or `dotenv`.

The loader searches the current working directory first, then `~/.config/atlassian-mcp/config.yaml`, before falling back to environment variables.

Key environment variables:

- `ATLASSIAN_JIRA_SITE` / `ATLASSIAN_CONFLUENCE_SITE` – Base URLs for each product (`https://…`).
- `ATLASSIAN_JIRA_API_BASE` / `ATLASSIAN_CONFLUENCE_API_BASE` – Optional REST overrides if your deployment is proxied.
- `ATLASSIAN_JIRA_EMAIL` & `ATLASSIAN_JIRA_API_TOKEN` – Basic auth credentials for Jira.
- `ATLASSIAN_CONFLUENCE_EMAIL` & `ATLASSIAN_CONFLUENCE_API_TOKEN` – Basic auth credentials for Confluence.
- `ATLASSIAN_JIRA_OAUTH_TOKEN` / `ATLASSIAN_CONFLUENCE_OAUTH_TOKEN` – OAuth bearer token alternatives to email/API tokens.
- `ATLASSIAN_SITE` – Legacy shared hostname fallback used when per-product sites are omitted.
- `SERVER_LOG_LEVEL` – Optional log level (`debug`, `info`, `warn`, `error`).

For local development, prefer environment variables over committing secrets to `config.yaml`.

## Development Workflow

- `make deps` – Run `go mod tidy` using the repo-scoped cache.
- `make fmt` – Format Go sources with `gofmt` across all packages.
- `make lint` – Run `golangci-lint`; use v1.55 or newer (config schema version 2).
- `make test` – Execute unit tests with `CGO_ENABLED=0`.
- `make build` – Produce the stdio binary in `bin/`.
- `make run` – Launch the MCP server via `go run ./cmd/server`.

The lint target can also be invoked manually:

```bash
CGO_ENABLED=0 XDG_CACHE_HOME=$(pwd)/.cache GOLANGCI_LINT_CACHE=$(pwd)/.cache/golangci golangci-lint run ./...
```

## Testing

- Unit tests: `make test` or `go test ./...`.
- Integration smoke tests (require real credentials):

  ```bash
  MCP_INTEGRATION=1 go test -tags=integration ./integration
  ```

  These tests respect the same environment variables as the server and will be skipped unless the required credentials are present.

## CI/CD

GitHub Actions (`.github/workflows/ci.yml`) runs lint and test jobs on every push and pull request. A legacy `.gitlab-ci.yml` is provided for teams executing the pipeline in GitLab.

## License

MIT
