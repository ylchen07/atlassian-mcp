package jira

import (
	"strings"
	"testing"
	"time"

	"net/http"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

func TestNewV2ClientRequiresSite(t *testing.T) {
	t.Parallel()

	_, err := NewV2Client("  ", config.ServiceCredentials{Email: "user", APIToken: "token"})
	if err == nil || !strings.Contains(err.Error(), "site") {
		t.Fatalf("expected site validation error, got %v", err)
	}
}

func TestNewV2ClientBasicAuth(t *testing.T) {
	t.Parallel()

	client, err := NewV2Client("https://example.atlassian.net", config.ServiceCredentials{
		Email:    "user@example.com",
		APIToken: "secret",
	})
	if err != nil {
		t.Fatalf("NewV2Client error: %v", err)
	}

	if !client.Auth.HasBasicAuth() {
		t.Fatalf("expected basic auth to be configured")
	}

	mail, token := client.Auth.GetBasicAuth()
	if mail != "user@example.com" || token != "secret" {
		t.Fatalf("unexpected basic auth credentials: %s %s", mail, token)
	}

	if !client.Auth.HasUserAgent() {
		t.Fatalf("expected user agent to be configured")
	}

	if agent := client.Auth.GetUserAgent(); agent != "atlassian-mcp" {
		t.Fatalf("expected default user agent, got %s", agent)
	}
}

func TestNewV2ClientOAuthToken(t *testing.T) {
	t.Parallel()

	client, err := NewV2Client("https://example.atlassian.net", config.ServiceCredentials{
		OAuthToken: "bearer-token",
	})
	if err != nil {
		t.Fatalf("NewV2Client error: %v", err)
	}

	if client.Auth.GetBearerToken() != "bearer-token" {
		t.Fatalf("expected bearer token to be set")
	}

	if client.Auth.HasBasicAuth() {
		t.Fatalf("did not expect basic auth to be configured")
	}
}

func TestNewV2ClientOptions(t *testing.T) {
	t.Parallel()

	customHTTP := &http.Client{Timeout: 5 * time.Second}

	client, err := NewV2Client(
		"https://example.atlassian.net",
		config.ServiceCredentials{Email: "user", APIToken: "token"},
		WithV2HTTPClient(customHTTP),
		WithV2UserAgent("custom-agent"),
	)
	if err != nil {
		t.Fatalf("NewV2Client error: %v", err)
	}

	httpClient, ok := client.HTTP.(*http.Client)
	if !ok || httpClient != customHTTP {
		t.Fatalf("expected HTTP client override to be applied")
	}

	if agent := client.Auth.GetUserAgent(); agent != "custom-agent" {
		t.Fatalf("expected custom user agent, got %s", agent)
	}
}
