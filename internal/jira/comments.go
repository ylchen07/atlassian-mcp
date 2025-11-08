package jira

import (
	"context"
	"fmt"
	"net/url"
)

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
