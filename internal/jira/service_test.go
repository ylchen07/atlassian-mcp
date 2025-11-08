package jira

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

func TestListProjects(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "/rest/api/2/project") {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}

		projects := []Project{
			{ID: "1", Key: "DEMO", Name: "Demo Project"},
			{ID: "2", Key: "TEST", Name: "Test Project"},
		}

		data, _ := json.Marshal(projects)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(data)),
			Header:     make(http.Header),
		}, nil
	})

	service := NewService(client)
	projects, err := service.ListProjects(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListProjects error: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	if projects[0].Key != "DEMO" {
		t.Fatalf("expected first project key DEMO, got %s", projects[0].Key)
	}
}

func TestSearchIssues(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "POST" {
			t.Fatalf("expected POST, got %s", req.Method)
		}

		if !strings.Contains(req.URL.Path, "/rest/api/2/search") {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}

		searchResult := SearchResult{
			Total:     1,
			StartAt:   0,
			MaxResult: 50,
			Issues: []Issue{
				{
					ID:  "1",
					Key: "DEMO-1",
					Fields: IssueFields{
						Summary: "Test issue",
						Status:  struct{ Name string `json:"name"` }{Name: "To Do"},
					},
				},
			},
		}

		data, _ := json.Marshal(searchResult)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(data)),
			Header:     make(http.Header),
		}, nil
	})

	service := NewService(client)
	result, err := service.SearchIssues(context.Background(), SearchRequest{
		JQL:        "project = DEMO",
		MaxResults: 50,
	})

	if err != nil {
		t.Fatalf("SearchIssues error: %v", err)
	}

	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}

	if result.Issues[0].Key != "DEMO-1" {
		t.Fatalf("expected issue key DEMO-1, got %s", result.Issues[0].Key)
	}
}

func TestCreateIssue(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "POST" {
			t.Fatalf("expected POST, got %s", req.Method)
		}

		created := Issue{
			ID:  "100",
			Key: "DEMO-10",
		}

		data, _ := json.Marshal(created)
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(bytes.NewReader(data)),
			Header:     make(http.Header),
		}, nil
	})

	service := NewService(client)
	issue, err := service.CreateIssue(context.Background(), IssueInput{
		ProjectKey:  "DEMO",
		Summary:     "New issue",
		IssueType:   "Task",
		Description: "Test description",
	})

	if err != nil {
		t.Fatalf("CreateIssue error: %v", err)
	}

	if issue.Key != "DEMO-10" {
		t.Fatalf("expected issue key DEMO-10, got %s", issue.Key)
	}
}

func TestUpdateIssue(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "PUT" {
			t.Fatalf("expected PUT, got %s", req.Method)
		}

		return &http.Response{
			StatusCode: 204,
			Body:       io.NopCloser(bytes.NewReader([]byte(""))),
			Header:     make(http.Header),
		}, nil
	})

	service := NewService(client)
	err := service.UpdateIssue(context.Background(), "DEMO-1", map[string]any{
		"summary": "Updated summary",
	})

	if err != nil {
		t.Fatalf("UpdateIssue error: %v", err)
	}
}

func TestAddComment(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "/comment") {
			t.Fatalf("expected /comment in path: %s", req.URL.Path)
		}

		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
			Header:     make(http.Header),
		}, nil
	})

	service := NewService(client)
	err := service.AddComment(context.Background(), "DEMO-1", "Test comment")

	if err != nil {
		t.Fatalf("AddComment error: %v", err)
	}
}

func TestListTransitions(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, func(req *http.Request) (*http.Response, error) {
		response := struct {
			Transitions []Transition `json:"transitions"`
		}{
			Transitions: []Transition{
				{ID: "1", Name: "In Progress", To: struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{ID: "3", Name: "In Progress"}},
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
	transitions, err := service.ListTransitions(context.Background(), "DEMO-1")

	if err != nil {
		t.Fatalf("ListTransitions error: %v", err)
	}

	if len(transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(transitions))
	}
}

func TestTransitionIssue(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 204,
			Body:       io.NopCloser(bytes.NewReader([]byte(""))),
			Header:     make(http.Header),
		}, nil
	})

	service := NewService(client)
	err := service.TransitionIssue(context.Background(), "DEMO-1", "2", nil)

	if err != nil {
		t.Fatalf("TransitionIssue error: %v", err)
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
			parts:    []string{"project"},
			expected: "/rest/api/2/project",
		},
		{
			name:     "multiple parts",
			parts:    []string{"issue", "DEMO-1", "comment"},
			expected: "/rest/api/2/issue/DEMO-1/comment",
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
