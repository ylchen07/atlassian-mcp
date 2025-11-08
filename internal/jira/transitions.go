package jira

import (
	"context"
	"fmt"
	"net/url"
)

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
