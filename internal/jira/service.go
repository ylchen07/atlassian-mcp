package jira

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	jiraapi "github.com/ctreminiom/go-atlassian/v2/jira/v2"
)

const apiPrefix = "/rest/api/2"

// Service exposes Jira REST endpoints used by the MCP server.
type Service struct {
	client *jiraapi.Client
}

// NewService creates a Jira service using the provided go-atlassian client.
func NewService(client *jiraapi.Client) *Service {
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
	params.Set("expand", "lead")
	if maxResults > 0 {
		params.Set("maxResults", strconv.Itoa(maxResults))
	}

	path := apiPath("project/search")
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, "", nil)
	if err != nil {
		return nil, err
	}

	var res struct {
		Values []Project `json:"values"`
	}
	if _, err := s.client.Call(req, &res); err != nil {
		return nil, err
	}

	return res.Values, nil
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

	req, err := s.client.NewRequest(ctx, http.MethodPost, apiPath("search"), "", body)
	if err != nil {
		return nil, err
	}

	var out SearchResult
	if _, err := s.client.Call(req, &out); err != nil {
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

	req, err := s.client.NewRequest(ctx, http.MethodPost, apiPath("issue"), "", body)
	if err != nil {
		return nil, err
	}

	var created Issue
	if _, err := s.client.Call(req, &created); err != nil {
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

	req, err := s.client.NewRequest(ctx, http.MethodPut, path, "", body)
	if err != nil {
		return err
	}

	_, err = s.client.Call(req, nil)
	return err
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

	req, err := s.client.NewRequest(ctx, http.MethodPost, path, "", body)
	if err != nil {
		return err
	}

	_, err = s.client.Call(req, nil)
	return err
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

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, "", nil)
	if err != nil {
		return nil, err
	}

	var out struct {
		Transitions []Transition `json:"transitions"`
	}

	if _, err := s.client.Call(req, &out); err != nil {
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
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, "", body)
	if err != nil {
		return err
	}

	_, err = s.client.Call(req, nil)
	return err
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

	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("jira: create attachment part: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("jira: write attachment: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("jira: close attachment writer: %w", err)
	}

	path := apiPath("issue", url.PathEscape(key), "attachments")
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, writer.FormDataContentType(), buf)
	if err != nil {
		return err
	}

	_, err = s.client.Call(req, nil)
	return err
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
