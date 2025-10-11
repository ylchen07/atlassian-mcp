//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ylchen07/atlassian-mcp/internal/atlassian"
	"github.com/ylchen07/atlassian-mcp/internal/config"
	"github.com/ylchen07/atlassian-mcp/internal/confluence"
	"github.com/ylchen07/atlassian-mcp/internal/jira"
)

func TestJiraListProjectsIntegration(t *testing.T) {
	requireIntegration(t)

	jiraSite := ensureHTTPS(resolveEnv(
		"ATLASSIAN_JIRA_SITE",
		"ATLASSIAN_SITE",
	))
	if jiraSite == "" {
		t.Skip("ATLASSIAN_JIRA_SITE not set")
	}

	creds := loadCredentials(os.Getenv("ATLASSIAN_JIRA_EMAIL"), os.Getenv("ATLASSIAN_JIRA_API_TOKEN"), os.Getenv("ATLASSIAN_JIRA_OAUTH_TOKEN"))
	if !credsValid(creds) {
		t.Skip("Jira credentials not provided")
	}

	apiBase := ensureHTTPS(os.Getenv("ATLASSIAN_JIRA_API_BASE"))
	if apiBase != "" {
		jiraSite = apiBase
	}

	client, err := jira.NewV2Client(jiraSite, creds)
	if err != nil {
		t.Fatalf("NewV2Client: %v", err)
	}
	jiraSite = strings.TrimRight(client.Site.String(), "/")

	svc := jira.NewService(client)
	projects, err := svc.ListProjects(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) == 0 {
		t.Logf("no projects returned from Jira site %s", jiraSite)
	}
}

func TestConfluenceListSpacesIntegration(t *testing.T) {
	requireIntegration(t)

	confluenceSite := ensureHTTPS(resolveEnv(
		"ATLASSIAN_CONFLUENCE_SITE",
		"ATLASSIAN_SITE",
	))
	if confluenceSite == "" {
		t.Skip("ATLASSIAN_CONFLUENCE_SITE not set")
	}

	creds := loadCredentials(os.Getenv("ATLASSIAN_CONFLUENCE_EMAIL"), os.Getenv("ATLASSIAN_CONFLUENCE_API_TOKEN"), os.Getenv("ATLASSIAN_CONFLUENCE_OAUTH_TOKEN"))
	if !credsValid(creds) {
		creds = loadCredentials(os.Getenv("ATLASSIAN_JIRA_EMAIL"), os.Getenv("ATLASSIAN_JIRA_API_TOKEN"), os.Getenv("ATLASSIAN_JIRA_OAUTH_TOKEN"))
	}
	if !credsValid(creds) {
		t.Skip("Confluence credentials not provided")
	}

	apiBase := ensureHTTPS(os.Getenv("ATLASSIAN_CONFLUENCE_API_BASE"))
	if apiBase == "" {
		apiBase = fmt.Sprintf("%s/wiki/rest/api", confluenceSite)
	}

	client, err := atlassian.NewClient(apiBase, creds, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	svc := confluence.NewService(client)
	spaces, err := svc.ListSpaces(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListSpaces: %v", err)
	}
	if len(spaces) == 0 {
		t.Logf("no spaces returned from Confluence site %s", confluenceSite)
	}
}

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("MCP_INTEGRATION") == "" {
		t.Skip("MCP_INTEGRATION not set; skipping integration tests")
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

func resolveEnv(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); strings.TrimSpace(val) != "" {
			return val
		}
	}
	return ""
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
