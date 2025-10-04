package mcp

import (
	"context"
	"fmt"
	"strings"

	"gitlab.com/your-org/jira-mcp/internal/jira"
	"gitlab.com/your-org/jira-mcp/internal/state"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// JiraTools wires Jira services into MCP tools.
type JiraTools struct {
	service *jira.Service
	cache   *state.Cache
	siteURL string
}

// NewJiraTools registers Jira tools on the server.
func NewJiraTools(s *server.MCPServer, service *jira.Service, cache *state.Cache, siteURL string) *JiraTools {
	jt := &JiraTools{
		service: service,
		cache:   cache,
		siteURL: strings.TrimRight(siteURL, "/"),
	}

	s.AddTool(
		mcp.NewTool(
			"jira.list_projects",
			mcp.WithDescription("List available Jira projects accessible to the configured account"),
			mcp.WithInputSchema[JiraListProjectsArgs](),
			mcp.WithOutputSchema[JiraProjectListResult](),
		),
		mcp.NewTypedToolHandler(jt.handleListProjects),
	)

	s.AddTool(
		mcp.NewTool(
			"jira.search_issues",
			mcp.WithDescription("Execute a JQL search and return matching issues"),
			mcp.WithInputSchema[JiraSearchIssuesArgs](),
			mcp.WithOutputSchema[JiraSearchIssuesResult](),
		),
		mcp.NewTypedToolHandler(jt.handleSearchIssues),
	)

	s.AddTool(
		mcp.NewTool(
			"jira.create_issue",
			mcp.WithDescription("Create a new Jira issue in the specified project"),
			mcp.WithInputSchema[JiraCreateIssueArgs](),
			mcp.WithOutputSchema[JiraIssueResult](),
		),
		mcp.NewTypedToolHandler(jt.handleCreateIssue),
	)

	s.AddTool(
		mcp.NewTool(
			"jira.update_issue",
			mcp.WithDescription("Update fields on an existing Jira issue"),
			mcp.WithInputSchema[JiraUpdateIssueArgs](),
			mcp.WithOutputSchema[OperationStatus](),
		),
		mcp.NewTypedToolHandler(jt.handleUpdateIssue),
	)

	s.AddTool(
		mcp.NewTool(
			"jira.add_comment",
			mcp.WithDescription("Add a comment to an existing Jira issue"),
			mcp.WithInputSchema[JiraAddCommentArgs](),
			mcp.WithOutputSchema[OperationStatus](),
		),
		mcp.NewTypedToolHandler(jt.handleAddComment),
	)

	return jt
}

// JiraListProjectsArgs parameters for listing projects.
type JiraListProjectsArgs struct {
	MaxResults int `json:"maxResults,omitempty" jsonschema_description:"Maximum number of projects to fetch" jsonschema:"minimum=1,maximum=100"`
}

// JiraProject represents project metadata returned to clients.
type JiraProject struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// JiraProjectListResult wraps the project list response.
type JiraProjectListResult struct {
	Projects []JiraProject `json:"projects"`
}

// OperationStatus represents an acknowledgement response for state-changing operations.
type OperationStatus struct {
	Message string `json:"message"`
}

func (j *JiraTools) handleListProjects(ctx context.Context, _ mcp.CallToolRequest, args JiraListProjectsArgs) (*mcp.CallToolResult, error) {
	limit := args.MaxResults
	if limit == 0 {
		limit = 50
	}

	projects, err := j.service.ListProjects(ctx, limit)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("jira list projects failed", err), nil
	}

	result := JiraProjectListResult{Projects: make([]JiraProject, 0, len(projects))}
	for _, p := range projects {
		result.Projects = append(result.Projects, JiraProject{
			ID:   p.ID,
			Key:  p.Key,
			Name: p.Name,
			URL:  fmt.Sprintf("%s/browse/%s", j.siteURL, p.Key),
		})
	}

	j.cache.SetProjects(projects)

	fallback := fmt.Sprintf("Found %d Jira projects", len(result.Projects))
	return mcp.NewToolResultStructured(result, fallback), nil
}

// JiraSearchIssuesArgs parameters for JQL searches.
type JiraSearchIssuesArgs struct {
	JQL        string   `json:"jql" jsonschema:"required" jsonschema_description:"JQL query string"`
	MaxResults int      `json:"maxResults,omitempty" jsonschema_description:"Maximum number of issues to fetch" jsonschema:"minimum=1,maximum=100"`
	StartAt    int      `json:"startAt,omitempty" jsonschema_description:"Pagination offset" jsonschema:"minimum=0"`
	Fields     []string `json:"fields,omitempty" jsonschema_description:"Additional fields to include"`
}

// JiraIssueSummary summarises issue details.
type JiraIssueSummary struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Summary     string `json:"summary"`
	Status      string `json:"status"`
	Assignee    string `json:"assignee,omitempty"`
	Description any    `json:"description,omitempty"`
	URL         string `json:"url"`
}

// JiraSearchIssuesResult response payload.
type JiraSearchIssuesResult struct {
	Total     int                `json:"total"`
	StartAt   int                `json:"startAt"`
	MaxResult int                `json:"maxResults"`
	Issues    []JiraIssueSummary `json:"issues"`
}

func (j *JiraTools) handleSearchIssues(ctx context.Context, _ mcp.CallToolRequest, args JiraSearchIssuesArgs) (*mcp.CallToolResult, error) {
	if strings.TrimSpace(args.JQL) == "" {
		return mcp.NewToolResultError("JQL query must not be empty"), nil
	}

	req := jira.SearchRequest{
		JQL:        args.JQL,
		StartAt:    args.StartAt,
		MaxResults: args.MaxResults,
		Fields:     args.Fields,
	}

	result, err := j.service.SearchIssues(ctx, req)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("jira search issues failed", err), nil
	}

	response := JiraSearchIssuesResult{
		Total:     result.Total,
		StartAt:   result.StartAt,
		MaxResult: result.MaxResult,
		Issues:    make([]JiraIssueSummary, 0, len(result.Issues)),
	}

	for _, issue := range result.Issues {
		summary := JiraIssueSummary{
			ID:          issue.ID,
			Key:         issue.Key,
			Summary:     issue.Fields.Summary,
			Status:      issue.Fields.Status.Name,
			Description: issue.Fields.Description,
			URL:         fmt.Sprintf("%s/browse/%s", j.siteURL, issue.Key),
		}
		if issue.Fields.Assignee.DisplayName != "" {
			summary.Assignee = issue.Fields.Assignee.DisplayName
		}
		response.Issues = append(response.Issues, summary)
	}

	j.cache.SetLastJQL(args.JQL)

	fallback := fmt.Sprintf("Found %d/%d issues for JQL", len(response.Issues), response.Total)
	return mcp.NewToolResultStructured(response, fallback), nil
}

// JiraCreateIssueArgs define creation parameters.
type JiraCreateIssueArgs struct {
	ProjectKey  string         `json:"projectKey" jsonschema:"required" jsonschema_description:"Project key"`
	IssueType   string         `json:"issueType" jsonschema:"required" jsonschema_description:"Issue type name"`
	Summary     string         `json:"summary" jsonschema:"required" jsonschema_description:"Issue summary"`
	Description any            `json:"description,omitempty" jsonschema_description:"Issue description, plain text or Atlassian document"`
	Fields      map[string]any `json:"fields,omitempty" jsonschema_description:"Additional field overrides"`
}

// JiraIssueResult describes a single issue.
type JiraIssueResult struct {
	Key string `json:"key"`
	ID  string `json:"id"`
	URL string `json:"url"`
}

func (j *JiraTools) handleCreateIssue(ctx context.Context, _ mcp.CallToolRequest, args JiraCreateIssueArgs) (*mcp.CallToolResult, error) {
	created, err := j.service.CreateIssue(ctx, jira.IssueInput{
		ProjectKey:  args.ProjectKey,
		Summary:     args.Summary,
		IssueType:   args.IssueType,
		Description: args.Description,
		Fields:      args.Fields,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("jira create issue failed", err), nil
	}

	result := JiraIssueResult{
		Key: created.Key,
		ID:  created.ID,
		URL: fmt.Sprintf("%s/browse/%s", j.siteURL, created.Key),
	}

	fallback := fmt.Sprintf("Created Jira issue %s", result.Key)
	return mcp.NewToolResultStructured(result, fallback), nil
}

// JiraUpdateIssueArgs define fields for updates.
type JiraUpdateIssueArgs struct {
	Key         string         `json:"key" jsonschema:"required" jsonschema_description:"Issue key"`
	Summary     *string        `json:"summary,omitempty" jsonschema_description:"New summary"`
	Description any            `json:"description,omitempty" jsonschema_description:"New description"`
	Fields      map[string]any `json:"fields,omitempty" jsonschema_description:"Additional field updates"`
}

func (j *JiraTools) handleUpdateIssue(ctx context.Context, _ mcp.CallToolRequest, args JiraUpdateIssueArgs) (*mcp.CallToolResult, error) {
	updates := map[string]any{}
	if args.Fields != nil {
		for k, v := range args.Fields {
			updates[k] = v
		}
	}
	if args.Summary != nil {
		updates["summary"] = *args.Summary
	}
	if args.Description != nil {
		updates["description"] = args.Description
	}

	if len(updates) == 0 {
		return mcp.NewToolResultError("no updates provided"), nil
	}

	if err := j.service.UpdateIssue(ctx, args.Key, updates); err != nil {
		return mcp.NewToolResultErrorFromErr("jira update issue failed", err), nil
	}

	fallback := fmt.Sprintf("Updated Jira issue %s", args.Key)
	return mcp.NewToolResultStructured(OperationStatus{Message: fallback}, fallback), nil
}

// JiraAddCommentArgs parameters for commenting.
type JiraAddCommentArgs struct {
	Key  string `json:"key" jsonschema:"required" jsonschema_description:"Issue key"`
	Body any    `json:"body" jsonschema:"required" jsonschema_description:"Comment body as plain text or Atlassian document"`
}

func (j *JiraTools) handleAddComment(ctx context.Context, _ mcp.CallToolRequest, args JiraAddCommentArgs) (*mcp.CallToolResult, error) {
	if err := j.service.AddComment(ctx, args.Key, args.Body); err != nil {
		return mcp.NewToolResultErrorFromErr("jira add comment failed", err), nil
	}

	fallback := fmt.Sprintf("Added comment to Jira issue %s", args.Key)
	return mcp.NewToolResultStructured(OperationStatus{Message: fallback}, fallback), nil
}
