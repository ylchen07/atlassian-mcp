package jira

import (
	"strings"

	"github.com/ylchen07/atlassian-mcp/internal/atlassian"
)

const apiPrefix = "/rest/api/2"

// Service exposes Jira REST endpoints used by the MCP server.
type Service struct {
	client *atlassian.HTTPClient
}

// NewService creates a Jira service using the provided HTTP client.
func NewService(client *atlassian.HTTPClient) *Service {
	return &Service{client: client}
}

// apiPath constructs Jira API paths by joining parts with the API prefix.
func apiPath(parts ...string) string {
	builder := strings.Builder{}
	builder.WriteString(strings.TrimRight(apiPrefix, "/"))

	for _, part := range parts {
		if trimmed := strings.Trim(part, "/"); trimmed != "" {
			builder.WriteByte('/')
			builder.WriteString(trimmed)
		}
	}

	return builder.String()
}
