package confluence

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

func TestNewClientRequiresSite(t *testing.T) {
	t.Parallel()

	if _, err := NewClient(" ", config.ServiceCredentials{Email: "user", APIToken: "token"}); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestNewClientBasicAuth(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://example.atlassian.net/wiki/rest/api", config.ServiceCredentials{
		Email:    "user@example.com",
		APIToken: "secret",
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if !client.Auth.HasBasicAuth() {
		t.Fatalf("expected basic auth")
	}

	if ua := client.Auth.GetUserAgent(); ua != "atlassian-mcp" {
		t.Fatalf("expected default user agent, got %s", ua)
	}

	if site := strings.TrimRight(client.Site.String(), "/"); site != "https://example.atlassian.net" {
		t.Fatalf("expected trimmed site, got %s", site)
	}
}

func TestNewClientOAuth(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://example.atlassian.net", config.ServiceCredentials{
		OAuthToken: "token",
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if client.Auth.GetBearerToken() != "token" {
		t.Fatalf("expected bearer token to be configured")
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
		t.Fatalf("expected HTTP override to be used")
	}

	if ua := client.Auth.GetUserAgent(); ua != "custom-agent" {
		t.Fatalf("expected user agent override, got %s", ua)
	}
}

func TestNormalizeSite(t *testing.T) {
	t.Parallel()

	out, err := normalizeSite("https://example.atlassian.net/wiki/rest/api")
	if err != nil {
		t.Fatalf("normalizeSite error: %v", err)
	}
	if out != "https://example.atlassian.net" {
		t.Fatalf("unexpected site: %s", out)
	}
}
