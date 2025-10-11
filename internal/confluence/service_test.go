package confluence

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	cf "github.com/ctreminiom/go-atlassian/v2/confluence"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(t *testing.T, fn roundTripFunc) *cf.Client {
	t.Helper()
	creds := config.ServiceCredentials{Email: "user", APIToken: "token"}
	client, err := NewClient(
		"https://example.atlassian.net/wiki/rest/api",
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

func TestServiceListSpaces(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/wiki/rest/api/space" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "2" {
			t.Fatalf("expected limit=2, got %s", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("expand") != "description.plain" {
			t.Fatalf("expected expand=description.plain, got %s", r.URL.Query().Get("expand"))
		}
		resp := jsonResponse(t, http.StatusOK, map[string]any{
			"results": []map[string]any{{
				"id":   1,
				"key":  "SPACE",
				"name": "Space",
				"description": map[string]any{
					"plain": map[string]any{"value": "desc"},
				},
			}},
		})
		resp.Request = r
		return resp, nil
	})

	svc := NewService(client)
	spaces, err := svc.ListSpaces(context.Background(), 2)
	if err != nil {
		t.Fatalf("ListSpaces error: %v", err)
	}
	if len(spaces) != 1 || spaces[0].Key != "SPACE" {
		t.Fatalf("unexpected spaces %#v", spaces)
	}
}

func TestServiceSearchContent(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/wiki/rest/api/content/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("cql") != "type=page" {
			t.Fatalf("unexpected CQL %s", r.URL.Query().Get("cql"))
		}
		resp := jsonResponse(t, http.StatusOK, map[string]any{
			"results": []map[string]any{{
				"id":     "1",
				"title":  "Page",
				"type":   "page",
				"status": "current",
				"version": map[string]any{
					"number": 2,
				},
			}},
		})
		resp.Request = r
		return resp, nil
	})

	svc := NewService(client)
	pages, err := svc.SearchContent(context.Background(), "type=page", 10)
	if err != nil {
		t.Fatalf("SearchContent error: %v", err)
	}
	if len(pages) != 1 || pages[0].ID != "1" {
		t.Fatalf("unexpected search results %#v", pages)
	}
}

func TestServiceCreatePage(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/wiki/rest/api/content" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		resp := jsonResponse(t, http.StatusOK, map[string]any{
			"id":    "1",
			"title": body["title"],
			"version": map[string]any{
				"number": 1,
			},
		})
		resp.Request = r
		return resp, nil
	})

	svc := NewService(client)
	res, err := svc.CreatePage(context.Background(), PageInput{SpaceKey: "SPACE", Title: "Page", Body: "body"})
	if err != nil {
		t.Fatalf("CreatePage error: %v", err)
	}
	if res.ID != "1" || res.Title != "Page" {
		t.Fatalf("unexpected response %#v", res)
	}
}

func TestServiceUpdatePage(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/wiki/rest/api/content/1" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		version, _ := body["version"].(map[string]any)
		if version["number"].(float64) != 2 {
			t.Fatalf("expected version 2, got %v", version["number"])
		}
		resp := jsonResponse(t, http.StatusOK, map[string]any{
			"id":    "1",
			"title": body["title"],
			"version": map[string]any{
				"number": 2,
			},
		})
		resp.Request = r
		return resp, nil
	})

	svc := NewService(client)
	res, err := svc.UpdatePage(context.Background(), "1", PageInput{Title: "Updated", Body: "body", Version: 2})
	if err != nil {
		t.Fatalf("UpdatePage error: %v", err)
	}
	if res.Version.Number != 2 || res.Title != "Updated" {
		t.Fatalf("unexpected update response %#v", res)
	}
}
