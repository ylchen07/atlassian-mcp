# Architecture Guide

This document explains the architectural patterns and design decisions in the Atlassian MCP Server codebase.

## Table of Contents

- [Overview](#overview)
- [Layered Architecture](#layered-architecture)
- [Client vs Service Pattern](#client-vs-service-pattern)
- [Data Flow](#data-flow)
- [Design Patterns](#design-patterns)
- [Adding New Features](#adding-new-features)

## Overview

The Atlassian MCP Server follows a **layered architecture** pattern that separates concerns into distinct responsibilities:

```
┌─────────────────────────────────────────────────┐
│  MCP Layer (internal/mcp)                       │
│  - Tool registration                            │
│  - Input/output schema definitions              │
│  - MCP protocol handling                        │
└─────────────────────────────────────────────────┘
                    ↓ Uses
┌─────────────────────────────────────────────────┐
│  Service Layer (internal/jira, internal/conf)   │
│  - Business logic                               │
│  - Domain models                                │
│  - API operations                               │
└─────────────────────────────────────────────────┘
                    ↓ Uses
┌─────────────────────────────────────────────────┐
│  HTTP Client (internal/atlassian)               │
│  - Authentication (OAuth/Basic)                 │
│  - HTTP request/response handling               │
│  - Common utilities (Get/Post/Put/Delete)       │
└─────────────────────────────────────────────────┘
                    ↓ Uses
┌─────────────────────────────────────────────────┐
│  Atlassian REST APIs                            │
│  - Jira REST API v2                             │
│  - Confluence REST API                          │
└─────────────────────────────────────────────────┘
```

## Layered Architecture

### 1. MCP Layer (`internal/mcp/`)

**Purpose**: Expose Atlassian operations as MCP tools

**Responsibilities**:

- Register MCP tools with the server
- Define JSON schemas for tool inputs and outputs
- Map MCP tool calls to service layer methods
- Handle MCP-specific error formatting

**Files**:

- `server.go` - MCP server construction and dependency injection
- `jira.go` - Jira tool registration (8 tools)
- `confluence.go` - Confluence tool registration (4 tools)

**Example**:

```go
// Register a tool
s.AddTool(
    mcp.NewTool(
        "jira.list_projects",
        mcp.WithDescription("Return accessible Jira projects"),
        mcp.WithInputSchema[JiraListProjectsArgs](),
        mcp.WithOutputSchema[JiraListProjectsResult](),
    ),
    mcp.NewTypedToolHandler(jt.handleListProjects),
)

// Handler delegates to service layer
func (jt *JiraTools) handleListProjects(ctx context.Context, args JiraListProjectsArgs) (*mcp.ToolResult, error) {
    projects, err := jt.service.ListProjects(ctx, args.MaxResults)
    // ... format and return
}
```

### 2. Service Layer (`internal/jira/service.go`, `internal/confluence/service.go`)

**Purpose**: Implement business logic for Atlassian operations

**Responsibilities**:

- Define domain models (Project, Issue, Space, Content)
- Expose business operations (ListProjects, SearchIssues, CreatePage)
- Transform API responses into simplified domain models
- Construct API paths and query parameters
- Handle context propagation for cancellation

**Example**:

```go
// Domain model
type Project struct {
    ID   string `json:"id"`
    Key  string `json:"key"`
    Name string `json:"name"`
}

// Business operation
func (s *Service) ListProjects(ctx context.Context, maxResults int) ([]Project, error) {
    // Build query parameters
    params := url.Values{}
    if maxResults > 0 {
        params.Set("maxResults", strconv.Itoa(maxResults))
    }

    // Construct API path
    path := apiPath("project")
    if encoded := params.Encode(); encoded != "" {
        path += "?" + encoded
    }

    // Make request using HTTP client
    var projects []Project
    if err := s.client.Get(ctx, path, &projects); err != nil {
        return nil, err
    }

    return projects, nil
}
```

### 3. HTTP Client Layer (`internal/atlassian/httpclient.go`)

**Purpose**: Shared HTTP client for all Atlassian API calls

**Responsibilities**:

- Create authenticated HTTP client instances
- Configure authentication (OAuth or Basic Auth)
- Set up HTTP client with timeouts
- Provide convenience methods (Get, Post, Put, Delete)
- Handle common HTTP request/response patterns
- Normalize site URLs

**Example**:

```go
// HTTPClient is a simple HTTP client for Atlassian REST APIs
type HTTPClient struct {
    BaseURL    string
    Email      string
    APIToken   string
    OAuthToken string
    HTTPClient *http.Client
}

// Create client with authentication
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
        return nil, fmt.Errorf("atlassian: credentials required")
    }

    return &HTTPClient{
        BaseURL:    strings.TrimRight(baseURL, "/"),
        Email:      creds.Email,
        APIToken:   creds.APIToken,
        OAuthToken: creds.OAuthToken,
        HTTPClient: &http.Client{Timeout: 30 * time.Second},
    }, nil
}

// Convenience method for GET requests
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
```

## HTTP Client vs Service Pattern

A key architectural pattern in this codebase is the **separation of infrastructure (HTTP client) from business logic (service layer)**.

### HTTP Client Layer (`internal/atlassian/httpclient.go`)

**Responsibility**: "How do we connect to Atlassian?"

**What it does**:

- ✅ HTTP client construction and initialization
- ✅ Authentication setup (OAuth or Basic Auth)
- ✅ HTTP configuration (timeouts, headers)
- ✅ URL normalization and validation
- ✅ Common HTTP operations (Get, Post, Put, Delete)
- ✅ Request/response marshaling

**What it does NOT do**:

- ❌ Business operations (list projects, create issues)
- ❌ API path construction (e.g., `/rest/api/2/project`)
- ❌ Domain model transformation
- ❌ Business logic or validation

**Key characteristics**:

- Created **once per service** at startup
- Reused for **all** operations in that service
- No knowledge of business concepts (projects, issues, spaces)
- Pure infrastructure concerns

### Service Layer (`internal/jira/service.go`, `internal/confluence/service.go`)

**Responsibility**: "What operations can we perform on Atlassian?"

**What it does**:

- ✅ Define domain models (Project, Issue, Space, Content)
- ✅ Implement business operations (ListProjects, SearchIssues, CreatePage)
- ✅ Construct API paths (e.g., `/rest/api/2/project`)
- ✅ Handle query parameters
- ✅ Transform API responses to domain models
- ✅ Accept context for cancellation
- ✅ Business logic and validation

**What it does NOT do**:

- ❌ Authentication setup
- ❌ HTTP client initialization
- ❌ URL normalization
- ❌ HTTP transport configuration

**Key characteristics**:

- Uses HTTP client created at startup
- Methods called **per-request**
- Contains business logic and API knowledge
- Returns simplified, MCP-friendly types

### Side-by-Side Comparison

| Aspect           | HTTP Client (`internal/atlassian`)        | Service Layer (`service.go`)               |
| ---------------- | ----------------------------------------- | ------------------------------------------ |
| **Purpose**      | Infrastructure setup                      | Business operations                        |
| **Concerns**     | Authentication, URLs, HTTP                | Projects, Issues, Spaces, Pages            |
| **Returns**      | `*atlassian.HTTPClient`                   | Domain models (`Project`, `Issue`)         |
| **Dependencies** | `config` package only                     | Uses `atlassian.HTTPClient`                |
| **Lifecycle**    | Created once per service at startup       | Methods called per-request                 |
| **Testability**  | Mock HTTP transport or responses          | Mock HTTP client                           |
| **Changes when** | Auth method changes, HTTP patterns change | New features, API endpoints added          |
| **Knowledge**    | Infrastructure/plumbing                   | Business domain/API structure              |
| **Location**     | `internal/atlassian/httpclient.go`        | `internal/jira/` or `internal/confluence/` |

### Why This Separation?

#### 1. Single Responsibility Principle

- HTTP Client: "I know how to connect to Atlassian"
- Service: "I know what operations are possible"

#### 2. Easier Testing

```go
// Test HTTP client - Mock HTTP transport
mockTransport := &MockHTTPTransport{
    RoundTripFunc: func(req *http.Request) (*http.Response, error) {
        // Return canned responses
        return &http.Response{
            StatusCode: 200,
            Body:       io.NopCloser(strings.NewReader(`{"key":"DEMO"}`)),
        }, nil
    },
}
client := &atlassian.HTTPClient{
    BaseURL:    "https://example.atlassian.net",
    OAuthToken: "token",
    HTTPClient: &http.Client{Transport: mockTransport},
}

// Test service - Mock HTTP client methods
mockClient := &MockHTTPClient{
    GetFunc: func(ctx context.Context, path string, result interface{}) error {
        // Return mock data
        return nil
    },
}
service := jira.NewService(mockClient)
```

#### 3. Flexibility and Maintainability

- **Swap authentication** without touching service layer
- **Add new operations** without changing HTTP client
- **Change HTTP implementation** entirely by only changing `internal/atlassian`
- **Update API paths** without modifying authentication

#### 4. Reusability

- One HTTP client instance serves **all** service operations
- Service methods share the same authenticated connection
- No redundant client creation
- Both Jira and Confluence use the same HTTP client implementation

### Consistent Implementation Across Services

Both Jira and Confluence services use the same HTTP client pattern:

```go
// Both services use the shared atlassian.HTTPClient

// Jira service
func (s *Service) ListProjects(ctx context.Context, maxResults int) ([]Project, error) {
    path := apiPath("project") + "?" + params.Encode()

    var projects []Project
    if err := s.client.Get(ctx, path, &projects); err != nil {
        return nil, err
    }

    return projects, nil
}

// Confluence service
func (s *Service) ListSpaces(ctx context.Context, limit int) ([]Space, error) {
    path := apiPath("space") + "?" + params.Encode()

    var response struct {
        Results []Space `json:"results"`
    }
    if err := s.client.Get(ctx, path, &response); err != nil {
        return nil, err
    }

    return response.Results, nil
}
```

**Benefits of unified approach**:

- Consistent patterns across all services
- Easier to maintain and understand
- Shared HTTP client code reduces duplication
- Direct control over API requests and responses

## Data Flow

### Startup Flow

```
┌─────────────────────────────────────────────────────────┐
│ 1. cmd/server/main.go (Application Entry Point)        │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 2. Load Configuration (config.Load)                     │
│    - Read config.yaml or environment variables          │
│    - Validate credentials                               │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 3. Create HTTP Clients (atlassian.NewHTTPClient)       │
│    - Normalize URLs                                     │
│    - Set up authentication                              │
│    - Configure HTTP client with timeouts                │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 4. Create Services (jira.NewService, conf.NewService)   │
│    - Wrap clients                                       │
│    - Expose business operations                         │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 5. Initialize State Cache (state.NewCache)              │
│    - Thread-safe session cache                          │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 6. Create MCP Server (mcp.NewServer)                    │
│    - Inject dependencies (services, cache, logger)      │
│    - Register all MCP tools                             │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 7. Start Server (server.ServeStdio)                     │
│    - Listen on stdin/stdout                             │
│    - Handle MCP protocol messages                       │
└─────────────────────────────────────────────────────────┘
```

### Request Flow

```
┌─────────────────────────────────────────────────────────┐
│ 1. User/Client Invokes MCP Tool                         │
│    Example: jira.list_projects {"maxResults": 10}       │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 2. MCP Layer (internal/mcp/jira.go)                     │
│    - Validate input schema                              │
│    - Call registered handler                            │
│    handleListProjects(ctx, args)                        │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 3. Check Cache (internal/state/cache.go)                │
│    - cache.Projects() → Returns cached if available     │
│    - If cache miss, proceed to API call                 │
└─────────────────────────────────────────────────────────┘
                    ↓ (cache miss)
┌─────────────────────────────────────────────────────────┐
│ 4. Service Layer (internal/jira/service.go)             │
│    - service.ListProjects(ctx, maxResults)              │
│    - Build API path and query params                    │
│    - Make HTTP request via client                       │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 5. HTTP Client (internal/atlassian/httpclient.go)      │
│    - client.Get() creates authenticated GET request     │
│    - Executes HTTP call and decodes response            │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 6. External API (Jira REST API)                         │
│    GET /rest/api/2/project?maxResults=10                │
│    Returns: [{id, key, name}, ...]                      │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 7. Service Layer (Response Handling)                    │
│    - HTTP client automatically decodes JSON             │
│    - Service receives []Project domain model            │
│    - Return to MCP layer                                │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 8. Update Cache                                         │
│    - cache.SetProjects(projects)                        │
│    - Store for future requests                          │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 9. MCP Layer (Response Formatting)                      │
│    - Format as MCP ToolResult                           │
│    - Add human-readable fallback text                   │
│    - Return to client                                   │
└─────────────────────────────────────────────────────────┘
```

### Subsequent Request (Cache Hit)

```
User → MCP Tool → Check Cache → Return Cached Data
                      ↓
                  (No API call!)
```

## Design Patterns

### 1. Dependency Injection

**Pattern**: Dependencies struct passed to constructors

**Example**:

```go
// Define dependencies
type Dependencies struct {
    JiraService       *jira.Service
    ConfluenceService *confluence.Service
    Cache             *state.Cache
    JiraBaseURL       string
    ConfluenceBaseURL string
    Logger            *slog.Logger
}

// Inject at construction time
srv := mcp.NewServer(deps)
```

**Benefits**:

- Testable: Can inject mocks
- Flexible: Easy to add new dependencies
- Explicit: Clear what each component needs

### 2. Functional Options

**Pattern**: Variadic options for customization

**Example**:

```go
// Define option type
type ClientOption func(*jiraapi.Client)

// Provide option constructors
func WithUserAgent(agent string) ClientOption {
    return func(client *jiraapi.Client) {
        client.Auth.SetUserAgent(agent)
    }
}

func WithHTTPClient(httpClient *http.Client) ClientOption {
    return func(client *jiraapi.Client) {
        client.HTTP = httpClient
    }
}

// Use options
client, err := jira.NewClient(site, creds,
    jira.WithUserAgent("custom-agent"),
    jira.WithHTTPClient(customHTTP),
)
```

**Benefits**:

- Backward compatible: Adding options doesn't break existing code
- Clean API: Optional parameters without complex signatures
- Composable: Mix and match options

### 3. Defensive Copying

**Pattern**: Copy slices when storing/retrieving from cache

**Example**:

```go
// Setter copies input
func (c *Cache) SetProjects(projects []jira.Project) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.jiraProjects = append([]jira.Project(nil), projects...)  // Copy
}

// Getter returns copy
func (c *Cache) Projects() []jira.Project {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return append([]jira.Project(nil), c.jiraProjects...)  // Copy
}
```

**Benefits**:

- Safety: External code can't modify internal state
- Concurrency: No shared mutable state
- Correctness: Cache remains consistent

### 4. Thread-Safe Cache

**Pattern**: Read-Write mutex for concurrent access

**Example**:

```go
type Cache struct {
    mu           sync.RWMutex  // Protects all fields
    jiraProjects []jira.Project
    lastJQL      string
}

// Multiple readers allowed
func (c *Cache) Projects() []jira.Project {
    c.mu.RLock()  // Read lock
    defer c.mu.RUnlock()
    return append([]jira.Project(nil), c.jiraProjects...)
}

// Exclusive writer
func (c *Cache) SetProjects(projects []jira.Project) {
    c.mu.Lock()  // Write lock
    defer c.mu.Unlock()
    c.jiraProjects = append([]jira.Project(nil), projects...)
}
```

**Benefits**:

- Performance: Multiple concurrent reads
- Safety: Exclusive writes prevent races
- Correctness: No data corruption

### 5. Context Propagation

**Pattern**: Pass `context.Context` through all layers

**Example**:

```go
// MCP handler
func (jt *JiraTools) handleListProjects(ctx context.Context, args Args) (*mcp.ToolResult, error) {
    projects, err := jt.service.ListProjects(ctx, args.MaxResults)
    // ...
}

// Service method
func (s *Service) ListProjects(ctx context.Context, maxResults int) ([]Project, error) {
    req, err := s.client.NewRequest(ctx, http.MethodGet, path, "", nil)
    // ...
}
```

**Benefits**:

- Cancellation: Can abort long-running operations
- Timeouts: Request-level timeout control
- Tracing: Can attach trace IDs for debugging

## Adding New Features

### Example: Add "Get Issue Comments" Feature

Follow these steps to add a new Jira operation:

#### 1. Add Domain Model to `service.go`

```go
// internal/jira/service.go

// Comment represents a Jira issue comment.
type Comment struct {
    ID      string `json:"id"`
    Author  string `json:"author"`
    Body    string `json:"body"`
    Created string `json:"created"`
}
```

#### 2. Add Service Method to `service.go`

```go
// internal/jira/service.go

// GetComments retrieves all comments for an issue.
func (s *Service) GetComments(ctx context.Context, issueKey string) ([]Comment, error) {
    if issueKey == "" {
        return nil, fmt.Errorf("jira: issue key required")
    }

    // Build API path
    path := apiPath("issue", url.PathEscape(issueKey), "comment")

    // Make request using HTTP client
    var res struct {
        Comments []Comment `json:"comments"`
    }
    if err := s.client.Get(ctx, path, &res); err != nil {
        return nil, fmt.Errorf("jira: get comments: %w", err)
    }

    return res.Comments, nil
}
```

#### 3. Add MCP Tool to `mcp/jira.go`

```go
// internal/mcp/jira.go

// Input schema
type JiraGetCommentsArgs struct {
    IssueKey string `json:"issueKey" jsonschema:"required" jsonschema_description:"Issue key (e.g., DEMO-123)"`
}

// Output schema
type JiraGetCommentsResult struct {
    Comments []jira.Comment `json:"comments"`
}

// Register tool
func (jt *JiraTools) registerTools(s *server.MCPServer) {
    // ... existing tools ...

    s.AddTool(
        mcp.NewTool(
            "jira.get_comments",
            mcp.WithDescription("Get all comments for a Jira issue"),
            mcp.WithInputSchema[JiraGetCommentsArgs](),
            mcp.WithOutputSchema[JiraGetCommentsResult](),
        ),
        mcp.NewTypedToolHandler(jt.handleGetComments),
    )
}

// Handler
func (jt *JiraTools) handleGetComments(ctx context.Context, args JiraGetCommentsArgs) (*mcp.ToolResult, error) {
    comments, err := jt.service.GetComments(ctx, args.IssueKey)
    if err != nil {
        return mcp.NewToolResultErrorFromErr(err), nil
    }

    result := JiraGetCommentsResult{
        Comments: comments,
    }

    fallback := fmt.Sprintf("Found %d comments on issue %s", len(comments), args.IssueKey)
    return mcp.NewToolResultStructured(result, fallback), nil
}
```

#### 4. Add Tests

```go
// internal/jira/service_test.go

func TestService_GetComments(t *testing.T) {
    mockTransport := &mockRoundTripper{
        response: `{"comments": [{"id": "1", "author": "user@example.com", "body": "Test comment"}]}`,
    }

    client := newTestClient(t, mockTransport)
    service := jira.NewService(client)

    comments, err := service.GetComments(context.Background(), "DEMO-123")
    if err != nil {
        t.Fatalf("GetComments failed: %v", err)
    }

    if len(comments) != 1 {
        t.Fatalf("expected 1 comment, got %d", len(comments))
    }
}
```

#### 5. Update Documentation

```markdown
# README.md

### Jira tools

| Tool ID             | Description                        |
| ------------------- | ---------------------------------- |
| `jira.get_comments` | Get all comments for a Jira issue. |
```

**Notice**: No changes needed to `internal/atlassian/httpclient.go` - infrastructure layer is unaffected!

## Summary

### Architectural Principles

1. **Separation of Concerns**: Each layer has a single, clear responsibility
2. **Dependency Injection**: Dependencies passed explicitly, not global
3. **Thread Safety**: Concurrent access handled with proper synchronization
4. **Testability**: Layers can be tested independently with mocks
5. **Flexibility**: Easy to extend with new features or swap implementations

### Key Takeaways

- **`internal/atlassian/httpclient.go`** = Infrastructure (how to connect)
- **`internal/jira/client.go` & `internal/confluence/client.go`** = Client factory (create HTTP clients)
- **`internal/jira/service.go` & `internal/confluence/service.go`** = Business logic (what to do)
- **`internal/mcp/*.go`** = MCP integration (how to expose)
- **`internal/state/cache.go`** = Session state (what to remember)

### Benefits of This Architecture

✅ **Maintainable**: Clear separation makes code easy to understand and modify
✅ **Testable**: Each layer can be mocked and tested independently
✅ **Scalable**: Adding features doesn't require touching infrastructure
✅ **Flexible**: Swap authentication or HTTP implementation without major rewrites
✅ **Safe**: Thread-safe cache and defensive copying prevent bugs
✅ **Unified**: Both Jira and Confluence use the same HTTP client infrastructure

## Further Reading

- [CLAUDE.md](./CLAUDE.md) - Project overview and development guide
- [README.md](./README.md) - Quick start and configuration guide
- [internal/state/README.md](./internal/state/README.md) - State cache documentation
- [Jira REST API](https://developer.atlassian.com/cloud/jira/platform/rest/v2/) - Official Jira API documentation
- [Confluence REST API](https://developer.atlassian.com/cloud/confluence/rest/v1/) - Official Confluence API documentation
