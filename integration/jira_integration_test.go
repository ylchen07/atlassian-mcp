//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"

	"github.com/ylchen07/atlassian-mcp/internal/jira"
)

func TestJiraListProjects(t *testing.T) {
	requireIntegration(t)

	svc, siteURL := setupJiraClient(t)

	projects, err := svc.ListProjects(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}

	if len(projects) == 0 {
		t.Logf("no projects returned from Jira site %s", siteURL)
		return
	}

	t.Logf("Found %d projects on %s", len(projects), siteURL)
	for i, project := range projects {
		t.Logf("  [%d] %s (%s) - %s", i+1, project.Key, project.ID, project.Name)
	}
}

func TestJiraSearchIssues(t *testing.T) {
	requireIntegration(t)

	svc, siteURL := setupJiraClient(t)

	// First, get projects to construct a valid JQL
	projects, err := svc.ListProjects(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	skipIfEmpty(t, projects, "projects")

	projectKey := projects[0].Key

	// Search for recent issues in the first project
	req := jira.SearchRequest{
		JQL:        "project = " + projectKey + " ORDER BY created DESC",
		MaxResults: 5,
		Fields:     []string{"summary", "status", "assignee"},
	}

	result, err := svc.SearchIssues(context.Background(), req)
	if err != nil {
		t.Fatalf("SearchIssues failed: %v", err)
	}

	t.Logf("Found %d issues in project %s on %s", result.Total, projectKey, siteURL)
	for i, issue := range result.Issues {
		assignee := "Unassigned"
		if issue.Fields.Assignee.DisplayName != "" {
			assignee = issue.Fields.Assignee.DisplayName
		}
		t.Logf("  [%d] %s: %s [%s] - %s",
			i+1,
			issue.Key,
			issue.Fields.Summary,
			issue.Fields.Status.Name,
			assignee,
		)
	}
}

func TestJiraListTransitions(t *testing.T) {
	requireIntegration(t)

	svc, siteURL := setupJiraClient(t)

	// First, find an issue to check transitions
	projects, err := svc.ListProjects(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	skipIfEmpty(t, projects, "projects")

	projectKey := projects[0].Key

	// Search for one issue
	req := jira.SearchRequest{
		JQL:        "project = " + projectKey + " ORDER BY created DESC",
		MaxResults: 1,
	}

	result, err := svc.SearchIssues(context.Background(), req)
	if err != nil {
		t.Fatalf("SearchIssues failed: %v", err)
	}
	skipIfEmpty(t, result.Issues, "issues")

	issueKey := result.Issues[0].Key

	// List available transitions
	transitions, err := svc.ListTransitions(context.Background(), issueKey)
	if err != nil {
		t.Fatalf("ListTransitions failed: %v", err)
	}

	t.Logf("Found %d transitions for issue %s on %s", len(transitions), issueKey, siteURL)
	for i, transition := range transitions {
		t.Logf("  [%d] %s (ID: %s) -> %s", i+1, transition.Name, transition.ID, transition.To.Name)
	}
}
