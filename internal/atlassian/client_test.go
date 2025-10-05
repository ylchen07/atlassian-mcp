package atlassian

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

func TestNewClientValidation(t *testing.T) {
	t.Parallel()

	if _, err := NewClient("", config.ServiceCredentials{}, nil); err == nil {
		t.Fatalf("expected error when base URL is empty")
	}
}

func TestNewClientDefaults(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://example.atlassian.net", config.ServiceCredentials{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.baseURL == nil || client.baseURL.String() != "https://example.atlassian.net" {
		t.Fatalf("unexpected base URL: %v", client.baseURL)
	}

	if client.logger == nil {
		t.Fatalf("expected logger to default")
	}

	if client.httpClient == nil {
		t.Fatalf("expected http client to be initialised")
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Fatalf("unexpected timeout: %v", client.httpClient.Timeout)
	}

	if client.httpClient.Transport == nil {
		t.Fatalf("expected transport to be configured")
	}
}

func TestClientNewRequest(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)

	t.Run("json body", func(t *testing.T) {
		req, err := client.NewRequest(
			context.Background(),
			http.MethodPost,
			"search",
			map[string]string{"expand": "names", "maxResults": "10"},
			map[string]string{"jql": "project = TEST"},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := req.URL.Path; got != "/rest/api/search" {
			t.Fatalf("unexpected path: %s", got)
		}
		if got := req.URL.Query().Get("expand"); got != "names" {
			t.Fatalf("unexpected query value: %s", got)
		}
		if got := req.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content-type: %s", got)
		}
		var body map[string]string
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["jql"] != "project = TEST" {
			t.Fatalf("unexpected body: %#v", body)
		}
	})

	t.Run("raw body", func(t *testing.T) {
		req, err := client.NewRequest(
			context.Background(),
			http.MethodPost,
			"/search",
			nil,
			RawBody{Reader: strings.NewReader("payload"), ContentType: "text/plain"},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := req.Header.Get("Content-Type"); got != "text/plain" {
			t.Fatalf("unexpected content-type: %s", got)
		}
		data, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(data) != "payload" {
			t.Fatalf("unexpected body: %s", string(data))
		}
	})
}

func TestClientDoSuccess(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)
	client.SetTransport(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/rest/api/search" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("{\"value\":\"ok\"}")),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	}))

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/search", nil, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	var out struct {
		Value string `json:"value"`
	}
	if err := client.Do(req, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	if out.Value != "ok" {
		t.Fatalf("unexpected value: %s", out.Value)
	}
}

func TestClientDoJSONError(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)
	client.SetTransport(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("{\"message\":\"boom\"}")),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	}))

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/", nil, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	err = client.Do(req, nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest || apiErr.Message != "boom" {
		t.Fatalf("unexpected error: %#v", apiErr)
	}
}

func TestClientDoDecodeFailure(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)
	client.SetTransport(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("{\"value\"")),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	}))

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/", nil, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	var out struct{}
	if err := client.Do(req, &out); err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestClientSetTransport(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)
	original := client.httpClient.Transport

	client.SetTransport(nil)
	if client.httpClient.Transport != original {
		t.Fatalf("nil transport should be ignored")
	}

	called := false
	client.SetTransport(roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	}))

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	if _, err := client.httpClient.Transport.RoundTrip(req); err != nil {
		t.Fatalf("round trip: %v", err)
	}
	if !called {
		t.Fatalf("expected custom transport to be used")
	}
}

func TestParseErrorFallback(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		body       string
		wantText   string
		wantStatus int
	}{
		{"message field", `{"message":"boom"}`, "atlassian: 400 boom", 400},
		{"error messages", `{"errorMessages":["nope"]}`, "atlassian: 500 nope", 500},
		{"raw body", "unexpected", "atlassian: 404 unexpected", 404},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res := &http.Response{
				StatusCode: tc.wantStatus,
				Body:       io.NopCloser(strings.NewReader(tc.body)),
			}
			err := parseError(res)
			var apiErr *Error
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected *Error, got %T", err)
			}
			if apiErr.StatusCode != tc.wantStatus {
				t.Fatalf("unexpected status: %d", apiErr.StatusCode)
			}
			if apiErr.Error() != tc.wantText {
				t.Fatalf("unexpected error string: %s", apiErr.Error())
			}
		})
	}
}

func newTestClient(t *testing.T) *Client {
	return newTestClientWithBase(t, "https://example.atlassian.net/rest/api")
}

func newTestClientWithBase(t *testing.T, base string) *Client {
	t.Helper()

	client, err := NewClient(base, config.ServiceCredentials{OAuthToken: "token"}, nil)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
