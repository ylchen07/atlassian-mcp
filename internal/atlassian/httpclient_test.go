package atlassian

import (
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
