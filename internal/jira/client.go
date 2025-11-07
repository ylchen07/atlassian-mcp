package jira

import (
	"fmt"

	"github.com/ylchen07/atlassian-mcp/internal/atlassian"
	"github.com/ylchen07/atlassian-mcp/internal/config"
)

// NewClient creates a Jira HTTP client using the shared Atlassian HTTP client.
// The site can be any Jira instance URL (Cloud, Data Center, Server).
// For self-hosted instances with context paths (e.g. https://domain.com/jira),
// include the full path in the site URL.
func NewClient(site string, creds config.ServiceCredentials) (*atlassian.HTTPClient, error) {
	if site == "" {
		return nil, fmt.Errorf("jira: site is required")
	}

	client, err := atlassian.NewHTTPClient(site, creds)
	if err != nil {
		return nil, fmt.Errorf("jira: %w", err)
	}

	return client, nil
}
