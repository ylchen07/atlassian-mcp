package jira

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	jiraapi "github.com/ctreminiom/go-atlassian/v2/jira/v2"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

// ClientOption allows callers to customise construction of the Jira SDK client.
type ClientOption func(*jiraapi.Client)

// WithUserAgent sets a custom user agent on the Jira client.
func WithUserAgent(agent string) ClientOption {
	return func(client *jiraapi.Client) {
		if strings.TrimSpace(agent) != "" {
			client.Auth.SetUserAgent(agent)
		}
	}
}

// WithHTTPClient overrides the HTTP client used by the Jira SDK.
// Note: The SDK stores the http.Client by reference, so customise transport/timeouts before passing it in.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *jiraapi.Client) {
		if httpClient != nil {
			client.HTTP = httpClient
		}
	}
}

// NewClient creates a Jira REST API client backed by the go-atlassian SDK.
// The site must be the Atlassian base URL (e.g. https://<tenant>.atlassian.net).
// Authentication is derived from the provided credentials using OAuth bearer tokens
// when available, or falling back to basic auth (email/API token).
func NewClient(site string, creds config.ServiceCredentials, opts ...ClientOption) (*jiraapi.Client, error) {
	base, err := normalizeSite(site)
	if err != nil {
		return nil, err
	}

	defaultHTTPClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	client, err := jiraapi.New(defaultHTTPClient, base)
	if err != nil {
		return nil, fmt.Errorf("jira: initialise client: %w", err)
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
		return nil, fmt.Errorf("jira: insufficient credentials for client")
	}

	return client, nil
}

func normalizeSite(site string) (string, error) {
	trimmed := strings.TrimSpace(site)
	if trimmed == "" {
		return "", fmt.Errorf("jira: site is required to construct client")
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
