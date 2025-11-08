package jira

import (
	"context"
	"fmt"
	"net/url"
)

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
