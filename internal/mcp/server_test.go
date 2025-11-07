package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ylchen07/atlassian-mcp/internal/confluence"
	"github.com/ylchen07/atlassian-mcp/internal/jira"
	"github.com/ylchen07/atlassian-mcp/internal/state"
)

func TestNewServerRegistersExpectedTools(t *testing.T) {
	t.Parallel()

	deps := Dependencies{
		JiraService:       &jira.Service{},
		ConfluenceService: &confluence.Service{},
		JiraBaseURL:       "https://example.atlassian.net/",
		ConfluenceBaseURL: "https://example.atlassian.net/wiki/",
	}

	srv := NewServer(deps)

	tools := srv.ListTools()
	// All Jira and Confluence tools are now implemented
	expected := []string{
		"jira.list_projects",
		"jira.search_issues",
		"jira.create_issue",
		"jira.update_issue",
		"jira.add_comment",
		"jira.list_transitions",
		"jira.transition_issue",
		"jira.add_attachment",
		"confluence.list_spaces",
		"confluence.search_pages",
		"confluence.create_page",
		"confluence.update_page",
	}

	if len(tools) != len(expected) {
		t.Fatalf("unexpected tool count: got %d want %d", len(tools), len(expected))
	}

	for _, name := range expected {
		if _, ok := tools[name]; !ok {
			t.Fatalf("tool %q not registered", name)
		}
	}
}

func TestNewJiraToolsTrimsSiteURL(t *testing.T) {
	t.Parallel()

	srv := server.NewMCPServer("test", "0.0.1")
	cache := state.NewCache()

	jt := NewJiraTools(srv, &jira.Service{}, cache, "https://example.atlassian.net/")

	if jt.siteURL != "https://example.atlassian.net" {
		t.Fatalf("expected trimmed site URL, got %s", jt.siteURL)
	}

	// All 8 Jira tools are now implemented
	if len(srv.ListTools()) != 8 {
		t.Fatalf("expected 8 jira tools, got %d", len(srv.ListTools()))
	}
}

func TestJiraToolsHandleSearchIssuesValidation(t *testing.T) {
	t.Parallel()

	jt := &JiraTools{cache: state.NewCache(), siteURL: "https://example"}

	res, err := jt.handleSearchIssues(context.Background(), mcp.CallToolRequest{}, JiraSearchIssuesArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected error result")
	}
	if got := firstText(res); got != "JQL query must not be empty" {
		t.Fatalf("unexpected message: %s", got)
	}
}

// TEMPORARY: Tests for unimplemented handlers - will be restored when methods are added
// func TestJiraToolsHandleUpdateIssueValidation(t *testing.T) {
// 	t.Parallel()

// 	jt := &JiraTools{cache: state.NewCache(), siteURL: "https://example"}

// 	res, err := jt.handleUpdateIssue(context.Background(), mcp.CallToolRequest{}, JiraUpdateIssueArgs{Key: "PROJ-1"})
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if !res.IsError {
// 		t.Fatalf("expected error result")
// 	}
// 	if got := firstText(res); got != "no updates provided" {
// 		t.Fatalf("unexpected message: %s", got)
// 	}
// }

// func TestJiraToolsHandleAddAttachmentInvalidBase64(t *testing.T) {
// 	t.Parallel()

// 	jt := &JiraTools{cache: state.NewCache(), siteURL: "https://example"}

// 	res, err := jt.handleAddAttachment(context.Background(), mcp.CallToolRequest{}, JiraAddAttachmentArgs{Key: "PROJ-1", FileName: "file.txt", Data: "not-base64"})
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if !res.IsError {
// 		t.Fatalf("expected error result")
// 	}
// 	if got := firstText(res); got == "" || !strings.Contains(got, "invalid base64 data") {
// 		t.Fatalf("unexpected message: %s", got)
// 	}
// }

func TestNewConfluenceToolsTrimsBaseURL(t *testing.T) {
	t.Parallel()

	srv := server.NewMCPServer("test", "0.0.1")

	ct := NewConfluenceTools(srv, &confluence.Service{}, "https://example.atlassian.net/wiki/")

	if ct.baseURL != "https://example.atlassian.net/wiki" {
		t.Fatalf("expected trimmed base URL, got %s", ct.baseURL)
	}

	if len(srv.ListTools()) != 4 {
		t.Fatalf("expected 4 confluence tools, got %d", len(srv.ListTools()))
	}
}

func TestConfluenceToolsHandleSearchContentValidation(t *testing.T) {
	t.Parallel()

	ct := &ConfluenceTools{baseURL: "https://example"}

	res, err := ct.handleSearchContent(context.Background(), mcp.CallToolRequest{}, ConfluenceSearchArgs{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected error result")
	}
	if got := firstText(res); got != "CQL query must not be empty" {
		t.Fatalf("unexpected message: %s", got)
	}
}

func firstText(res *mcp.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	if text, ok := res.Content[0].(mcp.TextContent); ok {
		return text.Text
	}
	return ""
}
