package jira

import (
	"context"
	"fmt"
	"net/url"

	"gitlab.com/your-org/jira-mcp/internal/atlassian"
)

// Service exposes Jira REST endpoints used by the MCP server.
type Service struct {
	client *atlassian.Client
}

// NewService creates a Jira service using the provided Atlassian client.
func NewService(client *atlassian.Client) *Service {
	return &Service{client: client}
}

// Project represents a simplified Jira project.
type Project struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Issue represents a simplified Jira issue payload.
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

// IssueFields reflect the subset of issue fields we surface.
type IssueFields struct {
	Summary     string `json:"summary"`
	Description any    `json:"description"`
	Status      struct {
		Name string `json:"name"`
	} `json:"status"`
	Assignee struct {
		DisplayName string `json:"displayName"`
		AccountID   string `json:"accountId"`
	} `json:"assignee"`
}

// SearchRequest defines parameters for JQL searches.
type SearchRequest struct {
	JQL        string
	StartAt    int
	MaxResults int
	Fields     []string
	Expand     []string
}

// SearchResult represents the Jira search response.
type SearchResult struct {
	Total     int     `json:"total"`
	Issues    []Issue `json:"issues"`
	StartAt   int     `json:"startAt"`
	MaxResult int     `json:"maxResults"`
}

// ListProjects returns the accessible projects.
func (s *Service) ListProjects(ctx context.Context, maxResults int) ([]Project, error) {
	query := map[string]string{
		"expand":     "lead",
		"maxResults": fmt.Sprintf("%d", maxResults),
	}

	req, err := s.client.NewRequest(ctx, "GET", "/project/search", query, nil)
	if err != nil {
		return nil, err
	}

	var res struct {
		Values []Project `json:"values"`
	}

	if err := s.client.Do(req, &res); err != nil {
		return nil, err
	}

	return res.Values, nil
}

// SearchIssues executes a JQL search.
func (s *Service) SearchIssues(ctx context.Context, req SearchRequest) (*SearchResult, error) {
	body := map[string]any{
		"jql": req.JQL,
	}

	if req.StartAt > 0 {
		body["startAt"] = req.StartAt
	}

	if req.MaxResults > 0 {
		body["maxResults"] = req.MaxResults
	}

	if len(req.Fields) > 0 {
		body["fields"] = req.Fields
	}

	if len(req.Expand) > 0 {
		body["expand"] = req.Expand
	}

	httpReq, err := s.client.NewRequest(ctx, "POST", "/search", nil, body)
	if err != nil {
		return nil, err
	}

	var out SearchResult
	if err := s.client.Do(httpReq, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// IssueInput represents fields for creating a new issue.
type IssueInput struct {
	ProjectKey  string
	Summary     string
	IssueType   string
	Description any
	Fields      map[string]any
}

// CreateIssue creates a new Jira issue and returns the created resource.
func (s *Service) CreateIssue(ctx context.Context, input IssueInput) (*Issue, error) {
	if input.ProjectKey == "" {
		return nil, fmt.Errorf("jira: project key required")
	}
	if input.Summary == "" {
		return nil, fmt.Errorf("jira: summary required")
	}
	if input.IssueType == "" {
		return nil, fmt.Errorf("jira: issue type required")
	}

	fields := map[string]any{
		"project":   map[string]string{"key": input.ProjectKey},
		"summary":   input.Summary,
		"issuetype": map[string]string{"name": input.IssueType},
	}

	if input.Description != nil {
		fields["description"] = input.Description
	}

	for k, v := range input.Fields {
		fields[k] = v
	}

	body := map[string]any{"fields": fields}

	req, err := s.client.NewRequest(ctx, "POST", "/issue", nil, body)
	if err != nil {
		return nil, err
	}

	var created Issue
	if err := s.client.Do(req, &created); err != nil {
		return nil, err
	}

	return &created, nil
}

// UpdateIssue updates the specified issue fields.
func (s *Service) UpdateIssue(ctx context.Context, key string, fields map[string]any) error {
	if key == "" {
		return fmt.Errorf("jira: issue key required")
	}
	if len(fields) == 0 {
		return fmt.Errorf("jira: fields required")
	}

	body := map[string]any{"fields": fields}
	path := fmt.Sprintf("/issue/%s", url.PathEscape(key))

	req, err := s.client.NewRequest(ctx, "PUT", path, nil, body)
	if err != nil {
		return err
	}

	return s.client.Do(req, nil)
}

// AddComment appends a comment to the issue.
func (s *Service) AddComment(ctx context.Context, key string, comment any) error {
	if key == "" {
		return fmt.Errorf("jira: issue key required")
	}
	if comment == nil {
		return fmt.Errorf("jira: comment body required")
	}

	body := map[string]any{"body": comment}
	path := fmt.Sprintf("/issue/%s/comment", url.PathEscape(key))

	req, err := s.client.NewRequest(ctx, "POST", path, nil, body)
	if err != nil {
		return err
	}

	return s.client.Do(req, nil)
}
