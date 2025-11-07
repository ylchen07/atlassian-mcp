package confluence

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ylchen07/atlassian-mcp/internal/atlassian"
)

const apiPrefix = "/rest/api"

// Service exposes Confluence REST endpoints used by the MCP server.
type Service struct {
	client *atlassian.HTTPClient
}

// NewService constructs a Confluence service.
func NewService(client *atlassian.HTTPClient) *Service {
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
	if limit <= 0 {
		limit = 25
	}

	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("expand", "description.plain")

	path := apiPath("space")
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var response struct {
		Results []Space `json:"results"`
	}

	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// SearchContent performs a CQL search across content.
func (s *Service) SearchContent(ctx context.Context, cql string, limit int) ([]Content, error) {
	if cql == "" {
		return nil, fmt.Errorf("confluence: cql required")
	}

	if limit <= 0 {
		limit = 25
	}

	params := url.Values{}
	params.Set("cql", cql)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("expand", "body.storage,version")

	path := apiPath("content/search")
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var response struct {
		Results []Content `json:"results"`
	}

	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
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
