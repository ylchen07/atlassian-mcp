package jira

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	jirav2 "github.com/ctreminiom/go-atlassian/v2/jira/v2"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

// V2ClientOption allows callers to customise construction of the Jira v2 SDK client.
type V2ClientOption func(*jirav2.Client)

// WithV2UserAgent sets a custom user agent on the Jira v2 client.
func WithV2UserAgent(agent string) V2ClientOption {
	return func(client *jirav2.Client) {
		if strings.TrimSpace(agent) != "" {
			client.Auth.SetUserAgent(agent)
		}
	}
}

// WithV2HTTPClient overrides the HTTP client used by the Jira v2 SDK.
// Note: The SDK stores the http.Client by reference, so customise transport/timeouts before passing it in.
func WithV2HTTPClient(httpClient *http.Client) V2ClientOption {
	return func(client *jirav2.Client) {
		if httpClient != nil {
			client.HTTP = httpClient
		}
	}
}

// NewV2Client creates a Jira REST API v2 client backed by the go-atlassian SDK.
// The site must be the Atlassian base URL (e.g. https://<tenant>.atlassian.net).
// Authentication is derived from the provided credentials using OAuth bearer tokens
// when available, or falling back to basic auth (email/API token).
func NewV2Client(site string, creds config.ServiceCredentials, opts ...V2ClientOption) (*jirav2.Client, error) {
	base, err := normalizeSite(site)
	if err != nil {
		return nil, err
	}

	defaultHTTPClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	client, err := jirav2.New(defaultHTTPClient, base)
	if err != nil {
		return nil, fmt.Errorf("jira: initialise v2 client: %w", err)
	}

	client.Auth.SetUserAgent("atlassian-mcp")

	for _, opt := range opts {
		opt(client)
	}

	switch {
	case strings.TrimSpace(creds.OAuthToken) != "":
		client.Auth.SetBearerToken(creds.OAuthToken)
	case strings.TrimSpace(creds.Email) != "" && strings.TrimSpace(creds.APIToken) != "":
		client.Auth.SetBasicAuth(creds.Email, creds.APIToken)
	default:
		return nil, fmt.Errorf("jira: insufficient credentials for v2 client")
	}

	return client, nil
}

func normalizeSite(site string) (string, error) {
	trimmed := strings.TrimSpace(site)
	if trimmed == "" {
		return "", fmt.Errorf("jira: site is required to construct v2 client")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("jira: parse site: %w", err)
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	for _, suffix := range []string{"/rest/api/3", "/rest/api/2"} {
		if strings.HasSuffix(parsed.Path, suffix) {
			parsed.Path = strings.TrimSuffix(parsed.Path, suffix)
			parsed.Path = strings.TrimRight(parsed.Path, "/")
			break
		}
	}

	if parsed.Path != "" && !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}

	return parsed.String(), nil
}
