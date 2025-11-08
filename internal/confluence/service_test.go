package confluence

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ylchen07/atlassian-mcp/internal/atlassian"
	"github.com/ylchen07/atlassian-mcp/internal/config"
)

// Create a real HTTP client with mock transport for testing
func newMockClient(t *testing.T, handler func(*http.Request) (*http.Response, error)) *atlassian.HTTPClient {
	t.Helper()

	client, err := atlassian.NewHTTPClient("https://example.com", config.ServiceCredentials{
		Email:    "test@example.com",
		APIToken: "token",
	})
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// Replace the underlying HTTP client with our mock transport
	client.HTTPClient = &http.Client{
		Transport: roundTripFunc(handler),
	}

	return client
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewService(t *testing.T) {
	t.Parallel()

	client := &atlassian.HTTPClient{BaseURL: "https://example.com"}
	service := NewService(client)

	if service == nil {
		t.Fatal("expected service to be created")
	}
}

func TestListSpaces(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "/rest/api/space") {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}

		response := struct {
			Results []Space `json:"results"`
		}{
			Results: []Space{
				{
					ID:   "1",
					Key:  "DEMO",
					Name: "Demo Space",
					Description: struct {
						Plain struct {
							Value string `json:"value"`
						} `json:"plain"`
					}{
						Plain: struct {
							Value string `json:"value"`
						}{Value: "Test description"},
					},
				},
			},
		}

		data, _ := json.Marshal(response)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(data)),
			Header:     make(http.Header),
		}, nil
	})

	service := NewService(client)
	spaces, err := service.ListSpaces(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListSpaces error: %v", err)
	}

	if len(spaces) != 1 {
		t.Fatalf("expected 1 space, got %d", len(spaces))
	}

	if spaces[0].Key != "DEMO" {
		t.Fatalf("expected space key DEMO, got %s", spaces[0].Key)
	}
}

func TestSearchContent(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "/rest/api/content/search") {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}

		response := struct {
			Results []Content `json:"results"`
		}{
			Results: []Content{
				{
					ID:     "12345",
					Type:   "page",
					Status: "current",
					Title:  "Test Page",
					Version: struct {
						Number int `json:"number"`
					}{Number: 1},
				},
			},
		}

		data, _ := json.Marshal(response)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(data)),
			Header:     make(http.Header),
		}, nil
	})

	service := NewService(client)
	pages, err := service.SearchContent(context.Background(), "type=page", 10)
	if err != nil {
		t.Fatalf("SearchContent error: %v", err)
	}

	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}

	if pages[0].Title != "Test Page" {
		t.Fatalf("expected title 'Test Page', got %s", pages[0].Title)
	}
}

func TestSearchContentValidation(t *testing.T) {
	t.Parallel()

	client := &atlassian.HTTPClient{BaseURL: "https://example.com"}
	service := NewService(client)

	_, err := service.SearchContent(context.Background(), "", 10)
	if err == nil {
		t.Fatal("expected error for empty CQL")
	}

	if !strings.Contains(err.Error(), "cql required") {
		t.Fatalf("expected 'cql required' error, got: %v", err)
	}
}

func TestCreatePage(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "POST" {
			t.Fatalf("expected POST, got %s", req.Method)
		}

		created := Content{
			ID:    "67890",
			Title: "New Page",
			Version: struct {
				Number int `json:"number"`
			}{Number: 1},
		}

		data, _ := json.Marshal(created)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(data)),
			Header:     make(http.Header),
		}, nil
	})

	service := NewService(client)
	page, err := service.CreatePage(context.Background(), PageInput{
		SpaceKey: "DEMO",
		Title:    "New Page",
		Body:     "<p>Test content</p>",
	})

	if err != nil {
		t.Fatalf("CreatePage error: %v", err)
	}

	if page.Title != "New Page" {
		t.Fatalf("expected title 'New Page', got %s", page.Title)
	}
}

func TestCreatePageValidation(t *testing.T) {
	t.Parallel()

	client := &atlassian.HTTPClient{BaseURL: "https://example.com"}
	service := NewService(client)

	tests := []struct {
		name   string
		input  PageInput
		errMsg string
	}{
		{
			name:   "missing space key",
			input:  PageInput{Title: "Test", Body: "body"},
			errMsg: "space key required",
		},
		{
			name:   "missing title",
			input:  PageInput{SpaceKey: "DEMO", Body: "body"},
			errMsg: "title required",
		},
		{
			name:   "missing body",
			input:  PageInput{SpaceKey: "DEMO", Title: "Test"},
			errMsg: "body required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreatePage(context.Background(), tt.input)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Fatalf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestUpdatePage(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "PUT" {
			t.Fatalf("expected PUT, got %s", req.Method)
		}

		updated := Content{
			ID:    "12345",
			Title: "Updated Page",
			Version: struct {
				Number int `json:"number"`
			}{Number: 2},
		}

		data, _ := json.Marshal(updated)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(data)),
			Header:     make(http.Header),
		}, nil
	})

	service := NewService(client)
	page, err := service.UpdatePage(context.Background(), "12345", PageInput{
		Title:   "Updated Page",
		Body:    "<p>Updated content</p>",
		Version: 2,
	})

	if err != nil {
		t.Fatalf("UpdatePage error: %v", err)
	}

	if page.Version.Number != 2 {
		t.Fatalf("expected version 2, got %d", page.Version.Number)
	}
}

func TestUpdatePageValidation(t *testing.T) {
	t.Parallel()

	client := &atlassian.HTTPClient{BaseURL: "https://example.com"}
	service := NewService(client)

	tests := []struct {
		name   string
		id     string
		input  PageInput
		errMsg string
	}{
		{
			name:   "missing page ID",
			id:     "",
			input:  PageInput{Title: "Test", Body: "body", Version: 2},
			errMsg: "page id required",
		},
		{
			name:   "missing version",
			id:     "123",
			input:  PageInput{Title: "Test", Body: "body"},
			errMsg: "version required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.UpdatePage(context.Background(), tt.id, tt.input)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Fatalf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestAPIPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{
			name:     "single part",
			parts:    []string{"space"},
			expected: "/rest/api/space",
		},
		{
			name:     "multiple parts",
			parts:    []string{"content", "12345"},
			expected: "/rest/api/content/12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := apiPath(tt.parts...)
			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
