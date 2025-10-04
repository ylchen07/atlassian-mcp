package state

import (
	"testing"

	"gitlab.com/your-org/atlassian-mcp/internal/jira"
)

func TestCacheProjects(t *testing.T) {
	cache := NewCache()

	projects := []jira.Project{{ID: "1", Key: "KEY", Name: "Project"}}
	cache.SetProjects(projects)

	got := cache.Projects()
	if len(got) != 1 {
		t.Fatalf("expected 1 project, got %d", len(got))
	}

	if got[0].Key != "KEY" {
		t.Fatalf("unexpected project key %s", got[0].Key)
	}

	// mutate original slice to ensure defensive copy
	projects[0].Key = "MUTATED"
	if cache.Projects()[0].Key != "KEY" {
		t.Fatalf("cache should not reflect external mutation")
	}
}

func TestCacheLastJQL(t *testing.T) {
	cache := NewCache()
	cache.SetLastJQL("project = KEY")
	if got := cache.LastJQL(); got != "project = KEY" {
		t.Fatalf("expected stored JQL, got %s", got)
	}
}
