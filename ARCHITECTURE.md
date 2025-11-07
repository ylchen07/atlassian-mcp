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
│  Client Layer (client.go files)                 │
│  - SDK client construction                      │
│  - Authentication                               │
│  - HTTP configuration                           │
└─────────────────────────────────────────────────┘
                    ↓ Uses
┌─────────────────────────────────────────────────┐
│  External SDK (go-atlassian/v2)                 │
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
- Transform SDK responses into simplified domain models
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
    params.Set("expand", "lead")
    if maxResults > 0 {
        params.Set("maxResults", strconv.Itoa(maxResults))
    }

    // Construct API path
    path := apiPath("project/search") + "?" + params.Encode()

    // Make request using client
    req, err := s.client.NewRequest(ctx, http.MethodGet, path, "", nil)
    if err != nil {
        return nil, err
    }

    // Parse and return domain model
    var res struct {
        Values []Project `json:"values"`
    }
    if err := s.client.Do(req, &res); err != nil {
        return nil, err
    }

    return res.Values, nil
}
```

### 3. Client Layer (`internal/jira/client.go`, `internal/confluence/client.go`)

**Purpose**: Configure and construct SDK clients

**Responsibilities**:

- Create go-atlassian SDK client instances
- Configure authentication (OAuth or Basic Auth)
- Set up HTTP client with timeouts
- Normalize site URLs
- Provide customization options via functional options pattern

**Example**:

```go
func NewClient(site string, creds config.ServiceCredentials, opts ...ClientOption) (*jiraapi.Client, error) {
    // Normalize URL
    base, err := normalizeSite(site)
    if err != nil {
        return nil, err
    }

    // Create HTTP client with timeout
    defaultHTTPClient := &http.Client{
        Timeout: 30 * time.Second,
    }

    // Initialize SDK client
    client, err := jiraapi.New(defaultHTTPClient, base)
    if err != nil {
        return nil, fmt.Errorf("jira: initialise client: %w", err)
    }

    // Set User-Agent
    client.Auth.SetUserAgent("atlassian-mcp")

    // Apply custom options
    for _, opt := range opts {
        opt(client)
    }

    // Configure authentication
    switch {
    case strings.TrimSpace(creds.OAuthToken) != "":
        client.Auth.SetBearerToken(creds.OAuthToken)
    case strings.TrimSpace(creds.Email) != "" && strings.TrimSpace(creds.APIToken) != "":
        client.Auth.SetBasicAuth(creds.Email, creds.APIToken)
    default:
        return nil, fmt.Errorf("jira: insufficient credentials")
    }

    return client, nil
}
```

## Client vs Service Pattern

A key architectural pattern in this codebase is the **separation of infrastructure (`client.go`) from business logic (`service.go`)**.

### `client.go` - Infrastructure Layer

**Responsibility**: "How do we connect to Atlassian?"

**What it does**:

- ✅ Client construction and initialization
- ✅ Authentication setup (OAuth or Basic Auth)
- ✅ HTTP client configuration (timeouts, transport)
- ✅ URL normalization and validation
- ✅ Customization options (WithUserAgent, WithHTTPClient)

**What it does NOT do**:

- ❌ Business operations (list projects, create issues)
- ❌ API path construction
- ❌ Request/response handling
- ❌ Domain model transformation

**Key characteristics**:

- Created **once** at startup
- Reused for **all** operations
- No knowledge of business concepts (projects, issues, spaces)
- Pure infrastructure concerns

### `service.go` - Business Logic Layer

**Responsibility**: "What operations can we perform on Atlassian?"

**What it does**:

- ✅ Define domain models (Project, Issue, Space, Content)
- ✅ Implement business operations (ListProjects, SearchIssues, CreatePage)
- ✅ Construct API paths (/rest/api/2/project/search)
- ✅ Handle query parameters
- ✅ Transform SDK responses to domain models
- ✅ Accept context for cancellation

**What it does NOT do**:

- ❌ Authentication setup
- ❌ Client initialization
- ❌ URL normalization
- ❌ HTTP transport configuration

**Key characteristics**:

- Uses client created by `client.go`
- Methods called **per-request**
- Contains business logic and API knowledge
- Returns simplified, MCP-friendly types

### Side-by-Side Comparison

| Aspect           | `client.go`                       | `service.go`                       |
| ---------------- | --------------------------------- | ---------------------------------- |
| **Purpose**      | Infrastructure setup              | Business operations                |
| **Concerns**     | Authentication, URLs, HTTP        | Projects, Issues, Spaces, Pages    |
| **Returns**      | `*jiraapi.Client` or `*cf.Client` | Domain models (`Project`, `Issue`) |
| **Dependencies** | `config` package only             | Uses client from `client.go`       |
| **Lifecycle**    | Created once at startup           | Methods called per-request         |
| **Testability**  | Mock HTTP transport               | Mock client interface              |
| **Changes when** | Auth method changes, SDK updates  | New features, API endpoints added  |
| **Knowledge**    | Infrastructure/plumbing           | Business domain/API structure      |

### Why This Separation?

#### 1. Single Responsibility Principle

- `client.go`: "I know how to connect to Atlassian"
- `service.go`: "I know what operations are possible"

#### 2. Easier Testing

```go
// Test client.go - Mock HTTP transport
mockTransport := &MockHTTPTransport{
    RoundTripFunc: func(req *http.Request) (*http.Response, error) {
        // Return canned responses
    },
}
client, _ := jira.NewClient(site, creds, jira.WithHTTPClient(&http.Client{
    Transport: mockTransport,
}))

// Test service.go - Mock entire client
mockClient := &MockJiraClient{
    NewRequestFunc: func(...) (*http.Request, error) {
        // Return mock request
    },
}
service := jira.NewService(mockClient)
```

#### 3. Flexibility and Maintainability

- **Swap authentication** without touching service layer
- **Add new operations** without changing client setup
- **Replace SDK** entirely by only changing `client.go`
- **Update API paths** without modifying authentication

#### 4. Reusability

- One client instance serves **all** service operations
- Service methods share the same authenticated connection
- No redundant client creation

### Implementation Differences: Jira vs Confluence

The two services use the go-atlassian SDK differently:

#### Jira: Low-Level HTTP Requests

```go
// jira/service.go - Manual HTTP request construction
func (s *Service) ListProjects(ctx context.Context, maxResults int) ([]Project, error) {
    path := apiPath("project/search") + "?" + params.Encode()
    req, err := s.client.NewRequest(ctx, http.MethodGet, path, "", nil)

    var res struct {
        Values []Project `json:"values"`
    }
    if err := s.client.Do(req, &res); err != nil {
        return nil, err
    }

    return res.Values, nil
}
```

**Why**: Fine-grained control over requests, custom response parsing.

#### Confluence: High-Level SDK Methods

```go
// confluence/service.go - Uses SDK's built-in methods
func (s *Service) ListSpaces(ctx context.Context, limit int) ([]Space, error) {
    options := &models.GetSpacesOptionScheme{
        Expand: []string{"description.plain"},
    }

    page, _, err := s.client.Space.Gets(ctx, options, 0, limit)
    if err != nil {
        return nil, err
    }

    // Transform SDK types to domain models
    return transformSpaces(page.Results), nil
}
```

**Why**: Confluence SDK has more complete high-level API, cleaner to use.

**Both approaches are valid** - demonstrates flexibility of the pattern.

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
│ 3. Create Clients (jira.NewClient, confluence.NewClient)│
│    - Normalize URLs                                     │
│    - Set up authentication                              │
│    - Configure HTTP client                              │
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
│ 5. Client Layer (internal/jira/client.go)               │
│    - client.NewRequest() creates authenticated request  │
│    - client.Do() executes HTTP call                     │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 6. External API (Jira REST API)                         │
│    GET /rest/api/2/project/search?maxResults=10         │
│    Returns: {"values": [...]}                           │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ 7. Service Layer (Response Handling)                    │
│    - Parse JSON response                                │
│    - Transform to []Project domain model                │
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
    path := apiPath("issue", issueKey, "comment")

    // Make request
    req, err := s.client.NewRequest(ctx, http.MethodGet, path, "", nil)
    if err != nil {
        return nil, err
    }

    // Parse response
    var res struct {
        Comments []Comment `json:"comments"`
    }
    if err := s.client.Do(req, &res); err != nil {
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

**Notice**: No changes needed to `client.go` - infrastructure layer is unaffected!

## Summary

### Architectural Principles

1. **Separation of Concerns**: Each layer has a single, clear responsibility
2. **Dependency Injection**: Dependencies passed explicitly, not global
3. **Thread Safety**: Concurrent access handled with proper synchronization
4. **Testability**: Layers can be tested independently with mocks
5. **Flexibility**: Easy to extend with new features or swap implementations

### Key Takeaways

- **`client.go`** = Infrastructure (how to connect)
- **`service.go`** = Business logic (what to do)
- **`mcp/*.go`** = MCP integration (how to expose)
- **`state/cache.go`** = Session state (what to remember)

### Benefits of This Architecture

✅ **Maintainable**: Clear separation makes code easy to understand and modify
✅ **Testable**: Each layer can be mocked and tested independently
✅ **Scalable**: Adding features doesn't require touching infrastructure
✅ **Flexible**: Swap authentication, clients, or SDKs without major rewrites
✅ **Safe**: Thread-safe cache and defensive copying prevent bugs

## Further Reading

- [CLAUDE.md](./CLAUDE.md) - Project overview and development guide
- [internal/state/README.md](./internal/state/README.md) - State cache documentation
- [go-atlassian Documentation](https://deepwiki.com/ctreminiom/go-atlassian) - SDK reference
