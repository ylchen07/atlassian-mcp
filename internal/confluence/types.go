package confluence

import (
	"encoding/json"
)

// Space represents a Confluence space summary.
// Note: Space IDs are numeric in Confluence Data Center/Server.
type Space struct {
	ID          json.Number `json:"id"`
	Key         string      `json:"key"`
	Name        string      `json:"name"`
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

// PageInput describes a page create/update request.
type PageInput struct {
	SpaceKey string
	Title    string
	Body     string
	ParentID string
	Version  int
}
