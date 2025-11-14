package confluence

import (
	"context"
	"fmt"
	"strings"
)

// CreatePage creates a Confluence page.
func (s *Service) CreatePage(ctx context.Context, in PageInput) (*Content, error) {
	if in.SpaceKey == "" {
		return nil, fmt.Errorf("confluence: space key required")
	}
	if in.Title == "" {
		return nil, fmt.Errorf("confluence: title required")
	}
	if in.Body == "" {
		return nil, fmt.Errorf("confluence: body required")
	}

	payload := map[string]interface{}{
		"type":  "page",
		"title": in.Title,
		"space": map[string]string{
			"key": in.SpaceKey,
		},
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          in.Body,
				"representation": "storage",
			},
		},
	}

	if in.ParentID != "" {
		payload["ancestors"] = []map[string]string{
			{"id": in.ParentID},
		}
	}

	var created Content
	if err := s.client.Post(ctx, apiPath("content"), payload, &created); err != nil {
		return nil, err
	}

	return &created, nil
}

// UpdatePage updates an existing Confluence page.
func (s *Service) UpdatePage(ctx context.Context, id string, in PageInput) (*Content, error) {
	if id == "" {
		return nil, fmt.Errorf("confluence: page id required")
	}
	if in.Title == "" {
		return nil, fmt.Errorf("confluence: title required")
	}
	if in.Body == "" {
		return nil, fmt.Errorf("confluence: body required")
	}
	if in.Version == 0 {
		return nil, fmt.Errorf("confluence: version required")
	}

	payload := map[string]interface{}{
		"type":  "page",
		"title": in.Title,
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          in.Body,
				"representation": "storage",
			},
		},
		"version": map[string]int{
			"number": in.Version,
		},
	}

	if in.SpaceKey != "" {
		payload["space"] = map[string]string{
			"key": in.SpaceKey,
		}
	}

	if in.ParentID != "" {
		payload["ancestors"] = []map[string]string{
			{"id": in.ParentID},
		}
	}

	var updated Content
	if err := s.client.Put(ctx, apiPath("content", id), payload, &updated); err != nil {
		return nil, err
	}

	return &updated, nil
}

// GetPage retrieves a Confluence page by ID with full content.
func (s *Service) GetPage(ctx context.Context, id string, expand []string) (*Content, error) {
	if id == "" {
		return nil, fmt.Errorf("confluence: page id required")
	}

	// Build query parameters for expand
	path := apiPath("content", id)
	if len(expand) > 0 {
		path += "?expand=" + strings.Join(expand, ",")
	}

	var page Content
	if err := s.client.Get(ctx, path, &page); err != nil {
		return nil, err
	}

	return &page, nil
}
