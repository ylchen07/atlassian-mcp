package confluence

import (
	"context"
	"fmt"
	"strconv"

	cf "github.com/ctreminiom/go-atlassian/v2/confluence"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

// Service exposes Confluence REST endpoints used by the MCP server.
type Service struct {
	client *cf.Client
}

// NewService constructs a Confluence service.
func NewService(client *cf.Client) *Service {
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

	options := &models.GetSpacesOptionScheme{
		Expand: []string{"description.plain"},
	}

	page, _, err := s.client.Space.Gets(ctx, options, 0, limit)
	if err != nil {
		return nil, err
	}

	out := make([]Space, 0, len(page.Results))
	for _, space := range page.Results {
		if space == nil {
			continue
		}
		out = append(out, Space{
			ID:   strconv.Itoa(space.ID),
			Key:  space.Key,
			Name: space.Name,
			Description: struct {
				Plain struct {
					Value string `json:"value"`
				} `json:"plain"`
			}{
				Plain: struct {
					Value string `json:"value"`
				}{Value: ""},
			},
		})
	}

	return out, nil
}

// SearchContent performs a CQL search across content.
func (s *Service) SearchContent(ctx context.Context, cql string, limit int) ([]Content, error) {
	if cql == "" {
		return nil, fmt.Errorf("confluence: cql required")
	}

	if limit <= 0 {
		limit = 25
	}

	page, _, err := s.client.Content.Search(ctx, cql, "", []string{"body.storage", "version"}, "", limit)
	if err != nil {
		return nil, err
	}

	results := make([]Content, 0, len(page.Results))
	for _, item := range page.Results {
		if item == nil {
			continue
		}
		content := Content{
			ID:     item.ID,
			Type:   item.Type,
			Status: item.Status,
			Title:  item.Title,
		}
		if item.Version != nil {
			content.Version.Number = item.Version.Number
		}
		if item.Body != nil && item.Body.Storage != nil {
			content.Body.Storage.Value = item.Body.Storage.Value
			content.Body.Storage.Representation = item.Body.Storage.Representation
		}
		results = append(results, content)
	}

	return results, nil
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

	payload := &models.ContentScheme{
		Type:  "page",
		Title: in.Title,
		Space: &models.SpaceScheme{Key: in.SpaceKey},
		Body: &models.BodyScheme{
			Storage: &models.BodyNodeScheme{
				Value:          in.Body,
				Representation: "storage",
			},
		},
	}

	if in.ParentID != "" {
		payload.Ancestors = []*models.ContentScheme{{ID: in.ParentID}}
	}

	created, _, err := s.client.Content.Create(ctx, payload)
	if err != nil {
		return nil, err
	}

	return toContent(created), nil
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

	payload := &models.ContentScheme{
		Type:  "page",
		Title: in.Title,
		Body: &models.BodyScheme{
			Storage: &models.BodyNodeScheme{
				Value:          in.Body,
				Representation: "storage",
			},
		},
		Version: &models.ContentVersionScheme{Number: in.Version},
	}

	if in.SpaceKey != "" {
		payload.Space = &models.SpaceScheme{Key: in.SpaceKey}
	}

	if in.ParentID != "" {
		payload.Ancestors = []*models.ContentScheme{{ID: in.ParentID}}
	}

	updated, _, err := s.client.Content.Update(ctx, id, payload)
	if err != nil {
		return nil, err
	}

	return toContent(updated), nil
}

func toContent(in *models.ContentScheme) *Content {
	if in == nil {
		return nil
	}

	var number int
	if in.Version != nil {
		number = in.Version.Number
	}

	var value, representation string
	if in.Body != nil && in.Body.Storage != nil {
		value = in.Body.Storage.Value
		representation = in.Body.Storage.Representation
	}

	out := &Content{
		ID:     in.ID,
		Type:   in.Type,
		Status: in.Status,
		Title:  in.Title,
		Version: struct {
			Number int `json:"number"`
		}{Number: number},
	}

	out.Body.Storage.Value = value
	out.Body.Storage.Representation = representation

	return out
}
