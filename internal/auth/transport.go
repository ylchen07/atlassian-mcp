package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"

	"gitlab.com/your-org/jira-mcp/internal/config"
)

// Transport injects Atlassian authentication headers into outbound requests.
type Transport struct {
	base       http.RoundTripper
	authHeader string
	once       sync.Once
	initErr    error
	creds      config.ServiceCredentials
}

// NewTransport creates a new auth transport wrapping the provided RoundTripper.
func NewTransport(base http.RoundTripper, creds config.ServiceCredentials) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &Transport{base: base, creds: creds}
}

// RoundTrip implements http.RoundTripper.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.initialize(); err != nil {
		return nil, err
	}

	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", t.authHeader)
	clone.Header.Set("Accept", "application/json")
	return t.base.RoundTrip(clone)
}

func (t *Transport) initialize() error {
	t.once.Do(func() {
		switch {
		case t.creds.OAuthToken != "":
			t.authHeader = fmt.Sprintf("Bearer %s", t.creds.OAuthToken)
		case t.creds.Email != "" && t.creds.APIToken != "":
			token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", t.creds.Email, t.creds.APIToken)))
			t.authHeader = fmt.Sprintf("Basic %s", token)
		default:
			t.initErr = fmt.Errorf("auth: insufficient credentials")
		}
	})
	return t.initErr
}
