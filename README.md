# Atlassian MCP Server

A Go implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io) that exposes Jira and Confluence operations as tools for AI assistants and automation clients.

## Features

- **Jira Tools**: Projects, issues, search (JQL) - *more tools coming soon*
- **Confluence Tools**: Spaces, pages, search (CQL), content management
- **Flexible Configuration**: YAML files, environment variables, or hybrid approach
- **Smart Caching**: Session-based project caching to minimize API calls
- **Dual Authentication**: OAuth or Basic Auth (email + API token)
- **Self-Hosted Support**: Works with Jira/Confluence Data Center (with context paths)

## Quick Start

### 1. Install Dependencies

```bash
git clone https://github.com/ylchen07/atlassian-mcp.git
cd atlassian-mcp
make deps
```

### 2. Configure

**Option A: Environment Variables (Recommended)**

```bash
export ATLASSIAN_JIRA_SITE=https://your-domain.atlassian.net
export ATLASSIAN_JIRA_EMAIL=user@example.com
export ATLASSIAN_JIRA_API_TOKEN=your_api_token

export ATLASSIAN_CONFLUENCE_SITE=https://your-domain.atlassian.net
export ATLASSIAN_CONFLUENCE_EMAIL=user@example.com
export ATLASSIAN_CONFLUENCE_API_TOKEN=your_api_token
```

**Option B: Configuration File**

```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your credentials
```

See [Configuration](#configuration) for all options.

### 3. Build & Run

**Option A: Install to PATH (Recommended)**

```bash
make install
# Installs to ~/.local/bin/atlassian-mcp

# Run from anywhere
atlassian-mcp
```

**Option B: Build Only**

```bash
make build
./bin/atlassian-mcp
```

**Option C: Run Directly (Development)**

```bash
make run
```

The server communicates over stdio and can be connected to any MCP-compatible client.

## Available Tools

### Jira

| Tool                    | Status | Description                        |
| ----------------------- | ------ | ---------------------------------- |
| `jira.list_projects`    | ✅      | List accessible projects (cached)  |
| `jira.search_issues`    | ✅      | Execute JQL queries                |
| `jira.create_issue`     | 🚧      | Create new issues                  |
| `jira.update_issue`     | 🚧      | Update issue fields                |
| `jira.add_comment`      | 🚧      | Add comments to issues             |
| `jira.list_transitions` | 🚧      | Get available workflow transitions |
| `jira.transition_issue` | 🚧      | Move issues through workflow       |
| `jira.add_attachment`   | 🚧      | Upload file attachments            |

✅ = Available now | 🚧 = Coming soon

### Confluence

| Tool                      | Description            |
| ------------------------- | ---------------------- |
| `confluence.list_spaces`  | List accessible spaces |
| `confluence.search_pages` | Execute CQL queries    |
| `confluence.create_page`  | Create new pages       |
| `confluence.update_page`  | Update existing pages  |

## Configuration

### Configuration Sources

The server loads configuration from multiple sources (in precedence order):

1. **Environment variables** - Highest priority, always override other sources
2. **`config.yaml`** - Searched in: `--config` flag path → current directory → `~/.config/atlassian-mcp/`
3. **`.netrc` file** - Automatic credential loading from `~/.netrc` or `$NETRC` path
4. **Defaults** - Built-in fallback values (e.g., `log_level: info`)

### Required Settings

**Per Service (Jira and Confluence)**:

- Site URL: `ATLASSIAN_JIRA_SITE` / `ATLASSIAN_CONFLUENCE_SITE`
- Authentication (choose one):
  - Basic Auth: `*_EMAIL` + `*_API_TOKEN`
  - OAuth: `*_OAUTH_TOKEN`

**Example 1**: Using `.netrc` for credentials (recommended for security)

```bash
# ~/.netrc (chmod 600)
machine your-domain.atlassian.net
  login user@example.com
  password your_api_token
```

```yaml
# config.yaml (only site URLs needed)
atlassian:
  jira:
    site: https://your-domain.atlassian.net
  confluence:
    site: https://your-domain.atlassian.net
```

**Example 2**: Using environment variables

```bash
export ATLASSIAN_JIRA_SITE=https://your-domain.atlassian.net
export ATLASSIAN_JIRA_EMAIL=user@example.com
export ATLASSIAN_JIRA_API_TOKEN=secret_token
export ATLASSIAN_CONFLUENCE_SITE=https://your-domain.atlassian.net
export ATLASSIAN_CONFLUENCE_EMAIL=user@example.com
export ATLASSIAN_CONFLUENCE_API_TOKEN=secret_token
```

### Environment Variables Reference

| Variable                           | Description                       | Required                |
| ---------------------------------- | --------------------------------- | ----------------------- |
| `ATLASSIAN_JIRA_SITE`              | Jira base URL                     | Yes                     |
| `ATLASSIAN_JIRA_EMAIL`             | Email for basic auth              | If not using OAuth      |
| `ATLASSIAN_JIRA_API_TOKEN`         | API token for basic auth          | If not using OAuth      |
| `ATLASSIAN_JIRA_OAUTH_TOKEN`       | OAuth token                       | If not using basic auth |
| `ATLASSIAN_CONFLUENCE_SITE`        | Confluence base URL               | Yes                     |
| `ATLASSIAN_CONFLUENCE_EMAIL`       | Email for basic auth              | If not using OAuth      |
| `ATLASSIAN_CONFLUENCE_API_TOKEN`   | API token                         | If not using OAuth      |
| `ATLASSIAN_CONFLUENCE_OAUTH_TOKEN` | OAuth token                       | If not using basic auth |
| `SERVER_LOG_LEVEL`                 | Log level (debug/info/warn/error) | No (default: info)      |

**Advanced Options**:

- `ATLASSIAN_JIRA_API_BASE` - Override REST API base URL
- `ATLASSIAN_CONFLUENCE_API_BASE` - Override REST API base URL
- `ATLASSIAN_SITE` - Legacy shared hostname fallback
- `NETRC` - Custom path to .netrc file (default: `~/.netrc`)

**Mapping**: YAML keys map to uppercase with underscores: `atlassian.jira.site` → `ATLASSIAN_JIRA_SITE`

### Using .netrc for Credentials

The server automatically reads credentials from `.netrc` file if email/api_token are not provided via config or environment variables.

**Format**:

```
machine your-domain.atlassian.net
  login user@example.com
  password your_api_token
```

**Benefits**:

- ✅ Standard Unix credential storage (used by `curl`, `git`, etc.)
- ✅ Keeps secrets out of config files and environment variables
- ✅ One credential file for multiple tools
- ✅ Supports multiple machines in one file

**Security**: Ensure `.netrc` has proper permissions: `chmod 600 ~/.netrc`

See [`config.example.yaml`](config.example.yaml) for complete schema with inline documentation.

## Development

### Build Commands

```bash
make deps          # Install dependencies
make fmt           # Format code
make lint          # Run linters (requires golangci-lint v1.55+)
make test          # Run unit tests
make test-coverage # Generate test coverage report
make build         # Build binary to bin/atlassian-mcp
make run           # Run server directly
make clean         # Remove build artifacts and cache
```

### Testing

**Unit Tests**:

```bash
make test
```

**Test Coverage**:

```bash
make test-coverage  # Generates coverage.out and coverage.html
```

Opens `coverage.html` in your browser to see detailed line-by-line coverage visualization.

**Integration Tests** (requires real Atlassian credentials):

```bash
MCP_INTEGRATION=1 go test -tags=integration ./integration
```

Integration tests use the same environment variables as the server and skip when credentials are missing.

### Project Structure

```
cmd/server          → CLI entry point
internal/
  config/          → Viper-based configuration
  jira/            → Jira client & service layer
  confluence/      → Confluence client & service layer
  mcp/             → MCP server & tool registration
  state/           → Thread-safe session cache
pkg/logging/       → Structured logging (slog)
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design documentation.

## Prerequisites

- **Go 1.25+** (see `go.mod`)
- **Atlassian Account**: Jira and/or Confluence with API access
- **API Token**: Generate at <https://id.atlassian.com/manage-profile/security/api-tokens>
- **MCP Client**: Any MCP-compatible client to connect to the server

## CI/CD

GitHub Actions runs linting and testing on every push and pull request. See [`.github/workflows/ci.yml`](.github/workflows/ci.yml).

## Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) - Design patterns and layered architecture
- [CLAUDE.md](CLAUDE.md) - Development guide and project overview
- [internal/state/README.md](internal/state/README.md) - Session cache documentation

## External References

- [MCP Documentation](https://modelcontextprotocol.io)
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) - Go MCP framework
- [go-atlassian SDK](https://deepwiki.com/ctreminiom/go-atlassian) - Atlassian API client

## License

MIT
