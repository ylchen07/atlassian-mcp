package integration

import (
	"os"
	"strings"
	"testing"

	"github.com/ylchen07/atlassian-mcp/internal/atlassian"
	"github.com/ylchen07/atlassian-mcp/internal/config"
	"github.com/ylchen07/atlassian-mcp/internal/confluence"
	"github.com/ylchen07/atlassian-mcp/internal/jira"
)

// requireIntegration skips the test if MCP_INTEGRATION environment variable is not set.
func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("MCP_INTEGRATION") == "" {
		t.Skip("MCP_INTEGRATION not set; skipping integration tests")
	}
}

// ensureHTTPS adds https:// prefix to URLs if not already present.
func ensureHTTPS(site string) string {
	trimmed := strings.TrimSpace(site)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return strings.TrimRight(trimmed, "/")
	}
	return "https://" + strings.TrimRight(trimmed, "/")
}

// resolveEnv returns the first non-empty environment variable value from the provided keys.
func resolveEnv(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); strings.TrimSpace(val) != "" {
			return val
		}
	}
	return ""
}

// loadCredentials creates ServiceCredentials from environment variables.
func loadCredentials(email, token, oauth string) config.ServiceCredentials {
	return config.ServiceCredentials{
		Email:      email,
		APIToken:   token,
		OAuthToken: oauth,
	}
}

// credsValid checks if credentials are valid (either OAuth token or email+API token).
func credsValid(creds config.ServiceCredentials) bool {
	if creds.OAuthToken != "" {
		return true
	}
	return creds.Email != "" && creds.APIToken != ""
}

// setupJiraClient creates and configures a Jira client from environment variables.
// Returns nil and skips the test if credentials are not available.
func setupJiraClient(t *testing.T) (*jira.Service, string) {
	t.Helper()

	jiraSite := ensureHTTPS(resolveEnv(
		"ATLASSIAN_JIRA_SITE",
		"ATLASSIAN_SITE",
	))
	if jiraSite == "" {
		t.Skip("ATLASSIAN_JIRA_SITE not set")
	}

	creds := loadCredentials(
		os.Getenv("ATLASSIAN_JIRA_EMAIL"),
		os.Getenv("ATLASSIAN_JIRA_API_TOKEN"),
		os.Getenv("ATLASSIAN_JIRA_OAUTH_TOKEN"),
	)
	if !credsValid(creds) {
		t.Skip("Jira credentials not provided")
	}

	apiBase := ensureHTTPS(os.Getenv("ATLASSIAN_JIRA_API_BASE"))
	if apiBase != "" {
		jiraSite = apiBase
	}

	httpClient, err := atlassian.NewHTTPClient(jiraSite, creds)
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}

	svc := jira.NewService(httpClient)
	return svc, strings.TrimRight(jiraSite, "/")
}

// setupConfluenceClient creates and configures a Confluence client from environment variables.
// Returns nil and skips the test if credentials are not available.
func setupConfluenceClient(t *testing.T) (*confluence.Service, string) {
	t.Helper()

	confluenceSite := ensureHTTPS(resolveEnv(
		"ATLASSIAN_CONFLUENCE_SITE",
		"ATLASSIAN_SITE",
	))
	if confluenceSite == "" {
		t.Skip("ATLASSIAN_CONFLUENCE_SITE not set")
	}

	creds := loadCredentials(
		os.Getenv("ATLASSIAN_CONFLUENCE_EMAIL"),
		os.Getenv("ATLASSIAN_CONFLUENCE_API_TOKEN"),
		os.Getenv("ATLASSIAN_CONFLUENCE_OAUTH_TOKEN"),
	)
	if !credsValid(creds) {
		// Fallback to Jira credentials (common for Atlassian Cloud)
		creds = loadCredentials(
			os.Getenv("ATLASSIAN_JIRA_EMAIL"),
			os.Getenv("ATLASSIAN_JIRA_API_TOKEN"),
			os.Getenv("ATLASSIAN_JIRA_OAUTH_TOKEN"),
		)
	}
	if !credsValid(creds) {
		t.Skip("Confluence credentials not provided")
	}

	apiBase := ensureHTTPS(os.Getenv("ATLASSIAN_CONFLUENCE_API_BASE"))
	if apiBase != "" {
		confluenceSite = apiBase
	}

	httpClient, err := atlassian.NewHTTPClient(confluenceSite, creds)
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}

	svc := confluence.NewService(httpClient)
	return svc, strings.TrimRight(confluenceSite, "/")
}

// skipIfEmpty skips the test if the provided slice is empty with a helpful message.
func skipIfEmpty[T any](t *testing.T, items []T, itemType string) {
	t.Helper()
	if len(items) == 0 {
		t.Skipf("no %s found; cannot proceed with test", itemType)
	}
}
