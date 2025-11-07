package confluence

import (
	"strings"
	"testing"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

func TestNewClientRequiresSite(t *testing.T) {
	t.Parallel()

	_, err := NewClient("", config.ServiceCredentials{Email: "user", APIToken: "token"})
	if err == nil || !strings.Contains(err.Error(), "site") {
		t.Fatalf("expected site validation error, got %v", err)
	}
}

func TestNewClientRequiresCredentials(t *testing.T) {
	t.Parallel()

	_, err := NewClient("https://example.atlassian.net", config.ServiceCredentials{})
	if err == nil || !strings.Contains(err.Error(), "credentials") {
		t.Fatalf("expected credentials validation error, got %v", err)
	}
}

func TestNewClientBasicAuth(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://example.atlassian.net/wiki", config.ServiceCredentials{
		Email:    "user@example.com",
		APIToken: "secret",
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if client.Email != "user@example.com" {
		t.Fatalf("expected email to be set, got %s", client.Email)
	}

	if client.APIToken != "secret" {
		t.Fatalf("expected API token to be set")
	}

	if client.BaseURL != "https://example.atlassian.net/wiki" {
		t.Fatalf("expected base URL to be set, got %s", client.BaseURL)
	}
}

func TestNewClientOAuthToken(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://example.atlassian.net", config.ServiceCredentials{
		OAuthToken: "bearer-token",
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if client.OAuthToken != "bearer-token" {
		t.Fatalf("expected OAuth token to be set, got %s", client.OAuthToken)
	}

	if client.BaseURL != "https://example.atlassian.net" {
		t.Fatalf("expected base URL to be set, got %s", client.BaseURL)
	}
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://confluence.example.com/", config.ServiceCredentials{
		Email:    "user",
		APIToken: "token",
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if client.BaseURL != "https://confluence.example.com" {
		t.Fatalf("expected trailing slash to be trimmed, got %s", client.BaseURL)
	}
}

func TestNewClientAddsHTTPS(t *testing.T) {
	t.Parallel()

	client, err := NewClient("confluence.example.com", config.ServiceCredentials{
		Email:    "user",
		APIToken: "token",
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if !strings.HasPrefix(client.BaseURL, "https://") {
		t.Fatalf("expected https:// prefix to be added, got %s", client.BaseURL)
	}
}

func TestNewClientWithContextPath(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://example.com/wiki", config.ServiceCredentials{
		Email:    "user",
		APIToken: "token",
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if client.BaseURL != "https://example.com/wiki" {
		t.Fatalf("expected context path to be preserved, got %s", client.BaseURL)
	}
}
