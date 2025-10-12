package jira

import (
	"strings"
	"testing"
	"time"

	"net/http"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

func TestNewClientRequiresSite(t *testing.T) {
	t.Parallel()

	_, err := NewClient("  ", config.ServiceCredentials{Email: "user", APIToken: "token"})
	if err == nil || !strings.Contains(err.Error(), "site") {
		t.Fatalf("expected site validation error, got %v", err)
	}
}

func TestNewClientBasicAuth(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://example.atlassian.net/rest/api/3", config.ServiceCredentials{
		Email:    "user@example.com",
		APIToken: "secret",
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
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

	if site := strings.TrimRight(client.Site.String(), "/"); site != "https://example.atlassian.net" {
		t.Fatalf("expected site trimmed to base, got %s", site)
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

	if client.Auth.GetBearerToken() != "bearer-token" {
		t.Fatalf("expected bearer token to be set")
	}

	if client.Auth.HasBasicAuth() {
		t.Fatalf("did not expect basic auth to be configured")
	}
}

func TestNewClientOptions(t *testing.T) {
	t.Parallel()

	customHTTP := &http.Client{Timeout: 5 * time.Second}

	client, err := NewClient(
		"https://example.atlassian.net",
		config.ServiceCredentials{Email: "user", APIToken: "token"},
		WithHTTPClient(customHTTP),
		WithUserAgent("custom-agent"),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	httpClient, ok := client.HTTP.(*http.Client)
	if !ok || httpClient != customHTTP {
		t.Fatalf("expected HTTP client override to be applied")
	}

	if agent := client.Auth.GetUserAgent(); agent != "custom-agent" {
		t.Fatalf("expected custom user agent, got %s", agent)
	}
}

func TestNormalizeSiteFromAPIBase(t *testing.T) {
	t.Parallel()

	input := "https://example.atlassian.net/rest/api/2"
	out, err := normalizeSite(input)
	if err != nil {
		t.Fatalf("normalizeSite unexpected error: %v", err)
	}
	if out != "https://example.atlassian.net" {
		t.Fatalf("expected trimmed site, got %s", out)
	}
}
