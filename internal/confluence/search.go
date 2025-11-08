package confluence

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

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
