package confluence

import (
	"strings"

	"github.com/ylchen07/atlassian-mcp/internal/atlassian"
)

const apiPrefix = "/rest/api"

// Service exposes Confluence REST endpoints used by the MCP server.
type Service struct {
	client *atlassian.HTTPClient
}

// NewService constructs a Confluence service.
func NewService(client *atlassian.HTTPClient) *Service {
	return &Service{client: client}
}

// apiPath constructs Confluence API paths by joining parts with the API prefix.
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
