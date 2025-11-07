package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

// HTTPClient is a simple HTTP client for Jira REST API.
type HTTPClient struct {
	BaseURL    string
	Email      string
	APIToken   string
	HTTPClient *http.Client
}

// NewHTTPClient creates a simple Jira HTTP client with basic auth.
func NewHTTPClient(baseURL string, creds config.ServiceCredentials) (*HTTPClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	// Ensure HTTPS
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	// Validate credentials
	if creds.Email == "" || creds.APIToken == "" {
		return nil, fmt.Errorf("email and api_token are required")
	}

	return &HTTPClient{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Email:    creds.Email,
		APIToken: creds.APIToken,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Do executes an HTTP request with authentication.
func (c *HTTPClient) Do(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	// Build full URL
	fullURL := c.BaseURL + path

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.Email, c.APIToken)

	return c.HTTPClient.Do(req)
}

// Get is a helper for GET requests.
func (c *HTTPClient) Get(ctx context.Context, path string, result interface{}) error {
	resp, err := c.Do(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// Post is a helper for POST requests.
func (c *HTTPClient) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.Do(ctx, "POST", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// Put is a helper for PUT requests.
func (c *HTTPClient) Put(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.Do(ctx, "PUT", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
