package confluence

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	cf "github.com/ctreminiom/go-atlassian/v2/confluence"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

// ClientOption allows callers to customise the underlying Confluence SDK client.
type ClientOption func(*cf.Client)

// WithHTTPClient overrides the HTTP transport used by the SDK client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *cf.Client) {
		if httpClient != nil {
			client.HTTP = httpClient
		}
	}
}

// WithUserAgent overrides the User-Agent header applied to outbound requests.
func WithUserAgent(agent string) ClientOption {
	return func(client *cf.Client) {
		if strings.TrimSpace(agent) != "" {
			client.Auth.SetUserAgent(agent)
		}
	}
}

// NewClient constructs a Confluence REST client backed by go-atlassian.
// The site may be a tenant base (https://tenant.atlassian.net) or a REST URL
// such as https://tenant.atlassian.net/wiki/rest/api. Credentials favour OAuth
// bearer tokens, with basic auth (email/API token) as a fallback.
func NewClient(site string, creds config.ServiceCredentials, opts ...ClientOption) (*cf.Client, error) {
	base, err := normalizeSite(site)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}

	client, err := cf.New(httpClient, base)
	if err != nil {
		return nil, fmt.Errorf("confluence: initialise client: %w", err)
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
		return nil, fmt.Errorf("confluence: insufficient credentials for client")
	}

	return client, nil
}

func normalizeSite(site string) (string, error) {
	trimmed := strings.TrimSpace(site)
	if trimmed == "" {
		return "", fmt.Errorf("confluence: site is required to construct client")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("confluence: parse site: %w", err)
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	for _, suffix := range []string{
		"/wiki/rest/api",
		"/rest/api",
		"/wiki",
	} {
		if strings.HasSuffix(parsed.Path, suffix) {
			parsed.Path = strings.TrimSuffix(parsed.Path, suffix)
			parsed.Path = strings.TrimRight(parsed.Path, "/")
		}
	}

	if parsed.Path != "" && !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}

	return parsed.String(), nil
}
