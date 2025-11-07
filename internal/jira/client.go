package jira

import (
	"fmt"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

// NewClient creates a simple Jira HTTP client with basic authentication.
// The site can be any Jira instance URL (Cloud, Data Center, Server).
// For self-hosted instances with context paths (e.g. https://domain.com/jira),
// include the full path in the site URL.
func NewClient(site string, creds config.ServiceCredentials) (*HTTPClient, error) {
	if site == "" {
		return nil, fmt.Errorf("jira: site is required")
	}

	// For now, only support basic auth (email + API token)
	// OAuth support can be added later if needed
	if creds.Email == "" || creds.APIToken == "" {
		return nil, fmt.Errorf("jira: email and api_token are required")
	}

	return NewHTTPClient(site, creds)
}
