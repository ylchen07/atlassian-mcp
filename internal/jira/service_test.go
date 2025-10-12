package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	jiraapi "github.com/ctreminiom/go-atlassian/v2/jira/v2"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(t *testing.T, fn roundTripFunc) *jiraapi.Client {
	t.Helper()
	creds := config.ServiceCredentials{Email: "user", APIToken: "token"}
	client, err := NewClient(
		"https://example.atlassian.net",
		creds,
		WithHTTPClient(&http.Client{Transport: roundTripFunc(fn)}),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
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
		if r.URL.Path != "/rest/api/2/project/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("maxResults") != "5" {
			t.Fatalf("expected maxResults=5, got %s", r.URL.Query().Get("maxResults"))
		}
		if got := r.Header.Get("Authorization"); got == "" {
			t.Fatalf("expected auth header")
		}
		resp := jsonResponse(t, http.StatusOK, map[string]any{
			"values": []map[string]string{{"id": "1", "key": "PRJ", "name": "Project"}},
		})
		resp.Request = r
		return resp, nil
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
		if r.URL.Path != "/rest/api/2/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["jql"] != "project = PRJ" {
			t.Fatalf("unexpected JQL %#v", body["jql"])
		}
		resp := jsonResponse(t, http.StatusOK, map[string]any{
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
		})
		resp.Request = r
		return resp, nil
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

func TestServiceListTransitions(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/rest/api/2/issue/PRJ-1/transitions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("expand") != "transitions.fields" {
			t.Fatalf("missing expand param")
		}
		resp := jsonResponse(t, http.StatusOK, map[string]any{
			"transitions": []map[string]any{{
				"id":   "1",
				"name": "Done",
				"to": map[string]any{
					"id":   "100",
					"name": "Done",
				},
			}},
		})
		resp.Request = r
		return resp, nil
	})

	svc := NewService(client)
	transitions, err := svc.ListTransitions(context.Background(), "PRJ-1")
	if err != nil {
		t.Fatalf("ListTransitions error: %v", err)
	}
	if len(transitions) != 1 || transitions[0].ID != "1" {
		t.Fatalf("unexpected transitions %#v", transitions)
	}
}

func TestServiceTransitionIssue(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/rest/api/2/issue/PRJ-1/transitions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		transition := body["transition"].(map[string]any)
		if transition["id"] != "2" {
			t.Fatalf("unexpected transition %#v", transition)
		}
		resp := jsonResponse(t, http.StatusNoContent, nil)
		resp.Request = r
		return resp, nil
	})

	svc := NewService(client)
	if err := svc.TransitionIssue(context.Background(), "PRJ-1", "2", nil); err != nil {
		t.Fatalf("TransitionIssue error: %v", err)
	}
}

func TestServiceAddAttachment(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/rest/api/2/issue/PRJ-1/attachments" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-Atlassian-Token") != "no-check" {
			t.Fatalf("missing no-check header")
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Fatalf("unexpected content type %s", ct)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		boundaryIdx := strings.Index(ct, "boundary=")
		if boundaryIdx == -1 {
			t.Fatalf("boundary not found")
		}
		boundary := ct[boundaryIdx+9:]
		mr := multipart.NewReader(bytes.NewReader(data), boundary)
		part, err := mr.NextPart()
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		if part.FileName() != "file.txt" {
			t.Fatalf("unexpected filename %s", part.FileName())
		}
		content, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read part: %v", err)
		}
		if string(content) != "hello" {
			t.Fatalf("unexpected data %s", string(content))
		}
		resp := jsonResponse(t, http.StatusCreated, map[string]any{"id": "att-1"})
		resp.Request = r
		return resp, nil
	})

	svc := NewService(client)
	if err := svc.AddAttachment(context.Background(), "PRJ-1", "file.txt", []byte("hello")); err != nil {
		t.Fatalf("AddAttachment error: %v", err)
	}
}
