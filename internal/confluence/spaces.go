package confluence

import (
	"context"
	"net/url"
	"strconv"
)

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
