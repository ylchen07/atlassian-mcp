package atlassian

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

// HTTPClient is a simple HTTP client for Atlassian REST APIs (Jira, Confluence).
// It supports both basic authentication (email + API token) and OAuth bearer tokens.
type HTTPClient struct {
	BaseURL    string
	Email      string
	APIToken   string
	OAuthToken string
	HTTPClient *http.Client
}

// NewHTTPClient creates an HTTP client for Atlassian services with proper authentication.
// The baseURL should include any context paths (e.g., https://domain.com/jira).
func NewHTTPClient(baseURL string, creds config.ServiceCredentials) (*HTTPClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("atlassian: base URL is required")
	}

	// Ensure HTTPS
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	// Validate credentials - either OAuth token OR email+API token
	hasOAuth := strings.TrimSpace(creds.OAuthToken) != ""
	hasBasicAuth := strings.TrimSpace(creds.Email) != "" && strings.TrimSpace(creds.APIToken) != ""

	if !hasOAuth && !hasBasicAuth {
		return nil, fmt.Errorf("atlassian: credentials required (either oauth_token or email+api_token)")
	}

	return &HTTPClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Email:      creds.Email,
		APIToken:   creds.APIToken,
		OAuthToken: creds.OAuthToken,
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

	// Set authentication - prefer OAuth if available
	if c.OAuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.OAuthToken)
	} else {
		req.SetBasicAuth(c.Email, c.APIToken)
	}

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

// Delete is a helper for DELETE requests.
func (c *HTTPClient) Delete(ctx context.Context, path string) error {
	resp, err := c.Do(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
