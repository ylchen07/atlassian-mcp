package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/ylchen07/atlassian-mcp/internal/confluence"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ConfluenceTools wires Confluence services into MCP tools.
type ConfluenceTools struct {
	service *confluence.Service
	baseURL string
}

// NewConfluenceTools registers Confluence tools on the server.
func NewConfluenceTools(s *server.MCPServer, service *confluence.Service, baseURL string) *ConfluenceTools {
	ct := &ConfluenceTools{
		service: service,
		baseURL: strings.TrimRight(baseURL, "/"),
	}

	s.AddTool(
		mcp.NewTool(
			"confluence.list_spaces",
			mcp.WithDescription("List Confluence spaces accessible to the configured account"),
			mcp.WithInputSchema[ConfluenceListSpacesArgs](),
			mcp.WithOutputSchema[ConfluenceSpacesResult](),
		),
		mcp.NewTypedToolHandler(ct.handleListSpaces),
	)

	s.AddTool(
		mcp.NewTool(
			"confluence.search_pages",
			mcp.WithDescription("Search Confluence content using CQL"),
			mcp.WithInputSchema[ConfluenceSearchArgs](),
			mcp.WithOutputSchema[ConfluenceSearchResult](),
		),
		mcp.NewTypedToolHandler(ct.handleSearchContent),
	)

	s.AddTool(
		mcp.NewTool(
			"confluence.create_page",
			mcp.WithDescription("Create a Confluence page in the specified space"),
			mcp.WithInputSchema[ConfluencePageArgs](),
			mcp.WithOutputSchema[ConfluencePageResult](),
		),
		mcp.NewTypedToolHandler(ct.handleCreatePage),
	)

	s.AddTool(
		mcp.NewTool(
			"confluence.update_page",
			mcp.WithDescription("Update an existing Confluence page"),
			mcp.WithInputSchema[ConfluenceUpdateArgs](),
			mcp.WithOutputSchema[ConfluencePageResult](),
		),
		mcp.NewTypedToolHandler(ct.handleUpdatePage),
	)

	return ct
}

// ConfluenceListSpacesArgs parameters for list spaces.
type ConfluenceListSpacesArgs struct {
	Limit int `json:"limit,omitempty" jsonschema_description:"Maximum spaces to return" jsonschema:"minimum=1,maximum=100"`
}

// ConfluenceSpace models the response for spaces.
type ConfluenceSpace struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// ConfluenceSpacesResult wraps the list response.
type ConfluenceSpacesResult struct {
	Spaces []ConfluenceSpace `json:"spaces"`
}

func (c *ConfluenceTools) handleListSpaces(ctx context.Context, _ mcp.CallToolRequest, args ConfluenceListSpacesArgs) (*mcp.CallToolResult, error) {
	limit := args.Limit
	if limit == 0 {
		limit = 25
	}

	spaces, err := c.service.ListSpaces(ctx, limit)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("confluence list spaces failed", err), nil
	}

	result := ConfluenceSpacesResult{Spaces: make([]ConfluenceSpace, 0, len(spaces))}
	for _, space := range spaces {
		description := strings.TrimSpace(space.Description.Plain.Value)
		result.Spaces = append(result.Spaces, ConfluenceSpace{
			ID:          space.ID,
			Key:         space.Key,
			Name:        space.Name,
			Description: description,
			URL:         fmt.Sprintf("%s/spaces/%s", c.baseURL, space.Key),
		})
	}

	fallback := fmt.Sprintf("Found %d Confluence spaces", len(result.Spaces))
	return mcp.NewToolResultStructured(result, fallback), nil
}

// ConfluenceSearchArgs parameters for CQL search.
type ConfluenceSearchArgs struct {
	CQL   string `json:"cql" jsonschema:"required" jsonschema_description:"CQL query"`
	Limit int    `json:"limit,omitempty" jsonschema_description:"Maximum results to return" jsonschema:"minimum=1,maximum=100"`
}

// ConfluencePageSummary summarises content results.
type ConfluencePageSummary struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Version int    `json:"version"`
	URL     string `json:"url"`
}

// ConfluenceSearchResult search response payload.
type ConfluenceSearchResult struct {
	Results []ConfluencePageSummary `json:"results"`
}

func (c *ConfluenceTools) handleSearchContent(ctx context.Context, _ mcp.CallToolRequest, args ConfluenceSearchArgs) (*mcp.CallToolResult, error) {
	if strings.TrimSpace(args.CQL) == "" {
		return mcp.NewToolResultError("CQL query must not be empty"), nil
	}

	limit := args.Limit
	if limit == 0 {
		limit = 25
	}

	results, err := c.service.SearchContent(ctx, args.CQL, limit)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("confluence search failed", err), nil
	}

	payload := ConfluenceSearchResult{Results: make([]ConfluencePageSummary, 0, len(results))}
	for _, content := range results {
		payload.Results = append(payload.Results, ConfluencePageSummary{
			ID:      content.ID,
			Title:   content.Title,
			Type:    content.Type,
			Status:  content.Status,
			Version: content.Version.Number,
			URL:     fmt.Sprintf("%s/pages/%s", c.baseURL, content.ID),
		})
	}

	fallback := fmt.Sprintf("Found %d Confluence results", len(payload.Results))
	return mcp.NewToolResultStructured(payload, fallback), nil
}

// ConfluencePageArgs parameters for page creation.
type ConfluencePageArgs struct {
	SpaceKey string `json:"spaceKey" jsonschema:"required" jsonschema_description:"Space key"`
	Title    string `json:"title" jsonschema:"required" jsonschema_description:"Page title"`
	Body     string `json:"body" jsonschema:"required" jsonschema_description:"Page body in storage format"`
	ParentID string `json:"parentId,omitempty" jsonschema_description:"Ancestor page ID"`
}

// ConfluenceUpdateArgs parameters for page update.
type ConfluenceUpdateArgs struct {
	ID       string `json:"id" jsonschema:"required" jsonschema_description:"Page ID"`
	SpaceKey string `json:"spaceKey,omitempty" jsonschema_description:"Space key"`
	Title    string `json:"title" jsonschema:"required" jsonschema_description:"Page title"`
	Body     string `json:"body" jsonschema:"required" jsonschema_description:"Page body in storage format"`
	ParentID string `json:"parentId,omitempty" jsonschema_description:"Ancestor page ID"`
	Version  int    `json:"version" jsonschema:"required" jsonschema_description:"Next version number"`
}

// ConfluencePageResult response for create/update.
type ConfluencePageResult struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Version int    `json:"version"`
	URL     string `json:"url"`
}

func (c *ConfluenceTools) handleCreatePage(ctx context.Context, _ mcp.CallToolRequest, args ConfluencePageArgs) (*mcp.CallToolResult, error) {
	created, err := c.service.CreatePage(ctx, confluence.PageInput{
		SpaceKey: args.SpaceKey,
		Title:    args.Title,
		Body:     args.Body,
		ParentID: args.ParentID,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("confluence create page failed", err), nil
	}

	result := ConfluencePageResult{
		ID:      created.ID,
		Title:   created.Title,
		Version: created.Version.Number,
		URL:     fmt.Sprintf("%s/pages/%s", c.baseURL, created.ID),
	}

	fallback := fmt.Sprintf("Created Confluence page %s", created.Title)
	return mcp.NewToolResultStructured(result, fallback), nil
}

func (c *ConfluenceTools) handleUpdatePage(ctx context.Context, _ mcp.CallToolRequest, args ConfluenceUpdateArgs) (*mcp.CallToolResult, error) {
	updated, err := c.service.UpdatePage(ctx, args.ID, confluence.PageInput{
		SpaceKey: args.SpaceKey,
		Title:    args.Title,
		Body:     args.Body,
		ParentID: args.ParentID,
		Version:  args.Version,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("confluence update page failed", err), nil
	}

	result := ConfluencePageResult{
		ID:      updated.ID,
		Title:   updated.Title,
		Version: updated.Version.Number,
		URL:     fmt.Sprintf("%s/pages/%s", c.baseURL, updated.ID),
	}

	fallback := fmt.Sprintf("Updated Confluence page %s", updated.Title)
	return mcp.NewToolResultStructured(result, fallback), nil
}
