package jira

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ylchen07/atlassian-mcp/internal/atlassian"
)

const apiPrefix = "/rest/api/2"

// Service exposes Jira REST endpoints used by the MCP server.
type Service struct {
	client *atlassian.HTTPClient
}

// NewService creates a Jira service using the provided HTTP client.
func NewService(client *atlassian.HTTPClient) *Service {
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

// Transition represents a workflow transition available to an issue.
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"to"`
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
	params := url.Values{}
	if maxResults > 0 {
		params.Set("maxResults", strconv.Itoa(maxResults))
	}

	path := apiPath("project")
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var projects []Project
	if err := s.client.Get(ctx, path, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// SearchIssues executes a JQL search.
func (s *Service) SearchIssues(ctx context.Context, sr SearchRequest) (*SearchResult, error) {
	body := map[string]any{
		"jql": sr.JQL,
	}

	if sr.StartAt > 0 {
		body["startAt"] = sr.StartAt
	}

	if sr.MaxResults > 0 {
		body["maxResults"] = sr.MaxResults
	}

	if len(sr.Fields) > 0 {
		body["fields"] = sr.Fields
	}

	if len(sr.Expand) > 0 {
		body["expand"] = sr.Expand
	}

	var result SearchResult
	if err := s.client.Post(ctx, apiPath("search"), body, &result); err != nil {
		return nil, err
	}

	return &result, nil
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

	var created Issue
	if err := s.client.Post(ctx, apiPath("issue"), body, &created); err != nil {
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
	path := apiPath("issue", url.PathEscape(key))

	return s.client.Put(ctx, path, body, nil)
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
	path := apiPath("issue", url.PathEscape(key), "comment")

	return s.client.Post(ctx, path, body, nil)
}

// ListTransitions retrieves available workflow transitions for an issue.
func (s *Service) ListTransitions(ctx context.Context, key string) ([]Transition, error) {
	if key == "" {
		return nil, fmt.Errorf("jira: issue key required")
	}

	params := url.Values{}
	params.Set("expand", "transitions.fields")

	path := apiPath("issue", url.PathEscape(key), "transitions")
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var out struct {
		Transitions []Transition `json:"transitions"`
	}

	if err := s.client.Get(ctx, path, &out); err != nil {
		return nil, err
	}

	return out.Transitions, nil
}

// TransitionIssue moves an issue through a workflow transition.
func (s *Service) TransitionIssue(ctx context.Context, key, transitionID string, fields map[string]any) error {
	if key == "" {
		return fmt.Errorf("jira: issue key required")
	}
	if transitionID == "" {
		return fmt.Errorf("jira: transition id required")
	}

	body := map[string]any{
		"transition": map[string]string{"id": transitionID},
	}
	if len(fields) > 0 {
		body["fields"] = fields
	}

	path := apiPath("issue", url.PathEscape(key), "transitions")
	return s.client.Post(ctx, path, body, nil)
}

// AddAttachment uploads a file attachment to the specified issue.
func (s *Service) AddAttachment(ctx context.Context, key, filename string, data []byte) error {
	if key == "" {
		return fmt.Errorf("jira: issue key required")
	}
	if filename == "" {
		return fmt.Errorf("jira: attachment filename required")
	}
	if len(data) == 0 {
		return fmt.Errorf("jira: attachment data required")
	}

	// For multipart upload, we'll need to implement a specialized method in the HTTP client
	// For now, return an error indicating this needs implementation
	return fmt.Errorf("jira: attachment upload not yet implemented in HTTP client")
}

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
