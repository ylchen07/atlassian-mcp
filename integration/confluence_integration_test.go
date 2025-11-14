//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
)

func TestConfluenceListSpaces(t *testing.T) {
	requireIntegration(t)

	svc, siteURL := setupConfluenceClient(t)

	spaces, err := svc.ListSpaces(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListSpaces failed: %v", err)
	}

	if len(spaces) == 0 {
		t.Logf("no spaces returned from Confluence site %s", siteURL)
		return
	}

	t.Logf("Found %d spaces on %s", len(spaces), siteURL)
	for i, space := range spaces {
		desc := space.Description.Plain.Value
		if desc == "" {
			desc = "(no description)"
		}
		t.Logf("  [%d] %s (%s) - %s: %s", i+1, space.Key, space.ID, space.Name, desc)
	}
}

func TestConfluenceSearchPages(t *testing.T) {
	requireIntegration(t)

	svc, siteURL := setupConfluenceClient(t)

	// Search for pages with a common term
	cql := "type=page ORDER BY lastmodified DESC"
	pages, err := svc.SearchContent(context.Background(), cql, 5)
	if err != nil {
		t.Fatalf("SearchContent failed: %v", err)
	}

	if len(pages) == 0 {
		t.Logf("no pages found on %s with CQL: %s", siteURL, cql)
		return
	}

	t.Logf("Found %d pages on %s", len(pages), siteURL)
	for i, page := range pages {
		t.Logf("  [%d] %s (ID: %s) - %s [%s] v%d",
			i+1,
			page.Title,
			page.ID,
			page.Type,
			page.Status,
			page.Version.Number,
		)
	}
}

func TestConfluenceGetPage(t *testing.T) {
	requireIntegration(t)

	svc, siteURL := setupConfluenceClient(t)

	// First, search for a page to get its ID
	cql := "type=page ORDER BY lastmodified DESC"
	pages, err := svc.SearchContent(context.Background(), cql, 1)
	if err != nil {
		t.Fatalf("SearchContent failed: %v", err)
	}
	skipIfEmpty(t, pages, "pages")

	pageID := pages[0].ID
	pageTitle := pages[0].Title

	// Now retrieve the full page content
	page, err := svc.GetPage(context.Background(), pageID, []string{"body.storage", "version", "space"})
	if err != nil {
		t.Fatalf("GetPage failed for ID %s: %v", pageID, err)
	}

	if page == nil {
		t.Fatalf("GetPage returned nil for ID %s", pageID)
	}

	t.Logf("Retrieved page '%s' (ID: %s) from %s", pageTitle, pageID, siteURL)
	t.Logf("  Type: %s", page.Type)
	t.Logf("  Status: %s", page.Status)
	t.Logf("  Version: %d", page.Version.Number)
	t.Logf("  Body length: %d characters", len(page.Body.Storage.Value))

	if page.Body.Storage.Value == "" {
		t.Logf("  Warning: page body is empty")
	}
}

func TestConfluenceSearchAndGetPage(t *testing.T) {
	requireIntegration(t)

	svc, siteURL := setupConfluenceClient(t)

	// Search for pages with "Network" keyword
	searchTerm := "Network"
	cql := "type=page AND text ~ \"" + searchTerm + "\" ORDER BY lastmodified DESC"

	t.Logf("Searching for '%s' on %s", searchTerm, siteURL)
	pages, err := svc.SearchContent(context.Background(), cql, 3)
	if err != nil {
		t.Fatalf("SearchContent failed: %v", err)
	}

	if len(pages) == 0 {
		t.Skipf("no pages found matching '%s' on %s", searchTerm, siteURL)
		return
	}

	t.Logf("Found %d pages matching '%s'", len(pages), searchTerm)
	for i, page := range pages {
		t.Logf("  [%d] %s (ID: %s) v%d", i+1, page.Title, page.ID, page.Version.Number)
	}

	// Get the first page with full content
	firstPageID := pages[0].ID
	firstPageTitle := pages[0].Title

	t.Logf("\nRetrieving full content for '%s' (ID: %s)...", firstPageTitle, firstPageID)
	fullPage, err := svc.GetPage(context.Background(), firstPageID, []string{"body.storage", "version", "space"})
	if err != nil {
		t.Fatalf("GetPage failed for ID %s: %v", firstPageID, err)
	}

	t.Logf("Successfully retrieved page:")
	t.Logf("  Title: %s", fullPage.Title)
	t.Logf("  Type: %s", fullPage.Type)
	t.Logf("  Status: %s", fullPage.Status)
	t.Logf("  Version: %d", fullPage.Version.Number)
	t.Logf("  Body representation: %s", fullPage.Body.Storage.Representation)
	t.Logf("  Body length: %d characters", len(fullPage.Body.Storage.Value))

	if fullPage.Body.Storage.Value == "" {
		t.Errorf("expected page body to have content, got empty string")
	}

	// Verify the content contains expected format
	if fullPage.Body.Storage.Representation != "storage" {
		t.Errorf("expected storage representation, got %s", fullPage.Body.Storage.Representation)
	}
}
