package mcp

import (
	"log/slog"

	"gitlab.com/your-org/jira-mcp/internal/confluence"
	"gitlab.com/your-org/jira-mcp/internal/jira"
	"gitlab.com/your-org/jira-mcp/internal/state"

	"github.com/mark3labs/mcp-go/server"
)

// Dependencies bundles the services required for MCP server construction.
type Dependencies struct {
	JiraService       *jira.Service
	ConfluenceService *confluence.Service
	Cache             *state.Cache
	JiraBaseURL       string
	ConfluenceBaseURL string
	Logger            *slog.Logger
}

// NewServer builds an MCP server with registered Jira and Confluence tools.
func NewServer(deps Dependencies) *server.MCPServer {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	srv := server.NewMCPServer(
		"Atlassian MCP",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithInstructions("Tools for Jira and Confluence operations."),
		server.WithRecovery(),
	)

	if deps.Cache == nil {
		deps.Cache = state.NewCache()
	}

	if deps.JiraService != nil {
		NewJiraTools(srv, deps.JiraService, deps.Cache, deps.JiraBaseURL)
	}

	if deps.ConfluenceService != nil {
		NewConfluenceTools(srv, deps.ConfluenceService, deps.ConfluenceBaseURL)
	}

	return srv
}
