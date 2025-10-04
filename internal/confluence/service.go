package confluence

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ylchen07/atlassian-mcp/internal/atlassian"
)

// Service exposes Confluence REST endpoints used by the MCP server.
type Service struct {
	client *atlassian.Client
}

// NewService constructs a Confluence service.
func NewService(client *atlassian.Client) *Service {
	return &Service{client: client}
}

// Space represents a Confluence space summary.
type Space struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description struct {
		Plain struct {
			Value string `json:"value"`
		} `json:"plain"`
	} `json:"description"`
}

// Content represents Confluence content (pages, blog posts).
type Content struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Title   string `json:"title"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"storage"`
	} `json:"body"`
}

// ListSpaces retrieves Confluence spaces.
func (s *Service) ListSpaces(ctx context.Context, limit int) ([]Space, error) {
	query := map[string]string{
		"limit": fmt.Sprintf("%d", limit),
	}

	req, err := s.client.NewRequest(ctx, "GET", "/space", query, nil)
	if err != nil {
		return nil, err
	}

	var res struct {
		Results []Space `json:"results"`
	}

	if err := s.client.Do(req, &res); err != nil {
		return nil, err
	}

	return res.Results, nil
}

// SearchContent performs a CQL search across content.
func (s *Service) SearchContent(ctx context.Context, cql string, limit int) ([]Content, error) {
	if cql == "" {
		return nil, fmt.Errorf("confluence: cql required")
	}

	query := map[string]string{
		"cql":    cql,
		"limit":  fmt.Sprintf("%d", limit),
		"expand": "body.storage,version",
	}

	req, err := s.client.NewRequest(ctx, "GET", "/content/search", query, nil)
	if err != nil {
		return nil, err
	}

	var res struct {
		Results []Content `json:"results"`
	}

	if err := s.client.Do(req, &res); err != nil {
		return nil, err
	}

	return res.Results, nil
}

// PageInput describes a page create/update request.
type PageInput struct {
	SpaceKey string
	Title    string
	Body     string
	ParentID string
	Version  int
}

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

	body := map[string]any{
		"type":  "page",
		"title": in.Title,
		"space": map[string]string{"key": in.SpaceKey},
		"body": map[string]any{
			"storage": map[string]string{
				"value":          in.Body,
				"representation": "storage",
			},
		},
	}

	if in.ParentID != "" {
		body["ancestors"] = []map[string]string{{"id": in.ParentID}}
	}

	req, err := s.client.NewRequest(ctx, "POST", "/content", nil, body)
	if err != nil {
		return nil, err
	}

	var out Content
	if err := s.client.Do(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
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

	body := map[string]any{
		"id":    id,
		"type":  "page",
		"title": in.Title,
		"version": map[string]any{
			"number": in.Version,
		},
		"body": map[string]any{
			"storage": map[string]string{
				"value":          in.Body,
				"representation": "storage",
			},
		},
	}

	if in.SpaceKey != "" {
		body["space"] = map[string]string{"key": in.SpaceKey}
	}

	if in.ParentID != "" {
		body["ancestors"] = []map[string]string{{"id": in.ParentID}}
	}

	path := fmt.Sprintf("/content/%s", url.PathEscape(id))

	req, err := s.client.NewRequest(ctx, "PUT", path, nil, body)
	if err != nil {
		return nil, err
	}

	var out Content
	if err := s.client.Do(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
