package atlassian

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"log/slog"

	"gitlab.com/your-org/jira-mcp/internal/auth"
	"gitlab.com/your-org/jira-mcp/internal/config"
)

// Client is a helper around the Atlassian REST API.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	logger     *slog.Logger
}

// RawBody allows callers to provide a pre-encoded body when constructing requests.
type RawBody struct {
	Reader      io.Reader
	ContentType string
}

// NewClient constructs a Client for the specified base URL and credentials.
func NewClient(base string, creds config.ServiceCredentials, logger *slog.Logger) (*Client, error) {
	if base == "" {
		return nil, fmt.Errorf("atlassian: base URL required")
	}

	parsed, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("atlassian: parse base url: %w", err)
	}

	transport := auth.NewTransport(nil, creds)
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Client{
		baseURL:    parsed,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

// NewRequest builds an HTTP request with optional query parameters and JSON body.
func (c *Client) NewRequest(ctx context.Context, method, path string, query map[string]string, body any) (*http.Request, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	u := *c.baseURL
	u.Path = strings.TrimRight(c.baseURL.Path, "/") + path

	if len(query) > 0 {
		q := u.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	var bodyReader io.Reader
	contentType := ""
	switch b := body.(type) {
	case nil:
		// no body
	case RawBody:
		bodyReader = b.Reader
		contentType = b.ContentType
	default:
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return nil, fmt.Errorf("atlassian: encode body: %w", err)
		}
		bodyReader = buf
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return req, nil
}

// Do executes the request and decodes the response JSON into out if provided.
func (c *Client) Do(req *http.Request, out any) error {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return parseError(res)
	}

	if out == nil {
		io.Copy(io.Discard, res.Body)
		return nil
	}

	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("atlassian: decode response: %w", err)
	}

	return nil
}

// SetTransport overrides the underlying HTTP transport. Useful for testing.
func (c *Client) SetTransport(rt http.RoundTripper) {
	if rt == nil {
		return
	}
	c.httpClient.Transport = rt
}
