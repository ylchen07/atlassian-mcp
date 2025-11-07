package jira

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/ylchen07/atlassian-mcp/internal/atlassian"
)

const apiPrefix = "/rest/api/latest"

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

// TEMPORARY: // IssueInput represents fields for creating a new issue.
// TEMPORARY: type IssueInput struct {
// TEMPORARY: 	ProjectKey  string
// TEMPORARY: 	Summary     string
// TEMPORARY: 	IssueType   string
// TEMPORARY: 	Description any
// TEMPORARY: 	Fields      map[string]any
// TEMPORARY: }
// TEMPORARY: 
// TEMPORARY: // CreateIssue creates a new Jira issue and returns the created resource.
// TEMPORARY: func (s *Service) CreateIssue(ctx context.Context, input IssueInput) (*Issue, error) {
// TEMPORARY: 	if input.ProjectKey == "" {
// TEMPORARY: 		return nil, fmt.Errorf("jira: project key required")
// TEMPORARY: 	}
// TEMPORARY: 	if input.Summary == "" {
// TEMPORARY: 		return nil, fmt.Errorf("jira: summary required")
// TEMPORARY: 	}
// TEMPORARY: 	if input.IssueType == "" {
// TEMPORARY: 		return nil, fmt.Errorf("jira: issue type required")
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	fields := map[string]any{
// TEMPORARY: 		"project":   map[string]string{"key": input.ProjectKey},
// TEMPORARY: 		"summary":   input.Summary,
// TEMPORARY: 		"issuetype": map[string]string{"name": input.IssueType},
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	if input.Description != nil {
// TEMPORARY: 		fields["description"] = input.Description
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	for k, v := range input.Fields {
// TEMPORARY: 		fields[k] = v
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	body := map[string]any{"fields": fields}
// TEMPORARY: 
// TEMPORARY: 	req, err := s.client.NewRequest(ctx, http.MethodPost, apiPath("issue"), "", body)
// TEMPORARY: 	if err != nil {
// TEMPORARY: 		return nil, err
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	var created Issue
// TEMPORARY: 	if _, err := s.client.Call(req, &created); err != nil {
// TEMPORARY: 		return nil, err
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	return &created, nil
// TEMPORARY: }
// TEMPORARY: 
// TEMPORARY: // UpdateIssue updates the specified issue fields.
// TEMPORARY: func (s *Service) UpdateIssue(ctx context.Context, key string, fields map[string]any) error {
// TEMPORARY: 	if key == "" {
// TEMPORARY: 		return fmt.Errorf("jira: issue key required")
// TEMPORARY: 	}
// TEMPORARY: 	if len(fields) == 0 {
// TEMPORARY: 		return fmt.Errorf("jira: fields required")
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	body := map[string]any{"fields": fields}
// TEMPORARY: 	path := apiPath("issue", url.PathEscape(key))
// TEMPORARY: 
// TEMPORARY: 	req, err := s.client.NewRequest(ctx, http.MethodPut, path, "", body)
// TEMPORARY: 	if err != nil {
// TEMPORARY: 		return err
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	_, err = s.client.Call(req, nil)
// TEMPORARY: 	return err
// TEMPORARY: }
// TEMPORARY: 
// TEMPORARY: // AddComment appends a comment to the issue.
// TEMPORARY: func (s *Service) AddComment(ctx context.Context, key string, comment any) error {
// TEMPORARY: 	if key == "" {
// TEMPORARY: 		return fmt.Errorf("jira: issue key required")
// TEMPORARY: 	}
// TEMPORARY: 	if comment == nil {
// TEMPORARY: 		return fmt.Errorf("jira: comment body required")
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	body := map[string]any{"body": comment}
// TEMPORARY: 	path := apiPath("issue", url.PathEscape(key), "comment")
// TEMPORARY: 
// TEMPORARY: 	req, err := s.client.NewRequest(ctx, http.MethodPost, path, "", body)
// TEMPORARY: 	if err != nil {
// TEMPORARY: 		return err
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	_, err = s.client.Call(req, nil)
// TEMPORARY: 	return err
// TEMPORARY: }
// TEMPORARY: 
// TEMPORARY: // ListTransitions retrieves available workflow transitions for an issue.
// TEMPORARY: func (s *Service) ListTransitions(ctx context.Context, key string) ([]Transition, error) {
// TEMPORARY: 	if key == "" {
// TEMPORARY: 		return nil, fmt.Errorf("jira: issue key required")
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	params := url.Values{}
// TEMPORARY: 	params.Set("expand", "transitions.fields")
// TEMPORARY: 
// TEMPORARY: 	path := apiPath("issue", url.PathEscape(key), "transitions")
// TEMPORARY: 	if encoded := params.Encode(); encoded != "" {
// TEMPORARY: 		path += "?" + encoded
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	req, err := s.client.NewRequest(ctx, http.MethodGet, path, "", nil)
// TEMPORARY: 	if err != nil {
// TEMPORARY: 		return nil, err
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	var out struct {
// TEMPORARY: 		Transitions []Transition `json:"transitions"`
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	if _, err := s.client.Call(req, &out); err != nil {
// TEMPORARY: 		return nil, err
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	return out.Transitions, nil
// TEMPORARY: }
// TEMPORARY: 
// TEMPORARY: // TransitionIssue moves an issue through a workflow transition.
// TEMPORARY: func (s *Service) TransitionIssue(ctx context.Context, key, transitionID string, fields map[string]any) error {
// TEMPORARY: 	if key == "" {
// TEMPORARY: 		return fmt.Errorf("jira: issue key required")
// TEMPORARY: 	}
// TEMPORARY: 	if transitionID == "" {
// TEMPORARY: 		return fmt.Errorf("jira: transition id required")
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	body := map[string]any{
// TEMPORARY: 		"transition": map[string]string{"id": transitionID},
// TEMPORARY: 	}
// TEMPORARY: 	if len(fields) > 0 {
// TEMPORARY: 		body["fields"] = fields
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	path := apiPath("issue", url.PathEscape(key), "transitions")
// TEMPORARY: 	req, err := s.client.NewRequest(ctx, http.MethodPost, path, "", body)
// TEMPORARY: 	if err != nil {
// TEMPORARY: 		return err
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	_, err = s.client.Call(req, nil)
// TEMPORARY: 	return err
// TEMPORARY: }
// TEMPORARY: 
// TEMPORARY: // AddAttachment uploads a file attachment to the specified issue.
// TEMPORARY: func (s *Service) AddAttachment(ctx context.Context, key, filename string, data []byte) error {
// TEMPORARY: 	if key == "" {
// TEMPORARY: 		return fmt.Errorf("jira: issue key required")
// TEMPORARY: 	}
// TEMPORARY: 	if filename == "" {
// TEMPORARY: 		return fmt.Errorf("jira: attachment filename required")
// TEMPORARY: 	}
// TEMPORARY: 	if len(data) == 0 {
// TEMPORARY: 		return fmt.Errorf("jira: attachment data required")
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	buf := new(bytes.Buffer)
// TEMPORARY: 	writer := multipart.NewWriter(buf)
// TEMPORARY: 	part, err := writer.CreateFormFile("file", filename)
// TEMPORARY: 	if err != nil {
// TEMPORARY: 		return fmt.Errorf("jira: create attachment part: %w", err)
// TEMPORARY: 	}
// TEMPORARY: 	if _, err := part.Write(data); err != nil {
// TEMPORARY: 		return fmt.Errorf("jira: write attachment: %w", err)
// TEMPORARY: 	}
// TEMPORARY: 	if err := writer.Close(); err != nil {
// TEMPORARY: 		return fmt.Errorf("jira: close attachment writer: %w", err)
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	path := apiPath("issue", url.PathEscape(key), "attachments")
// TEMPORARY: 	req, err := s.client.NewRequest(ctx, http.MethodPost, path, writer.FormDataContentType(), buf)
// TEMPORARY: 	if err != nil {
// TEMPORARY: 		return err
// TEMPORARY: 	}
// TEMPORARY: 
// TEMPORARY: 	_, err = s.client.Call(req, nil)
// TEMPORARY: 	return err
// TEMPORARY: }
// TEMPORARY: 
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
