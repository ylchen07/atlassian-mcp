package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"gitlab.com/your-org/jira-mcp/internal/atlassian"
	"gitlab.com/your-org/jira-mcp/internal/auth"
	"gitlab.com/your-org/jira-mcp/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(t *testing.T, fn roundTripFunc) *atlassian.Client {
	t.Helper()
	creds := config.ServiceCredentials{Email: "user", APIToken: "token"}
	client, err := atlassian.NewClient("https://example.atlassian.net", creds, nil)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	client.SetTransport(auth.NewTransport(fn, creds))
	return client
}

func jsonResponse(t *testing.T, status int, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(data)),
		Header:     make(http.Header),
	}
}

func TestServiceListProjects(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/project/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("maxResults") != "5" {
			t.Fatalf("expected maxResults=5, got %s", r.URL.Query().Get("maxResults"))
		}
		if got := r.Header.Get("Authorization"); got == "" {
			t.Fatalf("expected auth header")
		}
		return jsonResponse(t, http.StatusOK, map[string]any{
			"values": []map[string]string{{"id": "1", "key": "PRJ", "name": "Project"}},
		}), nil
	})

	svc := NewService(client)
	projects, err := svc.ListProjects(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListProjects error: %v", err)
	}
	if len(projects) != 1 || projects[0].Key != "PRJ" {
		t.Fatalf("unexpected projects %#v", projects)
	}
}

func TestServiceSearchIssues(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["jql"] != "project = PRJ" {
			t.Fatalf("unexpected JQL %#v", body["jql"])
		}
		return jsonResponse(t, http.StatusOK, map[string]any{
			"total":      1,
			"startAt":    0,
			"maxResults": 50,
			"issues": []map[string]any{{
				"id":  "1",
				"key": "PRJ-1",
				"fields": map[string]any{
					"summary":     "Issue",
					"description": "Details",
					"status":      map[string]any{"name": "To Do"},
					"assignee":    map[string]any{"displayName": "User"},
				},
			}},
		}), nil
	})

	svc := NewService(client)
	res, err := svc.SearchIssues(context.Background(), SearchRequest{JQL: "project = PRJ"})
	if err != nil {
		t.Fatalf("SearchIssues error: %v", err)
	}
	if res.Total != 1 || len(res.Issues) != 1 {
		t.Fatalf("unexpected result %#v", res)
	}
	if res.Issues[0].Key != "PRJ-1" {
		t.Fatalf("expected issue key PRJ-1, got %s", res.Issues[0].Key)
	}
}
