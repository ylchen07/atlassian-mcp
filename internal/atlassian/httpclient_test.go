package atlassian

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

func TestNewHTTPClientRequiresBaseURL(t *testing.T) {
	t.Parallel()

	_, err := NewHTTPClient("", config.ServiceCredentials{Email: "user", APIToken: "token"})
	if err == nil || !strings.Contains(err.Error(), "base URL") {
		t.Fatalf("expected base URL validation error, got %v", err)
	}
}

func TestNewHTTPClientRequiresCredentials(t *testing.T) {
	t.Parallel()

	_, err := NewHTTPClient("https://example.com", config.ServiceCredentials{})
	if err == nil || !strings.Contains(err.Error(), "credentials") {
		t.Fatalf("expected credentials validation error, got %v", err)
	}
}

func TestNewHTTPClientBasicAuth(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient("https://example.com", config.ServiceCredentials{
		Email:    "user@example.com",
		APIToken: "secret",
	})
	if err != nil {
		t.Fatalf("NewHTTPClient error: %v", err)
	}

	if client.Email != "user@example.com" {
		t.Fatalf("expected email to be set, got %s", client.Email)
	}

	if client.APIToken != "secret" {
		t.Fatalf("expected API token to be set")
	}

	if client.OAuthToken != "" {
		t.Fatalf("expected OAuth token to be empty")
	}

	if client.BaseURL != "https://example.com" {
		t.Fatalf("expected base URL to be set, got %s", client.BaseURL)
	}
}

func TestNewHTTPClientOAuthToken(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient("https://example.com", config.ServiceCredentials{
		OAuthToken: "bearer-token",
	})
	if err != nil {
		t.Fatalf("NewHTTPClient error: %v", err)
	}

	if client.OAuthToken != "bearer-token" {
		t.Fatalf("expected OAuth token to be set, got %s", client.OAuthToken)
	}

	if client.Email != "" || client.APIToken != "" {
		t.Fatalf("expected email and API token to be empty when using OAuth")
	}

	if client.BaseURL != "https://example.com" {
		t.Fatalf("expected base URL to be set, got %s", client.BaseURL)
	}
}

func TestNewHTTPClientOAuthPreferredOverBasicAuth(t *testing.T) {
	t.Parallel()

	// When both are provided, OAuth should be stored (though the Do() method prefers OAuth)
	client, err := NewHTTPClient("https://example.com", config.ServiceCredentials{
		Email:      "user@example.com",
		APIToken:   "secret",
		OAuthToken: "bearer-token",
	})
	if err != nil {
		t.Fatalf("NewHTTPClient error: %v", err)
	}

	if client.OAuthToken != "bearer-token" {
		t.Fatalf("expected OAuth token to be set")
	}

	// Basic auth credentials should also be stored
	if client.Email != "user@example.com" || client.APIToken != "secret" {
		t.Fatalf("expected basic auth credentials to be preserved")
	}
}

func TestNewHTTPClientTrimsTrailingSlash(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient("https://example.com/", config.ServiceCredentials{
		Email:    "user",
		APIToken: "token",
	})
	if err != nil {
		t.Fatalf("NewHTTPClient error: %v", err)
	}

	if client.BaseURL != "https://example.com" {
		t.Fatalf("expected trailing slash to be trimmed, got %s", client.BaseURL)
	}
}

func TestNewHTTPClientAddsHTTPS(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient("example.com", config.ServiceCredentials{
		Email:    "user",
		APIToken: "token",
	})
	if err != nil {
		t.Fatalf("NewHTTPClient error: %v", err)
	}

	if !strings.HasPrefix(client.BaseURL, "https://") {
		t.Fatalf("expected https:// prefix to be added, got %s", client.BaseURL)
	}

	if client.BaseURL != "https://example.com" {
		t.Fatalf("expected base URL, got %s", client.BaseURL)
	}
}

func TestNewHTTPClientPreservesHTTP(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient("http://example.com", config.ServiceCredentials{
		Email:    "user",
		APIToken: "token",
	})
	if err != nil {
		t.Fatalf("NewHTTPClient error: %v", err)
	}

	if client.BaseURL != "http://example.com" {
		t.Fatalf("expected http:// to be preserved, got %s", client.BaseURL)
	}
}

func TestNewHTTPClientWithContextPath(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient("https://example.com/jira", config.ServiceCredentials{
		Email:    "user",
		APIToken: "token",
	})
	if err != nil {
		t.Fatalf("NewHTTPClient error: %v", err)
	}

	if client.BaseURL != "https://example.com/jira" {
		t.Fatalf("expected context path to be preserved, got %s", client.BaseURL)
	}
}

func TestNewHTTPClientWithComplexPath(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient("https://example.com/path/to/jira/", config.ServiceCredentials{
		Email:    "user",
		APIToken: "token",
	})
	if err != nil {
		t.Fatalf("NewHTTPClient error: %v", err)
	}

	if client.BaseURL != "https://example.com/path/to/jira" {
		t.Fatalf("expected complex path to be preserved without trailing slash, got %s", client.BaseURL)
	}
}

func TestNewHTTPClientHasHTTPClient(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient("https://example.com", config.ServiceCredentials{
		Email:    "user",
		APIToken: "token",
	})
	if err != nil {
		t.Fatalf("NewHTTPClient error: %v", err)
	}

	if client.HTTPClient == nil {
		t.Fatalf("expected HTTP client to be initialized")
	}

	if client.HTTPClient.Timeout == 0 {
		t.Fatalf("expected timeout to be set")
	}
}

// Mock RoundTripper for testing HTTP methods
type mockRoundTripper struct {
	response *http.Response
	err      error
	requests []*http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestHTTPClientGet(t *testing.T) {
	t.Parallel()

	responseBody := map[string]string{"key": "value", "foo": "bar"}
	responseJSON, _ := json.Marshal(responseBody)

	mock := &mockRoundTripper{
		response: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(responseJSON)),
			Header:     make(http.Header),
		},
	}

	client := &HTTPClient{
		BaseURL:    "https://example.com",
		Email:      "user@example.com",
		APIToken:   "secret",
		HTTPClient: &http.Client{Transport: mock},
	}

	var result map[string]string
	err := client.Get(context.Background(), "/api/test", &result)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}

	if result["key"] != "value" {
		t.Fatalf("expected key=value, got %v", result)
	}

	// Verify request
	if len(mock.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(mock.requests))
	}

	req := mock.requests[0]
	if req.Method != "GET" {
		t.Fatalf("expected GET method, got %s", req.Method)
	}

	if req.URL.String() != "https://example.com/api/test" {
		t.Fatalf("unexpected URL: %s", req.URL.String())
	}

	// Check basic auth header
	username, password, ok := req.BasicAuth()
	if !ok || username != "user@example.com" || password != "secret" {
		t.Fatalf("expected basic auth to be set correctly")
	}
}

func TestHTTPClientGetWithOAuth(t *testing.T) {
	t.Parallel()

	mock := &mockRoundTripper{
		response: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"result":"ok"}`))),
			Header:     make(http.Header),
		},
	}

	client := &HTTPClient{
		BaseURL:    "https://example.com",
		OAuthToken: "bearer-token-123",
		HTTPClient: &http.Client{Transport: mock},
	}

	var result map[string]string
	err := client.Get(context.Background(), "/api/test", &result)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}

	// Verify OAuth header
	req := mock.requests[0]
	authHeader := req.Header.Get("Authorization")
	if authHeader != "Bearer bearer-token-123" {
		t.Fatalf("expected Bearer token, got %s", authHeader)
	}
}

func TestHTTPClientGetError(t *testing.T) {
	t.Parallel()

	mock := &mockRoundTripper{
		response: &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewReader([]byte("Not Found"))),
			Header:     make(http.Header),
		},
	}

	client := &HTTPClient{
		BaseURL:    "https://example.com",
		Email:      "user",
		APIToken:   "token",
		HTTPClient: &http.Client{Transport: mock},
	}

	var result map[string]string
	err := client.Get(context.Background(), "/api/test", &result)
	if err == nil {
		t.Fatalf("expected error for 404 response")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected 404 in error message, got: %v", err)
	}
}

func TestHTTPClientPost(t *testing.T) {
	t.Parallel()

	responseJSON, _ := json.Marshal(map[string]string{"id": "123"})

	mock := &mockRoundTripper{
		response: &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(bytes.NewReader(responseJSON)),
			Header:     make(http.Header),
		},
	}

	client := &HTTPClient{
		BaseURL:    "https://example.com",
		Email:      "user",
		APIToken:   "token",
		HTTPClient: &http.Client{Transport: mock},
	}

	requestBody := map[string]string{"name": "test"}
	var result map[string]string
	err := client.Post(context.Background(), "/api/create", requestBody, &result)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}

	if result["id"] != "123" {
		t.Fatalf("unexpected result: %v", result)
	}

	// Verify request
	req := mock.requests[0]
	if req.Method != "POST" {
		t.Fatalf("expected POST method, got %s", req.Method)
	}

	// Verify request body
	var sentBody map[string]string
	bodyBytes, _ := io.ReadAll(req.Body)
	json.Unmarshal(bodyBytes, &sentBody)
	if sentBody["name"] != "test" {
		t.Fatalf("unexpected request body: %v", sentBody)
	}
}

func TestHTTPClientPut(t *testing.T) {
	t.Parallel()

	responseJSON, _ := json.Marshal(map[string]int{"version": 2})

	mock := &mockRoundTripper{
		response: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(responseJSON)),
			Header:     make(http.Header),
		},
	}

	client := &HTTPClient{
		BaseURL:    "https://example.com",
		Email:      "user",
		APIToken:   "token",
		HTTPClient: &http.Client{Transport: mock},
	}

	requestBody := map[string]string{"title": "updated"}
	var result map[string]int
	err := client.Put(context.Background(), "/api/update/123", requestBody, &result)
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}

	if result["version"] != 2 {
		t.Fatalf("unexpected result: %v", result)
	}

	// Verify request
	req := mock.requests[0]
	if req.Method != "PUT" {
		t.Fatalf("expected PUT method, got %s", req.Method)
	}
}

func TestHTTPClientPutNoContent(t *testing.T) {
	t.Parallel()

	mock := &mockRoundTripper{
		response: &http.Response{
			StatusCode: 204,
			Body:       io.NopCloser(bytes.NewReader([]byte(""))),
			Header:     make(http.Header),
		},
	}

	client := &HTTPClient{
		BaseURL:    "https://example.com",
		Email:      "user",
		APIToken:   "token",
		HTTPClient: &http.Client{Transport: mock},
	}

	requestBody := map[string]string{"title": "updated"}
	var result map[string]int
	err := client.Put(context.Background(), "/api/update/123", requestBody, &result)
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}

	// Result should be empty for 204 responses
	if len(result) != 0 {
		t.Fatalf("expected empty result for 204, got %v", result)
	}
}

func TestHTTPClientDelete(t *testing.T) {
	t.Parallel()

	mock := &mockRoundTripper{
		response: &http.Response{
			StatusCode: 204,
			Body:       io.NopCloser(bytes.NewReader([]byte(""))),
			Header:     make(http.Header),
		},
	}

	client := &HTTPClient{
		BaseURL:    "https://example.com",
		Email:      "user",
		APIToken:   "token",
		HTTPClient: &http.Client{Transport: mock},
	}

	err := client.Delete(context.Background(), "/api/delete/123")
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	// Verify request
	req := mock.requests[0]
	if req.Method != "DELETE" {
		t.Fatalf("expected DELETE method, got %s", req.Method)
	}

	if req.URL.Path != "/api/delete/123" {
		t.Fatalf("unexpected path: %s", req.URL.Path)
	}
}

func TestHTTPClientDeleteError(t *testing.T) {
	t.Parallel()

	mock := &mockRoundTripper{
		response: &http.Response{
			StatusCode: 403,
			Body:       io.NopCloser(bytes.NewReader([]byte("Forbidden"))),
			Header:     make(http.Header),
		},
	}

	client := &HTTPClient{
		BaseURL:    "https://example.com",
		Email:      "user",
		APIToken:   "token",
		HTTPClient: &http.Client{Transport: mock},
	}

	err := client.Delete(context.Background(), "/api/delete/123")
	if err == nil {
		t.Fatalf("expected error for 403 response")
	}

	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected 403 in error message, got: %v", err)
	}
}
