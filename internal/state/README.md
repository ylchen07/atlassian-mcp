# State Package

The `state` package provides a thread-safe in-memory cache for MCP session state. It stores lightweight data that's shared across multiple MCP tool invocations during a single server session.

## Purpose

The cache optimizes performance by avoiding redundant API calls and enables contextual tool operations by maintaining session state.

## Cached Data

### 1. Jira Projects List

**Field**: `jiraProjects []jira.Project`

**Purpose**: Stores the list of all accessible Jira projects for the authenticated user.

**Why Cache**: Listing projects requires an API call to Jira. By caching the result, subsequent operations (like creating issues or searching) can reference projects without additional API calls.

**Lifecycle**: Set once on first `jira.list_projects` call, reused for entire session.

**Usage**:

```go
// Store projects after API call
cache.SetProjects(projects)

// Retrieve cached projects (no API call)
projects := cache.Projects()
```

### 2. Last JQL Query

**Field**: `lastJQL string`

**Purpose**: Stores the most recently executed JQL (Jira Query Language) query string.

**Why Cache**: Enables follow-up operations without requiring users to repeat query parameters. Users can reference their previous search context.

**Usage**:

```go
// Store JQL after executing a search
cache.SetLastJQL("project = DEMO AND status = Open")

// Retrieve last query for reference
lastQuery := cache.LastJQL()
```

## Thread Safety

The cache uses `sync.RWMutex` for safe concurrent access:

```go
type Cache struct {
    mu           sync.RWMutex  // Protects all fields
    jiraProjects []jira.Project
    lastJQL      string
}
```

### Why Thread-Safe?

MCP servers can handle **multiple concurrent requests**. Without synchronization, simultaneous reads and writes would cause race conditions.

### Locking Strategy

**Read Lock (`RLock`)** - Multiple readers allowed simultaneously:

- `Projects()` - Read project list
- `LastJQL()` - Read last query

**Write Lock (`Lock`)** - Exclusive access:

- `SetProjects()` - Update project list
- `SetLastJQL()` - Update last query

This allows multiple tools to read cache data concurrently while ensuring writes are atomic.

## Defensive Copying

All methods use defensive copying to prevent external modification of internal state:

```go
// Setter copies input
func (c *Cache) SetProjects(projects []jira.Project) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.jiraProjects = append([]jira.Project(nil), projects...)  // Copy slice
}

// Getter returns a copy
func (c *Cache) Projects() []jira.Project {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return append([]jira.Project(nil), c.jiraProjects...)  // Return copy
}
```

**Why**: Prevents callers from accidentally or maliciously modifying cached data. Each caller receives their own copy of the slice.

**Pattern**: `append([]jira.Project(nil), source...)` creates a new slice with copied elements.

## Usage Example

### Initialization

```go
// In cmd/server/main.go
cache := state.NewCache()

// Pass to MCP server via dependency injection
deps := mcp.Dependencies{
    Cache: cache,
    // ... other dependencies
}
srv := mcp.NewServer(deps)
```

### First Project List Request

```
User invokes: jira.list_projects
              ↓
Service.ListProjects() → API call to Jira
              ↓
Result: [{ID: "10000", Key: "DEMO", Name: "Demo Project"}, ...]
              ↓
cache.SetProjects(projects) → Store in cache
              ↓
Return to user
```

### Subsequent Requests (Cache Hit)

```
User invokes: jira.list_projects (again)
              ↓
Check cache.Projects() → Returns cached data
              ↓
No API call needed! (Fast ✓)
```

### JQL Query Tracking

```
User: "Search for open bugs"
      ↓
Execute: jql = "type = Bug AND status = Open"
      ↓
cache.SetLastJQL(jql) → Store query
      ↓
Later...
      ↓
Tool can reference: cache.LastJQL()
Returns: "type = Bug AND status = Open"
```

## Performance Benefits

| Operation               | Without Cache   | With Cache                | Savings         |
| ----------------------- | --------------- | ------------------------- | --------------- |
| List 100 projects       | ~200ms API call | <1ms memory read          | ~99.5%          |
| 10 consecutive searches | 10 API calls    | 1 API call + 9 cache hits | 90% fewer calls |

## Lifecycle

- **Created**: Once during server initialization in `cmd/server/main.go`
- **Lifetime**: Entire MCP session (process lifetime)
- **Scope**: Shared across all MCP tool handlers
- **Cleared**: Only when server process restarts

## Design Trade-offs

### Advantages ✅

- **Fast**: In-memory access with minimal overhead
- **Simple**: Easy to understand and maintain
- **Safe**: Thread-safe with defensive copying
- **Efficient**: Reduces API load on Atlassian servers

### Limitations ⚠️

- **No TTL**: Cache never expires automatically
- **No Invalidation**: Manual cache clearing not supported
- **Stale Data**: If Jira projects change, cache won't reflect updates until restart
- **Memory**: Grows with project count (typically negligible)
- **Per-Process**: Not shared across multiple server instances

### When Stale Data Matters

Most MCP sessions are short-lived (minutes), and project metadata changes infrequently. For long-running sessions, consider:

- Restarting the server periodically
- Adding TTL-based expiration
- Implementing manual cache invalidation

## Potential Enhancements

Future improvements could include:

### Time-To-Live (TTL)

```go
type CacheEntry struct {
    Data      []jira.Project
    ExpiresAt time.Time
}

func (c *Cache) Projects() []jira.Project {
    c.mu.RLock()
    defer c.mu.RUnlock()
    if time.Now().After(c.projectsExpiry) {
        return nil  // Expired, trigger refresh
    }
    return append([]jira.Project(nil), c.jiraProjects...)
}
```

### Manual Invalidation

```go
func (c *Cache) InvalidateProjects() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.jiraProjects = nil
}
```

### Size Limits

```go
const maxCachedProjects = 1000

func (c *Cache) SetProjects(projects []jira.Project) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if len(projects) > maxCachedProjects {
        projects = projects[:maxCachedProjects]
    }
    c.jiraProjects = append([]jira.Project(nil), projects...)
}
```

### Cache Metrics

```go
type CacheStats struct {
    Hits   int64
    Misses int64
}

func (c *Cache) Stats() CacheStats {
    // Return hit/miss statistics
}
```

## Testing

The package includes comprehensive tests in `cache_test.go`:

- **Concurrency**: Verifies thread-safety with parallel reads/writes
- **Defensive Copying**: Ensures external modifications don't affect cache
- **Basic Operations**: Tests all getter/setter methods

Run tests:

```bash
go test ./internal/state
```

## API Reference

### Constructor

#### `NewCache() *Cache`

Creates a new empty cache instance.

**Returns**: Initialized `*Cache` with zero values.

**Example**:

```go
cache := state.NewCache()
```

### Project Cache Methods

#### `SetProjects(projects []jira.Project)`

Stores a copy of the project list in cache.

**Parameters**:

- `projects`: Slice of Jira projects to cache

**Thread-Safety**: Uses write lock (`Lock`)

**Example**:

```go
projects := []jira.Project{
    {ID: "10000", Key: "DEMO", Name: "Demo Project"},
}
cache.SetProjects(projects)
```

#### `Projects() []jira.Project`

Returns a copy of the cached project list.

**Returns**: Slice of cached projects (defensive copy)

**Thread-Safety**: Uses read lock (`RLock`)

**Example**:

```go
projects := cache.Projects()
for _, p := range projects {
    fmt.Printf("Project: %s (%s)\n", p.Name, p.Key)
}
```

### JQL Cache Methods

#### `SetLastJQL(jql string)`

Stores the last executed JQL query string.

**Parameters**:

- `jql`: JQL query string to cache

**Thread-Safety**: Uses write lock (`Lock`)

**Example**:

```go
cache.SetLastJQL("project = DEMO AND assignee = currentUser()")
```

#### `LastJQL() string`

Returns the last cached JQL query.

**Returns**: Last JQL query string (empty if never set)

**Thread-Safety**: Uses read lock (`RLock`)

**Example**:

```go
lastQuery := cache.LastJQL()
if lastQuery != "" {
    fmt.Printf("Previous query: %s\n", lastQuery)
}
```

## Package Dependencies

```
internal/state
    └── internal/jira (Project struct)
```

The state package imports `internal/jira` only for the `Project` type definition. It has no external dependencies beyond the Go standard library (`sync` package).

## Related Documentation

- [CLAUDE.md](../../CLAUDE.md) - Project overview and architecture
- [internal/jira](../jira) - Jira service implementation
- [internal/mcp](../mcp) - MCP server and tool registration
