package jira

import (
	"context"
	"net/url"
	"strconv"
)

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
