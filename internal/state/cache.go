package state

import (
	"sync"

	"gitlab.com/your-org/atlassian-mcp/internal/jira"
)

// Cache holds lightweight shared state for the MCP session.
type Cache struct {
	mu           sync.RWMutex
	jiraProjects []jira.Project
	lastJQL      string
}

// NewCache creates a Cache.
func NewCache() *Cache {
	return &Cache{}
}

// SetProjects stores the list of Jira projects.
func (c *Cache) SetProjects(projects []jira.Project) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.jiraProjects = append([]jira.Project(nil), projects...)
}

// Projects returns the cached Jira projects.
func (c *Cache) Projects() []jira.Project {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]jira.Project(nil), c.jiraProjects...)
}

// SetLastJQL stores the last executed JQL query string.
func (c *Cache) SetLastJQL(jql string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastJQL = jql
}

// LastJQL retrieves the previous JQL query.
func (c *Cache) LastJQL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastJQL
}
