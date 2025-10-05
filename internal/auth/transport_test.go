package auth

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ylchen07/atlassian-mcp/internal/config"
)

func TestNewTransportDefaultBase(t *testing.T) {
	t.Parallel()

	transport := NewTransport(nil, config.ServiceCredentials{OAuthToken: "token"})
	if transport == nil {
		t.Fatalf("expected transport")
	}
	if transport.base == nil {
		t.Fatalf("expected default base transport")
	}
}

func TestRoundTripSetsOAuthHeader(t *testing.T) {
	t.Parallel()

	var original *http.Request

	rt := NewTransport(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req == original {
			t.Fatalf("request should be cloned")
		}
		if got := req.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		if got := req.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("unexpected accept header: %s", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	}), config.ServiceCredentials{OAuthToken: "token"})

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	original = req

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("round trip: %v", err)
	}
	if req.Header.Get("Authorization") != "" {
		t.Fatalf("original request must not be mutated")
	}
}

func TestRoundTripSetsBasicHeader(t *testing.T) {
	t.Parallel()

	creds := config.ServiceCredentials{Email: "user@example.com", APIToken: "s3cret"}
	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:s3cret"))

	rt := NewTransport(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != expected {
			t.Fatalf("unexpected auth header: %s", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	}), creds)

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("round trip: %v", err)
	}
}

func TestRoundTripInsufficientCredentials(t *testing.T) {
	t.Parallel()

	base := roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatalf("base transport should not be called")
		return nil, nil
	})

	rt := NewTransport(base, config.ServiceCredentials{})

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	if _, err := rt.RoundTrip(req); err == nil || !strings.Contains(err.Error(), "insufficient credentials") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := rt.RoundTrip(req); err == nil || !strings.Contains(err.Error(), "insufficient credentials") {
		t.Fatalf("error should persist: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
