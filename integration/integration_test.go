//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"gitlab.com/your-org/jira-mcp/internal/atlassian"
	"gitlab.com/your-org/jira-mcp/internal/config"
	"gitlab.com/your-org/jira-mcp/internal/confluence"
	"gitlab.com/your-org/jira-mcp/internal/jira"
)

func TestJiraListProjectsIntegration(t *testing.T) {
	requireIntegration(t)

	site := ensureHTTPS(os.Getenv("JIRA_MCP_ATLASSIAN_SITE"))
	if site == "" {
		t.Skip("JIRA_MCP_ATLASSIAN_SITE not set")
	}

	creds := loadCredentials(os.Getenv("JIRA_MCP_ATLASSIAN_JIRA_EMAIL"), os.Getenv("JIRA_MCP_ATLASSIAN_JIRA_API_TOKEN"), os.Getenv("JIRA_MCP_ATLASSIAN_JIRA_OAUTH"))
	if !credsValid(creds) {
		t.Skip("Jira credentials not provided")
	}

	client, err := atlassian.NewClient(fmt.Sprintf("%s/rest/api/3", site), creds, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	svc := jira.NewService(client)
	projects, err := svc.ListProjects(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) == 0 {
		t.Logf("no projects returned from Jira site %s", site)
	}
}

func TestConfluenceListSpacesIntegration(t *testing.T) {
	requireIntegration(t)

	site := ensureHTTPS(os.Getenv("JIRA_MCP_ATLASSIAN_SITE"))
	if site == "" {
		t.Skip("JIRA_MCP_ATLASSIAN_SITE not set")
	}

	creds := loadCredentials(os.Getenv("JIRA_MCP_ATLASSIAN_CONFLUENCE_EMAIL"), os.Getenv("JIRA_MCP_ATLASSIAN_CONFLUENCE_API_TOKEN"), os.Getenv("JIRA_MCP_ATLASSIAN_CONFLUENCE_OAUTH"))
	if !credsValid(creds) {
		creds = loadCredentials(os.Getenv("JIRA_MCP_ATLASSIAN_JIRA_EMAIL"), os.Getenv("JIRA_MCP_ATLASSIAN_JIRA_API_TOKEN"), os.Getenv("JIRA_MCP_ATLASSIAN_JIRA_OAUTH"))
	}
	if !credsValid(creds) {
		t.Skip("Confluence credentials not provided")
	}

	client, err := atlassian.NewClient(fmt.Sprintf("%s/wiki/rest/api", site), creds, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	svc := confluence.NewService(client)
	spaces, err := svc.ListSpaces(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListSpaces: %v", err)
	}
	if len(spaces) == 0 {
		t.Logf("no spaces returned from Confluence site %s", site)
	}
}

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("JIRA_MCP_INTEGRATION") == "" {
		t.Skip("JIRA_MCP_INTEGRATION not set; skipping integration tests")
	}
}

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

func loadCredentials(email, token, oauth string) config.ServiceCredentials {
	return config.ServiceCredentials{
		Email:      email,
		APIToken:   token,
		OAuthToken: oauth,
	}
}

func credsValid(creds config.ServiceCredentials) bool {
	if creds.OAuthToken != "" {
		return true
	}
	return creds.Email != "" && creds.APIToken != ""
}
