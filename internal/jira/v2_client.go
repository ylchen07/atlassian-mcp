package jira

import (
	"fmt"
	"net/http"
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
	trimmedSite := strings.TrimSpace(site)
	if trimmedSite == "" {
		return nil, fmt.Errorf("jira: site is required to construct v2 client")
	}

	defaultHTTPClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	client, err := jirav2.New(defaultHTTPClient, trimmedSite)
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
